// Copyright (C) 2022-2023, Chain4Travel AG. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"fmt"
	"time"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils"
	"github.com/ava-labs/avalanchego/utils/set"
	"github.com/ava-labs/avalanchego/utils/timer/mockable"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/components/multisig"
	"github.com/ava-labs/avalanchego/vms/platformvm/config"
	"github.com/ava-labs/avalanchego/vms/platformvm/deposit"
	"github.com/ava-labs/avalanchego/vms/platformvm/locked"
)

func NewCaminoDiff(
	parentID ids.ID,
	stateVersions Versions,
) (Diff, error) {
	parentState, ok := stateVersions.GetState(parentID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingParentState, parentID)
	}
	return &diff{
		parentID:      parentID,
		stateVersions: stateVersions,
		timestamp:     parentState.GetTimestamp(),
		caminoDiff:    newCaminoDiff(),
	}, nil
}

func (d *diff) LockedUTXOs(txIDs set.Set[ids.ID], addresses set.Set[ids.ShortID], lockState locked.State) ([]*avax.UTXO, error) {
	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}

	retUtxos, err := parentState.LockedUTXOs(txIDs, addresses, lockState)
	if err != nil {
		return nil, err
	}

	// Apply modifiedUTXO's
	// Step 1: remove / update existing UTXOs
	remaining := set.NewSet[ids.ID](len(d.modifiedUTXOs))
	for k := range d.modifiedUTXOs {
		remaining.Add(k)
	}
	for i := len(retUtxos) - 1; i >= 0; i-- {
		if utxo, exists := d.modifiedUTXOs[retUtxos[i].InputID()]; exists {
			if utxo.utxo == nil {
				retUtxos = append(retUtxos[:i], retUtxos[i+1:]...)
			} else {
				retUtxos[i] = utxo.utxo
			}
			delete(remaining, utxo.utxoID)
		}
	}

	// Step 2: Append new UTXOs
	for utxoID := range remaining {
		utxo := d.modifiedUTXOs[utxoID].utxo
		if utxo != nil {
			if lockedOut, ok := utxo.Out.(*locked.Out); ok &&
				lockedOut.IDs.Match(lockState, txIDs) {
				retUtxos = append(retUtxos, utxo)
			}
		}
	}

	return retUtxos, nil
}

func (d *diff) Config() (*config.Config, error) {
	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}
	return parentState.Config()
}

func (d *diff) CaminoConfig() (*CaminoConfig, error) {
	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}
	return parentState.CaminoConfig()
}

func (d *diff) SetAddressStates(address ids.ShortID, states uint64) {
	d.caminoDiff.modifiedAddressStates[address] = states
}

func (d *diff) GetAddressStates(address ids.ShortID) (uint64, error) {
	if states, ok := d.caminoDiff.modifiedAddressStates[address]; ok {
		return states, nil
	}

	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}

	return parentState.GetAddressStates(address)
}

func (d *diff) SetDepositOffer(offer *deposit.Offer) {
	d.caminoDiff.modifiedDepositOffers[offer.ID] = offer
}

func (d *diff) GetDepositOffer(offerID ids.ID) (*deposit.Offer, error) {
	if offer, ok := d.caminoDiff.modifiedDepositOffers[offerID]; ok {
		return offer, nil
	}

	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}

	return parentState.GetDepositOffer(offerID)
}

func (d *diff) GetAllDepositOffers() ([]*deposit.Offer, error) {
	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}

	parentOffers, err := parentState.GetAllDepositOffers()
	if err != nil {
		return nil, err
	}

	var offers []*deposit.Offer

	for _, offer := range d.caminoDiff.modifiedDepositOffers {
		if offer != nil {
			offers = append(offers, offer)
		}
	}

	for _, offer := range parentOffers {
		if _, ok := d.caminoDiff.modifiedDepositOffers[offer.ID]; !ok {
			offers = append(offers, offer)
		}
	}

	return offers, nil
}

func (d *diff) AddDeposit(depositTxID ids.ID, deposit *deposit.Deposit) {
	d.caminoDiff.modifiedDeposits[depositTxID] = &depositDiff{Deposit: deposit, added: true}
}

func (d *diff) ModifyDeposit(depositTxID ids.ID, deposit *deposit.Deposit) {
	d.caminoDiff.modifiedDeposits[depositTxID] = &depositDiff{Deposit: deposit}
}

func (d *diff) RemoveDeposit(depositTxID ids.ID, deposit *deposit.Deposit) {
	d.caminoDiff.modifiedDeposits[depositTxID] = &depositDiff{Deposit: deposit, removed: true}
}

func (d *diff) GetDeposit(depositTxID ids.ID) (*deposit.Deposit, error) {
	if depositDiff, ok := d.caminoDiff.modifiedDeposits[depositTxID]; ok {
		if depositDiff.removed {
			return nil, database.ErrNotFound
		}
		return depositDiff.Deposit, nil
	}

	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}

	return parentState.GetDeposit(depositTxID)
}

func (d *diff) GetNextToUnlockDepositTime(removedDepositIDs set.Set[ids.ID]) (time.Time, error) {
	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return time.Time{}, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}

	for depositID, depositDiff := range d.caminoDiff.modifiedDeposits {
		if depositDiff.removed {
			removedDepositIDs.Add(depositID)
		}
	}

	nextUnlockTime, err := parentState.GetNextToUnlockDepositTime(removedDepositIDs)
	if err != nil && err != database.ErrNotFound {
		return time.Time{}, err
	}

	// calculating earliest unlock time from added deposits and parent unlock time
	for depositID, depositDiff := range d.caminoDiff.modifiedDeposits {
		depositEndtime := depositDiff.EndTime()
		if depositDiff.added && depositEndtime.Before(nextUnlockTime) && !removedDepositIDs.Contains(depositID) {
			nextUnlockTime = depositEndtime
		}
	}

	// no deposits
	if nextUnlockTime.Equal(mockable.MaxTime) {
		return mockable.MaxTime, database.ErrNotFound
	}

	return nextUnlockTime, nil
}

func (d *diff) GetNextToUnlockDepositIDsAndTime(removedDepositIDs set.Set[ids.ID]) ([]ids.ID, time.Time, error) {
	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return nil, time.Time{}, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}

	for depositID, depositDiff := range d.caminoDiff.modifiedDeposits {
		if depositDiff.removed {
			removedDepositIDs.Add(depositID)
		}
	}

	parentNextDepositIDs, parentNextUnlockTime, err := parentState.GetNextToUnlockDepositIDsAndTime(removedDepositIDs)
	if err != nil && err != database.ErrNotFound {
		return nil, time.Time{}, err
	}

	// calculating earliest unlock time from added deposits and parent unlock time
	nextUnlockTime := parentNextUnlockTime
	for depositID, depositDiff := range d.caminoDiff.modifiedDeposits {
		depositEndtime := depositDiff.EndTime()
		if depositDiff.added && depositEndtime.Before(nextUnlockTime) && !removedDepositIDs.Contains(depositID) {
			nextUnlockTime = depositEndtime
		}
	}

	// no deposits
	if nextUnlockTime.Equal(mockable.MaxTime) {
		return nil, mockable.MaxTime, database.ErrNotFound
	}

	var nextDepositIDs []ids.ID
	if !parentNextUnlockTime.After(nextUnlockTime) {
		nextDepositIDs = parentNextDepositIDs
	}

	// getting added deposits with endtime matching nextUnlockTime
	needSort := false
	for depositID, depositDiff := range d.caminoDiff.modifiedDeposits {
		if depositDiff.added && depositDiff.EndTime().Equal(nextUnlockTime) && !removedDepositIDs.Contains(depositID) {
			nextDepositIDs = append(nextDepositIDs, depositID)
			needSort = true
		}
	}

	if needSort {
		utils.Sort(nextDepositIDs)
	}

	return nextDepositIDs, nextUnlockTime, nil
}

func (d *diff) SetMultisigAlias(owner *multisig.Alias) {
	d.caminoDiff.modifiedMultisigOwners[owner.ID] = owner
}

func (d *diff) GetMultisigAlias(alias ids.ShortID) (*multisig.Alias, error) {
	if msigOwner, ok := d.caminoDiff.modifiedMultisigOwners[alias]; ok {
		if msigOwner == nil {
			return nil, database.ErrNotFound
		}
		return msigOwner, nil
	}

	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}

	return parentState.GetMultisigAlias(alias)
}

func (d *diff) SetShortIDLink(id ids.ShortID, key ShortLinkKey, link *ids.ShortID) {
	d.caminoDiff.modifiedShortLinks[toShortLinkKey(id, key)] = link
}

func (d *diff) GetShortIDLink(id ids.ShortID, key ShortLinkKey) (ids.ShortID, error) {
	if addr, ok := d.caminoDiff.modifiedShortLinks[toShortLinkKey(id, key)]; ok {
		if addr == nil {
			return ids.ShortEmpty, database.ErrNotFound
		}
		return *addr, nil
	}

	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return ids.ShortEmpty, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}

	return parentState.GetShortIDLink(id, key)
}

func (d *diff) SetClaimable(ownerID ids.ID, claimable *Claimable) {
	d.caminoDiff.modifiedClaimables[ownerID] = claimable
}

func (d *diff) GetClaimable(ownerID ids.ID) (*Claimable, error) {
	if claimable, ok := d.caminoDiff.modifiedClaimables[ownerID]; ok {
		if claimable == nil {
			return nil, database.ErrNotFound
		}
		return claimable, nil
	}

	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}

	return parentState.GetClaimable(ownerID)
}

func (d *diff) SetNotDistributedValidatorReward(reward uint64) {
	d.caminoDiff.modifiedNotDistributedValidatorReward = &reward
}

func (d *diff) GetNotDistributedValidatorReward() (uint64, error) {
	if d.caminoDiff.modifiedNotDistributedValidatorReward != nil {
		return *d.caminoDiff.modifiedNotDistributedValidatorReward, nil
	}

	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}

	return parentState.GetNotDistributedValidatorReward()
}

func (d *diff) GetDeferredValidator(subnetID ids.ID, nodeID ids.NodeID) (*Staker, error) {
	// If the validator was modified in this diff, return the modified
	// validator.
	newValidator, ok := d.caminoDiff.deferredStakerDiffs.GetValidator(subnetID, nodeID)
	if ok {
		if newValidator == nil {
			return nil, database.ErrNotFound
		}
		return newValidator, nil
	}

	// If the validator wasn't modified in this diff, ask the parent state.
	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}
	return parentState.GetDeferredValidator(subnetID, nodeID)
}

func (d *diff) PutDeferredValidator(staker *Staker) {
	d.caminoDiff.deferredStakerDiffs.PutValidator(staker)
}

func (d *diff) DeleteDeferredValidator(staker *Staker) {
	d.caminoDiff.deferredStakerDiffs.DeleteValidator(staker)
}

func (d *diff) GetDeferredStakerIterator() (StakerIterator, error) {
	parentState, ok := d.stateVersions.GetState(d.parentID)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrMissingParentState, d.parentID)
	}

	parentIterator, err := parentState.GetDeferredStakerIterator()
	if err != nil {
		return nil, err
	}

	return d.caminoDiff.deferredStakerDiffs.GetStakerIterator(parentIterator), nil
}

// Finally apply all changes
func (d *diff) ApplyCaminoState(baseState State) {
	if d.caminoDiff.modifiedNotDistributedValidatorReward != nil {
		baseState.SetNotDistributedValidatorReward(*d.caminoDiff.modifiedNotDistributedValidatorReward)
	}

	for k, v := range d.caminoDiff.modifiedAddressStates {
		baseState.SetAddressStates(k, v)
	}

	for _, depositOffer := range d.caminoDiff.modifiedDepositOffers {
		baseState.SetDepositOffer(depositOffer)
	}

	for depositTxID, depositDiff := range d.caminoDiff.modifiedDeposits {
		switch {
		case depositDiff.added:
			baseState.AddDeposit(depositTxID, depositDiff.Deposit)
		case depositDiff.removed:
			baseState.RemoveDeposit(depositTxID, depositDiff.Deposit)
		default:
			baseState.ModifyDeposit(depositTxID, depositDiff.Deposit)
		}
	}

	for _, v := range d.caminoDiff.modifiedMultisigOwners {
		baseState.SetMultisigAlias(v)
	}

	for fullKey, link := range d.caminoDiff.modifiedShortLinks {
		id, key := fromShortLinkKey(fullKey)
		baseState.SetShortIDLink(id, key, link)
	}

	for ownerID, claimable := range d.caminoDiff.modifiedClaimables {
		baseState.SetClaimable(ownerID, claimable)
	}

	for _, validatorDiffs := range d.caminoDiff.deferredStakerDiffs.validatorDiffs {
		for _, validatorDiff := range validatorDiffs {
			if validatorDiff.validatorModified {
				if validatorDiff.validatorDeleted {
					baseState.DeleteDeferredValidator(validatorDiff.validator)
				} else {
					baseState.PutDeferredValidator(validatorDiff.validator)
				}
			}
		}
	}
}
