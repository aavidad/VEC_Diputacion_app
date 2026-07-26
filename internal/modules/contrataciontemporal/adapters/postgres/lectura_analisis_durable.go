package postgres

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	funcionLeerAnalisisDurableO3        = "vec_contratacion_temporal.leer_expediente_analisis_durable_o3_v1"
	maximoIntentosLeerAnalisisDurableO3 = 3
	maximoBytesExpedienteDurableO3      = 262_144
	maximaProfundidadExpedienteO3       = 24
	maximosElementosJSONExpedienteO3    = 16_384
)

var _ cobertura.LectorExpedienteAnalisisDurableO3 = (*LectorExpedienteAnalisisDurableO3PostgreSQL)(nil)

// LectorExpedienteAnalisisDurableO3PostgreSQL restaura exclusivamente la
// versión vigente O3 confirmada mediante la función cerrada del módulo.
type LectorExpedienteAnalisisDurableO3PostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevoLectorExpedienteAnalisisDurableO3PostgreSQL(
	pool *pgxpool.Pool,
) (*LectorExpedienteAnalisisDurableO3PostgreSQL, error) {
	return nuevoLectorExpedienteAnalisisDurableO3PostgreSQL(pool)
}

func nuevoLectorExpedienteAnalisisDurableO3PostgreSQL(
	pool iniciadorTransacciones,
) (*LectorExpedienteAnalisisDurableO3PostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, cobertura.ErrInstantaneaAnalisisDurableNoDisponible
	}
	return &LectorExpedienteAnalisisDurableO3PostgreSQL{pool: pool}, nil
}

func (l *LectorExpedienteAnalisisDurableO3PostgreSQL) LeerExpedienteAnalisisDurableO3(
	ctx context.Context,
	solicitud cobertura.SolicitudInstantaneaAnalisisDurableO3,
) (domain.Expediente, error) {
	if ctx == nil || l == nil || dependenciaNula(l.pool) {
		return domain.Expediente{},
			cobertura.ErrSolicitudInstantaneaAnalisisDurableInvalida
	}
	organizacionRef, expedienteRef, versionEsperada, err :=
		solicitud.Coordenadas()
	if err != nil {
		return domain.Expediente{},
			cobertura.ErrSolicitudInstantaneaAnalisisDurableInvalida
	}
	for intento := 1; intento <= maximoIntentosLeerAnalisisDurableO3; intento++ {
		expediente, err := l.leerEnTransaccion(
			ctx,
			organizacionRef,
			expedienteRef,
			versionEsperada,
		)
		if err == nil {
			return expediente, nil
		}
		if ctx.Err() != nil {
			return domain.Expediente{}, ctx.Err()
		}
		if !errorPostgreSQLReintentable(err) ||
			intento == maximoIntentosLeerAnalisisDurableO3 {
			return domain.Expediente{},
				cobertura.ErrInstantaneaAnalisisDurableNoDisponible
		}
	}
	return domain.Expediente{},
		cobertura.ErrInstantaneaAnalisisDurableNoDisponible
}

func (l *LectorExpedienteAnalisisDurableO3PostgreSQL) leerEnTransaccion(
	ctx context.Context,
	organizacionRef string,
	expedienteRef string,
	versionEsperada uint64,
) (domain.Expediente, error) {
	tx, err := l.iniciar(ctx)
	if err != nil {
		return domain.Expediente{}, err
	}
	defer revertirTransaccion(tx)

	filas, err := tx.Query(ctx, `
		SELECT expediente_json, analisis_huella_sha256
		  FROM `+funcionLeerAnalisisDurableO3+`($1, $2, $3::numeric)`,
		organizacionRef,
		expedienteRef,
		versionEsperada,
	)
	if err != nil {
		return domain.Expediente{}, err
	}
	defer filas.Close()
	if !filas.Next() {
		if err := filas.Err(); err != nil {
			return domain.Expediente{}, err
		}
		return domain.Expediente{},
			cobertura.ErrInstantaneaAnalisisDurableNoDisponible
	}
	var contenido string
	var huellaSQL string
	if err := filas.Scan(&contenido, &huellaSQL); err != nil {
		return domain.Expediente{}, err
	}
	if filas.Next() {
		return domain.Expediente{},
			cobertura.ErrInstantaneaAnalisisDurableNoConfiable
	}
	if err := filas.Err(); err != nil {
		return domain.Expediente{}, err
	}
	filas.Close()
	expediente, err := decodificarExpedienteAnalisisDurableO3(
		[]byte(contenido),
		organizacionRef,
		expedienteRef,
		versionEsperada,
		huellaSQL,
	)
	if err != nil {
		return domain.Expediente{}, err
	}
	if err := ctx.Err(); err != nil {
		return domain.Expediente{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Expediente{}, err
	}
	return expediente, nil
}

func (l *LectorExpedienteAnalisisDurableO3PostgreSQL) iniciar(
	ctx context.Context,
) (pgx.Tx, error) {
	tx, err := l.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return nil, err
	}
	_, err = tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`)
	if err != nil {
		revertirTransaccion(tx)
		return nil, err
	}
	return tx, nil
}

func decodificarExpedienteAnalisisDurableO3(
	contenido []byte,
	organizacionRef string,
	expedienteRef string,
	versionEsperada uint64,
	huellaSQL string,
) (domain.Expediente, error) {
	if len(contenido) < 2 || len(contenido) > maximoBytesExpedienteDurableO3 ||
		!huellaSHA256PostgreSQLValida(huellaSQL) ||
		validarEncuadreJSONExpedienteO3(contenido) != nil {
		return domain.Expediente{},
			cobertura.ErrInstantaneaAnalisisDurableNoConfiable
	}
	var expediente domain.Expediente
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&expediente); err != nil {
		return domain.Expediente{},
			cobertura.ErrInstantaneaAnalisisDurableNoConfiable
	}
	var sobrante any
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) ||
		expediente.Validar() != nil ||
		expediente.OrganizacionRef != organizacionRef ||
		expediente.Referencia != expedienteRef ||
		expediente.Version != versionEsperada ||
		expediente.Analisis == nil {
		return domain.Expediente{},
			cobertura.ErrInstantaneaAnalisisDurableNoConfiable
	}
	huellaGo, err := ports.HuellaAnalisisRRHHRehidratadoO3(
		*expediente.Analisis,
	)
	if err != nil ||
		subtle.ConstantTimeCompare([]byte(huellaGo), []byte(huellaSQL)) != 1 {
		return domain.Expediente{},
			cobertura.ErrInstantaneaAnalisisDurableNoConfiable
	}
	return expediente, nil
}

func huellaSHA256PostgreSQLValida(valor string) bool {
	if len(valor) != 64 || valor == "00000000000000000000000000000000"+
		"00000000000000000000000000000000" {
		return false
	}
	for _, caracter := range valor {
		if (caracter < '0' || caracter > '9') &&
			(caracter < 'a' || caracter > 'f') {
			return false
		}
	}
	return true
}
