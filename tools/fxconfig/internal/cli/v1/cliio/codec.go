// Copyright IBM Corp. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

package cliio

import (
	"encoding/json"
	"errors"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x/tools/fxconfig/internal/transaction"
)

// Codec handles encoding and decoding of transactions for CLI I/O.
type Codec interface {
	Encode(id transaction.ID, tx *applicationpb.Tx) ([]byte, error)
	Decode(data []byte) (id transaction.ID, tx *applicationpb.Tx, err error)
}

// JSONCodec implements Codec using JSON format with protobuf marshaling.
type JSONCodec struct{}

// Encode converts a transaction to JSON format with transaction ID and nonce.
func (*JSONCodec) Encode(id transaction.ID, tx *applicationpb.Tx) ([]byte, error) {
	var txJSON map[string]any

	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	if err := id.Validate(); err != nil {
		return nil, err
	}

	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}

	b, err := marshaler.Marshal(tx)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &txJSON); err != nil {
		return nil, err
	}

	output := map[string]any{
		"txID":  id.TxID,
		"nonce": id.Nonce,
		"tx":    txJSON,
	}

	return json.MarshalIndent(output, "", "  ")
}

// Decode parses JSON data into transaction ID, nonce, and transaction.
func (*JSONCodec) Decode(data []byte) (transaction.ID, *applicationpb.Tx, error) {
	carrier := struct {
		TxID  string          `json:"txID"`
		Nonce []byte          `json:"nonce"`
		Tx    json.RawMessage `json:"tx"`
	}{}

	err := json.Unmarshal(data, &carrier)
	if err != nil {
		return transaction.ID{}, nil, err
	}

	id := transaction.ID{TxID: carrier.TxID, Nonce: carrier.Nonce}
	if err = id.Validate(); err != nil {
		return transaction.ID{}, nil, err
	}

	var tx applicationpb.Tx
	err = protojson.Unmarshal(carrier.Tx, &tx)
	if err != nil {
		return transaction.ID{}, nil, err
	}

	return id, &tx, nil
}
