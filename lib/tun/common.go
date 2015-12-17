package tun

import (
	"log"
	"os"
)

var (
	Debug *log.Logger = log.New(os.Stderr, "Debug: ", log.Lshortfile)
	Error *log.Logger = log.New(os.Stderr, "Error: ", log.Lshortfile)
)

const (
	DEF_BUF_SIZE      int = 1500
	DEF_SENDSTRING_ID int = -5 // negative number to distinguish those normal ip packet id
	//DEF_TOP_DOMAIN string = "b.jannotti.com"
	//DEF_DOMAIN_PORT string = ":53"

    DEF_UPSTREAM_ENCODING_BASE int = 32
    DEF_DOWNSTREAM_ENCODING_BASE int = 64

	DEF_UPSTREAM_LABELS_PER_DNS   int = 4
	DEF_DOWNSTREAM_LABELS_PER_DNS int = 2

	DEF_UPSTREAM_LABEL_SIZE   int = 52
	DEF_DOWNSTREAM_LABEL_SIZE int = 200

)
