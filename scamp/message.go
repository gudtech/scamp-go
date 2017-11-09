package scamp

import (
	"bytes"
	"encoding/json"
)

type MessageChan chan *Message

type Message struct {
	Action   string
	Envelope envelopeFormat
	// TODO: how do RequestId's fit in again? NOTE: from (SCAMP repo) -"Set to 18 random base64 bytes"
	RequestId        int
	Version          int
	MessageType      messageType
	packets          []*Packet
	bytesWritten     uint64
	Ticket           string
	IdentifyingToken string
	Error            string
	ErrorCode        string
}

// Notes from scamp repo:
// action	    request	  The full action name, e.g. Customer.Order.create
// envelope	  request	  String identifying the envelope format, e.g. json
// error	    reply	    Humanreadable error if present
// error_code	reply	    Error code if present
// message_id	both	    String used to correlate requests to responses in the event of reordering. RECOMMENDED 18 random bytes base64-encoded.
// station	  request	  A station ticket, see Ticket format. Optional
// ticket	    request	  An authorization ticket, see Ticket format. Optional
// type	      both	    request or reply
// version	  request	  Action version number

func NewMessage() (msg *Message) {
	msg = new(Message)
	return
}

func NewRequestMessage() (msg *Message) {
	msg = new(Message)
	msg.SetMessageType(1)
	return
}

func NewResponseMessage() (msg *Message) {
	msg = new(Message)
	msg.SetMessageType(2)
	return
}

func (msg *Message) SetAction(action string) {
	msg.Action = action
}

func (msg *Message) SetEnvelope(env envelopeFormat) {
	msg.Envelope = env
}

func (msg *Message) SetVersion(version int) {
	msg.Version = version
}

func (msg *Message) SetMessageType(mtype messageType) {
	msg.MessageType = mtype
}

func (msg *Message) SetRequestId(requestId int) {
	msg.RequestId = requestId
}

func (msg *Message) SetTicket(ticket string) {
	msg.Ticket = ticket
}

func (msg *Message) SetIdentifyingToken(token string) {
	msg.IdentifyingToken = token
}

func (msg *Message) SetError(err string) {
	msg.Error = err
}

func (msg *Message) SetErrorCode(errCode string) {
	msg.ErrorCode = errCode
}

func (msg *Message) GetError() (err string) {
	return msg.Error
}

func (msg *Message) GetErrorCode() (errCode string) {
	return msg.ErrorCode
}

func (msg *Message) GetTicket() (ticket string) {
	return msg.Ticket
}

func (msg *Message) GetIdentifyingToken() (token string) {
	return msg.IdentifyingToken
}

func (msg *Message) Write(blob []byte) (n int, err error) {
	// TODO: should this be a sync add?
	msg.bytesWritten += uint64(len(blob))

	msg.packets = append(msg.packets, &Packet{packetType: DATA, body: blob})
	return len(blob), nil
}

var MSG_CHUNK_SIZE = 256 * 1024

func (msg *Message) WriteJson(data interface{}) (n int, err error) {
	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(data)
	if err != nil {
		return
	}

	msg.bytesWritten += uint64(len(buf.Bytes()))

	// Trace.Printf("WriteJson data size: %d", len(buf.Bytes()))

	if len(buf.Bytes()) > MSG_CHUNK_SIZE {
		slice := buf.Bytes()[:]
		for {
			// Trace.Printf("slice size: %d", len(slice))

			if len(slice) < MSG_CHUNK_SIZE {
				msg.packets = append(msg.packets, &Packet{packetType: DATA, body: slice})
				break
			} else {
				chunk := make([]byte, MSG_CHUNK_SIZE)
				copy(chunk, slice[0:MSG_CHUNK_SIZE])
				slice = slice[MSG_CHUNK_SIZE:]
				msg.packets = append(msg.packets, &Packet{packetType: DATA, body: chunk})
			}
		}

	} else {
		msg.packets = append(msg.packets, &Packet{packetType: DATA, body: buf.Bytes()})
	}

	return
}

func (msg *Message) BytesWritten() uint64 {
	return msg.bytesWritten
}

func (msg *Message) toPackets(msgNo uint64) []*Packet {
	headerHeader := PacketHeader{
		Action:           msg.Action,
		Envelope:         msg.Envelope,
		Version:          msg.Version,
		RequestId:        msg.RequestId, // TODO: nope, can't do this
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

func (msg *Message) Bytes() []byte {
	buf := new(bytes.Buffer)
	// Info.Printf("packet count: %d", len(msg.packets))
	// for _,pkt := range msg.packets {
	//   Info.Printf("packet len: %d", len(pkt.body))
	// }
	for _, pkt := range msg.packets {
		buf.Write(pkt.body)
	}

	return buf.Bytes()
}
