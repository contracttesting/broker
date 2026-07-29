#!/bin/bash

# paths below are relative to the repository root, so run from anywhere
cd "$(dirname "$0")/../.." || exit 1

sudo psql -q -U alefcastelo -d contracttesting -v ON_ERROR_STOP=1 <<'SQL'
DO $$
DECLARE tables text;
BEGIN
  SELECT string_agg(format('public.%I', tablename), ', ')
    INTO tables
    FROM pg_tables
   WHERE schemaname = 'public' AND tablename <> 'schema_migrations';
  EXECUTE format('TRUNCATE %s RESTART IDENTITY CASCADE', tables);
END $$;
SQL
sudo go build -C ./cli -o /usr/local/bin/ctio ./cmd/

ctio create-environment production

ctio create-participant users
ctio create-participant pets
ctio create-participant app

ctio publish ./examples/petstore/users/users_v1.yaml --participant users --version v1
ctio publish ./examples/petstore/pets/pets_v1.yaml --participant pets --version v1
ctio publish ./examples/petstore/app/app_v1.yaml --participant app --version v1

ctio can-i-deploy users --version v1 --environment production
ctio record-deployment users --version v1 --environment production

# ctio can-i-deploy pets --version v1 --environment production
ctio record-deployment pets --version v1 --environment production

# ctio can-i-deploy app --version v1 --environment production
# ctio record-deployment app --version v1 --environment production