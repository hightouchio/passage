#!/usr/bin/env ruby

require 'httparty'
require 'json'

PASSAGE_URL="http://passage:8080"

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
    post('api/tunnel/standard', {
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
File.write('/bastion-ssh/authorized_keys', public_key)

# Check tunnel status.
MAX_ATTEMPTS = 5
attempts = 0
until check_tunnel(tunnel_id) do
    attempts += 1
    if attempts >= MAX_ATTEMPTS
        puts 'Tunnel did not come online.'
        exit 1
    end

    puts "Checking tunnel status..."
    sleep 2
end
puts "Tunnel online."

# Get data from tunnel.
response = HTTParty.get("http://#{connection['host']}:#{connection['port']}/")
response_body = response.body&.strip
if response_body != ENV['EXPECTED_SERVICE_RESPONSE']
    puts <<-EOF
ERROR: Unexpected remote service response.

Body: #{response.body}
Length: #{response.body.length}
EOF
    exit 1
end

puts "Received expected external service response. Success!"