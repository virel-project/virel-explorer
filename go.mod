module virel-explorer

go 1.23.4

require github.com/labstack/echo/v4 v4.13.3

require virel-blockchain v1.0.0

require (
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/petermattis/goid v0.0.0-20250508124226-395b08cebbdb // indirect
	github.com/sasha-s/go-deadlock v0.3.5 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	github.com/virel-project/go-randomvirel v1.0.0 // indirect
	github.com/zeebo/blake3 v0.2.4 // indirect
	golang.org/x/crypto v0.38.0 // indirect
	golang.org/x/net v0.33.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.25.0 // indirect
)

replace virel-blockchain => ../virel-blockchain

replace github.com/virel-project/go-randomvirel => ../go-randomvirel
