package server

import (
	"strings"

	dalfox "github.com/emrekara369/dalfox_new/v2/lib"
	"github.com/emrekara369/dalfox_new/v2/pkg/model"
	scan "github.com/emrekara369/dalfox_new/v2/pkg/scanning"
	vlogger "github.com/hahwul/volt/logger"
)

// ScanFromAPI is scanning dalfox with REST API
// @Summary scan
// @Description add dalfox scan
// @Accept  json
// @Produce  json
// @Param data body Req true "json data"
// @Success 200 {object} Res
// @Router /scan [post]
func ScanFromAPI(url string, rqOptions model.Options, options model.Options, sid string) {
	vLog := vlogger.GetLogger(options.Debug)
	target := dalfox.Target{
		URL:     url,
		Method:  rqOptions.Method,
		Options: dalfox.Options{},
	}
	newOptions := dalfox.Initialize(target, target.Options)
	newOptions.Scan = options.Scan
	if rqOptions.Method != "" {
		newOptions.Method = options.Method
	} else {
		newOptions.Method = "GET"
	}
	escapedURL := strings.Replace(url, "\n", "", -1)
	escapedURL = strings.Replace(escapedURL, "\r", "", -1)
	vLog.WithField("data1", sid).Debug(escapedURL)
	vLog.WithField("data1", sid).Debug(newOptions)
	_, _ = scan.Scan(url, newOptions, sid)
}

// GetScan is get scan information
// @Summary scan
// @Description get scan info
// @Accept  json
// @Produce  json
// @Param scanid path string true "scan id"
// @Success 200 {object} Res
// @Router /scan/{scanid} [get]
func GetScan(sid string, options model.Options) model.Scan {
	return options.Scan[sid]
}

// GetScans is list of scan
// @Summary scan
// @Description show scan list
// @Accept  json
// @Produce  json
// @Success 200 {array} string
// @Router /scans [get]
func GetScans() {

}
