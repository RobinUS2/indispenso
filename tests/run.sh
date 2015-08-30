#!/bin/bash
docker rm -f indispenso_tests
docker build --tag=indispenso_tests .
docker run --name indispenso_tests -p 897:897 indispenso_tests
