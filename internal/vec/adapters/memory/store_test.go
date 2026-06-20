package memory

import (
	"context"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestStorePersistsModulesAuditAndEvents(t *testing.T) {
	store := NewStore()
	ctx := context.Background()
	if err := store.SaveModule(ctx, domain.ModuleManifest{
		ID:      "vec.module.demo",
		NameKey: "ui.module.demo",
		Menu:    []domain.MenuEntry{{ID: "home", ModuleID: "vec.module.demo", LabelKey: "ui.home", Path: "/demo"}},
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
