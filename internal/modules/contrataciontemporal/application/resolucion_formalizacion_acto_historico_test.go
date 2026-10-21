package application

import (
	"context"
	"errors"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type sesionActoAplicacionPrueba struct {
	*sesionPreparacionAplicacionPrueba
	acto ports.ResultadoResolucionFormalizacion
}

func (s *sesionActoAplicacionPrueba) ConsultarActoYRegistrarAcceso(ctx context.Context, o ports.OrdenConsultaDetalleRRHH) (ports.DetalleExpedienteRRHH, ports.ResultadoResolucionFormalizacion, error) {
	d, _, err := s.ConsultarPreparacionYRegistrarAcceso(ctx, o)
	return d, s.acto, err
}

func TestActoResolucionHistoricoConLecturaNominal(t *testing.T) {
	for _, caso := range []string{"v7_sin_acto", "v8", "v9", "v12", "otro_expediente", "acto_adulterado", "permiso_denegado"} {
		t.Run(caso, func(t *testing.T) {
			e := nuevoEntornoConsultaRRHH(t)
			configurarDetalleCompletoRRHHPrueba(t, e)
			d := e.sesion.detalle.Clonar()
			version := uint64(9)
			switch caso {
			case "v7_sin_acto":
				version = 7
			case "v8":
				version = 8
			case "v12":
				version = 12
			}
			for n := uint64(5); n <= version; n++ {
				h := d.Hitos[len(d.Hitos)-1]
				h.Secuencia, h.VersionExpediente = n, n
				h.AccionClave = "actuacion_ejercicio"
				h.FaseOrigen = h.FaseDestino
				if n == 7 {
					h.AccionClave, h.FaseDestino = "registrar_propuesta_formalizacion", "nombramiento"
				}
				if n == 8 {
					h.AccionClave = "registrar_resolucion_formalizacion"
				}
				d.Hitos = append(d.Hitos, h)
			}
			d.Resumen.Version, d.Resumen.FaseClave = version, "nombramiento"
			q := solicitudResolucionPrueba()
			q.ExpedienteRef = d.Resumen.ExpedienteRef
			r := ports.ResultadoResolucionFormalizacion{}
			if version >= 8 {
				r = reciboResolucionPrueba(q)
				r.RegistradaEn = d.Hitos[7].RealizadaEn
			}
			if caso == "otro_expediente" {
				r.Solicitud.ExpedienteRef = "expediente:ajeno"
			}
			if caso == "acto_adulterado" {
				r.DocumentoSHA256 = "invalida"
			}
			ra, _ := ports.NuevaReferenciaHitoAnalisisRRHH(2)
			rc, _ := ports.NuevaReferenciaHitoCoberturaRRHH(3)
			ru, _ := ports.NuevaReferenciaHitoAsignacionRRHH(4)
			entrada, err := ports.NuevaEntradaDetalleExpedienteRRHHMinimizada(d.Resumen, d.Solicitud, d.Analisis, ra, d.Cobertura, rc, d.Asignacion, ru, d.Hitos)
			if err != nil {
				t.Fatal(err)
			}
			sesion := &sesionActoAplicacionPrueba{
				sesionPreparacionAplicacionPrueba: &sesionPreparacionAplicacionPrueba{t: t, entrada: entrada, p: ports.PreparacionResolucionFormalizacion{VersionActual: version}},
				acto:                              r,
			}
			if caso == "permiso_denegado" {
				e.autoridad.err = ports.ErrConsultaRRHHNoObservable
			}
			svc, err := NuevoServicioPreparacionResolucionFormalizacion(e.autoridad, e.emisor, sesion, e.reloj)
			if err != nil {
				t.Fatal(err)
			}
			actual, err := svc.ConsultarActoResolucionFormalizacion(context.Background(), q.ExpedienteRef)
			if caso == "v8" || caso == "v9" || caso == "v12" {
				if err != nil || actual != r || sesion.llamadas != 1 || e.emision.detalle.llamadas != 1 {
					t.Fatal(actual, err)
				}
			} else if err == nil || actual != (ports.ResultadoResolucionFormalizacion{}) ||
				(caso == "permiso_denegado" && sesion.llamadas != 0) {
				t.Fatal("lectura histórica abierta", actual, err)
			}
			if caso == "permiso_denegado" && !errors.Is(err, ports.ErrConsultaRRHHNoObservable) {
				t.Fatal("permiso denegado perdió causa", err)
			}
		})
	}
}
