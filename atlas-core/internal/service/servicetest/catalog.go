package servicetest

import (
	"encoding/json"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/catalog"
)

// defaultCatalogJSON is minimal valid command catalog document bytes used by tests. It must
// stay stable so [catalog.ObjectIDFromContentBytes] and active-store checks stay aligned.
const defaultCatalogJSON = `{"catalog_id":"c1","version":"1","commands":[{"type":"move_to_location","display_name":"M","description":"D","parameters_schema":{"type":"object","required":["latitude"],"additionalProperties":false,"properties":{"latitude":{"type":"number"}}}}]}`

// NewDefaultCommandCatalog returns a validated in-memory [catalog.Catalog] with [Catalog.Raw] and
// [Catalog.ObjectID] set coherently (as production [catalog.Load] would).
func NewDefaultCommandCatalog() (catalog.Catalog, error) {
	raw := []byte(defaultCatalogJSON)
	var c catalog.Catalog
	if err := json.Unmarshal(raw, &c); err != nil {
		return catalog.Catalog{}, err
	}
	c.Raw = raw
	if err := catalog.Validate(c); err != nil {
		return catalog.Catalog{}, err
	}
	// Mirror [catalog.Load]: derive ContentHash from the raw bytes via the
	// canonical helper, then build ObjectID from that hash. Slicing a known
	// prefix off ObjectID would silently produce wrong values if the prefix
	// format ever changes.
	c.ContentHash = catalog.ContentHashOfBytes(raw)
	c.ObjectID = catalog.ObjectIDFromContentHash(c.ContentHash)
	c.ByType = make(map[string]catalog.Command, len(c.Commands))
	for _, cmd := range c.Commands {
		c.ByType[cmd.Type] = cmd
	}
	return c, nil
}
