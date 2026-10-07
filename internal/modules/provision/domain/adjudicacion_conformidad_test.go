package domain_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	d "vec-diputacion-granada/internal/modules/provision/domain"
	b "vec-diputacion-granada/internal/shared/baremacion"
)

func adjConfiguraciones(t *testing.T) (d.Configuracion, d.ConfiguracionAdjudicacion) {
	t.Helper()
	cursos, titulos := concRegla(t, d.Cursos), concRegla(t, d.Titulaciones)
	cero := concRacional(t, 0, 1)
	cursos.HorasMinimas = &cero
	c := concConfig(t, cursos, titulos)
	r := concCalcular(t, c, concEntrada())
	a := d.ConfiguracionAdjudicacion{
		SchemaVersion: d.VersionAdjudicacion, ProcesoRef: c.ConvocatoriaRef, Version: "ensayo:v1", BasesRef: c.BasesRef, HuellaBases: strings.Repeat("a", 64),
		PoliticaRef: "politica:sintetica:1", PoliticaVersion: "politica:v1", Metodo: d.MetodoAdjudicacionEnsayo,
		Prioridad: "total_descendente", Incompatibilidad: "un_puesto_por_persona",
		Desempates:             []d.DesempateAdjudicacion{{ReglaID: cursos.ID, Sentido: "mayor"}, {ReglaID: titulos.ID, Sentido: "mayor"}},
		VersionMotorValoracion: d.VersionMotor, VersionReglasValoracion: c.Version, HuellaReglasValoracion: r.HuellaReglas,
	}
	return c, a
}
func adjValoracion(t *testing.T, c d.Configuracion, persona, puesto string, horas int64, titulos int, fuente bool) *d.Resultado {
	t.Helper()
	e := concEntrada()
	e.InstantaneaRef, e.PuestoRef = "instantanea:"+persona, puesto
	if !fuente {
		e.Disponibles = []d.Familia{d.Titulaciones}
	}
	if horas > 0 {
		e.Cursos = []d.Curso{{ID: "curso:sintetico", EvidenciaRef: "evidencia:curso", Tipo: "formacion:admitida", Horas: concRacional(t, horas, 1), Fecha: concFecha(t, "2024-01-01"), Relacionado: true, Acreditado: true}}
	}
	for i := 0; i < titulos; i++ {
		id := "titulo:" + strings.Repeat("x", i+1)
		e.Titulaciones = append(e.Titulaciones, d.Titulo{ID: id, EvidenciaRef: "evidencia:" + id, Tipo: "titulo:admitido", Fecha: concFecha(t, "2024-01-01"), Acreditado: true})
	}
	r := concCalcular(t, c, e)
	return &r
}
func adjEntrada(t *testing.T, c d.Configuracion) d.EntradaAdjudicacion {
	t.Helper()
	e := d.EntradaAdjudicacion{UniversoRef: "universo:sintetico", UniversoVersion: "universo:v1", Cerrado: true, Sintetico: true,
		Vacantes: []d.VacanteAdjudicacion{{VacanteRef: "vacante:a", PuestoRef: "puesto:a"}, {VacanteRef: "vacante:b", PuestoRef: "puesto:b"}}, Solicitudes: []d.SolicitudAdjudicacion{}}
	for _, p := range []string{"persona:a", "persona:b"} {
		s := d.SolicitudAdjudicacion{SolicitudRef: "solicitud:" + p, Version: "v1", PersonaRef: p, InstantaneaRef: "instantanea:" + p, Preferencias: []d.PreferenciaAdjudicacion{}}
		for _, v := range e.Vacantes {
			s.Preferencias = append(s.Preferencias, d.PreferenciaAdjudicacion{VacanteRef: v.VacanteRef, Admision: "admitida", Valoracion: adjValoracion(t, c, p, v.PuestoRef, 10, 0, true)})
		}
		e.Solicitudes = append(e.Solicitudes, s)
	}
	return e
}
func adjEjecutar(t *testing.T, c d.ConfiguracionAdjudicacion, e d.EntradaAdjudicacion) d.ResultadoAdjudicacion {
	t.Helper()
	r, err := d.SimularAdjudicacion(c, e)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func adjAsignaciones(r d.ResultadoAdjudicacion) map[string]string {
	out := map[string]string{}
	for _, a := range r.Asignaciones {
		out[a.PersonaRef] = a.VacanteRef
	}
	return out
}

func TestAdjudicacionGlobalDesplazaYRespetaPreferencias(t *testing.T) {
	c, a := adjConfiguraciones(t)
	e := adjEntrada(t, c)
	// B propone primero a B; tras rechazo, desplaza a A de la vacante A.
	e.Solicitudes[0].Preferencias[0].Valoracion = adjValoracion(t, c, "persona:a", "puesto:a", 10, 0, true)
	e.Solicitudes[0].Preferencias[1].Valoracion = adjValoracion(t, c, "persona:a", "puesto:b", 30, 0, true)
	e.Solicitudes[1].Preferencias[0].Valoracion = adjValoracion(t, c, "persona:b", "puesto:a", 20, 0, true)
	e.Solicitudes[1].Preferencias[1].Valoracion = adjValoracion(t, c, "persona:b", "puesto:b", 25, 0, true)
	// Tercera persona supera a B en su primera elección B y obliga a continuar.
	s := d.SolicitudAdjudicacion{SolicitudRef: "solicitud:c", Version: "v1", PersonaRef: "persona:c", InstantaneaRef: "instantanea:persona:c", Preferencias: []d.PreferenciaAdjudicacion{{VacanteRef: "vacante:b", Admision: "admitida", Valoracion: adjValoracion(t, c, "persona:c", "puesto:b", 40, 0, true)}}}
	e.Solicitudes = append(e.Solicitudes, s)
	e.Solicitudes[1].Preferencias[0], e.Solicitudes[1].Preferencias[1] = e.Solicitudes[1].Preferencias[1], e.Solicitudes[1].Preferencias[0]
	r := adjEjecutar(t, a, e)
	esperado := map[string]string{"persona:b": "vacante:a", "persona:c": "vacante:b"}
	if r.Estado != "propuesta_simulada" || !reflect.DeepEqual(adjAsignaciones(r), esperado) || !reflect.DeepEqual(r.PersonasSinAsignacion, []string{"persona:a"}) {
		t.Fatalf("resultado global: %+v", r)
	}
	if r.Asignaciones[0].Preferencia != 2 {
		t.Fatal("se perdió la preferencia original")
	}
}

func TestAdjudicacionPrimeraPreferenciaNoRankingAislado(t *testing.T) {
	c, a := adjConfiguraciones(t)
	e := adjEntrada(t, c)
	// La misma persona lidera ambas vacantes; solo recibe su primera preferencia.
	for j, v := range e.Vacantes {
		e.Solicitudes[0].Preferencias[j].Valoracion = adjValoracion(t, c, "persona:a", v.PuestoRef, 20, 0, true)
	}
	r := adjEjecutar(t, a, e)
	if !reflect.DeepEqual(adjAsignaciones(r), map[string]string{"persona:a": "vacante:a", "persona:b": "vacante:b"}) {
		t.Fatal(r)
	}
	// Un empate entre segundas preferencias no interviene si ambas obtienen primera.
	e.Solicitudes[1].Preferencias[0], e.Solicitudes[1].Preferencias[1] = e.Solicitudes[1].Preferencias[1], e.Solicitudes[1].Preferencias[0]
	e.Solicitudes[0].Preferencias[1].Valoracion = adjValoracion(t, c, "persona:a", "puesto:b", 10, 0, true)
	e.Solicitudes[1].Preferencias[1].Valoracion = adjValoracion(t, c, "persona:b", "puesto:a", 20, 0, true)
	if adjEjecutar(t, a, e).Estado != "propuesta_simulada" {
		t.Fatal("empate no competido bloqueó preferencias distintas")
	}
}

func TestAdjudicacionCadenaCompletaYDireccion(t *testing.T) {
	c, a := adjConfiguraciones(t)
	c.MaximoTotal = concPuntos(t, 5_000_000)
	rbase := concCalcular(t, c, concEntrada())
	a.HuellaReglasValoracion = rbase.HuellaReglas
	e := adjEntrada(t, c)
	e.Solicitudes[0].Preferencias[0].Valoracion = adjValoracion(t, c, "persona:a", "puesto:a", 10, 1, true)
	e.Solicitudes[1].Preferencias[0].Valoracion = adjValoracion(t, c, "persona:b", "puesto:a", 10, 2, true)
	// Iguales total y cursos: debe llegar al segundo criterio, titulaciones.
	r := adjEjecutar(t, a, e)
	if adjAsignaciones(r)["persona:b"] != "vacante:a" {
		t.Fatal(r)
	}
	a.Desempates[1].Sentido = "menor"
	if adjAsignaciones(adjEjecutar(t, a, e))["persona:a"] != "vacante:a" {
		t.Fatal("sentido de segundo desempate ignorado")
	}
}

func TestAdjudicacionEmpateBloqueaConjuntoYNoUsaReferencias(t *testing.T) {
	c, a := adjConfiguraciones(t)
	e := adjEntrada(t, c)
	for _, inversion := range []bool{false, true} {
		if inversion {
			e.Solicitudes[0], e.Solicitudes[1] = e.Solicitudes[1], e.Solicitudes[0]
		}
		r := adjEjecutar(t, a, e)
		if r.Estado != "pendiente" || len(r.Asignaciones) != 0 || len(r.PersonasSinAsignacion) != 0 || len(r.Incidencias) != 1 || r.Incidencias[0].Codigo != "empate_residual" {
			t.Fatal(r)
		}
	}
}

func TestAdjudicacionPendientesNuncaCero(t *testing.T) {
	for _, caso := range []string{"admision", "fuente", "universo"} {
		t.Run(caso, func(t *testing.T) {
			c, a := adjConfiguraciones(t)
			e := adjEntrada(t, c)
			switch caso {
			case "admision":
				e.Solicitudes[0].Preferencias[0].Admision = "pendiente"
				e.Solicitudes[0].Preferencias[0].Valoracion = nil
			case "fuente":
				e.Solicitudes[0].Preferencias[0].Valoracion = adjValoracion(t, c, "persona:a", "puesto:a", 0, 0, false)
			case "universo":
				e.Cerrado = false
			}
			r := adjEjecutar(t, a, e)
			if r.Estado != "pendiente" || len(r.Asignaciones) > 0 || len(r.Incidencias) == 0 {
				t.Fatal(r)
			}
		})
	}
}

func TestAdjudicacionExclusionRenunciaYRectificacion(t *testing.T) {
	c, a := adjConfiguraciones(t)
	e := adjEntrada(t, c)
	e.Solicitudes[0].Preferencias[0].Admision = "renunciada"
	e.Solicitudes[0].Preferencias[0].Valoracion = nil
	e.Solicitudes[1].Preferencias[1].Admision = "excluida"
	e.Solicitudes[1].Preferencias[1].Valoracion = nil
	r := adjEjecutar(t, a, e)
	if !reflect.DeepEqual(adjAsignaciones(r), map[string]string{"persona:a": "vacante:b", "persona:b": "vacante:a"}) {
		t.Fatal(r)
	}
	// Rectificar exige otra solicitud/universo: cambia la huella, conserva anterior.
	e.Solicitudes[0].Version = "v2"
	e.UniversoVersion = "universo:v2"
	e.Solicitudes[0].Preferencias[0].Admision = "admitida"
	e.Solicitudes[0].Preferencias[0].Valoracion = adjValoracion(t, c, "persona:a", "puesto:a", 20, 0, true)
	siguiente := adjEjecutar(t, a, e)
	if siguiente.HuellaEntrada == r.HuellaEntrada || siguiente.HuellaResultado == r.HuellaResultado || adjAsignaciones(r)["persona:a"] != "vacante:b" {
		t.Fatal("rectificación no causal")
	}
}

func TestAdjudicacionDeterministaSinMutarEntrada(t *testing.T) {
	c, a := adjConfiguraciones(t)
	e := adjEntrada(t, c)
	e.Solicitudes[0].Preferencias[0].Valoracion = adjValoracion(t, c, "persona:a", "puesto:a", 20, 0, true)
	antes, _ := json.Marshal(e)
	r := adjEjecutar(t, a, e)
	despues, _ := json.Marshal(e)
	if string(antes) != string(despues) {
		t.Fatal("entrada mutada")
	}
	e.Solicitudes[0], e.Solicitudes[1] = e.Solicitudes[1], e.Solicitudes[0]
	e.Vacantes[0], e.Vacantes[1] = e.Vacantes[1], e.Vacantes[0]
	if !reflect.DeepEqual(r, adjEjecutar(t, a, e)) {
		t.Fatal("orden de transporte influyó en resultado")
	}
	a.PoliticaVersion = "politica:v2"
	if adjEjecutar(t, a, e).HuellaConfiguracion == r.HuellaConfiguracion {
		t.Fatal("versión política no sellada")
	}
}

func TestAdjudicacionCierreDeContratos(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*d.ConfiguracionAdjudicacion, *d.EntradaAdjudicacion)
	}{
		{"cadena_vacia", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) { a.Desempates = nil }},
		{"desempate_uuid", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) { a.Desempates[0].ReglaID = "uuid" }},
		{"persona_repetida", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) {
			e.Solicitudes[1].PersonaRef = e.Solicitudes[0].PersonaRef
		}},
		{"solicitud_repetida", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) {
			e.Solicitudes[1].SolicitudRef = e.Solicitudes[0].SolicitudRef
		}},
		{"preferencia_repetida", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) {
			e.Solicitudes[0].Preferencias[1] = e.Solicitudes[0].Preferencias[0]
		}},
		{"vacante_ajena", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) {
			e.Solicitudes[0].Preferencias[0].VacanteRef = "vacante:ajena"
		}},
		{"valoracion_ausente", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) {
			e.Solicitudes[0].Preferencias[0].Valoracion = nil
		}},
		{"motor_ajeno", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) {
			e.Solicitudes[0].Preferencias[0].Valoracion.VersionMotor = "bolsa.v1"
		}},
		{"proceso_ajeno", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) {
			e.Solicitudes[0].Preferencias[0].Valoracion.ConvocatoriaRef = "concurso:otro"
		}},
		{"config_ajena", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) {
			e.Solicitudes[0].Preferencias[0].Valoracion.HuellaReglas = strings.Repeat("b", 64)
		}},
		{"instantanea_ajena", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) {
			e.Solicitudes[0].Preferencias[0].Valoracion.InstantaneaRef = "instantanea:otra"
		}},
		{"resultado_alterado", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) {
			e.Solicitudes[0].Preferencias[0].Valoracion.Total = new(b.Puntos)
		}},
		{"no_sintetico", func(a *d.ConfiguracionAdjudicacion, e *d.EntradaAdjudicacion) { e.Sintetico = false }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c, a := adjConfiguraciones(t)
			e := adjEntrada(t, c)
			caso.cambiar(&a, &e)
			if _, err := d.SimularAdjudicacion(a, e); err == nil {
				t.Fatal("contrato inválido aceptado")
			}
		})
	}
}

func TestAdjudicacionMetodoSinDefaultYUniversoVacio(t *testing.T) {
	c, a := adjConfiguraciones(t)
	e := adjEntrada(t, c)
	a.Metodo = "metodo:desconocido"
	r := adjEjecutar(t, a, e)
	if r.Estado != "no_soportado" || len(r.Asignaciones) != 0 {
		t.Fatal(r)
	}
	a.Metodo = d.MetodoAdjudicacionEnsayo
	e.Solicitudes = []d.SolicitudAdjudicacion{}
	r = adjEjecutar(t, a, e)
	if r.Estado != "propuesta_simulada" || len(r.Asignaciones) != 0 || len(r.VacantesSinAsignacion) != 2 {
		t.Fatal(r)
	}
}

func TestAdjudicacionPuntosDifierenEnUnMicropunto(t *testing.T) {
	c, a := adjConfiguraciones(t)
	c.Reglas[0].Coeficiente = concPuntos(t, 1)
	a.HuellaReglasValoracion = concCalcular(t, c, concEntrada()).HuellaReglas
	e := adjEntrada(t, c)
	e.Solicitudes[0].Preferencias[0].Valoracion = adjValoracion(t, c, "persona:a", "puesto:a", 1, 0, true)
	e.Solicitudes[1].Preferencias[0].Valoracion = adjValoracion(t, c, "persona:b", "puesto:a", 2, 0, true)
	if adjAsignaciones(adjEjecutar(t, a, e))["persona:b"] != "vacante:a" {
		t.Fatal("se perdió una diferencia exacta de un micropunto")
	}
}

func TestAdjudicacionLimitesCardinalidad(t *testing.T) {
	t.Run("personas", func(t *testing.T) {
		c, a := adjConfiguraciones(t)
		e := adjEntrada(t, c)
		e.Solicitudes = make([]d.SolicitudAdjudicacion, d.MaximoPersonasAdjudicacion+1)
		if _, err := d.SimularAdjudicacion(a, e); err == nil {
			t.Fatal("sin límite de personas")
		}
	})
	t.Run("vacantes", func(t *testing.T) {
		c, a := adjConfiguraciones(t)
		e := adjEntrada(t, c)
		e.Vacantes = make([]d.VacanteAdjudicacion, d.MaximoVacantesAdjudicacion+1)
		if _, err := d.SimularAdjudicacion(a, e); err == nil {
			t.Fatal("sin límite de vacantes")
		}
	})
	t.Run("preferencias_totales", func(t *testing.T) {
		_, a := adjConfiguraciones(t)
		e := d.EntradaAdjudicacion{UniversoRef: "universo:sintetico", UniversoVersion: "v1", Cerrado: true, Sintetico: true, Vacantes: []d.VacanteAdjudicacion{}, Solicitudes: []d.SolicitudAdjudicacion{}}
		for j := 0; j < 17; j++ {
			e.Vacantes = append(e.Vacantes, d.VacanteAdjudicacion{VacanteRef: fmt.Sprintf("vacante:%d", j), PuestoRef: fmt.Sprintf("puesto:%d", j)})
		}
		for i := 0; i < 256; i++ {
			s := d.SolicitudAdjudicacion{SolicitudRef: fmt.Sprintf("solicitud:%d", i), Version: "v1", PersonaRef: fmt.Sprintf("persona:%d", i), InstantaneaRef: fmt.Sprintf("instantanea:%d", i), Preferencias: []d.PreferenciaAdjudicacion{}}
			for _, v := range e.Vacantes {
				s.Preferencias = append(s.Preferencias, d.PreferenciaAdjudicacion{VacanteRef: v.VacanteRef, Admision: "excluida"})
			}
			e.Solicitudes = append(e.Solicitudes, s)
		}
		_, err := d.SimularAdjudicacion(a, e)
		nominal, ok := err.(*d.Error)
		if !ok || nominal.Codigo != "adjudicacion_entrada_excesiva" {
			t.Fatal(err)
		}
	})
}

func TestAdjudicacionRechazaDosVacantesDelMismoPuestoIndividual(t *testing.T) {
	c, a := adjConfiguraciones(t)
	e := adjEntrada(t, c)
	e.Vacantes[1].PuestoRef = e.Vacantes[0].PuestoRef
	// A solicita solo va y B solo vb. Ambas valoraciones proceden del motor
	// propio contra el mismo puesto individual y sus respectivas instantáneas.
	for i := range e.Solicitudes {
		s := &e.Solicitudes[i]
		v := e.Vacantes[i]
		s.Preferencias = []d.PreferenciaAdjudicacion{{VacanteRef: v.VacanteRef, Admision: "admitida", Valoracion: adjValoracion(t, c, s.PersonaRef, v.PuestoRef, 10, 0, true)}}
	}
	r, err := d.SimularAdjudicacion(a, e)
	nominal, ok := err.(*d.Error)
	if !ok || nominal.Codigo != "adjudicacion_vacante_invalida" || nominal.Campo != "vacantes" || len(r.Asignaciones) != 0 {
		t.Fatalf("puesto individual duplicado aceptado: resultado=%+v error=%v", r, err)
	}
}
