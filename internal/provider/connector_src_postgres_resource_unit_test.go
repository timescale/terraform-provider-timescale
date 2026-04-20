package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestComputeTableDiff_AddNew(t *testing.T) {
	state := []tableModel{}
	plan := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("events")},
	}

	add, drop := computeTableDiff(state, plan)

	if len(add) != 1 {
		t.Fatalf("expected 1 add, got %d", len(add))
	}
	if len(drop) != 0 {
		t.Fatalf("expected 0 drops, got %d", len(drop))
	}
	table, ok := add[0]["table"].(map[string]interface{})
	if !ok {
		t.Fatal("expected table to be map[string]interface{}")
	}
	if table["schemaName"] != "public" || table["tableName"] != "events" {
		t.Errorf("unexpected add table: %v", table)
	}
}

func TestComputeTableDiff_DropExisting(t *testing.T) {
	state := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("events")},
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("users")},
	}
	plan := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("events")},
	}

	add, drop := computeTableDiff(state, plan)

	if len(add) != 0 {
		t.Fatalf("expected 0 adds, got %d", len(add))
	}
	if len(drop) != 1 {
		t.Fatalf("expected 1 drop, got %d", len(drop))
	}
	if drop[0]["schemaName"] != "public" || drop[0]["tableName"] != "users" {
		t.Errorf("unexpected drop table: %v", drop[0])
	}
}

func TestComputeTableDiff_NoChanges(t *testing.T) {
	tables := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("events")},
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("users")},
	}

	add, drop := computeTableDiff(tables, tables)

	if len(add) != 0 {
		t.Fatalf("expected 0 adds, got %d", len(add))
	}
	if len(drop) != 0 {
		t.Fatalf("expected 0 drops, got %d", len(drop))
	}
}

func TestComputeTableDiff_TableMappingChanged(t *testing.T) {
	state := []tableModel{
		{
			SchemaName: types.StringValue("public"),
			TableName:  types.StringValue("events"),
			TableMapping: &tableMappingModel{
				SchemaName: types.StringValue("public"),
				TableName:  types.StringValue("events_old"),
			},
		},
	}
	plan := []tableModel{
		{
			SchemaName: types.StringValue("public"),
			TableName:  types.StringValue("events"),
			TableMapping: &tableMappingModel{
				SchemaName: types.StringValue("public"),
				TableName:  types.StringValue("events_new"),
			},
		},
	}

	add, drop := computeTableDiff(state, plan)

	if len(add) != 1 {
		t.Fatalf("expected 1 add (re-add), got %d", len(add))
	}
	if len(drop) != 1 {
		t.Fatalf("expected 1 drop (for re-add), got %d", len(drop))
	}
}

func TestComputeTableDiff_PublicationNameChanged(t *testing.T) {
	state := []tableModel{
		{
			SchemaName:      types.StringValue("public"),
			TableName:       types.StringValue("events"),
			PublicationName: types.StringValue("pub_old"),
		},
	}
	plan := []tableModel{
		{
			SchemaName:      types.StringValue("public"),
			TableName:       types.StringValue("events"),
			PublicationName: types.StringValue("pub_new"),
		},
	}

	add, drop := computeTableDiff(state, plan)

	if len(add) != 1 {
		t.Fatalf("expected 1 add (re-add), got %d", len(add))
	}
	if len(drop) != 1 {
		t.Fatalf("expected 1 drop (for re-add), got %d", len(drop))
	}
}

func TestComputeTableDiff_TableMappingAdded(t *testing.T) {
	state := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("events")},
	}
	plan := []tableModel{
		{
			SchemaName: types.StringValue("public"),
			TableName:  types.StringValue("events"),
			TableMapping: &tableMappingModel{
				SchemaName: types.StringValue("archive"),
				TableName:  types.StringValue("events"),
			},
		},
	}

	add, drop := computeTableDiff(state, plan)

	if len(add) != 1 {
		t.Fatalf("expected 1 add (re-add), got %d", len(add))
	}
	if len(drop) != 1 {
		t.Fatalf("expected 1 drop (for re-add), got %d", len(drop))
	}
}

func TestComputeTableDiff_TableMappingRemoved(t *testing.T) {
	state := []tableModel{
		{
			SchemaName: types.StringValue("public"),
			TableName:  types.StringValue("events"),
			TableMapping: &tableMappingModel{
				SchemaName: types.StringValue("archive"),
				TableName:  types.StringValue("events"),
			},
		},
	}
	plan := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("events")},
	}

	add, drop := computeTableDiff(state, plan)

	if len(add) != 1 {
		t.Fatalf("expected 1 add (re-add), got %d", len(add))
	}
	if len(drop) != 1 {
		t.Fatalf("expected 1 drop (for re-add), got %d", len(drop))
	}
}

func TestComputeTableDiff_Mixed(t *testing.T) {
	state := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("keep")},
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("drop_me")},
		{
			SchemaName:      types.StringValue("public"),
			TableName:       types.StringValue("change_me"),
			PublicationName: types.StringValue("old_pub"),
		},
	}
	plan := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("keep")},
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("new_one")},
		{
			SchemaName:      types.StringValue("public"),
			TableName:       types.StringValue("change_me"),
			PublicationName: types.StringValue("new_pub"),
		},
	}

	add, drop := computeTableDiff(state, plan)

	// drop_me dropped, change_me dropped for re-add = 2 drops
	if len(drop) != 2 {
		t.Fatalf("expected 2 drops, got %d", len(drop))
	}
	// new_one added, change_me re-added = 2 adds
	if len(add) != 2 {
		t.Fatalf("expected 2 adds, got %d", len(add))
	}
}

func TestTableConfigChanged_NoChange(t *testing.T) {
	a := tableModel{
		SchemaName:      types.StringValue("public"),
		TableName:       types.StringValue("events"),
		PublicationName: types.StringValue("my_pub"),
	}
	if tableConfigChanged(a, a) {
		t.Error("expected no change")
	}
}

func TestTableConfigChanged_NilMappings(t *testing.T) {
	a := tableModel{SchemaName: types.StringValue("public"), TableName: types.StringValue("events")}
	b := tableModel{SchemaName: types.StringValue("public"), TableName: types.StringValue("events")}
	if tableConfigChanged(a, b) {
		t.Error("expected no change for two nil mappings")
	}
}

func TestReorderTablesToMatch_MatchesOrder(t *testing.T) {
	api := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("c")},
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("a")},
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("b")},
	}
	ref := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("a")},
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("b")},
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("c")},
	}

	result := reorderTablesToMatch(api, ref)

	if len(result) != 3 {
		t.Fatalf("expected 3, got %d", len(result))
	}
	expected := []string{"a", "b", "c"}
	for i, e := range expected {
		if result[i].TableName.ValueString() != e {
			t.Errorf("index %d: expected %q, got %q", i, e, result[i].TableName.ValueString())
		}
	}
}

func TestReorderTablesToMatch_ExtraFromAPI(t *testing.T) {
	api := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("new_one")},
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("a")},
	}
	ref := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("a")},
	}

	result := reorderTablesToMatch(api, ref)

	if len(result) != 2 {
		t.Fatalf("expected 2, got %d", len(result))
	}
	if result[0].TableName.ValueString() != "a" {
		t.Errorf("index 0: expected 'a', got %q", result[0].TableName.ValueString())
	}
	if result[1].TableName.ValueString() != "new_one" {
		t.Errorf("index 1: expected 'new_one', got %q", result[1].TableName.ValueString())
	}
}

func TestReorderTablesToMatch_EmptyAPI(t *testing.T) {
	ref := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("a")},
	}
	result := reorderTablesToMatch(nil, ref)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestReorderTablesToMatch_EmptyRef(t *testing.T) {
	api := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("a")},
	}
	result := reorderTablesToMatch(api, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

func TestBuildTableSpecInput_Basic(t *testing.T) {
	tm := tableModel{
		SchemaName: types.StringValue("public"),
		TableName:  types.StringValue("events"),
	}

	spec := buildTableSpecInput(tm)

	table, ok := spec["table"].(map[string]interface{})
	if !ok {
		t.Fatal("expected table to be map[string]interface{}")
	}
	if table["schemaName"] != "public" || table["tableName"] != "events" {
		t.Errorf("unexpected table: %v", table)
	}
	if _, ok := spec["tableMapping"]; ok {
		t.Error("expected no tableMapping")
	}
	if _, ok := spec["publicationName"]; ok {
		t.Error("expected no publicationName")
	}
}

func TestBuildTableSpecInput_WithMapping(t *testing.T) {
	tm := tableModel{
		SchemaName: types.StringValue("public"),
		TableName:  types.StringValue("events"),
		TableMapping: &tableMappingModel{
			SchemaName: types.StringValue("archive"),
			TableName:  types.StringValue("events_archive"),
		},
	}

	spec := buildTableSpecInput(tm)

	mapping, ok := spec["tableMapping"].(map[string]interface{})
	if !ok {
		t.Fatal("expected tableMapping to be map[string]interface{}")
	}
	if mapping["schemaName"] != "archive" || mapping["tableName"] != "events_archive" {
		t.Errorf("unexpected mapping: %v", mapping)
	}
}

func TestBuildTableSpecInput_WithPublication(t *testing.T) {
	tm := tableModel{
		SchemaName:      types.StringValue("public"),
		TableName:       types.StringValue("events"),
		PublicationName: types.StringValue("my_pub"),
	}

	spec := buildTableSpecInput(tm)

	if spec["publicationName"] != "my_pub" {
		t.Errorf("expected publicationName 'my_pub', got %v", spec["publicationName"])
	}
}

func TestBuildTableSpecInput_EmptyPublicationOmitted(t *testing.T) {
	tm := tableModel{
		SchemaName:      types.StringValue("public"),
		TableName:       types.StringValue("events"),
		PublicationName: types.StringValue(""),
	}

	spec := buildTableSpecInput(tm)

	if _, ok := spec["publicationName"]; ok {
		t.Error("expected empty publicationName to be omitted")
	}
}

func TestBuildTableSpecInput_WithHypertableSpec(t *testing.T) {
	tm := tableModel{
		SchemaName: types.StringValue("public"),
		TableName:  types.StringValue("events"),
		HypertableSpec: &hypertableSpecModel{
			PrimaryDimension: &rangeDimensionModel{
				ColumnName:        types.StringValue("created_at"),
				PartitionInterval: types.StringValue("7d"),
			},
		},
	}

	spec := buildTableSpecInput(tm)

	htSpec, ok := spec["hypertableSpec"].(map[string]interface{})
	if !ok {
		t.Fatal("expected hypertableSpec to be map[string]interface{}")
	}
	pd, ok := htSpec["primaryDimension"].(map[string]interface{})
	if !ok {
		t.Fatal("expected primaryDimension to be map[string]interface{}")
	}
	if pd["columnName"] != "created_at" || pd["partitionInterval"] != "7d" {
		t.Errorf("unexpected primaryDimension: %v", pd)
	}
	if _, ok := htSpec["secondaryDimensions"]; ok {
		t.Error("expected no secondaryDimensions")
	}
}

func TestBuildTableSpecInput_WithSecondaryDimensions(t *testing.T) {
	tm := tableModel{
		SchemaName: types.StringValue("public"),
		TableName:  types.StringValue("events"),
		HypertableSpec: &hypertableSpecModel{
			PrimaryDimension: &rangeDimensionModel{
				ColumnName:        types.StringValue("created_at"),
				PartitionInterval: types.StringValue("7d"),
			},
			SecondaryDimensions: []dimensionModel{
				{
					Hash: &hashDimensionModel{
						ColumnName:       types.StringValue("device_id"),
						NumberPartitions: types.Int64Value(4),
					},
				},
			},
		},
	}

	spec := buildTableSpecInput(tm)

	htSpec, ok := spec["hypertableSpec"].(map[string]interface{})
	if !ok {
		t.Fatal("expected hypertableSpec to be map[string]interface{}")
	}
	dims, ok := htSpec["secondaryDimensions"].([]map[string]interface{})
	if !ok {
		t.Fatal("expected secondaryDimensions to be []map[string]interface{}")
	}
	if len(dims) != 1 {
		t.Fatalf("expected 1 secondary dimension, got %d", len(dims))
	}
	hash, ok := dims[0]["hash"].(map[string]interface{})
	if !ok {
		t.Fatal("expected hash to be map[string]interface{}")
	}
	if hash["columnName"] != "device_id" {
		t.Errorf("expected columnName 'device_id', got %v", hash["columnName"])
	}
	if hash["numberPartitions"] != int64(4) {
		t.Errorf("expected numberPartitions 4, got %v", hash["numberPartitions"])
	}
}

func TestBuildTableSpecInput_NoHypertableSpec(t *testing.T) {
	tm := tableModel{
		SchemaName: types.StringValue("public"),
		TableName:  types.StringValue("events"),
	}

	spec := buildTableSpecInput(tm)

	if _, ok := spec["hypertableSpec"]; ok {
		t.Error("expected no hypertableSpec when not set")
	}
}

func TestTableConfigChanged_HypertableSpecIgnored(t *testing.T) {
	// hypertable_spec changes should NOT trigger tableConfigChanged — they are
	// handled separately by validateHypertableSpecChanges because the data plane
	// cannot re-apply hypertable config on an existing table.
	state := tableModel{
		SchemaName: types.StringValue("public"),
		TableName:  types.StringValue("events"),
	}
	plan := tableModel{
		SchemaName: types.StringValue("public"),
		TableName:  types.StringValue("events"),
		HypertableSpec: &hypertableSpecModel{
			PrimaryDimension: &rangeDimensionModel{
				ColumnName:        types.StringValue("ts"),
				PartitionInterval: types.StringValue("7d"),
			},
		},
	}

	if tableConfigChanged(state, plan) {
		t.Error("hypertable_spec changes should not trigger tableConfigChanged")
	}
}

func TestValidateHypertableSpecChanges_AddToExisting(t *testing.T) {
	state := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("events")},
	}
	plan := []tableModel{
		{
			SchemaName: types.StringValue("public"),
			TableName:  types.StringValue("events"),
			HypertableSpec: &hypertableSpecModel{
				PrimaryDimension: &rangeDimensionModel{
					ColumnName:        types.StringValue("ts"),
					PartitionInterval: types.StringValue("7d"),
				},
			},
		},
	}

	errs := validateHypertableSpecChanges(state, plan)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestValidateHypertableSpecChanges_RemoveFromExisting(t *testing.T) {
	// Removing hypertable_spec is allowed — the target table is already a hypertable
	// and won't change. This just updates the Terraform state.
	state := []tableModel{
		{
			SchemaName: types.StringValue("public"),
			TableName:  types.StringValue("events"),
			HypertableSpec: &hypertableSpecModel{
				PrimaryDimension: &rangeDimensionModel{
					ColumnName:        types.StringValue("ts"),
					PartitionInterval: types.StringValue("7d"),
				},
			},
		},
	}
	plan := []tableModel{
		{SchemaName: types.StringValue("public"), TableName: types.StringValue("events")},
	}

	errs := validateHypertableSpecChanges(state, plan)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors (removal is allowed), got %d", len(errs))
	}
}

func TestValidateHypertableSpecChanges_ModifyExisting(t *testing.T) {
	state := []tableModel{
		{
			SchemaName: types.StringValue("public"),
			TableName:  types.StringValue("events"),
			HypertableSpec: &hypertableSpecModel{
				PrimaryDimension: &rangeDimensionModel{
					ColumnName:        types.StringValue("ts"),
					PartitionInterval: types.StringValue("7d"),
				},
			},
		},
	}
	plan := []tableModel{
		{
			SchemaName: types.StringValue("public"),
			TableName:  types.StringValue("events"),
			HypertableSpec: &hypertableSpecModel{
				PrimaryDimension: &rangeDimensionModel{
					ColumnName:        types.StringValue("ts"),
					PartitionInterval: types.StringValue("1d"),
				},
			},
		},
	}

	errs := validateHypertableSpecChanges(state, plan)
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestValidateHypertableSpecChanges_NewTableAllowed(t *testing.T) {
	state := []tableModel{}
	plan := []tableModel{
		{
			SchemaName: types.StringValue("public"),
			TableName:  types.StringValue("events"),
			HypertableSpec: &hypertableSpecModel{
				PrimaryDimension: &rangeDimensionModel{
					ColumnName:        types.StringValue("ts"),
					PartitionInterval: types.StringValue("7d"),
				},
			},
		},
	}

	errs := validateHypertableSpecChanges(state, plan)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors for new table, got %d", len(errs))
	}
}

func TestValidateHypertableSpecChanges_UnchangedAllowed(t *testing.T) {
	ht := &hypertableSpecModel{
		PrimaryDimension: &rangeDimensionModel{
			ColumnName:        types.StringValue("ts"),
			PartitionInterval: types.StringValue("7d"),
		},
	}
	tables := []tableModel{
		{
			SchemaName:     types.StringValue("public"),
			TableName:      types.StringValue("events"),
			HypertableSpec: ht,
		},
	}
	plan := []tableModel{
		{
			SchemaName: types.StringValue("public"),
			TableName:  types.StringValue("events"),
			HypertableSpec: &hypertableSpecModel{
				PrimaryDimension: &rangeDimensionModel{
					ColumnName:        types.StringValue("ts"),
					PartitionInterval: types.StringValue("7d"),
				},
			},
		},
	}

	errs := validateHypertableSpecChanges(tables, plan)
	if len(errs) != 0 {
		t.Fatalf("expected 0 errors for unchanged spec, got %d", len(errs))
	}
}

func TestComputeTableDiff_HypertableSpecNotTriggered(t *testing.T) {
	// hypertable_spec changes should NOT cause a drop+re-add
	state := []tableModel{
		{
			SchemaName: types.StringValue("public"),
			TableName:  types.StringValue("events"),
			HypertableSpec: &hypertableSpecModel{
				PrimaryDimension: &rangeDimensionModel{
					ColumnName:        types.StringValue("ts"),
					PartitionInterval: types.StringValue("7d"),
				},
			},
		},
	}
	plan := []tableModel{
		{
			SchemaName: types.StringValue("public"),
			TableName:  types.StringValue("events"),
			HypertableSpec: &hypertableSpecModel{
				PrimaryDimension: &rangeDimensionModel{
					ColumnName:        types.StringValue("ts"),
					PartitionInterval: types.StringValue("1d"),
				},
			},
		},
	}

	add, drop := computeTableDiff(state, plan)

	if len(add) != 0 {
		t.Fatalf("expected 0 adds, got %d", len(add))
	}
	if len(drop) != 0 {
		t.Fatalf("expected 0 drops, got %d", len(drop))
	}
}
