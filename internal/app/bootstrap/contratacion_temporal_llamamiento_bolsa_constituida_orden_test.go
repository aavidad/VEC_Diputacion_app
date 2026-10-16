package bootstrap

import (
	"errors"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestOrigenCTSeleccionaActaCincoConPrimerRangoVigente(t *testing.T) {
	entradas := make([]bolsa.EntradaOrigenLlamamientoCT, 5)
	for i := 0; i < 4; i++ {
		entradas[i] = bolsa.EntradaOrigenLlamamientoCT{
			OrdenActa: uint64(i + 1), ParticipacionRef: "participacion:no-disponible",
			Situacion: "no_disponible", Razon: "situacion_no_disponible",
		}
	}
	primerRango := uint64(1)
	entradas[4] = bolsa.EntradaOrigenLlamamientoCT{
		OrdenActa: 5, OrdenVigente: &primerRango,
		ParticipacionRef: "participacion:quinta", CandidatoRef: "candidato:quinta",
		Situacion: "disponible", Razon: "elegible", Elegible: true,
	}
	origen := bolsa.OrigenLlamamientoCT{BolsaRef: "bolsa:prueba", Entradas: entradas}
	elegida, err := primeraElegibleOrigenConstituidoCT(origen)
	if err != nil || elegida.ParticipacionRef != "participacion:quinta" || *elegida.OrdenVigente != 1 {
		t.Fatalf("no conservó la primera posición vigente: %+v, %v", elegida, err)
	}
	ordenActa, err := ordenActaPrimeraElegibleOrigenCT(elegida)
	if err != nil || ordenActa != 5 {
		t.Fatalf("el origen B110 debe reservar posición del acta 5: %d, %v", ordenActa, err)
	}
	entradas[0].Razon = "pendiente_comprobacion_rrhh"
	if _, err := primeraElegibleOrigenConstituidoCT(origen); !errors.Is(err, ct.ErrCriteriosOfrecimientoPendientesRRHH) {
		t.Fatalf("saltó comprobación anterior al acta 5: %v", err)
	}
	entradas[0].Razon = "situacion_no_disponible"
	segundoRango := uint64(2)
	entradas[4].OrdenVigente = &segundoRango
	if _, err := primeraElegibleOrigenConstituidoCT(origen); !errors.Is(err, ct.ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("aceptó emitir sin primer rango vigente: %v", err)
	}
}
