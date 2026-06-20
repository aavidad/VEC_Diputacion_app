package application

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/memory"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestBuildMenuFiltersByPermission(t *testing.T) {
	store := memory.NewStore()
	service, err := NewService(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	manifest := domain.ModuleManifest{
		ID:      "vec.module.demo",
		NameKey: "ui.module.demo",
		Menu: []domain.MenuEntry{
			{ID: "visible", ModuleID: "vec.module.demo", LabelKey: "ui.visible", Path: "/visible", Order: 2, RequiredPermissions: []string{"demo.read"}},
			{ID: "hidden", ModuleID: "vec.module.demo", LabelKey: "ui.hidden", Path: "/hidden", Order: 1, RequiredPermissions: []string{"demo.admin"}},
		},
	}
	if err := service.RegisterModule(context.Background(), manifest); err != nil {
		t.Fatalf("RegisterModule() error = %v", err)
	}
	menu, err := service.BuildMenu(context.Background(), domain.Principal{
		ID: "u1", AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh,
		Permissions: []string{"demo.read"},
	})
	if err != nil {
		t.Fatalf("BuildMenu() error = %v", err)
	}
	if len(menu) != 1 || menu[0].ID != "visible" {
		t.Fatalf("menu = %#v, want only visible", menu)
	}
}

func TestRecordAuditChainsEntries(t *testing.T) {
	store := memory.NewStore()
	service, err := NewService(store, store, store)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	principal := domain.Principal{ID: "staff", AuthMethod: domain.AuthMethodDemo, AuthAssurance: domain.AuthAssuranceHigh}
	first, err := service.RecordAudit(context.Background(), AuditCommand{Principal: principal, Action: "create", ModuleID: "vec.module.demo", Result: "ok"})
	if err != nil {
		t.Fatalf("RecordAudit(first) error = %v", err)
	}
	second, err := service.RecordAudit(context.Background(), AuditCommand{Principal: principal, Action: "update", ModuleID: "vec.module.demo", Result: "ok"})
	if err != nil {
		t.Fatalf("RecordAudit(second) error = %v", err)
	}
	if first.Seq != 1 || second.Seq != 2 || second.PrevSignature != first.Signature || second.Signature == "" {
		t.Fatalf("audit chain first=%+v second=%+v", first, second)
	}
}
