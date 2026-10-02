package postgres

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	administracion "vec-diputacion-granada/internal/modules/administracion"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

const (
	consultaVersionGobiernoModulos = `SELECT catalogo_canonico,huella_sha256 FROM vec_catalogos_configurables.obtener_catalogo_modulos_v1($1::text,$2::integer,$3::integer)`
	consultaCabezaGobiernoModulos  = `SELECT catalogo_canonico,huella_sha256 FROM vec_catalogos_configurables.obtener_cabeza_modulos_v1($1::text,$2::integer)`
	consultaListaGobiernoModulos   = `SELECT catalogo_canonico,huella_sha256 FROM vec_catalogos_configurables.listar_catalogos_modulos_v1($1::text,$2::integer,$3::integer)`
)

var _ ports.ConsultaCatalogosConfigurables = (*RepositorioGobiernoModulosPostgreSQL)(nil)
var _ ports.ConsultaCatalogosConfigurablesAcotada = (*RepositorioGobiernoModulosPostgreSQL)(nil)
var _ ports.ConsultaCabezaCatalogoOperativo = (*RepositorioGobiernoModulosPostgreSQL)(nil)

func limitesGobiernoModulos() ports.LimitesConsultaCatalogosAcotada {
	return ports.LimitesConsultaCatalogosAcotada{Versiones: ports.MaximoVersionesConsultaCatalogosAcotada, Entradas: ports.MaximoEntradasConsultaCatalogosAcotada, Atributos: ports.MaximoAtributosConsultaCatalogosAcotada, BytesAproximados: ports.MaximoBytesConsultaCatalogosAcotada}
}

// Estas prelecturas sólo sirven a la composición y al servicio compartido. No
// emiten evidencia de acceso HTTP ni habilitan efectos; CAT6 revalida CAS/V3.
func (r *RepositorioGobiernoModulosPostgreSQL) ObtenerCatalogo(ctx context.Context, id string, version int) (domain.CatalogoConfigurable, error) {
	result, err := r.ObtenerCatalogoAcotado(ctx, id, version, limitesGobiernoModulos())
	if err != nil {
		return domain.CatalogoConfigurable{}, err
	}
	if result.Truncado {
		return domain.CatalogoConfigurable{}, ports.ErrLimitesConsultaCatalogosInvalidos
	}
	return result.Catalogo, nil
}
func (r *RepositorioGobiernoModulosPostgreSQL) ListarVersionesCatalogo(ctx context.Context, id string) ([]domain.CatalogoConfigurable, error) {
	result, err := r.ListarVersionesCatalogoAcotado(ctx, id, limitesGobiernoModulos())
	if err != nil {
		return nil, err
	}
	if result.Truncado {
		return nil, ports.ErrLimitesConsultaCatalogosInvalidos
	}
	return result.Catalogos, nil
}
func (r *RepositorioGobiernoModulosPostgreSQL) ObtenerCatalogoAcotado(ctx context.Context, id string, version int, l ports.LimitesConsultaCatalogosAcotada) (ports.ResultadoConsultaCatalogoAcotado, error) {
	if version < 1 || version > 1_000_000 {
		return ports.ResultadoConsultaCatalogoAcotado{}, domain.ErrCatalogoConfigurableInvalido
	}
	return r.leerUna(ctx, id, version, l, false)
}
func (r *RepositorioGobiernoModulosPostgreSQL) ObtenerCabezaCatalogoOperativo(ctx context.Context, id string, l ports.LimitesConsultaCatalogosAcotada) (ports.ResultadoConsultaCatalogoAcotado, error) {
	return r.leerUna(ctx, id, 0, l, true)
}
func (r *RepositorioGobiernoModulosPostgreSQL) iniciarLectura(ctx context.Context, id string, l ports.LimitesConsultaCatalogosAcotada) (pgx.Tx, ports.ConfiguracionGobiernoModulosAprobada, error) {
	var cfg ports.ConfiguracionGobiernoModulosAprobada
	if err := r.disponible(ctx); err != nil {
		return nil, cfg, err
	}
	if l.Validar() != nil {
		return nil, cfg, ports.ErrLimitesConsultaCatalogosInvalidos
	}
	cfg, err := r.configuracion(ctx)
	if err != nil {
		return nil, cfg, err
	}
	if id != cfg.CatalogoID {
		return nil, cfg, domain.ErrAutorizacionDenegada
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, cfg, errorGobiernoModulos(ctx, err)
	}
	if nuloGobiernoModulos(tx) {
		return nil, cfg, errGobiernoModulos
	}
	if _, err := tx.Exec(ctx, configurarGobiernoModulos); err != nil {
		rollbackGobiernoModulos(tx)
		return nil, cfg, errorGobiernoModulos(ctx, err)
	}
	return tx, cfg, nil
}
func (r *RepositorioGobiernoModulosPostgreSQL) leerUna(ctx context.Context, id string, version int, l ports.LimitesConsultaCatalogosAcotada, cabeza bool) (ports.ResultadoConsultaCatalogoAcotado, error) {
	var cero ports.ResultadoConsultaCatalogoAcotado
	tx, cfg, err := r.iniciarLectura(ctx, id, l)
	if err != nil {
		return cero, err
	}
	defer rollbackGobiernoModulos(tx)
	query, args := consultaVersionGobiernoModulos, []any{id, version, l.BytesAproximados}
	if cabeza {
		query, args = consultaCabezaGobiernoModulos, []any{id, l.BytesAproximados}
	}
	var canon []byte
	var huella string
	if err := tx.QueryRow(ctx, query, args...).Scan(&canon, &huella); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return cero, ports.ErrCatalogoNoEncontrado
		}
		return cero, errorGobiernoModulos(ctx, err)
	}
	defer clear(canon)
	if len(canon) > l.BytesAproximados {
		return cero, ports.ErrLimitesConsultaCatalogosInvalidos
	}
	c, err := comprobarCatalogoGobiernoModulos(canon, huella)
	if err != nil || c.ID != id || c.ModuloID != administracion.ModuleID || (!cabeza && c.Version != version) || (cabeza && c.Estado != domain.EstadoCatalogoPublicado) || validarEntradasGobiernoModulos(c, cfg) != nil {
		return cero, ports.ErrReciboCatalogoOperativoInvalido
	}
	uso, ok := ports.MedirCatalogoConfigurable(c)
	_, cabe := ports.ConsumoConsultaCatalogosAcotada{}.Agregar(uso, l)
	if !ok || !cabe {
		return cero, ports.ErrLimitesConsultaCatalogosInvalidos
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return cero, errorGobiernoModulos(ctx, err)
	}
	return ports.ResultadoConsultaCatalogoAcotado{Catalogo: c}, nil
}
func (r *RepositorioGobiernoModulosPostgreSQL) ListarVersionesCatalogoAcotado(ctx context.Context, id string, l ports.LimitesConsultaCatalogosAcotada) (ports.ResultadoConsultaCatalogosAcotada, error) {
	var cero ports.ResultadoConsultaCatalogosAcotada
	tx, cfg, err := r.iniciarLectura(ctx, id, l)
	if err != nil {
		return cero, err
	}
	defer rollbackGobiernoModulos(tx)
	rows, err := tx.Query(ctx, consultaListaGobiernoModulos, id, l.Versiones, l.BytesAproximados)
	if err != nil {
		return cero, errorGobiernoModulos(ctx, err)
	}
	defer rows.Close()
	resultado := ports.ResultadoConsultaCatalogosAcotada{Catalogos: []domain.CatalogoConfigurable{}}
	var consumo ports.ConsumoConsultaCatalogosAcotada
	anterior := 0
	for rows.Next() {
		if len(resultado.Catalogos) == l.Versiones {
			resultado.Truncado = true
			break
		}
		var canon []byte
		var huella string
		if err := rows.Scan(&canon, &huella); err != nil {
			return cero, errorGobiernoModulos(ctx, err)
		}
		if len(canon) > l.BytesAproximados-consumo.BytesAproximados {
			clear(canon)
			resultado.Truncado = true
			break
		}
		c, err := comprobarCatalogoGobiernoModulos(canon, huella)
		clear(canon)
		if err != nil || c.ID != id || c.ModuloID != administracion.ModuleID || c.Version <= anterior || validarEntradasGobiernoModulos(c, cfg) != nil {
			return cero, ports.ErrReciboCatalogoOperativoInvalido
		}
		uso, ok := ports.MedirCatalogoConfigurable(c)
		siguiente, cabe := consumo.Agregar(uso, l)
		if !ok || !cabe {
			resultado.Truncado = true
			break
		}
		consumo = siguiente
		anterior = c.Version
		resultado.Catalogos = append(resultado.Catalogos, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return cero, errorGobiernoModulos(ctx, err)
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return cero, errorGobiernoModulos(ctx, err)
	}
	return resultado, nil
}
