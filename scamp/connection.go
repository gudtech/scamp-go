package scamp

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io"

	"strings"
	"sync"
	"sync/atomic"
)

type IncomingMsgNo uint64
type OutgoingMsgNo uint64

// Connection a scamp connection
type Connection struct {
	conn        *tls.Conn
	Fingerprint string

	// reader         *bufio.Reader
	// writer         *bufio.Writer
	readWriter     *bufio.ReadWriter
	readWriterLock sync.Mutex

	incomingmsgno IncomingMsgNo
	outgoingmsgno OutgoingMsgNo

	pktToMsg map[IncomingMsgNo](*Message)
	msgs     MessageChan

	client *Client

	isClosed    bool
	closedMutex sync.Mutex

	scampDebugger *ScampDebugger
}

// DialConnection Used by Client to establish a secure connection to the remote service.
// TODO: You must use the *connection.Fingerprint to verify the
// remote host
func DialConnection(connspec string) (conn *Connection, err error) {
	Info.Printf("Dialing connection to `%s`", connspec)
	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	config.BuildNameToCertificate()

	tlsConn, err := tls.Dial("tcp", connspec, config)
	if err != nil {
		return
	}
	Trace.Printf("Past TLS")
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
		conn.scampDebugger, err = NewScampDebugger(conn.conn, connType)
		if err != nil {
			panic(fmt.Sprintf("could not create debugger: %s", err))
		}
		// reader = conn.scampDebugger.WrapReader(reader)
		writer = io.MultiWriter(writer, conn.scampDebugger)
		debuggerReaderWriter := ScampDebuggerReader{
			wraps: conn.scampDebugger,
		}
		reader = io.TeeReader(reader, &debuggerReaderWriter)
		// nothing
	}

	conn.readWriter = bufio.NewReadWriter(bufio.NewReader(reader), bufio.NewWriter(writer))
	conn.incomingmsgno = 0
	conn.outgoingmsgno = 0

	conn.pktToMsg = make(map[IncomingMsgNo](*Message))
	conn.msgs = make(MessageChan)

	conn.isClosed = false

	go conn.packetReader()

	return
}

// SetClient sets the client for a *Connection
func (conn *Connection) SetClient(client *Client) {
	conn.client = client
}

func (conn *Connection) packetReader() (err error) {
	// I think we only need to lock on writes, packetReader is only running
	// from one spot.
	// conn.readWriterLock.Lock()
	// defer conn.readWriterLock.Unlock()

	// Trace.Printf("starting packetrouter")
	var pkt *Packet

PacketReaderLoop:
	for {
		Trace.Printf("reading packet...")

		pkt, err = ReadPacket(conn.readWriter)
		if err != nil {
			if strings.Contains(err.Error(), "readline error: EOF") {
				Trace.Printf("%s", err)
			} else if strings.Contains(err.Error(), "use of closed network connection") {
				Trace.Printf("%s", err)
			} else if strings.Contains(err.Error(), "connection reset by peer") {
				Trace.Printf("%s", err)
			} else {
				Trace.Printf("%s", err)
				Error.Printf("err: %s", err)
			}
			break PacketReaderLoop
		}

		err = conn.routePacket(pkt)
		if err != nil {
			Trace.Printf("breaking PacketReaderLoop")
			break PacketReaderLoop
		}
	}

	// we close after routePacket is no longer possible
	// to avoid any send after close panics
	close(conn.msgs)

	return
}

func (conn *Connection) routePacket(pkt *Packet) (err error) {
	var msg *Message
	Trace.Printf("routing packet...")
	switch {
	case pkt.packetType == HEADER:
		Trace.Printf("HEADER")
		// Allocate new msg
		// First verify it's the expected incoming msgno
		incomingmsgno := atomic.LoadUint64((*uint64)(&conn.incomingmsgno))
		if pkt.msgNo != incomingmsgno {
			err = fmt.Errorf("out of sequence msgno: expected %d but got %d", incomingmsgno, pkt.msgNo)
			Error.Printf("%s", err)
			return err
		}

		msg = conn.pktToMsg[IncomingMsgNo(pkt.msgNo)]
		if msg != nil {
			err = fmt.Errorf("Bad HEADER; already tracking msgno %d", pkt.msgNo)
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
		msg.SetRequestId(pkt.packetHeader.RequestId)
		msg.SetError(pkt.packetHeader.Error)
		msg.SetErrorCode(pkt.packetHeader.ErrorCode)
		msg.SetTicket(pkt.packetHeader.Ticket)
		// TODO: Do we need the requestId?

		conn.pktToMsg[IncomingMsgNo(pkt.msgNo)] = msg
		// This is for sending out data
		// conn.incomingNotifiers[pktMsgNo] = &make((chan *Message),1)

		atomic.AddUint64((*uint64)(&conn.incomingmsgno), 1)
	case pkt.packetType == DATA:
		Trace.Printf("DATA")
		// Append data
		// Verify we are tracking that message
		msg = conn.pktToMsg[IncomingMsgNo(pkt.msgNo)]
		if msg == nil {
			err = fmt.Errorf("not tracking message number %d", pkt.msgNo)
			Error.Printf("unexpected error: `%s`", err)
			return err
		}

		msg.Write(pkt.body)
		conn.ackBytes(IncomingMsgNo(pkt.msgNo), msg.BytesWritten())

	case pkt.packetType == EOF:
		Trace.Printf("EOF")
		// Deliver message
		msg = conn.pktToMsg[IncomingMsgNo(pkt.msgNo)]
		if msg == nil {
			err = fmt.Errorf("cannot process EOF for unknown msgno %d", pkt.msgNo)
			Error.Printf("err: `%s`", err)
			return
		}

		delete(conn.pktToMsg, IncomingMsgNo(pkt.msgNo))
		Trace.Printf("Delivering message number %d up the stack", pkt.msgNo)
		Trace.Printf("Adding message to channel:")
		conn.msgs <- msg

	case pkt.packetType == TXERR:
		Trace.Printf("TXERR")

		msg = conn.pktToMsg[IncomingMsgNo(pkt.msgNo)]
		if msg == nil {
			err = fmt.Errorf("cannot process EOF for unknown msgno %d", pkt.msgNo)
			Error.Printf("err: `%s`", err)
			return
		}
		//get the error
		if len(pkt.body) > 0 {
			Trace.Printf("getting error from packet body: %s", pkt.body)
			errMessage := string(pkt.body)
			msg.Error = errMessage
		} else {
			msg.Error = "There was an unkown error with the connection"
		}
		msg.Write(pkt.body)
		conn.ackBytes(IncomingMsgNo(pkt.msgNo), msg.BytesWritten())

		delete(conn.pktToMsg, IncomingMsgNo(pkt.msgNo))
		conn.msgs <- msg

		// Info.Printf("Sending err over channel")
		// conn.errors <- err
		// TODO: add 'error' path on connection
		// Kill connection
		// conn.Close() // is this the correct way to kill connection?

	case pkt.packetType == ACK:
		Trace.Printf("ACK `%v` for msgno %v", len(pkt.body), pkt.msgNo)
		// panic("Xavier needs to support this")
		// TODO: Add bytes to message stream tally
	}

	return
}

// Send sends a scamp message using the current *Connection
func (conn *Connection) Send(msg *Message) (err error) {
	conn.readWriterLock.Lock()
	defer conn.readWriterLock.Unlock()
	if msg.RequestId == 0 {
		err = fmt.Errorf("must specify `ReqestId` on msg before sending")
		return
	}

	outgoingmsgno := atomic.LoadUint64((*uint64)(&conn.outgoingmsgno))
	atomic.AddUint64((*uint64)(&conn.outgoingmsgno), 1)

	Trace.Printf("sending msgno %d", outgoingmsgno)

	for i, pkt := range msg.toPackets(outgoingmsgno) {
		Trace.Printf("sending pkt %d", i)

		if enableWriteTee {
			writer := io.MultiWriter(conn.readWriter, conn.scampDebugger)
			_, err := pkt.Write(writer)
			conn.scampDebugger.file.Write([]byte("\n"))
			if err != nil {
				Error.Printf("error writing packet: `%s`", err)
				return err
			}
		} else {
			for {
				_, err := pkt.Write(conn.readWriter)
				// TODO: should we actually blacklist this error?
				if err != nil {
					Error.Printf("error writing packet: `%s` (retrying)", err)
					continue
				}
				break
			}
		}

	}
	conn.readWriter.Flush()
	Trace.Printf("done sending msg")

	return
}

func (conn *Connection) ackBytes(msgno IncomingMsgNo, unackedByteCount uint64) (err error) {
	Trace.Printf("ACKing msg %v, unacked bytes = %v", msgno, unackedByteCount)
	conn.readWriterLock.Lock()
	defer conn.readWriterLock.Unlock()

	ackPacket := Packet{
		packetType: ACK,
		msgNo:      uint64(msgno),
		body:       []byte(fmt.Sprintf("%d", unackedByteCount)),
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

// Close closes the current *Connection
func (conn *Connection) Close() {
	conn.closedMutex.Lock()
	if conn.isClosed {
		Trace.Printf("connection already closed. skipping shutdown.")
		conn.closedMutex.Unlock()
		return
	}

	Trace.Printf("connection is closing")

	conn.conn.Close()

	conn.isClosed = true
	conn.closedMutex.Unlock()
}
