package bootstrap

import (
	"context"
	"crypto/ed25519"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	confianzaatestacion "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/auditoria"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

const (
	rolEjecucionPostgreSQLContratacionTemporalDesarrollo            = "vec_contratacion_temporal_ejecutor"
	rolGobiernoPostgreSQLContratacionTemporalDesarrollo             = "vec_autorizacion_atestada_v3_migrador"
	rolRegistroAutorizacionPostgreSQLContratacionTemporalDesarrollo = "vec_autorizacion_registro"
	rolConfirmadorPostgreSQLContratacionTemporalDesarrollo          = "vec_contratacion_temporal_confirmador_cobertura"
	rolLectorPostgreSQLContratacionTemporalDesarrollo               = "vec_contratacion_temporal_lector_resultado_cobertura"
	rolAuditoriaFronteraPostgreSQLContratacionTemporalDesarrollo    = "vec_contratacion_temporal_registrador_frontera"
	audienciaAtestacionContratacionTemporalDesarrollo               = "vec:desarrollo:contratacion-temporal:atestacion:v3"
	audienciaConsumoAltaContratacionTemporal                        = "vec_contratacion_temporal.confirmar_alta_atestada.v1"
)

var (
	errPostgreSQLContratacionTemporalDesarrolloNoDisponible = errors.New(
		"bootstrap: PostgreSQL de contratacion temporal no disponible",
	)
	errGobiernoPostgreSQLContratacionTemporalDesarrolloAjeno = errors.New(
		"gobierno PostgreSQL de desarrollo ajeno",
	)
	errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente = errors.New(
		"gobierno PostgreSQL de desarrollo incoherente",
	)
	errGobiernoPostgreSQLContratacionTemporalDesarrolloAgotado = errors.New(
		"versiones de gobierno PostgreSQL de desarrollo agotadas",
	)
)

type materialAtestacionContratacionTemporalDesarrollo struct {
	fuenteConfianza     *fuenteConfianzaRenovableCTDesarrollo
	claveID             string
	claveVersion        uint64
	privada             ed25519.PrivateKey
	raiz                confianzaatestacion.RaizPublicaAtestacionAutorizacionV3
	configuracion       confianzaatestacion.ConfiguracionConfianzaAtestacionAutorizacionV3
	configuracionRef    string
	configuracionOrden  uint64
	configuracionHuella string
	publicadaEn         time.Time
	expiraEn            time.Time
	validaDesde         time.Time
	validaHasta         time.Time
	spki                []byte
	spkiHuella          string
	claveHMACID         string
	claveHMACVersion    uint64
	claveHMACOrden      uint64
	claveHMAC           []byte
	claveHMACRevision   uint64
	claveHMACHuella     string
	claveHMACSecreto    string
	emisorID            string
	audienciaConsumo    string
	capacidad           confianzaatestacion.ClaveHMACCapacidadAtestacionV3
}

type dependenciasPostgreSQLContratacionTemporalDesarrollo struct {
	auditoriaLecturasBolsa            puertosvec.RegistradorIntentosAuditoria
	procesoAuditoriaLecturasBolsa     string
	cerrarAuditoriaLecturasBolsa      func()
	cerrarFirmasR5V2                  func()
	ejecucion                         *pgxpool.Pool
	bolsa                             *pgxpool.Pool
	calculadorPoliticaOfertas         *pgxpool.Pool
	gobierno                          *pgxpool.Pool
	registroAutorizacion              *pgxpool.Pool
	confirmador                       *pgxpool.Pool
	lectorResultado                   *postgrescontratacion.PoolRecuperacionCoberturaO405PostgreSQL
	registradorAuditoriaFrontera      *postgresvec.RegistradorAuditoriaFronteraRutaExactaPostgreSQL
	auditoriaFrontera                 *pgxpool.Pool
	candidaturas                      ports.ResolutorCandidaturaAlta
	transaccionAlta                   ports.TransaccionAltasCandidata
	proveedorMaterial                 *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialBolsa            *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialMiBolsa          *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialHistorialMiBolsa *proveedorMaterialAltaContratacionTemporalDesarrollo
	// proveedoresMaterialPortal: uno por acción propia del candidato que
	// tiene consumidor compuesto (AD3-84 con Bolsa 000030).
	proveedoresMaterialPortal                        map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialBorradorCrear                   *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialBorradorConsulta                *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialSituacion                       *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialConsultaSolicitudesDocumentales *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialConsultaReincorporacionTitular  *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialContacto                        *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialConsultaContacto                *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialDatosContacto                   *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialConsultaDatosContacto           *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialEmision                         *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialPoliticaOfertas                 *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialConsultaPoliticaOfertas         *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialAuditoriaCT                     *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialAuditoriaBolsa                  *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialPlantillasCatalogo              *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialAjustesReglasCT                 *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialPlantillasDocumental            *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialDespachoCorreo                  *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialResultadoCorreo                 *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialFirmaDocumento                  *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialConsultaFirmasDocumento         *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialDietas                                   materialDietasDesdeCTDesarrollo
	materialCronos                                   materialCronosDesdeCTDesarrollo
	materialDocumentos                               *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialPersonalFichaPropia                      *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialPersonalExportacionServicios             *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialPersonalHistoriaServicios                *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialPersonalHistoriaRelaciones               *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialUsuariosPreferenciasConsultaInterna      *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialUsuariosPreferenciasActualizacionInterna *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialUsuariosPreferenciasConsultaExterna      *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialUsuariosPreferenciasActualizacionExterna *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialUsuariosCorreos                          proveedoresMaterialCorreosUsuarios
	materialUsuariosImagen                           proveedoresMaterialImagenUsuarios
	materialAspirantes                               proveedoresMaterialAspirantes
	materialPreparacionBases                         [2]*proveedorMaterialAltaContratacionTemporalDesarrollo
	materialPersonalB2                               [8]CapacidadPublicadaPersonalB2V3
	materialOrganizacionHistorica                    CapacidadPublicadaOrganizacionHistoricaV3
	errMaterialOrganizacionHistorica                 error
	detenerRenovacion                                func()
	detenerEntregaContratos                          func()
	detenerEntregaCeses                              func()
	catalogoMaterial                                 catalogoMaterialAutorizacionComunDesarrollo
	cerrarUnaVez                                     func()
}

func (d *dependenciasPostgreSQLContratacionTemporalDesarrollo) cerrar() {
	if d == nil {
		return
	}
	if d.cerrarAuditoriaLecturasBolsa != nil {
		d.cerrarAuditoriaLecturasBolsa()
	}
	if d.cerrarFirmasR5V2 != nil {
		d.cerrarFirmasR5V2()
	}
	if d.cerrarUnaVez != nil {
		d.cerrarUnaVez()
	}
}

func nuevasDependenciasPostgreSQLContratacionTemporalDesarrollo(
	cfg config.Config,
	derivador *derivadorIdentidadOperacionDesarrollo,
	soporte *soporteAltaContratacionTemporalDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (_ dependenciasPostgreSQLContratacionTemporalDesarrollo, errFinal error) {
	vacias := dependenciasPostgreSQLContratacionTemporalDesarrollo{}
	// etapa nombra el último paso iniciado; si la composición se detiene, se
	// registra junto con la clasificación de la causa (sin datos sensibles).
	etapa := "requisitos"
	defer func() {
		if errFinal != nil {
			registrarFalloPostgreSQLContratacionTemporalDesarrollo(
				"composicion:"+etapa, causaFalloPostgreSQLCTDesarrollo(errFinal),
			)
		}
	}()
	if !cfg.DevelopmentEnabledByDoubleKey() ||
		derivador == nil || !derivador.valido() || soporte == nil {
		return vacias, falloPostgreSQLCTDesarrollo(nil)
	}
	etapa = "dsn"
	configuracion := cfg.Normalize().ContratacionTemporalPostgreSQL
	dsnEjecucion, dsnGobierno, err := configuracion.DSNSeparados()
	if err != nil {
		return vacias, err
	}
	dsnConfirmador, dsnLectorResultado, err :=
		configuracion.DSNCoberturaSeparados()
	if err != nil {
		return vacias, err
	}
	dsnRegistroAutorizacion, err := configuracion.DSNRegistroAutorizacionSeparado()
	if err != nil {
		return vacias, err
	}
	dsnAuditoriaFrontera, err := configuracion.DSNAuditoriaFronteraSeparado()
	if err != nil {
		return vacias, err
	}
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
	defer cancelar()
	etapa = "derivar_material_atestacion"
	material, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(
		derivador, reloj.Ahora(),
	)
	if err != nil {
		registrarFalloPostgreSQLContratacionTemporalDesarrollo(
			"derivar_material_atestacion", "material_no_disponible",
		)
		return vacias, err
	}
	defer material.borrarCopiasEfimeras()
	etapa = "abrir_conexion_gobierno"
	gobierno, usuarioGobierno, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(
		ctx, dsnGobierno, "vec-ct-desarrollo-gobierno",
		rolGobiernoPostgreSQLContratacionTemporalDesarrollo,
	)
	if err != nil {
		registrarFalloPostgreSQLContratacionTemporalDesarrollo(
			"abrir_conexion_gobierno", "conexion_no_disponible",
		)
		return vacias, err
	}
	etapa = "publicar_gobierno_atestacion"
	if err := publicarGobiernoAtestacionContratacionTemporalDesarrollo(
		ctx, gobierno, &material,
	); err != nil {
		registrarFalloPostgreSQLContratacionTemporalDesarrollo(
			"publicar_gobierno_atestacion", codigoFalloGobiernoPostgreSQLContratacionTemporalDesarrollo(err),
		)
		gobierno.Close()
		return vacias, falloPostgreSQLCTDesarrollo(nil)
	}
	etapa = "publicar_gobierno_cobertura"
	if err := publicarGobiernoCoberturaPostgreSQLContratacionTemporalDesarrollo(
		ctx, gobierno, soporte,
	); err != nil {
		registrarFalloPostgreSQLContratacionTemporalDesarrollo(
			"publicar_gobierno_cobertura", "gobierno_cobertura_no_disponible",
		)
		gobierno.Close()
		return vacias, falloPostgreSQLCTDesarrollo(nil)
	}
	etapa = "publicar_autoridad_alta"
	if err := publicarAutoridadPostgreSQLContratacionTemporalDesarrollo(
		ctx, gobierno, soporte,
	); err != nil {
		registrarFalloPostgreSQLContratacionTemporalDesarrollo(
			"publicar_autoridad_alta", "autoridad_no_disponible",
		)
		gobierno.Close()
		return vacias, err
	}
	etapa = "asegurar_perfiles_fijos_rrhh"
	if err := asegurarPerfilesFijosCTDesarrollo(ctx, gobierno, soporte, aprobacionProvisionPerfilesRRHHDesdeConfig(cfg),
		soporte.perfilesFijosRegistrados()...); err != nil {
		gobierno.Close()
		return vacias, err
	}
	soporte.mu.Lock()
	soporte.autoridadAsignaciones = &autoridadPostgreSQLContratacionTemporalDesarrollo{
		pool: gobierno, soporte: soporte,
	}
	soporte.mu.Unlock()
	etapa = "abrir_conexion_ejecucion"
	ejecucion, usuarioEjecucion, err := abrirPoolPostgreSQLContratacionTemporalDesarrollo(
		ctx, dsnEjecucion, "vec-ct-desarrollo-ejecucion",
		rolEjecucionPostgreSQLContratacionTemporalDesarrollo,
	)
	if err != nil || usuarioEjecucion == usuarioGobierno {
		if ejecucion != nil {
			ejecucion.Close()
		}
		gobierno.Close()
		return vacias, falloPostgreSQLCTDesarrollo(err)
	}
	// Vías de cobertura y numeración del catálogo de reglas: solo se publica
	// una versión nueva si su contenido difiere del vigente.
	etapa = "publicar_gobierno_cobertura_catalogo"
	if err := sincronizarGobiernoCoberturaCatalogoPostgreSQLCT(
		ctx, gobierno, ejecucion, soporte, reloj,
	); err != nil {
		registrarFalloPostgreSQLContratacionTemporalDesarrollo(
			"publicar_gobierno_cobertura_catalogo", "gobierno_cobertura_no_disponible",
		)
		ejecucion.Close()
		gobierno.Close()
		return vacias, falloPostgreSQLCTDesarrollo(nil)
	}
	etapa = "publicar_numeracion_expedientes"
	if err := publicarNumeracionExpedientesCT(
		ctx, gobierno, soporte.opcionesCatalogo.numeracionVigente(),
	); err != nil {
		registrarFalloPostgreSQLContratacionTemporalDesarrollo(
			"publicar_numeracion_expedientes", codigoFalloNumeracionExpedientesCT(err),
		)
		ejecucion.Close()
		gobierno.Close()
		return vacias, err
	}
	dependencias := dependenciasPostgreSQLContratacionTemporalDesarrollo{
		ejecucion: ejecucion,
		gobierno:  gobierno,
	}
	var cierre sync.Once
	dependencias.cerrarUnaVez = func() {
		cierre.Do(func() {
			if dependencias.detenerRenovacion != nil {
				dependencias.detenerRenovacion()
			}
			if dependencias.detenerEntregaContratos != nil {
				dependencias.detenerEntregaContratos()
			}
			if dependencias.detenerEntregaCeses != nil {
				dependencias.detenerEntregaCeses()
			}
			if dependencias.bolsa != nil {
				dependencias.bolsa.Close()
			}
			if dependencias.calculadorPoliticaOfertas != nil {
				dependencias.calculadorPoliticaOfertas.Close()
			}
			if dependencias.lectorResultado != nil {
				dependencias.lectorResultado.Cerrar()
			}
			if dependencias.auditoriaFrontera != nil {
				dependencias.auditoriaFrontera.Close()
			}
			if dependencias.confirmador != nil {
				dependencias.confirmador.Close()
			}
			if dependencias.ejecucion != nil {
				dependencias.ejecucion.Close()
			}
			if dependencias.registroAutorizacion != nil {
				dependencias.registroAutorizacion.Close()
			}
			if dependencias.gobierno != nil {
				dependencias.gobierno.Close()
			}
		})
	}
	completa := false
	defer func() {
		if !completa {
			dependencias.cerrar()
		}
	}()
	etapa = "abrir_conexion_registro_autorizacion"
	registroAutorizacion, usuarioRegistroAutorizacion, err :=
		abrirPoolPostgreSQLContratacionTemporalDesarrollo(
			ctx,
			dsnRegistroAutorizacion,
			"vec-ct-desarrollo-registro-autorizacion",
			rolRegistroAutorizacionPostgreSQLContratacionTemporalDesarrollo,
		)
	if err != nil || usuarioRegistroAutorizacion == usuarioEjecucion ||
		usuarioRegistroAutorizacion == usuarioGobierno {
		if registroAutorizacion != nil {
			registroAutorizacion.Close()
		}
		return vacias, falloPostgreSQLCTDesarrollo(err)
	}
	registroDecisiones, err := postgresvec.NuevoAlmacenAutorizacion(registroAutorizacion)
	if err != nil {
		registroAutorizacion.Close()
		return vacias, falloPostgreSQLCTDesarrollo(err)
	}
	dependencias.registroAutorizacion = registroAutorizacion
	etapa = "abrir_conexion_auditoria_frontera"
	auditoriaFrontera, usuarioAuditoriaFrontera, err :=
		abrirPoolPostgreSQLContratacionTemporalDesarrollo(
			ctx,
			dsnAuditoriaFrontera,
			"vec-ct-desarrollo-auditoria-frontera",
			rolAuditoriaFronteraPostgreSQLContratacionTemporalDesarrollo,
		)
	if err != nil || usuarioAuditoriaFrontera == usuarioEjecucion ||
		usuarioAuditoriaFrontera == usuarioGobierno ||
		usuarioAuditoriaFrontera == usuarioRegistroAutorizacion {
		if auditoriaFrontera != nil {
			auditoriaFrontera.Close()
		}
		return vacias, falloPostgreSQLCTDesarrollo(err)
	}
	registradorAuditoriaFrontera, err := postgresvec.NuevoRegistradorAuditoriaFronteraRutaExactaPostgreSQL(auditoriaFrontera)
	if err != nil {
		auditoriaFrontera.Close()
		return vacias, falloPostgreSQLCTDesarrollo(err)
	}
	if err := registradorAuditoriaFrontera.PreflightAuditoriaFronteraRutaExacta(ctx); err != nil {
		auditoriaFrontera.Close()
		return vacias, falloPostgreSQLCTDesarrollo(err)
	}
	dependencias.auditoriaFrontera = auditoriaFrontera
	dependencias.registradorAuditoriaFrontera = registradorAuditoriaFrontera
	soporte.mu.Lock()
	soporte.registroDecisionesAnalisis = registroDecisiones
	soporte.mu.Unlock()
	etapa = "resolutor_candidaturas"
	resolver, err := postgrescontratacion.NuevoResolutorCandidaturaAltaPostgreSQL(ejecucion)
	if err != nil {
		return vacias, err
	}
	etapa = "fuente_confianza_renovable"
	material.fuenteConfianza, err = nuevaFuenteConfianzaRenovableCTDesarrollo(gobierno, material, reloj)
	if err != nil {
		return vacias, err
	}
	etapa = "seleccion_material"
	seleccion, err := seleccionMaterialCTDesarrolloDesdeConfig(cfg)
	if err != nil {
		return vacias, err
	}
	// CT137/AD3-100 deben existir y conservar sus ACL antes de publicar la
	// audiencia documental. Se reutiliza el pool ejecutor ya acreditado.
	if seleccion.plantillasDocumental {
		etapa = "preflight_plantillas_documental"
		if err := preflightCatalogoPlantillasCT(ctx, ejecucion); err != nil {
			return vacias, err
		}
	}
	// B55 se comprueba con el LOGIN Bolsa que consumirá la lectura. La
	// comprobación precede a la publicación de su clave en el gobierno V3.
	if seleccion.reincorporacionTitular {
		etapa = "preflight_lectura_reincorporacion_bolsa"
		dependencias.bolsa, err = abrirBolsaLlamamientosPostgreSQLDesarrollo(ctx, configuracion)
		if err != nil {
			return vacias, err
		}
		if err := comprobarLecturaReincorporacionTitularB55Desarrollo(ctx, dependencias.bolsa); err != nil {
			return vacias, err
		}
	}
	firmaDocumento, personalB2 := seleccion.firmaDocumento, seleccion.personalB2
	descriptoresMaterial := descriptoresMaterialSeleccionadosCTDesarrollo(seleccion)
	usuariosPreferenciasActivas, err := selectorCapacidadRRHHDesarrollo(cfg, envUsuariosPreferenciasDesarrollo)
	if err != nil {
		return vacias, err
	}
	if usuariosPreferenciasActivas {
		etapa = "preflight_sql_usuarios_preferencias"
		descriptoresUsuarios, falloPreflight := descriptoresMaterialPreferenciasTrasPreflight(func() error { return preflightSQLPreferenciasUsuariosDesarrollo(cfg, derivador, gobierno) })
		if falloPreflight != nil {
			return vacias, falloPreflight
		}
		descriptoresMaterial = append(descriptoresMaterial, descriptoresUsuarios...)
	}
	etapa = "preflight_sql_usuarios_correos"
	usuariosCorreosActivos, descriptoresCorreos, err := seleccionCorreosUsuariosDesarrollo(cfg, usuariosPreferenciasActivas)
	if err != nil {
		return vacias, err
	}
	descriptoresMaterial = append(descriptoresMaterial, descriptoresCorreos...)
	etapa = "preflight_sql_usuarios_imagen"
	usuariosImagenActiva, descriptoresImagen, err := seleccionImagenUsuariosDesarrollo(cfg, usuariosPreferenciasActivas)
	if err != nil {
		return vacias, err
	}
	descriptoresMaterial = append(descriptoresMaterial, descriptoresImagen...)
	etapa = "preflight_sql_aspirantes"
	aspirantesActivo, descriptoresAspirantes, err := seleccionAspirantesDesarrollo(cfg, usuariosPreferenciasActivas)
	if err != nil {
		return vacias, err
	}
	descriptoresMaterial = append(descriptoresMaterial, descriptoresAspirantes...)
	_, preparacionBasesActiva, err := leerConfiguracionPreparacionBasesV3(cfg)
	if err != nil {
		return vacias, err
	}
	if preparacionBasesActiva {
		if !cfg.BolsaBorradoresEnabled || !cfg.ContratacionTemporalPostgreSQL.ConsultasRRHHConfiguradas() {
			return vacias, errMontajePreparacionBasesV3
		}
		descriptores := DescriptoresMaterialPreparacionBasesV3()
		descriptoresMaterial = append(descriptoresMaterial, descriptores[:]...)
	}
	auditoriaActiva, err := selectorCapacidadRRHHDesarrollo(cfg, envRRHHAuditoriaEnabled)
	if err != nil {
		return vacias, err
	}
	if auditoriaActiva {
		if _, err := cfg.RutaCatalogoAuditoriaConsultaDesarrollo(); err != nil {
			return vacias, err
		}
		descriptoresMaterial = append(descriptoresMaterial, descriptorMaterialAuditoriaConsultaDesarrollo())
	}
	catalogoMaterial, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptoresMaterial)
	if err != nil {
		return vacias, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	dependencias.catalogoMaterial = catalogoMaterial
	if preparacionBasesActiva {
		etapa = "material_preparacion_bases"
		for i, descriptor := range DescriptoresMaterialPreparacionBasesV3() {
			dependencias.materialPreparacionBases[i], err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, descriptor.Audiencia)
			if err != nil {
				return vacias, err
			}
		}
	}
	if usuariosPreferenciasActivas {
		etapa = "material_usuarios_preferencias"
		lote, fallo := publicarMaterialPreferenciasUsuariosEnLote(ctx, gobierno, material, reloj, catalogoMaterial)
		if fallo != nil {
			return vacias, fallo
		}
		dependencias.materialUsuariosPreferenciasConsultaInterna = lote[0]
		dependencias.materialUsuariosPreferenciasActualizacionInterna = lote[1]
		dependencias.materialUsuariosPreferenciasConsultaExterna = lote[2]
		dependencias.materialUsuariosPreferenciasActualizacionExterna = lote[3]
	}
	if etapa = "material_usuarios_correos"; usuariosCorreosActivos {
		if dependencias.materialUsuariosCorreos, err = publicarMaterialCorreosUsuariosEnLote(ctx, gobierno, material, reloj, catalogoMaterial); err != nil {
			return vacias, err
		}
	}
	if etapa = "material_usuarios_imagen"; usuariosImagenActiva {
		if dependencias.materialUsuariosImagen, err = publicarMaterialImagenUsuariosEnLote(ctx, gobierno, material, reloj, catalogoMaterial); err != nil {
			return vacias, err
		}
	}
	if etapa = "material_aspirantes"; aspirantesActivo {
		if dependencias.materialAspirantes, err = publicarMaterialAspirantesEnLote(ctx, gobierno, material, reloj, catalogoMaterial); err != nil {
			return vacias, err
		}
	}
	if seleccion.plantillasCatalogo {
		etapa = "material_plantillas_catalogo"
		dependencias.proveedorMaterialPlantillasCatalogo, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(
			ctx, gobierno, material, reloj, catalogoMaterial, audienciaCatalogoPlantillasCT)
		if err != nil {
			return vacias, err
		}
	}
	if seleccion.ajustesReglasCT {
		etapa = "material_ajustes_reglas_ct"
		dependencias.proveedorMaterialAjustesReglasCT, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(
			ctx, gobierno, material, reloj, catalogoMaterial, "vec_contratacion_temporal.ajustes_reglas.v1")
		if err != nil {
			return vacias, err
		}
	}
	if seleccion.plantillasDocumental {
		etapa = "material_plantillas_documental"
		dependencias.proveedorMaterialPlantillasDocumental, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(
			ctx, gobierno, material, reloj, catalogoMaterial, audienciaDocumentalPlantillasCT)
		if err != nil {
			return vacias, err
		}
	}
	etapa = "material_dietas"
	if dietasBorradoresSolicitadas(cfg.DietasBorradoresEnabled) {
		proveedoresDietas := make(map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo)
		for _, descriptor := range descriptoresMaterialDietasDesarrollo() {
			proveedoresDietas[descriptor.Audiencia], err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, descriptor.Audiencia)
			if err != nil {
				return vacias, err
			}
		}
		dependencias.materialDietas = materialDietasDesdeCTDesarrollo{
			personal:    proveedoresDietas[audienciaConsumoPersonalDietasDesarrollo],
			crear:       proveedoresDietas[audienciaConsumoCrearDietasDesarrollo],
			consultar:   proveedoresDietas[audienciaConsumoConsultarDietasDesarrollo],
			adicionales: proveedoresDietas,
		}
	}
	etapa = "material_cronos"
	if cronosEmpleadoSolicitado(cfg.CronosEmpleadoEnabled) {
		var cronos [8]*proveedorMaterialAltaContratacionTemporalDesarrollo
		for i, audiencia := range audienciasCronosEmpleadoDesarrollo() {
			cronos[i], err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, audiencia)
			if err != nil {
				return vacias, err
			}
		}
		dependencias.materialCronos = materialCronosDesdeProveedores(cronos)
		if cronosResolucionSolicitada(cfg.CronosEmpleadoEnabled, cfg.CronosResolucionEnabled) {
			var resolucion [4]*proveedorMaterialAltaContratacionTemporalDesarrollo
			for i, audiencia := range audienciasCronosResolucionDesarrollo() {
				resolucion[i], err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, audiencia)
				if err != nil {
					return vacias, err
				}
			}
			dependencias.materialCronos = dependencias.materialCronos.conResolucion(resolucion)
		}
		if cronosNotificacionesSolicitadas(cfg.CronosEmpleadoEnabled, cfg.CronosNotificacionesEnabled) {
			var notificaciones [4]*proveedorMaterialAltaContratacionTemporalDesarrollo
			for i, audiencia := range audienciasCronosNotificacionesDesarrollo() {
				notificaciones[i], err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, audiencia)
				if err != nil {
					return vacias, err
				}
			}
			dependencias.materialCronos = dependencias.materialCronos.conNotificaciones(notificaciones)
		}
	}
	etapa = "material_documentos"
	if documentosSolicitados(cfg.DocumentosEnabled) {
		dependencias.materialDocumentos, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, descriptoresMaterialDocumentosDesarrollo()[0].Audiencia)
		if err != nil {
			return vacias, err
		}
	}
	etapa = "material_personal_ficha_propia"
	if personalEmpleadoSolicitado(cfg.PersonalEmpleadoEnabled) {
		dependencias.materialPersonalFichaPropia, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, personaldomain.AudienciaFichaPropia)
		if err != nil {
			return vacias, err
		}
	}
	seleccionExportacion, err := exportacionServiciosPersonalSolicitada(cfg)
	if err != nil {
		return vacias, err
	}
	if seleccionExportacion {
		etapa = "material_personal_exportacion_servicios"
		dependencias.materialPersonalExportacionServicios, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, personaldomain.AudienciaExportacionServiciosPropios)
		if err != nil {
			return vacias, err
		}
	}
	seleccionHistoria, err := historiaServiciosPersonalSolicitada(cfg)
	if err != nil {
		return vacias, err
	}
	if seleccionHistoria {
		etapa = "material_personal_historia_servicios"
		dependencias.materialPersonalHistoriaServicios, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, personaldomain.AudienciaHistoriaServiciosPropia)
		if err != nil {
			return vacias, err
		}
	}
	seleccionHistoriaRelaciones, err := historiaRelacionesPersonalSolicitada(cfg)
	if err != nil {
		return vacias, err
	}
	if seleccionHistoriaRelaciones {
		etapa = "material_personal_historia_relaciones"
		dependencias.materialPersonalHistoriaRelaciones, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, personaldomain.AudienciaHistoriaRelacionesPropia)
		if err != nil {
			return vacias, err
		}
	}
	etapa = "material_firma_documento"
	if firmaDocumento {
		dependencias.proveedorMaterialFirmaDocumento, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, ports.AudienciaFirmaDocumentoV3)
		if err != nil {
			return vacias, err
		}
		dependencias.proveedorMaterialConsultaFirmasDocumento, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, ports.AudienciaConsultaFirmasDocumentoV3)
		if err != nil {
			return vacias, err
		}
	}
	etapa = "material_auditoria_rrhh"
	if auditoriaActiva {
		dependencias.proveedorMaterialAuditoriaCT, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(
			ctx, gobierno, material, reloj, catalogoMaterial, auditoria.AudienciaConsumo)
		if err != nil {
			return vacias, err
		}
		dependencias.proveedorMaterialAuditoriaBolsa, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(
			ctx, gobierno, material, reloj, catalogoMaterial, auditoria.AudienciaConsumo)
		if err != nil {
			return vacias, err
		}
	}
	etapa = "publicar_gobierno_personal_b2"
	if personalB2 {
		// vec-server no consume B2: sólo publica sus claves para vec-interno.
		dependencias.materialPersonalB2, err = publicarMaterialPersonalB2Desarrollo(ctx, gobierno, material, catalogoMaterial)
		if err != nil {
			registrarFalloPostgreSQLContratacionTemporalDesarrollo(
				"publicar_gobierno_personal_b2", codigoFalloGobiernoPostgreSQLContratacionTemporalDesarrollo(err),
			)
			return vacias, falloPostgreSQLCTDesarrollo(err)
		}
	}
	etapa = "publicar_gobierno_incorporacion_b2"
	if seleccion.incorporacionB2 {
		if err := publicarMaterialIncorporacionB2Desarrollo(ctx, gobierno, material, catalogoMaterial); err != nil {
			registrarFalloPostgreSQLContratacionTemporalDesarrollo(
				"publicar_gobierno_incorporacion_b2", codigoFalloGobiernoPostgreSQLContratacionTemporalDesarrollo(err),
			)
			return vacias, falloPostgreSQLCTDesarrollo(err)
		}
	}
	etapa = "proveedor_material_alta"
	proveedor, err := nuevoProveedorMaterialAltaContratacionTemporalDesarrollo(
		material, soporte, reloj,
	)
	if err != nil {
		return vacias, err
	}
	etapa = "material_bolsa"
	if configuracion.BolsaLlamamientosConfigurada() {
		if dependencias.bolsa == nil {
			dependencias.bolsa, err = abrirBolsaLlamamientosPostgreSQLDesarrollo(ctx, configuracion)
			if err != nil {
				return vacias, err
			}
		}
		bolsa := dependencias.bolsa
		if seleccion.politicaOfertas {
			dsnCalculador, err := cfg.DSNBolsaPoliticaOfertasCalculadorSeparado()
			if err != nil {
				return vacias, err
			}
			dependencias.calculadorPoliticaOfertas, err = abrirPoolRelevoBolsaDesarrollo(ctx, dsnCalculador,
				"vec_bolsa_llamamientos_calculador_politica", "vec-bolsa-calculador-politica-ofertas")
			if err != nil {
				return vacias, falloPostgreSQLCTDesarrollo(err)
			}
		}
		if seleccion.portalCandidato {
			if err := comprobarMigracionesPortalCandidatoDesarrollo(ctx, bolsa); err != nil {
				slog.Error("portal del candidato de Bolsa encendido sin sus migraciones", "causa", err)
				return vacias, err
			}
		}
		proveedorBolsa, err := nuevoProveedorMaterialBolsaDesarrollo(ctx, gobierno, material, soporte, reloj)
		if err != nil {
			return vacias, err
		}
		dependencias.proveedorMaterialBolsa = proveedorBolsa
		if seleccion.miBolsa {
			proveedorMiBolsa, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaMiBolsa)
			if err != nil {
				return vacias, err
			}
			dependencias.proveedorMaterialMiBolsa = proveedorMiBolsa
			proveedorHistorial, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaHistorialMiBolsa)
			if err != nil {
				return vacias, err
			}
			dependencias.proveedorMaterialHistorialMiBolsa = proveedorHistorial
		}
		if seleccion.portalCandidato {
			dependencias.proveedoresMaterialPortal = make(map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo, len(accionesPropiasPortalDesarrollo()))
			for _, par := range accionesPropiasPortalDesarrollo() {
				proveedor, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, par[1])
				if err != nil {
					return vacias, err
				}
				dependencias.proveedoresMaterialPortal[par[0]] = proveedor
			}
		}
		if cfg.BolsaBorradoresEnabled {
			if seleccion.reincorporacionTitular {
				etapa = "material_consulta_reincorporacion_titular_bolsa"
				dependencias.proveedorMaterialConsultaReincorporacionTitular, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(
					ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaConsultarReincorporacionTitular)
				if err != nil {
					return vacias, err
				}
			}
			proveedorBorradorCrear, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(
				ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaCrearBorradorLlamamientoInterno,
			)
			if err != nil {
				return vacias, err
			}
			proveedorBorradorConsulta, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(
				ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaConsultarBorradorLlamamientoInterno,
			)
			if err != nil {
				return vacias, err
			}
			dependencias.proveedorMaterialBorradorCrear = proveedorBorradorCrear
			dependencias.proveedorMaterialBorradorConsulta = proveedorBorradorConsulta
			proveedorSituacion, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaCambiarSituacionParticipacion)
			if err != nil {
				return vacias, err
			}
			dependencias.proveedorMaterialSituacion = proveedorSituacion
			proveedorDocumentales, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaConsultarSolicitudesDocumentalesRRHH)
			if err != nil {
				return vacias, err
			}
			dependencias.proveedorMaterialConsultaSolicitudesDocumentales = proveedorDocumentales
			proveedorContacto, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaRegistrarContactoParticipacion)
			if err != nil {
				return vacias, err
			}
			dependencias.proveedorMaterialContacto = proveedorContacto
			proveedorConsultaContacto, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaConsultarContactoParticipacion)
			if err != nil {
				return vacias, err
			}
			dependencias.proveedorMaterialConsultaContacto = proveedorConsultaContacto
			proveedorDatosContacto, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaRegistrarDatosContactoParticipacion)
			if err != nil {
				return vacias, err
			}
			dependencias.proveedorMaterialDatosContacto = proveedorDatosContacto
			if dependencias.proveedorMaterialConsultaDatosContacto, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaConsultarDatosContactoParticipacion); err != nil {
				return vacias, err
			}
			proveedorEmision, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaEmitirLlamamiento)
			if err != nil {
				return vacias, err
			}
			dependencias.proveedorMaterialEmision = proveedorEmision
			if seleccion.politicaOfertas {
				dependencias.proveedorMaterialPoliticaOfertas, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(
					ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaPublicarPoliticaOfertas)
				if err != nil {
					return vacias, err
				}
				dependencias.proveedorMaterialConsultaPoliticaOfertas, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(
					ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaConsultarPoliticaOfertas)
				if err != nil {
					return vacias, err
				}
			}
		}
	}
	etapa = "transaccion_altas"
	transaccion, err := postgrescontratacion.NuevaTransaccionAltasPostgreSQLCandidata(
		ejecucion, proveedor,
	)
	if err != nil {
		return vacias, err
	}
	etapa = "abrir_conexion_confirmador"
	confirmador, usuarioConfirmador, err :=
		abrirPoolPostgreSQLContratacionTemporalDesarrollo(
			ctx,
			dsnConfirmador,
			"vec-ct-desarrollo-confirmador",
			rolConfirmadorPostgreSQLContratacionTemporalDesarrollo,
		)
	if err != nil || usuarioConfirmador == usuarioEjecucion ||
		usuarioConfirmador == usuarioGobierno ||
		usuarioConfirmador == usuarioRegistroAutorizacion {
		if confirmador != nil {
			confirmador.Close()
		}
		return vacias, falloPostgreSQLCTDesarrollo(err)
	}
	dependencias.confirmador = confirmador
	etapa = "abrir_conexion_lector_resultado"
	inspectorLector, usuarioLector, err :=
		abrirPoolPostgreSQLContratacionTemporalDesarrollo(
			ctx,
			dsnLectorResultado,
			"vec-ct-desarrollo-lector-preflight",
			rolLectorPostgreSQLContratacionTemporalDesarrollo,
		)
	if inspectorLector != nil {
		inspectorLector.Close()
	}
	if err != nil || usuarioLector == usuarioEjecucion ||
		usuarioLector == usuarioGobierno ||
		usuarioLector == usuarioRegistroAutorizacion || usuarioLector == usuarioConfirmador {
		return vacias, falloPostgreSQLCTDesarrollo(err)
	}
	lectorResultado, err :=
		postgrescontratacion.NuevoPoolRecuperacionCoberturaO405PostgreSQL(
			ctx,
			dsnLectorResultado,
		)
	if err != nil {
		return vacias, err
	}
	dependencias.lectorResultado = lectorResultado
	dependencias.candidaturas = resolver
	dependencias.transaccionAlta = transaccion
	dependencias.proveedorMaterial = proveedor
	dependencias.detenerRenovacion = iniciarRenovacionProgramadaCTDesarrollo(material.fuenteConfianza, esperarTemporizadorCTDesarrollo)
	if dependencias.bolsa != nil && cfg.BolsaBorradoresEnabled {
		// B13: el relevo es opcional; si su configuración es inválida no arranca.
		if dependencias.detenerEntregaContratos, err = iniciarEntregaContratosCTBolsaDesarrollo(cfg.BolsaContratosCT, ejecucion, dependencias.bolsa); err != nil {
			slog.Error("entrega de contratos CT a Bolsa no iniciada", "causa", err)
		}
	}
	ceseActivo, err := selectorCapacidadRRHHDesarrollo(cfg, envBolsaCeseCTEnabled)
	if err != nil {
		return vacias, err
	}
	if ceseActivo {
		if dependencias.detenerEntregaContratos == nil {
			return vacias, puertosbolsa.ErrContratosParticipacionNoDisponible
		}
		dependencias.detenerEntregaCeses, err = iniciarEntregaCesesCTBolsaDesarrollo(ctx, cfg, ejecucion)
		if err != nil {
			return vacias, err
		}
	}
	// Organización histórica es opcional e independiente de B2. La selección
	// y la publicación usan el gobierno central, pero su fallo sólo deja esta
	// capacidad sin material; no interrumpe las capacidades anteriores.
	catalogoOH, activoOH, falloOH := seleccionarMaterialOrganizacionHistorica(cfg, descriptoresMaterial)
	if falloOH == nil && activoOH {
		dependencias.materialOrganizacionHistorica, falloOH = publicarMaterialOrganizacionHistorica(ctx, gobierno, material, catalogoOH)
	}
	dependencias.errMaterialOrganizacionHistorica = falloOH
	if falloOH != nil {
		registrarFalloPostgreSQLContratacionTemporalDesarrollo(
			"material_organizacion_historica", "capacidad_no_disponible",
		)
	}
	completa = true
	return dependencias, nil
}

func descriptorMaterialMiBolsaDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        puertosbolsa.AudienciaMiBolsa,
		Dominio:          "vec.bolsa.mi-bolsa.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:bolsa-mi-bolsa:",
		ProveedorNominal: "proveedor-material-bolsa-mi-bolsa",
	}
}

func descriptorMaterialHistorialMiBolsaDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        puertosbolsa.AudienciaHistorialMiBolsa,
		Dominio:          "vec.bolsa.mi-bolsa.historial.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:bolsa-mi-bolsa-historial:",
		ProveedorNominal: "proveedor-material-bolsa-mi-bolsa-historial",
	}
}

// descriptoresMaterialPortalCandidatoDesarrollo declara las audiencias
// de AD3-84 en el catálogo común de material.
func descriptoresMaterialPortalCandidatoDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	descriptores := make([]descriptorMaterialConsumidorV3Desarrollo, 0, len(puertosbolsa.AccionesPortalCandidato()))
	for _, par := range puertosbolsa.AccionesPortalCandidato() {
		nombre := strings.TrimPrefix(par[0], "bolsa.participaciones_propias.")
		descriptores = append(descriptores, descriptorMaterialConsumidorV3Desarrollo{
			Audiencia:        par[1],
			Dominio:          "vec.bolsa.mi-bolsa." + nombre + ".desarrollo.capacidad-v3",
			Prefijo:          "clave:capacidad:bolsa-mi-bolsa-" + strings.ReplaceAll(nombre, "_", "-") + ":",
			ProveedorNominal: "proveedor-material-bolsa-mi-bolsa-" + strings.ReplaceAll(nombre, "_", "-"),
		})
	}
	return descriptores
}

// debeComponerPortalCandidatoDesarrollo: las acciones propias existen solo
// si se piden expresamente (VEC_BOLSA_PORTAL_CANDIDATO_ENABLED, que exige
// AD3-84, AD3-86 y Bolsa 000029, 000030 y 000040 instaladas; el arranque lo
// comprueba en comprobarMigracionesPortalCandidatoDesarrollo), con «Mi bolsa»
// compuesta y catálogo de reglas de Bolsa que las rija. Un selector inválido
// se rechaza al arrancar.
func debeComponerPortalCandidatoDesarrollo(cfg config.Config) bool {
	activo, err := cfg.BolsaPortalCandidatoDesarrolloActivo()
	if err != nil {
		slog.Error("portal del candidato de Bolsa no compuesto: selector inválido", "causa", err)
		return false
	}
	if !activo {
		return false
	}
	rutas, activas, err := cfg.ReglasEjemploDesarrollo()
	return debeComponerMiBolsaDesarrollo(cfg) && err == nil && activas && rutas.BolsaSourcePath != ""
}

func debeComponerMiBolsaDesarrollo(cfg config.Config) bool {
	if !cfg.ContratacionTemporalPostgreSQL.BolsaLlamamientosConfigurada() {
		return false
	}
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "bolsa-candidato.json")
	_, err := os.Lstat(ruta)
	return err == nil
}

// La lectura B55 usa el ejecutor nominal de Bolsa y la función v2. La v1
// permanece como historia B46, pero no puede seguir ejecutable por ese LOGIN.
// La fachada AD3-101 sólo la invoca el propietario de Bolsa desde la v2.
const consultaLecturaReincorporacionTitularB55Desarrollo = `WITH RECURSIVE membresias_efectivas(rol_id) AS (
 SELECT directa.roleid FROM pg_catalog.pg_auth_members AS directa
 WHERE directa.member = session_user::pg_catalog.regrole
 UNION
 SELECT siguiente.roleid FROM pg_catalog.pg_auth_members AS siguiente
 JOIN membresias_efectivas AS previa ON previa.rol_id = siguiente.member
), funciones AS (
 SELECT
  pg_catalog.to_regprocedure('vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v2(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') AS nueva,
  pg_catalog.to_regprocedure('vec_bolsa_llamamientos.listar_reincorporaciones_titular_ct_v1(text,text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)') AS anterior
), fachada AS (
 SELECT p.oid FROM pg_catalog.pg_proc AS p
 JOIN pg_catalog.pg_namespace AS n ON n.oid = p.pronamespace
 WHERE n.nspname = 'vec_autorizacion_atestada_v3'
  AND p.proname = 'consumir_consulta_reincorporacion_titular_bolsa_v3_atestada'
  AND p.pronargs = 10
  AND p.proargtypes[0] = 'pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[1] = 'pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[2] = 'pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[3] = 'pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[4] = 'pg_catalog.numeric'::pg_catalog.regtype
  AND p.proargtypes[5] = 'pg_catalog.numeric'::pg_catalog.regtype
  AND p.proargtypes[6] = 'pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[7] = 'pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[8] = 'pg_catalog.bytea'::pg_catalog.regtype
  AND p.proargtypes[9] = 'pg_catalog.bytea'::pg_catalog.regtype
  AND p.proowner = 'vec_autorizacion_atestada_v3_propietario'::pg_catalog.regrole
  AND p.prosecdef
), tabla_lectura AS (
 SELECT c.oid FROM pg_catalog.pg_class AS c
 JOIN pg_catalog.pg_namespace AS n ON n.oid = c.relnamespace
 WHERE n.nspname = 'vec_bolsa_llamamientos'
  AND c.relname = 'reincorporacion_titular_lectura_v3'
  AND c.relkind = 'r'
  AND c.relowner = 'vec_bolsa_llamamientos_propietario'::pg_catalog.regrole
  AND EXISTS (SELECT 1 FROM pg_catalog.pg_trigger AS t
   WHERE t.tgrelid = c.oid AND t.tgname = 'reincorporacion_titular_lectura_inmutable'
    AND NOT t.tgisinternal AND t.tgenabled = 'O' AND t.tgtype = 27
    AND t.tgqual IS NULL AND t.tgattr::text = ''
    AND t.tgfoid = pg_catalog.to_regprocedure('vec_bolsa_llamamientos.constitucion_rechazar_mutacion()'))
)
SELECT session_user = current_user
 AND identidad.rolcanlogin AND identidad.rolinherit
 AND NOT identidad.rolsuper AND NOT identidad.rolcreatedb
 AND NOT identidad.rolcreaterole AND NOT identidad.rolreplication
 AND NOT identidad.rolbypassrls
 AND pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_ejecutor','MEMBER')
 AND NOT EXISTS (SELECT 1 FROM membresias_efectivas
  WHERE rol_id <> 'vec_bolsa_llamamientos_ejecutor'::pg_catalog.regrole)
 AND NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_propietario','MEMBER')
 AND NOT pg_catalog.pg_has_role(session_user,'vec_bolsa_llamamientos_migrador','MEMBER')
 AND funciones.nueva IS NOT NULL AND funciones.anterior IS NOT NULL
 AND pg_catalog.has_schema_privilege(session_user,'vec_bolsa_llamamientos','USAGE')
 AND pg_catalog.has_schema_privilege('vec_bolsa_llamamientos_propietario','vec_autorizacion_atestada_v3','USAGE')
 AND COALESCE(pg_catalog.has_function_privilege(session_user,funciones.nueva,'EXECUTE'),false)
 AND NOT COALESCE(pg_catalog.has_function_privilege(session_user,funciones.anterior,'EXECUTE'),false)
 AND pg_catalog.has_function_privilege('vec_bolsa_llamamientos_propietario',fachada.oid,'EXECUTE')
 AND NOT pg_catalog.has_function_privilege(session_user,fachada.oid,'EXECUTE')
 AND pg_catalog.has_table_privilege(session_user,tabla_lectura.oid,
  'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN') IS FALSE
 AND pg_catalog.has_table_privilege('vec_bolsa_llamamientos_ejecutor',tabla_lectura.oid,
  'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER,MAINTAIN') IS FALSE
 AND pg_catalog.has_any_column_privilege(session_user,tabla_lectura.oid,
  'SELECT,INSERT,UPDATE,REFERENCES') IS FALSE
 AND pg_catalog.has_any_column_privilege('vec_bolsa_llamamientos_ejecutor',tabla_lectura.oid,
  'SELECT,INSERT,UPDATE,REFERENCES') IS FALSE
FROM pg_catalog.pg_roles AS identidad CROSS JOIN funciones CROSS JOIN fachada CROSS JOIN tabla_lectura
WHERE identidad.rolname = session_user`

func comprobarLecturaReincorporacionTitularB55Desarrollo(ctx context.Context, consultador interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) error {
	if ctx == nil || consultador == nil {
		return puertosbolsa.ErrReincorporacionTitularNoDisponible
	}
	var autorizada bool
	if err := consultador.QueryRow(ctx, consultaLecturaReincorporacionTitularB55Desarrollo).Scan(&autorizada); err != nil || !autorizada {
		return puertosbolsa.ErrReincorporacionTitularNoDisponible
	}
	return nil
}
