package application

import (
	"context"
	"errors"
	"reflect"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var ErrServicioParticipacionesPropiasInvalido = errors.New("bolsa: servicio de participaciones propias invalido")

type OrdenConsultaParticipacionesPropias struct{ ContextoActor dominiovec.ContextoActor }

type ServicioParticipacionesPropias struct {
	autorizador  puertosbolsa.AutorizadorParticipacionesPropias
	persistencia puertosbolsa.ConsultaParticipacionesPropiasPersistente
	reloj        puertosvec.Reloj
}

func NuevoServicioParticipacionesPropias(a puertosbolsa.AutorizadorParticipacionesPropias, p puertosbolsa.ConsultaParticipacionesPropiasPersistente, r puertosvec.Reloj) (*ServicioParticipacionesPropias, error) {
	if dependenciaParticipacionesPropiasNula(a) || dependenciaParticipacionesPropiasNula(p) || dependenciaParticipacionesPropiasNula(r) {
		return nil, ErrServicioParticipacionesPropiasInvalido
	}
	return &ServicioParticipacionesPropias{autorizador: a, persistencia: p, reloj: r}, nil
}

func (s *ServicioParticipacionesPropias) Consultar(ctx context.Context, orden OrdenConsultaParticipacionesPropias) (puertosbolsa.ResultadoParticipacionesPropias, error) {
	vacio := puertosbolsa.ResultadoParticipacionesPropias{}
	if ctx == nil || s == nil || dependenciaParticipacionesPropiasNula(s.autorizador) || dependenciaParticipacionesPropiasNula(s.persistencia) || dependenciaParticipacionesPropiasNula(s.reloj) {
		return vacio, ErrServicioParticipacionesPropiasInvalido
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	ahora := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	candidato, err := candidatoVigenteUnico(orden.ContextoActor, ahora)
	if err != nil {
		return vacio, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	consulta, err := puertosbolsa.NuevaConsultaParticipacionesPropias(candidato, ahora)
	if err != nil {
		return vacio, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	recurso, err := puertosbolsa.RecursoAutorizableParticipacionesPropias(consulta)
	if err != nil {
		return vacio, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	material, err := s.autorizador.AutorizarOperacion(ctx, puertosbolsa.AccionConsultarParticipacionesPropias, recurso)
	if err != nil {
		return vacio, errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	if material.ValidarEstructura() != nil {
		return vacio, errors.Join(dominiovec.ErrAutorizacionDenegada, puertosvec.ErrExportacionMaterialConsumoAutorizacionAtestadaV3Invalida)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	resumen := material.ResumenCapacidad()
	if err != nil || resumen.Operacion() != puertosbolsa.AccionConsultarParticipacionesPropias || resumen.EfectoRef() != recurso.Referencia || resumen.EfectoHuellaSHA256() != huella || resumen.AudienciaConsumo() != puertosbolsa.AudienciaParticipacionesPropias {
		return vacio, dominiovec.ErrAutorizacionDenegada
	}
	resultado, err := s.persistencia.ConsultarParticipacionesPropias(ctx, consulta, material)
	if err != nil {
		return vacio, err
	}
	return resultado.ClonarValidadoPara(consulta)
}

func candidatoVigenteUnico(actor dominiovec.ContextoActor, ahora time.Time) (string, error) {
	if actor.Validar() != nil || ahora.IsZero() {
		return "", puertosbolsa.ErrConsultaParticipacionesPropiasInvalida
	}
	candidatos := make([]string, 0, 1)
	for _, v := range actor.Instantanea.Vinculos {
		if v.Tipo == dominiovec.TipoReferenciaContextoActorCandidato && v.VigenteEn(ahora) {
			candidatos = append(candidatos, v.Referencia)
		}
	}
	if len(candidatos) != 1 {
		return "", puertosbolsa.ErrConsultaParticipacionesPropiasInvalida
	}
	return candidatos[0], nil
}
func dependenciaParticipacionesPropiasNula(d any) bool {
	if d == nil {
		return true
	}
	v := reflect.ValueOf(d)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	}
	return false
}
