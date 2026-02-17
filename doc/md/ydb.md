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

## Automatic Retry Mechanism

YDB requires special handling for transient errors (network issues, temporary unavailability, etc.).
The ent YDB driver integrates with [ydb-go-sdk's retry package](https://pkg.go.dev/github.com/ydb-platform/ydb-go-sdk/v3/retry)
to automatically handle these scenarios.

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

## YDB-Specific SQL Features

The ent SQL builder supports several YDB-specific SQL constructs that are used internally
or can be leveraged for advanced use cases.

### UPSERT and REPLACE

YDB doesn't support the standard `ON CONFLICT` clause. Instead, it uses `UPSERT` and `REPLACE` statements:

- **UPSERT**: Inserts a new row or updates an existing row if a conflict occurs
- **REPLACE**: Replaces the entire row if a conflict occurs

Ent automatically uses `UPSERT` when you configure `OnConflict` options in your create operations.

### BATCH Operations

For large tables, YDB provides `BATCH UPDATE` and `BATCH DELETE` statements that process data
in batches, minimizing lock invalidation risk. These are used internally by ent when appropriate.

### Special JOIN Types

YDB supports additional JOIN types beyond the standard SQL joins:

| Join Type | Description |
|-----------|-------------|
| `LEFT SEMI JOIN` | Returns rows from left table that have matches in right table |
| `RIGHT SEMI JOIN` | Returns rows from right table that have matches in left table |
| `LEFT ONLY JOIN` | Returns rows from left table that have no matches in right table |
| `RIGHT ONLY JOIN` | Returns rows from right table that have no matches in left table |
| `EXCLUSION JOIN` | Returns rows that don't have matches in either table |

### VIEW Clause for Secondary Indexes

YDB allows explicit use of secondary indexes via the `VIEW` clause. This is an optimization
hint that tells YDB to use a specific index for the query.

### Named Parameters

YDB uses named parameters with the `$paramName` syntax (e.g., `$p1`, `$p2`) instead of
positional placeholders. Ent handles this automatically.

## Known Limitations

When using ent with YDB, be aware of the following limitations:

### No Correlated Subqueries

YDB doesn't support correlated subqueries with `EXISTS` or `NOT EXISTS`. Ent automatically
rewrites such queries to use `IN` with subqueries instead.

### No Nested Transactions

YDB uses flat transactions and doesn't support nested transactions. The ent YDB driver
handles this by returning a no-op transaction when nested transactions are requested.

### Float/Double Index Restriction

YDB doesn't allow `Float` or `Double` types as index keys. If you define an index on a
float field, it will be skipped during migration.

### No Native Enum Types

YDB doesn't support enum types in DDL statements. Ent maps enum fields to `Utf8` (string)
type, with validation handled at the application level.

### RowsAffected Behavior

YDB's `RowsAffected()` doesn't return accurate counts. The ent driver uses the `RETURNING`
clause internally to count affected rows when needed.

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
