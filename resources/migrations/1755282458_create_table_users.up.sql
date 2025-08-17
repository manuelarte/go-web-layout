CREATE TABLE users
(
    id         uuid NOT NULL,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP,
    username   text NOT NULL UNIQUE,
    password   text NOT NULL,
    PRIMARY KEY (id)
)
