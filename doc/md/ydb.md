---
id: ydb
title: YDB
---

## Overview

[YDB](https://ydb.tech/) is a distributed SQL database developed by Yandex. It provides horizontal scalability,
strong consistency, and automatic handling of transient errors. This page covers YDB-specific features and
considerations when using ent with YDB.

:::note
YDB support is currently in **preview** and requires the [Atlas migration engine](migrate.md#atlas-integration).
:::

## Opening a Connection

To connect to YDB, use the `ydb.Open()` function from the `entgo.io/ent/dialect/ydb` package:

```go
package main

import (
	"context"
	"log"

	"entdemo/ent"

	"entgo.io/ent/dialect/ydb"
)

func main() {
	// Open connection to YDB
	client, err := ent.Open("ydb", "grpc://localhost:2136/local")
	if err != nil {
		log.Fatalf("failed opening connection to ydb: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	// Run the auto migration tool
	if err := client.Schema.Create(ctx); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}
}
```

## Interactive & Non-Interactive Transactions

It is important to say that every request in YDB is executed in a transaction.
YDB supports two modes of working with transactions, and ent uses both depending on the API you choose.

### Non-interactive transactions

When you use the standard CRUD builders (`.Create()`, `.Query()`, `.Update()`, `.Delete()`), ent executes them through ydb-go-sdk's retry helpers:

- **Write operations** (Create, Update, Delete) go through
  [`retry.DoTx`](https://pkg.go.dev/github.com/ydb-platform/ydb-go-sdk/v3/retry#DoTx) — the SDK
  begins a transaction, executes the operation as a **callback**, commits, and on a transient error
  rolls back and re-executes the callback from scratch.
- **Read operations** (Query) go through
  [`retry.Do`](https://pkg.go.dev/github.com/ydb-platform/ydb-go-sdk/v3/retry#Do) — the SDK
  obtains a connection, executes the read callback, and retries on transient errors. No explicit
  transaction is created; the read runs with an implicit snapshot.

This is the recommended way to work with YDB through ent. Automatic retries and session management
are handled transparently.

### Interactive transactions

When you call `Client.BeginTx()`, ent opens a transaction via the standard `database/sql` API and **returns a `Tx` object** to the caller. You then perform operations on it and manually call
`Commit()` or `Rollback()`. In this model:

- There is **no callback** for the SDK to re-execute, so automatic retries are not possible.
- Session and transaction lifetime are managed by your code.

Use interactive transactions only when you need explicit control over commit/rollback boundaries
that can't be expressed through the standard builders.

## Automatic Retry Mechanism

Since YDB is a distributed database, it requires special handling for transient errors (network issues, temporary unavailability, etc.).
The ent YDB driver integrates with [ydb-go-sdk's retry package](https://pkg.go.dev/github.com/ydb-platform/ydb-go-sdk/v3/retry)
to automatically handle these scenarios.

:::note
However, ent does not use automatic retries when you create an interactive transaction using `Client.BeginTx()`. [Read more](ydb.md#no-automatic-retries-when-using-clientbegintx).
:::

### Using WithRetryOptions

All CRUD operations support the `WithRetryOptions()` method to configure retry behavior:

```go
import "github.com/ydb-platform/ydb-go-sdk/v3/retry"

// Create with retry options
user, err := client.User.Create().
	SetName("John").
	SetAge(30).
	WithRetryOptions(retry.WithIdempotent(true)).
	Save(ctx)

// Query with retry options
users, err := client.User.Query().
	Where(user.AgeGT(18)).
	WithRetryOptions(retry.WithIdempotent(true)).
	All(ctx)

// Update with retry options
affected, err := client.User.Update().
	Where(user.NameEQ("John")).
	SetAge(31).
	WithRetryOptions(retry.WithIdempotent(true)).
	Save(ctx)

// Delete with retry options
affected, err := client.User.Delete().
	Where(user.NameEQ("John")).
	WithRetryOptions(retry.WithIdempotent(true)).
	Exec(ctx)
```

### Retry Options

Common retry options from `ydb-go-sdk`:

| Option | Description |
|--------|-------------|
| `retry.WithIdempotent(true)` | Mark operation as idempotent, allowing retries on more error types |
| `retry.WithLabel(string)` | Add a label for debugging/tracing |
| `retry.WithTrace(trace.Retry)` | Enable retry tracing |

## Known Limitations

When using ent with YDB, be aware of the following limitations:

### No automatic retries when using Client.BeginTx

`Client.BeginTx()` returns a transaction object to the caller instead of accepting a callback,
so the retry mechanism from ydb-go-sdk cannot be applied. See
[Interactive transactions](ydb.md#interactive-transactions) for a detailed explanation.

If you still need interactive transactions, you may write a retry wrapper manually, as done in
[ydb-go-sdk's examples](https://github.com/ydb-platform/ydb-go-sdk/blob/master/examples/basic/database/sql/series.go).

### No Nested Transactions

YDB uses flat transactions and doesn't support nested transactions. The ent YDB driver
handles this by returning a no-op transaction when nested transactions are requested.

### No Correlated Subqueries

YDB doesn't support correlated subqueries with `EXISTS` or `NOT EXISTS`. Ent automatically
rewrites such queries to use `IN` with subqueries instead.

### Float/Double Index Restriction

YDB doesn't allow `Float` or `Double` types as index keys. If you define an index on a
float field, it will be skipped during migration.

### No Native Enum Types

YDB doesn't support enum types in DDL statements. Ent maps enum fields to `Utf8` (string)
type, with validation handled at the application level.

### Primary Key Requirements

YDB requires explicit primary keys for all tables. Make sure your ent schemas define
appropriate ID fields.

## Example Project

A complete working example is available in the ent repository:
[examples/ydb](https://github.com/ent/ent/tree/master/examples/ydb)

This example demonstrates:
- Opening a YDB connection
- Schema creation with migrations
- CRUD operations with retry options
- Edge traversals
