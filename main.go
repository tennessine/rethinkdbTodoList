package main

import "flag"

var (
	addr string = "localhost:3000"
)

func main() {

	flag.StringVar(&addr, "addr", "localhost:3000", "")
	flag.Parse()

	server := NewServer(addr)
	StartServer(server)
}
