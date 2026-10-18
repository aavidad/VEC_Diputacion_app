package bootstrap

import (
	"errors"
	"testing"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestSeleccionCTPausaAnteComprobacionPrevia(t *testing.T) {
	uno, dos := uint64(1), uint64(2)
	pendiente := bolsa.EntradaOrigenLlamamientoCT{OrdenActa: 1, Razon: "pendiente_comprobacion_rrhh"}
	posterior := bolsa.EntradaOrigenLlamamientoCT{OrdenActa: 2, OrdenVigente: &uno, ParticipacionRef: "participacion:dos", CandidatoRef: "candidato:dos", Elegible: true}
	_, err := primeraElegibleOrigenConstituidoCT(bolsa.OrigenLlamamientoCT{BolsaRef: "bolsa:prueba", Entradas: []bolsa.EntradaOrigenLlamamientoCT{pendiente, posterior}})
	if !errors.Is(err, ct.ErrCriteriosOfrecimientoPendientesRRHH) {
		t.Fatalf("saltó comprobación previa: %v", err)
	}
	var detallado *ct.ErrorCriteriosOfrecimientoPendientesRRHH
	if !errors.As(err, &detallado) || detallado.Comprobacion.BolsaRef != "bolsa:prueba" ||
		detallado.Comprobacion.OrdenActa != 1 || detallado.Comprobacion.MotivoClave != "pendiente_comprobacion_rrhh" {
		t.Fatalf("pausa sin diagnóstico mínimo: %v", err)
	}

	pendiente.OrdenActa = 2
	posterior.OrdenActa = 1
	elegida, err := primeraElegibleOrigenConstituidoCT(bolsa.OrigenLlamamientoCT{Entradas: []bolsa.EntradaOrigenLlamamientoCT{posterior, pendiente}})
	if err != nil || elegida.CandidatoRef != posterior.CandidatoRef {
		t.Fatalf("bloqueó candidato anterior: %+v, %v", elegida, err)
	}

	_, err = primeraElegibleOrigenConstituidoCT(bolsa.OrigenLlamamientoCT{Entradas: []bolsa.EntradaOrigenLlamamientoCT{pendiente}})
	if !errors.Is(err, ct.ErrCriteriosOfrecimientoPendientesRRHH) {
		t.Fatalf("sin elegibles omitió pausa: %v", err)
	}

	pendiente.OrdenVigente = &dos
	_, err = primeraElegibleOrigenConstituidoCT(bolsa.OrigenLlamamientoCT{Entradas: []bolsa.EntradaOrigenLlamamientoCT{pendiente}})
	if !errors.Is(err, ct.ErrRespuestaBolsaNoConfiable) {
		t.Fatalf("aceptó pendiente con orden vigente: %v", err)
	}
}
