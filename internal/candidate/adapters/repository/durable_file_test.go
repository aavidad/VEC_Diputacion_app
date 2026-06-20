package repository

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

func TestDurableFileStoreReloadsSnapshotAndContinuesAuditChain(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "bolsa.json")
	at := time.Date(2026, 6, 19, 17, 0, 0, 0, time.UTC)
	store, err := NewDurableFileStore(path)
	if err != nil {
		t.Fatalf("NewDurableFileStore() error = %v", err)
	}
	saveDurableFixture(t, ctx, store, at)

	reopened, err := NewDurableFileStore(path)
	if err != nil {
		t.Fatalf("NewDurableFileStore(reopen) error = %v", err)
	}
	candidate, err := reopened.CandidateRepository().GetByID(ctx, "cand-1")
	if err != nil || candidate.ID != "cand-1" {
		t.Fatalf("GetByID() = %+v, %v", candidate, err)
	}
	merits, err := reopened.MeritRepository().ListByCandidate(ctx, "cand-1")
	if err != nil || len(merits) != 1 {
		t.Fatalf("ListByCandidate() = %+v, %v", merits, err)
	}
	result, ok, err := reopened.BaremoResultRepository().GetByCandidate(ctx, "cand-1")
	if err != nil || !ok || result.TotalPoints != 2 {
		t.Fatalf("GetByCandidate() = %+v/%v, %v", result, ok, err)
	}
	if documents, err := reopened.CandidateDocumentRepository().ListByCandidate(ctx, "cand-1"); err != nil || len(documents) != 1 {
		t.Fatalf("ListByCandidate(documents) = %+v, %v", documents, err)
	}
	audit := reopened.AdministrativeAuditTrail()
	firstEntries, err := audit.ListByScope(ctx, "candidate:cand-1")
	if err != nil || len(firstEntries) != 1 {
		t.Fatalf("ListByScope(initial) = %+v, %v", firstEntries, err)
	}
	second, err := audit.Append(ctx, "candidate:cand-1", domain.AuditEnvelope{
		Actor: "gestor-1", Action: "candidate.notification.sent", OccurredAt: at.Add(time.Minute), Payload: []byte("sent"),
	})
	if err != nil {
		t.Fatalf("Append(after reopen) error = %v", err)
	}
	if second.Sequence != 2 || second.PrevSignature != firstEntries[0].Signature {
		t.Fatalf("continued audit chain = %+v after %+v", second, firstEntries[0])
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !json.Valid(data) {
		t.Fatalf("durable file is not valid JSON")
	}
}

func TestDurableFileStoreReloadsProcedureState(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "bolsa.json")
	store, err := NewDurableFileStore(path)
	if err != nil {
		t.Fatalf("NewDurableFileStore() error = %v", err)
	}
	convocatorias := store.ProcedureConvocatoriaRepository()
	solicitudes := store.ProcedureSolicitudRepository()
	if err := convocatorias.Save(ctx, ports.ConvocatoriaRecord{
		Convocatoria: procedureConvocatoria(t),
		RuleSet:      procedureRuleSet(t),
	}); err != nil {
		t.Fatalf("Save(convocatoria) error = %v", err)
	}
	if err := solicitudes.Save(ctx, procedureSolicitud("sol-1", "cand-1")); err != nil {
		t.Fatalf("Save(solicitud) error = %v", err)
	}

	reopened, err := NewDurableFileStore(path)
	if err != nil {
		t.Fatalf("NewDurableFileStore(reopen) error = %v", err)
	}
	convocatoria, err := reopened.ProcedureConvocatoriaRepository().GetByID(ctx, "conv-1")
	if err != nil || convocatoria.Convocatoria.Version != "v1" {
		t.Fatalf("GetByID(convocatoria) = %+v, %v", convocatoria, err)
	}
	gotSolicitudes, err := reopened.ProcedureSolicitudRepository().ListByConvocatoria(ctx, "conv-1")
	if err != nil || len(gotSolicitudes) != 1 || gotSolicitudes[0].ID != "sol-1" {
		t.Fatalf("ListByConvocatoria() = %+v, %v", gotSolicitudes, err)
	}
}

func TestDurableFileStoreFallsBackToLastGoodBackup(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "bolsa.json")
	store, err := NewDurableFileStore(path)
	if err != nil {
		t.Fatalf("NewDurableFileStore() error = %v", err)
	}
	saveDurableFixture(t, ctx, store, time.Date(2026, 6, 19, 17, 0, 0, 0, time.UTC))
	if err := store.persist(); err != nil {
		t.Fatalf("persist() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"schema_version":`), 0o600); err != nil {
		t.Fatalf("corrupt primary file: %v", err)
	}

	reopened, err := NewDurableFileStore(path)
	if err != nil {
		t.Fatalf("NewDurableFileStore(corrupt primary) error = %v", err)
	}
	candidate, err := reopened.CandidateRepository().GetByID(ctx, "cand-1")
	if err != nil || candidate.ID != "cand-1" {
		t.Fatalf("backup candidate = %+v, %v", candidate, err)
	}
}

func saveDurableFixture(t *testing.T, ctx context.Context, store *DurableFileStore, at time.Time) {
	t.Helper()
	if err := store.CandidateRepository().Save(ctx, "call-1", candidateFixture("cand-1")); err != nil {
		t.Fatalf("Save(candidate) error = %v", err)
	}
	if err := store.MeritRepository().Save(ctx, "cand-1", meritFixture("m1")); err != nil {
		t.Fatalf("Save(merit) error = %v", err)
	}
	if err := store.BaremoResultRepository().Save(ctx, "cand-1", baremoResultFixture()); err != nil {
		t.Fatalf("Save(baremo) error = %v", err)
	}
	document := administrativeDocument(t, "doc-1", "cand-1", "sol-1", at)
	if err := store.CandidateDocumentRepository().Save(ctx, document); err != nil {
		t.Fatalf("Save(document) error = %v", err)
	}
	if err := store.ClaimRepository().Save(ctx, administrativeClaim(t, "claim-1", "cand-1", "sol-1", document, at)); err != nil {
		t.Fatalf("Save(claim) error = %v", err)
	}
	if err := store.NotificationRepository().Save(ctx, administrativeNotification(t, "note-1", "cand-1", "sol-1", at)); err != nil {
		t.Fatalf("Save(notification) error = %v", err)
	}
	if _, err := store.AdministrativeAuditTrail().Append(ctx, "candidate:cand-1", domain.AuditEnvelope{
		Actor: "gestor-1", Action: "candidate.notification.created", OccurredAt: at, Payload: []byte("created"),
	}); err != nil {
		t.Fatalf("Append(audit) error = %v", err)
	}
}
