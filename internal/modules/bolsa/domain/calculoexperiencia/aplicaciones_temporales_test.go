package calculoexperiencia

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

func TestAplicacionesTemporalesNormalizanCalendarioCivil(t *testing.T) {
	casos := []struct {
		nombre        string
		corte         [3]int
		desde         [3]int
		hasta         [3]int
		enCurso       bool
		extremo       reglasbaremo.TratamientoExtremoFinal
		dias          int64
		desdeEsperado string
		hastaEsperado string
		exclusion     razonExclusionTemporal
	}{
		{
			nombre: "inclusivo_un_dia", corte: [3]int{2025, 1, 31},
			desde: [3]int{2025, 1, 7}, hasta: [3]int{2025, 1, 7},
			extremo: reglasbaremo.ExtremoFinalInclusivo, dias: 1,
			desdeEsperado: "2025-01-07", hastaEsperado: "2025-01-08",
		},
		{
			nombre: "exclusivo", corte: [3]int{2025, 1, 31},
			desde: [3]int{2025, 1, 1}, hasta: [3]int{2025, 1, 3},
			extremo: reglasbaremo.ExtremoFinalExclusivo, dias: 2,
			desdeEsperado: "2025-01-01", hastaEsperado: "2025-01-03",
		},
		{
			nombre: "abierto_hasta_corte_inclusivo", corte: [3]int{2025, 1, 5},
			desde: [3]int{2025, 1, 3}, enCurso: true,
			extremo: reglasbaremo.ExtremoFinalExclusivo, dias: 3,
			desdeEsperado: "2025-01-03", hastaEsperado: "2025-01-06",
		},
		{
			nombre: "recortado_por_corte", corte: [3]int{2025, 1, 5},
			desde: [3]int{2025, 1, 1}, hasta: [3]int{2025, 1, 20},
			extremo: reglasbaremo.ExtremoFinalInclusivo, dias: 5,
			desdeEsperado: "2025-01-01", hastaEsperado: "2025-01-06",
		},
		{
			nombre: "intervalo_vacio", corte: [3]int{2025, 1, 31},
			desde: [3]int{2025, 1, 7}, hasta: [3]int{2025, 1, 7},
			extremo:   reglasbaremo.ExtremoFinalExclusivo,
			exclusion: exclusionTemporalIntervaloVacio,
		},
		{
			nombre: "fuera_de_corte", corte: [3]int{2025, 1, 5},
			desde: [3]int{2025, 1, 6}, hasta: [3]int{2025, 1, 20},
			extremo:   reglasbaremo.ExtremoFinalInclusivo,
			exclusion: exclusionTemporalFueraDeCorte,
		},
		{
			nombre: "bisiesto", corte: [3]int{2024, 12, 31},
			desde: [3]int{2024, 2, 28}, hasta: [3]int{2024, 3, 1},
			extremo: reglasbaremo.ExtremoFinalInclusivo, dias: 3,
			desdeEsperado: "2024-02-28", hastaEsperado: "2024-03-02",
		},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			corte := fechaTemporalPrueba(t, caso.corte[0], caso.corte[1], caso.corte[2])
			desde := fechaTemporalPrueba(t, caso.desde[0], caso.desde[1], caso.desde[2])
			var periodo PeriodoServicio
			if caso.enCurso {
				periodo = debeTemporal(NuevoPeriodoServicioEnCurso(desde))
			} else {
				hasta := fechaTemporalPrueba(t, caso.hasta[0], caso.hasta[1], caso.hasta[2])
				periodo = periodoCerradoTemporalPrueba(t, desde, hasta)
			}
			plan := planTemporalPrueba(t, corte, []reglaTemporalPrueba{
				{"regla", "grupo", caso.extremo, 1},
			})
			tramo := tramoTemporalPrueba(t, caso.nombre, caso.nombre, periodo)
			entrada := entradaTemporalPrueba(t, tramo)
			seleccion := seleccionTemporalPrueba(t, plan, asignacionTemporalPrueba{tramo, "regla"})

			resultado, err := resolverAplicacionesTemporales(plan, entrada, seleccion)
			if err != nil {
				t.Fatalf("resolver: %v", err)
			}
			if resultado.aplicacionesProcesadas != 1 {
				t.Fatalf("aplicaciones procesadas = %d", resultado.aplicacionesProcesadas)
			}
			if caso.exclusion != "" {
				if len(resultado.aplicaciones) != 0 || len(resultado.exclusiones) != 1 ||
					resultado.exclusiones[0].razon != caso.exclusion ||
					resultado.eventosProcesados != 0 {
					t.Fatalf("exclusion incorrecta: %#v", resultado)
				}
				return
			}
			if len(resultado.aplicaciones) != 1 || len(resultado.exclusiones) != 0 ||
				resultado.eventosProcesados != 2 {
				t.Fatalf("resultado temporal incorrecto: %#v", resultado)
			}
			aplicacion := resultado.aplicaciones[0]
			if aplicacion.dias != caso.dias ||
				aplicacion.intervalo.Desde().String() != caso.desdeEsperado ||
				aplicacion.intervalo.Hasta().String() != caso.hastaEsperado {
				t.Fatalf("intervalo = [%s,%s), %d dias", aplicacion.intervalo.Desde(),
					aplicacion.intervalo.Hasta(), aplicacion.dias)
			}
		})
	}
}

func TestAplicacionesTemporalesFusionanReglasDelMismoTramo(t *testing.T) {
	plan := planTemporalPrueba(t, fechaTemporalPrueba(t, 2025, 12, 31), []reglaTemporalPrueba{
		{"inclusiva", "grupo", reglasbaremo.ExtremoFinalInclusivo, 1},
		{"exclusiva", "grupo", reglasbaremo.ExtremoFinalExclusivo, 2},
	})
	tramo := tramoTemporalPrueba(t, "unico", "unico", periodoCerradoTemporalPrueba(
		t, fechaTemporalPrueba(t, 2025, 1, 1), fechaTemporalPrueba(t, 2025, 1, 10),
	))
	entrada := entradaTemporalPrueba(t, tramo)
	seleccion := seleccionTemporalPrueba(t, plan,
		asignacionTemporalPrueba{tramo, "inclusiva"},
		asignacionTemporalPrueba{tramo, "exclusiva"},
	)

	resultado, err := resolverAplicacionesTemporales(plan, entrada, seleccion)
	if err != nil {
		t.Fatal(err)
	}
	if len(resultado.aplicaciones) != 2 || len(resultado.bloqueos) != 0 ||
		resultado.eventosProcesados != 2 {
		t.Fatalf("el mismo tramo se trato como solape: %#v", resultado)
	}
}

func TestAplicacionesTemporalesResuelvenSolapesPorGrupo(t *testing.T) {
	t.Run("distintos_tramos_mismo_servicio", func(t *testing.T) {
		plan := planTemporalPrueba(t, fechaTemporalPrueba(t, 2025, 12, 31), []reglaTemporalPrueba{
			{"regla", "grupo", reglasbaremo.ExtremoFinalExclusivo, 1},
		})
		primero := tramoTemporalPrueba(t, "primero", "compartido", periodoCerradoTemporalPrueba(
			t, fechaTemporalPrueba(t, 2025, 1, 1), fechaTemporalPrueba(t, 2025, 1, 10),
		))
		segundo := tramoTemporalPrueba(t, "segundo", "compartido", periodoCerradoTemporalPrueba(
			t, fechaTemporalPrueba(t, 2025, 1, 5), fechaTemporalPrueba(t, 2025, 1, 15),
		))
		if primero.ServicioRef() != segundo.ServicioRef() {
			t.Fatal("la prueba no comparte servicioRef")
		}
		resultado, err := resolverAplicacionesTemporales(
			plan,
			entradaTemporalPrueba(t, primero, segundo),
			seleccionTemporalPrueba(t, plan,
				asignacionTemporalPrueba{primero, "regla"},
				asignacionTemporalPrueba{segundo, "regla"},
			),
		)
		if err != nil || len(resultado.bloqueos) != 1 || !resultado.bloqueada() {
			t.Fatalf("solape no bloqueado: %#v, %v", resultado, err)
		}
		bloqueo := resultado.bloqueos[0]
		if bloqueo.codigo != bloqueoTemporalSolapeRechazado || bloqueo.grupoClave != "grupo" ||
			compararReferenciasTemporales(bloqueo.tramoPrimero, bloqueo.tramoSegundo) >= 0 {
			t.Fatalf("bloqueo no canonico: %#v", bloqueo)
		}
	})

	t.Run("contiguos", func(t *testing.T) {
		plan := planTemporalPrueba(t, fechaTemporalPrueba(t, 2025, 12, 31), []reglaTemporalPrueba{
			{"regla", "grupo", reglasbaremo.ExtremoFinalExclusivo, 1},
		})
		primero := tramoTemporalPrueba(t, "contiguo-a", "a", periodoCerradoTemporalPrueba(
			t, fechaTemporalPrueba(t, 2025, 1, 1), fechaTemporalPrueba(t, 2025, 1, 10),
		))
		segundo := tramoTemporalPrueba(t, "contiguo-b", "b", periodoCerradoTemporalPrueba(
			t, fechaTemporalPrueba(t, 2025, 1, 10), fechaTemporalPrueba(t, 2025, 1, 20),
		))
		resultado, err := resolverAplicacionesTemporales(
			plan,
			entradaTemporalPrueba(t, primero, segundo),
			seleccionTemporalPrueba(t, plan,
				asignacionTemporalPrueba{primero, "regla"},
				asignacionTemporalPrueba{segundo, "regla"},
			),
		)
		if err != nil || len(resultado.bloqueos) != 0 {
			t.Fatalf("contigüidad confundida con solape: %#v, %v", resultado, err)
		}
	})

	t.Run("grupos_separados", func(t *testing.T) {
		plan := planTemporalPrueba(t, fechaTemporalPrueba(t, 2025, 12, 31), []reglaTemporalPrueba{
			{"regla_a", "grupo_a", reglasbaremo.ExtremoFinalInclusivo, 1},
			{"regla_b", "grupo_b", reglasbaremo.ExtremoFinalInclusivo, 1},
		})
		periodo := periodoCerradoTemporalPrueba(
			t, fechaTemporalPrueba(t, 2025, 1, 1), fechaTemporalPrueba(t, 2025, 1, 20),
		)
		primero := tramoTemporalPrueba(t, "grupo-a", "a", periodo)
		segundo := tramoTemporalPrueba(t, "grupo-b", "b", periodo)
		resultado, err := resolverAplicacionesTemporales(
			plan,
			entradaTemporalPrueba(t, primero, segundo),
			seleccionTemporalPrueba(t, plan,
				asignacionTemporalPrueba{primero, "regla_a"},
				asignacionTemporalPrueba{segundo, "regla_b"},
			),
		)
		if err != nil || len(resultado.bloqueos) != 0 {
			t.Fatalf("grupos independientes cruzados: %#v, %v", resultado, err)
		}
	})
}

func TestAplicacionesTemporalesSonInvariantesAntePermutacion(t *testing.T) {
	plan := planTemporalPrueba(t, fechaTemporalPrueba(t, 2025, 12, 31), []reglaTemporalPrueba{
		{"regla", "grupo", reglasbaremo.ExtremoFinalInclusivo, 1},
	})
	primero := tramoTemporalPrueba(t, "perm-a", "a", periodoCerradoTemporalPrueba(
		t, fechaTemporalPrueba(t, 2025, 1, 1), fechaTemporalPrueba(t, 2025, 1, 10),
	))
	segundo := tramoTemporalPrueba(t, "perm-b", "b", periodoCerradoTemporalPrueba(
		t, fechaTemporalPrueba(t, 2025, 1, 5), fechaTemporalPrueba(t, 2025, 1, 7),
	))
	tercero := tramoTemporalPrueba(t, "perm-c", "c", periodoCerradoTemporalPrueba(
		t, fechaTemporalPrueba(t, 2025, 2, 1), fechaTemporalPrueba(t, 2025, 2, 3),
	))
	entrada := entradaTemporalPrueba(t, primero, segundo, tercero)
	ordenUno := seleccionTemporalPrueba(t, plan,
		asignacionTemporalPrueba{primero, "regla"},
		asignacionTemporalPrueba{segundo, "regla"},
		asignacionTemporalPrueba{tercero, "regla"},
	)
	ordenDos := seleccionTemporalPrueba(t, plan,
		asignacionTemporalPrueba{tercero, "regla"},
		asignacionTemporalPrueba{segundo, "regla"},
		asignacionTemporalPrueba{primero, "regla"},
	)
	resultadoUno, err := resolverAplicacionesTemporales(plan, entrada, ordenUno)
	if err != nil {
		t.Fatal(err)
	}
	resultadoDos, err := resolverAplicacionesTemporales(plan, entrada, ordenDos)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(resultadoUno, resultadoDos) {
		t.Fatalf("la permutacion cambio el resultado:\n%#v\n%#v", resultadoUno, resultadoDos)
	}
}

func TestAplicacionesTemporalesEstabilizanParConEmpatesYAnidacion(t *testing.T) {
	plan := planTemporalPrueba(t, fechaTemporalPrueba(t, 2025, 12, 31), []reglaTemporalPrueba{
		{"regla", "grupo", reglasbaremo.ExtremoFinalExclusivo, 1},
	})
	largo := tramoTemporalPrueba(t, "empate-largo", "largo", periodoCerradoTemporalPrueba(
		t, fechaTemporalPrueba(t, 2025, 1, 1), fechaTemporalPrueba(t, 2025, 1, 20),
	))
	corto := tramoTemporalPrueba(t, "empate-corto", "corto", periodoCerradoTemporalPrueba(
		t, fechaTemporalPrueba(t, 2025, 1, 1), fechaTemporalPrueba(t, 2025, 1, 10),
	))
	anidado := tramoTemporalPrueba(t, "anidado", "anidado", periodoCerradoTemporalPrueba(
		t, fechaTemporalPrueba(t, 2025, 1, 5), fechaTemporalPrueba(t, 2025, 1, 6),
	))
	entrada := entradaTemporalPrueba(t, largo, corto, anidado)
	selecciones := []seleccionExperiencia{
		seleccionTemporalPrueba(t, plan,
			asignacionTemporalPrueba{largo, "regla"},
			asignacionTemporalPrueba{corto, "regla"},
			asignacionTemporalPrueba{anidado, "regla"},
		),
		seleccionTemporalPrueba(t, plan,
			asignacionTemporalPrueba{anidado, "regla"},
			asignacionTemporalPrueba{corto, "regla"},
			asignacionTemporalPrueba{largo, "regla"},
		),
	}
	esperadoPrimero, esperadoSegundo := ordenarParTramosTemporal(
		corto.Referencia(), largo.Referencia(),
	)
	for indice, seleccion := range selecciones {
		resultado, err := resolverAplicacionesTemporales(plan, entrada, seleccion)
		if err != nil || len(resultado.bloqueos) != 1 {
			t.Fatalf("orden %d: %#v, %v", indice, resultado, err)
		}
		bloqueo := resultado.bloqueos[0]
		if !referenciasTemporalesIguales(bloqueo.tramoPrimero, esperadoPrimero) ||
			!referenciasTemporalesIguales(bloqueo.tramoSegundo, esperadoSegundo) {
			t.Fatalf("orden %d produjo par inestable: %#v", indice, bloqueo)
		}
	}
}

func TestAplicacionesTemporalesCierranLimitesEIntegridad(t *testing.T) {
	t.Run("seleccion_bloqueada_no_avanza", func(t *testing.T) {
		seleccion := seleccionExperiencia{bloqueos: []bloqueoSeleccion{{
			codigo: bloqueoCoincidenciaRechazada,
		}}}
		_, err := resolverAplicacionesTemporales(PlanExperiencia{}, EntradaExperiencia{}, seleccion)
		if !errors.Is(err, ErrSeleccionTemporalBloqueada) {
			t.Fatalf("seleccion bloqueada: %v", err)
		}
	})

	corte := fechaTemporalPrueba(t, 2025, 1, 31)
	plan := planTemporalPrueba(t, corte, []reglaTemporalPrueba{
		{"regla", "grupo", reglasbaremo.ExtremoFinalInclusivo, 1},
	})
	periodo := periodoCerradoTemporalPrueba(
		t, fechaTemporalPrueba(t, 2025, 1, 1), fechaTemporalPrueba(t, 2025, 1, 10),
	)
	primero := tramoTemporalPrueba(t, "limite-a", "a", periodo)
	segundo := tramoTemporalPrueba(t, "limite-b", "b", periodo)
	entrada := entradaTemporalPrueba(t, primero, segundo)
	seleccion := seleccionTemporalPrueba(t, plan,
		asignacionTemporalPrueba{primero, "regla"},
		asignacionTemporalPrueba{segundo, "regla"},
	)

	t.Run("limite_aplicaciones", func(t *testing.T) {
		_, err := resolverAplicacionesTemporalesConLimites(
			plan, entrada, seleccion,
			limitesAplicacionesTemporales{aplicaciones: 1, eventos: 10},
		)
		if !errors.Is(err, ErrLimiteAplicacionesTemporales) {
			t.Fatalf("limite aplicaciones: %v", err)
		}
	})

	t.Run("limite_eventos", func(t *testing.T) {
		una := seleccionTemporalPrueba(t, plan, asignacionTemporalPrueba{primero, "regla"})
		_, err := resolverAplicacionesTemporalesConLimites(
			plan, entrada, una,
			limitesAplicacionesTemporales{aplicaciones: 1, eventos: 1},
		)
		if !errors.Is(err, ErrLimiteEventosTemporales) {
			t.Fatalf("limite eventos: %v", err)
		}
	})

	t.Run("identidad_exacta_del_tramo", func(t *testing.T) {
		base := primero.Referencia()
		alteraciones := []reglasbaremo.ReferenciaVersionada{
			referenciaTokenTemporalPrueba(t, prefijoTramoEntrada, "tramo-otra-ref", 1),
			debeTemporal(reglasbaremo.NuevaReferenciaVersionada(
				base.Referencia(), base.Version()+1, base.HuellaSHA256(),
			)),
			debeTemporal(reglasbaremo.NuevaReferenciaVersionada(
				base.Referencia(), base.Version(), strings.Repeat("f", 64),
			)),
		}
		for indice, referencia := range alteraciones {
			alterada := seleccionTemporalPrueba(
				t, plan, asignacionTemporalPrueba{primero, "regla"},
			)
			alterada.aplicaciones[0].tramo = referencia
			_, err := resolverAplicacionesTemporales(plan, entrada, alterada)
			if !errors.Is(err, ErrSeleccionTemporalInvalida) {
				t.Fatalf("alteracion %d admitida: %v", indice, err)
			}
		}
	})

	t.Run("limite_exacto_y_copias", func(t *testing.T) {
		una := seleccionTemporalPrueba(t, plan, asignacionTemporalPrueba{primero, "regla"})
		acotado, err := resolverAplicacionesTemporalesConLimites(
			plan, entrada, una,
			limitesAplicacionesTemporales{aplicaciones: 1, eventos: 2},
		)
		if err != nil || acotado.aplicacionesProcesadas != 1 || acotado.eventosProcesados != 2 {
			t.Fatalf("limite exacto: %#v, %v", acotado, err)
		}

		futuro := tramoTemporalPrueba(t, "copia-futuro", "futuro", periodoCerradoTemporalPrueba(
			t, fechaTemporalPrueba(t, 2025, 2, 1), fechaTemporalPrueba(t, 2025, 2, 3),
		))
		completo, err := resolverAplicacionesTemporales(
			plan,
			entradaTemporalPrueba(t, primero, segundo, futuro),
			seleccionTemporalPrueba(t, plan,
				asignacionTemporalPrueba{primero, "regla"},
				asignacionTemporalPrueba{segundo, "regla"},
				asignacionTemporalPrueba{futuro, "regla"},
			),
		)
		if err != nil || len(completo.aplicaciones) != 2 ||
			len(completo.exclusiones) != 1 || len(completo.bloqueos) != 1 {
			t.Fatalf("precondicion de copias: %#v, %v", completo, err)
		}
		aplicaciones := completo.aplicacionesCopia()
		exclusiones := completo.exclusionesCopia()
		bloqueos := completo.bloqueosCopia()
		aplicaciones[0] = aplicacionTemporal{}
		exclusiones[0] = exclusionAplicacionTemporal{}
		bloqueos[0] = bloqueoAplicacionesTemporales{}
		if !completo.aplicaciones[0].intervalo.EsValido() ||
			completo.exclusiones[0].razon == "" || completo.bloqueos[0].codigo == "" {
			t.Fatal("una copia externa altero el resultado temporal")
		}
	})
}
