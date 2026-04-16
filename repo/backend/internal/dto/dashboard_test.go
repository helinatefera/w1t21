package dto

import (
	"encoding/json"
	"testing"
)

func TestDashboardResponse_JSONFields(t *testing.T) {
	resp := DashboardResponse{
		OwnedCollectibles:   5,
		OpenOrders:          3,
		UnreadNotifications: 12,
		SellerOpenOrders:    2,
		ListedItems:         8,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	required := []string{
		"owned_collectibles",
		"open_orders",
		"unread_notifications",
		"seller_open_orders",
		"listed_items",
	}

	for _, field := range required {
		if _, ok := parsed[field]; !ok {
			t.Errorf("DashboardResponse JSON missing required field: %s", field)
		}
	}
}

func TestFunnelResponse_JSONFields(t *testing.T) {
	resp := FunnelResponse{
		Views:  100,
		Orders: 25,
		Rate:   0.25,
		Days:   7,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, field := range []string{"views", "orders", "rate", "days"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("FunnelResponse JSON missing required field: %s", field)
		}
	}
}

func TestABTestAssignment_JSONFields(t *testing.T) {
	a := ABTestAssignment{TestName: "catalog_layout", Variant: "grid"}
	data, _ := json.Marshal(a)
	var parsed map[string]interface{}
	_ = json.Unmarshal(data, &parsed)

	if parsed["test_name"] != "catalog_layout" {
		t.Errorf("expected test_name=catalog_layout, got %v", parsed["test_name"])
	}
	if parsed["variant"] != "grid" {
		t.Errorf("expected variant=grid, got %v", parsed["variant"])
	}
}

func TestMetricsResponse_JSONFields(t *testing.T) {
	resp := MetricsResponse{
		ActiveUsers:    42,
		OrdersByStatus: map[string]int64{"pending": 5, "completed": 10},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, field := range []string{"active_users", "orders_by_status", "collected_at"} {
		if _, ok := parsed[field]; !ok {
			t.Errorf("MetricsResponse JSON missing field: %s", field)
		}
	}
}
