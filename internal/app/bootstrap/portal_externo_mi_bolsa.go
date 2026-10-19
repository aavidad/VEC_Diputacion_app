package bootstrap

import (
	"context"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	bolsapg "vec-diputacion-granada/internal/modules/bolsa/adapters/postgres"
	mibolsa "vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	bolsapuertos "vec-diputacion-granada/internal/modules/bolsa/ports"
	postgresqlcompartido "vec-diputacion-granada/internal/shared/postgresql"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"

	"vec-diputacion-granada/internal/shared/telemetria"
)

var (
	huellaPreimagenMiBolsaPortalExterno = regexp.MustCompile(`^[a-f0-9]{64}$`)
	claveNombreMiBolsaPortalExterno     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9.]*$`)
)

// semillaMiBolsaPortalExterno es el material fijo que prepara el operador en
// el lado interno. Devuelve rol, control y asignación para que Autorización
// publique primero el rol/control y después la asignación exterior por CAS.
// nombreClave procede del catálogo i18n y se conserva como clave, no texto.
func semillaMiBolsaPortalExterno(identidad *identidadCandidatoBolsaDesarrollo, ahora time.Time,
	nombreClave string) (dominiovec.InstantaneaAutorizacion, error) {
	if !strings.HasPrefix(nombreClave, "areaPersonal.miBolsa.") || len(nombreClave) > 128 ||
		!claveNombreMiBolsaPortalExterno.MatchString(nombreClave) {
		return dominiovec.InstantaneaAutorizacion{}, errMiBolsaNoDisponible
	}
	semilla, err := nuevaInstantaneaMiBolsaDesarrollo(identidad, ahora, true, false)
	if err != nil || semilla.VersionRol.RolID != rolPortalMiBolsaDesarrollo {
		return dominiovec.InstantaneaAutorizacion{}, errMiBolsaNoDisponible
	}
	semilla.VersionRol.Nombre = nombreClave
	if semilla.Validar() != nil {
		return dominiovec.InstantaneaAutorizacion{}, errMiBolsaNoDisponible
	}
	return semilla, nil
}

// documentoAsignacionMiBolsaPortalExterno es la propuesta canónica para la
// fachada AUT16. En alta se exige preimagen explícita (0,nil); en revisión,
// versión y huella esperadas. No escribe ni decide la aprobación del operador.
type documentoAsignacionMiBolsaPortalExterno struct {
	Documento       []byte
	HuellaSHA256    string
	VersionEsperada int64
	HuellaEsperada  *string
}

// documentosRolMiBolsaPortalExterno prepara los dos documentos que AUT16
// publicará bajo la autorización interna del operador. V1 se conserva si ya
// existe exacta; cualquier contenido distinto requiere una versión explícita.
type documentosRolMiBolsaPortalExterno struct {
	RolDocumento          []byte
	RolHuellaSHA256       string
	ControlDocumento      []byte
	ControlHuellaSHA256   string
	RevisionEsperada      int64
	ControlHuellaEsperada *string
}

func (documentosRolMiBolsaPortalExterno) String() string   { return "[ROL PRIVADO BOLSA]" }
func (documentosRolMiBolsaPortalExterno) GoString() string { return "[ROL PRIVADO BOLSA]" }

func prepararRolMiBolsaPortalExterno(semilla dominiovec.InstantaneaAutorizacion,
	revisionEsperada int64, huellaControlEsperada string,
) (documentosRolMiBolsaPortalExterno, error) {
	vacio := documentosRolMiBolsaPortalExterno{}
	if semilla.Validar() != nil || semilla.VersionRol.RolID != rolPortalMiBolsaDesarrollo ||
		revisionEsperada < 0 || revisionEsperada >= 1<<31 ||
		(revisionEsperada == 0 && huellaControlEsperada != "") ||
		(revisionEsperada > 0 && !huellaPreimagenMiBolsaPortalExterno.MatchString(huellaControlEsperada)) {
		return vacio, errMiBolsaNoDisponible
	}
	control := semilla.ControlVigenciaVersionRol
	control.Revision = uint64(revisionEsperada + 1)
	if control.Validar() != nil || control.VersionRolRef != semilla.VersionRol.Referencia() {
		return vacio, errMiBolsaNoDisponible
	}
	rolDocumento, err := json.Marshal(semilla.VersionRol)
	if err != nil {
		return vacio, errMiBolsaNoDisponible
	}
	controlDocumento, err := json.Marshal(control)
	if err != nil {
		return vacio, errMiBolsaNoDisponible
	}
	huellaRol := sha256.Sum256(rolDocumento)
	huellaControl := sha256.Sum256(controlDocumento)
	resultado := documentosRolMiBolsaPortalExterno{RolDocumento: rolDocumento,
		RolHuellaSHA256: hex.EncodeToString(huellaRol[:]), ControlDocumento: controlDocumento,
		ControlHuellaSHA256: hex.EncodeToString(huellaControl[:]), RevisionEsperada: revisionEsperada}
	if revisionEsperada > 0 {
		resultado.ControlHuellaEsperada = &huellaControlEsperada
	}
	return resultado, nil
}

func (documentoAsignacionMiBolsaPortalExterno) String() string   { return "[ASIGNACION PRIVADA BOLSA]" }
func (documentoAsignacionMiBolsaPortalExterno) GoString() string { return "[ASIGNACION PRIVADA BOLSA]" }

func prepararAsignacionMiBolsaPortalExterno(semilla dominiovec.InstantaneaAutorizacion,
	rolPublicado dominiovec.VersionRol, versionEsperada int64, huellaEsperada string,
) (documentoAsignacionMiBolsaPortalExterno, error) {
	vacia := documentoAsignacionMiBolsaPortalExterno{}
	if semilla.Validar() != nil || rolPublicado.Validar() != nil ||
		semilla.VersionRol.RolID != rolPortalMiBolsaDesarrollo ||
		rolPublicado.RolID != rolPortalMiBolsaDesarrollo ||
		rolPublicado.Estado != dominiovec.EstadoVersionRolPublicada ||
		!reflect.DeepEqual(semilla.VersionRol.Concesiones, rolPublicado.Concesiones) ||
		versionEsperada < 0 || versionEsperada >= 1<<31 ||
		(versionEsperada == 0 && huellaEsperada != "") ||
		(versionEsperada > 0 && !huellaPreimagenMiBolsaPortalExterno.MatchString(huellaEsperada)) {
		return vacia, errMiBolsaNoDisponible
	}
	asignacion := semilla.AsignacionPerfil
	asignacion.Version = int(versionEsperada + 1)
	asignacion.VersionRolRef = rolPublicado.Referencia()
	if asignacion.Validar() != nil {
		return vacia, errMiBolsaNoDisponible
	}
	documento, err := json.Marshal(asignacion)
	if err != nil {
		return vacia, errMiBolsaNoDisponible
	}
	huella := sha256.Sum256(documento)
	resultado := documentoAsignacionMiBolsaPortalExterno{Documento: documento,
		HuellaSHA256: hex.EncodeToString(huella[:]), VersionEsperada: versionEsperada}
	if versionEsperada > 0 {
		resultado.HuellaEsperada = &huellaEsperada
	}
	return resultado, nil
}

// abrirPoolMiBolsaPortalExterno comprueba un LOGIN nominal del proceso
// exterior: una sola membresía directa heredada, sin SET ni administración,
// sin cadena de roles y sin atributos privilegiados. No registra el DSN.
func abrirPoolMiBolsaPortalExterno(ctx context.Context, dsn, rol string) (*pgxpool.Pool, string, error) {
	return abrirPoolNominalPortalExterno(ctx, dsn, rol, "vec-portal-externo-mi-bolsa")
}

// abrirPoolNominalPortalExterno comparte la guarda de TLS, membresía única y
// telemetría; cada módulo conserva LOGIN, rol y nombre técnico propios.
func abrirPoolNominalPortalExterno(ctx context.Context, dsn, rol, nombreAplicacion string, otrosRoles ...string) (*pgxpool.Pool, string, error) {
	if ctx == nil || ctx.Err() != nil || dsn == "" || rol == "" || nombreAplicacion == "" {
		return nil, "", errMiBolsaNoDisponible
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil || cfg.ConnConfig.User == "" || validarTLSPostgreSQLBorradores(&cfg.ConnConfig.Config, true) != nil {
		return nil, "", errMiBolsaNoDisponible
	}
	// 8 por defecto: con 2, miles de candidatos a la vez hacían cola en el pool.
	postgresqlcompartido.FijarTamanoPool(cfg, dsn, 8)
	cfg.MinConns = 0
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	for clave, valor := range map[string]string{
		"application_name": nombreAplicacion, "timezone": "UTC", "search_path": "pg_catalog",
		"statement_timeout": "10s", "lock_timeout": "2s", "idle_in_transaction_session_timeout": "15s",
	} {
		cfg.ConnConfig.RuntimeParams[clave] = valor
	}
	telemetria.Instrumentar(cfg) // consultas por petición en el registro de acceso
	pool, err := postgresqlcompartido.NuevoPoolConPreflightTEMP(ctx, cfg)
	if err != nil {
		return nil, "", errMiBolsaNoDisponible
	}
	roles := append([]string{rol}, otrosRoles...)
	vistos := map[string]bool{}
	for _, esperado := range roles {
		if esperado == "" || vistos[esperado] {
			pool.Close()
			return nil, "", errMiBolsaNoDisponible
		}
		vistos[esperado] = true
	}
	var login string
	var permitido bool
	err = pool.QueryRow(ctx, `SELECT session_user::text,
 session_user=current_user AND r.rolcanlogin AND r.rolinherit AND
 NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND (SELECT count(*)=cardinality($1::text[]) FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)
 AND (SELECT count(*)=cardinality($1::text[]) FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid
             WHERE m.member=r.oid AND g.rolname=ANY($1::text[]) AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
               AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
               AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members superior WHERE superior.member=g.oid))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`, roles).Scan(&login, &permitido)
	if err != nil || !permitido {
		pool.Close()
		return nil, "", errMiBolsaNoDisponible
	}
	return pool, login, nil
}

func funcionesMiBolsaPortalExternoConInscripcion() []string {
	return append(funcionesMiBolsaPortalExternoSinInscripcion(), "solicitar_inscripcion_v1")
}

func funcionesMiBolsaPortalExternoConPreferencias(inscripcion bool) []string {
	var funciones []string
	if inscripcion {
		funciones = funcionesMiBolsaPortalExternoConInscripcion()
	} else {
		funciones = funcionesMiBolsaPortalExternoSinInscripcion()
	}
	return append(funciones, "consultar_preferencias_bolsas_v1", "ordenar_preferencias_bolsas_v1",
		"registrar_intento_lectura_preferencias_bolsas_v1")
}

func funcionesMiBolsaPortalExternoSinInscripcion() []string {
	return []string{
		"consultar_mi_bolsa_v1", "consultar_mi_bolsa_portal_v1", "consultar_historial_mi_bolsa_v1",
		"manifestar_disposicion_oferta_v1", "listar_ofertas_candidato_v1", "solicitar_portal_candidato_v1",
		"responder_llamamiento_portal_v1", "preparar_respuesta_portal_v1", "leer_portal_candidato_v1",
		"confirmar_contacto_propio_v1", "leer_contacto_candidato_v1", "solicitar_documental_portal_v1",
	}
}

const firmaSolicitarDocumentalPortalExterno = "vec_bolsa_llamamientos.solicitar_documental_portal_v1(text,text,text,text,text,text,text,date,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"

const firmaSolicitarInscripcionPortalExterno = "vec_bolsa_llamamientos.solicitar_inscripcion_v1(text,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"

const firmaOrdenarPreferenciasBolsasPortalExterno = "vec_bolsa_llamamientos.ordenar_preferencias_bolsas_v1(text,text[],bigint,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"

const firmaConsultarPreferenciasBolsasPortalExterno = "vec_bolsa_llamamientos.consultar_preferencias_bolsas_v1(text,bytea,bytea,jsonb)"

const firmaIntentoPreferenciasBolsasPortalExterno = "vec_bolsa_llamamientos.registrar_intento_lectura_preferencias_bolsas_v1(text,bytea,bytea,jsonb)"

// B138 concede al mismo grupo la escritura de modalidades (F1-13). Solo se
// admite con la firma exacta y junto a las tres funciones de preferencias B104.
const firmaGuardarModalidadesPortalExterno = "vec_bolsa_llamamientos.guardar_preferencias_modalidades_v1(text,text[],jsonb,bigint,text,timestamp with time zone,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"

const firmaPresentarAlegacionPortalExterno = "vec_bolsa_llamamientos.presentar_alegacion_v1(text,jsonb,bytea,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)"

// El estado SQL se decide por todas las sobrecargas B96 existentes, no por
// el selector HTTP ni por la sola lista de GRANT del LOGIN.
func funcionesMiBolsaPortalExternoExactas(obtenidas []string, funcionesB96 int, b96Exacta bool) bool {
	return funcionesMiBolsaPortalExternoExactasConPreferencias(obtenidas, funcionesB96, b96Exacta,
		0, false, 0, false, 0, false)
}

func funcionesMiBolsaPortalExternoExactasConPreferencias(obtenidas []string, funcionesB96 int, b96Exacta bool,
	funcionesB104 int, b104Exacta bool, funcionesLecturaB104 int, lecturaB104Exacta bool,
	funcionesIntentoB104 int, intentoB104Exacta bool,
) bool {
	return funcionesMiBolsaPortalExternoExactasConAlegaciones(obtenidas, funcionesB96, b96Exacta,
		funcionesB104, b104Exacta, funcionesLecturaB104, lecturaB104Exacta,
		funcionesIntentoB104, intentoB104Exacta, 0, false, 0, false)
}

func funcionesMiBolsaPortalExternoExactasConAlegaciones(obtenidas []string, funcionesB96 int, b96Exacta bool,
	funcionesB104 int, b104Exacta bool, funcionesLecturaB104 int, lecturaB104Exacta bool,
	funcionesIntentoB104 int, intentoB104Exacta bool, funcionesB114 int, b114Exacta bool,
	funcionesB138 int, b138Exacta bool,
) bool {
	preferenciasExactas := funcionesB104 == 1 && b104Exacta && funcionesLecturaB104 == 1 && lecturaB104Exacta &&
		funcionesIntentoB104 == 1 && intentoB104Exacta
	sinPreferencias := funcionesB104 == 0 && !b104Exacta && funcionesLecturaB104 == 0 && !lecturaB104Exacta &&
		funcionesIntentoB104 == 0 && !intentoB104Exacta
	var esperadas []string
	switch {
	case funcionesB96 == 0 && !b96Exacta:
		if preferenciasExactas {
			esperadas = funcionesMiBolsaPortalExternoConPreferencias(false)
		} else if sinPreferencias {
			esperadas = funcionesMiBolsaPortalExternoSinInscripcion()
		} else {
			return false
		}
	case funcionesB96 == 1 && b96Exacta:
		if preferenciasExactas {
			esperadas = funcionesMiBolsaPortalExternoConPreferencias(true)
		} else if sinPreferencias {
			esperadas = funcionesMiBolsaPortalExternoConInscripcion()
		} else {
			return false
		}
	default:
		return false
	}
	if funcionesB114 == 1 && b114Exacta && funcionesB96 == 1 && b96Exacta {
		esperadas = append(esperadas, "presentar_alegacion_v1")
	} else if funcionesB114 != 0 || b114Exacta {
		return false
	}
	if funcionesB138 == 1 && b138Exacta && preferenciasExactas {
		esperadas = append(esperadas, "guardar_preferencias_modalidades_v1")
	} else if funcionesB138 != 0 || b138Exacta {
		return false
	}
	if len(obtenidas) != len(esperadas) {
		return false
	}
	observadas := slices.Clone(obtenidas)
	slices.Sort(observadas)
	slices.Sort(esperadas)
	return slices.Equal(observadas, esperadas)
}

// abrirBolsaMiBolsaPortalExterno acepta las concesiones B59+B77 antes de B96
// y añade exclusivamente la inscripción después. Un GRANT lateral o una firma B96 distinta impide arrancar.
func abrirBolsaMiBolsaPortalExterno(ctx context.Context, dsn string) (*pgxpool.Pool, string, error) {
	pool, login, err := abrirPoolMiBolsaPortalExterno(ctx, dsn, "vec_bolsa_llamamientos_portal_externo")
	if err != nil {
		return nil, "", errMiBolsaNoDisponible
	}
	if !comprobarACLMiBolsaPortalExterno(ctx, pool, false) {
		pool.Close()
		return nil, "", errMiBolsaNoDisponible
	}
	return pool, login, nil
}

type consultadorACLMiBolsaPortalExterno interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func comprobarACLMiBolsaPortalExterno(ctx context.Context, consultador consultadorACLMiBolsaPortalExterno, requiereInscripcion bool) bool {
	return comprobarACLMiBolsaPortalExternoConAlegaciones(ctx, consultador, requiereInscripcion, false)
}

func comprobarACLMiBolsaPortalExternoConAlegaciones(ctx context.Context, consultador consultadorACLMiBolsaPortalExterno,
	requiereInscripcion, requiereAlegaciones bool) bool {
	if ctx == nil || ctx.Err() != nil || consultador == nil {
		return false
	}
	const acl = `WITH funciones AS (
 SELECT p.oid,p.proname,p.prosecdef FROM pg_catalog.pg_proc p
 JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
), tablas AS (
 SELECT c.oid FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND c.relkind IN ('r','p','v','m','f')
), secuencias AS (
 SELECT c.oid FROM pg_catalog.pg_class c JOIN pg_catalog.pg_namespace n ON n.oid=c.relnamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND c.relkind='S'
), b96 AS (
 SELECT count(*)::integer AS total,
  coalesce(bool_and(
   p.oid=pg_catalog.to_regprocedure($1::text)::oid AND p.prosecdef
   AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
   AND EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=(SELECT r.oid FROM pg_catalog.pg_roles r WHERE r.rolname='vec_bolsa_llamamientos_portal_externo')
      AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=0 AND a.privilege_type='EXECUTE')
  ),false) AS exacta
 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='solicitar_inscripcion_v1'
), b104 AS (
 SELECT count(*)::integer AS total,
  coalesce(bool_and(
   p.oid=pg_catalog.to_regprocedure($3::text)::oid AND p.prosecdef
   AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
   AND EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=(SELECT r.oid FROM pg_catalog.pg_roles r WHERE r.rolname='vec_bolsa_llamamientos_portal_externo')
      AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=0 AND a.privilege_type='EXECUTE')
  ),false) AS exacta
 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='ordenar_preferencias_bolsas_v1'
), b104_lectura AS (
 SELECT count(*)::integer AS total,
  coalesce(bool_and(
   p.oid=pg_catalog.to_regprocedure($4::text)::oid AND p.prosecdef
   AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
   AND EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=(SELECT r.oid FROM pg_catalog.pg_roles r WHERE r.rolname='vec_bolsa_llamamientos_portal_externo')
      AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=0 AND a.privilege_type='EXECUTE')
  ),false) AS exacta
 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='consultar_preferencias_bolsas_v1'
), b104_intento AS (
 SELECT count(*)::integer AS total,
  coalesce(bool_and(
   p.oid=pg_catalog.to_regprocedure($5::text)::oid AND p.prosecdef
   AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
   AND EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=(SELECT r.oid FROM pg_catalog.pg_roles r WHERE r.rolname='vec_bolsa_llamamientos_portal_externo')
      AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=0 AND a.privilege_type='EXECUTE')
  ),false) AS exacta
 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='registrar_intento_lectura_preferencias_bolsas_v1'
), b114 AS (
 SELECT count(*)::integer AS total,
  coalesce(bool_and(
   p.oid=pg_catalog.to_regprocedure($6::text)::oid AND p.prosecdef
   AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
   AND EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=(SELECT r.oid FROM pg_catalog.pg_roles r WHERE r.rolname='vec_bolsa_llamamientos_portal_externo')
      AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=0 AND a.privilege_type='EXECUTE')
  ),false) AS exacta
 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='presentar_alegacion_v1'
), b138 AS (
 SELECT count(*)::integer AS total,
  coalesce(bool_and(
   p.oid=pg_catalog.to_regprocedure($7::text)::oid AND p.prosecdef
   AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
   AND EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=(SELECT r.oid FROM pg_catalog.pg_roles r WHERE r.rolname='vec_bolsa_llamamientos_portal_externo')
      AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=0 AND a.privilege_type='EXECUTE')
  ),false) AS exacta
 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
 WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='guardar_preferencias_modalidades_v1'
)
SELECT COALESCE((SELECT array_agg(proname::text ORDER BY proname::text) FROM funciones),ARRAY[]::text[]),
 (SELECT coalesce(bool_and(prosecdef),false) FROM funciones)
 AND (SELECT count(*)=1 AND coalesce(bool_and(
   p.oid=pg_catalog.to_regprocedure($2::text)::oid AND p.prosecdef
   AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE')
   AND EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=(SELECT r.oid FROM pg_catalog.pg_roles r WHERE r.rolname='vec_bolsa_llamamientos_portal_externo')
      AND a.privilege_type='EXECUTE' AND NOT a.is_grantable)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.aclexplode(p.proacl) a
    WHERE a.grantee=0 AND a.privilege_type='EXECUTE')
  ),false) FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
  WHERE n.nspname='vec_bolsa_llamamientos' AND p.proname='solicitar_documental_portal_v1')
 AND NOT EXISTS (SELECT 1 FROM tablas t WHERE
   pg_catalog.has_table_privilege(session_user,t.oid,'SELECT,INSERT,UPDATE,DELETE,TRUNCATE,REFERENCES,TRIGGER')
   OR pg_catalog.has_any_column_privilege(session_user,t.oid,'SELECT,INSERT,UPDATE,REFERENCES'))
 AND NOT EXISTS (SELECT 1 FROM secuencias s WHERE
   pg_catalog.has_sequence_privilege(session_user,s.oid,'USAGE,SELECT,UPDATE'))
 AND NOT pg_catalog.has_schema_privilege(session_user,'vec_bolsa_llamamientos','CREATE')
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_proc p JOIN pg_catalog.pg_namespace n ON n.oid=p.pronamespace
   WHERE n.nspname='vec_autorizacion_atestada_v3'
     AND p.proname='registrar_intento_lectura_preferencias_bolsas_v1'
     AND pg_catalog.has_function_privilege(session_user,p.oid,'EXECUTE'))
 AND NOT EXISTS (SELECT 1 FROM pg_catalog.pg_namespace n
   WHERE left(n.nspname,4)='vec_' AND n.nspname<>'vec_bolsa_llamamientos'
     AND (pg_catalog.has_schema_privilege(session_user,n.oid,'USAGE')
          OR pg_catalog.has_schema_privilege(session_user,n.oid,'CREATE'))),
 (SELECT total FROM b96),(SELECT exacta FROM b96),
 (SELECT total FROM b104),(SELECT exacta FROM b104),
 (SELECT total FROM b104_lectura),(SELECT exacta FROM b104_lectura),
 (SELECT total FROM b104_intento),(SELECT exacta FROM b104_intento),
 (SELECT total FROM b114),(SELECT exacta FROM b114),
 (SELECT total FROM b138),(SELECT exacta FROM b138)`
	var funciones []string
	var permitido bool
	var funcionesB96 int
	var b96Exacta bool
	var funcionesB104 int
	var b104Exacta bool
	var funcionesLecturaB104 int
	var lecturaB104Exacta bool
	var funcionesIntentoB104 int
	var intentoB104Exacta bool
	var funcionesB114 int
	var b114Exacta bool
	var funcionesB138 int
	var b138Exacta bool
	return consultador.QueryRow(ctx, acl, firmaSolicitarInscripcionPortalExterno, firmaSolicitarDocumentalPortalExterno,
		firmaOrdenarPreferenciasBolsasPortalExterno, firmaConsultarPreferenciasBolsasPortalExterno,
		firmaIntentoPreferenciasBolsasPortalExterno, firmaPresentarAlegacionPortalExterno,
		firmaGuardarModalidadesPortalExterno).
		Scan(&funciones, &permitido, &funcionesB96, &b96Exacta, &funcionesB104, &b104Exacta,
			&funcionesLecturaB104, &lecturaB104Exacta, &funcionesIntentoB104, &intentoB104Exacta,
			&funcionesB114, &b114Exacta, &funcionesB138, &b138Exacta) == nil &&
		permitido && (!requiereInscripcion || funcionesB96 == 1 && b96Exacta) &&
		(!requiereAlegaciones || requiereInscripcion && funcionesB114 == 1 && b114Exacta) &&
		funcionesMiBolsaPortalExternoExactasConAlegaciones(funciones, funcionesB96, b96Exacta,
			funcionesB104, b104Exacta, funcionesLecturaB104, lecturaB104Exacta,
			funcionesIntentoB104, intentoB104Exacta, funcionesB114, b114Exacta, funcionesB138, b138Exacta)
}

// nuevaSesionMiBolsaPortalExterno recibe los puertos nominales exteriores de
// Identidad y ContextoActor. Reutiliza las reglas de sesión, sin la cuenta ni
// el perfil de Preferencias ni sus credenciales SQL.
func nuevaSesionMiBolsaPortalExterno(identidad *resolvedorIdentidadDesarrollo,
	registro httpseguridad.RegistroSesiones, revalidador dominiovec.RevalidadorAutenticacionActorV1,
	contextos dominiovec.ResolutorContextoActorRegistradoV2,
) (*autoridadPreferenciasUsuariosDesarrollo, error) {
	if identidad == nil || registro == nil || revalidador == nil || contextos == nil {
		return nil, errMiBolsaNoDisponible
	}
	reloj := relojRutasDietas{}
	instancia, err := nonceRutasDietas()
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	base := &autoridadRutasDietasDesarrollo{resolvedor: identidad, registro: registro,
		revalidador: revalidador, contextos: contextos, reloj: reloj, instancia: instancia}
	return &autoridadPreferenciasUsuariosDesarrollo{base: base,
		superficie: dominiovec.SuperficieAutenticacionExternaPersonalV1, reloj: reloj}, nil
}

// nuevaAutorizacionMiBolsaPortalExterno recibe exclusivamente puertos de
// lectura/registro/motivos del portal exterior. Sus adaptadores se conectan
// con los tres logins nominales de AUT-15, nunca con gobierno ni CT.
func nuevaAutorizacionMiBolsaPortalExterno(fuente puertosvec.FuenteAutorizacion,
	registro puertosvec.RegistroConcesionesCandidatasAutorizacionLigadaV3,
	denegaciones puertosvec.RegistroDenegacionesAutorizacionLigadaV3,
	motivos puertosvec.ValidadorReferenciaMotivoAutorizacionV2,
) (*aplicacionvec.ServicioAutorizacionSolicitudLigadaV3, error) {
	autorizador, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		fuente, registro, denegaciones, motivos, relojRutasDietas{},
		seguridadvec.GeneradorReferenciasCriptograficas{},
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second},
	)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	return autorizador, nil
}

// preparadorMiBolsaPortalExterno exige en cada petición el certificado mTLS,
// la cuenta exterior y el vínculo vigente único con el candidato. Comparte el
// registro de sesión y contexto ya acreditado para el Área personal externa.
type preparadorMiBolsaPortalExterno struct {
	preferencias  *autoridadPreferenciasUsuariosDesarrollo
	identidad     *identidadCandidatoBolsaDesarrollo
	superficie    dominiovec.SuperficieAutenticacionActorV1
	cuentaInterna cuentaUsuariosPreferenciasDesarrollo
}

func cuentaSesionMiBolsa(preferencias *autoridadPreferenciasUsuariosDesarrollo,
	identidad *identidadCandidatoBolsaDesarrollo, principal dominiovec.Principal,
	superficie dominiovec.SuperficieAutenticacionActorV1,
	cuentaInterna cuentaUsuariosPreferenciasDesarrollo) (cuentaUsuariosPreferenciasDesarrollo, bool) {
	vacia := cuentaUsuariosPreferenciasDesarrollo{}
	if preferencias == nil || identidad == nil || principal.ID == "" {
		return vacia, false
	}
	huella := principal.Attributes["certificate_sha256"]
	if superficie == dominiovec.SuperficieAutenticacionInternaCorporativaV1 {
		return cuentaInterna, cuentaInterna.CertificadoSHA256 == huella && cuentaInterna.Sujeto == principal.ID &&
			cuentaInterna.CuentaRef != "" && cuentaInterna.PerfilRef != "" &&
			cuentaInterna.CuentaRef != identidad.cuentaRef && cuentaInterna.PerfilRef != identidad.perfilRef
	}
	if superficie != dominiovec.SuperficieAutenticacionExternaPersonalV1 {
		return vacia, false
	}
	cuenta := cuentaUsuariosPreferenciasDesarrollo{cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{
		Sujeto: principal.ID, CuentaRef: identidad.cuentaRef,
		PerfilRef: identidad.perfilRef, CertificadoSHA256: huella,
	}}
	return cuenta, cuenta.CuentaRef != "" && cuenta.PerfilRef != ""
}

func (p *preparadorMiBolsaPortalExterno) PrepararMiBolsa(r *http.Request) (mibolsa.Orden, error) {
	if p == nil || p.preferencias == nil || p.preferencias.base == nil || p.identidad == nil ||
		r == nil || r.URL == nil || !bolsahttp.EsRutaPortal(r.URL.Path) || r.TLS == nil ||
		cabeceraLibreComisionesDietas(r.Header) || len(r.TLS.VerifiedChains) != 1 ||
		len(r.TLS.VerifiedChains[0]) == 0 || r.TLS.VerifiedChains[0][0] == nil {
		return mibolsa.Orden{}, bolsahttp.ErrAutenticacionAusente
	}
	principal, err := p.preferencias.base.resolvedor.ResolveDemoIdentity(r.Context(), peticionIdentidadConsultasContratacionTemporalDesarrollo(r))
	if err != nil || principal.ID != p.identidad.identidad.principal.ID ||
		principal.Attributes["certificate_sha256"] != p.identidad.identidad.principal.Attributes["certificate_sha256"] ||
		len(principal.Roles) != 1 || principal.Roles[0] != "candidato_bolsa" ||
		principal.AuthMethod != dominiovec.AuthMethodCertificate || principal.AuthAssurance != dominiovec.AuthAssuranceHigh {
		return mibolsa.Orden{}, bolsahttp.ErrAutenticacionAusente
	}
	// Cada superficie toma su cuenta de la autoridad que la acreditó.
	superficie := p.superficie
	if superficie == "" {
		superficie = dominiovec.SuperficieAutenticacionExternaPersonalV1
	}
	cuenta, valida := cuentaSesionMiBolsa(p.preferencias, p.identidad, principal, superficie, p.cuentaInterna)
	if !valida || p.preferencias.superficie != superficie {
		return mibolsa.Orden{}, bolsahttp.ErrAutenticacionAusente
	}
	ahora := p.preferencias.reloj.Ahora().UTC().Truncate(time.Microsecond)
	certificado := r.TLS.VerifiedChains[0][0]
	if ahora.Before(certificado.NotBefore) || !ahora.Before(certificado.NotAfter) ||
		p.identidad.verificadoEn.After(ahora) || !ahora.Before(p.identidad.validoHasta) {
		return mibolsa.Orden{}, bolsahttp.ErrAutenticacionAusente
	}
	vinculo, resultado, err := p.preferencias.resolverSesion(r, cuenta, ahora)
	// La resolución registra un instante posterior al de entrada. Volver a
	// leer el reloj mantiene las mismas guardas sobre el resultado ya resuelto.
	ahora = p.preferencias.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err != nil || vinculo.ValidarPara(resultado) != nil ||
		resultado.Contexto.PersonaRef != p.identidad.personaRef ||
		resultado.Contexto.PerfilActivoRef != cuenta.PerfilRef ||
		!vigenciaMiBolsaPortalExterno(certificado, p.identidad, vinculo, resultado, ahora) {
		return mibolsa.Orden{}, errMiBolsaNoDisponible
	}
	candidatos := 0
	for _, v := range resultado.Contexto.Instantanea.Vinculos {
		if v.Tipo == dominiovec.TipoReferenciaContextoActorCandidato && v.VigenteEn(ahora) {
			candidatos++
			if v.Referencia != p.identidad.candidatoRef {
				return mibolsa.Orden{}, errMiBolsaNoDisponible
			}
		}
	}
	if candidatos != 1 {
		return mibolsa.Orden{}, errMiBolsaNoDisponible
	}
	correlacion, err := dominiovec.GenerarReferenciaCorrelacionAutorizacionV2(r.Context(), seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return mibolsa.Orden{}, errMiBolsaNoDisponible
	}
	motivo := motivoMiBolsaDesarrollo()
	if r.URL.Path == bolsahttp.RutaMiBolsaHistorial {
		motivo = motivoHistorialMiBolsaDesarrollo()
	} else if r.URL.Path != bolsahttp.RutaMiBolsa {
		motivo = motivoPortalMiBolsaDesarrollo()
	}
	return mibolsa.Orden{ResultadoContexto: resultado, Vinculo: vinculo, Motivo: motivo, Correlacion: correlacion}, nil
}

// vigenciaMiBolsaPortalExterno revalida al terminar la resolución. Si el
// certificado, la identidad o la sesión caducaron entre medias, deniega.
func vigenciaMiBolsaPortalExterno(certificado *x509.Certificate,
	identidad *identidadCandidatoBolsaDesarrollo, vinculo dominiovec.VinculoAutenticacionActorV2,
	resultado dominiovec.ResultadoContextoActorRegistradoV2, ahora time.Time,
) bool {
	return certificado != nil && identidad != nil && !ahora.Before(certificado.NotBefore) &&
		ahora.Before(certificado.NotAfter) && !identidad.verificadoEn.After(ahora) &&
		ahora.Before(identidad.validoHasta) && vinculo.VigenteEn(ahora, resultado)
}

// dependenciasMiBolsaPortalExterno son capacidades resueltas por la
// composición exterior. El constructor no publica perfiles, motivos, cuentas
// ni claves; esa provisión pertenece al proceso interno y exige huella y CAS.
type dependenciasMiBolsaPortalExterno struct {
	justificantes         http.Handler
	preparador            bolsahttp.Preparador
	autorizador           *aplicacionvec.ServicioAutorizacionSolicitudLigadaV3
	bolsa                 *pgxpool.Pool
	proveedores           map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo
	fuentePreferencias    puertosvec.FuenteAutorizacion
	proveedorPreferencias *proveedorMaterialAltaContratacionTemporalDesarrollo
	catalogoModalidades   bolsapuertos.CatalogoPreferenciasModalidades
	modalidadesSQL        bool
	campos                bolsapuertos.CamposPortalMiBolsa
	reglas                bolsapuertos.ReglasPortalCandidato
	reloj                 relojContratacionTemporalDesarrollo
}

// nuevasRutasMiBolsaPortalExterno compone únicamente las seis rutas propias
// del candidato a partir de un perfil ya provisionado y material V3 cotejado.
// Una dependencia incompleta detiene el montaje antes de exponer una ruta.
func nuevasRutasMiBolsaPortalExterno(d dependenciasMiBolsaPortalExterno) ([]vechttp.RutaExacta, error) {
	if d.preparador == nil || d.autorizador == nil || d.bolsa == nil ||
		d.proveedores[bolsapuertos.AudienciaMiBolsa] == nil ||
		d.proveedores[bolsapuertos.AudienciaHistorialMiBolsa] == nil {
		return nil, errMiBolsaNoDisponible
	}
	if d.reglas != nil {
		for _, par := range accionesPropiasPortalDesarrollo() {
			if d.proveedores[par[1]] == nil {
				return nil, errMiBolsaNoDisponible
			}
		}
	}
	propio := d.autorizador
	preparador := d.preparador
	var err error
	consulta, err := bolsapg.NuevaConsultaMiBolsaPostgreSQL(d.bolsa)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	servicio, err := mibolsa.Nuevo(consulta, propio, proveedorMiBolsaDesarrollo{delegado: d.proveedores[bolsapuertos.AudienciaMiBolsa]}, d.reloj)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	if d.reglas != nil {
		if servicio, err = servicio.ConReglasPortal(d.reglas); err != nil {
			return nil, errMiBolsaNoDisponible
		}
		if servicio, err = servicio.ConOfertas(); err != nil {
			return nil, errMiBolsaNoDisponible
		}
		if servicio, err = servicio.ConContacto(); err != nil {
			return nil, errMiBolsaNoDisponible
		}
	}
	var consultaHTTP http.Handler
	if d.campos == nil {
		consultaHTTP, err = bolsahttp.Nuevo(preparador, servicio)
	} else {
		consultaHTTP, err = bolsahttp.NuevoConCampos(preparador, servicio, d.campos)
	}
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	rutas := []vechttp.RutaExacta{{Ruta: bolsahttp.RutaMiBolsa, Manejador: consultaHTTP}}
	justificantes := d.justificantes
	if justificantes == nil {
		justificantes = bolsahttp.NuevoHandlerJustificanteRenunciaNoDisponible()
	}
	rutas = append(rutas, vechttp.RutaExacta{Ruta: bolsahttp.RutaJustificantesMiBolsa, Manejador: justificantes})
	preferenciasHTTP := bolsahttp.NuevoHandlerPreferenciasBolsasNoDisponible()
	if d.proveedorPreferencias != nil {
		if d.fuentePreferencias == nil {
			return nil, errMiBolsaNoDisponible
		}
		lecturaPG, e := bolsapg.NuevaLecturaPreferenciasBolsasPostgreSQL(d.bolsa)
		if e != nil {
			return nil, errMiBolsaNoDisponible
		}
		auditoriaPG, e := bolsapg.NuevaAuditoriaIntentosPreferenciasBolsasPostgreSQL(d.bolsa)
		if e != nil {
			return nil, errMiBolsaNoDisponible
		}
		lectura, e := mibolsa.NuevaLecturaPreferenciasBolsas(lecturaPG, auditoriaPG,
			d.fuentePreferencias, d.reloj)
		if e != nil {
			return nil, errMiBolsaNoDisponible
		}
		escrituraPG, e := bolsapg.NuevaEscrituraPreferenciasBolsasPostgreSQL(d.bolsa)
		if e != nil {
			return nil, errMiBolsaNoDisponible
		}
		escritura, e := mibolsa.NuevoServicioPreferenciasBolsas(escrituraPG, propio,
			proveedorPreferenciasBolsasPortalExterno{delegado: d.proveedorPreferencias}, d.reloj)
		if e != nil {
			return nil, errMiBolsaNoDisponible
		}
		preferenciasHTTP, e = bolsahttp.NuevoHandlerPreferenciasBolsasConLectura(preparador, escritura, lectura)
		if e != nil {
			return nil, errMiBolsaNoDisponible
		}
	}
	preferenciasHTTP, err = bolsahttp.NuevoSelectorPreferenciasModalidades(preferenciasHTTP, handlerPreferenciasModalidadesPortalExterno(d))
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	rutas = append(rutas, vechttp.RutaExacta{Ruta: bolsahttp.RutaPreferenciasBolsas, Manejador: preferenciasHTTP})
	historial, err := mibolsa.NuevoHistorial(consulta, propio, proveedorHistorialMiBolsaDesarrollo{delegado: d.proveedores[bolsapuertos.AudienciaHistorialMiBolsa]}, d.reloj)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	historialHTTP, err := bolsahttp.NuevoHistorial(preparador, historial)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	rutas = append(rutas, vechttp.RutaExacta{Ruta: bolsahttp.RutaMiBolsaHistorial, Manejador: historialHTTP})
	if d.reglas == nil {
		return rutas, nil
	}
	registro, err := bolsapg.NuevoRegistroPortalCandidatoPostgreSQL(d.bolsa)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	proveedoresPortal := make(map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo)
	for _, par := range accionesPropiasPortalDesarrollo() {
		proveedoresPortal[par[0]] = d.proveedores[par[1]]
	}
	portal, err := mibolsa.NuevoPortal(registro, propio, proveedorPortalMiBolsaDesarrollo{porAccion: proveedoresPortal}, d.reglas, d.reloj)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	for _, ruta := range []string{bolsahttp.RutaMiBolsaSolicitudes, bolsahttp.RutaMiBolsaSolicitudesDocumentales, bolsahttp.RutaMiBolsaRespuestas} {
		h, err := bolsahttp.NuevoPortal(ruta, preparador, portal)
		if err != nil {
			return nil, errMiBolsaNoDisponible
		}
		rutas = append(rutas, vechttp.RutaExacta{Ruta: ruta, Manejador: h})
	}
	ofertas, err := bolsapg.NuevoRegistroDisposicionOfertaPostgreSQL(d.bolsa)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	conOfertas, err := portal.ConRegistroOfertas(ofertas)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	disposicion, err := bolsahttp.NuevoDisposicion(preparador, conOfertas)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	rutas = append(rutas, vechttp.RutaExacta{Ruta: bolsahttp.RutaMiBolsaDisposiciones, Manejador: disposicion})
	contactos, err := bolsapg.NuevoRegistroConfirmacionContactoPostgreSQL(d.bolsa)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	conContacto, err := portal.ConRegistroContacto(contactos)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	contacto, err := bolsahttp.NuevoContacto(preparador, conContacto)
	if err != nil {
		return nil, errMiBolsaNoDisponible
	}
	return append(rutas, vechttp.RutaExacta{Ruta: bolsahttp.RutaMiBolsaContacto, Manejador: contacto}), nil
}
