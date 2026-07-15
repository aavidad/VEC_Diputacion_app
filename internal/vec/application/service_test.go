package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestBuildMenuFiltersByPermission(t *testing.T) {
	store := memory.NewStore()
	service, internal, err := NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	manifest := domain.ModuleManifest{
		ID:      "vec.module.demo",
		NameKey: "ui.module.demo",
		Permissions: []domain.Permission{
			{Key: "demo.read", LabelKey: "ui.permission.demo.read"},
			{Key: "demo.admin", LabelKey: "ui.permission.demo.admin"},
		},
		Menu: []domain.MenuEntry{
			{ID: "visible", ModuleID: "vec.module.demo", LabelKey: "ui.visible", Path: "/visible", Order: 2, RequiredPermissions: []string{"demo.read"}},
			{ID: "hidden", ModuleID: "vec.module.demo", LabelKey: "ui.hidden", Path: "/hidden", Order: 1, RequiredPermissions: []string{"demo.admin"}},
		},
	}
	if err := internal.RegisterModule(context.Background(), manifest); err != nil {
		t.Fatalf("RegisterModule() error = %v", err)
	}
	menu, err := service.BuildMenu(context.Background(), domain.Principal{
		ID: "u1", AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh,
		Permissions: []string{menuReadPermission, "demo.read"},
	})
	if err != nil {
		t.Fatalf("BuildMenu() error = %v", err)
	}
	if len(menu) != 1 || menu[0].ID != "visible" {
		t.Fatalf("menu = %#v, want only visible", menu)
	}
}

func TestBuildMenuDeniegaSiElRepositorioDevuelveUnManifestInseguro(t *testing.T) {
	store := memory.NewStore()
	service, err := NewService(repositorioManifestInseguro{}, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	menu, err := service.BuildMenu(context.Background(), domain.Principal{
		ID: "u1", AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh,
		Permissions: []string{menuReadPermission, "demo.read"},
	})
	if err == nil || menu != nil {
		t.Fatalf("BuildMenu() menu=%#v error=%v; un manifiesto alterado debe denegar todo", menu, err)
	}
}

func TestLecturasDeServicioExigenPermisoPositivoExacto(t *testing.T) {
	store := memory.NewStore()
	service, err := NewService(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	for _, permisos := range [][]string{nil, {"vec.audit.read"}, {"vec.modules.*"}} {
		principal := domain.Principal{
			ID: "u1", AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh,
			Permissions: permisos,
		}
		if modules, err := service.Modules(context.Background(), principal); !errors.Is(err, domain.ErrPermissionDenied) || modules != nil {
			t.Fatalf("Modules(permisos=%#v) = (%#v, %v), debe denegar", permisos, modules, err)
		}
	}
	for _, permisos := range [][]string{nil, {"vec.modules.read"}, {"vec.menu.*"}} {
		principal := domain.Principal{
			ID: "u1", AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh,
			Permissions: permisos,
		}
		if menu, err := service.BuildMenu(context.Background(), principal); !errors.Is(err, domain.ErrPermissionDenied) || menu != nil {
			t.Fatalf("BuildMenu(permisos=%#v) = (%#v, %v), debe denegar", permisos, menu, err)
		}
	}
}

func TestLecturasDeServicioNoExponenElCatalogoMutable(t *testing.T) {
	store := memory.NewStore()
	service, internal, err := NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatalf("NewServiceWithInternalOperations() error = %v", err)
	}
	manifest := domain.ModuleManifest{
		ID:      "vec.module.demo",
		NameKey: "ui.module.demo",
		Permissions: []domain.Permission{
			{Key: "demo.read", LabelKey: "ui.permission.demo.read"},
		},
		Menu: []domain.MenuEntry{{
			ID: "home", ModuleID: "vec.module.demo", LabelKey: "ui.home", Path: "/demo",
			RequiredPermissions: []string{"demo.read"},
		}},
	}
	if err := internal.RegisterModule(context.Background(), manifest); err != nil {
		t.Fatalf("RegisterModule() error = %v", err)
	}
	principal := domain.Principal{
		ID: "u1", AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh,
		Permissions: []string{moduleReadPermission, menuReadPermission, "demo.read"},
	}
	modules, err := service.Modules(context.Background(), principal)
	if err != nil || len(modules) != 1 {
		t.Fatalf("Modules() = (%#v, %v)", modules, err)
	}
	menu, err := service.BuildMenu(context.Background(), principal)
	if err != nil || len(menu) != 1 {
		t.Fatalf("BuildMenu() = (%#v, %v)", menu, err)
	}
	modules[0].Permissions[0].Key = "demo.admin"
	modules[0].Menu[0].RequiredPermissions[0] = "demo.admin"
	menu[0].RequiredPermissions[0] = "demo.admin"

	modules, err = service.Modules(context.Background(), principal)
	if err != nil || modules[0].Permissions[0].Key != "demo.read" ||
		modules[0].Menu[0].RequiredPermissions[0] != "demo.read" {
		t.Fatalf("la respuesta Modules altero el catalogo: (%#v, %v)", modules, err)
	}
	menu, err = service.BuildMenu(context.Background(), principal)
	if err != nil || len(menu) != 1 || menu[0].RequiredPermissions[0] != "demo.read" {
		t.Fatalf("la respuesta BuildMenu altero el catalogo: (%#v, %v)", menu, err)
	}
}

type repositorioManifestInseguro struct{}

func (repositorioManifestInseguro) SaveModule(context.Context, domain.ModuleManifest) error {
	return nil
}

func (repositorioManifestInseguro) ListModules(context.Context) ([]domain.ModuleManifest, error) {
	return []domain.ModuleManifest{{
		ID:          "vec.module.demo",
		NameKey:     "ui.module.demo",
		Permissions: []domain.Permission{{Key: "demo.read", LabelKey: "ui.permission.demo.read"}},
		Menu: []domain.MenuEntry{{
			ID: "inseguro", ModuleID: "vec.module.demo", LabelKey: "ui.inseguro", Path: "/inseguro",
			RequiredPermissions: nil,
		}},
	}}, nil
}

func TestRecordAuditChainsEntries(t *testing.T) {
	store := memory.NewStore()
	_, internal, err := NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	principal := domain.Principal{ID: "staff", Permissions: []string{"demo.write"}, AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh}
	firstCommand, err := NewAuthorizedAuditCommand(AuditCommand{Principal: principal, Action: "create", ModuleID: "vec.module.demo", Result: "ok"}, "demo.write", "")
	if err != nil {
		t.Fatalf("NewAuthorizedAuditCommand(first) error = %v", err)
	}
	firstReceipt, err := internal.RecordAudit(context.Background(), firstCommand)
	if err != nil {
		t.Fatalf("RecordAudit(first) error = %v", err)
	}
	secondCommand, err := NewAuthorizedAuditCommand(AuditCommand{Principal: principal, Action: "update", ModuleID: "vec.module.demo", Result: "ok"}, "demo.write", "")
	if err != nil {
		t.Fatalf("NewAuthorizedAuditCommand(second) error = %v", err)
	}
	secondReceipt, err := internal.RecordAudit(context.Background(), secondCommand)
	if err != nil {
		t.Fatalf("RecordAudit(second) error = %v", err)
	}
	first, second := firstReceipt.Entry(), secondReceipt.Entry()
	if first.Seq != 1 || second.Seq != 2 || second.PrevSignature != first.Signature || second.Signature == "" {
		t.Fatalf("audit chain first=%+v second=%+v", first, second)
	}
}

func TestAuditNoInterpretaAmbitoVacioComoAccesoGlobal(t *testing.T) {
	store := memory.NewStore()
	_, internal, err := NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	principal := domain.Principal{ID: "auditor", Permissions: []string{auditReadPermission}, AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh}
	for _, referencia := range []string{
		"", " ", " expediente:uno", "*", "expediente:*", "expediente:\nuno",
		strings.Repeat("x", 513), string([]byte{'e', 'x', 'p', 0xff}),
	} {
		query, err := NewAuditQuery(principal, referencia)
		if !errors.Is(err, domain.ErrPermissionDenied) || !reflect.ValueOf(query).IsZero() {
			t.Fatalf("NewAuditQuery(%q) = (%#v, %v), debe denegar", referencia, query, err)
		}
	}
	if auditoria, err := internal.Audit(context.Background(), AuditQuery{}); !errors.Is(err, domain.ErrPermissionDenied) || auditoria != nil {
		t.Fatalf("Audit(query cero) = (%#v, %v), debe denegar", auditoria, err)
	}
}

func TestServiceNoExponeOperacionesInternas(t *testing.T) {
	tipo := reflect.TypeOf((*Service)(nil))
	permitidos := map[string]struct{}{"BuildMenu": {}, "Modules": {}}
	if tipo.NumMethod() != len(permitidos) {
		t.Fatalf("*Service expone %d metodos; lista positiva = %#v", tipo.NumMethod(), permitidos)
	}
	for indice := 0; indice < tipo.NumMethod(); indice++ {
		metodo := tipo.Method(indice).Name
		if _, permitido := permitidos[metodo]; !permitido {
			t.Fatalf("*Service expone el metodo no revisado %s", metodo)
		}
	}
}

type repositorioCapturaManifest struct {
	guardado domain.ModuleManifest
}

func (r *repositorioCapturaManifest) SaveModule(_ context.Context, manifest domain.ModuleManifest) error {
	r.guardado = manifest
	return nil
}

func (*repositorioCapturaManifest) ListModules(context.Context) ([]domain.ModuleManifest, error) {
	return nil, nil
}

func TestRegisterModuleNoEntregaMemoriaMutableAlConector(t *testing.T) {
	repositorio := &repositorioCapturaManifest{}
	infraestructura := memory.NewStore()
	_, internal, err := NewServiceWithInternalOperations(repositorio, infraestructura, infraestructura)
	if err != nil {
		t.Fatalf("NewServiceWithInternalOperations() error = %v", err)
	}
	manifest := domain.ModuleManifest{
		ID:      "vec.module.demo",
		NameKey: "ui.module.demo",
		Permissions: []domain.Permission{
			{Key: "demo.read", LabelKey: "ui.permission.demo.read"},
		},
		Menu: []domain.MenuEntry{{
			ID: "home", ModuleID: "vec.module.demo", LabelKey: "ui.home", Path: "/demo",
			RequiredPermissions: []string{"demo.read"},
		}},
	}
	if err := internal.RegisterModule(context.Background(), manifest); err != nil {
		t.Fatalf("RegisterModule() error = %v", err)
	}
	manifest.Permissions[0].Key = "demo.admin"
	manifest.Menu[0].RequiredPermissions[0] = "demo.admin"
	if repositorio.guardado.Permissions[0].Key != "demo.read" ||
		repositorio.guardado.Menu[0].RequiredPermissions[0] != "demo.read" {
		t.Fatalf("el conector retuvo memoria del llamador: %#v", repositorio.guardado)
	}
}

func TestOperacionesInternasExigenCapacidadPositivaExacta(t *testing.T) {
	store := memory.NewStore()
	_, internal, err := NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatalf("NewServiceWithInternalOperations() error = %v", err)
	}
	principal := domain.Principal{
		ID: "staff", Permissions: []string{"demo.write"},
		AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh,
	}
	command := AuditCommand{Principal: principal, Action: "create", ModuleID: "vec.module.demo", SubjectRef: "demo:1", Result: "ok"}
	for _, permiso := range []string{"", "demo.read", "demo.*", " demo.write"} {
		if authorized, err := NewAuthorizedAuditCommand(command, permiso, "demo.created"); !errors.Is(err, domain.ErrPermissionDenied) || !reflect.ValueOf(authorized).IsZero() {
			t.Fatalf("permiso %q produjo capacidad (%#v, %v)", permiso, authorized, err)
		}
	}
	if receipt, err := internal.RecordAudit(context.Background(), AuthorizedAuditCommand{}); !errors.Is(err, domain.ErrPermissionDenied) || !reflect.ValueOf(receipt).IsZero() {
		t.Fatalf("comando cero produjo recibo (%#v, %v)", receipt, err)
	}
	if audit, err := store.ListAudit(context.Background(), "demo:1"); err != nil || len(audit) != 0 {
		t.Fatalf("la denegacion produjo auditoria: (%#v, %v)", audit, err)
	}
}

func TestEventoExigeReciboOpacoYVinculoExacto(t *testing.T) {
	store := memory.NewStore()
	_, internal, err := NewServiceWithInternalOperations(store, store, store)
	if err != nil {
		t.Fatalf("NewServiceWithInternalOperations() error = %v", err)
	}
	principal := domain.Principal{
		ID: "staff", Permissions: []string{"demo.write"},
		AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh,
	}
	authorized, err := NewAuthorizedAuditCommand(AuditCommand{
		Principal: principal, Action: "create", ModuleID: "vec.module.demo", SubjectRef: "demo:1", Result: "ok",
	}, "demo.write", "demo.created")
	if err != nil {
		t.Fatalf("NewAuthorizedAuditCommand() error = %v", err)
	}
	receipt, err := internal.RecordAudit(context.Background(), authorized)
	if err != nil {
		t.Fatalf("RecordAudit() error = %v", err)
	}
	entry := receipt.Entry()
	valid := domain.Event{
		Type: "demo.created", ModuleID: entry.ModuleID, SubjectRef: entry.SubjectRef,
		ActorID: entry.ActorID, Payload: map[string]string{"audit_id": entry.ID},
	}
	for name, mutate := range map[string]func(*domain.Event){
		"tipo":      func(event *domain.Event) { event.Type = "demo.other" },
		"actor":     func(event *domain.Event) { event.ActorID = "otro" },
		"sujeto":    func(event *domain.Event) { event.SubjectRef = "demo:2" },
		"auditoria": func(event *domain.Event) { event.Payload["audit_id"] = "audit-forjada" },
	} {
		t.Run(name, func(t *testing.T) {
			event := valid
			event.Payload = map[string]string{"audit_id": valid.Payload["audit_id"]}
			mutate(&event)
			if err := internal.PublishEvent(context.Background(), receipt, event); !errors.Is(err, domain.ErrPermissionDenied) {
				t.Fatalf("PublishEvent() error = %v, debe denegar", err)
			}
		})
	}
	if err := internal.PublishEvent(context.Background(), AuditReceipt{}, valid); !errors.Is(err, domain.ErrPermissionDenied) {
		t.Fatalf("PublishEvent(recibo cero) error = %v, debe denegar", err)
	}
	if events, err := store.ListEvents(context.Background(), []string{"demo.created", "demo.other"}); err != nil || len(events) != 0 {
		t.Fatalf("las denegaciones publicaron eventos: (%#v, %v)", events, err)
	}
}

func TestConsultaAuditoriaExigePermisoExacto(t *testing.T) {
	for _, permisos := range [][]string{nil, {"vec.modules.read"}, {"vec.audit.*"}} {
		principal := domain.Principal{
			ID: "auditor", Permissions: permisos,
			AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh,
		}
		query, err := NewAuditQuery(principal, "expediente:uno")
		if !errors.Is(err, domain.ErrPermissionDenied) || !reflect.ValueOf(query).IsZero() {
			t.Fatalf("permisos %#v produjeron consulta (%#v, %v)", permisos, query, err)
		}
	}
}
