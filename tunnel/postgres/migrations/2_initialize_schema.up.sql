-- define passage schema
BEGIN;

CREATE SEQUENCE IF NOT EXISTS passage.sshd_ports AS INTEGER MINVALUE 49152 MAXVALUE 57343;
CREATE SEQUENCE IF NOT EXISTS passage.tunnel_ports AS INTEGER MINVALUE 57344 MAXVALUE 65535;

CREATE TYPE passage.tunnel_type AS ENUM('normal', 'reverse');

CREATE TABLE IF NOT EXISTS passage.tunnels(
    id                  UUID DEFAULT uuid_generate_v4(),
    created_at          TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    enabled             BOOLEAN NOT NULL DEFAULT true,

    tunnel_port         INT DEFAULT nextval('passage.tunnel_ports'),
    ssh_user            VARCHAR,
    ssh_host            VARCHAR NOT NULL,
    ssh_port            INTEGER NOT NULL,
    service_host        VARCHAR NOT NULL,
    service_port        INTEGER NOT NULL,

    PRIMARY KEY(id)
);

CREATE TABLE IF NOT EXISTS passage.reverse_tunnels(
    id          UUID DEFAULT uuid_generate_v4(),
    created_at  TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    enabled     BOOLEAN NOT NULL DEFAULT true,

    sshd_port       INT DEFAULT nextval('passage.sshd_ports') UNIQUE,
    tunnel_port     INT DEFAULT nextval('passage.tunnel_ports') UNIQUE,
    last_used_at    TIMESTAMP WITHOUT TIME ZONE,

    PRIMARY KEY(id)
);

CREATE TABLE IF NOT EXISTS passage.key_authorizations(
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,

    key_id      UUID NOT NULL,
    tunnel_type passage.tunnel_type NOT NULL,
    tunnel_id   UUID NOT NULL,

    -- This is converted to a primary key in a later migration.
    UNIQUE(key_id, tunnel_type, tunnel_id)
);

COMMIT;
