package scamp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
)

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
	ClientID         flexInt        `json:"client_id"`            // both
	Ticket           string         `json:"ticket"`               // request
	IdentifyingToken string         `json:"identifying_token"`
	MessageType      messageType    `json:"type"`    // both
	Version          int            `json:"version"` // request
}

var (
	envelopeJSONBytes      = []byte(`"json"`)
	envelopeJSONStoreBytes = []byte(`"jsonstore"`)
)

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

// flexInt used for unmarshalling data, which may be a JSON string, or int, into a golang int
type flexInt int

func (value *flexInt) UnmarshalJSON(data []byte) error {
	intValue, stringValue := 0, ""

	if err := json.Unmarshal(data, &stringValue); err == nil {
		v, err := strconv.Atoi(stringValue)
		if err != nil {
			return fmt.Errorf("Could not parse string `\"%s\"` as int value", stringValue)
		}

		*value = flexInt(v)

		return nil
	}

	if err := json.Unmarshal(data, &intValue); err == nil {
		*value = flexInt(intValue)

		return nil
	}

	return fmt.Errorf("Could not parse data `%s` as int value", data)
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

var (
	requestBytes = []byte(`"request"`)
	replyBytes   = []byte(`"reply"`)
)

func (messageType messageType) MarshalJSON() (retval []byte, err error) {
	switch messageType {
	case MessageTypeRequest:
		retval = requestBytes
	case MessageTypeReply:
		retval = replyBytes
	default:
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
		err = fmt.Errorf("unknown message type `%s`", incoming)
	}

	return
}

func (pktHdr *PacketHeader) Write(writer io.Writer) (err error) {
	jsonEncoder := json.NewEncoder(writer)
	err = jsonEncoder.Encode(pktHdr)

	return
}
