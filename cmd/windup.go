package main

import (
	"bufio"
	"github.com/konveyor/tackle2-addon/command"
	"github.com/konveyor/tackle2-addon/repository"
	"github.com/konveyor/tackle2-hub/api"
	"github.com/konveyor/tackle2-hub/nas"
	"os"
	pathlib "path"
	"strconv"
	"strings"
)

//
// Windup application analyzer.
type Windup struct {
	application *api.Application
	*Data
}

//
// Run windup.
func (r *Windup) Run() (err error) {
	cmd := command.Command{Path: "/opt/windup"}
	cmd.Options, err = r.options()
	if err != nil {
		return
	}
	err = cmd.Run()
	if err != nil {
		r.reportLog()
	}

	return
}

//
// reportLog reports the log content.
func (r *Windup) reportLog() {
	path := pathlib.Join(
		HomeDir,
		".mta",
		"log",
		"mta.log")
	f, err := os.Open(path)
	if err != nil {
		return
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		addon.Activity(">> %s\n", scanner.Text())
	}
	_ = f.Close()
}

//
// options builds CLL options.
func (r *Windup) options() (options command.Options, err error) {
	options = command.Options{
		"--batchMode",
		"--output",
		ReportDir,
	}
	err = r.maven(&options)
	if err != nil {
		return
	}
	err = r.Mode.AddOptions(&options)
	if err != nil {
		return
	}
	if r.Sources != nil {
		err = r.Sources.AddOptions(&options)
		if err != nil {
			return
		}
	}
	if r.Targets != nil {
		err = r.Targets.AddOptions(&options)
		if err != nil {
			return
		}
	}
	err = r.Scope.AddOptions(&options)
	if err != nil {
		return
	}
	if r.Rules != nil {
		err = r.Rules.AddOptions(&options)
		if err != nil {
			return
		}
	}
	return
}

//
// maven add --input for maven artifacts.
func (r *Windup) maven(options *command.Options) (err error) {
	found, err := nas.HasDir(DepDir)
	if found {
		options.Add("--input", DepDir)
	}
	return
}

//
// Mode settings.
type Mode struct {
	Binary     bool   `json:"binary"`
	Artifact   string `json:"artifact"`
	WithDeps   bool   `json:"withDeps"`
	Diva       bool   `json:"diva"`
	Repository repository.Repository
}

//
// AddOptions adds windup options.
func (r *Mode) AddOptions(options *command.Options) (err error) {
	if r.Binary {
		if r.Artifact != "" {
			bucket := addon.Bucket()
			err = bucket.Get(r.Artifact, BinDir)
			if err != nil {
				return
			}
			options.Add("--input", BinDir)
		}
	} else {
		options.Add("--input", AppDir)
	}
	if r.Diva {
		options.Add("--enableTransactionAnalysis")
	}

	return
}

//
// Sources list of sources.
type Sources []string

//
// AddOptions add options.
func (r Sources) AddOptions(options *command.Options) (err error) {
	for _, source := range r {
		options.Add("--source", source)
	}
	return
}

//
// Targets list of target.
type Targets []string

//
// AddOptions add options.
func (r Targets) AddOptions(options *command.Options) (err error) {
	for _, target := range r {
		options.Add("--target", target)
	}
	return
}

//
// Scope settings.
type Scope struct {
	WithKnown bool `json:"withKnown"`
	Packages  struct {
		Included []string `json:"included,omitempty"`
		Excluded []string `json:"excluded,omitempty"`
	} `json:"packages"`
}

//
// AddOptions adds windup options.
func (r *Scope) AddOptions(options *command.Options) (err error) {
	if r.WithKnown {
		options.Add("--analyzeKnownLibraries")
	}
	if len(r.Packages.Included) > 0 {
		options.Add("--packages", r.Packages.Included...)
	}
	if len(r.Packages.Excluded) > 0 {
		options.Add("--excludePackages", r.Packages.Excluded...)
	}
	return
}

//
// Rules settings.
type Rules struct {
	Path       string          `json:"path" binding:"required"`
	Bundles    []api.Ref       `json:"bundles"`
	Repository *api.Repository `json:"repository"`
	Identity   *api.Ref        `json:"identity"`
	Tags       struct {
		Included []string `json:"included,omitempty"`
		Excluded []string `json:"excluded,omitempty"`
	} `json:"tags"`
}

//
// AddOptions adds windup options.
func (r *Rules) AddOptions(options *command.Options) (err error) {
	ruleDir := pathlib.Join(RuleDir, "/files")
	err = nas.MkDir(ruleDir, 0755)
	if err != nil {
		return
	}
	options.Add(
		"--userRulesDirectory",
		ruleDir)
	bucket := addon.Bucket()
	err = bucket.Get(r.Path, ruleDir)
	if err != nil {
		return
	}
	err = r.addRepository(options)
	if err != nil {
		return
	}
	err = r.addBundles(options)
	if err != nil {
		return
	}
	if len(r.Tags.Included) > 0 {
		options.Add("--includeTags", r.Tags.Included...)
	}
	if len(r.Tags.Excluded) > 0 {
		options.Add("--excludeTags", r.Tags.Excluded...)
	}
	return
}

//
// AddBundles adds bundles.
func (r *Rules) addBundles(options *command.Options) (err error) {
	for _, ref := range r.Bundles {
		var bundle *api.RuleBundle
		bundle, err = addon.RuleBundle.Get(ref.ID)
		if err != nil {
			return
		}
		err = r.addRuleSets(options, bundle)
		if err != nil {
			return
		}
		err = r.addBundleRepository(options, bundle)
		if err != nil {
			return
		}
	}
	return
}

//
// addRuleSets adds ruleSets
func (r *Rules) addRuleSets(options *command.Options, bundle *api.RuleBundle) (err error) {
	ruleDir := pathlib.Join(
		RuleDir,
		"/bundles",
		strconv.Itoa(int(bundle.ID)),
		"rulesets")
	err = nas.MkDir(ruleDir, 0755)
	if err != nil {
		return
	}
	options.Add(
		"--userRulesDirectory",
		ruleDir)
	for _, ruleset := range bundle.RuleSets {
		fileRef := ruleset.File
		if fileRef == nil {
			continue
		}
		name := strings.Join(
			[]string{
				strconv.Itoa(int(ruleset.ID)),
				fileRef.Name},
			"-")
		path := pathlib.Join(ruleDir, name)
		addon.Activity("[FILE] Get rule: %s", path)
		err = addon.File.Get(ruleset.File.ID, path)
		if err != nil {
			break
		}
	}
	return
}

//
// addBundleRepository adds bundle repository.
func (r *Rules) addBundleRepository(options *command.Options, bundle *api.RuleBundle) (err error) {
	if bundle.Repository == nil {
		return
	}
	rootDir := pathlib.Join(
		RuleDir,
		"/bundles",
		strconv.Itoa(int(bundle.ID)),
		"repository")
	err = nas.MkDir(rootDir, 0755)
	if err != nil {
		return
	}
	owner := &api.Application{}
	owner.Repository = bundle.Repository
	if bundle.Identity != nil {
		owner.Identities = []api.Ref{*bundle.Identity}
	}
	rp, err := repository.New(rootDir, owner)
	if err != nil {
		return
	}
	err = rp.Fetch()
	if err != nil {
		return
	}
	ruleDir := pathlib.Join(rootDir, bundle.Repository.Path)
	options.Add(
		"--userRulesDirectory",
		ruleDir)
	entries, err := os.ReadDir(ruleDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := ".windup.xml"
		if !strings.HasSuffix(entry.Name(), ext) {
			addon.Activity(
				"[WARNING] File %s without extension (%s) ignored.",
				pathlib.Join(
					ruleDir,
					entry.Name()),
				ext)
		}
	}
	return
}

//
// addRepository adds custom repository.
func (r *Rules) addRepository(options *command.Options) (err error) {
	if r.Repository == nil {
		return
	}
	rootDir := pathlib.Join(
		RuleDir,
		"repository")
	err = nas.MkDir(rootDir, 0755)
	if err != nil {
		return
	}
	owner := &api.Application{}
	owner.Repository = r.Repository
	if r.Identity != nil {
		owner.Identities = []api.Ref{*r.Identity}
	}
	rp, err := repository.New(rootDir, owner)
	if err != nil {
		return
	}
	err = rp.Fetch()
	if err != nil {
		return
	}
	ruleDir := pathlib.Join(rootDir, r.Repository.Path)
	options.Add(
		"--userRulesDirectory",
		ruleDir)
	entries, err := os.ReadDir(ruleDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := ".windup.xml"
		if !strings.HasSuffix(entry.Name(), ext) {
			addon.Activity(
				"[WARNING] File %s without extension (%s) ignored.",
				pathlib.Join(
					ruleDir,
					entry.Name()),
				ext)
		}
	}
	return
}
