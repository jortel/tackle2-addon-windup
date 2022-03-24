package main

import (
	"errors"
	"github.com/konveyor/tackle2-hub/api"
	"os"
	pathlib "path"
	"strings"
)

var (
	// SourceDir repository path.
	SourceDir = "source"
)

func init() {
	SourceDir = pathlib.Join(Dir, SourceDir)
}

//
// Factory.
func newRepository(homeDir string, application *api.Application) (r Repository, err error) {
	kind := application.Repository.Kind
	if kind == "" {
		if strings.HasSuffix(application.Repository.URL, ".git") {
			kind = "git"
		} else {
			kind = "svn"
		}
	}
	switch kind {
	case "svn":
		r = &Subversion{}
	case "git":
		r = &Git{}
	default:
		err = errors.New("unknown kind")
		return
	}
	r.With(homeDir, application)
	err = r.Validate()
	return
}

//
// Repository interface.
type Repository interface {
	With(homeDir string, application *api.Application)
	Fetch() (err error)
	Validate() (err error)
}

//
// SCM - source code manager.
type SCM struct {
	Application *api.Application
	HomeDir     string
}

//
// With settings.
func (r *SCM) With(homeDir string, application *api.Application) {
	r.HomeDir = homeDir
	r.Application = application
}

//
// EnsureDir creates a directory if not exists.
func (r *SCM) EnsureDir(path string, mode os.FileMode) (err error) {
	err = os.MkdirAll(path, mode)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			err = nil
		}
	}
	return
}

//
// WriteSSH writes the SSH key.
func (r *SCM) WriteSSH(id *api.Identity) (err error) {
	if id.Key == "" {
		return
	}
	dir := pathlib.Join(HomeDir, ".ssh")
	err = r.EnsureDir(dir, 0700)
	err = os.Mkdir(dir, 0700)
	if err != nil {
		return
	}
	path := pathlib.Join(dir, "id_git")
	_, err = os.Stat(path)
	if !errors.Is(err, os.ErrNotExist) {
		err = os.ErrExist
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	_, err = f.Write([]byte(id.Key))
	_ = f.Close()
	return
}
