package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

var errUnexpectedContextPortAccess = errors.New("unexpected port access")

func TestUseCasesRejectInvalidContextBeforeAnyPortAccess(t *testing.T) {
	accesses := 0
	procedure, err := NewProcedureUseCase(
		contextProbeConvocatoriaRepository{accesses: &accesses},
		contextProbeSolicitudRepository{accesses: &accesses},
	)
	if err != nil {
		t.Fatalf("NewProcedureUseCase() error = %v", err)
	}
	baremo, err := NewBaremoUseCase(
		contextProbeMeritRepository{accesses: &accesses},
		contextProbeBaremoResultRepository{accesses: &accesses},
	)
	if err != nil {
		t.Fatalf("NewBaremoUseCase() error = %v", err)
	}
	administrative, err := NewAdministrativeFlowUseCase(
		contextProbeDocumentRepository{accesses: &accesses},
		contextProbeClaimRepository{accesses: &accesses},
		contextProbeNotificationRepository{accesses: &accesses},
		contextProbeAuditTrail{accesses: &accesses},
	)
	if err != nil {
		t.Fatalf("NewAdministrativeFlowUseCase() error = %v", err)
	}

	operations := []struct {
		name string
		run  func(context.Context) error
	}{
		{name: "crear convocatoria", run: func(ctx context.Context) error {
			_, err := procedure.CrearConvocatoria(ctx, CrearConvocatoriaCommand{})
			return err
		}},
		{name: "asegurar convocatoria", run: func(ctx context.Context) error {
			_, err := procedure.EnsureConvocatoria(ctx, CrearConvocatoriaCommand{})
			return err
		}},
		{name: "registrar solicitud", run: func(ctx context.Context) error {
			_, err := procedure.RegistrarSolicitud(ctx, RegistrarSolicitudCommand{})
			return err
		}},
		{name: "asegurar solicitud", run: func(ctx context.Context) error {
			_, err := procedure.EnsureSolicitud(ctx, RegistrarSolicitudCommand{})
			return err
		}},
		{name: "publicar listado provisional", run: func(ctx context.Context) error {
			_, err := procedure.PublicarListadoProvisional(ctx, "", nil)
			return err
		}},
		{name: "publicar listado definitivo", run: func(ctx context.Context) error {
			_, err := procedure.PublicarListadoDefinitivo(ctx, "", nil)
			return err
		}},
		{name: "consultar listado actual", run: func(ctx context.Context) error {
			_, err := procedure.ListadoActual(ctx, "")
			return err
		}},
		{name: "presentar solicitud baremada", run: func(ctx context.Context) error {
			_, err := baremo.PresentarSolicitud(ctx, "", domain.BaremoRuleSet{})
			return err
		}},
		{name: "calcular autobaremo", run: func(ctx context.Context) error {
			_, err := baremo.CalcularAutobaremo(ctx, "", domain.BaremoRuleSet{})
			return err
		}},
		{name: "consultar puntuacion provisional", run: func(ctx context.Context) error {
			_, err := baremo.PuntuacionProvisional(ctx, "", domain.BaremoRuleSet{})
			return err
		}},
		{name: "registrar documento", run: func(ctx context.Context) error {
			_, _, err := administrative.RegisterCandidateDocument(ctx, RegisterCandidateDocumentCommand{})
			return err
		}},
		{name: "presentar alegacion", run: func(ctx context.Context) error {
			_, _, err := administrative.PresentClaim(ctx, PresentClaimCommand{})
			return err
		}},
		{name: "crear notificacion", run: func(ctx context.Context) error {
			_, _, err := administrative.CreateNotification(ctx, CreateNotificationCommand{})
			return err
		}},
		{name: "enviar notificacion", run: func(ctx context.Context) error {
			_, _, err := administrative.SendNotification(ctx, ReceiptCommand{})
			return err
		}},
		{name: "marcar notificacion leida", run: func(ctx context.Context) error {
			_, _, err := administrative.MarkNotificationRead(ctx, ReceiptCommand{})
			return err
		}},
		{name: "listar alegaciones", run: func(ctx context.Context) error {
			_, err := administrative.ListClaimsBySolicitud(ctx, "")
			return err
		}},
		{name: "listar notificaciones", run: func(ctx context.Context) error {
			_, err := administrative.ListNotificationsByCandidate(ctx, "")
			return err
		}},
		{name: "listar auditoria", run: func(ctx context.Context) error {
			_, err := administrative.ListAuditByScope(ctx, "")
			return err
		}},
	}

	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, cancelExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelExpired()
	contexts := []struct {
		name string
		ctx  context.Context
		want error
	}{
		{name: "nulo", ctx: nil, want: ErrContextRequired},
		{name: "cancelado", ctx: cancelled, want: context.Canceled},
		{name: "vencido", ctx: expired, want: context.DeadlineExceeded},
	}

	for _, operation := range operations {
		for _, testContext := range contexts {
			t.Run(operation.name+"/"+testContext.name, func(t *testing.T) {
				accesses = 0
				err := operation.run(testContext.ctx)
				if !errors.Is(err, testContext.want) {
					t.Fatalf("error = %v, want %v", err, testContext.want)
				}
				if accesses != 0 {
					t.Fatalf("port accesses = %d, want 0", accesses)
				}
			})
		}
	}
}

func TestPresentarSolicitudStopsAfterCancellationBetweenMeritWrites(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	merits := &cancelAfterFirstMeritSaveRepository{
		cancel: cancel,
		merits: []domain.Merit{
			{ID: "m1", Tipo: domain.MeritTypeExperienciaMismaCategoria, Datos: domain.MeritData{Meses: 1}, Estado: domain.MeritStateBorrador},
			{ID: "m2", Tipo: domain.MeritTypeExperienciaMismaCategoria, Datos: domain.MeritData{Meses: 1}, Estado: domain.MeritStateBorrador},
		},
	}
	results := &countingBaremoResultRepository{}
	useCase, err := NewBaremoUseCase(merits, results)
	if err != nil {
		t.Fatalf("NewBaremoUseCase() error = %v", err)
	}

	_, err = useCase.PresentarSolicitud(ctx, "cand-1", testBaremoRuleSet(t))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PresentarSolicitud() error = %v, want %v", err, context.Canceled)
	}
	if merits.saveCalls != 1 {
		t.Fatalf("merit saves = %d, want 1", merits.saveCalls)
	}
	if results.saveCalls != 0 {
		t.Fatalf("result saves = %d, want 0", results.saveCalls)
	}
}

func TestEnsureSolicitudRechecksCancellationBeforeSave(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	convocatoria, err := domain.NewConvocatoria("conv-1", "v1")
	if err != nil {
		t.Fatalf("NewConvocatoria() error = %v", err)
	}
	convocatorias := fixedConvocatoriaRepository{record: ports.ConvocatoriaRecord{
		Convocatoria: convocatoria,
		RuleSet:      testBaremoRuleSet(t),
	}}
	solicitudes := &cancelOnSolicitudReadRepository{cancel: cancel}
	useCase, err := NewProcedureUseCase(convocatorias, solicitudes)
	if err != nil {
		t.Fatalf("NewProcedureUseCase() error = %v", err)
	}

	_, err = useCase.EnsureSolicitud(ctx, RegistrarSolicitudCommand{
		ID: "sol-1", ConvocatoriaID: "conv-1", CandidateID: "cand-1",
		Merits: []domain.Merit{validBaremoMerit()},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("EnsureSolicitud() error = %v, want %v", err, context.Canceled)
	}
	if solicitudes.saveCalls != 0 {
		t.Fatalf("solicitud saves = %d, want 0", solicitudes.saveCalls)
	}
}

func TestRegisterCandidateDocumentRechecksCancellationBeforeAudit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	documents := &cancelOnDocumentSaveRepository{cancel: cancel}
	audit := &countingAdministrativeAuditTrail{}
	useCase, err := NewAdministrativeFlowUseCase(
		documents,
		contextProbeClaimRepository{accesses: new(int)},
		contextProbeNotificationRepository{accesses: new(int)},
		audit,
	)
	if err != nil {
		t.Fatalf("NewAdministrativeFlowUseCase() error = %v", err)
	}
	at := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)

	_, _, err = useCase.RegisterCandidateDocument(ctx, RegisterCandidateDocumentCommand{
		ID: "doc-1", CandidateID: "cand-1", SolicitudID: "sol-1", ProcedureID: "proc-1",
		Purpose: domain.DocumentPurposeAlegacion, Evidence: mustUsecaseDocumentEvidence(t, domain.AVStatusClean, at),
		RegisteredBy: "cand-1", RegisteredAt: at,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("RegisterCandidateDocument() error = %v, want %v", err, context.Canceled)
	}
	if documents.saveCalls != 1 {
		t.Fatalf("document saves = %d, want 1", documents.saveCalls)
	}
	if audit.appendCalls != 0 {
		t.Fatalf("audit appends = %d, want 0", audit.appendCalls)
	}
}

type contextProbeConvocatoriaRepository struct{ accesses *int }

func (r contextProbeConvocatoriaRepository) Save(context.Context, ports.ConvocatoriaRecord) error {
	(*r.accesses)++
	return errUnexpectedContextPortAccess
}
func (r contextProbeConvocatoriaRepository) GetByID(context.Context, string) (ports.ConvocatoriaRecord, error) {
	(*r.accesses)++
	return ports.ConvocatoriaRecord{}, errUnexpectedContextPortAccess
}

type contextProbeSolicitudRepository struct{ accesses *int }

func (r contextProbeSolicitudRepository) Save(context.Context, ports.SolicitudRecord) error {
	(*r.accesses)++
	return errUnexpectedContextPortAccess
}
func (r contextProbeSolicitudRepository) GetByID(context.Context, string) (ports.SolicitudRecord, error) {
	(*r.accesses)++
	return ports.SolicitudRecord{}, errUnexpectedContextPortAccess
}
func (r contextProbeSolicitudRepository) ListByConvocatoria(context.Context, string) ([]ports.SolicitudRecord, error) {
	(*r.accesses)++
	return nil, errUnexpectedContextPortAccess
}

type contextProbeMeritRepository struct{ accesses *int }

func (r contextProbeMeritRepository) Save(context.Context, string, domain.Merit) error {
	(*r.accesses)++
	return errUnexpectedContextPortAccess
}
func (r contextProbeMeritRepository) ListByCandidate(context.Context, string) ([]domain.Merit, error) {
	(*r.accesses)++
	return nil, errUnexpectedContextPortAccess
}

type contextProbeBaremoResultRepository struct{ accesses *int }

func (r contextProbeBaremoResultRepository) Save(context.Context, string, domain.BaremoResult) error {
	(*r.accesses)++
	return errUnexpectedContextPortAccess
}

type contextProbeDocumentRepository struct{ accesses *int }

func (r contextProbeDocumentRepository) Save(context.Context, domain.CandidateDocument) error {
	(*r.accesses)++
	return errUnexpectedContextPortAccess
}
func (r contextProbeDocumentRepository) GetByID(context.Context, string) (domain.CandidateDocument, error) {
	(*r.accesses)++
	return domain.CandidateDocument{}, errUnexpectedContextPortAccess
}
func (r contextProbeDocumentRepository) ListByCandidate(context.Context, string) ([]domain.CandidateDocument, error) {
	(*r.accesses)++
	return nil, errUnexpectedContextPortAccess
}

type contextProbeClaimRepository struct{ accesses *int }

func (r contextProbeClaimRepository) Save(context.Context, domain.Claim) error {
	(*r.accesses)++
	return errUnexpectedContextPortAccess
}
func (r contextProbeClaimRepository) GetByID(context.Context, string) (domain.Claim, error) {
	(*r.accesses)++
	return domain.Claim{}, errUnexpectedContextPortAccess
}
func (r contextProbeClaimRepository) ListBySolicitud(context.Context, string) ([]domain.Claim, error) {
	(*r.accesses)++
	return nil, errUnexpectedContextPortAccess
}

type contextProbeNotificationRepository struct{ accesses *int }

func (r contextProbeNotificationRepository) Save(context.Context, domain.Notification) error {
	(*r.accesses)++
	return errUnexpectedContextPortAccess
}
func (r contextProbeNotificationRepository) GetByID(context.Context, string) (domain.Notification, error) {
	(*r.accesses)++
	return domain.Notification{}, errUnexpectedContextPortAccess
}
func (r contextProbeNotificationRepository) ListByCandidate(context.Context, string) ([]domain.Notification, error) {
	(*r.accesses)++
	return nil, errUnexpectedContextPortAccess
}

type contextProbeAuditTrail struct{ accesses *int }

func (r contextProbeAuditTrail) Append(context.Context, string, domain.AuditEnvelope) (domain.AuditEntry, error) {
	(*r.accesses)++
	return domain.AuditEntry{}, errUnexpectedContextPortAccess
}
func (r contextProbeAuditTrail) ListByScope(context.Context, string) ([]domain.AuditEntry, error) {
	(*r.accesses)++
	return nil, errUnexpectedContextPortAccess
}

type cancelAfterFirstMeritSaveRepository struct {
	cancel    context.CancelFunc
	merits    []domain.Merit
	saveCalls int
}

func (r *cancelAfterFirstMeritSaveRepository) Save(context.Context, string, domain.Merit) error {
	r.saveCalls++
	if r.saveCalls == 1 {
		r.cancel()
	}
	return nil
}
func (r *cancelAfterFirstMeritSaveRepository) ListByCandidate(context.Context, string) ([]domain.Merit, error) {
	return append([]domain.Merit(nil), r.merits...), nil
}

type countingBaremoResultRepository struct{ saveCalls int }

func (r *countingBaremoResultRepository) Save(context.Context, string, domain.BaremoResult) error {
	r.saveCalls++
	return nil
}

type fixedConvocatoriaRepository struct{ record ports.ConvocatoriaRecord }

func (r fixedConvocatoriaRepository) Save(context.Context, ports.ConvocatoriaRecord) error {
	return nil
}
func (r fixedConvocatoriaRepository) GetByID(context.Context, string) (ports.ConvocatoriaRecord, error) {
	return r.record, nil
}

type cancelOnSolicitudReadRepository struct {
	cancel    context.CancelFunc
	saveCalls int
}

func (r *cancelOnSolicitudReadRepository) Save(context.Context, ports.SolicitudRecord) error {
	r.saveCalls++
	return nil
}
func (r *cancelOnSolicitudReadRepository) GetByID(context.Context, string) (ports.SolicitudRecord, error) {
	r.cancel()
	return ports.SolicitudRecord{}, ports.ErrSolicitudNotFound
}
func (r *cancelOnSolicitudReadRepository) ListByConvocatoria(context.Context, string) ([]ports.SolicitudRecord, error) {
	return nil, nil
}

type cancelOnDocumentSaveRepository struct {
	cancel    context.CancelFunc
	saveCalls int
}

func (r *cancelOnDocumentSaveRepository) Save(context.Context, domain.CandidateDocument) error {
	r.saveCalls++
	r.cancel()
	return nil
}
func (r *cancelOnDocumentSaveRepository) GetByID(context.Context, string) (domain.CandidateDocument, error) {
	return domain.CandidateDocument{}, ports.ErrCandidateDocumentNotFound
}
func (r *cancelOnDocumentSaveRepository) ListByCandidate(context.Context, string) ([]domain.CandidateDocument, error) {
	return nil, nil
}

type countingAdministrativeAuditTrail struct{ appendCalls int }

func (r *countingAdministrativeAuditTrail) Append(context.Context, string, domain.AuditEnvelope) (domain.AuditEntry, error) {
	r.appendCalls++
	return domain.AuditEntry{}, nil
}
func (r *countingAdministrativeAuditTrail) ListByScope(context.Context, string) ([]domain.AuditEntry, error) {
	return nil, nil
}
