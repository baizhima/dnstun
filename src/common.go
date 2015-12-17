package main

import (
	"log"
	"os"
)

var (
	Debug *log.Logger = log.New(os.Stderr, "Debug: ", log.Lshortfile)
	Error *log.Logger = log.New(os.Stderr, "Error: ", log.Lshortfile)
)

const (
	DEF_BUF_SIZE    int    = 1500
	DEF_TOP_DOMAIN  string = "b.jannotti.com."
	DEF_DOMAIN_PORT string = ":53"
	DEF_LOCAL_DNS   string = "8.8.8.8:53"
)
