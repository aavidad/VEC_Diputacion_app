package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func solicitudResolucionPrueba() ports.SolicitudResolucionFormalizacion {
	return ports.SolicitudResolucionFormalizacion{ExpedienteRef: "expediente:prueba", VersionEsperada: 7, PropuestaRef: "propuesta:prueba",
		ClaveIdempotencia: "31111111-1111-4111-8111-111111111111", NumeroResolucion: "EJ-2026/1", FechaResolucion: "2026-09-06",
		Motivo: "Revisión manual del ejercicio.", ConfirmaRevisionPropuesta: true, ConfirmaEjercicioManual: true}
}
func reciboResolucionPrueba(s ports.SolicitudResolucionFormalizacion) ports.ResultadoResolucionFormalizacion {
	return ports.ResultadoResolucionFormalizacion{Solicitud: s, Estado: "registrada", ResolucionRef: "resolucion:prueba",
		DocumentoRef: ports.ReferenciaDocumentoResolucion(s.PropuestaRef), DocumentoSHA256: strings.Repeat("a", 64), DocumentoVersion: 7,
		ActuacionRef: "resolucion:prueba", AuditoriaRef: "auditoria:prueba", OutboxRef: "evento:prueba", ReciboRef: "recibo:prueba",
		VersionResultante: 8, RegistradaEn: time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)}
}

type sesionPreparacionAplicacionPrueba struct {
	t        *testing.T
	entrada  ports.EntradaDetalleExpedienteRRHHMinimizada
	p        ports.PreparacionResolucionFormalizacion
	llamadas int
	err      error
	cancelar context.CancelFunc
}

func (s *sesionPreparacionAplicacionPrueba) ConsultarPreparacionYRegistrarAcceso(_ context.Context, o ports.OrdenConsultaDetalleRRHH) (ports.DetalleExpedienteRRHH, ports.PreparacionResolucionFormalizacion, error) {
	s.llamadas++
	if o.Solicitud().VersionObservada() != 0 {
		s.t.Fatal("no lee versión actual")
	}
	lectura := reciboConsultaRRHHPrueba(s.t, o.Contexto(), o.Capacidad(), o.Instante(), o.Solicitud().ExpedienteRef(), s.p.VersionActual, 1)
	d, e := ports.NuevoDetalleExpedienteRRHHMinimizado(s.entrada, lectura)
	if e != nil {
		s.t.Fatal(e)
	}
	if s.cancelar != nil {
		s.cancelar()
	}
	return d, s.p, s.err
}
func TestResolucionFormalizacionPreparacionNominalAplicacion(t *testing.T) {
	for _, caso := range []string{"v7", "v8", "denegado", "recibo_ajeno", "sesion_falla", "cancelado"} {
		t.Run(caso, func(t *testing.T) {
			entorno := nuevoEntornoConsultaRRHH(t)
			configurarDetalleCompletoRRHHPrueba(t, entorno)
			d := entorno.sesion.detalle.Clonar()
			v := uint64(7)
			if caso == "v8" {
				v = 8
			}
			for i := uint64(5); i <= v; i++ {
				h := d.Hitos[len(d.Hitos)-1]
				h.Secuencia = i
				h.VersionExpediente = i
				h.AccionClave = "actuacion_ejercicio"
				h.FaseOrigen = h.FaseDestino
				if i == 7 {
					h.AccionClave = "registrar_propuesta_formalizacion"
					h.FaseDestino = "nombramiento"
				}
				if i == 8 {
					h.AccionClave = "registrar_resolucion_formalizacion"
				}
				d.Hitos = append(d.Hitos, h)
			}
			d.Resumen.Version = v
			d.Resumen.FaseClave = "nombramiento"
			ra, _ := ports.NuevaReferenciaHitoAnalisisRRHH(2)
			rc, _ := ports.NuevaReferenciaHitoCoberturaRRHH(3)
			ru, _ := ports.NuevaReferenciaHitoAsignacionRRHH(4)
			in, e := ports.NuevaEntradaDetalleExpedienteRRHHMinimizada(d.Resumen, d.Solicitud, d.Analisis, ra, d.Cobertura, rc, d.Asignacion, ru, d.Hitos)
			if e != nil {
				t.Fatal("entrada", e)
			}
			q := solicitudResolucionPrueba()
			q.ExpedienteRef = d.Resumen.ExpedienteRef
			p := ports.PreparacionResolucionFormalizacion{ExpedienteRef: q.ExpedienteRef, PropuestaRef: q.PropuestaRef, VersionEsperada: 7, VersionActual: v}
			if v == 8 {
				r := reciboResolucionPrueba(q)
				r.RegistradaEn = entorno.ahora
				p.Recibo = &r
			}
			sesion := &sesionPreparacionAplicacionPrueba{t: t, entrada: in, p: p}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			switch caso {
			case "denegado":
				entorno.autoridad.err = ports.ErrConsultaRRHHNoObservable
			case "recibo_ajeno":
				sesion.p.ExpedienteRef = "expediente:ajeno"
			case "sesion_falla":
				sesion.err = ports.ErrConsultaRRHHNoDisponible
			case "cancelado":
				sesion.cancelar = cancel
			}
			svc, e := NuevoServicioPreparacionResolucionFormalizacion(entorno.autoridad, entorno.emisor, sesion, entorno.reloj)
			if e != nil {
				t.Fatal(e)
			}
			r, e := svc.ConsultarPreparacionResolucionFormalizacion(ctx, q.ExpedienteRef)
			if caso == "v7" || caso == "v8" {
				if e != nil || r.ValidarPara(q.ExpedienteRef) != nil || sesion.llamadas != 1 ||
					entorno.emision.detalle.llamadas != 1 || entorno.emision.cuadro.llamadas != 0 {
					t.Fatal(r, e)
				}
				if r.Recibo != nil {
					r.Recibo.ReciboRef = "recibo:mutado"
					if sesion.p.Recibo.ReciboRef == r.Recibo.ReciboRef {
						t.Fatal("sin copia")
					}
				}
			} else if e == nil || r.ExpedienteRef != "" || (caso == "denegado" && sesion.llamadas != 0) {
				t.Fatal("fallo abierto", r, e)
			}
		})
	}
}

type transaccionResolucionPrueba struct {
	ejecutar func(context.Context, ports.SolicitudResolucionFormalizacion) (ports.ResultadoResolucionFormalizacion, error)
	llamadas int
}

func (t *transaccionResolucionPrueba) RegistrarResolucionFormalizacion(c context.Context, s ports.SolicitudResolucionFormalizacion) (ports.ResultadoResolucionFormalizacion, error) {
	t.llamadas++
	return t.ejecutar(c, s)
}
func TestResolucionFormalizacionServicioCancelacionYFalloAtomico(t *testing.T) {
	s := solicitudResolucionPrueba()
	for _, caso := range []string{"registro", "replay", "cancelado", "cancelacion_despues", "error_con_recibo", "recibo_ajeno", "confirmacion"} {
		t.Run(caso, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p := &transaccionResolucionPrueba{ejecutar: func(_ context.Context, q ports.SolicitudResolucionFormalizacion) (ports.ResultadoResolucionFormalizacion, error) {
				r := reciboResolucionPrueba(q)
				switch caso {
				case "replay":
					r.Estado = "replay_registrada"
				case "cancelacion_despues":
					cancel()
				case "error_con_recibo":
					return r, errors.New("fallo de commit")
				case "recibo_ajeno":
					r.Solicitud.PropuestaRef = "propuesta:otra"
				}
				return r, nil
			}}
			svc, _ := NuevoServicioResolucionFormalizacion(p)
			q := s
			if caso == "cancelado" {
				cancel()
			}
			if caso == "confirmacion" {
				q.ConfirmaRevisionPropuesta = false
			}
			r, err := svc.RegistrarResolucionFormalizacion(ctx, q)
			if caso == "registro" || caso == "replay" {
				if err != nil || r.ValidarPara(q) != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil || r != (ports.ResultadoResolucionFormalizacion{}) {
				t.Fatal("éxito falso", r, err)
			}
			if (caso == "cancelado" || caso == "confirmacion") && p.llamadas != 0 {
				t.Fatal("puerto invocado")
			}
		})
	}
}
