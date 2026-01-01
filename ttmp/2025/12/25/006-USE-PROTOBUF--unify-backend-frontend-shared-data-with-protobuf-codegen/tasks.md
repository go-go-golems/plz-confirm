# Tasks

## TODO

- [x] Step 1: Create protobuf definitions (request.proto, widgets.proto, image.proto) and set up build pipeline (buf/Makefile, CI/CD integration)
- [x] Step 2: Migrate server to use protobuf types (replace internal/types imports, update server.go/ws.go/store.go, use protojson for JSON serialization)
- [x] Step 3: Migrate CLI to use protobuf types (replace internal/types imports, update all CLI commands, remove double marshal/unmarshal, update client.go)
- [x] Step 4: Migrate frontend to use protobuf types (replace types/schemas.ts imports, update all components/store/websocket client, delete schemas.ts)
- [x] Step 5: Cleanup (delete internal/types/types.go, update documentation, verify all tests pass)
