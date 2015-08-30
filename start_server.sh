#!/bin/bash
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
cd $DIR
# Auto create foldder
mkdir -p /etc/indispenso

# This will start the server of indispenso and will generate a unique token the first time
if [ ! -f "/etc/indispenso/indispenso.conf" ]; then
	echo "Server has no secure token yet, generating token"
	TOKEN=`openssl rand -base64 128 | tr -d '\n'`
	echo "Your token is:"
	echo "$TOKEN"

	# Update conf file
	touch /etc/indispenso/indispenso.conf
	cat /etc/indispenso/indispenso.conf | grep -v "secure_token" > /etc/indispenso/indispenso.conf.bak
	echo "secure_token: $TOKEN" >> /etc/indispenso/indispenso.conf.bak
	echo "server_enabled: true" >> /etc/indispenso/indispenso.conf.bak
	mv /etc/indispenso/indispenso.conf.bak /etc/indispenso/indispenso.conf
fi

# Start server
./indispenso --debug=false
