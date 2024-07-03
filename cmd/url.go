package cmd

import (
	"github.com/emrekara369/dalfox_new/v2/pkg/printing"
	"github.com/emrekara369/dalfox_new/v2/pkg/scanning"
	"github.com/spf13/cobra"
)

// urlCmd represents the url command
var urlCmd = &cobra.Command{
	Use:   "url [target] [flags]",
	Short: "Use single target mode",
	Run: func(cmd *cobra.Command, args []string) {
		printing.Banner(options)
		if len(args) == 0 {
			printing.DalLog("ERROR", "Input target url", options)
			printing.DalLog("ERROR", "e.g dalfox url https://google.com/?q=1", options)
			return
		}
		printing.Summary(options, args[0])
		if len(args) >= 1 {
			printing.DalLog("SYSTEM", "Using single target mode", options)
			if options.Format == "json" {
				printing.DalLog("PRINT", "[", options)
			}
			_, _ = scanning.Scan(args[0], options, "Single")
			if options.Format == "json" {
				printing.DalLog("PRINT", "{}]", options)
			}
		} else {
			printing.DalLog("ERROR", "Input target url", options)
			printing.DalLog("ERROR", "e.g dalfox url https://google.com/?q=1", options)
		}
	},
}

func init() {
	rootCmd.AddCommand(urlCmd)
}
