# watcher

install LocalTerra:
git clone git@github.com:terra-money/LocalTerra.git
docker compose up

watcher directory:
Start the pulsar and pulsar manager containers
docker compose up -d

run the rpcwatcher:
go run cmd/rpcwatcher/main.go

new terminal:
python3 subscriber.py
