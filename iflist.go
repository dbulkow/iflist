package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
)

func main() {
	var mclen int
	var iflen int
	var idxlen int
	var mtulen int

	fmt.Println("Hostname")
	fmt.Println("--------")
	hostname, _ := os.Hostname()
	fmt.Println(hostname)
	fmt.Println("")

	ifaces, err := net.Interfaces()
	if err != nil {
		fmt.Println(err)
		return
	}

	mclen = 0
	iflen = 0
	idxlen = 0
	mtulen = 0
	for i := range ifaces {
		buf := fmt.Sprintf("%v", ifaces[i].Name)
		if iflen < len(buf) {
			iflen = len(buf)
		}
		buf = fmt.Sprintf("%v", ifaces[i].HardwareAddr)
		if mclen < len(buf) {
			mclen = len(buf)
		}
		buf = fmt.Sprintf("%d", ifaces[i].Index)
		if idxlen < len(buf) {
			idxlen = len(buf)
		}
		buf = fmt.Sprintf("%d", ifaces[i].MTU)
		if mtulen < len(buf) {
			mtulen = len(buf)
		}
	}

	fmt.Println("Interfaces\n----------")
	for i := range ifaces {
		fmt.Printf("%*d %*d %-*s %*v %v\n",
			idxlen,
			ifaces[i].Index,
			mtulen,
			ifaces[i].MTU,
			iflen,
			ifaces[i].Name,
			mclen,
			ifaces[i].HardwareAddr,
			ifaces[i].Flags)
	}

	ifaddrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
		return
	}

	addrlen := 0
	for _, a := range ifaddrs {
		if addrlen < len(a.String()) {
			addrlen = len(a.String())
		}
	}

	fmt.Println("\nAddresses\n---------")
	for _, a := range ifaddrs {
		ip, _, _ := net.ParseCIDR(a.String())
		fmt.Printf("%-*s", addrlen, a.String())
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			fmt.Printf("\tlocal")
		}
		fmt.Printf("\n")
	}

	fmt.Println("\nAddress by Interface\n--------------------")
	for i := range ifaces {
		ifname := ifaces[i].Name
		fmt.Printf("%-*s %*s ", iflen, ifname, mclen, ifaces[i].HardwareAddr)
		addrs, err := ifaces[i].Addrs()
		if err != nil {
			fmt.Println(err)
			return
		}
		first := true
		for j := range addrs {
			if !first {
				fmt.Printf("\n%*s", iflen+mclen+2, "")
			}
			fmt.Printf("%s", addrs[j].String())
			first = false
		}
		fmt.Println()
	}

	fmt.Println("\nMulticast Addresses by Interface")
	fmt.Println("--------------------------------")
	for i := range ifaces {
		fmt.Printf("%-*s ", iflen, ifaces[i].Name)
		addrs, err := ifaces[i].MulticastAddrs()
		if err != nil {
			fmt.Println(err)
			return
		}
		first := true
		for j := range addrs {
			if !first {
				fmt.Printf("\n%*s", iflen+1, "")
			}
			fmt.Printf("%s", addrs[j].String())
			first = false
		}
		fmt.Println()
	}

	fmt.Println()
	fmt.Println("Routes")
	fmt.Println("------")

	routes, err := ReadRoutes()
	if err != nil {
		log.Fatal(err)
	}

	rtitles := []string{
		"Dest",
		"Gateway",
		"Interface",
		"Priority",
		"Table",
	}

	rval := make([][]string, len(routes))
	rlen := make([]int, len(rtitles))

	for i, r := range routes {
		rval[i] = []string{
			r.Dest,
			r.Gateway,
			ifaces[r.OutputInterface-1].Name,
			strconv.Itoa(r.Priority),
			r.Table,
		}
		for f := range rval[i] {
			if rlen[f] < len(rval[i][f]) {
				rlen[f] = len(rval[i][f])
			}
		}
	}

	for f, title := range rtitles {
		if rlen[f] < len(title) {
			rlen[f] = len(title)
		}
	}

	fmt.Println()
	for f, title := range rtitles {
		fmt.Printf("%-*s ", rlen[f], title)
	}
	fmt.Println()

	for _, line := range rval {
		for f, val := range line {
			fmt.Printf("%-*s ", rlen[f], val)
		}
		fmt.Println()
	}

	rtnum, err := DefaultRoute(routes)
	if err == nil {
		fmt.Println()
		fmt.Println("Default Interface")
		fmt.Println("-----------------")
		fmt.Println(ifaces[routes[rtnum].OutputInterface-1].Name)
	}
}
