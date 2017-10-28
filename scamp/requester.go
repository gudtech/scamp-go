package scamp

import (
	"fmt"
)

var msgJson = "json"
var msgJsonStore = "jsonstore"

func MakeJsonRequest(sector, action string, version int, msg *Message) (responseChan chan *Message, err error) {
	var msgType string
	if msg.Envelope == ENVELOPE_JSON {
		msgType = msgJson
	} else if msg.Envelope == ENVELOPE_JSONSTORE {
		msgType = msgJsonStore
	} else {
		err = fmt.Errorf("unsupported envelope type: `%d`", msg.Envelope)
		return
	}

	err = defaultCache.Refresh()
	if err != nil {
		return
	}

	serviceProxies := defaultCache.SearchByAction(sector, action, version, msgType)
	if serviceProxies == nil {
		err = fmt.Errorf("could not find %s:%s~%d#%s", sector, action, version, msgType)
		return
	}

	msg.SetAction(action)
	msg.SetVersion(version)

	// TODO: shuffle serviceProxies

	sent := false
LOOPING_THROUGH_PROXIES:
	for _, serviceProxy := range serviceProxies {
		client, err := serviceProxy.GetClient()
		if err != nil {
			continue
		}

		responseChan, err = client.Send(msg)
		if err == nil {
			sent = true
			break LOOPING_THROUGH_PROXIES
		}
	}

	if !sent {
		err = fmt.Errorf("no valid clients were created. request failed.")
		return
	}

	return
}
