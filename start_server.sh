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

	# Update conf file
	cat /etc/indispenso/indispenso.conf | grep -v "secure_token" > /etc/indispenso/indispenso.conf.bak
	echo "secure_token: $TOKEN" >> /etc/indispenso/indispenso.conf.bak
	echo "server: true" >> /etc/indispenso/indispenso.conf.bak
	mv /etc/indispenso/indispenso.conf.bak /etc/indispenso/indispenso.conf
fi

# Start server
./indispenso
