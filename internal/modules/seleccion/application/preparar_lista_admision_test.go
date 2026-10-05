package application_test

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

// catalogoFijo sirve una única versión; el adaptador de fichero tiene sus pruebas.
type catalogoFijo struct{ c domain.CatalogoAdmision }

func (f catalogoFijo) CatalogoAdmision(_ context.Context, ref, version string) (domain.CatalogoAdmision, error) {
	if ref != f.c.Referencia || version != f.c.Version {
		return domain.CatalogoAdmision{}, errors.New("no existe")
	}
	return f.c, nil
}

func catalogoEjemplo(*testing.T) catalogoFijo {
	return catalogoFijo{domain.CatalogoAdmision{Referencia: "seleccion-admision-ejemplo", Version: "ejemplo-1", PaqueteEjemplo: true, DudaRef: "dudas-139",
		PlazoSubsanacion: domain.PlazoSubsanacion{Unidad: "dias_habiles", Cantidad: 10},
		Motivos: []domain.MotivoExclusion{{Codigo: "titulacion_no_acreditada", Subsanable: true},
			{Codigo: "tasa_no_justificada", Subsanable: true}}}}
}

func materialLista(t *testing.T) ports.MaterialListaAdmision {
	t.Helper()
	m := ports.MaterialListaAdmision{ListaRef: "lista:provisional", Revision: 1, Alcance: domain.AlcanceAdmisionPreparacion,
		CatalogoRef: "seleccion-admision-ejemplo", CatalogoVersion: "ejemplo-1"}
	for i, ref := range []string{"preparacion:uno", "preparacion:dos"} {
		s4 := materialAdmision(t)
		s4.PreparacionRef = ref
		m.Bases = s4.Bases
		a, err := application.IdentificarAntecedenteAdmision(context.Background(), s4)
		if err != nil {
			t.Fatal(err)
		}
		d := domain.DecisionAdmision{Antecedente: a, Decision: domain.DecisionAdmitida}
		if i == 1 {
			d = domain.DecisionAdmision{Antecedente: a, Decision: domain.DecisionExcluida, Motivos: []string{"titulacion_no_acreditada"}}
		}
		m.RevisionesS4 = append(m.RevisionesS4, s4)
		m.Decisiones = append(m.Decisiones, d)
	}
	return m
}

func TestListaProvisionalUsaRevisionesS4YCatalogoConfigurado(t *testing.T) {
	l, err := application.PrepararListaAdmisionProvisional(context.Background(), materialLista(t), catalogoEjemplo(t))
	if err != nil {
		t.Fatal(err)
	}
	if l.Resumen.Admitidas != 1 || l.Resumen.ExcluidasSubsanables != 1 || l.PlazoSubsanacion.Cantidad != 10 ||
		!l.Catalogo.PaqueteEjemplo || l.Catalogo.Version != "ejemplo-1" || l.Aprobada || l.Publicada {
		t.Fatalf("%+v", l)
	}
}

type catalogoCaido struct{}

func (catalogoCaido) CatalogoAdmision(context.Context, string, string) (domain.CatalogoAdmision, error) {
	return domain.CatalogoAdmision{}, errors.New("caído")
}

func TestListaProvisionalFallaCerradaAnteMaterialIncoherente(t *testing.T) {
	cambios := map[string]func(*ports.MaterialListaAdmision){
		"falta_decision": func(m *ports.MaterialListaAdmision) { m.Decisiones = m.Decisiones[:1] },
		"huella_alterada": func(m *ports.MaterialListaAdmision) {
			m.Decisiones[0].Antecedente.HuellaMaterialSHA256 = m.Decisiones[1].Antecedente.HuellaMaterialSHA256
		},
		"s4_modificada":      func(m *ports.MaterialListaAdmision) { m.RevisionesS4[0].Revision = 2 },
		"otras_bases":        func(m *ports.MaterialListaAdmision) { m.Bases.Version = "2" },
		"revision_repetida":  func(m *ports.MaterialListaAdmision) { m.RevisionesS4[1] = m.RevisionesS4[0] },
		"decision_duplicada": func(m *ports.MaterialListaAdmision) { m.Decisiones[1] = m.Decisiones[0] },
		"alcance":            func(m *ports.MaterialListaAdmision) { m.Alcance = "oficial" },
	}
	for nombre, cambiar := range cambios {
		m := materialLista(t)
		cambiar(&m)
		if _, err := application.PrepararListaAdmisionProvisional(context.Background(), m, catalogoEjemplo(t)); !errors.Is(err, domain.ErrListaAdmision) {
			t.Errorf("%s: %v", nombre, err)
		}
	}
	m := materialLista(t)
	m.CatalogoVersion = "ejemplo-2"
	if _, err := application.PrepararListaAdmisionProvisional(context.Background(), m, catalogoEjemplo(t)); err != application.ErrCatalogoAdmisionNoDisponible {
		t.Fatalf("versión inexistente: %v", err)
	}
	if _, err := application.PrepararListaAdmisionProvisional(context.Background(), materialLista(t), catalogoCaido{}); err != application.ErrCatalogoAdmisionNoDisponible {
		t.Fatalf("catálogo caído: %v", err)
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := application.PrepararListaAdmisionProvisional(ctx, materialLista(t), catalogoEjemplo(t)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación: %v", err)
	}
}
