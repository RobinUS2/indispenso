#!/bin/bash
docker kill `docker ps | grep indispenso_tests | awk '{print $1}'`
