package application

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// DespachadorAvisosOferta solo continúa intenciones ya publicadas. Reclamar
// confirma la marca incierta antes de tocar la red: una caída no autoriza a
// repetir un envío cuyo resultado no se conoce.
type DespachadorAvisosOferta struct {
	repositorio            ports.RepositorioAvisosOferta
	correos                ports.FuenteCorreoParticipacion
	emisor                 ports.EmisorAvisoOferta
	asunto, cuerpo, enlace string
}

func NuevoDespachadorAvisosOferta(r ports.RepositorioAvisosOferta, c ports.FuenteCorreoParticipacion, e ports.EmisorAvisoOferta, asunto, cuerpo, enlace string) (*DespachadorAvisosOferta, error) {
	if r == nil || c == nil || e == nil || asunto == "" || cuerpo == "" || enlace == "" ||
		strings.ContainsAny(asunto, "\r\n") || len(asunto) > 512 || len(cuerpo) > 8192 {
		return nil, ports.ErrOfertaInvalida
	}
	return &DespachadorAvisosOferta{r, c, e, asunto, cuerpo, enlace}, nil
}

func (s *ServicioOfertasPublicadas) EstablecerAvisosOferta(d *DespachadorAvisosOferta) {
	s.avisos = d
}

func (s *ServicioOfertasPublicadas) despacharAvisos(ctx context.Context, o ports.OfertaPublicada) ports.OfertaPublicada {
	if o.Aviso == nil || o.Aviso.Pendientes == 0 {
		return o
	}
	if s.avisos == nil {
		o.AvisoConfiguracionPendiente = true
		return o
	}
	// Presupuesto técnico por petición: no limita destinatarios ni cambia el
	// plazo de la oferta. El resto conserva su intención pendiente.
	ctx, cancelar := context.WithTimeout(ctx, 10*time.Second)
	defer cancelar()
	d := s.avisos
	agregado := *o.Aviso
	o.Aviso = &agregado
	reemplazos := strings.NewReplacer("{categoria}", o.Datos.Categoria, "{centro}", o.Datos.Centro,
		"{descripcion}", o.Datos.Descripcion, "{fecha_inicio}", o.Datos.FechaInicio, "{enlace}", d.enlace)
	for n := 0; n < 25 && ctx.Err() == nil; n++ {
		i, encontrado, err := d.repositorio.ReclamarAvisoOferta(ctx, o)
		if err != nil {
			slog.Warn("aviso_oferta_reclamacion_no_disponible")
			break
		}
		if !encontrado {
			break
		}
		// Esta marca ya está confirmada en la base; también debe verse si
		// falla la finalización posterior al envío.
		o.Aviso.Pendientes--
		o.Aviso.SinConfirmacion++
		estado := ports.AvisoOfertaTransitorio
		correo, err := d.correos.CorreoParticipacion(ctx, i.ParticipacionRef)
		if err == nil {
			if correo == "" {
				estado = ports.AvisoOfertaPermanente
			} else {
				estado = d.emisor.EnviarAvisoOferta(ctx, ports.MensajeAvisoOferta{Destino: correo,
					Asunto: reemplazos.Replace(d.asunto), Cuerpo: reemplazos.Replace(d.cuerpo), MessageID: i.MessageID, Fecha: o.PublicadaEn})
			}
		}
		// Conservar la marca incierta si no puede confirmarse el resultado.
		fin, cancelarFin := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		actual, err := d.repositorio.FinalizarAvisoOferta(fin, o, i, estado)
		cancelarFin()
		if err != nil {
			slog.Warn("aviso_oferta_resultado_no_confirmado")
			break
		}
		o = actual
		// Un fallo transitorio no se convierte en un bucle de envíos dentro
		// de la misma petición. La recuperación posterior retoma pendientes.
		if estado == ports.AvisoOfertaTransitorio {
			break
		}
	}
	return o
}
