#!/bin/sh
set -eu

HTTP_CONF="/etc/nginx/templates/http.conf"
HTTPS_CONF="/etc/nginx/templates/https.conf"
ACTIVE_CONF="/etc/nginx/conf.d/default.conf"
SSL_CERT="/etc/nginx/ssl/tls.crt"
SSL_KEY="/etc/nginx/ssl/tls.key"

if [ -s "$SSL_CERT" ] && [ -s "$SSL_KEY" ]; then
  cp "$HTTPS_CONF" "$ACTIVE_CONF"
  echo "[entrypoint] SSL certificates detected, HTTPS config enabled"
else
  cp "$HTTP_CONF" "$ACTIVE_CONF"
  echo "[entrypoint] SSL certificates not found, HTTP-only config enabled"
fi

exec nginx -g 'daemon off;'
