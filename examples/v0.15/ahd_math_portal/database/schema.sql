-- Ahd Akademi Matematik -- schema for the AhdCode v0.15 Web reference
-- application.
--
--   mysql -u <user> -p <database> < database/schema.sql
--
-- Everything is utf8mb4/InnoDB. The application never issues DDL at runtime:
-- this file is the single place the shape of the data is defined.

SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS users (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name          VARCHAR(120)    NOT NULL,
    email         VARCHAR(190)    NOT NULL,
    password_hash VARCHAR(255)    NOT NULL,
    role          ENUM('user', 'admin') NOT NULL DEFAULT 'user',
    is_active     TINYINT(1)      NOT NULL DEFAULT 1,
    -- auth_version invalidates sessions established before a password change.
    -- The session stores the value it authenticated with; a mismatch on a
    -- later request ends that session.
    auth_version  INT UNSIGNED    NOT NULL DEFAULT 1,
    created_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY users_email_unique (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS questions (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    title        VARCHAR(200)    NOT NULL,
    topic        VARCHAR(120)    NOT NULL,
    body         TEXT            NOT NULL,
    status       ENUM('draft', 'published') NOT NULL DEFAULT 'draft',
    created_by   BIGINT UNSIGNED NULL,
    published_at DATETIME        NULL,
    created_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY questions_status_index (status, published_at),
    CONSTRAINT questions_created_by_foreign FOREIGN KEY (created_by)
        REFERENCES users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS solutions (
    id                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id           BIGINT UNSIGNED NOT NULL,
    question_id       BIGINT UNSIGNED NOT NULL,
    -- The uploader's filename, kept for display only. It is never used to
    -- build a path: stored_filename comes from UploadedFile.save.
    original_filename VARCHAR(255)    NOT NULL,
    stored_filename   VARCHAR(255)    NOT NULL,
    mime_type         VARCHAR(120)    NOT NULL,
    file_size         BIGINT UNSIGNED NOT NULL,
    created_at        DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    -- The one-solution-per-user-per-question rule. The application checks it
    -- too, but this constraint is the authority: it is what decides the race
    -- between two simultaneous uploads.
    UNIQUE KEY solutions_user_question_unique (user_id, question_id),
    CONSTRAINT solutions_user_foreign FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT solutions_question_foreign FOREIGN KEY (question_id)
        REFERENCES questions (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS password_resets (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id       BIGINT UNSIGNED NOT NULL,
    -- selector is the public lookup handle; verifier_hash is the Argon2id
    -- hash of the secret half. The usable reset credential is never stored:
    -- a database copy alone cannot be turned back into a working link.
    selector      VARCHAR(64)     NOT NULL,
    verifier_hash VARCHAR(255)    NOT NULL,
    expires_at    DATETIME        NOT NULL,
    used_at       DATETIME        NULL,
    created_at    DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY password_resets_selector_unique (selector),
    KEY password_resets_user_index (user_id),
    CONSTRAINT password_resets_user_foreign FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS site_settings (
    setting_key   VARCHAR(64)  NOT NULL,
    setting_value VARCHAR(255) NOT NULL,
    updated_at    DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (setting_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Application settings, not deployment secrets. Never put database, SMTP, or
-- API credentials in this table.
INSERT INTO site_settings (setting_key, setting_value) VALUES
    ('site_name', 'Ahd Akademi Matematik'),
    ('header_color', '#0d6efd')
ON DUPLICATE KEY UPDATE setting_value = setting_value;
