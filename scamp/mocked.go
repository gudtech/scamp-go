package scamp

import (
	"context"
	"fmt"
)

// Extremely simple mock for ScampRequester
// mockRequests are maps with keys as paths (aka 'actions') and values as json strings
type mockedScampRequester struct {
	mockRequests map[string]string
	mockErrors   []error
}

func NewMockedScampRequester(mocks map[string]string) *mockedScampRequester {
	return &mockedScampRequester{
		mockRequests: mocks,
	}
}

func (r *mockedScampRequester) getMockedResponse(action string) (*Message, error) {
	response, ok := r.mockRequests[action]

	if !ok || (response == "") {
		return nil, fmt.Errorf("No response was found for action: %s", action)
	}

	respMessage := NewResponseMessage()
	respMessage.SetAction(action)
	respMessage.Write([]byte(response))

	return respMessage, nil
}

func (r *mockedScampRequester) MockErrors() []error {
	return r.mockErrors
}

func (r *mockedScampRequester) addError(err error) {
	r.mockErrors = append(r.mockErrors, err)
}

// MakeJSONRequest - required to satisfy the ScampRequester interface definition
func (r *mockedScampRequester) MakeJSONRequest(ctx context.Context, mssg *Message, _ int, _ bool) (*Message, error) {

	response, err := r.getMockedResponse(mssg.Action)

	if err != nil {
		err = fmt.Errorf("Error getting mocked response: %s", err)
		r.addError(err)
		return nil, err
	}

	return response, nil
}

// MakeChannelJSONRequest - required to satisfy the ScampRequester interface definition
func (r *mockedScampRequester) MakeChannelJSONRequest(
	c context.Context,
	m *Message,
	t int,
	al bool,
	ch chan *Message,
	er chan error,
) {
	panic("MakeChannelJSONRequest NOT YET SUPPORTED")
}
