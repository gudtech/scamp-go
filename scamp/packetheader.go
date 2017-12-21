package scamp

import "io"
import "encoding/json"

import "fmt"
import "bytes"

/******
  ENVELOPE FORMAT
******/

type envelopeFormat int

const (
	// EnvelopeJSON JSON message envelope
	EnvelopeJSON envelopeFormat = iota
	// EnvelopeJSONSTORE JSONSTORE message envelope
	EnvelopeJSONSTORE
)

// PacketHeader Serialized to JSON and stuffed in the 'header' property
// of each packet
type PacketHeader struct {
	Action           string         `json:"action"`               // request
	Envelope         envelopeFormat `json:"envelope"`             // request
	Error            string         `json:"error,omitempty"`      // reply
	ErrorCode        string         `json:"error_code,omitempty"` // reply
	RequestID        int            `json:"request_id"`           // both
	Ticket           string         `json:"ticket"`               // request
	IdentifyingToken string         `json:"identifying_token"`
	MessageType      messageType    `json:"type"`    // both
	Version          int            `json:"version"` // request
}

var envelopeJSONBytes = []byte(`"json"`)
var envelopeJSONStoreBytes = []byte(`"jsonstore"`)

func (envFormat envelopeFormat) MarshalJSON() (retval []byte, err error) {
	switch envFormat {
	case EnvelopeJSON:
		retval = envelopeJSONBytes
	case EnvelopeJSONSTORE:
		retval = envelopeJSONStoreBytes
	default:
		err = fmt.Errorf("unknown format `%d`", envFormat)
	}

	return
}

func (envFormat *envelopeFormat) UnmarshalJSON(incoming []byte) error {
	if bytes.Equal(envelopeJSONBytes, incoming) {
		*envFormat = EnvelopeJSON
	} else if bytes.Equal(envelopeJSONStoreBytes, incoming) {
		*envFormat = EnvelopeJSONSTORE
	} else {
		return fmt.Errorf("unknown envelope type `%s`", incoming)
	}
	return nil
}

/******
  MESSAGE TYPE
******/

type messageType int

const (
	_ = iota
	// MessageTypeRequest represents a request
	MessageTypeRequest
	// MessageTypeReply represents a request
	MessageTypeReply
)

var requestBytes = []byte(`"request"`)
var replyBytes = []byte(`"reply"`)

func (messageType messageType) MarshalJSON() (retval []byte, err error) {
	switch messageType {
	case MessageTypeRequest:
		retval = requestBytes
	case MessageTypeReply:
		retval = replyBytes
	default:
		Error.Printf("unknown message type `%d`", messageType)
		err = fmt.Errorf("unknown message type `%d`", messageType)
	}

	return
}

func (messageType *messageType) UnmarshalJSON(incoming []byte) (err error) {
	if bytes.Equal(requestBytes, incoming) {
		*messageType = MessageTypeRequest
	} else if bytes.Equal(replyBytes, incoming) {
		*messageType = MessageTypeReply
	} else {
		Error.Printf(fmt.Sprintf("unknown message type `%s`", incoming))
		err = fmt.Errorf("unknown message type `%s`", incoming)
	}

	return
}

func (pktHdr *PacketHeader) Write(writer io.Writer) (err error) {
	jsonEncoder := json.NewEncoder(writer)
	err = jsonEncoder.Encode(pktHdr)

	return
}
