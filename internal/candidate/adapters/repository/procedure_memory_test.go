package repository

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"vec-diputacion-granada/internal/candidate/domain"
	"vec-diputacion-granada/internal/candidate/ports"
)

func TestProcedureMemoryRepositoriesPersistAndListDeterministically(t *testing.T) {
	ctx := context.Background()
	convocatorias, solicitudes := NewProcedureRepositories()
	if err := convocatorias.Save(ctx, ports.ConvocatoriaRecord{
		Convocatoria: procedureConvocatoria(t),
		RuleSet:      procedureRuleSet(t),
	}); err != nil {
		t.Fatalf("Save(convocatoria) error = %v", err)
	}
	for _, solicitud := range []ports.SolicitudRecord{
		procedureSolicitud("sol-2", "cand-b"),
		procedureSolicitud("sol-1", "cand-a"),
	} {
		if err := solicitudes.Save(ctx, solicitud); err != nil {
			t.Fatalf("Save(%s) error = %v", solicitud.ID, err)
		}
	}

	gotConvocatoria, err := convocatorias.GetByID(ctx, "conv-1")
	if err != nil {
		t.Fatalf("GetByID(conv-1) error = %v", err)
	}
	if gotConvocatoria.Convocatoria.Version != "v1" {
		t.Fatalf("convocatoria version = %q, want v1", gotConvocatoria.Convocatoria.Version)
	}

	gotSolicitudes, err := solicitudes.ListByConvocatoria(ctx, "conv-1")
	if err != nil {
		t.Fatalf("ListByConvocatoria() error = %v", err)
	}
	if got := solicitudIDs(gotSolicitudes); !reflect.DeepEqual(got, []string{"sol-1", "sol-2"}) {
		t.Fatalf("solicitud IDs = %#v, want sorted", got)
	}
}

func TestProcedureMemorySolicitudRepositoryIsConcurrentSafeAndDoesNotLeakResults(t *testing.T) {
	ctx := context.Background()
	_, solicitudes := NewProcedureRepositories()
	const total = 32
	var wg sync.WaitGroup
	for i := 0; i < total; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			solicitud := procedureSolicitud(fmt.Sprintf("sol-%02d", i), fmt.Sprintf("cand-%02d", i))
			if err := solicitudes.Save(ctx, solicitud); err != nil {
				t.Errorf("Save(%s) error = %v", solicitud.ID, err)
			}
		}(i)
	}
	wg.Wait()

	gotSolicitudes, err := solicitudes.ListByConvocatoria(ctx, "conv-1")
	if err != nil {
		t.Fatalf("ListByConvocatoria() error = %v", err)
	}
	if len(gotSolicitudes) != total {
		t.Fatalf("solicitudes = %d, want %d", len(gotSolicitudes), total)
	}

	gotSolicitudes[0].Result.SectionPoints[domain.BaremoSectionExperiencia] = 99
	gotSolicitudes[0].Result.Details[0].AppliedPoints = 99
	got, err := solicitudes.GetByID(ctx, gotSolicitudes[0].ID)
	if err != nil {
		t.Fatalf("GetByID(%s) error = %v", gotSolicitudes[0].ID, err)
	}
	if got.Result.SectionPoints[domain.BaremoSectionExperiencia] == 99 || got.Result.Details[0].AppliedPoints == 99 {
		t.Fatalf("repository leaked mutable baremo result")
	}
}

func TestProcedureMemorySolicitudRepositoryMovesConvocatoriaIndex(t *testing.T) {
	ctx := context.Background()
	_, solicitudes := NewProcedureRepositories()
	solicitud := procedureSolicitud("sol-1", "cand-a")
	if err := solicitudes.Save(ctx, solicitud); err != nil {
		t.Fatalf("Save(initial) error = %v", err)
	}
	solicitud.ConvocatoriaID = "conv-2"
	if err := solicitudes.Save(ctx, solicitud); err != nil {
		t.Fatalf("Save(moved) error = %v", err)
	}
	oldList, err := solicitudes.ListByConvocatoria(ctx, "conv-1")
	if err != nil {
		t.Fatalf("ListByConvocatoria(old) error = %v", err)
	}
	newList, err := solicitudes.ListByConvocatoria(ctx, "conv-2")
	if err != nil {
		t.Fatalf("ListByConvocatoria(new) error = %v", err)
	}
	if len(oldList) != 0 || len(newList) != 1 {
		t.Fatalf("moved index old=%d new=%d", len(oldList), len(newList))
	}
}

func procedureConvocatoria(t *testing.T) domain.Convocatoria {
	t.Helper()
	convocatoria, err := domain.NewConvocatoria("conv-1", "v1")
	if err != nil {
		t.Fatalf("NewConvocatoria() error = %v", err)
	}
	return convocatoria
}

func procedureRuleSet(t *testing.T) domain.BaremoRuleSet {
	t.Helper()
	ruleSet, err := domain.NewBaremoRuleSet(domain.BaremoRuleSetConfig{
		ConvocatoriaID: "conv-1",
		Version:        "v1",
		MeritRules: []domain.BaremoMeritRule{
			{MeritType: domain.MeritTypeExperienciaMismaCategoria, Section: domain.BaremoSectionExperiencia, Unit: domain.BaremoUnitMeses, PointsPerUnit: 0.2},
		},
	})
	if err != nil {
		t.Fatalf("NewBaremoRuleSet() error = %v", err)
	}
	return ruleSet
}

func procedureSolicitud(id, candidateID string) ports.SolicitudRecord {
	return ports.SolicitudRecord{
		ID: id, ConvocatoriaID: "conv-1", CandidateID: candidateID,
		SorteoKey: candidateID, Estado: domain.SolicitudStateInscrita,
		Result: domain.BaremoResult{
			TotalPoints:    1,
			RuleSetID:      "conv-1",
			RuleSetVersion: "v1",
			SectionPoints: map[domain.BaremoSection]float64{
				domain.BaremoSectionExperiencia: 1,
			},
			Details: []domain.BaremoMeritScore{{
				MeritID: "m-" + id, MeritType: domain.MeritTypeExperienciaMismaCategoria,
				Section: domain.BaremoSectionExperiencia, RawPoints: 1, AppliedPoints: 1,
			}},
		},
	}
}

func solicitudIDs(solicitudes []ports.SolicitudRecord) []string {
	ids := make([]string, 0, len(solicitudes))
	for _, solicitud := range solicitudes {
		ids = append(ids, solicitud.ID)
	}
	return ids
}
