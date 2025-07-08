#!/bin/bash

# Check if domain argument is provided
if [ -z "$1" ]; then
  echo "Usage: $0 <domain>"
  exit 1
fi

DOMAIN=$1
SECRET_NAME="$DOMAIN"
CERT_NAME="$DOMAIN"
DAYS_VALID=365
KEY_SIZE=2048

# Create temporary OpenSSL config file with SAN
CONFIG_FILE=$(mktemp)

cat > "$CONFIG_FILE" <<EOF
[ req ]
default_bits       = $KEY_SIZE
prompt             = no
default_md         = sha256
distinguished_name = dn
x509_extensions    = v3_req

[ dn ]
C = US
ST = State
L = City
O = Organization
OU = Unit
CN = $DOMAIN

[ v3_req ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = $DOMAIN
EOF

# Generate private key
openssl genrsa -out "$CERT_NAME.key" $KEY_SIZE

# Generate self-signed certificate using the config with SAN
openssl req -x509 -nodes -days $DAYS_VALID -key "$CERT_NAME.key" \
  -out "$CERT_NAME.crt" -config "$CONFIG_FILE" -extensions v3_req

# Clean up config file
rm "$CONFIG_FILE"

# Create Kubernetes TLS secret
kubectl create secret tls "${SECRET_NAME//./-}" \
  --cert="$CERT_NAME.crt" \
  --key="$CERT_NAME.key" \
  --dry-run=client -o yaml | \
  grep -v creationTimestamp | \
  yq eval ".metadata.annotations.\"envoy.kaasops.io/domains\" = \"$DOMAIN\" \
  | .metadata.labels.\"envoy.kaasops.io/secret-type\" = \"sds-cached\"" \
  - > "secret-${SECRET_NAME//./-}.yaml"

# Cleanup cert and key files
rm "$CERT_NAME.crt"
rm "$CERT_NAME.key"

echo "TLS Secret YAML created: secret-${SECRET_NAME//./-}.yaml"
