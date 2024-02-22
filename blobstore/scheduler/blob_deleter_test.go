// Copyright 2022 The CubeFS Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.

package scheduler

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/cubefs/cubefs/blobstore/common/counter"
	errcode "github.com/cubefs/cubefs/blobstore/common/errors"
	"github.com/cubefs/cubefs/blobstore/common/proto"
	"github.com/cubefs/cubefs/blobstore/common/recordlog"
	"github.com/cubefs/cubefs/blobstore/common/taskswitch"
	"github.com/cubefs/cubefs/blobstore/scheduler/base"
	"github.com/cubefs/cubefs/blobstore/scheduler/client"
	"github.com/cubefs/cubefs/blobstore/testing/mocks"
	"github.com/cubefs/cubefs/blobstore/util/taskpool"
)

func newDeleteTopicConsumer(t *testing.T) *deleteTopicConsumer {
	ctr := gomock.NewController(t)
	clusterMgrCli := NewMockClusterMgrAPI(ctr)
	clusterMgrCli.EXPECT().GetConfig(any, any).AnyTimes().Return("", nil)

	volCache := NewMockVolumeCache(ctr)
	volCache.EXPECT().Get(any).AnyTimes().DoAndReturn(
		func(vid proto.Vid) (*client.VolumeInfoSimple, error) {
			return &client.VolumeInfoSimple{Vid: vid}, nil
		},
	)

	switchMgr := taskswitch.NewSwitchMgr(clusterMgrCli)
	taskSwitch, err := switchMgr.AddSwitch(proto.TaskTypeBlobDelete.String())
	require.NoError(t, err)

	blobnodeCli := NewMockBlobnodeAPI(ctr)
	blobnodeCli.EXPECT().MarkDelete(any, any, any).AnyTimes().Return(nil)
	blobnodeCli.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

	producer := NewMockProducer(ctr)
	producer.EXPECT().SendMessage(any).AnyTimes().Return(nil)
	consumer := NewMockConsumer(ctr)

	delLogger := mocks.NewMockRecordLogEncoder(ctr)
	delLogger.EXPECT().Close().AnyTimes().Return(nil)
	delLogger.EXPECT().Encode(any).AnyTimes().Return(nil)
	tp := taskpool.New(2, 2)

	return &deleteTopicConsumer{
		taskSwitch:     taskSwitch,
		topicConsumers: []base.IConsumer{consumer},
		taskPool:       &tp,

		consumeIntervalMs: time.Duration(0),
		safeDelayTime:     time.Hour,
		volCache:          volCache,
		blobnodeCli:       blobnodeCli,
		failMsgSender:     producer,

		delSuccessCounter:    base.NewCounter(1, "delete", base.KindSuccess),
		delFailCounter:       base.NewCounter(1, "delete", base.KindFailed),
		errStatsDistribution: base.NewErrorStats(),
		delLogger:            delLogger,

		delSuccessCounterByMin: &counter.Counter{},
		delFailCounterByMin:    &counter.Counter{},
	}
}

func TestDeleteTopicConsumer(t *testing.T) {
	ctr := gomock.NewController(t)
	mockTopicConsumeDelete := newDeleteTopicConsumer(t)

	consumer := mockTopicConsumeDelete.topicConsumers[0].(*MockConsumer)
	consumer.EXPECT().CommitOffset(any).AnyTimes().Return(nil)

	{
		// nothing todo
		consumer.EXPECT().ConsumeMessages(any, any).Return([]*sarama.ConsumerMessage{})
		mockTopicConsumeDelete.consumeAndDelete(consumer, 0)
	}
	{
		// return one invalid message
		consumer.EXPECT().ConsumeMessages(any, any).DoAndReturn(
			func(ctx context.Context, msgCnt int) (msgs []*sarama.ConsumerMessage) {
				msg := proto.DeleteMsg{}
				msgByte, _ := json.Marshal(msg)
				kafkaMgs := &sarama.ConsumerMessage{
					Value: msgByte,
				}
				return []*sarama.ConsumerMessage{kafkaMgs}
			},
		)
		mockTopicConsumeDelete.consumeAndDelete(consumer, 1)
	}
	{
		// return 2 same messages and consume one time
		consumer.EXPECT().ConsumeMessages(any, any).DoAndReturn(
			func(ctx context.Context, msgCnt int) (msgs []*sarama.ConsumerMessage) {
				msg := proto.DeleteMsg{Bid: 1, Vid: 1, ReqId: "123456"}
				msgByte, _ := json.Marshal(msg)
				kafkaMgs := &sarama.ConsumerMessage{
					Value: msgByte,
				}
				return []*sarama.ConsumerMessage{kafkaMgs, kafkaMgs}
			},
		)
		mockTopicConsumeDelete.consumeAndDelete(consumer, 2)
	}
	{
		// return 2 diff messages adn consume success
		consumer.EXPECT().ConsumeMessages(any, any).DoAndReturn(
			func(ctx context.Context, msgCnt int) (msgs []*sarama.ConsumerMessage) {
				msg := proto.DeleteMsg{Bid: 2, Vid: 2, ReqId: "msg1"}
				msgByte, _ := json.Marshal(msg)
				kafkaMgs := &sarama.ConsumerMessage{
					Value: msgByte,
				}

				msg2 := proto.DeleteMsg{Bid: 1, Vid: 1, ReqId: "msg2"}
				msgByte2, _ := json.Marshal(msg2)
				kafkaMgs2 := &sarama.ConsumerMessage{
					Value: msgByte2,
				}
				return []*sarama.ConsumerMessage{kafkaMgs, kafkaMgs2}
			},
		)
		mockTopicConsumeDelete.consumeAndDelete(consumer, 2)
	}
	{
		// return one message and delete protected
		oldCache := mockTopicConsumeDelete.volCache
		volCache := NewMockVolumeCache(ctr)
		volCache.EXPECT().Get(any).AnyTimes().DoAndReturn(
			func(vid proto.Vid) (*client.VolumeInfoSimple, error) {
				return &client.VolumeInfoSimple{
					Vid:            vid,
					VunitLocations: []proto.VunitLocation{{Vuid: 1}},
				}, nil
			},
		)
		mockTopicConsumeDelete.volCache = volCache

		consumer.EXPECT().ConsumeMessages(any, any).DoAndReturn(
			func(ctx context.Context, msgCnt int) (msgs []*sarama.ConsumerMessage) {
				msg := proto.DeleteMsg{
					Bid:   2,
					Vid:   2,
					ReqId: "msg with volume return",
					Time:  time.Now().Unix() - 1,
				}
				msgByte, _ := json.Marshal(msg)
				kafkaMgs := &sarama.ConsumerMessage{
					Value: msgByte,
				}
				return []*sarama.ConsumerMessage{kafkaMgs}
			},
		)
		mockTopicConsumeDelete.safeDelayTime = 2 * time.Second
		mockTopicConsumeDelete.consumeAndDelete(consumer, 2)
		mockTopicConsumeDelete.volCache = oldCache
	}
	{
		// return one message and blobnode delete failed
		oldCache := mockTopicConsumeDelete.volCache
		volCache := NewMockVolumeCache(ctr)
		volCache.EXPECT().Get(any).AnyTimes().DoAndReturn(
			func(vid proto.Vid) (*client.VolumeInfoSimple, error) {
				return &client.VolumeInfoSimple{
					Vid:            vid,
					VunitLocations: []proto.VunitLocation{{Vuid: 1}},
				}, nil
			},
		)
		mockTopicConsumeDelete.volCache = volCache

		oldBlobNode := mockTopicConsumeDelete.blobnodeCli
		blobnodeCli := NewMockBlobnodeAPI(ctr)
		blobnodeCli.EXPECT().MarkDelete(any, any, any).AnyTimes().Return(errMock)
		mockTopicConsumeDelete.blobnodeCli = blobnodeCli

		consumer.EXPECT().ConsumeMessages(any, any).DoAndReturn(
			func(ctx context.Context, msgCnt int) (msgs []*sarama.ConsumerMessage) {
				msg := proto.DeleteMsg{Bid: 2, Vid: 2, ReqId: "delete failed"}
				msgByte, _ := json.Marshal(msg)
				kafkaMgs := &sarama.ConsumerMessage{
					Value: msgByte,
				}
				return []*sarama.ConsumerMessage{kafkaMgs}
			},
		)
		mockTopicConsumeDelete.consumeAndDelete(consumer, 2)
		mockTopicConsumeDelete.volCache = oldCache
		mockTopicConsumeDelete.blobnodeCli = oldBlobNode
	}
	{
		// return one message and blobnode return ErrDiskBroken
		oldCache := mockTopicConsumeDelete.volCache
		volCache := NewMockVolumeCache(ctr)
		volCache.EXPECT().Get(any).AnyTimes().DoAndReturn(
			func(vid proto.Vid) (*client.VolumeInfoSimple, error) {
				return &client.VolumeInfoSimple{
					Vid:            vid,
					VunitLocations: []proto.VunitLocation{{Vuid: 1}},
				}, nil
			},
		)
		volCache.EXPECT().Update(any).AnyTimes().DoAndReturn(
			func(vid proto.Vid) (*client.VolumeInfoSimple, error) {
				return &client.VolumeInfoSimple{
					Vid:            vid,
					VunitLocations: []proto.VunitLocation{{Vuid: 1}},
				}, nil
			},
		)
		mockTopicConsumeDelete.volCache = volCache

		oldBlobNode := mockTopicConsumeDelete.blobnodeCli
		blobnodeCli := NewMockBlobnodeAPI(ctr)
		blobnodeCli.EXPECT().MarkDelete(any, any, any).AnyTimes().Return(errcode.ErrDiskBroken)
		mockTopicConsumeDelete.blobnodeCli = blobnodeCli

		consumer.EXPECT().ConsumeMessages(any, any).DoAndReturn(
			func(ctx context.Context, msgCnt int) (msgs []*sarama.ConsumerMessage) {
				msg := proto.DeleteMsg{Bid: 2, Vid: 2, ReqId: "delete failed"}
				msgByte, _ := json.Marshal(msg)
				kafkaMgs := &sarama.ConsumerMessage{
					Value: msgByte,
				}
				return []*sarama.ConsumerMessage{kafkaMgs}
			},
		)
		mockTopicConsumeDelete.consumeAndDelete(consumer, 2)
		mockTopicConsumeDelete.volCache = oldCache
		mockTopicConsumeDelete.blobnodeCli = oldBlobNode
	}
	{
		// return one message, blobnode return ErrDiskBroken, and volCache update not eql
		oldCache := mockTopicConsumeDelete.volCache
		volCache := NewMockVolumeCache(ctr)
		volCache.EXPECT().Get(any).AnyTimes().DoAndReturn(
			func(vid proto.Vid) (*client.VolumeInfoSimple, error) {
				return &client.VolumeInfoSimple{
					Vid:            vid,
					VunitLocations: []proto.VunitLocation{{Vuid: 1}},
				}, nil
			},
		)
		volCache.EXPECT().Update(any).DoAndReturn(
			func(vid proto.Vid) (*client.VolumeInfoSimple, error) {
				return &client.VolumeInfoSimple{
					Vid:            vid,
					VunitLocations: []proto.VunitLocation{{Vuid: 1}, {Vuid: 2}},
				}, nil
			},
		)
		mockTopicConsumeDelete.volCache = volCache

		oldBlobNode := mockTopicConsumeDelete.blobnodeCli
		blobnodeCli := NewMockBlobnodeAPI(ctr)
		blobnodeCli.EXPECT().MarkDelete(any, any, any).AnyTimes().Return(errcode.ErrDiskBroken)
		mockTopicConsumeDelete.blobnodeCli = blobnodeCli

		consumer.EXPECT().ConsumeMessages(any, any).AnyTimes().DoAndReturn(
			func(ctx context.Context, msgCnt int) (msgs []*sarama.ConsumerMessage) {
				msg := proto.DeleteMsg{Bid: 2, Vid: 2, ReqId: "delete failed"}
				msgByte, _ := json.Marshal(msg)
				kafkaMgs := &sarama.ConsumerMessage{
					Value: msgByte,
				}
				return []*sarama.ConsumerMessage{kafkaMgs}
			},
		)
		mockTopicConsumeDelete.consumeAndDelete(consumer, 2)

		volCache.EXPECT().Update(any).AnyTimes().DoAndReturn(
			func(vid proto.Vid) (*client.VolumeInfoSimple, error) {
				return &client.VolumeInfoSimple{
					Vid:            vid,
					VunitLocations: []proto.VunitLocation{{Vuid: 2}},
				}, nil
			},
		)
		mockTopicConsumeDelete.volCache = volCache
		mockTopicConsumeDelete.consumeAndDelete(consumer, 2)

		mockTopicConsumeDelete.volCache = oldCache
		mockTopicConsumeDelete.blobnodeCli = oldBlobNode
	}
}

// comment temporary
func TestNewDeleteMgr(t *testing.T) {
	ctr := gomock.NewController(t)
	broker0 := NewBroker(t)
	defer broker0.Close()

	testDir, err := ioutil.TempDir(os.TempDir(), "delete_log")
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	blobCfg := &BlobDeleteConfig{
		ClusterID:            0,
		TaskPoolSize:         2,
		NormalHandleBatchCnt: 10,
		FailHandleBatchCnt:   10,
		DeleteLog: recordlog.Config{
			Dir:       testDir,
			ChunkBits: 22,
		},
		Kafka: BlobDeleteKafkaConfig{
			BrokerList: []string{broker0.Addr()},
			Normal: TopicConfig{
				Topic:      testTopic,
				Partitions: []int32{0},
			},
			Failed: TopicConfig{
				Topic:      testTopic,
				Partitions: []int32{0},
			},
			FailMsgSenderTimeoutMs: 0,
		},
	}

	clusterMgrCli := NewMockClusterMgrAPI(ctr)
	clusterMgrCli.EXPECT().GetConfig(any, any).AnyTimes().Return("", errMock)
	clusterMgrCli.EXPECT().GetConsumeOffset(any, any, any).AnyTimes().Return(int64(0), nil)
	clusterMgrCli.EXPECT().SetConsumeOffset(any, any, any, any).AnyTimes().Return(nil)

	volCache := NewMockVolumeCache(ctr)
	blobnodeCli := NewMockBlobnodeAPI(ctr)
	switchMgr := taskswitch.NewSwitchMgr(clusterMgrCli)

	service, err := NewBlobDeleteMgr(blobCfg, volCache, blobnodeCli, switchMgr, clusterMgrCli)
	require.NoError(t, err)

	// run task
	service.RunTask()

	// get stats
	service.GetTaskStats()
	service.GetErrorStats()
}
