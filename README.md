Hyperledger Fabric

The assignment contains: Level-1 instructions, Level-2 chaincode in Go, Level-3 REST API in Go + Dockerfile.

-> In this we have 3 levels
level-1-test-network/` - instructions to start Fabric test network (uses fabric-samples/test-network)
level-2-chaincode/accountcc/` - Go chaincode implementing account asset with attributes:
DEALERID, MSISDN, MPIN, BALANCE, STATUS, TRANSAMOUNT, TRANSTYPE, REMARKS`
level-2-chaincode/deploy_chaincode.sh` - example script showing package/install/approve/commit commands (edit package id and paths for your environment)
level-3-rest-api/` - Go REST API that calls the chaincode via Fabric Gateway client and a Dockerfile to containerize it


-> Level-1: Test network (instructions)
1. Clone fabric-samples and bring up the test network.
2. Ensure environment variables and docker images are in the right versions as used by binaries.

-> Level-2: Chaincode (how to deploy)
1. we have to edit level-2-chaincode/accountcc/main.go  to change logic/attributes.
2. Build/package (example):
   
   -> from level-2-chaincode directory
   -> queryinstalled to get package ID
   -> approve, checkcommitreadiness, commit etc. See deploy_chaincode.sh.

-> Level-3: REST API
1. In level-3-rest-api/main.go to point certificate, key, peer endpoint and gateway connection details for the network.
2. Build and run:
   rest-api
   go mod tidy
   go run main.go

3. Or build Docker image and run (update envs & volumes):
   
   docker build -t fabric-rest-api.
   The REST service will connect to the gateway and expose endpoints on port 8080.

-> I've completed.
- Implemented chaincode methods:
  - InitLedger
  - CreateAccount
  - QueryAccount
  - GetAccountHistory
  - UpdateAccountBalance
  - AccountExists
- Provided a REST API:
  - POST /accounts -> CreateAccount
  - GET  /accounts/{id} -> QueryAccount
  - PUT  /accounts/{id}/balance -> UpdateAccountBalance
  - GET  /accounts/{id}/history -> GetAccountHistory
- Dockerfile to containerize the REST API.
