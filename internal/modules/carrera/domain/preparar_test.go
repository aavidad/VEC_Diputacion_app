package domain

import (
	"reflect"
	"testing"
)

func fixture() Escenario {
	f := Fuente{"fuente", "v1"}
	ev := Evidencia{"evidencia", f.Referencia, f.Version}
	nivel, grado := 22, 20
	return Escenario{Alcance: AlcanceSintetico, Version: "ensayo-1", Fuente: f, Casos: []Caso{{Referencia: "caso-1", Via: "grado", PersonaNombre: "Clara Montoya", Regimen: "regimen-configurado", NivelPuesto: &nivel, GradoPersonal: &grado, GrupoSubgrupo: "grupo-configurado", Fuentes: []Fuente{f}, Periodos: []Periodo{{"2020-01-01", "2022-12-31", ev}}, Evidencias: []Evidencia{ev}, Politica: Politica{Referencia: "politica", Version: "v1", Fuente: "fuente", AprobacionReferencia: "referencia-sintetica", Regimenes: []string{"regimen-configurado"}}}}}
}
func estado(p Preparacion, key string) Comprobacion {
	for _, c := range p.Casos[0].Comprobaciones {
		if c.Clave == "carrera.comprobacion."+key {
			return c
		}
	}
	return Comprobacion{}
}
func TestPreparacionNoReconoceNiEquiparaNivelYGrado(t *testing.T) {
	e := fixture()
	before := *e.Casos[0].GradoPersonal
	p, err := Preparar(e)
	if err != nil {
		t.Fatal(err)
	}
	if *p.Casos[0].NivelPuesto != 22 || *p.Casos[0].GradoPersonal != 20 || *e.Casos[0].GradoPersonal != before {
		t.Fatal("separacion alterada")
	}
	if p.Casos[0].EstadoGlobal != "pendiente" || estado(p, "reconocimiento").Estado != "pendiente" || estado(p, "registro").Estado != "pendiente" {
		t.Fatal("reconocimiento implicito")
	}
	*p.Casos[0].GradoPersonal = 99
	if *e.Casos[0].GradoPersonal != before {
		t.Fatal("salida comparte grado mutable")
	}
}
func TestPeriodosFechasYSolapesSinSumar(t *testing.T) {
	for _, tc := range []struct {
		name, ini, fin, motivo string
		extra                  bool
	}{
		{"fecha imposible", "2021-02-29", "2022-01-01", "fecha_invalida", false},
		{"invertido", "2023-01-01", "2022-01-01", "fecha_invalida", false},
		{"solape", "2022-12-31", "2023-01-01", "periodos_solapados", true},
		{"adyacente", "2023-01-01", "2024-01-01", "disponible", true},
		{"larga duración", "1900-01-01", "2099-12-31", "disponible", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := fixture()
			periodo := e.Casos[0].Periodos[0]
			periodo.Inicio = tc.ini
			periodo.Fin = tc.fin
			if tc.extra {
				e.Casos[0].Periodos = append(e.Casos[0].Periodos, periodo)
			} else {
				e.Casos[0].Periodos = []Periodo{periodo}
			}
			p, err := Preparar(e)
			if err != nil {
				t.Fatal(err)
			}
			if estado(p, "periodos").MotivoClave != "carrera.motivo."+tc.motivo || p.Casos[0].EstadoGlobal != "pendiente" {
				t.Fatalf("%+v", p)
			}
		})
	}
}
func TestPoliticaYCompatibilidadProcedenDeDatos(t *testing.T) {
	e := fixture()
	e.Casos[0].Politica.AprobacionReferencia = ""
	p, _ := Preparar(e)
	if estado(p, "politica").Estado != "pendiente" || estado(p, "compatibilidad").Estado != "pendiente" {
		t.Fatal("politica no aprobada activada")
	}
	e = fixture()
	e.Casos[0].Politica.Regimenes = []string{"otro-regimen"}
	p, _ = Preparar(e)
	if estado(p, "compatibilidad").Estado != "incompatible" {
		t.Fatal("ignora compatibilidad configurada")
	}
	e = fixture()
	e.Casos[0].Fuentes[0].Version = "otra-version"
	p, _ = Preparar(e)
	if estado(p, "evidencias").Estado != "pendiente" || estado(p, "politica").Estado != "pendiente" {
		t.Fatal("versiones desacopladas")
	}
}
func TestProgresionExigeConvenioYCursoPrueba(t *testing.T) {
	e := fixture()
	c := &e.Casos[0]
	c.Via = "progresion"
	c.GrupoProfesional = "grupo"
	c.Convenio = c.Evidencias[0]
	c.ConvenioConsolidado = true
	c.Curso = c.Evidencias[0]
	c.Prueba = c.Evidencias[0]
	p, _ := Preparar(e)
	if estado(p, "convenio").Estado != "disponible" || estado(p, "curso_prueba").Estado != "pendiente" {
		t.Fatal("curso sin aprobacion")
	}
	c.CursoAprobacion = "aprobacion-curso-sintetica"
	c.PruebaAprobacion = "aprobacion-prueba-sintetica"
	p, _ = Preparar(e)
	if estado(p, "curso_prueba").Estado != "disponible" || p.Casos[0].EstadoGlobal != "pendiente" {
		t.Fatal("curso concede progresion")
	}
}
func TestPromocionSoloReferenciaConvocatoria(t *testing.T) {
	e := fixture()
	e.Casos[0].Via = "promocion"
	p, _ := Preparar(e)
	if estado(p, "convocatoria").Estado != "pendiente" {
		t.Fatal("convocatoria inventada")
	}
	e.Casos[0].Convocatoria = e.Casos[0].Evidencias[0]
	p, _ = Preparar(e)
	if estado(p, "convocatoria").Estado != "disponible" || p.Casos[0].EstadoGlobal != "pendiente" {
		t.Fatal("motor selectivo paralelo")
	}
	p2, _ := Preparar(e)
	if !reflect.DeepEqual(p, p2) {
		t.Fatal("salida no determinista")
	}
}
func TestAlcanceYCardinalidadCerrados(t *testing.T) {
	e := fixture()
	e.Alcance = "produccion"
	if _, err := Preparar(e); err == nil {
		t.Fatal("admite produccion")
	}
	e = fixture()
	e.Casos = append(e.Casos, e.Casos[0])
	if _, err := Preparar(e); err == nil {
		t.Fatal("referencias duplicadas")
	}
	e = fixture()
	e.Casos[0].Periodos = make([]Periodo, 129)
	if _, err := Preparar(e); err == nil {
		t.Fatal("sin limite")
	}
}
func TestAntecedentesConservanDatosSinCompartirEntrada(t *testing.T) {
	e := fixture()
	c := &e.Casos[0]
	c.GrupoProfesional = "grupo-declarado"
	c.Convenio = Evidencia{"convenio", "fuente", "v1"}
	c.Curso = Evidencia{"curso", "fuente", "v1"}
	c.Prueba = Evidencia{"prueba", "fuente", "v1"}
	c.Convocatoria = Evidencia{"convocatoria-selectivos-A", "fuente", "v1"}
	c.CursoAprobacion, c.PruebaAprobacion = "aprobacion-curso", "aprobacion-prueba"
	p, err := Preparar(e)
	if err != nil {
		t.Fatal(err)
	}
	a := p.Casos[0].Antecedentes
	if a.GrupoSubgrupo != c.GrupoSubgrupo || a.GrupoProfesional != c.GrupoProfesional ||
		!reflect.DeepEqual(a.Periodos, c.Periodos) || !reflect.DeepEqual(a.Evidencias, c.Evidencias) ||
		!reflect.DeepEqual(a.Politica, c.Politica) || a.Convenio != c.Convenio || a.Curso != c.Curso ||
		a.Prueba != c.Prueba || a.Convocatoria != c.Convocatoria || a.CursoAprobacion != c.CursoAprobacion ||
		a.PruebaAprobacion != c.PruebaAprobacion {
		t.Fatal("antecedentes perdidos")
	}
	c.Fuentes[0].Referencia = "mutada"
	c.Periodos[0].Inicio = "mutado"
	c.Periodos[0].Evidencia.Referencia = "mutada"
	c.Evidencias[0].Referencia = "mutada"
	c.Politica.Regimenes[0] = "mutado"
	c.Convocatoria.Referencia = "mutada"
	if p.Casos[0].Fuentes[0].Referencia != "fuente" || a.Periodos[0].Inicio != "2020-01-01" ||
		a.Periodos[0].Evidencia.Referencia != "evidencia" || a.Evidencias[0].Referencia != "evidencia" ||
		a.Politica.Regimenes[0] != "regimen-configurado" || a.Convocatoria.Referencia != "convocatoria-selectivos-A" {
		t.Fatal("salida comparte entrada mutable")
	}
}
