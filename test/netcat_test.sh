#!/bin/bash

set -o allexport; source .env; set +o allexport

docker build -f ./DockerfileNetcat -t netcat_test:latest .
response=$(docker run --env-file ./.env --network testing_net netcat_test:latest)

response=${response//\"}
expected_response="$MESSAGE"

if [ "$response" = "$expected_response" ]; then
    echo "✅ Test Success ✅"
else
    echo "❌ Test Failed ❌"
    echo "Received: $response"
    echo "Expected: $expected_response"
fi