package container

import (
	"bytes"
	proto "code.google.com/p/goprotobuf/proto"
	"errors"
	"fmt"
	"github.com/cloudfoundry/gordon"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var _ = Describe("Container", func() {
	Context("Setup", func() {
		It("Sets the handle and ports", func() {
			container := FakeContainer(nil, nil)

			container.Setup("handle", 1, 2)
			Expect(container.Handle()).To(Equal("handle"))
			Expect(container.NetworkPort(HOST_PORT)).To(Equal(uint32(1)))
			Expect(container.NetworkPort(CONTAINER_PORT)).To(Equal(uint32(2)))
		})
	})
	Context("Create", func() {
		It("creates a container", func() {
			buffer := bytes.NewBuffer([]byte{})
			hostPort, containerPort := uint32(5000), uint32(6000)
			container := FakeContainer(
				messages(&warden.CreateResponse{Handle: proto.String("foo")},
					&warden.LimitDiskResponse{},
					&warden.LimitMemoryResponse{},
					&warden.NetInResponse{HostPort: &hostPort, ContainerPort: &containerPort}),
				buffer)

			err := container.Create(nil, 16*1024*1024, 8*1024, true)
			Expect(err).To(BeNil())
			Expect(container.Handle()).NotTo(Equal(""))
		})

		It("creates a container without a network", func() {
			buffer := bytes.NewBuffer([]byte{})
			container := FakeContainer(
				messages(&warden.CreateResponse{Handle: proto.String("foo")},
					&warden.LimitDiskResponse{},
					&warden.LimitMemoryResponse{},
				),
				buffer)
			err := container.Create(nil, 16*1024*1024, 8*1024, false)
			Expect(err).To(BeNil())
		})

		It("destroys a container, if limits or network fail", func() {
			buffer := bytes.NewBuffer(make([]byte, 0, 256))
			container := FakeContainer(
				messages(&warden.CreateResponse{Handle: proto.String("foo")},
					&warden.LimitMemoryResponse{},
				),
				buffer)

			err := container.Create(nil, 16*1024*1024, 8*1024, true)
			_, err = parseResponse(buffer, &warden.CreateRequest{})
			Expect(err).To(BeNil())
			_, err = parseResponse(buffer, &warden.LimitDiskRequest{})
			Expect(err).To(BeNil())
			_, err = parseResponse(buffer, &warden.DestroyRequest{})
			Expect(err).To(BeNil())
		})

	})

	Context("RunScript", func() {
		It("Runs and no error when exit status = 0", func() {
			buffer := bytes.NewBuffer([]byte{})
			status := uint32(0)
			container := FakeContainer(
				messages(&warden.RunResponse{ExitStatus: &status}),
				buffer)

			rsp, err := container.RunScript("doIt")
			Expect(err).To(BeNil())
			Expect(*rsp.ExitStatus).To(Equal(status))
		})

		It("Runs and err when exit status != 0", func() {
			buffer := bytes.NewBuffer([]byte{})
			status := uint32(1)
			container := FakeContainer(
				messages(&warden.RunResponse{ExitStatus: &status}),
				buffer)

			rsp, err := container.RunScript("doIt")
			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("exited with status"))
			Expect(*rsp.ExitStatus).To(Equal(status))
		})
	})

	Context("Spawn", func() {
		It("Spawns", func() {
			buffer := bytes.NewBuffer([]byte{})
			jobId := uint32(1234)
			container := FakeContainer(
				messages(&warden.SpawnResponse{JobId: &jobId}),
				buffer)

			rsp, err := container.Spawn("doIt", 512, 512, true)
			Expect(err).To(BeNil())
			Expect(*rsp.JobId).To(Equal(jobId))
		})
	})

	PContext("Stop", func() {
	})

	PContext("Destroy", func() {
	})
	PContext("CloseAllConnections", func() {
	})

	PContext("Link", func() {
	})

	PContext("CopyOut", func() {
	})
})

type FakeConnectionProvider struct {
	ReadBuffer  *bytes.Buffer
	WriteBuffer *bytes.Buffer
}

func (c *FakeConnectionProvider) ProvideConnection() (*warden.Connection, error) {
	return warden.NewConnection(
		&fakeConn{
			ReadBuffer:  c.ReadBuffer,
			WriteBuffer: c.WriteBuffer,
		},
	), nil
}

func FakeContainer(read, write *bytes.Buffer) Container {
	return New(&FakeConnectionProvider{read, write})
}

type fakeConn struct {
	ReadBuffer  *bytes.Buffer
	WriteBuffer *bytes.Buffer
	WriteChan   chan string
	Closed      bool
}

func (f *fakeConn) Read(b []byte) (n int, err error) {
	if f.Closed {
		return 0, errors.New("buffer closed")
	}

	return f.ReadBuffer.Read(b)
}

func (f *fakeConn) Write(b []byte) (n int, err error) {
	if f.Closed {
		return 0, errors.New("buffer closed")
	}

	if f.WriteChan != nil {
		f.WriteChan <- string(b)
	}

	return f.WriteBuffer.Write(b)
}

func (f *fakeConn) Close() error {
	f.Closed = true
	return nil
}

func (f *fakeConn) SetDeadline(time.Time) error {
	return nil
}

func (f *fakeConn) SetReadDeadline(time.Time) error {
	return nil
}

func (f *fakeConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (f *fakeConn) LocalAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:4222")
	return addr
}

func (f *fakeConn) RemoteAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:65525")
	return addr
}

func messages(msgs ...proto.Message) *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})

	for _, msg := range msgs {
		payload, err := proto.Marshal(msg)
		if err != nil {
			panic(err.Error())
		}

		message := &warden.Message{
			Type:    warden.Message_Type(message2type(msg)).Enum(),
			Payload: payload,
		}

		messagePayload, err := proto.Marshal(message)
		if err != nil {
			panic("failed to marshal message")
		}

		buf.Write([]byte(fmt.Sprintf("%d\r\n%s\r\n", len(messagePayload), messagePayload)))
	}

	return buf
}

const (
	Request  = "Request"
	Response = "Response"
)

func message2type(msg proto.Message) int32 {
	t := reflect.TypeOf(msg)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	var name = t.Name()
	if strings.HasSuffix(name, Request) {
		name = name[:len(name)-len(Request)]
	} else if strings.HasSuffix(name, Response) {
		name = name[:len(name)-len(Response)]
	}

	return warden.Message_Type_value[name]
}

func parseResponse(b *bytes.Buffer, response proto.Message) (proto.Message, error) {
	msgHeader, err := b.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	msgLen, err := strconv.ParseUint(string(msgHeader[0:len(msgHeader)-2]), 10, 0)
	if err != nil {
		return nil, err
	}

	payload := make([]byte, msgLen)
	_, err = b.Read(payload)
	if err != nil {
		return nil, err
	}

	// skip CRLF
	_, err = b.ReadByte()
	if err == nil {
		_, err = b.ReadByte()
	}
	if err != nil {
		return nil, err
	}

	message := &warden.Message{}
	err = proto.Unmarshal(payload, message)
	if err != nil {
		return nil, err
	}

	response_type := warden.Message_Type(message2type(response))
	if message.GetType() != response_type {
		return nil, errors.New(
			fmt.Sprintf(
				"expected message type %s, got %s\n",
				response_type.String(),
				message.GetType().String(),
			),
		)
	}

	err = proto.Unmarshal(message.GetPayload(), response)
	return response, err
}
