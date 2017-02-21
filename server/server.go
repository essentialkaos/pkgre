package server

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2017 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"runtime"

	"pkg.re/essentialkaos/ek.v6/arg"
	"pkg.re/essentialkaos/ek.v6/fmtc"
	"pkg.re/essentialkaos/ek.v6/fsutil"
	"pkg.re/essentialkaos/ek.v6/knf"
	"pkg.re/essentialkaos/ek.v6/log"
	"pkg.re/essentialkaos/ek.v6/signal"
	"pkg.re/essentialkaos/ek.v6/usage"

	"github.com/essentialkaos/pkgre/server/morpher"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP  = "PkgRE Morpher Server"
	VER  = "2.1.0"
	DESC = "HTTP Server for morphing go get requests"
)

const (
	ARG_CONFIG   = "c:config"
	ARG_NO_COLOR = "nc:no-color"
	ARG_HELP     = "h:help"
	ARG_VER      = "v:version"
)

const (
	MIN_PROCS = 1
	MAX_PROCS = 32
	MIN_PORT  = 1025
	MAX_PORT  = 65535
)

const (
	MAIN_PROCS    = "main:procs"
	HTTP_IP       = "http:ip"
	HTTP_PORT     = "http:port"
	HTTP_REDIRECT = "http:redirect"
	LOG_LEVEL     = "log:level"
	LOG_DIR       = "log:dir"
	LOG_FILE      = "log:file"
	LOG_PERMS     = "log:perms"
)

// ////////////////////////////////////////////////////////////////////////////////// //

var argMap = arg.Map{
	ARG_CONFIG:   &arg.V{Value: "/etc/morpher.knf"},
	ARG_NO_COLOR: &arg.V{Type: arg.BOOL},
	ARG_HELP:     &arg.V{Type: arg.BOOL, Alias: "u:usage"},
	ARG_VER:      &arg.V{Type: arg.BOOL, Alias: "ver"},
}

// ////////////////////////////////////////////////////////////////////////////////// //

func Init() {
	_, errs := arg.Parse(argMap)

	if len(errs) != 0 {
		printError("Arguments parsing errors:")

		for _, err := range errs {
			printError("  %v", err)
		}

		os.Exit(1)
	}

	if arg.GetB(ARG_NO_COLOR) {
		fmtc.DisableColors = true
	}

	if arg.GetB(ARG_VER) {
		showAbout()
		return
	}

	if arg.GetB(ARG_HELP) {
		showUsage()
		return
	}

	err := knf.Global(arg.GetS(ARG_CONFIG))

	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	prepare()

	log.Aux("Starting %s %s...", APP, VER)

	start()
}

// prepare prepare service for start
func prepare() {
	// Register signal handlers
	signal.Handlers{
		signal.TERM: termSignalHandler,
		signal.INT:  intSignalHandler,
		signal.HUP:  hupSignalHandler,
	}.Track()

	validateConfig()
	setupLogger()
}

// validateConfig validate config values
func validateConfig() {
	var permsChecker = func(config *knf.Config, prop string, value interface{}) error {
		if !fsutil.CheckPerms(value.(string), config.GetS(prop)) {
			switch value.(string) {
			case "DWX":
				return fmt.Errorf("Property %s must be path to writable directory.", prop)
			}
		}

		return nil
	}

	validators := []*knf.Validator{
		{MAIN_PROCS, knf.Less, MIN_PROCS},
		{MAIN_PROCS, knf.Greater, MAX_PROCS},
		{HTTP_PORT, knf.Less, MIN_PORT},
		{HTTP_PORT, knf.Greater, MAX_PORT},
		{LOG_DIR, permsChecker, "DWX"},
	}

	errs := knf.Validate(validators)

	if len(errs) != 0 {
		printError("Error while config validation:")

		for _, err := range errs {
			printError("  %v", err)
		}

		os.Exit(1)
	}
}

// setupLogger init and setup global logger
func setupLogger() {
	err := log.Set(knf.GetS(LOG_FILE), knf.GetM(LOG_PERMS, 0644))

	if err != nil {
		printError("Can't setup logger: %v", err)
		os.Exit(1)
	}

	err = log.MinLevel(knf.GetS(LOG_LEVEL, "info"))

	if err != nil {
		printError("Can't set log level: %v", err)
	}
}

// start start web server
func start() {
	runtime.GOMAXPROCS(knf.GetI(MAIN_PROCS))

	log.Debug("GOMAXPROCS set to %d", knf.GetI(MAIN_PROCS))

	err := morpher.Start()

	if err != nil {
		log.Crit(err.Error())
		exit(1)
	}
}

// printError print error message
func printError(message string, args ...interface{}) {
	if len(args) == 0 {
		fmtc.Printf("{r}%s{!}\n", message)
	} else {
		fmtc.Printf("{r}%s{!}\n", fmt.Sprintf(message, args...))
	}
}

// exit exits from app
func exit(code int) {
	// Flush buffered log data
	log.Flush()

	os.Exit(code)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// INT signal handler
func intSignalHandler() {
	log.Aux("Received INT signal, shutdown...")
	exit(0)
}

// TERM signal handler
func termSignalHandler() {
	log.Aux("Received TERM signal, shutdown...")
	exit(0)
}

// HUP signal handler
func hupSignalHandler() {
	log.Info("Received HUP signal, log will be reopened...")
	log.Reopen()
}

// ////////////////////////////////////////////////////////////////////////////////// //

func showUsage() {
	info := usage.NewInfo("")

	info.AddOption(ARG_CONFIG, "Path to config file", "file")
	info.AddOption(ARG_NO_COLOR, "Disable colors in output")
	info.AddOption(ARG_HELP, "Show this help message")
	info.AddOption(ARG_VER, "Show version")

	info.Render()
}

func showAbout() {
	about := &usage.About{
		App:     APP,
		Version: VER,
		Desc:    DESC,
		Year:    2009,
		Owner:   "Essential Kaos",
		License: "Essential Kaos Open Source License <https://essentialkaos.com/ekol>",
	}

	about.Render()
}
