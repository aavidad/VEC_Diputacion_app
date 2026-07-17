package calculoexperiencia

import (
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

func TestCalcularExperienciaV1CompletaYSellaElRecorrido(t *testing.T) {
	periodo := fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 30})
	escenario := construirEscenarioPuntuacionPrueba(
		t, configuracionPuntuacionPrueba(t), [][2]baremacion.FechaCivil{periodo},
		baremacion.JornadaCompleta(), false,
	)

	resultado, err := CalcularExperienciaV1(escenario.plan, escenario.entrada)
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Estado() != ResultadoExperienciaCompletado ||
		resultado.Fase() != FaseResultadoCompletado || len(resultado.Intervalos()) != 1 ||
		len(resultado.Aplicaciones()) != 1 || len(resultado.Reglas()) != 1 ||
		len(resultado.Secciones()) != 1 || len(resultado.Bloqueos()) != 0 {
		t.Fatalf("resultado completo inesperado: estado=%s fase=%s", resultado.Estado(), resultado.Fase())
	}
	total, existe := resultado.Total()
	if !existe || total.Micropuntos() != 60 {
		t.Fatalf("total inesperado: %d presente=%t", total.Micropuntos(), existe)
	}
	canonico, err := resultado.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := resultado.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	restaurado, err := RestaurarResultadoExperienciaV1ConHuellaSHA256(canonico, huella)
	if err != nil || restaurado.Validar() != nil {
		t.Fatalf("resultado sellado no restaurable: %v", err)
	}
}

func TestCalcularExperienciaV1ConservaBloqueosDeCadaFase(t *testing.T) {
	t.Run("seleccion", func(t *testing.T) {
		periodo := fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 1})
		escenario := construirEscenarioPuntuacionPrueba(
			t, configuracionPuntuacionPrueba(t), [][2]baremacion.FechaCivil{periodo},
			baremacion.JornadaCompleta(), false,
		)
		entrada := entradaConCatalogoIncompatibleCalculoPrueba(t, escenario)
		resultado, err := CalcularExperienciaV1(escenario.plan, entrada)
		exigirBloqueoCalculoPrueba(
			t, resultado, err, FaseResultadoSeleccion, BloqueoResultadoCatalogoIncompatible,
		)
		if len(resultado.Intervalos()) != 0 || len(resultado.Aplicaciones()) != 0 {
			t.Fatal("un bloqueo de seleccion avanzo a fases posteriores")
		}
	})

	t.Run("intervalos", func(t *testing.T) {
		periodos := [][2]baremacion.FechaCivil{
			fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 20}),
			fechasPuntuacionPrueba(t, [3]int{2026, 1, 10}, [3]int{2026, 1, 30}),
		}
		escenario := construirEscenarioPuntuacionPrueba(
			t, configuracionPuntuacionPrueba(t), periodos,
			baremacion.JornadaCompleta(), false,
		)
		resultado, err := CalcularExperienciaV1(escenario.plan, escenario.entrada)
		exigirBloqueoCalculoPrueba(
			t, resultado, err, FaseResultadoIntervalos, BloqueoResultadoSolape,
		)
		if len(resultado.Intervalos()) != 2 || len(resultado.Aplicaciones()) != 0 {
			t.Fatal("el bloqueo temporal no conservo exactamente sus intervalos")
		}
	})

	t.Run("puntuacion", func(t *testing.T) {
		configuracion := configuracionPuntuacionPrueba(t)
		configuracion.puntosPorUnidad = debePuntuacion(baremacion.PuntosDesdeMicropuntos(1))
		periodo := fechasPuntuacionPrueba(t, [3]int{2026, 1, 1}, [3]int{2026, 1, 1})
		escenario := construirEscenarioPuntuacionPrueba(
			t, configuracion, [][2]baremacion.FechaCivil{periodo},
			debePuntuacion(baremacion.NuevaFraccionJornada(1, 2)), false,
		)
		resultado, err := CalcularExperienciaV1(escenario.plan, escenario.entrada)
		exigirBloqueoCalculoPrueba(
			t, resultado, err, FaseResultadoPuntuacion, BloqueoResultadoRedondeoNoExacto,
		)
		if len(resultado.Intervalos()) != 1 || len(resultado.Aplicaciones()) != 1 {
			t.Fatal("el bloqueo de puntuacion perdio su explicacion por aplicacion")
		}
	})
}

func TestCalcularExperienciaV1DistingueCeroDeErrorTecnico(t *testing.T) {
	corte := fechaTemporalPrueba(t, 2026, 12, 31)
	plan := planTemporalPrueba(t, corte, []reglaTemporalPrueba{{
		clave: "experiencia.general", grupo: "grupo.general",
		extremo: reglasbaremo.ExtremoFinalInclusivo, prioridad: 1,
	}})
	tramo := tramoTemporalPrueba(
		t, "sin-coincidencia", "sin-coincidencia",
		periodoCerradoTemporalPrueba(
			t, fechaTemporalPrueba(t, 2026, 1, 1), fechaTemporalPrueba(t, 2026, 1, 30),
		),
	)
	resultado, err := CalcularExperienciaV1(plan, entradaTemporalPrueba(t, tramo))
	if err != nil {
		t.Fatal(err)
	}
	total, existe := resultado.Total()
	if resultado.Estado() != ResultadoExperienciaCompletado || !existe ||
		total.Micropuntos() != 0 || len(resultado.Seleccion().SinCoincidencia()) != 1 {
		t.Fatal("el cero explicable no se completo como resultado de negocio")
	}

	fallido, err := CalcularExperienciaV1(PlanExperiencia{}, EntradaExperiencia{})
	if err == nil || fallido.Estado() != "" || fallido.Fase() != "" ||
		len(fallido.Intervalos()) != 0 || len(fallido.Bloqueos()) != 0 {
		t.Fatal("un error tecnico devolvio un resultado parcial")
	}
}

func entradaConCatalogoIncompatibleCalculoPrueba(
	t *testing.T,
	escenario escenarioPuntuacionPrueba,
) EntradaExperiencia {
	t.Helper()
	original := escenario.entrada.Tramos()[0]
	criterio := escenario.plan.Reglas()[0].Criterios()[0]
	catalogoDistinto := referenciaSeleccionPrueba(t, "catalogo:calculo:incompatible", 1)
	atributo := debePuntuacion(NuevoAtributoCatalogado(
		criterio.Clave(), catalogoDistinto, criterio.Valores()[0],
	))
	tramo := debePuntuacion(NuevoTramoExperiencia(
		original.referencia, original.servicioRef, original.periodo, original.jornada,
		original.atestacion, []AtributoCatalogado{atributo},
	))
	return debePuntuacion(NuevaEntradaExperiencia(
		referenciaTokenSeleccionPrueba(t, prefijoInstantaneaEntrada, "calculo-incompatible", 1),
		[]TramoExperiencia{tramo},
	))
}

func exigirBloqueoCalculoPrueba(
	t *testing.T,
	resultado ResultadoExperienciaV1,
	err error,
	fase FaseResultadoExperienciaV1,
	codigo CodigoBloqueoResultadoExperienciaV1,
) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	bloqueos := resultado.Bloqueos()
	if resultado.Estado() != ResultadoExperienciaBloqueado || resultado.Fase() != fase ||
		len(bloqueos) != 1 || bloqueos[0].Codigo() != codigo {
		t.Fatalf("bloqueo inesperado: estado=%s fase=%s bloqueos=%d", resultado.Estado(), resultado.Fase(), len(bloqueos))
	}
	if _, existe := resultado.Total(); existe || len(resultado.Reglas()) != 0 ||
		len(resultado.Secciones()) != 0 {
		t.Fatal("un resultado bloqueado publico material parcial")
	}
}
