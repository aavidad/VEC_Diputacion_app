package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	dietas "vec-diputacion-granada/internal/modules/dietas"
	dietaspg "vec-diputacion-granada/internal/modules/dietas/adapters/postgres"
	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	dp "vec-diputacion-granada/internal/modules/dietas/ports"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	"vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// El manifiesto privado selecciona cuentas/perfiles ya registrados. No crea
// identidades, publica roles ni convierte cargos del certificado en permisos.
type configuracionRutasDietasDesarrollo struct {
	redaccionMaterialRutasDietas
	Version                  int                                `json:"version"`
	Autoridad                string                             `json:"autoridad"`
	Cuentas                  []cuentaRutasDietasDesarrollo      `json:"cuentas"`
	DSNRegistroIdentidad     string                             `json:"dsn_registro_identidad"`
	DSNRevalidacionIdentidad string                             `json:"dsn_revalidacion_identidad"`
	DSNContexto              string                             `json:"dsn_contexto"`
	DSNFuenteAutorizacion    string                             `json:"dsn_fuente_autorizacion"`
	DSNRegistroAutorizacion  string                             `json:"dsn_registro_autorizacion"`
	DSNMotivos               string                             `json:"dsn_motivos"`
	DSNConsumo               string                             `json:"dsn_consumo"`
	MotivoCatalogo           core.ReferenciaEntradaCatalogo     `json:"motivo_catalogo"`
	MotivoCalculo            core.ReferenciaEntradaCatalogo     `json:"motivo_calculo"`
	Material                 datosMaterialRutasDietasDesarrollo `json:"material"`
}

func (configuracionRutasDietasDesarrollo) String() string { return "[CONFIGURACION PRIVADA DIETAS]" }

type cuentaRutasDietasDesarrollo struct {
	CertificadoSHA256 string `json:"certificado_sha256"`
	Sujeto            string `json:"sujeto"`
	CuentaRef         string `json:"cuenta_ref"`
	PerfilRef         string `json:"perfil_ref"`
}
type relojRutasDietas struct{}

func (relojRutasDietas) Ahora() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

// servicioContextoActorDietas compone la resolución de actor de Dietas con el
// alcance {empleado}: Dietas exige el empleado canónico que proyecta Personal
// y deniega sin él o con varios. CT y el resto conservan el alcance vacío.
func servicioContextoActorDietas(resolutor vp.ResolutorRegistroContextoActorV2, reloj vp.Reloj) (*vecapp.ServicioContextoActor, error) {
	alcance, err := core.NuevoAlcanceProyeccionesContextoActor(core.ProyeccionContextoActorEmpleado)
	if err != nil {
		return nil, err
	}
	return vecapp.NuevoServicioContextoActorProductivoV2ConAlcance(resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj, alcance)
}

type servicioAccesoRutasDietas interface {
	AutorizarYConsumirAccesoRutas(context.Context, dp.SolicitudAccesoRutasDietas) (dp.ReciboAccesoRutasDietas, error)
}
type autoridadRutasDietasDesarrollo struct {
	resolvedor                                            *resolvedorIdentidadDesarrollo
	cuentas                                               map[string]cuentaRutasDietasDesarrollo
	registro                                              httpseguridad.RegistroSesiones
	revalidador                                           core.RevalidadorAutenticacionActorV1
	contextos                                             core.ResolutorContextoActorRegistradoV2
	reloj                                                 vp.Reloj
	servicio                                              servicioAccesoRutasDietas
	instancia, ambito, grafo, huellaGrafo, huellaCatalogo string
	motivoCatalogo, motivoCalculo                         core.ReferenciaEntradaCatalogo
}

// Cápsula privada de una sola petición autenticada. No tiene representación
// transportable y una copia conserva el mismo estado de consumo.
type capsulaRutasDietasDesarrollo struct {
	autoridad *autoridadRutasDietasDesarrollo
	peticion  *http.Request
	cuenta    cuentaRutasDietasDesarrollo
	instante  time.Time
	consumida atomic.Bool
}

// Ausencia mantiene sólo estas rutas en 503. Un manifiesto presente e inválido
// impide arrancar. Provisiones y gobierno pertenecen al integrador autorizado.
func nuevasRutasDietasDesarrollo(cfg config.Config, resolvedor httpapi.DemoIdentityResolver, derivador *derivadorIdentidadOperacionDesarrollo) (httpapi.AutoridadPeticionRutasDietas, func(), error) {
	vacio := func() {}
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "dietas-rutas.json")
	if _, e := os.Lstat(ruta); errors.Is(e, os.ErrNotExist) {
		return nil, vacio, nil
	} else if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	if !cfg.DevelopmentEnabledByDoubleKey() {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	identidad, ok := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !ok || identidad == nil || derivador == nil || !derivador.valido() {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	contenido, e := leerFicheroMaterialSeguro(ruta, 128<<10)
	if e != nil || validarClavesJSONUnicas(contenido) != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	defer borrarBytes(contenido)
	var c configuracionRutasDietasDesarrollo
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa ||
		len(c.Cuentas) == 0 || len(c.Cuentas) > 64 || !core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoCatalogo) ||
		!core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoCalculo) || c.MotivoCatalogo.CatalogoID != c.MotivoCalculo.CatalogoID ||
		cfg.OSRMGraphVersion == "" || cfg.OSRMScopeName == "" || cfg.OSRMScopeBounds == "" {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	defer c.Material.borrarCopiasEfimeras()
	cuentas := map[string]cuentaRutasDietasDesarrollo{}
	for _, cuenta := range c.Cuentas {
		huella, err := hex.DecodeString(cuenta.CertificadoSHA256)
		if err != nil || len(huella) != 32 || hex.EncodeToString(huella) != cuenta.CertificadoSHA256 || cuenta.Sujeto == "" || cuenta.CuentaRef == "" || cuenta.PerfilRef == "" {
			return nil, vacio, httpapi.ErrRutaDietasNoDisponible
		}
		var digest [32]byte
		copy(digest[:], huella)
		principal, existe := identidad.porHuella[digest]
		if !existe || principal.ID != cuenta.Sujeto || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh {
			return nil, vacio, httpapi.ErrRutaDietasNoDisponible
		}
		if _, repetida := cuentas[cuenta.CertificadoSHA256]; repetida {
			return nil, vacio, httpapi.ErrRutaDietasNoDisponible
		}
		cuentas[cuenta.CertificadoSHA256] = cuenta
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	var pools []*pgxpool.Pool
	var cerrarMaterial func()
	var unaVez sync.Once
	cerrar := func() {
		unaVez.Do(func() {
			if cerrarMaterial != nil {
				cerrarMaterial()
			}
			for _, p := range pools {
				p.Close()
			}
		})
	}
	completa := false
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	entradas := []struct{ dsn, rol string }{
		{c.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"}, {c.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"},
		{c.DSNContexto, "vec_contexto_actor_v1_runtime"}, {c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"},
		{c.DSNRegistroAutorizacion, "vec_autorizacion_registro"}, {c.DSNMotivos, "vec_autorizacion_motivos_evaluador"}, {c.DSNConsumo, "vec_dietas_ejecutor"},
	}
	usuarios := map[string]bool{}
	for _, entrada := range entradas {
		pool, usuario, err := abrirPoolRutasDietas(ctx, entrada.dsn, entrada.rol)
		if err != nil {
			return nil, vacio, httpapi.ErrRutaDietasNoDisponible
		}
		pools = append(pools, pool)
		if usuarios[usuario] {
			return nil, vacio, httpapi.ErrRutaDietasNoDisponible
		}
		usuarios[usuario] = true
	}
	registro, e := identidadpg.NuevoRegistroSesionesPostgreSQL(ctx, pools[0], pools[1], &seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	revalidador, e := identidadpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, pools[1])
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	resolutor, e := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pools[2])
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	reloj := relojRutasDietas{}
	servicioContexto, e := servicioContextoActorDietas(resolutor, reloj)
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	contextos, e := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	fuente, e := vecpg.NuevoAlmacenAutorizacion(pools[3])
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	registroAutorizacion, e := vecpg.NuevoAlmacenAutorizacion(pools[4])
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	motivos, e := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools[5], c.MotivoCatalogo.CatalogoID)
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	autorizador, e := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registroAutorizacion, registroAutorizacion, motivos, reloj, seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	material, e := nuevoMaterialAtestacionRutasDietasDesarrollo(c.Material)
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	defer material.borrarCopiasEfimeras()
	proveedor, e := nuevoProveedorMaterialAccesoRutasDietasDesarrollo(material, reloj)
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	cerrarMaterial = proveedor.Cerrar
	consumo, e := dietaspg.NuevoRepositorioAccesoRutas(pools[6])
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	servicio, e := dietasapp.NuevoServicioAutorizacionRutasDietas(autorizador, proveedor, consumo)
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	nonce, e := nonceRutasDietas()
	if e != nil {
		return nil, vacio, e
	}
	catalogo, e := json.Marshal(struct {
		Puntos []map[string]any
		Matriz map[string]any
	}{dietas.ProvinceRoutePointMaps(), dietas.ProvinceRouteMatrixStatus()})
	if e != nil {
		return nil, vacio, httpapi.ErrRutaDietasNoDisponible
	}
	a := &autoridadRutasDietasDesarrollo{resolvedor: identidad, cuentas: cuentas, registro: registro, revalidador: revalidador, contextos: contextos, reloj: reloj, servicio: servicio, instancia: nonce, ambito: cfg.OSRMScopeName, grafo: cfg.OSRMGraphVersion, huellaGrafo: huellaRutasDietas(cfg.OSRMGraphVersion + "|" + cfg.OSRMScopeName + "|" + cfg.OSRMScopeBounds), huellaCatalogo: huellaRutasDietas(string(catalogo)), motivoCatalogo: c.MotivoCatalogo, motivoCalculo: c.MotivoCalculo}
	completa = true
	return a, cerrar, nil
}

func abrirPoolRutasDietas(ctx context.Context, dsn, rol string) (*pgxpool.Pool, string, error) {
	fallo := httpapi.ErrRutaDietasNoDisponible
	if dsn == "" {
		return nil, "", fallo
	}
	c, e := pgxpool.ParseConfig(dsn)
	if e != nil || c.ConnConfig.User == "" || validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, "", fallo
	}
	c.MaxConns = 2
	c.MinConns = 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{"application_name": "vec-dietas-rutas-desarrollo", "timezone": "UTC", "search_path": "pg_catalog", "statement_timeout": "10s", "lock_timeout": "2s", "idle_in_transaction_session_timeout": "15s"} {
		c.ConnConfig.RuntimeParams[k] = v
	}
	pool, e := pgxpool.NewWithConfig(ctx, c)
	if e != nil {
		return nil, "", fallo
	}
	var usuario string
	var valido bool
	e = pool.QueryRow(ctx, `SELECT session_user::text,
 session_user=current_user AND r.rolcanlogin AND r.rolinherit AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid WHERE m.member=r.oid AND g.rolname=$1 AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option AND g.rolinherit = (g.rolname IN ('vec_autorizacion_fuente','vec_autorizacion_registro','vec_dietas_ejecutor')) AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
 AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members superior WHERE superior.member=g.oid))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`, rol).Scan(&usuario, &valido)
	if e != nil || !valido {
		pool.Close()
		return nil, "", fallo
	}
	return pool, usuario, nil
}
func nonceRutasDietas() (string, error) {
	var b [32]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", httpapi.ErrRutaDietasNoDisponible
	}
	return hex.EncodeToString(b[:]), nil
}
func huellaRutasDietas(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (a *autoridadRutasDietasDesarrollo) AutorizarPeticionRutaDietas(ctx context.Context, r *http.Request) error {
	if a == nil || ctx == nil || ctx.Err() != nil || r == nil || r.URL == nil || a.resolvedor == nil || a.servicio == nil {
		return httpapi.ErrRutaDietasNoDisponible
	}
	if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
		return httpapi.ErrRutaDietasNoAutenticada
	}
	principal, e := a.resolvedor.ResolveDemoIdentity(ctx, r)
	if e != nil || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh {
		return httpapi.ErrRutaDietasNoAutenticada
	}
	ahora := a.reloj.Ahora()
	cert := r.TLS.VerifiedChains[0][0]
	if ahora.Before(cert.NotBefore) || !ahora.Before(cert.NotAfter) {
		return httpapi.ErrRutaDietasNoAutenticada
	}
	cuenta, ok := a.cuentas[principal.Attributes["certificate_sha256"]]
	if !ok || cuenta.Sujeto != principal.ID {
		return httpapi.ErrRutaDietasDenegada
	}
	accion, tipo, version, huella, motivo := dp.AccionConsultarCatalogoRutasDietas, dp.TipoCatalogoRutasDietas, "sha256:"+a.huellaCatalogo, a.huellaCatalogo, a.motivoCatalogo
	if r.URL.Path == "/api/vec/dietas/road-route" && r.Method == http.MethodPost {
		accion, tipo, version, huella, motivo = dp.AccionSolicitarCalculoRutasDietas, dp.TipoCalculoRutasDietas, a.grafo, a.huellaGrafo, a.motivoCalculo
	} else if r.URL.Path != "/api/vec/dietas/route-catalog" || r.Method != http.MethodGet {
		return httpapi.ErrRutaDietasDenegada
	}
	vinculo, resultado, e := a.resolverSesion(ctx, r, &capsulaRutasDietasDesarrollo{autoridad: a, peticion: r, cuenta: cuenta, instante: ahora})
	if e != nil {
		return e
	}
	correlacion, e := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if e != nil {
		return httpapi.ErrRutaDietasNoDisponible
	}
	ref, e := correlacion.ValorCanonico()
	if e != nil {
		return httpapi.ErrRutaDietasNoDisponible
	}
	canal := huellaRutasDietas(a.instancia + "|" + cuenta.CertificadoSHA256)
	recurso := core.RecursoAutorizable{Referencia: "dietas:rutas:" + tipo, ModuloID: "dietas", Tipo: tipo, Ambitos: map[string]string{"ambito_ref": a.ambito}, Atributos: map[string]string{"version": version, "huella_sha256": huella, "canal": canal, "instancia": a.instancia, "ruta": r.URL.Path, "metodo": r.Method, "correlacion_ref": ref}}
	solicitud := dp.SolicitudAccesoRutasDietas{ResultadoContexto: resultado, Vinculo: vinculo, Accion: accion, Recurso: recurso, Audiencia: dp.AudienciaAccesoRutasDietas, Finalidad: dp.FinalidadConsultarItinerarioDietas, Correlacion: correlacion, ReferenciaMotivo: motivo}
	if !vinculo.VigenteEn(a.reloj.Ahora(), resultado) {
		return httpapi.ErrRutaDietasDenegada
	}
	_, e = a.servicio.AutorizarYConsumirAccesoRutas(ctx, solicitud)
	if e != nil {
		if errors.Is(e, dp.ErrAccesoRutasDietasDenegado) {
			return httpapi.ErrRutaDietasDenegada
		}
		return httpapi.ErrRutaDietasNoDisponible
	}
	return nil
}

func (a *autoridadRutasDietasDesarrollo) resolverSesion(ctx context.Context, r *http.Request, capsula *capsulaRutasDietasDesarrollo) (core.VinculoAutenticacionActorV2, core.ResultadoContextoActorRegistradoV2, error) {
	vacio := core.VinculoAutenticacionActorV2{}
	resultadoVacio := core.ResultadoContextoActorRegistradoV2{}
	if capsula == nil || capsula.autoridad != a || capsula.peticion != r || !capsula.consumida.CompareAndSwap(false, true) {
		return vacio, resultadoVacio, httpapi.ErrRutaDietasDenegada
	}
	cuenta, ahora := capsula.cuenta, capsula.instante
	asercion, e := nonceRutasDietas()
	if e != nil {
		return vacio, resultadoVacio, e
	}
	sesion, e := nonceRutasDietas()
	if e != nil {
		return vacio, resultadoVacio, e
	}
	hasta := ahora.Add(2 * time.Minute)
	if limite := r.TLS.VerifiedChains[0][0].NotAfter.UTC().Truncate(time.Microsecond); limite.Before(hasta) {
		hasta = limite
	}
	politica := "dev-certificado-mtls-v1;solo-sintetico;canal-privado-validado;garantia-alta-desarrollo;vigencia-120s;sin-kerberos;no-corporativa"
	alta := httpseguridad.AltaSesionAtomica{AsercionID: asercion, SesionID: sesion, SujetoID: cuenta.Sujeto, CuentaID: "desarrollo:" + cuenta.CuentaRef, Superficie: httpseguridad.SuperficieInternaCorporativa, EspacioIdentidad: espacioIdentidadSesionDesarrollo, MetodoObservado: core.AuthMethodCertificate, GarantiaObservada: core.AuthAssuranceHigh, AutenticacionVerificadaEn: ahora, SesionEmitidaEn: ahora, AsercionExpiraEn: hasta, PoliticaGarantiaRef: referenciaAltaContratacionTemporalDesarrollo("pga_", "dev-certificado-mtls-v1"), PoliticaGarantiaHuellaSHA256: huellaRutasDietas(politica), AutenticacionHuellaSHA256: huellaRutasDietas(a.instancia + "|" + asercion + "|" + sesion + "|" + cuenta.CertificadoSHA256 + "|" + cuenta.CuentaRef + "|" + cuenta.PerfilRef + "|" + r.URL.Path + "|" + r.Method + "|" + ahora.Format(time.RFC3339Nano))}
	confirmacion, e := a.registro.ConsumirAsercionYRegistrar(ctx, alta)
	if e != nil || confirmacion.ValidarPara(alta) != nil || confirmacion.CuentaRef != cuenta.CuentaRef {
		return vacio, resultadoVacio, httpapi.ErrRutaDietasNoDisponible
	}
	// Decorador común: cotejo exacto de la sesión central recién registrada.
	// No recibe soporte, autorización ni perfiles de Contratación.
	revalidador := revalidadorSesionConsultaRRHHDesarrollo{delegado: a.revalidador, alta: alta, confirmacion: confirmacion, reloj: a.reloj, superficie: httpseguridad.SuperficieInternaCorporativa}
	vinculo, resultado, e := core.CrearVinculoAutenticacionActorV2ConResultado(ctx, revalidador, core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: confirmacion.AutenticacionRef, SesionRef: confirmacion.SesionRef}, a.contextos, core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{CuentaRef: cuenta.CuentaRef, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}, PerfilActivoRef: cuenta.PerfilRef}, a.reloj)
	if e != nil {
		return vacio, resultadoVacio, httpapi.ErrRutaDietasDenegada
	}
	datos, e := vinculo.Datos()
	if e != nil || datos.CuentaRef != cuenta.CuentaRef || datos.PerfilActivoRef != cuenta.PerfilRef || datos.CuentaPrivilegiada || datos.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 || !vinculo.VigenteEn(a.reloj.Ahora(), resultado) {
		return vacio, resultadoVacio, httpapi.ErrRutaDietasDenegada
	}
	return vinculo, resultado, nil
}
