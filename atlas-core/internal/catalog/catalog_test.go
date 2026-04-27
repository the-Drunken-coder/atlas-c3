package catalog

import "testing"

func TestCatalogCloneDeepCopiesMetadata(t *testing.T) {
	inner := map[string]any{"k": 1}
	c := Catalog{
		Metadata: map[string]any{"outer": inner},
		Commands: []Command{{Type: "t", ParametersSchema: map[string]any{"a": 1}}},
	}
	cl := c.Clone()
	inner["k"] = 2
	if c.Metadata["outer"].(map[string]any)["k"] != 2 {
		t.Fatal("expected original metadata to be mutable from inner ref")
	}
	if cl.Metadata["outer"].(map[string]any)["k"] != 1 {
		t.Fatalf("clone should not share nested metadata maps; got %#v", cl.Metadata["outer"])
	}
}
