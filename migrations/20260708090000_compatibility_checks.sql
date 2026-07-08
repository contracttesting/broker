CREATE TABLE compatibility_checks (
  id              BIGSERIAL PRIMARY KEY,
  participant_id  BIGINT NOT NULL REFERENCES participants(id),
  version         text NOT NULL,
  environment_id  BIGINT NOT NULL REFERENCES environments(id),
  deployable      boolean NOT NULL,
  created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ON compatibility_checks (participant_id, version, environment_id, created_at DESC);

CREATE TABLE compatibility_check_results (
  id                         BIGSERIAL PRIMARY KEY,
  check_id                   BIGINT NOT NULL REFERENCES compatibility_checks(id),
  counterpart_participant_id BIGINT NOT NULL REFERENCES participants(id),
  counterpart_version        text,
  deployable                 boolean NOT NULL,
  reason                     text
);

DROP TABLE compatibility_matrix;
