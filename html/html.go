package html

import (
	"bytes"
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"
	"virel-explorer/util"

	"github.com/virel-project/virel-blockchain/rpc/daemonrpc"
	sutil "github.com/virel-project/virel-blockchain/util"

	"github.com/labstack/echo/v4"
)

func parse(file string) *template.Template {
	// const path = "templates/"
	const path = "./html/templates/"

	//return template.Must(template.New("layout.html").Funcs(funcs).ParseFS(files, path+"layout.html", path+file))
	return template.Must(
		template.New("layout.html").Funcs(funcs).ParseFiles(path+"layout.html", path+file, path+"header.html"))
}

var funcs = template.FuncMap{
	"toUpper": func(s string) string {
		return strings.ToUpper(s)
	},
	"toLower": func(s string) string {
		return strings.ToLower(s)
	},
	"div": func(a, b float64) float64 {
		return a / b
	},
	"isGreater": func(a, b uint64) bool {
		return a > b
	},
	"isGreaterEq": func(a, b uint64) bool {
		return a >= b
	},
	"age_ms": func(t uint64) string {
		return time.Since(time.UnixMilli(int64(t))).Round(time.Second).String()
	},
	"fmt_coin": func(n uint64) string {
		return sutil.FormatCoin(n)
	},
	"add": func(a, b uint64) uint64 {
		return a + b
	},
	"sub": func(a, b uint64) uint64 {
		return a - b
	},
	"entity": func(s string) string {
		if len(Entities[s]) > 0 {
			return Entities[s]
		}
		return s
	},
}

type IndexParams struct {
	Blocks []*daemonrpc.GetBlockResponse
	Info   *InfoRes
}
type InfoRes daemonrpc.GetInfoResponse

func Index(c echo.Context, p IndexParams) error {
	b := bytes.NewBuffer([]byte{})
	err := parse("index.html").Execute(b, p)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return c.HTMLBlob(200, b.Bytes())
}

type BlockParams struct {
	Block *BlockRes
	Info  *daemonrpc.GetInfoResponse
}
type BlockRes daemonrpc.GetBlockResponse

func Block(c echo.Context, p BlockParams) error {
	b := bytes.NewBuffer([]byte{})
	err := parse("block.html").Execute(b, p)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return c.HTMLBlob(200, b.Bytes())
}

type TransactionParams struct {
	Tx   *daemonrpc.GetTransactionResponse
	Txid string
}

func Transaction(c echo.Context, p TransactionParams) error {
	b := bytes.NewBuffer([]byte{})
	err := parse("transaction.html").Execute(b, p)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return c.HTMLBlob(200, b.Bytes())
}

/* * Rich List * */
type RichListItem struct {
	Rank    int
	Address string
	Balance uint64
	Percent float64
}

type RichListParams struct {
	List []RichListItem
}

func RichList(c echo.Context, p RichListParams) error {
	b := bytes.NewBuffer([]byte{})
	err := parse("richlist.html").Execute(b, p)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return c.HTMLBlob(200, b.Bytes())
}

type TransactionItem struct {
	Tx     *daemonrpc.GetTransactionResponse
	Txid   string
	Amount uint64
}

type AddressParams struct {
	Address string
	Info    *daemonrpc.GetAddressResponse

	// Transactions
	Page         uint64            // page number for pagination
	MaxPage      uint64            // total number of available pages
	TransferType string            // side: incoming / outgoing
	TxList       []TransactionItem // list of transaction (id + tx)
	BlockTimes   map[uint64]string // block timestamps (to show transaction timestamps in UTC)
}

func Address(c echo.Context, p AddressParams) error {
	b := bytes.NewBuffer([]byte{})
	err := parse("address.html").Execute(b, p)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return c.HTMLBlob(200, b.Bytes())
}

func (b *BlockRes) PrintReward() string {
	return sutil.FormatCoin(b.MinerReward)
}
func (b *BlockRes) UTC() string {
	return time.UnixMilli(int64(b.Block.Timestamp)).Format("2006-01-02 15:04")
}
func (b *BlockRes) Prev() uint64 {
	if b.Block.Height == 0 {
		return 0
	}
	return b.Block.Height - 1
}
func (b *BlockRes) Next() uint64 {
	return b.Block.Height + 1
}

func (i *InfoRes) Hashrate() string {
	diff, err := strconv.ParseFloat(i.Difficulty, 64)
	if err != nil {
		panic(err)
	}

	return util.Unit(diff/float64(i.Target)) + "H/s"
}

func (i *InfoRes) Reward() string {
	return strconv.FormatFloat(float64(i.BlockReward)/float64(i.Coin), 'f', 2, 64) + " VRL"
}

func formatNumber(n float64) string {
	switch {
	case n >= 1_000_000_000:
		return strconv.FormatFloat(n/1_000_000_000, 'f', 2, 64) + "G"
	case n >= 1_000_000:
		return strconv.FormatFloat(n/1_000_000, 'f', 2, 64) + "M"
	case n >= 1_000:
		return strconv.FormatFloat(n/1_000, 'f', 2, 64) + "K"
	default:
		return strconv.FormatFloat(n, 'f', 2, 64)
	}
}

func (i *InfoRes) Circulating() string {
	return formatNumber(float64(i.CirculatingSupply) / float64(i.Coin))
}

func (i *InfoRes) CirculatingPercent() string {
	return strconv.FormatFloat(float64(i.CirculatingSupply)/float64(i.MaxSupply)*100, 'f', 2, 64) + "%"
}
