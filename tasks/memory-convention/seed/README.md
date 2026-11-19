# payments

A small in-memory payments service: customers, payments (authorize, capture, void).

    go build ./...
    go test ./...
    PORT=8080 go run .

## Layout

    main.go                 entry point
    internal/api/           HTTP handlers and routing
    internal/apperr/        error codes returned to clients
    internal/store/         in-memory storage
    internal/xlog/          structured logging
    docs/                   design notes, error code reference, contributing notes

## API

    POST /customers              {"name","email"}                      -> 201 customer
    GET  /customers/{id}
    POST /payments               {"customer_id","amount_cents","currency"} -> 201 payment (status "authorized")
    GET  /payments/{id}
    POST /payments/{id}/capture  authorized -> captured
    POST /payments/{id}/void     authorized -> voided
    GET  /healthz
    GET  /debug/routes

Errors are JSON: `{"error":{"code":"...","message":"..."}}`.
