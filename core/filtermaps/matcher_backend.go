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
	"context"

	"github.com/ethereum/go-ethereum/core/types"
)

// FilterMapsMatcherBackend implements MatcherBackend.
type FilterMapsMatcherBackend struct {
	f *FilterMaps

	// these fields should be accessed under f.matchersLock mutex.
	valid                 bool
	firstValid, lastValid uint64
	syncCh                chan SyncRange
}

// NewMatcherBackend returns a FilterMapsMatcherBackend after registering it in
// the active matcher set.
// Note that Close should always be called when the matcher is no longer used.
func (f *FilterMaps) NewMatcherBackend() *FilterMapsMatcherBackend {
	f.indexLock.RLock()
	f.matchersLock.Lock()
	defer func() {
		f.matchersLock.Unlock()
		f.indexLock.RUnlock()
	}()

	fm := &FilterMapsMatcherBackend{
		f:          f,
		valid:      f.indexedRange.initialized && f.indexedRange.afterLastIndexedBlock > f.indexedRange.firstIndexedBlock,
		firstValid: f.indexedRange.firstIndexedBlock,
		lastValid:  f.indexedRange.afterLastIndexedBlock - 1,
	}
	f.matchers[fm] = struct{}{}
	return fm
}

// GetParams returns the filtermaps parameters.
// GetParams implements MatcherBackend.
func (fm *FilterMapsMatcherBackend) GetParams() *Params {
	return &fm.f.Params
}

// Close removes the matcher from the set of active matchers and ensures that
// any SyncLogIndex calls are cancelled.
// Close implements MatcherBackend.
func (fm *FilterMapsMatcherBackend) Close() {
	fm.f.matchersLock.Lock()
	defer fm.f.matchersLock.Unlock()

	delete(fm.f.matchers, fm)
}

// GetFilterMapRow returns the given row of the given map. If the row is empty
// then a non-nil zero length row is returned. If baseLayerOnly is true then
// only the first baseRowLength entries of the row are guaranteed to be
// returned.
// Note that the returned slices should not be modified, they should be copied
// on write.
// GetFilterMapRow implements MatcherBackend.
func (fm *FilterMapsMatcherBackend) GetFilterMapRow(ctx context.Context, mapIndex, rowIndex uint32, baseLayerOnly bool) (FilterRow, error) {
	return fm.f.getFilterMapRow(mapIndex, rowIndex, baseLayerOnly)
}

// GetBlockLvPointer returns the starting log value index where the log values
// generated by the given block are located. If blockNumber is beyond the current
// head then the first unoccupied log value index is returned.
// GetBlockLvPointer implements MatcherBackend.
func (fm *FilterMapsMatcherBackend) GetBlockLvPointer(ctx context.Context, blockNumber uint64) (uint64, error) {
	fm.f.indexLock.RLock()
	defer fm.f.indexLock.RUnlock()

	return fm.f.getBlockLvPointer(blockNumber)
}

// GetLogByLvIndex returns the log at the given log value index.
// Note that this function assumes that the log index structure is consistent
// with the canonical chain at the point where the given log value index points.
// If this is not the case then an invalid result may be returned or certain
// logs might not be returned at all.
// No error is returned though because of an inconsistency between the chain and
// the log index. It is the caller's responsibility to verify this consistency
// using SyncLogIndex and re-process certain blocks if necessary.
// GetLogByLvIndex implements MatcherBackend.
func (fm *FilterMapsMatcherBackend) GetLogByLvIndex(ctx context.Context, lvIndex uint64) (*types.Log, error) {
	fm.f.indexLock.RLock()
	defer fm.f.indexLock.RUnlock()

	return fm.f.getLogByLvIndex(lvIndex)
}

// synced signals to the matcher that has triggered a synchronisation that it
// has been finished and the log index is consistent with the chain head passed
// as a parameter.
// Note that if the log index head was far behind the chain head then it might not
// be synced up to the given head in a single step. Still, the latest chain head
// should be passed as a parameter and the existing log index should be consistent
// with that chain.
func (fm *FilterMapsMatcherBackend) synced() {
	fm.f.indexLock.RLock()
	fm.f.matchersLock.Lock()
	defer func() {
		fm.f.matchersLock.Unlock()
		fm.f.indexLock.RUnlock()
	}()

	var (
		indexed                     bool
		lastIndexed, subLastIndexed uint64
	)
	if !fm.f.indexedRange.headBlockIndexed {
		subLastIndexed = 1
	}
	if fm.f.indexedRange.afterLastIndexedBlock-subLastIndexed > fm.f.indexedRange.firstIndexedBlock {
		indexed, lastIndexed = true, fm.f.indexedRange.afterLastIndexedBlock-subLastIndexed-1
	}
	fm.syncCh <- SyncRange{
		HeadNumber:   fm.f.indexedView.headNumber,
		Valid:        fm.valid,
		FirstValid:   fm.firstValid,
		LastValid:    fm.lastValid,
		Indexed:      indexed,
		FirstIndexed: fm.f.indexedRange.firstIndexedBlock,
		LastIndexed:  lastIndexed,
	}
	fm.valid = indexed
	fm.firstValid = fm.f.indexedRange.firstIndexedBlock
	fm.lastValid = lastIndexed
	fm.syncCh = nil
}

// SyncLogIndex ensures that the log index is consistent with the current state
// of the chain and is synced up to the current head. It blocks until this state
// is achieved or the context is cancelled.
// If successful, it returns a SyncRange that contains the latest chain head,
// the indexed range that is currently consistent with the chain and the valid
// range that has not been changed and has been consistent with all states of the
// chain since the previous SyncLogIndex or the creation of the matcher backend.
func (fm *FilterMapsMatcherBackend) SyncLogIndex(ctx context.Context) (SyncRange, error) {
	if fm.f.disabled {
		return SyncRange{HeadNumber: fm.f.targetView.headNumber}, nil
	}

	syncCh := make(chan SyncRange, 1)
	fm.f.matchersLock.Lock()
	fm.syncCh = syncCh
	fm.f.matchersLock.Unlock()

	select {
	case fm.f.matcherSyncCh <- fm:
	case <-ctx.Done():
		return SyncRange{}, ctx.Err()
	}
	select {
	case vr := <-syncCh:
		return vr, nil
	case <-ctx.Done():
		return SyncRange{}, ctx.Err()
	}
}

// updateMatchersValidRange iterates through active matchers and limits their
// valid range with the current indexed range. This function should be called
// whenever a part of the log index has been removed, before adding new blocks
// to it.
// Note that this function assumes that the index read lock is being held.
func (f *FilterMaps) updateMatchersValidRange() {
	f.matchersLock.Lock()
	defer f.matchersLock.Unlock()

	for fm := range f.matchers {
		if !f.indexedRange.hasIndexedBlocks() {
			fm.valid = false
		}
		if !fm.valid {
			continue
		}
		if fm.firstValid < f.indexedRange.firstIndexedBlock {
			fm.firstValid = f.indexedRange.firstIndexedBlock
		}
		if fm.lastValid >= f.indexedRange.afterLastIndexedBlock {
			fm.lastValid = f.indexedRange.afterLastIndexedBlock - 1
		}
		if fm.firstValid > fm.lastValid {
			fm.valid = false
		}
	}
}
