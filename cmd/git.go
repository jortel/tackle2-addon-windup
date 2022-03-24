package main

import (
	"errors"
	"fmt"
	"github.com/konveyor/tackle2-hub/api"
	urllib "net/url"
	"os"
	pathlib "path"
)

//
// Git repository.
type Git struct {
	SCM
}

//
// Validate settings.
func (r *Git) Validate() (err error) {
	u, err := urllib.Parse(r.Application.Repository.URL)
	if err != nil {
		return
	}
	insecure, err := addon.Setting.Bool("git.insecure.enabled")
	if err != nil {
		return
	}
	switch u.Scheme {
	case "http":
		if !insecure {
			err = errors.New(
				"http URL used with git.insecure.enabled = FALSE")
			return
		}
	}
	return
}

//
// Fetch clones the repository.
func (r *Git) Fetch() (err error) {
	url := r.URL()
	addon.Activity("[GIT] Cloning: %s", url.String())
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
	cmd := Command{Path: "/usr/bin/git"}
	cmd.Options.add("clone", url.String(), SourceDir)
	err = cmd.Run()
	return
}

//
// URL returns the parsed URL.
func (r *Git) URL() (u *urllib.URL) {
	u, _ = urllib.Parse(r.Application.Repository.URL)
	return
}

//
// writeConfig writes config file.
func (r *Git) writeConfig() (err error) {
	path := pathlib.Join(r.HomeDir, ".gitconfig")
	_, err = os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		err = os.ErrExist
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	insecure, err := addon.Setting.Bool("git.insecure.enabled")
	if err != nil {
		return
	}
	proxy, err := r.proxy()
	if err != nil {
		return
	}
	s := "[credential]\n"
	s += "helper = store\n"
	s += "[http]\n"
	s += fmt.Sprintf("sslVerify = %t\n", !insecure)
	if proxy != "" {
		s += fmt.Sprintf("proxy = %s\n", proxy)
	}
	_, err = f.Write([]byte(s))
	_ = f.Close()
	return
}

//
// writeCreds writes credentials (store) file.
func (r *Git) writeCreds(id *api.Identity) (err error) {
	path := pathlib.Join(r.HomeDir, ".git-credentials")
	_, err = os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		err = os.ErrExist
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	url := r.URL()
	entry := url.Scheme
	entry += "://"
	if id.User != "" {
		entry += id.User
		entry += ":"
	}
	if id.Password != "" {
		entry += id.Password
		entry += "@"
	}
	entry += url.Host
	_, err = f.Write([]byte(entry + "\n"))
	_ = f.Close()
	return
}

//
// proxy builds the proxy.
func (r *Git) proxy() (proxy string, err error) {
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
	auth := ""
	if p.Identity != nil {
		var id *api.Identity
		id, err = addon.Identity.Get(p.Identity.ID)
		if err != nil {
			return
		}
		auth = fmt.Sprintf(
			"%s:%s@",
			id.User,
			id.Password)
	}
	proxy = fmt.Sprintf(
		"%s://%s%s",
		p.Kind,
		auth,
		p.Host)
	if p.Port > 0 {
		proxy = fmt.Sprintf(
			"%s:%d",
			proxy,
			p.Port)
	}
	return
}
