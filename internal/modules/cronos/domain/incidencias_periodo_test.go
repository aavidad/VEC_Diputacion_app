package domain

import (
	"testing"
	"time"
)

func instanteIncidenciasTest(s string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(err)
	}
	return t
}
func consultaIncidenciasTest() ConsultaIncidenciasPeriodo {
	return ConsultaIncidenciasPeriodo{Desde: "2026-10-24", Hasta: "2026-10-27", Zona: "Europe/Madrid", CorteUTC: instanteIncidenciasTest("2026-10-25T12:00:00Z"), AntecedentesCompletos: true, Cobertura: []CoberturaRegistro{{"2026-10-24", CoberturaRegistroCompleta}, {"2026-10-25", CoberturaRegistroCompleta}, {"2026-10-26", CoberturaRegistroCompleta}}, Hechos: []HechoRegistroIncidencia{{"demo:registro:entrada", 1, PunchEntry, instanteIncidenciasTest("2026-10-24T20:00:00Z")}}}
}

func TestIncidenciasAbiertaSeDetieneEnCorteDuranteDST(t *testing.T) {
	q := consultaIncidenciasTest()
	dias, err := ConsultarIncidenciasPeriodo(q)
	if err != nil {
		t.Fatal(err)
	}
	if len(dias) != 3 || dias[0].Estado != "registro_incompleto" || dias[1].Estado != "registro_incompleto" || dias[2].Estado != "no_evaluado" || dias[2].HechosRegistrados != nil || dias[2].EvaluadoHastaUTC != nil {
		t.Fatalf("unexpected evaluation: %+v", dias)
	}
	if !dias[1].EvaluadoHastaUTC.Equal(q.CorteUTC) {
		t.Fatal("open sequence extended beyond cutoff")
	}
	if !dias[0].EvaluadoHastaUTC.Equal(instanteIncidenciasTest("2026-10-24T22:00:00Z")) {
		t.Fatal("wrong civil day boundary before DST")
	}
}

func TestIncidenciasTurnoQueCruzaPeriodoYDiaDST(t *testing.T) {
	q := consultaIncidenciasTest()
	q.Desde = "2026-10-25"
	q.Hasta = "2026-10-26"
	q.CorteUTC = instanteIncidenciasTest("2026-10-26T12:00:00Z")
	q.Cobertura = []CoberturaRegistro{{"2026-10-25", CoberturaRegistroCompleta}}
	q.Hechos = append(q.Hechos, HechoRegistroIncidencia{"demo:registro:salida", 1, PunchExit, instanteIncidenciasTest("2026-10-25T04:00:00Z")})
	dias, err := ConsultarIncidenciasPeriodo(q)
	if err != nil {
		t.Fatal(err)
	}
	if dias[0].Estado != "registrado" || *dias[0].HechosRegistrados != 1 || !dias[0].EvaluadoHastaUTC.Equal(instanteIncidenciasTest("2026-10-25T23:00:00Z")) {
		t.Fatalf("wrong midnight or antecedent: %+v", dias[0])
	}
}

func TestIncidenciasFuenteIncompletaNuncaDaCeroSaludable(t *testing.T) {
	for _, estado := range []string{"", CoberturaRegistroIncompleta, CoberturaRegistroNoDisponible} {
		t.Run(estado, func(t *testing.T) {
			q := consultaIncidenciasTest()
			q.Hechos = nil
			q.Cobertura = q.Cobertura[1:]
			if estado != "" {
				q.Cobertura = append(q.Cobertura, CoberturaRegistro{"2026-10-24", estado})
			}
			dias, err := ConsultarIncidenciasPeriodo(q)
			if err != nil {
				t.Fatal(err)
			}
			for _, d := range dias[:2] {
				if d.Estado != "indeterminado" || d.HechosRegistrados != nil {
					t.Fatalf("healthy unknown source: %+v", d)
				}
			}
		})
	}
}

func TestIncidenciasSinRegistrosNoEsAusencia(t *testing.T) {
	q := consultaIncidenciasTest()
	q.Hechos = nil
	dias, err := ConsultarIncidenciasPeriodo(q)
	if err != nil {
		t.Fatal(err)
	}
	if dias[0].Estado != "sin_registros" || dias[1].Estado != "sin_registros" || *dias[0].HechosRegistrados != 0 {
		t.Fatalf("unexpected empty source: %+v", dias)
	}
	q.AntecedentesCompletos = false
	dias, err = ConsultarIncidenciasPeriodo(q)
	if err != nil {
		t.Fatal(err)
	}
	if dias[0].Estado != "indeterminado" || dias[0].HechosRegistrados != nil {
		t.Fatal("unknown antecedents became healthy")
	}
}

func TestIncidenciasEntradaSalidaYSalidaSolaUsanSoloMotor(t *testing.T) {
	q := consultaIncidenciasTest()
	q.CorteUTC = instanteIncidenciasTest("2026-10-27T12:00:00Z")
	q.Hechos = append(q.Hechos, HechoRegistroIncidencia{"demo:salida:uno", 1, PunchExit, instanteIncidenciasTest("2026-10-24T21:00:00Z")}, HechoRegistroIncidencia{"demo:salida:dos", 1, PunchExit, instanteIncidenciasTest("2026-10-25T10:00:00Z")})
	dias, err := ConsultarIncidenciasPeriodo(q)
	if err != nil {
		t.Fatal(err)
	}
	if dias[0].Estado != "registrado" || dias[1].Estado != "registro_incompleto" || dias[2].Estado != "sin_registros" {
		t.Fatalf("unexpected states: %+v", dias)
	}
}

func TestIncidenciasRechazaAmbiguedadesYFueraContrato(t *testing.T) {
	cases := map[string]func(*ConsultaIncidenciasPeriodo){
		"timestamp repeated": func(q *ConsultaIncidenciasPeriodo) {
			q.Hechos = append(q.Hechos, HechoRegistroIncidencia{"demo:otro", 1, PunchExit, q.Hechos[0].InstanteUTC})
		},
		"reference repeated": func(q *ConsultaIncidenciasPeriodo) {
			q.Hechos = append(q.Hechos, HechoRegistroIncidencia{q.Hechos[0].Ref, 1, PunchExit, q.Hechos[0].InstanteUTC.Add(time.Hour)})
		},
		"at cutoff":    func(q *ConsultaIncidenciasPeriodo) { q.Hechos[0].InstanteUTC = q.CorteUTC },
		"after cutoff": func(q *ConsultaIncidenciasPeriodo) { q.Hechos[0].InstanteUTC = q.CorteUTC.Add(time.Microsecond) },
		"non UTC fact": func(q *ConsultaIncidenciasPeriodo) {
			q.Hechos[0].InstanteUTC = q.Hechos[0].InstanteUTC.In(time.FixedZone("offset", 3600))
		},
		"non UTC cutoff": func(q *ConsultaIncidenciasPeriodo) { q.CorteUTC = q.CorteUTC.In(time.FixedZone("offset", 3600)) },
		"nanosecond": func(q *ConsultaIncidenciasPeriodo) {
			q.Hechos[0].InstanteUTC = q.Hechos[0].InstanteUTC.Add(time.Nanosecond)
		},
		"unknown movement":   func(q *ConsultaIncidenciasPeriodo) { q.Hechos[0].Movimiento = "otro" },
		"duplicate coverage": func(q *ConsultaIncidenciasPeriodo) { q.Cobertura = append(q.Cobertura, q.Cobertura[0]) },
		"unknown coverage":   func(q *ConsultaIncidenciasPeriodo) { q.Cobertura[0].Estado = "healthy" },
		"outside coverage":   func(q *ConsultaIncidenciasPeriodo) { q.Cobertura[0].Fecha = q.Hasta },
		"invalid date":       func(q *ConsultaIncidenciasPeriodo) { q.Desde = "2026-02-30" },
		"local zone":         func(q *ConsultaIncidenciasPeriodo) { q.Zona = "Local" },
		"unknown zone":       func(q *ConsultaIncidenciasPeriodo) { q.Zona = "Missing/Zone" },
		"overlong period":    func(q *ConsultaIncidenciasPeriodo) { q.Hasta = "2028-10-26" },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			q := consultaIncidenciasTest()
			mutate(&q)
			if r, err := ConsultarIncidenciasPeriodo(q); err == nil || r != nil {
				t.Fatalf("accepted %s: %+v", name, r)
			}
		})
	}
}

func TestIncidenciasAceptaZonaDeclaradaYSoloFuturo(t *testing.T) {
	q := consultaIncidenciasTest()
	q.Zona = "UTC"
	q.CorteUTC = instanteIncidenciasTest("2026-10-23T00:00:00Z")
	q.Hechos = nil
	dias, err := ConsultarIncidenciasPeriodo(q)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dias {
		if d.Estado != "no_evaluado" || d.EvaluadoHastaUTC != nil || d.HechosRegistrados != nil {
			t.Fatalf("future evaluated: %+v", d)
		}
	}
}

func TestIncidenciasNoCierraEntradaConContextoFueraPeriodo(t *testing.T) {
	q := consultaIncidenciasTest()
	q.Desde = "2026-09-28"
	q.Hasta = "2026-09-29"
	q.CorteUTC = instanteIncidenciasTest("2026-09-30T12:00:00Z")
	q.Cobertura = []CoberturaRegistro{{"2026-09-28", CoberturaRegistroCompleta}}
	q.Hechos = []HechoRegistroIncidencia{{"demo:entrada:28", 1, PunchEntry, instanteIncidenciasTest("2026-09-28T06:00:00Z")}, {"demo:salida:29", 1, PunchExit, instanteIncidenciasTest("2026-09-29T06:00:00Z")}}
	if r, err := ConsultarIncidenciasPeriodo(q); err == nil || r != nil {
		t.Fatalf("out of period context closed source: %+v", r)
	}
}
func TestIncidenciasGapActualNoPuedeCerrarEntradaPreviaComoSana(t *testing.T) {
	q := consultaIncidenciasTest()
	q.Cobertura[1].Estado = CoberturaRegistroNoDisponible
	q.Hechos = append(q.Hechos, HechoRegistroIncidencia{"demo:salida:gap", 1, PunchExit, instanteIncidenciasTest("2026-10-25T04:00:00Z")})
	dias, err := ConsultarIncidenciasPeriodo(q)
	if err != nil {
		t.Fatal(err)
	}
	if dias[0].Estado != "indeterminado" || dias[0].HechosRegistrados != nil || dias[1].Estado != "indeterminado" {
		t.Fatalf("gap closed previous source as healthy: %+v", dias)
	}
}
func TestIncidenciasGapFuturoNoInvalidaFechaEvaluada(t *testing.T) {
	q := consultaIncidenciasTest()
	q.Cobertura[2].Estado = CoberturaRegistroNoDisponible
	q.Hechos = nil
	dias, err := ConsultarIncidenciasPeriodo(q)
	if err != nil {
		t.Fatal(err)
	}
	if dias[0].Estado != "sin_registros" || dias[1].Estado != "sin_registros" || dias[2].Estado != "no_evaluado" {
		t.Fatalf("future source gap changed past result: %+v", dias)
	}
}
