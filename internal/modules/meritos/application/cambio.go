package application

import (
	"reflect"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

func prepararCambio(o ports.OrdenOperacion, actual *ports.RegistroActual, ahora time.Time) (ports.Cambio, error) {
	nuevo := ports.RegistroActual{Hecho: copiarHecho(o.Hecho), DeclaranteRef: o.ActorRef}
	switch o.Accion {
	case accionDeclarar:
		if actual != nil || o.VersionEsperada != 0 || o.Hecho.PersonaRef != o.ActorRef {
			return ports.Cambio{}, vec.ErrAutorizacionDenegada
		}
	case accionRechazar, accionRectificar:
		if actual == nil || actual.Hecho.Validar() != nil || !domain.ReferenciaValida(actual.DeclaranteRef) ||
			actual.Hecho.Version != o.VersionEsperada || actual.Hecho.Referencia != o.Hecho.Referencia ||
			actual.Hecho.PersonaRef != o.Hecho.PersonaRef || actual.Hecho.Tipo != o.Hecho.Tipo ||
			actual.Hecho.Procedencia.FuenteRef != o.Hecho.Procedencia.FuenteRef || actual.Hecho.Procedencia.HechoOrigenRef != o.Hecho.Procedencia.HechoOrigenRef {
			return ports.Cambio{}, ports.ErrConflictoVersion
		}
		nuevo.DeclaranteRef = actual.DeclaranteRef
		if o.Accion == accionRectificar {
			if o.ActorRef != actual.DeclaranteRef || o.ActorRef != actual.Hecho.PersonaRef {
				return ports.Cambio{}, vec.ErrAutorizacionDenegada
			}
			nuevo.Hecho.Estado = domain.Pendiente
		} else {
			if o.ActorRef == actual.DeclaranteRef || o.ActorRef == actual.Hecho.PersonaRef {
				return ports.Cambio{}, vec.ErrAutorizacionDenegada
			}
			if actual.Hecho.Estado != domain.Declarado && actual.Hecho.Estado != domain.Pendiente {
				return ports.Cambio{}, ErrSolicitud
			}
			previo := copiarHecho(actual.Hecho)
			previo.Estado, previo.Revision, previo.Version = domain.Declarado, nil, o.Hecho.Version
			if !reflect.DeepEqual(previo, o.Hecho) {
				return ports.Cambio{}, ErrSolicitud
			}
			nuevo.Hecho.Estado = domain.Rechazado
			nuevo.Hecho.Revision = &domain.Revision{Referencia: "revision:" + o.HuellaComando, ActorRef: o.ActorRef,
				MotivoRef: o.Motivo.Referencia(), Fecha: ahora.UTC().Format(time.RFC3339Nano)}
		}
	default:
		return ports.Cambio{}, ErrAcreditacionPendiente
	}
	if nuevo.Hecho.Validar() != nil {
		return ports.Cambio{}, ErrSolicitud
	}
	var anterior *ports.RegistroActual
	if actual != nil {
		copia := *actual
		copia.Hecho = copiarHecho(copia.Hecho)
		anterior = &copia
	}
	return ports.Cambio{Orden: o, Anterior: anterior, Nuevo: nuevo}, nil
}

func reciboCoincide(r ports.Recibo, o ports.OrdenOperacion) bool {
	if !domain.ReferenciaValida(r.Referencia) || !domain.ReferenciaValida(r.AuditoriaRef) || !domain.ReferenciaValida(r.EventoRef) ||
		r.Accion != o.Accion || r.ActorRef != o.ActorRef || r.ClaveIdempotencia != o.ClaveIdempotencia || r.HuellaComando != o.HuellaComando ||
		r.VersionEsperada != o.VersionEsperada || r.Registro.Hecho.Validar() != nil || !domain.ReferenciaValida(r.Registro.DeclaranteRef) ||
		r.RegistradoEn.IsZero() || r.Registro.Hecho.Version != o.VersionEsperada+1 {
		return false
	}
	h := copiarHecho(r.Registro.Hecho)
	h.Estado, h.Revision = domain.Declarado, nil
	if !reflect.DeepEqual(h, o.Hecho) {
		return false
	}
	switch o.Accion {
	case accionDeclarar:
		return r.Registro.Hecho.Estado == domain.Declarado && r.Registro.DeclaranteRef == o.ActorRef
	case accionRectificar:
		return r.Registro.Hecho.Estado == domain.Pendiente && r.Registro.DeclaranteRef == o.ActorRef
	case accionRechazar:
		revision := r.Registro.Hecho.Revision
		return r.Registro.Hecho.Estado == domain.Rechazado && r.Registro.DeclaranteRef != o.ActorRef && revision != nil && revision.ActorRef == o.ActorRef &&
			revision.Referencia == "revision:"+o.HuellaComando && revision.MotivoRef == o.Motivo.Referencia()
	}
	return false
}
