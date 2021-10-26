#!/usr/bin/env bash

PASSAGE_URL="http://passage:6000"

check_tunnel() {
  tunnel_id=$1

  http_response=$(curl $PASSAGE_URL/api/tunnel/${tunnel_id}/check)
  success=$(echo $http_response | jq -r ".success")

  echo "HTTP Response: ${http_response}"
  echo "Success: ${success}"

  if [ "$success" == "true" ]; then
    return 0
  else
    return 1
  fi
}

curl -o create_response.json $PASSAGE_URL/api/tunnel/normal \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  --data @<(cat <<EOF
{
  "sshHost": "remote_bastion",
  "sshPort": 2222,
  "serviceHost": "remote_service",
  "servicePort": 5678,
  "createKeyPair": true
}
EOF
  ) >/dev/null

# Extract some useful stuff
tunnel_id=$(cat "create_response.json" | jq -r ".tunnel.id")
public_key=$(cat "create_response.json" | jq -r ".publicKey")
connection=$(cat "create_response.json" | jq -r ".connection")

cat <<EOF
Tunnel ID: ${tunnel_id}
Public Key: ${public_key}
Connection Details: ${connection}
EOF

# Wait for tunnel to connect.
until check_tunnel "$tunnel_id"
do
  echo "Checking tunnel status..."
  sleep 2
done
echo "Tunnel is online."