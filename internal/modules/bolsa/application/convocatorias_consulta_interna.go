package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrServicioConsultaConvocatoriaInvalido  = errors.New("bolsa: servicio de consulta interna de convocatoria invalido")
	ErrOrdenConsultaConvocatoriaInvalida     = errors.New("bolsa: orden de consulta interna de convocatoria invalida")
	ErrResultadoConsultaConvocatoriaInseguro = errors.New("bolsa: resultado de consulta interna de convocatoria no confiable")
)

const motivoConsultaInternaConvocatoria = "consulta interna de version exacta de convocatoria"

// OrdenConsultaVersionConvocatoria contiene capacidades resueltas por la
// frontera interna. Ninguno de estos campos debe reconstruirse desde JSON.
type OrdenConsultaVersionConvocatoria struct {
	ContextoActor             dominiovec.ContextoActor
	VinculoAutenticacionActor dominiovec.VinculoAutenticacionActorV1
	Selector                  puertosbolsa.SelectorVersionConvocatoriaExacta
	IncluirInstanciaFlujo     bool
	CorrelacionRef            string
}

// ServicioConsultaVersionConvocatoria compone identidad, PEP, PDP y lectura
// durable. El adaptador PostgreSQL vuelve a revalidar y consumir la decision
// en la misma transaccion que deja la auditoria.
type ServicioConsultaVersionConvocatoria struct {
	consulta       puertosbolsa.ConsultaGobiernoConvocatorias
	exigidor       aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacion
	reloj          puertosvec.Reloj
	politicaSimple aplicacionvec.PoliticaUsoDecisionAutorizacion
	politicaFlujo  aplicacionvec.PoliticaUsoDecisionAutorizacion
}

func NuevoServicioConsultaVersionConvocatoria(
	consulta puertosbolsa.ConsultaGobiernoConvocatorias,
	exigidor aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacion,
	reloj puertosvec.Reloj,
) (*ServicioConsultaVersionConvocatoria, error) {
	if dependenciaConsultaConvocatoriaNula(consulta) ||
		dependenciaConsultaConvocatoriaNula(exigidor) ||
		dependenciaConsultaConvocatoriaNula(reloj) {
		return nil, ErrServicioConsultaConvocatoriaInvalido
	}
	simple, err := nuevaPoliticaConsultaConvocatoria(false)
	if err != nil {
		return nil, errors.Join(ErrServicioConsultaConvocatoriaInvalido, err)
	}
	flujo, err := nuevaPoliticaConsultaConvocatoria(true)
	if err != nil {
		return nil, errors.Join(ErrServicioConsultaConvocatoriaInvalido, err)
	}
	return &ServicioConsultaVersionConvocatoria{
		consulta: consulta, exigidor: exigidor, reloj: reloj,
		politicaSimple: simple, politicaFlujo: flujo,
	}, nil
}

func (s *ServicioConsultaVersionConvocatoria) ObtenerExacta(
	ctx context.Context,
	orden OrdenConsultaVersionConvocatoria,
) (puertosbolsa.ResultadoConsultaVersionConvocatoria, error) {
	if ctx == nil || s == nil || dependenciaConsultaConvocatoriaNula(s.consulta) ||
		dependenciaConsultaConvocatoriaNula(s.exigidor) ||
		dependenciaConsultaConvocatoriaNula(s.reloj) {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{},
			ErrServicioConsultaConvocatoriaInvalido
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, err
	}
	consultadaEn := s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	actor, err := validarOrdenConsultaConvocatoria(orden, consultadaEn)
	if err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, err
	}
	recurso, err := puertosbolsa.RecursoAutorizableConsultaVersionConvocatoria(orden.Selector)
	if err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada, ErrOrdenConsultaConvocatoriaInvalida, err,
		)
	}
	politica := s.politicaSimple
	if orden.IncluirInstanciaFlujo {
		politica = s.politicaFlujo
	}
	evidencia, err := s.exigidor.ExigirEvidencia(
		ctx, actor, orden.VinculoAutenticacionActor, recurso,
		orden.CorrelacionRef, motivoConsultaInternaConvocatoria, politica,
	)
	if err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{},
			errors.Join(dominiovec.ErrAutorizacionDenegada, err)
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, err
	}
	consultadaEn = s.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if consultadaEn.IsZero() || evidencia.ValidarEn(consultadaEn) != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida,
		)
	}
	solicitud := puertosbolsa.SolicitudConsultaVersionConvocatoriaAutorizada{
		Selector: orden.Selector, IncluirInstanciaFlujo: orden.IncluirInstanciaFlujo,
		Autorizacion: evidencia, ConsultadaEn: consultadaEn,
	}
	if solicitud.Validar() != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada, ErrOrdenConsultaConvocatoriaInvalida,
		)
	}
	resultado, err := s.consulta.ObtenerVersionExacta(ctx, solicitud)
	if err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, err
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, err
	}
	resultado, err = resultado.Clonar()
	if err != nil || resultado.ValidarPara(solicitud) != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, errors.Join(
			ErrResultadoConsultaConvocatoriaInseguro, err,
		)
	}
	return resultado, nil
}

func nuevaPoliticaConsultaConvocatoria(
	incluirFlujo bool,
) (aplicacionvec.PoliticaUsoDecisionAutorizacion, error) {
	accion := puertosbolsa.AccionConsultarVersionConvocatoria
	campos := []string{"version_convocatoria"}
	if incluirFlujo {
		accion = puertosbolsa.AccionConsultarVersionConFlujoConvocatoria
		campos = []string{"instancia_flujo", "version_convocatoria"}
	}
	return aplicacionvec.NuevaPoliticaUsoDecisionAutorizacion(
		accion, puertosbolsa.ModuloGobiernoConvocatorias,
		puertosbolsa.TipoRecursoVersionConvocatoriaGobernada,
		puertosbolsa.FinalidadConsultaInternaConvocatorias, campos,
		aplicacionvec.PerfilProteccionUsoAutorizacionInternoAlto,
	)
}

func validarOrdenConsultaConvocatoria(
	orden OrdenConsultaVersionConvocatoria,
	instante time.Time,
) (dominiovec.ContextoActor, error) {
	actor, errActor := orden.ContextoActor.Clonar()
	datosVinculo, errVinculo := orden.VinculoAutenticacionActor.Datos()
	superficieOrdinaria := errVinculo == nil &&
		datosVinculo.Superficie == dominiovec.SuperficieAutenticacionInternaCorporativaV1 &&
		!datosVinculo.CuentaPrivilegiada
	superficiePrivilegiada := errVinculo == nil &&
		datosVinculo.Superficie == dominiovec.SuperficieAutenticacionAdministracionPrivilegiadaV1 &&
		datosVinculo.CuentaPrivilegiada
	if instante.IsZero() || errActor != nil || errVinculo != nil ||
		actor.Principal.AuthMethod == dominiovec.AuthMethodDemo ||
		actor.Principal.AuthAssurance != dominiovec.AuthAssuranceHigh ||
		datosVinculo.MetodoObservado == dominiovec.AuthMethodDemo ||
		datosVinculo.GarantiaObservada != dominiovec.AuthAssuranceHigh ||
		(!superficieOrdinaria && !superficiePrivilegiada) ||
		orden.VinculoAutenticacionActor.ValidarPara(actor) != nil ||
		!orden.VinculoAutenticacionActor.VigenteEn(instante, actor) ||
		orden.Selector.Validar() != nil || !correlacionConsultaConvocatoriaValida(orden.CorrelacionRef) {
		return dominiovec.ContextoActor{}, errors.Join(
			dominiovec.ErrAutorizacionDenegada,
			ErrOrdenConsultaConvocatoriaInvalida,
			errActor,
			errVinculo,
		)
	}
	return actor, nil
}

func correlacionConsultaConvocatoriaValida(valor string) bool {
	if valor == "" || len(valor) > 512 || valor != strings.TrimSpace(valor) ||
		strings.ContainsRune(valor, '*') {
		return false
	}
	for indice := 0; indice < len(valor); indice++ {
		if valor[indice] < 0x21 || valor[indice] > 0x7e {
			return false
		}
	}
	return true
}

func dependenciaConsultaConvocatoriaNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}
