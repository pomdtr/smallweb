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
	ID            int        `json:"id"`
	PublicID      string     `json:"public_id"`
	PublicKey     *PublicKey `json:"public_key,omitempty"`
	Name          string     `json:"name"`
	Email         string     `json:"email"`
	EmailVerified bool       `json:"email_verified"`
	CreatedAt     *time.Time `json:"created_at"`
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
		return nil, fmt.Errorf("no user found for key: %w", err)
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

func (me *DB) createUser(tx *sql.Tx, key string, email string) error {
	publicID := uuid.New().String()
	err := me.insertUser(tx, publicID, email)
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

func (me *DB) insertUser(tx *sql.Tx, charmID string, email string) error {
	_, err := tx.Exec(sqlInsertUser, charmID, email)
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
	var username sql.NullString
	var email string
	var emailVerified bool
	var createdAt time.Time
	err := r.Scan(&u.ID, &u.PublicID, &username, &email, &emailVerified, &createdAt)
	if err != nil {
		return nil, err
	}
	if username.Valid {
		u.Name = username.String
	}
	u.Email = email
	u.EmailVerified = emailVerified

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

func (me *DB) createVerificationCode(user *User, code string) error {
	if _, err := me.db.Exec(sqlDeleteUserVerificationCodes, user.ID); err != nil {
		return err
	}

	expiresAt := time.Now().Add(time.Minute * 15)
	if _, err := me.db.Exec(sqlInsertVerificationCode, code, user.ID, user.Email, expiresAt); err != nil {
		return err
	}

	return nil
}

type VerificationCode struct {
	ID        int
	Code      string
	UserID    int
	Email     string
	ExpiresAt time.Time
}

func (me *DB) verifyVerificationCode(user *User, code string) (bool, error) {
	row := me.db.QueryRow(sqlSelectVerificationCode, user.ID)

	var storedCode VerificationCode
	if err := row.Scan(&storedCode.ID, &storedCode.Code, &storedCode.UserID, &storedCode.Email, &storedCode.ExpiresAt); err != nil {
		return false, err
	}

	if _, err := me.db.Exec(sqlDeleteUserVerificationCodes, user.ID); err != nil {
		return false, err
	}

	if _, err := me.db.Exec(sqlVerifyUserEmail, user.ID); err != nil {
		return false, err
	}

	if time.Now().After(storedCode.ExpiresAt) {
		return false, nil
	}

	return storedCode.Code == code, nil
}

func (me *DB) selectUserWithName(tx *sql.Tx, name string) *sql.Row {
	return tx.QueryRow(sqlSelectUserWithName, name)
}

func (me *DB) updateUser(tx *sql.Tx, charmID string, name string) error {
	_, err := tx.Exec(sqlUpdateUser, name, charmID)
	return err
}
