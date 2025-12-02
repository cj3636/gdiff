package export

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

// CopyToClipboard writes the content to the terminal clipboard using OSC52.
// The writer defaults to stdout when nil.
func CopyToClipboard(content string, w io.Writer) error {
	if w == nil {
		w = os.Stdout
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	_, err := fmt.Fprintf(w, "\u001b]52;c;%s\u0007", encoded)
	return err
}
