// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package contact_test

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"storj.io/common/pb"
	"storj.io/common/rpc/rpcpeer"
	"storj.io/common/testcontext"
	"storj.io/storj/private/testplanet"
	"storj.io/storj/storagenode"
)

func TestSatelliteContactEndpoint(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 1, UplinkCount: 0,
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		nodeInfo := planet.StorageNodes[0].Contact.Service.Local()
		ident := planet.StorageNodes[0].Identity

		peer := rpcpeer.Peer{
			Addr: &net.TCPAddr{
				IP:   net.ParseIP(nodeInfo.Address),
				Port: 5,
			},
			State: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{ident.Leaf, ident.CA},
			},
		}
		peerCtx := rpcpeer.NewContext(ctx, &peer)
		resp, err := planet.Satellites[0].Contact.Endpoint.CheckIn(peerCtx, &pb.CheckInRequest{
			Address:  nodeInfo.Address,
			Version:  &nodeInfo.Version,
			Capacity: &nodeInfo.Capacity,
			Operator: &nodeInfo.Operator,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.PingNodeSuccess)
		require.True(t, resp.PingNodeSuccessQuic)

		peerID, err := planet.Satellites[0].DB.PeerIdentities().Get(ctx, nodeInfo.ID)
		require.NoError(t, err)
		require.Equal(t, ident.PeerIdentity(), peerID)
	})
}

func TestSatelliteContactEndpoint_Failure(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 1, UplinkCount: 0,
		Reconfigure: testplanet.Reconfigure{
			StorageNode: func(index int, config *storagenode.Config) {
				config.Server.DisableTCPTLS = true
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		nodeInfo := planet.StorageNodes[0].Contact.Service.Local()
		ident := planet.StorageNodes[0].Identity

		peer := rpcpeer.Peer{
			Addr: &net.TCPAddr{
				IP:   net.ParseIP(nodeInfo.Address),
				Port: 5,
			},
			State: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{ident.Leaf, ident.CA},
			},
		}
		peerCtx := rpcpeer.NewContext(ctx, &peer)
		resp, err := planet.Satellites[0].Contact.Endpoint.CheckIn(peerCtx, &pb.CheckInRequest{
			Address:  nodeInfo.Address,
			Version:  &nodeInfo.Version,
			Capacity: &nodeInfo.Capacity,
			Operator: &nodeInfo.Operator,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.False(t, resp.PingNodeSuccess)
		require.False(t, resp.PingNodeSuccessQuic)

		peerID, err := planet.Satellites[0].DB.PeerIdentities().Get(ctx, nodeInfo.ID)
		require.NoError(t, err)
		require.Equal(t, ident.PeerIdentity(), peerID)
	})
}
func TestSatelliteContactEndpoint_QUIC_Unreachable(t *testing.T) {
	testplanet.Run(t, testplanet.Config{
		SatelliteCount: 1, StorageNodeCount: 1, UplinkCount: 0,
		Reconfigure: testplanet.Reconfigure{
			StorageNode: func(index int, config *storagenode.Config) {
				config.Server.DisableQUIC = true
			},
		},
	}, func(t *testing.T, ctx *testcontext.Context, planet *testplanet.Planet) {
		nodeInfo := planet.StorageNodes[0].Contact.Service.Local()
		ident := planet.StorageNodes[0].Identity

		peer := rpcpeer.Peer{
			Addr: &net.TCPAddr{
				IP:   net.ParseIP(nodeInfo.Address),
				Port: 5,
			},
			State: tls.ConnectionState{
				PeerCertificates: []*x509.Certificate{ident.Leaf, ident.CA},
			},
		}
		peerCtx := rpcpeer.NewContext(ctx, &peer)
		resp, err := planet.Satellites[0].Contact.Endpoint.CheckIn(peerCtx, &pb.CheckInRequest{
			Address:  nodeInfo.Address,
			Version:  &nodeInfo.Version,
			Capacity: &nodeInfo.Capacity,
			Operator: &nodeInfo.Operator,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.True(t, resp.PingNodeSuccess)
		require.False(t, resp.PingNodeSuccessQuic)

		peerID, err := planet.Satellites[0].DB.PeerIdentities().Get(ctx, nodeInfo.ID)
		require.NoError(t, err)
		require.Equal(t, ident.PeerIdentity(), peerID)
	})
}
