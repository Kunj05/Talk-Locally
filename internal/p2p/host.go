package p2p

import (
	"fmt"
	"log"

	"github.com/fatih/color"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
)

// CreateHost initializes a libp2p host with a given port string.
// Example: "9000" → /ip4/127.0.0.1/tcp/9000
func CreateHost(PORT_HOST string) (host.Host, string, error) {

	PORT := "/ip4/0.0.0.0/tcp/" + PORT_HOST

	// Create host node (This host will automatically generate a
	// unique Peer ID (like your fingerprint in the P2P network).)
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(
			PORT,
		),
	)
	if err != nil {
		log.Fatalf("Failed to create host: %v", err)
		return nil, "", err
	}

	red := color.New(color.FgRed).SprintFunc()

	// Create the connection string to share with others
	CONNECTION_STRING := PORT + "/p2p/" + h.ID().String()
	fmt.Println()
	fmt.Println()
	log.Println(red("Your Details:"))
	log.Println(red("Hello,my Peer ID is: ", h.ID()))
	log.Println(red("Listening on: ", PORT))
	log.Println(red("My host Address: ", h.Addrs()))

	return h, CONNECTION_STRING, nil
}
