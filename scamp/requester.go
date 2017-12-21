package scamp

import (
	"fmt"
	"time"
)

// MakeJSONRequest retreives the appropriate service proxy based on the message action, and makes a
// JSON request.
func MakeJSONRequest(sector, action string, version int, msg *Message) (message *Message, err error) {
	var msgType string
	if msg.Envelope == EnvelopeJSON {
		msgType = "json"
	} else if msg.Envelope == EnvelopeJSONSTORE {
		msgType = "jsonstore"
	} else {
		err = fmt.Errorf("unsupported envelope type: `%d`", msg.Envelope)
		return
	}

	//TODO: add retry logic in case service proxies are nil
	var serviceProxies []*serviceProxy
	serviceProxies = defaultCache.SearchByAction(sector, action, version, msgType)
	if serviceProxies == nil {
		err = fmt.Errorf("could not find %s:%s~%d#%s", sector, action, version, msgType)
		return
	}

	msg.SetAction(action)
	msg.SetVersion(version)

	sent := false
	var responseChan chan *Message

	for _, serviceProxy := range serviceProxies {
		Info.Printf("service proxy %s\n", serviceProxy.ident)

		client, err := serviceProxy.GetClient()
		if err != nil {
			continue
		}

		responseChan, err = client.Send(msg)
		if err == nil {
			sent = true
			break
		}
	}

	if !sent {
		err = fmt.Errorf("no valid clients were created, request failed: %s", err)
		return
	}

	for {
		select {
		case msg, ok := <-responseChan:
			if !ok {
				break
			}
			if msg == nil {
				break
			}
			message = msg
			return
		case <-time.After(60 * time.Second):
			close(responseChan)
			err = fmt.Errorf("request timed out")
			return
		}
	}
}
