package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

func TestAdministrativeFlowMemoryRepositoriesPersistAndListDeterministically(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 6, 19, 14, 0, 0, 0, time.UTC)
	store := NewAdministrativeFlowMemoryStore()
	documents := NewAdministrativeCandidateDocumentRepository(store)
	claims := NewAdministrativeClaimRepository(store)
	notifications := NewAdministrativeNotificationRepository(store)
	audit := NewAdministrativeAuditTrail(store)

	for _, document := range []domain.CandidateDocument{
		administrativeDocument(t, "doc-2", "cand-1", "sol-1", at),
		administrativeDocument(t, "doc-1", "cand-1", "sol-1", at),
	} {
		if err := documents.Save(ctx, document); err != nil {
			t.Fatalf("Save(document %s) error = %v", document.ID, err)
		}
	}
	gotDocuments, err := documents.ListByCandidate(ctx, "cand-1")
	if err != nil {
		t.Fatalf("ListByCandidate(documents) error = %v", err)
	}
	if got := documentIDs(gotDocuments); !reflect.DeepEqual(got, []string{"doc-1", "doc-2"}) {
		t.Fatalf("document IDs = %#v, want sorted", got)
	}

	claim := administrativeClaim(t, "claim-1", "cand-1", "sol-1", gotDocuments[0], at)
	if err := claims.Save(ctx, claim); err != nil {
		t.Fatalf("Save(claim) error = %v", err)
	}
	gotClaims, err := claims.ListBySolicitud(ctx, "sol-1")
	if err != nil {
		t.Fatalf("ListBySolicitud() error = %v", err)
	}
	if got := claimIDs(gotClaims); !reflect.DeepEqual(got, []string{"claim-1"}) {
		t.Fatalf("claim IDs = %#v", got)
	}

	notification := administrativeNotification(t, "not-1", "cand-1", "sol-1", at)
	if err := notifications.Save(ctx, notification); err != nil {
		t.Fatalf("Save(notification) error = %v", err)
	}
	gotNotifications, err := notifications.ListByCandidate(ctx, "cand-1")
	if err != nil {
		t.Fatalf("ListByCandidate(notifications) error = %v", err)
	}
	if got := notificationIDs(gotNotifications); !reflect.DeepEqual(got, []string{"not-1"}) {
		t.Fatalf("notification IDs = %#v", got)
	}

	first := appendAudit(t, ctx, audit, "candidate:cand-1", "candidate.document.registered", at)
	second := appendAudit(t, ctx, audit, "candidate:cand-1", "candidate.claim.presented", at.Add(time.Minute))
	if first.Sequence != 1 || second.Sequence != 2 || second.PrevSignature != first.Signature {
		t.Fatalf("audit chain first=%+v second=%+v", first, second)
	}
	gotAudit, err := audit.ListByScope(ctx, "candidate:cand-1")
	if err != nil {
		t.Fatalf("ListByScope() error = %v", err)
	}
	if len(gotAudit) != 2 || gotAudit[0].Action != "candidate.document.registered" {
		t.Fatalf("audit by scope = %+v", gotAudit)
	}
}

func TestAdministrativeFlowMemoryRepositoriesMoveIndexesAndReturnNotFound(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 6, 19, 15, 0, 0, 0, time.UTC)
	store := NewAdministrativeFlowMemoryStore()
	documents := NewAdministrativeCandidateDocumentRepository(store)
	claims := NewAdministrativeClaimRepository(store)
	notifications := NewAdministrativeNotificationRepository(store)

	document := administrativeDocument(t, "doc-1", "cand-1", "sol-1", at)
	if err := documents.Save(ctx, document); err != nil {
		t.Fatalf("Save(document initial) error = %v", err)
	}
	document.CandidateID = "cand-2"
	if err := documents.Save(ctx, document); err != nil {
		t.Fatalf("Save(document moved) error = %v", err)
	}
	if oldDocs, _ := documents.ListByCandidate(ctx, "cand-1"); len(oldDocs) != 0 {
		t.Fatalf("old candidate documents = %d, want 0", len(oldDocs))
	}

	claim := administrativeClaim(t, "claim-1", "cand-2", "sol-1", document, at)
	if err := claims.Save(ctx, claim); err != nil {
		t.Fatalf("Save(claim initial) error = %v", err)
	}
	claim.SolicitudID = "sol-2"
	if err := claims.Save(ctx, claim); err != nil {
		t.Fatalf("Save(claim moved) error = %v", err)
	}
	if oldClaims, _ := claims.ListBySolicitud(ctx, "sol-1"); len(oldClaims) != 0 {
		t.Fatalf("old solicitud claims = %d, want 0", len(oldClaims))
	}

	notification := administrativeNotification(t, "not-1", "cand-2", "sol-2", at)
	if err := notifications.Save(ctx, notification); err != nil {
		t.Fatalf("Save(notification initial) error = %v", err)
	}
	notification.CandidateID = "cand-3"
	if err := notifications.Save(ctx, notification); err != nil {
		t.Fatalf("Save(notification moved) error = %v", err)
	}
	if oldNotifications, _ := notifications.ListByCandidate(ctx, "cand-2"); len(oldNotifications) != 0 {
		t.Fatalf("old candidate notifications = %d, want 0", len(oldNotifications))
	}

	if _, err := documents.GetByID(ctx, "missing"); !errors.Is(err, ports.ErrCandidateDocumentNotFound) {
		t.Fatalf("GetByID(document missing) error = %v", err)
	}
	if _, err := claims.GetByID(ctx, "missing"); !errors.Is(err, ports.ErrClaimNotFound) {
		t.Fatalf("GetByID(claim missing) error = %v", err)
	}
	if _, err := notifications.GetByID(ctx, "missing"); !errors.Is(err, ports.ErrNotificationNotFound) {
		t.Fatalf("GetByID(notification missing) error = %v", err)
	}
}

func TestAdministrativeFlowMemoryRepositoriesConcurrentSafeAndDefensiveCopies(t *testing.T) {
	ctx := context.Background()
	at := time.Date(2026, 6, 19, 16, 0, 0, 0, time.UTC)
	documents := NewAdministrativeCandidateDocumentRepository(nil)
	const total = 32
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("doc-%02d", i)
			if err := documents.Save(ctx, administrativeDocument(t, id, "cand-1", "sol-1", at)); err != nil {
				t.Errorf("Save(%s) error = %v", id, err)
			}
		}(i)
	}
	wg.Wait()

	gotDocuments, err := documents.ListByCandidate(ctx, "cand-1")
	if err != nil {
		t.Fatalf("ListByCandidate() error = %v", err)
	}
	if len(gotDocuments) != total || gotDocuments[0].ID != "doc-00" {
		t.Fatalf("documents len/first = %d/%q", len(gotDocuments), gotDocuments[0].ID)
	}
	gotDocuments[0].Evidence.Signatures[0].SignatureRef = "mutated"
	got, err := documents.GetByID(ctx, gotDocuments[0].ID)
	if err != nil {
		t.Fatalf("GetByID(%s) error = %v", gotDocuments[0].ID, err)
	}
	if got.Evidence.Signatures[0].SignatureRef == "mutated" {
		t.Fatalf("repository leaked mutable document signatures")
	}

	audit := NewAdministrativeAuditTrail(nil)
	appendAudit(t, ctx, audit, "candidate:cand-1", "candidate.document.registered", at)
	gotAudit, err := audit.ListByScope(ctx, "candidate:cand-1")
	if err != nil {
		t.Fatalf("ListByScope() error = %v", err)
	}
	gotAudit[0].Action = "mutated"
	gotAudit, err = audit.ListByScope(ctx, "candidate:cand-1")
	if err != nil {
		t.Fatalf("ListByScope(second) error = %v", err)
	}
	if gotAudit[0].Action == "mutated" {
		t.Fatalf("repository leaked mutable audit entries")
	}
}

func administrativeDocument(t *testing.T, id, candidateID, solicitudID string, at time.Time) domain.CandidateDocument {
	t.Helper()
	document, err := domain.NewCandidateDocument(domain.CandidateDocumentInput{
		ID: id, CandidateID: candidateID, SolicitudID: solicitudID, ProcedureID: "proc-1",
		Purpose: domain.DocumentPurposeAlegacion, Evidence: administrativeEvidence(t, id, at),
		RegisteredBy: candidateID, RegisteredAt: at,
	})
	if err != nil {
		t.Fatalf("NewCandidateDocument(%s) error = %v", id, err)
	}
	return document
}

func administrativeEvidence(t *testing.T, id string, at time.Time) domain.DocumentEvidence {
	t.Helper()
	evidence, err := domain.NewDocumentEvidence(domain.DocumentEvidenceInput{
		CSV: "CSV-" + id, DigestSHA256: "abc123",
		Refs: domain.DocumentExternalRefs{StorageObjectRef: "object-" + id, TSAStampRef: "tsa-" + id},
		ENI: domain.ENIMetadata{
			DocumentID: id, Organ: "Diputacion de Granada", Procedure: "proc-1",
			DocumentType: "alegacion", Title: "Alegacion", Format: "application/pdf",
			Language: "es", CreatedAt: at,
		},
		AVStatus: domain.AVStatusClean, SubmittedBy: "cand-1", SubmittedAt: at,
		Signatures: []domain.SignatureEvidence{{SignerID: "cand-1", SignatureRef: "sig-" + id, SignedAt: at}},
	})
	if err != nil {
		t.Fatalf("NewDocumentEvidence(%s) error = %v", id, err)
	}
	return evidence
}

func administrativeClaim(t *testing.T, id, candidateID, solicitudID string, document domain.CandidateDocument, at time.Time) domain.Claim {
	t.Helper()
	claim, err := domain.NewClaim(domain.ClaimInput{
		ID: id, CandidateID: candidateID, SolicitudID: solicitudID,
		Text: "Solicito revision.", Documents: []domain.CandidateDocument{document},
		PresentedBy: candidateID, PresentedAt: at, ReceiptCSV: "CSV-" + id,
	})
	if err != nil {
		t.Fatalf("NewClaim(%s) error = %v", id, err)
	}
	return claim
}

func administrativeNotification(t *testing.T, id, candidateID, solicitudID string, at time.Time) domain.Notification {
	t.Helper()
	notification, err := domain.NewNotification(domain.NotificationInput{
		ID: id, CandidateID: candidateID, SolicitudID: solicitudID,
		Type: "alegacion", Subject: "Alegacion registrada", Body: "Registro correcto.",
		CreatedBy: "gestor-1", CreatedAt: at,
	})
	if err != nil {
		t.Fatalf("NewNotification(%s) error = %v", id, err)
	}
	return notification
}

func appendAudit(
	t *testing.T,
	ctx context.Context,
	audit *AdministrativeAuditTrail,
	scope string,
	action string,
	at time.Time,
) domain.AuditEntry {
	t.Helper()
	entry, err := audit.Append(ctx, scope, domain.AuditEnvelope{
		Actor: "gestor-1", Action: action, OccurredAt: at, Payload: []byte(action),
	})
	if err != nil {
		t.Fatalf("Append(%s) error = %v", action, err)
	}
	return entry
}

func documentIDs(documents []domain.CandidateDocument) []string {
	ids := make([]string, 0, len(documents))
	for _, document := range documents {
		ids = append(ids, document.ID)
	}
	return ids
}

func claimIDs(claims []domain.Claim) []string {
	ids := make([]string, 0, len(claims))
	for _, claim := range claims {
		ids = append(ids, claim.ID)
	}
	return ids
}

func notificationIDs(notifications []domain.Notification) []string {
	ids := make([]string, 0, len(notifications))
	for _, notification := range notifications {
		ids = append(ids, notification.ID)
	}
	return ids
}
