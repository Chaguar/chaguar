// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cliio

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
)

var testNonce = []byte("nonce-123")

func TestJSONCodec_Encode(t *testing.T) {
	t.Parallel()

	codec := &JSONCodec{}

	t.Run("success with valid transaction", func(t *testing.T) {
		t.Parallel()

		tx := &applicationpb.Tx{}
		txID := "tx-123"

		data, err := codec.Encode(transaction.ID{TxID: txID, Nonce: testNonce}, tx)
		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Verify JSON structure; []byte is encoded as base64
		var result map[string]any
		err = json.Unmarshal(data, &result)
		require.NoError(t, err)
		require.Equal(t, txID, result["txID"])
		require.Equal(t, "bm9uY2UtMTIz", result["nonce"])
		require.NotNil(t, result["tx"])
	})

	t.Run("error with incomplete ID", func(t *testing.T) {
		t.Parallel()

		_, err := codec.Encode(transaction.ID{TxID: "tx-123"}, &applicationpb.Tx{})
		require.ErrorContains(t, err, "no nonce")

		_, err = codec.Encode(transaction.ID{Nonce: testNonce}, &applicationpb.Tx{})
		require.ErrorContains(t, err, "no txID")
	})

	t.Run("error with nil transaction", func(t *testing.T) {
		t.Parallel()

		txID := "tx-123"

		data, err := codec.Encode(transaction.ID{TxID: txID, Nonce: testNonce}, nil)
		require.Error(t, err)
		require.Nil(t, data)
		require.Contains(t, err.Error(), "tx is nil")
	})
}

func TestJSONCodec_Decode(t *testing.T) {
	t.Parallel()

	codec := &JSONCodec{}

	t.Run("success with valid JSON", func(t *testing.T) {
		t.Parallel()

		input := `{
  "tx": {
    "endorsements": [
      {
        "endorsements_with_identity": [
          {
            "endorsement": "cGF5bWVudHM=",
            "identity": {
              "certificate_id": "03ca0f49dc4a7732672611c4ee1ca0362f73b495270eca1dd7a60fede56e4f92",
              "msp_id": "Org1MSP"
            }
          }
        ]
      }
    ],
    "namespaces": [
      {
        "ns_id": "_meta",
        "read_writes": [
          {
            "key": "cGF5bWVudHM=",
            "value": "EigSDBIKCAISAggAEgIIARoLEgkKB09yZzFNU1AaCxIJCgdPcmcyTVNQ"
          }
        ]
      }
    ]
  },
  "nonce": "bm9uY2UtMTIz",
  "txID": "1ca34644c2ac3b7570cc09fee527757d603e282ed03392df805c2e00437ab224"
}`

		id, tx, err := codec.Decode([]byte(input))
		require.NoError(t, err)
		require.Equal(t, "1ca34644c2ac3b7570cc09fee527757d603e282ed03392df805c2e00437ab224", id.TxID)
		require.Equal(t, testNonce, id.Nonce)
		require.NotNil(t, tx)
		require.Len(t, tx.GetNamespaces(), 1)
		require.Equal(t, "_meta", tx.GetNamespaces()[0].GetNsId())
	})

	for name, input := range map[string]string{
		"invalid JSON":  `{invalid json}`,
		"missing txID":  `{"nonce": "bm9uY2UtMTIz", "tx": {}}`,
		"missing nonce": `{"txID": "tx-123", "tx": {}}`,
		"empty nonce":   `{"txID": "tx-123", "nonce": "", "tx": {}}`,
		"missing tx":    `{"txID": "tx-123", "nonce": "bm9uY2UtMTIz"}`,
		"empty input":   ``,
	} {
		t.Run("error with "+name, func(t *testing.T) {
			t.Parallel()

			id, tx, err := codec.Decode([]byte(input))
			require.Error(t, err)
			require.Empty(t, id)
			require.Nil(t, tx)
		})
	}

	t.Run("missing nonce error explains how to fix it", func(t *testing.T) {
		t.Parallel()

		_, _, err := codec.Decode([]byte(`{"txID": "tx-123", "tx": {}}`))
		require.ErrorContains(t, err, "no nonce")
	})
}

func TestJSONCodec_RoundTrip(t *testing.T) {
	t.Parallel()

	codec := &JSONCodec{}

	tx := &applicationpb.Tx{
		Namespaces: []*applicationpb.TxNamespace{{
			NsId:      "some_namespace",
			NsVersion: 0,
			BlindWrites: []*applicationpb.Write{{
				Key:   []byte("key"),
				Value: []byte("value"),
			}},
		}},
	}
	originalTxID := "tx-round-trip"

	encoded, err := codec.Encode(transaction.ID{TxID: originalTxID, Nonce: testNonce}, tx)
	require.NoError(t, err)

	decoded, decodedTx, err := codec.Decode(encoded)
	require.NoError(t, err)

	require.Equal(t, transaction.ID{TxID: originalTxID, Nonce: testNonce}, decoded)
	require.Equal(t, tx.GetNamespaces()[0].GetNsId(), decodedTx.GetNamespaces()[0].GetNsId())
	require.Equal(t, tx.GetNamespaces()[0].GetBlindWrites()[0], decodedTx.GetNamespaces()[0].GetBlindWrites()[0])
}
