package application

import (
	"context"
	"errors"
	"reflect"
	"regexp"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrCalendarioContactosNoDisponible = errors.New("bolsa: calendario de contactos no disponible")
var ErrCalendarioContactosEnConflicto = errors.New("bolsa: calendario de contactos en conflicto")

var referenciaActorCalendario = regexp.MustCompile(`^per_[A-Za-z0-9_-]{22,128}$`)
var claveCalendario = regexp.MustCompile(`^[A-Za-z0-9:_-]{8,256}$`)
var reciboCalendario = regexp.MustCompile(`^recibo:calendario-contactos:[0-9a-f]{64}$`)

type SolicitudImportarCalendarioContactos struct {
	Tipo, SedeRef                          string
	Anio                                   int
	ActorRef, ClaveIdempotencia, ReciboRef string
	Material                               puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// ServicioCalendarioContactos importa solo una proyección producida por el
// puerto confiable de Calendarios. El repositorio aplica el CAS y consume la
// autorización nominal en la transacción de la nueva versión y su recibo.
type ServicioCalendarioContactos struct {
	fuente      puertosbolsa.ConsultaCalendarioContactosPublicado
	repositorio puertosbolsa.PublicacionCalendarioContactos
}

func NuevoServicioCalendarioContactos(
	fuente puertosbolsa.ConsultaCalendarioContactosPublicado,
	repositorio puertosbolsa.PublicacionCalendarioContactos,
) (*ServicioCalendarioContactos, error) {
	if interfazNulaCalendario(fuente) || interfazNulaCalendario(repositorio) {
		return nil, ErrCalendarioContactosNoDisponible
	}
	return &ServicioCalendarioContactos{fuente: fuente, repositorio: repositorio}, nil
}

func (s *ServicioCalendarioContactos) Importar(
	ctx context.Context, solicitud SolicitudImportarCalendarioContactos,
) (puertosbolsa.ReciboCalendarioContactos, error) {
	var vacio puertosbolsa.ReciboCalendarioContactos
	if ctx == nil || s == nil || solicitud.Tipo != dominiobolsa.TipoCalendarioHabilSede ||
		solicitud.SedeRef == "" || solicitud.Anio < 2000 || solicitud.Anio > 2100 ||
		!referenciaActorCalendario.MatchString(solicitud.ActorRef) ||
		!claveCalendario.MatchString(solicitud.ClaveIdempotencia) ||
		!reciboCalendario.MatchString(solicitud.ReciboRef) ||
		solicitud.Material.ValidarEstructura() != nil {
		return vacio, ErrCalendarioContactosNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	fuente, err := s.fuente.ObtenerPublicado(ctx, solicitud.Tipo, solicitud.SedeRef, solicitud.Anio)
	if err != nil {
		return vacio, ErrCalendarioContactosNoDisponible
	}
	base, err := prepararFuenteCalendarioContactos(fuente, solicitud)
	if err != nil {
		return vacio, err
	}
	actual, existe, err := s.repositorio.VersionActual(ctx, solicitud.Tipo, solicitud.SedeRef, solicitud.Anio)
	if err != nil {
		return vacio, ErrCalendarioContactosNoDisponible
	}
	var esperada uint64
	if existe {
		if actual.Validar() != nil || actual.Tipo != solicitud.Tipo || actual.SedeRef != solicitud.SedeRef || actual.Anio != solicitud.Anio || actual.Version >= 10000 {
			return vacio, ErrCalendarioContactosNoDisponible
		}
		esperada = actual.Version
		base.Version = esperada + 1
		base.VersionAnterior = actual.HuellaSHA256
	}
	base.HuellaSHA256, err = base.HuellaCanonica()
	if err != nil {
		return vacio, ErrCalendarioContactosNoDisponible
	}
	recibo, err := s.repositorio.PublicarSiVersion(ctx, puertosbolsa.OrdenPublicarCalendarioContactos{
		Calendario: base, VersionEsperada: esperada, HuellaFuente: fuente.HuellaFuenteSHA256,
		ActorRef: solicitud.ActorRef, ClaveIdempotencia: solicitud.ClaveIdempotencia,
		ReciboRef: solicitud.ReciboRef, Material: solicitud.Material,
	})
	if err != nil {
		return vacio, err
	}
	coincideNueva := recibo.Version == base.Version && recibo.HuellaSHA256 == base.HuellaSHA256
	coincideReutilizada := existe && recibo.Reutilizado && recibo.Version == actual.Version &&
		recibo.HuellaSHA256 == actual.HuellaSHA256 && mismaFuenteCalendario(actual, fuente)
	if recibo.ReciboRef != solicitud.ReciboRef || recibo.Tipo != solicitud.Tipo || recibo.SedeRef != solicitud.SedeRef ||
		recibo.Anio != solicitud.Anio || (!coincideNueva && !coincideReutilizada) ||
		(recibo.Reutilizado && !coincideReutilizada) {
		return vacio, ErrCalendarioContactosNoDisponible
	}
	return recibo, nil
}

func prepararFuenteCalendarioContactos(
	f puertosbolsa.FuenteCalendarioContactos, s SolicitudImportarCalendarioContactos,
) (dominiobolsa.CalendarioContactos, error) {
	base := dominiobolsa.CalendarioContactos{
		Esquema: dominiobolsa.EsquemaCalendarioContactos, Tipo: f.Tipo,
		SedeRef: f.SedeRef, Anio: f.Anio, Version: 1,
		Fuentes: append([]dominiobolsa.VersionFuenteCalendario(nil), f.Fuentes...),
		Dias:    append([]dominiobolsa.DiaCalendarioContactos(nil), f.Dias...),
	}
	if f.Tipo != s.Tipo || f.SedeRef != s.SedeRef || f.Anio != s.Anio {
		return dominiobolsa.CalendarioContactos{}, ErrCalendarioContactosNoDisponible
	}
	huella, err := base.HuellaCanonica()
	if err != nil || huella != f.HuellaFuenteSHA256 {
		return dominiobolsa.CalendarioContactos{}, ErrCalendarioContactosNoDisponible
	}
	base.HuellaSHA256 = huella
	return base, nil
}

func mismaFuenteCalendario(c dominiobolsa.CalendarioContactos, f puertosbolsa.FuenteCalendarioContactos) bool {
	return reflect.DeepEqual(c.Fuentes, f.Fuentes) && reflect.DeepEqual(c.Dias, f.Dias)
}

func interfazNulaCalendario(valor any) bool {
	if valor == nil {
		return true
	}
	r := reflect.ValueOf(valor)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}
