package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	meritoshttp "vec-diputacion-granada/internal/modules/meritos/adapters/http"
	meritospg "vec-diputacion-granada/internal/modules/meritos/adapters/postgres"
	meritosapp "vec-diputacion-granada/internal/modules/meritos/application"
	meritosports "vec-diputacion-granada/internal/modules/meritos/ports"
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
	Version        int                           `json:"version"`
	MotivoRetirada vec.ReferenciaEntradaCatalogo `json:"motivo_retirada"`
}

// La referencia técnica se publica en el catálogo de Autorización y viaja en
// material privado 0600. Ninguna clave de motivo se decide en código.
func leerMotivoCarteraExterna(directorio string) (vec.ReferenciaEntradaCatalogo, error) {
	vacio := vec.ReferenciaEntradaCatalogo{}
	b, err := leerFicheroMaterialSeguro(filepath.Join(directorio, rutaMotivoCarteraExterna), 4096)
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
		c.MotivoRetirada.CatalogoID != "motivos_autorizacion_meritos_cartera" {
		return vacio, errMontajeCarteraExterna
	}
	return c.MotivoRetirada, nil
}

// montarCarteraPortalExterno comparte la sesión certificada de Preferencias,
// pero separa los cinco LOGIN de datos, frontera, fuente, registro y motivos.
// Sólo publica ambas rutas cuando todo su material y topología están listos.
func montarCarteraPortalExterno(ctx context.Context, cfg config.Config,
	externa *autoridadPreferenciasUsuariosDesarrollo, preflight *pgxpool.Pool,
) (func(), error) {
	if ctx == nil || externa == nil || externa.base == nil || externa.superficie != vec.SuperficieAutenticacionExternaPersonalV1 ||
		preflight == nil {
		return nil, falloMontajeCartera("dependencia")
	}
	datosDSN, fronteraDSN, fuenteDSN, registroDSN, motivosDSN, err := cfg.DSNMeritosCarteraSeparados()
	if err != nil {
		return nil, falloMontajeCartera("configuracion_dsn")
	}
	motivo, err := leerMotivoCarteraExterna(cfg.DevelopmentMaterialDir)
	if err != nil {
		return nil, falloMontajeCartera("motivo_privado")
	}
	proveedores, err := nuevosProveedoresV3PortalExterno(ctx, cfg.DevelopmentMaterialDir, preflight,
		consumidorMeritosCarteraPortalExternoV3, relojContratacionTemporalDesarrollo{})
	if err != nil || proveedores[meritosports.AudienciaRetirarCartera] == nil {
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
	reloj := relojRutasDietas{}
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
		return &autoridadPreferenciasUsuariosDesarrollo{base: externa.base, cuentas: externa.cuentas,
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
