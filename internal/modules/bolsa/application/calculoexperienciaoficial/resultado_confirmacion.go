package calculoexperienciaoficial

import (
	"encoding/hex"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func (p PerfilConfirmacionDuradera) valido() bool {
	return p == PerfilConfirmacionExternoOrdinario || p == PerfilConfirmacionInternoAlto
}

func (r ResultadoConfirmacionDuradera) validarEstructura() error {
	if r.datos == nil || r.datos.Recibo.Validar() != nil ||
		!referenciaIntentoValida(r.datos.ReferenciaIntento) ||
		!huellaSHA256Valida(r.datos.IndiceEfectoHMACSHA256) ||
		!huellaSHA256Valida(r.datos.HuellaResultadoSHA256) ||
		!r.datos.Desenlace.valido() ||
		r.datos.Recibo.IndiceHMACSHA256() != r.datos.IndiceEfectoHMACSHA256 ||
		r.datos.Recibo.HuellaResultadoSHA256() != r.datos.HuellaResultadoSHA256 {
		return ErrReciboNoConfiable
	}
	return nil
}

func (r ResultadoConfirmacionDuradera) datosClonados() (
	datosResultadoConfirmacionDuradera,
	error,
) {
	if err := r.validarEstructura(); err != nil {
		return datosResultadoConfirmacionDuradera{}, err
	}
	return *r.datos, nil
}

func (r ResultadoConfirmacionDuradera) validarPara(
	solicitud SolicitudConfirmacionDuradera,
) error {
	if err := r.validarEstructura(); err != nil {
		return err
	}
	datos, err := solicitud.Datos()
	if err != nil || r.datos.Recibo.ValidarPara(
		r.datos.IndiceEfectoHMACSHA256, datos.Intencion,
	) != nil || r.datos.HuellaResultadoSHA256 != datos.HuellaResultadoSHA256 ||
		r.datos.ReferenciaIntento != datos.ReferenciaIntento {
		return ErrReciboNoConfiable
	}
	return nil
}

func (r ResultadoConfirmacionDuradera) ValidarParaReconciliacion(
	solicitud SolicitudReconciliacionDuradera,
) error {
	if err := r.validarEstructura(); err != nil {
		return err
	}
	datos, err := solicitud.Datos()
	if err != nil || r.datos.Recibo.HuellaIntencionSHA256() != datos.HuellaIntencionSHA256 ||
		r.datos.ReferenciaIntento != datos.ReferenciaIntento {
		return ErrReciboNoConfiable
	}
	return nil
}

func perfilEvidenciaValido(
	perfil PerfilConfirmacionDuradera,
	decision dominiovec.DecisionAutorizacion,
) bool {
	vinculo, err := decision.VinculoAutenticacionActor.Datos()
	if err != nil || decision.PrincipalID != vinculo.PrincipalID ||
		decision.PerfilActivoRef != vinculo.PerfilActivoRef {
		return false
	}
	if perfil == PerfilConfirmacionExternoOrdinario {
		return vinculo.Superficie == dominiovec.SuperficieAutenticacionExternaPersonalV1 &&
			!vinculo.CuentaPrivilegiada && dominiovec.CumpleGarantiaAutenticacion(
			vinculo.GarantiaObservada, dominiovec.AuthAssuranceSubstantial,
		) && dominiovec.CumpleGarantiaAutenticacion(
			decision.GarantiaMinima, dominiovec.AuthAssuranceSubstantial,
		)
	}
	if perfil != PerfilConfirmacionInternoAlto ||
		vinculo.GarantiaObservada != dominiovec.AuthAssuranceHigh ||
		decision.GarantiaMinima != dominiovec.AuthAssuranceHigh {
		return false
	}
	return vinculo.Superficie == dominiovec.SuperficieAutenticacionInternaCorporativaV1 &&
		!vinculo.CuentaPrivilegiada ||
		vinculo.Superficie == dominiovec.SuperficieAutenticacionAdministracionPrivilegiadaV1 &&
			vinculo.CuentaPrivilegiada
}

func huellaSHA256Valida(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	decodificada, err := hex.DecodeString(valor)
	if err != nil || len(decodificada) != 32 {
		return false
	}
	for _, caracter := range valor {
		if caracter >= 'A' && caracter <= 'F' {
			return false
		}
	}
	return true
}

func referenciaIntentoValida(valor string) bool {
	if valor == "" || len(valor) > 512 {
		return false
	}
	for indice := 0; indice < len(valor); indice++ {
		if valor[indice] < 0x21 || valor[indice] > 0x7e || valor[indice] == '*' {
			return false
		}
	}
	return true
}
