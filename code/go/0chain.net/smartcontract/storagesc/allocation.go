package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"time"

	chainstate "0chain.net/chaincore/chain/state"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/util"
)

// getAllocation by ID
func (sc *StorageSmartContract) getAllocation(
	allocID string,
	sci chainstate.StateContextI) (allocation *StorageAllocation, err error) {
	errorCode := "get_allocation"
	allocation = &StorageAllocation{ID: allocID}
	err = sci.GetDecodedTrieNode(allocation.GetKey(sc.ID), allocation)
	if err == util.ErrValueNotPresent {
		return nil, err
	}
	if err != nil {
		return nil, common.NewError(errorCode, err.Error())
	}
	return
}

func (sc *StorageSmartContract) getAllAllocations(
	sci chainstate.StateContextI) (allocations *Allocations, err error) {
	errorCode := "get_all_allocations"
	allocations = &Allocations{}
	err = sci.GetDecodedTrieNode(ALL_ALLOCATIONS_KEY, allocations)
	if err == util.ErrValueNotPresent {
		return nil, err
	}
	if err != nil {
		return nil, common.NewError(errorCode, err.Error())
	}
	return
}

func (sc *StorageSmartContract) getClientAllocations(
	clientID string, sci cstate.StateContextI) (allocations *Allocations, err error) {
	errorCode := "get_client_allocations"
	clientAllocation := &ClientAllocation{ClientID: clientID}
	err = sci.GetDecodedTrieNode(clientAllocation.GetKey(sc.ID), clientAllocation)
	if err == util.ErrValueNotPresent {
		return nil, err
	}
	if err != nil {
		return nil, common.NewError(errorCode, err.Error())
	}
	return clientAllocation.Allocations, nil
}

func (sc *StorageSmartContract) addAllocation(
	alloc *StorageAllocation, sci chainstate.StateContextI) (resp string, err error) {

	errorCode := "add_allocation"

	// ensure allocation doesn't exist already
	_, err = sc.getAllocation(alloc.ID, sci)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewErrorf(errorCode, "get_allocation: %v", err)
	}
	if err == nil {
		return "", common.NewErrorf(errorCode, "add_allocation: already exists: id=%v", alloc.ID)
	}

	// update all allocations in MPT
	allAllocations, err := sc.getAllAllocations(sci)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewErrorf(errorCode, "get_all_allocations: %v", err)
	}
	if err == util.ErrValueNotPresent {
		allAllocations = &Allocations{}
	}
	allAllocations.List.add(alloc.ID)
	_, err = sci.InsertTrieNode(ALL_ALLOCATIONS_KEY, allAllocations)
	if err != nil {
		return "", common.NewErrorf(errorCode, "save_allocations: %v", err)
	}

	// Update client allocations in MPT
	clientAllocations, err := sc.getClientAllocations(alloc.Owner, sci)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError(errorCode, err.Error())
	}
	if err == util.ErrValueNotPresent {
		clientAllocations = &Allocations{}
	}
	clientAllocations.List.add(alloc.ID)
	clientAllocation := &ClientAllocation{
		ClientID:    alloc.Owner,
		Allocations: clientAllocations,
	}
	_, err = sci.InsertTrieNode(clientAllocation.GetKey(sc.ID), clientAllocation)
	if err != nil {
		return "", common.NewErrorf(
			errorCode, "save_client_allocations: (client_id: %s): %v", alloc.Owner, err)
	}

	// Save new allocation in MPT
	_, err = sci.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewErrorf(errorCode, "save_new_allocation: %v", err)
	}

	return string(alloc.Encode()), nil
}

type newAllocationRequest struct {
	DataShards                 int              `json:"data_shards"`
	ParityShards               int              `json:"parity_shards"`
	Size                       int64            `json:"size"`
	Expiration                 common.Timestamp `json:"expiration_date"`
	Owner                      string           `json:"owner_id"`
	OwnerPublicKey             string           `json:"owner_public_key"`
	PreferredBlobbers          []string         `json:"preferred_blobbers"`
	ReadPriceRange             PriceRange       `json:"read_price_range"`
	WritePriceRange            PriceRange       `json:"write_price_range"`
	MaxChallengeCompletionTime time.Duration    `json:"max_challenge_completion_time"`
}

// storageAllocation from the request
func (nar *newAllocationRequest) storageAllocation() (sa *StorageAllocation) {
	sa = new(StorageAllocation)
	sa.DataShards = nar.DataShards
	sa.ParityShards = nar.ParityShards
	sa.Size = nar.Size
	sa.Expiration = nar.Expiration
	sa.Owner = nar.Owner
	sa.OwnerPublicKey = nar.OwnerPublicKey
	sa.PreferredBlobbers = nar.PreferredBlobbers
	sa.ReadPriceRange = nar.ReadPriceRange
	sa.WritePriceRange = nar.WritePriceRange
	sa.MaxChallengeCompletionTime = nar.MaxChallengeCompletionTime
	return
}

func (nar *newAllocationRequest) decode(b []byte) error {
	return json.Unmarshal(b, nar)
}

// Update blobbers' capacity used and stake
// (1) adjust blobber capacity used,
// (2) add offer (stake lock boundary),
// (3) save updated blobber
func (sc *StorageSmartContract) addBlobbersOffers(
	sa *StorageAllocation,
	blobbers []*StorageNode,
	sci chainstate.StateContextI,
) (err error) {
	errorCode := "add_blobbers_offers"
	for i, b := range blobbers {
		b.Used += sa.BlobberDetails[i].Size // adjust used size
		sp, err := sc.getStakePool(b.ID, sci)
		if err != nil {
			if err == util.ErrValueNotPresent {
				return common.NewErrorf(errorCode, "get_stake_pool: %v", err)
			}
			return common.NewError(errorCode, err.Error())
		}
		sp.addOffer(sa, sa.BlobberDetails[i])
		err = b.save(sc.ID, sci)
		if err != nil {
			return common.NewError(errorCode, err.Error())
		}
		err = sp.save(sc.ID, b.ID, sci)
		if err != nil {
			return common.NewError(errorCode, err.Error())
		}
	}
	return
}

// Update blobbers list in the all blobbers list
func updateAllBlobbers(
	sn *StorageNodes,
	updates []*StorageNode,
	sci chainstate.StateContextI,
) (err error) {
	for _, b := range updates {
		sn.Nodes.update(b)
		// don't replace if blobber has removed from the all blobbers list;
		// for example, if the blobber has removed, then it shouldn't be
		// in the all blobbers list
	}
	_, err = sci.InsertTrieNode(ALL_BLOBBERS_KEY, sn)
	if err != nil {
		return fmt.Errorf("can't save all blobber list: %v", err)
	}
	return
}

// convert time.Duration to common.Timestamp truncating to seconds
func toSeconds(dur time.Duration) common.Timestamp {
	return common.Timestamp(dur / time.Second)
}

// size in gigabytes
func bytesToGB(size int64) float64 {
	return float64(size) / GB
}

// exclude blobbers with not enough token in stake pool to fit the size
func (sc *StorageSmartContract) getBlobberWithFreeSpaceFilterFunc(
	now common.Timestamp,
	size int64,
	sci chainstate.StateContextI,
) (filter filterBlobberFunc) {
	return func(b *StorageNode) (kick bool) {
		var sp, err = sc.getStakePool(b.ID, sci)
		if err != nil {
			return true // kick off
		}
		if b.Terms.WritePrice == 0 {
			return false // keep, ok or already filtered by bid
		}
		// clean capacity (without delegate pools want to 'unstake')
		var free = sp.cleanCapacity(now, b.Terms.WritePrice)
		return free < size // kick off if it hasn't enough free space
	}
}

// newAllocationRequest creates new allocation
func (sc *StorageSmartContract) newAllocationRequest(
	t *transaction.Transaction,
	input []byte,
	sci chainstate.StateContextI,
) (resp string, err error) {

	errorCode := "new_allocation_request"

	if t.ClientID == "" {
		return "", common.NewError(
			errorCode, "missing client_id in transaction")
	}

	cfg, err := sc.getConfig(sci, true)
	if err != nil {
		return "", common.NewErrorf(errorCode, "get_config: %v", err)
	}

	allBlobbers, err := sc.getAllBlobbers(sci)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError(errorCode, err.Error())
	}
	if err == util.ErrValueNotPresent || len(allBlobbers.Nodes) == 0 {
		return "", common.NewError(errorCode, "no blobbers registered")
	}

	var request newAllocationRequest
	if err = request.decode(input); err != nil {
		return "", common.NewErrorf(errorCode, "malformed request: %v", err)
	}

	var sa = request.storageAllocation() // (set fields, including expiration)
	sa.TimeUnit = cfg.TimeUnit           // keep the initial time unit

	if err = sa.validate(t.CreationDate, cfg); err != nil {
		return "", common.NewErrorf(errorCode, "invalid request: %v", err)
	}

	var (
		numBlobbers      = sa.DataShards + sa.ParityShards                       // number of blobbers required
		sizeInBytes      = (sa.Size + int64(numBlobbers-1)) / int64(numBlobbers) // size of allocation for a blobber
		sizeInGB         = bytesToGB(sizeInBytes)                                // size in gigabytes
		filteredBlobbers = sa.filterBlobbers(                                    // filtered blobbers
			allBlobbers.Nodes,
			t.CreationDate,
			sizeInBytes,
			getHealthyBlobberFilterFunc(t.CreationDate),
			sc.getBlobberWithFreeSpaceFilterFunc(t.CreationDate, sizeInBytes, sci),
		)
	)

	if len(filteredBlobbers) < numBlobbers {
		return "", common.NewError(
			errorCode, "not enough blobbers to honor the allocation")
	}

	sa.Stats = &StorageAllocationStats{}
	sa.BlobberDetails = make([]*BlobberAllocation, 0)
	allocatedBlobbers := make([]*StorageNode, 0)

	var selectedBlobbers []*StorageNode
	if len(sa.PreferredBlobbers) > 0 {
		selectedBlobbers, err = getPreferredBlobbers(sa.PreferredBlobbers, filteredBlobbers)
		if err != nil {
			return "", common.NewError(errorCode, err.Error())
		}
	}

	// randomize blobber nodes
	if len(selectedBlobbers) < numBlobbers {
		seed, err := strconv.ParseInt(t.Hash[0:8], 16, 64)
		if err != nil {
			return "", common.NewError(errorCode, "failed to create seed to select random nodes")
		}
		selectedBlobbers = randomizeNodes(filteredBlobbers, selectedBlobbers, numBlobbers, seed)
	}

	selectedBlobbers = selectedBlobbers[:numBlobbers]

	for _, b := range selectedBlobbers {
		allocatedBlobbers = append(allocatedBlobbers, b)
		sa.BlobberDetails = append(sa.BlobberDetails, &BlobberAllocation{
			BlobberID:    b.ID,
			AllocationID: t.Hash,
			Terms:        b.Terms,
			Size:         sizeInBytes,
			Stats:        &StorageAllocationStats{},
			// MinLockDemand
			// Expiration and TimeUnit are already set for the 'sa' and
			// we can use the restDurationInTimeUnits method here
			MinLockDemand: b.Terms.minLockDemand(
				sizeInGB, sa.restDurationInTimeUnits(t.CreationDate)),
		})
		if b.Terms.ChallengeCompletionTime > sa.ChallengeCompletionTime {
			sa.ChallengeCompletionTime = b.Terms.ChallengeCompletionTime
		}
	}

	sort.SliceStable(allocatedBlobbers, func(i, j int) bool {
		return allocatedBlobbers[i].ID < allocatedBlobbers[j].ID
	})

	sa.ID = t.Hash
	sa.Tx = t.Hash // keep
	sa.Blobbers = allocatedBlobbers
	sa.StartTime = t.CreationDate // offer start time

	if err = sc.addBlobbersOffers(sa, selectedBlobbers, sci); err != nil {
		return "", common.NewError(errorCode, err.Error())
	}

	err = updateAllBlobbers(allBlobbers, selectedBlobbers, sci)
	if err != nil {
		return "", common.NewError(errorCode, err.Error())
	}

	// create write pool and lock tokens
	if err = sc.createWritePool(t, sa, sci); err != nil {
		return "", common.NewError(errorCode, err.Error())
	}

	// create challenge pool
	if err = sc.createChallengePool(t, sa, sci); err != nil {
		return "", common.NewError(errorCode, err.Error())
	}

	// save
	resp, err = sc.addAllocation(sa, sci)
	if err != nil {
		return "", common.NewError(errorCode, err.Error())
	}

	return
}

// update allocation request
type updateAllocationRequest struct {
	ID         string           `json:"id"`              // allocation id
	OwnerID    string           `json:"owner_id"`        // Owner of the allocation
	Size       int64            `json:"size"`            // difference
	Expiration common.Timestamp `json:"expiration_date"` // difference
}

func (uar *updateAllocationRequest) decode(b []byte) error {
	return json.Unmarshal(b, uar)
}

// validate request
func (uar *updateAllocationRequest) validate(conf *scConfig,
	alloc *StorageAllocation) (err error) {

	if uar.Size == 0 && uar.Expiration == 0 {
		return errors.New("update allocation changes nothing")
	}

	if ns := alloc.Size + uar.Size; ns < conf.MinAllocSize {
		return fmt.Errorf("new allocation size is too small: %d < %d",
			ns, conf.MinAllocSize)
	}

	if len(alloc.BlobberDetails) == 0 {
		return errors.New("invalid allocation for updating: no blobbers")
	}

	return
}

// calculate size difference for every blobber of the allocations
func (uar *updateAllocationRequest) getBlobbersSizeDiff(
	alloc *StorageAllocation) (diff int64) {

	var size = alloc.DataShards + alloc.ParityShards
	if uar.Size > 0 {
		diff = (uar.Size + int64(size-1)) / int64(size)
	} else if uar.Size < 0 {
		diff = (uar.Size - int64(size-1)) / int64(size)
	}
	// else -> (0), no changes, avoid unnecessary calculation

	return
}

// new size of blobbers' allocation
func (uar *updateAllocationRequest) getNewBlobbersSize(
	alloc *StorageAllocation) (newSize int64) {

	return alloc.BlobberDetails[0].Size + uar.getBlobbersSizeDiff(alloc)
}

// getAllocationBlobbers loads blobbers of an allocation from store
func (sc *StorageSmartContract) getAllocationBlobbers(
	alloc *StorageAllocation,
	sci chainstate.StateContextI) (blobbers []*StorageNode, err error) {

	errorCode := "get_allocation_blobbers"

	blobbers = make([]*StorageNode, 0, len(alloc.BlobberDetails))

	for _, details := range alloc.BlobberDetails {
		var blobber *StorageNode
		blobber, err = sc.getBlobber(details.BlobberID, sci)
		if err != nil {
			return nil, common.NewError(errorCode, err.Error())
		}
		blobbers = append(blobbers, blobber)
	}

	return
}

// closeAllocation making it expired; the allocation will be alive the
// challenge_completion_time and be closed then
func (sc *StorageSmartContract) closeAllocation(t *transaction.Transaction,
	alloc *StorageAllocation, balances chainstate.StateContextI) (
	resp string, err error) {

	if alloc.Expiration-t.CreationDate <
		toSeconds(alloc.ChallengeCompletionTime) {
		return "", common.NewError("allocation_closing_failed",
			"doesn't need to close allocation is about to expire")
	}

	// mark as expired, but it will be alive at least chellenge_competion_time
	alloc.Expiration = t.CreationDate

	// stake pool (offers)

	for _, ba := range alloc.BlobberDetails {
		if err = sc.updateSakePoolOffer(ba, alloc, balances); err != nil {
			return "", common.NewError("allocation_closing_failed",
				err.Error())
		}
	}

	// save allocation

	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("allocation_closing_failed",
			"can't save allocation: "+err.Error())
	}

	return string(alloc.Encode()), nil // closing
}

func (sc *StorageSmartContract) saveUpdatedAllocation(all *StorageNodes,
	alloc *StorageAllocation, blobbers []*StorageNode,
	balances chainstate.StateContextI) (err error) {

	// save all
	if err = updateAllBlobbers(all, blobbers, balances); err != nil {
		return
	}

	// save related blobbers
	for _, b := range blobbers {
		if _, err = balances.InsertTrieNode(b.GetKey(sc.ID), b); err != nil {
			return
		}
	}

	// save allocation
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return
	}

	return
}

// allocation period used to calculate weighted average prices
type allocPeriod struct {
	read   state.Balance    // read price
	write  state.Balance    // write price
	period common.Timestamp // period (duration)
	size   int64            // size for period
}

func (ap *allocPeriod) weight() float64 {
	return float64(ap.period) * float64(ap.size)
}

// returns weighted average read and write prices
func (ap *allocPeriod) join(np *allocPeriod) (avgRead, avgWrite state.Balance) {
	var (
		apw, npw = ap.weight(), np.weight() // weights
		ws       = apw + npw                // weights sum
		rp, wp   float64                    // read sum, write sum (weighted)
	)

	rp = (float64(ap.read) * apw) + (float64(np.read) * npw)
	wp = (float64(ap.write) * apw) + (float64(np.write) * npw)

	avgRead = state.Balance(rp / ws)
	avgWrite = state.Balance(wp / ws)
	return
}

func weightedAverage(prev, next *Terms, tx, pexp, expDiff common.Timestamp,
	psize, sizeDiff int64) (avg Terms) {

	// allocation periods
	var left, added allocPeriod
	left.read, left.write = prev.ReadPrice, prev.WritePrice   // } prices
	added.read, added.write = next.ReadPrice, next.WritePrice // }
	left.size, added.size = psize, psize+sizeDiff             // sizes
	left.period, added.period = pexp-tx, pexp+expDiff-tx      // periods
	// join
	avg.ReadPrice, avg.WritePrice = left.join(&added)

	// just copy from next
	avg.MinLockDemand = next.MinLockDemand
	avg.MaxOfferDuration = next.MaxOfferDuration
	avg.ChallengeCompletionTime = next.ChallengeCompletionTime
	return
}

// The adjustChallengePool moves more or moves some tokens back from or to
// challenge pool during allocation extending or reducing.
func (sc *StorageSmartContract) adjustChallengePool(alloc *StorageAllocation,
	wp *writePool, odr, ndr common.Timestamp, oterms []Terms,
	now common.Timestamp, balances chainstate.StateContextI) (err error) {

	var (
		changes = alloc.challengePoolChanges(odr, ndr, oterms)
		cp      *challengePool
	)

	if cp, err = sc.getChallengePool(alloc.ID, balances); err != nil {
		return fmt.Errorf("adjust_challenge_pool: %v", err)
	}

	var changed bool

	for i, ch := range changes {
		var blobID = alloc.BlobberDetails[i].BlobberID
		switch {
		case ch > 0:
			err = wp.moveToChallenge(alloc.ID, blobID, cp, now, ch)
			changed = true
		case ch < 0:
			// only if the challenge pool has the tokens; all the tokens
			// can be moved back already, or moved to a blobber due to
			// challenge process
			if cp.Balance >= -ch {
				err = cp.moveToWritePool(alloc.ID, blobID, alloc.Until(), wp,
					-ch)
				changed = true
			}
		default:
			// no changes for the blobber
		}
		if err != nil {
			return fmt.Errorf("adjust_challenge_pool: %v", err)
		}
	}

	if changed {
		err = cp.save(sc.ID, alloc.ID, balances)
	}

	return
}

// extendAllocation extends size or/and expiration (one of them can be reduced);
// here we use new terms of blobbers
func (sc *StorageSmartContract) extendAllocation(
	t *transaction.Transaction,
	all *StorageNodes,
	alloc *StorageAllocation,
	blobbers []*StorageNode,
	uar *updateAllocationRequest,
	sci chainstate.StateContextI) (resp string, err error) {

	var (
		diff   = uar.getBlobbersSizeDiff(alloc) // size difference
		size   = uar.getNewBlobbersSize(alloc)  // blobber size
		gbSize = bytesToGB(size)                // blobber size in GB
		cct    time.Duration                    // new challenge_completion_time

		// keep original terms to adjust challenge pool value
		oterms = make([]Terms, 0, len(alloc.BlobberDetails))
		// original allocation duration remains
		odr = alloc.Expiration - t.CreationDate
	)

	// adjust the expiration if changed, boundaries has already checked
	var prevExpiration = alloc.Expiration
	alloc.Expiration += uar.Expiration // new expiration
	alloc.Size += uar.Size             // new size

	// 1. update terms
	for i, details := range alloc.BlobberDetails {
		oterms = append(oterms, details.Terms) // keep original terms will be changed

		var b = blobbers[i]
		if b.Capacity == 0 {
			return "", common.NewErrorf("allocation_extending_failed",
				"blobber %s no longer provides its service", b.ID)
		}
		if uar.Size > 0 {
			if b.Capacity-b.Used-diff < 0 {
				return "", common.NewErrorf("allocation_extending_failed",
					"blobber %s doesn't have enough free space", b.ID)
			}
		}

		b.Used += diff // new capacity used

		// update terms using weighted average
		details.Terms = weightedAverage(&details.Terms, &b.Terms,
			t.CreationDate, prevExpiration, alloc.Expiration, details.Size,
			diff)

		details.Size = size // new size

		if uar.Expiration > toSeconds(b.Terms.MaxOfferDuration) {
			return "", common.NewErrorf("allocation_extending_failed",
				"blobber %s doesn't allow so long offers", b.ID)
		}

		if b.Terms.ChallengeCompletionTime > cct {
			cct = b.Terms.ChallengeCompletionTime // seek max CCT
		}

		// since, new terms is weighted average based on previous terms and
		// past allocation time and new terms and new allocation time; then
		// we can easily recalculate new min_lock_demand value from allocation
		// start to its new end using the new weighted average terms; but, we
		// can't reduce the min_lock_demand_value; that's all;

		// new blobber's min lock demand (alloc.Expiration is already updated
		// and we can use restDurationInTimeUnits method here)
		var nbmld = details.Terms.minLockDemand(gbSize,
			alloc.restDurationInTimeUnits(alloc.StartTime))

		// min_lock_demand can be increased only
		if nbmld > details.MinLockDemand {
			details.MinLockDemand = nbmld
		}
	}

	// update max challenge_completion_time
	alloc.ChallengeCompletionTime = cct

	// extend offers after alloc.challenge_completion_time is known
	for _, ba := range alloc.BlobberDetails {
		if err = sc.updateSakePoolOffer(ba, alloc, sci); err != nil {
			return "", common.NewError("allocation_extending_failed",
				err.Error())
		}
	}

	// get related write pool
	var wp *writePool
	if wp, err = sc.getWritePool(alloc.Owner, sci); err != nil {
		return "", common.NewErrorf("allocation_extending_failed",
			"can't get write pool: %v", err)
	}

	var until = alloc.Until()

	// lock tokens if this transaction provides them
	if t.Value > 0 {
		if err = checkFill(t, sci); err != nil {
			return "", common.NewError("allocation_extending_failed",
				err.Error())
		}
		if _, err = wp.fill(t, alloc, until, sci); err != nil {
			return "", common.NewErrorf("allocation_extending_failed",
				"write pool filling: %v", err)
		}
	}

	// is it about size increasing? if so, we should make sure the write
	// pool has enough tokens
	if diff > 0 {
		if mldLeft := alloc.restMinLockDemand(); mldLeft > 0 {
			if wp.allocUntil(alloc.ID, until) < mldLeft {
				return "", common.NewError("allocation_extending_failed",
					"not enough tokens in write pool to extend allocation")
			}
		}
	}

	// add more tokens to related challenge pool, or move some tokens back
	var ndr = alloc.Expiration - t.CreationDate
	err = sc.adjustChallengePool(alloc, wp, odr, ndr, oterms, t.CreationDate,
		sci)
	if err != nil {
		return "", common.NewErrorf("allocation_extending_failed", "%v", err)
	}

	// save the write pool
	if err = wp.save(sc.ID, alloc.Owner, sci); err != nil {
		return "", common.NewErrorf("allocation_extending_failed", "%v", err)
	}

	// save all

	err = sc.saveUpdatedAllocation(all, alloc, blobbers, sci)
	if err != nil {
		return "", common.NewErrorf("allocation_extending_failed", "%v", err)
	}

	return string(alloc.Encode()), nil
}

// reduceAllocation reduces size or/and expiration (no one can be increased);
// here we use the same terms of related blobbers
func (sc *StorageSmartContract) reduceAllocation(t *transaction.Transaction,
	all *StorageNodes, alloc *StorageAllocation, blobbers []*StorageNode,
	uar *updateAllocationRequest, balances chainstate.StateContextI) (
	resp string, err error) {

	var (
		diff = uar.getBlobbersSizeDiff(alloc) // size difference
		size = uar.getNewBlobbersSize(alloc)  // blobber size

		// original allocation duration remains
		odr = alloc.Expiration - t.CreationDate
	)

	// adjust the expiration if changed, boundaries has already checked
	alloc.Expiration += uar.Expiration
	alloc.Size += uar.Size

	// 1. update terms
	for i, ba := range alloc.BlobberDetails {
		var b = blobbers[i]
		b.Used += diff // new capacity used
		ba.Size = size // new size
		// update stake pool
		if err = sc.updateSakePoolOffer(ba, alloc, balances); err != nil {
			return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
		}
	}

	// get related write pool
	var wp *writePool
	if wp, err = sc.getWritePool(alloc.Owner, balances); err != nil {
		return "", common.NewErrorf("allocation_reducing_failed",
			"can't get write pool: %v", err)
	}

	// lock tokens if this transaction provides them
	if t.Value > 0 {
		if err = checkFill(t, balances); err != nil {
			return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
		}
		var until = alloc.Until()
		if _, err = wp.fill(t, alloc, until, balances); err != nil {
			return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
		}
	}

	// new allocation duration remains
	var ndr = alloc.Expiration - t.CreationDate
	err = sc.adjustChallengePool(alloc, wp, odr, ndr, nil, t.CreationDate,
		balances)
	if err != nil {
		return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
	}

	// save the write pool
	if err = wp.save(sc.ID, alloc.Owner, balances); err != nil {
		return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
	}

	// save all

	err = sc.saveUpdatedAllocation(all, alloc, blobbers, balances)
	if err != nil {
		return "", common.NewErrorf("allocation_reducing_failed", "%v", err)
	}

	return string(alloc.Encode()), nil
}

// Update allocation allows to change allocation size or expiration;
// if expiration reduced or unchanged, then existing terms of blobbers used,
// otherwise new terms used; also, it locks additional tokens if size is
// extended and it checks blobbers for required stake;
func (sc *StorageSmartContract) updateAllocationRequest(
	t *transaction.Transaction, input []byte,
	sci chainstate.StateContextI) (resp string, err error) {

	errorCode := "update_allocation"

	if t.ClientID == "" {
		return "", common.NewError(
			errorCode, "missing client_id in transaction")
	}

	cfg, err := sc.getConfig(sci, false)
	if err != nil {
		return "", common.NewErrorf(errorCode, err.Error())
	}

	allBlobbers, err := sc.getAllBlobbers(sci)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError(errorCode, err.Error())
	}
	if err == util.ErrValueNotPresent || len(allBlobbers.Nodes) == 0 {
		return "", common.NewError(errorCode, "no blobbers registered")
	}

	var request updateAllocationRequest
	if err = request.decode(input); err != nil {
		return "", common.NewErrorf(errorCode, "invalid request: %v", err)
	}

	if request.OwnerID == "" {
		request.OwnerID = t.ClientID
		// return "", common.NewError(errorCode, 		//	"invalid request: missing owner_id")
	}

	clientAllocations, err := sc.getClientAllocations(request.OwnerID, sci)
	if err != nil && err != util.ErrValueNotPresent {
		return "", common.NewError(errorCode, err.Error())
	}
	if err == util.ErrValueNotPresent || len(clientAllocations.List) == 0 {
		return "", common.NewError(errorCode, "no client allocations")
	}

	if !clientAllocations.has(request.ID) {
		return "", common.NewErrorf(errorCode,
			"can't find allocation in client's allocations list: %s (%d)",
			request.ID, len(clientAllocations.List))
	}

	alloc, err := sc.getAllocation(request.ID, sci)
	if err != nil {
		return "", common.NewError(errorCode, err.Error())
	}

	if err = request.validate(cfg, alloc); err != nil {
		return "", common.NewError(errorCode, err.Error())
	}

	// can't update expired allocation
	if alloc.Expiration < t.CreationDate {
		return "", common.NewError(errorCode, "can't update expired allocation")
	}

	// get blobber of the allocation to update them
	var blobbers []*StorageNode
	if blobbers, err = sc.getAllocationBlobbers(alloc, sci); err != nil {
		return "", common.NewError(errorCode, err.Error())
	}

	// adjust expiration
	var newExpiration = alloc.Expiration + request.Expiration

	// update allocation transaction hash
	alloc.Tx = t.Hash

	// close allocation now
	if newExpiration <= t.CreationDate {
		return sc.closeAllocation(t, alloc, sci)
	}

	// an allocation can't be shorter than configured in SC
	// (prevent allocation shortening for entire period)
	if request.Expiration < 0 &&
		newExpiration-t.CreationDate < toSeconds(cfg.MinAllocDuration) {

		return "", common.NewError(errorCode, "allocation duration becomes too short")
	}

	if request.Size < 0 && alloc.Size+request.Size < cfg.MinAllocSize {
		return "", common.NewError(errorCode, "allocation size becomes too small")
	}

	// if size or expiration increased, then we use new terms
	// otherwise, we use the same terms
	if request.Size > 0 || request.Expiration > 0 {
		return sc.extendAllocation(t, allBlobbers, alloc, blobbers, &request, sci)
	}

	return sc.reduceAllocation(t, allBlobbers, alloc, blobbers, &request, sci)
}

func getPreferredBlobbers(
	preferredBlobbers []string,
	allBlobbers []*StorageNode) (selectedBlobbers []*StorageNode, err error) {
	blobberByURL := make(map[string]*StorageNode)
	for _, sn := range allBlobbers {
		blobberByURL[sn.BaseURL] = sn
	}
	for _, blobberURL := range preferredBlobbers {
		selectedBlobber, ok := blobberByURL[blobberURL]
		if !ok {
			err = common.NewError("allocation_creation_failed", "Invalid preferred blobber URL")
			return
		}
		selectedBlobbers = append(selectedBlobbers, selectedBlobber)
	}
	return
}

func randomizeNodes(in []*StorageNode, out []*StorageNode, n int, seed int64) []*StorageNode {
	nOut := minInt(len(in), n)
	nOut = maxInt(1, nOut)
	randGen := rand.New(rand.NewSource(seed))
	for {
		i := randGen.Intn(len(in))
		if !checkExists(in[i], out) {
			out = append(out, in[i])
		}
		if len(out) >= nOut {
			break
		}
	}
	return out
}

func minInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func checkExists(c *StorageNode, sl []*StorageNode) bool {
	for _, s := range sl {
		if s.ID == c.ID {
			return true
		}
	}
	return false
}

func passOnes(n int) (po []float64) {
	po = make([]float64, n)
	for i := range po {
		po[i] = 1.0
	}
	return
}

// a blobber can not send a challenge response, thus we have to check out
// challenge requests and their expiration
func (sc *StorageSmartContract) adjustChallenges(alloc *StorageAllocation,
	now common.Timestamp, balances chainstate.StateContextI) (
	passRates []float64, err error) {

	// are there chellenges
	if alloc.Stats == nil || alloc.Stats.OpenChallenges == 0 {
		return passOnes(len(alloc.BlobberDetails)), nil // no open challenges
	}
	passRates = make([]float64, 0, len(alloc.BlobberDetails))
	// range over all related blobbers
	for _, d := range alloc.BlobberDetails {
		// check out blobber challenges
		var (
			bc               *BlobberChallenge //
			success, failure int               // for the ending
		)
		bc, err = sc.getBlobberChallenge(d.BlobberID, balances)
		if err != nil && err != util.ErrValueNotPresent {
			return nil, fmt.Errorf("getting blobber challenge: %v", err)
		}
		// no blobber challenges, no failures
		if err == util.ErrValueNotPresent {
			passRates, err = append(passRates, 1.0), nil
			continue // no challenges for the blobber
		}
		if d.Stats == nil {
			d.Stats = new(StorageAllocationStats) // make sure
		}
		// all expired open challenges are failed, all other
		// challenges we are treating as successful
		for _, c := range bc.Challenges {
			if c.Response != nil {
				continue // already accepted, already rewarded/penalized
			}
			var expire = c.Created + toSeconds(d.Terms.ChallengeCompletionTime)
			if expire < now {
				alloc.Stats.FailedChallenges++
				d.Stats.FailedChallenges++
				failure++
			} else {
				alloc.Stats.SuccessChallenges++
				d.Stats.SuccessChallenges++
				success++
			}
			d.Stats.OpenChallenges--
			alloc.Stats.OpenChallenges--
		}
		if success == 0 && failure == 0 {
			passRates = append(passRates, 1.0)
			continue
		}
		// success rate for the blobber allocation
		passRates = append(passRates, float64(success)/float64(success+failure))
	}

	return // ok
}

// If blobbers doesn't provide their services, then user can use this
// cancel_allocation transaction to close allocation and unlock all tokens
// of write pool back to himself. The cacnel_allocation doesn't pays min_lock
// demand to blobbers.
func (sc *StorageSmartContract) cancelAllocationRequest(
	t *transaction.Transaction,
	input []byte,
	sci chainstate.StateContextI) (resp string, err error) {

	var req lockRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("alloc_cacnel_failed", err.Error())
	}

	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(req.AllocationID, sci)
	if err != nil {
		return "", common.NewError("alloc_cacnel_failed", err.Error())
	}

	if alloc.Owner != t.ClientID {
		return "", common.NewError("alloc_cacnel_failed",
			"only owner can cancel an allocation")
	}

	if alloc.Expiration < t.CreationDate {
		return "", common.NewError("alloc_cacnel_failed",
			"trying to cancel expired allocation")
	}

	var passRates []float64
	passRates, err = sc.adjustChallenges(alloc, t.CreationDate, sci)
	if err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"calculating rest challenges success/fail rates: "+err.Error())
	}

	// SC configurations
	var conf *scConfig
	if conf, err = sc.getConfig(sci, false); err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"can't get SC configurations: "+err.Error())
	}

	if fctc := conf.FailedChallengesToCancel; fctc > 0 {
		if alloc.Stats == nil || alloc.Stats.FailedChallenges < int64(fctc) {
			return "", common.NewError("alloc_cacnel_failed",
				"not enough failed challenges of allocation to cancel")
		}
	}

	// can cancel

	// write pool
	var wp *writePool
	if wp, err = sc.getWritePool(alloc.Owner, sci); err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"can't get user's write pools: "+err.Error())
	}

	// challenge pool
	var cp *challengePool
	if cp, err = sc.getChallengePool(req.AllocationID, sci); err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"can't get related challenge pool: "+err.Error())
	}

	// blobbers
	var blobbers []*StorageNode
	if blobbers, err = sc.getAllocationBlobbers(alloc, sci); err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"invalid state: can't get related blobbers: "+err.Error())
	}

	// new values
	alloc.Expiration, alloc.ChallengeCompletionTime = t.CreationDate, 0

	var allb *StorageNodes
	if allb, err = sc.getAllBlobbers(sci); err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"can't get all blobbers list: "+err.Error())
	}

	var until = alloc.Until()

	// we can use the i for the blobbers list above because of algorithm
	// of the getAllocationBlobbers method; also, we can use the i in the
	// passRates list above because of algorithm of the adjustChallenges
	var left = cp.Balance // tokens left in related challenge pool
	for i, d := range alloc.BlobberDetails {
		var (
			sp       *stakePool     // related stake pool
			b        = blobbers[i]  // related blobber
			passRate = passRates[i] // related success rate
		)
		if sp, err = sc.getStakePool(d.BlobberID, sci); err != nil {
			return "", common.NewError("alloc_cacnel_failed",
				"can't get stake pool of "+d.BlobberID+": "+err.Error())
		}
		if err = sp.extendOffer(alloc, d); err != nil {
			return "", common.NewError("alloc_cacnel_failed",
				"removing stake pool offer for "+d.BlobberID+": "+err.Error())
		}
		// challenge pool rest
		if alloc.usedSize > 0 && left > 0 && passRate > 0 && d.Stats != nil {
			var (
				ratio = float64(d.Stats.UsedSize) / float64(alloc.usedSize)
				move  = state.Balance(float64(left) * ratio * passRate)
			)
			if err = cp.moveToBlobber(sc.ID, sp, move, sci); err != nil {
				return "", common.NewError("alloc_cacnel_failed",
					"moving tokens to stake pool of "+d.BlobberID+": "+
						err.Error())
			}
			d.Spent += move       // }
			d.FinalReward += move // } stat
		}
		// min lock demand rest
		var fctrml = conf.FailedChallengesToRevokeMinLock
		if d.Stats == nil || d.Stats.FailedChallenges < int64(fctrml) {
			if lack := d.MinLockDemand - d.Spent; lack > 0 {
				err = wp.moveToStake(sc.ID, alloc.ID, d.BlobberID, sp,
					until, lack, sci)
				if err != nil {
					return "", common.NewError("alloc_cacnel_failed",
						"paying min_lock for "+d.BlobberID+": "+err.Error())
				}
				d.Spent += lack
				d.FinalReward += lack
			}
		}
		// -------
		var info *stakePoolUpdateInfo
		info, err = sp.update(conf, sc.ID, t.CreationDate, sci)
		if err != nil {
			return "", common.NewError("alloc_cacnel_failed",
				"updating stake pool of "+d.BlobberID+": "+err.Error())
		}
		if err = sp.save(sc.ID, d.BlobberID, sci); err != nil {
			return "", common.NewError("alloc_cacnel_failed",
				"saving stake pool of "+d.BlobberID+": "+err.Error())
		}
		conf.Minted += info.minted
		// update the blobber
		b.Used -= d.Size
		if _, err = sci.InsertTrieNode(b.GetKey(sc.ID), b); err != nil {
			return "", common.NewError("alloc_cacnel_failed",
				"saving blobber "+d.BlobberID+": "+err.Error())
		}
		// update the blobber in all (replace with existing one)
		allb.Nodes.update(b)
	}

	// move challenge pool left to write pool
	alloc.MovedBack += cp.Balance
	err = cp.moveToWritePool(alloc.ID, "", alloc.Until(), wp, cp.Balance)
	if err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"moving challenge pool rest back to write pool: "+err.Error())
	}

	// save all blobbers list
	_, err = sci.InsertTrieNode(ALL_BLOBBERS_KEY, allb)
	if err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"saving all blobbers list: "+err.Error())
	}

	// save all rest and remove allocation from all allocations list

	if err = cp.save(sc.ID, alloc.ID, sci); err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"saving challenge pool: "+err.Error())
	}

	if err = wp.save(sc.ID, alloc.Owner, sci); err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"saving write pool: "+err.Error())
	}

	alloc.Finalized, alloc.Canceled = true, true
	_, err = sci.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"saving allocation: "+err.Error())
	}

	var all *Allocations
	if all, err = sc.getAllAllocations(sci); err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"getting all allocations list: "+err.Error())
	}

	if !all.List.remove(alloc.ID) {
		return "", common.NewError("alloc_cacnel_failed",
			"invalid state: allocation not found in all allocations list")
	}

	_, err = sci.InsertTrieNode(ALL_ALLOCATIONS_KEY, all)
	if err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"saving all allocations list: "+err.Error())
	}

	// save configuration (minted tokens)
	_, err = sci.InsertTrieNode(scConfigKey(sc.ID), conf)
	if err != nil {
		return "", common.NewError("alloc_cacnel_failed",
			"saving configurations: "+err.Error())
	}

	return "canceled", nil
}

//
// finalize an allocation (after expire + challenge completion time)
//

// 1. challenge pool                  -> blobbers or write pool
// 2. write pool min_lock_demand left -> blobbers
// 3. remove offer from blobber (stake pool)
// 4. update blobbers used and in all blobbers list too
// 5. write pool                      -> client
func (sc *StorageSmartContract) finalizeAllocation(
	t *transaction.Transaction, input []byte,
	balances chainstate.StateContextI) (resp string, err error) {

	var req lockRequest
	if err = req.decode(input); err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	var alloc *StorageAllocation
	alloc, err = sc.getAllocation(req.AllocationID, balances)
	if err != nil {
		return "", common.NewError("fini_alloc_failed", err.Error())
	}

	// should be owner or one of blobbers of the allocation
	if !alloc.IsValidFinalizer(t.ClientID) {
		return "", common.NewError("fini_alloc_failed",
			"not allowed, unknown finalization initiator")
	}

	// should not be finalized
	if alloc.Finalized {
		return "", common.NewError("fini_alloc_failed",
			"allocation already finalized")
	}

	// should be expired
	var until = alloc.Until()
	if until > t.CreationDate {
		return "", common.NewError("fini_alloc_failed",
			"allocation is not expired yet, or waiting a challenge completion")
	}

	var passRates []float64
	passRates, err = sc.adjustChallenges(alloc, t.CreationDate, balances)
	if err != nil {
		return "", common.NewError("fini_alloc_failed",
			"calculating rest challenges success/fail rates: "+err.Error())
	}

	// SC configurations
	var conf *scConfig
	if conf, err = sc.getConfig(balances, false); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"can't get SC configurations: "+err.Error())
	}

	// write pool
	var wp *writePool
	if wp, err = sc.getWritePool(alloc.Owner, balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"can't get user's write pools: "+err.Error())
	}

	// challenge pool
	var cp *challengePool
	if cp, err = sc.getChallengePool(req.AllocationID, balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"can't get related challenge pool: "+err.Error())
	}

	// blobbers
	var blobbers []*StorageNode
	if blobbers, err = sc.getAllocationBlobbers(alloc, balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"invalid state: can't get related blobbers: "+err.Error())
	}

	var allb *StorageNodes
	if allb, err = sc.getAllBlobbers(balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"can't get all blobbers list: "+err.Error())
	}

	// we can use the i for the blobbers list above because of algorithm
	// of the getAllocationBlobbers method; also, we can use the i in the
	// passRates list above because of algorithm of the adjustChallenges
	var left = cp.Balance // tokens left in related challenge pool
	for i, d := range alloc.BlobberDetails {
		var (
			sp       *stakePool     // related stake pool
			b        = blobbers[i]  // related blobber
			passRate = passRates[i] // related success rate
		)
		if sp, err = sc.getStakePool(d.BlobberID, balances); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"can't get stake pool of "+d.BlobberID+": "+err.Error())
		}
		// challenge pool rest
		if alloc.usedSize > 0 && left > 0 && passRate > 0 && d.Stats != nil {
			var (
				ratio = float64(d.Stats.UsedSize) / float64(alloc.usedSize)
				move  = state.Balance(float64(left) * ratio * passRate)
			)
			if err = cp.moveToBlobber(sc.ID, sp, move, balances); err != nil {
				return "", common.NewError("fini_alloc_failed",
					"moving tokens to stake pool of "+d.BlobberID+": "+
						err.Error())
			}
			d.Spent += move       // }
			d.FinalReward += move // } stat
		}
		// min lock demand rest
		var fctrml = conf.FailedChallengesToRevokeMinLock
		if d.Stats == nil || d.Stats.FailedChallenges < int64(fctrml) {
			if lack := d.MinLockDemand - d.Spent; lack > 0 {
				err = wp.moveToStake(sc.ID, alloc.ID, d.BlobberID, sp,
					until, lack, balances)
				if err != nil {
					return "", common.NewError("alloc_cacnel_failed",
						"paying min_lock for "+d.BlobberID+": "+err.Error())
				}
				d.Spent += lack
				d.FinalReward += lack
			}
		}
		// -------
		var info *stakePoolUpdateInfo
		info, err = sp.update(conf, sc.ID, t.CreationDate, balances)
		if err != nil {
			return "", common.NewError("fini_alloc_failed",
				"updating stake pool of "+d.BlobberID+": "+err.Error())
		}
		if err = sp.save(sc.ID, d.BlobberID, balances); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"saving stake pool of "+d.BlobberID+": "+err.Error())
		}
		conf.Minted += info.minted
		// update the blobber
		b.Used -= d.Size
		if _, err = balances.InsertTrieNode(b.GetKey(sc.ID), b); err != nil {
			return "", common.NewError("fini_alloc_failed",
				"saving blobber "+d.BlobberID+": "+err.Error())
		}
		// update the blobber in all (replace with existing one)
		allb.Nodes.update(b)
	}

	// move challenge pool rest to write pool
	alloc.MovedBack += cp.Balance
	err = cp.moveToWritePool(alloc.ID, "", alloc.Until(), wp, cp.Balance)
	if err != nil {
		return "", common.NewError("fini_alloc_failed",
			"moving challenge pool rest back to write pool: "+err.Error())
	}

	// save all blobbers list
	_, err = balances.InsertTrieNode(ALL_BLOBBERS_KEY, allb)
	if err != nil {
		return "", common.NewError("fini_alloc_failed",
			"saving all blobbers list: "+err.Error())
	}

	// save all rest and remove allocation from all allocations list

	if err = cp.save(sc.ID, alloc.ID, balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"saving challenge pool: "+err.Error())
	}

	if err = wp.save(sc.ID, alloc.Owner, balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"saving write pool: "+err.Error())
	}

	alloc.Finalized = true
	_, err = balances.InsertTrieNode(alloc.GetKey(sc.ID), alloc)
	if err != nil {
		return "", common.NewError("fini_alloc_failed",
			"saving allocation: "+err.Error())
	}

	var all *Allocations
	if all, err = sc.getAllAllocations(balances); err != nil {
		return "", common.NewError("fini_alloc_failed",
			"getting all allocations list: "+err.Error())
	}

	if !all.List.remove(alloc.ID) {
		return "", common.NewError("fini_alloc_failed",
			"invalid state: allocation not found in all allocations list")
	}

	_, err = balances.InsertTrieNode(ALL_ALLOCATIONS_KEY, all)
	if err != nil {
		return "", common.NewError("fini_alloc_failed",
			"saving all allocations list: "+err.Error())
	}

	// save configuration (minted tokens)
	_, err = balances.InsertTrieNode(scConfigKey(sc.ID), conf)
	if err != nil {
		return "", common.NewError("fini_alloc_failed",
			"saving configurations: "+err.Error())
	}

	return "finalized", nil
}
