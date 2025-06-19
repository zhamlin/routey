# Routey
Go library enabling declarative HTTP handlers with OpenAPI Support.

Chi like routing interface supporting any `net/http` compatible router, by default uses `http.ServeMux`.

## Features
- `net/http` compatible
- Incremental adoption
  - Supports any router that implements the `routey.Mux` interface
  - OpenAPI docs can be added to normal `net/http` Handlers
- Declarative HTTP handlers
  - Request Body
  - Request Parameters: Path, Query, Cookie, Header
  - Responses
- Useful errors
  - Catch common mistakes early
  - Detailed errors make it easy to see what is wrong
- Minimal overhead
- Minimal dependencies
- Code first OpenAPI 3.1 support
  - OpenAPI spec generated from router
  - JSON Schema from Go types
  - Automatic request validation

## Inspiration
- https://github.com/tokio-rs/axum
- https://github.com/fastapi/fastapi
- https://github.com/danielgtaylor/huma
- https://github.com/go-chi/chi
