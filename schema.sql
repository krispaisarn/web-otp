CREATE TABLE IF NOT EXISTS otps (
    id         BIGINT       AUTO_RANDOM PRIMARY KEY,
    email      VARCHAR(255) NOT NULL,
    otp        VARCHAR(6)   NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP    NOT NULL,
    used       BOOLEAN      NOT NULL DEFAULT FALSE,
    used_at    TIMESTAMP    NULL     DEFAULT NULL,
    INDEX idx_email      (email),
    INDEX idx_email_otp  (email, otp),
    INDEX idx_expires_at (expires_at),
    INDEX idx_created_at (created_at)
);
