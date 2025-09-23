
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    password TEXT NOT NULL
);


CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    from_user BIGINT NOT NULL,
    to_user BIGINT NOT NULL,
    amount BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_from_user FOREIGN KEY (from_user) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_to_user FOREIGN KEY (to_user) REFERENCES users(id) ON DELETE CASCADE
);


CREATE INDEX IF NOT EXISTS idx_transactions_from_user ON transactions(from_user);
CREATE INDEX IF NOT EXISTS idx_transactions_to_user ON transactions(to_user);


INSERT INTO users (id, name, balance, password)
VALUES 
    (1, 'alice', 10000, '$2a$10$6Uo9VmU.2UTwJ6iXf/3E5eL3zQiQiyN6VvT54StM2Q/Dye09zyGrW'),
    (2, 'bob', 5000, '$2a$10$0Bch0Vk5AasCtDE1kYrmg.2K3b0DGfxMFjGxa4WcO77eExKcGtB3O'),
    (3, 'bb', 5000, '$2a$10$QOIEuaIEl2KAd0Jfb3iJVeQIUjoFnYPKFA0JLS3cE.pOwUEOLfv9a')
ON CONFLICT (id) DO NOTHING;


