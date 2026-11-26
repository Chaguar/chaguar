/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package client

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	msppb "github.com/hyperledger/fabric-protos-go-apiv2/msp"
	ab "github.com/hyperledger/fabric-protos-go-apiv2/orderer"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x-common/protoutil"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/config"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
)

// mockBroadcastStream implements grpc.BidiStreamingClient[cb.Envelope, ab.BroadcastResponse].
type mockBroadcastStream struct {
	sendErr  error
	recvResp *ab.BroadcastResponse
	recvErr  error
	sent     *cb.Envelope
}

func (m *mockBroadcastStream) Send(env *cb.Envelope) error {
	m.sent = env
	return m.sendErr
}
func (m *mockBroadcastStream) Recv() (*ab.BroadcastResponse, error) { return m.recvResp, m.recvErr }
func (*mockBroadcastStream) Header() (metadata.MD, error)           { return nil, nil }
func (*mockBroadcastStream) Trailer() metadata.MD                   { return nil }
func (*mockBroadcastStream) CloseSend() error                       { return nil }
func (*mockBroadcastStream) Context() context.Context               { return context.Background() }
func (*mockBroadcastStream) SendMsg(_ any) error                    { return nil }
func (*mockBroadcastStream) RecvMsg(_ any) error                    { return nil }

// mockAtomicBroadcastClient implements ab.AtomicBroadcastClient.
type mockAtomicBroadcastClient struct {
	stream       grpc.BidiStreamingClient[cb.Envelope, ab.BroadcastResponse]
	broadcastErr error
}

func (m *mockAtomicBroadcastClient) Broadcast(
	_ context.Context,
	_ ...grpc.CallOption,
) (grpc.BidiStreamingClient[cb.Envelope, ab.BroadcastResponse], error) {
	return m.stream, m.broadcastErr
}

func (*mockAtomicBroadcastClient) Deliver(
	_ context.Context,
	_ ...grpc.CallOption,
) (grpc.BidiStreamingClient[cb.Envelope, ab.DeliverResponse], error) {
	return nil, errors.New("not implemented")
}

// testSigningIdentity is a minimal mock of msp.SigningIdentity.
type testSigningIdentity struct {
	signErr error
	creator []byte // serialized identity; defaults to testCreator
}

var (
	testCreator = []byte("test-creator")
	testNonce   = []byte("test-nonce")
	testTxID    = protoutil.ComputeTxID(testNonce, testCreator)
	testID      = transaction.ID{TxID: testTxID, Nonce: testNonce}
)

const errWrongCreator = "created by a different identity"

func (m *testSigningIdentity) Sign(_ []byte) ([]byte, error) {
	if m.signErr != nil {
		return nil, m.signErr
	}
	return []byte("mock-sig"), nil
}

func (*testSigningIdentity) SerializeWithIDOfCert() ([]byte, error) { return []byte{}, nil }
func (m *testSigningIdentity) Serialize() ([]byte, error) {
	if m.creator != nil {
		return m.creator, nil
	}
	return testCreator, nil
}
func (*testSigningIdentity) GetCertificatePEM() ([]byte, error)             { return nil, nil }
func (*testSigningIdentity) GetIdentifier() *msp.IdentityIdentifier         { return nil }
func (*testSigningIdentity) GetPublicVersion() msp.Identity                 { return nil } //nolint:ireturn
func (*testSigningIdentity) Verify(_, _ []byte) error                       { return nil }
func (*testSigningIdentity) GetOrganizationalUnits() []*msp.OUIdentifier    { return nil }
func (*testSigningIdentity) Anonymous() bool                                { return false }
func (*testSigningIdentity) ExpiresAt() time.Time                           { return time.Time{} }
func (*testSigningIdentity) GetMSPIdentifier() string                       { return "TestMSP" }
func (*testSigningIdentity) Validate() error                                { return nil }
func (*testSigningIdentity) SatisfiesPrincipal(_ *msppb.MSPPrincipal) error { return nil }

func newTestOrdererClient(mock ab.AtomicBroadcastClient) *OrdererClient {
	return &OrdererClient{
		cfg: config.OrdererConfig{
			EndpointServiceConfig: config.EndpointServiceConfig{
				ConnectionTimeout: time.Second,
			},
			Channel: "mychannel",
		},
		client: mock,
	}
}

func someBroadcastTx() *applicationpb.Tx {
	return &applicationpb.Tx{Namespaces: []*applicationpb.TxNamespace{{NsId: "ns1"}}}
}

func TestOrdererClient_Broadcast_NilClient(t *testing.T) {
	t.Parallel()

	oc := &OrdererClient{}
	err := oc.Broadcast(t.Context(), &testSigningIdentity{}, testID, someBroadcastTx())
	require.Error(t, err)
}

func TestOrdererClient_Broadcast_NilSigner(t *testing.T) {
	t.Parallel()

	oc := newTestOrdererClient(&mockAtomicBroadcastClient{
		stream: &mockBroadcastStream{recvResp: &ab.BroadcastResponse{Status: cb.Status_SUCCESS}},
	})
	err := oc.Broadcast(t.Context(), nil, testID, someBroadcastTx())
	require.Error(t, err)
}

func TestOrdererClient_Broadcast_SignerError(t *testing.T) {
	t.Parallel()

	oc := newTestOrdererClient(&mockAtomicBroadcastClient{
		stream: &mockBroadcastStream{recvResp: &ab.BroadcastResponse{Status: cb.Status_SUCCESS}},
	})
	err := oc.Broadcast(t.Context(),
		&testSigningIdentity{signErr: errors.New("sign failed")}, testID, someBroadcastTx())
	require.Error(t, err)
}

func TestOrdererClient_Broadcast_StreamError(t *testing.T) {
	t.Parallel()

	oc := newTestOrdererClient(&mockAtomicBroadcastClient{
		broadcastErr: errors.New("stream unavailable"),
	})
	err := oc.Broadcast(t.Context(), &testSigningIdentity{}, testID, someBroadcastTx())
	require.Error(t, err)
}

func TestOrdererClient_Broadcast_SendError(t *testing.T) {
	t.Parallel()

	oc := newTestOrdererClient(&mockAtomicBroadcastClient{
		stream: &mockBroadcastStream{sendErr: errors.New("send failed")},
	})
	err := oc.Broadcast(t.Context(), &testSigningIdentity{}, testID, someBroadcastTx())
	require.Error(t, err)
}

func TestOrdererClient_Broadcast_RecvError(t *testing.T) {
	t.Parallel()

	oc := newTestOrdererClient(&mockAtomicBroadcastClient{
		stream: &mockBroadcastStream{recvErr: errors.New("recv failed")},
	})
	err := oc.Broadcast(t.Context(), &testSigningIdentity{}, testID, someBroadcastTx())
	require.Error(t, err)
}

func TestOrdererClient_Broadcast_NonSuccessStatus(t *testing.T) {
	t.Parallel()

	oc := newTestOrdererClient(&mockAtomicBroadcastClient{
		stream: &mockBroadcastStream{recvResp: &ab.BroadcastResponse{Status: cb.Status_BAD_REQUEST}},
	})
	err := oc.Broadcast(t.Context(), &testSigningIdentity{}, testID, someBroadcastTx())
	require.Error(t, err)
}

func TestOrdererClient_Broadcast_Success(t *testing.T) {
	t.Parallel()

	stream := &mockBroadcastStream{recvResp: &ab.BroadcastResponse{Status: cb.Status_SUCCESS}}
	oc := newTestOrdererClient(&mockAtomicBroadcastClient{stream: stream})
	err := oc.Broadcast(t.Context(), &testSigningIdentity{}, testID, someBroadcastTx())
	require.NoError(t, err)

	// the envelope carries what the orderer checks: TxId == ComputeTxID(nonce, creator)
	payload, err := protoutil.UnmarshalPayload(stream.sent.GetPayload())
	require.NoError(t, err)
	chdr, err := protoutil.UnmarshalChannelHeader(payload.GetHeader().GetChannelHeader())
	require.NoError(t, err)
	shdr, err := protoutil.UnmarshalSignatureHeader(payload.GetHeader().GetSignatureHeader())
	require.NoError(t, err)
	require.Equal(t, testTxID, chdr.GetTxId())
	require.Equal(t, testNonce, shdr.GetNonce())
	require.Equal(t, testCreator, shdr.GetCreator())
}

func TestOrdererClient_Broadcast_RejectsBadTxID(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		signer *testSigningIdentity
		id     transaction.ID
		errMsg string
	}{
		"empty nonce": {&testSigningIdentity{}, transaction.ID{TxID: testTxID}, "no nonce"},
		"different nonce": {
			&testSigningIdentity{}, transaction.ID{TxID: testTxID, Nonce: []byte("x")}, errWrongCreator,
		},
		"different identity": {&testSigningIdentity{creator: []byte("other")}, testID, errWrongCreator},
		"arbitrary txID":     {&testSigningIdentity{}, transaction.ID{TxID: "tx-1", Nonce: testNonce}, errWrongCreator},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			stream := &mockBroadcastStream{recvResp: &ab.BroadcastResponse{Status: cb.Status_SUCCESS}}
			oc := newTestOrdererClient(&mockAtomicBroadcastClient{stream: stream})
			err := oc.Broadcast(t.Context(), tc.signer, tc.id, someBroadcastTx())
			require.ErrorContains(t, err, tc.errMsg)
			require.Nil(t, stream.sent, "nothing must be sent to the orderer")
		})
	}
}

func TestOrdererClient_Close_CallsCloseFunc(t *testing.T) {
	t.Parallel()

	closed := false
	oc := &OrdererClient{closeF: func() { closed = true }}
	require.NoError(t, oc.Close())
	require.True(t, closed)
}

func TestOrdererClient_Close_NilFunc(t *testing.T) {
	t.Parallel()

	oc := &OrdererClient{}
	require.NoError(t, oc.Close())
}
