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

package controller

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/cubefs/cubefs/blobstore/api/blobnode"
	"github.com/cubefs/cubefs/blobstore/api/clustermgr"
	"github.com/cubefs/cubefs/blobstore/common/proto"
	"github.com/cubefs/cubefs/blobstore/common/trace"
	"github.com/cubefs/cubefs/blobstore/util/defaulter"
	"github.com/cubefs/cubefs/blobstore/util/errors"
	"github.com/cubefs/cubefs/blobstore/util/log"
)

const (
	_diskHostServicePrefix = "diskhost"

	// default service punish check valid interval
	defaultServicePinishValidIntervalS int = 30
	// default service punish check threshold
	defaultServicePinishThreshold uint32 = 3
)

// HostIDC item of host with idc
type HostIDC struct {
	Host     string
	IDC      string
	Punished bool
}

// ServiceController support for both data node discovery and normal service discovery
type ServiceController interface {
	// GetServiceHost return an available service host
	GetServiceHost(ctx context.Context, name string) (host string, err error)
	// GetServiceHosts return all available service random sorted hosts
	GetServiceHosts(ctx context.Context, name string) (hosts []string, err error)
	// GetDiskHost return an disk's related data node host
	GetDiskHost(ctx context.Context, diskID proto.DiskID) (hostIDC *HostIDC, err error)
	// PunishService will punish an service host for an punishTimeSec interval
	PunishService(ctx context.Context, service, host string, punishTimeSec int)
	// PunishServiceWithThreshold will punish an service host for
	// an punishTimeSec interval if service failed times satisfied with threshold during some interval time
	PunishServiceWithThreshold(ctx context.Context, service, host string, punishTimeSec int)
	// PunishDisk will punish a disk host for an punishTimeSec interval
	PunishDisk(ctx context.Context, diskID proto.DiskID, punishTimeSec int)
	// PunishDiskWithThreshold will punish a disk host for
	// an punishTimeSec interval if disk host failed times satisfied with threshold
	PunishDiskWithThreshold(ctx context.Context, diskID proto.DiskID, punishTimeSec int)
}

type (
	serviceList []*hostItem
	serviceMap  map[string]*atomic.Value
)

// hostItem represent a service or host item info
type hostItem struct {
	host string
	idc  string

	// punish time record the punish end time unix of host item
	punishTimeUnix int64
	// modify time record the last modify time unix of host item
	lastModifyTime int64
	// failedTimes record the service host failed times during some interval
	failedTimes uint32
}

func (h *hostItem) isPunish() bool {
	return time.Since(time.Unix(atomic.LoadInt64(&h.punishTimeUnix), 0)) < 0
}

// ServiceConfig service config
type ServiceConfig struct {
	ClusterID                   proto.ClusterID
	IDC                         string
	ReloadSec                   int
	ServicePunishThreshold      uint32
	ServicePunishValidIntervalS int
}

type serviceControllerImpl struct {
	// allServices hold all disk/service host map, use for quickly find out
	allServices  sync.Map
	serviceHosts serviceMap

	group        singleflight.Group
	serviceLocks map[string]*sync.RWMutex
	cmClient     clustermgr.APIAccess

	config ServiceConfig
}

// NewServiceController returns a service controller
func NewServiceController(cfg ServiceConfig, cmCli clustermgr.APIAccess) (ServiceController, error) {
	defaulter.Equal(&cfg.ServicePunishThreshold, defaultServicePinishThreshold)
	defaulter.LessOrEqual(&cfg.ServicePunishValidIntervalS, defaultServicePinishValidIntervalS)

	controller := &serviceControllerImpl{
		serviceHosts: serviceMap{
			proto.ServiceNameProxy: &atomic.Value{},
		},
		cmClient: cmCli,
		serviceLocks: map[string]*sync.RWMutex{
			proto.ServiceNameProxy: {},
		},
		config: cfg,
	}

	err := controller.load(cfg.ClusterID, cfg.IDC)
	if err != nil {
		return nil, errors.Base(err, "load service failed")
	}

	if cfg.ReloadSec <= 0 {
		cfg.ReloadSec = 10
	}
	tick := time.NewTicker(time.Duration(cfg.ReloadSec) * time.Second)
	go func() {
		defer tick.Stop()
		for range tick.C {
			if err := controller.load(cfg.ClusterID, cfg.IDC); err != nil {
				log.Warn("load timer error", err)
			}
		}
	}()
	return controller, nil
}

// load initial all service and service hosts
func (s *serviceControllerImpl) load(cid proto.ClusterID, idc string) error {
	span, ctx := trace.StartSpanFromContext(context.Background(), "access_cluster_service")
	span.Debug("service loader for cluster:", cid)

	serviceName := proto.ServiceNameProxy
	service, err := s.cmClient.GetService(ctx, clustermgr.GetServiceArgs{Name: serviceName})
	if err != nil {
		span.Warn("get service from cluster manager failed", err)
		return err
	}

	span.Debugf("found %d server nodes of %s in the cluster", len(service.Nodes), serviceName)
	hostItems := make(serviceList, 0, len(service.Nodes))
	for _, node := range service.Nodes {
		if node.Idc != idc {
			continue
		}
		hostItems = append(hostItems, &hostItem{idc: node.Idc, host: node.Host})
	}
	if len(hostItems) > 0 {
		for _, item := range hostItems {
			s.allServices.Store(serviceName+item.host, item)
			span.Debugf("store node %+v", item)
		}
		s.serviceHosts[serviceName].Store(hostItems)
	}
	return nil
}

// GetServiceHost return an available service host
func (s *serviceControllerImpl) GetServiceHost(ctx context.Context, name string) (host string, err error) {
	serviceList, ok := s.serviceHosts[name].Load().(serviceList)
	if !ok {
		return "", errors.Newf("not found host of %s", name)
	}

	lock := s.getServiceLock(name)
	idx := 0

RETRY:

	lock.RLock()
	length := len(serviceList)
	if length == 0 {
		lock.RUnlock()
		return "", errors.Newf("no any host of %s", name)
	}
	idx = rand.Intn(length)

	item := serviceList[idx]
	lock.RUnlock()

	if !item.isPunish() {
		return item.host, nil
	}

	lock.Lock()
	// double check
	v := serviceList[idx]
	// if serviceList[idx] still equal to item, then remove it
	if v == item {
		serviceList = append(serviceList[:idx], serviceList[idx+1:]...)
	}
	s.serviceHosts[name].Store(serviceList)
	lock.Unlock()

	goto RETRY
}

// GetServiceHosts return all available random-sorted hosts of service
func (s *serviceControllerImpl) GetServiceHosts(ctx context.Context, name string) (hosts []string, err error) {
	serviceList, ok := s.serviceHosts[name].Load().(serviceList)
	if !ok {
		return nil, errors.Newf("not found host of %s", name)
	}

	lock := s.getServiceLock(name)

	lock.RLock()
	length := len(serviceList)
	if length == 0 {
		lock.RUnlock()
		return nil, errors.Newf("no any host of %s", name)
	}

	hosts = make([]string, 0, length)
	for _, item := range serviceList {
		if !item.isPunish() {
			hosts = append(hosts, item.host)
		}
	}
	lock.RUnlock()

	if len(hosts) == 0 {
		return nil, errors.Newf("no available host of %s", name)
	}

	rand.Shuffle(len(hosts), func(i, j int) {
		hosts[i], hosts[j] = hosts[j], hosts[i]
	})
	return hosts, nil
}

// GetDiskHost return an disk's related data node host
func (s *serviceControllerImpl) GetDiskHost(ctx context.Context, diskID proto.DiskID) (*HostIDC, error) {
	span := trace.SpanFromContextSafe(ctx)

	v, ok := s.allServices.Load(_diskHostServicePrefix + (diskID.ToString()))
	if ok {
		item := v.(*hostItem)
		return &HostIDC{
			Host:     item.host,
			IDC:      item.idc,
			Punished: item.isPunish(),
		}, nil
	}
	ret, err, _ := s.group.Do("get-diskinfo-"+diskID.ToString(), func() (interface{}, error) {
		diskInfo, err := s.cmClient.DiskInfo(ctx, diskID)
		if err != nil {
			return nil, err
		}
		return diskInfo, nil
	})
	if err != nil {
		span.Error("can't get disk host from clustermgr", err)
		return nil, errors.Base(err, "get disk info", diskID)
	}
	diskInfo := ret.(*blobnode.DiskInfo)

	item := &hostItem{host: diskInfo.Host, idc: diskInfo.Idc}
	s.allServices.Store(_diskHostServicePrefix+(diskInfo.DiskID.ToString()), item)
	return &HostIDC{
		Host:     item.host,
		IDC:      item.idc,
		Punished: item.isPunish(),
	}, nil
}

// PunishService will punish an service host for an punishTimeSec interval
func (s *serviceControllerImpl) PunishService(ctx context.Context, service, host string, punishTimeSec int) {
	v, ok := s.allServices.Load(service + host)
	if !ok {
		panic(fmt.Sprintf("can't find host in all services map, %s-%s", service, host))
	}
	item := v.(*hostItem)

	// atomic set item's punish time unix
	atomic.StoreInt64(&item.punishTimeUnix, time.Now().Add(time.Duration(punishTimeSec)*time.Second).Unix())
}

// PunishDisk will punish a disk host for an punishTimeSec interval
func (s *serviceControllerImpl) PunishDisk(ctx context.Context, diskID proto.DiskID, punishTimeSec int) {
	s.PunishService(ctx, _diskHostServicePrefix, diskID.ToString(), punishTimeSec)
}

// PunishDiskWithThreshold will punish a disk host for
// an punishTimeSec interval if disk host failed times satisfied with threshold
func (s *serviceControllerImpl) PunishDiskWithThreshold(ctx context.Context, diskID proto.DiskID, punishTimeSec int) {
	s.PunishServiceWithThreshold(ctx, _diskHostServicePrefix, diskID.ToString(), punishTimeSec)
}

// PunishServiceWithThreshold will punish an service host for
// an punishTimeSec interval if service failed times satisfied with threshold
func (s *serviceControllerImpl) PunishServiceWithThreshold(ctx context.Context, service, host string, punishTimeSec int) {
	v, ok := s.allServices.Load(service + host)
	if !ok {
		panic(fmt.Sprintf("can't can host in all services map, %s-%s", service, host))
	}
	item := v.(*hostItem)
	new := atomic.AddUint32(&item.failedTimes, 1)
	// failedTimes larger than threshold, then check the lastModifyTime
	if new >= s.config.ServicePunishThreshold {
		if time.Since(time.Unix(atomic.LoadInt64(&item.lastModifyTime), 0)) < time.Duration(s.config.ServicePunishValidIntervalS)*time.Second {
			s.PunishService(ctx, service, host, punishTimeSec)
			return
		}
		atomic.AddUint32(&item.failedTimes, -(new - 1))
	}
	atomic.StoreInt64(&item.lastModifyTime, time.Now().Unix())
}

func (s *serviceControllerImpl) getServiceLock(name string) *sync.RWMutex {
	return s.serviceLocks[name]
}
