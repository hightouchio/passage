#!/usr/bin/env ruby

require 'httparty'
require 'json'

PASSAGE_URL="http://passage:6000"

def post(path, body)
    HTTParty.post(
        File.join(PASSAGE_URL, path),
        headers: { 'Content-Type': 'application/json' },
        body: body.to_json
    )
end

def get(path)
    HTTParty.get(File.join(PASSAGE_URL, path))
end

def create_tunnel
    post('api/tunnel/normal', {
        "sshHost": "remote_bastion",
        "sshPort": 2222,
        "serviceHost": "remote_service",
        "servicePort": 5678,
        "createKeyPair": true
    })
end

def check_tunnel(tunnel_id)
    response = get("/api/tunnel/#{tunnel_id}/check")
    success = response['success']
    success == true
end

# Check Passage status.
puts "Healthcheck", get('healthcheck')

# Create tunnel.
response = create_tunnel()
tunnel_id = response['tunnel']['id']
public_key = response['publicKey']
connection = response['connection']
puts <<-EOF
Tunnel ID: #{tunnel_id}
Public Key: #{public_key}
Connection Details: #{connection}
EOF

# Write public key to shared volume for bastion server
File.write('/public-keys/passage-normal-tunnel.pub', public_key)

# Check tunnel status.
until check_tunnel(tunnel_id)
    puts "Checking tunnel status..."
    sleep 2
end
puts "Tunnel online."