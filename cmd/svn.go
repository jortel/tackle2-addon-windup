package main

import (
	"errors"
	"fmt"
	"github.com/konveyor/tackle2-hub/api"
	urllib "net/url"
	"os"
	pathlib "path"
	"strings"
)

//
// Subversion repository.
type Subversion struct {
	SCM
}

//
// Validate settings.
func (r *Subversion) Validate() (err error) {
	u, err := urllib.Parse(r.Application.Repository.URL)
	if err != nil {
		return
	}
	insecure, err := addon.Setting.Bool("svn.insecure.enabled")
	if err != nil {
		return
	}
	switch u.Scheme {
	case "http":
		if !insecure {
			err = errors.New(
				"http URL used with snv.insecure.enabled = FALSE")
			return
		}
	}
	return
}

//
// Fetch clones the repository.
func (r *Subversion) Fetch() (err error) {
	url := r.URL()
	addon.Activity("[SVN] Cloning: %s", url.String())
	_ = os.RemoveAll(SourceDir)
	id, hasCreds, err := addon.Application.FindIdentity(r.Application.ID, "source")
	if err != nil {
		return
	}
	err = r.writeConfig()
	if err != nil {
		return
	}
	if hasCreds {
		err = r.writeCreds(id)
		if err != nil {
			return
		}
		err = r.WriteSSH(id)
		if err != nil {
			return
		}
	}
	insecure, err := addon.Setting.Bool("svn.insecure.enabled")
	if err != nil {
		return
	}
	cmd := Command{Path: "/usr/bin/svn"}
	cmd.Options.add("checkout", url.String(), SourceDir)
	cmd.Options.add("--non-interactive")
	if insecure {
		cmd.Options.add("--trust-server-cert")
	}
	err = cmd.Run()
	return
}

//
// URL returns the parsed URL.
func (r *Subversion) URL() (u *urllib.URL) {
	repository := r.Application.Repository
	u, _ = urllib.Parse(repository.URL)
	branch := r.Application.Repository.Branch
	if branch == "" {
		branch = "trunk"
	}
	u.Path += "/" + branch
	return
}

//
// writeConfig writes config file.
func (r *Subversion) writeConfig() (err error) {
	path := pathlib.Join(
		r.HomeDir,
		".subversion",
		"servers")
	_, err = os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		err = os.ErrExist
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	proxy, err := r.proxy()
	if err != nil {
		return
	}
	_, err = f.Write([]byte(proxy))
	_ = f.Close()
	return
}

//
// writeCreds writes credentials (store) file.
func (r *Subversion) writeCreds(id *api.Identity) (err error) {
	path := pathlib.Join(
		r.HomeDir,
		".subversion",
		"auth",
		"svn.simple",
		"entry")
	_, err = os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		err = os.ErrExist
		return
	}
	err = r.EnsureDir(pathlib.Dir(path), 0700)
	if err != nil {
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	s := "K 15\n"
	s += "svn:realmstring\n"
	s += "V 31\n"
	s += "<https://github.com:443> GitHub\n"
	s += "K 8\n"
	s += "username\n"
	s += fmt.Sprintf("V %d\n", len(id.User))
	s += fmt.Sprintf("%s\n", id.User)
	s += "K 8\n"
	s += "passtype\n"
	s += "V 6\n"
	s += "simple\n"
	s += "K 8\n"
	s += "password\n"
	s += fmt.Sprintf("V %d\n", len(id.Password))
	s += fmt.Sprintf("%s\n", id.Password)
	s += "END\n"
	_, err = f.Write([]byte(s))
	_ = f.Close()
	return
}

//
// proxy builds the proxy.
func (r *Subversion) proxy() (proxy string, err error) {
	kind := ""
	url := r.URL()
	switch url.Scheme {
	case "http":
		kind = "http"
	case "https",
		"git@github.com":
		kind = "https"
	default:
		return
	}
	p, err := addon.Proxy.Find(kind)
	if err != nil || p == nil || !p.Enabled {
		return
	}
	for _, h := range p.Excluded {
		if h == url.Host {
			return
		}
	}
	var id *api.Identity
	if p.Identity != nil {
		id, err = addon.Identity.Get(p.Identity.ID)
		if err != nil {
			return
		}
	}
	proxy = "[global]\n"
	proxy += fmt.Sprintf("http-proxy-host = %s\n", p.Host)
	if p.Port > 0 {
		proxy += fmt.Sprintf("http-proxy-port = %d\n", p.Port)
	}
	if id != nil {
		proxy += fmt.Sprintf("http-proxy-username = %s\n", id.User)
		proxy += fmt.Sprintf("http-proxy-password = %s\n", id.Password)
	}
	proxy += fmt.Sprintf(
		"(http-proxy-exceptions = %s\n",
		strings.Join(p.Excluded, " "))
	return
}
