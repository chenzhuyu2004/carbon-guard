package output

import (
	"encoding/json"
	"fmt"
	"os"

	cgerrors "github.com/czy/carbon-guard/internal/errors"
)

type ErrorResponse struct {
	Error string `json:"error"`
	Code  int    `json:"code"`
}

func HandleExit(err error, asJSON bool) {
	if err == nil {
		os.Exit(cgerrors.Success)
	}

	code := cgerrors.GetCode(err)
	if asJSON {
		resp := ErrorResponse{
			Error: err.Error(),
			Code:  code,
		}
		if encodeErr := json.NewEncoder(os.Stderr).Encode(resp); encodeErr != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		os.Exit(code)
	}

	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(code)
}
