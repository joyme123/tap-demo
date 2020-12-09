package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/songgao/packets/ethernet"
	"github.com/songgao/water"
	"github.com/vishvananda/netlink"
)

func main() {
	laddr := flag.String("laddr", "", "local addr")
	lport := flag.Int("lport", 0, "local port")
	raddr := flag.String("raddr", "", "remote addr")
	rport := flag.Int("rport", 0, "remote port")
	tapAddr := flag.String("addr", "", "tap addr")
	flag.Parse()

	c := water.Config{
		DeviceType: water.TAP,
	}
	tap, err := water.New(c)
	if err != nil {
		log.Fatalf("new tap device error:%v", err)
	}
	link, err := netlink.LinkByName(tap.Name())
	if err != nil {
		log.Fatalf("get tap device error: %v", err)
	}

	addr, err := netlink.ParseAddr(*tapAddr)
	if err != nil {
		log.Fatalf("parse tap addr error: %v", err)
	}

	err = netlink.AddrAdd(link, addr)
	if err != nil {
		log.Fatalf("add addr error: %v", err)
	}

	err = netlink.LinkSetUp(link)
	if err != nil {
		log.Fatalf("link set up error")
	}

	rAddr := &net.UDPAddr{IP: net.ParseIP(*raddr), Port: *rport}
	remoteConn, err := net.DialUDP("udp", nil, rAddr)
	if err != nil {
		log.Fatalf("conn udp error: %v", err)
	}

	// read from tap and send to remote server
	go func() {
		var frame ethernet.Frame
		for {
			frame.Resize(1500)
			rn, err := tap.Read(frame)
			frame = frame[:rn]
			log.Println("===============read from tap ================")
			log.Printf("Dst: %s\n", frame.Destination())
			log.Printf("Src: %s\n", frame.Source())
			log.Printf("Ethertype: % x\n", frame.Ethertype())
			log.Printf("Payload: % x\n", frame.Payload())

			if err != nil {
				log.Fatalf("read from tap device error: %v", err)
			}

			_, err = remoteConn.Write(frame)
			if err != nil {
				log.Printf("udp connect write error: %v\n", err)
				time.Sleep(10 * time.Second)
			}
		}
	}()

	//==========================================================
	// read from remote and send to tap
	// create udp
	lAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", *laddr, *lport))
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.ListenUDP("udp", lAddr)
	if err != nil {
		log.Fatalf("listen udp error: %v", err)
	}
	log.Printf("udp listen at: %v", lAddr.String())
	var frame ethernet.Frame
	for {
		frame.Resize(1500)
		n, err := listener.Read(frame)
		if err != nil {
			log.Fatalf("read from udp conn error: %v", err)
		}
		frame = frame[:n]
		log.Println("===============read from udp================")
		log.Printf("Dst: %s\n", frame.Destination())
		log.Printf("Src: %s\n", frame.Source())
		log.Printf("Ethertype: % x\n", frame.Ethertype())
		log.Printf("Payload: % x\n", frame.Payload())
		n, err = tap.Write(frame)
		if err != nil {
			log.Fatalf("write to tap device error: %v", err)
		}
	}
}
