package cmd

const (
	sqlInsertUser            = `INSERT INTO user (public_id, email) VALUES (?, ?)`
	sqlSelectUserWithCharmID = `SELECT id, public_id, name, email, email_verified, created_at FROM user WHERE public_id = ?`
	sqlSelectUserWithID      = `SELECT id, public_id, name, email, email_verified, created_at FROM user WHERE id = ?`
	sqlSelectUserWithName    = `SELECT id, public_id, name, email, email_verified, created_at FROM user WHERE name like ?`
	sqlSelectUserPublicKeys  = `SELECT id, public_key, created_at FROM public_key WHERE user_id = ?`
	sqlInsertPublicKey       = `INSERT INTO public_key (user_id, public_key) VALUES (?, ?)
	ON CONFLICT (user_id, public_key) DO UPDATE SET
	user_id = excluded.user_id,
	public_key = excluded.public_key`
	sqlSelectPublicKey             = `SELECT id, user_id, public_key FROM public_key WHERE public_key = ?`
	sqlInsertVerificationCode      = `INSERT INTO email_verification_code (code, user_id, email, expires_at) VALUES  (?, ?, ?, ?)`
	sqlSelectVerificationCode      = `SELECT id, code, user_id, email, expires_at FROM email_verification_code WHERE user_id = ?`
	sqlDeleteUserVerificationCodes = `DELETE FROM email_verification_code WHERE user_id = ?`
	sqlVerifyUserEmail             = `UPDATE user SET email_verified = true WHERE id = ?`
	sqlUpdateUser                  = `UPDATE user SET name = ? WHERE public_id = ?`
)
