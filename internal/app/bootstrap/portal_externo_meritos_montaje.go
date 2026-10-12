package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	meritoshttp "vec-diputacion-granada/internal/modules/meritos/adapters/http"
	meritospg "vec-diputacion-granada/internal/modules/meritos/adapters/postgres"
	meritosapp "vec-diputacion-granada/internal/modules/meritos/application"
	meritosports "vec-diputacion-granada/internal/modules/meritos/ports"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var errMontajeCarteraExterna = errors.New("bootstrap: cartera de meritos externa no disponible")

func falloMontajeCartera(clave string) error {
	return fmt.Errorf("%w: clave=%s", errMontajeCarteraExterna, clave)
}

const rutaMotivoCarteraExterna = "identidad/meritos-cartera-externa.json"

type configuracionMotivoCarteraExterna struct {
	Version           int                           `json:"version"`
	MotivoRetirada    vec.ReferenciaEntradaCatalogo `json:"motivo_retirada"`
	MotivoDeclaracion vec.ReferenciaEntradaCatalogo `json:"motivo_declaracion,omitempty"`
	Cuentas           []cuentaCarteraExterna        `json:"cuentas"`
}

type cuentaCarteraExterna struct {
	CertificadoSHA256 string `json:"certificado_sha256"`
	CuentaRef         string `json:"cuenta_ref"`
	PerfilRef         string `json:"perfil_ref"`
}

var (
	huellaCertificadoCarteraExterna = regexp.MustCompile(`^[a-f0-9]{64}$`)
	cuentaCarteraExternaValida      = regexp.MustCompile(`^cta_[A-Za-z0-9_-]{22,128}$`)
	perfilCarteraExternaValido      = regexp.MustCompile(`^prf_[A-Za-z0-9_-]{22,128}$`)
)

// La referencia técnica se publica en el catálogo de Autorización y viaja en
// material privado 0600. Ninguna clave de motivo se decide en código.
func leerConfiguracionCarteraExterna(directorio string) (configuracionMotivoCarteraExterna, error) {
	vacio := configuracionMotivoCarteraExterna{}
	b, err := leerFicheroMaterialSeguro(filepath.Join(directorio, rutaMotivoCarteraExterna), 128<<10)
	if err != nil || validarClavesJSONUnicas(b) != nil {
		return vacio, errMontajeCarteraExterna
	}
	defer borrarBytes(b)
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var c configuracionMotivoCarteraExterna
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 ||
		!vec.ReferenciaMotivoAutorizacionV2Valida(c.MotivoRetirada) ||
		c.MotivoRetirada.CatalogoID != "motivos_autorizacion_meritos_cartera" ||
		len(c.Cuentas) < 1 || len(c.Cuentas) > 64 {
		return vacio, errMontajeCarteraExterna
	}
	for _, cuenta := range c.Cuentas {
		if !huellaCertificadoCarteraExterna.MatchString(cuenta.CertificadoSHA256) ||
			!cuentaCarteraExternaValida.MatchString(cuenta.CuentaRef) ||
			!perfilCarteraExternaValido.MatchString(cuenta.PerfilRef) {
			return vacio, errMontajeCarteraExterna
		}
	}
	return c, nil
}

// La misma hoja TLS identifica a la persona; M4 exige otra cuenta y otro
// perfil CA15 aprobados. El fichero no añade autorización a ninguno de los dos.
func cuentasNominalesCarteraExterna(usuarios map[string]cuentaUsuariosPreferenciasDesarrollo,
	configuradas []cuentaCarteraExterna,
) (map[string]cuentaUsuariosPreferenciasDesarrollo, error) {
	if len(usuarios) == 0 || len(configuradas) == 0 || len(configuradas) > len(usuarios) {
		return nil, errMontajeCarteraExterna
	}
	reservados := map[string]bool{}
	for _, cuenta := range usuarios {
		reservados[cuenta.CuentaRef] = true
		reservados[cuenta.PerfilRef] = true
	}
	salida := make(map[string]cuentaUsuariosPreferenciasDesarrollo, len(configuradas))
	for _, configurada := range configuradas {
		origen, existe := usuarios[configurada.CertificadoSHA256]
		if !existe || salida[configurada.CertificadoSHA256].CuentaRef != "" ||
			reservados[configurada.CuentaRef] || reservados[configurada.PerfilRef] ||
			configurada.CuentaRef == configurada.PerfilRef {
			return nil, errMontajeCarteraExterna
		}
		origen.CuentaRef, origen.PerfilRef = configurada.CuentaRef, configurada.PerfilRef
		salida[configurada.CertificadoSHA256] = origen
		reservados[configurada.CuentaRef], reservados[configurada.PerfilRef] = true, true
	}
	return salida, nil
}

// montarCarteraPortalExterno comparte la hoja certificada ya reconocida por
// el portal. Identidad externa y CA15 resuelven una cuenta/perfil M4 propios.
// Sólo publica ambas rutas cuando todo su material y topología están listos.
func montarCarteraPortalExterno(ctx context.Context, cfg config.Config,
	externa *autoridadPreferenciasUsuariosDesarrollo, derivador *derivadorIdentidadOperacionDesarrollo,
	preflight *pgxpool.Pool,
) (func(), error) {
	if ctx == nil || externa == nil || externa.base == nil || externa.superficie != vec.SuperficieAutenticacionExternaPersonalV1 ||
		derivador == nil || !derivador.valido() || preflight == nil {
		return nil, falloMontajeCartera("dependencia")
	}
	datosDSN, fronteraDSN, fuenteDSN, registroDSN, motivosDSN, contextoDSN, err := cfg.DSNMeritosCarteraSeparados()
	if err != nil {
		return nil, falloMontajeCartera("configuracion_dsn")
	}
	privada, err := leerConfiguracionCarteraExterna(cfg.DevelopmentMaterialDir)
	if err != nil {
		return nil, falloMontajeCartera("motivo_privado")
	}
	cuentas, err := cuentasNominalesCarteraExterna(externa.cuentas, privada.Cuentas)
	if err != nil {
		return nil, falloMontajeCartera("cuentas_privadas")
	}
	motivo := privada.MotivoRetirada
	proveedores, err := nuevosProveedoresV3PortalExterno(ctx, cfg.DevelopmentMaterialDir, preflight,
		consumidorMeritosCarteraPortalExternoV3, relojContratacionTemporalDesarrollo{})
	if err != nil || proveedores[meritosports.AudienciaRetirarCartera] == nil || proveedores[descriptoresMaterialMeritosInternosDesarrollo()[0].Audiencia] == nil {
		return nil, falloMontajeCartera("material_v3")
	}
	topologia, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, preflight)
	if err != nil {
		return nil, falloMontajeCartera("topologia_preflight")
	}
	entradas := [...]struct {
		dsn, login, rol string
		funciones       []string
	}{
		{datosDSN, "vec_externo_meritos_cartera", "vec_meritos_cartera_externo", []string{
			"vec_meritos.listar_cartera_propia_v1(bytea,bytea,jsonb)",
			"vec_meritos.retirar_hecho_propio_v1(bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)",
			"vec_autorizacion_atestada_v3.registrar_intento_lectura_cartera_meritos_v1(bytea,bytea,jsonb)",
			"vec_autorizacion_atestada_v3.registrar_intento_retirada_cartera_meritos_v1(bytea,bytea,jsonb)",
		}},
		{fronteraDSN, "vec_externo_meritos_frontera", "vec_meritos_frontera_externo", []string{
			"vec_meritos.registrar_denegacion_frontera_cartera_v1(text,text,text,text,text)",
		}},
		{fuenteDSN, "vec_externo_meritos_fuente_autorizacion", "vec_autorizacion_fuente_meritos_externa", []string{
			"vec_autorizacion.obtener_instantanea_meritos_externo_v1(text,text)",
		}},
		{registroDSN, "vec_externo_meritos_registro_autorizacion", "vec_autorizacion_registro_meritos_externo", []string{
			"vec_autorizacion.registrar_y_revalidar_decision_meritos_externo_v3(bytea,bytea,numeric,numeric)",
		}},
		{motivosDSN, "vec_externo_meritos_motivos_autorizacion", "vec_autorizacion_motivos_meritos_externo", []string{
			"vec_autorizacion.resolver_motivo_meritos_externo_v1(text,integer,text,text,timestamptz)",
		}},
		{contextoDSN, "vec_externo_meritos_contexto", "vec_contexto_actor_v1_usuarios_externo", []string{
			"vec_contexto_actor_v1.acreditar_runtime_usuarios_externo_v1()",
			"vec_contexto_actor_v1.resolver_contexto_usuarios_externo_v1(text,text,text,text,timestamptz)",
			"vec_contexto_actor_v1.reconciliar_contexto_usuarios_externo_v1(text,text,text,text,timestamptz)",
		}},
	}
	var pools []*pgxpool.Pool
	var unaVez sync.Once
	cerrar := func() {
		unaVez.Do(func() {
			for _, pool := range pools {
				pool.Close()
			}
		})
	}
	completo := false
	defer func() {
		if !completo {
			cerrar()
		}
	}()
	logins := map[string]bool{}
	for indice, entrada := range entradas {
		pool, login, fallo := abrirPoolNominalPortalExterno(ctx, entrada.dsn, entrada.rol, "vec-portal-externo-meritos-cartera")
		if fallo != nil {
			return nil, falloMontajeCartera(fmt.Sprintf("pool_%d", indice))
		}
		pools = append(pools, pool)
		if login != entrada.login || logins[login] || externa.logins[login] ||
			acreditarFuncionesCarteraExterna(ctx, pool, entrada.funciones) != nil ||
			cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil {
			return nil, falloMontajeCartera(fmt.Sprintf("login_o_funciones_%d", indice))
		}
		logins[login] = true
	}
	for indice, entrada := range [...]struct {
		configuracion config.ConfiguracionPostgreSQLExterna
		rol           string
	}{
		{cfg.ExternoIdentidadRegistroPostgreSQL, "vec_identidad_externa_v1_registrador"},
		{cfg.ExternoIdentidadRevalidacionPostgreSQL, "vec_identidad_externa_v1_revalidador"},
	} {
		dsn, e := entrada.configuracion.DSN()
		if e != nil {
			return nil, falloMontajeCartera(fmt.Sprintf("identidad_config_%d", indice))
		}
		pool, login, e := abrirPoolNominalPortalExterno(ctx, dsn, entrada.rol,
			"vec-portal-externo-meritos-cartera")
		if e != nil {
			return nil, falloMontajeCartera(fmt.Sprintf("identidad_pool_%d", indice))
		}
		pools = append(pools, pool)
		if logins[login] || externa.logins[login] ||
			cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool, topologia) != nil {
			return nil, falloMontajeCartera(fmt.Sprintf("identidad_topologia_%d", indice))
		}
		logins[login] = true
	}
	reloj := relojRutasDietas{}
	registroSesion, err := identidadpg.NuevoRegistroSesionesExternoPostgreSQL(ctx, pools[6], pools[7],
		&seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo,
		dominioIdentidadSesionDesarrollo)
	if err != nil {
		return nil, falloMontajeCartera("identidad_registro")
	}
	revalidador, err := identidadpg.NuevoRevalidadorAutenticacionActorExternoPostgreSQL(ctx, pools[7])
	if err != nil {
		return nil, falloMontajeCartera("identidad_revalidacion")
	}
	resolutor, err := contextopg.NuevoResolutorRegistroContextoUsuariosExternoPostgreSQLV1(ctx, pools[5])
	if err != nil {
		return nil, falloMontajeCartera("contexto_externo")
	}
	servicioContexto, err := vecapp.NuevoServicioContextoActorProductivoV2(resolutor,
		contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
	if err != nil {
		return nil, falloMontajeCartera("contexto_servicio")
	}
	contextos, err := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if err != nil {
		return nil, falloMontajeCartera("contexto_autoridad")
	}
	instancia, err := nonceRutasDietas()
	if err != nil {
		return nil, falloMontajeCartera("identidad_instancia")
	}
	base := &autoridadRutasDietasDesarrollo{resolvedor: externa.base.resolvedor,
		registro: registroSesion, revalidador: revalidador, contextos: contextos,
		reloj: reloj, instancia: instancia}
	lector, err := meritospg.NuevaLecturaCartera(pools[0])
	if err != nil {
		return nil, falloMontajeCartera("lector")
	}
	retiro, err := meritospg.NuevaRetiradaCartera(pools[0])
	if err != nil {
		return nil, falloMontajeCartera("retirada")
	}
	auditor, err := meritospg.NuevaAuditoriaIntentosCartera(pools[0])
	if err != nil {
		return nil, falloMontajeCartera("auditor_intentos")
	}
	registrador, err := meritospg.NuevoRegistradorFronteraCartera(ctx, pools[1])
	if err != nil {
		return nil, falloMontajeCartera("frontera")
	}
	fuente, err := vecpg.NuevoAlmacenAutorizacionMeritosExterna(pools[2])
	if err != nil {
		return nil, falloMontajeCartera("fuente_autorizacion")
	}
	registro, err := vecpg.NuevoAlmacenAutorizacionRegistroMeritosExterno(pools[3])
	if err != nil {
		return nil, falloMontajeCartera("registro_autorizacion")
	}
	motivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2MeritosExterno(pools[4])
	if err != nil {
		return nil, falloMontajeCartera("motivos_autorizacion")
	}
	autorizador, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registro, registro,
		motivos, reloj, seguridad.GeneradorReferenciasCriptograficas{},
		vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if err != nil {
		return nil, falloMontajeCartera("servicio_autorizacion")
	}
	servicioLectura, err := meritosapp.NuevoServicioCartera(lector, auditor, fuente, reloj)
	if err != nil {
		return nil, falloMontajeCartera("servicio_lectura")
	}
	servicioRetirada, err := meritosapp.NuevoServicioRetiradaCartera(retiro, auditor, autorizador,
		proveedorRetiradaCartera{delegado: proveedores[meritosports.AudienciaRetirarCartera], motivo: motivo}, reloj)
	if err != nil {
		return nil, falloMontajeCartera("servicio_retirada")
	}
	nuevaAutoridad := func(ruta string) *autoridadPreferenciasUsuariosDesarrollo {
		return &autoridadPreferenciasUsuariosDesarrollo{base: base, cuentas: cuentas,
			reloj: reloj, registrador: registrador, incidencias: externa.incidencias,
			ruta: ruta, superficie: vec.SuperficieAutenticacionExternaPersonalV1,
			superficieAuditoria: vecports.SuperficieAuditoriaFronteraRutaExactaMeritosCartera,
			prefijoError:        "meritos.error.", metodoEscritura: "POST"}
	}
	cartera := nuevaAutoridad(meritoshttp.RutaCartera)
	retirada := nuevaAutoridad(meritoshttp.RutaRetiradasCartera)
	cartera.manejador, err = meritoshttp.NuevoHandlerCartera(preparadorCarteraExterna{autoridad: cartera, ruta: cartera.ruta}, servicioLectura)
	if err != nil {
		return nil, falloMontajeCartera("handler_lectura")
	}
	retirada.manejador, err = meritoshttp.NuevoHandlerRetiradasCartera(preparadorCarteraExterna{
		autoridad: retirada, ruta: retirada.ruta, motivoAUT: motivo}, servicioRetirada)
	if err != nil {
		return nil, falloMontajeCartera("handler_retirada")
	}
	declaracion := nuevaAutoridad(meritoshttp.RutaDeclaracionesCartera)
	poolOperacion, err := montarDeclaracionMeritosCartera(ctx, cfg, declaracion, privada, topologia, pools[0], autorizador,
		proveedores[descriptoresMaterialMeritosInternosDesarrollo()[0].Audiencia])
	if err != nil {
		return nil, err
	}
	pools = append(pools, poolOperacion)
	externa.declaracion = declaracion
	externa.cartera, externa.retirada = cartera, retirada
	completo = true
	return cerrar, nil
}

func acreditarFuncionesCarteraExterna(ctx context.Context, pool *pgxpool.Pool, firmas []string) error {
	if ctx == nil || pool == nil || len(firmas) == 0 {
		return errMontajeCarteraExterna
	}
	var acreditadas bool
	err := pool.QueryRow(ctx, `SELECT COALESCE(bool_and(
 CASE WHEN pg_catalog.to_regprocedure(f.firma) IS NULL THEN false
 ELSE pg_catalog.has_schema_privilege(session_user,pg_catalog.split_part(f.firma,'.',1),'USAGE')
  AND pg_catalog.has_function_privilege(session_user,f.firma,'EXECUTE') END),false)
 FROM pg_catalog.unnest($1::text[]) AS f(firma)`, firmas).Scan(&acreditadas)
	if err != nil || !acreditadas {
		return errMontajeCarteraExterna
	}
	return nil
}
