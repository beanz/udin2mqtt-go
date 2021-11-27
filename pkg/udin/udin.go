package udin

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/acomagu/bufpipe"
	"github.com/tarm/serial"
)

type UdinCommand int

const (
	UdinQuery UdinCommand = iota
	UdinOn
	UdinOff
	UdinSet
	UdinStatus
	UdinInput
)

type UdinRequest struct {
	Command  UdinCommand
	Instance uint
}

func (r UdinRequest) String() string {
	switch r.Command {
	case UdinQuery:
		return "?"
	case UdinOn:
		return fmt.Sprintf("n%d", r.Instance)
	case UdinOff:
		return fmt.Sprintf("f%d", r.Instance)
	case UdinSet:
		return fmt.Sprintf("r%d", r.Instance)
	case UdinStatus:
		return fmt.Sprintf("s%d", r.Instance)
	case UdinInput:
		return fmt.Sprintf("i%d", r.Instance)
	}
	return "unknown command"
}

type MockUdin struct {
	model string
	r     io.ReadCloser
	w     io.WriteCloser
}

func (m *MockUdin) Write(b []byte) (int, error) {
	c, err := m.w.Write(b)
	if err != nil {
		return c, err
	}
	n, err := m.w.Write([]byte{'\n'})
	c += n
	if err != nil {
		return c, err
	}
	if len(b) == 2 && b[0] == '?' && b[1] == '\r' {
		n, err = m.w.Write([]byte(m.model + "\r\n"))
		c += n
		if err != nil {
			return c, err
		}
	}
	return c, nil
}

func (m *MockUdin) Read(p []byte) (int, error) {
	n, err := m.r.Read(p)
	return n, err
}

func (m *MockUdin) Close() error {
	err := m.r.Close()
	if err != nil {
		return err
	}
	return m.w.Close()
}

func NewUdinMock(dev string, logger *log.Logger) (*UdinDevice, error) {
	model := "UDIN-8R 8 x Relay V1.0"
	if s := strings.Split(dev, ":"); len(s) > 1 {
		model = s[1]
	}
	r, w := bufpipe.New(nil)
	mock := &MockUdin{model, r, w}
	return udinInit(dev, mock, model[:7], logger)
}

type UdinDevice struct {
	port      io.ReadWriteCloser
	reader    *bufio.Reader
	model     string
	name      string
	numRelays uint
	numInputs uint
	logger    *log.Logger
}

func udinInit(dev string, rwc io.ReadWriteCloser, name string, logger *log.Logger) (*UdinDevice, error) {
	udin := &UdinDevice{
		port:   rwc,
		reader: bufio.NewReader(rwc),
		name:   strings.ToLower(name),
		logger: logger,
	}
	m, err := udin.Send(UdinRequest{Command: UdinQuery})
	if err != nil {
		return nil, fmt.Errorf("failed to query udin device %s: %+v", dev, err)
	}
	switch m[0:7] {
	case "UDIN-8R":
		udin.numRelays = 8
	case "UDIN-44":
		udin.numRelays = 4
		udin.numInputs = 4
	case "UDIN-8I":
		udin.numInputs = 8
	default:
		return nil, fmt.Errorf("unsupported udin device %s: %v %s", dev, []byte(m[0:7]), m[0:7])
	}
	if logger != nil {
		logger.Printf("found device %s: %s\n", udin.name, udin.model)
	}
	return udin, nil
}

func NewUdinSerial(tty string, logger *log.Logger) (*UdinDevice, error) {
	c := &serial.Config{Name: tty, Baud: 9600}
	s, err := serial.OpenPort(c)
	if err != nil {
		return nil, err
	}
	return udinInit(tty, s, strings.TrimPrefix(tty, "/dev/"), logger)
}

func (u *UdinDevice) Send(r UdinRequest) (string, error) {
	cmd := r.String()
	_, err := u.port.Write([]byte(cmd + "\r"))
	if err != nil {
		return "", fmt.Errorf("udin write failed: %+v", err)
	}
	s, err := u.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("udin read failed: %+v", err)
	}
	if u.logger != nil {
		u.logger.Printf("read: %s %v\n", s[:len(s)-2], []byte(s))
	}
	if r.Command != UdinQuery {
		return s[:len(s)-2], nil
	}
	model, err := u.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("udin model read failed: %+v", err)
	}
	if u.logger != nil {
		u.logger.Printf("read model: %s %v\n",
			model[:len(model)-2], []byte(model))
	}
	u.model = model[:len(model)-2]
	return model[:len(model)-2], nil
}

func (u *UdinDevice) String() string {
	return fmt.Sprintf("%s: %s (r=%d i=%d)",
		u.Name(), u.Model(), u.NumRelays(), u.NumInputs())
}

func (u *UdinDevice) Close() error {
	return u.port.Close()
}

func (u *UdinDevice) Model() string {
	return u.model
}

func (u *UdinDevice) Name() string {
	return u.name
}

func (u *UdinDevice) NumRelays() uint {
	return u.numRelays
}

func (u *UdinDevice) NumInputs() uint {
	return u.numInputs
}

func NewUdin(dev string, logger *log.Logger) (*UdinDevice, error) {
	if strings.HasPrefix(dev, "mock") {
		return NewUdinMock(dev, logger)
	}
	return NewUdinSerial(dev, logger)
}

func (u *UdinDevice) On(r uint) error {
	if r > u.numRelays {
		return fmt.Errorf("invalid relay %d", r)
	}
	_, err := u.Send(UdinRequest{Command: UdinOn, Instance: r})
	return err
}

func (u *UdinDevice) Off(r uint) error {
	if r > u.numRelays {
		return fmt.Errorf("invalid relay %d", r)
	}
	_, err := u.Send(UdinRequest{Command: UdinOff, Instance: r})
	return err
}

func (u *UdinDevice) Pulse(r uint, d time.Duration) error {
	err := u.On(r)
	if err != nil {
		return err
	}
	time.Sleep(d)
	return u.Off(r)
}
