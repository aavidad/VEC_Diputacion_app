package calculoexperienciaoficial

import (
	"context"
	"errors"
	"reflect"
	"time"

	calculo "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperiencia"
	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type perfilServicio uint8

const (
	perfilExternoOrdinario perfilServicio = iota + 1
	perfilInternoAlto
)

type Servicio struct {
	perfil        perfilServicio
	perfilPDP     aplicacionvec.PerfilProteccionUsoAutorizacion
	polLectura    aplicacionvec.PoliticaUsoDecisionAutorizacion
	polAlta       aplicacionvec.PoliticaUsoDecisionAutorizacion
	polRectifica  aplicacionvec.PoliticaUsoDecisionAutorizacion
	fuente        puertosbolsa.FuenteReglasBaremoParaCalculo
	exigidor      aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2
	confirmador   ConfirmadorDuradero
	reconciliador ReconciliadorDuradero
	reloj         puertosvec.Reloj
}

func NuevoServicioExternoOrdinario(
	fuente puertosbolsa.FuenteReglasBaremoParaCalculo,
	exigidor aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	confirmador ConfirmadorDuradero,
	reconciliador ReconciliadorDuradero,
	reloj puertosvec.Reloj,
) (*Servicio, error) {
	return nuevoServicio(
		perfilExternoOrdinario,
		aplicacionvec.PerfilProteccionUsoAutorizacionOrdinario,
		fuente, exigidor, confirmador, reconciliador, reloj,
	)
}

func NuevoServicioInternoAlto(
	fuente puertosbolsa.FuenteReglasBaremoParaCalculo,
	exigidor aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	confirmador ConfirmadorDuradero,
	reconciliador ReconciliadorDuradero,
	reloj puertosvec.Reloj,
) (*Servicio, error) {
	return nuevoServicio(
		perfilInternoAlto,
		aplicacionvec.PerfilProteccionUsoAutorizacionInternoAlto,
		fuente, exigidor, confirmador, reconciliador, reloj,
	)
}

func nuevoServicio(
	perfil perfilServicio,
	perfilPDP aplicacionvec.PerfilProteccionUsoAutorizacion,
	fuente puertosbolsa.FuenteReglasBaremoParaCalculo,
	exigidor aplicacionvec.ExigidorEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2,
	confirmador ConfirmadorDuradero,
	reconciliador ReconciliadorDuradero,
	reloj puertosvec.Reloj,
) (*Servicio, error) {
	if dependenciaNula(fuente) || dependenciaNula(exigidor) ||
		dependenciaNula(confirmador) || dependenciaNula(reconciliador) || dependenciaNula(reloj) {
		return nil, ErrServicioInvalido
	}
	lectura, alta, rectifica, err := nuevasPoliticas(perfilPDP)
	if err != nil {
		return nil, errors.Join(ErrServicioInvalido, err)
	}
	return &Servicio{
		perfil: perfil, perfilPDP: perfilPDP, polLectura: lectura,
		polAlta: alta, polRectifica: rectifica, fuente: fuente, exigidor: exigidor,
		confirmador: confirmador, reconciliador: reconciliador, reloj: reloj,
	}, nil
}

type calculoPreparado struct {
	resultado calculo.ResultadoExperienciaV1
	canonico  []byte
	huella    string
	clave     oficial.ClaveEfectoV1
	intencion oficial.IntencionResultadoV1
}

func (s *Servicio) Ejecutar(
	ctx context.Context,
	orden OrdenCalculoExperienciaOficial,
) (ResultadoEjecucion, error) {
	if ctx == nil || !s.configuracionValida() {
		return ResultadoEjecucion{}, ErrServicioInvalido
	}
	if err := ctx.Err(); err != nil {
		return ResultadoEjecucion{}, err
	}
	instante := instanteCanonico(s.reloj.Ahora())
	datos, err := orden.datosClonados()
	if err != nil || validarOrdenEn(datos, instante, s.perfil) != nil {
		return ResultadoEjecucion{}, errors.Join(ErrOrdenInvalida, err)
	}
	autLectura, instanteLectura, err := s.autorizarLectura(ctx, datos, instante)
	if err != nil {
		return ResultadoEjecucion{}, err
	}
	fuente, instanteFuente, err := s.obtenerFuente(ctx, datos, autLectura, instanteLectura)
	if err != nil {
		return ResultadoEjecucion{}, err
	}
	preparado, err := prepararCalculo(datos, fuente)
	if err != nil {
		return ResultadoEjecucion{}, err
	}
	autEscritura, instanteEscritura, err := s.autorizarEscritura(
		ctx, datos, preparado.intencion, instanteFuente,
	)
	if err != nil {
		return ResultadoEjecucion{}, err
	}
	solicitud, err := nuevaSolicitudConfirmacion(
		s.perfil, datos, fuente, preparado, autLectura, autEscritura,
		instante, instanteLectura, instanteFuente, instanteEscritura,
	)
	if err != nil {
		return ResultadoEjecucion{}, err
	}
	if err := ctx.Err(); err != nil {
		return ResultadoEjecucion{}, err
	}
	intento, err := nuevoIntentoReconciliacion(
		s.perfil, datos.CorrelacionEscritura, preparado.intencion, preparado.resultado,
	)
	if err != nil {
		return ResultadoEjecucion{}, err
	}
	confirmacion, err := s.confirmador.Confirmar(ctx, solicitud)
	if err == nil && confirmacion.validarPara(solicitud) == nil {
		return construirResultadoEjecucion(preparado.resultado, confirmacion)
	}
	if ctx.Err() == nil {
		if reconciliado, errReconciliacion := s.reconciliar(ctx, intento); errReconciliacion == nil {
			return reconciliado, nil
		}
	}
	indeterminado := nuevoErrorConfirmacionIndeterminada(intento)
	if err == nil {
		return ResultadoEjecucion{}, errors.Join(indeterminado, ErrReciboNoConfiable)
	}
	return ResultadoEjecucion{}, indeterminado
}

func (s *Servicio) configuracionValida() bool {
	return s != nil && (s.perfil == perfilExternoOrdinario || s.perfil == perfilInternoAlto) &&
		!dependenciaNula(s.fuente) && !dependenciaNula(s.exigidor) &&
		!dependenciaNula(s.confirmador) && !dependenciaNula(s.reconciliador) &&
		!dependenciaNula(s.reloj)
}

func dependenciaNula(dependencia any) bool {
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

func instanteCanonico(instante time.Time) time.Time {
	return instante.UTC().Truncate(time.Microsecond)
}
