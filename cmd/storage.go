package cmd

import (
	"context"
	"crypto/sha1"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	log "log/slog"
	"time"

	"github.com/google/uuid"
	"modernc.org/sqlite"
	sqlitelib "modernc.org/sqlite/lib"
)

var (
	ErrMissingUser = errors.New("no user found")
)

func NewDB(dbPath string) (*DB, error) {
	dbName := fmt.Sprintf("file:%s", dbPath)
	if !exists(dbPath) {
		db, err := sql.Open("sqlite", dbName)
		if err != nil {
			return nil, err
		}

		_, err = db.Exec(initQuery)
		if err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", dbName)
	if err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

//go:embed sql/init.sql
var initQuery string

type DB struct {
	db *sql.DB
}

type User struct {
	ID        int        `json:"id"`
	PublicID  string     `json:"public_id"`
	PublicKey *PublicKey `json:"public_key,omitempty"`
	Name      string     `json:"name"`
	CreatedAt *time.Time `json:"created_at"`
}

// PublicKey represents to public SSH key for a Smallweb user.
type PublicKey struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id,omitempty"`
	Key       string     `json:"key"`
	CreatedAt *time.Time `json:"created_at"`
}

// UserForKey returns the user for the given key, or optionally creates a new user with it.
func (me *DB) UserForKey(key string, create bool) (*User, error) {
	pk := &PublicKey{}
	u := &User{}
	err := me.WrapTransaction(func(tx *sql.Tx) error {
		var err error
		r := me.selectPublicKey(tx, key)
		err = r.Scan(&pk.ID, &pk.UserID, &pk.Key)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows && !create {
			return ErrMissingUser
		}
		if err == sql.ErrNoRows {
			log.Debug("Creating user for key", "key", PublicKeySha(key))
			err = me.createUser(tx, key)
			if err != nil {
				return err
			}
		}
		r = me.selectPublicKey(tx, key)
		err = r.Scan(&pk.ID, &pk.UserID, &pk.Key)
		if err != nil {
			return err
		}

		r = me.selectUserWithID(tx, pk.UserID)
		u, err = me.scanUser(r)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows {
			return ErrMissingUser
		}
		u.PublicKey = pk
		return nil
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (me *DB) createUser(tx *sql.Tx, key string) error {
	publicID := uuid.New().String()
	err := me.insertUser(tx, publicID)
	if err != nil {
		return err
	}
	r := me.selectUserWithPublicID(tx, publicID)
	u, err := me.scanUser(r)
	if err != nil {
		return err
	}
	return me.insertPublicKey(tx, u.ID, key)
}

func (me *DB) insertUser(tx *sql.Tx, charmID string) error {
	_, err := tx.Exec(sqlInsertUser, charmID)
	return err
}

func (me *DB) selectUserWithPublicID(tx *sql.Tx, charmID string) *sql.Row {
	return tx.QueryRow(sqlSelectUserWithCharmID, charmID)
}

func (me *DB) selectUserWithID(tx *sql.Tx, userID int) *sql.Row {
	return tx.QueryRow(sqlSelectUserWithID, userID)
}

func (me *DB) insertPublicKey(tx *sql.Tx, userID int, key string) error {
	_, err := tx.Exec(sqlInsertPublicKey, userID, key)
	return err
}

func (me *DB) selectPublicKey(tx *sql.Tx, key string) *sql.Row {
	return tx.QueryRow(sqlSelectPublicKey, key)
}

func (me *DB) scanUser(r *sql.Row) (*User, error) {
	u := &User{}
	var un sql.NullString
	var ca sql.NullTime
	err := r.Scan(&u.ID, &u.PublicID, &un, &ca)
	if err != nil {
		return nil, err
	}
	if un.Valid {
		u.Name = un.String
	}
	if ca.Valid {
		u.CreatedAt = &ca.Time
	}
	return u, nil
}

// PublicKeySha returns the SHA for a public key in hex format.
func PublicKeySha(key string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(key))) // nolint: gosec
}

// WrapTransaction runs the given function within a transaction.
func (me *DB) WrapTransaction(f func(tx *sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("error starting transaction", "err", err)
		return err
	}
	for {
		err = f(tx)
		if err != nil {
			serr, ok := err.(*sqlite.Error)
			if ok && serr.Code() == sqlitelib.SQLITE_BUSY {
				continue
			}
			log.Error("error in transaction", "err", err)
			return err
		}
		err = tx.Commit()
		if err != nil {
			log.Error("error committing transaction", "err", err)
			return err
		}
		break
	}
	return nil
}

// SetUserName sets a user name for the given user id.
func (me *DB) SetUserName(charmID string, name string) (*User, error) {
	var u *User
	log.Debug("Setting name for user", "name", name, "id", charmID)
	err := me.WrapTransaction(func(tx *sql.Tx) error {
		// nolint: godox
		// TODO: this should be handled with unique constraints in the database instead.
		var err error
		r := me.selectUserWithName(tx, name)
		u, err = me.scanUser(r)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows {
			r := me.selectUserWithPublicID(tx, charmID)
			u, err = me.scanUser(r)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == sql.ErrNoRows {
				return ErrMissingUser
			}

			err = me.updateUser(tx, charmID, name)
			if err != nil {
				return err
			}

			r = me.selectUserWithName(tx, name)
			u, err = me.scanUser(r)
			if err != nil {
				return err
			}
		}
		// if u.CharmID != charmID {
		// 	return charm.ErrNameTaken
		// }
		return nil
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (me *DB) selectUserWithName(tx *sql.Tx, name string) *sql.Row {
	return tx.QueryRow(sqlSelectUserWithName, name)
}

func (me *DB) updateUser(tx *sql.Tx, charmID string, name string) error {
	_, err := tx.Exec(sqlUpdateUser, name, charmID)
	return err
}
