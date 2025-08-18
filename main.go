package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"virel-explorer/html"

	"github.com/labstack/echo/v4"
	"github.com/virel-project/virel-blockchain/rpc/daemonrpc"
	"github.com/virel-project/virel-blockchain/util"
)

const MAX_BLOCKS_HISTORY = 50
const daemonURL = "http://127.0.0.1:6311/json_rpc"

var hex64 = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

// --- generic JSON-RPC request/response wrapper ---
type jsonRPCReq struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonRPCRes[T any] struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  *T     `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// --- helper: get_address ---
func getAddressViaRPC(addr string) (*daemonrpc.GetAddressResponse, error) {
	req := jsonRPCReq{
		JSONRPC: "2.0",
		ID:      0,
		Method:  "get_address",
		Params:  map[string]string{"address": addr},
	}
	body, _ := json.Marshal(req)

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Post(daemonURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var out jsonRPCRes[daemonrpc.GetAddressResponse]
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, fmt.Errorf("daemon error %d: %s", out.Error.Code, out.Error.Message)
	}
	return out.Result, nil
}

// --- helper: get_transaction ---
// NOTE: uses html.TxResponse (single source of truth for the transaction shape)
func getTransactionViaRPC(txid string) (*html.TxResponse, error) {
	req := jsonRPCReq{
		JSONRPC: "2.0",
		ID:      0,
		Method:  "get_transaction",
		Params:  map[string]string{"txid": txid},
	}
	body, _ := json.Marshal(req)

	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Post(daemonURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	var out jsonRPCRes[html.TxResponse]
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.Error != nil {
		return nil, fmt.Errorf("daemon error %d: %s", out.Error.Code, out.Error.Message)
	}
	return out.Result, nil
}

func main() {
	d := daemonrpc.NewRpcClient("http://127.0.0.1:6311")

	bls := NewBlocks(d)
	go bls.Updater()

	e := echo.New()

	// Home
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

	// Block by height OR hash
	e.GET("/block/:bl", func(c echo.Context) error {
		bl := c.Param("bl")
		var res *daemonrpc.GetBlockResponse
		var err error

		if hex64.MatchString(bl) {
			hb, errDec := hex.DecodeString(bl)
			if errDec != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid block hash")
			}
			res, err = d.GetBlockByHash(daemonrpc.GetBlockByHashRequest{
				Hash: util.Hash(hb),
			})
		} else {
			height, errParse := strconv.ParseUint(bl, 10, 64)
			if errParse != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "block id must be height or 64-hex hash")
			}
			res, err = d.GetBlockByHeight(daemonrpc.GetBlockByHeightRequest{Height: height})
		}

		if err != nil {
			fmt.Println("get block failed:", err)
			return c.String(http.StatusNotFound, "failed to find block")
		}

		info, err := d.GetInfo(daemonrpc.GetInfoRequest{})
		if err != nil {
			fmt.Println("get info failed:", err)
			return c.String(http.StatusInternalServerError, "failed to get info")
		}

		return html.Block(c, html.BlockParams{
			Block: (*html.BlockRes)(res),
			Info:  info,
		})
	})

	// Transaction page (uses RPC helper)
	e.GET("/tx/:txid", func(c echo.Context) error {
		txid := strings.TrimSpace(c.Param("txid"))
		if !hex64.MatchString(txid) {
			return echo.NewHTTPError(http.StatusBadRequest, "txid must be 64-hex")
		}

		res, err := getTransactionViaRPC(txid)
		if err != nil {
			fmt.Println("get tx failed:", err)
			return c.String(http.StatusNotFound, "transaction not found")
		}

		return html.Transaction(c, html.TransactionParams{
			Tx:   res,
			Txid: txid,
		})
	})

	// Address page
	e.GET("/account/:walletaddr", func(c echo.Context) error {
		walletaddr := strings.TrimSpace(c.Param("walletaddr"))

		info, err := getAddressViaRPC(walletaddr)
		if err != nil {
			fmt.Println("get address failed:", err)
			return c.String(http.StatusNotFound, "address not found or invalid")
		}

		return html.Address(c, html.AddressParams{
			Info:    info,
			Address: walletaddr,
		})
	})

	// Search
	e.GET("/search", func(c echo.Context) error {
		q := strings.TrimSpace(c.QueryParam("q"))
		if q == "" {
			return c.Redirect(http.StatusFound, "/")
		}
		if h, err := strconv.ParseUint(q, 10, 64); err == nil {
			return c.Redirect(http.StatusFound, "/block/"+strconv.FormatUint(h, 10))
		}
		if hex64.MatchString(q) {
			if hb, err := hex.DecodeString(q); err == nil {
				if _, err := d.GetBlockByHash(daemonrpc.GetBlockByHashRequest{Hash: util.Hash(hb)}); err == nil {
					return c.Redirect(http.StatusFound, "/block/"+q)
				}
			}
			return c.Redirect(http.StatusFound, "/tx/"+q)
		}
		return c.Redirect(http.StatusFound, "/account/"+q)
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
	_ = c.String(code, fmt.Sprintf("error: %d", code))
}
