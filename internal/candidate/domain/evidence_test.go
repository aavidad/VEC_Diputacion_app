package domain

import (
	"errors"
	"testing"
	"time"
)

func TestDocumentEvidenceQuarantinedUntilAVClean(t *testing.T) {
	doc := mustDocumentEvidence(t, AVStatusPending)

	if !doc.IsQuarantined() {
		t.Fatalf("pending document should be quarantined")
	}
	if err := doc.EnsureExportable(); !errors.Is(err, ErrDocumentQuarantined) {
		t.Fatalf("EnsureExportable() error = %v, want %v", err, ErrDocumentQuarantined)
	}
	if err := doc.MarkAVStatus(AVStatusClean); err != nil {
		t.Fatalf("MarkAVStatus(CLEAN) error = %v", err)
	}
	if doc.IsQuarantined() {
		t.Fatalf("clean document should not be quarantined")
	}
	if err := doc.EnsureExportable(); err != nil {
		t.Fatalf("EnsureExportable() clean error = %v", err)
	}
}

func TestElectronicFileExportManifestReconstructsTraceability(t *testing.T) {
	submittedAt := time.Date(2026, 6, 19, 9, 30, 0, 0, time.UTC)
	doc := mustDocumentEvidenceAt(t, AVStatusClean, submittedAt)
	file := ElectronicFile{
		ID:          "exp-1",
		CandidateID: "cand-1",
		ProcedureID: "proc-1",
		CreatedBy:   "gestor-1",
		CreatedAt:   submittedAt,
		Documents:   []DocumentEvidence{doc},
	}

	manifest, err := file.ExportManifest()
	if err != nil {
		t.Fatalf("ExportManifest() error = %v", err)
	}
	if manifest.FileID != "exp-1" || len(manifest.Items) != 1 {
		t.Fatalf("manifest = %+v", manifest)
	}
	item := manifest.Items[0]
	if item.Who != "cand-1" || item.What != "Solicitud bolsa" || !item.When.Equal(submittedAt) {
		t.Fatalf("traceability item = %+v", item)
	}
	if len(item.SignatureRefs) != 1 || item.SignatureRefs[0] != "signature-ref-1" {
		t.Fatalf("SignatureRefs = %+v", item.SignatureRefs)
	}
	if item.TSAStampRef != "tsa-ref-1" {
		t.Fatalf("TSAStampRef = %q", item.TSAStampRef)
	}
}

func TestDocumentEvidenceRequiresENIMinimum(t *testing.T) {
	input := validDocumentEvidenceInput(AVStatusClean, time.Date(2026, 6, 19, 9, 30, 0, 0, time.UTC))
	input.ENI.DocumentID = " "

	_, err := NewDocumentEvidence(input)
	if !errors.Is(err, ErrDocumentInvalid) {
		t.Fatalf("NewDocumentEvidence() error = %v, want %v", err, ErrDocumentInvalid)
	}
}

func mustDocumentEvidence(t *testing.T, status AVStatus) DocumentEvidence {
	t.Helper()
	return mustDocumentEvidenceAt(t, status, time.Date(2026, 6, 19, 9, 30, 0, 0, time.UTC))
}

func mustDocumentEvidenceAt(t *testing.T, status AVStatus, submittedAt time.Time) DocumentEvidence {
	t.Helper()
	doc, err := NewDocumentEvidence(validDocumentEvidenceInput(status, submittedAt))
	if err != nil {
		t.Fatalf("NewDocumentEvidence() error = %v", err)
	}
	return doc
}

func validDocumentEvidenceInput(status AVStatus, submittedAt time.Time) DocumentEvidenceInput {
	return DocumentEvidenceInput{
		CSV:          "CSV-2026-0001",
		DigestSHA256: "abc123",
		Refs: DocumentExternalRefs{
			StorageObjectRef: "minio-object-ref-1",
			VaultSecretRef:   "openbao-secret-ref-1",
			TSAStampRef:      "tsa-ref-1",
		},
		ENI: ENIMetadata{
			DocumentID:   "doc-1",
			Organ:        "Diputacion de Granada",
			Procedure:    "proc-1",
			DocumentType: "solicitud",
			Title:        "Solicitud bolsa",
			Format:       "application/pdf",
			Language:     "es",
			CreatedAt:    submittedAt,
		},
		AVStatus:    status,
		SubmittedBy: "cand-1",
		SubmittedAt: submittedAt,
		Signatures: []SignatureEvidence{{
			SignerID:     "cand-1",
			SignatureRef: "signature-ref-1",
			SignedAt:     submittedAt,
		}},
	}
}
