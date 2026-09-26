package bootstrap

import (
	"context"
	"crypto/ed25519"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"vec-diputacion-granada/config"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgrescontratacion "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	confianzaatestacion "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
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
	ejecucion                    *pgxpool.Pool
	bolsa                        *pgxpool.Pool
	gobierno                     *pgxpool.Pool
	registroAutorizacion         *pgxpool.Pool
	confirmador                  *pgxpool.Pool
	lectorResultado              *postgrescontratacion.PoolRecuperacionCoberturaO405PostgreSQL
	registradorAuditoriaFrontera *postgresvec.RegistradorAuditoriaFronteraRutaExactaPostgreSQL
	auditoriaFrontera            *pgxpool.Pool
	candidaturas                 ports.ResolutorCandidaturaAlta
	transaccionAlta              ports.TransaccionAltasCandidata
	proveedorMaterial            *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialBolsa       *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialMiBolsa     *proveedorMaterialAltaContratacionTemporalDesarrollo
	// proveedoresMaterialPortal: uno por acción propia del candidato que
	// tiene consumidor compuesto (AD3-84 con Bolsa 000030).
	proveedoresMaterialPortal         map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialBorradorCrear    *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialBorradorConsulta *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialSituacion        *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialContacto         *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialConsultaContacto *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialDatosContacto    *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialEmision          *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialDespachoCorreo   *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialResultadoCorreo  *proveedorMaterialAltaContratacionTemporalDesarrollo
	proveedorMaterialFirmaDocumento   *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialDietas                    materialDietasDesdeCTDesarrollo
	materialCronos                    materialCronosDesdeCTDesarrollo
	materialDocumentos                *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialPersonalFichaPropia       *proveedorMaterialAltaContratacionTemporalDesarrollo
	materialPersonalB2                [8]CapacidadPublicadaPersonalB2V3
	detenerRenovacion                 func()
	detenerEntregaContratos           func()
	catalogoMaterial                  catalogoMaterialAutorizacionComunDesarrollo
	cerrarUnaVez                      func()
}

func (d *dependenciasPostgreSQLContratacionTemporalDesarrollo) cerrar() {
	if d == nil {
		return
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
	ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
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
			if dependencias.bolsa != nil {
				dependencias.bolsa.Close()
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
	firmaDocumento, personalB2 := seleccion.firmaDocumento, seleccion.personalB2
	descriptoresMaterial := descriptoresMaterialSeleccionadosCTDesarrollo(seleccion)
	catalogoMaterial, err := nuevoCatalogoMaterialAutorizacionComunDesarrollo(descriptoresMaterial)
	if err != nil {
		return vacias, errGobiernoPostgreSQLContratacionTemporalDesarrolloIncoherente
	}
	dependencias.catalogoMaterial = catalogoMaterial
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
	etapa = "material_firma_documento"
	if firmaDocumento {
		dependencias.proveedorMaterialFirmaDocumento, err = nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, ports.AudienciaFirmaDocumentoV3)
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
	etapa = "proveedor_material_alta"
	proveedor, err := nuevoProveedorMaterialAltaContratacionTemporalDesarrollo(
		material, soporte, reloj,
	)
	if err != nil {
		return vacias, err
	}
	etapa = "material_bolsa"
	if configuracion.BolsaLlamamientosConfigurada() {
		bolsa, err := abrirBolsaLlamamientosPostgreSQLDesarrollo(ctx, configuracion)
		if err != nil {
			return vacias, err
		}
		dependencias.bolsa = bolsa
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
			proveedorEmision, err := nuevoProveedorMaterialBorradorLlamamientoDesarrollo(ctx, gobierno, material, reloj, catalogoMaterial, puertosbolsa.AudienciaEmitirLlamamiento)
			if err != nil {
				return vacias, err
			}
			dependencias.proveedorMaterialEmision = proveedorEmision
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

// descriptoresMaterialPortalCandidatoDesarrollo declara las cuatro audiencias
// de AD3-84 en el catálogo común de material.
func descriptoresMaterialPortalCandidatoDesarrollo() []descriptorMaterialConsumidorV3Desarrollo {
	descriptores := make([]descriptorMaterialConsumidorV3Desarrollo, 0, 4)
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
