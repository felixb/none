package main

import (
	"encoding/binary"
	"fmt"
	"net"

	log "github.com/golang/glog"
	"github.com/mesos/mesos-go/detector"
	"github.com/mesos/mesos-go/detector/zoo"
	mesos "github.com/mesos/mesos-go/mesosproto"
)

type LeaderDetector interface {
	Detect(*string) (*string, error)
}

type ZkLeaderDetector struct {
	newLeader chan *string
}

func NewZkLeaderDetector() *ZkLeaderDetector {
	return &ZkLeaderDetector{
		newLeader: make(chan *string, 1),
	}
}

func (d *ZkLeaderDetector) Detect(zkUrl *string) (*string, error) {
	ld, err := zoo.NewMasterDetector(*zkUrl)
	if err != nil {
		return nil, fmt.Errorf("Failed to create master detector: %v", err)
	}
	if err := ld.Detect(detector.OnMasterChanged(d.onLeaderChange)); err != nil {
		return nil, fmt.Errorf("Failed to initialize master detector: %v", err)
	}

	// wait for callback to write new leader into this private channel
	return <-d.newLeader, nil
}

func (d *ZkLeaderDetector) onLeaderChange(info *mesos.MasterInfo) {
	if info == nil {
		log.Errorln("No leader available in Zookeeper")
	} else {
		leader := ""
		if host := info.GetHostname(); host != "" {
			leader = host
		} else {
			// unpack IPv4
			octets := make([]byte, 4, 4)
			binary.BigEndian.PutUint32(octets, info.GetIp())
			ipv4 := net.IP(octets)
			leader = ipv4.String()
		}
		leader = fmt.Sprintf("%s:%d", leader, info.GetPort())
		log.Infoln("New master in Zookeeper", leader)
		d.newLeader <- &leader
	}
}
