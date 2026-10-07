package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	usuariospg "vec-diputacion-granada/internal/modules/usuarios/adapters/postgres"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
)

type consultaFuncionAutorizacionPreferencias interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

type topologiaPostgreSQLPreferenciasUsuarios struct {
	base, direccion, inicio string
	puerto                  int
}

func (t topologiaPostgreSQLPreferenciasUsuarios) coincide(otra topologiaPostgreSQLPreferenciasUsuarios) bool {
	return t.base != "" && t.direccion != "" && t.puerto > 0 && t.inicio != "" && t == otra
}

// La sonda toma el destino observado de cada sesión. El inicio del postmaster
// distingue dos servidores que expongan el mismo nombre de base y dirección.
// Un socket Unix sin dirección TCP observada se rechaza en este montaje.
const sondaTopologiaPostgreSQLPreferenciasUsuarios = `SELECT current_database()::text,
 COALESCE(inet_server_addr()::text,''), COALESCE(inet_server_port(),0),
 extract(epoch FROM pg_postmaster_start_time())::text`

func acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx context.Context, pool *pgxpool.Pool) (topologiaPostgreSQLPreferenciasUsuarios, error) {
	vacia := topologiaPostgreSQLPreferenciasUsuarios{}
	if ctx == nil || pool == nil {
		return vacia, errComposicionUsuariosPreferencias
	}
	configuracion := pool.Config()
	if configuracion == nil || configuracion.ConnConfig == nil || len(configuracion.ConnConfig.Fallbacks) != 0 {
		return vacia, errComposicionUsuariosPreferencias
	}
	var observada topologiaPostgreSQLPreferenciasUsuarios
	if err := pool.QueryRow(ctx, sondaTopologiaPostgreSQLPreferenciasUsuarios).Scan(&observada.base, &observada.direccion, &observada.puerto, &observada.inicio); err != nil {
		return vacia, errComposicionUsuariosPreferencias
	}
	if !observada.coincide(observada) {
		return vacia, errComposicionUsuariosPreferencias
	}
	return observada, nil
}

func cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx context.Context, pool *pgxpool.Pool, esperada topologiaPostgreSQLPreferenciasUsuarios) error {
	observada, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, pool)
	if err != nil || !esperada.coincide(observada) {
		return errComposicionUsuariosPreferencias
	}
	return nil
}

const sondaFuncionAutorizacionPreferenciasSQL = `SELECT session_user=current_user
 AND has_schema_privilege(session_user,'vec_autorizacion','USAGE')
 AND (SELECT count(*)=1 FROM pg_catalog.pg_proc p
      JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
      WHERE n.nspname='vec_autorizacion' AND p.proname=$1 AND p.prokind='f'
        AND has_function_privilege(session_user,p.oid,'EXECUTE'))`

func acreditarFuncionAutorizacionPreferencias(ctx context.Context, consulta consultaFuncionAutorizacionPreferencias, nombre string) error {
	if ctx == nil || consulta == nil || (nombre != "obtener_instantanea" && nombre != "registrar_decision_contexto_actor_v3" && nombre != "resolver_motivo_autorizacion_v2_historico") {
		return errComposicionUsuariosPreferencias
	}
	var acreditada bool
	if err := consulta.QueryRow(ctx, sondaFuncionAutorizacionPreferenciasSQL, nombre).Scan(&acreditada); err != nil || !acreditada {
		return errComposicionUsuariosPreferencias
	}
	return nil
}

func leerConfiguracionUsuariosPreferenciasDesarrollo(cfg config.Config, superficie core.SuperficieAutenticacionActorV1) (configuracionUsuariosPreferenciasDesarrollo, error) {
	vacia := configuracionUsuariosPreferenciasDesarrollo{}
	nombre := nombreConfiguracionPreferencias(superficie)
	if nombre == "" {
		return vacia, errComposicionUsuariosPreferencias
	}
	b, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", nombre), 128<<10)
	if err != nil || validarClavesJSONUnicas(b) != nil {
		return vacia, errComposicionUsuariosPreferencias
	}
	defer borrarBytes(b)
	var c configuracionUsuariosPreferenciasDesarrollo
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	var extra any
	if d.Decode(&c) != nil || !errors.Is(d.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa || c.Superficie != superficie ||
		len(c.Cuentas) == 0 || len(c.Cuentas) > 64 || !core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoConsulta) ||
		!core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoActualizacion) || c.MotivoConsulta.CatalogoID != c.MotivoActualizacion.CatalogoID {
		return vacia, errComposicionUsuariosPreferencias
	}
	return c, nil
}

func configuracionesPreferenciasSeparadas(interna, externa configuracionUsuariosPreferenciasDesarrollo) bool {
	if interna.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 || externa.Superficie != core.SuperficieAutenticacionExternaPersonalV1 {
		return false
	}
	certificados := map[string]bool{}
	cuentas := map[string]bool{}
	perfiles := map[string]bool{}
	for _, c := range interna.Cuentas {
		b, err := hex.DecodeString(c.CertificadoSHA256)
		if err == nil && len(b) == sha256.Size && hex.EncodeToString(b) == c.CertificadoSHA256 && c.Sujeto != "" && c.CuentaRef != "" && c.PerfilRef != "" &&
			!certificados[c.CertificadoSHA256] && !cuentas[c.CuentaRef] && !perfiles[c.PerfilRef] {
			certificados[c.CertificadoSHA256] = true
			cuentas[c.CuentaRef] = true
			perfiles[c.PerfilRef] = true
			continue
		}
		return false
	}
	for _, c := range externa.Cuentas {
		b, err := hex.DecodeString(c.CertificadoSHA256)
		if err == nil && len(b) == sha256.Size && hex.EncodeToString(b) == c.CertificadoSHA256 && c.Sujeto != "" && c.CuentaRef != "" && c.PerfilRef != "" &&
			!certificados[c.CertificadoSHA256] && !cuentas[c.CuentaRef] && !perfiles[c.PerfilRef] {
			certificados[c.CertificadoSHA256] = true
			cuentas[c.CuentaRef] = true
			perfiles[c.PerfilRef] = true
			continue
		}
		return false
	}
	return true
}

// Los cuatro roles de Usuarios y las seis identidades de infraestructura por
// superficie se comprueban antes de publicar cualquiera de las cuatro claves
// V3. Esta sonda no conserva pools: el montaje vuelve a abrirlos y los posee.
func preflightSQLPreferenciasUsuariosDesarrollo(cfg config.Config, derivador *derivadorIdentidadOperacionDesarrollo, gobierno *pgxpool.Pool) error {
	if derivador == nil || !derivador.valido() || gobierno == nil {
		return errComposicionUsuariosPreferencias
	}
	cInterna, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, core.SuperficieAutenticacionInternaCorporativaV1)
	if err != nil {
		return errComposicionUsuariosPreferencias
	}
	configuraciones := []configuracionUsuariosPreferenciasDesarrollo{cInterna}
	if superficieExternaUsuariosEnProceso(cfg) {
		cExterna, err := leerConfiguracionUsuariosPreferenciasDesarrollo(cfg, core.SuperficieAutenticacionExternaPersonalV1)
		if err != nil || !configuracionesPreferenciasSeparadas(cInterna, cExterna) {
			return errComposicionUsuariosPreferencias
		}
		configuraciones = append(configuraciones, cExterna)
	}
	ctx, cancel := context.WithTimeout(context.Background(), plazoarranque.Ampliar(30*time.Second))
	defer cancel()
	topologiaGobierno, err := acreditarTopologiaPostgreSQLPreferenciasUsuarios(ctx, gobierno)
	if err != nil {
		return errComposicionUsuariosPreferencias
	}
	var pools []*pgxpool.Pool
	defer func() {
		for _, p := range pools {
			p.Close()
		}
	}()
	logins := map[string]bool{}
	for _, c := range configuraciones {
		superficie := c.Superficie
		entradas := []struct{ dsn, rol string }{
			{c.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"}, {c.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"},
			{c.DSNContexto, "vec_contexto_actor_v1_runtime"}, {c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"},
			{c.DSNRegistroAutorizacion, "vec_autorizacion_registro"}, {c.DSNMotivos, "vec_autorizacion_motivos_evaluador"},
		}
		inicio := len(pools)
		for _, entrada := range entradas {
			p, login, e := abrirPoolRutasDietas(ctx, entrada.dsn, entrada.rol)
			if e != nil || login == "" || logins[login] {
				if p != nil {
					p.Close()
				}
				return errComposicionUsuariosPreferencias
			}
			pools = append(pools, p)
			logins[login] = true
		}
		// La sonda genérica de membresía no acredita las funciones/ACL de
		// sesión y Contexto. Sus constructores ejecutan esa acreditación SQL
		// antes de que el gobierno publique ninguna de las cuatro audiencias.
		if _, err = identidadpg.NuevoRegistroSesionesPostgreSQL(ctx, pools[inicio], pools[inicio+1],
			&seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo); err != nil {
			return errComposicionUsuariosPreferencias
		}
		if _, err = identidadpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, pools[inicio+1]); err != nil {
			return errComposicionUsuariosPreferencias
		}
		resolutor, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pools[inicio+2])
		if err != nil {
			return errComposicionUsuariosPreferencias
		}
		reloj := relojRutasDietas{}
		servicioContexto, err := vecapp.NuevoServicioContextoActorProductivoV2(resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
		if err != nil {
			return errComposicionUsuariosPreferencias
		}
		if _, err = vecapp.NuevaAutoridadContextoActorRegistradoV2(servicioContexto); err != nil {
			return errComposicionUsuariosPreferencias
		}
		fuente, err := vecpg.NuevoAlmacenAutorizacion(pools[inicio+3])
		if err != nil {
			return errComposicionUsuariosPreferencias
		}
		registroAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(pools[inicio+4])
		if err != nil {
			return errComposicionUsuariosPreferencias
		}
		motivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools[inicio+5], c.MotivoConsulta.CatalogoID)
		if err != nil {
			return errComposicionUsuariosPreferencias
		}
		if _, err = vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registroAutorizacion, registroAutorizacion, motivos, reloj,
			seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second}); err != nil {
			return errComposicionUsuariosPreferencias
		}
		for _, sonda := range []struct {
			pool   *pgxpool.Pool
			nombre string
		}{{pools[inicio+3], "obtener_instantanea"}, {pools[inicio+4], "registrar_decision_contexto_actor_v3"}, {pools[inicio+5], "resolver_motivo_autorizacion_v2_historico"}} {
			if acreditarFuncionAutorizacionPreferencias(ctx, sonda.pool, sonda.nombre) != nil {
				return errComposicionUsuariosPreferencias
			}
		}
		ejecutor, loginE, err := abrirPoolUsuariosPreferencias(ctx, c.DSNUsuarios, rolEjecutorPreferencias(string(superficie)))
		if err != nil || loginE == "" || logins[loginE] || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, ejecutor, topologiaGobierno) != nil {
			if ejecutor != nil {
				ejecutor.Close()
			}
			return errComposicionUsuariosPreferencias
		}
		pools = append(pools, ejecutor)
		logins[loginE] = true
		registrador, loginR, err := abrirPoolUsuariosPreferencias(ctx, c.DSNUsuariosFrontera, rolRegistradorPreferencias(string(superficie)))
		if err != nil || loginR == "" || logins[loginR] || cotejarTopologiaPostgreSQLPreferenciasUsuarios(ctx, registrador, topologiaGobierno) != nil {
			if registrador != nil {
				registrador.Close()
			}
			return errComposicionUsuariosPreferencias
		}
		pools = append(pools, registrador)
		logins[loginR] = true
		if _, err = usuariospg.NuevoRegistroPreferenciasPostgreSQL(ctx, ejecutor, superficie); err != nil {
			return errComposicionUsuariosPreferencias
		}
		if _, err = usuariospg.NuevoRegistradorDenegacionPreferenciasPostgreSQL(ctx, registrador, superficie); err != nil {
			return errComposicionUsuariosPreferencias
		}
	}
	return nil
}
