package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/virel-project/virel-blockchain/rpc/daemonrpc"
)

type RichList struct {
	mut    sync.RWMutex
	client *daemonrpc.RpcClient
	list   []daemonrpc.StateInfo
	supply uint64
}

func NewRichList(cl *daemonrpc.RpcClient) *RichList {
	return &RichList{client: cl}
}

func (r *RichList) Updater() {
	ticker := time.NewTicker(time.Minute)
	for {
		if err := r.update(); err != nil {
			fmt.Println("failed to update rich list:", err)
		}
		<-ticker.C
	}
}

func (r *RichList) Get() ([]daemonrpc.StateInfo, uint64) {
	r.mut.RLock()
	defer r.mut.RUnlock()

	out := make([]daemonrpc.StateInfo, len(r.list))
	copy(out, r.list)
	return out, r.supply
}

func (r *RichList) update() error {
	info, err := r.client.GetInfo(daemonrpc.GetInfoRequest{})
	if err != nil {
		return err
	}
	res, err := r.client.GetRichList(daemonrpc.RichListRequest{})
	if err != nil {
		return err
	}

	r.mut.Lock()
	r.list = res.Richest
	r.supply = info.CirculatingSupply
	r.mut.Unlock()

	return nil
}
