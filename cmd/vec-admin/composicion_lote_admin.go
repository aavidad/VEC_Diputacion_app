package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/administracion"
	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/ports"
)

// componerLoteADMIN monta la autoridad del lote sobre su pool propio. La
// cadena V3 comparte fuente, registro y motivos con las lecturas de usuarios,
// pero sólo tiene la capacidad del lote. Los fallos se auditan con el mismo
// registrador común y los motivos del overlay de usuarios.
func componerLoteADMIN(ctx context.Context, base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada,
	lote configuracionLotePrivada, poolFuente, poolRegistro, poolMotivos, poolLote *pgxpool.Pool,
	firmante ports.FirmanteAtestacionesAutorizacionV3, registrador ports.RegistradorIntentosAuditoria, reloj ports.Reloj,
) (*administracion.LoteADMIN, error) {
	meta, err := decodificarMetadatosConfianzaPerfiles(lote.ConfianzaJSON)
	if err != nil {
		return nil, errorArranque("lote_confianza_metadatos")
	}
	conf, err := confianzaDesdeMetadata(meta)
	if err != nil {
		return nil, errorArranque("lote_confianza_material")
	}
	defer func() {
		for i := range conf.EntradasCapacidad {
			clear(conf.EntradasCapacidad[i].Material)
		}
	}()
	cadena, err := administracion.NuevaConfianzaLoteV3(conf, administracion.DependenciasConfianzaPerfilesV3{
		PoolFuente: poolFuente, PoolRegistro: poolRegistro, PoolMotivos: poolMotivos, CatalogoMotivosID: base.CatalogoMotivosID,
		Firmante: firmante, Reloj: reloj, Generador: seguridad.GeneradorReferenciasCriptograficas{},
		VigenciaDecision: time.Duration(base.VigenciaDecisionSegundos) * time.Second})
	if err != nil {
		return nil, errorArranque("lote_confianza_cadena")
	}
	emisor, err := administracion.NuevoEmisorLote(cadena.Emisores[administracion.AudienciaLoteOrdinarioV3], lote.MotivoLote, reloj)
	if err != nil {
		return nil, errorArranque("lote_emisor")
	}
	proveedor, err := lote.proveedorAmbitos(u.OrganizacionRef)
	if err != nil {
		return nil, errorArranque("lote_ambitos")
	}
	autoridad, err := pg.NuevaAutoridadLoteOrdinario(ctx, poolLote, emisor, proveedor, registrador, pg.ConfiguracionAuditoriaLote{
		Proceso: u.Proceso, Canal: u.Canal, MotivoDenegado: u.MotivoDenegado, MotivoError: u.MotivoError,
		Plazo: u.plazoAuditoria()}, u.OrganizacionRef, reloj)
	if err != nil {
		return nil, errorArranque("lote_autoridad")
	}
	servicio, err := application.NuevoServicioLotesAdministracionPerfiles(autoridad, autoridad, autoridad, reloj)
	if err != nil {
		return nil, errorArranque("lote_servicio")
	}
	return &administracion.LoteADMIN{Organizacion: u.OrganizacionRef, Motivos: lote.motivosCambio(), Catalogo: autoridad, Servicio: servicio}, nil
}
