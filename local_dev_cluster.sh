#!/bin/bash

# $1 is cluster name

# kubectl doesn't properly export the ca data below, so hard coding here

#CA_DATA_RAW=$(kubectl config view -o jsonpath='{.clusters[?(@.name == "'$1'")].cluster.certificate-authority-data}')
CA_DATA_RAW='LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURKRENDQWd5Z0F3SUJBZ0lVVERNQ1VRUU9BMWhraHU3bG1VSTkyREZmSWlnd0RRWUpLb1pJaHZjTkFRRUwKQlFBd0RURUxNQWtHQTFVRUF4TUNZMkV3SGhjTk1UZ3dOek14TWpBd056UTNXaGNOTVRrd056TXhNakF3TnpRMwpXakFOTVFzd0NRWURWUVFERXdKallUQ0NBU0l3RFFZSktvWklodmNOQVFFQkJRQURnZ0VQQURDQ0FRb0NnZ0VCCkFKcUNzaEVFZHIyZ2ZQYVZEejlhckZyUG1MV3V2U2JGTllLUXR4dTBkTmRMQXBQbkE4UUhBenBDdzc4RVcxR0oKRmY3M0pFTXE3MTJhaDZBRWFmcjBQRkhGN0ZqQXNVaTlqVDhEWi9OWm5FVFMrSXFHK2RUMFh4Tm0rQTkyME9VRApMazZnMmNzVnczb0JCbGVQWmJiM3dUQ1FpMWYvb1RDK0NKQ3RSdVZ3M1lTaVVBNzZWZmkxR0hiWC9LV0dMQ1FyCkEwYVo3ajhiMDBOYTBsT2ZPejBkc2Vtek9CZkJoMUFsb3RrRXhPVUZhMksyWDdYY1NjUjBTSVhEeFNlbSswUGwKbUhKWUlTQitja1k2eS8rMUJwUkdzVHpLTHZ2SkdvSUtRSWZPZ3dYZ3pRSDNvVG02R3hoYmlnNE8rVXdLUzd2Wgp1bXNIMkI1ZWNuNTBMaUlBVjc4dWlSc0NBd0VBQWFOOE1Ib3dIUVlEVlIwT0JCWUVGR0ZJMUJ4UmNkZkpTU21GCngrRWd3ckJGN0xWTU1FZ0dBMVVkSXdSQk1EK0FGR0ZJMUJ4UmNkZkpTU21GeCtFZ3dyQkY3TFZNb1JHa0R6QU4KTVFzd0NRWURWUVFERXdKallZSVVURE1DVVFRT0ExaGtodTdsbVVJOTJERmZJaWd3RHdZRFZSMFRBUUgvQkFVdwpBd0VCL3pBTkJna3Foa2lHOXcwQkFRc0ZBQU9DQVFFQUVydkk0ak1uYXdNekZTcmlTVWdpN0YwemNtdk44bWF2Ck52cWo2dVF4ckc5R0dTaE8wUkZ4cjNCdFJRemk2QzRSb21JckI2eGNrRGt4YTJUdWJwRXJEV3ltOXBKRGRIaU0KdWRyZWdVaTZMekQxaVhXbWNKN2FJcEN5dU1rTDdRSnpiV1FWVjJrbVcyTGYyMVIwZzdUbVNDQ0dVWnNoYUxiZwo0SHREZGpDY3JyZ2RhUkFxRUNrVXArVDdKSkdrUUdmS3k5UmdIdktsNUtGMXZ0NkJhdmxueTZOYmVmZnYyOEFWCnhwaCtPUnM1U0plRUsremRjNmNQai9sL2dSbkdVTC92dnhrTFgra3FCOVEyK1NkM0ErSHpKM0cwRzFVbU9LUlgKQTRlY0t5WkkvY1NzYysrY3RiNkE2bmNDUHFIbHphZkNVWVdITXQrdHZJU2xweXJaM2pCZFh3PT0KLS0tLS1FTkQgQ0VSVElGSUNBVEUtLS0tLQo='
export SERVER=$(kubectl config view -o jsonpath='{.clusters[?(@.name == "'$1'")].cluster.server}')

USER_ID=$(kubectl config view -o jsonpath='{.contexts[?(@.context.cluster == "'$1'")].context.user}')
 
secret_val=$(kubectl config view -o jsonpath='{.users[?(@.name == "'$USER_ID'")].user.token}')

export TOKEN=$secret_val

export CA_DATA=$(echo $CA_DATA_RAW | base64 -D)

export SECURITY_USER_NAME=admin
export SECURITY_USER_PASSWORD=pass

LDFLAGS="-X github.com/cf-platform-eng/kibosh/pkg/helm.tillerTag=$(cat tiller-version)"

go run -ldflags "${LDFLAGS}" cmd/kibosh/main.go