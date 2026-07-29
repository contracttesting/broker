#!/bin/bash

# paths below are relative to the repository root, so run from anywhere
cd "$(dirname "$0")/../.." || exit 1
go build -C ./cli -o "$HOME/.local/bin/ctio" ./cmd/ || exit 1

ctio create-environment production

ctio create-participant users
ctio create-participant pets
ctio create-participant app

ctio publish ./examples/petstore/users_v1.yaml --participant users --version v1
ctio publish ./examples/petstore/pets_v1.yaml --participant pets --version v1
ctio publish ./examples/petstore/app_v1.yaml --participant app --version v1

ctio can-i-deploy users --version v1 --environment production
ctio record-deployment users --version v1 --environment production

ctio can-i-deploy pets --version v1 --environment production
ctio record-deployment pets --version v1 --environment production

ctio can-i-deploy app --version v1 --environment production
ctio record-deployment app --version v1 --environment production