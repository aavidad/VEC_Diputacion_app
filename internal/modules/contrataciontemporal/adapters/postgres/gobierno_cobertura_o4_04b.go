package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	funcionResolverGobiernoCoberturaO404B = "vec_contratacion_temporal.gobi_o404b_resolver"
	maximoIntentosGobiernoCoberturaO404B  = 3
	maximoBytesCatalogoGobiernoO404B      = 1_048_576
	maximoBytesPoliticaGobiernoO404B      = 1_048_576
	maximoBytesActuacionGobiernoO404B     = 65_536
)

var _ cobertura.ResolutorGobiernoOperacionCobertura = (*ResolutorGobiernoCoberturaO404BPostgreSQL)(nil)

// ResolutorGobiernoCoberturaO404BPostgreSQL solo puede leer la proyección
// autoritativa mediante su función SECURITY DEFINER. No publica gobierno ni
// recibe acceso directo a las tablas.
type ResolutorGobiernoCoberturaO404BPostgreSQL struct {
	pool iniciadorTransacciones
}

func NuevoResolutorGobiernoCoberturaO404BPostgreSQL(
	pool *pgxpool.Pool,
) (*ResolutorGobiernoCoberturaO404BPostgreSQL, error) {
	return nuevoResolutorGobiernoCoberturaO404BPostgreSQL(pool)
}

func nuevoResolutorGobiernoCoberturaO404BPostgreSQL(
	pool iniciadorTransacciones,
) (*ResolutorGobiernoCoberturaO404BPostgreSQL, error) {
	if dependenciaNula(pool) {
		return nil, cobertura.ErrGobiernoOperacionCoberturaNoDisponible
	}
	return &ResolutorGobiernoCoberturaO404BPostgreSQL{pool: pool}, nil
}

func (r *ResolutorGobiernoCoberturaO404BPostgreSQL) ResolverGobiernoOperacionCobertura(
	ctx context.Context,
	solicitud cobertura.SolicitudResolucionGobiernoOperacionCobertura,
) (cobertura.PublicacionGobiernoOperacionCobertura, error) {
	if ctx == nil || r == nil || dependenciaNula(r.pool) {
		return cobertura.PublicacionGobiernoOperacionCobertura{},
			cobertura.ErrGobiernoOperacionCoberturaNoDisponible
	}
	organizacion, expediente, version, accion, instante, err :=
		solicitud.Coordenadas()
	if err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{},
			cobertura.ErrGobiernoOperacionCoberturaNoConfiable
	}
	for intento := 1; intento <= maximoIntentosGobiernoCoberturaO404B; intento++ {
		publicacion, err := r.resolverEnTransaccion(
			ctx,
			organizacion,
			expediente,
			version,
			string(accion),
			instante,
		)
		if err == nil {
			return publicacion, nil
		}
		if errors.Is(
			err,
			cobertura.ErrGobiernoOperacionCoberturaNoConfiable,
		) {
			return cobertura.PublicacionGobiernoOperacionCobertura{},
				cobertura.ErrGobiernoOperacionCoberturaNoConfiable
		}
		if errContexto := ctx.Err(); errContexto != nil {
			return cobertura.PublicacionGobiernoOperacionCobertura{},
				errContexto
		}
		if !errorPostgreSQLReintentable(err) ||
			intento == maximoIntentosGobiernoCoberturaO404B {
			return cobertura.PublicacionGobiernoOperacionCobertura{},
				cobertura.ErrGobiernoOperacionCoberturaNoDisponible
		}
	}
	return cobertura.PublicacionGobiernoOperacionCobertura{},
		cobertura.ErrGobiernoOperacionCoberturaNoDisponible
}

func (r *ResolutorGobiernoCoberturaO404BPostgreSQL) resolverEnTransaccion(
	ctx context.Context,
	organizacion string,
	expediente string,
	version uint64,
	accion string,
	instante time.Time,
) (cobertura.PublicacionGobiernoOperacionCobertura, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.Serializable,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{}, err
	}
	defer revertirTransaccion(tx)
	if _, err := tx.Exec(ctx, `
		SELECT set_config('search_path', 'pg_catalog', true),
		       set_config('row_security', 'on', true),
		       set_config('timezone', 'UTC', true),
		       set_config('lock_timeout', '2s', true),
		       set_config('statement_timeout', '15s', true),
		       set_config('idle_in_transaction_session_timeout', '20s', true)`); err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{}, err
	}
	filas, err := tx.Query(ctx, `
		SELECT catalogo_json, politica_json, actuacion_json
		  FROM `+funcionResolverGobiernoCoberturaO404B+`(
		       $1, $2, $3::numeric, $4, $5)`,
		organizacion,
		expediente,
		version,
		accion,
		instante,
	)
	if err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{}, err
	}
	defer filas.Close()
	if !filas.Next() {
		if err := filas.Err(); err != nil {
			return cobertura.PublicacionGobiernoOperacionCobertura{}, err
		}
		return cobertura.PublicacionGobiernoOperacionCobertura{},
			cobertura.ErrGobiernoOperacionCoberturaNoDisponible
	}
	var catalogoJSON, politicaJSON, actuacionJSON string
	if err := filas.Scan(
		&catalogoJSON,
		&politicaJSON,
		&actuacionJSON,
	); err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{}, err
	}
	if filas.Next() {
		return cobertura.PublicacionGobiernoOperacionCobertura{},
			cobertura.ErrGobiernoOperacionCoberturaNoConfiable
	}
	if err := filas.Err(); err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{}, err
	}
	filas.Close()
	publicacion, err := decodificarGobiernoCoberturaO404B(
		catalogoJSON,
		politicaJSON,
		actuacionJSON,
		organizacion,
		expediente,
		version,
		accion,
	)
	if err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{}, err
	}
	if err := ctx.Err(); err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return cobertura.PublicacionGobiernoOperacionCobertura{}, err
	}
	return publicacion, nil
}

func decodificarGobiernoCoberturaO404B(
	catalogoJSON string,
	politicaJSON string,
	actuacionJSON string,
	organizacion string,
	expediente string,
	version uint64,
	accion string,
) (cobertura.PublicacionGobiernoOperacionCobertura, error) {
	catalogoPublicado, err := decodificarJSONGobiernoO404B[domain.PublicacionCatalogoViasCobertura](catalogoJSON, maximoBytesCatalogoGobiernoO404B)
	if err != nil {
		return gobiernoCoberturaO404BNoConfiable()
	}
	catalogo, err := domain.RestaurarCatalogoViasCobertura(catalogoPublicado)
	if err != nil {
		return gobiernoCoberturaO404BNoConfiable()
	}
	politicaPublicada, err := decodificarJSONGobiernoO404B[domain.PublicacionPoliticaDecisionCobertura](politicaJSON, maximoBytesPoliticaGobiernoO404B)
	if err != nil {
		return gobiernoCoberturaO404BNoConfiable()
	}
	politica, err := domain.RestaurarPoliticaDecisionCobertura(
		politicaPublicada,
		catalogo,
	)
	if err != nil {
		return gobiernoCoberturaO404BNoConfiable()
	}
	actuacion, err := decodificarJSONGobiernoO404B[cobertura.PublicacionPoliticaActuacionCobertura](actuacionJSON, maximoBytesActuacionGobiernoO404B)
	finalidadClave, finalidadRef := politica.Finalidad()
	if err != nil ||
		actuacion.Validar() != nil ||
		actuacion.OrganizacionRef != organizacion ||
		string(actuacion.Accion) != accion ||
		!actuacion.Catalogo.CoincideExactamente(catalogo.Identidad()) ||
		actuacion.Politica != politica.Identidad() ||
		actuacion.FinalidadContratacionClave != finalidadClave ||
		actuacion.FinalidadContratacionRef != finalidadRef {
		return gobiernoCoberturaO404BNoConfiable()
	}
	return cobertura.PublicacionGobiernoOperacionCobertura{
		OrganizacionRef:   organizacion,
		ExpedienteRef:     expediente,
		VersionExpediente: version,
		Catalogo:          catalogo,
		Politica:          politica,
		PoliticaActuacion: actuacion,
	}, nil
}

func decodificarJSONGobiernoO404B[T any](
	contenido string,
	maximo int,
) (T, error) {
	var cero T
	if len(contenido) < 2 || len(contenido) > maximo {
		return cero, cobertura.ErrGobiernoOperacionCoberturaNoConfiable
	}
	decodificador := json.NewDecoder(bytes.NewBufferString(contenido))
	decodificador.DisallowUnknownFields()
	var valor T
	if err := decodificador.Decode(&valor); err != nil {
		return cero, err
	}
	var sobrante any
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return cero, cobertura.ErrGobiernoOperacionCoberturaNoConfiable
	}
	return valor, nil
}

func gobiernoCoberturaO404BNoConfiable() (
	cobertura.PublicacionGobiernoOperacionCobertura,
	error,
) {
	return cobertura.PublicacionGobiernoOperacionCobertura{},
		cobertura.ErrGobiernoOperacionCoberturaNoConfiable
}
