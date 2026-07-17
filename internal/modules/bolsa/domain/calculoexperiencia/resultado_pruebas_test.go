package calculoexperiencia

import (
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

const (
	reglaResultadoPrueba   = "experiencia.general"
	grupoResultadoPrueba   = "grupo.experiencia"
	seccionResultadoPrueba = "experiencia"
)

func resultadoCompletadoPrueba(t *testing.T) ResultadoExperienciaV1 {
	t.Helper()
	tramo := referenciaTramoResultadoPrueba(t, 'a')
	registrador := registradorResultadoPrueba(t, seleccionExperiencia{
		aplicaciones: []aplicacionSeleccion{{
			tramo: tramo, reglaClave: reglaResultadoPrueba, grupoClave: grupoResultadoPrueba,
			seccionClave: seccionResultadoPrueba, prioridad: 1, razon: motivoAplicacionUnica,
		}},
		evaluaciones: 1,
	})
	registrarDetalleResultadoPrueba(t, registrador, tramo, "1/2", 2_000_000, "1000000/1")
	seccionFinal, _ := baremacion.PuntosDesdeMicropuntos(1_000_000)
	if err := registrador.registrarSeccion(SubtotalSeccionResultadoExperienciaV1{
		seccionClave:  seccionResultadoPrueba,
		antesTope:     exactoResultadoPrueba(t, "1000000/1"),
		tope:          topeLimitadoResultadoPrueba(t, "1000000/1", "10000000/1", "1000000/1", false),
		puntosFinales: seccionFinal,
	}); err != nil {
		t.Fatal(err)
	}
	resultado, err := registrador.sellarCompletado(seccionFinal)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func resultadoGrandeRescatadoPrueba(t *testing.T) ResultadoExperienciaV1 {
	t.Helper()
	tramo := referenciaTramoResultadoPrueba(t, 'b')
	registrador := registradorResultadoPrueba(t, seleccionExperiencia{
		aplicaciones: []aplicacionSeleccion{{
			tramo: tramo, reglaClave: reglaResultadoPrueba, grupoClave: grupoResultadoPrueba,
			seccionClave: seccionResultadoPrueba, prioridad: 1, razon: motivoAplicacionUnica,
		}},
		evaluaciones: 1,
	})
	registrarDetalleResultadoPrueba(
		t, registrador, tramo, "2/1", baremacion.MaximoMicropuntos, "18000000000000000/1",
	)
	seccionFinal, _ := baremacion.PuntosDesdeMicropuntos(1_000_000)
	if err := registrador.registrarSeccion(SubtotalSeccionResultadoExperienciaV1{
		seccionClave: seccionResultadoPrueba,
		antesTope:    exactoResultadoPrueba(t, "18000000000000000/1"),
		tope: topeLimitadoResultadoPrueba(
			t, "18000000000000000/1", "1000000/1", "1000000/1", true,
		),
		puntosFinales: seccionFinal,
	}); err != nil {
		t.Fatal(err)
	}
	resultado, err := registrador.sellarCompletado(seccionFinal)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func registrarDetalleResultadoPrueba(
	t *testing.T,
	registrador *registradorResultadoExperienciaV1,
	tramo reglasbaremo.ReferenciaVersionada,
	unidades string,
	coeficienteMicropuntos int64,
	bruto string,
) {
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
	if err := registrador.registrarIntervalo(IntervaloAplicacionResultadoExperienciaV1{
		tramo: tramo, reglaClave: reglaResultadoPrueba, periodo: periodo,
		extremo: reglasbaremo.ExtremoFinalExclusivo, efectivo: efectivo,
		tieneEfectivo: true, dias: 30,
	}); err != nil {
		t.Fatal(err)
	}
	fraccion := baremacion.JornadaCompleta()
	razon := RazonJornadaIntegra
	factor := "1/1"
	if unidades == "1/2" {
		fraccion, err = baremacion.NuevaFraccionJornada(1, 2)
		if err != nil {
			t.Fatal(err)
		}
		razon = RazonJornadaProporcional
		factor = "1/2"
	}
	if err := registrador.registrarAplicacion(AplicacionCalculadaResultadoExperienciaV1{
		tramo: tramo, reglaClave: reglaResultadoPrueba,
		jornada: JornadaResultadoExperienciaV1{
			origen: fraccion, modo: modoJornadaResultadoPrueba(unidades),
			factor: exactoResultadoPrueba(t, factor), razon: razon,
		},
		unidades: UnidadesAplicacionResultadoExperienciaV1{
			exactas:   exactoResultadoPrueba(t, unidades),
			aportadas: exactoResultadoPrueba(t, unidades),
			resto:     exactoResultadoPrueba(t, "0/1"), frontera: FronteraRestosResultadoExacta,
		},
		puntuacion: PuntuacionPeriodoResultadoExperienciaV1{
			bruto: exactoResultadoPrueba(t, bruto),
		},
	}); err != nil {
		t.Fatal(err)
	}
	coeficiente, err := baremacion.PuntosDesdeMicropuntos(coeficienteMicropuntos)
	if err != nil {
		t.Fatal(err)
	}
	if err := registrador.registrarRegla(ResultadoReglaExperienciaV1{
		seccionClave: seccionResultadoPrueba, reglaClave: reglaResultadoPrueba,
		unidadesAgregadas:  exactoResultadoPrueba(t, unidades),
		unidadesTrasRestos: exactoResultadoPrueba(t, unidades),
		restoRegla:         exactoResultadoPrueba(t, "0/1"),
		topeUnidades:       topeIlimitadoResultadoPrueba(t, unidades),
		coeficiente:        coeficiente, bruto: exactoResultadoPrueba(t, bruto),
		redondeo: RedondeoResultadoExperienciaV1{
			momento: reglasbaremo.RedondearPorRegla, modo: baremacion.RedondeoExacto,
			entrada: exactoResultadoPrueba(t, bruto), salida: exactoResultadoPrueba(t, bruto),
		},
		topePuntos:    topeIlimitadoResultadoPrueba(t, bruto),
		puntosFinales: exactoResultadoPrueba(t, bruto),
	}); err != nil {
		t.Fatal(err)
	}
}

func modoJornadaResultadoPrueba(unidades string) reglasbaremo.ModoJornada {
	if unidades == "1/2" {
		return reglasbaremo.JornadaProporcional
	}
	return reglasbaremo.JornadaIntegra
}

func registradorResultadoPrueba(
	t *testing.T,
	seleccion seleccionExperiencia,
) *registradorResultadoExperienciaV1 {
	t.Helper()
	registrador, err := nuevoRegistradorResultadoConVinculosV1(
		vinculosResultadoPrueba(t), seleccion,
	)
	if err != nil {
		t.Fatal(err)
	}
	return registrador
}

func vinculosResultadoPrueba(t *testing.T) VinculosResultadoExperienciaV1 {
	t.Helper()
	motor, err := vinculoMotorResultadoExperienciaV1()
	if err != nil {
		t.Fatal(err)
	}
	conjunto := referenciaResultadoPrueba(t, "reglas:resultado:1", 'c')
	huellaPlan, err := huellaPlanResultadoExperienciaV1(motor, conjunto)
	if err != nil {
		t.Fatal(err)
	}
	return VinculosResultadoExperienciaV1{
		motor: motor,
		plan: VinculoPlanResultadoExperienciaV1{
			esquema: esquemaPlanResultadoV1, huellaSHA256: huellaPlan,
		},
		conjunto: conjunto,
		entrada: VinculoEntradaResultadoExperienciaV1{
			instantanea:           referenciaResultadoPrueba(t, "iex_"+strings.Repeat("d", 64), 'd'),
			huellaContenidoSHA256: strings.Repeat("e", 64),
		},
		fechaCorte: fechaResultadoPrueba(t, 2026, 12, 31),
	}
}

func resultadoBloqueadoSeleccionPrueba(t *testing.T) ResultadoExperienciaV1 {
	t.Helper()
	registrador := registradorResultadoPrueba(t, seleccionExperiencia{
		bloqueos: []bloqueoSeleccion{{
			codigo:         bloqueoCatalogoIncompatible,
			tramo:          referenciaTramoResultadoPrueba(t, 'f'),
			claveGobernada: "categoria",
		}},
	})
	resultado, err := registrador.sellarBloqueado(FaseResultadoSeleccion)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func topeIlimitadoResultadoPrueba(t *testing.T, valor string) TopeResultadoExperienciaV1 {
	t.Helper()
	exacto := exactoResultadoPrueba(t, valor)
	return TopeResultadoExperienciaV1{antes: exacto, despues: exacto}
}

func topeLimitadoResultadoPrueba(
	t *testing.T,
	antes string,
	limite string,
	despues string,
	aplicado bool,
) TopeResultadoExperienciaV1 {
	t.Helper()
	return TopeResultadoExperienciaV1{
		antes: exactoResultadoPrueba(t, antes), limitado: true,
		limite: exactoResultadoPrueba(t, limite), despues: exactoResultadoPrueba(t, despues),
		aplicado: aplicado,
	}
}

func exactoResultadoPrueba(t *testing.T, valor string) exactoResultadoV1 {
	t.Helper()
	exacto, err := nuevoExactoResultadoV1(valor)
	if err != nil {
		t.Fatal(err)
	}
	return exacto
}

func referenciaTramoResultadoPrueba(t *testing.T, caracter byte) reglasbaremo.ReferenciaVersionada {
	t.Helper()
	return referenciaResultadoPrueba(t, "trm_"+strings.Repeat(string(caracter), 64), caracter)
}

func referenciaResultadoPrueba(
	t *testing.T,
	referencia string,
	caracter byte,
) reglasbaremo.ReferenciaVersionada {
	t.Helper()
	resultado, err := reglasbaremo.NuevaReferenciaVersionada(
		referencia, 1, strings.Repeat(string(caracter), 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	return resultado
}

func fechaResultadoPrueba(t *testing.T, anio, mes, dia int) baremacion.FechaCivil {
	t.Helper()
	fecha, err := baremacion.NuevaFechaCivil(anio, mes, dia)
	if err != nil {
		t.Fatal(err)
	}
	return fecha
}
