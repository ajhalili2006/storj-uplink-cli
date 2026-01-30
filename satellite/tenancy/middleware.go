// Copyright (C) 2025 Storj Labs, Inc.
// See LICENSE for copying information.

package tenancy

import (
	"net/http"

	"github.com/spacemonkeygo/monkit/v3"
)

var mon = monkit.Package()

// Middleware returns an HTTP middleware that resolves tenant ID from the request hostname.
// If defaultTenantID is provided (non-empty), it will be used when no hostname match is found.
// This supports single white label deployments where all requests should use the same tenant ID.
func Middleware(lookupMap map[string]string, defaultTenantID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var err error
			ctx := r.Context()

			defer mon.Task()(&ctx)(&err)

			tenantID := FromHostname(r.Host, lookupMap)
			if tenantID == "" && defaultTenantID != "" {
				tenantID = defaultTenantID
			}

			ctx = WithContext(ctx, &Context{
				TenantID: tenantID,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
