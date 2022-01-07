// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package chain

import (
	"bytes"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/quarkvm/parser"
)

var _ UnsignedTransaction = &SetTx{}

type SetTx struct {
	*BaseTx `serialize:"true" json:"baseTx"`

	// Key is parsed from the given input, with its prefix removed.
	Key []byte `serialize:"true" json:"key"`
	// Value is empty if and only if set transaction is issued for the delete.
	// If non-empty, the transaction writes the key-value pair to the storage.
	// If empty, the transaction deletes the value for the "prefix/key".
	Value []byte `serialize:"true" json:"value"`

	// TODO: support range deletes?
}

func (s *SetTx) Execute(db database.Database, blockTime int64) error {
	// assume prefix is already validated via "BaseTx"
	if err := parser.CheckKey(s.Key); err != nil {
		return err
	}
	if len(s.Value) > MaxValueLength {
		return ErrValueTooBig
	}

	i, has, err := GetPrefixInfo(db, s.Prefix)
	if err != nil {
		return err
	}
	// Cannot set key if prefix doesn't exist
	if !has {
		return ErrPrefixMissing
	}
	// Prefix cannot be updated if not owned by modifier
	if !bytes.Equal(i.Owner[:], s.Sender[:]) {
		return ErrUnauthorized
	}
	// Prefix cannot be updated if expired
	if i.Expiry < blockTime {
		return ErrPrefixExpired
	}
	return s.updatePrefix(db, blockTime, i)
}

func (s *SetTx) updatePrefix(db database.Database, blockTime int64, i *PrefixInfo) error {
	v, exists, err := GetValue(db, s.Prefix, s.Key)
	if err != nil {
		return err
	}

	timeRemaining := (i.Expiry - i.LastUpdated) * i.Units
	if len(s.Value) == 0 { //nolint:nestif
		if !exists {
			return ErrKeyMissing
		}
		i.Units -= Units(v)
		if err := DeletePrefixKey(db, s.Prefix, s.Key); err != nil {
			return err
		}
	} else {
		if exists {
			i.Units -= Units(v)
		}
		i.Units += Units(s.Value)
		if err := PutPrefixKey(db, s.Prefix, s.Key, s.Value); err != nil {
			return err
		}
	}
	newTimeRemaining := timeRemaining / i.Units
	i.LastUpdated = blockTime
	lastExpiry := i.Expiry
	i.Expiry = blockTime + newTimeRemaining
	return PutPrefixInfo(db, s.Prefix, i, lastExpiry)
}

func (s *SetTx) Units() int64 {
	if len(s.Value) == 0 {
		return 1
	}
	u := Units(s.Value)
	return u/SetValueDiscount + 1
}
