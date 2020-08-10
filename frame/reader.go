package frame

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"io"
	"strings"
	"time"
)

const (
	bufferSize = 4096
	newline    = byte(10)
	cr         = byte(13)
	colon      = byte(58)
	nullByte   = byte(0)
)

var (
	ErrInvalidCommand     = errors.New("invalid command")
	ErrInvalidFrameFormat = errors.New("invalid frame format")
)

// The Reader type reads STOMP frames from an underlying io.Reader.
// The reader is buffered, and the size of the buffer is the maximum
// size permitted for the STOMP frame command and header section.
// A STOMP frame is rejected if its command and header section exceed
// the buffer size.
type Reader struct {
	reader *bufio.Reader
}

// NewReader creates a Reader with the default underlying buffer size.
func NewReader(reader io.Reader) *Reader {
	return NewReaderSize(reader, bufferSize)
}

// NewReaderSize creates a Reader with an underlying bufferSize
// of the specified size.
func NewReaderSize(reader io.Reader, bufferSize int) *Reader {
	return &Reader{reader: bufio.NewReaderSize(reader, bufferSize)}
}

// Read a STOMP frame from the input. If the input contains one
// or more heart-beat characters and no frame, then nil will
// be returned for the frame. Calling programs should always check
// for a nil frame.
func (r *Reader) Read() (*Frame, error) {
	commandSlice, err := r.readLine()
	if err != nil {
		return nil, err
	}

	if len(commandSlice) == 0 {
		// received a heart-beat newline char (or cr-lf)
		return nil, nil
	}

	f := New(string(commandSlice))
	//println("RX:", f.Command)
	switch f.Command {
	// TODO(jpj): Is it appropriate to perform validation on the
	// command at this point. Probably better to validate higher up,
	// this way this type can be useful for any other non-STOMP protocols
	// which happen to use the same frame format.
	case CONNECT, STOMP, SEND, SUBSCRIBE,
		UNSUBSCRIBE, ACK, NACK, BEGIN,
		COMMIT, ABORT, DISCONNECT, CONNECTED,
		MESSAGE, RECEIPT, ERROR:
		// valid command
	default:
		return nil, ErrInvalidCommand
	}

	// read headers
	for {
		headerSlice, err := r.readLine()
		if err != nil {
			return nil, err
		}

		if len(headerSlice) == 0 {
			// empty line means end of headers
			break
		}

		index := bytes.IndexByte(headerSlice, colon)
		if index <= 0 {
			// colon is missing or header name is zero length
			return nil, ErrInvalidFrameFormat
		}

		name, err := unencodeValue(headerSlice[0:index])
		if err != nil {
			return nil, err
		}
		value, err := unencodeValue(headerSlice[index+1:])
		if err != nil {
			return nil, err
		}

		//println("   ", name, ":", value)

		f.Header.Add(name, value)
	}

	// get content length from the headers
	if contentLength, ok, err := f.Header.ContentLength(); err != nil {
		// happens if the content is malformed
		return nil, err
	} else if ok {
		// content length specified in the header, so use that
		f.Body = make([]byte, contentLength)
		for bytesRead := 0; bytesRead < contentLength; {
			n, err := r.reader.Read(f.Body[bytesRead:contentLength])
			if err != nil {
				return nil, err
			}
			bytesRead += n
		}

		// read the next byte and verify that it is a null byte
		terminator, err := r.reader.ReadByte()
		if err != nil {
			return nil, err
		}
		if terminator != 0 {
			return nil, ErrInvalidFrameFormat
		}
	} else {
		f.Body, err = r.reader.ReadBytes(nullByte)
		if err != nil {
			return nil, err
		}
		// remove trailing null
		f.Body = f.Body[0 : len(f.Body)-1]
	}

	// pass back frame
	return f, nil
}

// read one line from input and strip off terminating LF or terminating CR-LF
func (r *Reader) readLine() (line []byte, err error) {
	line, err = r.reader.ReadBytes(newline)
	if err != nil {
		return
	}

	switch {
	case bytes.HasSuffix(line, crlfSlice):
		line = line[0 : len(line)-len(crlfSlice)]
	case bytes.HasSuffix(line, newlineSlice):
		line = line[0 : len(line)-len(newlineSlice)]
	}

	return
}

type FrameReader struct {
	wsConn *websocket.Conn
}

func NewFrameReader(wsConn *websocket.Conn) *FrameReader {
	return &FrameReader{wsConn: wsConn}
}

func (r *FrameReader) ReadWithTimeout(second int) (*Frame, error) {
	timeout := time.After(5 * time.Second)

	for {
		select {
		case <-timeout:
			return nil, errors.New("timed out")
		default:
			f, err := r.Read()
			if err != nil {
				return nil, err
			}
			if f != nil {
				return f, nil
			}
		}
	}
}

func (r *FrameReader) Read() (*Frame, error) {
	commandSlice, err := r.readLine()
	if err != nil {
		return nil, err
	}

	for len(commandSlice) > 1 && (commandSlice[0] != '[' || commandSlice[len(commandSlice)-1] != ']') {
		if commandSlice[0] != '[' {
			commandSlice = commandSlice[1:]
		}
		_len := len(commandSlice)
		if commandSlice[_len-1] != ']' {
			commandSlice = commandSlice[0 : _len-1]
		}
	}

	if len(commandSlice) <= 1 {
		// received a heart-beat newline char (or cr-lf)
		return nil, nil
	}

	var messages []string
	_ = json.Unmarshal([]byte(commandSlice), &messages)

	if len(messages) < 1 {
		return nil, nil
	}

	message := messages[0]

	frames := strings.Split(message, "\n\n")
	headers := strings.Split(frames[0], "\n")

	// headers[0] : command
	// herader[1:] : headers
	// frames[1] : body (maybe)

	command := headers[0]

	f := New(command)
	//println("RX:", f.Command)
	switch f.Command {
	// TODO(jpj): Is it appropriate to perform validation on the
	// command at this point. Probably better to validate higher up,
	// this way this type can be useful for any other non-STOMP protocols
	// which happen to use the same frame format.
	case CONNECT, STOMP, SEND, SUBSCRIBE,
		UNSUBSCRIBE, ACK, NACK, BEGIN,
		COMMIT, ABORT, DISCONNECT, CONNECTED,
		MESSAGE, RECEIPT, ERROR:
		// valid command
	default:
		return nil, ErrInvalidCommand
	}

	// read headers
	for i := 1; i < len(headers); i++ {
		header := headers[i]

		if len(header) == 0 {
			// empty line means end of headers
			break
		}

		headerSlice := []byte(header)

		index := bytes.IndexByte(headerSlice, colon)
		if index <= 0 {
			// colon is missing or header name is zero length
			return nil, ErrInvalidFrameFormat
		}

		name, err := unencodeValue(headerSlice[0:index])
		if err != nil {
			return nil, err
		}
		value, err := unencodeValue(headerSlice[index+1:])
		if err != nil {
			return nil, err
		}

		//println("   ", name, ":", value)

		f.Header.Add(name, value)
	}

	if len(frames) == 2 {
		// frames[1] : body
		body := []byte(frames[1])
		first := bytes.IndexByte(body, byte('{'))
		last := bytes.LastIndexByte(body, byte('}'))

		if first < 0 || last < 0 {
			f.Body = nil
		} else {
			f.Body = body[first : last+1]
		}
	}

	// pass back frame
	return f, nil
}

// read one line from input and strip off terminating LF or terminating CR-LF
func (r *FrameReader) readLine() (line []byte, err error) {
	_, line, err = r.wsConn.ReadMessage()

	if err != nil {
		return
	}

	switch {
	case bytes.HasSuffix(line, crlfSlice):
		line = line[0 : len(line)-len(crlfSlice)]
	case bytes.HasSuffix(line, newlineSlice):
		line = line[0 : len(line)-len(newlineSlice)]
	}

	return
}
