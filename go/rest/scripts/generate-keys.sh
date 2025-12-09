#!/bin/bash

# RSA key pair generation script for JWT (RS256)
# This script generates private and public keys for JWT signing and verification

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
KEYS_DIR="$PROJECT_ROOT/.keys"

# Create keys directory if it doesn't exist
mkdir -p "$KEYS_DIR"

echo "Generating RSA key pair for JWT (RS256)..."

# Generate private key (2048 bit)
openssl genrsa -out "$KEYS_DIR/private_key.pem" 2048

# Extract public key from private key
openssl rsa -in "$KEYS_DIR/private_key.pem" -pubout -out "$KEYS_DIR/public_key.pem"

# Generate Key ID (kid) with timestamp
KID="key-$(date +%Y-%m-%d)"
echo -n "$KID" > "$KEYS_DIR/kid"

# Set secure permissions
chmod 600 "$KEYS_DIR/private_key.pem"
chmod 644 "$KEYS_DIR/public_key.pem"
chmod 644 "$KEYS_DIR/kid"

echo "✓ RSA key pair generated successfully!"
echo ""
echo "Private key: $KEYS_DIR/private_key.pem (keep this secret!)"
echo "Public key:  $KEYS_DIR/public_key.pem"
echo "Key ID:      $KID (saved to $KEYS_DIR/kid)"
echo ""
echo "NOTE: Add .keys/ to your .gitignore to prevent committing private keys."
