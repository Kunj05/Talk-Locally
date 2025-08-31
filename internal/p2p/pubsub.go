package p2p

import (
	"context"
	"encoding/json"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// ChatRoom represents a subscription to a single PubSub topic. Messages
// can be published to the topic with ChatRoom.Publish, and received
// messages are pushed to the Messages channel.
type ChatRoom struct {
	// Messages is a channel of messages received from other peers in the chat room
	Messages chan *ChatMessage

	ctx    context.Context
	cancel context.CancelFunc
	ps     *pubsub.PubSub
	topic  *pubsub.Topic
	sub    *pubsub.Subscription

	roomName    string
	self        peer.ID
	nick        string
	peersInRoom map[peer.ID]bool
}

// ChatMessage gets converted to/from JSON and sent in the body of pubsub messages.
type ChatMessage struct {
	Message    string
	SenderID   string
	SenderNick string
}

// ChatRoomBufSize is the number of incoming messages to buffer for each topic.
const ChatRoomBufSize = 128

func (cr *ChatRoom) ActivePeers() []peer.ID {
	peers := make([]peer.ID, 0, len(cr.peersInRoom))
	for pid := range cr.peersInRoom {
		peers = append(peers, pid)
	}
	return peers
}

func (cr *ChatRoom) RemoveSelf() {
	delete(cr.peersInRoom, cr.self)
	cr.Close()
}

func (cr *ChatRoom) RoomName() string {
	return cr.roomName
}

func (cr *ChatRoom) GetNickName() string {
	return cr.nick
}

func (cr *ChatRoom) Close() {
	cr.sub.Cancel()    // ✅ unsubscribe from the topic
	close(cr.Messages) // ✅ close the message channel
	cr.cancel()        // ✅ stop the readLoop
}

func topicName(roomName string) string {
	return "chat-room:" + roomName
}

// JoinDiscoveryRoom lets you join the special room used for announcing available chat rooms
func JoinDiscoveryRoom(ctx context.Context, ps *pubsub.PubSub, selfID peer.ID, nickname string) (*ChatRoom, error) {
	return JoinChatRoom(ctx, ps, selfID, nickname, "room-discovery")
}

// JoinChatRoom tries to subscribe to the PubSub topic for the room name, returning
// a ChatRoom on success.
var joinedTopics = make(map[string]*pubsub.Topic) // track joined topics per process

func JoinChatRoom(ctx context.Context, ps *pubsub.PubSub, selfID peer.ID, nickname string, roomName string) (*ChatRoom, error) {
	tn := topicName(roomName)

	var topic *pubsub.Topic
	var err error

	// check if we already joined this topic
	if t, ok := joinedTopics[tn]; ok {
		topic = t
	} else {
		topic, err = ps.Join(tn)
		if err != nil {
			return nil, err
		}
		joinedTopics[tn] = topic
	}

	// create a new subscription for this join
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)

	cr := &ChatRoom{
		ctx:         ctx,
		cancel:      cancel,
		ps:          ps,
		topic:       topic,
		sub:         sub,
		self:        selfID,
		nick:        nickname,
		roomName:    roomName,
		Messages:    make(chan *ChatMessage, ChatRoomBufSize),
		peersInRoom: make(map[peer.ID]bool),
	}

	cr.peersInRoom[selfID] = true

	go cr.readLoop()
	return cr, nil
}
func (cr *ChatRoom) Publish(message string) error {
	m := ChatMessage{
		Message:    message,
		SenderID:   cr.self.String(),
		SenderNick: cr.nick,
	}
	msgBytes, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return cr.topic.Publish(cr.ctx, msgBytes)
}

func (cr *ChatRoom) readLoop() {
	for {
		msg, err := cr.sub.Next(cr.ctx)
		if err != nil {
			return
		}
		// only forward messages delivered by others
		if msg.ReceivedFrom == cr.self {
			continue
		}
		cr.peersInRoom[msg.ReceivedFrom] = true

		cm := new(ChatMessage)
		err = json.Unmarshal(msg.Data, cm)
		if err != nil {
			continue
		}
		// send valid messages onto the Messages channel
		select {
		case cr.Messages <- cm:
		case <-cr.ctx.Done():
			return
		}
	}
}
