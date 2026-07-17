package calculoexperiencia

import (
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestResultadoExperienciaV1RedondeoPorPeriodoConservaDosSumas(t *testing.T) {
	resultado := resultadoCompletadoPrueba(t).clonar()
	coeficiente, _ := baremacion.PuntosDesdeMicropuntos(1)
	resultado.aplicaciones[0].puntuacion.bruto = exactoResultadoPrueba(t, "1/2")
	resultado.aplicaciones[0].puntuacion.redondeado = exactoResultadoPrueba(t, "1/1")
	resultado.aplicaciones[0].puntuacion.tieneRedondeado = true
	resultado.reglas[0].coeficiente = coeficiente
	resultado.reglas[0].bruto = exactoResultadoPrueba(t, "1/2")
	resultado.reglas[0].redondeo = RedondeoResultadoExperienciaV1{
		momento: reglasbaremo.RedondearPorPeriodo,
		modo:    baremacion.RedondeoHaciaArriba,
		entrada: exactoResultadoPrueba(t, "1/2"),
		salida:  exactoResultadoPrueba(t, "1/1"),
	}
	resultado.reglas[0].topePuntos = topeIlimitadoResultadoPrueba(t, "1/1")
	resultado.reglas[0].puntosFinales = exactoResultadoPrueba(t, "1/1")
	resultado.secciones[0].antesTope = exactoResultadoPrueba(t, "1/1")
	resultado.secciones[0].tope = topeLimitadoResultadoPrueba(
		t, "1/1", "10000000/1", "1/1", false,
	)
	punto, _ := baremacion.PuntosDesdeMicropuntos(1)
	resultado.secciones[0].puntosFinales = punto
	resultado.total = punto
	if err := resultado.Validar(); err != nil {
		t.Fatal(err)
	}
	regla := resultado.Reglas()[0]
	if regla.BrutoExacto() != "1/2" || regla.Redondeo().SalidaExacta() != "1/1" {
		t.Fatal("se confundio suma bruta con suma de periodos redondeados")
	}
}

func TestResultadoExperienciaV1RechazaRedondeoPorPeriodoFalso(t *testing.T) {
	resultado := resultadoCompletadoPrueba(t).clonar()
	coeficiente, _ := baremacion.PuntosDesdeMicropuntos(1)
	resultado.aplicaciones[0].puntuacion.bruto = exactoResultadoPrueba(t, "1/2")
	resultado.aplicaciones[0].puntuacion.redondeado = exactoResultadoPrueba(t, "0/1")
	resultado.aplicaciones[0].puntuacion.tieneRedondeado = true
	resultado.reglas[0].coeficiente = coeficiente
	resultado.reglas[0].bruto = exactoResultadoPrueba(t, "1/2")
	resultado.reglas[0].redondeo = RedondeoResultadoExperienciaV1{
		momento: reglasbaremo.RedondearPorPeriodo,
		modo:    baremacion.RedondeoHaciaArriba,
		entrada: exactoResultadoPrueba(t, "1/2"),
		salida:  exactoResultadoPrueba(t, "0/1"),
	}
	resultado.reglas[0].topePuntos = topeIlimitadoResultadoPrueba(t, "0/1")
	resultado.reglas[0].puntosFinales = exactoResultadoPrueba(t, "0/1")
	resultado.secciones[0].antesTope = exactoResultadoPrueba(t, "0/1")
	resultado.secciones[0].tope = topeLimitadoResultadoPrueba(
		t, "0/1", "10000000/1", "0/1", false,
	)
	cero, _ := baremacion.PuntosDesdeMicropuntos(0)
	resultado.secciones[0].puntosFinales = cero
	resultado.total = cero
	if resultado.Validar() == nil {
		t.Fatal("se acepto salida por periodo distinta del redondeo publicado")
	}

	resultado.reglas[0].redondeo.modo = baremacion.RedondeoExacto
	if resultado.Validar() == nil {
		t.Fatal("RedondeoExacto fraccional aparecio como completado")
	}
}
