package model

import (
    "encoding/json"
    "fmt"
    "regexp"
    "sort"
    "strings"
    "time"
)

type JSONMap map[string]any

type Entity struct {
    EntityID  string    `json:"entity_id"`
    Type      string    `json:"type"`
    Subtype   string    `json:"subtype,omitempty"`
    Alias     string    `json:"alias,omitempty"`
    JSON      JSONMap   `json:"json"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}

type Observation struct {
    ObservationID string    `json:"observation_id"`
    SourceAssetID string    `json:"source_asset_id"`
    JSON          JSONMap   `json:"json"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}

type Task struct {
    TaskID                 string    `json:"task_id"`
    Status                 string    `json:"status"`
    AssetID                string    `json:"asset_id"`
    CommandCatalogObjectID string    `json:"command_catalog_object_id"`
    JSON                   JSONMap   `json:"json"`
    CreatedAt              time.Time `json:"created_at"`
    UpdatedAt              time.Time `json:"updated_at"`
}

type Object struct {
    ObjectID   string    `json:"object_id"`
    Type       string    `json:"type"`
    OwnerType  string    `json:"owner_type"`
    OwnerID    string    `json:"owner_id"`
    JSON       JSONMap   `json:"json"`
    CreatedAt  time.Time `json:"created_at"`
    UpdatedAt  time.Time `json:"updated_at"`
}

type ObjectFile struct {
    FileID      string    `json:"file_id"`
    ObjectID    string    `json:"object_id"`
    Path        string    `json:"path"`
    ContentType string    `json:"content_type"`
    SizeBytes   int64     `json:"size_bytes"`
    UsageHint   string    `json:"usage_hint,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

type QuerySnapshot struct {
    GeneratedAt time.Time    `json:"generated_at"`
    Service     ServiceBrief `json:"service"`
    Entities    []Entity     `json:"entities"`
    Observations []Observation `json:"observations"`
    Tasks       []Task       `json:"tasks"`
    Objects     []Object     `json:"objects"`
    ObjectFiles []ObjectFile `json:"object_files"`
}

type ServiceDescriptor struct {
    Service                    string            `json:"service"`
    Status                     string            `json:"status"`
    Version                    string            `json:"version"`
    StartedAt                  time.Time         `json:"started_at"`
    ActiveCommandCatalogObjectID string          `json:"active_command_catalog_object_id"`
    Links                      map[string]string `json:"links"`
}

type ServiceBrief struct {
    Service                    string `json:"service"`
    ActiveCommandCatalogObjectID string `json:"active_command_catalog_object_id"`
}

type HealthResponse struct {
    Status    string    `json:"status"`
    Service   string    `json:"service"`
    Timestamp time.Time `json:"timestamp"`
}

type DependencyStatus struct {
    Status  string `json:"status"`
    Message string `json:"message,omitempty"`
}

type ReadinessResponse struct {
    Status       string                      `json:"status"`
    Timestamp    time.Time                   `json:"timestamp"`
    Dependencies map[string]DependencyStatus `json:"dependencies"`
}

type EventEnvelope struct {
    EventID      string      `json:"event_id"`
    Type         string      `json:"type"`
    ResourceType string      `json:"resource_type"`
    ResourceID   string      `json:"resource_id"`
    Mutation     string      `json:"mutation"`
    OccurredAt   time.Time   `json:"occurred_at"`
    Data         any         `json:"data"`
}

type ObjectEventResource struct {
    Object
    FileCount     int          `json:"file_count"`
    AffectedFiles []ObjectFile `json:"affected_files,omitempty"`
}

type DeleteEventData struct {
    DeletedAt time.Time `json:"deleted_at"`
}

const (
    ServiceName = "atlas-core"
)

var (
    idPattern       = regexp.MustCompile(`^[a-zA-Z0-9._:-]+$`)
    snakeCasePattern = regexp.MustCompile(`^[a-z0-9]+(?:_[a-z0-9]+)*$`)
    inaccuracyPattern = regexp.MustCompile(`^(\+-|\+|-)[0-9]+(\.[0-9]+)?$`)
)

func NormalizeJSONMap(m JSONMap) JSONMap {
    if m == nil {
        return JSONMap{}
    }
    return m
}

func CloneJSONMap(input JSONMap) JSONMap {
    if input == nil {
        return JSONMap{}
    }
    raw, _ := json.Marshal(input)
    var out JSONMap
    _ = json.Unmarshal(raw, &out)
    if out == nil {
        return JSONMap{}
    }
    return out
}

func MarshalJSONMap(input JSONMap) ([]byte, error) {
    if input == nil {
        input = JSONMap{}
    }
    return json.Marshal(input)
}

func ValidateID(name, value string) *FieldError {
    if value == "" {
        return &FieldError{Field: name, Code: "required", Message: fmt.Sprintf("%s is required", name)}
    }
    if len(value) > 50 {
        return &FieldError{Field: name, Code: "too_long", Message: fmt.Sprintf("%s must be 50 characters or fewer", name)}
    }
    if !idPattern.MatchString(value) {
        return &FieldError{Field: name, Code: "invalid_value", Message: fmt.Sprintf("%s contains unsupported characters", name)}
    }
    return nil
}

func ParseTimestamp(value string) (time.Time, error) {
    return time.Parse(time.RFC3339, value)
}

func ValidateRFC3339Field(name, value string) *FieldError {
    if value == "" {
        return &FieldError{Field: name, Code: "required", Message: fmt.Sprintf("%s is required", name)}
    }
    if _, err := ParseTimestamp(value); err != nil {
        return &FieldError{Field: name, Code: "invalid_value", Message: fmt.Sprintf("%s must be RFC3339", name)}
    }
    return nil
}

func ValidateSnakeCase(name, value string) *FieldError {
    if err := ValidateID(name, value); err != nil {
        return err
    }
    if !snakeCasePattern.MatchString(value) {
        return &FieldError{Field: name, Code: "invalid_value", Message: fmt.Sprintf("%s must be lowercase snake_case", name)}
    }
    return nil
}

func ValidateInaccuracyString(name, value string) *FieldError {
    if value == "" {
        return nil
    }
    if !inaccuracyPattern.MatchString(value) {
        return &FieldError{Field: name, Code: "invalid_value", Message: fmt.Sprintf("%s must use +, -, or +- numeric encoding", name)}
    }
    return nil
}

func SortedKeys(m map[string]any) []string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    sort.Strings(keys)
    return keys
}

func MergeNamedSections(base, patch JSONMap) JSONMap {
    out := CloneJSONMap(base)
    for key, value := range patch {
        if value == nil {
            delete(out, key)
            continue
        }
        out[key] = value
    }
    return out
}

func TopLevelUnknownFields(payload map[string]json.RawMessage, allowed ...string) []string {
    allowedSet := map[string]struct{}{}
    for _, field := range allowed {
        allowedSet[field] = struct{}{}
    }
    var unknown []string
    for key := range payload {
        if _, ok := allowedSet[key]; !ok {
            unknown = append(unknown, key)
        }
    }
    sort.Strings(unknown)
    return unknown
}

func EnsureNoPromotedFieldDuplication(jsonMap JSONMap, fields ...string) *FieldError {
    for _, field := range fields {
        if _, ok := jsonMap[field]; ok {
            return &FieldError{Field: "json." + field, Code: "invalid_value", Message: fmt.Sprintf("promoted field %s must not be duplicated in json", field)}
        }
    }
    return nil
}

func ReadStringSlice(input any) ([]string, bool) {
    raw, ok := input.([]any)
    if !ok {
        return nil, false
    }
    out := make([]string, 0, len(raw))
    for _, item := range raw {
        s, ok := item.(string)
        if !ok {
            return nil, false
        }
        out = append(out, s)
    }
    return out, true
}

func StringValue(input any) (string, bool) {
    value, ok := input.(string)
    return value, ok
}

func JSONSize(input any) int {
    raw, _ := json.Marshal(input)
    return len(raw)
}

func CountJSONFields(input any) int {
    switch v := input.(type) {
    case map[string]any:
        total := len(v)
        for _, child := range v {
            total += CountJSONFields(child)
        }
        return total
    case []any:
        total := 0
        for _, child := range v {
            total += CountJSONFields(child)
        }
        return total
    default:
        return 0
    }
}

func JSONDepth(input any) int {
    switch v := input.(type) {
    case map[string]any:
        maxDepth := 1
        for _, child := range v {
            if depth := 1 + JSONDepth(child); depth > maxDepth {
                maxDepth = depth
            }
        }
        return maxDepth
    case []any:
        maxDepth := 1
        for _, child := range v {
            if depth := 1 + JSONDepth(child); depth > maxDepth {
                maxDepth = depth
            }
        }
        return maxDepth
    default:
        return 1
    }
}

func ValidateCustomComponent(name string, value any) *FieldError {
    obj, ok := value.(map[string]any)
    if !ok {
        return &FieldError{Field: name, Code: "invalid_value", Message: "custom components must be JSON objects"}
    }
    if JSONSize(obj) > 16*1024 {
        return &FieldError{Field: name, Code: "out_of_range", Message: "custom component exceeds 16 KiB"}
    }
    if JSONDepth(obj) > 8 {
        return &FieldError{Field: name, Code: "out_of_range", Message: "custom component exceeds max depth 8"}
    }
    if CountJSONFields(obj) > 100 {
        return &FieldError{Field: name, Code: "out_of_range", Message: "custom component exceeds max field count 100"}
    }
    var walk func(map[string]any) *FieldError
    walk = func(node map[string]any) *FieldError {
        for key, child := range node {
            if len(key) > 100 {
                return &FieldError{Field: name, Code: "too_long", Message: "custom component key exceeds 100 characters"}
            }
            nested, ok := child.(map[string]any)
            if ok {
                if err := walk(nested); err != nil {
                    return err
                }
            }
        }
        return nil
    }
    return walk(obj)
}

func ContainsString(values []string, want string) bool {
    for _, value := range values {
        if value == want {
            return true
        }
    }
    return false
}

func CleanOriginList(raw string) []string {
    if raw == "" {
        return nil
    }
    parts := strings.Split(raw, ",")
    out := make([]string, 0, len(parts))
    for _, part := range parts {
        part = strings.TrimSpace(part)
        if part != "" {
            out = append(out, part)
        }
    }
    sort.Strings(out)
    return out
}
