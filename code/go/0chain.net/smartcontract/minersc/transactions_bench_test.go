package minersc

import (
	"sort"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"

	"github.com/stretchr/testify/require"
)

func init() {
	config.DevConfiguration.ViewChange = true
}

// # measure transactions performance #
//
// for 1k miners/sharder and settings
//
//    min_s: 1
//    max_s: 1000
//
// one of outside sharders comes up by its stake; e.g. there is 1k active
// sharders and 1001 at all, one of the sharders is outside and can't join
// because of of the max_s configurations; but adding stake for the outside
// sharder we can make it active, kicking out one of other sharders
//
// go test -v -timeout 1h -bench swapSharderByStake

//    min_n: 1
//    max_n: 1000
//
// one of outside miners comes up by its stake; e.g. there is 1k active
// miners and 1001 at all, one of the miners is outside and can't join
// because of of the max_n configurations; but adding stake for the outside
// miner we can make it active, kicking out one of other miners
//
// go test -v -timeout 1h -bench swapMinerByStake
//

// affects:
//   phases functions:
//     - moveToContribute                           [✓]
//     - moveToShareOrPublish                       [✓]
//   sharders transactions:
//     - sharderKeep                                [✓]
//   miner sc transactions:
//     - createMagicBlockForWait (publish phase)    [✓]
//     - viewChangePoolsWork -> payFees             [✓]

// benchmark                                              iter    time/iter
// ---------                                              ----    -------------
// Benchmark_swapSharderByStake/moveToContribute-8         74     15.92   ms/op
// Benchmark_swapSharderByStake/sharderKeep-8              27     44.39   ms/op
// Benchmark_swapSharderByStake/moveToShareOrPublish-8     81     14.57   ns/op
// Benchmark_swapSharderByStake/createMagicBlockForWait-8  142    8.33    ms/op
// Benchmark_swapSharderByStake/VCPoolsWork->payFees-8     1      1681.8  ms/op
// Benchmark_swapMinerByStake/moveToContribute-8           75     15.97   ms/op
// Benchmark_swapMinerByStake/moveToShareOrPublish-8       79     14.92   ms/op
// Benchmark_swapMinerByStake/createMagicBlockForWait-8    136    8.73    ms/op
// Benchmark_swapMinerByStake/VCPoolsWork->payFees-8       1      1045.98 ms/op

func Benchmark_swapSharderByStake(b *testing.B) {

	const stakeVal, stakeHolders = 10e10, 5

	var (
		balances = newTestBalances(b, true)
		msc      = newTestMinerSC()
		now      int64
		err      error
		tp       int64 // time point

		gn *GlobalNode // configurations

		miners   = make([]*miner, 0, 1000)
		sharders = make([]*sharder, 0, 1001)
	)

	defer balances.mpts.Close()
	balances.skipMerge = true

	gn = setConfig(b, balances)
	gn.MinS = 1
	gn.MaxS = 1000
	gn.MinN = 3
	gn.MaxN = 1000
	mustSave(b, GlobalNodeKey, gn, balances)

	// add miners
	for i := 0; i < cap(miners); i++ {
		miners = append(miners, newMiner(b, msc, now, stakeHolders,
			stakeVal, balances))
		now += 10
	}

	// add sharders
	for i := 0; i < cap(sharders); i++ {
		sharders = append(sharders, newSharder(b, msc, now, stakeHolders,
			stakeVal, balances))
		now += 10
	}

	// find out the latest sharder by ID
	sort.Slice(sharders, func(i, j int) bool {
		return sharders[i].sharder.id < sharders[j].sharder.id
	})

	// the outside sharder that going to come up
	var outside = sharders[len(sharders)-1]

	// add TotalStaked for the outside sharder //
	//
	var all *MinerNodes
	all, err = msc.getShardersList(balances, AllShardersKey)
	require.NoError(b, err)

	var sh = all.FindNodeById(outside.sharder.id)
	require.NotNil(b, sh)

	sh.TotalStaked = 1e10 // 1 token
	balances.InsertTrieNode(AllShardersKey, all)
	//
	/////////////////////////////////////////////

	// create previous MB with all miners, and all sharders excluding the
	// latest one
	balances.setLFMB(
		createPreviousMagicBlock(miners, sharders[:len(sharders)-1]),
	)

	// set fake block
	var (
		blk       = block.Provider().(*block.Block)
		generator = miners[0]
	)
	blk.Round = 251                                 // VC round
	blk.MinerID = generator.miner.id                // block generator
	blk.PrevBlock = block.Provider().(*block.Block) // stub
	balances.block = blk

	var pn *PhaseNode
	pn, err = msc.getPhaseNode(balances)
	require.NoError(b, err)

	// measure speed of the moveToContribute
	b.Run("moveToContribute", func(b *testing.B) {
		balances.skipMerge = false
		balances.mpts.merge(b)

		for i := 0; i < b.N; i++ {
			require.True(b, msc.moveToContribute(balances, pn, gn))
		}

		balances.skipMerge = true
	})

	// prepare for next benchmarks
	err = msc.createDKGMinersForContribute(balances, gn)
	require.NoError(b, err)

	// switch phase to contribute
	pn, err = msc.getPhaseNode(balances)
	require.NoError(b, err)

	pn.Phase = Contribute
	mustSave(b, PhaseKey, pn, balances)

	// fill up the sharders keep list with all other sharders first
	tp += 1
	for _, sh := range sharders[:len(sharders)-1] {
		var rq = NewMinerNode()
		rq.ID = sh.sharder.id
		rq.PublicKey = sh.sharder.pk
		rq.N2NHost = "http://" + sh.sharder.id + "/"

		var tx = newTransaction(sh.sharder.id, msc.ID, 0, tp)
		balances.setTransaction(b, tx)

		_, err = msc.sharderKeep(tx, mustEncode(b, rq), gn, balances)
		require.NoError(b, err)
	}

	// measure speed of the sharderKeep
	b.Run("sharderKeep", func(b *testing.B) {
		balances.skipMerge = false
		balances.mpts.merge(b)

		tp += 1

		var (
			tx    *transaction.Transaction
			input []byte

			request = NewMinerNode()
		)

		request.ID = outside.sharder.id
		request.PublicKey = outside.sharder.pk
		request.N2NHost = "http://" + outside.sharder.id + "/"

		input = mustEncode(b, request)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// don't measure the MPT merge time in the setTransaction
			b.StopTimer()
			{
				// remove the sharder from keep list if found
				var skeep *MinerNodes
				skeep, err = msc.getShardersList(balances, ShardersKeepKey)
				require.NoError(b, err)
				var i int
				for _, sh := range skeep.Nodes {
					if sh.ID == outside.sharder.id {
						continue
					}
					skeep.Nodes[i] = sh
					i++
				}
				skeep.Nodes = skeep.Nodes[:i]
				_, err = balances.InsertTrieNode(ShardersKeepKey, skeep)
				require.NoError(b, err)

				tx = newTransaction(outside.sharder.id, msc.ID, 0, tp)
				balances.setTransaction(b, tx)
			}
			b.StartTimer()

			_, err = msc.sharderKeep(tx, input, gn, balances)
			require.NoError(b, err)
		}

		balances.skipMerge = true
	})

	// prepare for the moving to sharder or publish move function

	// create fake mpks
	var mpks = block.NewMpks()
	for _, mn := range miners {
		mpks.Mpks[mn.miner.id] = nil // we don't need a real Mpk for the bench
	}
	_, err = balances.InsertTrieNode(MinersMPKKey, mpks)
	require.NoError(b, err)

	// measure moveToShareOrPublish execution time
	b.Run("moveToShareOrPublish", func(b *testing.B) {
		balances.skipMerge = false
		balances.mpts.merge(b)

		for i := 0; i < b.N; i++ {
			require.True(b, msc.moveToShareOrPublish(balances, pn, gn))
		}

		balances.skipMerge = true
	})

	b.Run("createMagicBlockForWait", func(b *testing.B) {
		balances.skipMerge = false
		balances.mpts.merge(b)
		for i := 0; i < b.N; i++ {
			msc.createMagicBlockForWait(balances, gn)
		}
		balances.skipMerge = true
	})

	for _, mn := range miners[:len(miners)-1] {
		for _, st := range mn.stakers {
			_, err = st.callAddToDelegatePool(b, msc, now, stakeVal,
				mn.miner.id, balances)
			require.NoError(b, err, "staking miner")
			now += 10
		}
	}

	for _, sh := range sharders {
		for _, st := range sh.stakers {
			_, err = st.callAddToDelegatePool(b, msc, now, stakeVal,
				sh.sharder.id, balances)
			require.NoError(b, err, "staking sharder")
			now += 10
		}
	}
	blk.Round = 251                                 // VC round
	blk.MinerID = generator.miner.id                // block generator
	blk.PrevBlock = block.Provider().(*block.Block) // stub
	balances.block = blk

	var mb = block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)
	mb.Sharders = node.NewPool(node.NodeTypeSharder)

	for _, mn := range miners {
		var n = node.Provider()
		n.SetID(mn.miner.id)
		n.Type = node.NodeTypeMiner
		mb.Miners.AddNode(n)
	}
	for _, sh := range sharders[:len(sharders)-1] {
		var n = node.Provider()
		n.SetID(sh.sharder.id)
		n.Type = node.NodeTypeSharder
		mb.Sharders.AddNode(n)
	}
	_, err = balances.InsertTrieNode(MagicBlockKey, mb)
	require.NoError(b, err, "setting magic block")

	b.Run("VCPoolsWork->payFees", func(b *testing.B) {

		balances.skipMerge = false
		balances.mpts.merge(b)
		for i := 0; i < b.N; i++ {
			err = msc.viewChangePoolsWork(gn, mb, 501, balances)
			require.NoError(b, err, "vcPoolsWork")
		}

		balances.skipMerge = true
	})

}

func Benchmark_swapMinerByStake(b *testing.B) {
	const stakeVal, stakeHolders = 10e10, 5

	var (
		balances = newTestBalances(b, true)
		msc      = newTestMinerSC()
		now      int64
		err      error
		tp       int64 // time point

		gn *GlobalNode // configurations

		miners   = make([]*miner, 0, 1001)
		sharders = make([]*sharder, 0, 1000)
	)
	defer balances.mpts.Close()
	balances.skipMerge = true

	gn = setConfig(b, balances)
	gn.MinS = 1
	gn.MaxS = 1000
	gn.MinN = 3
	gn.MaxN = 1000
	mustSave(b, GlobalNodeKey, gn, balances)

	// add miners
	for i := 0; i < cap(miners); i++ {
		miners = append(miners, newMiner(b, msc, now, stakeHolders,
			stakeVal, balances))
		now += 10
	}

	// add sharders
	for i := 0; i < cap(sharders); i++ {
		sharders = append(sharders, newSharder(b, msc, now, stakeHolders,
			stakeVal, balances))
		now += 10
	}
	// find out the latest miner by ID
	sort.Slice(miners, func(i, j int) bool {
		return miners[i].miner.id < miners[j].miner.id
	})

	// the outside miner that going to come up
	var outside = miners[len(miners)-1]

	// add TotalStaked for the outside miner //
	//
	var all *MinerNodes
	all, err = msc.getMinersList(balances)
	require.NoError(b, err)

	var mn = all.FindNodeById(outside.miner.id)
	require.NotNil(b, mn)

	mn.TotalStaked = 1e10 // 1 token
	balances.InsertTrieNode(AllMinersKey, all)
	tp++

	balances.setLFMB(
		createPreviousMagicBlock(miners[:len(miners)-1], sharders),
	)

	// set fake block
	var (
		blk       = block.Provider().(*block.Block)
		generator = miners[0]
	)
	blk.Round = 251                                 // VC round
	blk.MinerID = generator.miner.id                // block generator
	blk.PrevBlock = block.Provider().(*block.Block) // stub
	balances.block = blk

	var pn *PhaseNode
	pn, err = msc.getPhaseNode(balances)
	require.NoError(b, err)

	// measure speed of the moveToContribute
	b.Run("moveToContribute", func(b *testing.B) {
		balances.skipMerge = false
		balances.mpts.merge(b)

		for i := 0; i < b.N; i++ {
			require.True(b, msc.moveToContribute(balances, pn, gn))
		}

		balances.skipMerge = true
	})

	// prepare for next benchmarks
	err = msc.createDKGMinersForContribute(balances, gn)
	require.NoError(b, err)

	// switch phase to contribute
	pn, err = msc.getPhaseNode(balances)
	require.NoError(b, err)

	pn.Phase = Contribute
	mustSave(b, PhaseKey, pn, balances)

	tp += 1
	for _, sh := range sharders {
		var rq = NewMinerNode()
		rq.ID = sh.sharder.id
		rq.PublicKey = sh.sharder.pk
		rq.N2NHost = "http://" + sh.sharder.id + "/"

		var tx = newTransaction(sh.sharder.id, msc.ID, 0, tp)
		balances.setTransaction(b, tx)

		_, err = msc.sharderKeep(tx, mustEncode(b, rq), gn, balances)
		require.NoError(b, err)
	}
	// create fake mpks
	var mpks = block.NewMpks()
	for _, mn := range miners {
		mpks.Mpks[mn.miner.id] = nil // we don't need a real Mpk for the bench
	}
	_, err = balances.InsertTrieNode(MinersMPKKey, mpks)
	require.NoError(b, err)

	// measure moveToShareOrPublish execution time
	b.Run("moveToShareOrPublish", func(b *testing.B) {
		balances.skipMerge = false
		balances.mpts.merge(b)

		for i := 0; i < b.N; i++ {
			require.True(b, msc.moveToShareOrPublish(balances, pn, gn))
		}

		balances.skipMerge = true
	})

	b.Run("createMagicBlockForWait", func(b *testing.B) {
		balances.skipMerge = false
		balances.mpts.merge(b)
		for i := 0; i < b.N; i++ {
			msc.createMagicBlockForWait(balances, gn)
		}
		balances.skipMerge = true
	})
	for _, mn := range miners[:len(miners)-1] {
		for _, st := range mn.stakers {
			_, err = st.callAddToDelegatePool(b, msc, now, stakeVal,
				mn.miner.id, balances)
			require.NoError(b, err, "staking miner")
			now += 10
		}
	}

	for _, sh := range sharders {
		for _, st := range sh.stakers {
			_, err = st.callAddToDelegatePool(b, msc, now, stakeVal,
				sh.sharder.id, balances)
			require.NoError(b, err, "staking sharder")
			now += 10
		}
	}
	blk.Round = 251                                 // VC round
	blk.MinerID = generator.miner.id                // block generator
	blk.PrevBlock = block.Provider().(*block.Block) // stub
	balances.block = blk

	var mb = block.NewMagicBlock()
	mb.Miners = node.NewPool(node.NodeTypeMiner)
	mb.Sharders = node.NewPool(node.NodeTypeSharder)

	for _, mn := range miners[:len(miners)-1] {
		var n = node.Provider()
		n.SetID(mn.miner.id)
		n.Type = node.NodeTypeMiner
		mb.Miners.AddNode(n)
	}
	for _, sh := range sharders {
		var n = node.Provider()
		n.SetID(sh.sharder.id)
		n.Type = node.NodeTypeSharder
		mb.Sharders.AddNode(n)
	}
	_, err = balances.InsertTrieNode(MagicBlockKey, mb)
	require.NoError(b, err, "setting magic block")

	b.Run("VCPoolsWork->payFees", func(b *testing.B) {
		balances.skipMerge = false
		for i := 0; i < b.N; i++ {
			err = msc.viewChangePoolsWork(gn, mb, 501, balances)
			require.NoError(b, err, "vcPoolsWork")
		}
		balances.skipMerge = true
	})
}
