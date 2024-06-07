package cmd

const (
	sqlInsertUser            = `INSERT INTO user (public_id, email, name) VALUES (?, ?, ?)`
	sqlSelectUserWithCharmID = `SELECT id, public_id, name, email, created_at FROM user WHERE public_id = ?`
	sqlSelectUserWithID      = `SELECT id, public_id, name, email, created_at FROM user WHERE id = ?`
	sqlSelectUserWithName    = `SELECT id, public_id, name, email, created_at FROM user WHERE name like ?`
	sqlSelectUserWithEmail   = `SELECT id, public_id, name, email, created_at FROM user WHERE email = ?`
	sqlSelectUserPublicKeys  = `SELECT id, public_key, created_at FROM public_key WHERE user_id = ?`
	sqlInsertPublicKey       = `INSERT INTO public_key (user_id, public_key) VALUES (?, ?)
	ON CONFLICT (user_id, public_key) DO UPDATE SET
	user_id = excluded.user_id,
	public_key = excluded.public_key`
	sqlDeletePublicKey     = `DELETE FROM public_key WHERE user_id = ? AND public_key = ?`
	sqlSelectPublicKey     = `SELECT id, user_id, public_key FROM public_key WHERE public_key = ?`
	sqlVerifyUserEmail     = `UPDATE user SET email_verified = true WHERE id = ?`
	sqlUpdateUser          = `UPDATE user SET name = ? WHERE public_id = ?`
	sqlCreateSession       = `INSERT INTO session (id, email, host, expires_at) VALUES (?, ?, ?, ?)`
	sqlSelectSessionWithID = `SELECT id, email, host, expires_at FROM session WHERE id = ?`
)
