package domain

import (
	"testing"
	"time"
)

func TestEvaluarCriteriosOfrecimiento(t *testing.T) {
	fecha := func(a int, m time.Month, d int) time.Time { return time.Date(a, m, d, 0, 0, 0, 0, time.UTC) }
	verdad, falso := true, false
	inicio := fecha(2025, 1, 15)
	ahora := fecha(2025, 10, 15)
	finCorto := fecha(2025, 10, 14)
	finIgual := ahora
	seleccionVacia := []string{}
	seleccionVacante := []string{"vacante"}
	politica := CriteriosOfrecimiento{
		FuenteNormativa:       "Reglamento bolsa, art. 9.4",
		ModalidadesAceptables: []string{"vacante", "sustitucion", "programa"},
		ModalidadNoAceptada:   RechazoModalidadSinRenuncia,
		Mejora: &ReglaMejoraOfrecimiento{
			Habilitada:         true,
			ModalidadesDestino: []string{"vacante"},
			CondicionesActuales: []CondicionRelacionActual{
				{Modalidad: "acumulacion"},
				{Modalidad: "sustitucion", Causa: "baja", RequiereBajaComprobada: true,
					MesesTranscurridos: &LimiteMeses{Meses: 9, IncluyeLimite: true}},
				{Modalidad: "programa",
					MesesDuracionInicial: &LimiteMeses{Meses: 9, IncluyeLimite: false}},
			},
		},
	}
	base := EvaluacionOfrecimiento{Ahora: ahora, Situacion: SituacionDisponible, ModalidadOfrecida: "vacante"}
	acumulacion := RelacionActual{Modalidad: "acumulacion", Causa: "tareas", Inicio: inicio}
	sustitucion := RelacionActual{Modalidad: "sustitucion", Causa: "baja", Inicio: inicio, BajaComprobada: &verdad}
	programaCorto := RelacionActual{Modalidad: "programa", Causa: "programa", Inicio: inicio, FinInicial: &finCorto}
	trabajando := base
	trabajando.Situacion = SituacionTrabajando
	trabajando.RelacionesActuales = []RelacionActual{acumulacion}

	casos := []struct {
		nombre   string
		politica CriteriosOfrecimiento
		entrada  EvaluacionOfrecimiento
		motivo   string
		renuncia bool
	}{
		{"sin_configuracion_disponible", CriteriosOfrecimiento{}, base, MotivoOfrecimientoElegible, false},
		{"sin_configuracion_preferencia_vacia", CriteriosOfrecimiento{}, func() EvaluacionOfrecimiento { e := base; e.ModalidadesElegidas = &seleccionVacia; return e }(), MotivoOfrecimientoModalidadNoElegida, false},
		{"sin_configuracion_trabajando", CriteriosOfrecimiento{}, trabajando, MotivoOfrecimientoTrabajando, false},
		{"acumulacion_para_vacante", politica, trabajando, MotivoOfrecimientoElegible, false},
		{"mejora_desactivada", func() CriteriosOfrecimiento {
			p := politica
			m := *p.Mejora
			m.Habilitada = false
			p.Mejora = &m
			return p
		}(), trabajando, MotivoOfrecimientoTrabajando, false},
		{"restriccion", politica, func() EvaluacionOfrecimiento { e := trabajando; e.RestriccionActiva = true; return e }(), MotivoOfrecimientoRestriccionActiva, false},
		{"cese_pendiente", politica, func() EvaluacionOfrecimiento { e := trabajando; e.CesePendiente = true; return e }(), MotivoOfrecimientoCesePendiente, false},
		{"respuesta_pendiente", politica, func() EvaluacionOfrecimiento { e := trabajando; e.RespuestaPendiente = true; return e }(), MotivoOfrecimientoRespuestaPendiente, false},
		{"otra_oferta_no_es_mejora", politica, func() EvaluacionOfrecimiento { e := trabajando; e.ModalidadOfrecida = "sustitucion"; return e }(), MotivoOfrecimientoTrabajando, false},
		{"sustitucion_baja_nueve_meses", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			e.RelacionesActuales = []RelacionActual{sustitucion}
			return e
		}(), MotivoOfrecimientoElegible, false},
		{"sustitucion_baja_supera_nueve", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			e.Ahora = fecha(2025, 10, 16)
			e.RelacionesActuales = []RelacionActual{sustitucion}
			return e
		}(), MotivoOfrecimientoTrabajando, false},
		{"sustitucion_otra_causa", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := sustitucion
			r.Causa = "vacaciones"
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoTrabajando, false},
		{"baja_comprobada_negativa", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := sustitucion
			r.BajaComprobada = &falso
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoTrabajando, false},
		{"baja_sin_comprobar", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := sustitucion
			r.BajaComprobada = nil
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoDatosRelacionIncompletos, false},
		{"programa_inicial_menor_nueve", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := programaCorto
			r.Causa = ""
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoElegible, false},
		{"programa_inicial_igual_nueve", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := programaCorto
			r.FinInicial = &finIgual
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoTrabajando, false},
		{"programa_sin_fin_inicial", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := programaCorto
			r.FinInicial = nil
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoDatosRelacionIncompletos, false},
		{"programa_fin_inicial_invalido", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := programaCorto
			fin := inicio
			r.FinInicial = &fin
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoDatosRelacionIncompletos, false},
		{"acumulacion_sin_causa_no_exigida", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := acumulacion
			r.Causa = ""
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoElegible, false},
		{"sustitucion_sin_causa_exigida", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := sustitucion
			r.Causa = ""
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoDatosRelacionIncompletos, false},
		{"ninguna_relacion", politica, func() EvaluacionOfrecimiento { e := trabajando; e.RelacionesActuales = nil; return e }(), MotivoOfrecimientoDatosRelacionIncompletos, false},
		{"todas_las_relaciones_admitidas", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			e.RelacionesActuales = []RelacionActual{acumulacion, sustitucion}
			return e
		}(), MotivoOfrecimientoElegible, false},
		{"una_relacion_no_admitida", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := sustitucion
			r.Causa = "vacaciones"
			e.RelacionesActuales = []RelacionActual{acumulacion, r}
			return e
		}(), MotivoOfrecimientoTrabajando, false},
		{"disponible_desde_antes", politica, func() EvaluacionOfrecimiento {
			e := base
			e.Situacion = SituacionDisponibleDesde
			e.DisponibleDesde = &finIgual
			e.Ahora = fecha(2025, 10, 14)
			return e
		}(), MotivoOfrecimientoDisponibleDesde, false},
		{"disponible_desde_igual", politica, func() EvaluacionOfrecimiento {
			e := base
			e.Situacion = SituacionDisponibleDesde
			e.DisponibleDesde = &finIgual
			return e
		}(), MotivoOfrecimientoElegible, false},
		{"situacion_excluida", politica, func() EvaluacionOfrecimiento { e := base; e.Situacion = SituacionExcluido; return e }(), MotivoOfrecimientoSituacionNoDisponible, false},
		{"preferencia_nil_todas", politica, base, MotivoOfrecimientoElegible, false},
		{"preferencia_vacia_ninguna", politica, func() EvaluacionOfrecimiento { e := base; e.ModalidadesElegidas = &seleccionVacia; return e }(), MotivoOfrecimientoModalidadNoElegida, false},
		{"preferencia_ajena", politica, func() EvaluacionOfrecimiento {
			e := base
			e.ModalidadOfrecida = "programa"
			e.ModalidadesElegidas = &seleccionVacante
			return e
		}(), MotivoOfrecimientoModalidadNoElegida, false},
		{"modalidad_fuera_de_bases", politica, func() EvaluacionOfrecimiento { e := base; e.ModalidadOfrecida = "desconocida"; return e }(), MotivoOfrecimientoModalidadNoAdmitida, false},
		{"oferta_sin_modalidad", politica, func() EvaluacionOfrecimiento { e := base; e.ModalidadOfrecida = ""; return e }(), MotivoOfrecimientoDatosOfertaIncompletos, false},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			decision := EvaluarCriteriosOfrecimiento(tc.politica, tc.entrada)
			if decision.Motivo != tc.motivo || decision.Elegible != (tc.motivo == MotivoOfrecimientoElegible) || decision.RenunciaCuenta != tc.renuncia {
				t.Fatalf("decision = %+v; motivo esperado %q, renuncia %t", decision, tc.motivo, tc.renuncia)
			}
		})
	}

	renuncia := politica
	renuncia.ModalidadNoAceptada = RechazoModalidadConRenuncia
	entrada := base
	entrada.ModalidadesElegidas = &seleccionVacia
	if decision := EvaluarCriteriosOfrecimiento(renuncia, entrada); decision.Motivo != MotivoOfrecimientoModalidadNoElegida || !decision.RenunciaCuenta {
		t.Fatalf("la politica de renuncia debe constar solo en modalidad rechazada: %+v", decision)
	}
}

func TestLimiteMesesUsaUltimoDiaDelMes(t *testing.T) {
	inicio := time.Date(2025, time.May, 31, 0, 0, 0, 0, time.UTC)
	ultimoDia := time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC)
	limite := &LimiteMeses{Meses: 9, IncluyeLimite: true}
	if !cumpleMeses(inicio, ultimoDia, limite) || cumpleMeses(inicio, ultimoDia.AddDate(0, 0, 1), limite) {
		t.Fatal("el limite de nueve meses debe caer el ultimo dia de febrero")
	}
	limite.IncluyeLimite = false
	if cumpleMeses(inicio, ultimoDia, limite) {
		t.Fatal("la duracion inicial de nueve meses exactos no es menor de nueve")
	}
}

func TestCriteriosOfrecimientoValidar(t *testing.T) {
	if err := (CriteriosOfrecimiento{}).Validar(); err != nil {
		t.Fatal(err)
	}
	casos := []CriteriosOfrecimiento{
		{Mejora: &ReglaMejoraOfrecimiento{ModalidadesDestino: []string{"vacante"}, CondicionesActuales: []CondicionRelacionActual{{Modalidad: "acumulacion"}}}},
		{FuenteNormativa: "art. 9.4", ModalidadNoAceptada: "otro"},
		{FuenteNormativa: "art. 9.4", ModalidadNoAceptada: RechazoModalidadSinRenuncia, ModalidadesAceptables: []string{"vacante", "vacante"}},
		{FuenteNormativa: "art. 9.4", ModalidadNoAceptada: RechazoModalidadSinRenuncia, Mejora: &ReglaMejoraOfrecimiento{ModalidadesDestino: []string{"vacante"}}},
		{FuenteNormativa: "art. 9.4", ModalidadNoAceptada: RechazoModalidadSinRenuncia, Mejora: &ReglaMejoraOfrecimiento{ModalidadesDestino: []string{"vacante"}, CondicionesActuales: []CondicionRelacionActual{{Modalidad: "programa", MesesDuracionInicial: &LimiteMeses{}}}}},
		{FuenteNormativa: "art. 9.4", ModalidadNoAceptada: RechazoModalidadSinRenuncia, Mejora: &ReglaMejoraOfrecimiento{ModalidadesDestino: []string{"vacante"}, CondicionesActuales: []CondicionRelacionActual{{Modalidad: "programa", MesesDuracionInicial: &LimiteMeses{Meses: 1201}}}}},
		{FuenteNormativa: "art. 9.4", ModalidadNoAceptada: RechazoModalidadSinRenuncia, ModalidadesAceptables: []string{"VACANTE"}},
		{FuenteNormativa: "art. 9.4", ModalidadNoAceptada: RechazoModalidadSinRenuncia, ModalidadesAceptables: []string{"vacante"}, Mejora: &ReglaMejoraOfrecimiento{ModalidadesDestino: []string{"otra"}, CondicionesActuales: []CondicionRelacionActual{{Modalidad: "programa"}}}},
	}
	for i, criterio := range casos {
		if err := criterio.Validar(); err == nil {
			t.Fatalf("caso %d: se admitio una configuracion incompleta", i)
		}
	}
}
