package minersc

import (
	"fmt"
	"math/rand"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/state"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestClient struct {
	client   *Client
	delegate *Client
	stakers  []*Client
}

func createLFMB(miners []*TestClient, sharders []*TestClient) (
	b *block.Block) {

	b = new(block.Block)

	b.MagicBlock = block.NewMagicBlock()
	b.MagicBlock.Miners = node.NewPool(node.NodeTypeMiner)
	b.MagicBlock.Sharders = node.NewPool(node.NodeTypeSharder)

	for _, miner := range miners {
		b.MagicBlock.Miners.NodesMap[miner.client.id] = new(node.Node)
	}
	for _, sharder := range sharders {
		b.MagicBlock.Sharders.NodesMap[sharder.client.id] = new(node.Node)
	}
	return
}

func (msc *MinerSmartContract) setDKGMiners(t *testing.T,
	miners []*TestClient, balances *testBalances) {

	t.Helper()

	var global, err = msc.getGlobalNode(balances)
	require.NoError(t, err)

	var nodes *DKGMinerNodes
	nodes, err = msc.getMinersDKGList(balances)
	require.NoError(t, err)

	nodes.setConfigs(global)
	for _, miner := range miners {
		nodes.SimpleNodes[miner.client.id] = &SimpleNode{ID: miner.client.id}
		nodes.Waited[miner.client.id] = true
	}

	_, err = balances.InsertTrieNode(DKGMinersKey, nodes)
	require.NoError(t, err)
}

func Test_payFees(t *testing.T) {
	const sharderStakeValue, minerStakeValue, generatorStakeValue = 5, 3, 2
	const sharderStakersAmount, minerStakersAmount, generatorStakersAmount = 13, 11, 7
    const minersAmount, shardersAmount = 17, 19
	const generatorIdx = 0

    const timeDelta = 10

	var (
		balances = newTestBalances()
		msc      = newTestMinerSC()
		now      int64
		err      error

		miners   []*TestClient
		sharders []*TestClient
	)

	setConfig(t, balances)

	{
		var generator *TestClient

		//t.Run("add miners", func(t *testing.T) {
		generator = newClientWithStakers(true, t, msc, now,
			generatorStakersAmount, generatorStakeValue, balances)

		for idx := 0; idx < minersAmount; idx++ {
			if idx == generatorIdx {
				miners = append(miners, generator)
			} else {
				miners = append(miners, newClientWithStakers(true, t, msc, now,
					minerStakersAmount, minerStakeValue, balances))
			}
			now += timeDelta
		}
		//})
	}

	//t.Run("add sharders", func(t *testing.T) {
	for idx := 0; idx < shardersAmount; idx++ {
		sharders = append(sharders, newClientWithStakers(false, t, msc, now,
			sharderStakersAmount, sharderStakeValue, balances))
		now += timeDelta
	}
	//})

	//todo: advanced test case: create pool of N stakers and assign them to different nodes randomly,
	//      this way 1 staker might be stake holder of several different miners/sharders at the same time
	//      and more complicated computation is required in order to test such case

	msc.setDKGMiners(t, miners, balances)
	balances.setLFMB(createLFMB(miners, sharders))

	//t.Run("stake miners", func(t *testing.T) {
    for idx, miner := range miners {
        var stakeValue int64
        if idx == generatorIdx {
            stakeValue = generatorStakeValue
        } else {
            stakeValue = minerStakeValue
        }

        for _, staker := range miner.stakers {
            _, err = staker.callAddToDelegatePool(t, msc, now,
                stakeValue, miner.client.id, balances)

            require.NoError(t, err, "staking miner")
            assert.Zero(t, balances.balances[staker.id], "stakers' balances should be updated later")

            now += timeDelta
        }

        assert.Zero(t, balances.balances[miner.client.id], "miner's balance shouldn't be changed yet")
        assert.Zero(t, balances.balances[miner.delegate.id], "miner's delegate balance shouldn't be changed yet")
    }
    //})

	//t.Run("stake sharders", func(t *testing.T) {
    for _, sharder := range sharders {
        for _, staker := range sharder.stakers {
            _, err = staker.callAddToDelegatePool(t, msc, now,
                sharderStakeValue, sharder.client.id, balances)

            require.NoError(t, err, "staking sharder")
            assert.Zero(t, balances.balances[staker.id], "stakers' balance should be updated later")

            now += timeDelta
        }

        assert.Zero(t, balances.balances[sharder.client.id], "sharder's balance shouldn't be changed yet")
        assert.Zero(t, balances.balances[sharder.delegate.id], "sharder's balance shouldn't be changed yet")
    }
    //})

	msc.setDKGMiners(t, miners, balances)

	t.Run("pay fees -> view change", func(t *testing.T) {
		assertBalancesAreZeros(t, balances)
		setRounds(t, msc, 250, 251, balances)

		fmt.Println("=== [0] ===")
		msc.debug_pools(balances)

		setMagicBlock(t, unwrapClients(miners), unwrapClients(sharders),
			balances)

		fmt.Println("=== [1] ===")
		msc.debug_pools(balances)

		var generator, blck = prepareGeneratorAndBlock(miners, 0, 251)

		fmt.Println("=== [2] ===")
		msc.debug_pools(balances)

		// payFees transaction
		now += timeDelta
		var tx = newTransaction(generator.client.id, ADDRESS, 0, now)
		balances.txn = tx
		balances.block = blck
		balances.blockSharders = selectRandom(sharders, 3)

		fmt.Println("=== [3] ===")
		msc.debug_pools(balances)

		var global, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")

		_, err = msc.payFees(tx, nil, global, balances)
		require.NoError(t, err, "pay_fees error")

		fmt.Println("=== [4] ===")
		msc.debug_pools(balances)

		// pools become active, nothing should be paid

		for _, miner := range miners {
			assert.Zero(t, balances.balances[miner.client.id],
				"miner balance")
			assert.Zero(t, balances.balances[miner.delegate.id],
				"miner delegate balance?")
			for _, staker := range miner.stakers {
				assert.Zero(t, balances.balances[staker.id], "stake balance?")
			}
		}
		for _, sharder := range sharders {
			assert.Zero(t, balances.balances[sharder.client.id],
				"sharder balance")
			assert.Zero(t, balances.balances[sharder.delegate.id],
				"sharder delegate balance?")
			for _, staker := range sharder.stakers {
				assert.Zero(t, balances.balances[staker.id], "stake balance?")
			}
		}

		global, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "can't get global node")
		assert.EqualValues(t, 251, global.LastRound)
		assert.EqualValues(t, 0, global.Minted)
	})

	msc.setDKGMiners(t, miners, balances)

	t.Run("pay fees -> no fees", func(t *testing.T) {
		assertBalancesAreZeros(t, balances)
		setRounds(t, msc, 251, 501, balances)

		var generator, blck = prepareGeneratorAndBlock(miners, 0, 252)

		// payFees transaction
		now += timeDelta
		var tx = newTransaction(generator.client.id, ADDRESS, 0, now)
		balances.txn = tx
		balances.block = blck
		balances.blockSharders = selectRandom(sharders, 3)

		var global, err = msc.getGlobalNode(balances)
		require.NoError(t, err, "getting global node")

		_, err = msc.payFees(tx, nil, global, balances)
		require.NoError(t, err, "pay_fees error")

		// pools active, no fees, rewards should be payed for
		// generator's and block sharders' stake holders

		var (
			expected = make(map[string]state.Balance)
			actual   = make(map[string]state.Balance)
		)

		for idx, miner := range miners {
			assert.Zero(t, balances.balances[miner.client.id])
			assert.Zero(t, balances.balances[miner.delegate.id])

			var stakeValue state.Balance = 0;
			if idx == generatorIdx {
				stakeValue = generatorStakeValue;
			} else {
				stakeValue = minerStakeValue;
			}

			for _, staker := range miner.stakers {
				expected[staker.id] = stakeValue;
				actual[staker.id] = balances.balances[staker.id]
			}
		}

		assert.Equal(t, len(expected), len(actual), "sizes of balance maps")
		assert.Equal(t, expected, actual, "balances")

		for _, sharder := range sharders {
			assert.Zero(t, balances.balances[sharder.client.id])
			assert.Zero(t, balances.balances[sharder.delegate.id])

			for _, staker := range sharder.stakers {
				expected[staker.id] = 0 //only block sharders get paid
				actual[staker.id] = balances.balances[staker.id]
			}
		}

		for _, sharder := range filterClientsById(sharders, balances.blockSharders) {
			for _, staker := range sharder.stakers {
				expected[staker.id] += sharderStakeValue
			}
		}

		assert.Equal(t, len(expected), len(actual), "sizes of balance maps")
		assert.Equal(t, expected, actual, "balances")
	})

	// don't set DKG miners list, because no VC is expected

	// reset all balances
	balances.balances = make(map[string]state.Balance)

	//t.Run("pay fees -> with fees", func(t *testing.T) {
	//	setRounds(t, msc, 252, 501, balances)
	//
	//	var generator, blck = prepareGeneratorAndBlock(miners, 0, 253)
	//
	//	// payFees transaction
	//	now += timeDelta
	//	var tx = newTransaction(generator.miner.id, ADDRESS, 0, now)
	//	balances.txn = tx
	//	balances.block = blck
	//	balances.blockSharders = selectRandom(sharders, 3)
	//
	//	// add fees
	//	tx.Fee = 100e10
	//	blck.Txns = append(blck.Txns, tx)
	//
	//	var global, err = msc.getGlobalNode(balances)
	//	require.NoError(t, err, "getting global node")
	//
	//	_, err = msc.payFees(tx, nil, global, balances)
	//	require.NoError(t, err, "pay_fees error")
	//
	//	// pools are active, rewards as above and +fees
	//
	//	var (
	//		expected = make(map[string]state.Balance)
	//		actual      = make(map[string]state.Balance)
	//	)
	//
	//	for _, miner := range miners {
	//		assert.Zero(t, balances.balances[miner.client.id])
	//		assert.Zero(t, balances.balances[miner.delegate.id])
	//		for _, staker:= range miner.stakers {
	//			if miner == generator {
	//				expected[staker.id] += 77e7 + 11e10 // + generator fees
	//			} else {
	//				expected[staker.id] += 0
	//			}
	//			actual[staker.id] = balances.balances[staker.id]
	//		}
	//	}
	//
	//	for _, sharder := range sharders {
	//		assert.Zero(t, balances.balances[sh.sharder.id])
	//		assert.Zero(t, balances.balances[sh.delegate.id])
	//		for _, staker := range sharder.stakers {
	//			expected[staker.id] += 0
	//			actual[staker.id] = balances.balances[staker.id]
	//		}
	//	}
	//
	//	for _, sharder := range filterClientsById(sharders, balances.blockSharders) {
	//		for _, staker := range sharder.stakers {
	//			expected[staker.id] += 21e7 + 3e10 // + block sharders fees
	//		}
	//	}
	//
	//	assert.Equal(t, len(expected), len(actual), "sizes of balance maps")
	//	assert.Equal(t, expected, actual, "balances")
	//})

	// don't set DKG miners list, because no VC is expected

	// reset all balances
	balances.balances = make(map[string]state.Balance)

	//t.Run("pay fees -> view change interests", func(t *testing.T) {
	//	setRounds(t, msc, 500, 501, balances)
	//
	//	var generator, blck = prepareGeneratorAndBlock(miners, 0, 501)
	//
	//	// payFees transaction
	//	now += timeDelta
	//	var tx = newTransaction(generator.miner.id, ADDRESS, 0, now)
	//	balances.txn = tx
	//	balances.block = blck
	//	balances.blockSharders = selectRandom(sharders, 3)
	//
	//	// add fees
	//	var gn, err = msc.getGlobalNode(balances)
	//	require.NoError(t, err, "getting global node")
	//
	//	_, err = msc.payFees(tx, nil, gn, balances)
	//	require.NoError(t, err, "pay_fees error")
	//
	//	// pools are active, rewards as above and +fees
	//
	//	var (
	//		expected = make(map[string]state.Balance)
	//		actual      = make(map[string]state.Balance)
	//	)
	//
	//	for _, miner := range miners {
	//		assert.Zero(t, balances.balances[miner.miner.id])
	//		assert.Zero(t, balances.balances[miner.delegate.id])
	//		for _, staker := range miner.stakers {
	//			if miner == generator {
	//				expected[staker.id] += 77e7 + 1e10
	//			} else {
	//				expected[staker.id] += 1e10
	//			}
	//			actual[staker.id] = balances.balances[staker.id]
	//		}
	//	}
	//
	//	for _, sharder := range sharders {
	//		assert.Zero(t, balances.balances[sharder.sharder.id])
	//		assert.Zero(t, balances.balances[sharder.delegate.id])
	//		for _, staker := range sharder.stakers {
	//			expected[staker.id] += 1e10
	//			actual[staker.id] = balances.balances[staker.id]
	//		}
	//	}
	//
	//	for _, sharder := range filterClientsById(sharders, balances.blockSharders) {
	//		for _, staker := range sharder.stakers {
	//			expected[staker.id] += 21e7
	//		}
	//	}
	//
	//	assert.Equal(t, len(expected), len(actual), "sizes of balance maps")
	//	assert.Equal(t, expected, actual, "balances")
	//})

	t.Run("epoch", func(t *testing.T) {
		var global, err = msc.getGlobalNode(balances)
		require.NoError(t, err)

		var interestRate, rewardRate = global.InterestRate, global.RewardRate
		global.epochDecline()

		assert.True(t, global.InterestRate < interestRate)
		assert.True(t, global.RewardRate < rewardRate)
	})

}

func prepareGeneratorAndBlock(miners []*TestClient, idx int, round int64) (
	generator *TestClient, blck *block.Block) {

	generator = miners[idx]

	blck = block.Provider().(*block.Block)
	blck.Round = round                                // VC round
	blck.MinerID = generator.client.id                // block generator
	blck.PrevBlock = block.Provider().(*block.Block)  // stub

	return generator, blck
}

func unwrapClients(clients []*TestClient) (list []*Client) {
	list = make([]*Client, 0, len(clients))
	for _, miner := range clients {
		list = append(list, miner.client)
	}
	return
}

func selectRandom(clients []*TestClient, n int) (selection []string) {
	if n > len(clients) {
		panic("too many elements requested")
	}

	selection = make([]string, 0, n)

	var permutations = rand.Perm(len(clients))
	for i := 0; i < n; i++ {
		selection = append(selection, clients[permutations[i]].client.id)
	}
	return
}

func filterClientsById(clients []*TestClient, ids []string) (
	selection []*TestClient) {

	selection = make([]*TestClient, 0, len(ids))

	for _, client := range clients {
		for _, id := range ids {
			if client.client.id == id {
				selection = append(selection, client)
			}
		}
	}
	return
}

func assertBalancesAreZeros(t *testing.T, balances *testBalances) {
	for id, value := range balances.balances {
		if id == ADDRESS {
			continue
		}
		assert.Zerof(t, value, "unexpected balance: %s", id)
	}
}

func (msc *MinerSmartContract) debug_pools(balances *testBalances) {
	var miners, sharders *ConsensusNodes
	var err error

	if miners, err = msc.getMinersList(balances); err == nil {
		for idx, miner := range miners.Nodes {
			fmt.Printf("=-- miner #%d: %d active pools , %d pending pools\n",
				idx, len(miner.Active), len(miner.Pending))
		}
	} else {
		fmt.Println(">-- couldn't retrieve miners:")
		fmt.Printf(">-- %v\n", err)
	}

	if sharders, err = msc.getShardersList(balances, AllShardersKey); err == nil {
		for idx, sharder := range sharders.Nodes {
			fmt.Printf("=-- sharder #%d: %d active pools , %d pending pools\n",
				idx, len(sharder.Active), len(sharder.Pending))
		}
	} else {
		fmt.Println(">-- couldn't retrieve sharders:")
		fmt.Printf(">-- %v\n", err)
	}
}