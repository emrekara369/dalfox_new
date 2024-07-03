package scanning

import (
	"os/exec"
	"strings"

	"github.com/emrekara369/dalfox_new/v2/pkg/model"
	"github.com/emrekara369/dalfox_new/v2/pkg/printing"
)

// foundAction is after command function.
func foundAction(options model.Options, target, query, ptype string) {
	afterCmd := options.FoundAction
	afterCmd = strings.ReplaceAll(afterCmd, "@@query@@", query)
	afterCmd = strings.ReplaceAll(afterCmd, "@@target@@", target)
	afterCmd = strings.ReplaceAll(afterCmd, "@@type@@", ptype)
	cmd := exec.Command(options.FoundActionShell, "-c", afterCmd)
	err := cmd.Start()
	if err != nil {
		printing.DalLog("ERROR", "execution error from found-action", options)
	}
}
