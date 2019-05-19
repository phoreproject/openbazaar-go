package blockchain

import (
	"encoding/binary"
	"math/big"
	"sort"
	"time"

	"github.com/phoreproject/btcd/chaincfg/chainhash"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (b *BlockChain) selectBlockFromCandidates(nodesSortedByTimestamp []timeAndBlockNode, selectedBlocks map[chainhash.Hash]*blockNode, selectionIntervalStop int64, stakeModifier uint64) (*blockNode, error) {
	modifierV2 := false
	selected := false

	hashBest := big.NewInt(0)
	var selectedBlock *blockNode

	modifierV2 = len(nodesSortedByTimestamp) > 0 && nodesSortedByTimestamp[0].node.height >= b.chainParams.ModifierV2StartBlock

	for i := range nodesSortedByTimestamp {
		node := nodesSortedByTimestamp[i].node
		if selected && node.timestamp > selectionIntervalStop {
			break
		}

		// skip blocks that have already been selected
		if _, ok := selectedBlocks[node.hash]; ok {
			continue
		}

		var hashProof chainhash.Hash
		if !modifierV2 && b.index.IsProofOfStake(node) {
			for i := range hashProof {
				hashProof[i] = 0
			}
		} else {
			hashProof = node.hash
		}

		smb := make([]byte, 8)
		binary.LittleEndian.PutUint64(smb, stakeModifier)

		toHash := append(hashProof[:], smb...)

		hashSelection := chainhash.HashH(toHash)

		if b.index.IsProofOfStake(node) {
			// shift right by 4
			copy(hashSelection[:], append([]byte{0x00, 0x00, 0x00, 0x00}, hashSelection[:4]...))
		}

		hashSelectionBig := HashToBig(&hashSelection)

		if selected && hashSelectionBig.Cmp(hashBest) < 0 {
			hashBest = hashSelectionBig
			selectedBlock = node
		} else if !selected {
			selected = true
			hashBest = hashSelectionBig
			selectedBlock = node
		}
	}

	return selectedBlock, nil
}

type timeAndBlockNode struct {
	node *blockNode
	time time.Time
}

const blockOneStakeModifier = 4539363955

// computeNextStakeModifier gets the next block's stake modifier and returns whether one
// was generated and if so, what the stake modifier is
// For the genesis block, the stake modifier is 0.
func (b *BlockChain) computeNextStakeModifier(node *blockNode) (uint64, bool, error) {
	if node.height == 0 || node.height == 1 {
		return blockOneStakeModifier, true, nil
	}

	lastStakeModifierBlock := node
	for lastStakeModifierBlock != nil && !b.index.GeneratedStakeModifier(lastStakeModifierBlock) {
		lastStakeModifierBlock = lastStakeModifierBlock.parent
	}

	stakeModifier := lastStakeModifierBlock.stakeModifier

	modifierTime := lastStakeModifierBlock.timestamp

	if modifierTime/60 >= node.timestamp/60 {
		return b.index.GetStakeModifier(lastStakeModifierBlock), false, nil
	}

	selectionIntervalStart := (node.timestamp/60)*60 - int64(stakeModifierSelectionInterval.Seconds())

	var nodesSortedByTimestamp []timeAndBlockNode

	for node != nil && node.timestamp >= selectionIntervalStart {
		nodesSortedByTimestamp = append([]timeAndBlockNode{
			{time: time.Unix(node.timestamp, 0), node: node},
		}, nodesSortedByTimestamp...)
		node = node.parent
	}

	sort.Slice(nodesSortedByTimestamp, func(a, b int) bool {
		if nodesSortedByTimestamp[a].time.Unix() == nodesSortedByTimestamp[b].time.Unix() {
			return HashToBig(&nodesSortedByTimestamp[a].node.hash).Cmp(HashToBig(&nodesSortedByTimestamp[b].node.hash)) < 0
		}
		return nodesSortedByTimestamp[a].time.Before(nodesSortedByTimestamp[b].time)
	})

	stakeModifierNew := uint64(0)
	selectionIntervalStop := selectionIntervalStart
	selectedBlocks := make(map[chainhash.Hash]*blockNode)

	for round := 0; round < min(64, len(nodesSortedByTimestamp)); round++ {
		selectionIntervalStop += int64(3780 / (189 - round*2))

		selection, err := b.selectBlockFromCandidates(nodesSortedByTimestamp, selectedBlocks, selectionIntervalStop, stakeModifier)

		if err != nil {
			return 0, false, err
		}

		stakeModifierNew |= uint64(selection.GetStakeEntropyBit()) << uint(round)

		selectedBlocks[selection.hash] = selection
	}

	return stakeModifierNew, true, nil
}
