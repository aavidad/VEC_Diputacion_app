package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/app/administracion"
	"vec-diputacion-granada/internal/app/bootstrap"
	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	selector "vec-diputacion-granada/internal/vec/adapters/httpseguridad/adminperfiles"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	usuariosPG "vec-diputacion-granada/internal/vec/adapters/usuariosadministrables/postgres"
)

// Este montaje sólo existe con el overlay metadatos_v1 privado. Cada pool
// mantiene un LOGIN distinto. No construye autoridades de actos ni usa la
// función histórica de auditoría de frontera que nunca se instaló.
func componerProcesoUsuariosMetadatosADMIN(cfg administracion.Configuracion, base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada) (*http.Server, func(), error) {
	return componerProcesoUsuariosMetadatosADMINConRuntime(cfg, base, u, configuracionRuntimeADMIN{})
}

func componerProcesoUsuariosMetadatosADMINConRuntime(cfg administracion.Configuracion, base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN) (*http.Server, func(), error) {
	return componerProcesoUsuariosMetadatosADMINConLote(cfg, base, u, runtime, nil, nil)
}

// componerProcesoUsuariosMetadatosADMINConLote añade, si hay overlay del lote,
// su pool y LOGIN propios, su cadena V3 de una capacidad, el emisor, la
// autoridad PostgreSQL y el servicio de aplicación del lote. Sin overlay, el
// proceso es exactamente el de las lecturas de usuarios.
func componerProcesoUsuariosMetadatosADMINConLote(cfg administracion.Configuracion, base configuracionPerfilesPrivada, u configuracionUsuariosMetadatosPrivada, runtime configuracionRuntimeADMIN, lote *configuracionLotePrivada, plan *configuracionPlanFirmaPrivada) (*http.Server, func(), error) {
	fallo := func(etapa string) (*http.Server, func(), error) { return nil, nil, errorArranque(etapa) }
	// El emisor de la aserción es el espacio de identidad de la sesión y el
	// registro lo compara con éste: si difieren, toda petición acabaría en 403.
	if cfg.EmisorIdentidad != base.Identidad.EspacioIdentidad {
		return fallo("emisor_identidad")
	}
	if validarConfiguracionPerfilesPrivada(base) != nil || validarConfiguracionUsuariosMetadatosPrivada(u, base) != nil || validarConfiguracionRuntimeADMIN(runtime, base) != nil || runtime.PoolContexto == u.PoolLector || runtime.PoolContexto == u.PoolIntentos || runtime.PoolContexto == u.PoolSelector || runtime.PoolContexto == u.PoolFronteraTecnica || !base.Identidad.IncluirCuentaOrdinaria {
		return fallo("configuracion")
	}
	if lote != nil && validarConfiguracionLotePrivada(*lote, base, u, runtime) != nil {
		return fallo("lote_configuracion")
	}
	if plan != nil && (lote == nil || validarConfiguracionPlanFirmaPrivada(*plan, *lote, base, u, runtime) != nil) {
		return fallo("plan_firma_configuracion")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(base.TimeoutArranqueSegundos)*time.Second)
	defer cancel()
	rutas := []string{base.Pools.FuenteAutorizacion, base.Pools.RegistroAutorizacion, base.Pools.Motivos, base.Pools.RegistroSesiones, base.Pools.RevalidacionSesiones, base.Pools.CuentasADMIN, u.PoolLector, u.PoolIntentos, u.PoolSelector, u.PoolFronteraTecnica, runtime.PoolContexto}
	if lote != nil {
		rutas = append(rutas, lote.PoolLote) // índice 11
		if plan != nil {
			rutas = append(rutas, plan.Pool) // índice 12
		}
	}
	pools := make([]*pgxpool.Pool, 0, len(rutas))
	cierres := []func(){}
	cerrar := func() {
		for i := len(cierres) - 1; i >= 0; i-- {
			cierres[i]()
		}
		for _, p := range pools {
			p.Close()
		}
	}
	exito := false
	defer func() {
		if !exito {
			cerrar()
		}
	}()
	logins := map[string]bool{}
	for i, ruta := range rutas {
		dsn, err := cargarDSNPrivado(ruta)
		if err != nil {
			return fallo("pool_" + strconv.Itoa(i) + "_dsn")
		}
		pc, err := configurarPoolADMIN(dsn)
		if err != nil {
			return fallo("pool_" + strconv.Itoa(i) + "_config")
		}
		pool, err := pgxpool.NewWithConfig(ctx, pc)
		if err != nil {
			return fallo("pool_" + strconv.Itoa(i) + "_abrir")
		}
		pools = append(pools, pool)
		var login string
		var nominal bool
		if pool.QueryRow(ctx, `SELECT session_user,current_user=session_user AND r.rolcanlogin AND NOT(r.rolsuper OR r.rolcreaterole OR r.rolcreatedb OR r.rolreplication OR r.rolbypassrls) FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`).Scan(&login, &nominal) != nil || !nominal || logins[login] {
			return fallo("pool_" + strconv.Itoa(i) + "_login")
		}
		logins[login] = true
	}
	for _, capacidad := range []struct {
		indice int
		grupo  string
	}{
		{0, "vec_autorizacion_fuente"}, {1, "vec_autorizacion_registro"}, {2, "vec_autorizacion_motivos_evaluador"},
		{6, "vec_admin_usuarios_lector"}, {7, "vec_autorizacion_atestada_v3_registrador_intentos"}, {8, "vec_identidad_sesiones_v1_admin_preperfil"}, {9, "vec_admin_frontera_tecnica_ejecutor"},
	} {
		if acreditarPoolCentral(ctx, pools[capacidad.indice], capacidad.grupo) != nil {
			return fallo("pool_" + strconv.Itoa(capacidad.indice) + "_grupo")
		}
	}
	if lote != nil && acreditarPoolCentral(ctx, pools[11], "vec_admin_perfiles_lote_ejecutor") != nil {
		return fallo("pool_11_grupo")
	}
	reloj := relojADMIN{}
	publica, err := base64.StdEncoding.Strict().DecodeString(base.Firmante.PublicaEsperadaBase64)
	if err != nil {
		return fallo("firmante_publica")
	}
	firmante, cerrarFirmante, err := bootstrap.NuevoFirmanteAtestacionV3DesdeArchivo(bootstrap.ConfiguracionFirmanteAtestacionV3Privado{
		ClaveID: base.Firmante.ClaveID, Audiencia: base.Firmante.Audiencia, PrefijoEvidencia: base.Firmante.PrefijoEvidencia,
		ArchivoSemilla: base.Firmante.ClavePrivadaArchivo, PublicaEsperada: ed25519.PublicKey(publica)}, reloj)
	if err != nil {
		return fallo("firmante")
	}
	cierres = append(cierres, cerrarFirmante)
	meta, err := decodificarMetadatosConfianzaPerfiles(u.ConfianzaJSON)
	if err != nil {
		return fallo("confianza_metadatos")
	}
	conf, err := confianzaDesdeMetadata(meta)
	if err != nil {
		return fallo("confianza_material")
	}
	defer func() {
		for i := range conf.EntradasCapacidad {
			clear(conf.EntradasCapacidad[i].Material)
		}
	}()
	cadena, err := administracion.NuevaConfianzaUsuariosV3(conf, administracion.DependenciasConfianzaUsuariosV3{
		PoolFuente: pools[0], PoolRegistro: pools[1], PoolMotivos: pools[2], CatalogoMotivosID: base.CatalogoMotivosID, Firmante: firmante,
		Reloj: reloj, Generador: seguridad.GeneradorReferenciasCriptograficas{}, VigenciaDecision: time.Duration(base.VigenciaDecisionSegundos) * time.Second})
	if err != nil {
		return fallo("confianza_cadena")
	}
	emisor, err := administracion.NuevoEmisorUsuarios(cadena.Emisores, u.MotivosUsuarios, reloj)
	if err != nil {
		return fallo("emisor_usuarios")
	}
	registrador, err := vecpg.NuevoRegistradorIntentosAuditoriaPostgreSQL(pools[7], u.Proceso, u.Canal, u.plazoAuditoria())
	if err != nil || registrador.PreflightIntentoAuditoria(ctx) != nil {
		return fallo("auditoria_intentos")
	}
	auditorNominal, err := pg.NuevaAuditorFronteraNominal(registrador, pg.ConfiguracionAuditoriaFronteraNominal{
		Proceso: u.Proceso, Canal: u.Canal, MotivoDenegado: u.MotivoDenegado, MotivoError: u.MotivoError,
		Plazo: u.plazoAuditoria(), Destinos: u.destinosAuditoria()})
	if err != nil {
		return fallo("auditoria_nominal")
	}
	tecnico, err := pg.NuevoRegistradorFronteraTecnica(ctx, pools[9], pg.ConfiguracionFronteraTecnica{
		Proceso: u.Proceso, Canal: u.Canal, Plazo: u.plazoAuditoria()})
	if err != nil {
		return fallo("frontera_tecnica")
	}
	auditor, err := pg.NuevoAuditorFronteraCompuesto(auditorNominal, tecnico)
	if err != nil {
		return fallo("auditor_compuesto")
	}
	fuente, err := usuariosPG.Nueva(ctx, pools[6], emisor, cadena.Fuente, registrador, reloj, usuariosPG.Configuracion{
		OrganizacionRef: u.OrganizacionRef, UnidadRef: u.UnidadRef, Proceso: u.Proceso, Canal: u.Canal,
		MotivoDenegado: u.MotivoDenegado, MotivoError: u.MotivoError})
	if err != nil {
		return fallo("lector_usuarios")
	}
	lecturas, err := administracion.NuevaFuenteLecturasUsuariosMetadatos(fuente)
	if err != nil {
		return fallo("lecturas_usuarios")
	}
	seleccion, err := selector.NuevaSeleccionAuditadaPostgreSQL(ctx, pools[8], reloj)
	if err != nil {
		return fallo("selector")
	}
	seudonimos, cerrarSeudonimos, err := bootstrap.NuevoSeudonimizadorSesionDesdeArchivo(bootstrap.ConfiguracionSeudonimosSesionPrivada{
		DirectorioMaterial: base.Identidad.DirectorioMaterial, RutaConfiguracionHMAC: base.Identidad.RutaConfiguracionHMAC,
		EspacioIdentidad: base.Identidad.EspacioIdentidad, DominioRef: base.Identidad.DominioRef, EspacioClave: base.Identidad.EspacioClave,
		DominioHMAC: base.Identidad.DominioHMAC, IncluirCuentaOrdinaria: true})
	if err != nil {
		return fallo("seudonimos")
	}
	cierres = append(cierres, cerrarSeudonimos)
	fuenteIdentificadores, err := selector.NuevaFuenteIdentificadoresADMINDesdeArchivo(runtime.FuenteIdentificadoresArchivo, runtime.FuenteIdentificadoresSHA256)
	if err != nil {
		return fallo("identificadores")
	}
	var autoridadLote *administracion.LoteADMIN
	if lote != nil {
		autoridadLote, err = componerLoteADMIN(ctx, base, u, *lote, pools[0], pools[1], pools[2], pools[11], firmante, registrador, reloj)
		if err != nil {
			return nil, nil, err
		}
		if plan != nil {
			gobierno, err := componerGobiernoPlanFirmaADMIN(ctx, base, u, *plan, pools[0], pools[1], pools[2], pools[12], firmante, registrador, reloj)
			if err != nil {
				return nil, nil, err
			}
			autoridadLote.GobiernoPlan = gobierno
		}
	}
	servidor, err := administracion.ComponerServidorPerfiles(ctx, cfg, administracion.DependenciasComposicionPerfiles{
		Confianza: cadena, PoolCuentas: pools[5], PoolContextoADMIN: pools[10],
		FuenteIdentificadoresADMIN: fuenteIdentificadores, ConfiguracionContextoADMIN: selector.ConfiguracionContextoADMIN{Proceso: runtime.ProcesoContexto}, PoolRegistroSesion: pools[3], PoolRevalidacionSesion: pools[4],
		Seudonimizador: seudonimos, EspacioIdentidad: base.Identidad.EspacioIdentidad, DominioHMACRef: base.Identidad.DominioRef,
		Lecturas: lecturas, FuenteSeleccion: seleccion, Auditor: auditor, Reloj: reloj, Activos: os.DirFS(base.ActivosDirectorio), SoloUsuariosMetadatos: true, Lote: autoridadLote})
	if err != nil {
		return fallo("servidor")
	}
	exito = true
	return servidor, cerrar, nil
}
