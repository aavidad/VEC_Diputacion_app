// La consulta de gobierno de convocatorias es deliberadamente transaccional:
// incluso una lectura consume una decision de autorizacion y deja auditoria.
// La cuenta de ejecucion solo puede invocar la funcion SECURITY DEFINER de
// contrato cerrado; no recibe permisos directos sobre las tablas.
package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	funcionObtenerVersionExactaConvocatoriaV1 = "vec_bolsa_convocatorias.obtener_version_exacta_v1"
	esquemaOperacionConsultaConvocatoriaV1    = "vec.bolsa.convocatoria.consulta-postgresql.v1"
	maximoBytesInstanciaFlujoConvocatoria     = 2 * 1024 * 1024
)

var _ puertosbolsa.ConsultaGobiernoConvocatorias = (*ConsultaGobiernoConvocatoriasPostgreSQL)(nil)

// ConsultaGobiernoConvocatoriasPostgreSQL recupera una version exacta. No
// contiene busquedas por "ultima", listados amplios ni rutas degradadas.
type ConsultaGobiernoConvocatoriasPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevaConsultaGobiernoConvocatoriasPostgreSQL(
	pool *pgxpool.Pool,
) (*ConsultaGobiernoConvocatoriasPostgreSQL, error) {
	return nuevaConsultaGobiernoConvocatoriasPostgreSQL(pool)
}

func nuevaConsultaGobiernoConvocatoriasPostgreSQL(
	pool iniciadorTransacciones,
) (*ConsultaGobiernoConvocatoriasPostgreSQL, error) {
	if valorNulo(pool) {
		return nil, puertosbolsa.ErrFuenteGobiernoConvocatoriasNoDisponible
	}
	return &ConsultaGobiernoConvocatoriasPostgreSQL{pool: pool}, nil
}

type operacionConsultaConvocatoriaPostgreSQL struct {
	Esquema               string `json:"esquema"`
	ConvocatoriaID        string `json:"convocatoria_id"`
	Secuencia             int    `json:"secuencia"`
	IncluirInstanciaFlujo bool   `json:"incluir_instancia_flujo"`
	Accion                string `json:"accion"`
	RecursoRef            string `json:"recurso_ref"`
	SolicitadaEn          string `json:"solicitada_en"`
}

type pruebaConsultaConvocatoriaPostgreSQL struct {
	EsquemaHuella        string `json:"esquema_huella"`
	DecisionRef          string `json:"decision_ref"`
	HuellaDecisionSHA256 string `json:"huella_decision_sha256"`
	VerificadaEn         string `json:"verificada_en"`
	PrincipalRef         string `json:"principal_ref"`
}

type contextoRecursoConsultaConvocatoriaPostgreSQL struct {
	Ambitos   map[string]string `json:"ambitos"`
	Atributos map[string]string `json:"atributos"`
}

func (r *ConsultaGobiernoConvocatoriasPostgreSQL) ObtenerVersionExacta(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudConsultaVersionConvocatoriaAutorizada,
) (puertosbolsa.ResultadoConsultaVersionConvocatoria, error) {
	if ctx == nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{},
			puertosbolsa.ErrConsultaGobiernoConvocatoriaInvalida
	}
	if err := ctx.Err(); err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, err
	}
	if r == nil || valorNulo(r.pool) {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{},
			puertosbolsa.ErrFuenteGobiernoConvocatoriasNoDisponible
	}
	if solicitud.Validar() != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{},
			puertosbolsa.ErrConsultaGobiernoConvocatoriaInvalida
	}

	operacion, prueba, decisionCanonica, recursoCanonico, err :=
		serializarConsultaConvocatoriaPostgreSQL(solicitud)
	if err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, err
	}
	defer borrarBytesPostgreSQL(operacion, prueba, decisionCanonica, recursoCanonico)

	tx, err := r.iniciar(ctx)
	if err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, err
	}
	defer revertir(tx)

	var (
		estado, huellaVersion, huellaInstancia      string
		autorizacionRef, huellaAutorizacion         string
		atestacionRef, huellaAtestacion, consumoRef string
		auditoriaRef, huellaAuditoria               string
		versionCanonica, instanciaCanonica          []byte
		consultadaEn                                pgtype.Timestamptz
	)
	err = tx.QueryRow(ctx, `
		SELECT resultado, version_canonica, huella_version_sha256,
		       instancia_flujo_canonica, huella_instancia_flujo_sha256,
		       autorizacion_ref, huella_autorizacion_sha256,
		       atestacion_autorizacion_ref, huella_atestacion_autorizacion_sha256,
		       consumo_autorizacion_ref, auditoria_ref, huella_auditoria_sha256,
		       consultada_en
		FROM `+funcionObtenerVersionExactaConvocatoriaV1+`(
			$1::jsonb, $2::jsonb, $3::bytea, $4::bytea
		)`, operacion, prueba, decisionCanonica, recursoCanonico,
	).Scan(
		&estado, &versionCanonica, &huellaVersion,
		&instanciaCanonica, &huellaInstancia,
		&autorizacionRef, &huellaAutorizacion,
		&atestacionRef, &huellaAtestacion, &consumoRef,
		&auditoriaRef, &huellaAuditoria, &consultadaEn,
	)
	defer borrarBytesPostgreSQL(versionCanonica, instanciaCanonica)
	if err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{},
			errorPostgreSQLConsultaConvocatoria(ctx, err)
	}
	if estado == "no_encontrada" {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{},
			puertosbolsa.ErrVersionGobernadaConvocatoriaNoEncontrada
	}
	if estado != "obtenida" || !consultadaEn.Valid {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{},
			puertosbolsa.ErrEvidenciaConsultaConvocatoriaNoConfiable
	}

	version, err := dominiobolsa.DecodificarVersionConvocatoriaGobernadaCanonica(versionCanonica)
	if err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{},
			puertosbolsa.ErrEvidenciaConsultaConvocatoriaNoConfiable
	}
	instancia, err := decodificarInstanciaFlujoConvocatoriaCanonica(instanciaCanonica)
	if err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{}, err
	}
	resultado := puertosbolsa.ResultadoConsultaVersionConvocatoria{
		Version: version, InstanciaFlujo: instancia,
		HuellaVersionSHA256: huellaVersion, HuellaInstanciaFlujoSHA256: huellaInstancia,
		AutorizacionRef: autorizacionRef, HuellaAutorizacionSHA256: huellaAutorizacion,
		AtestacionAutorizacionRef:          atestacionRef,
		HuellaAtestacionAutorizacionSHA256: huellaAtestacion,
		ConsumoAutorizacionRef:             consumoRef, AuditoriaRef: auditoriaRef,
		HuellaAuditoriaSHA256: huellaAuditoria, ConsultadaEn: consultadaEn.Time.UTC(),
	}
	if resultado.ValidarPara(solicitud) != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{},
			puertosbolsa.ErrEvidenciaConsultaConvocatoriaNoConfiable
	}
	if err := tx.Commit(ctx); err != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{},
			errorPostgreSQLConsultaConvocatoria(ctx, err)
	}
	resultado, err = resultado.Clonar()
	if err != nil || resultado.ValidarPara(solicitud) != nil {
		return puertosbolsa.ResultadoConsultaVersionConvocatoria{},
			puertosbolsa.ErrEvidenciaConsultaConvocatoriaNoConfiable
	}
	return resultado, nil
}

func serializarConsultaConvocatoriaPostgreSQL(
	solicitud puertosbolsa.SolicitudConsultaVersionConvocatoriaAutorizada,
) ([]byte, []byte, []byte, []byte, error) {
	if solicitud.Validar() != nil {
		return nil, nil, nil, nil, puertosbolsa.ErrConsultaGobiernoConvocatoriaInvalida
	}
	datos, err := solicitud.Autorizacion.Datos()
	if err != nil {
		return nil, nil, nil, nil, puertosbolsa.ErrConsultaGobiernoConvocatoriaInvalida
	}
	decisionCanonica, err := datos.RepresentacionCanonica()
	if err != nil {
		return nil, nil, nil, nil, puertosbolsa.ErrConsultaGobiernoConvocatoriaInvalida
	}
	recurso, err := puertosbolsa.RecursoAutorizableConsultaVersionConvocatoria(solicitud.Selector)
	if err != nil {
		return nil, nil, nil, nil, puertosbolsa.ErrConsultaGobiernoConvocatoriaInvalida
	}
	recursoCanonico, err := json.Marshal(contextoRecursoConsultaConvocatoriaPostgreSQL{
		Ambitos: map[string]string{}, Atributos: map[string]string{},
	})
	if err != nil {
		return nil, nil, nil, nil, puertosbolsa.ErrConsultaGobiernoConvocatoriaInvalida
	}
	huellaRecurso := sha256.Sum256(recursoCanonico)
	if hex.EncodeToString(huellaRecurso[:]) != datos.Decision.ContextoRecursoHuellaSHA256 {
		return nil, nil, nil, nil, puertosbolsa.ErrConsultaGobiernoConvocatoriaInvalida
	}
	operacion, err := json.Marshal(operacionConsultaConvocatoriaPostgreSQL{
		Esquema:        esquemaOperacionConsultaConvocatoriaV1,
		ConvocatoriaID: solicitud.Selector.ID, Secuencia: solicitud.Selector.Secuencia,
		IncluirInstanciaFlujo: solicitud.IncluirInstanciaFlujo,
		Accion:                datos.Decision.Accion, RecursoRef: recurso.Referencia,
		SolicitadaEn: solicitud.ConsultadaEn.UTC().Format(formatoInstanteMicrosegundo),
	})
	if err != nil {
		return nil, nil, nil, nil, puertosbolsa.ErrConsultaGobiernoConvocatoriaInvalida
	}
	prueba, err := json.Marshal(pruebaConsultaConvocatoriaPostgreSQL{
		EsquemaHuella: datos.EsquemaHuella, DecisionRef: datos.Decision.DecisionRef,
		HuellaDecisionSHA256: datos.HuellaDecisionSHA256,
		VerificadaEn:         datos.VerificadaEn.UTC().Format(formatoInstanteMicrosegundo),
		PrincipalRef:         datos.Decision.PrincipalID,
	})
	if err != nil {
		return nil, nil, nil, nil, puertosbolsa.ErrConsultaGobiernoConvocatoriaInvalida
	}
	return operacion, prueba, decisionCanonica, recursoCanonico, nil
}

func (r *ConsultaGobiernoConvocatoriasPostgreSQL) iniciar(ctx context.Context) (pgx.Tx, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return nil, errorPostgreSQLConsultaConvocatoria(ctx, err)
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
		return nil, errorPostgreSQLConsultaConvocatoria(ctx, err)
	}
	return tx, nil
}

func decodificarInstanciaFlujoConvocatoriaCanonica(
	contenido []byte,
) (*dominiovec.InstanciaFlujo, error) {
	if len(contenido) == 0 {
		return nil, nil
	}
	if len(contenido) > maximoBytesInstanciaFlujoConvocatoria || !utf8.Valid(contenido) {
		return nil, puertosbolsa.ErrEvidenciaConsultaConvocatoriaNoConfiable
	}
	var instancia dominiovec.InstanciaFlujo
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&instancia); err != nil {
		return nil, puertosbolsa.ErrEvidenciaConsultaConvocatoriaNoConfiable
	}
	var resto any
	if err := decodificador.Decode(&resto); !errors.Is(err, io.EOF) || instancia.Validar() != nil {
		return nil, puertosbolsa.ErrEvidenciaConsultaConvocatoriaNoConfiable
	}
	canonica := instancia
	canonica.CreadaEn = instancia.CreadaEn.UTC()
	if !instancia.ActualizadaEn.IsZero() {
		canonica.ActualizadaEn = instancia.ActualizadaEn.UTC()
	}
	representacion, err := json.Marshal(canonica)
	if err != nil || !bytes.Equal(representacion, contenido) {
		return nil, puertosbolsa.ErrEvidenciaConsultaConvocatoriaNoConfiable
	}
	return &canonica, nil
}

func errorPostgreSQLConsultaConvocatoria(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	var errorPostgreSQL *pgconn.PgError
	if errors.As(err, &errorPostgreSQL) {
		switch errorPostgreSQL.Code {
		case "40001", "40P01", "55P03", "57014":
			return puertosbolsa.ErrConsultaGobiernoConvocatoriaEnCurso
		case "22000", "22023", "23503", "23514", "55000":
			return puertosbolsa.ErrEvidenciaConsultaConvocatoriaNoConfiable
		}
	}
	return puertosbolsa.ErrFuenteGobiernoConvocatoriasNoDisponible
}
