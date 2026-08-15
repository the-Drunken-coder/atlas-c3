package catalog

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/model"
	"github.com/the-Drunken-coder/atlas-c3/atlas-core/internal/store"
)

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

func TestMaterializeRejectsMismatchedExistingCatalogFile(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"catalog_id":"c1","version":"1","commands":[{"type":"move_to_location","display_name":"Move","description":"Move","parameters_schema":{"type":"object","additionalProperties":true}}]}`)
	cat := Catalog{
		CatalogID: "c1",
		Version:   "1",
		Commands: []Command{{
			Type:             "move_to_location",
			DisplayName:      "Move",
			Description:      "Move",
			ParametersSchema: map[string]any{"type": "object", "additionalProperties": true},
		}},
		Raw:         raw,
		ContentHash: ContentHashOfBytes(raw),
	}
	cat.ObjectID = ObjectIDFromContentHash(cat.ContentHash)
	cases := []struct {
		name      string
		sizeBytes int64
		bytes     []byte
	}{
		{
			name:      "byte mismatch",
			sizeBytes: int64(len(raw)),
			bytes: func() []byte {
				badRaw := append([]byte(nil), raw...)
				badRaw[0] = '['
				return badRaw
			}(),
		},
		{
			name:      "size mismatch",
			sizeBytes: int64(len(raw) + 1),
			bytes:     append([]byte(nil), raw...),
		},
	}
	cat.ByType = map[string]Command{"move_to_location": cat.Commands[0]}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			stores := &materializeTestStore{
				object: model.Object{ObjectID: cat.ObjectID, Type: "command_catalog", OwnerType: "system", OwnerID: "active_command_catalog", JSON: model.JSONMap{}},
				file:   model.ObjectFile{ObjectID: cat.ObjectID, FileID: "catalog-json", SizeBytes: test.sizeBytes, ContentType: "application/json"},
				bytes:  test.bytes,
			}
			_, err := Materialize(context.Background(), stores, cat)
			if err == nil {
				t.Fatal("expected catalog mismatch error")
			}
			coreErr, ok := model.IsCoreError(err)
			if !ok || coreErr.ErrorCode != "catalog_unavailable" {
				t.Fatalf("expected catalog_unavailable, got %v", err)
			}
		})
	}
}

type materializeTestStore struct {
	object model.Object
	file   model.ObjectFile
	bytes  []byte
}

func (m *materializeTestStore) CreateObject(context.Context, model.Object) (model.Object, error) {
	panic("unexpected CreateObject call")
}

func (m *materializeTestStore) GetObject(_ context.Context, id string) (model.Object, error) {
	if id != m.object.ObjectID {
		return model.Object{}, model.NotFound("object", id)
	}
	return m.object, nil
}

func (m *materializeTestStore) ListObjects(context.Context, store.ObjectListFilter, model.Pagination) ([]model.Object, int, error) {
	panic("unexpected ListObjects call")
}

func (m *materializeTestStore) UpdateObject(context.Context, model.Object) (model.Object, error) {
	panic("unexpected UpdateObject call")
}

func (m *materializeTestStore) DeleteObject(context.Context, string) error {
	panic("unexpected DeleteObject call")
}

func (m *materializeTestStore) CreateObjectFile(context.Context, store.ObjectUploadInput) (model.ObjectFile, error) {
	panic("unexpected CreateObjectFile call")
}

func (m *materializeTestStore) GetObjectFile(context.Context, string, string) (model.ObjectFile, error) {
	panic("unexpected GetObjectFile call")
}

func (m *materializeTestStore) AppendObjectFile(context.Context, string, string, io.Reader, int64) (model.ObjectFile, error) {
	panic("unexpected AppendObjectFile call")
}

func (m *materializeTestStore) OpenObjectFile(_ context.Context, objectID, fileID string) (model.ObjectFile, io.ReadCloser, error) {
	if objectID != m.file.ObjectID || fileID != m.file.FileID {
		return model.ObjectFile{}, nil, model.NotFound("object_file", fileID)
	}
	return m.file, io.NopCloser(bytes.NewReader(m.bytes)), nil
}

func (m *materializeTestStore) DeleteObjectFile(context.Context, string, string) error {
	panic("unexpected DeleteObjectFile call")
}

func (m *materializeTestStore) ListObjectFilesForObject(_ context.Context, objectID string) ([]model.ObjectFile, error) {
	if objectID != m.file.ObjectID {
		return nil, nil
	}
	return []model.ObjectFile{m.file}, nil
}

func (m *materializeTestStore) StorageStatus(context.Context) model.DependencyStatus {
	return model.DependencyStatus{Status: "ready"}
}
