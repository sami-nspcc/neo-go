package neotest

import (
	"fmt"
	"testing"

	"github.com/nspcc-dev/neo-go/pkg/core/block"
	"github.com/nspcc-dev/neo-go/pkg/core/blockchainer"
	"github.com/nspcc-dev/neo-go/pkg/core/transaction"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/callflag"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/trigger"
	"github.com/nspcc-dev/neo-go/pkg/util"
)

var (
	Blocks   map[string][]testing.CoverBlock
	Counters map[string][]uint32
)

func init() {
	var cover testing.Cover
	cover.Mode = testing.CoverMode()
	Blocks = make(map[string][]testing.CoverBlock)
	Counters = make(map[string][]uint32)
	cover.Blocks = Blocks
	cover.Counters = Counters
	testing.RegisterCover(cover)
}

func calculateCoverage(t testing.TB, bc blockchainer.Blockchainer, tx *transaction.Transaction, b *block.Block) {
	fmt.Println("CALCULATE COVERAGE")
	ic := bc.GetTestVM(trigger.Application, tx, b)
	t.Cleanup(ic.Finalize)

	ic.VM.LoadWithFlags(tx.Script, callflag.All)

	hm := make(map[util.Uint160][]int)

runLoop:
	for {
		switch {
		case ic.VM.HasStopped():
			break runLoop
		default:
			h := ic.VM.GetCurrentScriptHash()
			if err := ic.VM.Step(); err != nil {
				break runLoop
			}
			if ic.VM.Context() != nil {
				hm[h] = append(hm[h], ic.VM.Context().IP())
			}
		}
	}

	// Calculate coverage.
coverageLoop:
	for h := range hm {
		fmt.Println("TRY HASH", h)
		for name, c := range contracts {
			fmt.Println("TRY CONTRACT", name, c.Hash)
			if !c.Hash.Equals(h) {
				continue
			}
			if c.DebugInfo == nil {
				continue coverageLoop
			}

			var coverBlocks []testing.CoverBlock
			var coverCounts []uint32

		ipLoop:
			for _, ip := range hm[h] {
				for _, m := range c.DebugInfo.Methods {
					for _, sp := range m.SeqPoints {
						if sp.Opcode == ip {
							fmt.Println("FOUND COVER BLOCK")
							coverCounts = append(coverCounts, 1)
							coverBlocks = append(coverBlocks, testing.CoverBlock{
								Line0: uint32(sp.StartLine),
								Col0:  uint16(sp.StartCol),
								Line1: uint32(sp.EndLine),
								Col1:  uint16(sp.EndCol),
								Stmts: 1,
							})
							continue ipLoop
						}
					}
				}
			}
			Blocks[name] = coverBlocks
			Counters[name] = coverCounts
			continue coverageLoop
		}
	}
}
