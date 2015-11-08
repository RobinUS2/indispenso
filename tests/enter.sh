#!/bin/bash
docker exec -it `docker ps | grep indispenso_tests | awk '{print $1}'` bash
