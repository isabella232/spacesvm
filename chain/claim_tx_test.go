// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestClaimTx(t *testing.T) {
	t.Parallel()

	priv, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender := crypto.PubkeyToAddress(priv.PublicKey)

	priv2, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	sender2 := crypto.PubkeyToAddress(priv2.PublicKey)

	db := memdb.New()
	defer db.Close()

	g := DefaultGenesis()
	ClaimReward := int64(g.ClaimReward)
	tt := []struct {
		tx        *ClaimTx
		blockTime int64
		sender    common.Address
		err       error
	}{
		{ // invalid claim, [20]byte prefix is reserved for pubkey
			tx:        &ClaimTx{BaseTx: &BaseTx{Pfx: bytes.Repeat([]byte{'a'}, common.AddressLength)}},
			blockTime: 1,
			sender:    sender,
			err:       ErrAddressMismatch,
		},
		{ // successful claim with expiry time "blockTime" + "expiryTime"
			tx:        &ClaimTx{BaseTx: &BaseTx{Pfx: []byte("foo")}},
			blockTime: 1,
			sender:    sender,
			err:       nil,
		},
		{ // invalid claim due to expiration
			tx:        &ClaimTx{BaseTx: &BaseTx{Pfx: []byte("foo")}},
			blockTime: 100,
			sender:    sender,
			err:       ErrPrefixNotExpired,
		},
		{ // successful new claim
			tx:        &ClaimTx{BaseTx: &BaseTx{Pfx: []byte("foo")}},
			blockTime: ClaimReward * 2,
			sender:    sender,
			err:       nil,
		},
		{ // successful new claim by different owner
			tx:        &ClaimTx{BaseTx: &BaseTx{Pfx: []byte("foo")}},
			blockTime: ClaimReward * 4,
			sender:    sender2,
			err:       nil,
		},
		{ // invalid claim due to expiration by different owner
			tx:        &ClaimTx{BaseTx: &BaseTx{Pfx: []byte("foo")}},
			blockTime: ClaimReward*4 + 3,
			sender:    sender2,
			err:       ErrPrefixNotExpired,
		},
	}
	for i, tv := range tt {
		if i > 0 {
			// Expire old prefixes between txs
			if err := ExpireNext(db, tt[i-1].blockTime, tv.blockTime, true); err != nil {
				t.Fatalf("#%d: ExpireNext errored %v", i, err)
			}
		}
		tc := &TransactionContext{
			Genesis:   g,
			Database:  db,
			BlockTime: uint64(tv.blockTime),
			TxID:      ids.Empty,
			Sender:    tv.sender,
		}
		err := tv.tx.Execute(tc)
		if !errors.Is(err, tv.err) {
			t.Fatalf("#%d: tx.Execute err expected %v, got %v", i, tv.err, err)
		}
		if tv.err != nil {
			continue
		}
		info, exists, err := GetPrefixInfo(db, tv.tx.Prefix())
		if err != nil {
			t.Fatalf("#%d: failed to get prefix info %v", i, err)
		}
		if !exists {
			t.Fatalf("#%d: failed to find prefix info", i)
		}
		if !bytes.Equal(info.Owner[:], tv.sender[:]) {
			t.Fatalf("#%d: unexpected owner found (expected pub key %q)", i, string(sender[:]))
		}
	}

	// Cleanup DB after all txs submitted
	if err := ExpireNext(db, 0, ClaimReward*10, true); err != nil {
		t.Fatal(err)
	}
	pruned, err := PruneNext(db, 100)
	if err != nil {
		t.Fatal(err)
	}
	if pruned != 3 {
		t.Fatalf("expected to prune 3 but got %d", pruned)
	}
	_, exists, err := GetPrefixInfo(db, []byte("foo"))
	if err != nil {
		t.Fatalf("failed to get prefix info %v", err)
	}
	if exists {
		t.Fatal("prefix should not exist")
	}
}
