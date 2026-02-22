package output

import (
	"encoding/json"
	"fmt"
	"os"

	cgerrors "github.com/chenzhuyu2004/carbon-guard/internal/errors"
	"github.com/chenzhuyu2004/carbon-guard/pkg"
)

type ErrorResponse struct {
	SchemaVersion string `json:"schema_version"`
	Error         string `json:"error"`
	Code          int    `json:"code"`
}

func HandleExit(err error, asJSON bool) {
	if err == nil {
		os.Exit(cgerrors.Success)
	}

	code := cgerrors.GetCode(err)
	if asJSON {
		resp := ErrorResponse{
			SchemaVersion: pkg.JSONSchemaVersion,
			Error:         err.Error(),
			Code:          code,
		}
		if encodeErr := json.NewEncoder(os.Stderr).Encode(resp); encodeErr != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		os.Exit(code)
	}

	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(code)
}
