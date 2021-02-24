package etchosts

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/docker/docker/opts"
)

// Code referenced from https://github.com/moby/libnetwork/blob/master/etchosts/etchosts.go

// Record Structure for a single host record
type Record struct {
	Hosts string
	IP    string
}

// WriteTo writes record to file and returns bytes written or error
func (r Record) WriteTo(w io.Writer) (int64, error) {
	n, err := fmt.Fprintf(w, "%s\t%s\n", r.IP, r.Hosts)
	return int64(n), err
}

var (
	// Default hosts config records slice
	defaultContent = []Record{
		{Hosts: "localhost", IP: "127.0.0.1"},
		{Hosts: "localhost ip6-localhost ip6-loopback", IP: "::1"},
		{Hosts: "ip6-localnet", IP: "fe00::0"},
		{Hosts: "ip6-mcastprefix", IP: "ff00::0"},
		{Hosts: "ip6-allnodes", IP: "ff02::1"},
		{Hosts: "ip6-allrouters", IP: "ff02::2"},
	}
)

// BuildEtcHosts builds NOMAD_TASK_DIR/etc_hosts with defaults.
func BuildEtcHosts(hostsFile string) error {
	content := bytes.NewBuffer(nil)

	// Write defaultContent slice to buffer
	for _, r := range defaultContent {
		if _, err := r.WriteTo(content); err != nil {
			return err
		}
	}

	return ioutil.WriteFile(hostsFile, content.Bytes(), 0644)
}

// CopyEtcHosts copies /etc/hosts to NOMAD_TASK_DIR/etc_hosts
func CopyEtcHosts(hostsFile string) error {
	srcFile, err := os.Open("/etc/hosts")
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.Create(hostsFile)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}

// AddExtraHosts add hosts, given as name:IP to container /etc/hosts.
func AddExtraHosts(hostsFile string, extraHosts []string) error {
	fd, err := os.OpenFile(hostsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer fd.Close()

	for _, extraHost := range extraHosts {
		// allow IPv6 addresses in extra hosts; only split on first ":"
		if _, err := opts.ValidateExtraHost(extraHost); err != nil {
			return err
		}

		hostnameIP := strings.SplitN(extraHost, ":", 2)
		msg := fmt.Sprintf("%s\t%s\n", hostnameIP[1], hostnameIP[0])
		if _, err := fd.WriteString(msg); err != nil {
			return err
		}
	}
	return nil
}
