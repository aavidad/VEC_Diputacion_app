package calculoexperienciaoficial

import (
	"strings"
	"time"

	oficial "vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperienciaoficial"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func validarOrdenEn(datos DatosOrdenConfiable, instante time.Time, perfil perfilServicio) error {
	actor, errActor := datos.ContextoActor.Clonar()
	datosVinculo, errVinculo := datos.VinculoAutenticacionActor.Datos()
	if instante.IsZero() || errActor != nil || errVinculo != nil ||
		actor.Principal.AuthMethod == dominiovec.AuthMethodDemo ||
		datosVinculo.MetodoObservado == dominiovec.AuthMethodDemo ||
		actor.Principal.AuthAssurance != datosVinculo.GarantiaObservada ||
		datos.VinculoAutenticacionActor.ValidarPara(actor) != nil ||
		!datos.VinculoAutenticacionActor.VigenteEn(instante, actor) ||
		validarDatosOrdenEstaticos(datos) != nil {
		return ErrSesionNoApta
	}
	switch perfil {
	case perfilExternoOrdinario:
		if datosVinculo.Superficie != dominiovec.SuperficieAutenticacionExternaPersonalV1 ||
			datosVinculo.CuentaPrivilegiada ||
			!dominiovec.CumpleGarantiaAutenticacion(
				actor.Principal.AuthAssurance, dominiovec.AuthAssuranceSubstantial,
			) || !dominiovec.CumpleGarantiaAutenticacion(
			datosVinculo.GarantiaObservada, dominiovec.AuthAssuranceSubstantial,
		) ||
			datos.TipoEfecto != oficial.EfectoCalculoInicial {
			return ErrSesionNoApta
		}
	case perfilInternoAlto:
		corporativa := datosVinculo.Superficie ==
			dominiovec.SuperficieAutenticacionInternaCorporativaV1 &&
			!datosVinculo.CuentaPrivilegiada
		privilegiada := datosVinculo.Superficie ==
			dominiovec.SuperficieAutenticacionAdministracionPrivilegiadaV1 &&
			datosVinculo.CuentaPrivilegiada
		if actor.Principal.AuthAssurance != dominiovec.AuthAssuranceHigh ||
			datosVinculo.GarantiaObservada != dominiovec.AuthAssuranceHigh ||
			(!corporativa && !privilegiada) {
			return ErrSesionNoApta
		}
	default:
		return ErrSesionNoApta
	}
	return nil
}

func validarDatosOrdenEstaticos(datos DatosOrdenConfiable) error {
	if datos.Motivo.Validar() != nil || datos.CorrelacionLectura.Validar() != nil ||
		datos.CorrelacionEscritura.Validar() != nil ||
		!correlacionesDistintas(datos.CorrelacionLectura, datos.CorrelacionEscritura) ||
		!selectorValido(datos.Selector) || !causaCoincideConMotivo(datos.Causa, datos.Motivo) ||
		!motorEsperadoValido(datos.MotorEsperado) || !efectoYPredecesorValidos(
		datos.TipoEfecto, datos.Predecesor,
	) {
		return ErrOrdenInvalida
	}
	return nil
}

func selectorValido(selector puertosbolsa.SelectorFuenteExactaCalculoReglasBaremo) bool {
	return selector.Validar() == nil
}

func causaCoincideConMotivo(
	causa oficial.CausaGobernadaV1,
	motivo dominiovec.ReferenciaEntradaCatalogo,
) bool {
	return causa.Catalogo.Referencia == motivo.CatalogoID &&
		causa.Catalogo.Version == uint64(motivo.CatalogoVersion) &&
		causa.Catalogo.HuellaSHA256 == motivo.CatalogoHuellaSHA256 &&
		causa.Clave == motivo.EntradaClave
}

func motorEsperadoValido(motor oficial.VinculoMotorV1) bool {
	return tokenTecnicoValido(motor.Contrato, 128) && motor.Version > 0 &&
		motor.Version <= 1_000_000_000 && huellaSHA256Valida(motor.HuellaContratoSHA256)
}

func efectoYPredecesorValidos(
	tipo oficial.TipoEfectoV1,
	predecesor *oficial.VinculoPredecesorV1,
) bool {
	if tipo == oficial.EfectoCalculoInicial {
		return predecesor == nil
	}
	return tipo == oficial.EfectoRectificacion && predecesor != nil &&
		referenciaOpacaValida(predecesor.ReferenciaRecibo) &&
		huellaSHA256Valida(predecesor.HuellaReciboSHA256)
}

func correlacionesDistintas(
	primera, segunda dominiovec.ReferenciaCorrelacionAutorizacionV2,
) bool {
	a, errA := primera.ValorCanonico()
	b, errB := segunda.ValorCanonico()
	return errA == nil && errB == nil && a != b
}

func tokenTecnicoValido(valor string, maximo int) bool {
	if valor == "" || len(valor) > maximo {
		return false
	}
	for _, caracter := range valor {
		if !(caracter >= 'a' && caracter <= 'z') && !(caracter >= '0' && caracter <= '9') &&
			caracter != '.' && caracter != ':' && caracter != '-' && caracter != '_' {
			return false
		}
	}
	return true
}

func referenciaOpacaValida(valor string) bool {
	if valor == "" || len(valor) > 512 ||
		!((valor[0] >= 'a' && valor[0] <= 'z') || (valor[0] >= 'A' && valor[0] <= 'Z') ||
			(valor[0] >= '0' && valor[0] <= '9')) {
		return false
	}
	for _, caracter := range valor {
		if !((caracter >= 'a' && caracter <= 'z') || (caracter >= 'A' && caracter <= 'Z') ||
			(caracter >= '0' && caracter <= '9') || strings.ContainsRune(":/#-_.", caracter)) {
			return false
		}
	}
	return true
}
