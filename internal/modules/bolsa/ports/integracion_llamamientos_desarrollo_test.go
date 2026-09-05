package ports

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/domain"
)

func resolucionIntegracionPrueba() ResolucionLlamamientoDesarrollo {
	return ResolucionLlamamientoDesarrollo{
		AperturaOperacionRef: "operacion:apertura", JustificanteRef: "justificante:unidad",
		EvaluacionPlazoRef: "evaluacion:unidad", PoliticaRef: "politica:unidad", PoliticaVersion: 1,
		PoliticaSHA256: strings.Repeat("a", 64), VersionEsperada: 1,
		ResueltaEn: instanteComandoLlamamientoPrueba,
	}
}

func TestIntegracionLlamamientosDesarrolloResolucionContratoYFechas(t *testing.T) {
	r := resolucionIntegracionPrueba()
	if err := r.Validar(); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]json.RawMessage
	if err := json.Unmarshal(b, &campos); err != nil || len(campos) != 8 {
		t.Fatal("resolución no conserva los ocho campos acordados")
	}
	for _, nombre := range []string{"apertura_operacion_ref", "justificante_ref", "evaluacion_plazo_ref", "politica_ref", "politica_version", "politica_sha256", "version_esperada", "resuelta_en"} {
		if _, ok := campos[nombre]; !ok {
			t.Fatalf("falta %s", nombre)
		}
	}
	casos := map[string]func(*ResolucionLlamamientoDesarrollo){
		"sin_apertura":            func(r *ResolucionLlamamientoDesarrollo) { r.AperturaOperacionRef = "" },
		"sin_justificante":        func(r *ResolucionLlamamientoDesarrollo) { r.JustificanteRef = "" },
		"sin_evaluacion":          func(r *ResolucionLlamamientoDesarrollo) { r.EvaluacionPlazoRef = "" },
		"sin_politica":            func(r *ResolucionLlamamientoDesarrollo) { r.PoliticaRef = "" },
		"version_politica_cero":   func(r *ResolucionLlamamientoDesarrollo) { r.PoliticaVersion = 0 },
		"version_politica_grande": func(r *ResolucionLlamamientoDesarrollo) { r.PoliticaVersion = 1 << 53 },
		"huella_cero":             func(r *ResolucionLlamamientoDesarrollo) { r.PoliticaSHA256 = strings.Repeat("0", 64) },
		"huella_no_canonica":      func(r *ResolucionLlamamientoDesarrollo) { r.PoliticaSHA256 = strings.Repeat("A", 64) },
		"otra_version":            func(r *ResolucionLlamamientoDesarrollo) { r.VersionEsperada = 2 },
		"sin_fecha":               func(r *ResolucionLlamamientoDesarrollo) { r.ResueltaEn = time.Time{} },
		"local":                   func(r *ResolucionLlamamientoDesarrollo) { r.ResueltaEn = r.ResueltaEn.In(time.Local) },
		"offset_cero": func(r *ResolucionLlamamientoDesarrollo) {
			r.ResueltaEn = r.ResueltaEn.In(time.FixedZone("no-canonica", 0))
		},
		"submicrosegundo": func(r *ResolucionLlamamientoDesarrollo) { r.ResueltaEn = r.ResueltaEn.Add(time.Nanosecond) },
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			alterada := r
			cambiar(&alterada)
			if alterada.Validar() == nil {
				t.Fatal("resolución inválida admitida")
			}
		})
	}
	p := PeticionResolverLlamamientoDesarrollo{OperacionRef: "operacion:aceptacion", Resolucion: r}
	if p.Validar() == nil {
		t.Fatal("la petición no puede fijar la fecha")
	}
	p.Resolucion.ResueltaEn = time.Time{}
	if err := p.Validar(); err != nil {
		t.Fatal(err)
	}
	p.OperacionRef = p.Resolucion.AperturaOperacionRef
	if p.Validar() == nil {
		t.Fatal("aceptación no puede sustituir a la apertura")
	}
}

func TestIntegracionLlamamientosDesarrolloCanonAceptacionYLigaduras(t *testing.T) {
	for _, caso := range []struct {
		tipo   string
		estado domain.EstadoLlamamiento
		accion string
	}{
		{"aceptacion_rrhh", domain.EstadoLlamamientoAceptado, AccionAceptarLlamamientoRRHHDesarrollo},
		{"renuncia_rrhh", domain.EstadoLlamamientoRenunciado, AccionRenunciarLlamamientoRRHHDesarrollo},
	} {
		t.Run(caso.tipo, func(t *testing.T) {
			i, propuesta, _ := materialesComandoLlamamientoPrueba(t, "aceptacion")
			r := RegistroLlamamientoDesarrollo{
				Esquema: "vec.bolsa.integracion-llamamientos-desarrollo.v1", OperacionRef: "operacion:apertura", Tipo: "propuesta",
				OrdenOperacionRef: "operacion:orden", NecesidadRef: propuesta.NecesidadRef, VersionNecesidad: propuesta.VersionNecesidad,
				CategoriaRef: "categoria:comando", UnidadRef: "unidad:comando", Fuente: json.RawMessage(`{}`), FirmaFuente: make([]byte, 64),
				Instantanea: i, Propuesta: &propuesta,
				Llamamiento:       &domain.DatosLlamamientoAbierto{LlamamientoRef: "llamamiento:unidad", BolsaRef: i.BolsaRef, NecesidadRef: propuesta.NecesidadRef, PropuestaRef: propuesta.PropuestaRef, Version: 1},
				EstadoLlamamiento: domain.EstadoLlamamientoAbierto,
			}
			anterior, err := r.Canonico()
			if err != nil || bytes.Contains(anterior, []byte(`"resolucion"`)) {
				t.Fatal("el canon de apertura debe omitir resolución")
			}
			resolucion := resolucionIntegracionPrueba()
			r.Resolucion = &resolucion
			if _, err := r.Canonico(); err == nil {
				t.Fatal("apertura no admite resolución terminal")
			}
			r.OperacionRef, r.Tipo = "operacion:terminal", caso.tipo
			r.Llamamiento.Version, r.EstadoLlamamiento = 2, caso.estado
			canon, err := r.Canonico()
			if err != nil || !bytes.Contains(canon, []byte(`"resolucion":`)) || r.Accion() != caso.accion {
				t.Fatalf("terminal válido rechazado: %v", err)
			}
			cruzado := r
			if caso.tipo == "aceptacion_rrhh" {
				cruzado.EstadoLlamamiento = domain.EstadoLlamamientoRenunciado
			} else {
				cruzado.EstadoLlamamiento = domain.EstadoLlamamientoAceptado
			}
			if _, err := cruzado.Canonico(); err == nil || cruzado.Accion() != "" {
				t.Fatal("tipo y estado cruzados obtuvieron canon o permiso")
			}
			cruzado = r
			cruzado.Tipo = "expiracion_rrhh"
			if _, err := cruzado.Canonico(); err == nil || cruzado.Accion() != "" {
				t.Fatal("tercer tipo terminal no autorizado")
			}
			recurso, err := r.RecursoAutorizable()
			if err != nil || recurso.Referencia != r.OperacionRef || recurso.Tipo != "integracion_llamamientos_bolsa" || recurso.ModuloID != "bolsa" {
				t.Fatal("recurso de aceptación divergente")
			}
			r.Resolucion.EvaluacionPlazoRef = "evaluacion:otra"
			otro, err := r.RecursoAutorizable()
			if err != nil || otro.Atributos["contenido_sha256"] == recurso.Atributos["contenido_sha256"] {
				t.Fatal("evaluación no queda ligada al material autorizado")
			}
			for _, cambiar := range []func(*RegistroLlamamientoDesarrollo){
				func(r *RegistroLlamamientoDesarrollo) { r.Resolucion = nil },
				func(r *RegistroLlamamientoDesarrollo) { r.Llamamiento = nil },
				func(r *RegistroLlamamientoDesarrollo) { r.EstadoLlamamiento = domain.EstadoLlamamientoAbierto },
				func(r *RegistroLlamamientoDesarrollo) { r.Propuesta = nil },
			} {
				alterado := r
				cambiar(&alterado)
				if _, err := alterado.Canonico(); err == nil {
					t.Fatal("aceptación parcial admitida")
				}
			}
			r.Resolucion.ResueltaEn = r.Propuesta.GeneradaEn.Add(-time.Microsecond)
			if _, err := r.Canonico(); err == nil {
				t.Fatal("aceptación anterior a su apertura admitida")
			}
		})
	}
}
