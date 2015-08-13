package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	log "github.com/golang/glog"
)

type Downloader struct {
	BaseUrl  string
	BasePath string
	Path     string
	writer   io.Writer
}

// get a downloader pointing to task's file
func NewDownloader(w io.Writer, master *string, command *Command, path string) (*Downloader, error) {
	if w == nil {
		return nil, fmt.Errorf("w must not be nil")
	}

	su, d, err := FetchSlaveDirInfo(master, command)
	if err != nil {
		return nil, err
	}

	return &Downloader{
		BaseUrl:  fmt.Sprintf("%s/files/download.json", su),
		BasePath: d,
		Path:     path,
		writer:   w,
	}, nil
}

// start the pailer
func (d *Downloader) Download() (int64, error) {
	log.Infof("Downloading: %s %s/%s", d.BaseUrl, d.BasePath, d.Path)
	url := fmt.Sprintf("%s?path=%s",
		d.BaseUrl,
		url.QueryEscape(fmt.Sprintf("%s/%s", d.BasePath, d.Path)))
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return io.Copy(d.writer, resp.Body)
}
