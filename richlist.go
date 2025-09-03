package main

import (
	"fmt"
	"sync"
	"time"
	"virel-explorer/html"

	"github.com/virel-project/virel-blockchain/v2/rpc/daemonrpc"
)

type Updater struct {
	mut        sync.RWMutex
	client     *daemonrpc.RpcClient
	list       []daemonrpc.StateInfo
	marketinfo *html.MarketInfo
}

func NewUpdater(cl *daemonrpc.RpcClient) *Updater {
	return &Updater{client: cl}
}

func (r *Updater) Updater() {
	ticker := time.NewTicker(time.Minute)
	for {
		if err := r.update(); err != nil {
			fmt.Println("failed to update rich list:", err)
		}
		<-ticker.C
	}
}

type UpdaterOutput struct {
	RichList   []daemonrpc.StateInfo
	MarketInfo *html.MarketInfo
}

func (r *Updater) Get() UpdaterOutput {
	r.mut.RLock()
	defer r.mut.RUnlock()

	out := make([]daemonrpc.StateInfo, len(r.list))
	copy(out, r.list)

	return UpdaterOutput{
		RichList:   out,
		MarketInfo: r.marketinfo,
	}
}

func (r *Updater) update() error {
	info, err := r.client.GetInfo(daemonrpc.GetInfoRequest{})
	if err != nil {
		return err
	}
	res, err := r.client.GetRichList(daemonrpc.RichListRequest{})
	if err != nil {
		return err
	}

	mkt, err := GetMarketInfo(info.CirculatingSupply)
	if err != nil {
		return err
	}

	r.mut.Lock()
	r.list = res.Richest
	r.marketinfo = mkt
	r.mut.Unlock()

	return nil
}
