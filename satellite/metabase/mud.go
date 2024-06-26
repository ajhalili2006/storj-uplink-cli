// Copyright (C) 2024 Storj Labs, Inc.
// See LICENSE for copying information.

package metabase

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	database "github.com/storj/exp-spanner/admin/database/apiv1"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"storj.io/storj/private/mud"
)

//go:embed adapter_spanner_scheme.sql
var spannerDDL string

// SpannerTestModule adds all the required dependencies for Spanner migration and adapter.
func SpannerTestModule(ball *mud.Ball, spannerConnection string) {
	mud.Provide[*SpannerAdapter](ball, NewSpannerAdapter)
	mud.Implementation[[]Adapter, *SpannerAdapter](ball)
	mud.RemoveTag[*SpannerAdapter, mud.Optional](ball)
	// Please note that SpannerTestDatabase creates / deletes temporary database via the lifecycle functions.
	mud.Provide[SpannerTestDatabase](ball, func(ctx context.Context, logger *zap.Logger) (SpannerTestDatabase, error) {
		return NewSpannerTestDatabase(ctx, logger, spannerConnection, true)
	})
	mud.Provide[SpannerConfig](ball, NewTestSpannerConfig)
}

// SpannerTestDatabase manages Spanner database and migration for tests.
type SpannerTestDatabase struct {
	Database string
	client   *database.DatabaseAdminClient
}

// NewSpannerTestDatabase creates the database (=creates / migrates the database).
func NewSpannerTestDatabase(ctx context.Context, logger *zap.Logger, spannerConnection string, withMigration bool) (SpannerTestDatabase, error) {
	spannerConnection = strings.TrimPrefix(spannerConnection, "spanner://")
	data := make([]byte, 8)
	_, err := rand.Read(data)
	if err != nil {
		return SpannerTestDatabase{}, errs.Wrap(err)
	}

	adminClient, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return SpannerTestDatabase{}, errs.Wrap(err)
	}

	databaseName := spannerConnection + "_" + hex.EncodeToString(data)
	logger.Info("Creating temporary spanner database", zap.String("db", databaseName))

	matches := regexp.MustCompile("^(.*)/databases/(.*)$").FindStringSubmatch(databaseName)
	if matches == nil || len(matches) != 3 {
		return SpannerTestDatabase{}, errs.New("database connection should be defined in the form of 'projects/<PROJECT>/instances/<INSTANCE>/databases/<DATABASE>', but it was %q", spannerConnection)
	}

	req := &databasepb.CreateDatabaseRequest{
		Parent:          matches[1],
		DatabaseDialect: databasepb.DatabaseDialect_GOOGLE_STANDARD_SQL,
		CreateStatement: "CREATE DATABASE " + matches[2],
	}

	if withMigration {
		for _, ddl := range strings.Split(spannerDDL, ";") {
			if strings.TrimSpace(ddl) != "" {
				req.ExtraStatements = append(req.ExtraStatements, ddl)
			}
		}
	}
	ddl, err := adminClient.CreateDatabase(ctx, req)
	if err != nil {
		return SpannerTestDatabase{}, errs.Wrap(err)
	}
	_, err = ddl.Wait(ctx)
	if err != nil {
		return SpannerTestDatabase{}, errs.Wrap(err)
	}
	return SpannerTestDatabase{
		Database: databaseName,
		client:   adminClient,
	}, nil
}

// Connection returns with the used connection string (with added unique suffix).
func (d SpannerTestDatabase) Connection() string {
	return "spanner://" + d.Database
}

// Close drops the temporary test database.
func (d SpannerTestDatabase) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := d.client.DropDatabase(ctx, &databasepb.DropDatabaseRequest{
		Database: d.Database,
	})
	return errs.Combine(err, d.client.Close())
}

// NewTestSpannerConfig creates SpannerConfig for testing.
func NewTestSpannerConfig(database SpannerTestDatabase) SpannerConfig {
	return SpannerConfig{
		Database: database.Database,
	}
}
