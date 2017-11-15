package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/cmatsuoka/dhcpcheck/dhcp"
)

var Version = "0.1"

func cmdDiscover() {
	var iface string
	var secs int
	var sendOnly bool

	flag.StringVar(&iface, "i", "", "network `interface` to use")
	flag.IntVar(&secs, "t", 5, "timeout in seconds")
	flag.BoolVar(&sendOnly, "s", false, "send discovery only and ignore offers")
	flag.Parse()

	if iface == "" {
		usage(os.Args[1])
		os.Exit(1)
	}

	timeout := time.Duration(secs) * time.Second
	if sendOnly {
		timeout = 0
	}

	setupSummary()

	discover(iface, timeout, false)
}

func discover(iface string, timeout time.Duration, silent bool) {

	mac, err := MACFromIface(iface)
	checkError(err)

	fmt.Printf("Interface: %s [%s]\n", iface, mac)

	var client *dhcp.Client

	if timeout <= 0 {
		client, err = dhcp.NewClientNotListening()
		checkError(err)
	} else {
		client, err = dhcp.NewClient()
		checkError(err)
		defer client.Close()
	}

	// Send discover packet
	p := dhcp.NewDiscoverPacket()
	p.SetClientMAC(mac)
	class := "dhcpcheck-" + Version
	p.AddOptions(append(
		[]byte{dhcp.VendorClassIdentifier, byte(len(class))},
		[]byte(class)...))

	fmt.Println("\n>>> Send DHCP discover")

	if !silent {
		showPacket(p, "")
	}
	err = client.Broadcast(p)
	checkError(err)

	stats.pksent++
	stats.count[mac]++

	if timeout <= 0 {
		return
	}

	t := time.Now()
	for time.Since(t) < timeout {
		o, remote, err := client.Receive(timeout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err.Error())
			break
		}

		stats.pkrec++

		cmac := o.Chaddr.MACAddress().String()

		if mac != cmac || o.Xid != p.Xid {
			continue
		}

		rip := remote.IP.String()
		rmac := MACFromIP(rip)

		stats.pkproc++
		stats.count[rmac]++

		fmt.Printf("\n<<< Receive DHCP offer from %s (%s)\n",
			rip, NameFromIP(rip))
		fmt.Printf("    MAC address: %s (%s)\n",
			rmac, VendorFromMAC(rmac))

		showPacket(&o, rip)
	}
	fmt.Println("No more offers.")
}
