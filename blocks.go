package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/virel-project/virel-blockchain/v3/bitcrypto"
	"github.com/virel-project/virel-blockchain/v3/rpc/daemonrpc"
)

type Blocks struct {
	mut            sync.RWMutex
	client         *daemonrpc.RpcClient
	blocks         []*daemonrpc.GetBlockResponse
	KnownDelegates []*KnownDelegate
	height         uint64
}

func NewBlocks(cl *daemonrpc.RpcClient) *Blocks {
	b := &Blocks{
		client:         cl,
		blocks:         make([]*daemonrpc.GetBlockResponse, 0),
		KnownDelegates: make([]*KnownDelegate, 0),
	}

	delegates, err := os.ReadFile("delegates.json")
	if err != nil {
		fmt.Println(err)
		return b
	}

	err = json.Unmarshal(delegates, &b.KnownDelegates)
	if err != nil {
		fmt.Println(err)
		return b
	}

	return b
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
func (b *Blocks) GetDelegates() []*KnownDelegate {
	b.mut.RLock()
	defer b.mut.RUnlock()

	return b.KnownDelegates
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

		if bl.Block.DelegateId != 0 {
			var deleg *KnownDelegate
			for _, v := range b.KnownDelegates {
				if v.Id == bl.Block.DelegateId {
					deleg = v
				}
			}
			if deleg == nil {
				deleg = &KnownDelegate{
					Id: bl.Block.DelegateId,
				}
				b.KnownDelegates = append(b.KnownDelegates, deleg)
			}
			if deleg.LastHeight < bl.Block.Height {
				deleg.LastHeight = bl.Block.Height
				if bl.Block.StakeSignature == bitcrypto.BlankSignature {
					deleg.BlocksMissed++
				} else {
					deleg.BlocksStaked++
				}
				delegs, err := json.Marshal(b.KnownDelegates)
				if err != nil {
					fmt.Println(err)
				} else {
					err = os.WriteFile("delegates.json", delegs, 0o660)
					if err != nil {
						fmt.Println(err)
					}
				}
			}
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
