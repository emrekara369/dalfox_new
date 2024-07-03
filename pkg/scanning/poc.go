package scanning

import (
	"io"
	"net/http"
	"net/http/httputil"

	"github.com/emrekara369/dalfox_new/v2/pkg/model"
)

// MakePoC is making poc codes
func MakePoC(poc string, req *http.Request, options model.Options) string {
	if options.PoCType == "http-request" {
		requestDump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			return "HTTP RAW REQUEST\n" + string(requestDump)
		}
	}
	if req != nil {
		if req.Body != nil && req.GetBody != nil {
			body, err := req.GetBody()
			if err == nil {
				reqBody, err := io.ReadAll(body)
				if err == nil {
					if string(reqBody) != "" {
						switch options.PoCType {
						case "curl":
							return "curl -i -k -X " + req.Method + " " + poc + " -d \"" + string(reqBody) + "\""
						case "httpie":
							return "http " + req.Method + " " + poc + " \"" + string(reqBody) + "\" --verify=false -f"
						default:
							return poc + " -d " + string(reqBody)
						}
					}
				}
			}
		} else {
			switch options.PoCType {
			case "curl":
				return "curl -i -k " + poc
			case "httpie":
				return "http " + poc + " --verify=false"
			default:
				return poc
			}
		}
	}
	return poc
}
