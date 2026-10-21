set -euo pipefail

BASE="${BASE_URL:-http://localhost:8080}"
EMAIL="e2e+$(date +%s%N)@example.com"
PASSWORD="S3cret-pass-123"
FAILED=0

pass() { printf '  \033[32mOK\033[0m   %s\n' "$1"; }
fail() { printf '  \033[31mFAIL\033[0m %s\n' "$1"; FAILED=1; }

call() {
  local method="$1" path="$2" body="${3:-}" token="${4:-}" out
  local tmp; tmp="$(mktemp)"
  local args=(-sS -o "$tmp" -D - -w '%{http_code}' -X "$method" "$BASE$path" -H 'Content-Type: application/json')
  [[ -n "$token" ]] && args+=(-H "Authorization: Bearer $token")
  [[ -n "$body"  ]] && args+=(-d "$body")
  out="$(curl "${args[@]}")"
  STATUS="${out: -3}"
  HEADERS="${out%???}"
  BODY="$(cat "$tmp")"; rm -f "$tmp"
}

expect() { # expect STATUS "описание"
  if [[ "$STATUS" == "$1" ]]; then pass "$2 → $STATUS"; else fail "$2 → ожидали $1, получили $STATUS: $BODY"; fi
}

echo "== 1. register"
call POST /v1/auth/register "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}"
expect 200 "POST /v1/auth/register"

echo "== 2. login (customer_id в JWT ещё нет)"
call POST /v1/auth/login "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}"
expect 200 "POST /v1/auth/login"
TOKEN="$(jq -r '.accessToken // empty' <<<"$BODY")"
[[ -n "$TOKEN" ]] || { fail "accessToken не получен"; exit 1; }

echo "== 3. GET /me до привязки профиля"
call GET /v1/customers/me "" "$TOKEN"
expect 412 "GET /v1/customers/me"
jq -e '.code=="FAILED_PRECONDITION" and (.details|type=="array") and (.request_id|length>0)' <<<"$BODY" >/dev/null \
  && pass "envelope: code/details[]/request_id" || fail "неверный формат ошибки: $BODY"

echo "== 4. ждём Kafka-привязку customer (≈2с)"
sleep 3

echo "== 5. повторный login (теперь customer_id в JWT)"
call POST /v1/auth/login "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}"
expect 200 "POST /v1/auth/login"
TOKEN="$(jq -r '.accessToken' <<<"$BODY")"

echo "== 6. GET /me"
call GET /v1/customers/me "" "$TOKEN"
expect 200 "GET /v1/customers/me"

echo "== 7. PATCH /me (NEW → PROFILE_FILLED)"
call PATCH /v1/customers/me '{"fullName":"Nino Beridze","dateOfBirth":"1995-04-12T00:00:00Z","address":"Tbilisi, Rustaveli 1","phone":"+995555123456","citizenship":"GE"}' "$TOKEN"
expect 200 "PATCH /v1/customers/me"
[[ "$(jq -r '.status' <<<"$BODY")" == "CUSTOMER_STATUS_PROFILE_FILLED" ]] \
  && pass "статус PROFILE_FILLED" || fail "статус: $(jq -r '.status' <<<"$BODY")"

echo "== 8. GET /me/status"
call GET /v1/customers/me/status "" "$TOKEN"
expect 200 "GET /v1/customers/me/status"

echo "== 9. CLIENT не может листать клиентов (RBAC в customer-service)"
call GET /v1/customers "" "$TOKEN"
expect 403 "GET /v1/customers (без роли OFFICER/ADMIN)"

echo "== 10. без токена"
call GET /v1/customers/me
expect 401 "GET /v1/customers/me без Authorization"

echo "== 11. rate limit: 50 быстрых запросов"
GOT429=0; RETRY=""
for _ in $(seq 1 50); do
  call GET /v1/customers/me/status "" "$TOKEN"
  if [[ "$STATUS" == "429" ]]; then GOT429=1; RETRY="$(grep -i '^retry-after:' <<<"$HEADERS" | tr -d '\r' || true)"; break; fi
done
if [[ $GOT429 -eq 1 && -n "$RETRY" ]]; then pass "получили 429 с заголовком [$RETRY]"; else fail "429 с Retry-After не получен"; fi

echo
if [[ $FAILED -eq 0 ]]; then echo "E2E PASSED"; else echo "E2E FAILED"; exit 1; fi
