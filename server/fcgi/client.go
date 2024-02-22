// Copyright 2012 Junqing Tan <ivan@mysqlab.net> and The Go Authors
// Use of this source code is governed by a BSD-style
// Part of source code is from Go fcgi package

package fcgi

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

const FCGI_LISTENSOCK_FILENO uint8 = 0
const FCGI_HEADER_LEN uint8 = 8
const VERSION_1 uint8 = 1
const FCGI_NULL_REQUEST_ID uint8 = 0
const FCGI_KEEP_CONN uint8 = 1
const doubleCRLF = "\r\n\r\n"

const (
	FCGI_BEGIN_REQUEST uint8 = iota + 1
	FCGI_ABORT_REQUEST
	FCGI_END_REQUEST
	FCGI_PARAMS
	FCGI_STDIN
	FCGI_STDOUT
	FCGI_STDERR
	FCGI_DATA
	FCGI_GET_VALUES
	FCGI_GET_VALUES_RESULT
	FCGI_UNKNOWN_TYPE
	FCGI_MAXTYPE = FCGI_UNKNOWN_TYPE
)

const (
	FCGI_RESPONDER uint8 = iota + 1
	FCGI_AUTHORIZER
	FCGI_FILTER
)

const (
	FCGI_REQUEST_COMPLETE uint8 = iota
	FCGI_CANT_MPX_CONN
	FCGI_OVERLOADED
	FCGI_UNKNOWN_ROLE
)

const (
	FCGI_MAX_CONNS  string = "MAX_CONNS"
	FCGI_MAX_REQS   string = "MAX_REQS"
	FCGI_MPXS_CONNS string = "MPXS_CONNS"
)

const (
	maxWrite = 65500 // 65530 may work, but for compatibility
	maxPad   = 255
)

type header struct {
	Version       uint8
	Type          uint8
	Id            uint16
	ContentLength uint16
	PaddingLength uint8
	Reserved      uint8
}

// for padding so we don't have to allocate all the time
// not synchronized because we don't care what the contents are
var pad [maxPad]byte

func (h *header) init(recType uint8, reqId uint16, contentLength int) {
	h.Version = 1
	h.Type = recType
	h.Id = reqId
	h.ContentLength = uint16(contentLength)
	h.PaddingLength = uint8(-contentLength & 7)
}

type record struct {
	h   header
	buf []byte
}

func (rec *record) read(r io.Reader) error {
	if err := binary.Read(r, binary.BigEndian, &rec.h); err != nil {
		return err
	}
	if rec.h.Version != 1 {
		err := errors.New("fcgi: invalid header version")
		return err
	}
	n := int(rec.h.ContentLength) + int(rec.h.PaddingLength)

	rec.buf = make([]byte, n)

	if _, err := io.ReadFull(r, rec.buf[:n]); err != nil {
		return err
	}

	rec.buf = rec.buf[:int(rec.h.ContentLength)]

	return nil
}

type FCGIClient struct {
	mutex     sync.Mutex
	rwc       io.ReadWriteCloser
	h         header
	bw        *bufio.Writer
	br        *bufio.Reader
	keepAlive bool
	reqId     uint16
}

// Connects to the fcgi responder at the specified network address.
// See func net.Dial for a description of the network and address parameters.
func Dial(network, address string) (fcgi *FCGIClient, err error) {
	var conn net.Conn

	conn, err = net.Dial(network, address)
	if err != nil {
		return
	}

	fcgi = &FCGIClient{
		rwc:       conn,
		keepAlive: false,
		reqId:     1,
		bw:        bufio.NewWriter(conn),
		br:        bufio.NewReader(conn),
	}

	return
}

// Connects to the fcgi responder at the specified network address with timeout
// See func net.DialTimeout for a description of the network, address and timeout parameters.
func DialTimeout(network, address string, timeout time.Duration) (fcgi *FCGIClient, err error) {

	var conn net.Conn

	conn, err = net.DialTimeout(network, address, timeout)
	if err != nil {
		return
	}

	fcgi = &FCGIClient{
		rwc:       conn,
		keepAlive: false,
		reqId:     1,
	}

	return
}

// Close fcgi connnection
func (this *FCGIClient) Close() {
	this.rwc.Close()
}

func (this *FCGIClient) writeRecord(recType uint8, content []byte) (err error) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	var buf bytes.Buffer

	this.h.init(recType, this.reqId, len(content))
	if err := binary.Write(&buf, binary.BigEndian, this.h); err != nil {
		return err
	}
	if _, err := buf.Write(content); err != nil {
		return err
	}
	if _, err := buf.Write(pad[:this.h.PaddingLength]); err != nil {
		return err
	}
	_, err = this.bw.Write(buf.Bytes())
	return err
}

func (this *FCGIClient) writeBeginRequest(role uint16, flags uint8) error {
	b := [8]byte{byte(role >> 8), byte(role), flags}
	return this.writeRecord(FCGI_BEGIN_REQUEST, b[:])
}

func (this *FCGIClient) writeEndRequest(appStatus int, protocolStatus uint8) error {
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b, uint32(appStatus))
	b[4] = protocolStatus
	return this.writeRecord(FCGI_END_REQUEST, b)
}

func (this *FCGIClient) writePairs(recType uint8, pairs map[string]string) error {
	w := newWriter(this, recType)
	b := make([]byte, 8)
	nn := 0
	for k, v := range pairs {
		m := 8 + len(k) + len(v)
		if m > maxWrite {
			// param data size exceed 65535 bytes"
			vl := maxWrite - 8 - len(k)
			v = v[:vl]
		}
		n := encodeSize(b, uint32(len(k)))
		n += encodeSize(b[n:], uint32(len(v)))
		m = n + len(k) + len(v)
		if (nn + m) > maxWrite {
			w.Flush()
			nn = 0
		}
		nn += m
		if _, err := w.Write(b[:n]); err != nil {
			return err
		}
		if _, err := w.WriteString(k); err != nil {
			return err
		}
		if _, err := w.WriteString(v); err != nil {
			return err
		}
	}
	w.Close()
	return nil
}

func readSize(s []byte) (uint32, int) {
	if len(s) == 0 {
		return 0, 0
	}
	size, n := uint32(s[0]), 1
	if size&(1<<7) != 0 {
		if len(s) < 4 {
			return 0, 0
		}
		n = 4
		size = binary.BigEndian.Uint32(s)
		size &^= 1 << 31
	}
	return size, n
}

func readString(s []byte, size uint32) string {
	if size > uint32(len(s)) {
		return ""
	}
	return string(s[:size])
}

func encodeSize(b []byte, size uint32) int {
	if size > 127 {
		size |= 1 << 31
		binary.BigEndian.PutUint32(b, size)
		return 4
	}
	b[0] = byte(size)
	return 1
}

// bufWriter encapsulates bufio.Writer but also closes the underlying stream when
// Closed.
type bufWriter struct {
	closer io.Closer
	*bufio.Writer
}

func (w *bufWriter) Close() error {
	if err := w.Writer.Flush(); err != nil {
		w.closer.Close()
		return err
	}
	return w.closer.Close()
}

func newWriter(c *FCGIClient, recType uint8) *bufWriter {
	s := &streamWriter{c: c, recType: recType}
	w := bufio.NewWriterSize(s, maxWrite)
	return &bufWriter{s, w}
}

// streamWriter abstracts out the separation of a stream into discrete records.
// It only writes maxWrite bytes at a time.
type streamWriter struct {
	c       *FCGIClient
	recType uint8
}

func (w *streamWriter) Write(p []byte) (int, error) {
	nn := 0
	for len(p) > 0 {
		n := len(p)
		if n > maxWrite {
			n = maxWrite
		}
		if err := w.c.writeRecord(w.recType, p[:n]); err != nil {
			return nn, err
		}
		nn += n
		p = p[n:]
	}
	return nn, nil
}

func (w *streamWriter) Close() error {
	// send empty record to close the stream
	return w.c.writeRecord(w.recType, nil)
}

type streamReader struct {
	c *FCGIClient

	stdout record
	stderr record
}

func (w *streamReader) readResp() error {
	for {
		rec := record{}
		err := rec.read(w.c.br)
		if err != nil {
			return err
		}

		if rec.h.Type == FCGI_STDOUT {
			w.stdout = rec
		} else if rec.h.Type == FCGI_STDERR {
			w.stderr = rec
		} else if rec.h.Type == FCGI_END_REQUEST {
			return nil
		}
	}
}

// Do made the request and returns a io.Reader that translates the data read
// from fcgi responder out of fcgi packet before returning it.
func (this *FCGIClient) Do(p map[string]string, req io.Reader) (stdout, stderr []byte, err error) {
	err = this.writeBeginRequest(uint16(FCGI_RESPONDER), 0)
	if err != nil {
		return
	}

	err = this.writePairs(FCGI_PARAMS, p)
	if err != nil {
		return
	}

	body := newWriter(this, FCGI_STDIN)
	if req != nil {
		io.Copy(body, req)
	}
	body.Close()

	err = this.bw.Flush()
	if err != nil {
		return
	}

	r := &streamReader{c: this}
	r.readResp()

	stdout = r.stdout.buf
	stderr = r.stderr.buf

	return
}

type badStringError struct {
	what string
	str  string
}

func (e *badStringError) Error() string { return fmt.Sprintf("%s %q", e.what, e.str) }

// Checks whether chunked is part of the encodings stack
func chunked(te []string) bool { return len(te) > 0 && te[0] == "chunked" }
