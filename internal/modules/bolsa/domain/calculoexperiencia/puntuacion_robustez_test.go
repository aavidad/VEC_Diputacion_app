package calculoexperiencia

import (
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestPuntuacionEmiteEnOrdenSeleccionAunqueTemporalLlegueEnOtroOrden(t *testing.T) {
	configuracion := configuracionPuntuacionPrueba(t)
	periodos := [][2]baremacion.FechaCivil{
		fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 1}),
		fechasPuntuacionPrueba(t, [3]int{2026, 1, 2}, [3]int{2026, 1, 2}),
	}
	escenario := construirEscenarioPuntuacionPrueba(
		t, configuracion, periodos, baremacion.JornadaCompleta(), false,
	)
	escenario.temporales.aplicaciones[0], escenario.temporales.aplicaciones[1] =
		escenario.temporales.aplicaciones[1], escenario.temporales.aplicaciones[0]
	calculado, _ := ejecutarEscenarioPuntuacionPrueba(t, escenario)
	for indice, seleccionada := range escenario.seleccion.aplicaciones {
		if !referenciasPlanIguales(calculado.intervalos[indice].tramo, seleccionada.tramo) ||
			!referenciasPlanIguales(calculado.aplicaciones[indice].tramo, seleccionada.tramo) {
			t.Fatal("la salida heredo el orden incidental de la fase temporal")
		}
	}
}

func TestPuntuacionRealNoExponeServicioAtributosNiReferenciaDeAtestacion(t *testing.T) {
	configuracion := configuracionPuntuacionPrueba(t)
	configuracion.jornada = debePuntuacion(reglasbaremo.NuevaPoliticaJornada(
		reglasbaremo.JornadaProtegidaIntegra,
	))
	periodo := fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 1})
	escenario := construirEscenarioPuntuacionPrueba(
		t, configuracion, [][2]baremacion.FechaCivil{periodo},
		debePuntuacion(baremacion.NuevaFraccionJornada(1, 2)), true,
	)
	_, resultado := ejecutarEscenarioPuntuacionPrueba(t, escenario)
	contenido := string(debePuntuacion(resultado.RepresentacionCanonica()))
	tramo := escenario.entrada.tramos[0]
	atestacion, _ := tramo.atestacion.Referencia()
	for _, prohibido := range []string{
		tramo.servicioRef, tramo.atributos[0].valor, atestacion.Referencia(),
	} {
		if strings.Contains(contenido, prohibido) {
			t.Fatalf("el resultado expuso material de entrada no minimizado")
		}
	}
}

func TestPuntuacionRechazaParticionTemporalManipulada(t *testing.T) {
	configuracion := configuracionPuntuacionPrueba(t)
	plan := debePuntuacion(Compilar(conjuntoCompilacionPrueba(t, configuracion)))
	periodo := fechasPuntuacionPrueba(t, [3]int{2026, 8, 1}, [3]int{2026, 8, 2})
	tramo := tramoPuntuacionPrueba(
		t, plan, "manipulado", periodo[0], periodo[1], baremacion.JornadaCompleta(), false,
	)
	entrada := debePuntuacion(NuevaEntradaExperiencia(
		referenciaTokenSeleccionPrueba(t, prefijoInstantaneaEntrada, "manipulada", 1),
		[]TramoExperiencia{tramo},
	))
	seleccion := debePuntuacion(seleccionarAplicaciones(plan, entrada))
	temporales := debePuntuacion(resolverAplicacionesTemporales(plan, entrada, seleccion))
	temporales.exclusiones[0].reglaClave = "regla_ajena"
	if _, err := puntuarExperienciaV1(plan, entrada, seleccion, temporales); !errors.Is(err, ErrContextoIncompatible) {
		t.Fatalf("particion manipulada aceptada: %v", err)
	}
}

func TestPuntuacionRechazaSeleccionValidaEnFormaPeroNoDerivadaDeLaEntrada(t *testing.T) {
	configuracion := configuracionPuntuacionPrueba(t)
	periodo := fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 1})
	escenario := construirEscenarioPuntuacionPrueba(
		t, configuracion, [][2]baremacion.FechaCivil{periodo},
		baremacion.JornadaCompleta(), false,
	)
	escenario.seleccion.aplicaciones[0].razon = motivoAplicacionAcumulada
	if _, err := puntuarExperienciaV1(
		escenario.plan, escenario.entrada, escenario.seleccion, escenario.temporales,
	); !errors.Is(err, ErrContextoIncompatible) {
		t.Fatalf("seleccion no rederivada aceptada: %v", err)
	}
}

func TestMaterializarIntervalosAdmiteSolapeSinIniciarAritmetica(t *testing.T) {
	configuracion := configuracionPuntuacionPrueba(t)
	periodos := [][2]baremacion.FechaCivil{
		fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 10}),
		fechasPuntuacionPrueba(t, [3]int{2026, 1, 5}, [3]int{2026, 1, 15}),
	}
	escenario := construirEscenarioPuntuacionPrueba(
		t, configuracion, periodos, baremacion.JornadaCompleta(), false,
	)
	if !escenario.temporales.bloqueada() {
		t.Fatal("la preparacion no detecto el solape")
	}
	intervalos, err := materializarIntervalosPuntuacionV1(
		escenario.plan, escenario.entrada, escenario.seleccion, escenario.temporales,
	)
	if err != nil || len(intervalos) != len(escenario.seleccion.aplicaciones) {
		t.Fatalf("no se materializaron todos los intervalos: %v", err)
	}
	for indice, seleccionada := range escenario.seleccion.aplicaciones {
		if !referenciasPlanIguales(intervalos[indice].tramo, seleccionada.tramo) {
			t.Fatal("intervalos de solape fuera del orden de seleccion")
		}
	}
	if _, err := puntuarExperienciaV1(
		escenario.plan, escenario.entrada, escenario.seleccion, escenario.temporales,
	); !errors.Is(err, ErrContextoIncompatible) {
		t.Fatalf("la aritmetica avanzo tras el solape: %v", err)
	}
}

func TestPuntuacionPreflightTipadoYContadorReal(t *testing.T) {
	permitido, err := comprobarPresupuestoPuntuacionV1(49_998, 0, 0)
	if err != nil || permitido != 999_992 {
		t.Fatalf("frontera permitida: %d, %v", permitido, err)
	}
	if _, err := comprobarPresupuestoPuntuacionV1(49_999, 0, 0); !errors.Is(err, ErrLimiteOperaciones) {
		t.Fatalf("presupuesto excesivo aceptado: %v", err)
	}
	configuracion := configuracionPuntuacionPrueba(t)
	periodo := fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 1})
	escenario := construirEscenarioPuntuacionPrueba(
		t, configuracion, [][2]baremacion.FechaCivil{periodo},
		baremacion.JornadaCompleta(), false,
	)
	calculado := debePuntuacion(puntuarExperienciaV1(
		escenario.plan, escenario.entrada, escenario.seleccion, escenario.temporales,
	))
	previstas := baseOperacionesPuntuacionV1 + porAplicacionPuntuacionV1 +
		porReglaPuntuacionV1 + porSeccionPuntuacionV1
	if calculado.operaciones == 0 || calculado.operaciones > previstas {
		t.Fatalf("contador real fuera del preflight: %d > %d", calculado.operaciones, previstas)
	}
}
