CREATE TABLE IF NOT EXISTS user (
    id INTEGER NOT NULL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    email TEXT NOT NULL,
    public_id TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (email, public_id)
);

CREATE TABLE IF NOT EXISTS public_key (
    id INTEGER NOT NULL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    public_key VARCHAR(2048) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, public_key),
    CONSTRAINT user_id_fk
        FOREIGN KEY (user_id)
        REFERENCES user (id)
        ON DELETE CASCADE
        ON UPDATE CASCADE
);
