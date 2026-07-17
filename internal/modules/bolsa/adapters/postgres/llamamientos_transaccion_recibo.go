package postgres

import (
	"bytes"
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

type reciboLlamamientoPostgreSQLV1 struct {
	Resultado              string
	PropuestaRef           string
	HuellaPropuestaSHA256  string
	PropuestaCanonica      []byte
	HuellaDocumentoSHA256  string
	DecisionRef            string
	HuellaDecisionSHA256   string
	AtestacionRef          string
	AtestacionCanonica     []byte
	HuellaAtestacionSHA256 string
	ConsumoRef             string
	ConsumoCanonico        []byte
	HuellaConsumoSHA256    string
	AuditoriaRef           string
	RegistroAuditoria      []byte
	HuellaAuditoriaSHA256  string
	EventoRef              string
	EventoCanonico         []byte
	HuellaEventoSHA256     string
	ConfirmadaEn           pgtype.Timestamptz
}

func ejecutarGuardadoLlamamientoPostgreSQL(
	ctx context.Context,
	tx pgx.Tx,
	operacion, prueba, decisionCanonica, propuestaCanonica []byte,
) (reciboLlamamientoPostgreSQLV1, error) {
	var recibo reciboLlamamientoPostgreSQLV1
	err := tx.QueryRow(ctx, `
		SELECT resultado, propuesta_ref, huella_propuesta_sha256,
		       propuesta_canonica, huella_documento_sha256,
		       decision_ref, huella_decision_sha256,
		       atestacion_ref, atestacion_canonica,
		       huella_atestacion_sha256,
		       consumo_ref, consumo_canonico, huella_consumo_sha256,
		       auditoria_ref, registro_auditoria,
		       huella_auditoria_sha256,
		       evento_ref, evento_canonico, huella_evento_sha256,
		       confirmada_en
		  FROM `+funcionGuardarPropuestaLlamamientoV1+`(
		       $1::jsonb, $2::jsonb, $3::bytea, $4::bytea
		  )`, operacion, prueba, decisionCanonica, propuestaCanonica,
	).Scan(
		&recibo.Resultado, &recibo.PropuestaRef, &recibo.HuellaPropuestaSHA256,
		&recibo.PropuestaCanonica, &recibo.HuellaDocumentoSHA256,
		&recibo.DecisionRef, &recibo.HuellaDecisionSHA256,
		&recibo.AtestacionRef, &recibo.AtestacionCanonica, &recibo.HuellaAtestacionSHA256,
		&recibo.ConsumoRef, &recibo.ConsumoCanonico, &recibo.HuellaConsumoSHA256,
		&recibo.AuditoriaRef, &recibo.RegistroAuditoria, &recibo.HuellaAuditoriaSHA256,
		&recibo.EventoRef, &recibo.EventoCanonico, &recibo.HuellaEventoSHA256,
		&recibo.ConfirmadaEn,
	)
	if err != nil {
		return reciboLlamamientoPostgreSQLV1{}, errorPostgreSQLLlamamiento(ctx, err)
	}
	return recibo, nil
}

type consumoLlamamientoPostgreSQLV1 struct {
	Esquema                string `json:"esquema"`
	ConsumoRef             string `json:"consumo_ref"`
	DecisionRef            string `json:"decision_ref"`
	PrincipalRef           string `json:"principal_ref"`
	PropuestaRef           string `json:"propuesta_ref"`
	NecesidadRef           string `json:"necesidad_ref"`
	VersionNecesidad       uint64 `json:"version_necesidad"`
	HuellaNecesidadSHA256  string `json:"huella_necesidad_sha256"`
	HuellaPropuestaSHA256  string `json:"huella_propuesta_sha256"`
	HuellaDocumentoSHA256  string `json:"huella_documento_sha256"`
	AtestacionRef          string `json:"atestacion_ref"`
	HuellaAtestacionSHA256 string `json:"huella_atestacion_sha256"`
	ConsumidaEn            string `json:"consumida_en"`
}

type auditoriaLlamamientoPostgreSQLV1 struct {
	Esquema               string `json:"esquema"`
	AuditoriaRef          string `json:"auditoria_ref"`
	Secuencia             uint64 `json:"secuencia"`
	HuellaAnteriorSHA256  string `json:"huella_anterior_sha256"`
	ConsumoRef            string `json:"consumo_ref"`
	DecisionRef           string `json:"decision_ref"`
	PropuestaRef          string `json:"propuesta_ref"`
	HuellaPropuestaSHA256 string `json:"huella_propuesta_sha256"`
	HuellaConsumoSHA256   string `json:"huella_consumo_sha256"`
	RegistradaEn          string `json:"registrada_en"`
}

type eventoLlamamientoPostgreSQLV1 struct {
	Esquema               string `json:"esquema"`
	EventoRef             string `json:"evento_ref"`
	Tipo                  string `json:"tipo"`
	AgregadoRef           string `json:"agregado_ref"`
	HuellaPropuestaSHA256 string `json:"huella_propuesta_sha256"`
	AuditoriaRef          string `json:"auditoria_ref"`
	HuellaAuditoriaSHA256 string `json:"huella_auditoria_sha256"`
	EmitidoEn             string `json:"emitido_en"`
}

func (r reciboLlamamientoPostgreSQLV1) validar(
	propuesta dominiobolsa.PropuestaLlamamiento,
	datos puertosvec.DatosEvidenciaUsoDecisionAutorizacion,
	propuestaDocumento []byte,
) error {
	if r.Resultado != "confirmada" && r.Resultado != "repetida" ||
		r.PropuestaRef != propuesta.PropuestaRef || r.HuellaPropuestaSHA256 != propuesta.HuellaContenidoSHA256 ||
		!bytes.Equal(r.PropuestaCanonica, propuestaDocumento) ||
		r.HuellaDocumentoSHA256 != huellaBytesPostgreSQLLlamamiento(propuestaDocumento) ||
		r.DecisionRef != datos.Decision.DecisionRef || r.HuellaDecisionSHA256 != datos.HuellaDecisionSHA256 ||
		!referenciaYHuellaLlamamientoValidas(r.AtestacionRef, r.HuellaAtestacionSHA256) ||
		!referenciaYHuellaLlamamientoValidas(r.ConsumoRef, r.HuellaConsumoSHA256) ||
		!referenciaYHuellaLlamamientoValidas(r.AuditoriaRef, r.HuellaAuditoriaSHA256) ||
		!referenciaYHuellaLlamamientoValidas(r.EventoRef, r.HuellaEventoSHA256) ||
		!r.ConfirmadaEn.Valid || !instantePostgreSQLLlamamientoValido(r.ConfirmadaEn.Time.UTC()) ||
		r.ConfirmadaEn.Time.UTC().Before(propuesta.GeneradaEn) ||
		r.ConfirmadaEn.Time.UTC().Before(datos.VerificadaEn) ||
		!r.ConfirmadaEn.Time.UTC().Before(datos.Decision.ValidaHasta) ||
		r.HuellaAtestacionSHA256 != huellaBytesPostgreSQLLlamamiento(r.AtestacionCanonica) ||
		r.HuellaConsumoSHA256 != huellaBytesPostgreSQLLlamamiento(r.ConsumoCanonico) ||
		r.HuellaAuditoriaSHA256 != huellaBytesPostgreSQLLlamamiento(r.RegistroAuditoria) ||
		r.HuellaEventoSHA256 != huellaBytesPostgreSQLLlamamiento(r.EventoCanonico) {
		return puertosbolsa.ErrPersistenciaPropuestaNoDisponible
	}
	instante := r.ConfirmadaEn.Time.UTC().Format(formatoInstanteMicrosegundo)
	var consumo consumoLlamamientoPostgreSQLV1
	if decodificarJSONExactoLlamamiento(r.ConsumoCanonico, &consumo) != nil ||
		consumo.Esquema != "vec.bolsa.llamamiento.consumo.v1" || consumo.ConsumoRef != r.ConsumoRef ||
		consumo.DecisionRef != r.DecisionRef || consumo.PrincipalRef != datos.Decision.PrincipalID ||
		consumo.PropuestaRef != propuesta.PropuestaRef || consumo.NecesidadRef != propuesta.NecesidadRef ||
		consumo.VersionNecesidad != propuesta.VersionNecesidad ||
		consumo.HuellaNecesidadSHA256 != propuesta.HuellaNecesidadSHA256 ||
		consumo.HuellaPropuestaSHA256 != propuesta.HuellaContenidoSHA256 ||
		consumo.HuellaDocumentoSHA256 != r.HuellaDocumentoSHA256 ||
		consumo.AtestacionRef != r.AtestacionRef || consumo.HuellaAtestacionSHA256 != r.HuellaAtestacionSHA256 ||
		consumo.ConsumidaEn != instante {
		return puertosbolsa.ErrPersistenciaPropuestaNoDisponible
	}
	var auditoria auditoriaLlamamientoPostgreSQLV1
	if decodificarJSONExactoLlamamiento(r.RegistroAuditoria, &auditoria) != nil ||
		auditoria.Esquema != "vec.bolsa.llamamiento.auditoria.v1" || auditoria.AuditoriaRef != r.AuditoriaRef ||
		auditoria.Secuencia == 0 || !huellaPostgreSQLLlamamientoValida(auditoria.HuellaAnteriorSHA256) ||
		auditoria.ConsumoRef != r.ConsumoRef || auditoria.DecisionRef != r.DecisionRef ||
		auditoria.PropuestaRef != r.PropuestaRef || auditoria.HuellaPropuestaSHA256 != r.HuellaPropuestaSHA256 ||
		auditoria.HuellaConsumoSHA256 != r.HuellaConsumoSHA256 || auditoria.RegistradaEn != instante {
		return puertosbolsa.ErrPersistenciaPropuestaNoDisponible
	}
	var evento eventoLlamamientoPostgreSQLV1
	if decodificarJSONExactoLlamamiento(r.EventoCanonico, &evento) != nil ||
		evento.Esquema != "vec.bolsa.llamamiento.outbox.v1" || evento.EventoRef != r.EventoRef ||
		evento.Tipo != "bolsa.llamamiento.propuesta_confirmada.v1" || evento.AgregadoRef != r.PropuestaRef ||
		evento.HuellaPropuestaSHA256 != r.HuellaPropuestaSHA256 || evento.AuditoriaRef != r.AuditoriaRef ||
		evento.HuellaAuditoriaSHA256 != r.HuellaAuditoriaSHA256 || evento.EmitidoEn != instante {
		return puertosbolsa.ErrPersistenciaPropuestaNoDisponible
	}
	return nil
}

func (r *reciboLlamamientoPostgreSQLV1) borrar() {
	if r == nil {
		return
	}
	borrarBytesPostgreSQL(
		r.PropuestaCanonica, r.AtestacionCanonica, r.ConsumoCanonico,
		r.RegistroAuditoria, r.EventoCanonico,
	)
}

func referenciaYHuellaLlamamientoValidas(referencia, huella string) bool {
	return puertosbolsa.ReferenciaOpacaLlamamientoValida(referencia) && huellaPostgreSQLLlamamientoValida(huella)
}

func huellaPostgreSQLLlamamientoValida(huella string) bool {
	if len(huella) != 64 {
		return false
	}
	_, err := strconv.ParseUint(huella[:16], 16, 64)
	if err != nil {
		return false
	}
	for _, caracter := range huella {
		if (caracter < '0' || caracter > '9') && (caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}

func errorPostgreSQLLlamamiento(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var pgError *pgconn.PgError
	if errors.As(err, &pgError) {
		if pgError.Code == "23505" {
			switch pgError.ConstraintName {
			case "propuesta_pkey":
				return puertosbolsa.ErrPropuestaLlamamientoYaExiste
			case "propuesta_necesidad_unica":
				return puertosbolsa.ErrNecesidadLlamamientoYaPropuesta
			case "propuesta_decision_unica", "uso_decision_pkey":
				return errors.Join(
					puertosbolsa.ErrDecisionAutorizacionLlamamientoUsada,
					puertosvec.ErrDecisionAutorizacionConsumida,
				)
			case "propuesta_instantanea_unica", "referencia_consumida_pkey":
				return puertosbolsa.ErrReferenciaLlamamientoYaUtilizada
			default:
				return puertosbolsa.ErrPersistenciaPropuestaNoDisponible
			}
		}
		switch pgError.Code {
		case "PBL01":
			return puertosbolsa.ErrPropuestaLlamamientoYaExiste
		case "PBL02":
			return puertosbolsa.ErrNecesidadLlamamientoYaPropuesta
		case "PBL03":
			return errors.Join(
				puertosbolsa.ErrDecisionAutorizacionLlamamientoUsada,
				puertosvec.ErrDecisionAutorizacionConsumida,
			)
		case "PBL04":
			return puertosbolsa.ErrReferenciaLlamamientoYaUtilizada
		case "42501", "22000", "22023", "23503", "23514", "55000":
			return puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida
		case "40001", "40P01", "55P03", "57014":
			return puertosbolsa.ErrPersistenciaPropuestaNoDisponible
		}
	}
	return puertosbolsa.ErrPersistenciaPropuestaNoDisponible
}
