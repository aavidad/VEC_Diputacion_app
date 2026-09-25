package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var _ ports.SesionConsultaCambiosRRHH = (*SesionConsultaRRHHPostgreSQL)(nil)

// Petición RRHH p.4 (000117): mismos argumentos que el detalle; la función
// consume AD-3 y compara versiones consecutivas del expediente.
const consultaCambiosExpedienteRRHHPostgreSQL = `
SELECT expediente_ref,
       version_expediente::bigint,
       cambios,
       consumo_vec_huella_sha256,
       auditoria_vec_ref,
       auditoria_vec_huella_sha256,
       consumida_en
  FROM vec_contratacion_temporal.consultar_cambios_expediente_rrhh_atestado_v1(
       ROW($1::text, $2::text, $3::text)::
           vec_contratacion_temporal.alcance_consulta_rrhh_v1,
       ROW($4::text, $5::numeric)::
           vec_contratacion_temporal.consulta_detalle_rrhh_v1,
       $6::bytea, $7::bytea, $8::bytea, $9::bytea,
       $10::numeric, $11::numeric,
       $12::bytea, $13::bytea, $14::bytea, $15::bytea
  )`

// maximoCambiosJSONRRHH acota el documento antes de decodificarlo: 500
// cambios de unos 700 bytes como mucho.
const maximoCambiosJSONRRHH = 1 << 20

type cambioExpedienteRRHHSQL struct {
	VersionExpediente uint64  `json:"version_expediente"`
	RegistradaEn      string  `json:"registrada_en"`
	OrigenVersion     string  `json:"origen_version"`
	OperacionRef      string  `json:"operacion_ref"`
	Ruta              string  `json:"ruta"`
	ValorAnterior     *string `json:"valor_anterior"`
	ValorNuevo        *string `json:"valor_nuevo"`
}

func decodificarCambiosExpedienteRRHH(documento []byte) ([]ports.CambioExpedienteRRHH, error) {
	if len(documento) == 0 || len(documento) > maximoCambiosJSONRRHH {
		return nil, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	decodificador := json.NewDecoder(bytes.NewReader(documento))
	decodificador.DisallowUnknownFields()
	var filas []cambioExpedienteRRHHSQL
	if err := decodificador.Decode(&filas); err != nil || filas == nil {
		return nil, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	if _, err := decodificador.Token(); err != io.EOF {
		return nil, ports.ErrResultadoConsultaRRHHNoConfiable
	}
	cambios := make([]ports.CambioExpedienteRRHH, 0, len(filas))
	for _, f := range filas {
		instante, err := time.Parse("2006-01-02T15:04:05.000000Z", f.RegistradaEn)
		if err != nil {
			return nil, ports.ErrResultadoConsultaRRHHNoConfiable
		}
		cambios = append(cambios, ports.CambioExpedienteRRHH{
			VersionExpediente: f.VersionExpediente, RegistradaEn: instante.UTC(),
			OrigenVersion: f.OrigenVersion, OperacionRef: f.OperacionRef, Ruta: f.Ruta,
			ValorAnterior: f.ValorAnterior, ValorNuevo: f.ValorNuevo,
		})
	}
	return cambios, nil
}

func (s *SesionConsultaRRHHPostgreSQL) ConsultarCambiosYRegistrar(
	ctx context.Context, orden ports.OrdenConsultaDetalleRRHH,
) (ports.ResultadoConsultaCambiosRRHH, error) {
	var cero ports.ResultadoConsultaCambiosRRHH
	if err := s.validarContexto(ctx); err != nil {
		return cero, err
	}
	if s.modo != modoConsultaDetalleRRHHOrdinaria || orden.Capacidad().AutorizaCamposSeguimiento() {
		return cero, ports.ErrConsultaRRHHNoDisponible
	}
	material, err := orden.ExportacionParaSQL()
	if err != nil || material.ValidarEstructura() != nil {
		return cero, ports.ErrConsultaRRHHNoDisponible
	}
	argumentos, err := nuevosArgumentosMaterialConsultaRRHH(material)
	if err != nil {
		return cero, ports.ErrConsultaRRHHNoDisponible
	}
	defer argumentos.limpiar()
	contexto, capacidad, solicitud := orden.Contexto(), orden.Capacidad(), orden.Solicitud()
	args := argumentosSQLDetalleConsultaRRHH(contexto.OrganizacionRef(),
		string(capacidad.ClaseAmbito()), capacidad.AmbitoRef(), solicitud, argumentos)
	var version int64
	var documento []byte
	var salida ports.ResultadoConsultaCambiosRRHH
	return ejecutarConsultaRRHHEnTransaccion(ctx, s.pool,
		consultaCambiosExpedienteRRHHPostgreSQL, args,
		[]any{&salida.ExpedienteRef, &version, &documento,
			&salida.ConsumoHuellaSHA256, &salida.AuditoriaRef,
			&salida.AuditoriaHuellaSHA256, &salida.ConsumidaEn},
		func() (ports.ResultadoConsultaCambiosRRHH, error) {
			if version < 1 || version > 9_007_199_254_740_991 {
				return cero, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			cambios, err := decodificarCambiosExpedienteRRHH(documento)
			if err != nil {
				return cero, err
			}
			salida.VersionExpediente = uint64(version)
			salida.ConsumidaEn = salida.ConsumidaEn.UTC()
			salida.Cambios = cambios
			if salida.ValidarPara(orden) != nil {
				return cero, ports.ErrResultadoConsultaRRHHNoConfiable
			}
			return salida, nil
		})
}
