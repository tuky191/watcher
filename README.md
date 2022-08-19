# watcher

Install LocalTerra:

```sh
git clone git@github.com:terra-money/LocalTerra.git
cd LocalTerra
docker compose up
```


Navigate to watcher's directory and start the pulsar and pulsar-manager containers

```sh
docker compose up -d
```

Run the rpcwatcher:
```sh
go run cmd/rpcwatcher/main.go
```

Open new terminal:

```sh
python3 subscriber.py
```


