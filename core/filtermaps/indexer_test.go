// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package filtermaps

import (
	crand "crypto/rand"
	"crypto/sha256"
	"math/big"
	"math/rand"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

var testParams = Params{
	logMapHeight:       2,
	logMapWidth:        24,
	logMapsPerEpoch:    4,
	logValuesPerMap:    4,
	baseRowGroupLength: 4,
	baseRowLengthRatio: 2,
	logLayerDiff:       2,
}

func TestIndexerRandomRange(t *testing.T) {
	ts := newTestSetup(t)
	defer ts.close()

	forks := make([][]common.Hash, 10)
	ts.chain.addBlocks(1000, 5, 2, 4, false) // 51 log values per block
	for i := range forks {
		if i != 0 {
			forkBlock := rand.Intn(1000)
			ts.chain.setHead(forkBlock)
			ts.chain.addBlocks(1000-forkBlock, 5, 2, 4, false) // 51 log values per block
		}
		forks[i] = ts.chain.getCanonicalChain()
	}
	lvPerBlock := uint64(51)
	ts.setHistory(0, false)
	var (
		history       int
		noHistory     bool
		fork, head    = len(forks) - 1, 1000
		checkSnapshot bool
	)
	ts.fm.WaitIdle()
	for i := 0; i < 200; i++ {
		switch rand.Intn(3) {
		case 0:
			// change history settings
			switch rand.Intn(10) {
			case 0:
				history, noHistory = 0, false
			case 1:
				history, noHistory = 0, true
			default:
				history, noHistory = rand.Intn(1000)+1, false
			}
			ts.testDisableSnapshots = rand.Intn(2) == 0
			ts.setHistory(uint64(history), noHistory)
		case 1:
			// change head to random position of random fork
			fork, head = rand.Intn(len(forks)), rand.Intn(1001)
			ts.chain.setCanonicalChain(forks[fork][:head+1])
		case 2:
			if head < 1000 {
				checkSnapshot = !noHistory && head != 0 // no snapshot generated for block 0
				// add blocks after the current head
				head += rand.Intn(1000-head) + 1
				ts.fm.testSnapshotUsed = false
				ts.chain.setCanonicalChain(forks[fork][:head+1])
			}
		}
		ts.fm.WaitIdle()
		if checkSnapshot {
			if ts.fm.testSnapshotUsed == ts.fm.testDisableSnapshots {
				ts.t.Fatalf("Invalid snapshot used state after head extension (used: %v, disabled: %v)", ts.fm.testSnapshotUsed, ts.fm.testDisableSnapshots)
			}
			checkSnapshot = false
		}
		if noHistory {
			if ts.fm.indexedRange.initialized {
				t.Fatalf("filterMapsRange initialized while indexing is disabled")
			}
			continue
		}
		if !ts.fm.indexedRange.initialized {
			t.Fatalf("filterMapsRange not initialized while indexing is enabled")
		}
		var tailBlock uint64
		if history > 0 && history <= head {
			tailBlock = uint64(head + 1 - history)
		}
		var tailEpoch uint32
		if tailBlock > 0 {
			tailLvPtr := (tailBlock - 1) * lvPerBlock // no logs in genesis block, only delimiter
			tailEpoch = uint32(tailLvPtr >> (testParams.logValuesPerMap + testParams.logMapsPerEpoch))
		}
		var expTailBlock uint64
		if tailEpoch > 0 {
			tailLvPtr := uint64(tailEpoch) << (testParams.logValuesPerMap + testParams.logMapsPerEpoch) // first available lv ptr
			// (expTailBlock-1)*lvPerBlock >= tailLvPtr
			expTailBlock = (tailLvPtr + lvPerBlock*2 - 1) / lvPerBlock
		}
		if ts.fm.indexedRange.afterLastIndexedBlock != uint64(head+1) {
			ts.t.Fatalf("Invalid index head (expected #%d, got #%d)", head, ts.fm.indexedRange.afterLastIndexedBlock-1)
		}
		if ts.fm.indexedRange.headBlockDelimiter != uint64(head)*lvPerBlock {
			ts.t.Fatalf("Invalid index head delimiter pointer (expected %d, got %d)", uint64(head)*lvPerBlock, ts.fm.indexedRange.headBlockDelimiter)
		}
		if ts.fm.indexedRange.firstIndexedBlock != expTailBlock {
			ts.t.Fatalf("Invalid index tail block (expected #%d, got #%d)", expTailBlock, ts.fm.indexedRange.firstIndexedBlock)
		}
	}
}

func TestIndexerCompareDb(t *testing.T) {
	ts := newTestSetup(t)
	defer ts.close()

	ts.chain.addBlocks(500, 10, 3, 4, true)
	ts.setHistory(0, false)
	ts.fm.WaitIdle()
	// revert points are stored after block 500
	ts.chain.addBlocks(500, 10, 3, 4, true)
	ts.fm.WaitIdle()
	chain1 := ts.chain.getCanonicalChain()
	ts.storeDbHash("chain 1 [0, 1000]")

	ts.chain.setHead(600)
	ts.fm.WaitIdle()
	ts.storeDbHash("chain 1/2 [0, 600]")

	ts.chain.addBlocks(600, 10, 3, 4, true)
	ts.fm.WaitIdle()
	chain2 := ts.chain.getCanonicalChain()
	ts.storeDbHash("chain 2 [0, 1200]")

	ts.chain.setHead(600)
	ts.fm.WaitIdle()
	ts.checkDbHash("chain 1/2 [0, 600]")

	ts.setHistory(800, false)
	ts.chain.setCanonicalChain(chain1)
	ts.fm.WaitIdle()
	ts.storeDbHash("chain 1 [201, 1000]")

	ts.setHistory(0, false)
	ts.fm.WaitIdle()
	ts.checkDbHash("chain 1 [0, 1000]")

	ts.setHistory(800, false)
	ts.chain.setCanonicalChain(chain2)
	ts.fm.WaitIdle()
	ts.storeDbHash("chain 2 [401, 1200]")

	ts.setHistory(0, true)
	ts.fm.WaitIdle()
	ts.storeDbHash("no index")

	ts.chain.setCanonicalChain(chain2[:501])
	ts.setHistory(0, false)
	ts.fm.WaitIdle()
	ts.chain.setCanonicalChain(chain2)
	ts.fm.WaitIdle()
	ts.checkDbHash("chain 2 [0, 1200]")

	ts.chain.setCanonicalChain(chain1)
	ts.fm.WaitIdle()
	ts.setHistory(800, false)
	ts.fm.WaitIdle()
	ts.checkDbHash("chain 1 [201, 1000]")

	ts.chain.setCanonicalChain(chain2)
	ts.fm.WaitIdle()
	ts.checkDbHash("chain 2 [401, 1200]")

	ts.setHistory(0, true)
	ts.fm.WaitIdle()
	ts.checkDbHash("no index")
}

type testSetup struct {
	t                    *testing.T
	fm                   *FilterMaps
	db                   ethdb.Database
	chain                *testChain
	params               Params
	dbHashes             map[string]common.Hash
	testDisableSnapshots bool
}

func newTestSetup(t *testing.T) *testSetup {
	params := testParams
	params.deriveFields()
	ts := &testSetup{
		t:        t,
		db:       rawdb.NewMemoryDatabase(),
		params:   params,
		dbHashes: make(map[string]common.Hash),
	}
	ts.chain = ts.newTestChain()
	return ts
}

func (ts *testSetup) setHistory(history uint64, noHistory bool) {
	if ts.fm != nil {
		ts.fm.Stop()
	}
	head := ts.chain.CurrentBlock()
	view := NewChainView(ts.chain, head.Number.Uint64(), head.Hash())
	config := Config{
		History:  history,
		Disabled: noHistory,
	}
	ts.fm = NewFilterMaps(ts.db, view, ts.params, config)
	ts.fm.testDisableSnapshots = ts.testDisableSnapshots
	ts.fm.Start()
}

func (ts *testSetup) storeDbHash(id string) {
	dbHash := ts.fmDbHash()
	for otherId, otherHash := range ts.dbHashes {
		if otherHash == dbHash {
			ts.t.Fatalf("Unexpected equal database hashes `%s` and `%s`", id, otherId)
		}
	}
	ts.dbHashes[id] = dbHash
}

func (ts *testSetup) checkDbHash(id string) {
	if ts.fmDbHash() != ts.dbHashes[id] {
		ts.t.Fatalf("Database `%s` hash mismatch", id)
	}
}

func (ts *testSetup) fmDbHash() common.Hash {
	hasher := sha256.New()
	it := ts.db.NewIterator(nil, nil)
	for it.Next() {
		hasher.Write(it.Key())
		hasher.Write(it.Value())
	}
	it.Release()
	var result common.Hash
	hasher.Sum(result[:0])
	return result
}

func (ts *testSetup) close() {
	if ts.fm != nil {
		ts.fm.Stop()
	}
	ts.db.Close()
	ts.chain.db.Close()
}

type testChain struct {
	ts        *testSetup
	db        ethdb.Database
	lock      sync.RWMutex
	canonical []common.Hash
	blocks    map[common.Hash]*types.Block
	receipts  map[common.Hash]types.Receipts
}

func (ts *testSetup) newTestChain() *testChain {
	return &testChain{
		ts:       ts,
		blocks:   make(map[common.Hash]*types.Block),
		receipts: make(map[common.Hash]types.Receipts),
	}
}

func (tc *testChain) CurrentBlock() *types.Header {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	if len(tc.canonical) == 0 {
		return nil
	}
	return tc.blocks[tc.canonical[len(tc.canonical)-1]].Header()
}

func (tc *testChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	if block := tc.blocks[hash]; block != nil {
		return block.Header()
	}
	return nil
}

func (tc *testChain) GetCanonicalHash(number uint64) common.Hash {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	if uint64(len(tc.canonical)) <= number {
		return common.Hash{}
	}
	return tc.canonical[number]
}

func (tc *testChain) GetReceiptsByHash(hash common.Hash) types.Receipts {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	return tc.receipts[hash]
}

func (tc *testChain) addBlocks(count, maxTxPerBlock, maxLogsPerReceipt, maxTopicsPerLog int, random bool) {
	tc.lock.Lock()
	blockGen := func(i int, gen *core.BlockGen) {
		var txCount int
		if random {
			txCount = rand.Intn(maxTxPerBlock + 1)
		} else {
			txCount = maxTxPerBlock
		}
		for k := txCount; k > 0; k-- {
			receipt := types.NewReceipt(nil, false, 0)
			var logCount int
			if random {
				logCount = rand.Intn(maxLogsPerReceipt + 1)
			} else {
				logCount = maxLogsPerReceipt
			}
			receipt.Logs = make([]*types.Log, logCount)
			for i := range receipt.Logs {
				log := &types.Log{}
				receipt.Logs[i] = log
				crand.Read(log.Address[:])
				var topicCount int
				if random {
					topicCount = rand.Intn(maxTopicsPerLog + 1)
				} else {
					topicCount = maxTopicsPerLog
				}
				log.Topics = make([]common.Hash, topicCount)
				for j := range log.Topics {
					crand.Read(log.Topics[j][:])
				}
			}
			gen.AddUncheckedReceipt(receipt)
			gen.AddUncheckedTx(types.NewTransaction(999, common.HexToAddress("0x999"), big.NewInt(999), 999, gen.BaseFee(), nil))
		}
	}

	var (
		blocks   []*types.Block
		receipts []types.Receipts
		engine   = ethash.NewFaker()
	)

	if len(tc.canonical) == 0 {
		gspec := &core.Genesis{
			Alloc:   types.GenesisAlloc{},
			BaseFee: big.NewInt(params.InitialBaseFee),
			Config:  params.TestChainConfig,
		}
		tc.db, blocks, receipts = core.GenerateChainWithGenesis(gspec, engine, count, blockGen)
		gblock := gspec.ToBlock()
		ghash := gblock.Hash()
		tc.canonical = []common.Hash{ghash}
		tc.blocks[ghash] = gblock
		tc.receipts[ghash] = types.Receipts{}
	} else {
		blocks, receipts = core.GenerateChain(params.TestChainConfig, tc.blocks[tc.canonical[len(tc.canonical)-1]], engine, tc.db, count, blockGen)
	}

	for i, block := range blocks {
		num, hash := int(block.NumberU64()), block.Hash()
		if len(tc.canonical) != num {
			panic("canonical chain length mismatch")
		}
		tc.canonical = append(tc.canonical, hash)
		tc.blocks[hash] = block
		if receipts[i] != nil {
			tc.receipts[hash] = receipts[i]
		} else {
			tc.receipts[hash] = types.Receipts{}
		}
	}
	tc.lock.Unlock()
	tc.setTargetHead()
}

func (tc *testChain) setHead(headNum int) {
	tc.lock.Lock()
	tc.canonical = tc.canonical[:headNum+1]
	tc.lock.Unlock()
	tc.setTargetHead()
}

func (tc *testChain) setTargetHead() {
	head := tc.CurrentBlock()
	if tc.ts.fm != nil {
		if !tc.ts.fm.disabled {
			//tc.ts.fm.targetViewCh <- NewChainView(tc, head.Number.Uint64(), head.Hash())
			tc.ts.fm.SetTargetView(NewChainView(tc, head.Number.Uint64(), head.Hash()))
		}
	}
}

func (tc *testChain) getCanonicalChain() []common.Hash {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	cc := make([]common.Hash, len(tc.canonical))
	copy(cc, tc.canonical)
	return cc
}

// restore an earlier state of the chain
func (tc *testChain) setCanonicalChain(cc []common.Hash) {
	tc.lock.Lock()
	tc.canonical = make([]common.Hash, len(cc))
	copy(tc.canonical, cc)
	tc.lock.Unlock()
	tc.setTargetHead()
}
