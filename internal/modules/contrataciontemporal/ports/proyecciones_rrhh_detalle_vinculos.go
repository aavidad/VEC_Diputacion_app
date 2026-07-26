package ports

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	bloqueAnalisisRRHH uint8 = 1 << iota
	bloqueCoberturaRRHH
	bloqueAsignacionRRHH
)

type vinculoHitoOperativoRRHH struct {
	secuencia         uint64
	versionExpediente uint64
	accionClave       domain.ClaveCatalogo
	faseDestino       domain.ClaveFase
	realizadaEn       time.Time
}

func (v vinculoHitoOperativoRRHH) validar() error {
	if v.secuencia < 2 || v.versionExpediente < 2 ||
		v.secuencia != v.versionExpediente ||
		!v.accionClave.Valida() || !v.faseDestino.Valida() ||
		!domain.InstanteUTCCanonico(v.realizadaEn) {
		return ErrResultadoConsultaRRHHNoConfiable
	}
	return nil
}

func (v vinculoHitoOperativoRRHH) coincide(
	hitos []HitoExpedienteRRHH,
) bool {
	if v.validar() != nil || v.secuencia > uint64(len(hitos)) {
		return false
	}
	hito := hitos[v.secuencia-1]
	return hito.Secuencia == v.secuencia &&
		hito.VersionExpediente == v.versionExpediente &&
		hito.AccionClave == v.accionClave &&
		hito.FaseDestino == v.faseDestino &&
		hito.RealizadaEn.Equal(v.realizadaEn)
}

func vinculoDesdeAnalisisRRHH(
	expediente domain.Expediente,
) vinculoHitoOperativoRRHH {
	if expediente.Analisis == nil ||
		expediente.Analisis.ActuacionRegistro == nil {
		return vinculoHitoOperativoRRHH{}
	}
	secuencia := expediente.Analisis.ActuacionRegistro.Secuencia
	if secuencia > uint64(len(expediente.Actuaciones)) {
		return vinculoHitoOperativoRRHH{}
	}
	return vinculoDesdeActuacionRRHH(expediente.Actuaciones[secuencia-1])
}

func vinculoDesdeCoberturaRRHH(
	expediente domain.Expediente,
) vinculoHitoOperativoRRHH {
	if expediente.ViaCobertura == nil {
		return vinculoHitoOperativoRRHH{}
	}
	if decision := expediente.ViaCobertura.DecisionGobernada; decision != nil {
		v := decision.Actuacion
		return vinculoHitoOperativoRRHH{
			secuencia: v.Secuencia, versionExpediente: v.VersionExpediente,
			accionClave: v.AccionClave, faseDestino: v.FaseDestino,
			realizadaEn: v.RealizadaEn,
		}
	}
	if expediente.Analisis == nil ||
		expediente.Analisis.ActuacionRegistro == nil {
		return vinculoHitoOperativoRRHH{}
	}
	secuencia := expediente.Analisis.ActuacionRegistro.Secuencia + 1
	if secuencia > uint64(len(expediente.Actuaciones)) {
		return vinculoHitoOperativoRRHH{}
	}
	return vinculoDesdeActuacionRRHH(expediente.Actuaciones[secuencia-1])
}

func vinculoDesdeAsignacionRRHH(
	asignacion domain.AsignacionUnidad,
) vinculoHitoOperativoRRHH {
	if asignacion.ActuacionRegistro == nil {
		return vinculoHitoOperativoRRHH{}
	}
	v := asignacion.ActuacionRegistro
	return vinculoHitoOperativoRRHH{
		secuencia: v.Secuencia, versionExpediente: v.VersionExpediente,
		accionClave: v.AccionClave, faseDestino: v.FaseDestino,
		realizadaEn: asignacion.AsignadaEn,
	}
}

func vinculoDesdeActuacionRRHH(
	actuacion domain.Actuacion,
) vinculoHitoOperativoRRHH {
	return vinculoHitoOperativoRRHH{
		secuencia:         actuacion.Secuencia,
		versionExpediente: actuacion.VersionExpediente,
		accionClave:       actuacion.AccionClave,
		faseDestino:       actuacion.FaseDestino,
		realizadaEn:       actuacion.RealizadaEn,
	}
}
