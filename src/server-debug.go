package main

import (
	"log"
	"net"
	"os"
	//"fmt"
	//"../lib/songgao/water"
	//"../lib/songgao/water/waterutil"
	//"../lib/tonnerre/golang-dns"
)

var (
	Debug *log.Logger = log.New(os.Stderr, "Debug: ", log.Lshortfile)
	Error *log.Logger = log.New(os.Stderr, "Error: ", log.Lshortfile)
)

func main() {

	raddr, _ := net.ResolveTCPAddr("tcp4", "52.90.132.77:53")
	laddr, _ := net.ResolveTCPAddr("tcp4", "0.0.0.0:5000")

	conn, err := net.DialTCP("tcp", laddr, raddr)
	if err != nil {
		Error.Println(err)
	}

	conn.Write([]byte("hello dns"))
	conn.Close()
}
