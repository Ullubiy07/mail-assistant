DROP TABLE IF EXISTS mail_requests;
DROP TABLE IF EXISTS users;

CREATE TABLE users (
    id         UUID PRIMARY KEY,
    username   VARCHAR(255) NOT NULL UNIQUE,
    email      VARCHAR(255) NOT NULL UNIQUE,
    password   VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE mail_requests (
    id           UUID PRIMARY KEY,
    user_id      UUID NOT NULL REFERENCES users(id),
    address      VARCHAR(255) NOT NULL,
    folder       VARCHAR(255) NOT NULL,
    uid_next     BIGINT NOT NULL,
    uid_validity BIGINT NOT NULL
);

CREATE INDEX idx_mail
ON mail_requests (user_id, address);
