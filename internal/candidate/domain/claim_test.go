package domain

import (
	"errors"
	"testing"
	"time"
)

func TestClaimStartsPresentedWithAuditableReceipt(t *testing.T) {
	at := time.Date(2026, 6, 19, 11, 0, 0, 0, time.UTC)
	claim := mustClaim(t, at)

	if claim.State != ClaimStatePresentada {
		t.Fatalf("state = %s, want %s", claim.State, ClaimStatePresentada)
	}
	if claim.Receipt.CSV != "CSV-ALEG-1" || claim.Receipt.PayloadHash == "" {
		t.Fatalf("receipt = %+v", claim.Receipt)
	}
	if err := claim.Transition(ClaimStateEnRevision); err != nil {
		t.Fatalf("Transition(EnRevision) error = %v", err)
	}
	if err := claim.Transition(ClaimStateEstimada); err != nil {
		t.Fatalf("Transition(Estimada) error = %v", err)
	}
}

func TestClaimRejectsQuarantinedDocuments(t *testing.T) {
	at := time.Date(2026, 6, 19, 11, 0, 0, 0, time.UTC)
	input := validClaimInput(at)
	input.Documents = []CandidateDocument{mustCandidateDocument(t, AVStatusPending, at)}

	_, err := NewClaim(input)
	if !errors.Is(err, ErrDocumentQuarantined) {
		t.Fatalf("NewClaim() error = %v, want %v", err, ErrDocumentQuarantined)
	}
}

func TestClaimRejectsInvalidTransition(t *testing.T) {
	claim := mustClaim(t, time.Date(2026, 6, 19, 11, 0, 0, 0, time.UTC))

	err := claim.Transition(ClaimStateEstimada)
	if !errors.Is(err, ErrClaimTransition) {
		t.Fatalf("Transition(Estimada) error = %v, want %v", err, ErrClaimTransition)
	}
}

func mustClaim(t *testing.T, at time.Time) Claim {
	t.Helper()
	claim, err := NewClaim(validClaimInput(at))
	if err != nil {
		t.Fatalf("NewClaim() error = %v", err)
	}
	return claim
}

func validClaimInput(at time.Time) ClaimInput {
	return ClaimInput{
		ID: "claim-1", CandidateID: "cand-1", SolicitudID: "sol-1",
		Text:        "Solicito revision del baremo provisional.",
		Documents:   []CandidateDocument{mustCandidateDocumentForClaim(at)},
		PresentedBy: "cand-1", PresentedAt: at, ReceiptCSV: "CSV-ALEG-1",
	}
}

func mustCandidateDocumentForClaim(at time.Time) CandidateDocument {
	document, err := NewCandidateDocument(validCandidateDocumentInput(AVStatusClean, at))
	if err != nil {
		panic(err)
	}
	return document
}
