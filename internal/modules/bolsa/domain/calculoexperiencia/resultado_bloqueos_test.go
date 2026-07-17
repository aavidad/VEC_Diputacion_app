package calculoexperiencia

import (
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestResultadoExperienciaV1BloqueoTemporalConservaIntervalosYOrden(t *testing.T) {
	tramoA := referenciaTramoResultadoPrueba(t, '1')
	tramoB := referenciaTramoResultadoPrueba(t, '2')
	seleccion := seleccionExperiencia{
		aplicaciones: []aplicacionSeleccion{
			{tramo: tramoA, reglaClave: reglaResultadoPrueba, grupoClave: grupoResultadoPrueba,
				seccionClave: seccionResultadoPrueba, prioridad: 1, razon: motivoAplicacionUnica},
			{tramo: tramoB, reglaClave: reglaResultadoPrueba, grupoClave: grupoResultadoPrueba,
				seccionClave: seccionResultadoPrueba, prioridad: 1, razon: motivoAplicacionUnica},
		},
		evaluaciones: 2,
	}
	registrador := registradorResultadoPrueba(t, seleccion)
	for _, tramo := range []reglasbaremo.ReferenciaVersionada{tramoA, tramoB} {
		if err := registrador.registrarIntervalo(intervaloEfectivoResultadoPrueba(t, tramo)); err != nil {
			t.Fatal(err)
		}
	}
	bloqueo := BloqueoResultadoExperienciaV1{
		codigo:     BloqueoResultadoSolape,
		tramos:     []reglasbaremo.ReferenciaVersionada{tramoA, tramoB},
		grupoClave: grupoResultadoPrueba,
	}
	if err := registrador.registrarBloqueo(bloqueo); err != nil {
		t.Fatal(err)
	}
	resultado, err := registrador.sellarBloqueado(FaseResultadoIntervalos)
	if err != nil {
		t.Fatal(err)
	}
	bloqueos := resultado.Bloqueos()
	if len(bloqueos) != 1 || bloqueos[0].GrupoClave() != grupoResultadoPrueba ||
		len(resultado.Intervalos()) != 2 ||
		len(resultado.Aplicaciones()) != 0 {
		t.Fatal("el bloqueo temporal no quedo canonico o avanzo de fase")
	}
	contenido, err := resultado.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RestaurarResultadoExperienciaV1(contenido); err != nil {
		t.Fatal(err)
	}
}

func TestResultadoExperienciaV1OrdenaBloqueosMultiples(t *testing.T) {
	tramoA := referenciaTramoResultadoPrueba(t, '5')
	tramoB := referenciaTramoResultadoPrueba(t, '6')
	registrador := registradorResultadoPrueba(t, seleccionExperiencia{
		bloqueos: []bloqueoSeleccion{
			{codigo: bloqueoCatalogoIncompatible, tramo: tramoB, claveGobernada: "categoria"},
			{codigo: bloqueoCatalogoIncompatible, tramo: tramoA, claveGobernada: "categoria"},
		},
	})
	resultado, err := registrador.sellarBloqueado(FaseResultadoSeleccion)
	if err != nil {
		t.Fatal(err)
	}
	bloqueos := resultado.Bloqueos()
	if len(bloqueos) != 2 ||
		compararReferenciasSeleccion(bloqueos[0].tramos[0], bloqueos[1].tramos[0]) >= 0 {
		t.Fatal("los bloqueos no quedaron en orden canonico")
	}
	contenido, err := resultado.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := RestaurarResultadoExperienciaV1(contenido); err != nil {
		t.Fatal(err)
	}
}

func TestResultadoExperienciaV1BloqueoRedondeoConservaValorExacto(t *testing.T) {
	tramo := referenciaTramoResultadoPrueba(t, '3')
	registrador := registradorResultadoPrueba(t, seleccionExperiencia{
		aplicaciones: []aplicacionSeleccion{{
			tramo: tramo, reglaClave: reglaResultadoPrueba, grupoClave: grupoResultadoPrueba,
			seccionClave: seccionResultadoPrueba, prioridad: 1, razon: motivoAplicacionUnica,
		}},
		evaluaciones: 1,
	})
	if err := registrador.registrarIntervalo(intervaloEfectivoResultadoPrueba(t, tramo)); err != nil {
		t.Fatal(err)
	}
	media, _ := baremacion.NuevaFraccionJornada(1, 2)
	if err := registrador.registrarAplicacion(AplicacionCalculadaResultadoExperienciaV1{
		tramo: tramo, reglaClave: reglaResultadoPrueba,
		jornada: JornadaResultadoExperienciaV1{
			origen: media, modo: reglasbaremo.JornadaProporcional,
			factor: exactoResultadoPrueba(t, "1/2"), razon: RazonJornadaProporcional,
		},
		unidades: UnidadesAplicacionResultadoExperienciaV1{
			exactas:   exactoResultadoPrueba(t, "1/2"),
			aportadas: exactoResultadoPrueba(t, "1/2"),
			resto:     exactoResultadoPrueba(t, "0/1"), frontera: FronteraRestosResultadoExacta,
		},
		puntuacion: PuntuacionPeriodoResultadoExperienciaV1{
			bruto: exactoResultadoPrueba(t, "1/2"),
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := registrador.registrarBloqueo(BloqueoResultadoExperienciaV1{
		codigo: BloqueoResultadoRedondeoNoExacto,
		tramos: []reglasbaremo.ReferenciaVersionada{tramo},
		reglas: []string{reglaResultadoPrueba}, seccionClave: seccionResultadoPrueba,
		valorExacto: exactoResultadoPrueba(t, "1/2"), tieneValorExacto: true,
	}); err != nil {
		t.Fatal(err)
	}
	resultado, err := registrador.sellarBloqueado(FaseResultadoPuntuacion)
	if err != nil {
		t.Fatal(err)
	}
	valor, presente := resultado.Bloqueos()[0].ValorExacto()
	if !presente || valor != "1/2" || len(resultado.Reglas()) != 0 || len(resultado.Secciones()) != 0 {
		t.Fatal("el bloqueo de redondeo perdio el valor o produjo parciales oficiales")
	}
	contenido, _ := resultado.RepresentacionCanonica()
	if _, err := RestaurarResultadoExperienciaV1(contenido); err != nil {
		t.Fatal(err)
	}
}

func TestResultadoExperienciaV1SeleccionConDescarteYAusencia(t *testing.T) {
	resultado := resultadoCompletadoPrueba(t).clonar()
	tramoAplicado := resultado.seleccion.aplicaciones[0].tramo
	tramoSinCoincidencia := referenciaTramoResultadoPrueba(t, '4')
	resultado.seleccion.descartes = []DescarteSeleccionResultadoExperienciaV1{{
		tramo: tramoAplicado, reglaClave: "experiencia.inferior", grupoClave: grupoResultadoPrueba,
		reglaSeleccionada: reglaResultadoPrueba, razon: RazonPrioridadInferior,
	}}
	resultado.seleccion.sinCoincidencia = []SinCoincidenciaResultadoExperienciaV1{{
		tramo: tramoSinCoincidencia, razon: RazonNingunaCoincidencia,
	}}
	if err := resultado.Validar(); err != nil {
		t.Fatal(err)
	}
	contenido, err := resultado.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	restaurado, err := RestaurarResultadoExperienciaV1(contenido)
	if err != nil {
		t.Fatal(err)
	}
	seleccion := restaurado.Seleccion()
	if len(seleccion.Descartes()) != 1 || len(seleccion.SinCoincidencia()) != 1 {
		t.Fatal("la restauracion perdio decisiones de seleccion")
	}
}

func intervaloEfectivoResultadoPrueba(
	t *testing.T,
	tramo reglasbaremo.ReferenciaVersionada,
) IntervaloAplicacionResultadoExperienciaV1 {
	t.Helper()
	desde := fechaResultadoPrueba(t, 2026, 1, 1)
	hasta := fechaResultadoPrueba(t, 2026, 1, 31)
	periodo, err := NuevoPeriodoServicioCerrado(desde, hasta)
	if err != nil {
		t.Fatal(err)
	}
	efectivo, err := baremacion.NuevoIntervaloCivil(desde, hasta)
	if err != nil {
		t.Fatal(err)
	}
	return IntervaloAplicacionResultadoExperienciaV1{
		tramo: tramo, reglaClave: reglaResultadoPrueba, periodo: periodo,
		extremo: reglasbaremo.ExtremoFinalExclusivo, efectivo: efectivo,
		tieneEfectivo: true, dias: 30,
	}
}
