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

	fmt.Printf("searching for action\n")
	serviceProxies, err = defaultCache.SearchByAction(sector, action, version, msgType)
	if err != nil {
		return
	}
	if len(serviceProxies) == 0 {
		err = fmt.Errorf("could not find %s:%s~%d#%s", sector, action, version, msgType)
		return
	}
	fmt.Printf("got action\n")

	msg.SetAction(action)
	msg.SetVersion(version)

	sent := false
	var responseChan chan *Message

	for _, serviceProxy := range serviceProxies {
		fmt.Printf("getting client\n")
		client, err := serviceProxy.GetClient()
		if err != nil {
			continue
		}
		fmt.Printf("got client\n")

		fmt.Printf("sending msg\n")
		responseChan, err = client.Send(msg)
		if err == nil {
			sent = true
			break
		}
		fmt.Printf("finished sending\n")
	}

	if !sent {
		err = fmt.Errorf("Request failed: %s.%s not found: %s", sector, action, err)
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
		case <-time.After(300 * time.Second):
			//close(responseChan)
			err = fmt.Errorf("request timed out")
			return
		}
	}
}
