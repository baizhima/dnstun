package main

import (
	"../lib/tun"
	"bufio"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func rpl(s *tun.Server) {

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fmt.Printf("server> ")
		cmd := strings.Split(scanner.Text(), " ")
		switch cmd[0] {
		case "ping":
			fmt.Println("server ping client not implemented")
		case "info":
			s.Info()
		case "send":
			if len(cmd) < 3 {
				fmt.Println("Usage: send userId abcdeabcde")
				continue
			}
			userId, err := strconv.Atoi(cmd[1])
			if err != nil {
				fmt.Println("Usage: send userId <message>")
                continue
			}
			if v, ok := s.Routes_By_UserId[userId]; ok {
				s.SendString(v, strings.Join(cmd[2:], " "))
			} else {
				fmt.Printf("client %d does not exist", userId)
			}
        case "help":
            printHelp()
		case "quit", "exit":
			fmt.Println("Goodbye!")
			return
		}
	}
	if err := scanner.Err(); err != nil {
		Error.Printf("reading standard input: %s\n", err)
	}
}

func printHelp() {
    fmt.Println("")
    fmt.Println("  ping")
    fmt.Println("  info")
    fmt.Println("  send <userId> <message>")
    fmt.Println("  quit/exit")
}


func main() {

	topDomainPtr := flag.String("d", DEF_TOP_DOMAIN, "Top Domain")
	laddrPtr := flag.String("l", DEF_DOMAIN_PORT, "Address of DNS Server")
	vaddrPtr := flag.String("v", "192.168.3.1", "Virtual IP Address of Tunneling Server")
	tunPtr := flag.String("t", "tun66", "Name of TUN Interface")

	flag.Parse()

	server, err := tun.NewServer(*topDomainPtr,
		*laddrPtr,
		*vaddrPtr,
		*tunPtr)
	if err != nil {
		Error.Println(err)
		return
	}

	go server.DNSRecv()
	go server.TUNRecv()
	rpl(server)

	return

}
