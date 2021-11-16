CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE IF NOT EXISTS passage.keys(
   id           UUID DEFAULT uuid_generate_v4(),
   created_at   timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
   contents     VARCHAR,

   PRIMARY KEY(id)
);
