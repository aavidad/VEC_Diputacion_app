package ports

import (
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/vec/reglas"
)

// MaximoGruposPlazoCuadroRRHH acota los grupos de plazo que se aceptan de
// una consulta antes de calcular plazos. Como la entrada en fase es un
// instante, en la práctica hay un grupo por expediente en trámite; el tope
// deja margen sobrado a la Diputación (cientos al año) y falla cerrado si se
// supera.
const MaximoGruposPlazoCuadroRRHH = 100_000

// maximoRecuentosCuadroRRHH: una entrada por (estado, fase).
const maximoRecuentosCuadroRRHH = 1_024

// RecuentoCuadroRRHH es el número de expedientes del corte filtrado con un
// estado y una fase (CT-000184).
type RecuentoCuadroRRHH struct {
	EstadoClave domain.EstadoOperativo
	FaseClave   domain.ClaveFase
	Numero      uint64
}

// GrupoPlazoCuadroRRHH agrupa los expedientes en trámite que entraron en la
// misma fase en el mismo instante y con la misma urgencia: comparten plazo.
// No lleva referencias de expediente.
type GrupoPlazoCuadroRRHH struct {
	FaseClave domain.ClaveFase
	Desde     time.Time
	Urgente   bool
	Numero    uint64
	// La captura puede ser nil en un tramo anterior a CT190.
	Instantanea *reglas.InstantaneaPersistidaRegla
}

// AgregadosCuadroRRHH son los agregados de todo el corte filtrado que la
// consulta atestada devuelve cuando se pide el resumen.
type AgregadosCuadroRRHH struct {
	Recuentos   []RecuentoCuadroRRHH
	GruposPlazo []GrupoPlazoCuadroRRHH
}

// ResumenCuadroRRHH es lo que la portada muestra: recuentos de todo el corte
// filtrado. «En trámite» es todo lo que no está completado ni cancelado,
// como en la portada. Los plazos los resuelve la aplicación por grupo.
type ResumenCuadroRRHH struct {
	EnTramite     uint64
	Vencidos      uint64
	VencenHoy     uint64
	VencenSemana  uint64
	SinCalcular   uint64
	PorFase       map[domain.ClaveFase]uint64
	ConIncidencia uint64
}

// EstadoTerminado indica si el estado ya no está en trámite.
func EstadoTerminado(estado domain.EstadoOperativo) bool {
	return estado == domain.EstadoCompletado || estado == domain.EstadoCancelado
}

// validarPara comprueba la forma de los agregados y que cuadran con los
// totales y los filtros de la consulta.
func (a AgregadosCuadroRRHH) validarPara(
	solicitud SolicitudCuadroRRHH,
	totales *TotalesCuadroRRHH,
	generadaEn time.Time,
) bool {
	if totales == nil || len(a.Recuentos) > maximoRecuentosCuadroRRHH ||
		len(a.GruposPlazo) > MaximoGruposPlazoCuadroRRHH {
		return false
	}
	var total, enTramite uint64
	vistos := make(map[RecuentoCuadroRRHH]struct{}, len(a.Recuentos))
	for _, recuento := range a.Recuentos {
		if !recuento.EstadoClave.Valido() || !recuento.FaseClave.Valida() || recuento.Numero == 0 ||
			(solicitud.estadoClave != "" && recuento.EstadoClave != solicitud.estadoClave) ||
			(solicitud.faseClave != "" && recuento.FaseClave != solicitud.faseClave) {
			return false
		}
		clave := RecuentoCuadroRRHH{EstadoClave: recuento.EstadoClave, FaseClave: recuento.FaseClave}
		if _, repetido := vistos[clave]; repetido {
			return false
		}
		vistos[clave] = struct{}{}
		total += recuento.Numero
		if !EstadoTerminado(recuento.EstadoClave) {
			enTramite += recuento.Numero
		}
	}
	var enGrupos uint64
	for _, grupo := range a.GruposPlazo {
		if !grupo.FaseClave.Valida() || grupo.Numero == 0 || grupo.Desde.IsZero() ||
			grupo.Desde.After(generadaEn) ||
			(solicitud.faseClave != "" && grupo.FaseClave != solicitud.faseClave) {
			return false
		}
		if grupo.Instantanea != nil && (grupo.Instantanea.Fase != string(grupo.FaseClave) ||
			!grupo.Instantanea.FaseDesde.Equal(grupo.Desde) ||
			grupo.Instantanea.PreparadaEn.After(generadaEn)) {
			return false
		}
		enGrupos += grupo.Numero
	}
	return total == totales.Total && enGrupos == enTramite
}
