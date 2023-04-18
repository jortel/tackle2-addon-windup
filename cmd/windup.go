package main

import (
	"bufio"
	"encoding/xml"
	"github.com/konveyor/tackle2-addon/command"
	"github.com/konveyor/tackle2-addon/repository"
	"github.com/konveyor/tackle2-hub/api"
	"github.com/konveyor/tackle2-hub/nas"
	"io/ioutil"
	"os"
	pathlib "path"
	"path/filepath"
	"strconv"
	"strings"
)

//
// EmptyRuleSet provides an empty ruleset.
// Windup requires at least 1 target passed.
var (
	EmptyRuleSet = `<?xml version="1.0"?>
<ruleset id="Empty"
    xmlns="http://windup.jboss.org/schema/jboss-ruleset"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
    xsi:schemaLocation="http://windup.jboss.org/schema/jboss-ruleset http://windup.jboss.org/schema/jboss-ruleset/windup-jboss-ruleset.xsd">
    <metadata>
        <targetTechnology id="Empty"/>
    </metadata>
    <rules/>
</ruleset>`
)

//
// RuleSet an XML document.
type RuleSet struct {
	Metadata struct {
		Target struct {
			ID string `xml:"id,attr"`
		} `xml:"targetTechnology"`
	} `xml:"metadata"`
}

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
	maxMemory := addon.Addon().MaxMemory()
	if maxMemory > 0 {
		_ = os.Setenv(
			"MAX_MEMORY",
			strconv.FormatInt(maxMemory, 10))
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
		"--exitCodes",
		"--batchMode",
		"--output",
		ReportDir,
	}
	err = r.maven(&options)
	if err != nil {
		return
	}
	err = r.Tagger.AddOptions(&options)
	if err != nil {
		return
	}
	err = r.Mode.AddOptions(&options)
	if err != nil {
		return
	}
	if r.Rules != nil {
		err = r.Rules.AddOptions(&options)
		if err != nil {
			return
		}
		r.Targets = append(
			r.Targets,
			r.Rules.foundTargets...)
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
	CSV        bool   `json:"csv"`
	Repository repository.SCM
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
	if r.CSV {
		options.Add("--exportCSV")
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
	if len(r) == 0 {
		err = r.addEmpty(options)
		return
	}
	for _, target := range r {
		options.Add("--target", target)
	}
	return
}

//
// addEmpty adds the empty target/ruleset.
func (r Targets) addEmpty(options *command.Options) (err error) {
	ruleDir := pathlib.Join(RuleDir, "/empty")
	err = nas.MkDir(ruleDir, 0755)
	if err != nil {
		return
	}
	options.Add(
		"--userRulesDirectory",
		ruleDir)
	ruleSet := &RuleSet{}
	err = xml.Unmarshal([]byte(EmptyRuleSet), ruleSet)
	if err != nil {
		return
	}
	options.Add(
		"--target",
		ruleSet.Metadata.Target.ID)
	path := pathlib.Join(ruleDir, "empty.windup.xml")
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()
	_, err = f.WriteString(EmptyRuleSet)
	if err != nil {
		return
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
	foundTargets []string
}

//
// AddOptions adds windup options.
func (r *Rules) AddOptions(options *command.Options) (err error) {
	err = r.addFiles(options)
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
// addFiles add uploaded rules files.
func (r *Rules) addFiles(options *command.Options) (err error) {
	if r.Path == "" {
		return
	}
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
	files := 0
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
		files++
	}
	if files > 0 {
		options.Add(
			"--userRulesDirectory",
			ruleDir)
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
	var ids []api.Ref
	if bundle.Identity != nil {
		ids = []api.Ref{*bundle.Identity}
	}
	rp, err := repository.New(
		rootDir,
		bundle.Repository,
		ids)
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
		"''"+ruleDir+"''")
	err = r.FindTargets(ruleDir)
	if err != nil {
		return
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
	var ids []api.Ref
	if r.Identity != nil {
		ids = []api.Ref{*r.Identity}
	}
	rp, err := repository.New(
		rootDir,
		r.Repository,
		ids)
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
		"''"+ruleDir+"''")
	err = r.FindTargets(ruleDir)
	if err != nil {
		return
	}
	return
}

//
// FindTargets find targets in ruleSets.
// Report invalid files.
func (r *Rules) FindTargets(path string) (err error) {
	err = filepath.Walk(
		path,
		func(path string, info os.FileInfo, wErr error) (err error) {
			if wErr != nil {
				err = wErr
				return
			}
			if info.IsDir() {
				if info.Name()[0] == '.' {
					err = filepath.SkipDir
				}
				return
			}
			if !strings.HasSuffix(info.Name(), ".windup.xml") {
				addon.Activity(
					"[WARNING] File: %s without extension (.windup.xml) ignored.",
					path)
				return
			}
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return
			}
			ruleSet := &RuleSet{}
			xErr := xml.Unmarshal(b, ruleSet)
			if xErr != nil {
				addon.Activity(
					"[WARNING] File: %s XML not valid, ignored.",
					path)
				return
			}
			id := ruleSet.Metadata.Target.ID
			if id != "" {
				r.foundTargets = append(
					r.foundTargets,
					id)
			}
			return
		})
	if err != nil {
		return
	}
	return
}
