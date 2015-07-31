package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/golang/glog"
	mesos "github.com/mesos/mesos-go/mesosproto"
)

const (
	PAILER_CHUNK_SIZE = 50000
	PAILER_INTERVAL   = 1 * time.Second
	PAILER_STOP_DELAY = 3 * time.Second
)

type Pailer struct {
	BaseUrl  string
	BasePath string
	Path     string
	Offset   int
	C        chan string
	running  bool
	ticker   *time.Ticker
	wait     chan bool
}

type update struct {
	Offset int
	Data   string
}

func NewPailer(master *string, slaveId *mesos.SlaveID, frameworkId *mesos.FrameworkID, taskId *mesos.TaskID, path string) (*Pailer, error) {
	ms, err := FetchMasterState(master)
	if err != nil {
		return nil, err
	}

	slv := ms.GetSlave(slaveId.GetValue())
	if slv == nil {
		return nil, fmt.Errorf("Unable to find slave with id %s", slaveId.GetValue())
	}
	ss, err := slv.GetState()
	if err != nil {
		return nil, err
	}

	d := ss.GetDirectory(frameworkId.GetValue(), taskId.GetValue())
	if d == nil {
		return nil, fmt.Errorf("Unable to find directory for framework %s with task %s", frameworkId.GetValue(), taskId.GetValue())
	}

	return &Pailer{
		BaseUrl:  fmt.Sprintf("%s/files/read.json", slv.GetUrl()),
		BasePath: *d,
		Path:     path,
		Offset:   0,
		C:        make(chan string),
		running:  false,
		wait:     make(chan bool),
	}, nil
}

func (p *Pailer) fetch() error {
	url := fmt.Sprintf("%s?length=%d&offset=%d&path=%s",
		p.BaseUrl,
		PAILER_CHUNK_SIZE,
		p.Offset,
		url.QueryEscape(fmt.Sprintf("%s/%s", p.BasePath, p.Path)))
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	d := json.NewDecoder(resp.Body)
	var u update
	if err := d.Decode(&u); err != nil {
		return err
	}

	p.Offset = u.Offset + len(u.Data)
	p.C <- u.Data
	return nil
}

func (p *Pailer) tick() {
	for p.running {
		if err := p.fetch(); err != nil {
			log.Errorf("Fetching pailer update failed: %s", err)
		}
		<-p.ticker.C
	}
	close(p.C)
	p.wait <- true
}

func (p *Pailer) Start() {
	log.Infof("Start pailing: %s %s/%s", p.BaseUrl, p.BasePath, p.Path)
	p.running = true
	p.ticker = time.NewTicker(PAILER_INTERVAL)
	go p.tick()
}

func (p *Pailer) Stop() {
	go p.stop()
}

func (p *Pailer) Wait() {
	log.Infof("Waiting for pailer: %s %s/%s", p.BaseUrl, p.BasePath, p.Path)
	<-p.wait
}

func (p *Pailer) stop() {
	time.Sleep(PAILER_STOP_DELAY)
	log.Infof("Stopping pailer: %s %s/%s", p.BaseUrl, p.BasePath, p.Path)
	p.running = false
}
