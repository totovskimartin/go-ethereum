#!/bin/sh

# Start geth in the background with the provided arguments
# First initialize with genesis block if data directory is empty
if [ ! -d "/app/data/geth" ]; then
  echo "Initializing genesis block..."
  geth --datadir /app/data init /app/genesis.json
else
  echo "Blockchain data already exists. Skipping initialization."
fi

# Then start geth with API options
geth --dev --http --http.addr=0.0.0.0 --http.api=eth,net,web3,txpool,debug,admin --dev.period 1 --datadir /app/data &
GETH_PID=$!

echo "Waiting for Geth RPC to be ready..."
until curl --silent --fail http://localhost:8545 > /dev/null; do
  sleep 1
done
echo "Geth RPC is available"

# Deploy contracts only if no previous deployments exist
if [ ! -f "/app/data/deployed.lock" ]; then
  echo "Deploying contracts..."
  cd /app/hardhat
  yes "y" | npx hardhat ignition deploy ignition/modules/Lock.js --network localhost
  touch /app/data/deployed.lock
else
  echo "Contracts already deployed. Skipping deployment."
fi

# Wait for the geth process to finish
wait $GETH_PID