// La persistencia de propuestas de llamamiento es deliberadamente una sola
// transaccion durable. La cuenta runtime no toca tablas: invoca una funcion
// SECURITY DEFINER de contrato cerrado que vuelve a comprobar toda autoridad.
package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	funcionGuardarPropuestaLlamamientoV1 = "vec_bolsa_llamamientos.guardar_propuesta_v1"
	esquemaOperacionLlamamientoV1        = "vec.bolsa.llamamiento.guardar-postgresql.v1"
	maximoDocumentoLlamamiento           = 32 * 1024 * 1024
)

var _ puertosbolsa.TransaccionPropuestasLlamamiento = (*TransaccionPropuestasLlamamientoPostgreSQL)(nil)

// TransaccionPropuestasLlamamientoPostgreSQL permanece cerrada aunque exista
// la funcion guardar_propuesta_v1: ese contrato antiguo no recibe ni confirma
// la instantanea completa generada. Conceder EXECUTE no basta para habilitarla.
type TransaccionPropuestasLlamamientoPostgreSQL struct {
	pool  iniciadorTransacciones
	reloj puertosbolsa.RelojLlamamientos
}

func NuevaTransaccionPropuestasLlamamientoPostgreSQL(
	pool *pgxpool.Pool,
	reloj puertosbolsa.RelojLlamamientos,
) (*TransaccionPropuestasLlamamientoPostgreSQL, error) {
	return nuevaTransaccionPropuestasLlamamientoPostgreSQL(pool, reloj)
}

func nuevaTransaccionPropuestasLlamamientoPostgreSQL(
	pool iniciadorTransacciones,
	reloj puertosbolsa.RelojLlamamientos,
) (*TransaccionPropuestasLlamamientoPostgreSQL, error) {
	if valorNulo(pool) || valorNulo(reloj) || !instantePostgreSQLLlamamientoValido(reloj.Ahora()) {
		return nil, puertosbolsa.ErrPersistenciaPropuestaNoDisponible
	}
	return &TransaccionPropuestasLlamamientoPostgreSQL{pool: pool, reloj: reloj}, nil
}

type operacionLlamamientoPostgreSQLV1 struct {
	Esquema               string `json:"esquema"`
	PropuestaRef          string `json:"propuesta_ref"`
	NecesidadRef          string `json:"necesidad_ref"`
	VersionNecesidad      uint64 `json:"version_necesidad"`
	HuellaNecesidadSHA256 string `json:"huella_necesidad_sha256"`
	HuellaPropuestaSHA256 string `json:"huella_propuesta_sha256"`
	HuellaDocumentoSHA256 string `json:"huella_documento_sha256"`
	Accion                string `json:"accion"`
	Finalidad             string `json:"finalidad"`
	TipoRecurso           string `json:"tipo_recurso"`
	SolicitadaEn          string `json:"solicitada_en"`
}

type pruebaLlamamientoPostgreSQLV1 struct {
	EsquemaHuella        string `json:"esquema_huella"`
	DecisionRef          string `json:"decision_ref"`
	HuellaDecisionSHA256 string `json:"huella_decision_sha256"`
	VerificadaEn         string `json:"verificada_en"`
	PrincipalRef         string `json:"principal_ref"`
}

// GuardarPropuestaLlamamiento valida el nuevo comando indivisible y falla
// cerrado antes de iniciar una transaccion. TODO(produccion): sustituir
// guardar_propuesta_v1 por un contrato SQL nuevo que inserte la instantanea
// completa, todas sus entradas, el prefijo de evaluaciones, la propuesta, el
// consumo de autorizacion, la atestacion COSE, auditoria y outbox en un unico
// COMMIT; solo entonces podra retirarse este cierre explicito.
func (r *TransaccionPropuestasLlamamientoPostgreSQL) GuardarPropuestaLlamamiento(
	ctx context.Context,
	comando puertosbolsa.ComandoGuardarPropuestaLlamamiento,
) error {
	if ctx == nil {
		return puertosbolsa.ErrPersistenciaPropuestaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if r == nil || valorNulo(r.pool) || valorNulo(r.reloj) {
		return puertosbolsa.ErrPersistenciaPropuestaNoDisponible
	}
	_, propuestaCanonica, _, err := comando.Datos()
	if err != nil {
		return errors.Join(puertosbolsa.ErrPersistenciaPropuestaNoDisponible, err)
	}
	ahora := r.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !instantePostgreSQLLlamamientoValido(ahora) || propuestaCanonica.GeneradaEn.After(ahora) || comando.ValidarEn(ahora) != nil {
		return puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	return puertosbolsa.ErrPersistenciaPropuestaNoDisponible
}

func (r *TransaccionPropuestasLlamamientoPostgreSQL) iniciar(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, errorPostgreSQLLlamamiento(ctx, err)
	}
	_, err = tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`)
	if err != nil {
		revertir(tx)
		return nil, errorPostgreSQLLlamamiento(ctx, err)
	}
	return tx, nil
}

func serializarPropuestaLlamamientoPostgreSQL(
	propuesta dominiobolsa.PropuestaLlamamiento,
	evidencia puertosvec.EvidenciaUsoDecisionAutorizacion,
	ahora time.Time,
) ([]byte, []byte, []byte, []byte, puertosvec.DatosEvidenciaUsoDecisionAutorizacion, error) {
	datos, err := evidencia.Datos()
	if err != nil || !decisionPostgreSQLLlamamientoExacta(datos.Decision, propuesta) {
		return nil, nil, nil, nil, puertosvec.DatosEvidenciaUsoDecisionAutorizacion{},
			puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	decisionCanonica, err := datos.RepresentacionCanonica()
	if err != nil {
		return nil, nil, nil, nil, puertosvec.DatosEvidenciaUsoDecisionAutorizacion{},
			puertosvec.ErrEvidenciaUsoDecisionAutorizacionInvalida
	}
	propuestaDocumento, err := json.Marshal(propuesta)
	if err != nil || len(propuestaDocumento) < 2 || len(propuestaDocumento) > maximoDocumentoLlamamiento {
		return nil, nil, nil, nil, puertosvec.DatosEvidenciaUsoDecisionAutorizacion{},
			puertosbolsa.ErrPersistenciaPropuestaNoDisponible
	}
	huellaDocumento := huellaBytesPostgreSQLLlamamiento(propuestaDocumento)
	operacion, err := json.Marshal(operacionLlamamientoPostgreSQLV1{
		Esquema: esquemaOperacionLlamamientoV1, PropuestaRef: propuesta.PropuestaRef,
		NecesidadRef: propuesta.NecesidadRef, VersionNecesidad: propuesta.VersionNecesidad,
		HuellaNecesidadSHA256: propuesta.HuellaNecesidadSHA256,
		HuellaPropuestaSHA256: propuesta.HuellaContenidoSHA256,
		HuellaDocumentoSHA256: huellaDocumento, Accion: puertosbolsa.AccionProponerLlamamiento,
		Finalidad: puertosbolsa.FinalidadProponerLlamamiento, TipoRecurso: puertosbolsa.TipoRecursoNecesidad,
		SolicitadaEn: ahora.Format(formatoInstanteMicrosegundo),
	})
	if err != nil {
		return nil, nil, nil, nil, puertosvec.DatosEvidenciaUsoDecisionAutorizacion{},
			puertosbolsa.ErrPersistenciaPropuestaNoDisponible
	}
	prueba, err := json.Marshal(pruebaLlamamientoPostgreSQLV1{
		EsquemaHuella: datos.EsquemaHuella, DecisionRef: datos.Decision.DecisionRef,
		HuellaDecisionSHA256: datos.HuellaDecisionSHA256,
		VerificadaEn:         datos.VerificadaEn.UTC().Format(formatoInstanteMicrosegundo),
		PrincipalRef:         datos.Decision.PrincipalID,
	})
	if err != nil {
		return nil, nil, nil, nil, puertosvec.DatosEvidenciaUsoDecisionAutorizacion{},
			puertosbolsa.ErrPersistenciaPropuestaNoDisponible
	}
	return operacion, prueba, decisionCanonica, propuestaDocumento, datos, nil
}

func decisionPostgreSQLLlamamientoExacta(
	decision dominiovec.DecisionAutorizacion,
	propuesta dominiobolsa.PropuestaLlamamiento,
) bool {
	vinculo, err := decision.VinculoAutenticacionActor.Datos()
	if err != nil {
		return false
	}
	superficieValida := vinculo.Superficie == dominiovec.SuperficieAutenticacionInternaCorporativaV1 &&
		!vinculo.CuentaPrivilegiada ||
		vinculo.Superficie == dominiovec.SuperficieAutenticacionAdministracionPrivilegiadaV1 &&
			vinculo.CuentaPrivilegiada
	return decision.ValidarEvidenciaInstantanea() == nil && decision.Concedida &&
		decision.Accion == puertosbolsa.AccionProponerLlamamiento &&
		decision.RecursoRef == propuesta.NecesidadRef && decision.ModuloID == puertosbolsa.ModuloLlamamientos &&
		decision.TipoRecurso == puertosbolsa.TipoRecursoNecesidad &&
		decision.Finalidad == puertosbolsa.FinalidadProponerLlamamiento &&
		decision.GarantiaMinima == dominiovec.AuthAssuranceHigh && len(decision.CamposPermitidos) == 0 &&
		len(decision.Obligaciones) == 0 && vinculo.GarantiaObservada == dominiovec.AuthAssuranceHigh &&
		vinculo.MetodoObservado != dominiovec.AuthMethodDemo && superficieValida
}

func instantePostgreSQLLlamamientoValido(instante time.Time) bool {
	return !instante.IsZero() && instante.Year() >= 1 && instante.Year() <= 9999 &&
		instante.Nanosecond()%1_000 == 0
}

func huellaBytesPostgreSQLLlamamiento(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}

func decodificarJSONExactoLlamamiento(contenido []byte, destino any) error {
	if err := validarJSONLlamamientoNoAmbiguo(contenido); err != nil {
		return err
	}
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(destino); err != nil {
		return err
	}
	var resto any
	if err := decodificador.Decode(&resto); !errors.Is(err, io.EOF) {
		return errors.New("contenido JSON no canonico")
	}
	return nil
}
