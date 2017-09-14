package scamp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	the_rest_size = 5
)

var packetSeenSinceBoot = 0

type Packet struct {
	packetType   PacketType
	msgNo        uint64
	packetHeader PacketHeader
	body         []byte
	// ackRequestId int
}

type PacketType int

const (
	HEADER PacketType = iota
	DATA
	EOF
	TXERR
	ACK
)

var header_bytes = []byte("HEADER")
var data_bytes = []byte("DATA")
var eof_bytes = []byte("EOF")
var txerr_bytes = []byte("TXERR")
var ack_bytes = []byte("ACK")
var theRestBytes = []byte("END\r\n")

/*
  Will parse an io stream in to a packet struct
*/
func ReadPacket(reader *bufio.ReadWriter) (pkt *Packet, err error) {
	pkt = new(Packet)
	var pktTypeBytes []byte
	var bodyBytesNeeded int

	hdrBytes, _, err := reader.ReadLine()
	if err != nil {
		return
	}

	// if enableWriteTee {
	// 	writeTeeTarget.file.Write([]byte("read: "))
	// 	writeTeeTarget.file.Write(hdrBytes)
	// 	writeTeeTarget.file.Write([]byte("\n"))
	// }

	if err != nil {
		return nil, fmt.Errorf("readline error: %s", err)
	}

	hdrValsRead, err := fmt.Sscanf(string(hdrBytes), "%s %d %d", &pktTypeBytes, &(pkt.msgNo), &bodyBytesNeeded)
	if hdrValsRead != 3 {
		return nil, fmt.Errorf("header must have 3 parts")
	} else if err != nil {
		return nil, fmt.Errorf("sscanf error: %s", err.Error)
	}

	Trace.Printf("reading pkt: (%v, `%s`)", pkt.msgNo, pktTypeBytes)

	if bytes.Equal(header_bytes, pktTypeBytes) {
		pkt.packetType = HEADER
	} else if bytes.Equal(data_bytes, pktTypeBytes) {
		pkt.packetType = DATA
	} else if bytes.Equal(eof_bytes, pktTypeBytes) {
		pkt.packetType = EOF
	} else if bytes.Equal(txerr_bytes, pktTypeBytes) {
		pkt.packetType = TXERR
	} else if bytes.Equal(ack_bytes, pktTypeBytes) {
		pkt.packetType = ACK
	} else {
		return nil, fmt.Errorf("unknown packet type `%s`", pktTypeBytes)
	}

	// Use the msg len to consume the rest of the connection
	Trace.Printf("(%v) reading rest of packet body (%d bytes)", packetSeenSinceBoot, bodyBytesNeeded)
	pkt.body = make([]byte, bodyBytesNeeded)
	bytesRead, err := io.ReadFull(reader, pkt.body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: `%s`", err)
	}

	theRest := make([]byte, the_rest_size)
	bytesRead, err = io.ReadFull(reader, theRest)
	if bytesRead != the_rest_size || !bytes.Equal(theRest, []byte("END\r\n")) {
		return nil, fmt.Errorf("packet was missing trailing bytes")
	}

	if pkt.packetType == HEADER {
		err := pkt.parseHeader()
		if err != nil {
			return nil, fmt.Errorf("parseHeader err: `%s`", err)
		}
		pkt.body = nil
	}

	Trace.Printf("(%d) done reading packet", packetSeenSinceBoot)
	packetSeenSinceBoot = packetSeenSinceBoot + 1
	return pkt, nil
}

//TODO: why are we unmarshalling pkt.body here?
func (pkt *Packet) parseHeader() (err error) {
	Trace.Printf("PARSING HEADER (%s)", pkt.body)
	err = json.Unmarshal(pkt.body, &pkt.packetHeader)
	if err != nil {
		Error.Printf("Error parseing scamp msg: %s ", err)
		return
	}

	return
}

func (pkt *Packet) Write(writer io.Writer) (written int, err error) {
	Trace.Printf("writing packet...")
	written = 0

	var packetTypeBytes []byte
	switch pkt.packetType {
	case HEADER:
		packetTypeBytes = header_bytes
	case DATA:
		packetTypeBytes = data_bytes
	case EOF:
		packetTypeBytes = eof_bytes
	case TXERR:
		packetTypeBytes = txerr_bytes
	case ACK:
		packetTypeBytes = ack_bytes
	default:
		err = errors.New(fmt.Sprintf("unknown packetType `%d`", pkt.packetType))
		Error.Printf("unknown packetType `%d`", pkt.packetType)
		return
	}

	bodyBuf := new(bytes.Buffer)
	// TODO this is why you use pointers so you can
	// carry nil values...
	if pkt.packetType == HEADER {
		err = pkt.packetHeader.Write(bodyBuf)
		if err != nil {
			err = fmt.Errorf("err writing packet header: `%s`", err)
			return
		}
		// } else if pkt.packetType == ACK {
		// 	_, err = fmt.Fprintf(bodyBuf, "%d", pkt.ackRequestId)
		// 	if err != nil {
		// 		return
		// 	}
	} else {
		bodyBuf.Write(pkt.body)
	}

	bodyBytes := bodyBuf.Bytes()
	Trace.Printf("writing pkt: (%d, `%s`)", pkt.msgNo, packetTypeBytes)
	Trace.Printf("packet_body: `%s`", bodyBytes)

	headerBytesWritten, err := fmt.Fprintf(writer, "%s %d %d\r\n", packetTypeBytes, pkt.msgNo, len(bodyBytes))
	written = written + headerBytesWritten
	if err != nil {
		err = fmt.Errorf("err writing packet header: `%s`", err)
		return
	}
	bodyBytesWritten, err := writer.Write(bodyBytes)
	written = written + bodyBytesWritten
	if err != nil {
		return
	}

	theRestBytesWritten, err := writer.Write(theRestBytes)
	written = written + theRestBytesWritten

	return
}
