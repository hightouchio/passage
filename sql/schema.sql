-- cleanup (REMOVE IF RUNNING IN PROD)
DROP SCHEMA IF EXISTS passage CASCADE;

-- define passage schema
BEGIN;

CREATE SCHEMA passage;

CREATE SEQUENCE IF NOT EXISTS passage.sshd_ports AS INTEGER MINVALUE 49152 MAXVALUE 57343;
CREATE SEQUENCE IF NOT EXISTS passage.tunnel_ports AS INTEGER MINVALUE 57344 MAXVALUE 65535;

CREATE TYPE passage.key_type AS ENUM('private', 'public');
CREATE TYPE passage.tunnel_type AS ENUM('normal', 'reverse');

CREATE TABLE IF NOT EXISTS passage.tunnels(
    id          INT GENERATED ALWAYS AS IDENTITY,
    created_at  TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,

    tunnel_port         INT DEFAULT nextval('passage.tunnel_ports'),
    server_endpoint     TEXT NOT NULL,
    server_port         INTEGER NOT NULL,
    service_endpoint    TEXT NOT NULL,
    service_port        INTEGER NOT NULL,

    PRIMARY KEY(id)
);

CREATE TABLE IF NOT EXISTS passage.reverse_tunnels(
    id          INT GENERATED ALWAYS AS IDENTITY,
    created_at  TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,

    sshd_port       INT DEFAULT nextval('passage.sshd_ports') UNIQUE,
    tunnel_port     INT DEFAULT nextval('passage.tunnel_ports') UNIQUE,
    last_used_at    TIMESTAMP WITHOUT TIME ZONE,

    PRIMARY KEY(id)
);

CREATE TABLE IF NOT EXISTS passage.keys(
    id INT GENERATED ALWAYS AS IDENTITY,

    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,

    key_type    passage.key_type NOT NULL,
    public_key  VARCHAR,
    private_key VARCHAR,

    PRIMARY KEY(id)
);

CREATE TABLE IF NOT EXISTS passage.key_authorizations(
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,

    key_id      INT NOT NULL,
    tunnel_type passage.tunnel_type NOT NULL,
    tunnel_id   INT NOT NULL,

    UNIQUE(tunnel_type, tunnel_id),
    CONSTRAINT fk_key FOREIGN KEY (key_id) REFERENCES passage.keys(id)
);

COMMIT;