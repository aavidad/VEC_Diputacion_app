package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/shared/plazoarranque"
	postgresqlcomun "vec-diputacion-granada/internal/shared/postgresql"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const consultaAppendIntentoAuditoria = `
	SELECT auditoria_ref, secuencia::bigint, huella_sha256,
	       correlacion_ref, registrada_en
	FROM vec_autorizacion_atestada_v3.registrar_intento_nominal_v1(
		$1::bytea, $2::bytea, $3::jsonb
	)`

const consultaPreflightIntentoAuditoria = `
	SELECT vec_autorizacion_atestada_v3.preflight_registrador_intentos_v1(
		$1::text, $2::text
	)`

type iniciadorIntentoAuditoriaPostgreSQL interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

// RegistradorIntentosAuditoriaPostgreSQL usa el mismo registro y cadena que
// los consumos V3. Proceso es configuración del servidor, no dato de HTTP.
type RegistradorIntentosAuditoriaPostgreSQL struct {
	pool    iniciadorIntentoAuditoriaPostgreSQL
	proceso string
	canal   string
	limite  time.Duration
}

var _ ports.RegistradorIntentosAuditoria = (*RegistradorIntentosAuditoriaPostgreSQL)(nil)

func NuevoRegistradorIntentosAuditoriaPostgreSQL(
	pool *pgxpool.Pool, proceso, canal string, limite time.Duration,
) (*RegistradorIntentosAuditoriaPostgreSQL, error) {
	return nuevoRegistradorIntentosAuditoriaPostgreSQL(pool, proceso, canal, limite)
}

func nuevoRegistradorIntentosAuditoriaPostgreSQL(
	pool iniciadorIntentoAuditoriaPostgreSQL, proceso, canal string, limite time.Duration,
) (*RegistradorIntentosAuditoriaPostgreSQL, error) {
	if valorNuloPostgreSQL(pool) || !procesoIntentoAuditoriaValido(proceso) ||
		!canalIntentoAuditoriaValido(canal) ||
		limite <= 0 || limite > 30*time.Second {
		return nil, ports.ErrIntentoAuditoriaNoDisponible
	}
	return &RegistradorIntentosAuditoriaPostgreSQL{
		pool: pool, proceso: proceso, canal: canal, limite: limite,
	}, nil
}

// Preflight exige EXECUTE de la función exacta antes de montar un consumidor.
func (r *RegistradorIntentosAuditoriaPostgreSQL) PreflightIntentoAuditoria(ctx context.Context) error {
	if r == nil || valorNuloPostgreSQL(r.pool) || ctx == nil {
		return ports.ErrIntentoAuditoriaNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var permitido bool
	if err := r.pool.QueryRow(ctx, consultaPreflightIntentoAuditoria, r.proceso, r.canal).Scan(&permitido); err != nil || !permitido {
		return ports.ErrIntentoAuditoriaNoDisponible
	}
	return nil
}

// AppendIntentoAuditoria se llama después del rollback o del retorno del
// repositorio fallido. La nueva transacción no hereda la cancelación HTTP.
// Ante COMMIT ambiguo, el consumidor reintenta la misma orden; SQL entrega
// el recibo anterior solo si la huella canónica coincide.
func (r *RegistradorIntentosAuditoriaPostgreSQL) AppendIntentoAuditoria(
	ctx context.Context, orden ports.OrdenIntentoAuditoria,
) (ports.AcuseIntentoAuditoria, error) {
	if r == nil || valorNuloPostgreSQL(r.pool) || ctx == nil {
		return ports.AcuseIntentoAuditoria{}, ports.ErrIntentoAuditoriaNoDisponible
	}
	datos, err := orden.Datos()
	if err != nil || datos.Datos.Proceso != r.proceso || datos.Datos.Canal != r.canal {
		return ports.AcuseIntentoAuditoria{}, ports.ErrOrdenIntentoAuditoriaInvalida
	}
	// Context.WithoutCancel conserva correlación de trazas, pero el tiempo de
	// auditoría es independiente de la vida de la conexión HTTP.
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(r.limite))
	defer cancelar()
	var acuse ports.AcuseIntentoAuditoria
	err = postgresqlcomun.RepetirTrasCarreraSerializable(ctxAuditoria, func() error {
		var errIntento error
		acuse, errIntento = r.appendEnTransaccion(ctxAuditoria, datos)
		return errIntento
	})
	if err != nil {
		return ports.AcuseIntentoAuditoria{}, errorIntentoAuditoriaPostgreSQL(err)
	}
	if err = acuse.ValidarPara(orden); err != nil {
		return ports.AcuseIntentoAuditoria{}, ports.ErrIntentoAuditoriaNoDisponible
	}
	return acuse, nil
}

type ordenIntentoAuditoriaSQL struct {
	IntentoRef          string `json:"intento_ref"`
	RegistroContextoRef string `json:"registro_contexto_ref"`
	ContextoSHA256      string `json:"contexto_sha256"`
	ProcedenciaSHA256   string `json:"procedencia_sha256"`
	AutenticacionRef    string `json:"autenticacion_ref"`
	SesionRef           string `json:"sesion_ref"`
	AutenticacionSHA256 string `json:"autenticacion_sha256"`
	Accion              string `json:"accion"`
	ModuloID            string `json:"modulo_id"`
	RecursoRef          string `json:"recurso_ref"`
	FinalidadRef        string `json:"finalidad_ref"`
	Resultado           string `json:"resultado"`
	MotivoRef           string `json:"motivo_ref"`
	Proceso             string `json:"proceso"`
	Canal               string `json:"canal"`
	CorrelacionRef      string `json:"correlacion_ref"`
}

func serializarIntentoAuditoria(
	datos ports.DatosOrdenIntentoAuditoria,
) (contextoCanonico, vinculoCanonico, ordenCanonica []byte, err error) {
	vinculo, err := datos.Vinculo.Datos()
	if err != nil || datos.Vinculo.ValidarPara(datos.ResultadoContexto) != nil ||
		datos.Datos.Validar() != nil {
		return nil, nil, nil, ports.ErrOrdenIntentoAuditoriaInvalida
	}
	vinculoCanonico, err = json.Marshal(vinculo)
	if err != nil {
		return nil, nil, nil, ports.ErrOrdenIntentoAuditoriaInvalida
	}
	ordenSQL := ordenIntentoAuditoriaSQL{
		IntentoRef:          datos.IntentoRef,
		RegistroContextoRef: datos.ResultadoContexto.RegistroContextoRef,
		ContextoSHA256:      datos.ResultadoContexto.HuellaSHA256,
		ProcedenciaSHA256:   datos.ResultadoContexto.ManifiestoProcedenciaHuellaSHA256,
		AutenticacionRef:    vinculo.AutenticacionRef,
		SesionRef:           vinculo.SesionRef,
		AutenticacionSHA256: vinculo.AutenticacionHuellaSHA256,
		Accion:              datos.Datos.Accion,
		ModuloID:            datos.Datos.ModuloID,
		RecursoRef:          datos.Datos.RecursoRef,
		FinalidadRef:        datos.Datos.FinalidadRef,
		Resultado:           string(datos.Datos.Resultado),
		MotivoRef:           datos.Datos.Motivo.Referencia(),
		Proceso:             datos.Datos.Proceso,
		Canal:               datos.Datos.Canal,
		CorrelacionRef:      datos.Datos.CorrelacionRef,
	}
	ordenCanonica, err = json.Marshal(ordenSQL)
	if err != nil {
		return nil, nil, nil, ports.ErrOrdenIntentoAuditoriaInvalida
	}
	return append([]byte(nil), datos.ResultadoContexto.RepresentacionCanonica...), vinculoCanonico, ordenCanonica, nil
}

func (r *RegistradorIntentosAuditoriaPostgreSQL) appendEnTransaccion(
	ctx context.Context, datos ports.DatosOrdenIntentoAuditoria,
) (ports.AcuseIntentoAuditoria, error) {
	contextoCanonico, vinculoCanonico, ordenCanonica, err := serializarIntentoAuditoria(datos)
	if err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil || valorNuloPostgreSQL(tx) {
		if err != nil {
			return ports.AcuseIntentoAuditoria{}, err
		}
		return ports.AcuseIntentoAuditoria{}, ports.ErrIntentoAuditoriaNoDisponible
	}
	defer revertirTransaccionPostgreSQL(tx)
	if err = configurarTransaccionAutorizacion(ctx, tx); err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	var acuse ports.AcuseIntentoAuditoria
	if err = tx.QueryRow(ctx, consultaAppendIntentoAuditoria,
		contextoCanonico, vinculoCanonico, ordenCanonica,
	).Scan(&acuse.AuditoriaRef, &acuse.Secuencia, &acuse.HuellaSHA256,
		&acuse.CorrelacionRef, &acuse.RegistradaEn); err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	acuse.RegistradaEn = acuse.RegistradaEn.UTC()
	if err = tx.Commit(ctx); err != nil {
		return ports.AcuseIntentoAuditoria{}, err
	}
	return acuse, nil
}

func procesoIntentoAuditoriaValido(valor string) bool {
	if len(valor) < 2 || len(valor) > 80 || valor[0] < 'a' || valor[0] > 'z' {
		return false
	}
	for _, r := range valor[1:] {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' && r != '_' && r != '.' {
			return false
		}
	}
	return true
}

func canalIntentoAuditoriaValido(valor string) bool {
	switch domain.SuperficieAutenticacionActorV1(valor) {
	case domain.SuperficieAutenticacionAdministracionPrivilegiadaV1,
		domain.SuperficieAutenticacionInternaCorporativaV1,
		domain.SuperficieAutenticacionExternaPersonalV1:
		return true
	default:
		return false
	}
}

func errorIntentoAuditoriaPostgreSQL(err error) error {
	if errors.Is(err, ports.ErrOrdenIntentoAuditoriaInvalida) {
		return err
	}
	var errorPG *pgconn.PgError
	if errors.As(err, &errorPG) && errorPG.Code == "23505" {
		return ports.ErrIntentoAuditoriaConflicto
	}
	return ports.ErrIntentoAuditoriaNoDisponible
}
