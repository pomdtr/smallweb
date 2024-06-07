package storage

import (
	"context"
	"crypto/sha1"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	log "log/slog"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/google/uuid"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

var (
	ErrMissingUser = errors.New("no user found")
)

func NewTursoDB(dbURL string) (*DB, error) {
	db, err := sql.Open("libsql", dbURL)
	if err != nil {
		return nil, err
	}

	return &DB{db: db}, nil
}

type DB struct {
	db *sql.DB
}

type User struct {
	ID        int        `json:"id"`
	PublicID  string     `json:"public_id"`
	PublicKey *PublicKey `json:"public_key,omitempty"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	CreatedAt *time.Time `json:"created_at"`
}

// PublicKey represents to public SSH key for a Smallweb user.
type PublicKey struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id,omitempty"`
	Key       string     `json:"key"`
	CreatedAt *time.Time `json:"created_at"`
}

func (me *DB) UserFromContext(ctx ssh.Context) (*User, error) {
	key, ok := ctx.Value("key").(string)
	if !ok {
		return nil, errors.New("no key found in context")
	}

	u, err := me.UserForKey(key)
	if err != nil {
		return nil, fmt.Errorf("no user found for key: %s", key)
	}

	return u, nil
}

// UserForKey returns the user for the given key, or optionally creates a new user with it.
func (me *DB) UserForKey(key string) (*User, error) {

	pk := &PublicKey{}
	u := &User{}
	if err := me.WrapTransaction(func(tx *sql.Tx) error {
		var err error
		r := me.selectPublicKey(tx, key)
		err = r.Scan(&pk.ID, &pk.UserID, &pk.Key)
		if err == sql.ErrNoRows {
			return ErrMissingUser
		}
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
	}); err != nil {
		return nil, err
	}
	return u, nil
}

func (me *DB) createUser(tx *sql.Tx, key string, email string, username string) error {
	publicID := uuid.New().String()
	err := me.insertUser(tx, publicID, email, username)
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

func (me *DB) insertUser(tx *sql.Tx, charmID string, email string, username string) error {
	_, err := tx.Exec(sqlInsertUser, charmID, email, username)
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

func (me *DB) deletePublicKey(tx *sql.Tx, userID int, key string) error {
	_, err := tx.Exec(sqlDeletePublicKey, userID, key)
	return err
}

func (me *DB) selectPublicKey(tx *sql.Tx, key string) *sql.Row {
	return tx.QueryRow(sqlSelectPublicKey, key)
}

func (me *DB) scanUser(r *sql.Row) (*User, error) {
	u := &User{}
	var username sql.NullString
	var email string
	var createdAt time.Time
	err := r.Scan(&u.ID, &u.PublicID, &username, &email, &createdAt)
	if err != nil {
		return nil, err
	}
	if username.Valid {
		u.Name = username.String
	}
	u.Email = email

	u.CreatedAt = &createdAt
	return u, nil
}

// PublicKeySha returns the SHA for a public key in hex format.
func PublicKeySha(key string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(key))) // nolint: gosec
}

// WrapTransaction runs the given function within a transaction.
func (me *DB) WrapTransaction(f func(tx *sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*100)
	defer cancel()
	tx, err := me.db.BeginTx(ctx, nil)
	if err != nil {
		log.Error("error starting transaction", "err", err)
		return err
	}
	err = f(tx)
	if err != nil {
		log.Error("error in transaction", "err", err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		log.Error("error committing transaction", "err", err)
		return err
	}
	return nil
}

// SetUserName sets a user name for the given user id.
func (me *DB) SetUserName(publicID string, name string) (*User, error) {
	var u *User
	log.Debug("Setting name for user", "name", name, "id", publicID)
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
			r := me.selectUserWithPublicID(tx, publicID)
			u, err = me.scanUser(r)
			if err != nil && err != sql.ErrNoRows {
				return err
			}
			if err == sql.ErrNoRows {
				return ErrMissingUser
			}

			err = me.updateUser(tx, publicID, name)
			if err != nil {
				return err
			}

			r = me.selectUserWithName(tx, name)
			u, err = me.scanUser(r)
			if err != nil {
				return err
			}
		}

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

func (me *DB) selectUserWithEmail(tx *sql.Tx, email string) *sql.Row {
	return tx.QueryRow(sqlSelectUserWithEmail, email)
}

func (me *DB) selectSession(tx *sql.Tx, sessionID string) *sql.Row {
	return tx.QueryRow(sqlSelectSessionWithID, sessionID)
}

func (me *DB) updateUser(tx *sql.Tx, charmID string, name string) error {
	_, err := tx.Exec(sqlUpdateUser, name, charmID)
	return err
}

type Session struct {
	ID        string
	Email     string
	Host      string
	ExpiresAt time.Time
}

func (me *DB) scanSession(r *sql.Row) (*Session, error) {
	s := &Session{}
	var expiresAt time.Time
	err := r.Scan(&s.ID, &s.Email, &s.Host, &expiresAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (me *DB) GetSession(sessionID string) (*Session, error) {
	var session *Session
	if err := me.WrapTransaction(func(tx *sql.Tx) error {
		sessionRow := me.selectSession(tx, sessionID)
		s, err := me.scanSession(sessionRow)
		if err != nil {
			return err
		}

		session = s
		return nil
	}); err != nil {
		return nil, err
	}

	return session, nil
}

func (me *DB) CreateUser(key string, email string, username string) (*User, error) {
	var u *User
	if err := me.WrapTransaction(func(tx *sql.Tx) error {
		var err error
		r := me.selectUserWithEmail(tx, email)
		u, err = me.scanUser(r)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == sql.ErrNoRows {
			err = me.createUser(tx, key, email, username)
			if err != nil {
				return err
			}
			r = me.selectUserWithEmail(tx, email)
			u, err = me.scanUser(r)
			if err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return u, nil
}

func (me *DB) CheckUserInfo(email string, username string) error {
	if err := me.WrapTransaction(func(tx *sql.Tx) error {
		if me.selectUserWithEmail(tx, email).Scan() != sql.ErrNoRows {
			return errors.New("user already exists")
		}

		if me.selectUserWithName(tx, username).Scan() != sql.ErrNoRows {
			return errors.New("username already exists")
		}

		return nil
	}); err != nil {
		return fmt.Errorf("error checking user info: %w", err)
	}

	return nil
}

func (me *DB) GetUserWithName(name string) (*User, error) {
	var user *User
	if err := me.WrapTransaction(func(tx *sql.Tx) error {
		r := me.selectUserWithName(tx, name)
		u, err := me.scanUser(r)
		if err != nil {
			return err
		}

		user = u
		return nil
	}); err != nil {
		return nil, err
	}
	return user, nil
}

func (me *DB) GetUserWithEmail(email string) (*User, error) {
	var user *User
	if err := me.WrapTransaction(func(tx *sql.Tx) error {
		r := me.selectUserWithEmail(tx, email)
		u, err := me.scanUser(r)
		if err != nil {
			return err
		}

		user = u
		return nil
	}); err != nil {
		return nil, err
	}
	return user, nil
}

func (me *DB) AddUserPublicKey(userID int, key string) error {
	if err := me.WrapTransaction(func(tx *sql.Tx) error {
		return me.insertPublicKey(tx, userID, key)
	}); err != nil {
		return err
	}

	return nil
}

func (me *DB) DeleteUserPublicKey(userID int, key string) error {
	if err := me.WrapTransaction(func(tx *sql.Tx) error {
		return me.deletePublicKey(tx, userID, key)
	}); err != nil {
		return err
	}

	return nil
}

func (me *DB) CreateSession(email string, host string) (string, error) {
	sessionID := uuid.New().String()
	if err := me.WrapTransaction(func(tx *sql.Tx) error {
		if err := me.insertSession(tx, sessionID, email, host, time.Now().Add(time.Hour*24)); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return "", err
	}

	return sessionID, nil
}

func (me *DB) insertSession(tx *sql.Tx, id string, email string, host string, expiresAt time.Time) error {
	_, err := tx.Exec(sqlCreateSession, id, email, host, expiresAt)
	return err
}
