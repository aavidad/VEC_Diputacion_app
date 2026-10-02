package bootstrap

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"path/filepath"
	"strings"
	"time"
	"vec-diputacion-granada/config"
	pgvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	core "vec-diputacion-granada/internal/vec/domain"
)

var errPreparacionMaterialOH = errors.New("vec: material de Organización histórica no disponible")

// DerivarClaveOrganizacionHistoricaDesdeMaterialDesarrollo recorre la misma
// ruta de idempotencia y el descriptor que usa el publicador propio OH.
// No consulta ni publica gobierno. Versión y revisión quedan a cero: sólo
// el cotejo posterior con la fila publicada puede acreditarlas.
func DerivarClaveOrganizacionHistoricaDesdeMaterialDesarrollo(dir string, ahora time.Time) (ClaveCapacidadOrganizacionHistoricaV3, error) {
	var vacia ClaveCapacidadOrganizacionHistoricaV3
	if !filepath.IsAbs(dir) || filepath.Clean(dir) != dir || filepath.Base(dir) != "idempotencia" || dentroDeRepositorioGit(dir) {
		return vacia, errPreparacionMaterialOH
	}
	if p, err := filepath.EvalSymlinks(dir); err != nil || p != dir {
		return vacia, errPreparacionMaterialOH
	}
	base := filepath.Dir(dir)
	i, err := cargarMaterialIdempotenciaDesarrollo(base, filepath.Join(base, config.DevelopmentIdempotencyHMACConfigRelativePath))
	if err != nil {
		return vacia, errPreparacionMaterialOH
	}
	defer i.borrar()
	d, err := nuevoDerivadorIdentidadOperacionDesarrollo(&i)
	if err != nil {
		return vacia, errPreparacionMaterialOH
	}
	defer d.borrar()
	m, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(d, ahora)
	if err != nil {
		return vacia, errPreparacionMaterialOH
	}
	defer m.borrarCopiasEfimeras()
	x, err := derivarMaterialConsumidorV3Desarrollo(m, descriptorMaterialOrganizacionHistorica())
	if err != nil {
		return vacia, errPreparacionMaterialOH
	}
	return ClaveCapacidadOrganizacionHistoricaV3{CapacidadPublicadaOrganizacionHistoricaV3: CapacidadPublicadaOrganizacionHistoricaV3{
		DescriptorCapacidadOrganizacionHistoricaV3: DescriptorCapacidadOrganizacionHistoricaV3Desarrollo(),
		ClaveID: x.claveHMACID, HuellaGobierno: x.claveHMACHuella, SHA256: x.claveHMACSecreto, EmisorID: x.emisorID, Desde: x.validaDesde, Hasta: x.validaHasta,
	}, secreto: x.claveHMAC}, nil
}

// ValidarMotivoPreparacionOrganizacionHistorica reutiliza el pool nominal y
// el lector histórico existentes. No resuelve contextos F1 ni registra uso.
func ValidarMotivoPreparacionOrganizacionHistorica(ctx context.Context, dsn, login string, motivo core.ReferenciaEntradaCatalogo, ahora time.Time) error {
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c.ConnConfig.User != login {
		return errPreparacionMaterialOH
	}
	p, _, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(ctx, strings.TrimSpace(dsn), "vec-preparar-organizacion-historica-motivo", "vec_autorizacion_motivos_evaluador")
	if err != nil {
		return errPreparacionMaterialOH
	}
	defer p.Close()
	v, err := pgvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(p, motivo.CatalogoID)
	if err != nil {
		return errPreparacionMaterialOH
	}
	return v.ValidarReferenciaMotivoAutorizacionV2(ctx, motivo, ahora.UTC().Truncate(time.Microsecond))
}
