package main

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"virel-blockchain/address"
	"virel-blockchain/rpc/daemonrpc"
	"virel-blockchain/util"
	"virel-explorer/html"

	"github.com/labstack/echo/v4"
)

const MAX_BLOCKS_HISTORY = 50

func main() {
	d := daemonrpc.NewRpcClient("http://127.0.0.1:6314")

	bls := NewBlocks(d)
	go bls.Updater()

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

		err = html.Transaction(c, html.TransactionParams{
			Tx:   res,
			Txid: txid,
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

		addrInfo, err := d.GetAddress(daemonrpc.GetAddressRequest{
			Address: addr,
		})
		if err != nil {
			return err
		}

		return html.Address(c, html.AddressParams{
			Info:    addrInfo,
			Address: walletaddr,
		})
	})
	e.GET("/search", func(c echo.Context) error {
		query := strings.Trim(c.QueryParam("q"), " ")

		if len(query) == 64 {
			hexdata, err := hex.DecodeString(query)
			if err != nil {
				return err
			}

			_, err = d.GetBlockByHash(daemonrpc.GetBlockByHashRequest{
				Hash: util.Hash(hexdata),
			})
			if err == nil {
				return c.Redirect(http.StatusTemporaryRedirect, "/block/"+query)
			} else {
				return c.Redirect(http.StatusTemporaryRedirect, "/tx/"+query)
			}
		} else if len(query) > 16 {
			return c.Redirect(http.StatusTemporaryRedirect, "/account/"+query)
		}

		return c.Redirect(http.StatusTemporaryRedirect, "/block/"+query)
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
