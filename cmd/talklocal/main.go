//docker run -it --rm --name talklocal-http-frontend --network talklocal -p 9001:9001 -p 3001:3001 talklocal-http-frontend
//docker build -f Dockerfile.http-frontend -t talklocal-http-frontend .

// docker run -it --rm --name clinode1 --network talklocal talklocally-clinode1 --port=ocally-clinode1 --port=9001 --nick=kunj --same_string=demo
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"talkLocally/internal/p2p"

	"github.com/fatih/color"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

type IncomingMsg struct {
	Message string `json:"message"`
}

var MessageArr []string
var messageMu sync.Mutex

var cr *p2p.ChatRoom

var discoveredRooms = make(map[string]bool)
var discoveredRoomsMu sync.Mutex

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Credentials", "true") // Allow credentials if needed
}

func StoreMessage(msg string) {
	messageMu.Lock()
	defer messageMu.Unlock()
	MessageArr = append(MessageArr, msg)
}

func GetMessages(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method == http.MethodOptions {
		return // Handle CORS preflight requests
	}

	if r.Method != "GET" {
		http.Error(w, "Only GET method supported", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(MessageArr)
	if err != nil {
		http.Error(w, "failed to encode messages", http.StatusInternalServerError)
		return
	}
}

func PostMessage(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method == http.MethodOptions {
		return // Handle CORS preflight requests
	}

	if r.Method != "POST" {
		http.Error(w, "Only POST method supported", http.StatusBadRequest)
		return
	}

	var msg_post IncomingMsg
	err := json.NewDecoder(r.Body).Decode(&msg_post)
	if err != nil || msg_post.Message == "" {
		http.Error(w, "failed to decode", http.StatusBadRequest)
		return
	}

	// Publish the message using the pubsub
	err_pub := cr.Publish(msg_post.Message)
	if err_pub != nil {
		fmt.Println("Sending message failed, trying again...")
		http.Error(w, "failed to publish", http.StatusInternalServerError)
		return
	}

	// Store the message in the backend array
	StoreMessage(msg_post.Message)

	// Respond back with success
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Message sent successfully")
}

func GetAvailableRooms(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)

	if r.Method == http.MethodOptions {
		return // Handle CORS preflight requests
	}

	if r.Method != "GET" {
		http.Error(w, "Only GET method supported", http.StatusBadRequest)
		return
	}

	discoveredRoomsMu.Lock()
	defer discoveredRoomsMu.Unlock()

	var rooms []string
	for room := range discoveredRooms {
		rooms = append(rooms, room)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rooms); err != nil {
		http.Error(w, "Failed to encode rooms", http.StatusInternalServerError)
	}
}

func main() {
	port := flag.String("port", "", "port")
	nickFlag := flag.String("nick", "", "nickname to use in chat. will be generated if empty")
	sameNetworkString := flag.String("same_string", "", "same_string to join same p2p network")
	httpPort := flag.String("http-port", "", "HTTP server port")

	flag.Parse()

	if *sameNetworkString == "" {
		log.Fatal("Please provide --same_string flag to join the correct p2p network")
	}

	h, _, err1 := p2p.CreateHost(*port)
	if err1 != nil {
		log.Fatal("error creating the host")
	}

	ctx := context.Background()

	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	peerChan := p2p.InitMDNS(h, *sameNetworkString)

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()

	go func() {
		for {
			peer := <-peerChan
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

			log.Println(green("Connection to the peer found through MDNS has been established"))
			log.Println(green("Peer Id:", peer.ID, "Peer Addrs: ", peer.Addrs))
		}
	}()

	nick := *nickFlag
	if len(nick) == 0 {
		nick = "KUNJ"
	}

	// Join discovery room for sharing room names
	discoveryRoom, err := p2p.JoinDiscoveryRoom(ctx, ps, h.ID(), nick)
	if err != nil {
		log.Fatal("failed to join room discovery topic")
	}

	// Automatically join a default room
	defaultRoomName := "default-room"
	cr, err = p2p.JoinChatRoom(ctx, ps, h.ID(), nick, defaultRoomName)
	if err != nil {
		log.Fatal("Failed to join default room:", err)
	}
	fmt.Println("Joined room:", defaultRoomName)

	// Announce our current room name periodically (every 5 seconds)
	go func() {
		for {
			if cr != nil {
				_ = discoveryRoom.Publish(cr.RoomName())
			}
			time.Sleep(5 * time.Second)
		}
	}()

	// Listen for rooms announced by other peers
	go func() {
		for msg := range discoveryRoom.Messages {
			discoveredRoomsMu.Lock()
			discoveredRooms[msg.Message] = true
			discoveredRoomsMu.Unlock()
		}
	}()

	http.Handle("/", http.FileServer(http.Dir("./frontend/dist")))

	log.Println("Serving frontend on :3001")

	// Start HTTP server
	http.HandleFunc("/send", PostMessage)
	http.HandleFunc("/messages", GetMessages)
	http.HandleFunc("/available-rooms", GetAvailableRooms)

	fmt.Println("HTTP server started on :" + *httpPort)
	err = http.ListenAndServe(":"+*httpPort, nil)
	if err != nil {
		log.Fatal(err)
	}
}
