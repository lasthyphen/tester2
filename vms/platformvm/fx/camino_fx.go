// Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package fx

import (
	"github.com/ava-labs/avalanchego/vms/components/verify"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

type CaminoFx interface {
	// Recovers signers addresses from [verifies] credentials for [utx] transaction
	RecoverAddresses(utx secp256k1fx.UnsignedTx, verifies []verify.Verifiable) (secp256k1fx.RecoverMap, error)
	// Verifies that Multisig aliases are on inputs are only used in supported hierarchy
	VerifyMultisigOwner(outIntf, msigIntf interface{}) error
	// VerifyMultisigTransfer verifies that the specified transaction can spend the
	// provided utxo with no restrictions on the destination. If the transaction
	// can't spend the output based on the input and credential, a non-nil error
	// should be returned. Multisig aliases supported.
	VerifyMultisigTransfer(txIntf, inIntf, credIntf, utxoIntf, msigIntf interface{}) error

	// VerifyMultisigPermission returns nil if credential [credIntf] proves that [controlGroup] assents to transaction [utx].
	// Multisig aliases supported.
	VerifyMultisigPermission(txIntf, inIntf, credIntf, controlGroup, msigIntf interface{}) error

	// VerifyMultisigUnorderedPermission returns nil if credential [credIntf] proves [ownerIntf].
	// Multisig aliases supported. Signatures order and number doesn't matter.
	VerifyMultisigUnorderedPermission(txIntf, credIntf, ownerIntf, msigIntf interface{}) error
}
