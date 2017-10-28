package scamp

import "io"
import "encoding/json"
import "errors"
import "fmt"
import "bytes"

/******
  ENVELOPE FORMAT
******/

type envelopeFormat int

const (
	ENVELOPE_JSON envelopeFormat = iota
	ENVELOPE_JSONSTORE
)

// Serialized to JSON and stuffed in the 'header' property
// of each packet
type PacketHeader struct {
	Action           string         `json:"action"`               // request
	Envelope         envelopeFormat `json:"envelope"`             // request
	Error            string         `json:"error,omitempty"`      // reply
	ErrorCode        string         `json:"error_code,omitempty"` // reply
	RequestId        int            `json:"request_id"`           // both
	Ticket           string         `json:"ticket"`               // request
	IdentifyingToken string         `json:"identifying_token"`
	MessageType      messageType    `json:"type"`    // both
	Version          int            `json:"version"` // request
}

var envelope_json_bytes = []byte(`"json"`)
var envelope_jsonstore_bytes = []byte(`"jsonstore"`)

func (envFormat envelopeFormat) MarshalJSON() (retval []byte, err error) {
	switch envFormat {
	case ENVELOPE_JSON:
		retval = envelope_json_bytes
	case ENVELOPE_JSONSTORE:
		retval = envelope_jsonstore_bytes
	default:
		err = errors.New(fmt.Sprintf("unknown format `%d`", envFormat))
	}

	return
}

func (envFormat *envelopeFormat) UnmarshalJSON(incoming []byte) error {
	if bytes.Equal(envelope_json_bytes, incoming) {
		*envFormat = ENVELOPE_JSON
	} else if bytes.Equal(envelope_jsonstore_bytes, incoming) {
		*envFormat = ENVELOPE_JSONSTORE
	} else {
		return errors.New(fmt.Sprintf("unknown envelope type `%s`", incoming))
	}
	return nil
}

/******
  MESSAGE TYPE
******/

type messageType int

const (
	_ = iota
	MESSAGE_TYPE_REQUEST
	MESSAGE_TYPE_REPLY
)

var request_bytes = []byte(`"request"`)
var reply_bytes = []byte(`"reply"`)

func (messageType messageType) MarshalJSON() (retval []byte, err error) {
	switch messageType {
	case MESSAGE_TYPE_REQUEST:
		retval = request_bytes
	case MESSAGE_TYPE_REPLY:
		retval = reply_bytes
	default:
		Error.Printf("unknown message type `%d`", messageType)
		err = errors.New(fmt.Sprintf("unknown message type `%d`", messageType))
	}

	return
}

func (msgType *messageType) UnmarshalJSON(incoming []byte) (err error) {
	if bytes.Equal(request_bytes, incoming) {
		*msgType = MESSAGE_TYPE_REQUEST
	} else if bytes.Equal(reply_bytes, incoming) {
		*msgType = MESSAGE_TYPE_REPLY
	} else {
		Error.Printf(fmt.Sprintf("unknown message type `%s`", incoming))
		err = errors.New(fmt.Sprintf("unknown message type `%s`", incoming))
	}

	return
}

func (pktHdr *PacketHeader) Write(writer io.Writer) (err error) {
	jsonEncoder := json.NewEncoder(writer)
	err = jsonEncoder.Encode(pktHdr)

	return
}
