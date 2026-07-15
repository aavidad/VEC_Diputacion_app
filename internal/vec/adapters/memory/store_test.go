package memory

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

type contextoCanceladoEnSegundaComprobacion struct {
	context.Context
	cancelar       context.CancelFunc
	comprobaciones int
}

func nuevoContextoCanceladoEnSegundaComprobacion() *contextoCanceladoEnSegundaComprobacion {
	base, cancelar := context.WithCancel(context.Background())
	return &contextoCanceladoEnSegundaComprobacion{Context: base, cancelar: cancelar}
}

func (c *contextoCanceladoEnSegundaComprobacion) Err() error {
	c.comprobaciones++
	if c.comprobaciones == 2 {
		c.cancelar()
	}
	return c.Context.Err()
}

func TestStorePersistsModulesAuditAndEvents(t *testing.T) {
	store := NewStore()
	ctx := context.Background()
	if err := store.SaveModule(ctx, domain.ModuleManifest{
		ID:          "vec.module.demo",
		NameKey:     "ui.module.demo",
		Permissions: []domain.Permission{{Key: "demo.read", LabelKey: "ui.permission.demo.read"}},
		Menu: []domain.MenuEntry{{
			ID: "home", ModuleID: "vec.module.demo", LabelKey: "ui.home", Path: "/demo",
			RequiredPermissions: []string{"demo.read"},
		}},
	}); err != nil {
		t.Fatalf("SaveModule() error = %v", err)
	}
	modules, err := store.ListModules(ctx)
	if err != nil || len(modules) != 1 {
		t.Fatalf("ListModules() modules=%v err=%v", modules, err)
	}
	audit, err := store.AppendAudit(ctx, domain.AuditEntry{ActorID: "staff", Action: "demo", ModuleID: "vec.module.demo", Result: "ok"})
	if err != nil || audit.Signature == "" || audit.Seq != 1 {
		t.Fatalf("AppendAudit() audit=%+v err=%v", audit, err)
	}
	if err := store.PublishEvent(ctx, domain.Event{Type: "demo.created", ModuleID: "vec.module.demo"}); err != nil {
		t.Fatalf("PublishEvent() error = %v", err)
	}
	events, err := store.ListEvents(ctx, []string{"demo.created"})
	if err != nil || len(events) != 1 {
		t.Fatalf("ListEvents() events=%v err=%v", events, err)
	}
}

func TestStoreRespetaContextoCanceladoSinEfectos(t *testing.T) {
	store := NewStore()
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	manifest := domain.ModuleManifest{
		ID:          "vec.module.cancelado",
		NameKey:     "ui.module.cancelado",
		Permissions: []domain.Permission{{Key: "cancelado.read", LabelKey: "ui.permission.cancelado.read"}},
		Menu: []domain.MenuEntry{{
			ID: "home", ModuleID: "vec.module.cancelado", LabelKey: "ui.home", Path: "/cancelado",
			RequiredPermissions: []string{"cancelado.read"},
		}},
	}

	if err := store.SaveModule(ctx, manifest); !errors.Is(err, context.Canceled) {
		t.Fatalf("SaveModule() error = %v", err)
	}
	if _, err := store.AppendAudit(ctx, domain.AuditEntry{SubjectRef: "cancelado"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("AppendAudit() error = %v", err)
	}
	if err := store.PublishEvent(ctx, domain.Event{Type: "cancelado"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("PublishEvent() error = %v", err)
	}
	if _, err := store.ListModules(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListModules() error = %v", err)
	}
	if _, err := store.ListAudit(ctx, "cancelado"); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListAudit() error = %v", err)
	}
	if _, err := store.ListEvents(ctx, []string{"cancelado"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("ListEvents() error = %v", err)
	}

	store.mu.RLock()
	defer store.mu.RUnlock()
	if len(store.modules) != 0 || len(store.audit) != 0 || len(store.events) != 0 {
		t.Fatalf("el contexto cancelado produjo efectos: modulos=%d auditoria=%d eventos=%d",
			len(store.modules), len(store.audit), len(store.events))
	}
}

func TestStoreRevalidaContextoJustoAntesDeCadaEfecto(t *testing.T) {
	manifest := domain.ModuleManifest{
		ID:          "vec.module.cancelado",
		NameKey:     "ui.module.cancelado",
		Permissions: []domain.Permission{{Key: "cancelado.read", LabelKey: "ui.permission.cancelado.read"}},
		Menu: []domain.MenuEntry{{
			ID: "home", ModuleID: "vec.module.cancelado", LabelKey: "ui.home", Path: "/cancelado",
			RequiredPermissions: []string{"cancelado.read"},
		}},
	}
	casos := []struct {
		nombre   string
		ejecutar func(*Store, context.Context) error
	}{
		{nombre: "modulo", ejecutar: func(store *Store, ctx context.Context) error {
			return store.SaveModule(ctx, manifest)
		}},
		{nombre: "auditoria", ejecutar: func(store *Store, ctx context.Context) error {
			_, err := store.AppendAudit(ctx, domain.AuditEntry{SubjectRef: "cancelado"})
			return err
		}},
		{nombre: "evento", ejecutar: func(store *Store, ctx context.Context) error {
			return store.PublishEvent(ctx, domain.Event{Type: "cancelado"})
		}},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			store := NewStore()
			ctx := nuevoContextoCanceladoEnSegundaComprobacion()
			if err := caso.ejecutar(store, ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v", err)
			}
			store.mu.RLock()
			defer store.mu.RUnlock()
			if len(store.modules) != 0 || len(store.audit) != 0 || len(store.events) != 0 {
				t.Fatalf("el efecto se confirmo tras cancelar: modulos=%d auditoria=%d eventos=%d",
					len(store.modules), len(store.audit), len(store.events))
			}
		})
	}
}

func TestStoreNoConvierteFiltrosVaciosOComodinesEnConsultaGlobal(t *testing.T) {
	store := NewStore()
	ctx := context.Background()
	for _, referencia := range []string{"", " ", " expediente:uno"} {
		registros, err := store.ListAudit(ctx, referencia)
		if !errors.Is(err, domain.ErrPermissionDenied) || registros != nil {
			t.Fatalf("ListAudit(%q) = (%#v, %v), debe denegar", referencia, registros, err)
		}
	}
	for _, tipos := range [][]string{nil, {}, {"*"}, {"demo.created", "demo.created"}, {" demo.created"}} {
		eventos, err := store.ListEvents(ctx, tipos)
		if !errors.Is(err, domain.ErrPermissionDenied) || eventos != nil {
			t.Fatalf("ListEvents(%#v) = (%#v, %v), debe denegar", tipos, eventos, err)
		}
	}
}

func TestStoreAislaLosManifiestosDeMutacionesExternas(t *testing.T) {
	store := NewStore()
	ctx := context.Background()
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
	if err := store.SaveModule(ctx, manifest); err != nil {
		t.Fatalf("SaveModule() error = %v", err)
	}

	manifest.Permissions[0].Key = "demo.admin"
	manifest.Menu[0].ID = "alterado"
	manifest.Menu[0].RequiredPermissions[0] = "demo.admin"
	primera, err := store.ListModules(ctx)
	if err != nil || len(primera) != 1 {
		t.Fatalf("ListModules() = (%#v, %v)", primera, err)
	}
	if primera[0].Permissions[0].Key != "demo.read" || primera[0].Menu[0].ID != "home" ||
		primera[0].Menu[0].RequiredPermissions[0] != "demo.read" {
		t.Fatalf("la mutacion del argumento altero el almacen: %#v", primera[0])
	}

	primera[0].Permissions[0].Key = "demo.admin"
	primera[0].Menu[0].ID = "alterado"
	primera[0].Menu[0].RequiredPermissions[0] = "demo.admin"
	segunda, err := store.ListModules(ctx)
	if err != nil || len(segunda) != 1 {
		t.Fatalf("segunda ListModules() = (%#v, %v)", segunda, err)
	}
	if segunda[0].Permissions[0].Key != "demo.read" || segunda[0].Menu[0].ID != "home" ||
		segunda[0].Menu[0].RequiredPermissions[0] != "demo.read" {
		t.Fatalf("la mutacion de la respuesta altero el almacen: %#v", segunda[0])
	}
}
