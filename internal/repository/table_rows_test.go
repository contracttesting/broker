package repository

import (
	"database/sql"
	"encoding/json"
	"testing"
)

func TestToResourceModelEmptyOptionalFieldsMarshalAsNull(t *testing.T) {
	row := tableRow{
		ResourceConsumedProvider:   sql.NullString{Valid: true, String: ""},
		ResourceResponseStatusCode: sql.NullString{Valid: true, String: ""},
		ResourceVersion:            "",
	}

	resource := row.toResourceModel()

	if resource.ConsumedProvider.Valid {
		t.Errorf("expected ConsumedProvider to be null for empty value, got valid=true %q", resource.ConsumedProvider.String)
	}
	if resource.ResponseStatusCode.Valid {
		t.Errorf("expected ResponseStatusCode to be null for empty value, got valid=true %q", resource.ResponseStatusCode.String)
	}
	if resource.Version.Valid {
		t.Errorf("expected Version to be null for empty value, got valid=true %q", resource.Version.String)
	}

	if provider, _ := json.Marshal(resource.ConsumedProvider); string(provider) != "null" {
		t.Errorf("expected ConsumedProvider to marshal as null, got %s", provider)
	}
	if statusCode, _ := json.Marshal(resource.ResponseStatusCode); string(statusCode) != "null" {
		t.Errorf("expected ResponseStatusCode to marshal as null, got %s", statusCode)
	}
	if version, _ := json.Marshal(resource.Version); string(version) != "null" {
		t.Errorf("expected Version to marshal as null, got %s", version)
	}
}

func TestToResourceModelPopulatedOptionalFieldsArePreserved(t *testing.T) {
	row := tableRow{
		ResourceConsumedProvider:   sql.NullString{Valid: true, String: "pets"},
		ResourceResponseStatusCode: sql.NullString{Valid: true, String: "200"},
		ResourceVersion:            "v1",
	}

	resource := row.toResourceModel()

	if !resource.ConsumedProvider.Valid || resource.ConsumedProvider.String != "pets" {
		t.Errorf("expected ConsumedProvider \"pets\", got valid=%v %q", resource.ConsumedProvider.Valid, resource.ConsumedProvider.String)
	}
	if !resource.ResponseStatusCode.Valid || resource.ResponseStatusCode.String != "200" {
		t.Errorf("expected ResponseStatusCode \"200\", got valid=%v %q", resource.ResponseStatusCode.Valid, resource.ResponseStatusCode.String)
	}
	if !resource.Version.Valid || resource.Version.String != "v1" {
		t.Errorf("expected Version \"v1\", got valid=%v %q", resource.Version.Valid, resource.Version.String)
	}
}

func TestToResourceModelNullOptionalFieldsMarshalAsNull(t *testing.T) {
	row := tableRow{
		ResourceConsumedProvider:   sql.NullString{Valid: false},
		ResourceResponseStatusCode: sql.NullString{Valid: false},
	}

	resource := row.toResourceModel()

	if resource.ConsumedProvider.Valid {
		t.Error("expected ConsumedProvider to be null for SQL NULL")
	}
	if resource.ResponseStatusCode.Valid {
		t.Error("expected ResponseStatusCode to be null for SQL NULL")
	}
}
