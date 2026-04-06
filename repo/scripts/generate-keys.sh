#!/bin/bash
# Generate keys for LedgerMint deployment.
#
# Usage:
#   ./generate-keys.sh              Print keys for .env (development only)
#   ./generate-keys.sh --keyfile    Create an encrypted keyfile for staging/production

set -euo pipefail

JWT_KEY=$(openssl rand -base64 32)
AES_KEY=$(openssl rand -hex 32)
DB_PASS=$(openssl rand -base64 24)

if [[ "${1:-}" == "--keyfile" ]]; then
    read -rsp "Enter passphrase for keyfile encryption: " PASSPHRASE
    echo
    if [[ -z "$PASSPHRASE" ]]; then
        echo "Error: passphrase must not be empty" >&2
        exit 1
    fi

    DB_URL="postgres://ledgermint:${DB_PASS}@postgres:5432/ledgermint?sslmode=require"

    TMPFILE=$(mktemp)
    trap 'rm -f "$TMPFILE"' EXIT

    cat > "$TMPFILE" <<SECRETS_EOF
{
  "database_url": "${DB_URL}",
  "jwt_signing_key": "${JWT_KEY}",
  "aes_master_key": "${AES_KEY}"
}
SECRETS_EOF

    OUTFILE="${2:-secrets.enc}"
    go run ./backend/cmd/keyfile-tool encrypt "$TMPFILE" "$OUTFILE" "$PASSPHRASE"

    echo ""
    echo "Database password (set in your database, not in the keyfile):"
    echo "  DB_PASSWORD=$DB_PASS"
    echo ""
    echo "To use in staging/production, set:"
    echo "  APP_ENV=production"
    echo "  SECRETS_KEYFILE=$OUTFILE"
    echo "  SECRETS_PASSPHRASE=<your-passphrase>"
else
    echo "=== Development Keys (plaintext, .env only) ==="
    echo ""
    echo "JWT_SIGNING_KEY=$JWT_KEY"
    echo "AES_MASTER_KEY=$AES_KEY"
    echo "DB_PASSWORD=$DB_PASS"
    echo ""
    echo "Add these to your .env file. You MUST also set APP_ENV=development"
    echo "for plaintext secrets to be accepted."
    echo ""
    echo "For staging/production, use: $0 --keyfile [output-path]"
fi
