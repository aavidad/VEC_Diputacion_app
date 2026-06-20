package domain

import (
	"errors"
	"testing"
	"time"
)

func TestAuditChainValidatesLinkedSignatures(t *testing.T) {
	entries := mustAuditChain(t)

	if err := VerifyAuditChain(entries, "openbao-signing-ref-1"); err != nil {
		t.Fatalf("VerifyAuditChain() error = %v", err)
	}
	if entries[1].PrevSignature != entries[0].Signature {
		t.Fatalf("PrevSignature was not chained")
	}
}

func TestAuditChainDetectsPayloadHashManipulation(t *testing.T) {
	entries := mustAuditChain(t)
	entries[0].PayloadHash = HashAuditPayload([]byte(`{"csv":"tampered"}`))

	if err := VerifyAuditChain(entries, "openbao-signing-ref-1"); !errors.Is(err, ErrAuditChainInvalid) {
		t.Fatalf("VerifyAuditChain() error = %v, want %v", err, ErrAuditChainInvalid)
	}
}

func TestAuditChainDetectsPreviousSignatureManipulation(t *testing.T) {
	entries := mustAuditChain(t)
	entries[1].PrevSignature = "bad-prev-signature"

	if err := VerifyAuditChain(entries, "openbao-signing-ref-1"); !errors.Is(err, ErrAuditChainInvalid) {
		t.Fatalf("VerifyAuditChain() error = %v, want %v", err, ErrAuditChainInvalid)
	}
}

func mustAuditChain(t *testing.T) []AuditEntry {
	t.Helper()
	at := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)
	first, err := NewAuditEntry(nil, AuditEnvelope{
		Actor:      "cand-1",
		Action:     "document.uploaded",
		OccurredAt: at,
		Payload:    []byte(`{"csv":"CSV-2026-0001"}`),
	}, "openbao-signing-ref-1")
	if err != nil {
		t.Fatalf("NewAuditEntry(first) error = %v", err)
	}
	second, err := NewAuditEntry(&first, AuditEnvelope{
		Actor:      "antivirus",
		Action:     "document.clean",
		OccurredAt: at.Add(time.Minute),
		Payload:    []byte(`{"av_status":"CLEAN"}`),
	}, "openbao-signing-ref-1")
	if err != nil {
		t.Fatalf("NewAuditEntry(second) error = %v", err)
	}
	return []AuditEntry{first, second}
}
