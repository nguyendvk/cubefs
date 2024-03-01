// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/cubefs/cubefs/blobstore/scheduler/base (interfaces: KafkaConsumer,GroupConsumer,IProducer)

// Package scheduler is a generated GoMock package.
package scheduler

import (
	reflect "reflect"

	sarama "github.com/Shopify/sarama"
	proto "github.com/cubefs/cubefs/blobstore/common/proto"
	base "github.com/cubefs/cubefs/blobstore/scheduler/base"
	gomock "github.com/golang/mock/gomock"
)

// MockKafkaConsumer is a mock of KafkaConsumer interface.
type MockKafkaConsumer struct {
	ctrl     *gomock.Controller
	recorder *MockKafkaConsumerMockRecorder
}

// MockKafkaConsumerMockRecorder is the mock recorder for MockKafkaConsumer.
type MockKafkaConsumerMockRecorder struct {
	mock *MockKafkaConsumer
}

// NewMockKafkaConsumer creates a new mock instance.
func NewMockKafkaConsumer(ctrl *gomock.Controller) *MockKafkaConsumer {
	mock := &MockKafkaConsumer{ctrl: ctrl}
	mock.recorder = &MockKafkaConsumerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockKafkaConsumer) EXPECT() *MockKafkaConsumerMockRecorder {
	return m.recorder
}

// StartKafkaConsumer mocks base method.
func (m *MockKafkaConsumer) StartKafkaConsumer(arg0 proto.TaskType, arg1 string, arg2 func(*sarama.ConsumerMessage, base.ConsumerPause) bool) (base.GroupConsumer, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StartKafkaConsumer", arg0, arg1, arg2)
	ret0, _ := ret[0].(base.GroupConsumer)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StartKafkaConsumer indicates an expected call of StartKafkaConsumer.
func (mr *MockKafkaConsumerMockRecorder) StartKafkaConsumer(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StartKafkaConsumer", reflect.TypeOf((*MockKafkaConsumer)(nil).StartKafkaConsumer), arg0, arg1, arg2)
}

// MockGroupConsumer is a mock of GroupConsumer interface.
type MockGroupConsumer struct {
	ctrl     *gomock.Controller
	recorder *MockGroupConsumerMockRecorder
}

// MockGroupConsumerMockRecorder is the mock recorder for MockGroupConsumer.
type MockGroupConsumerMockRecorder struct {
	mock *MockGroupConsumer
}

// NewMockGroupConsumer creates a new mock instance.
func NewMockGroupConsumer(ctrl *gomock.Controller) *MockGroupConsumer {
	mock := &MockGroupConsumer{ctrl: ctrl}
	mock.recorder = &MockGroupConsumerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockGroupConsumer) EXPECT() *MockGroupConsumerMockRecorder {
	return m.recorder
}

// Stop mocks base method.
func (m *MockGroupConsumer) Stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop.
func (mr *MockGroupConsumerMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockGroupConsumer)(nil).Stop))
}

// MockProducer is a mock of IProducer interface.
type MockProducer struct {
	ctrl     *gomock.Controller
	recorder *MockProducerMockRecorder
}

// MockProducerMockRecorder is the mock recorder for MockProducer.
type MockProducerMockRecorder struct {
	mock *MockProducer
}

// NewMockProducer creates a new mock instance.
func NewMockProducer(ctrl *gomock.Controller) *MockProducer {
	mock := &MockProducer{ctrl: ctrl}
	mock.recorder = &MockProducerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockProducer) EXPECT() *MockProducerMockRecorder {
	return m.recorder
}

// SendMessage mocks base method.
func (m *MockProducer) SendMessage(arg0 []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMessage", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMessage indicates an expected call of SendMessage.
func (mr *MockProducerMockRecorder) SendMessage(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessage", reflect.TypeOf((*MockProducer)(nil).SendMessage), arg0)
}

// SendMessages mocks base method.
func (m *MockProducer) SendMessages(arg0 [][]byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMessages", arg0)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendMessages indicates an expected call of SendMessages.
func (mr *MockProducerMockRecorder) SendMessages(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessages", reflect.TypeOf((*MockProducer)(nil).SendMessages), arg0)
}
