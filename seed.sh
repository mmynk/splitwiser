#!/usr/bin/env bash
# Seed script: creates 3 users, friendships, a group, and a couple of bills.
# Usage: ./seed.sh [BASE_URL]
# Default BASE_URL is http://localhost:9090
# Safe to re-run: skips steps that already succeeded.

set -euo pipefail

BASE="${1:-http://localhost:9090}"
AUTH="$BASE/splitwiser.v1.AuthService"
FRIEND="$BASE/splitwiser.v1.FriendService"
GROUP="$BASE/splitwiser.v1.GroupService"
BILL="$BASE/splitwiser.v1.SplitService"

post() {
  local url="$1" token="${2:-}" body="$3"
  local auth_header=()
  [[ -n "$token" ]] && auth_header=(-H "Authorization: Bearer $token")
  curl -sf -X POST "$url" \
    -H "Content-Type: application/json" \
    "${auth_header[@]}" \
    -d "$body"
}

register_or_login() {
  local email="$1" password="$2" display_name="$3"
  local resp
  resp=$(post "$AUTH/Register" "" \
    "{\"email\":\"$email\",\"password\":\"$password\",\"displayName\":\"$display_name\"}" 2>/dev/null) \
    || resp=$(post "$AUTH/Login" "" \
        "{\"email\":\"$email\",\"password\":\"$password\"}")
  echo "$resp"
}

echo "==> Registering users (or logging in if already exist)..."

alice=$(register_or_login "alice@example.com" "password123" "Alice")
alice_token=$(echo "$alice" | jq -r '.token')
alice_id=$(echo "$alice" | jq -r '.user.id')
echo "    Alice: $alice_id"

bob=$(register_or_login "bob@example.com" "password123" "Bob")
bob_token=$(echo "$bob" | jq -r '.token')
bob_id=$(echo "$bob" | jq -r '.user.id')
echo "    Bob:   $bob_id"

carol=$(register_or_login "carol@example.com" "password123" "Carol")
carol_token=$(echo "$carol" | jq -r '.token')
carol_id=$(echo "$carol" | jq -r '.user.id')
echo "    Carol: $carol_id"

echo "==> Alice sends friend requests to Bob and Carol (skips if already friends)..."
post "$FRIEND/SendFriendRequest" "$alice_token" "{\"addresseeId\":\"$bob_id\"}" > /dev/null 2>&1 || true
post "$FRIEND/SendFriendRequest" "$alice_token" "{\"addresseeId\":\"$carol_id\"}" > /dev/null 2>&1 || true

echo "==> Bob accepts any pending friend requests..."
bob_requests=$(post "$FRIEND/ListFriendRequests" "$bob_token" '{"incoming":true}')
for req_id in $(echo "$bob_requests" | jq -r '(.requests // []) | .[].id'); do
  post "$FRIEND/RespondToFriendRequest" "$bob_token" "{\"requestId\":\"$req_id\",\"accept\":true}" > /dev/null
done

echo "==> Carol accepts any pending friend requests..."
carol_requests=$(post "$FRIEND/ListFriendRequests" "$carol_token" '{"incoming":true}')
for req_id in $(echo "$carol_requests" | jq -r '(.requests // []) | .[].id'); do
  post "$FRIEND/RespondToFriendRequest" "$carol_token" "{\"requestId\":\"$req_id\",\"accept\":true}" > /dev/null
done

echo "==> Alice creates group 'Weekend Crew' (or reuses existing)..."
existing_groups=$(post "$GROUP/ListGroups" "$alice_token" '{}')
group_id=$(echo "$existing_groups" | jq -r '(.groups // []) | .[] | select(.name=="Weekend Crew") | .id' | head -1)

if [[ -z "$group_id" ]]; then
  group=$(post "$GROUP/CreateGroup" "$alice_token" "$(jq -n \
    --arg bn "Alice" --arg bi "$alice_id" \
    --arg cn "Bob"   --arg ci "$bob_id" \
    --arg dn "Carol" --arg di "$carol_id" \
    '{name:"Weekend Crew",members:[
      {displayName:$bn,userId:$bi},
      {displayName:$cn,userId:$ci},
      {displayName:$dn,userId:$di}
    ]}')")
  group_id=$(echo "$group" | jq -r '.group.id')
fi
echo "    Group: $group_id"

echo "==> Creating bills in group..."
existing_bills=$(post "$BILL/ListBillsByGroup" "$alice_token" "{\"groupId\":\"$group_id\"}")

if ! echo "$existing_bills" | jq -r '(.bills // []) | .[].title' | grep -q "Pizza Night"; then
  post "$BILL/CreateBill" "$alice_token" "$(jq -n \
    --arg gid "$group_id" \
    --arg ai "$alice_id" --arg bi "$bob_id" --arg ci "$carol_id" \
    '{
      title: "Pizza Night",
      total: 38.50,
      subtotal: 35.00,
      payerId: "Alice",
      groupId: $gid,
      items: [
        {description:"Margherita",   amount:14.00, participantIds:["Alice","Bob"]},
        {description:"Pepperoni",    amount:16.00, participantIds:["Alice","Carol"]},
        {description:"Garlic Bread", amount:5.00,  participantIds:["Alice","Bob","Carol"]}
      ],
      participants:[
        {displayName:"Alice", userId:$ai},
        {displayName:"Bob",   userId:$bi},
        {displayName:"Carol", userId:$ci}
      ]
    }')" > /dev/null
  echo "    Created: Pizza Night"
else
  echo "    Skipped: Pizza Night (already exists)"
fi

if ! echo "$existing_bills" | jq -r '(.bills // []) | .[].title' | grep -q "Groceries"; then
  post "$BILL/CreateBill" "$alice_token" "$(jq -n \
    --arg gid "$group_id" \
    --arg ai "$alice_id" --arg bi "$bob_id" --arg ci "$carol_id" \
    '{
      title: "Groceries",
      total: 60.00,
      subtotal: 60.00,
      payerId: "Alice",
      groupId: $gid,
      items: [
        {description:"Shared groceries", amount:60.00, participantIds:["Alice","Bob","Carol"]}
      ],
      participants:[
        {displayName:"Alice", userId:$ai},
        {displayName:"Bob",   userId:$bi},
        {displayName:"Carol", userId:$ci}
      ]
    }')" > /dev/null
  echo "    Created: Groceries"
else
  echo "    Skipped: Groceries (already exists)"
fi

echo ""
echo "Seed complete. Credentials: *@example.com / password123"
