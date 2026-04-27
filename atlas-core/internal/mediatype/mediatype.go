package mediatype

import (
	"mime"
	"strings"
)

// NormalizeContentType returns a normalized media type and whether the input
// was a valid media type. Empty input is valid and normalizes to
// application/octet-stream.
func NormalizeContentType(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "application/octet-stream", true
	}
	mediaType, params, err := mime.ParseMediaType(value)
	if err != nil || mediaType == "" {
		return "application/octet-stream", false
	}
	if len(params) == 0 {
		return mediaType, true
	}
	formatted := mime.FormatMediaType(mediaType, params)
	if formatted == "" {
		return mediaType, true
	}
	return formatted, true
}
