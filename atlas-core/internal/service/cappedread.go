package service

import (
	"io"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
)

// maxPinnedCatalogBytes limits how many bytes of catalog JSON are read from object storage
// for task validation. One byte over the limit fails with an oversize error.
const maxPinnedCatalogBytes = 8 << 20

// readAllCapped reads r until EOF or (max+1) bytes, whichever comes first. If the body
// exceeds max bytes, returns [model.PayloadTooLarge].
func readAllCapped(r io.Reader, max int64) ([]byte, error) {
	buf, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(buf)) > max {
		return nil, model.PayloadTooLarge("pinned command catalog JSON exceeds read limit")
	}
	return buf, nil
}
