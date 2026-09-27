package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type ConsultaPoliticaCesePostgreSQL struct{ pool *pgxpool.Pool }

var _ ports.ConsultaPoliticaCese = (*ConsultaPoliticaCesePostgreSQL)(nil)

func NuevaConsultaPoliticaCesePostgreSQL(pool *pgxpool.Pool) (*ConsultaPoliticaCesePostgreSQL, error) {
	if pool == nil {
		return nil, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	return &ConsultaPoliticaCesePostgreSQL{pool: pool}, nil
}

// La función B45 valida la huella del JSONB y el LOGIN ejecutor. Este
// adaptador vuelve a cerrar tipos y catálogo antes de entregarlos al caso de
// uso; nunca lee la tabla ni reutiliza el catálogo B14.
func (c *ConsultaPoliticaCesePostgreSQL) ConsultarPoliticaCese(ctx context.Context, material vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (domain.PoliticaCese, error) {
	if c == nil || c.pool == nil || ctx == nil || ctx.Err() != nil || material.ValidarEstructura() != nil ||
		material.ResumenCapacidad().Operacion() != ports.AccionConsultarPoliticaCese ||
		material.ResumenCapacidad().AudienciaConsumo() != ports.AudienciaConsultarPoliticaCese ||
		material.ResumenCapacidad().EfectoRef() != ports.RecursoPoliticaCeseRef {
		return domain.PoliticaCese{}, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	tx, err := c.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return domain.PoliticaCese{}, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),set_config('row_security','on',true),set_config('timezone','UTC',true),set_config('lock_timeout','2s',true),set_config('statement_timeout','15s',true)`); err != nil {
		return domain.PoliticaCese{}, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	var decisionRef, efectoRef, huellaEfecto, huellaConsumo, auditoriaRef string
	var consumidaEn time.Time
	var nueva bool
	err = tx.QueryRow(ctx, `SELECT decision_ref,efecto_ref,huella_efecto_sha256,consumo_huella_sha256,auditoria_ref,consumida_en,consumo_nuevo FROM vec_autorizacion_atestada_v3.registrar_y_consumir_consulta_politica_cese_bolsa_v3_atestada($1,$2,$3,$4,$5::numeric,$6::numeric,$7,$8,$9,$10)`,
		material.CapacidadCanonica(), material.DecisionCanonica(), material.MotivoCanonico(), material.ContextoActorCanonico(),
		material.PersonaVersion(), material.PerfilVersion(), material.PayloadVECAD3(), material.SobreCOSESign1(),
		material.EvidenciaVerificacion(), material.RaizPublicaSPKI()).Scan(&decisionRef, &efectoRef, &huellaEfecto, &huellaConsumo, &auditoriaRef, &consumidaEn, &nueva)
	if err != nil || !nueva || decisionRef == "" || efectoRef != ports.RecursoPoliticaCeseRef || huellaEfecto == "" || huellaConsumo == "" || auditoriaRef == "" || consumidaEn.IsZero() {
		return domain.PoliticaCese{}, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	politica, err := escanearPoliticaCese(tx.QueryRow(ctx, `SELECT version,catalogo_ref,catalogo_sha256,mapeo,meses_general,meses_acumulacion,computo,estado,publicada_en FROM vec_bolsa_llamamientos.consultar_politica_cese_bolsa_v1()`))
	if err != nil || ctx.Err() != nil || tx.Commit(ctx) != nil {
		return domain.PoliticaCese{}, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	return politica, nil
}

func escanearPoliticaCese(fila pgx.Row) (domain.PoliticaCese, error) {
	var version int64
	var ref, sha, computo, estado string
	var mapeo []byte
	var general, acumulacion int32
	var publicada time.Time
	if err := fila.Scan(&version, &ref, &sha, &mapeo, &general, &acumulacion, &computo, &estado, &publicada); err != nil {
		return domain.PoliticaCese{}, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	var clases map[string]string
	if version < 1 || len(mapeo) == 0 || len(mapeo) > 8192 || json.Unmarshal(mapeo, &clases) != nil || clases == nil {
		return domain.PoliticaCese{}, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	p := domain.PoliticaCese{Version: uint64(version), CatalogoRef: ref, CatalogoSHA256: sha,
		Mapeo: clases, MesesGeneral: int(general), MesesAcumulacion: int(acumulacion),
		Computo: computo, Estado: estado, PublicadaEn: publicada.UTC()}
	if p.Validar() != nil {
		return domain.PoliticaCese{}, ports.ErrConsultaPoliticaCeseNoDisponible
	}
	return p, nil
}
