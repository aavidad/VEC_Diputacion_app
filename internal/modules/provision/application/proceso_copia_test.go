package application_test

import (
	"encoding/json"
	"testing"

	"vec-diputacion-granada/internal/modules/provision/application"
	"vec-diputacion-granada/internal/modules/provision/domain"
	"vec-diputacion-granada/internal/modules/provision/ports"
	b "vec-diputacion-granada/internal/shared/baremacion"
)

func procesoJSON(t *testing.T, v any) string {
	t.Helper()
	datos, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(datos)
}

func mutacionesProceso() []struct {
	nombre string
	mutar  func(*domain.ProcesoProvision)
} {
	return []struct {
		nombre string
		mutar  func(*domain.ProcesoProvision)
	}{
		{"rpt", func(p *domain.ProcesoProvision) { p.Puestos[0].RPTVersion = "cambiada" }},
		{"requisito_puesto", func(p *domain.ProcesoProvision) { p.Puestos[0].Requisitos[0].Version = "cambiada" }},
		{"regla", func(p *domain.ProcesoProvision) { p.Configuracion.Reglas[0].ID = "cambiada" }},
		{"tramo", func(p *domain.ProcesoProvision) { p.Configuracion.Reglas[0].Tramos[0].ID = "cambiada" }},
		{"conversion", func(p *domain.ProcesoProvision) { p.Configuracion.Reglas[1].Conversion.Divisor++ }},
		{"tipos", func(p *domain.ProcesoProvision) { p.Configuracion.Reglas[3].Tipos[0] = "cambiado" }},
		{"horas_minimas", func(p *domain.ProcesoProvision) {
			*p.Configuracion.Reglas[4].HorasMinimas = b.Racional{}
		}},
	}
}

func TestPrepararProcesoConservaCopiaProfundaEnAmbosSentidos(t *testing.T) {
	for _, caso := range mutacionesProceso() {
		for _, modificarSalida := range []bool{false, true} {
			t.Run(caso.nombre+map[bool]string{false: "_entrada", true: "_salida"}[modificarSalida], func(t *testing.T) {
				p := peticionProceso(t).Proceso
				r, err := application.PrepararProceso(p)
				if err != nil {
					t.Fatal(err)
				}
				if modificarSalida {
					antes := procesoJSON(t, p)
					caso.mutar(&r)
					if procesoJSON(t, p) != antes {
						t.Fatal("la salida modifica el borrador recibido")
					}
				} else {
					antes := procesoJSON(t, r)
					caso.mutar(&p)
					if procesoJSON(t, r) != antes {
						t.Fatal("la entrada modifica el borrador preparado")
					}
				}
			})
		}
	}
}

func TestSimularProcesoConservaCopiaProfundaEnAmbosSentidos(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*ports.PeticionProceso)
	}{}
	for _, c := range mutacionesProceso() {
		casos = append(casos, struct {
			nombre string
			mutar  func(*ports.PeticionProceso)
		}{c.nombre, func(p *ports.PeticionProceso) { c.mutar(&p.Proceso) }})
	}
	casos = append(casos, []struct {
		nombre string
		mutar  func(*ports.PeticionProceso)
	}{
		{"preferencias", func(p *ports.PeticionProceso) { p.Solicitud.Preferencias[0].Orden = 9 }},
		{"comprobaciones", func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Requisitos[0].Estado = domain.NoCumple }},
		{"grado", func(p *ports.PeticionProceso) { *p.Solicitud.Valoraciones[0].Entrada.GradoPersonal = 99 }},
		{"disponibles", func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Entrada.Disponibles[0] = domain.Cursos }},
		{"periodo", func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Entrada.Periodos[0].ID = "cambiado" }},
		{"fecha_periodo", func(p *ports.PeticionProceso) {
			*p.Solicitud.Valoraciones[0].Entrada.Periodos[0].Hasta = p.Proceso.Configuracion.VentanaDesde
		}},
		{"familias_periodo", func(p *ports.PeticionProceso) {
			p.Solicitud.Valoraciones[0].Entrada.Periodos[0].Familias[0] = domain.Cursos
		}},
		{"curso", func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Entrada.Cursos[0].ID = "cambiado" }},
		{"fecha_curso", func(p *ports.PeticionProceso) {
			*p.Solicitud.Valoraciones[0].Entrada.Cursos[0].VigenteHasta = p.Proceso.Configuracion.VentanaDesde
		}},
		{"titulo", func(p *ports.PeticionProceso) { p.Solicitud.Valoraciones[0].Entrada.Titulaciones[0].ID = "cambiado" }},
	}...)
	for _, c := range casos {
		for _, modificarSalida := range []bool{false, true} {
			t.Run(c.nombre+map[bool]string{false: "_entrada", true: "_salida"}[modificarSalida], func(t *testing.T) {
				p := peticionProceso(t)
				fecha := p.Proceso.Configuracion.FechaCorte
				p.Solicitud.Valoraciones[0].Entrada.Cursos[0].VigenteHasta = &fecha
				r, err := application.SimularProceso(p)
				if err != nil {
					t.Fatal(err)
				}
				if modificarSalida {
					antes := procesoJSON(t, p)
					// Ambas ramas del resultado pertenecen a su propia instantánea.
					copiaSalida := ports.PeticionProceso{Proceso: r.Proceso, Solicitud: r.Solicitud}
					c.mutar(&copiaSalida)
					if procesoJSON(t, p) != antes {
						t.Fatal("el resultado modifica la entrada")
					}
				} else {
					antes := procesoJSON(t, r)
					c.mutar(&p)
					if procesoJSON(t, r) != antes || r.HuellaSimulacion != domain.HuellaSimulacionProceso(r) {
						t.Fatal("la entrada altera la instantánea o deja huella obsoleta")
					}
				}
			})
		}
	}
}

func TestSimularProcesoComprobacionesSalidaIndependientes(t *testing.T) {
	p := peticionProceso(t)
	r, err := application.SimularProceso(p)
	if err != nil {
		t.Fatal(err)
	}
	originalSolicitud := procesoJSON(t, r.Solicitud)
	originalPeticion := procesoJSON(t, p)
	r.Valoraciones[1].Requisitos[0].Estado = domain.NoCumple
	if procesoJSON(t, r.Solicitud) != originalSolicitud || procesoJSON(t, p) != originalPeticion {
		t.Fatal("el desglose modifica solicitud o petición")
	}
	if r.Valoraciones[0].RequisitosEstado != domain.Cumple {
		t.Fatal("modifica otro puesto")
	}
	r, err = application.SimularProceso(p)
	if err != nil {
		t.Fatal(err)
	}
	originalDesglose := procesoJSON(t, r.Valoraciones)
	r.Solicitud.Valoraciones[0].Requisitos[0].Estado = domain.NoCumple
	if procesoJSON(t, r.Valoraciones) != originalDesglose {
		t.Fatal("la solicitud modifica las comprobaciones del desglose")
	}
}
