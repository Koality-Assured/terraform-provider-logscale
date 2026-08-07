package provider

import (
	"strings"
	"testing"
)

func TestParseAndRenderCSVContentRoundTrip(t *testing.T) {
	input := "host,ip\nweb-01,10.0.0.1\nweb-02,10.0.0.2\n"

	snapshot, err := parseCSVContent(input)
	if err != nil {
		t.Fatalf("unexpected error parsing csv content: %v", err)
	}
	if len(snapshot.Headers) != 2 || snapshot.Headers[0] != "host" {
		t.Fatalf("unexpected headers: %#v", snapshot.Headers)
	}
	if len(snapshot.Rows) != 2 || snapshot.Rows[1][0] != "web-02" {
		t.Fatalf("unexpected rows: %#v", snapshot.Rows)
	}

	rendered, err := renderCSVContent(snapshot.Headers, snapshot.Rows)
	if err != nil {
		t.Fatalf("unexpected error rendering csv content: %v", err)
	}
	if normalizeNewlines(rendered) != normalizeNewlines(input) {
		t.Fatalf("expected rendered csv to round-trip, got %q", rendered)
	}
}

func TestCSVRowsToUpdatePayload(t *testing.T) {
	payload, err := csvRowsToUpdatePayload([][]string{
		{"web-01", "10.0.0.1"},
		{"web,02", "10.0.0.2"},
	})
	if err != nil {
		t.Fatalf("unexpected error converting csv rows: %v", err)
	}
	if len(payload) != 2 {
		t.Fatalf("expected 2 payload rows, got %d", len(payload))
	}
	if !strings.Contains(payload[1], "\"web,02\"") {
		t.Fatalf("expected csv escaping in payload, got %q", payload[1])
	}
}

func TestComputeColumnChanges(t *testing.T) {
	changes := computeColumnChanges([]string{"host"}, []string{"host", "ip"})
	if len(changes) != 1 || changes[0]["changeKind"] != "Add" {
		t.Fatalf("expected an Add column change, got %#v", changes)
	}

	changes = computeColumnChanges([]string{"host", "ip"}, []string{"host"})
	if len(changes) != 1 || changes[0]["changeKind"] != "Remove" {
		t.Fatalf("expected a Remove column change, got %#v", changes)
	}
}
