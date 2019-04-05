package scamp

import (
	"time"

	"fmt"

	"io"
	"os"

	"bytes"

	"crypto/rand"
	"crypto/tls"
	"encoding/base64"

	"sync/atomic"
)

// TODO: crazy debug mode enabled for now
var randomDebuggerString string
var enableWriteTee = false
var writeTeeTargetPath = "/tmp/scamp_proto.bin"

type scampDebugger struct {
	file          *os.File
	wrappedWriter io.Writer
}

var scampDebuggerID = uint64(0)

func newScampDebugger(conn *tls.Conn, clientType string) (handle *scampDebugger, err error) {
	worked := false
	var thisDebuggerID uint64

	for i := 0; i < 10; i++ {
		loadedVal := atomic.LoadUint64(&scampDebuggerID)
		thisDebuggerID = loadedVal + 1
		worked = atomic.CompareAndSwapUint64(&scampDebuggerID, loadedVal, thisDebuggerID)
		if worked {
			break
		}
	}
	if !worked {
		panic("never should happen...")
	}

	handle = new(scampDebugger)

	var path = fmt.Sprintf("%s.%s.%s.%d", writeTeeTargetPath, randomDebuggerString, clientType, thisDebuggerID)
	handle.file, err = os.Create(path)
	if err != nil {
		return
	}

	return
}
func (handle *scampDebugger) Write(p []byte) (n int, err error) {
	formattedStr := fmt.Sprintf("write: %d %s", time.Now().Unix(), p)
	_, err = handle.file.Write([]byte(formattedStr))
	if err != nil {
		return
	}

	return len(p), nil
}

func (handle *scampDebugger) ReadWriter(p []byte) (n int, err error) {
	formattedStr := fmt.Sprintf("read: %d %s", time.Now().Unix(), p)
	_, err = handle.file.Write([]byte(formattedStr))
	if err != nil {
		return
	}

	return len(p), nil
}

func scampDebuggerRandomString() string {
	randBytes := make([]byte, 4, 4)
	_, err := rand.Read(randBytes)
	if err != nil {
		panic("shouldn't happen")
	}
	base64RandBytes := base64.StdEncoding.EncodeToString(randBytes)

	var buffer bytes.Buffer
	buffer.WriteString(base64RandBytes[0:])
	return buffer.String()
}

type scampDebuggerReader struct {
	wraps *scampDebugger
}

func (sdr *scampDebuggerReader) Write(p []byte) (n int, err error) {
	return sdr.wraps.ReadWriter(p)
}
