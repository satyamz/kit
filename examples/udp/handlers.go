package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/ardanlabs/kit/log"
	"github.com/ardanlabs/kit/udp"
)

// udpConnHandler is required to process data.
type udpConnHandler struct{}

// Bind is called to init to reader and writer.
func (udpConnHandler) Bind(traceID string, listener *net.UDPConn) (io.Reader, io.Writer) {
	return listener, listener
}

//==============================================================================

// udpReqHandler is required to process client messages.
type udpReqHandler struct{}

// Read implements the udp.ReqHandler interface. It is provided a request
// value to popular and a io.Reader that was created in the Bind above.
func (udpReqHandler) Read(traceID string, reader io.Reader) (*net.UDPAddr, []byte, int, error) {
	log.Dev(traceID, "Read", "Started : Waiting For Data")

	listener := reader.(*net.UDPConn)

	// Each package is 20 bytes in lengrh.
	data := make([]byte, 20)
	length, udpAddr, err := listener.ReadFromUDP(data)
	if err != nil {
		log.Error(traceID, "Read", err, "Completed")
		return nil, nil, 0, err
	}

	log.Dev(traceID, "Read", "Completed : IP[%s] Length[%d]", udpAddr.String(), length)
	return udpAddr, data, length, nil
}

// Process is used to handle the processing of the message. This method
// is called on a routine from a pool of routines.
func (udpReqHandler) Process(traceID string, r *udp.Request) {
	log.Dev(traceID, "Process", "Started : IP[%s] Length[%d] ReadAt[%v]", r.UDPAddr.String(), r.Length, r.ReadAt)

	if r.Length != 20 {
		err := fmt.Errorf("Invalid package length of %d", r.Length)
		log.Error(traceID, "Process", err, "Completed")
		return
	}

	// Extract the header from the first 8 bytes.
	h := struct {
		Raw           []byte
		Length        int
		Version       uint8
		TransactionID uint8
		OpCode        uint8
		StatusCode    uint8
		StreamHandle  uint32
	}{
		Raw:           r.Data,
		Length:        r.Length,
		Version:       uint8(r.Data[0]),
		TransactionID: uint8(r.Data[1]),
		OpCode:        uint8(r.Data[2]),
		StatusCode:    uint8(r.Data[3]),
		StreamHandle:  uint32(binary.BigEndian.Uint32(r.Data[4:8])),
	}

	log.Dev(traceID, "Process", "%+v", h)

	resp := udp.Response{
		UDPAddr: r.UDPAddr,
		Data:    []byte("GOT IT"),
		Length:  6,
		Complete: func(rsp *udp.Response) {
			log.Dev(traceID, "Process", "*****************> %v", rsp)
		},
	}

	r.UDP.Do(traceID, &resp)

	log.Dev(traceID, "Process", "Completed")
}

//==============================================================================

type udpRespHandler struct{}

// Write is provided the user-defined writer and the data to write.
func (udpRespHandler) Write(traceID string, r *udp.Response, writer io.Writer) {
	log.Dev(traceID, "Write", "Started")

	listener := writer.(*net.UDPConn)
	listener.WriteToUDP(r.Data, r.UDPAddr)

	log.Dev(traceID, "Write", "Completed")
}
