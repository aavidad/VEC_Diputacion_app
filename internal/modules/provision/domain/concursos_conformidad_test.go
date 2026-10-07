package domain_test

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	d "vec-diputacion-granada/internal/modules/provision/domain"
	b "vec-diputacion-granada/internal/shared/baremacion"
)

// Estos vectores usan reglas hipotéticas y hechos sintéticos. No transcriben
// bases aprobadas ni acreditan una valoración administrativa real.
func concPuntos(t *testing.T, micro int64) b.Puntos {
	t.Helper()
	p, err := b.PuntosDesdeMicropuntos(micro)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func concFecha(t *testing.T, fecha string) b.FechaCivil {
	t.Helper()
	var f b.FechaCivil
	if err := json.Unmarshal([]byte(`"`+fecha+`"`), &f); err != nil {
		t.Fatal(err)
	}
	return f
}

func concFin(t *testing.T, fecha string) *b.FechaCivil {
	f := concFecha(t, fecha)
	return &f
}

func concRacional(t *testing.T, n, den int64) b.Racional {
	t.Helper()
	r, err := b.NuevoRacional(n, den)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func concJornada(t *testing.T, n, den int64) b.FraccionJornada {
	t.Helper()
	j, err := b.NuevaFraccionJornada(n, den)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func concEntrada() d.Entrada {
	return d.Entrada{
		InstantaneaRef: "instantanea:sintetica:v1", PuestoRef: "puesto:sintetico:1", NivelPuesto: 20,
		Disponibles: []d.Familia{d.Grado, d.Antiguedad, d.Permanencia, d.Cursos, d.Titulaciones, d.ValoracionTrabajo},
		Periodos:    []d.Periodo{}, Cursos: []d.Curso{}, Titulaciones: []d.Titulo{},
	}
}

func concRegla(t *testing.T, familia d.Familia) d.Regla {
	t.Helper()
	r := d.Regla{ID: "regla:" + string(familia), Familia: familia, ReferenciaBase: "base:sintetica:apartado",
		Coeficiente: concPuntos(t, 1_000_000), Maximo: concPuntos(t, 100_000_000), Redondeo: b.RedondeoExacto}
	if familia == d.Antiguedad || familia == d.Permanencia || familia == d.ValoracionTrabajo {
		r.Conversion = &d.Conversion{Metodo: "dias_racionales", Divisor: 1}
		r.Agrupacion, r.Jornada, r.Solapes = "por_tramo", "proporcional", "rechazar"
	}
	if familia == d.Cursos {
		h := concRacional(t, 5, 1)
		r.HorasMinimas = &h
		r.Tipos = []string{"formacion:admitida"}
	}
	if familia == d.Grado || familia == d.ValoracionTrabajo {
		r.Diferencia, r.MinDiferencia, r.MaxDiferencia = "puesto_menos_hecho", -3, 3
		r.Tramos = []d.Tramo{
			{ID: "tramo:superior", MinDiferencia: -3, MaxDiferencia: -1, Coeficiente: concPuntos(t, 3_000_000), Maximo: r.Maximo},
			{ID: "tramo:igual", MinDiferencia: 0, MaxDiferencia: 0, Coeficiente: concPuntos(t, 2_000_000), Maximo: r.Maximo},
			{ID: "tramo:inferior", MinDiferencia: 1, MaxDiferencia: 3, Coeficiente: concPuntos(t, 1_000_000), Maximo: r.Maximo},
		}
	}
	return r
}

func concConfig(t *testing.T, reglas ...d.Regla) d.Configuracion {
	t.Helper()
	return d.Configuracion{SchemaVersion: d.VersionMotor, ConvocatoriaRef: "concurso:sintetico:1", Version: "reglas:v1",
		BasesRef: "bases:sinteticas:v1", VentanaDesde: concFecha(t, "2024-01-01"), FechaCorte: concFecha(t, "2025-01-01"),
		MaximoTotal: concPuntos(t, 100_000_000), Reglas: reglas}
}

func concPeriodo(t *testing.T, id, desde, hasta string, familia d.Familia) d.Periodo {
	t.Helper()
	p := d.Periodo{ID: id, EvidenciaRef: "evidencia:" + id, Desde: concFecha(t, desde), Tipo: "servicio:sintetico",
		Familias: []d.Familia{familia}, Nivel: 20, Jornada: b.JornadaCompleta()}
	if hasta != "" {
		p.Hasta = concFin(t, hasta)
	}
	return p
}

func concCalcular(t *testing.T, c d.Configuracion, e d.Entrada) d.Resultado {
	t.Helper()
	r, err := d.Calcular(c, e)
	if err != nil {
		t.Fatalf("Calcular: %v", err)
	}
	return r
}

func concTotal(t *testing.T, r d.Resultado, esperado int64) {
	t.Helper()
	if !r.Completo || r.Total == nil || len(r.Incidencias) != 0 {
		t.Fatalf("resultado incompleto: completo=%v total=%v incidencias=%v", r.Completo, r.Total, r.Incidencias)
	}
	if r.Total.Micropuntos() != esperado {
		t.Fatalf("total=%d, esperado=%d micropuntos", r.Total.Micropuntos(), esperado)
	}
}

func concDesglose(t *testing.T, r d.Resultado, familia d.Familia) d.Desglose {
	t.Helper()
	for _, parte := range r.Desglose {
		if parte.Familia == familia {
			return parte
		}
	}
	t.Fatalf("falta desglose de %s", familia)
	return d.Desglose{}
}

func TestConcursosConformidadGradoYExperienciaPorNivel(t *testing.T) {
	for _, familia := range []d.Familia{d.Grado, d.ValoracionTrabajo} {
		for _, caso := range []struct {
			nombre string
			nivel  int
			puntos int64
			tramo  string
		}{
			{"limite_superior", 23, 3_000_000, "tramo:superior"},
			{"superior", 21, 3_000_000, "tramo:superior"},
			{"igual", 20, 2_000_000, "tramo:igual"},
			{"inferior", 19, 1_000_000, "tramo:inferior"},
			{"limite_inferior", 17, 1_000_000, "tramo:inferior"},
		} {
			t.Run(string(familia)+"/"+caso.nombre, func(t *testing.T) {
				e := concEntrada()
				if familia == d.Grado {
					grado := caso.nivel
					e.GradoPersonal, e.GradoEvidenciaRef = &grado, "evidencia:grado:sintetico"
				} else {
					p := concPeriodo(t, "periodo:nivel", "2024-02-28", "2024-02-29", familia)
					p.Nivel = caso.nivel
					e.Periodos = []d.Periodo{p}
				}
				r := concCalcular(t, concConfig(t, concRegla(t, familia)), e)
				concTotal(t, r, caso.puntos)
				parte := concDesglose(t, r, familia)
				encontrado := false
				for _, detalle := range parte.Detalles {
					if detalle.EvidenciaRef != "" {
						encontrado = true
						if detalle.TramoID != caso.tramo {
							t.Fatalf("tramo incorrecto para el hecho: %+v", detalle)
						}
					}
				}
				if !encontrado {
					t.Fatalf("falta evidencia del hecho: %+v", parte.Detalles)
				}
			})
		}
	}
	t.Run("sentido_explicito", func(t *testing.T) {
		r := concRegla(t, d.Grado)
		r.Diferencia = "hecho_menos_puesto"
		e := concEntrada()
		grado := 23
		e.GradoPersonal, e.GradoEvidenciaRef = &grado, "evidencia:grado:sintetico"
		concTotal(t, concCalcular(t, concConfig(t, r), e), 1_000_000)
	})
	t.Run("grado_rectificado_conserva_calculo_anterior", func(t *testing.T) {
		c := concConfig(t, concRegla(t, d.Grado))
		e := concEntrada()
		grado := 19
		e.GradoPersonal, e.GradoEvidenciaRef = &grado, "evidencia:grado:v1"
		anterior := concCalcular(t, c, e)
		concTotal(t, anterior, 1_000_000)
		rectificada := e
		otroGrado := 21
		rectificada.InstantaneaRef, rectificada.GradoPersonal, rectificada.GradoEvidenciaRef = "instantanea:sintetica:v2", &otroGrado, "evidencia:grado:v2"
		nuevo := concCalcular(t, c, rectificada)
		concTotal(t, nuevo, 3_000_000)
		if anterior.HuellaReglas != nuevo.HuellaReglas || anterior.HuellaEntrada == nuevo.HuellaEntrada || anterior.HuellaResultado == nuevo.HuellaResultado {
			t.Fatal("rectificación del grado sin causalidad en las huellas")
		}
		if !reflect.DeepEqual(anterior, concCalcular(t, c, e)) {
			t.Fatal("la rectificación alteró la reproducción histórica")
		}
	})
}

func TestConcursosConformidadTiempoCivilYJornada(t *testing.T) {
	for _, caso := range []struct {
		nombre, desde, hasta, corte, ventana, metodo, politica string
		divisor, umbral, numerador, denominador, esperado      int64
		protegido                                              bool
	}{
		{"bisiesto", "2024-02-28", "2024-03-01", "2025-01-01", "2024-01-01", "dias_racionales", "proporcional", 1, 0, 1, 1, 2_000_000, false},
		{"no_bisiesto", "2023-02-28", "2023-03-01", "2024-01-01", "2023-01-01", "dias_racionales", "proporcional", 1, 0, 1, 1, 1_000_000, false},
		{"corte_ventana", "2023-12-01", "2024-02-01", "2024-01-03", "2024-01-01", "dias_racionales", "proporcional", 1, 0, 1, 1, 2_000_000, false},
		{"abierto", "2024-12-29", "", "2025-01-01", "2024-01-01", "dias_racionales", "proporcional", 1, 0, 1, 1, 3_000_000, false},
		{"fin_exclusivo", "2024-01-01", "2024-01-02", "2025-01-01", "2024-01-01", "dias_racionales", "proporcional", 1, 0, 1, 1, 1_000_000, false},
		{"fuera_ventana", "2023-01-01", "2023-02-01", "2025-01-01", "2024-01-01", "dias_racionales", "proporcional", 1, 0, 1, 1, 0, false},
		{"inicio_en_corte", "2025-01-01", "2025-01-02", "2025-01-01", "2024-01-01", "dias_racionales", "proporcional", 1, 0, 1, 1, 0, false},
		{"tercio_exacto", "2024-01-01", "2024-01-04", "2025-01-01", "2024-01-01", "dias_racionales", "proporcional", 1, 0, 1, 3, 1_000_000, false},
		{"parcial_integra", "2024-01-01", "2024-01-04", "2025-01-01", "2024-01-01", "dias_racionales", "integra", 1, 0, 1, 3, 3_000_000, false},
		{"protegida_atestada", "2024-01-01", "2024-01-04", "2025-01-01", "2024-01-01", "dias_racionales", "protegida_integra", 1, 0, 1, 3, 3_000_000, true},
		{"protegida_sin_atestacion", "2024-01-01", "2024-01-04", "2025-01-01", "2024-01-01", "dias_racionales", "protegida_integra", 1, 0, 1, 3, 1_000_000, false},
		{"dias_divisor_30", "2024-01-01", "2024-03-02", "2025-01-01", "2024-01-01", "dias_completos", "integra", 30, 0, 1, 1, 2_000_000, false},
		{"mes_civil_bisiesto", "2024-02-01", "2024-03-01", "2025-01-01", "2024-01-01", "meses_completos", "integra", 1, 0, 1, 1, 1_000_000, false},
		{"mes_civil_incompleto", "2024-02-01", "2024-02-29", "2025-01-01", "2024-01-01", "meses_completos", "integra", 1, 0, 1, 1, 0, false},
		{"meses_civiles_tercio", "2024-01-01", "2024-04-01", "2025-01-01", "2024-01-01", "meses_completos", "proporcional", 1, 0, 1, 3, 1_000_000, false},
		{"ano_resto_en_umbral", "2023-01-01", "2024-07-01", "2025-01-01", "2023-01-01", "anos_desde_meses", "integra", 12, 6, 1, 1, 1_000_000, false},
		{"ano_resto_supera_umbral", "2023-01-01", "2024-08-01", "2025-01-01", "2023-01-01", "anos_desde_meses", "integra", 12, 6, 1, 1, 2_000_000, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			r := concRegla(t, d.Antiguedad)
			r.Conversion = &d.Conversion{Metodo: caso.metodo, Divisor: caso.divisor, UmbralResto: caso.umbral}
			r.Jornada = caso.politica
			if caso.metodo == "meses_completos" || caso.metodo == "anos_desde_meses" {
				r.Agrupacion = "por_periodo"
			}
			c := concConfig(t, r)
			c.VentanaDesde, c.FechaCorte = concFecha(t, caso.ventana), concFecha(t, caso.corte)
			p := concPeriodo(t, "periodo:tiempo", caso.desde, caso.hasta, d.Antiguedad)
			p.Jornada = concJornada(t, caso.numerador, caso.denominador)
			if caso.protegido {
				p.AtestacionProtegidaRef = "atestacion:sintetica:computo-integro"
			}
			e := concEntrada()
			e.Periodos = []d.Periodo{p}
			concTotal(t, concCalcular(t, c, e), caso.esperado)
		})
	}
}

func TestConcursosConformidadAgrupacionYRedondeo(t *testing.T) {
	for _, caso := range []struct {
		nombre, agrupacion, metodo     string
		coeficiente, divisor, esperado int64
		modo                           b.ModoRedondeo
	}{
		{"restos_por_tramo", "por_tramo", "dias_completos", 1_000_000, 30, 1_000_000, b.RedondeoExacto},
		{"restos_por_periodo", "por_periodo", "dias_completos", 1_000_000, 30, 0, b.RedondeoExacto},
		{"micropuntos_agrupados", "por_tramo", "dias_racionales", 1, 3, 1, b.RedondeoTruncar},
		{"micropuntos_por_periodo", "por_periodo", "dias_racionales", 1, 3, 0, b.RedondeoTruncar},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			r := concRegla(t, d.Antiguedad)
			r.Agrupacion, r.Redondeo = caso.agrupacion, caso.modo
			r.Conversion = &d.Conversion{Metodo: caso.metodo, Divisor: caso.divisor}
			r.Coeficiente = concPuntos(t, caso.coeficiente)
			e := concEntrada()
			if caso.metodo == "dias_completos" {
				e.Periodos = []d.Periodo{concPeriodo(t, "periodo:1", "2024-01-01", "2024-01-16", d.Antiguedad), concPeriodo(t, "periodo:2", "2024-01-16", "2024-01-31", d.Antiguedad)}
			} else {
				e.Periodos = []d.Periodo{concPeriodo(t, "periodo:1", "2024-01-01", "2024-01-02", d.Antiguedad), concPeriodo(t, "periodo:2", "2024-01-02", "2024-01-04", d.Antiguedad)}
			}
			concTotal(t, concCalcular(t, concConfig(t, r), e), caso.esperado)
		})
	}
	t.Run("cambios_jornada", func(t *testing.T) {
		r := concRegla(t, d.Antiguedad)
		r.Conversion = &d.Conversion{Metodo: "meses_completos", Divisor: 1}
		r.Agrupacion = "por_periodo"
		r.Coeficiente = concPuntos(t, 100_000)
		e := concEntrada()
		p := concPeriodo(t, "periodo:media", "2024-07-01", "2025-01-01", d.Antiguedad)
		p.Jornada = concJornada(t, 1, 2)
		e.Periodos = []d.Periodo{concPeriodo(t, "periodo:completa", "2024-01-01", "2024-07-01", d.Antiguedad), p}
		concTotal(t, concCalcular(t, concConfig(t, r), e), 900_000)
	})
	t.Run("tope_tramo_unico_por_periodo", func(t *testing.T) {
		r := concRegla(t, d.ValoracionTrabajo)
		r.Agrupacion = "por_periodo"
		r.Tramos[1].Maximo = concPuntos(t, 3_000_000)
		e := concEntrada()
		e.Periodos = []d.Periodo{concPeriodo(t, "periodo:1", "2024-01-01", "2024-01-02", d.ValoracionTrabajo), concPeriodo(t, "periodo:2", "2024-01-02", "2024-01-03", d.ValoracionTrabajo)}
		concTotal(t, concCalcular(t, concConfig(t, r), e), 3_000_000)
	})
}

func concCurso(t *testing.T, id string, horas int64) d.Curso {
	t.Helper()
	return d.Curso{ID: id, EvidenciaRef: "evidencia:" + id, Tipo: "formacion:admitida", Horas: concRacional(t, horas, 1),
		Fecha: concFecha(t, "2024-03-01"), Relacionado: true, Acreditado: true}
}

func TestConcursosConformidadCursos(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		cambiar  func(*testing.T, *d.Curso)
		esperado int64
	}{
		{"admitido", func(t *testing.T, c *d.Curso) {}, 10_000_000},
		{"no_relacionado", func(t *testing.T, c *d.Curso) { c.Relacionado = false }, 0},
		{"no_acreditado", func(t *testing.T, c *d.Curso) { c.Acreditado = false }, 0},
		{"caducado", func(t *testing.T, c *d.Curso) { c.VigenteHasta = concFin(t, "2024-12-01") }, 0},
		{"posterior_corte", func(t *testing.T, c *d.Curso) { c.Fecha = concFecha(t, "2025-01-02") }, 0},
		{"tipo_no_admitido", func(t *testing.T, c *d.Curso) { c.Tipo = "formacion:otra" }, 0},
		{"duracion_inferior", func(t *testing.T, c *d.Curso) { c.Horas = concRacional(t, 4, 1) }, 0},
		{"duracion_en_limite", func(t *testing.T, c *d.Curso) { c.Horas = concRacional(t, 5, 1) }, 5_000_000},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			curso := concCurso(t, "curso:1", 10)
			caso.cambiar(t, &curso)
			e := concEntrada()
			e.Cursos = []d.Curso{curso}
			concTotal(t, concCalcular(t, concConfig(t, concRegla(t, d.Cursos)), e), caso.esperado)
		})
	}
	t.Run("duplicado_exacto", func(t *testing.T) {
		e := concEntrada()
		curso := concCurso(t, "curso:1", 10)
		e.Cursos = []d.Curso{curso, curso}
		concTotal(t, concCalcular(t, concConfig(t, concRegla(t, d.Cursos)), e), 10_000_000)
	})
	t.Run("duplicado_conflictivo", func(t *testing.T) {
		e := concEntrada()
		a := concCurso(t, "curso:1", 10)
		otro := a
		otro.Horas = concRacional(t, 11, 1)
		e.Cursos = []d.Curso{a, otro}
		if _, err := d.Calcular(concConfig(t, concRegla(t, d.Cursos)), e); err == nil {
			t.Fatal("se aceptó un duplicado con horas contradictorias")
		}
	})
	t.Run("evidencia_repetida_con_otro_identificador", func(t *testing.T) {
		e := concEntrada()
		a := concCurso(t, "curso:1", 10)
		otro := a
		otro.ID = "curso:2"
		e.Cursos = []d.Curso{a, otro}
		if _, err := d.Calcular(concConfig(t, concRegla(t, d.Cursos)), e); err == nil {
			t.Fatal("la misma evidencia se puntuó con dos identificadores")
		}
	})
	t.Run("tope_conserva_bruto", func(t *testing.T) {
		r := concRegla(t, d.Cursos)
		r.Maximo = concPuntos(t, 12_000_000)
		c := concConfig(t, r)
		c.MaximoTotal = concPuntos(t, 9_000_000)
		e := concEntrada()
		e.Cursos = []d.Curso{concCurso(t, "curso:1", 10), concCurso(t, "curso:2", 8)}
		res := concCalcular(t, c, e)
		concTotal(t, res, 9_000_000)
		parte := concDesglose(t, res, d.Cursos)
		if parte.Bruto.Micropuntos() != 18_000_000 || parte.Resultado.Micropuntos() != 12_000_000 || parte.Maximo.Micropuntos() != 12_000_000 {
			t.Fatalf("se perdió el bruto o el tope de regla: %+v", parte)
		}
	})
	t.Run("limite_elige_mayor_unidad_elegible", func(t *testing.T) {
		r := concRegla(t, d.Cursos)
		r.MaximoElementos, r.SeleccionElementos = 1, "mayor_unidades"
		e := concEntrada()
		noRelacionado := concCurso(t, "curso:0", 20)
		noRelacionado.Relacionado = false
		e.Cursos = []d.Curso{concCurso(t, "curso:1", 8), concCurso(t, "curso:2", 10), noRelacionado}
		concTotal(t, concCalcular(t, concConfig(t, r), e), 10_000_000)
	})
}

func TestConcursosConformidadTituloRequisito(t *testing.T) {
	e := concEntrada()
	titulo := d.Titulo{ID: "titulo:requisito", EvidenciaRef: "evidencia:titulo:requisito", Tipo: "titulo:admitido",
		Fecha: concFecha(t, "2020-01-01"), Acreditado: true, UsadoRequisito: true}
	adicional := titulo
	adicional.ID, adicional.EvidenciaRef, adicional.UsadoRequisito = "titulo:adicional", "evidencia:titulo:adicional", false
	e.Titulaciones = []d.Titulo{titulo, adicional}
	for _, caso := range []struct {
		excluir  bool
		esperado int64
	}{{true, 1_000_000}, {false, 2_000_000}} {
		r := concRegla(t, d.Titulaciones)
		r.ExcluirRequisito = caso.excluir
		r.Tipos = []string{"titulo:admitido"}
		concTotal(t, concCalcular(t, concConfig(t, r), e), caso.esperado)
	}
}

func TestConcursosConformidadFuentesAusentesYSolapes(t *testing.T) {
	for _, familia := range []d.Familia{d.Antiguedad, d.Permanencia, d.Cursos, d.Titulaciones, d.ValoracionTrabajo} {
		t.Run(string(familia), func(t *testing.T) {
			e := concEntrada()
			c := concConfig(t, concRegla(t, familia))
			concTotal(t, concCalcular(t, c, e), 0)
			e.Disponibles = []d.Familia{}
			r := concCalcular(t, c, e)
			if r.Completo || r.Total != nil || len(r.Incidencias) == 0 {
				t.Fatalf("la ausencia de fuente se convirtió en cero: %+v", r)
			}
		})
	}
	t.Run("grado_ausente", func(t *testing.T) {
		r := concCalcular(t, concConfig(t, concRegla(t, d.Grado)), concEntrada())
		if r.Completo || r.Total != nil || len(r.Incidencias) == 0 {
			t.Fatalf("grado desconocido valorado como cero: %+v", r)
		}
	})
	t.Run("solape_rechazado", func(t *testing.T) {
		e := concEntrada()
		e.Periodos = []d.Periodo{concPeriodo(t, "periodo:1", "2024-01-01", "2024-01-03", d.Antiguedad), concPeriodo(t, "periodo:2", "2024-01-02", "2024-01-04", d.Antiguedad)}
		_, err := d.Calcular(concConfig(t, concRegla(t, d.Antiguedad)), e)
		var nominal *d.Error
		if !errors.As(err, &nominal) || nominal.Codigo != "periodos_solapados" {
			t.Fatalf("solape no rechazado con error nominal: %v", err)
		}
	})
	t.Run("familias_distintas_explicitas", func(t *testing.T) {
		e := concEntrada()
		p := concPeriodo(t, "periodo:compartido", "2024-01-01", "2024-01-03", d.Antiguedad)
		p.Familias = []d.Familia{d.Antiguedad, d.Permanencia}
		e.Periodos = []d.Periodo{p}
		r := concCalcular(t, concConfig(t, concRegla(t, d.Antiguedad), concRegla(t, d.Permanencia)), e)
		concTotal(t, r, 4_000_000)
		if len(r.Desglose) != 2 {
			t.Fatalf("familias mezcladas: %+v", r.Desglose)
		}
	})
}

func TestConcursosConformidadReproduccion(t *testing.T) {
	c := concConfig(t, concRegla(t, d.Antiguedad), concRegla(t, d.Cursos), concRegla(t, d.Titulaciones))
	c.Reglas[1].Tipos = []string{"formacion:admitida", "formacion:alternativa"}
	e := concEntrada()
	e.Periodos = []d.Periodo{concPeriodo(t, "periodo:2", "2024-01-03", "2024-01-05", d.Antiguedad), concPeriodo(t, "periodo:1", "2024-01-01", "2024-01-03", d.Antiguedad)}
	e.Cursos = []d.Curso{concCurso(t, "curso:2", 6), concCurso(t, "curso:1", 5)}
	e.Titulaciones = []d.Titulo{
		{ID: "titulo:2", EvidenciaRef: "evidencia:titulo:2", Tipo: "titulo:sintetico", Fecha: concFecha(t, "2020-01-01"), Acreditado: true},
		{ID: "titulo:1", EvidenciaRef: "evidencia:titulo:1", Tipo: "titulo:sintetico", Fecha: concFecha(t, "2021-01-01"), Acreditado: true},
	}
	antesC, _ := json.Marshal(c)
	antesE, _ := json.Marshal(e)
	original := concCalcular(t, c, e)
	concTotal(t, original, 17_000_000)
	despuesC, _ := json.Marshal(c)
	despuesE, _ := json.Marshal(e)
	if string(antesC) != string(despuesC) || string(antesE) != string(despuesE) {
		t.Fatal("Calcular alteró la configuración o la instantánea del llamante")
	}
	for _, huella := range []string{original.HuellaReglas, original.HuellaEntrada, original.HuellaResultado} {
		if bytes, err := hex.DecodeString(huella); err != nil || len(bytes) != 32 {
			t.Fatalf("huella SHA256 no canónica: %q", huella)
		}
	}
	if original.VersionMotor != d.VersionMotor || original.VersionReglas != c.Version || original.InstantaneaRef != e.InstantaneaRef {
		t.Fatalf("procedencia incompleta: %+v", original)
	}
	if repetido := concCalcular(t, c, e); !reflect.DeepEqual(original, repetido) {
		t.Fatal("el mismo cálculo no reprodujo el desglose y las huellas")
	}

	// Permutar conjuntos conserva tanto las huellas como la explicación canónica.
	c.Reglas[0], c.Reglas[2] = c.Reglas[2], c.Reglas[0]
	c.Reglas[1].Tipos[0], c.Reglas[1].Tipos[1] = c.Reglas[1].Tipos[1], c.Reglas[1].Tipos[0]
	e.Periodos[0], e.Periodos[1] = e.Periodos[1], e.Periodos[0]
	e.Cursos[0], e.Cursos[1] = e.Cursos[1], e.Cursos[0]
	e.Titulaciones[0], e.Titulaciones[1] = e.Titulaciones[1], e.Titulaciones[0]
	for i, j := 0, len(e.Disponibles)-1; i < j; i, j = i+1, j-1 {
		e.Disponibles[i], e.Disponibles[j] = e.Disponibles[j], e.Disponibles[i]
	}
	if permutado := concCalcular(t, c, e); !reflect.DeepEqual(original, permutado) {
		t.Fatal("el orden de los conjuntos cambió el resultado o sus huellas")
	}
	c.Version = "reglas:v2"
	revisado := concCalcular(t, c, e)
	concTotal(t, revisado, 17_000_000)
	if revisado.HuellaEntrada != original.HuellaEntrada || revisado.HuellaReglas == original.HuellaReglas || revisado.HuellaResultado == original.HuellaResultado {
		t.Fatal("cambio de versión no quedó aislado en sus huellas")
	}
	c.Reglas[1].Coeficiente = concPuntos(t, 2_000_000)
	recalculado := concCalcular(t, c, e)
	concTotal(t, recalculado, 28_000_000)
	if recalculado.HuellaReglas == revisado.HuellaReglas || recalculado.HuellaResultado == revisado.HuellaResultado {
		t.Fatal("cambio de coeficiente no quedó registrado")
	}
	e.Cursos[0].Horas = concRacional(t, 7, 1)
	nuevo := concCalcular(t, c, e)
	concTotal(t, nuevo, 32_000_000)
	if nuevo.HuellaEntrada == recalculado.HuellaEntrada || nuevo.HuellaResultado == recalculado.HuellaResultado {
		t.Fatal("rectificación del hecho no cambió sus huellas")
	}
}

func TestConcursosConformidadRechazoConfiguracion(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*d.Configuracion)
	}{
		{"version_schema", func(c *d.Configuracion) { c.SchemaVersion = "desconocida" }},
		{"conversion_ausente", func(c *d.Configuracion) { c.Reglas[0].Conversion = nil }},
		{"conversion_libre", func(c *d.Configuracion) { c.Reglas[0].Conversion.Metodo = "eval(formula)" }},
		{"divisor_cero", func(c *d.Configuracion) { c.Reglas[0].Conversion.Divisor = 0 }},
		{"jornada_ausente", func(c *d.Configuracion) { c.Reglas[0].Jornada = "" }},
		{"solapes_ausentes", func(c *d.Configuracion) { c.Reglas[0].Solapes = "" }},
		{"agrupacion_ausente", func(c *d.Configuracion) { c.Reglas[0].Agrupacion = "" }},
		{"redondeo_ausente", func(c *d.Configuracion) { c.Reglas[0].Redondeo = "" }},
		{"regla_duplicada", func(c *d.Configuracion) { c.Reglas = append(c.Reglas, c.Reglas[0]) }},
		{"meses_calendario_por_tramo", func(c *d.Configuracion) { c.Reglas[0].Conversion.Metodo = "meses_completos" }},
		{"anos_calendario_por_tramo", func(c *d.Configuracion) { c.Reglas[0].Conversion.Metodo = "anos_desde_meses" }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c := concConfig(t, concRegla(t, d.Antiguedad))
			caso.cambiar(&c)
			if err := d.ValidarConfiguracion(c); err == nil {
				t.Fatal("configuración inválida aceptada")
			}
			if _, err := d.Calcular(c, concEntrada()); err == nil {
				t.Fatal("Calcular omitió la validación de configuración")
			}
		})
	}
	for _, caso := range []struct {
		nombre  string
		cambiar func(*d.Regla)
	}{
		{"hueco", func(r *d.Regla) { r.Tramos[0].MaxDiferencia = -2 }},
		{"solape", func(r *d.Regla) { r.Tramos[0].MaxDiferencia = 0 }},
		{"cobertura_incompleta", func(r *d.Regla) { r.Tramos = r.Tramos[:2] }},
		{"sentido_ausente", func(r *d.Regla) { r.Diferencia = "" }},
	} {
		t.Run("tabla/"+caso.nombre, func(t *testing.T) {
			r := concRegla(t, d.Grado)
			caso.cambiar(&r)
			if err := d.ValidarConfiguracion(concConfig(t, r)); err == nil {
				t.Fatal("tabla ambigua aceptada")
			}
		})
	}
}

func TestConcursosConformidadRechazoEntrada(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*testing.T, *d.Entrada)
	}{
		{"familia_desconocida", func(t *testing.T, e *d.Entrada) { e.Disponibles = append(e.Disponibles, "desempeno_cualitativo") }},
		{"familia_repetida", func(t *testing.T, e *d.Entrada) { e.Disponibles = append(e.Disponibles, d.Cursos) }},
		{"periodos_nil", func(t *testing.T, e *d.Entrada) { e.Periodos = nil }},
		{"cursos_nil", func(t *testing.T, e *d.Entrada) { e.Cursos = nil }},
		{"titulos_nil", func(t *testing.T, e *d.Entrada) { e.Titulaciones = nil }},
		{"intervalo_vacio", func(t *testing.T, e *d.Entrada) {
			e.Periodos = []d.Periodo{concPeriodo(t, "periodo:1", "2024-01-01", "2024-01-01", d.Antiguedad)}
		}},
		{"intervalo_invertido", func(t *testing.T, e *d.Entrada) {
			e.Periodos = []d.Periodo{concPeriodo(t, "periodo:1", "2024-01-03", "2024-01-01", d.Antiguedad)}
		}},
		{"jornada_invalida", func(t *testing.T, e *d.Entrada) {
			p := concPeriodo(t, "periodo:1", "2024-01-01", "2024-01-02", d.Antiguedad)
			p.Jornada = b.FraccionJornada{}
			e.Periodos = []d.Periodo{p}
		}},
		{"grado_sin_evidencia", func(t *testing.T, e *d.Entrada) { grado := 20; e.GradoPersonal = &grado }},
		{"nivel_trabajo_ausente", func(t *testing.T, e *d.Entrada) {
			p := concPeriodo(t, "periodo:1", "2024-01-01", "2024-01-02", d.ValoracionTrabajo)
			p.Nivel = 0
			e.Periodos = []d.Periodo{p}
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := concEntrada()
			caso.cambiar(t, &e)
			if err := d.ValidarEntrada(e); err == nil {
				t.Fatal("entrada inválida aceptada")
			}
			_, err := d.Calcular(concConfig(t, concRegla(t, d.Antiguedad)), e)
			var nominal *d.Error
			if !errors.As(err, &nominal) || nominal.Codigo == "" || nominal.Campo == "" {
				t.Fatalf("falta error nominal de entrada: %v", err)
			}
		})
	}
}

func TestConcursosConformidadRegresionesPoliticas(t *testing.T) {
	t.Run("atestacion_blanca_no_concede_computo_integro", func(t *testing.T) {
		r := concRegla(t, d.Antiguedad)
		r.Jornada = "protegida_integra"
		p := concPeriodo(t, "periodo:1", "2024-01-01", "2024-01-04", d.Antiguedad)
		p.Jornada, p.AtestacionProtegidaRef = concJornada(t, 1, 3), " "
		e := concEntrada()
		e.Periodos = []d.Periodo{p}
		_, err := d.Calcular(concConfig(t, r), e)
		var nominal *d.Error
		if !errors.As(err, &nominal) || nominal.Codigo != "atestacion_invalida" {
			t.Fatalf("atestación blanca no rechazada: %v", err)
		}
	})
	for _, familia := range []d.Familia{d.Grado, d.Antiguedad, d.Permanencia, d.ValoracionTrabajo} {
		t.Run("limite_elementos_incompatible/"+string(familia), func(t *testing.T) {
			r := concRegla(t, familia)
			r.MaximoElementos = 1
			if err := d.ValidarConfiguracion(concConfig(t, r)); err == nil {
				t.Fatal("se aceptó un límite de elementos que la familia no aplica")
			}
		})
	}
	t.Run("filtros_disjuntos_no_duplican_grado", func(t *testing.T) {
		a, otro := concRegla(t, d.Grado), concRegla(t, d.Grado)
		a.ID, a.Tipos, otro.ID, otro.Tipos = "regla:grado:a", []string{"tipo:a"}, "regla:grado:b", []string{"tipo:b"}
		if err := d.ValidarConfiguracion(concConfig(t, a, otro)); err == nil {
			t.Fatal("dos filtros de tipo permitieron puntuar el mismo grado dos veces")
		}
	})
}
