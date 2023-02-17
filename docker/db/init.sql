CREATE TABLE settings (
     id SERIAL PRIMARY KEY,
     key VARCHAR(50) NOT NULL,
     value VARCHAR(50) NOT NULL,
     created_at TIMESTAMP NOT NULL DEFAULT NOW(),
     updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
