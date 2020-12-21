package healthcheck

// ////////////////////////////////////////////////////////////////////////////////// //
//                                                                                    //
//                         Copyright (c) 2020 ESSENTIAL KAOS                          //
//      Apache License, Version 2.0 <https://www.apache.org/licenses/LICENSE-2.0>     //
//                                                                                    //
// ////////////////////////////////////////////////////////////////////////////////// //

import (
	"time"

	"pkg.re/essentialkaos/ek.v12/log"

	"github.com/valyala/fasthttp"
)

// ////////////////////////////////////////////////////////////////////////////////// //

type Checker struct {
	client *fasthttp.Client
	url    string
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Start starts healthcheck pinger
func Start(url string, period time.Duration) {
	checker := &Checker{
		url: url,
		client: &fasthttp.Client{
			Name:                "PKGRE Morpher/4",
			MaxIdleConnDuration: 5 * time.Second,
			ReadTimeout:         5 * time.Second,
			WriteTimeout:        3 * time.Second,
			MaxConnsPerHost:     10,
		},
	}

	go checker.Run(period)
}

// ////////////////////////////////////////////////////////////////////////////////// //

// Run starts loop for sending pings
func (c *Checker) Run(period time.Duration) {
	for range time.NewTicker(period).C {
		req := fasthttp.AcquireRequest()

		req.SetRequestURI(c.url)
		req.Header.SetMethod("HEAD")

		err := c.client.Do(req, nil)

		if err != nil {
			log.Error("Can't send healthcheck request: %v", err)
		}
	}
}
