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
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"

	"vec-diputacion-granada/internal/shared/telemetria"
)

type relojADMIN struct{}

func (relojADMIN) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

func componerProcesoADMIN(cfg administracion.Configuracion, priv configuracionPerfilesPrivada) (*http.Server, func(), error) {
	return componerProcesoADMINConRuntime(cfg, priv, configuracionRuntimeADMIN{})
}

func componerProcesoADMINConRuntime(cfg administracion.Configuracion, priv configuracionPerfilesPrivada, runtime configuracionRuntimeADMIN) (*http.Server, func(), error) {
	fallo := func(etapa string) (*http.Server, func(), error) { return nil, nil, errorArranque(etapa) }
	// El emisor de la aserción es el espacio de identidad de la sesión y el
	// registro lo compara con éste: si difieren, toda petición acabaría en 403.
	if cfg.EmisorIdentidad != priv.Identidad.EspacioIdentidad {
		return fallo("emisor_identidad")
	}
	if validarConfiguracionPerfilesPrivada(priv) != nil || validarConfiguracionRuntimeADMIN(runtime, priv) != nil || !priv.Identidad.IncluirCuentaOrdinaria {
		return fallo("configuracion")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(priv.TimeoutArranqueSegundos)*time.Second)
	defer cancel()
	rutas := []string{priv.Pools.FuenteAutorizacion, priv.Pools.RegistroAutorizacion, priv.Pools.Motivos, priv.Pools.RegistroSesiones, priv.Pools.RevalidacionSesiones, priv.Pools.CuentasADMIN, priv.Pools.AuditoriaFrontera, runtime.PoolContexto}
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
		telemetria.Instrumentar(pc) // consultas por petición en el registro de acceso
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
	}{{0, "vec_autorizacion_fuente"}, {1, "vec_autorizacion_registro"}, {2, "vec_autorizacion_motivos_evaluador"}, {6, "vec_admin_perfiles_auditoria_ejecutor"}} {
		if acreditarPoolCentral(ctx, pools[capacidad.indice], capacidad.grupo) != nil {
			return fallo("pool_" + strconv.Itoa(capacidad.indice) + "_grupo")
		}
	}
	reloj := relojADMIN{}
	publica, err := base64.StdEncoding.Strict().DecodeString(priv.Firmante.PublicaEsperadaBase64)
	if err != nil {
		return fallo("firmante_publica")
	}
	firmante, cerrarFirmante, err := bootstrap.NuevoFirmanteAtestacionV3DesdeArchivo(bootstrap.ConfiguracionFirmanteAtestacionV3Privado{
		ClaveID: priv.Firmante.ClaveID, Audiencia: priv.Firmante.Audiencia, PrefijoEvidencia: priv.Firmante.PrefijoEvidencia,
		ArchivoSemilla: priv.Firmante.ClavePrivadaArchivo, PublicaEsperada: ed25519.PublicKey(publica)}, reloj)
	if err != nil {
		return fallo("firmante")
	}
	cierres = append(cierres, cerrarFirmante)
	meta, err := decodificarMetadatosConfianzaPerfiles(priv.ConfianzaJSON)
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
	cadena, err := administracion.NuevaConfianzaPerfilesV3(conf, administracion.DependenciasConfianzaPerfilesV3{
		PoolFuente: pools[0], PoolRegistro: pools[1], PoolMotivos: pools[2], CatalogoMotivosID: priv.CatalogoMotivosID, Firmante: firmante,
		Reloj: reloj, Generador: seguridad.GeneradorReferenciasCriptograficas{}, VigenciaDecision: time.Duration(priv.VigenciaDecisionSegundos) * time.Second})
	if err != nil {
		return fallo("confianza_cadena")
	}
	seudonimos, cerrarSeudonimos, err := bootstrap.NuevoSeudonimizadorSesionDesdeArchivo(bootstrap.ConfiguracionSeudonimosSesionPrivada{
		DirectorioMaterial: priv.Identidad.DirectorioMaterial, RutaConfiguracionHMAC: priv.Identidad.RutaConfiguracionHMAC,
		EspacioIdentidad: priv.Identidad.EspacioIdentidad, DominioRef: priv.Identidad.DominioRef, EspacioClave: priv.Identidad.EspacioClave,
		DominioHMAC: priv.Identidad.DominioHMAC, IncluirCuentaOrdinaria: true})
	if err != nil {
		return fallo("seudonimos")
	}
	cierres = append(cierres, cerrarSeudonimos)
	auditor, err := pg.NuevoAuditorFrontera(pools[6])
	if err != nil {
		return fallo("auditor_frontera")
	}
	fuenteIdentificadores, err := selector.NuevaFuenteIdentificadoresADMINDesdeArchivo(runtime.FuenteIdentificadoresArchivo, runtime.FuenteIdentificadoresSHA256)
	if err != nil {
		return fallo("identificadores")
	}
	servidor, err := administracion.ComponerServidorPerfiles(ctx, cfg, administracion.DependenciasComposicionPerfiles{
		Confianza: cadena, PoolCuentas: pools[5], PoolContextoADMIN: pools[7],
		FuenteIdentificadoresADMIN: fuenteIdentificadores, ConfiguracionContextoADMIN: selector.ConfiguracionContextoADMIN{Proceso: runtime.ProcesoContexto}, PoolRegistroSesion: pools[3], PoolRevalidacionSesion: pools[4],
		Seudonimizador: seudonimos, EspacioIdentidad: priv.Identidad.EspacioIdentidad, DominioHMACRef: priv.Identidad.DominioRef,
		Auditor: auditor, Reloj: reloj, Activos: os.DirFS(priv.ActivosDirectorio)})
	if err != nil {
		return fallo("servidor")
	}
	exito = true
	return servidor, cerrar, nil
}

func confianzaDesdeMetadata(m metadatosConfianzaPerfilesPrivados) (administracion.ConfiguracionConfianzaPerfilesV3, error) {
	var cero administracion.ConfiguracionConfianzaPerfilesV3
	publica, err := base64.StdEncoding.Strict().DecodeString(m.Raiz.PublicaBase64)
	if err != nil {
		return cero, administracion.ErrConfiguracion
	}
	c := administracion.ConfiguracionConfianzaPerfilesV3{
		Cabecera: domain.CabeceraAtestacionAutorizacionV3{FormatoVersion: m.Cabecera.FormatoVersion, Suite: m.Cabecera.Suite, ClaveID: m.Cabecera.ClaveID, Audiencia: m.Cabecera.Audiencia},
		Raiz:     administracion.MaterialRaizPerfilesV3{ClaveID: m.Raiz.ClaveID, Audiencia: m.Raiz.Audiencia, Version: m.Raiz.Version, Publica: ed25519.PublicKey(publica), Estado: confianza.EstadoClaveAtestacionAutorizacionV3(m.Raiz.Estado), ValidaDesde: m.Raiz.ValidaDesde, ValidaHasta: m.Raiz.ValidaHasta, RevocadaEn: m.Raiz.RevocadaEn},
		Gobierno: administracion.GobiernoConfianzaPerfilesV3{Revision: m.Gobierno.Revision, HuellaSHA256: m.Gobierno.HuellaSHA256, Secuencia: m.Gobierno.Secuencia, PublicadaEn: m.Gobierno.PublicadaEn, ExpiraEn: m.Gobierno.ExpiraEn}}
	for _, e := range m.EntradasCapacidad {
		b, err := leerArchivoPrivadoPerfiles(e.MaterialArchivo)
		if err != nil {
			for _, x := range c.EntradasCapacidad {
				clear(x.Material)
			}
			return cero, administracion.ErrConfiguracion
		}
		c.EntradasCapacidad = append(c.EntradasCapacidad, administracion.MaterialCapacidadPerfilesV3{
			Audiencia: e.Audiencia, ClaveID: e.ClaveID, EmisorID: e.EmisorID, HuellaGobierno: e.HuellaGobierno, Version: e.Version, RevisionGobierno: e.RevisionGobierno,
			Material: b, ValidaDesde: e.ValidaDesde, ValidaHasta: e.ValidaHasta, Estado: confianza.EstadoClaveHMACCapacidadAtestacionV3(e.Estado), RevocadaEn: e.RevocadaEn})
	}
	return c, nil
}
