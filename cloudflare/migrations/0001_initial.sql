CREATE TABLE IF NOT EXISTS records (
  collection TEXT NOT NULL,
  id TEXT NOT NULL,
  data TEXT NOT NULL DEFAULT '{}',
  created TEXT NOT NULL,
  updated TEXT NOT NULL,
  PRIMARY KEY (collection, id)
);

CREATE INDEX IF NOT EXISTS idx_records_collection
  ON records (collection, updated DESC);

CREATE TABLE IF NOT EXISTS auth_credentials (
  collection TEXT NOT NULL,
  record_id TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  PRIMARY KEY (collection, record_id),
  FOREIGN KEY (collection, record_id)
    REFERENCES records (collection, id) ON DELETE CASCADE
);
