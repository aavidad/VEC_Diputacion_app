package application

import (
	"context"
	"errors"
	"slices"
	"testing"
	"vec-diputacion-granada/internal/modules/carrera/domain"
	"vec-diputacion-granada/internal/modules/carrera/ports"
)

func TestExpedienteReunePoliticaAntecedentesYContradiccionesSinPrestarAprobacion(t *testing.T) {
	e, a := escenarioAntecedentes()
	n := 24
	e.Casos[0].NivelPuesto = &n
	e.Casos[0].Regimen = "laboral_fijo"
	a.Servicios = append(a.Servicios, a.Servicios[0])
	a.Servicios[1].Referencia = "servicio-solapado"
	a.Servicios = append(a.Servicios, ports.ServicioAntecedente{Referencia: "declarado", Estado: "declarado", Periodo: a.Servicios[0].Periodo})
	c := domain.CatalogoPoliticaGrado{Alcance: domain.AlcanceSintetico, Politica: domain.Politica{Referencia: "politica-sintetica", Version: "v1", AprobacionReferencia: "aprobacion-aportada"}}
	out, err := (Servicio{}).PrepararExpedienteSintetico(context.Background(), e, lectorAntecedentesFunc(func(context.Context, ports.ConsultaAntecedentesSinteticos) (ports.InstantaneaAntecedentesSinteticos, error) {
		return a, nil
	}), c)
	if err != nil {
		t.Fatal(err)
	}
	p, r := out.Preparacion.Casos[0], out.Casos[0]
	if p.EstadoGlobal != "pendiente" || *p.NivelPuesto != 22 || *p.GradoPersonal != 20 || p.Antecedentes.Politica.AprobacionReferencia != "" || out.Politica.Datos.Politica.AprobacionReferencia != "aprobacion-aportada" {
		t.Fatal("confunde datos y aprobación")
	}
	if !slices.Contains(r.Contradicciones, "nivel_puesto") || !slices.Contains(r.Contradicciones, "regimen") || !slices.Contains(r.Antecedentes.Faltantes, "servicio:declarado") || !slices.Contains(r.Antecedentes.Faltantes, "autorizacion_carrera_h08") {
		t.Fatal("pierde diferencias o faltantes")
	}
	if len(p.Antecedentes.Periodos) != 2 || len(r.Antecedentes.Instantanea.Servicios) != 3 {
		t.Fatal("suma periodos o pierde declaración")
	}
	solape := false
	for _, check := range p.Comprobaciones {
		if check.MotivoClave == "carrera.motivo.periodos_solapados" {
			solape = true
		}
	}
	if !solape {
		t.Fatal("omite solape")
	}
	n = 99
	a.Grado.Valor = 99
	a.Fuentes[0].Version = "otra"
	c.Politica.Referencia = "otra"
	if *r.Declaracion.NivelPuesto != 24 || *p.NivelPuesto != 22 || *p.GradoPersonal != 20 || out.Politica.Datos.Politica.Referencia != "politica-sintetica" || r.Antecedentes.Instantanea.Fuentes[0].Version != "v2" || e.Casos[0].Regimen != "laboral_fijo" {
		t.Fatal("comparte salida o modifica declaración")
	}
}

func TestExpedienteFallaSinSalidaParcialYPoliticaInvalidaNoConsulta(t *testing.T) {
	e, a := escenarioAntecedentes()
	for _, tipo := range []string{"politica", "caida", "version", "cancelado", "produccion"} {
		t.Run(tipo, func(t *testing.T) {
			c := domain.CatalogoPoliticaGrado{Alcance: domain.AlcanceSintetico}
			ctx := context.Background()
			var fallo error
			switch tipo {
			case "politica":
				c.Alcance = "produccion"
			case "caida":
				fallo = errors.New("dato privado")
			case "version":
				a.Version = "otra"
			case "cancelado":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "produccion":
				e.Alcance = "produccion"
			}
			out, err := (Servicio{}).PrepararExpedienteSintetico(ctx, e, lectorAntecedentesFunc(func(context.Context, ports.ConsultaAntecedentesSinteticos) (ports.InstantaneaAntecedentesSinteticos, error) {
				if tipo == "politica" || tipo == "produccion" || tipo == "cancelado" {
					t.Fatal("consulta inesperada")
				}
				return a, fallo
			}), c)
			if err == nil || len(out.Casos) != 0 || len(out.Preparacion.Casos) != 0 {
				t.Fatal("no deniega o entrega parcialmente")
			}
		})
	}
}
