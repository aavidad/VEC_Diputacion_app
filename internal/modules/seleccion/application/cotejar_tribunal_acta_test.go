package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	basess2 "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/seleccion/application"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
)

func materialTribunalActa() (domain.PreparacionTribunal, domain.MaterialActaPropuesto) {
	tribunal, _ := domain.PrepararTribunal(domain.MaterialTribunalPropuesto{
		Alcance: "preparacion_sintetica", IdentidadMaterial: "tribunal:local", VersionMaterial: 2,
		BasesS2:         basess2.Esperada{PreparacionRef: "bases:local", Revision: 1, HuellaMaterialSHA256: strings.Repeat("a", 64)},
		FasesPropuestas: []string{"fase:primera"},
	})
	acta := domain.MaterialActaPropuesto{
		Alcance: "preparacion_sintetica", IdentidadMaterial: "acta:local", VersionMaterial: 1,
		AntecedenteTribunal: domain.AntecedenteTribunalPropuesto{IdentidadMaterial: "tribunal:local", VersionMaterial: 2},
		FasePropuesta:       "fase:primera",
	}
	return tribunal, acta
}

func TestCotejarSalidaTribunalEnlazaFaseYHuella(t *testing.T) {
	tribunal, acta := materialTribunalActa()
	huella := strings.Repeat("c", 64)
	resultado, err := application.CotejarSalidaTribunal(context.Background(), tribunal, huella, acta)
	if err != nil || resultado.MaterialPropuesto.AntecedenteTribunal.HuellaAportadaSHA256 != huella || acta.AntecedenteTribunal.HuellaAportadaSHA256 != "" {
		t.Fatalf("cotejo: %+v, %v", resultado, err)
	}
	if len(resultado.Pendientes) < 2 || resultado.Pendientes[0] != (domain.PendienteActa{Campo: "antecedente_tribunal", Codigo: "antecedente_cotejado_local"}) ||
		resultado.Pendientes[1] != (domain.PendienteActa{Campo: "fase_propuesta", Codigo: "fase_cotejada_local"}) {
		t.Fatalf("el cotejo local no debe negar los hechos comprobados: %+v", resultado.Pendientes)
	}
}

func TestCotejarSalidaTribunalRechazaCambioDeAntecedente(t *testing.T) {
	for nombre, cambiar := range map[string]func(*domain.PreparacionTribunal, *domain.MaterialActaPropuesto){
		"identidad": func(_ *domain.PreparacionTribunal, a *domain.MaterialActaPropuesto) {
			a.AntecedenteTribunal.IdentidadMaterial = "tribunal:ajeno"
		},
		"revision": func(_ *domain.PreparacionTribunal, a *domain.MaterialActaPropuesto) {
			a.AntecedenteTribunal.VersionMaterial++
		},
		"fase": func(_ *domain.PreparacionTribunal, a *domain.MaterialActaPropuesto) { a.FasePropuesta = "fase:ajena" },
		"huella": func(_ *domain.PreparacionTribunal, a *domain.MaterialActaPropuesto) {
			a.AntecedenteTribunal.HuellaAportadaSHA256 = strings.Repeat("b", 64)
		},
		"estado": func(p *domain.PreparacionTribunal, _ *domain.MaterialActaPropuesto) { p.Estado = "aprobado" },
	} {
		t.Run(nombre, func(t *testing.T) {
			tribunal, acta := materialTribunalActa()
			cambiar(&tribunal, &acta)
			if _, err := application.CotejarSalidaTribunal(context.Background(), tribunal, strings.Repeat("c", 64), acta); !errors.Is(err, application.ErrCotejoTribunalActa) {
				t.Fatalf("debe rechazar %s: %v", nombre, err)
			}
		})
	}
}
