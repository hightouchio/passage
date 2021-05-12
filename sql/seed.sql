-- set up reverse tunnels
SET search_path='passage';

BEGIN;
INSERT INTO reverse_tunnels (sshd_port) VALUES
    (49500), (49501), (49502)
;
INSERT INTO keys (key_type, contents) VALUES
    ('public', 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQCwOxNoAjP6HDy0yvBz/ffj61TccwIyDcRHZeqSdMC5DS1WKDkksJBFUDDgjbuOzQ0bfX6J0DFwrUaKEQfdWeFldsSqMRP/4ZHiwV6+1/8nawGcYHhXS99ZkZqaLWEvk9BQkPUVYzuweH/VOdNvyHU3XnnFerkkRWdviE3boGIcIQ=='),
    ('public', 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQDgqCDxEfSZYd+9lJi+5qsa+pADWq4o6n5JQ8FLM/HBVGPO22zaet4PxmxiUqF3mK/PfSzLfGTR9h7I0mV23HnSc6+GmdQjjEzmEvmEGaH/Y2CYDdSlL8yZ4klSgZCbWT+Zgg+i0ggKuCqe4PJvwjXxocU670DC+Bow/7FdXUr9yQ=='),
    ('public', 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQDciH9/xcgV4LDVnz+g+NAapajo9tFWNKfgop7N3hOwDEiAk9+E2Z9CaxENA6dsIIBfqZ/ikIENqGkd8HkOI5HqbXqnTnxJKkTMQOVS420D8VC07+LFRCWR8ZxI+eDGs82OtAJ8AH3qp9xgp6jfAuv1I/Vw6CMbOsFeTvDSAsD/rQ==')
;
INSERT INTO passage.key_authorizations (key_id, tunnel_type, tunnel_id) VALUES
    (1, 'reverse', (SELECT id FROM passage.reverse_tunnels WHERE sshd_port=49500)),
    (2, 'reverse', (SELECT id FROM passage.reverse_tunnels WHERE sshd_port=49501)),
    (2, 'reverse', (SELECT id FROM passage.reverse_tunnels WHERE sshd_port=49502)),
    (3, 'reverse', (SELECT id FROM passage.reverse_tunnels WHERE sshd_port=49502))
;
COMMIT;

-- set up normal tunnels
BEGIN;
INSERT INTO tunnels (ssh_hostname, ssh_port, service_hostname, service_port) VALUES
    ('localhost', '23456', 'localhost', '3000')
;
INSERT INTO keys (key_type, contents) VALUES
    ('private', convert_from(decode('LS0tLS1CRUdJTiBPUEVOU1NIIFBSSVZBVEUgS0VZLS0tLS0KYjNCbGJuTnphQzFyWlhrdGRqRUFBQUFBQkc1dmJtVUFBQUFFYm05dVpRQUFBQUFBQUFBQkFBQUJsd0FBQUFkemMyZ3RjbgpOaEFBQUFBd0VBQVFBQUFZRUFzRmJzcTZmM3pVa3FpZmc0a2Vzd1BZZGRubFdicm02Y0JPT2U4YmUyVFlENk9KdWZaWHFkCkgyWnM5Y3pLUVB6WGpLSUxKWk9Wc0NxQWduOXJONFN4TDBIWnZVMHh6bng5bkpUeWVJdzRjZFBBaCtMM25FbEtpd1BpREcKNUlZbkJyWlVSWkdZL0ZmWmsyRkdpWC9hQXBlbEVESEZhOXBPakRQR05Wa0RuRFd4eE41QUF4UVpRNmlLSXdDL0R3QWwwUApBWDlNc0RvRWxjcUlycHh5c25GUEVNRXF1UGNZSTU2dmlUMUNWQjlwOEFJNHpMbStRV1VLS2hHTXBLelppWXRjWmh5UFB6CjlYejRQVUhxUjNYOHRyN2E0WUk0ZC9zVGp5cmRhdmlWVmd2dDlIZGFMazZpTHE2OHQyS1l2emlIUGQ5R0RYUk1SajBYU2kKdlhBQ2YvOG5VeVN4ekd5blNMb3htRTdraitMeDJraHZKWUNBSzIvQmZCVUs0a1liKzRzdzA0elB3dUpGZ2FyK0QvUGxjWApsUkxMSTFRZGRGblFEMWNvcHZrV252R0lKeVdSeWQ3elVoVDNNY3RwUjhQQTA1cm9IcGE1VEQ5cVhQMEcreklqWVhmYmZhClVoaWlqVEdUVDM3T282aFBWRUdYR1JJZUNaYmdtenRpVHo3bkFrcjVBQUFGb0Z0L2ZoeGJmMzRjQUFBQUIzTnphQzF5YzIKRUFBQUdCQUxCVzdLdW45ODFKS29uNE9KSHJNRDJIWFo1Vm02NXVuQVRqbnZHM3RrMkEramlibjJWNm5SOW1iUFhNeWtEOAoxNHlpQ3lXVGxiQXFnSUovYXplRXNTOUIyYjFOTWM1OGZaeVU4bmlNT0hIVHdJZmk5NXhKU29zRDRneHVTR0p3YTJWRVdSCm1QeFgyWk5oUm9sLzJnS1hwUkF4eFd2YVRvd3p4alZaQTV3MXNjVGVRQU1VR1VPb2lpTUF2dzhBSmREd0YvVExBNkJKWEsKaUs2Y2NySnhUeERCS3JqM0dDT2VyNGs5UWxRZmFmQUNPTXk1dmtGbENpb1JqS1NzMlltTFhHWWNqejgvVjgrRDFCNmtkMQovTGErMnVHQ09IZjdFNDhxM1dyNGxWWUw3ZlIzV2k1T29pNnV2TGRpbUw4NGh6M2ZSZzEwVEVZOUYwb3Ixd0FuLy9KMU1rCnNjeHNwMGk2TVpoTzVJL2k4ZHBJYnlXQWdDdHZ3WHdWQ3VKR0cvdUxNTk9NejhMaVJZR3EvZy96NVhGNVVTeXlOVUhYUloKMEE5WEtLYjVGcDd4aUNjbGtjbmU4MUlVOXpITGFVZkR3Tk9hNkI2V3VVdy9hbHo5QnZzeUkyRjMyMzJsSVlvbzB4azA5Kwp6cU9vVDFSQmx4a1NIZ21XNEpzN1lrOCs1d0pLK1FBQUFBTUJBQUVBQUFHQVZ0cjV4N0w4QTBrK2dxYVlkb2I1UTlVZzc1ClFMRW1lNHdVKzhRaUxTMEdudTFXTVJwOUkxQWZwWlFOaVY1bHlqYnNtQjVBaEZlbmYweEZJdVpUSWVjOEJqbHpJbWppWVQKV01Ta2lXdkhnWWxqRTR0Zm1zTWV6RW9sVlNvY3pSL0NSaWJjSEJNTW0waVRzb215RTJLSUM0OUxrUEhJaHlsK3dJZk01VApVT1hCb2M0MmlVMmxCQVpIdytYaU5XbEZOeHlMTUMrdW9ZeUE3eU04OHJUMmt3SUhpRXFvWktoWURyeFJVL1RtQUt0Z1RpCkpUUjMyWTRDNjZXcE9KTktzc3pJN0JSdmphS0FoZmp5TnE1SnhGUDNaMVZra2duU1VvYi91S1FaN0V4SGM0ZmUxeWlHTlQKZUhIUzdPNzBGTWIwcWthN3VWejBHSjFuek9TOHF0aXVieXFuTXlnVlFQNE1haCszQ0VqSGo3VUp0OXNvYTBEMUdYRDY3ego2K1hmTmFQYXdiK242SVlCN2VrbFVqVnVwdXduNEtxckkwQWhHQjBiWms3VGhpTlMwZFFXRWFKVzlHWGZwZEFyRVgrVkVBCmNYZlh3dUhuODhQdzF6aDVCVDFNTmR6c2hQQjBYTWJWUlJMWW0xaHJKcnFkZ3hYVWdQV1FKTGZaWXh6SjNWOEYrQkFBQUEKd0MvNU1FSXNjaE05ZUNQM0tiRjF3WHppSUswY1lTaDlIYm95aDNrWGNjRis0WW4wQ0JFV1JDaW9tNU9YalQyWlhtWTFRUwpBTnFHcFpHME55VXVjRkZyMDNBVmp3WjdHOGU5Nmd5QVRQaXNtYk9xdUpDdFQyZzRhV0ZpV3pwQS9nSytXRnJTV05lL0llCkR6YzREcFNPa09kazJ2OGNkUW1JNzBBb2w3TXhhaCt3eUNiOE9rclZncUJtbTh0Z0RmeWFYUlJ1VElsMlBLUExYUlZaOEUKUnZudGVHRzBsL3g5M2NqN0xiQkpEV0FqMVU0TnBxdlVVZVBqOFJ2RnJwSmlNZDZnQUFBTUVBNXovM0oyMHp5bVEzVXp4VgpWQnkxbmVWME1RUkw4TzZQVC9rbVE5UWVFT1lFTVZ4N2tLSXA3YW1rSVcwQzFJczBXWTZjVHRhTVBCTU9qaHRKTDh0K1FZCjlhNFZ0YW9pSERka1ZZWGx2VnVxUDN0M2hYL1pDdW1pYk5LdlJsTFZQZGk2R1Y2K0JJTjIyaFpnQlN4aGxqT3N5VThkcmMKeGRYYzJzWGhIM0NzWThBdU5uWU9QM0pEU2N2aWZ1clVjRFZhRmdQTys5elNuYVZhS3VMWUQzMGd1ZzJQay9CUGxNM0QwMAp1ZkFiNDZBMHVicjZXUW4yZWZCQnhIV0xqNGIrYzlBQUFBd1FERE5uZDNTZ0N0YmxpWUk2d2s4NVZyeWRDZDlBazFaamo5CkJLbE8ySGVvVlJ5ekxvT0J2clNGS01SSGtGbXpCeVpwNTduVDdZUW5RR3cxdXkwL0NUd3ZyRlNwYWR2ZThqOEtXai9rcHYKWDdDMUwyK1hMdTAybXFQZVl6K1YvTGRyUWQ1ak5kK0xUK3g0VHN2Syt4VVBkTDU3Q1NJZyt5MUNZVVd4aTNNMzdQbXMzawphSEd2NmNwVEx5N0l2WFl2ZTNBbWMveW8xd0pORDRuNUExZ0U1ZHpNWDBFSi85RWpKVlJDc2kycGwzV1hleUNjUzZhNGhxClhGUzhaKzNuc05qbTBBQUFBa1lXNWtjbVYzY0dGblpVQkJibVJ5WlhkekxVMWhZMEp2YjJzdFVISnZMbXh2WTJGc0FRSUQKQkFVR0J3PT0KLS0tLS1FTkQgT1BFTlNTSCBQUklWQVRFIEtFWS0tLS0tCg==', 'base64'), 'UTF8'))
;
INSERT INTO key_authorizations (key_id, tunnel_type, tunnel_id) VALUES
    (1, 'normal', (SELECT id FROM passage.tunnels WHERE ssh_port=23456))
;
COMMIT;