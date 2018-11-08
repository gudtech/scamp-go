package scamp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const retryLimit = 50

// TODO: remove or protect these global variables
type incomingMsgNo uint64
type outgoingMsgNo uint64

// Connection a scamp connection
type Connection struct {
	conn           *tls.Conn
	Fingerprint    string
	readWriter     *bufio.ReadWriter
	readWriterLock sync.Mutex
	incomingMsgNo  incomingMsgNo
	outgoingMsgNo  outgoingMsgNo
	pktToMsg       map[incomingMsgNo](*Message)
	msgs           chan *Message
	client         *Client
	isClosed       bool
	closedMutex    sync.Mutex
	scampDebugger  *scampDebugger
	closeConnOnce  sync.Once
}

// DialConnection Used by Client to establish a secure connection to the remote service.
// TODO: You must use the *connection.Fingerprint to verify the
// remote host
func DialConnection(connspec string) (conn *Connection, err error) {
	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	config.BuildNameToCertificate()

	dialer := &net.Dialer{
		Timeout: 100 * time.Millisecond,
	}
	// tlsConn, err := tls.Dial("tcp", connspec, config)
	// upgraded to DialWithDialer to handle timeouts
	tlsConn, err := tls.DialWithDialer(dialer, "tcp", connspec, config)
	if err != nil {
		return
	}
	conn = NewConnection(tlsConn, "client")
	return
}

// NewConnection Used by Service
func NewConnection(tlsConn *tls.Conn, connType string) (conn *Connection) {
	conn = new(Connection)
	conn.conn = tlsConn

	// TODO get the end entity certificate instead
	peerCerts := conn.conn.ConnectionState().PeerCertificates
	if len(peerCerts) == 1 {
		peerCert := peerCerts[0]
		conn.Fingerprint = sha1FingerPrint(peerCert)
	}

	var reader io.Reader = conn.conn
	var writer io.Writer = conn.conn
	if enableWriteTee {
		var err error
		conn.scampDebugger, err = newScampDebugger(conn.conn, connType)
		if err != nil {
			panic(fmt.Sprintf("could not create debugger: %s", err))
		}
		// reader = conn.scampDebugger.WrapReader(reader)
		writer = io.MultiWriter(writer, conn.scampDebugger)
		debuggerReaderWriter := scampDebuggerReader{
			wraps: conn.scampDebugger,
		}
		reader = io.TeeReader(reader, &debuggerReaderWriter)
		// nothing
	}

	conn.readWriter = bufio.NewReadWriter(bufio.NewReader(reader), bufio.NewWriter(writer))
	conn.incomingMsgNo = 0
	conn.outgoingMsgNo = 0

	conn.pktToMsg = make(map[incomingMsgNo](*Message))
	conn.msgs = make(chan *Message)

	conn.isClosed = false
	go conn.packetReader()

	return
}

// SetClient sets the client for a *Connection
func (conn *Connection) SetClient(client *Client) {
	conn.client = client
}

func (conn *Connection) packetReader() (err error) {
	if conn == nil {
		return
	}
	// I think we only need to lock on writes, packetReader is only running
	// from one spot.
	// conn.readWriterLock.Lock()
	// defer conn.readWriterLock.Unlock()
	var pkt *Packet

PacketReaderLoop:
	for {
		pkt, err = ReadPacket(conn.readWriter)
		if err != nil {
			// Warning.Printf("Client %v, packet reader go routine %v ReadPacket error %s\n", conn.client.ID, prNum, err)
			if strings.Contains(err.Error(), "readline error: EOF") {
				// Trace.Printf("%s", err)
			} else if strings.Contains(err.Error(), "use of closed network connection") {
				// Trace.Printf("%s", err)
			} else if strings.Contains(err.Error(), "connection reset by peer") {
				// Trace.Printf("%s", err)
			} else {
				// Trace.Printf("%s", err)
				Error.Printf("err: %s", err)
			}
			break PacketReaderLoop
		}

		err = conn.routePacket(pkt)
		if err != nil {
			break PacketReaderLoop
		}
	}

	close(conn.msgs) // TODO: sync.Once
	return
}

func (conn *Connection) routePacket(pkt *Packet) (err error) {
	var msg *Message
	switch {
	case pkt.packetType == HEADER:

		incomingmsgno := atomic.LoadUint64((*uint64)(&conn.incomingMsgNo))
		if pkt.msgNo != incomingmsgno {
			err = fmt.Errorf("out of sequence msgNo: expected %d but got %d", incomingmsgno, pkt.msgNo)
			Error.Printf("%s", err)
			return err
		}

		msg = conn.pktToMsg[incomingMsgNo(pkt.msgNo)]
		if msg != nil {
			err = fmt.Errorf("Bad HEADER; already tracking msgNo %d", pkt.msgNo)
			Error.Printf("%s", err)
			return err
		}

		// Allocate message and copy over header values so we don't have to track them
		// We copy out the packetHeader values and then we can discard it
		msg = NewMessage()
		msg.SetAction(pkt.packetHeader.Action)
		msg.SetEnvelope(pkt.packetHeader.Envelope)
		msg.SetVersion(pkt.packetHeader.Version)
		msg.SetMessageType(pkt.packetHeader.MessageType)
		msg.SetRequestID(pkt.packetHeader.RequestID)
		msg.SetError(pkt.packetHeader.Error)
		msg.SetErrorCode(pkt.packetHeader.ErrorCode)
		msg.SetTicket(pkt.packetHeader.Ticket)
		// TODO: Do we need the requestId?

		conn.pktToMsg[incomingMsgNo(pkt.msgNo)] = msg
		// This is for sending out data
		// conn.incomingNotifiers[pktMsgNo] = &make((chan *Message),1)

		atomic.AddUint64((*uint64)(&conn.incomingMsgNo), 1)
	case pkt.packetType == DATA:
		// Trace.Printf("DATA")
		// Append data
		// Verify we are tracking that message
		msg = conn.pktToMsg[incomingMsgNo(pkt.msgNo)]
		if msg == nil {
			return fmt.Errorf("not tracking message number %d", pkt.msgNo)
		}

		msg.Write(pkt.body)
		conn.ackBytes(incomingMsgNo(pkt.msgNo), msg.BytesWritten())

	case pkt.packetType == EOF:
		// Trace.Printf("EOF")
		// Deliver message
		msg = conn.pktToMsg[incomingMsgNo(pkt.msgNo)]
		if msg == nil {
			err = fmt.Errorf("cannot process EOF for unknown msgno %d", pkt.msgNo)
			Error.Printf("err: `%s`", err)
			return
		}

		delete(conn.pktToMsg, incomingMsgNo(pkt.msgNo))
		// Trace.Printf("Delivering message number %d up the stack", pkt.msgNo)
		// Trace.Printf("Adding message to channel:")
		conn.msgs <- msg

	case pkt.packetType == TXERR:
		msg = conn.pktToMsg[incomingMsgNo(pkt.msgNo)]
		if msg == nil {
			err = fmt.Errorf("cannot process EOF for unknown msgno %d", pkt.msgNo)
			Error.Printf("err: `%s`", err)
			return
		}
		//get the error
		if len(pkt.body) > 0 {
			// Trace.Printf("getting error from packet body: %s", pkt.body)
			errMessage := string(pkt.body)
			msg.Error = errMessage
		} else {
			msg.Error = "There was an unknown error with the connection"
		}
		msg.Write(pkt.body)
		conn.ackBytes(incomingMsgNo(pkt.msgNo), msg.BytesWritten())

		delete(conn.pktToMsg, incomingMsgNo(pkt.msgNo))
		conn.msgs <- msg

	case pkt.packetType == ACK:
		// Trace.Printf("ACK `%v` for msgno %v", len(pkt.body), pkt.msgNo)
		// panic("Xavier needs to support this")
		// TODO: Add bytes to message stream tally
	}

	return
}

// Send sends a scamp message using the current *Connection
func (conn *Connection) Send(msg *Message) (err error) {
	if conn == nil {
		return fmt.Errorf("cannot send on nil connection")
	}
	if conn.isClosed {
		err = fmt.Errorf("connection already closed")
	}

	conn.readWriterLock.Lock()
	defer conn.readWriterLock.Unlock()
	if msg.RequestID == 0 {
		err = fmt.Errorf("must specify `ReqestId` on msg before sending")
		return
	}

	outMsgNo := atomic.LoadUint64((*uint64)(&conn.outgoingMsgNo))
	atomic.AddUint64((*uint64)(&conn.outgoingMsgNo), 1)

	// Trace.Printf("sending outMsgNo %d", outgoingMsgNo)

	for _, pkt := range msg.toPackets(outMsgNo) {
		// Trace.Printf("sending pkt %d", i)

		retries := 0
		if enableWriteTee {
			writer := io.MultiWriter(conn.readWriter, conn.scampDebugger)
			_, err := pkt.Write(writer)
			// conn.scampDebugger.file.Write([]byte("\n"))
			if err != nil {
				Error.Printf("error writing packet: %s", err)
				return err
			}
		} else {
			for {
				_, err := pkt.Write(conn.readWriter)
				// TODO: should we actually blacklist this error?
				if err != nil {
					//temporarily
					if strings.Contains(err.Error(), "use of closed connection") {
						err = fmt.Errorf("connection closed")
						break
					}
					//TODO: attempt to reconnect
					if strings.Contains(err.Error(), "broken pipe") {
						err = fmt.Errorf("connection closed: %s", err)
						break
					}

					if retries > retryLimit {
						return fmt.Errorf("Retried too many times: %s", err)
					}

					Error.Printf("error writing packet: %s (retrying)", err)
					retries++
					continue
				}
				break
			}
		}

	}
	conn.readWriter.Flush()

	return
}

func (conn *Connection) ackBytes(msgno incomingMsgNo, unAckedByteCount uint64) (err error) {
	// Trace.Printf("ACKing msg %v, un-acknowledged bytes = %v", msgno, unAckedByteCount)
	conn.readWriterLock.Lock()
	defer conn.readWriterLock.Unlock()

	ackPacket := Packet{
		packetType: ACK,
		msgNo:      uint64(msgno),
		body:       []byte(fmt.Sprintf("%d", unAckedByteCount)),
	}

	var thisWriter io.Writer
	if enableWriteTee {
		thisWriter = io.MultiWriter(conn.readWriter, conn.scampDebugger)
	} else {
		thisWriter = conn.readWriter
	}

	_, err = ackPacket.Write(thisWriter)
	if err != nil {
		return err
	}

	conn.readWriter.Flush()

	return
}

// close closes the current *Connection
func (conn *Connection) close() {
	conn.closeConnOnce.Do(func() {
		conn.closedMutex.Lock()
		if conn.isClosed {
			conn.closedMutex.Unlock()
			return
		}

		err := conn.conn.Close()
		if err != nil {
			Error.Println("could not close client connection: ", err)
		}

		conn.isClosed = true
		conn.closedMutex.Unlock()
	})
}
