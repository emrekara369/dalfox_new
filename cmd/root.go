package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/emrekara369/dalfox_new/v2/pkg/har"

	"github.com/emrekara369/dalfox_new/v2/pkg/model"
	"github.com/emrekara369/dalfox_new/v2/pkg/printing"
	"github.com/logrusorgru/aurora"
	"github.com/spf13/cobra"
)

var cfgFile string
var optionsStr = make(map[string]string)
var optionsBool = make(map[string]bool)
var header, p, ignoreParams []string
var config, cookie, data, customPayload, userAgent, blind, output, format, foundAction, proxy, grep, cookieFromRaw string
var harFilePath string
var ignoreReturn, miningWord, method, customAlertValue, customAlertType, remotePayloads, remoteWordlists string
var timeout, concurrence, delay int
var onlyDiscovery, silence, followRedirect, mining, findingDOM, noColor, noSpinner, onlyCustomPayload, debug, useDeepDXSS, outputAll bool
var options model.Options
var skipMiningDom, skipMiningDict, skipMiningAll, skipXSSScan, skipBAV, skipGrep, skipHeadless, wafEvasion, reportBool, outputRequest, outputResponse bool
var onlyPoC, foundActionShell, pocType, reportFormat string

var rootCmd = &cobra.Command{
	Use: "dalfox",
	Run: func(cmd *cobra.Command, args []string) {
		printing.Banner(options)
		printing.DalLog("YELLOW", "Read the help page using the -h flag to see other options and flags!", options)
	},
}

// Execute is run rootCmd
func Execute() {
	defer func() {
		if options.HarWriter != nil {
			options.HarWriter.Close()
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	// Slice
	rootCmd.PersistentFlags().StringSliceVarP(&header, "header", "H", []string{}, "Add custom headers")
	rootCmd.PersistentFlags().StringSliceVarP(&p, "param", "p", []string{}, "Only testing selected parameters")
	rootCmd.PersistentFlags().StringSliceVar(&ignoreParams, "ignore-param", []string{}, "Ignores this parameter when scanning.\n  * Example: --ignore-param api_token --ignore-param csrf_token")

	//Str
	rootCmd.PersistentFlags().StringVar(&config, "config", "", "Using config from file")
	rootCmd.PersistentFlags().StringVarP(&cookie, "cookie", "C", "", "Add custom cookie")
	rootCmd.PersistentFlags().StringVarP(&data, "data", "d", "", "Using POST Method and add Body data")
	rootCmd.PersistentFlags().StringVar(&customPayload, "custom-payload", "", "Add custom payloads from file")
	rootCmd.PersistentFlags().StringVar(&customAlertValue, "custom-alert-value", "1", "Change alert value\n  * Example: --custom-alert-value=document.cookie")
	rootCmd.PersistentFlags().StringVar(&customAlertType, "custom-alert-type", "none", "Change alert value type\n  * Example: --custom-alert-type=none / --custom-alert-type=str,none")
	rootCmd.PersistentFlags().StringVar(&userAgent, "user-agent", "", "Add custom UserAgent")
	rootCmd.PersistentFlags().StringVarP(&blind, "blind", "b", "", "Add your blind xss\n  * Example: -b your-callback-url")
	rootCmd.PersistentFlags().StringVarP(&output, "output", "o", "", "Write to output file (By default, only the PoC code is saved)")
	rootCmd.PersistentFlags().StringVar(&format, "format", "plain", "Stdout output format\n  * Supported: plain / json")
	rootCmd.PersistentFlags().StringVar(&foundAction, "found-action", "", "If found weak/vuln, action(cmd) to next\n  * Example: --found-action='./notify.sh'")
	rootCmd.PersistentFlags().StringVar(&foundActionShell, "found-action-shell", "bash", "Select shell application for --found-action")
	rootCmd.PersistentFlags().StringVar(&proxy, "proxy", "", "Send all request to proxy server\n  * Example: --proxy http://127.0.0.1:8080")
	rootCmd.PersistentFlags().StringVar(&grep, "grep", "", "Using custom grepping file\n  * Example: --grep ./samples/sample_grep.json")
	rootCmd.PersistentFlags().StringVar(&ignoreReturn, "ignore-return", "", "Ignores scanning from return code\n  * Example: --ignore-return 302,403,404")
	rootCmd.PersistentFlags().StringVarP(&miningWord, "mining-dict-word", "W", "", "Custom wordlist file for param mining\n  * Example: --mining-dict-word word.txt")
	rootCmd.PersistentFlags().StringVarP(&method, "method", "X", "GET", "Force overriding HTTP Method\n  * Example: -X PUT")
	rootCmd.PersistentFlags().StringVarP(&cookieFromRaw, "cookie-from-raw", "", "", "Load cookie from burp raw http request\n  * Example: --cookie-from-raw request.txt")
	rootCmd.PersistentFlags().StringVar(&remotePayloads, "remote-payloads", "", "Using remote payload for XSS testing\n  * Supported: portswigger/payloadbox\n  * Example: --remote-payloads=portswigger,payloadbox")
	rootCmd.PersistentFlags().StringVar(&remoteWordlists, "remote-wordlists", "", "Using remote wordlists for param mining\n  * Supported: burp/assetnote\n  * Example: --remote-wordlists=burp")
	rootCmd.PersistentFlags().StringVar(&onlyPoC, "only-poc", "", "Shows only the PoC code for the specified pattern (g: grep / r: reflected / v: verified)\n * Example: --only-poc='g,v'")
	rootCmd.PersistentFlags().StringVar(&pocType, "poc-type", "plain", "Select PoC type \n * Supported: plain/curl/httpie/http-request\n * Example: --poc-type='curl'")
	rootCmd.PersistentFlags().StringVar(&reportFormat, "report-format", "plain", "Format of --report flag [plain/json]")
	rootCmd.PersistentFlags().StringVar(&harFilePath, "har-file-path", "", "Path to save HAR of scan requests to")

	//Int
	rootCmd.PersistentFlags().IntVar(&timeout, "timeout", 10, "Second of timeout")
	rootCmd.PersistentFlags().IntVar(&delay, "delay", 0, "Milliseconds between send to same host (1000==1s)")
	rootCmd.PersistentFlags().IntVarP(&concurrence, "worker", "w", 100, "Number of worker")

	//Bool
	rootCmd.PersistentFlags().BoolVar(&onlyDiscovery, "only-discovery", false, "Only testing parameter analysis (same '--skip-xss-scanning' option)")
	rootCmd.PersistentFlags().BoolVarP(&silence, "silence", "S", false, "Only print PoC Code and Progress(for pipe/file mode)")
	rootCmd.PersistentFlags().BoolVar(&mining, "mining-dict", true, "Find new parameter with dictionary attack, default is Gf-Patterns=>XSS")
	rootCmd.PersistentFlags().BoolVar(&findingDOM, "mining-dom", true, "Find new parameter in DOM (attribute/js value)")
	rootCmd.PersistentFlags().BoolVarP(&followRedirect, "follow-redirects", "F", false, "Following redirection")
	rootCmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "Not use colorize")
	rootCmd.PersistentFlags().BoolVar(&noSpinner, "no-spinner", false, "Not use spinner")
	rootCmd.PersistentFlags().BoolVar(&skipBAV, "skip-bav", false, "Skipping BAV(Basic Another Vulnerability) analysis")
	rootCmd.PersistentFlags().BoolVar(&skipMiningDom, "skip-mining-dom", false, "Skipping DOM base parameter mining")
	rootCmd.PersistentFlags().BoolVar(&skipMiningDict, "skip-mining-dict", false, "Skipping Dict base parameter mining")
	rootCmd.PersistentFlags().BoolVar(&skipMiningAll, "skip-mining-all", false, "Skipping ALL parameter mining")
	rootCmd.PersistentFlags().BoolVar(&skipXSSScan, "skip-xss-scanning", false, "Skipping XSS Scanning (same '--only-discovery' option)")
	rootCmd.PersistentFlags().BoolVar(&onlyCustomPayload, "only-custom-payload", false, "Only testing custom payload (required --custom-payload)")
	rootCmd.PersistentFlags().BoolVar(&skipGrep, "skip-grepping", false, "Skipping built-in grepping")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "debug mode, save all log using -o option")
	rootCmd.PersistentFlags().BoolVar(&skipHeadless, "skip-headless", false, "Skipping headless browser base scanning[DOM XSS and inJS verify]")
	rootCmd.PersistentFlags().BoolVar(&useDeepDXSS, "deep-domxss", false, "DOM XSS Testing with more payloads on headless [so slow]")
	rootCmd.PersistentFlags().BoolVar(&outputAll, "output-all", false, "All log write mode (-o or stdout)")
	rootCmd.PersistentFlags().BoolVar(&wafEvasion, "waf-evasion", false, "Avoid blocking by adjusting the speed when detecting WAF (worker=1 delay=3s)")
	rootCmd.PersistentFlags().BoolVar(&reportBool, "report", false, "Show detail report")
	rootCmd.PersistentFlags().BoolVar(&outputRequest, "output-request", false, "Include raw HTTP requests in the results.")
	rootCmd.PersistentFlags().BoolVar(&outputResponse, "output-response", false, "Include raw HTTP response in the results.")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	stime := time.Now()
	au := aurora.NewAurora(!noColor)
	options = model.Options{
		Header:            header,
		Cookie:            cookie,
		UniqParam:         p,
		BlindURL:          blind,
		CustomPayloadFile: customPayload,
		CustomAlertValue:  customAlertValue,
		CustomAlertType:   customAlertType,
		Data:              data,
		UserAgent:         userAgent,
		OutputFile:        output,
		Format:            format,
		FoundAction:       foundAction,
		FoundActionShell:  foundActionShell,
		ProxyAddress:      proxy,
		Grep:              grep,
		IgnoreReturn:      ignoreReturn,
		IgnoreParams:      ignoreParams,
		Timeout:           timeout,
		Concurrence:       concurrence,
		Delay:             delay,
		OnlyDiscovery:     onlyDiscovery,
		OnlyCustomPayload: onlyCustomPayload,
		Silence:           silence,
		FollowRedirect:    followRedirect,
		Scan:              make(map[string]model.Scan),
		Mining:            mining,
		MiningWordlist:    miningWord,
		FindingDOM:        findingDOM,
		NoColor:           noColor,
		Method:            method,
		NoSpinner:         noSpinner,
		NoBAV:             skipBAV,
		NoGrep:            skipGrep,
		Debug:             debug,
		CookieFromRaw:     cookieFromRaw,
		AuroraObject:      au,
		StartTime:         stime,
		MulticastMode:     false,
		RemotePayloads:    remotePayloads,
		RemoteWordlists:   remoteWordlists,
		UseHeadless:       !skipHeadless,
		UseDeepDXSS:       useDeepDXSS,
		OnlyPoC:           onlyPoC,
		OutputAll:         outputAll,
		WAF:               false,
		WAFEvasion:        wafEvasion,
		PoCType:           pocType,
		ReportBool:        reportBool,
		ReportFormat:      reportFormat,
		OutputRequest:     outputRequest,
		OutputResponse:    outputResponse,
	}
	// var skipMiningDom, skipMiningDict, skipMiningAll, skipXSSScan, skipBAV bool

	if harFilePath != "" {
		f, err := os.Create(harFilePath)
		if err != nil {
			fmt.Println(err)
		} else {
			options.HarWriter, err = har.NewWriter(f, &har.Creator{Name: "dalfox", Version: printing.VERSION})
			if err != nil {
				fmt.Println(err)
			}
		}
	}

	if skipMiningAll {
		options.FindingDOM = false
		options.Mining = false

	} else {
		if skipMiningDom {
			options.FindingDOM = false
		}
		if skipMiningDict {
			options.Mining = false
		}
	}

	if skipXSSScan {
		options.OnlyDiscovery = true
	}

	if grep != "" {
		// Open our jsonFile
		jsonFile, err := os.Open(grep)
		// if we os.Open returns an error then handle it
		if err != nil {
			fmt.Println(err)
		}
		printing.DalLog("SYSTEM", "Loaded "+grep+" file for grepping", options)
		// defer the closing of our jsonFile so that we can parse it later on
		defer jsonFile.Close()
		byteValue, _ := io.ReadAll(jsonFile)
		options.Grep = string(byteValue)

	}
	if config != "" {
		// Open our jsonFile
		jsonFile, err := os.Open(config)
		// if we os.Open returns an error then handle it
		if err != nil {
			fmt.Println(err)
		}
		printing.DalLog("SYSTEM", "Loaded "+config+" file for config option", options)
		// defer the closing of our jsonFile so that we can parse it later on
		defer jsonFile.Close()

		byteValue, _ := io.ReadAll(jsonFile)
		err = json.Unmarshal([]byte(byteValue), &options)
		if err != nil {
			printing.DalLog("SYSTEM", "Error while parsing config file", options)
		}
	}
}
