package server

import (
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"gopkg.in/yaml.v2"

	"github.com/square/blip"
)

// Options represents typical command line options: --addr, --config, etc.
type Options struct {
	BootCheck     bool   `arg:"--boot-check"`
	Config        string `env:"BLIP_CONFIG" default:"blip.yaml"`
	Debug         bool   `env:"BLIP_DEBUG"`
	Help          bool
	Plans         string `env:"BLIP_PLANS"`
	PrintConfig   bool   `arg:"--print-config"`
	PrintMonitors bool   `arg:"--print-monitors"`
	PrintPlans    bool   `arg:"--print-plans"`
	Strict        bool   `env:"BLIP_STRICT"`
	Version       bool   `arg:"-v"`
}

// CommandLine represents options (--addr, etc.) and args: entity type, return
// labels, and query predicates. The caller is expected to copy and use the embedded
// structs separately, like:
//
//   var o config.Options = cmdLine.Options
//   for i, arg := range cmdline.Args {
//
type CommandLine struct {
	Options
	Args []string `arg:"positional"`
}

// ParseCommandLine parses the command line and env vars. Command line options
// override env vars. Default options are used unless overridden by env vars or
// command line options. Defaults are usually parsed from config files.
func ParseCommandLine() (CommandLine, error) {
	var c CommandLine
	p, err := arg.NewParser(arg.Config{Program: "blip"}, &c)
	if err != nil {
		return c, err
	}
	if err := p.Parse(os.Args); err != nil {
		switch err {
		case arg.ErrHelp:
			c.Help = true
		case arg.ErrVersion:
			c.Version = true
		default:
			return c, fmt.Errorf("Error parsing command line: %s\n", err)
		}
	}
	return c, nil
}

func printHelp() {
	fmt.Printf("Usage:\n"+
		"  blip [options]\n\n"+
		"Options:\n"+
		"  --boot-check     Boot then exit\n"+
		"  --config         Config file (default: %s)\n"+
		"  --debug          Print debug to stderr\n"+
		"  --help           Print help\n"+
		"  --plans          Plans files (default: %s)\n"+
		"  --print-config   Print config and monitors\n"+
		"  --print-monitors Print monitors \n"+
		"  --print-plans    Print level plans\n"+
		"  --version        Print version\n"+
		"\n"+
		"blip %s\n",
		blip.DEFAULT_CONFIG_FILE, blip.DEFAULT_PLANS_FILES, blip.VERSION,
	)
}

func printYAML(v interface{}) {
	bytes, err := yaml.Marshal(v)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(bytes))
}
