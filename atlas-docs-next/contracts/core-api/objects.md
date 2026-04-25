# Objects API

Object endpoints manage file-container metadata and object file bytes.

Object record ownership is defined in [`../data-model/record-families.md`](../data-model/record-families.md). Shared API behavior is defined in [`conventions.md`](./conventions.md).

## Endpoints

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/objects` | List objects |
| `POST` | `/objects` | Create an object |
| `GET` | `/objects/{object_id}` | Read an object |
| `PATCH` | `/objects/{object_id}` | Update object metadata |
| `DELETE` | `/objects/{object_id}` | Delete an object |
| `POST` | `/objects/{object_id}/files` | Upload an object file |
| `GET` | `/objects/{object_id}/files/{file_id}` | Read object file metadata |
| `GET` | `/objects/{object_id}/files/{file_id}/content` | Stream object file content |
| `DELETE` | `/objects/{object_id}/files/{file_id}` | Delete an object file |

## Object List Filters

`GET /objects` should support filtering by owning or related record:

- `owner_type=entity&owner_id={entity_id}`
- `owner_type=observation&owner_id={observation_id}`
- `owner_type=task&owner_id={task_id}`
- `owner_type=system&owner_id={system_owned_id}`

## Notes

Create object requests must include `object_id`.

Upload object file requests must include `file_id`. Atlas Core should reject uploads without a caller-supplied file ID.

Objects are the authoritative owner of relationship links to records that use them.

Object file metadata is JSON. Object file content streams bytes directly.

File bytes should not be mutated in place. If file content needs to change, callers should delete and upload a file, or use a later replace-style operation if one is explicitly added to this contract.

Object file upload and delete operations should be treated as object changes for stream/event purposes. Event behavior is defined in [`stream.md`](./stream.md).

The command catalog is read through object and file endpoints after clients discover the active command catalog object ID from [`system.md`](./system.md).

