package main

import (
	"errors"
	hub "github.com/konveyor/tackle2-hub/addon"
	"os"
)

var (
	// hub integration.
	addon = hub.Addon
	// HomeDir directory.
	HomeDir = ""
	Dir     = ""
)

func init() {
	HomeDir, _ = os.UserHomeDir()
	Dir, _ = os.Getwd()
}

//
// Data Addon data passed in the secret.
type Data struct {
	// Output directory within application bucket.
	Output string `json:"output" binding:"required"`
	// Mode options.
	Mode Mode `json:"mode"`
	// Sources list.
	Sources Sources `json:"sources"`
	// Targets list.
	Targets Targets `json:"targets"`
	// Scope options.
	Scope Scope `json:"scope"`
	// Rules options.
	Rules *Rules `json:"rules"`
}

//
// main
func main() {
	addon.Run(func() (err error) {
		//
		// Get the addon data associated with the task.
		d := &Data{}
		err = addon.DataWith(d)
		if err != nil {
			return
		}
		windup := Windup{}
		windup.Data = d
		//
		// Fetch application.
		addon.Activity("Fetching application.")
		application, err := addon.Task.Application()
		if err == nil {
			windup.bucket = application.Bucket
		} else {
			return
		}
		//
		// Fetch repository.
		if !d.Mode.Binary {
			addon.Total(2)
			if application.Repository == nil {
				err = errors.New("Application repository not defined.")
				return
			}
			var r Repository
			r, err = newRepository(HomeDir, application)
			if err != nil {
				return
			}
			err = r.Fetch()
			if err == nil {
				addon.Increment()
				windup.Mode.Repository = r
			} else {
				return
			}
			if d.Mode.WithDeps {
				mvn := Maven{}
				mvn.Application = application
				err = mvn.Fetch()
				if err != nil {
					return
				}
			}
		}
		//
		// Run windup.
		err = windup.Run()
		if err == nil {
			addon.Increment()
		} else {
			return
		}
		return
	})
}
