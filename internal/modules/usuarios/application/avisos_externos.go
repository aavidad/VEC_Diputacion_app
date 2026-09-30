package application

import (
	"context"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
)

type ServicioAvisosExternos struct {
	registro      ports.RegistroAvisosExternos
	protector     ports.ProtectorDireccionCorreo
	transportador ports.TransportadorAvisoExterno
	catalogo      ports.CatalogoPlantillasAvisoExterno
	productorRef  string
}

func dependenciaAvisoNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return r.IsNil()
	}
	return false
}
func NuevoServicioAvisosExternos(r ports.RegistroAvisosExternos, p ports.ProtectorDireccionCorreo, t ports.TransportadorAvisoExterno, c ports.CatalogoPlantillasAvisoExterno, productorRef string) (*ServicioAvisosExternos, error) {
	if dependenciaAvisoNula(r) || dependenciaAvisoNula(p) || dependenciaAvisoNula(t) || dependenciaAvisoNula(c) || productorRef == "" {
		return nil, ports.ErrAvisoExternoNoDisponible
	}
	return &ServicioAvisosExternos{r, p, t, c, productorRef}, nil
}

// Aceptar confirma únicamente el inbox durable. No descifra ni envía correo.
func (s *ServicioAvisosExternos) Aceptar(ctx context.Context, e ports.EventoAvisoExterno) (ports.ReciboAvisoExterno, error) {
	var cero ports.ReciboAvisoExterno
	if s == nil || ctx == nil {
		return cero, ports.ErrAvisoExternoNoDisponible
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	b, err := canonico.MaterialAvisoExterno(e)
	if err != nil || e.ProductorRef != s.productorRef {
		return cero, ports.ErrAvisoExternoInvalido
	}
	if !s.catalogo.AdmitePlantillaAvisoExterno(ctx, e.PlantillaRef, e.PlantillaVersion, e.TipoVersionado) {
		return cero, ports.ErrAvisoExternoInvalido
	}
	return s.registro.AceptarAvisoExterno(ctx, b)
}

// Despachar obtiene una reserva durable única antes de abrir el sobre. Un
// proceso caído con reserva abierta requiere reconciliación, nunca otro SMTP.
func (s *ServicioAvisosExternos) Despachar(ctx context.Context, reciboRef string) (ports.ResultadoDespachoAvisoExterno, error) {
	var cero ports.ResultadoDespachoAvisoExterno
	if s == nil || ctx == nil {
		return cero, ports.ErrAvisoExternoNoDisponible
	}
	if ctx.Err() != nil {
		return cero, ctx.Err()
	}
	r, err := s.registro.ReservarAvisoExterno(ctx, reciboRef)
	if err != nil {
		return cero, err
	}
	resultado := ports.ResultadoDespachoAvisoExterno{ReciboRef: r.Recibo.ReciboRef, Estado: r.Estado, Replay: r.Replay}
	if r.Replay {
		return resultado, nil
	}
	estado := "sin_destino"
	if r.PersonaRef != "" && r.Sobre.CorreoRef != "" {
		estado = "no_aceptado"
		b, e := canonico.MaterialAvisoExterno(r.Evento)
		h, _ := canonico.HuellaAvisoExterno(r.Evento)
		if e == nil && len(b) > 0 && h == r.Recibo.Huella && r.Evento.ProductorRef == s.productorRef && s.catalogo.AdmitePlantillaAvisoExterno(ctx, r.Evento.PlantillaRef, r.Evento.PlantillaVersion, r.Evento.TipoVersionado) {
			err = s.protector.ConDireccionCorreoDescifrada(ctx, r.PersonaRef, r.Sobre, func(claro []byte) error {
				if !domain.DireccionCorreoValida(string(claro)) {
					return ports.ErrAvisoExternoNoDisponible
				}
				if ctx.Err() != nil {
					return ctx.Err()
				}
				switch s.transportador.EnviarAvisoExterno(ctx, ports.MensajeAvisoExterno{EnvioRef: r.Recibo.ReciboRef, Destino: string(claro), Evento: r.Evento}) {
				case ports.AvisoExternoAceptadoPorRelay:
					estado = "aceptado"
				case ports.AvisoExternoNoAceptado:
					estado = "no_aceptado"
				default:
					estado = "reservado_incierto"
				}
				return nil
			})
		} else {
			err = ports.ErrAvisoExternoNoDisponible
		}
	}
	// Un fallo posterior a SMTP no libera la reserva. Una recuperación sólo
	// consulta el estado durable y nunca repite la llamada al transportador.
	confirmarCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelar()
	if confirmar := s.registro.ConfirmarAvisoExterno(confirmarCtx, r.Recibo.ReciboRef, r.ReservaRef, estado); confirmar != nil {
		return cero, confirmar
	}
	resultado.Estado = estado
	if err != nil {
		return resultado, ports.ErrAvisoExternoNoDisponible
	}
	return resultado, nil
}
