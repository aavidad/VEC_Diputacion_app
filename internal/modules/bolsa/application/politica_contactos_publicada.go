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

var ErrPoliticaContactosNoDisponible = errors.New("bolsa: politica de contactos no disponible")

var reciboPoliticaContactos = regexp.MustCompile(`^recibo:politica-contactos:[0-9a-f]{64}$`)

type SolicitudPublicarPoliticaContactos struct {
	BolsaRef                               string
	ActorRef, ClaveIdempotencia, ReciboRef string
	Material                               puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

// ServicioPoliticaContactos solo publica una versión que venga del gobierno
// de reglas de Bolsa. La transacción durable verifica autorización y CAS.
type ServicioPoliticaContactos struct {
	fuente      puertosbolsa.ConsultaPoliticaContactosGobernada
	repositorio puertosbolsa.PublicacionPoliticaContactos
}

func NuevoServicioPoliticaContactos(
	fuente puertosbolsa.ConsultaPoliticaContactosGobernada,
	repositorio puertosbolsa.PublicacionPoliticaContactos,
) (*ServicioPoliticaContactos, error) {
	if interfazNulaCalendario(fuente) || interfazNulaCalendario(repositorio) {
		return nil, ErrPoliticaContactosNoDisponible
	}
	return &ServicioPoliticaContactos{fuente: fuente, repositorio: repositorio}, nil
}

func (s *ServicioPoliticaContactos) Publicar(
	ctx context.Context, solicitud SolicitudPublicarPoliticaContactos,
) (puertosbolsa.ReciboPoliticaContactos, error) {
	var vacio puertosbolsa.ReciboPoliticaContactos
	if ctx == nil || s == nil || solicitud.BolsaRef == "" ||
		!referenciaActorCalendario.MatchString(solicitud.ActorRef) ||
		!claveCalendario.MatchString(solicitud.ClaveIdempotencia) ||
		!reciboPoliticaContactos.MatchString(solicitud.ReciboRef) ||
		solicitud.Material.ValidarEstructura() != nil {
		return vacio, ErrPoliticaContactosNoDisponible
	}
	capacidad := solicitud.Material.ResumenCapacidad()
	if capacidad.Operacion() != puertosbolsa.AccionPublicarPoliticaContactos ||
		capacidad.AudienciaConsumo() != puertosbolsa.AudienciaPublicarPoliticaContactos ||
		capacidad.EfectoRef() != solicitud.BolsaRef {
		return vacio, ErrPoliticaContactosNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	fuente, err := s.fuente.ObtenerPublicada(ctx, solicitud.BolsaRef)
	if err != nil {
		return vacio, ErrPoliticaContactosNoDisponible
	}
	base, err := prepararFuentePoliticaContactos(fuente, solicitud.BolsaRef)
	if err != nil {
		return vacio, err
	}
	actual, existe, err := s.repositorio.VersionActual(ctx, solicitud.BolsaRef)
	if err != nil {
		return vacio, ErrPoliticaContactosNoDisponible
	}
	var esperada uint64
	if existe {
		if actual.Validar() != nil || actual.BolsaRef != solicitud.BolsaRef || actual.Version >= 10000 {
			return vacio, ErrPoliticaContactosNoDisponible
		}
		esperada = actual.Version
		base.Version = esperada + 1
		base.VersionAnterior = actual.HuellaSHA256
	}
	base.HuellaSHA256, err = base.HuellaCanonica()
	if err != nil {
		return vacio, ErrPoliticaContactosNoDisponible
	}
	recibo, err := s.repositorio.PublicarSiVersion(ctx, puertosbolsa.OrdenPublicarPoliticaContactos{
		Politica: base, VersionEsperada: esperada, HuellaFuente: fuente.HuellaFuenteSHA256,
		ActorRef: solicitud.ActorRef, ClaveIdempotencia: solicitud.ClaveIdempotencia,
		ReciboRef: solicitud.ReciboRef, Material: solicitud.Material,
	})
	if err != nil {
		return vacio, err
	}
	coincideNueva := recibo.Version == base.Version && recibo.HuellaSHA256 == base.HuellaSHA256
	coincideReutilizada := existe && recibo.Reutilizado && recibo.Version == actual.Version &&
		recibo.HuellaSHA256 == actual.HuellaSHA256 && mismaFuentePolitica(actual, fuente)
	if recibo.ReciboRef != solicitud.ReciboRef || recibo.BolsaRef != solicitud.BolsaRef ||
		(!coincideNueva && !coincideReutilizada) || (recibo.Reutilizado && !coincideReutilizada) {
		return vacio, ErrPoliticaContactosNoDisponible
	}
	return recibo, nil
}

func prepararFuentePoliticaContactos(f puertosbolsa.FuentePoliticaContactos, bolsaRef string) (dominiobolsa.PoliticaContactosPublicada, error) {
	p := dominiobolsa.PoliticaContactosPublicada{
		Esquema: dominiobolsa.EsquemaPoliticaContactos, BolsaRef: f.BolsaRef, Version: 1,
		CatalogoRef: f.CatalogoRef, CatalogoHuellaSHA256: f.CatalogoHuellaSHA256, Ejemplo: f.Ejemplo,
		TipoDia: f.TipoDia, SedeRef: f.SedeRef, Zona: f.Zona,
		DesdeMinuto: f.DesdeMinuto, HastaMinuto: f.HastaMinuto,
		ControlFranja: f.ControlFranja, IntentosPorCiclo: f.IntentosPorCiclo,
		Ciclos: f.Ciclos, SeparacionSegundos: f.SeparacionSegundos,
		ControlSeparacion:     f.ControlSeparacion,
		ResultadosSinContacto: append([]string(nil), f.ResultadosSinContacto...),
	}
	if f.BolsaRef != bolsaRef {
		return dominiobolsa.PoliticaContactosPublicada{}, ErrPoliticaContactosNoDisponible
	}
	huella, err := p.HuellaCanonica()
	if err != nil || huella != f.HuellaFuenteSHA256 {
		return dominiobolsa.PoliticaContactosPublicada{}, ErrPoliticaContactosNoDisponible
	}
	p.HuellaSHA256 = huella
	return p, nil
}

func mismaFuentePolitica(p dominiobolsa.PoliticaContactosPublicada, f puertosbolsa.FuentePoliticaContactos) bool {
	return p.BolsaRef == f.BolsaRef && p.CatalogoRef == f.CatalogoRef &&
		p.CatalogoHuellaSHA256 == f.CatalogoHuellaSHA256 && p.Ejemplo == f.Ejemplo && p.TipoDia == f.TipoDia &&
		p.SedeRef == f.SedeRef && p.Zona == f.Zona && p.DesdeMinuto == f.DesdeMinuto &&
		p.HastaMinuto == f.HastaMinuto && p.ControlFranja == f.ControlFranja &&
		p.IntentosPorCiclo == f.IntentosPorCiclo && p.Ciclos == f.Ciclos &&
		p.SeparacionSegundos == f.SeparacionSegundos && p.ControlSeparacion == f.ControlSeparacion &&
		reflect.DeepEqual(p.ResultadosSinContacto, f.ResultadosSinContacto)
}
