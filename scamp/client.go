package scamp

import (
	"sync"
)

// type ClientChan chan *Client

// Client represents a scamp client
type Client struct {
	conn           *Connection
	service        *Service
	requests       chan *Message
	openRepliesMut sync.Mutex
	openReplies    map[int]chan *Message
	isClosed       bool
	closedMut      sync.Mutex
	sendMut        sync.Mutex
	nextRequestID  int
	spIdent        string
}

// Dial calls DialConnection to establish a secure (tls) connection,
// and uses that connection to create a client
func Dial(connspec string) (client *Client, err error) {
	conn, err := DialConnection(connspec)
	if err != nil {
		return
	}
	client = NewClient(conn)
	return
}

// NewClient takes a scamp connection and creates a new scamp client
func NewClient(conn *Connection) (client *Client) {
	client = &Client{
		conn:        conn,
		requests:    make(chan *Message), //TODO: investigate using buffered channel here
		openReplies: make(map[int]chan *Message),
	}
	// client.conn = conn
	// client.requests = make(chan *Message)
	// client.openReplies = make(map[int]chan *Message)
	conn.SetClient(client)

	go client.splitReqsAndReps()

	return
}

// SetService assigns a *Service to client.serv
func (client *Client) SetService(s *Service) {
	client.service = s
}

// Send TODO: would be nice to have different code path for scamp responses
// so that we don't need to rely on garbage collection of channels
// when we're replying and don't expect or need a response
func (client *Client) Send(msg *Message) (responseChan chan *Message, err error) {
	client.sendMut.Lock()
	defer client.sendMut.Unlock()

	client.nextRequestID++
	msg.RequestID = client.nextRequestID
	err = client.conn.Send(msg)
	if err != nil {
		// Trace.Printf("SCAMP send error: %s", err)
		return
	}

	if msg.MessageType == MessageTypeRequest {
		// Trace.Printf("sending request so waiting for reply")
		responseChan = make(chan *Message)
		client.openRepliesMut.Lock()
		client.openReplies[msg.RequestID] = responseChan
		client.openRepliesMut.Unlock()
	} else {
		// Trace.Printf("sending reply so done with this message")
	}

	return
}

// Close unlocks a client mutex and closes the connection
func (client *Client) Close() {
	if len(client.spIdent) > 0 {
		sp := DefaultCache.Retrieve(client.spIdent)
		if sp != nil {
			sp.client = nil
		}
	}

	client.closedMut.Lock()
	defer client.closedMut.Unlock()

	if client.isClosed {
		return
	}
	client.closeConnection(client.conn)

	// Notify wrapper service that we're dead
	if client.service != nil {
		client.service.RemoveClient(client)
	}

	client.isClosed = true // write race
}

// closeConnection calls client.conn.Close() and sets the client.conn to nil
func (client *Client) closeConnection(conn *Connection) {
	if !client.conn.isClosed {
		client.conn.Close()
	}
	client.conn = nil // race
}

//func (client *Client) splitReqsAndReps(grNum, clientID int) (err error) {
func (client *Client) splitReqsAndReps() (err error) {
	var replyChan chan *Message

forLoop:
	for {
		// Trace.Printf("Entering forLoop splitReqsAndReps")
		select {
		case message, ok := <-client.conn.msgs: //race
			if !ok {
				// Trace.Printf("client.conn.msgs... CLOSED!")
				break forLoop
			}
			if message == nil {
				continue
			}

			// Trace.Printf("Splitting incoming message to reqs and reps")

			if message.MessageType == MessageTypeRequest {
				// interesting things happen if there are outstanding messages
				// and the client closes
				client.requests <- message
			} else if message.MessageType == MessageTypeReply {
				client.openRepliesMut.Lock()
				replyChan = client.openReplies[message.RequestID]
				if replyChan == nil {
					// Error.Printf("got an unexpected reply for requestId: %d. Skipping.", message.RequestID)
					client.openRepliesMut.Unlock()
					continue
				}

				delete(client.openReplies, message.RequestID)
				client.openRepliesMut.Unlock()

				replyChan <- message
			} else {
				// Trace.Printf("Could not handle msg, it's neither req or reply. Skipping.")
				continue
			}
		}
	}

	// Trace.Printf("done with SplitReqsAndReps")
	close(client.requests)
	client.openRepliesMut.Lock()
	for _, openReplyChan := range client.openReplies {
		close(openReplyChan)
	}
	defer client.openRepliesMut.Unlock()
	if !client.isClosed { //read (race)
		client.Close()
	}

	return
}

// Incoming returns a client's MessageChan
func (client *Client) Incoming() chan *Message {
	return client.requests
}
