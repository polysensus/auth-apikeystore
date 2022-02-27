# apikeystore
A service for creating and managing service api keys. Firestore specific.

# Plan
## General
* Follow resource-oriented design from here https://google.aip.dev/121 
* Persistence in firebase
* go lang with protobuf. same service hosts grpc & HTTP endpoints
* Seperate services for reading vs creating & deleting API keys (same repo for both)
    * reader to handle list, get and check
    * writer to handle create and delete
    * Each service gets its own workload identity bound service identity with the appropriate
* terraform doesn't have great support for firebase firestore database management so some tooling to do that will be needed. python / tusk /task etc

## Secret generation
* Use [argon2id](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html#argon2id). As per "other applications" on [rfc2898](https://www.ietf.org/rfc/rfc2898.txt) [go implementation](https://pkg.go.dev/golang.org/x/crypto/argon2)
* Or (if FIPS-140 is required) use [pkkdf2](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html#pbkdf2). As per "[go implementation](https://pkg.go.dev/golang.org/x/crypto/pbkdf2)

Recomendations taken from [here](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html

## API Key Schema

api_key_collection
* name: default nano id 11 base64 encoded
* secret_hash: password hash[1]
* user_id: foreighn key to the firebase identity for the user that owns the key
* applications: empty or platform defined strings restricting which platform applications can redeem the key.
* audience: defines how the audience for any generated token is created[2]
* scopes: defines the scopes requested for any access token obtained using the key[3]

api_key_usage_collection[3]
rate_cap: 
last_used:
access_count_indication: a not perfect count 

1. password is nanoid 21. The result of [secret generation](#Secret generation) is stored (it's a hash that is stored, plus possibly salt and iteration count)
2. audience can be a fixed string or a glob pattern. the interpretation of both is down to the consuming application. audience example for nodes network means automatically chose the audience for the node the user happens to be routed to. ethnode{N} means audience is set to a specific node
3. scopes example for nodes: "net_* eth_* admin_nodeInfo"
4. if/when rate use and cleanup of unused keys is implemented the state goes in a separate collection so the 'reader' can be granted access without giving access to update the keys
