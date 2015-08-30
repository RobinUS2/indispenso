#!/bin/bash
docker rm -f indispenso_tests
docker build --tag=indispenso_tests --no-cache .
docker run --name indispenso_tests indispenso_tests
