#!/usr/bin/env ruby

require 'json'
require 'fileutils'

def wait_for_file(path)
    start_time = Time.now
    while !File.file?(path) do
#         if Time.now - start_time > 15
#             raise "Timeout waiting for #{path}"
#         end
        sleep 0.25
    end

    puts "Found #{path}"
end

private_key_path = '/reverse_tunnel/private_key'
wait_for_file(private_key_path)
wait_for_file('/reverse_tunnel/connection_details')
connection_details = JSON.parse(File.read('/reverse_tunnel/connection_details'))

passage_host = 'passage'
tunnel_port = connection_details['tunnel_port']
sshd_port = connection_details['sshd_port']
service_host = 'remote_service'
service_port = 5678

FileUtils.chmod(0400, private_key_path)
exec("ssh -v -i #{private_key_path} -o 'StrictHostKeyChecking=no' -R 0.0.0.0:#{tunnel_port}:#{service_host}:#{service_port} #{passage_host} -p #{sshd_port} -o ExitOnForwardFailure=yes")