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
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestSingleMatch(t *testing.T) {
	params := DefaultParams
	params.deriveFields()

	for count := 0; count < 100000; count++ {
		// generate a row with a single random entry
		mapIndex := rand.Uint32()
		lvIndex := uint64(mapIndex)<<params.logValuesPerMap + uint64(rand.Intn(int(params.valuesPerMap)))
		var lvHash common.Hash
		crand.Read(lvHash[:])
		row := FilterRow{params.columnIndex(lvIndex, &lvHash)}
		matches := params.potentialMatches([]FilterRow{row}, mapIndex, lvHash)
		// check if it has been reverse transformed correctly
		if len(matches) != 1 {
			t.Fatalf("Invalid length of matches (got %d, expected 1)", len(matches))
		}
		if matches[0] != lvIndex {
			if len(matches) != 1 {
				t.Fatalf("Incorrect match returned (got %d, expected %d)", matches[0], lvIndex)
			}
		}
	}
}

const (
	testPmCount = 50
	testPmLen   = 1000
)

func TestPotentialMatches(t *testing.T) {
	params := DefaultParams
	params.deriveFields()

	var falsePositives int
	for count := 0; count < testPmCount; count++ {
		mapIndex := rand.Uint32()
		lvStart := uint64(mapIndex) << params.logValuesPerMap
		var row FilterRow
		lvIndices := make([]uint64, testPmLen)
		lvHashes := make([]common.Hash, testPmLen+1)
		for i := range lvIndices {
			// add testPmLen single entries with different log value hashes at different indices
			lvIndices[i] = lvStart + uint64(rand.Intn(int(params.valuesPerMap)))
			crand.Read(lvHashes[i][:])
			row = append(row, params.columnIndex(lvIndices[i], &lvHashes[i]))
		}
		// add the same log value hash at the first testPmLen log value indices of the map's range
		crand.Read(lvHashes[testPmLen][:])
		for lvIndex := lvStart; lvIndex < lvStart+testPmLen; lvIndex++ {
			row = append(row, params.columnIndex(lvIndex, &lvHashes[testPmLen]))
		}
		// randomly duplicate some entries
		for i := 0; i < testPmLen; i++ {
			row = append(row, row[rand.Intn(len(row))])
		}
		// randomly mix up order of elements
		for i := len(row) - 1; i > 0; i-- {
			j := rand.Intn(i)
			row[i], row[j] = row[j], row[i]
		}
		// split up into a list of rows if longer than allowed
		var rows []FilterRow
		for layerIndex := uint32(0); row != nil; layerIndex++ {
			maxLen := int(params.maxRowLength(layerIndex))
			if len(row) > maxLen {
				rows = append(rows, row[:maxLen])
				row = row[maxLen:]
			} else {
				rows = append(rows, row)
				row = nil
			}
		}
		// check retrieved matches while also counting false positives
		for i, lvHash := range lvHashes {
			matches := params.potentialMatches(rows, mapIndex, lvHash)
			if i < testPmLen {
				// check single entry match
				if len(matches) < 1 {
					t.Fatalf("Invalid length of matches (got %d, expected >=1)", len(matches))
				}
				var found bool
				for _, lvi := range matches {
					if lvi == lvIndices[i] {
						found = true
					} else {
						falsePositives++
					}
				}
				if !found {
					t.Fatalf("Expected match not found (got %v, expected %d)", matches, lvIndices[i])
				}
			} else {
				// check "long series" match
				if len(matches) < testPmLen {
					t.Fatalf("Invalid length of matches (got %d, expected >=%d)", len(matches), testPmLen)
				}
				// since results are ordered, first testPmLen entries should always match exactly
				for j := 0; j < testPmLen; j++ {
					if matches[j] != lvStart+uint64(j) {
						t.Fatalf("Incorrect match at index %d (got %d, expected %d)", j, matches[j], lvStart+uint64(j))
					}
				}
				// the rest are false positives
				falsePositives += len(matches) - testPmLen
			}
		}
	}
	// Whenever looking for a certain log value hash, each entry in the row that
	// was generated by another log value hash (a "foreign entry") has a
	// valuesPerMap // 2^32 chance of yielding a false positive if the reverse
	// transformed 32 bit integer is by random chance less than valuesPerMap and
	// is therefore considered a potentially valid match.
	// We have testPmLen unique hash entries and a testPmLen long series of entries
	// for the same hash. For each of the testPmLen unique hash entries there are
	// testPmLen*2-1 foreign entries while for the long series there are testPmLen
	// foreign entries. This means that after performing all these filtering runs,
	// we have processed 2*testPmLen^2 foreign entries, which given us an estimate
	// of how many false positives to expect.
	expFalse := int(uint64(testPmCount*testPmLen*testPmLen*2) * params.valuesPerMap >> params.logMapWidth)
	if falsePositives < expFalse/2 || falsePositives > expFalse*3/2 {
		t.Fatalf("False positive rate out of expected range (got %d, expected %d +-50%%)", falsePositives, expFalse)
	}
}
