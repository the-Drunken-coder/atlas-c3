package mediatype

import (
	"mime"
	"strings"
)

// NormalizeContentType returns a normalized media type and whether the input
// was a valid media type. Empty input is valid and normalizes to
// application/octet-stream. When parameters are present, they are included in
// the normalized output using standard formatting.
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
		// If parameter formatting fails, return the validated media type without
		// parameters so callers still get a safe header/storage value.
		return mediaType, true
	}
	return formatted, true
}
