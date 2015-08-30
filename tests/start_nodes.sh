#!/bin/bash

# Update repo
git pull

# Build
../build.sh

# Start the server
../start_server.sh &

# Make sure we wait (without wasting cpu cycles), we read "nothing" :)
cat
