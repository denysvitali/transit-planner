#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF_USAGE'
Usage:
  ./android/scripts/setup-gh-release-secrets.sh [--create-keystore] [--set-secrets]

Environment variables:
  KEYSTORE_PATH              Path to release keystore (default: upload.jks)
  KEYSTORE_STORE_PASSWORD    Keystore store password (auto-generated when creating)
  KEYSTORE_KEY_PASSWORD      Keystore key password (auto-generated when creating)
  KEYSTORE_VALIDITY_DAYS     Validity days for new key (default: 10000)
  KEYSTORE_DNAME             DName for new key (default: CN=TransitPlanner, OU=CI, O=TransitPlanner, L=Unknown, ST=Unknown, C=US)
  KEYSTORE_KEY_SIZE          RSA key size for new key (default: 2048)
  KEYSTORE_KEY_ALG           Key algorithm for new key (default: RSA)
  GITHUB_REPOSITORY          GitHub repo owner/repo for "gh secret set" (optional)

Options:
  --create-keystore  Force generation of a new keystore via keytool.
  --set-secrets     Push KEYSTORE_BASE64, KEYSTORE_STORE_PASSWORD,
                    KEYSTORE_KEY_PASSWORD to GitHub secrets.
  --force           Overwrite existing keystore when creating.
  -h, --help        Show this help.
EOF_USAGE
  exit 1
}

CREATE_KEYSTORE=false
SET_SECRETS=false
FORCE_CREATE=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --create-keystore)
      CREATE_KEYSTORE=true
      shift
      ;;
    --set-secrets)
      SET_SECRETS=true
      shift
      ;;
    --force)
      FORCE_CREATE=true
      shift
      ;;
    -h|--help)
      usage
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage
      ;;
  esac
done

KEYSTORE_PATH="${KEYSTORE_PATH:-upload.jks}"
KEY_ALIAS="${KEYSTORE_KEY_ALIAS:-upload}"
KEYSTORE_STORE_PASSWORD="${KEYSTORE_STORE_PASSWORD:-${STORE_PASSWORD:-}}"
KEYSTORE_KEY_PASSWORD="${KEYSTORE_KEY_PASSWORD:-${KEY_PASSWORD:-}}"
KEYSTORE_VALIDITY_DAYS="${KEYSTORE_VALIDITY_DAYS:-10000}"
KEYSTORE_DNAME="${KEYSTORE_DNAME:-CN=TransitPlanner, OU=CI, O=TransitPlanner, L=Unknown, ST=Unknown, C=US}"
KEYSTORE_KEY_SIZE="${KEYSTORE_KEY_SIZE:-2048}"
KEYSTORE_KEY_ALG="${KEYSTORE_KEY_ALG:-RSA}"
REPO="${GITHUB_REPOSITORY:-}"

for cmd in keytool base64; do
  if ! command -v "$cmd" >/dev/null; then
    echo "Missing required command: $cmd" >&2
    exit 1
  fi
done

random_password() {
  local length="$1"
  if command -v openssl >/dev/null; then
    openssl rand -base64 "$((length * 2))" | tr -dc 'A-Za-z0-9_-' | head -c "$length"
    return
  fi

  LC_ALL=C tr -dc 'A-Za-z0-9_-' < /dev/urandom | head -c "$length"
}

if [[ ! -f "$KEYSTORE_PATH" ]]; then
  CREATE_KEYSTORE=true
  echo "Keystore not found: $KEYSTORE_PATH. Generating a new keystore..." >&2
fi

if [[ "$CREATE_KEYSTORE" == "true" ]]; then
  if [[ -e "$KEYSTORE_PATH" && "$FORCE_CREATE" != "true" ]]; then
    echo "Keystore already exists: $KEYSTORE_PATH. Use --force to overwrite." >&2
    exit 1
  fi

  if [[ -z "$KEY_ALIAS" ]]; then
    echo "Missing KEYSTORE_KEY_ALIAS." >&2
    exit 1
  fi

  if [[ -z "$KEYSTORE_STORE_PASSWORD" ]]; then
    KEYSTORE_STORE_PASSWORD="$(random_password 32)"
  fi

  if [[ -z "$KEYSTORE_KEY_PASSWORD" ]]; then
    KEYSTORE_KEY_PASSWORD="$(random_password 32)"
  fi

  keytool -genkeypair \
    -v \
    -keystore "$KEYSTORE_PATH" \
    -alias "$KEY_ALIAS" \
    -keyalg "$KEYSTORE_KEY_ALG" \
    -keysize "$KEYSTORE_KEY_SIZE" \
    -validity "$KEYSTORE_VALIDITY_DAYS" \
    -storepass "$KEYSTORE_STORE_PASSWORD" \
    -keypass "$KEYSTORE_KEY_PASSWORD" \
    -dname "$KEYSTORE_DNAME"
fi

if [[ -z "$KEYSTORE_STORE_PASSWORD" || -z "$KEYSTORE_KEY_PASSWORD" ]]; then
  echo "Missing KEYSTORE_STORE_PASSWORD or KEYSTORE_KEY_PASSWORD for existing keystore inspection." >&2
  echo "If generating a keystore, these values are auto-generated." >&2
  exit 1
fi

if [[ -z "$KEY_ALIAS" ]]; then
  KEY_ALIAS="$(keytool -list -v -keystore "$KEYSTORE_PATH" -storepass "$KEYSTORE_STORE_PASSWORD" 2>/dev/null | awk '/Alias name:/{print $3; exit}')"
fi

if ! keytool -list -keystore "$KEYSTORE_PATH" -storepass "$KEYSTORE_STORE_PASSWORD" -alias "$KEY_ALIAS" >/dev/null; then
  echo "Keytool failed to read alias '$KEY_ALIAS'. Check password and alias." >&2
  exit 1
fi

KEYTOOL_INFO="$(keytool -list -v -keystore "$KEYSTORE_PATH" -storepass "$KEYSTORE_STORE_PASSWORD" -alias "$KEY_ALIAS")"
KEYSTORE_BASE64="$(base64 "$KEYSTORE_PATH" | tr -d '\n')"

echo "Keystore: $KEYSTORE_PATH"
echo "Alias: $KEY_ALIAS"
echo "$KEYTOOL_INFO" | awk '/Owner:/ {print; next} /SHA-256:|SHA256:/ {print; exit}'

echo
cat <<EOF_VALUES
# Copy these into GitHub -> Settings -> Secrets and variables -> Actions -> New repository secret
export KEYSTORE_BASE64="$KEYSTORE_BASE64"
export KEYSTORE_STORE_PASSWORD="$KEYSTORE_STORE_PASSWORD"
export KEYSTORE_KEY_PASSWORD="$KEYSTORE_KEY_PASSWORD"
EOF_VALUES

echo

echo "Apply each value manually in GitHub Actions secrets (KEYSTORE_BASE64, KEYSTORE_STORE_PASSWORD, KEYSTORE_KEY_PASSWORD)."
echo "Set KEYSTORE_KEY_ALIAS as a regular workflow var (not a secret)."

echo

if ! $SET_SECRETS; then
  exit 0
fi

if ! command -v gh >/dev/null; then
  echo "Missing gh CLI. Install and login, or use manual secret setup above." >&2
  exit 1
fi

if [[ -z "$REPO" ]]; then
  echo "GITHUB_REPOSITORY is required when using --set-secrets." >&2
  exit 1
fi

export KEYSTORE_KEY_ALIAS="$KEY_ALIAS"
gh secret set KEYSTORE_BASE64 --repo "$REPO" --body "$KEYSTORE_BASE64"
gh secret set KEYSTORE_STORE_PASSWORD --repo "$REPO" --body "$KEYSTORE_STORE_PASSWORD"
gh secret set KEYSTORE_KEY_PASSWORD --repo "$REPO" --body "$KEYSTORE_KEY_PASSWORD"
echo "Set KEYSTORE_KEY_ALIAS in CI vars as: $KEY_ALIAS"

echo "Uploaded KEYSTORE_BASE64, KEYSTORE_STORE_PASSWORD, KEYSTORE_KEY_PASSWORD to $REPO"
