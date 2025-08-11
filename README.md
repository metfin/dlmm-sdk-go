# DLMM SDK for Go

Golang SDK for Meteora's Dynamic Liquidity Market Maker (DLMM), designed for correctness, ergonomics, and performance.

This SDK is built around IDL-driven generation: accounts, types, and instruction builders are generated from the Anchor IDL, while a thin hand-written layer provides a clean, idiomatic public API.

## Features (initial scope)

- IDL-driven generated types, accounts, and instruction builders
- Thin high-level client for common DLMM actions (fetch pool, swap, add/remove liquidity)
- Deterministic Borsh encoding/decoding for account/layout fidelity
- Minimal dependencies; uses `solana-go` for RPC/tx/building

## Installation

```bash
go get github.com/metfin/dlmm-sdk-go
```

## Requirements

- Go 1.21+
- A Solana RPC endpoint
- Anchor IDL for the DLMM program (JSON)

## Project structure

```
.
├── idl/
│   └── dlmm.json           # Anchor IDL (checked in)
├── pkg/
│   ├── dlmm/               # Hand-written public API (client, helpers)
│   └── generated/            # Generated code from IDL (do not edit)
├── examples/
│   └── basic/              # Minimal usage examples
├── cmd/
│   └── cli/                # Optional demo CLI (future)
├── go.mod
├── LICENSE
└── README.md
```

- `idl/`: Source-of-truth for generated code. Regenerate when the IDL changes.
- `pkg/generated/`: Generated accounts, types, and instruction builders (overwritten on regen).
- `pkg/dlmm/`: Stable, idiomatic wrappers and helpers that call into `generated`.

## Quickstart

```go
package main

import (
    "context"
    "log"

    solanarpc "github.com/gagliardetto/solana-go/rpc"
    "github.com/gagliardetto/solana-go"
    "github.com/metfin/dlmm-sdk-go/pkg/dlmm"
)

func main() {
    ctx := context.Background()

    rpc := solanarpc.New("https://api.mainnet-beta.solana.com")

    // Initialize DLMM client (thin wrapper around generated code)
    client := dlmm.NewClient(rpc, dlmm.WithProgramID(solana.MustPublicKeyFromBase58("<DLMM_PROGRAM_ID>")))

    poolPubkey := solana.MustPublicKeyFromBase58("<POOL_PUBKEY>")
    pool, err := client.GetPool(ctx, poolPubkey)
    if err != nil {
        log.Fatalf("get pool: %v", err)
    }
    log.Printf("pool: %+v", pool)
}
```

Note: The high-level methods (`GetPool`, `BuildSwap`, etc.) are thin wrappers; they call into the generated `pkg/generated` package for instruction data/layouts.

## Roadmap

- v0.1 (Scaffold)

  - Project layout, lint/setup, examples
  - IDL-driven generation integrated (`pkg/generated`)
  - Public client skeleton in `pkg/dlmm` with configuration

- v0.2 (Core decoding + queries)

  - Implement account fetch + decode for key DLMM accounts (pool, position, config)
  - Helpers for PDAs and common derivations
  - Error mapping from on-chain error codes

- v0.3 (Instruction builders)

  - High-level builders for swap, add/remove liquidity, create position
  - Transaction assembly helpers

- v0.4 (Testing and docs)

  - Unit tests for encoding/decoding and builders
  - Example coverage for common user flows
  - Extended README and godoc

- v0.5 (Stabilization)
  - API review and refinements
  - Versioned releases and changelog

## Generate the IDL

```
# Install anchor-go
go install github.com/gagliardetto/anchor-go@latest


# Generate into pkg/generated with Go package name 'generated'
anchor-go --idl ./idl/dlmm.json --output ./pkg/generated --program-id LBUZKhRxPF3XUpBCjp4YzTKgLccjZhTSDM9YuVaPwxo
```

## Contributing

- Keep generated code isolated in `pkg/generated/`; do not modify it by hand.
- Place stable abstractions, helpers, and end-user APIs in `pkg/dlmm/`.
- Include tests for any new functionality. Prefer small, isolated units.
- Run `go fmt`, `go vet`, and ensure CI is green.

## License

GNU General Public License v3. See `LICENSE`.
