package domain

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCriterioNoRecibePreferenciasNiDecideRenuncia(t *testing.T) {
	if _, existe := reflect.TypeOf(EvaluacionOfrecimiento{}).FieldByName("ModalidadesElegidas"); existe {
		t.Fatal("las preferencias no pueden ser entrada del evaluador")
	}
	if _, existe := reflect.TypeOf(DecisionOfrecimiento{}).FieldByName("RenunciaCuenta"); existe {
		t.Fatal("el evaluador no puede decidir una renuncia a partir de preferencias")
	}
	contenido, err := json.Marshal(EvaluacionOfrecimiento{})
	if err != nil || strings.Contains(string(contenido), "modalidades_elegidas") {
		t.Fatalf("contrato de evaluacion incluye preferencias: %s, %v", contenido, err)
	}
}

func TestEvaluarCriteriosOfrecimiento(t *testing.T) {
	fecha := func(a int, m time.Month, d int) time.Time { return time.Date(a, m, d, 0, 0, 0, 0, time.UTC) }
	verdad, falso := true, false
	inicio := fecha(2025, 1, 15)
	ahora := fecha(2025, 10, 15)
	finCorto := fecha(2025, 10, 14)
	finIgual := ahora
	politica := CriteriosOfrecimiento{
		FuenteNormativa: "Reglamento bolsa, art. 9.4",
		Mejora: &ReglaMejoraOfrecimiento{
			Habilitada:                     true,
			ModalidadesDestino:             []string{"vacante"},
			CausasDestino:                  []string{"vacante"},
			MaximoVacanteMeses:             36,
			ReduccionPorModalidadesPrevias: []string{"acumulacion", "sustitucion"},
			CondicionesActuales: []CondicionRelacionActual{
				{Modalidad: "acumulacion"},
				{Modalidad: "sustitucion", Causa: "baja", RequiereBajaComprobada: true,
					MesesTranscurridos: &LimiteMeses{Meses: 9, IncluyeLimite: true}},
				{Modalidad: "programa",
					MesesDuracionInicial: &LimiteMeses{Meses: 9, IncluyeLimite: false}},
			},
		},
	}
	base := EvaluacionOfrecimiento{Ahora: ahora, Situacion: SituacionDisponible, ModalidadOfrecida: "vacante", CausaOfrecida: "vacante"}
	acumulacion := RelacionActual{EventoRef: "evt_acumulacion", Modalidad: "acumulacion", Causa: "tareas", Inicio: inicio}
	sustitucion := RelacionActual{EventoRef: "evt_sustitucion", Modalidad: "sustitucion", Causa: "baja", Inicio: inicio, BajaInicio: &inicio, BajaEventoRef: "evt_baja", BajaComprobada: &verdad}
	programaCorto := RelacionActual{EventoRef: "evt_programa", Modalidad: "programa", Causa: "programa", Inicio: inicio, FinInicial: &finCorto}
	trabajando := base
	trabajando.Situacion = SituacionTrabajando
	trabajando.RelacionesActuales = []RelacionActual{acumulacion}

	casos := []struct {
		nombre    string
		politica  CriteriosOfrecimiento
		entrada   EvaluacionOfrecimiento
		motivo    string
		pendiente bool
	}{
		{"sin_configuracion_disponible", CriteriosOfrecimiento{}, base, MotivoOfrecimientoElegible, false},
		{"sin_configuracion_trabajando", CriteriosOfrecimiento{}, trabajando, MotivoOfrecimientoTrabajando, false},
		{"acumulacion_para_vacante", politica, trabajando, MotivoOfrecimientoPendienteComprobacion, true},
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
		{"causa_destino_no_es_vacante", politica, func() EvaluacionOfrecimiento { e := trabajando; e.CausaOfrecida = "necesidad_temporal"; return e }(), MotivoOfrecimientoTrabajando, false},
		{"causa_destino_desconocida", politica, func() EvaluacionOfrecimiento { e := trabajando; e.CausaOfrecida = ""; return e }(), MotivoOfrecimientoDatosOfertaIncompletos, true},
		{"sustitucion_baja_nueve_meses", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			e.RelacionesActuales = []RelacionActual{sustitucion}
			return e
		}(), MotivoOfrecimientoPendienteComprobacion, true},
		{"sustitucion_baja_supera_nueve", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			e.Ahora = fecha(2025, 10, 16)
			e.RelacionesActuales = []RelacionActual{sustitucion}
			return e
		}(), MotivoOfrecimientoTrabajando, false},
		{"baja_empezo_antes_de_la_sustitucion", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := sustitucion
			r.Inicio = fecha(2025, 2, 15)
			e.Ahora = fecha(2025, 10, 16)
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoTrabajando, false},
		{"inicio_baja_no_acreditado", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := sustitucion
			r.BajaInicio = nil
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoDatosRelacionIncompletos, true},
		{"baja_sin_evento", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := sustitucion
			r.BajaEventoRef = ""
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoDatosRelacionIncompletos, true},
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
		}(), MotivoOfrecimientoDatosRelacionIncompletos, true},
		{"programa_inicial_menor_nueve", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := programaCorto
			r.Causa = ""
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoPendienteComprobacion, true},
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
		}(), MotivoOfrecimientoDatosRelacionIncompletos, true},
		{"programa_fin_inicial_invalido", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := programaCorto
			fin := inicio
			r.FinInicial = &fin
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoDatosRelacionIncompletos, true},
		{"acumulacion_sin_causa_no_exigida", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := acumulacion
			r.Causa = ""
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoPendienteComprobacion, true},
		{"sustitucion_sin_causa_exigida", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := sustitucion
			r.Causa = ""
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoDatosRelacionIncompletos, true},
		{"ninguna_relacion", politica, func() EvaluacionOfrecimiento { e := trabajando; e.RelacionesActuales = nil; return e }(), MotivoOfrecimientoDatosRelacionIncompletos, true},
		{"relacion_sin_evento", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := acumulacion
			r.EventoRef = ""
			e.RelacionesActuales = []RelacionActual{r}
			return e
		}(), MotivoOfrecimientoDatosRelacionIncompletos, true},
		{"hecho_incompleto_tras_otro_no_admitido", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			r := sustitucion
			r.Causa = "vacaciones"
			e.RelacionesActuales = []RelacionActual{r, {Modalidad: "programa"}}
			return e
		}(), MotivoOfrecimientoDatosRelacionIncompletos, true},
		{"todas_las_relaciones_admitidas", politica, func() EvaluacionOfrecimiento {
			e := trabajando
			e.RelacionesActuales = []RelacionActual{acumulacion, sustitucion}
			return e
		}(), MotivoOfrecimientoPendienteComprobacion, true},
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
		{"sin_preferencias_en_entrada", politica, base, MotivoOfrecimientoElegible, false},
		{"sin_filtro_de_modalidad_individual", politica, func() EvaluacionOfrecimiento { e := base; e.ModalidadOfrecida = "otra_modalidad"; return e }(), MotivoOfrecimientoElegible, false},
		{"oferta_sin_modalidad", politica, func() EvaluacionOfrecimiento { e := base; e.ModalidadOfrecida = ""; return e }(), MotivoOfrecimientoDatosOfertaIncompletos, true},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			decision := EvaluarCriteriosOfrecimiento(tc.politica, tc.entrada)
			if decision.Motivo != tc.motivo || decision.Elegible != (tc.motivo == MotivoOfrecimientoElegible) || decision.PendienteComprobacion != tc.pendiente {
				t.Fatalf("decision = %+v; motivo esperado %q, pendiente %t", decision, tc.motivo, tc.pendiente)
			}
		})
	}

	decision := EvaluarCriteriosOfrecimiento(politica, trabajando)
	if decision.MaximoVacanteMeses != 36 || len(decision.PeriodosReduccionVacante) != 1 ||
		decision.PeriodosReduccionVacante[0] != (PeriodoReduccionVacante{EventoRef: "evt_acumulacion", Desde: inicio, Hasta: ahora}) {
		t.Fatalf("la reduccion debe conservar periodo y evento comprobado: %+v", decision)
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
		{FuenteNormativa: "art. 9.4", Mejora: &ReglaMejoraOfrecimiento{ModalidadesDestino: []string{"vacante"}}},
		{FuenteNormativa: "art. 9.4", Mejora: &ReglaMejoraOfrecimiento{ModalidadesDestino: []string{"vacante"}, CondicionesActuales: []CondicionRelacionActual{{Modalidad: "programa", MesesDuracionInicial: &LimiteMeses{}}}, MaximoVacanteMeses: 36, ReduccionPorModalidadesPrevias: []string{"programa"}}},
		{FuenteNormativa: "art. 9.4", Mejora: &ReglaMejoraOfrecimiento{ModalidadesDestino: []string{"vacante"}, CondicionesActuales: []CondicionRelacionActual{{Modalidad: "programa", MesesDuracionInicial: &LimiteMeses{Meses: 1201}}}, MaximoVacanteMeses: 36, ReduccionPorModalidadesPrevias: []string{"programa"}}},
		{FuenteNormativa: "art. 9.4", Mejora: &ReglaMejoraOfrecimiento{ModalidadesDestino: []string{"VACANTE"}, CausasDestino: []string{"vacante"}, CondicionesActuales: []CondicionRelacionActual{{Modalidad: "programa"}}, MaximoVacanteMeses: 36, ReduccionPorModalidadesPrevias: []string{"programa"}}},
		{FuenteNormativa: "art. 9.4", Mejora: &ReglaMejoraOfrecimiento{ModalidadesDestino: []string{"vacante"}, CondicionesActuales: []CondicionRelacionActual{{Modalidad: "programa"}}, MaximoVacanteMeses: 36, ReduccionPorModalidadesPrevias: []string{"otra"}}},
	}
	for i, criterio := range casos {
		if err := criterio.Validar(); err == nil {
			t.Fatalf("caso %d: se admitio una configuracion incompleta", i)
		}
	}
}
