package calculoexperiencia

import (
	"testing"

	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestResultadoExperienciaV1ConservaRestoDescartadoPorPeriodo(t *testing.T) {
	resultado := resultadoCompletadoPrueba(t).clonar()
	cero := exactoResultadoPrueba(t, "0/1")
	medio := exactoResultadoPrueba(t, "1/2")

	resultado.aplicaciones[0].unidades.aportadas = cero
	resultado.aplicaciones[0].unidades.resto = medio
	resultado.aplicaciones[0].unidades.frontera = FronteraRestosResultadoPeriodo
	resultado.aplicaciones[0].puntuacion.bruto = cero
	resultado.reglas[0].unidadesAgregadas = medio
	resultado.reglas[0].unidadesTrasRestos = cero
	resultado.reglas[0].restoRegla = medio
	resultado.reglas[0].topeUnidades = topeIlimitadoResultadoPrueba(t, "0/1")
	resultado.reglas[0].bruto = cero
	resultado.reglas[0].redondeo.entrada = cero
	resultado.reglas[0].redondeo.salida = cero
	resultado.reglas[0].topePuntos = topeIlimitadoResultadoPrueba(t, "0/1")
	resultado.reglas[0].puntosFinales = cero
	resultado.secciones[0].antesTope = cero
	resultado.secciones[0].tope = topeLimitadoResultadoPrueba(
		t, "0/1", "10000000/1", "0/1", false,
	)
	ceroPuntos, err := baremacion.PuntosDesdeMicropuntos(0)
	if err != nil {
		t.Fatal(err)
	}
	resultado.secciones[0].puntosFinales = ceroPuntos
	resultado.total = ceroPuntos

	if err := resultado.Validar(); err != nil {
		t.Fatalf("resultado con resto explicito rechazado: %v", err)
	}

	// La representacion antigua perdia la fraccion descartada y aparentaba que
	// nunca habia existido. Aunque sus totales cuadren a cero, ya no es valida.
	resultado.reglas[0].unidadesAgregadas = cero
	resultado.reglas[0].restoRegla = cero
	if resultado.Validar() == nil {
		t.Fatal("se acepto un descarte por periodo que ocultaba las unidades exactas")
	}
}
