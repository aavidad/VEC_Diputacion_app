package application_test

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

func materialRevision(t *testing.T) ports.MaterialRevisionProvisional {
	t.Helper()
	primera := materialLista(t)
	anterior, err := application.PrepararListaAdmisionProvisional(context.Background(), primera, catalogoEjemplo(t))
	if err != nil {
		t.Fatal(err)
	}
	segunda := materialLista(t)
	segunda.Revision = 2
	s4 := materialAdmision(t)
	s4.PreparacionRef = "preparacion:omitida"
	a, err := application.IdentificarAntecedenteAdmision(context.Background(), s4)
	if err != nil {
		t.Fatal(err)
	}
	segunda.RevisionesS4 = append(segunda.RevisionesS4, s4)
	segunda.Decisiones = append(segunda.Decisiones, domain.DecisionAdmision{Antecedente: a, Decision: domain.DecisionAdmitida})
	return ports.MaterialRevisionProvisional{Material: segunda, Anterior: anterior}
}

func TestRevisionEnlazaLaAnteriorPorLaMismaHuellaQueLaDefinitiva(t *testing.T) {
	m := materialRevision(t)
	r, err := application.PrepararRevisionProvisional(context.Background(), m, catalogoEjemplo(t))
	if err != nil {
		t.Fatal(err)
	}
	esperada, err := application.IdentificarListaProvisional(context.Background(), materialLista(t), catalogoEjemplo(t))
	if err != nil || r.Anterior != esperada || len(r.Incorporadas) != 1 || r.Incorporadas[0] != "preparacion:omitida" || r.Lista.Revision != 2 {
		t.Fatalf("%+v %v", r, err)
	}
	// La nueva revisión se puede citar en la definitiva igual que una primera provisional.
	if _, err := application.IdentificarListaProvisional(context.Background(), m.Material, catalogoEjemplo(t)); err != nil {
		t.Fatal(err)
	}
}

func TestRevisionRechazaDecisionesCambiadasYCatalogoCaido(t *testing.T) {
	m := materialRevision(t)
	m.Material.Decisiones[1].Motivos = []string{"tasa_no_justificada"}
	if _, err := application.PrepararRevisionProvisional(context.Background(), m, catalogoEjemplo(t)); !errors.Is(err, domain.ErrRevisionLista) {
		t.Fatalf("decisión cambiada: %v", err)
	}
	if _, err := application.PrepararRevisionProvisional(context.Background(), materialRevision(t), catalogoCaido{}); err != application.ErrCatalogoAdmisionNoDisponible {
		t.Fatalf("catálogo caído: %v", err)
	}
}
