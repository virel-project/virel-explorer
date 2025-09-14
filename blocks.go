package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/virel-project/virel-blockchain/v3/rpc/daemonrpc"
)

type Blocks struct {
	mut    sync.RWMutex
	client *daemonrpc.RpcClient
	blocks []*daemonrpc.GetBlockResponse
	height uint64
}

func NewBlocks(cl *daemonrpc.RpcClient) *Blocks {
	return &Blocks{
		client: cl,
		blocks: make([]*daemonrpc.GetBlockResponse, 0),
	}
}

func (bl *Blocks) Updater() {
	var adj float64
	var n float64
	for {
		bl.mut.Lock()
		updated, adj2, err := bl.update(adj)
		bl.mut.Unlock()
		if err != nil {
			fmt.Println("failed to update:", err)
		}
		if adj != adj2 {
			n = min(n+1, 100)
			if adj == 0 {
				adj = adj2
			} else {
				adj = (adj*n + adj2) / (n + 1)
			}
			adj = min(max(adj, -30_000), 30_000) // limit timestamp adjustment to 30 seconds
			fmt.Println("adj2:", adj2, "adj:", adj)
		}
		if !updated {
			time.Sleep(2 * time.Second)
		}
	}
}
func (b *Blocks) GetList() []*daemonrpc.GetBlockResponse {
	b.mut.RLock()
	defer b.mut.RUnlock()

	fmt.Println("average block time:", float64(b.blocks[0].Block.Timestamp-b.blocks[len(b.blocks)-1].Block.Timestamp)/float64(len(b.blocks))/1000, "s")

	return b.blocks
}
func (b *Blocks) update(adj float64) (bool, float64, error) {
	info, err := b.client.GetInfo(daemonrpc.GetInfoRequest{})
	if err != nil {
		return false, adj, err
	}
	if b.height == 0 {
		if info.Height > MAX_BLOCKS_HISTORY {
			b.height = info.Height - MAX_BLOCKS_HISTORY
		} else {
			b.height = 1
		}
	}
	if b.height < info.Height {
		b.height++

		adj := float64(0)

		bl, err := b.client.GetBlockByHeight(daemonrpc.GetBlockByHeightRequest{
			Height: b.height,
		})
		if err != nil {
			return false, adj, err
		}

		if b.height == info.Height {
			adj = float64(bl.Block.Timestamp) - float64(time.Now().UnixMilli())
		}
		bl.Block.Timestamp = uint64(float64(bl.Block.Timestamp) - adj)

		b.blocks = append([]*daemonrpc.GetBlockResponse{bl}, b.blocks...)
		if len(b.blocks) > MAX_BLOCKS_HISTORY {
			b.blocks = b.blocks[:len(b.blocks)-1]
		}
		return true, adj, nil
	}
	return false, adj, nil
}
