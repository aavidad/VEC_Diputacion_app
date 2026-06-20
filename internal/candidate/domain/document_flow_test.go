package domain

import (
	"errors"
	"testing"
	"time"
)

func TestCandidateDocumentUsesENICSVAndBlocksQuarantinedExport(t *testing.T) {
	at := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)
	document := mustCandidateDocument(t, AVStatusPending, at)

	if document.Evidence.CSV != "CSV-2026-0001" || document.Evidence.ENI.DocumentID != "doc-1" {
		t.Fatalf("document evidence = %+v", document.Evidence)
	}
	if err := document.EnsureExportable(); !errors.Is(err, ErrDocumentQuarantined) {
		t.Fatalf("EnsureExportable() error = %v, want %v", err, ErrDocumentQuarantined)
	}

	document.Evidence.AVStatus = AVStatusClean
	item, err := document.ExportManifestItem()
	if err != nil {
		t.Fatalf("ExportManifestItem() error = %v", err)
	}
	if item.CSV != "CSV-2026-0001" || item.What != "Solicitud bolsa" {
		t.Fatalf("manifest item = %+v", item)
	}
}

func TestCandidateDocumentRequiresCandidateScope(t *testing.T) {
	input := validCandidateDocumentInput(AVStatusClean, time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC))
	input.CandidateID = " "

	_, err := NewCandidateDocument(input)
	if !errors.Is(err, ErrCandidateDocumentInvalid) {
		t.Fatalf("NewCandidateDocument() error = %v, want %v", err, ErrCandidateDocumentInvalid)
	}
}

func mustCandidateDocument(t *testing.T, status AVStatus, at time.Time) CandidateDocument {
	t.Helper()
	document, err := NewCandidateDocument(validCandidateDocumentInput(status, at))
	if err != nil {
		t.Fatalf("NewCandidateDocument() error = %v", err)
	}
	return document
}

func validCandidateDocumentInput(status AVStatus, at time.Time) CandidateDocumentInput {
	return CandidateDocumentInput{
		ID: "doc-flow-1", CandidateID: "cand-1", SolicitudID: "sol-1",
		ProcedureID: "proc-1", Purpose: DocumentPurposeSolicitud,
		Evidence:     mustDocumentEvidenceForInput(status, at),
		RegisteredBy: "cand-1", RegisteredAt: at,
	}
}

func mustDocumentEvidenceForInput(status AVStatus, at time.Time) DocumentEvidence {
	doc, err := NewDocumentEvidence(validDocumentEvidenceInput(status, at))
	if err != nil {
		panic(err)
	}
	return doc
}
