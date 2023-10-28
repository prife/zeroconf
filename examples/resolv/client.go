package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

var (
	// _remoted _remotepairing
	service  = flag.String("service", "_remoted._tcp", "Set the service category to look for devices.")
	domain   = flag.String("domain", "local", "Set the search domain. For local networks, default is fine.")
	waitTime = flag.Int("wait", 10, "Duration in [s] to run discovery.")
)

func listMulticastInterfaces() []net.Interface {
	var interfaces []net.Interface
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, ifi := range ifaces {
		fmt.Println("face:", ifi)
		if runtime.GOOS == "darwin" {
			if strings.HasPrefix(ifi.Name, "utun") {
				continue
			}
		}
		if (ifi.Flags & net.FlagUp) == 0 {
			continue
		}
		if (ifi.Flags & net.FlagMulticast) > 0 {
			interfaces = append(interfaces, ifi)
		}
	}

	return interfaces
}

func browse() {
	ifaces := listMulticastInterfaces()
	// ctx := context.Background()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	for _, iface := range ifaces {
		fmt.Println("---> iface:", iface)
		resolver, err := zeroconf.NewResolver(zeroconf.SelectIPTraffic(zeroconf.IPv6), zeroconf.SelectIfaces([]net.Interface{iface}))
		if err != nil {
			continue
		}
		entries := make(chan *zeroconf.ServiceEntry)
		go func(iface net.Interface, results <-chan *zeroconf.ServiceEntry) {
			addrs, _ := iface.Addrs()
			name := fmt.Sprintf("%#v:%v", iface, addrs)
			for entry := range results {
				log.Printf("[%s] %#v\n", name, entry)
				log.Printf("--> addr: %s:%%%d", entry.AddrIPv6[0].String(), iface.Index)
			}
		}(iface, entries)

		err = resolver.Browse(ctx, *service, *domain, entries)
		if err != nil {
			log.Fatalln("Failed to browse:", err.Error())
		}
	}
	<-ctx.Done()
	// Wait some additional time to see debug messages on go routine shutdown.
	time.Sleep(1 * time.Second)
}

func main() {
	flag.Parse()

	if true {
		browse()
		return
	}

	// Discover all services on the network (e.g. _workstation._tcp)
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		log.Fatalln("Failed to initialize resolver:", err.Error())
	}

	entries := make(chan *zeroconf.ServiceEntry)
	go func(results <-chan *zeroconf.ServiceEntry) {
		for entry := range results {
			log.Printf("%#v\n", entry)
		}
		log.Println("No more entries.")
	}(entries)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(*waitTime))
	defer cancel()
	err = resolver.Browse(ctx, *service, *domain, entries)
	if err != nil {
		log.Fatalln("Failed to browse:", err.Error())
	}

	<-ctx.Done()
	// Wait some additional time to see debug messages on go routine shutdown.
	time.Sleep(1 * time.Second)

}
