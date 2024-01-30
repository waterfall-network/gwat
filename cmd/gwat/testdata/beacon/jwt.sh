#!/bin/bash

# This script is used to test the JWT authentication scheme.
# It takes two arguments:
# 1. The file containing the JWT secret key
# 2. The URL to send the request to
#
# Examples:
# ./jwt.sh jwt.hex http://localhost:8551
# ./jwt.sh $DATA_DIR/geth/jwtsecret http://localhost:8551

set -e -u -o pipefail

# Get the JWT secret and URL from the command line
jwtsecretfile=$1
url=$2

# Construct the JWT header
JWT_HEADER=$(echo -n '{"typ":"JWT","alg":"HS256"}' | openssl base64 -e -A | tr '+' '-' | tr '/' '_' | tr -d '=')

# Read the secret key and convert to bytes
secret=$(cat $jwtsecretfile | cut -c 3-)
secretBytes=$(printf $secret | xxd -r -p)

# Construct the JWT payload with the iat claim and encode as base64
jwt_payload=$(echo -n "{\"iat\":$(date +%s)}" | openssl base64 -e -A | tr '+' '-' | tr '/' '_' | tr -d '=')

# Create the JWT claims
jwt_claims="$JWT_HEADER.$jwt_payload"

# Create the JWT signature
jwt_signature=$(echo -n "$jwt_claims" \
    | openssl dgst -sha256 -hmac "$secretBytes" -binary \
    | openssl base64 -e -A | tr '+' '-' | tr '/' '_' | tr -d '=')

# Create the JWT
jwt="$jwt_claims.$jwt_signature"

# Make the RPC request
data='{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
curl -s \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $jwt" \
    -d "$data" \
    $url