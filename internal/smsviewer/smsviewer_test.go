package smsviewer_test

import (
	"strings"
	"testing"

	"bfr-webui-go/internal/smsviewer"
)

func TestParseContentProviderOutputPreservesCommaBodyAndOrder(t *testing.T) {
	output := []byte("Row: 0 _id=2, address=+1002, date=1710000000000, read=1, type=1, body=Second, with commas\n" +
		"Row: 1 _id=1, address=+1001, date=1700000000000, read=0, type=1, body=First message\n")

	resp := smsviewer.ParseContentProviderOutput(output, 1, 0, "")
	if resp.Total != 2 || len(resp.Messages) != 1 {
		t.Fatalf("expected first page of two messages, got total=%d messages=%d", resp.Total, len(resp.Messages))
	}
	if resp.Messages[0].ID != 2 || resp.Messages[0].Body != "Second, with commas" {
		t.Fatalf("unexpected parsed message: %+v", resp.Messages[0])
	}

	next := smsviewer.ParseContentProviderOutput(output, 1, 1, "")
	if len(next.Messages) != 1 || next.Messages[0].ID != 1 {
		t.Fatalf("expected second message on next page, got %+v", next.Messages)
	}
}

func TestSMSMessage_Struct(t *testing.T) {
	msg := smsviewer.SMSMessage{
		ID:       1,
		Address:  "+628123456789",
		Body:     "Test OTP Code: 123456",
		Date:     1700000000,
		DateSent: 1700000000,
		Read:     1,
		Type:     1,
	}

	if msg.ID != 1 {
		t.Errorf("expected ID 1, got %d", msg.ID)
	}
	if msg.Address != "+628123456789" {
		t.Errorf("expected address +628123456789, got %s", msg.Address)
	}
	if !strings.Contains(msg.Body, "OTP") {
		t.Errorf("expected body to contain OTP, got %s", msg.Body)
	}
}

func TestReadSMSInbox_Defaults(t *testing.T) {
	// Call ReadSMSInbox in test env (where content provider / sqlite db won't exist or return empty)
	resp, err := smsviewer.ReadSMSInbox(-5, -10, "testQuery")
	if err != nil {
		t.Fatalf("unexpected error from ReadSMSInbox: %v", err)
	}

	// Verify limit & offset sanitization (limit bounded 1..100, offset >= 0)
	if resp.Limit != 20 {
		t.Errorf("expected default limit 20 when non-positive provided, got %d", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Errorf("expected sanitized offset 0 when negative provided, got %d", resp.Offset)
	}
	if resp.Search != "testQuery" {
		t.Errorf("expected Search 'testQuery', got %s", resp.Search)
	}
}

func TestReadSMSInbox_MaxLimit(t *testing.T) {
	resp, err := smsviewer.ReadSMSInbox(500, 10, "")
	if err != nil {
		t.Fatalf("unexpected error from ReadSMSInbox: %v", err)
	}

	if resp.Limit != 100 {
		t.Errorf("expected limit capped at 100, got %d", resp.Limit)
	}
}
