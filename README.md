# watcher

Install LocalTerra:

```sh
git clone git@github.com:terra-money/LocalTerra.git
cd LocalTerra
docker compose up
```


Clone the watcher repo and start the pulsar and pulsar-manager containers

```sh
git@github.com:tuky191/watcher.git
cd watcher
docker compose up -d
```

Optional:

Setup the pulsar-manager [pulsar-manager](https://github.com/apache/pulsar-manager) 

```sh
CSRF_TOKEN=$(curl http://backend-service:7750/pulsar-manager/csrf-token)
curl \
    -H "X-XSRF-TOKEN: $CSRF_TOKEN" \
    -H "Cookie: XSRF-TOKEN=$CSRF_TOKEN;" \
    -H 'Content-Type: application/json' \
    -X PUT http://backend-service:7750/pulsar-manager/users/superuser \
    -d '{"name": "admin", "password": "apachepulsar", "description": "test", "email": "username@test.org"}'
```

Run the rpcwatcher:
```sh
go run cmd/rpcwatcher/main.go
```

Open new terminal:

```sh
python3 subscriber.py
```


