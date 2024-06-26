// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information

package testplanet

import (
	"context"
	"runtime/pprof"
	"testing"

	"go.uber.org/zap"

	"storj.io/common/testcontext"
	"storj.io/storj/private/testmonkit"
	"storj.io/storj/satellite/satellitedb/satellitedbtest"
	"storj.io/storj/shared/dbutil/pgtest"
	"storj.io/storj/shared/dbutil/pgutil"
)

// Run runs testplanet in multiple configurations.
func Run(t *testing.T, config Config, test func(t *testing.T, ctx *testcontext.Context, planet *Planet)) {
	databases := satellitedbtest.Databases()
	if len(databases) == 0 {
		t.Fatal("Databases flag missing, set at least one:\n" +
			"-postgres-test-db=" + pgtest.DefaultPostgres + "\n" +
			"-cockroach-test-db=" + pgtest.DefaultCockroach + "\n" +
			"-spanner-test-db=" + pgtest.DefaultSpanner)
	}

	for _, satelliteDB := range databases {
		satelliteDB := satelliteDB
		// TODO(spanner): remove this check once full Spanner support is complete
		if !config.EnableSpanner && satelliteDB.Name == "Spanner" {
			t.Skipf("Test is not enabled to run on Spanner.")
		}
		t.Run(satelliteDB.Name, func(t *testing.T) {
			parallel := !config.NonParallel
			if parallel {
				t.Parallel()
			}

			if satelliteDB.MasterDB.URL == "" {
				t.Skipf("Database %s connection string not provided. %s", satelliteDB.MasterDB.Name, satelliteDB.MasterDB.Message)
			}
			planetConfig := config
			if planetConfig.Name == "" {
				planetConfig.Name = t.Name()
			}

			log := NewLogger(t)

			testmonkit.Run(context.Background(), t, func(parent context.Context) {
				defer pprof.SetGoroutineLabels(parent)
				parent = pprof.WithLabels(parent, pprof.Labels("test", t.Name()))

				timeout := config.Timeout
				if timeout == 0 {
					timeout = testcontext.DefaultTimeout
				}
				ctx := testcontext.NewWithContextAndTimeout(parent, t, timeout)
				defer ctx.Cleanup()

				planetConfig.applicationName = "testplanet" + pgutil.CreateRandomTestingSchemaName(6)
				planet, err := NewCustom(ctx, log, planetConfig, satelliteDB)
				if err != nil {
					t.Fatalf("%+v", err)
				}
				defer ctx.Check(planet.Shutdown)

				planet.Start(ctx)

				test(t, ctx, planet)
			})
		})
	}
}

// Bench makes benchmark with testplanet as easy as running unit tests with Run method.
func Bench(b *testing.B, config Config, bench func(b *testing.B, ctx *testcontext.Context, planet *Planet)) {
	databases := satellitedbtest.Databases()
	if len(databases) == 0 {
		b.Fatal("Databases flag missing, set at least one:\n" +
			"-postgres-test-db=" + pgtest.DefaultPostgres + "\n" +
			"-cockroach-test-db=" + pgtest.DefaultCockroach)
	}

	for _, satelliteDB := range databases {
		satelliteDB := satelliteDB
		b.Run(satelliteDB.Name, func(b *testing.B) {
			if satelliteDB.MasterDB.URL == "" {
				b.Skipf("Database %s connection string not provided. %s", satelliteDB.MasterDB.Name, satelliteDB.MasterDB.Message)
			}

			log := zap.NewNop()

			planetConfig := config
			if planetConfig.Name == "" {
				planetConfig.Name = b.Name()
			}

			testmonkit.Run(context.Background(), b, func(parent context.Context) {
				defer pprof.SetGoroutineLabels(parent)
				parent = pprof.WithLabels(parent, pprof.Labels("test", b.Name()))

				timeout := config.Timeout
				if timeout == 0 {
					timeout = testcontext.DefaultTimeout
				}
				ctx := testcontext.NewWithContextAndTimeout(parent, b, timeout)
				defer ctx.Cleanup()

				planetConfig.applicationName = "testplanet-bench"
				planet, err := NewCustom(ctx, log, planetConfig, satelliteDB)
				if err != nil {
					b.Fatalf("%+v", err)
				}
				defer ctx.Check(planet.Shutdown)

				planet.Start(ctx)

				bench(b, ctx, planet)
			})
		})
	}
}
