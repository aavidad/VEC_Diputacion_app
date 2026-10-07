package postgres

import (
	"context"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	vd "vec-diputacion-granada/internal/vec/domain"
)

const comprobarPlanFirmaPublicadoSQL178 = `SELECT vec_contratacion_temporal.comprobar_plan_firma_publicado_v1($1,$2::bigint,$3)::text`

var (
	shaPublicacionPlanFirma = regexp.MustCompile(`^[0-9a-f]{64}$`)
	idPublicacionPlanFirma  = regexp.MustCompile(`^[a-z][a-z0-9._-]{2,127}$`)
)

type consultorFilaPublicacionPlanFirma interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// PublicacionPlanFirmaPostgreSQL comprueba antes del PDP, con el LOGIN CT y la
// fachada CT178, que el catálogo del plan que usa el descriptor es la
// publicación vigente de Catálogos (CC8): mismo id, versión, SHA, revisión y
// fecha de publicación. No sustituye el pin de CT176 en la transacción final.
type PublicacionPlanFirmaPostgreSQL struct {
	pool consultorFilaPublicacionPlanFirma
}

func NuevaPublicacionPlanFirmaPostgreSQL(pool consultorFilaPublicacionPlanFirma) (*PublicacionPlanFirmaPostgreSQL, error) {
	if nuloRegistroTX(pool) {
		return nil, ct.ErrPlanCompetenciaFirmaV2
	}
	return &PublicacionPlanFirmaPostgreSQL{pool: pool}, nil
}

// ComprobarPublicacionPlanFirmaV2 cumple plannominal.PublicacionAutorizada.
// Cualquier rechazo, caída o respuesta incoherente es ErrPlanCompetenciaFirmaV2.
func (p *PublicacionPlanFirmaPostgreSQL) ComprobarPublicacionPlanFirmaV2(ctx context.Context, c vd.CatalogoConfigurable, sha string, en time.Time) error {
	if p == nil || nuloRegistroTX(p.pool) || ctx == nil {
		return ct.ErrPlanCompetenciaFirmaV2
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !idPublicacionPlanFirma.MatchString(c.ID) || c.Version < 1 || c.Revision < 1 || c.ModuloID != "contratacion_temporal" || !shaPublicacionPlanFirma.MatchString(sha) ||
		c.PublicadoEn.IsZero() || en.IsZero() || c.PublicadoEn.After(en) ||
		// PostgreSQL redondea al microsegundo y Go truncaría: una fecha con más
		// precisión nunca coincidiría y se rechaza aquí de forma explícita.
		c.PublicadoEn.Nanosecond()%1000 != 0 {
		return ct.ErrPlanCompetenciaFirmaV2
	}
	var bruto []byte
	if err := p.pool.QueryRow(ctx, comprobarPlanFirmaPublicadoSQL178, c.ID, int64(c.Version), sha).Scan(&bruto); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ct.ErrPlanCompetenciaFirmaV2
	}
	var r struct {
		PublicadaEn string `json:"publicada_en"`
		Revision    int64  `json:"revision"`
	}
	if decodificarFirma118(bruto, &r) != nil {
		return ct.ErrPlanCompetenciaFirmaV2
	}
	publicada, err := time.Parse("2006-01-02T15:04:05.000000Z", r.PublicadaEn)
	if err != nil || !publicada.Equal(c.PublicadoEn.UTC().Truncate(time.Microsecond)) || r.Revision != int64(c.Revision) {
		return ct.ErrPlanCompetenciaFirmaV2
	}
	return nil
}
