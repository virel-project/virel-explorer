package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/virel-project/virel-blockchain/rpc/daemonrpc"
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
	for {
		bl.mut.Lock()
		updated, err := bl.update()
		bl.mut.Unlock()
		if err != nil {
			fmt.Println("failed to update:", err)
		}
		if !updated {
			time.Sleep(5 * time.Second)
		}
	}
}
func (b *Blocks) GetList() []*daemonrpc.GetBlockResponse {
	b.mut.RLock()
	defer b.mut.RUnlock()

	fmt.Println("average block time:", float64(b.blocks[0].Block.Timestamp-b.blocks[len(b.blocks)-1].Block.Timestamp)/float64(len(b.blocks))/1000, "s")

	return b.blocks
}
func (b *Blocks) update() (bool, error) {
	info, err := b.client.GetInfo(daemonrpc.GetInfoRequest{})
	if err != nil {
		return false, err
	}
	if b.height == 0 {
		if info.Height > MAX_BLOCKS_HISTORY {
			b.height = info.Height - MAX_BLOCKS_HISTORY
		} else {
			b.height = 1
		}
	}
	if b.height != info.Height {
		b.height++

		bl, err := b.client.GetBlockByHeight(daemonrpc.GetBlockByHeightRequest{
			Height: b.height,
		})
		if err != nil {
			return false, err
		}
		b.blocks = append([]*daemonrpc.GetBlockResponse{bl}, b.blocks...)
		if len(b.blocks) > MAX_BLOCKS_HISTORY {
			b.blocks = b.blocks[:len(b.blocks)-1]
		}
		return true, nil
	}
	return false, nil
}
