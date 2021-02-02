// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowstorm/conflicts"
	"github.com/ava-labs/avalanchego/snow/engine/avalanche/vertex"
)

func newSerializer(t *testing.T, parse func([]byte) (conflicts.Transition, error)) (*vertex.TestVM, *Serializer) {
	vm := vertex.TestVM{}
	vm.T = t
	vm.Default(true)
	vm.ParseF = parse

	baseDB := memdb.New()
	ctx := snow.DefaultContextTest()
	s := &Serializer{}
	s.Initialize(ctx, &vm, baseDB)
	return &vm, s
}

func TestRestrictedTransition(t *testing.T) {
	trID := ids.Empty.Prefix(0)
	tr := &conflicts.TestTransition{
		IDV:    trID,
		BytesV: []byte{1},
	}

	vm, s := newSerializer(t, func(b []byte) (conflicts.Transition, error) {
		if bytes.Equal(b, tr.Bytes()) {
			return tr, nil
		}
		return nil, errors.New("unknown tx")
	})
	vm.GetF = func(id ids.ID) (conflicts.Transition, error) {
		switch id {
		case trID:
			return tr, nil
		default:
			return nil, errUnknownVertex
		}
	}

	vtx, err := s.Build(1, nil, nil, []ids.ID{trID})
	assert.NoError(t, err)

	txs, err := vtx.Txs()
	assert.NoError(t, err)
	assert.Len(t, txs, 1)

	tx := txs[0]

	err = tx.Verify()
	assert.NoError(t, err)

	err = tx.Accept()
	assert.NoError(t, err)
	assert.Equal(t, choices.Accepted, tx.Status())

	err = vtx.Accept()
	assert.NoError(t, err)
	assert.Equal(t, choices.Accepted, vtx.Status())

	newVtx, err := s.Build(0, nil, []conflicts.Transition{tr}, nil)
	assert.NoError(t, err)

	txs, err = newVtx.Txs()
	assert.NoError(t, err)
	assert.Len(t, txs, 1)

	tx = txs[0]

	err = tx.Verify()
	assert.Error(t, err)
}
