package scamp

import (
	"fmt"
)

type ActionOptions struct {
	Verify bool
	Privs  []int
	// Location of the ticket_verify_public_key.pem
	TicketVerifyPublicKey string
}

func DefaultActionOptions() ActionOptions {
	return ActionOptions{
		Verify:                true,
		Privs:                 []int{},
		TicketVerifyPublicKey: "",
	}
}

type ServiceOptionsFunc struct {
	callback ServiceActionFunc
	options  ActionOptions
}

func (function ServiceOptionsFunc) Call(message *Message, client *Client) {
	if function.options.Verify || len(function.options.Privs) > 0 {
		ticket, err := VerifyTicket(message.Ticket, function.options.TicketVerifyPublicKey)
		if err != nil {
			ReplyOnError(message, client, "verification", err)
			return
		}

		if len(function.options.Privs) > 0 {
			var missingPrivs []int
			for _, priv := range function.options.Privs {
				found, ok := ticket.Privileges[priv]
				if !found || !ok {
					missingPrivs = append(missingPrivs, priv)
				}
			}

			if len(missingPrivs) > 0 {
				err = fmt.Errorf("missing privileges: %v", missingPrivs)
				ReplyOnError(message, client, "verification", err)
				return
			}
		}
	}

	function.callback.Call(message, client)
}

func ReplyOnError(message *Message, client *Client, errorCode string, err error) {
	if client == nil {
		Info.Println("did not reply to client, missing client")
		return
	}

	if message == nil {
		Info.Println("did not reply to client, missing message")
		return
	}

	if err == nil {
		Info.Println("did not reply to client, missing err")
		return
	}

	respMsg := NewResponseMessage()
	respMsg.SetRequestID(message.RequestID)
	respMsg.SetErrorCode(errorCode)
	respMsg.SetError(err.Error())

	_, clientErr := client.Send(respMsg)
	if clientErr != nil {
		Error.Printf("(messageID: %v, messageAction: %v) send error: %v\n", message.RequestID, message.Action, clientErr)
	}
}
