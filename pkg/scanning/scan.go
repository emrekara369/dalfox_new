package scanning

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emrekara369/dalfox_new/v2/pkg/har"

	"golang.org/x/term"

	"github.com/briandowns/spinner"
	"github.com/emrekara369/dalfox_new/v2/pkg/model"
	"github.com/emrekara369/dalfox_new/v2/pkg/optimization"
	"github.com/emrekara369/dalfox_new/v2/pkg/printing"
	"github.com/emrekara369/dalfox_new/v2/pkg/report"
	"github.com/emrekara369/dalfox_new/v2/pkg/verification"
	voltFile "github.com/hahwul/volt/file"
)

var (
	scanObject model.Scan
	s          = spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(os.Stderr))
)

func simplifyHeaders(header http.Header) map[string]string {
	simple := make(map[string]string)
	for k, v := range header {
		if len(v) > 0 {
			simple[k] = v[0]
		}
	}
	return simple
}

// Scan is main scanning function
func Scan(target string, options model.Options, sid string) (model.Result, error) {
	var scanResult model.Result
	mutex := &sync.Mutex{}
	options.ScanResult = scanResult
	scanResult.StartTime = time.Now()
	if !(options.Silence || options.NoSpinner) {
		time.Sleep(1 * time.Second) // Waiting log
		s.Prefix = " "
		s.Suffix = ""
		if !options.NoColor {
			s.Color("red", "bold")
		}
		if options.SpinnerObject != nil {
			s = options.SpinnerObject
		} else {
			options.SpinnerObject = s
		}
		s.Start()
	}

	scanObject := model.Scan{
		ScanID: sid,
		URL:    target,
	}
	if !(options.Silence && options.MulticastMode) {
		printing.DalLog("SYSTEM", "Start Scan 🦊", options)
		//printing.DalLog("SYSTEM-M", "Start Scan 🦊", options)
		if options.AllURLS > 0 {
			snow, _ := strconv.Atoi(sid)
			percent := fmt.Sprintf("%0.2f%%", float64(snow)/float64(options.AllURLS)*100)
			printing.DalLog("SYSTEM-M", "🦊 Start scan [SID:"+sid+"]["+sid+"/"+strconv.Itoa(options.AllURLS)+"]["+percent+"%] / URL: "+target, options)
		} else {
			printing.DalLog("SYSTEM-M", "🦊 Start scan [SID:"+sid+"] / URL: "+target, options)
		}
	}

	// query is XSS payloads
	query := make(map[*http.Request]map[string]string)

	// params is "param name":true  (reflected?)
	// 1: non-reflected , 2: reflected , 3: reflected-with-sc
	params := make(map[string][]string)

	// durls is url for dom xss
	var durls []string

	vStatus := make(map[string]bool)
	vStatus["pleasedonthaveanamelikethis_plz_plz"] = false

	// policy is "CSP":domain..
	policy := make(map[string]string)

	// set up a rate limit
	rl := newRateLimiter(time.Duration(options.Delay * 1000000))

	parsedURL, err := url.Parse(target)
	if err != nil {
		printing.DalLog("SYSTEM", "Not running "+target+" url", options)
		return scanResult, err
	}
	treq := optimization.GenerateNewRequest(target, "", options)
	//treq, terr := http.NewRequest("GET", target, nil)
	if treq == nil {
	} else {
		transport := getTransport(options)
		t := options.Timeout
		client := &http.Client{
			Timeout:   time.Duration(t) * time.Second,
			Transport: transport,
		}

		if !options.FollowRedirect {
			client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
		}

		tres, err := client.Do(treq)
		if err != nil {
			msg := fmt.Sprintf("not running %v", err)
			printing.DalLog("ERROR", msg, options)
			return scanResult, err
		}
		if options.IgnoreReturn != "" {
			rcode := strings.Split(options.IgnoreReturn, ",")
			tcode := strconv.Itoa(tres.StatusCode)
			for _, v := range rcode {
				if tcode == v {
					printing.DalLog("SYSTEM", "Not running "+target+" url from --ignore-return option", options)
					return scanResult, nil
				}
			}
		}

		defer tres.Body.Close()
		body, err := io.ReadAll(tres.Body)
		printing.DalLog("SYSTEM", "Valid target [ code:"+strconv.Itoa(tres.StatusCode)+" / size:"+strconv.Itoa(len(body))+" ]", options)
	}

	var wait sync.WaitGroup
	task := 3
	sa := "SA: ✓ "
	pa := "PA: ✓ "
	bav := "BAV: ✓ "
	if options.NoBAV {
		task = 2
		bav = ""
	}

	wait.Add(task)
	printing.DalLog("SYSTEM", "["+sa+pa+bav+"] Waiting for analysis 🔍", options)
	go func() {
		defer wait.Done()
		policy, options.PathReflection = StaticAnalysis(target, options, rl)
		sa = options.AuroraObject.Green(sa).String()
		printing.DalLog("SYSTEM", "["+sa+pa+bav+"] Waiting for analysis 🔍", options)
	}()
	go func() {
		defer wait.Done()
		params = ParameterAnalysis(target, options, rl)
		pa = options.AuroraObject.Green(pa).String()
		printing.DalLog("SYSTEM", "["+sa+pa+bav+"] Waiting for analysis 🔍", options)
	}()
	if !options.NoBAV {
		go func() {
			defer wait.Done()
			var bavWaitGroup sync.WaitGroup
			bavTask := 5
			bavWaitGroup.Add(bavTask)
			go func() {
				defer bavWaitGroup.Done()
				ESIIAnalysis(target, options, rl)
			}()
			go func() {
				defer bavWaitGroup.Done()
				SqliAnalysis(target, options, rl)
			}()
			go func() {
				defer bavWaitGroup.Done()
				SSTIAnalysis(target, options, rl)
			}()
			go func() {
				defer bavWaitGroup.Done()
				CRLFAnalysis(target, options, rl)
			}()
			go func() {
				defer bavWaitGroup.Done()
				OpenRedirectorAnalysis(target, options, rl)
			}()
			bavWaitGroup.Wait()
			bav = options.AuroraObject.Green(bav).String()
			printing.DalLog("SYSTEM", "["+sa+pa+bav+"] Waiting for analysis 🔍", options)
		}()
	}

	if options.NowURL != 0 && !options.Silence {
		s.Suffix = "  [" + strconv.Itoa(options.NowURL) + "/" + strconv.Itoa(options.AllURLS) + " Tasks] Scanning.."
	}

	if !(options.Silence || options.NoSpinner) {
		time.Sleep(1 * time.Second) // Waiting log
		s.Start()                   // Start the spinner
		//time.Sleep(3 * time.Second) // Run for some time to simulate work
	}
	wait.Wait()

	if !(options.Silence || options.NoSpinner) {
		s.Stop()
	}
	for k, v := range policy {
		if len(v) != 0 {
			if k == "BypassCSP" {
				printing.DalLog("WEAK", k+": "+v, options)
			} else {
				printing.DalLog("INFO", k+" is "+v, options)
			}
		}
	}
	for k, v := range options.PathReflection {
		if len(parsedURL.Path) == 0 {
			str := options.AuroraObject.Yellow("dalfoxpathtest").String()
			printing.DalLog("INFO", "Reflected PATH '/"+str+"' => "+v+"]", options)
		} else {
			split := strings.Split(parsedURL.Path, "/")
			if len(split) > k+1 {
				split[k+1] = options.AuroraObject.Yellow("dalfoxpathtest").String()
				tempURL := strings.Join(split, "/")
				printing.DalLog("INFO", "Reflected PATH '"+tempURL+"' => "+v+"]", options)
			}
		}
	}

	for k, v := range params {
		temp := model.ParamResult{
			Name:      k,
			Reflected: false,
		}

		if len(v) != 0 {
			code, vv := v[len(v)-1], v[:len(v)-1]
			char := strings.Join(vv, "  ")
			//x, a = a[len(a)-1], a[:len(a)-1]
			printing.DalLog("INFO", "Reflected "+k+" param => "+char, options)
			printing.DalLog("CODE", code, options)
			arr := strings.Split(char, "  ")
			for _, value := range arr {
				if strings.Contains(value, "PTYPE:") {
					splitedValue := strings.Split(value, " ")
					temp.Type = splitedValue[1]
				} else if strings.Contains(value, "Injected:") {
					splitedValue := strings.Split(value, " ")
					temp.ReflectedPoint = splitedValue[1]
				} else {
					temp.Chars = append(temp.Chars, value)
				}
			}
			temp.ReflectedCode = code
			temp.Reflected = true
		}
		scanResult.Params = append(scanResult.Params, temp)
	}

	if !options.OnlyDiscovery {
		// XSS Scanning
		printing.DalLog("SYSTEM", "Generate XSS payload and optimization.Optimization.. 🛠", options)
		// optimization.Optimization..

		/*
			k: parama name
			v: pattern [injs, inhtml, ' < > ]
			av: reflected type, valid char
		*/

		// set vStatus
		for k, _ := range params {
			vStatus[k] = false
		}

		// set path base XSS
		for k, v := range options.PathReflection {
			if strings.Contains(v, "Injected:") {
				// Injected pattern
				injectedPoint := strings.Split(v, "/")
				injectedPoint = injectedPoint[1:]
				for _, ip := range injectedPoint {
					var arr []string
					if strings.Contains(ip, "inJS") {
						arr = optimization.SetPayloadValue(getInJsPayload(ip), options)
					}
					if strings.Contains(ip, "inHTML") {
						arr = optimization.SetPayloadValue(getHTMLPayload(ip), options)
					}
					if strings.Contains(ip, "inATTR") {
						arr = optimization.SetPayloadValue(getAttrPayload(ip), options)
					}
					for _, avv := range arr {
						var tempURL string
						if len(parsedURL.Path) == 0 {
							tempURL = target + "/" + avv
						} else {
							split := strings.Split(target, "/")
							split[k+3] = split[k+3] + avv
							tempURL = strings.Join(split, "/")
						}
						// Add Path XSS Query
						tq, tm := optimization.MakeRequestQuery(tempURL, "", "", ip, "toAppend", "NaN", options)
						tm["payload"] = avv
						query[tq] = tm
					}
				}
			}
		}

		// Custom Payload
		if isAllowType(policy["Content-Type"]) && options.CustomPayloadFile != "" {
			ff, err := voltFile.ReadLinesOrLiteral(options.CustomPayloadFile)
			if err != nil {
				printing.DalLog("SYSTEM", "Custom XSS payload load fail..", options)
			} else {
				for _, customPayload := range ff {
					if customPayload != "" {
						for k, v := range params {
							if optimization.CheckInspectionParam(options, k) {
								ptype := ""
								for _, av := range v {
									if strings.Contains(av, "PTYPE:") {
										ptype = GetPType(av)
									}
								}
								encoders := []string{
									"NaN",
									"urlEncode",
									"urlDoubleEncode",
									"htmlEncode",
								}
								for _, encoder := range encoders {
									tq, tm := optimization.MakeRequestQuery(target, k, customPayload, "inHTML"+ptype, "toAppend", encoder, options)
									query[tq] = tm
								}
							}
						}
					}
				}
				printing.DalLog("SYSTEM", "Added your "+strconv.Itoa(len(ff))+" custom xss payload", options)
			}
		}

		if isAllowType(policy["Content-Type"]) && !options.OnlyCustomPayload {
			// Set common payloads
			cu, err := url.Parse(target)
			var cp url.Values
			var cpd url.Values
			var cpArr []string
			var cpdArr []string
			hashParam := false
			if err == nil {
				if options.Data == "" {
					cp, _ = url.ParseQuery(cu.RawQuery)
					if len(cp) == 0 {
						cp, _ = url.ParseQuery(cu.Fragment)
						hashParam = true
					}
				} else {
					cp, _ = url.ParseQuery(cu.RawQuery)
					cpd, _ = url.ParseQuery(options.Data)
				}
			}

			for v := range cp {
				if optimization.CheckInspectionParam(options, v) {
					cpArr = append(cpArr, v)
					arc := optimization.SetPayloadValue(getCommonPayload(), options)
					for _, avv := range arc {
						encoders := []string{
							"NaN",
							"urlEncode",
							"urlDoubleEncode",
							"htmlEncode",
						}
						for _, encoder := range encoders {
							tq, tm := optimization.MakeRequestQuery(target, v, avv, "inHTML-URL", "toAppend", encoder, options)
							query[tq] = tm
						}
					}
				}
			}

			for v := range cpd {
				if optimization.CheckInspectionParam(options, v) {
					cpdArr = append(cpdArr, v)
					arc := optimization.SetPayloadValue(getCommonPayload(), options)
					for _, avv := range arc {
						encoders := []string{
							"NaN",
							"urlEncode",
							"urlDoubleEncode",
							"htmlEncode",
						}
						for _, encoder := range encoders {
							tq, tm := optimization.MakeRequestQuery(target, v, avv, "inHTML-FORM", "toAppend", encoder, options)
							query[tq] = tm
						}
					}
				}
			}

			// DOM XSS payload
			var dlst []string
			if options.UseHeadless {
				if options.UseDeepDXSS {
					dlst = getDeepDOMXSPayload()
				} else {
					dlst = getDOMXSSPayload()
				}
				dpayloads := optimization.SetPayloadValue(dlst, options)
				for v := range cp {
					if optimization.CheckInspectionParam(options, v) {
						// loop payload list
						if len(params[v]) == 0 {
							for _, dpayload := range dpayloads {
								var durl string
								u, _ := url.Parse(target)
								dp, _ := url.ParseQuery(u.RawQuery)
								if hashParam {
									dp, _ = url.ParseQuery(u.Fragment)
									dp.Set(v, dpayload)
									u.Fragment, _ = url.QueryUnescape(dp.Encode())
								} else {
									dp.Set(v, dpayload)
									u.RawQuery = dp.Encode()
								}
								durl = u.String()
								durls = append(durls, durl)
							}
						}
					}
				}
				for v := range cpd {
					if optimization.CheckInspectionParam(options, v) {
						// loop payload list
						if len(params[v]) == 0 {
							for _, dpayload := range dpayloads {
								var durl string
								u, _ := url.Parse(target)
								dp, _ := url.ParseQuery(u.RawQuery)
								if hashParam {
									dp, _ = url.ParseQuery(u.Fragment)
									dp.Set(v, dpayload)
									u.Fragment, _ = url.QueryUnescape(dp.Encode())
								} else {
									dp.Set(v, dpayload)
									u.RawQuery = dp.Encode()
								}
								durl = u.String()
								durls = append(durls, durl)
							}
						}
					}
				}
			}

			// Set param base xss
			for k, v := range params {
				if optimization.CheckInspectionParam(options, k) {
					ptype := ""
					chars := GetSpecialChar()
					var badchars []string

					for _, av := range v {
						if indexOf(av, chars) == -1 {
							badchars = append(badchars, av)
						}
						if strings.Contains(av, "PTYPE:") {
							ptype = GetPType(av)
						}

						if strings.Contains(av, "Injected:") {
							// Injected pattern
							injectedPoint := strings.Split(av, "/")
							injectedPoint = injectedPoint[1:]
							injectedChars := params[k][:len(params[k])-1]
							for _, ip := range injectedPoint {
								var arr []string
								if strings.Contains(ip, "inJS") {
									checkInJS := false
									if strings.Contains(ip, "double") {
										for _, injectedChar := range injectedChars {
											if strings.Contains(injectedChar, "\"") {
												checkInJS = true
											}
										}
									}
									if strings.Contains(ip, "single") {
										for _, injectedChar := range injectedChars {
											if strings.Contains(injectedChar, "'") {
												checkInJS = true
											}
										}
									}
									if checkInJS {
										arr = optimization.SetPayloadValue(getInJsPayload(ip), options)
									} else {
										arr = optimization.SetPayloadValue(getInJsBreakScriptPayload(ip), options)
									}
								}
								if strings.Contains(ip, "inHTML") {
									arr = optimization.SetPayloadValue(getHTMLPayload(ip), options)
								}
								if strings.Contains(ip, "inATTR") {
									arr = optimization.SetPayloadValue(getAttrPayload(ip), options)
								}
								for _, avv := range arr {
									if optimization.Optimization(avv, badchars) {
										encoders := []string{
											"NaN",
											"urlEncode",
											"urlDoubleEncode",
											"htmlEncode",
										}
										for _, encoder := range encoders {
											tq, tm := optimization.MakeRequestQuery(target, k, avv, ip+ptype, "toAppend", encoder, options)
											query[tq] = tm
										}
									}
								}
							}
						}
					}
					// common XSS for new param
					arc := optimization.SetPayloadValue(getCommonPayload(), options)
					for _, avv := range arc {
						if !containsFromArray(cpArr, k) {
							if optimization.Optimization(avv, badchars) {
								encoders := []string{
									"NaN",
									"urlEncode",
									"urlDoubleEncode",
									"htmlEncode",
								}
								for _, encoder := range encoders {
									tq, tm := optimization.MakeRequestQuery(target, k, avv, "inHTML"+ptype, "toAppend", encoder, options)
									query[tq] = tm
								}
							}
						}
					}
				}

			}
		} else {
			printing.DalLog("SYSTEM", "Type is '"+policy["Content-Type"]+"', It does not test except customized payload (custom/blind).", options)
		}

		// Blind payload
		if options.BlindURL != "" {
			bpayloads := getBlindPayload()

			//strings.HasPrefix("foobar", "foo") // true
			var bcallback string

			if strings.HasPrefix(options.BlindURL, "https://") || strings.HasPrefix(options.BlindURL, "http://") {
				bcallback = options.BlindURL
			} else {
				bcallback = "//" + options.BlindURL
			}

			for _, bpayload := range bpayloads {
				// header base blind xss
				bp := strings.Replace(bpayload, "CALLBACKURL", bcallback, 10)
				tq, tm := optimization.MakeHeaderQuery(target, "Referer", bp, options)
				tm["payload"] = "toBlind"
				query[tq] = tm
			}

			// loop parameter list
			for k, v := range params {
				if optimization.CheckInspectionParam(options, k) {
					ptype := ""
					for _, av := range v {
						if strings.Contains(av, "PTYPE:") {
							ptype = GetPType(av)
						}
					}
					// loop payload list
					for _, bpayload := range bpayloads {
						// Add plain XSS Query
						bp := strings.Replace(bpayload, "CALLBACKURL", bcallback, 10)
						encoders := []string{
							"NaN",
							"urlEncode",
							"urlDoubleEncode",
							"htmlEncode",
						}
						for _, encoder := range encoders {
							tq, tm := optimization.MakeRequestQuery(target, k, bp, "toBlind"+ptype, "toAppend", encoder, options)
							tm["payload"] = "toBlind"
							query[tq] = tm
						}
					}
				}
			}
			printing.DalLog("SYSTEM", "Added your blind XSS ("+options.BlindURL+")", options)
		}

		// Remote Payloads
		if options.RemotePayloads != "" {
			rp := strings.Split(options.RemotePayloads, ",")
			for _, endpoint := range rp {
				var payload []string
				var line string
				var size string
				if endpoint == "portswigger" {
					payload, line, size = getPortswiggerPayload()
				}
				if endpoint == "payloadbox" {
					payload, line, size = getPayloadBoxPayload()
				}
				if line != "" {
					printing.DalLog("INFO", "A '"+endpoint+"' payloads has been loaded ["+line+"L / "+size+"]               ", options)
					for _, remotePayload := range payload {
						if remotePayload != "" {
							for k, v := range params {
								if optimization.CheckInspectionParam(options, k) {
									ptype := ""
									for _, av := range v {
										if strings.Contains(av, "PTYPE:") {
											ptype = GetPType(av)
										}
									}
									encoders := []string{
										"NaN",
										"urlEncode",
										"urlDoubleEncode",
										"htmlEncode",
									}
									for _, encoder := range encoders {
										tq, tm := optimization.MakeRequestQuery(target, k, remotePayload, "inHTML"+ptype, "toAppend", encoder, options)
										query[tq] = tm
									}
								}
							}
						}
					}
				} else {
					printing.DalLog("SYSTEM", endpoint+" payload load fail..", options)
				}
			}
		}

		printing.DalLog("SYSTEM", "Start XSS Scanning.. with "+strconv.Itoa(len(query))+" queries 🗡", options)
		queryCount := 0
		printing.DalLog("SYSTEM", "[ Make "+strconv.Itoa(options.Concurrence)+" workers ] [ Allocated "+strconv.Itoa(len(query))+" queries ]", options)

		if !(options.Silence || options.NoSpinner) {
			s.Start() // Start the spinner
			//time.Sleep(3 * time.Second) // Run for some time to simulate work
		}

		showR := false
		showV := false
		if options.OnlyPoC != "" {
			_, showR, showV = printing.CheckToShowPoC(options.OnlyPoC)
		} else {
			showR = true
			showV = true
		}

		// make waiting group
		var wg sync.WaitGroup
		// set concurrency
		concurrency := options.Concurrence
		// make reqeust channel
		queries := make(chan Queries)

		if options.UseHeadless {
			// start DOM XSS checker
			wg.Add(1)
			go func() {
				dconcurrency := options.Concurrence / 2
				if dconcurrency < 1 {
					dconcurrency = 1
				}
				if dconcurrency > 10 {
					dconcurrency = 10
				}
				dchan := make(chan string)
				var wgg sync.WaitGroup
				for i := 0; i < dconcurrency; i++ {
					wgg.Add(1)
					go func() {
						for v := range dchan {
							if CheckXSSWithHeadless(v, options) {
								mutex.Lock()
								printing.DalLog("VULN", "Triggered XSS Payload (found dialog in headless)", options)
								poc := model.PoC{
									Type:       "V",
									InjectType: "headless",
									Method:     "GET",
									Data:       v,
									Param:      "",
									Payload:    "",
									Evidence:   "",
									CWE:        "CWE-79",
									Severity:   "High",
									PoCType:    options.PoCType,
									MessageStr: "Triggered XSS Payload (found dialog in headless)",
									StatusCode: 200,
									ResponseHeaders: map[string]string{
										"Content-Type": "text/html; charset=UTF-8",
										"Server":       "dalfox-headless",
									},
								}
									//MessageID:  -1, // we can't do HAR here because it's using chromedp}

								if showV {
									if options.Format == "json" {
										pocj, _ := json.Marshal(poc)
										printing.DalLog("PRINT", string(pocj)+",", options)
									} else {
										pocs := "[" + poc.Type + "][" + poc.Method + "][" + poc.InjectType + "] " + poc.Data
										printing.DalLog("PRINT", pocs, options)
									}
								}
								if options.FoundAction != "" {
									foundAction(options, target, v, "VULN")
								}
								scanObject.Results = append(scanObject.Results, poc)
								scanResult.PoCs = append(scanResult.PoCs, poc)
								mutex.Unlock()
							}
							queryCount = queryCount + 1
						}
						wgg.Done()
					}()
				}
				for _, dchanData := range durls {
					dchan <- dchanData
				}
				close(dchan)
				wgg.Wait()
				wg.Done()
			}()
		}
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				for reqJob := range queries {
					if checkVStatus(vStatus) {
						// if when all param found xss, break. (for passing speed up)
						continue
					}
					// quires.request : http.Request
					// queries.metadata : map[string]string
					k := reqJob.request
					v := reqJob.metadata
					checkVtype := false
					if checkPType(v["type"]) {
						checkVtype = true
					}

					if vStatus[v["param"]] == false || checkVtype {
						rl.Block(k.Host)
						resbody, resp, vds, vrs, err := SendReq(k, v["payload"], options)
						abs := optimization.Abstraction(resbody, v["payload"])
						if vrs {
							if !containsFromArray(abs, v["type"]) && !strings.Contains(v["type"], "inHTML") {
								vrs = false
							}
						}
						if err == nil {
							if checkPType(v["type"]) {
								if strings.Contains(v["type"], "inJS") {
									if vrs {
										protected := false
										if verification.VerifyReflection(resbody, "\\"+v["payload"]) {
											if !strings.Contains(v["payload"], "\\") {
												protected = true
											}
										}
										if !protected {
											if vStatus[v["param"]] == false {
												if options.UseHeadless {
													if CheckXSSWithHeadless(k.URL.String(), options) {
														mutex.Lock()
														printing.DalLog("VULN", "Triggered XSS Payload (found dialog in headless)", options)
														poc := model.PoC{
															Type:       "V",
															InjectType: v["type"],
															Method:     k.Method,
															Data:       k.URL.String(),
															Param:      v["param"],
															Payload:    "",
															Evidence:   "",
															CWE:        "CWE-79",
															Severity:   "High",
															PoCType:    options.PoCType,
															MessageID:  har.MessageIDFromRequest(k),
															MessageStr: "Triggered XSS Payload (found dialog in headless)",
															StatusCode: resp.StatusCode,
															ResponseHeaders: simplifyHeaders(resp.Header),}
														poc.Data = MakePoC(poc.Data, k, options)

														if options.OutputRequest {
															reqDump, err := httputil.DumpRequestOut(k, true)
															if err == nil {
																poc.RawHTTPRequest = string(reqDump)
																printing.DalLog("CODE", "\n"+string(reqDump), options)
															}
														}

														if options.OutputResponse {
															poc.RawHTTPResponse = resbody
															printing.DalLog("CODE", string(resbody), options)
														}

														if showV {
															if options.Format == "json" {
																pocj, _ := json.Marshal(poc)
																printing.DalLog("PRINT", string(pocj)+",", options)
															} else {
																pocs := "[" + poc.Type + "][" + poc.Method + "][" + poc.InjectType + "] " + poc.Data
																printing.DalLog("PRINT", pocs, options)
															}
														}
														vStatus[v["param"]] = true
														if options.FoundAction != "" {
															foundAction(options, target, k.URL.String(), "VULN")
														}
														scanObject.Results = append(scanObject.Results, poc)
														scanResult.PoCs = append(scanResult.PoCs, poc)
														mutex.Unlock()
													} else {
														mutex.Lock()
														if options.FoundAction != "" {
															foundAction(options, target, k.URL.String(), "WEAK")
														}
														printing.DalLog("WEAK", "Reflected Payload in JS: "+v["param"]+"="+v["payload"], options)
														poc := model.PoC{
															Type:       "R",
															InjectType: v["type"],
															Method:     k.Method,
															Data:       k.URL.String(),
															Param:      v["param"],
															Payload:    "",
															Evidence:   "",
															CWE:        "CWE-79",
															Severity:   "Medium",
															PoCType:    options.PoCType,
															MessageID:  har.MessageIDFromRequest(k),
															MessageStr: "Reflected Payload in JS: " + v["param"] + "=" + v["payload"],
															StatusCode: resp.StatusCode,
															ResponseHeaders: simplifyHeaders(resp.Header),}
														poc.Data = MakePoC(poc.Data, k, options)
														if options.OutputRequest {
															reqDump, err := httputil.DumpRequestOut(k, true)
															if err == nil {
																poc.RawHTTPRequest = string(reqDump)
																printing.DalLog("CODE", "\n"+string(reqDump), options)
															}
														}

														if options.OutputResponse {
															poc.RawHTTPResponse = resbody
															printing.DalLog("CODE", string(resbody), options)
														}

														scanObject.Results = append(scanObject.Results, poc)
														scanResult.PoCs = append(scanResult.PoCs, poc)
														mutex.Unlock()
													}
												} else {
													mutex.Lock()
													code := CodeView(resbody, v["payload"])
													printing.DalLog("WEAK", "Reflected Payload in JS: "+v["param"]+"="+v["payload"], options)
													printing.DalLog("CODE", code, options)
													poc := model.PoC{
														Type:       "R",
														InjectType: v["type"],
														Method:     k.Method,
														Data:       k.URL.String(),
														Param:      v["param"],
														Payload:    v["payload"],
														Evidence:   code,
														CWE:        "CWE-79",
														Severity:   "Medium",
														PoCType:    options.PoCType,
														MessageID:  har.MessageIDFromRequest(k),
														MessageStr: "Reflected Payload in JS: " + v["param"] + "=" + v["payload"],
														StatusCode: resp.StatusCode,
														ResponseHeaders: simplifyHeaders(resp.Header),}
													poc.Data = MakePoC(poc.Data, k, options)
													if options.OutputRequest {
														reqDump, err := httputil.DumpRequestOut(k, true)
														if err == nil {
															poc.RawHTTPRequest = string(reqDump)
															printing.DalLog("CODE", "\n"+string(reqDump), options)
														}
													}

													if options.OutputResponse {
														poc.RawHTTPResponse = resbody
														printing.DalLog("CODE", string(resbody), options)
													}

													if showR {
														if options.Format == "json" {
															pocj, _ := json.Marshal(poc)
															printing.DalLog("PRINT", string(pocj)+",", options)
														} else {
															pocs := "[" + poc.Type + "][" + poc.Method + "][" + poc.InjectType + "] " + poc.Data
															printing.DalLog("PRINT", pocs, options)
														}
													}
													if options.FoundAction != "" {
														foundAction(options, target, k.URL.String(), "WEAK")
													}
													scanObject.Results = append(scanObject.Results, poc)
													scanResult.PoCs = append(scanResult.PoCs, poc)
													mutex.Unlock()
												}
											}
										}
									}
								} else if strings.Contains(v["type"], "inATTR") {
									if vds {
										mutex.Lock()
										if vStatus[v["param"]] == false {
											code := CodeView(resbody, v["payload"])
											printing.DalLog("VULN", "Triggered XSS Payload (found DOM Object): "+v["param"]+"="+v["payload"], options)
											printing.DalLog("CODE", code, options)
											poc := model.PoC{
												Type:       "V",
												InjectType: v["type"],
												Method:     k.Method,
												Data:       k.URL.String(),
												Param:      v["param"],
												Payload:    v["payload"],
												Evidence:   code,
												CWE:        "CWE-83",
												Severity:   "High",
												PoCType:    options.PoCType,
												MessageID:  har.MessageIDFromRequest(k),
												MessageStr: "Triggered XSS Payload (found DOM Object): " + v["param"] + "=" + v["payload"],
												StatusCode: resp.StatusCode,
												ResponseHeaders: simplifyHeaders(resp.Header),}
											poc.Data = MakePoC(poc.Data, k, options)
											if options.OutputRequest {
												reqDump, err := httputil.DumpRequestOut(k, true)
												if err == nil {
													poc.RawHTTPRequest = string(reqDump)
													printing.DalLog("CODE", "\n"+string(reqDump), options)
												}
											}

											if options.OutputResponse {
												poc.RawHTTPResponse = resbody
												printing.DalLog("CODE", string(resbody), options)
											}

											if showV {
												if options.Format == "json" {
													pocj, _ := json.Marshal(poc)
													printing.DalLog("PRINT", string(pocj)+",", options)
												} else {
													pocs := "[" + poc.Type + "][" + poc.Method + "][" + poc.InjectType + "] " + poc.Data
													printing.DalLog("PRINT", pocs, options)
												}
											}
											vStatus[v["param"]] = true
											if options.FoundAction != "" {
												foundAction(options, target, k.URL.String(), "VULN")
											}
											scanObject.Results = append(scanObject.Results, poc)
											scanResult.PoCs = append(scanResult.PoCs, poc)
										}
										mutex.Unlock()
									} else if vrs {
										mutex.Lock()
										if vStatus[v["param"]] == false {
											code := CodeView(resbody, v["payload"])
											printing.DalLog("WEAK", "Reflected Payload in Attribute: "+v["param"]+"="+v["payload"], options)
											printing.DalLog("CODE", code, options)
											poc := model.PoC{
												Type:       "R",
												InjectType: v["type"],
												Method:     k.Method,
												Data:       k.URL.String(),
												Param:      v["param"],
												Payload:    v["payload"],
												Evidence:   code,
												CWE:        "CWE-83",
												Severity:   "Medium",
												PoCType:    options.PoCType,
												MessageID:  har.MessageIDFromRequest(k),
												MessageStr: "Reflected Payload in Attribute: " + v["param"] + "=" + v["payload"],
												StatusCode: resp.StatusCode,
												ResponseHeaders: simplifyHeaders(resp.Header),}
											poc.Data = MakePoC(poc.Data, k, options)
											if options.OutputRequest {
												reqDump, err := httputil.DumpRequestOut(k, true)
												if err == nil {
													poc.RawHTTPRequest = string(reqDump)
													printing.DalLog("CODE", "\n"+string(reqDump), options)
												}
											}

											if options.OutputResponse {
												poc.RawHTTPResponse = resbody
												printing.DalLog("CODE", string(resbody), options)
											}

											if showR {
												if options.Format == "json" {
													pocj, _ := json.Marshal(poc)
													printing.DalLog("PRINT", string(pocj)+",", options)
												} else {
													pocs := "[" + poc.Type + "][" + poc.Method + "][" + poc.InjectType + "] " + poc.Data
													printing.DalLog("PRINT", pocs, options)
												}
											}
											if options.FoundAction != "" {
												foundAction(options, target, k.URL.String(), "WEAK")
											}
											scanObject.Results = append(scanObject.Results, poc)
											scanResult.PoCs = append(scanResult.PoCs, poc)
										}
										mutex.Unlock()
									}
								} else {
									if vds {
										mutex.Lock()
										if vStatus[v["param"]] == false {
											code := CodeView(resbody, v["payload"])
											printing.DalLog("VULN", "Triggered XSS Payload (found DOM Object): "+v["param"]+"="+v["payload"], options)
											printing.DalLog("CODE", code, options)
											poc := model.PoC{
												Type:       "V",
												InjectType: v["type"],
												Method:     k.Method,
												Data:       k.URL.String(),
												Param:      v["param"],
												Payload:    v["payload"],
												Evidence:   code,
												CWE:        "CWE-79",
												Severity:   "High",
												PoCType:    options.PoCType,
												MessageID:  har.MessageIDFromRequest(k),
												MessageStr: "Triggered XSS Payload (found DOM Object): " + v["param"] + "=" + v["payload"],
												StatusCode: resp.StatusCode,
												ResponseHeaders: simplifyHeaders(resp.Header),}
											poc.Data = MakePoC(poc.Data, k, options)
											if options.OutputRequest {
												reqDump, err := httputil.DumpRequestOut(k, true)
												if err == nil {
													poc.RawHTTPRequest = string(reqDump)
													printing.DalLog("CODE", "\n"+string(reqDump), options)
												}
											}

											if options.OutputResponse {
												poc.RawHTTPResponse = resbody
												printing.DalLog("CODE", string(resbody), options)
											}

											if showV {
												if options.Format == "json" {
													pocj, _ := json.Marshal(poc)
													printing.DalLog("PRINT", string(pocj)+",", options)
												} else {
													pocs := "[" + poc.Type + "][" + poc.Method + "][" + poc.InjectType + "] " + poc.Data
													printing.DalLog("PRINT", pocs, options)
												}
											}
											vStatus[v["param"]] = true
											if options.FoundAction != "" {
												foundAction(options, target, k.URL.String(), "VULN")
											}
											scanObject.Results = append(scanObject.Results, poc)
											scanResult.PoCs = append(scanResult.PoCs, poc)
										}
										mutex.Unlock()
									} else if vrs {
										mutex.Lock()
										if vStatus[v["param"]] == false {
											code := CodeView(resbody, v["payload"])
											printing.DalLog("WEAK", "Reflected Payload in HTML: "+v["param"]+"="+v["payload"], options)
											printing.DalLog("CODE", code, options)
											poc := model.PoC{
												Type:       "R",
												InjectType: v["type"],
												Method:     k.Method,
												Data:       k.URL.String(),
												Param:      v["param"],
												Payload:    v["payload"],
												Evidence:   code,
												CWE:        "CWE-79",
												Severity:   "Medium",
												PoCType:    options.PoCType,
												MessageID:  har.MessageIDFromRequest(k),
												MessageStr: "Reflected Payload in HTML: " + v["param"] + "=" + v["payload"],
												StatusCode: resp.StatusCode,
												ResponseHeaders: simplifyHeaders(resp.Header),}
											poc.Data = MakePoC(poc.Data, k, options)
											if options.OutputRequest {
												reqDump, err := httputil.DumpRequestOut(k, true)
												if err == nil {
													poc.RawHTTPRequest = string(reqDump)
													printing.DalLog("CODE", "\n"+string(reqDump), options)
												}
											}

											if options.OutputResponse {
												poc.RawHTTPResponse = resbody
												printing.DalLog("CODE", string(resbody), options)
											}

											if showR {
												if options.Format == "json" {
													pocj, _ := json.Marshal(poc)
													printing.DalLog("PRINT", string(pocj)+",", options)
												} else {
													pocs := "[" + poc.Type + "][" + poc.Method + "][" + poc.InjectType + "] " + poc.Data
													printing.DalLog("PRINT", pocs, options)
												}
											}
											if options.FoundAction != "" {
												foundAction(options, target, k.URL.String(), "WEAK")
											}
											scanObject.Results = append(scanObject.Results, poc)
											scanResult.PoCs = append(scanResult.PoCs, poc)
										}
										mutex.Unlock()
									}
								}
							}
						}
					}
					mutex.Lock()
					queryCount = queryCount + 1

					if !(options.Silence || options.NoSpinner) {
						s.Lock()
						var msg string
						if vStatus[v["param"]] == false {
							if options.UseHeadless {
								msg = "Testing \"" + v["param"] + "\" param and waiting headless"
							} else {
								msg = "Testing \"" + v["param"] + "\" param"
							}
						} else {
							if options.UseHeadless {
								msg = "Passing \"" + v["param"] + "\" param queries and waiting headless"
							} else {
								msg = "Passing \"" + v["param"] + "\" param queries"
							}
						}

						percent := fmt.Sprintf("%0.2f%%", (float64(queryCount)/float64(len(query)+len(durls)))*100)
						if options.NowURL == 0 {
							s.Suffix = "  [" + strconv.Itoa(queryCount) + "/" + strconv.Itoa(len(query)+len(durls)) + " Queries][" + percent + "] " + msg
						} else if !options.Silence {
							percent2 := fmt.Sprintf("%0.2f%%", (float64(options.NowURL) / float64(options.AllURLS) * 100))
							s.Suffix = "  [" + strconv.Itoa(queryCount) + "/" + strconv.Itoa(len(query)+len(durls)) + " Queries][" + percent + "][" + strconv.Itoa(options.NowURL) + "/" + strconv.Itoa(options.AllURLS) + " Tasks][" + percent2 + "] " + msg
						}
						//s.Suffix = " Waiting routines.. (" + strconv.Itoa(queryCount) + " / " + strconv.Itoa(len(query)) + ") reqs"
						s.Unlock()
					}
					mutex.Unlock()
				}
				wg.Done()
			}()
		}

		// Send testing query to quires channel
		for k, v := range query {
			queries <- Queries{
				request:  k,
				metadata: v,
			}
		}
		close(queries)
		wg.Wait()
		if !(options.Silence || options.NoSpinner) {
			s.Stop()
		}
	}

	options.Scan[sid] = scanObject
	scanResult.EndTime = time.Now()
	scanResult.Duration = scanResult.EndTime.Sub(scanResult.StartTime)
	if !(options.Silence && options.MulticastMode) {
		if term.IsTerminal(0) {
			width, _, err := term.GetSize(0)
			if err == nil {
				var dash string
				for i := 0; i < width-5; i++ {
					dash = dash + "-"
				}
				printing.DalLog("SYSTEM-M", dash, options)
			}
		}
		printing.DalLog("SYSTEM-M", "[duration: "+scanResult.Duration.String()+"][issues: "+strconv.Itoa(len(scanResult.PoCs))+"] Finish Scan!", options)
	}
	if options.ReportBool {
		printing.DalLog("SYSTEM-M", "Report\n", options)
		if options.ReportFormat == "json" {
			jobject, err := json.MarshalIndent(scanResult, "", " ")
			if err == nil {
				fmt.Println(string(jobject))
			}
		} else {
			report.GenerateReport(scanResult, options)
		}
	}
	return scanResult, nil
}
