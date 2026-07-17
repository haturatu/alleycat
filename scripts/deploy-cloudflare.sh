#!/usr/bin/env bash
set -Eeuo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
ENV_FILE=${1:-"$ROOT_DIR/.env"}
API_BASE="https://api.cloudflare.com/client/v4"
DB_NAME="alleycat-db"
BUCKET_NAME="alleycat-media"
WORKER_NAME="alleycat"

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

for command_name in curl jq npm openssl sed; do
  command -v "$command_name" >/dev/null 2>&1 || die "$command_name is required"
done

[[ -f "$ENV_FILE" ]] || die "environment file not found: $ENV_FILE"
set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

for variable_name in CF_ACCOUNT_ID CF_API_TOKEN CF_SERCRET_KEY CF_S3_ENDPOINT; do
  [[ -n "${!variable_name:-}" ]] || die "$variable_name is missing in $ENV_FILE"
done
[[ "$CF_ACCOUNT_ID" =~ ^[a-fA-F0-9]{32}$ ]] || die "CF_ACCOUNT_ID must be a 32-character account ID"
[[ "$CF_S3_ENDPOINT" == https://* ]] || die "CF_S3_ENDPOINT must use https://"
[[ ${#CF_SERCRET_KEY} -ge 32 ]] || die "CF_SERCRET_KEY must contain at least 32 characters"

export CLOUDFLARE_ACCOUNT_ID="$CF_ACCOUNT_ID"
export CLOUDFLARE_API_TOKEN="$CF_API_TOKEN"

api_request() {
  local method=$1
  local path=$2
  local body=${3:-}
  local response code payload message
  if [[ -n "$body" ]]; then
    response=$(curl -sS -X "$method" -H "Authorization: Bearer $CF_API_TOKEN" -H 'Content-Type: application/json' --data "$body" -w '\n%{http_code}' "$API_BASE$path")
  else
    response=$(curl -sS -X "$method" -H "Authorization: Bearer $CF_API_TOKEN" -H 'Content-Type: application/json' -w '\n%{http_code}' "$API_BASE$path")
  fi
  code=${response##*$'\n'}
  payload=${response%$'\n'*}
  if [[ "$code" -lt 200 || "$code" -ge 300 ]] || [[ $(jq -r '.success // false' <<<"$payload") != true ]]; then
    message=$(jq -r '[.errors[]?.message] | join("; ")' <<<"$payload")
    die "Cloudflare API $method $path failed (HTTP $code): ${message:-unknown error}"
  fi
  printf '%s' "$payload"
}

printf 'Validating Cloudflare credentials...\n'
api_request GET "/accounts/$CF_ACCOUNT_ID/tokens/verify" >/dev/null

printf 'Preparing frontend assets...\n'
npm --prefix "$ROOT_DIR/frontend" ci
VITE_BASE=/admin/ npm --prefix "$ROOT_DIR/frontend" run build
mkdir -p "$ROOT_DIR/cloudflare/public/admin"
cp -R "$ROOT_DIR/frontend/dist/." "$ROOT_DIR/cloudflare/public/admin/"
cp -R "$ROOT_DIR/frontend/default-public-asset/." "$ROOT_DIR/cloudflare/public/"

printf 'Installing Cloudflare deployment dependencies...\n'
npm --prefix "$ROOT_DIR/cloudflare" ci

printf 'Ensuring D1 database exists...\n'
d1_list=$(api_request GET "/accounts/$CF_ACCOUNT_ID/d1/database?per_page=100")
d1_id=$(jq -r --arg name "$DB_NAME" '.result[] | select(.name == $name) | .uuid' <<<"$d1_list" | head -n 1)
if [[ -z "$d1_id" ]]; then
  d1_created=$(api_request POST "/accounts/$CF_ACCOUNT_ID/d1/database" "$(jq -cn --arg name "$DB_NAME" '{name:$name}')")
  d1_id=$(jq -r '.result.uuid' <<<"$d1_created")
  printf 'Created D1 database %s.\n' "$DB_NAME"
else
  printf 'Using existing D1 database %s.\n' "$DB_NAME"
fi
[[ "$d1_id" =~ ^[a-fA-F0-9-]{36}$ ]] || die "Cloudflare returned an invalid D1 database ID"

printf 'Ensuring R2 bucket exists...\n'
r2_list=$(api_request GET "/accounts/$CF_ACCOUNT_ID/r2/buckets?per_page=100")
if ! jq -e --arg name "$BUCKET_NAME" '.result.buckets[] | select(.name == $name)' <<<"$r2_list" >/dev/null; then
  api_request POST "/accounts/$CF_ACCOUNT_ID/r2/buckets" "$(jq -cn --arg name "$BUCKET_NAME" '{name:$name,locationHint:"apac"}')" >/dev/null
  printf 'Created R2 bucket %s (Standard storage).\n' "$BUCKET_NAME"
else
  printf 'Using existing R2 bucket %s.\n' "$BUCKET_NAME"
fi

sed "s/__D1_DATABASE_ID__/$d1_id/g" "$ROOT_DIR/cloudflare/wrangler.template.jsonc" > "$ROOT_DIR/cloudflare/wrangler.deploy.jsonc"

printf 'Applying D1 migrations...\n'
(
  cd "$ROOT_DIR/cloudflare"
  npx wrangler d1 migrations apply "$DB_NAME" --remote --config wrangler.deploy.jsonc
)

printf 'Deploying Worker and static assets...\n'
(
  cd "$ROOT_DIR/cloudflare"
  npx wrangler deploy --config wrangler.deploy.jsonc
)

printf 'Installing the Worker authentication secret...\n'
(
  cd "$ROOT_DIR/cloudflare"
  printf '%s' "$CF_SERCRET_KEY" | npx wrangler secret put AUTH_SECRET --config wrangler.deploy.jsonc
)

subdomain_response=$(api_request GET "/accounts/$CF_ACCOUNT_ID/workers/subdomain")
workers_subdomain=$(jq -r '.result.subdomain' <<<"$subdomain_response")
[[ -n "$workers_subdomain" && "$workers_subdomain" != null ]] || die "Workers.dev subdomain is not configured"
deployment_url="https://$WORKER_NAME.$workers_subdomain.workers.dev"

admin_email=${CF_ADMIN_EMAIL:-admin@alleycat.local}
generated_password=false
if [[ -z "${CF_ADMIN_PASSWORD:-}" ]]; then
  admin_password=$(openssl rand -base64 24 | tr -d '\n/=+' | cut -c1-24)
  generated_password=true
else
  admin_password=$CF_ADMIN_PASSWORD
fi
[[ ${#admin_password} -ge 12 ]] || die "CF_ADMIN_PASSWORD must contain at least 12 characters"

printf 'Bootstrapping the CMS administrator when needed...\n'
bootstrap_payload=$(jq -cn --arg email "$admin_email" --arg password "$admin_password" '{email:$email,password:$password}')
bootstrap_response=$(curl -sS -X POST -H "x-bootstrap-secret: $CF_SERCRET_KEY" -H 'Content-Type: application/json' --data "$bootstrap_payload" "$deployment_url/api/bootstrap")
bootstrap_created=$(jq -r '.created // false' <<<"$bootstrap_response")
if [[ "$bootstrap_created" == true ]]; then
  printf '\nCMS administrator created:\n'
  printf '  Email: %s\n' "$admin_email"
  printf '  Password: %s\n' "$admin_password"
  if [[ "$generated_password" == true ]]; then
    printf '  Save this password now; it will not be displayed again.\n'
  fi
fi

health_status=$(curl -sS -o /dev/null -w '%{http_code}' "$deployment_url/healthz")
[[ "$health_status" == 200 ]] || die "deployment health check failed (HTTP $health_status)"
printf '\nDeployment completed within the Cloudflare free-tier architecture.\n'
printf '  Site:  %s/\n' "$deployment_url"
printf '  Admin: %s/admin/\n' "$deployment_url"
printf '  API:   %s/api/\n' "$deployment_url"
