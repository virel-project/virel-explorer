package main

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"virel-explorer/html"

	"github.com/virel-project/virel-blockchain/v3/address"
	"github.com/virel-project/virel-blockchain/v3/config"
	"github.com/virel-project/virel-blockchain/v3/rpc/daemonrpc"
	"github.com/virel-project/virel-blockchain/v3/util"

	"github.com/labstack/echo/v4"
)

const MAX_BLOCKS_HISTORY = 50

func main() {
	d := daemonrpc.NewRpcClient("http://127.0.0.1:6311")

	bls := NewBlocks(d)
	go bls.Updater()

	updater := NewUpdater(d)
	go updater.Updater()

	e := echo.New()

	e.GET("/", func(c echo.Context) error {
		info, err := d.GetInfo(daemonrpc.GetInfoRequest{})
		if err != nil {
			return err
		}

		return html.Index(c, html.IndexParams{
			Info:   (*html.InfoRes)(info),
			Blocks: bls.GetList(),
		})
	})
	e.GET("/stats", func(c echo.Context) error {
		updaterOut := updater.Get()

		items := make([]html.RichListItem, len(updaterOut.RichList))

		for i, st := range updaterOut.RichList {
			items[i] = html.RichListItem{
				Rank:    i + 1,
				Address: st.Address,
				Balance: float64(st.State.Balance) / config.COIN,
				Percent: func() float64 {
					if updaterOut.MarketInfo.Supply == 0 {
						return 0
					}
					return float64(st.State.Balance) / config.COIN / float64(updaterOut.MarketInfo.Supply) * 100
				}(),
			}
		}
		info, err := d.GetInfo(daemonrpc.GetInfoRequest{})
		if err != nil {
			return err
		}

		ir := html.InfoRes(*info)

		return html.Stats(c, html.StatsParams{
			RichList: items,
			Market:   updaterOut.MarketInfo,
			Info:     &ir,
		})
	})
	e.GET("/delegates", func(c echo.Context) error {
		updaterOut := updater.Get()

		info, err := d.GetInfo(daemonrpc.GetInfoRequest{})
		if err != nil {
			return err
		}

		ir := html.InfoRes(*info)

		delegs := make([]*html.Delegate, 0, len(updaterOut.KnownDelegates))

		for _, v := range updaterOut.KnownDelegates {
			totStaked := max(v.BlocksMissed+v.BlocksStaked, 1)

			delegateInfo, err := d.GetDelegate(daemonrpc.GetDelegateRequest{
				DelegateAddress: v.Address,
			})
			if err != nil {
				return err
			}

			delegs = append(delegs, &html.Delegate{
				Address:        v.Address,
				Balance:        float64(delegateInfo.TotalAmount) / config.COIN,
				BalancePercent: float64(delegateInfo.TotalAmount) / float64(ir.Stake) * 100,
				UptimePercent:  float64(v.BlocksStaked) / float64(totStaked) * 100,
			})
		}

		return html.Delegates(c, html.DelegatesParams{
			Delegates: delegs,
		})
	})
	e.GET("/block/:bl", func(c echo.Context) error {
		bl := c.Param("bl")

		var res *daemonrpc.GetBlockResponse
		var err error
		if len(bl) == 64 {
			var hash []byte
			hash, err = hex.DecodeString(bl)
			if err != nil {
				return err
			}

			res, err = d.GetBlockByHash(daemonrpc.GetBlockByHashRequest{
				Hash: util.Hash(hash),
			})
		} else {
			var height uint64
			height, err = strconv.ParseUint(bl, 10, 64)
			if err != nil {
				return err
			}

			res, err = d.GetBlockByHeight(daemonrpc.GetBlockByHeightRequest{
				Height: height,
			})
		}
		if err != nil {
			return c.String(500, "failed to find block")
		}
		info, err := d.GetInfo(daemonrpc.GetInfoRequest{})
		if err != nil {
			return c.String(500, "failed to get info")
		}

		err = html.Block(c, html.BlockParams{
			Block: (*html.BlockRes)(res),
			Info:  info,
		})
		if err != nil {
			fmt.Println(err)
		}

		return err
	})
	e.GET("/tx/:txid", func(c echo.Context) error {
		txid := c.Param("txid")

		if len(txid) != 32*2 || !util.IsHex(txid) {
			return c.Redirect(http.StatusMovedPermanently, "/")
		}

		id, _ := hex.DecodeString(txid)

		res, err := d.GetTransaction(daemonrpc.GetTransactionRequest{
			Txid: [32]byte(id),
		})
		if err != nil {
			fmt.Println(err)
			return c.Redirect(http.StatusTemporaryRedirect, "/block/"+txid)
		}

		var confs uint64 = 0
		if res.Height != 0 && res.Height <= bls.height {
			confs = bls.height - res.Height + 1
		}

		err = html.Transaction(c, html.TransactionParams{
			Tx:    res,
			Txid:  txid,
			Confs: confs,
		})
		if err != nil {
			fmt.Println(err)
		}

		return err
	})
	e.GET("/account/:walletaddr", func(c echo.Context) error {
		walletaddr := c.Param(("walletaddr"))

		addr, err := address.FromString(walletaddr)
		if err != nil {
			return err
		}
		addr.PaymentId = 0

		addrInfo, err := d.GetAddress(daemonrpc.GetAddressRequest{
			Address: addr.String(),
		})
		if err != nil {
			return err
		}

		/* * Transactions * */
		// Pagination
		page := uint64(0)
		if p := c.QueryParam("page"); p != "" {
			if n, err := strconv.ParseUint(p, 10, 64); err == nil {
				page = n
			}
		}

		// Side
		transferType := c.QueryParam("transfer_type")
		if transferType != "incoming" && transferType != "outgoing" {
			transferType = "incoming"
		}

		// Transaction hash list
		txs, err := d.GetTxList(daemonrpc.GetTxListRequest{
			Address:      addr,
			TransferType: transferType,
			Page:         page,
		})
		if err != nil {
			return err
		}

		// Transaction list
		txList := make([]html.TransactionItem, 0, len(txs.Transactions))
		for _, id := range txs.Transactions {
			txRes, err := d.GetTransaction(daemonrpc.GetTransactionRequest{Txid: id})
			if err != nil {
				continue
			}

			txList = append(txList, html.TransactionItem{
				Tx:   txRes,
				Txid: id.String(),
				Amount: func() uint64 {
					if transferType == "incoming" {
						var sum uint64
						for _, o := range txRes.Outputs {
							if o.Recipient == addr.Addr {
								sum += o.Amount
							}
						}
						return sum
					}
					return txRes.TotalAmount
				}(),
			})
		}

		// Sort them by height
		sort.Slice(txList, func(a, b int) bool {
			return txList[a].Tx.Height > txList[b].Tx.Height
		})

		// For the timestamp of transactions, we need to fetch blocks
		blockTimes := make(map[uint64]string)
		for _, tx := range txList {
			if _, seen := blockTimes[tx.Tx.Height]; seen {
				continue
			}

			blkRes, err := d.GetBlockByHeight(daemonrpc.GetBlockByHeightRequest{Height: tx.Tx.Height})
			if err != nil {
				continue
			}

			// Convert Unix timestamp to UTC string
			blockTimes[tx.Tx.Height] = (*html.BlockRes)(blkRes).UTC()
		}

		return html.Address(c, html.AddressParams{
			Info:    addrInfo,
			Address: walletaddr,

			// Transactions
			Page:         page,
			MaxPage:      txs.MaxPage,
			TransferType: transferType,
			TxList:       txList,
			BlockTimes:   blockTimes,
		})
	})
	e.GET("/search", func(c echo.Context) error {
		query := strings.Trim(c.QueryParam("q"), " ")

		if len(query) == 64 {
			hexdata, err := hex.DecodeString(query)
			if err != nil {
				return err
			}

			_, err = d.GetTransaction(daemonrpc.GetTransactionRequest{
				Txid: util.Hash(hexdata),
			})
			if err == nil {
				return c.Redirect(http.StatusTemporaryRedirect, "/tx/"+query)
			} else {
				return c.Redirect(http.StatusTemporaryRedirect, "/block/"+query)
			}
		} else if len(query) > 16 {
			return c.Redirect(http.StatusTemporaryRedirect, "/account/"+query)
		}

		return c.Redirect(http.StatusTemporaryRedirect, "/block/"+query)
	})
	e.GET("/supply", func(c echo.Context) error {
		infoRes, err := d.GetInfo(daemonrpc.GetInfoRequest{})
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, fmt.Sprintf("%.2f", float64(infoRes.CirculatingSupply)/float64(infoRes.Coin)))
	})
	e.GET("/supply_rest", func(c echo.Context) error {
		infoRes, err := d.GetInfo(daemonrpc.GetInfoRequest{})
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, map[string]any{"result": fmt.Sprintf("%.2f", float64(infoRes.CirculatingSupply)/float64(infoRes.Coin))})
	})

	e.Static("/", "./static/")

	e.HTTPErrorHandler = customHTTPErrorHandler

	e.Logger.Fatal(e.Start(":8080"))
}
func customHTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
	} else {
		fmt.Println(err)
	}

	c.String(code, fmt.Sprintf("error: %d", code))
}
