#!/bin/sh
mkdir -p /var/lib/soju/uploads
echo "Checking DB permissions and existence..."
ls -l /var/lib/soju/soju.db
exec /usr/bin/soju -config /etc/soju.conf
