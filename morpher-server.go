package main

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2015 Essential Kaos                         //
//      Essential Kaos Open Source License <http://essentialkaos.com/ekol?en>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"pkg.re/essentialkaos/ek.v1/arg"
	"pkg.re/essentialkaos/ek.v1/fmtc"
	"pkg.re/essentialkaos/ek.v1/fsutil"
	"pkg.re/essentialkaos/ek.v1/knf"
	"pkg.re/essentialkaos/ek.v1/log"
	"pkg.re/essentialkaos/ek.v1/req"
	"pkg.re/essentialkaos/ek.v1/signal"
	"pkg.re/essentialkaos/ek.v1/usage"

	"pkg.re/essentialkaos/librato.v1"

	"github.com/essentialkaos/pkgre/morpher"
)

// ////////////////////////////////////////////////////////////////////////////////// //

const (
	APP  = "PkgRE Morpher Server"
	VER  = "0.1.5"
	DESC = "HTTP Server for morphing go get requests"
)

const (
	ARG_CONFIG   = "c:config"
	ARG_NO_COLOR = "nc:no-color"
	ARG_HELP     = "h:help"
	ARG_VER      = "v:version"
)

const (
	MIN_PROCS         = 1
	MAX_PROCS         = 32
	MIN_PORT          = 1025
	MAX_PORT          = 65535
	MIN_READ_TIMEOUT  = 1
	MAX_READ_TIMEOUT  = 120
	MIN_WRITE_TIMEOUT = 1
	MAX_WRITE_TIMEOUT = 120
	MIN_HEADER_SIZE   = 1024
	MAX_HEADER_SIZE   = 10 * 1024 * 1024
)

const (
	MAIN_PROCS           = "main:procs"
	HTTP_IP              = "http:ip"
	HTTP_PORT            = "http:port"
	HTTP_READ_TIMEOUT    = "http:read-timeout"
	HTTP_WRITE_TIMEOUT   = "http:write-timeout"
	HTTP_MAX_HEADER_SIZE = "http:max-header-size"
	HTTP_REDIRECT        = "http:redirect"
	LOG_LEVEL            = "log:level"
	LOG_DIR              = "log:dir"
	LOG_FILE             = "log:file"
	LOG_PERMS            = "log:perms"
	LIBRATO_ENABLED      = "librato:enabled"
	LIBRATO_MAIL         = "librato:mail"
	LIBRATO_TOKEN        = "librato:token"
	LIBRATO_PREFIX       = "librato:prefix"
)

// ////////////////////////////////////////////////////////////////////////////////// //

var argMap = arg.Map{
	ARG_CONFIG:   &arg.V{Value: "/etc/morpher.conf"},
	ARG_NO_COLOR: &arg.V{Type: arg.BOOL},
	ARG_HELP:     &arg.V{Type: arg.BOOL, Alias: "u:usage"},
	ARG_VER:      &arg.V{Type: arg.BOOL, Alias: "ver"},
}

// ////////////////////////////////////////////////////////////////////////////////// //

func main() {
	_, errs := arg.Parse(argMap)

	if len(errs) != 0 {
		fmtc.Println("{r}Arguments parsing errors:{!}")

		for _, err := range errs {
			fmtc.Printf("  {r}%s{!}\n", err.Error())
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
		fmtc.Printf("{r}%s{!}\n", err.Error())
		os.Exit(1)
	}

	prepare()

	log.Aux("Starting %s %s...", APP, VER)

	start()
}

// prepare prepare service for start
func prepare() {
	// Set default user agent for all requests
	req.UserAgent = fmtc.Sprintf("%s/%s (go; %s; %s-%s)",
		APP, VER, runtime.Version(),
		runtime.GOARCH, runtime.GOOS)

	// Register signal handlers
	signal.Handlers{
		signal.TERM: termSignalHandler,
		signal.INT:  intSignalHandler,
		signal.HUP:  hupSignalHandler,
	}.Track()

	validateConfig()
	setupLogger()
	setupLibrato()
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
		&knf.Validator{MAIN_PROCS, knf.Less, MIN_PROCS},
		&knf.Validator{MAIN_PROCS, knf.Greater, MAX_PROCS},
		&knf.Validator{HTTP_PORT, knf.Less, MIN_PORT},
		&knf.Validator{HTTP_PORT, knf.Greater, MAX_PORT},
		&knf.Validator{HTTP_READ_TIMEOUT, knf.Less, MIN_READ_TIMEOUT},
		&knf.Validator{HTTP_READ_TIMEOUT, knf.Greater, MAX_READ_TIMEOUT},
		&knf.Validator{HTTP_WRITE_TIMEOUT, knf.Less, MIN_WRITE_TIMEOUT},
		&knf.Validator{HTTP_WRITE_TIMEOUT, knf.Greater, MAX_WRITE_TIMEOUT},
		&knf.Validator{HTTP_MAX_HEADER_SIZE, knf.Less, MIN_HEADER_SIZE},
		&knf.Validator{HTTP_MAX_HEADER_SIZE, knf.Greater, MAX_HEADER_SIZE},
		&knf.Validator{LOG_DIR, permsChecker, "DWX"},
	}

	if knf.GetB(LIBRATO_ENABLED, false) {
		validators = append(validators,
			&knf.Validator{LIBRATO_MAIL, knf.Empty, nil},
			&knf.Validator{LIBRATO_TOKEN, knf.Empty, nil},
			&knf.Validator{LIBRATO_PREFIX, knf.Empty, nil},
		)
	}

	errs := knf.Validate(validators)

	if len(errs) != 0 {
		fmtc.Println("{r}Error while config validation:{!}")

		for _, err := range errs {
			fmtc.Printf("  {r}%s{!}\n", err.Error())
		}

		os.Exit(1)
	}
}

// setupLogger init and setup global logger
func setupLogger() {
	err := log.Set(knf.GetS(LOG_FILE), knf.GetM(LOG_PERMS, 0644))

	levels := map[string]int{
		"debug": log.DEBUG,
		"info":  log.INFO,
		"warn":  log.WARN,
		"error": log.ERROR,
		"crit":  log.CRIT,
	}

	log.MinLevel(levels[strings.ToLower(knf.GetS(LOG_LEVEL, "debug"))])

	if err != nil {
		fmtc.Printf("{r}Can't setup logger: %s{!}\n", err.Error())
		os.Exit(1)
	}
}

// setupLibrato set librato credentials
func setupLibrato() {
	if !knf.GetB(LIBRATO_ENABLED, false) {
		return
	}

	librato.Mail = knf.GetS(LIBRATO_MAIL)
	librato.Token = knf.GetS(LIBRATO_TOKEN)
}

// start start web server
func start() {
	runtime.GOMAXPROCS(knf.GetI(MAIN_PROCS))

	log.Debug("Max procs set to %d", knf.GetI(MAIN_PROCS))

	err := morpher.Start()

	if err != nil {
		log.Crit(err.Error())
		exit(1)
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
		License: "Essential Kaos Open Source License <https://essentialkaos.com/ekol?en>",
	}

	about.Render()
}
