package plantillascatalogo

import (
	"context"
	"encoding/json"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/informejuridico"
	app "vec-diputacion-granada/internal/modules/contrataciontemporal/application/plantillascatalogo"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const consultaPublicadaDocumental = `SELECT vec_contratacion_temporal.obtener_catalogo_plantillas_publicado_documental_v1()`

var huellaDocumental = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ProveedorDocumental sirve únicamente la última publicación a los
// renderizadores de un expediente ya leído con autorización V3. No ofrece
// borradores ni la API de gobierno funcional de plantillas.
type ProveedorDocumental struct {
	pool    *pgxpool.Pool
	inicial vecdomain.CatalogoConfigurable
}

func NuevoProveedorDocumental(pool *pgxpool.Pool, inicial *vecdomain.CatalogoConfigurable) (*ProveedorDocumental, error) {
	if pool == nil || inicial == nil {
		return nil, app.ErrNoDisponible
	}
	c, err := inicial.ClonarCanonico()
	if err != nil || c.ID != app.CatalogoID || c.ModuloID != app.ModuloID || c.Estado != vecdomain.EstadoCatalogoPublicado {
		return nil, app.ErrNoDisponible
	}
	return &ProveedorDocumental{pool: pool, inicial: c}, nil
}

func (p *ProveedorDocumental) ObtenerPlantillas(ctx context.Context, instante time.Time) (*informejuridico.PlantillasBorrador, error) {
	if p == nil || p.pool == nil || ctx == nil || instante.IsZero() {
		return nil, app.ErrNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, normalizar(ctx, err)
	}
	defer tx.Rollback(ctx)
	var b []byte
	if err = tx.QueryRow(ctx, consultaPublicadaDocumental).Scan(&b); err != nil {
		return nil, normalizar(ctx, err)
	}
	if len(b) == 0 || len(b) > maximoRespuesta {
		return nil, app.ErrNoDisponible
	}
	var v struct {
		Catalogo             vecdomain.CatalogoConfigurable `json:"catalogo"`
		CatalogoHuellaSHA256 string                         `json:"catalogo_huella_sha256"`
		ContenidoJSONSHA256  string                         `json:"contenido_json_sha256"`
		Version              int                            `json:"version"`
		Revision             int                            `json:"revision"`
	}
	if json.Unmarshal(b, &v) != nil || v.Catalogo.ID != app.CatalogoID || v.Catalogo.ModuloID != app.ModuloID || v.Catalogo.Estado != vecdomain.EstadoCatalogoPublicado || v.Version != v.Catalogo.Version || v.Revision != v.Catalogo.Revision || !huellaDocumental.MatchString(v.CatalogoHuellaSHA256) || !huellaDocumental.MatchString(v.ContenidoJSONSHA256) {
		return nil, app.ErrNoDisponible
	}
	h, err := v.Catalogo.HuellaSHA256()
	if err != nil || h != v.CatalogoHuellaSHA256 {
		return nil, app.ErrNoDisponible
	}
	if v.Version == p.inicial.Version {
		esperada, _ := p.inicial.HuellaSHA256()
		if h != esperada {
			return nil, app.ErrNoDisponible
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, normalizar(ctx, err)
	}
	plantillas, err := informejuridico.NuevasPlantillasBorrador(v.Catalogo, instante)
	if err != nil {
		return nil, app.ErrNoDisponible
	}
	return plantillas, nil
}

var _ interface {
	ObtenerPlantillas(context.Context, time.Time) (*informejuridico.PlantillasBorrador, error)
} = (*ProveedorDocumental)(nil)
