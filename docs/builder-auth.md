# Builder Authentication

Relayer requests require builder authentication headers for attribution and leaderboard/grants credit.
Official references: [Order Attribution](https://docs.polymarket.com/developers/builders/order-attribution), [Relayer Client](https://docs.polymarket.com/developers/builders/relayer-client).

## Required Headers
- `POLY_BUILDER_API_KEY`
- `POLY_BUILDER_PASSPHRASE`
- `POLY_BUILDER_SIGNATURE`
- `POLY_BUILDER_TIMESTAMP`

## Signature Payload
The signature is computed over the string:

```
<timestamp><method><path><body>
```

Notes:
- `timestamp` is Unix **milliseconds** (e.g., `Date.now()`).
- `method` should be uppercase (e.g., `POST`).
- `path` is the request path **including query string** if present (e.g., `/deployed?address=0x...`).
- `body` is the raw JSON payload string (do not re-marshal).

The signature is `HMAC-SHA256` with your Builder Secret, base64url-encoded.

## Local Signing (Go)
```go
body := `{"foo":"bar"}`
headers, err := builderCfg.Headers(ctx, "POST", "/submit", &body, 0)
if err != nil {
    panic(err)
}
```

## Remote Signer Contract
**Request** (sent by the SDK):
```json
{
  "method": "POST",
  "path": "/submit",
  "body": "{...}"
}
```

**Response** (from signer service):
```json
{
  "POLY_BUILDER_API_KEY": "...",
  "POLY_BUILDER_PASSPHRASE": "...",
  "POLY_BUILDER_SIGNATURE": "...",
  "POLY_BUILDER_TIMESTAMP": "1700000000000"
}
```

If your signer expects authentication, use a bearer token and validate the
`Authorization: Bearer <token>` header.

## Debug (Redacted)
```go
mask := func(s string) string {
    if len(s) <= 8 {
        return "********"
    }
    return s[:4] + "..." + s[len(s)-4:]
}

for _, k := range []string{
    relayer.HeaderPolyBuilderAPIKey,
    relayer.HeaderPolyBuilderPassphrase,
    relayer.HeaderPolyBuilderTimestamp,
    relayer.HeaderPolyBuilderSignature,
} {
    fmt.Printf("%s: %s\n", k, mask(headers.Get(k)))
}
```
