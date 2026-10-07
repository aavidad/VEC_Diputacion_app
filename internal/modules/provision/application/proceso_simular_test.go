package application_test

import (
	"encoding/json"
	"reflect"
	"testing"
	"vec-diputacion-granada/internal/modules/provision/adapters/simulacion"
	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
	"vec-diputacion-granada/internal/modules/provision/ports"
)

func peticionProceso(t *testing.T) ports.PeticionProceso {
	t.Helper()
	p, err := simulacion.EjemploProceso()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSimularProcesoReutilizaMotorPorPreferencia(t *testing.T) {
	p := peticionProceso(t)
	antes, _ := json.Marshal(p)
	r, err := application.SimularProceso(p)
	if err != nil {
		t.Fatal(err)
	}
	if r.Alcance != "simulacion" || r.Estado != "borrador" || r.SchemaVersion != domain.VersionProceso || len(r.Valoraciones) != 2 {
		t.Fatalf("resultado %#v", r)
	}
	for i, v := range r.Valoraciones {
		if v.Orden != i+1 || v.PuestoRef != p.Solicitud.Preferencias[i].PuestoRef || v.RequisitosEstado != domain.Cumple {
			t.Fatalf("preferencia %#v", v)
		}
		var entrada domain.Entrada
		for _, x := range p.Solicitud.Valoraciones {
			if x.PuestoRef == v.PuestoRef {
				entrada = x.Entrada
			}
		}
		motor, err := application.Simular(p.Proceso.Configuracion, entrada)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(motor, v.Resultado) {
			t.Fatal("no reutiliza resultado motor")
		}
	}
	despues, _ := json.Marshal(p)
	if string(antes) != string(despues) {
		t.Fatal("modifica instantanea original")
	}
	replay, err := application.SimularProceso(p)
	if err != nil || !reflect.DeepEqual(r, replay) {
		t.Fatal("no determinista", err)
	}
}

func TestProcesoRechazaVinculosYPreferenciasInvalidos(t *testing.T) {
	casos := []struct {
		nombre  string
		cambiar func(*ports.PeticionProceso)
	}{
		{"publicado", func(p *ports.PeticionProceso) { p.Proceso.Estado = "publicado" }},
		{"config_otro_proceso", func(p *ports.PeticionProceso) { p.Proceso.Configuracion.ConvocatoriaRef = "otro" }},
		{"proceso_otra_version", func(p *ports.PeticionProceso) { p.Solicitud.ProcesoVersion = "otra" }},
		{"reglas_otra_version", func(p *ports.PeticionProceso) { p.Solicitud.VersionReglas = "otra" }},
		{"empleado_distinto", func(p *ports.PeticionProceso) { p.Solicitud.Instantanea.EmpleadoRef = "otro" }},
		{"empleado_externo", func(p *ports.PeticionProceso) { p.Solicitud.Instantanea.CondicionInterna = domain.NoCumple }},
		{"empleado_pendiente", func(p *ports.PeticionProceso) { p.Solicitud.Instantanea.CondicionInterna = domain.Pendiente }},
		{"preferencia_duplicada", func(p *ports.PeticionProceso) {
			p.Solicitud.Preferencias[1].PuestoRef = p.Solicitud.Preferencias[0].PuestoRef
		}},
		{"preferencia_salto", func(p *ports.PeticionProceso) { p.Solicitud.Preferencias[1].Orden = 3 }},
		{"puesto_ajeno", func(p *ports.PeticionProceso) { p.Solicitud.Preferencias[0].PuestoRef = "otro" }},
		{"rpt_sin_version", func(p *ports.PeticionProceso) { p.Proceso.Puestos[0].RPTVersion = "" }},
		{"rpt_duplicada", func(p *ports.PeticionProceso) { p.Proceso.Puestos[1].RPTRef = p.Proceso.Puestos[0].RPTRef }},
		{"nivel_ajeno", func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Entrada.NivelPuesto++ }},
		{"instantanea_ajena", func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Entrada.InstantaneaRef = "otra" }},
		{"entrada_puesto_ajeno", func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Entrada.PuestoRef = "otro" }},
		{"requisito_otra_version", func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Requisitos[0].RequisitoVersion = "otra" }},
		{"requisito_omitido", func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Requisitos = nil }},
		{"estado_desconocido", func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Requisitos[0].Estado = "admitido" }},
		{"procedencia_omitida", func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Requisitos[0].FuenteRef = "" }},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			p := peticionProceso(t)
			c.cambiar(&p)
			r, err := application.SimularProceso(p)
			if err == nil || r.SchemaVersion != "" {
				t.Fatal("acepta entrada invalida", err)
			}
		})
	}
}

func TestProcesoFuentesAusentesPendientesSinTotal(t *testing.T) {
	p := peticionProceso(t)
	p.Solicitud.Valoraciones[0].Entrada.Disponibles = []domain.Familia{}
	p.Solicitud.Valoraciones[0].Entrada.GradoPersonal = nil
	p.Solicitud.Valoraciones[0].Entrada.GradoEvidenciaRef = ""
	p.Solicitud.Valoraciones[0].Entrada.Periodos = []domain.Periodo{}
	p.Solicitud.Valoraciones[0].Entrada.Cursos = []domain.Curso{}
	p.Solicitud.Valoraciones[0].Entrada.Titulaciones = []domain.Titulo{}
	p.Solicitud.Valoraciones[0].Requisitos[0].Estado = domain.Pendiente
	p.Solicitud.Valoraciones[0].Requisitos[0].MotivoCodigo = "fuente_no_disponible"
	r, err := application.SimularProceso(p)
	if err != nil {
		t.Fatal(err)
	}
	v := r.Valoraciones[1]
	if v.Resultado.Total != nil || v.Resultado.Completo || v.RequisitosEstado != domain.Pendiente {
		t.Fatal("dato ausente presentado como resultado completo")
	}
	for _, d := range v.Resultado.Desglose {
		if d.Estado != "pendiente_dato" {
			t.Fatal("dato ausente convertido a cero calculado")
		}
	}
	if !r.Valoraciones[0].Resultado.Completo {
		t.Fatal("afecta otro puesto")
	}
}

func TestRequisitoIncumplidoNoDeduceExclusionNiPuntos(t *testing.T) {
	p := peticionProceso(t)
	p.Solicitud.Valoraciones[0].Requisitos[0].Estado = domain.NoCumple
	r, err := application.SimularProceso(p)
	if err != nil {
		t.Fatal(err)
	}
	v := r.Valoraciones[1]
	if v.RequisitosEstado != domain.NoCumple || !v.Resultado.Completo || v.Resultado.Total == nil || r.Estado != "borrador" {
		t.Fatal("mezcla acceso con puntos/admisión")
	}
}

func TestHuellaProcesoVinculaPreferenciasRequisitosYVersionRPT(t *testing.T) {
	p := peticionProceso(t)
	r, err := application.SimularProceso(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.HuellaSimulacion) != 64 || r.HuellaSimulacion != domain.HuellaSimulacionProceso(r) {
		t.Fatal("huella incorrecta")
	}
	cambios := []func(*ports.PeticionProceso){
		func(p *ports.PeticionProceso) { p.Proceso.Puestos[0].RPTVersion = "2-sintetica" },
		func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Requisitos[0].Estado = domain.Pendiente },
		func(p *ports.PeticionProceso) {
			p.Solicitud.Preferencias[0].PuestoRef, p.Solicitud.Preferencias[1].PuestoRef = p.Solicitud.Preferencias[1].PuestoRef, p.Solicitud.Preferencias[0].PuestoRef
		},
	}
	for _, cambiar := range cambios {
		nuevo := peticionProceso(t)
		cambiar(&nuevo)
		rr, err := application.SimularProceso(nuevo)
		if err != nil || rr.HuellaSimulacion == r.HuellaSimulacion {
			t.Fatal("huella no conserva entrada completa", err)
		}
	}
}
