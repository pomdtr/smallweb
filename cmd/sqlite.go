package cmd

const (
	sqlInsertUser            = `INSERT INTO user (public_id) VALUES (?)`
	sqlSelectUserWithCharmID = `SELECT id, public_id, name, created_at FROM user WHERE public_id = ?`
	sqlSelectUserWithID      = `SELECT id, public_id, name, created_at FROM user WHERE id = ?`
	sqlSelectUserPublicKeys  = `SELECT id, public_key, created_at FROM public_key WHERE user_id = ?`
	sqlInsertPublicKey       = `INSERT INTO public_key (user_id, public_key) VALUES (?, ?)
	ON CONFLICT (user_id, public_key) DO UPDATE SET
	user_id = excluded.user_id,
	public_key = excluded.public_key`
	sqlSelectPublicKey    = `SELECT id, user_id, public_key FROM public_key WHERE public_key = ?`
	sqlSelectUserWithName = `SELECT id, public_id, name, created_at FROM user WHERE name like ?`
	sqlUpdateUser         = `UPDATE user SET name = ? WHERE public_id = ?`
)
