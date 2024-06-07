package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	PacketTypeEnter = 1
	PacketTypeDrop  = 2
	PacketTypePass  = 3
)

func listInterfaces() {
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("Error listing interfaces: %v\n", err)
		return
	}
	fmt.Println("Available network interfaces:")
	for _, iface := range interfaces {
		fmt.Printf("  - %s\n", iface.Name)
	}
}

func main() {
	// Define command-line flags
	interfaceName := flag.String("interface", "lo", "Network interface to attach the eBPF program")
	portArg := flag.String("port", "4040", "Allowed port number")
	help := flag.Bool("help", false, "Display help information")
	listIfaces := flag.Bool("list-interfaces", false, "List available network interfaces")

	// Parse the flags
	flag.Parse()

	// Display help information if requested
	if *help {
		fmt.Println("Usage: packet-dropper [--interface <interface>] [--port <port>] [--ebpf-file <file>] [--help] [--list-interfaces]")
		fmt.Println("  --interface: Network interface to attach the eBPF program (default: eth0)")
		fmt.Println("  --port: Allowed port number (default: 4040)")
		fmt.Println("  --help: Display this help message")
		fmt.Println("  --list-interfaces: List available network interfaces")
		return
	}

	// List interfaces if requested
	if *listIfaces {
		listInterfaces()
		return
	}

	// Lift memory lock restriction
	if err := rlimit.RemoveMemlock(); err != nil {
		fmt.Printf("Unable to lift MEMLOCK: %v\n", err)
		return
	}

	// Convert the allowed port to an integer
	allowedPort, err := strconv.Atoi(*portArg)
	if err != nil {
		fmt.Printf("Invalid port number: %v\n", err)
		return
	}

	// Load eBPF program from specified object file
	ebpfSpec, err := ebpf.LoadCollectionSpec("/etc/packet-dropper/packet_drop_kern.o")
	if err != nil {
		fmt.Printf("Failed to load eBPF program: %v\n", err)
		return
	}

	// Create eBPF collection
	ebpfColl, err := ebpf.NewCollection(ebpfSpec)
	if err != nil {
		fmt.Printf("Failed to create eBPF collection: %v\n", err)
		return
	}
	defer ebpfColl.Close()

	// Fetch eBPF program from collection
	prog := ebpfColl.Programs["xdp_packet_filter"]
	if prog == nil {
		fmt.Println("Program 'xdp_packet_filter' not found")
		return
	}

	// Get network interface
	iface, err := net.InterfaceByName(*interfaceName)
	if err != nil {
		fmt.Printf("Failed to get interface '%s': %v\n", *interfaceName, err)
		listInterfaces() // Suggest available interfaces
		return
	}

	// Attach eBPF program to the network interface
	linkOpts := link.XDPOptions{
		Program:   prog,
		Interface: iface.Index,
	}
	linkRef, err := link.AttachXDP(linkOpts)
	if err != nil {
		fmt.Printf("Failed to attach eBPF program: %v\n", err)
		return
	}
	defer linkRef.Close()

	// Update allowed port map
	portMap := ebpfColl.Maps["allowed_ports"]
	if portMap == nil {
		fmt.Println("Map 'allowed_ports' not found")
		return
	}

	var key uint32 = 0
	var port uint16 = uint16(allowedPort)
	if err := portMap.Update(key, port, ebpf.UpdateAny); err != nil {
		fmt.Printf("Failed to update port map: %v\n", err)
		return
	}

	fmt.Println("BPF program loaded and attached. Press Ctrl+C to exit.")

	// Wait for termination signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Display drop statistics periodically
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now().Format("15:04:05")
			dropStats := getDropStatistics(ebpfColl, key)

			// Calculate padding for the right alignment
			padding := ""
			terminalWidth, _, err := terminal.GetSize(int(os.Stdout.Fd()))
			if err == nil {
				paddingWidth := terminalWidth - len(now) - len(dropStats) - 4
				if paddingWidth > 0 {
					padding = fmt.Sprintf("%*s", paddingWidth, "")
				}
			}

			fmt.Printf("%s%s%s\n", dropStats, padding, now)
		case <-sigChan:
			fmt.Println("\nDetaching BPF program and exiting.")
			return
		}
	}
}

func getDropStatistics(ebpfColl *ebpf.Collection, key uint32) string {
	var dropCounts []uint16
	if err := ebpfColl.Maps["drop_counter"].Lookup(key, &dropCounts); err != nil {
		log.Errorf("Error looking up drop counter: %v", err)
		return ""
	}

	totalDropped := uint16(0)
	for _, count := range dropCounts {
		totalDropped += count
	}

	return fmt.Sprintf("Total packets dropped: %d", totalDropped)
}
