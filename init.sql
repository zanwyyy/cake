-- Tạo bảng users
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0
);

-- Tạo bảng transactions
CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    from_user TEXT NOT NULL,
    to_user   TEXT NOT NULL,
    amount BIGINT NOT NULL
);

-- Dữ liệu mẫu cho users
INSERT INTO users (name, balance) VALUES
('Alice', 1000),
('Bob', 5000),
('Charlie', 2000);

-- Dữ liệu mẫu cho transactions
INSERT INTO transactions (from_user, to_user, amount) VALUES
('1', '2', 200),
('2', '3', 100),
('1', '3', 50);
