// context - used for carrying deadlines, cancellation signals, and other request-scoped values across API boundaries and between processes
// flag - These flags provide a way to configure and control the behavior of a Go program when it is executed from the command line

// Each user runs the same main.go program locally on their machine.
// They provide the same --same_string flag value when starting the program. This acts like a "network ID" or "chat group identifier" so peers can discover each other.
// Optionally, users can specify their own nickname with --nick and port with --port.
// The program uses mDNS and libp2p under the hood to discover other peers running with the same --same_string and connect to them.
// Once connected, users can join or create chat rooms, and chat directly from the terminal (CLI).

// command -  docker network create talklocal
// command -  docker run -it --rm --network talklocal -p 9001:9001 talklocal -port=9001 -nick=kunj -same_string=demo
// command -  docker run -it --rm --network talklocal -p 9002:9001 talklocal -port=9001 -nick=nik -same_string=demo
// command -  docker run -it --rm --network talklocal -p 9003:9001 talklocal -port=9001 -nick=virat -same_string=demo

// go run main.go --port=9001 --nick=kunj --same_string=demo
// go run main.go --port=9002 --nick=nik  --same_string=demo
// go run main.go --port=9003 --nick=vinay --same_string=demo

//go run main.go --port=9001 --nick=kunj --same_string=demo --enable-http --http-port=3001

package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	fileshare "talkLocally/internal/fileshare"
	"talkLocally/internal/p2p"
	"time"

	"github.com/fatih/color"
	"github.com/ipfs/go-cid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var cr *p2p.ChatRoom
var discoveredRooms = make(map[string]bool)
var discoveredRoomsMu sync.Mutex

func main() {
	port := flag.String("port", "", "port")
	nickFlag := flag.String("nick", "", "nickname to use in chat. will be generated if empty")
	sameNetworkString := flag.String("same_string", "", "same_string for mDNS discovery")
	flag.Parse()

	// Create libp2p host
	h, _, err := p2p.CreateHost(*port)
	if err != nil {
		log.Fatal("Error creating the host:", err)
	}
	ctx := context.Background()

	// Setup PubSub
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Fatal("Error creating pubsub:", err)
	}

	// mDNS peer discovery
	peerChan := p2p.InitMDNS(h, *sameNetworkString)
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	go func() {
		for peer := range peerChan {
			fmt.Println()
			fmt.Println(green("New Peer Found:"))
			if peer.ID > h.ID() {
				fmt.Println(green("Found peer:", peer, " id is greater than us, wait for it to connect to us"))
				continue
			}
			fmt.Println(yellow("Discovered new peer via mDNS:", peer.ID, peer.Addrs))

			if err := h.Connect(ctx, peer); err != nil {
				fmt.Println("Connection failed:", err)
				continue
			}
			log.Println(green("Connected to peer via mDNS:", peer.ID))
		}
	}()

	// Nickname setup
	nick := *nickFlag
	if len(nick) == 0 {
		nick = "KUNJ"
	}

	// Join room discovery topic
	discoveryRoom, err := p2p.JoinDiscoveryRoom(ctx, ps, h.ID(), nick)
	if err != nil {
		log.Fatal("Failed to join room discovery topic:", err)
	}

	// Broadcast our room name periodically
	go func() {
		for {
			if cr != nil {
				_ = discoveryRoom.Publish(cr.RoomName())
			}
			time.Sleep(5 * time.Second)
		}
	}()

	// Receive room names from others
	go func() {
		for msg := range discoveryRoom.Messages {
			discoveredRoomsMu.Lock()
			discoveredRooms[msg.Message] = true
			discoveredRoomsMu.Unlock()
		}
	}()

	// CLI Interaction
	reader := bufio.NewReader(os.Stdin)
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Println("\nChoose an option:")
		fmt.Println("1. List available rooms")
		fmt.Println("2. Join/Create a room")
		fmt.Println("3. Exit")
		fmt.Print("Enter choice: ")

		choiceLine, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("Failed to read input:", err)
		}
		choiceLine = strings.TrimSpace(choiceLine)

		switch choiceLine {
		case "1":
			fmt.Println("Discovered Rooms:")
			discoveredRoomsMu.Lock()
			for room := range discoveredRooms {
				fmt.Println("-", room)
			}
			discoveredRoomsMu.Unlock()

		case "2":
			fmt.Print("Enter room name to join or create: ")
			roomName, err := reader.ReadString('\n')
			if err != nil {
				log.Fatal("Failed to read room name:", err)
			}
			roomName = strings.TrimSpace(roomName)

			cr, err = p2p.JoinChatRoom(ctx, ps, h.ID(), nick, roomName)
			if err != nil {
				fmt.Println("Failed to join room:", err)
				continue
			}
			fmt.Println("Joined room:", roomName)
			fmt.Print("> Enter message (or /exit to leave): ")

			// Logging messages
			f, err := os.OpenFile("logs.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatal("Error opening logs.txt:", err)
			}

			// Listen to messages
			go func() {
				blue := color.New(color.FgBlue).SprintFunc()
				for msg := range cr.Messages {
					text := fmt.Sprintf("Received message at %s from %s: %s\n", time.Now().Local(), msg.SenderNick, msg.Message)
					fmt.Print(blue(text))
					fmt.Print("> Enter message (or /exit to leave): ")
					_, err := f.WriteString(text)
					if err != nil {
						log.Fatal("Error writing to logs.txt:", err)
					}
				}
			}()

			// Send messages
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					log.Fatal("Error reading input:", err)
				}
				line = strings.TrimSpace(line)

				if line == "/exit" {
					fmt.Println("Leaving the room...")
					break
				}
				// Handle file send command
				if line == "/send-file" {
					fmt.Print("Enter the file path to upload: ")
					scanner.Scan()
					filePath := strings.TrimSpace(scanner.Text())

					cid, err := fileshare.AddFileToOfflineStore(filePath)
					if err != nil {
						log.Printf("Error adding file to store: %v", err)
						continue
					}

					fmt.Println("File uploaded successfully! CID:", cid.String())
				}

				// Handle file retrieve command
				if line == "/get-file" {
					fmt.Print("Enter the CID of the file to retrieve: ")
					scanner.Scan()
					cidStr := strings.TrimSpace(scanner.Text())

					fileCid, err := cid.Decode(cidStr)
					if err != nil {
						log.Printf("Error decoding CID '%s': %v", cidStr, err)
						continue
					}

					fileData, err := fileshare.RetrieveFileFromStore(fileCid)
					if err != nil {
						log.Printf("Error retrieving file with CID '%s': %v", cidStr, err)
						continue
					}

					fmt.Println("Retrieved file data:", string(fileData))
				}

				if line == "" {
					continue
				}
				if err := cr.Publish(line); err != nil {
					fmt.Println("Sending message failed, retrying...")
					_ = cr.Publish(line)
				}
			}

		case "3":
			fmt.Println("Exiting...")
			os.Exit(0)

		default:
			fmt.Println("Invalid choice. Please enter 1, 2, or 3.")
		}
	}
}
