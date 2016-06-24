package scamp

import (
  "bytes"
  "encoding/json"
)

type MessageChan chan *Message

type Message struct {
  Action  string
  Envelope envelopeFormat
  RequestId int // TODO: how do RequestId's fit in again?
  Version int64
  MessageType messageType
  packets []*Packet
  bytesWritten uint64

  Ticket string
  IdentifyingTicket string

  Error string
  ErrorCode string
}


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

func (msg *Message)SetAction(action string) {
  msg.Action = action
}

func (msg *Message)SetEnvelope(env envelopeFormat) {
  msg.Envelope = env
}

func (msg *Message)SetVersion(version int64) {
  msg.Version = version
}

func (msg *Message)SetMessageType(mtype messageType) {
  msg.MessageType = mtype
}

func (msg *Message)SetRequestId(requestId int) {
  msg.RequestId = requestId
}

func (msg *Message)SetTicket(ticket string) {
  msg.Ticket = ticket
}

func (msg *Message)SetIdentifyingTicket(ticket string) {
  msg.IdentifyingTicket = ticket
}

func (msg *Message)SetError(err string) {
  msg.Error = err
}

func (msg *Message)SetErrorCode(errCode string) {
  msg.ErrorCode = errCode
}

func (msg *Message)GetError() (err string) {
  return msg.Error
}

func (msg *Message)GetErrorCode() (errCode string) {
  return msg.ErrorCode
}

func (msg *Message)Write(blob []byte) (n int, err error){
  // TODO: should this be a sync add?
  msg.bytesWritten += uint64(len(blob))

  msg.packets = append(msg.packets, &Packet{packetType: DATA, body: blob})
  return len(blob), nil
}

var MSG_CHUNK_SIZE = 256*1024

func (msg *Message)WriteJson(data interface{}) (n int, err error) {
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
        msg.packets = append(msg.packets, &Packet{packetType: DATA, body: slice} )
        break
      } else {
        chunk := make([]byte, MSG_CHUNK_SIZE)
        copy(chunk,slice[0:MSG_CHUNK_SIZE])
        slice = slice[MSG_CHUNK_SIZE:]
        msg.packets = append(msg.packets, &Packet{packetType: DATA, body: chunk})
      }
    }

  } else {
    msg.packets = append(msg.packets, &Packet{packetType: DATA, body: buf.Bytes()})
  }

  return
}

func (msg *Message)BytesWritten() (uint64) {
  return msg.bytesWritten
}

func (msg *Message)toPackets(msgNo uint64) ([]*Packet) {
  headerHeader := PacketHeader{
    Action:      msg.Action,
    Envelope:    msg.Envelope,
    Version:     msg.Version,
    RequestId:   msg.RequestId, // TODO: nope, can't do this
    MessageType: msg.MessageType,
    Ticket:      msg.Ticket,
  }

  headerPacket := Packet {
    packetHeader: headerHeader,
    packetType:   HEADER,
    msgNo:        msgNo,
  }

  eofPacket := Packet {
    packetType:  EOF,
    msgNo:       msgNo,
  }

  packets := make([]*Packet, 1)
  packets[0] = &headerPacket

  for _,dataPacket := range msg.packets {
    dataPacket.msgNo = msgNo
    packets = append(packets, dataPacket)
  }

  packets = append(packets, &eofPacket)

  return packets
}

func (msg *Message)Bytes() ([]byte) {
  buf := new(bytes.Buffer)
  // Info.Printf("packet count: %d", len(msg.packets))
  // for _,pkt := range msg.packets {
  //   Info.Printf("packet len: %d", len(pkt.body))
  // }
  for _,pkt := range msg.packets {
    buf.Write(pkt.body)
  }

  return buf.Bytes()
}