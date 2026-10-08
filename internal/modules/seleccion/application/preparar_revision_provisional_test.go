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
	huella, err := application.IdentificarListaProvisional(context.Background(), primera, catalogoEjemplo(t))
	if err != nil {
		t.Fatal(err)
	}
	return ports.MaterialRevisionProvisional{Material: segunda, Anterior: anterior, Antecedente: huella}
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

// Ataque de la revisión independiente: una anterior coherente por dentro pero
// recortada, con la excluida no subsanable convertida en «incorporada» admitida.
func TestRevisionRechazaAnteriorQueNoEsLaDeclarada(t *testing.T) {
	m := materialRevision(t)
	excluida := m.Anterior.Excluidas[0]
	m.Anterior.Excluidas = nil
	m.Anterior.Resumen = domain.ResumenListaAdmision{Solicitudes: 1, Admitidas: 1}
	for i, d := range m.Material.Decisiones {
		if d.Antecedente == excluida.Antecedente {
			m.Material.Decisiones[i] = domain.DecisionAdmision{Antecedente: d.Antecedente, Decision: domain.DecisionAdmitida}
		}
	}
	if _, err := domain.ComprobarRevisionProvisional(m.Anterior, mustLista(t, m.Material)); err != nil {
		t.Fatalf("el ataque debe ser coherente por dentro para que la prueba tenga sentido: %v", err)
	}
	if _, err := application.PrepararRevisionProvisional(context.Background(), m, catalogoEjemplo(t)); !errors.Is(err, domain.ErrRevisionLista) {
		t.Fatalf("anterior recortada aceptada: %v", err)
	}
	sinHuella := materialRevision(t)
	sinHuella.Antecedente = domain.AntecedenteLista{}
	if _, err := application.PrepararRevisionProvisional(context.Background(), sinHuella, catalogoEjemplo(t)); !errors.Is(err, domain.ErrRevisionLista) {
		t.Fatalf("sin huella declarada: %v", err)
	}
}

func mustLista(t *testing.T, m ports.MaterialListaAdmision) domain.ListaAdmisionProvisional {
	t.Helper()
	l, err := application.PrepararListaAdmisionProvisional(context.Background(), m, catalogoEjemplo(t))
	if err != nil {
		t.Fatal(err)
	}
	return l
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
