package postgres

import (
	"context"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const consultaMotivoDecisionCoberturaPostgreSQL = `
WITH resultado AS MATERIALIZED (
	SELECT catalogo_version, catalogo_huella_sha256, modulo_id,
	       entrada_clave, clave_i18n
	  FROM vec_autorizacion.resolver_motivo_decision_cobertura_v1(
	       $1, $2, $3::timestamptz)
	 LIMIT 2
)
SELECT pg_catalog.count(*)::bigint,
	COALESCE(pg_catalog.max(catalogo_version), 0)::bigint,
	COALESCE(pg_catalog.max(catalogo_huella_sha256), '')::text,
	COALESCE(pg_catalog.max(modulo_id), '')::text,
	COALESCE(pg_catalog.max(entrada_clave), '')::text,
	COALESCE(pg_catalog.max(clave_i18n), '')::text
FROM resultado`

type consultadorMotivoDecisionCoberturaPostgreSQL interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// ConsultaMotivoDecisionCoberturaPostgreSQL solo obtiene la entrada vigente
// exacta mediante una función cerrada. No expone tablas ni reconstruye el
// catálogo configurable completo.
type ConsultaMotivoDecisionCoberturaPostgreSQL struct {
	consultador consultadorMotivoDecisionCoberturaPostgreSQL
	catalogoID  string
	moduloID    string
}

var _ cobertura.ConsultaMotivoDecisionCoberturaAcotada = (*ConsultaMotivoDecisionCoberturaPostgreSQL)(nil)

func NuevaConsultaMotivoDecisionCoberturaPostgreSQL(
	pool *pgxpool.Pool,
	catalogoID string,
	moduloID string,
) (*ConsultaMotivoDecisionCoberturaPostgreSQL, error) {
	return nuevaConsultaMotivoDecisionCoberturaPostgreSQL(
		pool,
		catalogoID,
		moduloID,
	)
}

func nuevaConsultaMotivoDecisionCoberturaPostgreSQL(
	consultador consultadorMotivoDecisionCoberturaPostgreSQL,
	catalogoID string,
	moduloID string,
) (*ConsultaMotivoDecisionCoberturaPostgreSQL, error) {
	if dependenciaNula(consultador) ||
		!identidadConsultaMotivoDecisionCoberturaValida(catalogoID, moduloID) {
		return nil, cobertura.ErrConfiguracionResolutorMotivoDecisionCobertura
	}
	return &ConsultaMotivoDecisionCoberturaPostgreSQL{
		consultador: consultador,
		catalogoID:  catalogoID,
		moduloID:    moduloID,
	}, nil
}

func (c *ConsultaMotivoDecisionCoberturaPostgreSQL) ConsultarMotivoDecisionCobertura(
	ctx context.Context,
	catalogoID string,
	moduloID string,
	clave domain.ClaveCatalogo,
	instante time.Time,
) (domain.MotivoGobernadoDecisionCobertura, error) {
	vacio := domain.MotivoGobernadoDecisionCobertura{}
	if dependenciaNula(ctx) || c == nil || dependenciaNula(c.consultador) ||
		catalogoID != c.catalogoID || moduloID != c.moduloID ||
		!clave.Valida() || !instanteResolucionMotivosRRHHValido(instante) {
		return vacio, cobertura.ErrMotivoDecisionCoberturaNoConfiable
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	var cardinalidad, version int64
	var huella, moduloResuelto, entrada, claveI18n string
	err := c.consultador.QueryRow(
		ctx,
		consultaMotivoDecisionCoberturaPostgreSQL,
		catalogoID,
		string(clave),
		instante,
	).Scan(
		&cardinalidad,
		&version,
		&huella,
		&moduloResuelto,
		&entrada,
		&claveI18n,
	)
	if errContexto := ctx.Err(); errContexto != nil {
		return vacio, errContexto
	}
	if err != nil || cardinalidad != 1 || version < 1 ||
		version > math.MaxInt32 || moduloResuelto != moduloID ||
		entrada != string(clave) {
		return vacio, cobertura.ErrMotivoDecisionCoberturaNoConfiable
	}
	motivo := domain.MotivoGobernadoDecisionCobertura{
		ReferenciaCatalogo: dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID:           catalogoID,
			CatalogoVersion:      int(version),
			CatalogoHuellaSHA256: huella,
			EntradaClave:         entrada,
		},
		ClaveI18n: domain.ClaveCatalogo(claveI18n),
	}
	if motivo.ReferenciaCatalogo.Validar() != nil ||
		!motivo.ClaveI18n.Valida() {
		return vacio, cobertura.ErrMotivoDecisionCoberturaNoConfiable
	}
	return motivo, nil
}

func identidadConsultaMotivoDecisionCoberturaValida(
	catalogoID string,
	moduloID string,
) bool {
	referencia := dominiovec.ReferenciaEntradaCatalogo{
		CatalogoID:           catalogoID,
		CatalogoVersion:      1,
		CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		EntradaClave:         "motivo",
	}
	return referencia.Validar() == nil &&
		domain.ClaveCatalogo(moduloID).Valida()
}
