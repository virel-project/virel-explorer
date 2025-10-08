package html

import (
	"bytes"
	"fmt"
	"html/template"
	"math"
	"strconv"
	"strings"
	"time"
	"virel-explorer/util"

	"github.com/virel-project/virel-blockchain/v3/address"
	"github.com/virel-project/virel-blockchain/v3/config"
	"github.com/virel-project/virel-blockchain/v3/rpc/daemonrpc"
	sutil "github.com/virel-project/virel-blockchain/v3/util"

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
	"fmt_coin_int": func(n uint64) string {
		s := strconv.FormatUint(n/config.COIN, 10)

		return s
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
	Tx    *daemonrpc.GetTransactionResponse
	Txid  string
	Confs uint64
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
	Balance float64
	Percent float64
}

type MarketInfo struct {
	Price     float64
	Marketcap float64
	Supply    float64
	Change    string
}

func (m *MarketInfo) IsPositiveChange() bool {
	return strings.HasPrefix(m.Change, "+")
}
func (m *MarketInfo) FormatPrice() string {
	return strconv.FormatFloat(m.Price, 'f', 4, 64) + " $"
}
func (m *MarketInfo) FormatMarketcap() string {
	mkt := math.Round(m.Marketcap)

	if mkt > 1_000_000 {
		return strconv.FormatFloat(mkt/1_000_000, 'f', 2, 64) + "M $"
	}
	if mkt > 1_000 {
		return strconv.FormatFloat(mkt/1_000, 'f', 2, 64) + "k $"
	}

	return strconv.FormatFloat(m.Marketcap, 'f', 0, 64) + " $"
}

type StatsParams struct {
	RichList []RichListItem
	Info     *InfoRes
	Market   *MarketInfo
}

func Stats(c echo.Context, p StatsParams) error {
	b := bytes.NewBuffer([]byte{})
	err := parse("stats.html").Execute(b, p)
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

type DelegateParams struct {
	Address string
	Info    *daemonrpc.GetDelegateResponse
	Height  uint64
	Funds   []*Fund
}

type Fund struct {
	Owner  address.Address `json:"owner"`
	Amount uint64          `json:"amount"`
	Unlock uint64          `json:"unlock"` // height of unlock of this fund
}

func (f *Fund) UnlockTime(height uint64) string {
	if f.Unlock < height {
		return "unlocked"
	}

	remaining := (time.Duration(int64(f.Unlock)-int64(height)) * config.TARGET_BLOCK_TIME * time.Second)
	if remaining > 24*time.Hour {
		days := math.Floor(remaining.Hours() / 24)
		hours := remaining.Hours() - days*24

		return fmt.Sprintf("%.0fd %.0fh", days, hours)
	}

	return remaining.String()
}

func (d *DelegateParams) Staked() string {
	return sutil.FormatCoin(d.Info.TotalAmount)
}

func Delegate(c echo.Context, p *DelegateParams) error {
	b := bytes.NewBuffer([]byte{})
	err := parse("delegate.html").Execute(b, p)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return c.HTMLBlob(200, b.Bytes())
}

func (b *BlockRes) PrintReward() string {
	return sutil.FormatCoin(b.TotalReward)
}
func (b *BlockRes) PrintMinerReward() string {
	return sutil.FormatCoin(b.MinerReward)
}
func (b *BlockRes) PrintStakerReward() string {
	return sutil.FormatCoin(b.StakerReward)
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
		return strconv.FormatFloat(n/1_000_000, 'f', 1, 64) + "M"
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
	return strconv.FormatFloat(float64(i.CirculatingSupply)/float64(i.SupplyCap)*100, 'f', 2, 64) + "%"
}

func (i *InfoRes) TotalSupplyStr() string {
	return formatNumber(float64(i.TotalSupply) / float64(i.Coin))
}

func (i *InfoRes) TotalSupplyPercent() string {
	return strconv.FormatFloat(float64(i.TotalSupply)/float64(i.SupplyCap)*100, 'f', 2, 64) + "%"
}

func (i *InfoRes) BurnedStr() string {
	return formatNumber(float64(i.Burned) / float64(i.Coin))
}

func (i *InfoRes) BurnedPercent() string {
	return strconv.FormatFloat(float64(i.Burned)/float64(i.SupplyCap)*100, 'f', 2, 64) + "%"
}

func (i *InfoRes) MaxSupplyStr() string {
	return formatNumber(float64(i.MaxSupply) / float64(i.Coin))
}

func (i *InfoRes) StakeStr() string {
	return formatNumber(float64(i.Stake) / float64(i.Coin))
}
func (i *InfoRes) StakePercent() string {
	return strconv.FormatFloat(float64(i.Stake)/float64(i.CirculatingSupply)*100, 'f', 2, 64) + "%"
}

func (i *InfoRes) Cap() string {
	return formatNumber(float64(i.SupplyCap) / float64(i.Coin))
}

type DelegatesParams struct {
	Delegates []*DelegateInfo
}

type DelegateInfo struct {
	Address        string
	Description    string
	Balance        float64
	BalancePercent float64
	UptimePercent  float64
}

func Delegates(c echo.Context, p DelegatesParams) error {
	b := bytes.NewBuffer([]byte{})
	err := parse("delegates.html").Execute(b, p)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return c.HTMLBlob(200, b.Bytes())
}
