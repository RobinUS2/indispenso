#!/bin/bash
# Auto create foldder
mkdir -p /etc/indispenso

# This will start the server of indispenso and will generate a unique token the first time
SECURE_TOKEN_FILE="server.token"
if [ ! -f $SECURE_TOKEN_FILE ]; then
	echo "Server has no secure token yet, generating token"
	TOKEN=`openssl rand -base64 128 | tr -d '\n'`
	echo $TOKEN > $SECURE_TOKEN_FILE
	echo "Your token is (make sure to store this):"
	echo "$TOKEN"
fi

# Start server with the token
TOKEN=`cat $SECURE_TOKEN_FILE`
./indispenso --server="true" --secure-token="$TOKEN"
