#!/bin/bash

# Update repo
#git pull

# Build
../build.sh

TOKEN=`openssl rand -base64 128 | tr -d '\n'`
# Start the server
export IND_HOME=`mktemp -d`
echo -e "Starting server with home dir: ${IND_HOME}"
../indispenso -d -s -t "${TOKEN}" > out_server.log 2>&1 &
sleep 3

# Start another few clients
export IND_HOME=`mktemp -d`
echo -e "Starting client 1 with home dir: ${IND_HOME}"
echo "clientPort: 1998" > ${IND_HOME}/indispenso.yaml
../indispenso -i "client-one" -d -t "${TOKEN}" -e "localhost" > out_client1.log 2>&1 &
export IND_HOME=`mktemp -d`
echo -e "Starting client 2 with home dir: ${IND_HOME}"
echo "clientPort: 1999" > ${IND_HOME}/indispenso.yaml
../indispenso -i "client-two" -d -t "${TOKEN}"  -e "localhost" > out_client2.log 2>&1 &

# Shutdown after 30 seconds
{
    sleep 30
    killall indispenso
    killall tail
    kill $$
} &

# Read output
#tail -f out_server.log
