package application_test

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

func materialDefinitiva(t *testing.T) ports.MaterialListaDefinitiva {
	t.Helper()
	provisional := materialLista(t)
	a, err := application.IdentificarListaProvisional(context.Background(), provisional, catalogoEjemplo(t))
	if err != nil {
		t.Fatal(err)
	}
	return ports.MaterialListaDefinitiva{ListaRef: "lista:definitiva", Revision: 1, Alcance: domain.AlcanceAdmisionPreparacion,
		Provisional: provisional, Antecedente: a, Resoluciones: []domain.ResolucionSubsanacion{
			{Antecedente: provisional.Decisiones[1].Antecedente, Resultado: domain.ResolucionEstimada, Via: domain.ViaSubsanacion}}}
}

func TestDefinitivaRecomponeLaProvisionalYCompruebaSuHuella(t *testing.T) {
	d, err := application.PrepararListaAdmisionDefinitiva(context.Background(), materialDefinitiva(t), catalogoEjemplo(t))
	if err != nil {
		t.Fatal(err)
	}
	if d.Resumen.Admitidas != 2 || d.Resumen.AdmitidasTrasEscrito != 1 || d.Resumen.Excluidas != 0 || d.Aprobada || d.Catalogo.Version != "ejemplo-1" {
		t.Fatalf("%+v", d)
	}
	cambios := map[string]func(*ports.MaterialListaDefinitiva){
		"huella_declarada": func(m *ports.MaterialListaDefinitiva) {
			m.Antecedente.HuellaSHA256 = m.Provisional.Decisiones[0].Antecedente.HuellaMaterialSHA256
		},
		// Mismas excluidas y resoluciones con otro motivo subsanable: solo la
		// huella de la provisional puede detectar el cambio.
		"provisional_modificada": func(m *ports.MaterialListaDefinitiva) {
			m.Provisional.Decisiones[1].Motivos = []string{"tasa_no_justificada"}
		},
		"alcance": func(m *ports.MaterialListaDefinitiva) { m.Alcance = "oficial" },
	}
	for nombre, cambiar := range cambios {
		m := materialDefinitiva(t)
		cambiar(&m)
		if _, err := application.PrepararListaAdmisionDefinitiva(context.Background(), m, catalogoEjemplo(t)); !errors.Is(err, domain.ErrListaDefinitiva) {
			t.Errorf("%s: %v", nombre, err)
		}
	}
	if _, err := application.PrepararListaAdmisionDefinitiva(context.Background(), materialDefinitiva(t), catalogoCaido{}); err != application.ErrCatalogoAdmisionNoDisponible {
		t.Fatalf("catálogo caído: %v", err)
	}
}
