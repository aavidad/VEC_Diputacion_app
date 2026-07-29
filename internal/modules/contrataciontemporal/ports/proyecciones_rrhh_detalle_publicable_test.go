package ports_test

import (
	"errors"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDetalleRRHHValidaContenidoPublicableContraSolicitud(t *testing.T) {
	for bloques := 0; bloques <= 3; bloques++ {
		entrada, lectura := entradaDetalleMinimizadaPrueba(t, bloques)
		detalle, err := ports.NuevoDetalleExpedienteRRHHMinimizado(
			entrada,
			lectura,
		)
		if err != nil {
			t.Fatalf("construir detalle con %d bloques: %v", bloques, err)
		}
		solicitud, err := ports.NuevaSolicitudDetalleRRHH(
			detalle.Resumen.ExpedienteRef,
			detalle.Resumen.Version,
		)
		if err != nil {
			t.Fatal(err)
		}
		if err := detalle.ValidarContenidoPublicablePara(
			solicitud,
		); err != nil {
			t.Fatalf("detalle con %d bloques rechazado: %v", bloques, err)
		}
	}
}

func TestDetalleRRHHRechazaCadaBloquePublicableCorrupto(t *testing.T) {
	entrada, lectura := entradaDetalleMinimizadaPrueba(t, 3)
	nominal, err := ports.NuevoDetalleExpedienteRRHHMinimizado(
		entrada,
		lectura,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitud, err := ports.NuevaSolicitudDetalleRRHH(
		nominal.Resumen.ExpedienteRef,
		nominal.Resumen.Version,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudRefAjena, err := ports.NuevaSolicitudDetalleRRHH(
		"expediente:rrhh:ajeno",
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	solicitudVersionAjena, err := ports.NuevaSolicitudDetalleRRHH(
		nominal.Resumen.ExpedienteRef,
		nominal.Resumen.Version-1,
	)
	if err != nil {
		t.Fatal(err)
	}

	casos := []struct {
		nombre    string
		detalle   ports.DetalleExpedienteRRHH
		solicitud ports.SolicitudDetalleRRHH
	}{
		{
			"solicitud operativa",
			func() ports.DetalleExpedienteRRHH {
				detalle := nominal.Clonar()
				detalle.Solicitud.PeriodoFin =
					detalle.Solicitud.PeriodoInicio.Add(-time.Second)
				return detalle
			}(),
			solicitud,
		},
		{
			"análisis",
			func() ports.DetalleExpedienteRRHH {
				detalle := nominal.Clonar()
				detalle.Analisis.ModalidadClave = ""
				return detalle
			}(),
			solicitud,
		},
		{
			"cobertura",
			func() ports.DetalleExpedienteRRHH {
				detalle := nominal.Clonar()
				detalle.Cobertura.Comprobaciones[0].Resultado = "desconocida"
				return detalle
			}(),
			solicitud,
		},
		{
			"asignación",
			func() ports.DetalleExpedienteRRHH {
				detalle := nominal.Clonar()
				detalle.Asignacion.UnidadRef = "!"
				return detalle
			}(),
			solicitud,
		},
		{
			"hitos",
			func() ports.DetalleExpedienteRRHH {
				detalle := nominal.Clonar()
				detalle.Hitos[1].FaseOrigen = "fase_ajena"
				return detalle
			}(),
			solicitud,
		},
		{"referencia ajena", nominal, solicitudRefAjena},
		{"versión ajena", nominal, solicitudVersionAjena},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if err := caso.detalle.ValidarContenidoPublicablePara(
				caso.solicitud,
			); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
				t.Fatalf("detalle corrupto aceptado: %v", err)
			}
		})
	}
}
