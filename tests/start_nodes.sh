#!/bin/bash

# Update repo
git pull

# Build
../build.sh

# Start the server
../start_server.sh &

# Start another client
../indispenso --server=false &

# Make sure we wait (without wasting cpu cycles), we read "nothing" :)
cat
