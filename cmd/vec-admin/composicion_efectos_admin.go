package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/administracion"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres/cargoadmin"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/adapters/postgres/certificadonominal"
	"vec-diputacion-granada/internal/vec/adapters/postgres/efectonominaladmin"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	"vec-diputacion-granada/internal/vec/ports"
)

// efectoADMIN describe un efecto nominal que vec-admin puede abrir con su
// archivo privado: audiencia, grupo exclusivo del LOGIN que exige la fachada,
// si la sesión debe ir en UTC, ruta, límite del material y contrato.
type efectoADMIN struct {
	nombre    string
	variable  string
	audiencia string
	grupo     string
	utc       bool
	ruta      string
	maximo    int
	contrato  func() efectonominaladmin.Contrato
}

// efectosADMIN devuelve la lista cerrada, nueva en cada llamada, en el orden
// en que se abren sus pools.
func efectosADMIN() []efectoADMIN {
	return []efectoADMIN{
		{nombre: "cargos", variable: "VEC_ADMIN_CARGOS_CONFIG_FILE", audiencia: administracion.AudienciaCargoCompetencialV3,
			grupo: "vec_personal_ejecutor", utc: true, ruta: api.RutaPublicacionCargoCompetencial,
			maximo: personalpg.MaximoMaterialPublicacionCargo, contrato: personalpg.ContratoPublicacionCargoCompetencial},
		{nombre: "certificados", variable: "VEC_ADMIN_CERTIFICADOS_CONFIG_FILE", audiencia: administracion.AudienciaCertificadoNominalV3,
			grupo: "vec_autorizacion_certificado_nominal_ejecutor", ruta: api.RutaPublicacionCertificadoNominal,
			maximo: certificadonominal.MaximoDescriptor, contrato: certificadonominal.Contrato},
	}
}

// efectoConfigurado es un efecto de la lista con su archivo privado cargado.
type efectoConfigurado struct {
	efectoADMIN
	cfg configuracionEfectoPrivada
}

// componerEfectoADMIN monta un efecto sobre su pool propio y una cadena V3 con
// sólo su capacidad. Los fallos se auditan con el registrador común y los
// motivos del overlay de usuarios.
func componerEfectoADMIN(ctx context.Context, base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada,
	e efectoConfigurado, poolFuente, poolRegistro, poolMotivos, poolEfecto *pgxpool.Pool,
	firmante ports.FirmanteAtestacionesAutorizacionV3, registrador ports.RegistradorIntentosAuditoria, reloj ports.Reloj,
) (administracion.EfectoNominalMontado, error) {
	var vacio administracion.EfectoNominalMontado
	if acreditarPoolCentral(ctx, poolEfecto, e.grupo) != nil {
		return vacio, errorArranque(e.nombre + "_pool")
	}
	if e.utc && acreditarZonaHorariaUTC(ctx, poolEfecto) != nil {
		return vacio, errorArranque(e.nombre + "_zona_horaria")
	}
	meta, err := decodificarMetadatosConfianzaPerfiles(e.cfg.ConfianzaJSON)
	if err != nil {
		return vacio, errorArranque(e.nombre + "_confianza_metadatos")
	}
	conf, err := confianzaDesdeMetadata(meta)
	if err != nil {
		return vacio, errorArranque(e.nombre + "_confianza_material")
	}
	defer func() {
		for i := range conf.EntradasCapacidad {
			clear(conf.EntradasCapacidad[i].Material)
		}
	}()
	cadena, err := administracion.NuevaConfianzaEfectoNominalV3(conf, administracion.DependenciasConfianzaPerfilesV3{
		PoolFuente: poolFuente, PoolRegistro: poolRegistro, PoolMotivos: poolMotivos, CatalogoMotivosID: base.CatalogoMotivosID,
		Firmante: firmante, Reloj: reloj, Generador: seguridad.GeneradorReferenciasCriptograficas{},
		VigenciaDecision: time.Duration(base.VigenciaDecisionSegundos) * time.Second}, e.audiencia)
	if err != nil {
		return vacio, errorArranque(e.nombre + "_confianza_cadena")
	}
	contrato := e.contrato()
	emisor, err := administracion.NuevoEmisorEfectoNominalADMIN(cadena.Emisores[e.audiencia], e.cfg.Motivo, reloj, contrato)
	if err != nil {
		return vacio, errorArranque(e.nombre + "_emisor")
	}
	ejecutor, err := efectonominaladmin.NuevoEjecutor(poolEfecto, emisor, registrador, efectonominaladmin.ConfiguracionAuditoria{
		Proceso: u.Proceso, MotivoDenegado: u.MotivoDenegado, MotivoError: u.MotivoError, Plazo: u.plazoAuditoria()}, reloj, contrato)
	if err != nil {
		return vacio, errorArranque(e.nombre + "_ejecutor")
	}
	servicio, err := administracion.NuevoServicioEfectoNominal(ejecutor)
	if err != nil {
		return vacio, errorArranque(e.nombre + "_servicio")
	}
	return administracion.EfectoNominalMontado{Ruta: e.ruta, Maximo: e.maximo, Servicio: servicio}, nil
}
