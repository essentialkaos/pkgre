package server

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2021 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"pkg.re/essentialkaos/ek.v12/fmtc"
	"pkg.re/essentialkaos/ek.v12/knf"
	"pkg.re/essentialkaos/ek.v12/log"
	"pkg.re/essentialkaos/ek.v12/options"
	"pkg.re/essentialkaos/ek.v12/signal"
	"pkg.re/essentialkaos/ek.v12/usage"

	knfv "pkg.re/essentialkaos/ek.v12/knf/validators"
	knff "pkg.re/essentialkaos/ek.v12/knf/validators/fs"

	"github.com/essentialkaos/pkgre/server/healthcheck"
	"github.com/essentialkaos/pkgre/server/morpher"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Application info
const (
	APP  = "PkgRE Morpher Server"
	VER  = "4.4.0"
	DESC = "HTTP Server for morphing go get requests"
)

// Supported command-line options
const (
	OPT_CONFIG   = "c:config"
	OPT_NO_COLOR = "nc:no-color"
	OPT_HELP     = "h:help"
	OPT_VER      = "v:version"
)

// Limits
const (
	MIN_PROCS = 1
	MAX_PROCS = 32
	MIN_PORT  = 1025
	MAX_PORT  = 65535
)

// Configuration file properties names
const (
	MAIN_PROCS      = "main:procs"
	HTTP_IP         = "http:ip"
	HTTP_PORT       = "http:port"
	HTTP_REDIRECT   = "http:redirect"
	HEALTHCHECK_URL = "healthcheck:url"
	LOG_LEVEL       = "log:level"
	LOG_DIR         = "log:dir"
	LOG_FILE        = "log:file"
	LOG_PERMS       = "log:perms"
)

// ////////////////////////////////////////////////////////////////////////////////// //

var optMap = options.Map{
	OPT_CONFIG:   {Value: "/etc/morpher.knf"},
	OPT_NO_COLOR: {Type: options.BOOL},
	OPT_HELP:     {Type: options.BOOL, Alias: "u:usage"},
	OPT_VER:      {Type: options.BOOL, Alias: "ver"},
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Init is main func
func Init() {
	_, errs := options.Parse(optMap)

	if len(errs) != 0 {
		printError("Arguments parsing errors:")

		for _, err := range errs {
			printError("  %v", err)
		}

		os.Exit(1)
	}

	if options.GetB(OPT_NO_COLOR) {
		fmtc.DisableColors = true
	}

	if options.GetB(OPT_VER) {
		showAbout()
		return
	}

	if options.GetB(OPT_HELP) {
		showUsage()
		return
	}

	err := knf.Global(options.GetS(OPT_CONFIG))

	if err != nil {
		printError(err.Error())
		os.Exit(1)
	}

	prepare()

	log.Aux(strings.Repeat("-", 88))
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

	runtime.GOMAXPROCS(knf.GetI(MAIN_PROCS))

	log.Debug("GOMAXPROCS set to %d", knf.GetI(MAIN_PROCS))
}

// validateConfig validate config values
func validateConfig() {
	errs := knf.Validate([]*knf.Validator{
		{MAIN_PROCS, knfv.Less, MIN_PROCS},
		{MAIN_PROCS, knfv.Greater, MAX_PROCS},
		{HTTP_PORT, knfv.Less, MIN_PORT},
		{HTTP_PORT, knfv.Greater, MAX_PORT},
		{LOG_DIR, knff.Perms, "DWX"},
	})

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
	if knf.HasProp(HEALTHCHECK_URL) {
		healthcheck.Start(knf.GetS(HEALTHCHECK_URL), time.Minute)
	}

	err := morpher.Start(VER)

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
	info := usage.NewInfo()

	info.AddOption(OPT_CONFIG, "Path to config file", "file")
	info.AddOption(OPT_NO_COLOR, "Disable colors in output")
	info.AddOption(OPT_HELP, "Show this help message")
	info.AddOption(OPT_VER, "Show version")

	info.Render()
}

func showAbout() {
	about := &usage.About{
		App:     APP,
		Version: VER,
		Desc:    DESC,
		Year:    2009,
		Owner:   "Essential Kaos",
		License: "Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>",
	}

	about.Render()
}
