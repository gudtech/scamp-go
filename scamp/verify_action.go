package scamp

import (
	"fmt"
)

type VerifyAction struct {
	callback              ServiceActionFunc
	ticketVerifyPublicKey string
}

func (function VerifyAction) Call(message *Message, client *Client) {
	_, err := VerifyTicket(message.Ticket, function.ticketVerifyPublicKey)
	if err != nil {
		ReplyOnError(message, client, "verification", err)
		return
	}

	function.callback.Call(message, client)
}

func WithVerification(callback ServiceActionFunc, ticketVerifyPublicKey string) VerifyAction {
	return VerifyAction{
		callback:              callback,
		ticketVerifyPublicKey: ticketVerifyPublicKey,
	}
}

type PrivAction struct {
	callback              ServiceActionFunc
	privs                 []int
	ticketVerifyPublicKey string
}

func (function PrivAction) Call(message *Message, client *Client) {
	err := function.Verify(message, client)
	if err != nil {
		ReplyOnError(message, client, "verification", err)
		return
	}

	function.callback.Call(message, client)
}

func (function PrivAction) Verify(message *Message, client *Client) error {
	ticket, err := VerifyTicket(message.Ticket, function.ticketVerifyPublicKey)
	if err != nil {
		return err
	}

	var missingPrivs []int
	for _, priv := range function.privs {
		found, ok := ticket.Privileges[priv]
		if !found || !ok {
			missingPrivs = append(missingPrivs, priv)
		}
	}

	if len(missingPrivs) > 0 {
		return fmt.Errorf("missing privileges: %v", missingPrivs)
	}

	return nil
}

func WithPrivs(callback ServiceActionFunc, privs []int, ticketVerifyPublicKey string) PrivAction {
	return PrivAction{
		callback:              callback,
		privs:                 privs,
		ticketVerifyPublicKey: ticketVerifyPublicKey,
	}
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
