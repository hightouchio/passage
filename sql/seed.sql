BEGIN;

INSERT INTO passage.reverse_tunnels (sshd_port)
VALUES
    (49500),
    (49501),
    (49502)
;

INSERT INTO passage.keys (key_type, contents)
VALUES
    ('public', 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQCwOxNoAjP6HDy0yvBz/ffj61TccwIyDcRHZeqSdMC5DS1WKDkksJBFUDDgjbuOzQ0bfX6J0DFwrUaKEQfdWeFldsSqMRP/4ZHiwV6+1/8nawGcYHhXS99ZkZqaLWEvk9BQkPUVYzuweH/VOdNvyHU3XnnFerkkRWdviE3boGIcIQ=='),
    ('public', 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQDgqCDxEfSZYd+9lJi+5qsa+pADWq4o6n5JQ8FLM/HBVGPO22zaet4PxmxiUqF3mK/PfSzLfGTR9h7I0mV23HnSc6+GmdQjjEzmEvmEGaH/Y2CYDdSlL8yZ4klSgZCbWT+Zgg+i0ggKuCqe4PJvwjXxocU670DC+Bow/7FdXUr9yQ=='),
    ('public', 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAAAgQDciH9/xcgV4LDVnz+g+NAapajo9tFWNKfgop7N3hOwDEiAk9+E2Z9CaxENA6dsIIBfqZ/ikIENqGkd8HkOI5HqbXqnTnxJKkTMQOVS420D8VC07+LFRCWR8ZxI+eDGs82OtAJ8AH3qp9xgp6jfAuv1I/Vw6CMbOsFeTvDSAsD/rQ==')
;

INSERT INTO passage.key_authorizations (key_id, tunnel_type, tunnel_id)
VALUES
    (1, 'reverse', 1),
    (2, 'reverse', 2),
    (2, 'reverse', 3),
    (3, 'reverse', 3)
;

COMMIT;