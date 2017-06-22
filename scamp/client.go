package scamp

import (
	"sync"
)

type ClientChan chan *Client

type Client struct {
	conn *Connection
	serv *Service

	requests        MessageChan
	openReplies     map[int]MessageChan
	openRepliesLock sync.Mutex

	isClosed bool
	closedM  sync.Mutex

	sendM sync.Mutex
	nextRequestId int
}

func Dial(connspec string) (client *Client, err error) {
	Info.Printf("connecting to `%s`", connspec)

	conn, err := DialConnection(connspec)
	if err != nil {
		return
	}
	client = NewClient(conn)

	return
}

func NewClient(conn *Connection) (client *Client) {
	Trace.Printf("client allocated")
	client = new(Client)

	client.conn = conn
	client.requests = make(MessageChan)
	client.openReplies = make(map[int]MessageChan)

	conn.SetClient(client)

	go client.splitReqsAndReps()

	return
}

func (client *Client) SetService(serv *Service) {
	client.serv = serv
}

// TODO: would be nice to have different code path for scamp responses
// so that we don't need to rely on garbage collection of channels
// when we're replying and don't expect or need a response
func (client *Client) Send(msg *Message) (responseChan MessageChan, err error) {
	client.sendM.Lock()
	defer client.sendM.Unlock()

	// Info.Printf("sending message `%d`", msg.RequestId)
	client.nextRequestId++
	msg.RequestId = client.nextRequestId
	err = client.conn.Send(msg)
	if err != nil {
		Info.Printf("SCAMP send error: %s", err)
		return
	}

	if msg.MessageType == MESSAGE_TYPE_REQUEST {
		// Info.Printf("sending request so waiting for reply")
		responseChan = make(MessageChan)
		client.openRepliesLock.Lock()
		client.openReplies[msg.RequestId] = responseChan
		client.openRepliesLock.Unlock()
	} else {
		// Info.Printf("sending reply so done with this message")
	}

	return
}

// Close unlocks a client mutex and closes the connection
func (client *Client) Close() {
	client.closedM.Lock()
	defer client.closedM.Unlock()
	if client.isClosed {
		Trace.Printf("client already closed. skipping shutdown.")
		return
	}

	Trace.Printf("closing client...")
	Trace.Printf("closing client conn...")
	client.conn.Close()

	// // Notify wrapper service that we're dead
	if client.serv != nil {
		Trace.Printf("removing client from service...")
		client.serv.RemoveClient(client)
	}

	Trace.Printf("marking client as closed...")
	client.isClosed = true
}

func (client *Client) splitReqsAndReps() (err error) {
	var replyChan MessageChan

forLoop:
	for {
		Trace.Printf("forLoop splitReqsAndReps")
		select {
		case message, ok := <-client.conn.msgs:
			if !ok {
				Trace.Printf("client.conn.msgs... CLOSED!")
				break forLoop
			}
			Trace.Printf("client.conn.msgs")

			Trace.Printf("splitting incoming message to reqs and reps")

			if message.MessageType == MESSAGE_TYPE_REQUEST {
				// interesting things happen if there are outstanding messages
				// and the client closes
				client.requests <- message
			} else if message.MessageType == MESSAGE_TYPE_REPLY {
				client.openRepliesLock.Lock()
				replyChan = client.openReplies[message.RequestId]
				if replyChan == nil {
					Error.Printf("got an unexpected reply for requestId: %d. Skipping.", message.RequestId)
					client.openRepliesLock.Unlock()
					continue
				}

				delete(client.openReplies, message.RequestId)
				client.openRepliesLock.Unlock()

				replyChan <- message

			} else {
				Error.Printf("Could not handle msg, it's neither req or reply. Skipping.")
				continue
			}
		}
	}

	Trace.Printf("done with SplitReqsAndReps")

	close(client.requests)
	client.openRepliesLock.Lock()
	defer client.openRepliesLock.Unlock()
	for _, openReplyChan := range client.openReplies {
		close(openReplyChan)
	}

	client.Close()

	return
}

// Incoming returns a client's MessageChan
func (client *Client) Incoming() MessageChan {
	return client.requests
}
