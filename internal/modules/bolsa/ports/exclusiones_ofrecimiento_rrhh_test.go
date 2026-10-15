package ports

import (
	"testing"
	"time"
)

func TestPaginaExclusionesRechazaHechosNoLigados(t *testing.T) {
	q := ConsultaExclusionesOfrecimientoRRHH{BolsaRef: "bolsa:prueba", LlamamientoRef: "llamamiento:prueba", Limite: 1000}
	p := PaginaExclusionesOfrecimientoRRHH{
		BolsaRef: q.BolsaRef, LlamamientoRef: q.LlamamientoRef,
		Exclusiones: []ExclusionOfrecimientoRRHH{{
			ParticipacionRef: "participacion:prueba", OrdenActa: 1,
			SituacionOrigen: "activa", MotivoClave: "pendiente_comprobacion_rrhh",
			EventosRelacion: []string{"evento:uno"},
			HechosRelacion:  []HechoRelacionOfrecimientoRRHH{{EventoRef: "evento:uno", Tipo: "incorporacion", OcurridoEn: time.Now().UTC()}},
		}},
		TotalExclusiones: 1, AuditoriaRef: "auditoria:prueba", CorrelacionRef: "correlacion:prueba", ConsultadaEn: time.Now().UTC(),
	}
	if !p.ValidaPara(q) {
		t.Fatal("rechazó hecho unido al evento de su exclusión")
	}
	p.Exclusiones[0].HechosRelacion[0].EventoRef = "evento:ajeno"
	if p.ValidaPara(q) {
		t.Fatal("aceptó hecho ajeno a la exclusión")
	}
	p.Exclusiones[0].HechosRelacion[0].EventoRef = "evento:uno"
	p.Exclusiones[0].HechosRelacion[0].Tipo = "otro"
	if p.ValidaPara(q) {
		t.Fatal("aceptó tipo de hecho no publicado")
	}
	p.Exclusiones[0].HechosRelacion = nil
	if p.ValidaPara(q) {
		t.Fatal("aceptó ausencia del campo de hechos")
	}
}
