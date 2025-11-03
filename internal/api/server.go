package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Server implements the ServerInterface from the generated code
type Server struct {
	rootDir string
	blobDir string
	dbPath  string
}

// NewServer creates a new API server
func NewServer(rootDir string) *Server {
	return &Server{
		rootDir: rootDir,
		blobDir: filepath.Join(rootDir, ".smallweb", "data", "blobs"),
		dbPath:  filepath.Join(rootDir, ".smallweb", "data", "db.sqlite"),
	}
}

// GetBlob implements the GET /blob/{key} endpoint
func (s *Server) GetBlob(w http.ResponseWriter, r *http.Request, key string, params GetBlobParams) {
	// Check if requesting keys list
	if key == "" || strings.HasSuffix(key, "/") || strings.HasSuffix(key, ":") {
		// List keys
		prefix := strings.TrimSuffix(strings.TrimSuffix(key, "/"), ":")
		keys, err := s.getKeys(prefix)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(keys)
		return
	}

	// Get item
	accept := ""
	if params.Accept != nil {
		accept = *params.Accept
	}
	value, isBinary, err := s.getItem(key, accept)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if isBinary {
		w.Header().Set("Content-Type", "application/octet-stream")
	} else {
		w.Header().Set("Content-Type", "text/plain")
	}
	w.Write(value)
}

// HasBlob implements the HEAD /blob/{key} endpoint
func (s *Server) HasBlob(w http.ResponseWriter, r *http.Request, key string) {
	if s.hasItem(key) {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// SetBlob implements the PUT /blob/{key} endpoint
func (s *Server) SetBlob(w http.ResponseWriter, r *http.Request, key string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.setItem(key, body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write([]byte("OK"))
}

// DeleteBlob implements the DELETE /blob/{key} endpoint
func (s *Server) DeleteBlob(w http.ResponseWriter, r *http.Request, key string) {
	// Check if clearing all keys with prefix
	if strings.HasSuffix(key, "/") || strings.HasSuffix(key, ":") {
		prefix := strings.TrimSuffix(strings.TrimSuffix(key, "/"), ":")
		if err := s.clear(prefix); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		// Remove single item
		if err := s.removeItem(key); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Write([]byte("OK"))
}

// ExecuteSQLite implements the POST /sqlite/execute endpoint
func (s *Server) ExecuteSQLite(w http.ResponseWriter, r *http.Request) {
	var body ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sqlText, args, err := s.parseStatement(body.Statement)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	res, err := s.executeStatement(r.Context(), db, sqlText, args)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// BatchSQLite implements the POST /sqlite/batch endpoint
func (s *Server) BatchSQLite(w http.ResponseWriter, r *http.Request) {
	var body BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Begin transaction. For read mode, use a read-only tx when possible.
	var tx *sql.Tx
	mode := ""
	if body.Mode != nil {
		mode = string(*body.Mode)
	}
	if strings.EqualFold(mode, "read") {
		tx, err = db.BeginTx(r.Context(), &sql.TxOptions{ReadOnly: true})
	} else {
		tx, err = db.BeginTx(r.Context(), &sql.TxOptions{})
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	results := make([]ExecuteResult, 0, len(body.Statements))
	for _, stmt := range body.Statements {
		sqlText, args, err := s.parseStatementItem(stmt)
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		res, err := s.executeStatementTx(r.Context(), tx, sqlText, args)
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		results = append(results, res)
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// Helper methods for blob storage

func (s *Server) getItem(key, accept string) ([]byte, bool, error) {
	filePath := filepath.Join(s.blobDir, key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false, err
	}

	isBinary := strings.Contains(accept, "application/octet-stream")
	return data, isBinary, nil
}

func (s *Server) hasItem(key string) bool {
	filePath := filepath.Join(s.blobDir, key)
	_, err := os.Stat(filePath)
	return err == nil
}

func (s *Server) setItem(key string, value []byte) error {
	filePath := filepath.Join(s.blobDir, key)

	// Create parent directories if they don't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, value, 0644)
}

func (s *Server) removeItem(key string) error {
	filePath := filepath.Join(s.blobDir, key)
	return os.Remove(filePath)
}

func (s *Server) getKeys(prefix string) ([]string, error) {
	keys := []string{}
	basePath := filepath.Join(s.blobDir, prefix)

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(s.blobDir, path)
			if err != nil {
				return err
			}
			keys = append(keys, relPath)
		}
		return nil
	})

	if err != nil && os.IsNotExist(err) {
		return keys, nil
	}

	return keys, err
}

func (s *Server) clear(prefix string) error {
	basePath := filepath.Join(s.blobDir, prefix)

	// If prefix is empty, remove all files but keep the directory structure
	if prefix == "" {
		return filepath.Walk(s.blobDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && path != s.blobDir {
				return os.Remove(path)
			}
			return nil
		})
	}

	// Remove the entire directory tree for the prefix
	return os.RemoveAll(basePath)
}

// Helper methods for SQLite operations

func (s *Server) parseStatement(stmt ExecuteRequest_Statement) (string, []any, error) {
	// Try to get as string
	str, err := stmt.AsExecuteRequestStatement0()
	if err == nil {
		return str, nil, nil
	}

	// Try to get as StatementObject
	obj, err := stmt.AsStatementObject()
	if err != nil {
		return "", nil, fmt.Errorf("statement must be string or object with sql")
	}

	if obj.Args == nil {
		return obj.Sql, nil, nil
	}

	// Parse args
	return s.parseArgs(obj.Sql, *obj.Args)
}

func (s *Server) parseStatementItem(item BatchRequest_Statements_Item) (string, []any, error) {
	// Try to get as string
	str, err := item.AsBatchRequestStatements0()
	if err == nil {
		return str, nil, nil
	}

	// Try to get as StatementObject
	obj, err := item.AsStatementObject()
	if err != nil {
		return "", nil, fmt.Errorf("statement must be string or object with sql")
	}

	if obj.Args == nil {
		return obj.Sql, nil, nil
	}

	// Parse args
	return s.parseArgs(obj.Sql, *obj.Args)
}

func (s *Server) parseArgs(sqlText string, args StatementObject_Args) (string, []any, error) {
	// Try array args
	arr, err := args.AsStatementObjectArgs0()
	if err == nil {
		return sqlText, arr, nil
	}

	// Try object args -> convert to deterministic positional slice by sorting keys
	m, err := args.AsStatementObjectArgs1()
	if err != nil {
		return "", nil, fmt.Errorf("could not parse args array or object")
	}

	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// sort keys to be deterministic
	sort.Strings(keys)
	res := make([]any, 0, len(keys))
	for _, k := range keys {
		res = append(res, m[k])
	}
	return sqlText, res, nil
}

func (s *Server) executeStatement(ctx context.Context, db *sql.DB, sqlText string, args []any) (ExecuteResult, error) {
	return s.executeUsing(ctx, db, nil, sqlText, args)
}

func (s *Server) executeStatementTx(ctx context.Context, tx *sql.Tx, sqlText string, args []any) (ExecuteResult, error) {
	return s.executeUsing(ctx, nil, tx, sqlText, args)
}

func (s *Server) executeUsing(ctx context.Context, db *sql.DB, tx *sql.Tx, sqlText string, args []any) (ExecuteResult, error) {
	isQuery := s.isLikelyQuery(sqlText)

	var result ExecuteResult

	if isQuery {
		var rows *sql.Rows
		var err error

		if tx != nil {
			rows, err = tx.QueryContext(ctx, sqlText, args...)
		} else {
			rows, err = db.QueryContext(ctx, sqlText, args...)
		}
		if err != nil {
			return result, err
		}
		defer rows.Close()

		cols, err := rows.Columns()
		if err != nil {
			return result, err
		}
		colTypes := make([]string, len(cols))
		if cts, err := rows.ColumnTypes(); err == nil {
			for i, ct := range cts {
				colTypes[i] = ct.DatabaseTypeName()
			}
		}

		results := make([][]interface{}, 0)
		for rows.Next() {
			vals := make([]interface{}, len(cols))
			scanArgs := make([]interface{}, len(cols))
			for i := range scanArgs {
				scanArgs[i] = &vals[i]
			}
			if err := rows.Scan(scanArgs...); err != nil {
				return result, err
			}

			// Convert SQL types to JSON friendly values
			for i, v := range vals {
				vals[i] = s.convertSQLValue(v)
			}
			results = append(results, vals)
		}

		result.Columns = &cols
		result.ColumnTypes = &colTypes
		result.Rows = &results
		rowsAffected := int64(0)
		result.RowsAffected = &rowsAffected
		return result, nil
	}

	// Non-query: Exec
	var res sql.Result
	var err error
	if tx != nil {
		res, err = tx.ExecContext(ctx, sqlText, args...)
	} else {
		res, err = db.ExecContext(ctx, sqlText, args...)
	}
	if err != nil {
		return result, err
	}

	ra, _ := res.RowsAffected()
	li, _ := res.LastInsertId()

	cols := []string{}
	colTypes := []string{}
	rows := [][]interface{}{}

	result.Columns = &cols
	result.ColumnTypes = &colTypes
	result.Rows = &rows
	result.RowsAffected = &ra
	if li != 0 {
		result.LastInsertRowid = &li
	}

	return result, nil
}

func (s *Server) isLikelyQuery(sqlText string) bool {
	t := strings.TrimSpace(sqlText)
	up := strings.ToUpper(t)
	return strings.HasPrefix(up, "SELECT") || strings.HasPrefix(up, "WITH") || strings.HasPrefix(up, "PRAGMA") || strings.HasPrefix(up, "EXPLAIN")
}

func (s *Server) convertSQLValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case int64, float64, bool:
		return t
	case []byte:
		s := string(t)
		// Try to convert numeric strings to numbers when it looks like a number
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
		return s
	case string:
		return t
	case time.Time:
		return t.Format(time.RFC3339)
	default:
		return fmt.Sprint(t)
	}
}

// NewHandler creates an http.Handler using the generated code and implementation
func NewHandler(rootDir string) http.Handler {
	server := NewServer(rootDir)
	return HandlerWithOptions(server, StdHTTPServerOptions{
		BaseURL: "/v1",
	})
}
