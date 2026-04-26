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
	c.ObjectID = catalog.ObjectIDFromContentBytes(raw)
	if len(c.ObjectID) > len("command-catalog-") {
		c.ContentHash = c.ObjectID[len("command-catalog-"):]
	}
	c.ByType = map[string]catalog.Command{"move_to_location": c.Commands[0]}
	return c, nil
}
