package scamp

import (
	"bytes"
	"encoding/json"
)

// Message represents a scamp message TODO: godoc
type Message struct {
	Action           string
	Envelope         envelopeFormat
	RequestID        int // TODO: how do RequestID's fit in again? NOTE: from (SCAMP repo) -"Set to 18 random base64 bytes"
	Version          int
	MessageType      messageType
	packets          []*Packet
	bytesWritten     uint64
	Ticket           string
	IdentifyingToken string
	Error            string
	ErrorCode        string
}

// NewMessage creates a new scamp message
func NewMessage() (msg *Message) {
	msg = new(Message)
	return
}

// NewRequestMessage creates a new scamp message and sets it's type to 1 (request)
func NewRequestMessage() (msg *Message) {
	msg = new(Message)
	msg.SetMessageType(1)
	return
}

// NewResponseMessage creates a new scamp message and sets it's type to 2 (response)
func NewResponseMessage() (msg *Message) {
	msg = new(Message)
	msg.SetMessageType(2)
	return
}

// SetAction sets teh scamp action name for a message
func (msg *Message) SetAction(action string) {
	msg.Action = action
}

// SetEnvelope sets the envelope type fr a message (JSON, or JSONSTORE)
func (msg *Message) SetEnvelope(env envelopeFormat) {
	msg.Envelope = env
}

// SetVersion sets the api version of the message
func (msg *Message) SetVersion(version int) {
	msg.Version = version
}

// SetMessageType sets the type of message (request or response)
func (msg *Message) SetMessageType(mtype messageType) {
	msg.MessageType = mtype
}

// SetRequestID sets the msg.RequestID
func (msg *Message) SetRequestID(requestID int) {
	msg.RequestID = requestID
}

// SetTicket sets the auth ticket for the message
func (msg *Message) SetTicket(ticket string) {
	msg.Ticket = ticket
}

// SetIdentifyingToken sets the msg.IdentifyingToken
func (msg *Message) SetIdentifyingToken(token string) {
	msg.IdentifyingToken = token
}

// SetError sets the msg.Error
func (msg *Message) SetError(err string) {
	msg.Error = err
}

// SetErrorCode sets the msg.ErrorCode
func (msg *Message) SetErrorCode(errCode string) {
	msg.ErrorCode = errCode
}

// GetError returns msg.Error
func (msg *Message) GetError() (err string) {
	return msg.Error
}

// GetErrorCode returns msg.ErrorCode
func (msg *Message) GetErrorCode() (errCode string) {
	return msg.ErrorCode
}

// GetTicket returns msg.Ticket
func (msg *Message) GetTicket() (ticket string) {
	return msg.Ticket
}

// GetIdentifyingToken returns msg.IdentifyingToken
func (msg *Message) GetIdentifyingToken() (token string) {
	return msg.IdentifyingToken
}

// Write writes the packet data (body) and appends it to msg.packets
func (msg *Message) Write(blob []byte) (n int, err error) {
	// TODO: should this be a sync add?
	msg.bytesWritten += uint64(len(blob))

	msg.packets = append(msg.packets, &Packet{packetType: DATA, body: blob})
	return len(blob), nil
}

var msgChunkSize = 256 * 1024

// WriteJSON takes the message payload, encodes it as JSON and appends it (in chunks)
// to msg.packets
func (msg *Message) WriteJSON(data interface{}) (n int, err error) {
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(data)
	if err != nil {
		return
	}

	msg.bytesWritten += uint64(len(buf.Bytes()))

	// Trace.Printf("WriteJson data size: %d", len(buf.Bytes()))

	if len(buf.Bytes()) > msgChunkSize {
		slice := buf.Bytes()[:]
		for {
			// Trace.Printf("slice size: %d", len(slice))

			if len(slice) < msgChunkSize {
				msg.packets = append(msg.packets, &Packet{packetType: DATA, body: slice})
				break
			} else {
				chunk := make([]byte, msgChunkSize)
				copy(chunk, slice[0:msgChunkSize])
				slice = slice[msgChunkSize:]
				msg.packets = append(msg.packets, &Packet{packetType: DATA, body: chunk})
			}
		}

	} else {
		msg.packets = append(msg.packets, &Packet{packetType: DATA, body: buf.Bytes()})
	}

	return
}

// BytesWritten returns msg.bytesWritten
func (msg *Message) BytesWritten() uint64 {
	return msg.bytesWritten
}

func (msg *Message) toPackets(msgNo uint64) []*Packet {
	headerHeader := PacketHeader{
		Action:           msg.Action,
		Envelope:         msg.Envelope,
		Version:          msg.Version,
		RequestID:        msg.RequestID, // TODO: nope, can't do this
		MessageType:      msg.MessageType,
		Error:            msg.Error,
		ErrorCode:        msg.ErrorCode,
		Ticket:           msg.GetTicket(),
		IdentifyingToken: msg.GetIdentifyingToken(),
	}

	headerPacket := Packet{
		packetHeader: headerHeader,
		packetType:   HEADER,
		msgNo:        msgNo,
	}

	eofPacket := Packet{
		packetType: EOF,
		msgNo:      msgNo,
	}

	packets := make([]*Packet, 1)
	packets[0] = &headerPacket

	for _, dataPacket := range msg.packets {
		dataPacket.msgNo = msgNo
		packets = append(packets, dataPacket)
	}

	packets = append(packets, &eofPacket)

	return packets
}

// Bytes reads from all message packets, writes them to a buffer and returns the buffer.Bytes()
func (msg *Message) Bytes() []byte {
	buf := new(bytes.Buffer)
	for _, pkt := range msg.packets {
		buf.Write(pkt.body)
	}

	return buf.Bytes()
}
