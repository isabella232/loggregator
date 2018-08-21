// This file was generated by github.com/nelsam/hel.  Do not
// edit this code by hand unless you *really* know what you're
// doing.  Expect any changes made manually to be overwritten
// the next time hel regenerates this file.

package proxy_test

import (
	"time"

	"code.cloudfoundry.org/loggregator/plumbing"
	"golang.org/x/net/context"
)

type mockPlumbingReceiver struct {
	RecvCalled chan bool
	RecvOutput struct {
		Ret0 chan *plumbing.Response
		Ret1 chan error
	}
}

func newMockPlumbingReceiver() *mockPlumbingReceiver {
	m := &mockPlumbingReceiver{}
	m.RecvCalled = make(chan bool, 100)
	m.RecvOutput.Ret0 = make(chan *plumbing.Response, 100)
	m.RecvOutput.Ret1 = make(chan error, 100)
	return m
}
func (m *mockPlumbingReceiver) Recv() (*plumbing.Response, error) {
	m.RecvCalled <- true
	return <-m.RecvOutput.Ret0, <-m.RecvOutput.Ret1
}

type mockReceiver struct {
	RecvCalled chan bool
	RecvOutput struct {
		Ret0 chan []byte
		Ret1 chan error
	}
}

func newMockReceiver() *mockReceiver {
	m := &mockReceiver{}
	m.RecvCalled = make(chan bool, 100)
	m.RecvOutput.Ret0 = make(chan []byte, 100)
	m.RecvOutput.Ret1 = make(chan error, 100)
	return m
}
func (m *mockReceiver) Recv() ([]byte, error) {
	m.RecvCalled <- true
	return <-m.RecvOutput.Ret0, <-m.RecvOutput.Ret1
}

type mockGrpcConnector struct {
	SubscribeCalled chan bool
	SubscribeInput  struct {
		Ctx chan context.Context
		Req chan *plumbing.SubscriptionRequest
	}
	SubscribeOutput struct {
		Ret0 chan func() ([]byte, error)
		Ret1 chan error
	}
	ContainerMetricsCalled chan bool
	ContainerMetricsInput  struct {
		Ctx   chan context.Context
		AppID chan string
	}
	ContainerMetricsOutput struct {
		Ret0 chan [][]byte
	}
}

func newMockGrpcConnector() *mockGrpcConnector {
	m := &mockGrpcConnector{}
	m.SubscribeCalled = make(chan bool, 100)
	m.SubscribeInput.Ctx = make(chan context.Context, 100)
	m.SubscribeInput.Req = make(chan *plumbing.SubscriptionRequest, 100)
	m.SubscribeOutput.Ret0 = make(chan func() ([]byte, error), 100)
	m.SubscribeOutput.Ret1 = make(chan error, 100)
	m.ContainerMetricsCalled = make(chan bool, 100)
	m.ContainerMetricsInput.Ctx = make(chan context.Context, 100)
	m.ContainerMetricsInput.AppID = make(chan string, 100)
	m.ContainerMetricsOutput.Ret0 = make(chan [][]byte, 100)
	return m
}
func (m *mockGrpcConnector) Subscribe(ctx context.Context, req *plumbing.SubscriptionRequest) (func() ([]byte, error), error) {
	m.SubscribeCalled <- true
	m.SubscribeInput.Ctx <- ctx
	m.SubscribeInput.Req <- req
	return <-m.SubscribeOutput.Ret0, <-m.SubscribeOutput.Ret1
}
func (m *mockGrpcConnector) ContainerMetrics(ctx context.Context, appID string) [][]byte {
	m.ContainerMetricsCalled <- true
	m.ContainerMetricsInput.Ctx <- ctx
	m.ContainerMetricsInput.AppID <- appID
	return <-m.ContainerMetricsOutput.Ret0
}

type mockContext struct {
	DeadlineCalled chan bool
	DeadlineOutput struct {
		Deadline chan time.Time
		Ok       chan bool
	}
	DoneCalled chan bool
	DoneOutput struct {
		Ret0 chan (<-chan struct{})
	}
	ErrCalled chan bool
	ErrOutput struct {
		Ret0 chan error
	}
	ValueCalled chan bool
	ValueInput  struct {
		Key chan interface{}
	}
	ValueOutput struct {
		Ret0 chan interface{}
	}
}

func newMockContext() *mockContext {
	m := &mockContext{}
	m.DeadlineCalled = make(chan bool, 100)
	m.DeadlineOutput.Deadline = make(chan time.Time, 100)
	m.DeadlineOutput.Ok = make(chan bool, 100)
	m.DoneCalled = make(chan bool, 100)
	m.DoneOutput.Ret0 = make(chan (<-chan struct{}), 100)
	m.ErrCalled = make(chan bool, 100)
	m.ErrOutput.Ret0 = make(chan error, 100)
	m.ValueCalled = make(chan bool, 100)
	m.ValueInput.Key = make(chan interface{}, 100)
	m.ValueOutput.Ret0 = make(chan interface{}, 100)
	return m
}
func (m *mockContext) Deadline() (deadline time.Time, ok bool) {
	m.DeadlineCalled <- true
	return <-m.DeadlineOutput.Deadline, <-m.DeadlineOutput.Ok
}
func (m *mockContext) Done() <-chan struct{} {
	m.DoneCalled <- true
	return <-m.DoneOutput.Ret0
}
func (m *mockContext) Err() error {
	m.ErrCalled <- true
	return <-m.ErrOutput.Ret0
}
func (m *mockContext) Value(key interface{}) interface{} {
	m.ValueCalled <- true
	m.ValueInput.Key <- key
	return <-m.ValueOutput.Ret0
}

type mockHealth struct {
	SetCalled chan bool
	SetInput  struct {
		Name  chan string
		Value chan float64
	}
	IncCalled chan bool
	IncInput  struct {
		Name chan string
	}
}

func newMockHealth() *mockHealth {
	m := &mockHealth{}
	m.SetCalled = make(chan bool, 100)
	m.SetInput.Name = make(chan string, 100)
	m.SetInput.Value = make(chan float64, 100)
	m.IncCalled = make(chan bool, 100)
	m.IncInput.Name = make(chan string, 100)
	return m
}

func (m *mockHealth) Set(name string, value float64) {
	m.SetCalled <- true
	m.SetInput.Name <- name
	m.SetInput.Value <- value
}

func (m *mockHealth) Inc(name string) {
	m.IncCalled <- true
	m.IncInput.Name <- name
}
