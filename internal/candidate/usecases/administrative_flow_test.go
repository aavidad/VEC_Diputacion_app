package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

func TestAdministrativeFlowRegistersDocumentsClaimsNotificationsAndAudit(t *testing.T) {
	ctx := context.Background()
	documents := newFakeCandidateDocumentRepository()
	claims := newFakeClaimRepository()
	notifications := newFakeNotificationRepository()
	audit := newFakeAdministrativeAuditTrail()
	useCase, err := NewAdministrativeFlowUseCase(documents, claims, notifications, audit)
	if err != nil {
		t.Fatalf("NewAdministrativeFlowUseCase() error = %v", err)
	}
	at := time.Date(2026, 6, 19, 13, 0, 0, 0, time.UTC)
	evidence := mustUsecaseDocumentEvidence(t, domain.AVStatusClean, at)

	document, documentAudit, err := useCase.RegisterCandidateDocument(ctx, RegisterCandidateDocumentCommand{
		ID: "doc-1", CandidateID: "cand-1", SolicitudID: "sol-1", ProcedureID: "proc-1",
		Purpose: domain.DocumentPurposeAlegacion, Evidence: evidence,
		RegisteredBy: "cand-1", RegisteredAt: at,
	})
	if err != nil {
		t.Fatalf("RegisterCandidateDocument() error = %v", err)
	}
	if documentAudit.Sequence != 1 || documentAudit.Action != "candidate.document.registered" {
		t.Fatalf("document audit = %+v", documentAudit)
	}

	claim, claimAudit, err := useCase.PresentClaim(ctx, PresentClaimCommand{
		ID: "claim-1", CandidateID: "cand-1", SolicitudID: "sol-1",
		Text: "Revision de baremo.", Documents: []domain.CandidateDocument{document},
		PresentedBy: "cand-1", PresentedAt: at.Add(time.Minute), ReceiptCSV: "CSV-ALEG-1",
	})
	if err != nil {
		t.Fatalf("PresentClaim() error = %v", err)
	}
	if claim.State != domain.ClaimStatePresentada || claimAudit.Sequence != 2 {
		t.Fatalf("claim/audit = %+v / %+v", claim, claimAudit)
	}

	notification, _, err := useCase.CreateNotification(ctx, CreateNotificationCommand{
		ID: "not-1", CandidateID: "cand-1", SolicitudID: "sol-1",
		Type: "resolucion_alegacion", Subject: "Alegacion recibida", Body: "Registro correcto.",
		CreatedBy: "gestor-1", CreatedAt: at.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("CreateNotification() error = %v", err)
	}
	notification, sentAudit, err := useCase.SendNotification(ctx, ReceiptCommand{
		NotificationID: notification.ID, CSV: "CSV-NOT-1", RecipientID: "cand-1",
		Channel: "vec", IssuedAt: at.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("SendNotification() error = %v", err)
	}
	if notification.State != domain.NotificationStateEnviada || sentAudit.Action != "candidate.notification.sent" {
		t.Fatalf("sent notification/audit = %+v / %+v", notification, sentAudit)
	}

	if len(audit.entries) != 4 || audit.entries[3].Action != "candidate.notification.sent" {
		t.Fatalf("audit entries = %#v", audit.entries)
	}

	gotClaims, err := useCase.ListClaimsBySolicitud(ctx, " sol-1 ")
	if err != nil {
		t.Fatalf("ListClaimsBySolicitud() error = %v", err)
	}
	if len(gotClaims) != 1 || gotClaims[0].ID != "claim-1" {
		t.Fatalf("claims = %+v", gotClaims)
	}
	gotNotifications, err := useCase.ListNotificationsByCandidate(ctx, "cand-1")
	if err != nil {
		t.Fatalf("ListNotificationsByCandidate() error = %v", err)
	}
	if len(gotNotifications) != 1 || gotNotifications[0].ID != "not-1" {
		t.Fatalf("notifications = %+v", gotNotifications)
	}
	gotAudit, err := useCase.ListAuditByScope(ctx, "candidate:cand-1")
	if err != nil {
		t.Fatalf("ListAuditByScope() error = %v", err)
	}
	if len(gotAudit) != 4 || gotAudit[3].Action != "candidate.notification.sent" {
		t.Fatalf("audit by scope = %+v", gotAudit)
	}
	gotClaims, err = useCase.ListClaimsBySolicitud(ctx, "sol-missing")
	if err != nil || len(gotClaims) != 0 {
		t.Fatalf("missing solicitud claims len/error = %d/%v", len(gotClaims), err)
	}
}

func TestAdministrativeFlowRejectsNotificationRecipientMismatch(t *testing.T) {
	ctx := context.Background()
	notifications := newFakeNotificationRepository()
	audit := newFakeAdministrativeAuditTrail()
	useCase, err := NewAdministrativeFlowUseCase(newFakeCandidateDocumentRepository(), newFakeClaimRepository(), notifications, audit)
	if err != nil {
		t.Fatalf("NewAdministrativeFlowUseCase() error = %v", err)
	}
	at := time.Date(2026, 6, 19, 13, 0, 0, 0, time.UTC)
	notification, _, err := useCase.CreateNotification(ctx, CreateNotificationCommand{
		ID: "not-1", CandidateID: "cand-1", SolicitudID: "sol-1",
		Type: "subsanacion", Subject: "Aportar titulo", Body: "Revise evidencia",
		CreatedBy: "gestor-1", CreatedAt: at,
	})
	if err != nil {
		t.Fatalf("CreateNotification() error = %v", err)
	}
	_, _, err = useCase.SendNotification(ctx, ReceiptCommand{
		NotificationID: notification.ID, CSV: "CSV-NOT-1", RecipientID: "cand-2",
		Channel: "vec", IssuedAt: at.Add(time.Minute),
	})
	if !errors.Is(err, ErrAdministrativeRecipientMismatch) {
		t.Fatalf("SendNotification() error = %v, want recipient mismatch", err)
	}
	stored, err := notifications.GetByID(ctx, notification.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.State != domain.NotificationStateCreada || len(audit.entries) != 1 {
		t.Fatalf("state/audit after mismatch = %s/%d", stored.State, len(audit.entries))
	}
}

func TestAdministrativeFlowRequiresPortsAndStopsAtOutboundPortFailure(t *testing.T) {
	ctx := context.Background()
	if _, err := NewAdministrativeFlowUseCase(nil, newFakeClaimRepository(), newFakeNotificationRepository(), newFakeAdministrativeAuditTrail()); !errors.Is(err, ErrAdministrativeFlowPortsRequired) {
		t.Fatalf("NewAdministrativeFlowUseCase() error = %v, want ports required", err)
	}
	saveErr := errors.New("document port unavailable")
	documents := newFakeCandidateDocumentRepository()
	documents.saveErr = saveErr
	audit := newFakeAdministrativeAuditTrail()
	useCase, err := NewAdministrativeFlowUseCase(documents, newFakeClaimRepository(), newFakeNotificationRepository(), audit)
	if err != nil {
		t.Fatalf("NewAdministrativeFlowUseCase() error = %v", err)
	}
	_, _, err = useCase.RegisterCandidateDocument(ctx, RegisterCandidateDocumentCommand{
		ID: "doc-1", CandidateID: "cand-1", SolicitudID: "sol-1", ProcedureID: "proc-1",
		Purpose: domain.DocumentPurposeAlegacion, Evidence: mustUsecaseDocumentEvidence(t, domain.AVStatusClean, time.Now().UTC()),
		RegisteredBy: "cand-1", RegisteredAt: time.Now().UTC(),
	})
	if !errors.Is(err, saveErr) {
		t.Fatalf("RegisterCandidateDocument() error = %v, want wrapped %v", err, saveErr)
	}
	if len(audit.entries) != 0 {
		t.Fatalf("audit entries after failed port = %#v, want none", audit.entries)
	}
}

type fakeCandidateDocumentRepository struct {
	byID    map[string]domain.CandidateDocument
	saveErr error
}

func newFakeCandidateDocumentRepository() *fakeCandidateDocumentRepository {
	return &fakeCandidateDocumentRepository{byID: map[string]domain.CandidateDocument{}}
}
func (r *fakeCandidateDocumentRepository) Save(ctx context.Context, document domain.CandidateDocument) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.saveErr != nil {
		return r.saveErr
	}
	r.byID[document.ID] = document
	return nil
}
func (r *fakeCandidateDocumentRepository) GetByID(ctx context.Context, id string) (domain.CandidateDocument, error) {
	if err := ctx.Err(); err != nil {
		return domain.CandidateDocument{}, err
	}
	document, ok := r.byID[id]
	if !ok {
		return domain.CandidateDocument{}, ports.ErrCandidateDocumentNotFound
	}
	return document, nil
}
func (r *fakeCandidateDocumentRepository) ListByCandidate(ctx context.Context, candidateID string) ([]domain.CandidateDocument, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []domain.CandidateDocument
	for _, document := range r.byID {
		if document.CandidateID == candidateID {
			out = append(out, document)
		}
	}
	return out, nil
}

type fakeClaimRepository struct {
	byID map[string]domain.Claim
}

func newFakeClaimRepository() *fakeClaimRepository {
	return &fakeClaimRepository{byID: map[string]domain.Claim{}}
}
func (r *fakeClaimRepository) Save(ctx context.Context, claim domain.Claim) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.byID[claim.ID] = claim
	return nil
}
func (r *fakeClaimRepository) GetByID(ctx context.Context, id string) (domain.Claim, error) {
	if err := ctx.Err(); err != nil {
		return domain.Claim{}, err
	}
	claim, ok := r.byID[id]
	if !ok {
		return domain.Claim{}, ports.ErrClaimNotFound
	}
	return claim, nil
}
func (r *fakeClaimRepository) ListBySolicitud(ctx context.Context, solicitudID string) ([]domain.Claim, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []domain.Claim
	for _, claim := range r.byID {
		if claim.SolicitudID == solicitudID {
			out = append(out, claim)
		}
	}
	return out, nil
}

type fakeNotificationRepository struct {
	byID map[string]domain.Notification
}

func newFakeNotificationRepository() *fakeNotificationRepository {
	return &fakeNotificationRepository{byID: map[string]domain.Notification{}}
}
func (r *fakeNotificationRepository) Save(ctx context.Context, notification domain.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.byID[notification.ID] = notification
	return nil
}
func (r *fakeNotificationRepository) GetByID(ctx context.Context, id string) (domain.Notification, error) {
	if err := ctx.Err(); err != nil {
		return domain.Notification{}, err
	}
	notification, ok := r.byID[id]
	if !ok {
		return domain.Notification{}, ports.ErrNotificationNotFound
	}
	return notification, nil
}
func (r *fakeNotificationRepository) ListByCandidate(ctx context.Context, candidateID string) ([]domain.Notification, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out []domain.Notification
	for _, notification := range r.byID {
		if notification.CandidateID == candidateID {
			out = append(out, notification)
		}
	}
	return out, nil
}

type fakeAdministrativeAuditTrail struct {
	entries []domain.AuditEntry
	byScope map[string][]domain.AuditEntry
}

func newFakeAdministrativeAuditTrail() *fakeAdministrativeAuditTrail {
	return &fakeAdministrativeAuditTrail{byScope: map[string][]domain.AuditEntry{}}
}
func (a *fakeAdministrativeAuditTrail) Append(ctx context.Context, scope string, envelope domain.AuditEnvelope) (domain.AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return domain.AuditEntry{}, err
	}
	var previous *domain.AuditEntry
	scoped := a.byScope[scope]
	if len(scoped) > 0 {
		previous = &scoped[len(scoped)-1]
	}
	entry, err := domain.NewAuditEntry(previous, envelope, "test-signing-ref")
	if err != nil {
		return domain.AuditEntry{}, err
	}
	a.entries = append(a.entries, entry)
	a.byScope[scope] = append(scoped, entry)
	return entry, nil
}
func (a *fakeAdministrativeAuditTrail) ListByScope(ctx context.Context, scope string) ([]domain.AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return append([]domain.AuditEntry(nil), a.byScope[scope]...), nil
}

func mustUsecaseDocumentEvidence(t *testing.T, status domain.AVStatus, at time.Time) domain.DocumentEvidence {
	t.Helper()
	doc, err := domain.NewDocumentEvidence(domain.DocumentEvidenceInput{
		CSV: "CSV-2026-0001", DigestSHA256: "abc123",
		Refs: domain.DocumentExternalRefs{StorageObjectRef: "object-ref", TSAStampRef: "tsa-ref"},
		ENI: domain.ENIMetadata{
			DocumentID: "doc-1", Organ: "Diputacion de Granada", Procedure: "proc-1",
			DocumentType: "alegacion", Title: "Alegacion", Format: "application/pdf",
			Language: "es", CreatedAt: at,
		},
		AVStatus: status, SubmittedBy: "cand-1", SubmittedAt: at,
		Signatures: []domain.SignatureEvidence{{SignerID: "cand-1", SignatureRef: "sig-ref", SignedAt: at}},
	})
	if err != nil {
		t.Fatalf("NewDocumentEvidence() error = %v", err)
	}
	return doc
}
