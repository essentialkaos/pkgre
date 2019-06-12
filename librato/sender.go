package librato

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                     Copyright (c) 2009-2019 ESSENTIAL KAOS                         //
//        Essential Kaos Open Source License <https://essentialkaos.com/ekol>         //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"fmt"
	"os"
	"time"

	"pkg.re/essentialkaos/ek.v10/knf"
	"pkg.re/essentialkaos/ek.v10/options"
	"pkg.re/essentialkaos/ek.v10/req"

	"pkg.re/essentialkaos/librato.v8"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Supported command-line options
const (
	OPT_CONFIG = "c:config"
)

// Configuration file properties names
const (
	MAIN_ENABLED   = "main:enabled"
	METRICS_URL    = "metrics:url"
	LIBRATO_MAIL   = "librato:mail"
	LIBRATO_TOKEN  = "librato:token"
	LIBRATO_PREFIX = "librato:prefix"
)

// ////////////////////////////////////////////////////////////////////////////////// //

// Metrics contains metrics data
type Metrics struct {
	Hits      uint64 `json:"hits"`
	Misses    uint64 `json:"misses"`
	Errors    uint64 `json:"errors"`
	Redirects uint64 `json:"redirects"`
	Docs      uint64 `json:"docs"`
	Goget     uint64 `json:"goget"`
}

// ////////////////////////////////////////////////////////////////////////////////// //

var optMap = options.Map{
	OPT_CONFIG: {Value: "/etc/morpher-librato.knf"},
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Init main function
func Init() {
	_, errs := options.Parse(optMap)

	if len(errs) != 0 {
		fmt.Println("Arguments parsing errors:")

		for _, err := range errs {
			fmt.Printf("  %v\n", err)
		}

		os.Exit(1)
	}

	err := knf.Global(options.GetS(OPT_CONFIG))

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if !knf.GetB(MAIN_ENABLED, false) {
		os.Exit(0)
	}

	process()
}

// process start metrics processing
func process() {
	librato.Mail = knf.GetS(LIBRATO_MAIL)
	librato.Token = knf.GetS(LIBRATO_TOKEN)

	metrics, err := fetchMetrics(knf.GetS(METRICS_URL))

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	err = sendMetrics(metrics)

	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

// fetchMetrics fetch metrics from morpher server
func fetchMetrics(url string) (*Metrics, error) {
	resp, err := req.Request{
		URL:         url,
		Accept:      req.CONTENT_TYPE_JSON,
		AutoDiscard: true,
	}.Get()

	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Morpher return status code %d", resp.StatusCode)
	}

	metrics := &Metrics{}
	err = resp.JSON(metrics)

	if err != nil {
		return nil, err
	}

	return metrics, nil
}

// sendMetrics send metrics to librato
func sendMetrics(metrics *Metrics) error {
	now := time.Now()
	prefix := knf.GetS(LIBRATO_PREFIX)

	mt := time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		now.Minute(),
		0, 0, time.Local,
	).Unix()

	errs := librato.AddMetric(
		librato.Counter{MeasureTime: mt, Name: prefix + ".hits", Value: metrics.Hits},
		librato.Counter{MeasureTime: mt, Name: prefix + ".misses", Value: metrics.Misses},
		librato.Counter{MeasureTime: mt, Name: prefix + ".errors", Value: metrics.Errors},
		librato.Counter{MeasureTime: mt, Name: prefix + ".redirects", Value: metrics.Redirects},
		librato.Counter{MeasureTime: mt, Name: prefix + ".docs", Value: metrics.Docs},
		librato.Counter{MeasureTime: mt, Name: prefix + ".goget", Value: metrics.Goget},
	)

	if len(errs) == 0 {
		return nil
	}

	return errs[0]
}
