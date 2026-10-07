package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	ctdomain "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"

	bolsaauditoria "vec-diputacion-granada/internal/modules/bolsa/adapters/auditoriaconsulta"
	ctauditoria "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/auditoriaconsulta"
	"vec-diputacion-granada/internal/vec/adapters/fichero"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	postgresvec "vec-diputacion-granada/internal/vec/adapters/postgres"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	aplicacionvec "vec-diputacion-granada/internal/vec/application"
	"vec-diputacion-granada/internal/vec/auditoria"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

type claveCorrelacionAuditoriaLocal struct{}
type claveActorAuditoriaLocal struct{}

type actorAuditoriaLocal struct {
	mu  sync.Mutex
	ref string
}

func (a *actorAuditoriaLocal) fijar(ref string) {
	if a == nil || ref == "" {
		return
	}
	a.mu.Lock()
	if a.ref == "" {
		a.ref = ref
	}
	a.mu.Unlock()
}

func (a *actorAuditoriaLocal) valor() string {
	if a == nil {
		return ""
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ref
}

type respuestaEstadoAuditoriaLocal struct {
	http.ResponseWriter
	codigo int
}

func (r *respuestaEstadoAuditoriaLocal) WriteHeader(codigo int) {
	if r.codigo != 0 {
		return
	}
	r.codigo = codigo
	r.ResponseWriter.WriteHeader(codigo)
}

func (r *respuestaEstadoAuditoriaLocal) Write(b []byte) (int, error) {
	if r.codigo == 0 {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(b)
}

type manejadorAuditoriaDenegacionesLocales struct {
	siguiente   http.Handler
	registrador vecports.RegistradorAuditoriaFronteraRutaExacta
	soporte     *soporteAltaContratacionTemporalDesarrollo
}

func (m manejadorAuditoriaDenegacionesLocales) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if w == nil {
		return
	}
	if r == nil || r.URL == nil || m.siguiente == nil || dependenciaAuditoriaConsultaNula(m.registrador) {
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	ruta := r.URL.Path
	if ruta != auditoria.RutaOpciones && ruta != auditoria.RutaConsulta {
		http.NotFound(w, r)
		return
	}
	correlacion := "corr_no_disponible"
	if generada, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(r.Context(), seguridadvec.GeneradorReferenciasCriptograficas{}); err == nil {
		if canonica, err := generada.ValorCanonico(); err == nil {
			if strings.HasPrefix(canonica, "correlacion_") && len(canonica) == len("correlacion_")+32 {
				correlacion = "corr_" + strings.TrimPrefix(canonica, "correlacion_")
				r = r.WithContext(context.WithValue(r.Context(), claveCorrelacionAuditoriaLocal{}, generada))
			}
		}
	}
	actor := &actorAuditoriaLocal{}
	if m.soporte != nil {
		if capacidad, valida := m.soporte.capacidadValida(r.Context()); valida && capacidad.ruta == ruta {
			ahora := m.soporte.reloj.Ahora().UTC().Truncate(time.Microsecond)
			if !capacidad.certificadoVerificadoEn.After(ahora) && ahora.Before(capacidad.certificadoValidoHasta) {
				actor.fijar(capacidad.principal.ID)
			}
		}
	}
	r = r.WithContext(context.WithValue(r.Context(), claveActorAuditoriaLocal{}, actor))
	w.Header().Set("X-Correlation-Ref", correlacion)
	respuesta := &respuestaEstadoAuditoriaLocal{ResponseWriter: w}
	m.siguiente.ServeHTTP(respuesta, r)
	if respuesta.codigo != http.StatusForbidden && respuesta.codigo != http.StatusUnauthorized {
		return
	}
	motivo := vecports.MotivoAuditoriaFronteraRutaExactaAccesoDenegado
	if respuesta.codigo == http.StatusUnauthorized {
		motivo = vecports.MotivoAuditoriaFronteraRutaExactaAutenticacionRequerida
	}
	ctxRegistro, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), plazoarranque.Ampliar(250*time.Millisecond))
	defer cancelar()
	if err := m.registrador.RegistrarAuditoriaFronteraRutaExacta(ctxRegistro, vecports.OrdenAuditoriaFronteraRutaExacta{
		CorrelacionRef: correlacion, Motivo: motivo,
		Superficie: vecports.SuperficieAuditoriaFronteraRutaExactaAuditoria,
		Ruta:       ruta, ActorRef: actor.valor(),
	}); err != nil {
		slog.Error("auditoria: denegacion local sin bitacora", "correlacion_ref", correlacion,
			"ruta", ruta, "causa", "registro_frontera_no_disponible")
	}
}

// La raíz aporta pools nominales, emisores independientes y fronteras ya
// compuestas. Esta fábrica no activa la capacidad ni publica concesiones.
type dependenciasAuditoriaConsultaRRHH struct {
	PoolCT, PoolBolsa     *pgxpool.Pool
	EmisorCT, EmisorBolsa auditoria.EmisorMaterialV3
	Identidad             auditoria.IdentidadConsulta
	Opciones              auditoria.ProveedorOpciones
	Intentos              auditoria.ConfiguracionIntentos
}

func nuevasRutasAuditoriaConsultaRRHH(d dependenciasAuditoriaConsultaRRHH) ([]vechttp.RutaExacta, error) {
	if d.PoolCT == nil || d.PoolBolsa == nil || dependenciaAuditoriaConsultaNula(d.EmisorCT) ||
		dependenciaAuditoriaConsultaNula(d.EmisorBolsa) || dependenciaAuditoriaConsultaNula(d.Identidad) ||
		dependenciaAuditoriaConsultaNula(d.Opciones) || dependenciaAuditoriaConsultaNula(d.Intentos.Registrador) {
		return nil, auditoria.ErrNoDisponible
	}
	ct, err := ctauditoria.NuevaFuente(d.PoolCT)
	if err != nil {
		return nil, err
	}
	bolsa, err := bolsaauditoria.NuevaFuente(d.PoolBolsa)
	if err != nil {
		return nil, err
	}
	servicio, err := auditoria.NuevoServicio(emisorAuditoriaConsultaRRHH{ct: d.EmisorCT, bolsa: d.EmisorBolsa}, ct, bolsa)
	if err != nil {
		return nil, err
	}
	if err := servicio.ConfigurarIntentos(d.Intentos); err != nil {
		return nil, err
	}
	manejador, err := auditoria.NuevoManejador(servicio, d.Opciones, d.Identidad)
	if err != nil {
		return nil, err
	}
	return []vechttp.RutaExacta{
		{Ruta: auditoria.RutaOpciones, Manejador: manejador},
		{Ruta: auditoria.RutaConsulta, Manejador: manejador},
	}, nil
}

func dependenciaAuditoriaConsultaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return r.IsNil()
	}
	return false
}

// No se infiere la fuente de la referencia, del perfil ni de una cabecera.
// La propia solicitud V3, creada por el servicio tras validar el filtro,
// conserva el ámbito exacto y determina un único emisor.
type emisorAuditoriaConsultaRRHH struct {
	ct, bolsa auditoria.EmisorMaterialV3
}

var _ auditoria.EmisorMaterialV3 = emisorAuditoriaConsultaRRHH{}

func (e emisorAuditoriaConsultaRRHH) EmitirMaterialAutorizacionAtestadaV3(
	ctx context.Context,
	s vecdomain.SolicitudAutorizacionLigadaV3,
	c vecdomain.ResultadoContextoActorRegistradoV2,
) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	var decision vecdomain.DecisionAutorizacionLigadaV3
	var confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	if ctx == nil || ctx.Err() != nil || dependenciaAuditoriaConsultaNula(e.ct) || dependenciaAuditoriaConsultaNula(e.bolsa) {
		return decision, confirmacion, nil, auditoria.ErrDenegada
	}
	datos, err := s.Datos()
	if err != nil || datos.Accion != auditoria.AccionConsultar ||
		datos.Recurso.ModuloID != auditoria.ModuloAutorizacion ||
		datos.Recurso.Tipo != auditoria.TipoRecurso || len(datos.Recurso.Ambitos) != 2 ||
		datos.Recurso.Ambitos["expediente_ref"] != datos.Recurso.Referencia ||
		len(datos.Recurso.Atributos) != 1 || !huellaAuditoriaConsultaValida(datos.Recurso.Atributos["filtro_sha256"]) {
		return decision, confirmacion, nil, auditoria.ErrDenegada
	}
	switch datos.Recurso.Ambitos["fuente"] {
	case "ct":
		return e.ct.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
	case "bolsa":
		return e.bolsa.EmitirMaterialAutorizacionAtestadaV3(ctx, s, c)
	default:
		return decision, confirmacion, nil, auditoria.ErrDenegada
	}
}

func huellaAuditoriaConsultaValida(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	for _, caracter := range valor {
		if !((caracter >= '0' && caracter <= '9') || (caracter >= 'a' && caracter <= 'f')) {
			return false
		}
	}
	return true
}

// El registrador CT136 usa LOGIN y transacción propios; la frontera nunca
// transporta certificados, cabeceras ni el cuerpo del intento denegado.
type registradorFronteraAuditoriaConsultaDesarrollo struct{ pool *pgxpool.Pool }

func (r *registradorFronteraAuditoriaConsultaDesarrollo) RegistrarAuditoriaFronteraRutaExacta(ctx context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	if r == nil || r.pool == nil || ctx == nil || ctx.Err() != nil ||
		orden.Superficie != vecports.SuperficieAuditoriaFronteraRutaExactaAuditoria || orden.Validar() != nil {
		return auditoria.ErrNoDisponible
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable, AccessMode: pgx.ReadWrite})
	if err != nil {
		return auditoria.ErrNoDisponible
	}
	defer tx.Rollback(context.Background())
	if _, err := tx.Exec(ctx, `SELECT set_config('search_path','pg_catalog',true),
	 set_config('row_security','on',true),set_config('timezone','UTC',true),
	 set_config('lock_timeout','1s',true),set_config('statement_timeout','2s',true)`); err != nil {
		return auditoria.ErrNoDisponible
	}
	var actor any
	if orden.ActorRef != "" {
		actor = orden.ActorRef
	}
	var registrado bool
	if err := tx.QueryRow(ctx, `SELECT vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1(
	 $1::text,$2::text,$3::text,$4::text,$5::text)`, orden.CorrelacionRef, string(orden.Motivo), orden.Superficie, orden.Ruta, actor).Scan(&registrado); err != nil || !registrado {
		return auditoria.ErrNoDisponible
	}
	if err := tx.Commit(ctx); err != nil {
		return auditoria.ErrNoDisponible
	}
	return nil
}

type registradorFronterasPorSuperficieDesarrollo struct {
	ct, auditoria vecports.RegistradorAuditoriaFronteraRutaExacta
}

func (r registradorFronterasPorSuperficieDesarrollo) RegistrarAuditoriaFronteraRutaExacta(ctx context.Context, orden vecports.OrdenAuditoriaFronteraRutaExacta) error {
	if orden.Validar() != nil {
		return auditoria.ErrDenegada
	}
	switch orden.Superficie {
	case vecports.SuperficieAuditoriaFronteraRutaExactaContratacionTemporal:
		if dependenciaAuditoriaConsultaNula(r.ct) {
			return auditoria.ErrNoDisponible
		}
		return r.ct.RegistrarAuditoriaFronteraRutaExacta(ctx, orden)
	case vecports.SuperficieAuditoriaFronteraRutaExactaAuditoria:
		if dependenciaAuditoriaConsultaNula(r.auditoria) {
			return auditoria.ErrNoDisponible
		}
		return r.auditoria.RegistrarAuditoriaFronteraRutaExacta(ctx, orden)
	default:
		return auditoria.ErrDenegada
	}
}

func nuevoRegistradorFronteraAuditoriaConsultaDesarrollo(ctx context.Context, cfg config.Config) (*registradorFronteraAuditoriaConsultaDesarrollo, func(), error) {
	dsn, err := cfg.DSNFronteraAuditoriaDesarrollo()
	if err != nil {
		return nil, nil, auditoria.ErrNoDisponible
	}
	pool, err := abrirPoolAutoridadAuditoriaDesarrollo(ctx, dsn, config.RolFronteraAuditoriaDesarrollo, "vec-auditoria-frontera")
	if err != nil {
		return nil, nil, err
	}
	if preflightRegistradorFronteraAuditoriaConsultaDesarrollo(ctx, pool) != nil {
		pool.Close()
		return nil, nil, auditoria.ErrNoDisponible
	}
	return &registradorFronteraAuditoriaConsultaDesarrollo{pool: pool}, pool.Close, nil
}

func preflightRegistradorFronteraAuditoriaConsultaDesarrollo(ctx context.Context, q interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) error {
	if ctx == nil || ctx.Err() != nil || q == nil {
		return auditoria.ErrNoDisponible
	}
	const funcion = "vec_contratacion_temporal.registrar_auditoria_frontera_auditoria_v1(text,text,text,text,text)"
	const sql = `WITH propia AS (SELECT to_regprocedure($1)::oid AS oid)
	SELECT p.oid IS NOT NULL AND f.prosecdef
	 AND f.proowner='vec_contratacion_temporal_propietario'::regrole
	 AND coalesce(has_function_privilege(session_user,p.oid,'EXECUTE'),false)
	 AND coalesce(has_function_privilege('vec_contratacion_temporal_registrador_auditoria'::regrole,p.oid,'EXECUTE'),false)
	 AND has_schema_privilege(session_user,'vec_contratacion_temporal','USAGE')
	 AND has_schema_privilege('vec_contratacion_temporal_registrador_auditoria'::regrole,'vec_contratacion_temporal','USAGE')
	 AND NOT has_schema_privilege(session_user,'vec_contratacion_temporal','CREATE')
	 AND NOT has_schema_privilege('vec_contratacion_temporal_registrador_auditoria'::regrole,'vec_contratacion_temporal','CREATE')
	 AND current_setting('transaction_read_only')='off' AND NOT pg_is_in_recovery()
	 AND NOT EXISTS (SELECT 1 FROM pg_proc x
	  WHERE x.pronamespace='vec_contratacion_temporal'::regnamespace AND x.oid<>p.oid
	    AND has_function_privilege(session_user,x.oid,'EXECUTE'))
	 AND NOT EXISTS (SELECT 1 FROM pg_class x
	  WHERE x.relnamespace='vec_contratacion_temporal'::regnamespace AND x.relkind IN ('r','p','v','m')
	    AND (has_table_privilege(session_user,x.oid,'SELECT') OR has_table_privilege(session_user,x.oid,'INSERT')
	     OR has_table_privilege(session_user,x.oid,'UPDATE') OR has_table_privilege(session_user,x.oid,'DELETE')
	     OR has_table_privilege(session_user,x.oid,'TRUNCATE') OR has_table_privilege(session_user,x.oid,'REFERENCES')
	     OR has_table_privilege(session_user,x.oid,'TRIGGER') OR has_table_privilege(session_user,x.oid,'MAINTAIN')
	     OR has_any_column_privilege(session_user,x.oid,'SELECT')
	     OR has_any_column_privilege(session_user,x.oid,'INSERT')
	     OR has_any_column_privilege(session_user,x.oid,'UPDATE')
	     OR has_any_column_privilege(session_user,x.oid,'REFERENCES')))
	 AND NOT EXISTS (SELECT 1 FROM pg_class x
	  WHERE x.relnamespace='vec_contratacion_temporal'::regnamespace AND x.relkind='S'
	    AND (has_sequence_privilege(session_user,x.oid,'USAGE')
	     OR has_sequence_privilege(session_user,x.oid,'SELECT')
	     OR has_sequence_privilege(session_user,x.oid,'UPDATE')))
	 FROM propia p LEFT JOIN pg_proc f ON f.oid=p.oid`
	var valida bool
	if err := q.QueryRow(ctx, sql, funcion).Scan(&valida); err != nil || !valida {
		return auditoria.ErrNoDisponible
	}
	return nil
}

var errAutoridadesAuditoriaConsultaDesarrollo = errors.New("bootstrap: autoridades de consulta de auditoria no disponibles")

const catalogoMotivosAuditoriaConsultaDesarrollo = "motivos_autorizacion_auditoria"

// AD3-91 define una sola audiencia de consumo para ambas fuentes. La clave
// material se publica una vez en el catálogo común; el perfil V3 y el
// consumidor SQL nominal separan CT de Bolsa.
func descriptorMaterialAuditoriaConsultaDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia:        auditoria.AudienciaConsumo,
		Dominio:          "vec.auditoria.consulta-rrhh.desarrollo.capacidad-v3",
		Prefijo:          "clave:capacidad:auditoria-consulta:",
		ProveedorNominal: "proveedor-material-auditoria-consulta",
	}
}

// Ambos discriminadores comparten la cuenta y persona ya acreditadas en la
// raíz CT, pero crean perfiles, vínculos y sesiones distintos entre sí y de
// CT/B-BACK. Ningún identificador procede de parámetros HTTP.
func discriminadorContextoAuditoriaConsultaDesarrollo(fuente string) discriminadorContextoSinteticoDesarrollo {
	etiqueta := "auditoria-consulta-" + fuente + "-v1"
	return discriminadorContextoSinteticoDesarrollo{
		perfil: "perfil-" + etiqueta, vinculo: "vinculo-" + etiqueta,
		procedencia: "procedencia", registro: "registro-contexto-" + etiqueta,
		autenticacion: "autenticacion-" + etiqueta, asercion: "asercion-" + etiqueta,
		sesion: "sesion-" + etiqueta, controlSesion: "control-sesion-" + etiqueta,
		politicaGarantia: "politica-garantia-" + etiqueta,
	}
}

type soportesAuditoriaConsultaDesarrollo struct {
	CT, Bolsa              *soporteAltaContratacionTemporalDesarrollo
	PerfilCT, PerfilBolsa  string
	OperacionContextoCT    string
	OperacionContextoBolsa string
}

// Deriva dos soportes para el proveedor de sesión mTLS existente. La raíz
// publica cada Resultado con su OperacionContexto nominal y crea las sesiones
// con nuevoProveedorSesionConsultaRRHHConCatalogoDesarrollo; nunca sustituye
// la asignación del perfil CT o del perfil B-BACK.
func nuevosSoportesAuditoriaConsultaDesarrollo(
	cfg config.Config, alta *dependenciasAltaContratacionTemporalDesarrollo,
	bolsa *soporteSesionBorradorBolsaDesarrollo,
	reloj relojContratacionTemporalDesarrollo,
) (soportesAuditoriaConsultaDesarrollo, error) {
	vacio := soportesAuditoriaConsultaDesarrollo{}
	cfg = cfg.Normalize()
	if !cfg.DevelopmentEnabledByDoubleKey() || cfg.DevelopmentMaterialDir == "" ||
		alta == nil || alta.soporte == nil || bolsa == nil || bolsa.soporteCanal == nil {
		return vacio, errAutoridadesAuditoriaConsultaDesarrollo
	}
	soportes, err := nuevosSoportesAuditoriaConsultaDesdeBaseDesarrollo(alta.soporte, reloj.Ahora())
	if err != nil || !contextoSinteticoBolsaSeparadoDeCT(alta.soporte.contexto, bolsa.soporteCanal.contexto) ||
		!contextoSinteticoBolsaSeparadoDeCT(bolsa.soporteCanal.contexto, soportes.CT.contexto) ||
		!contextoSinteticoBolsaSeparadoDeCT(bolsa.soporteCanal.contexto, soportes.Bolsa.contexto) {
		return vacio, errAutoridadesAuditoriaConsultaDesarrollo
	}
	return soportes, nil
}

func nuevosSoportesAuditoriaConsultaDesdeBaseDesarrollo(
	base *soporteAltaContratacionTemporalDesarrollo, ahora time.Time,
) (soportesAuditoriaConsultaDesarrollo, error) {
	vacio := soportesAuditoriaConsultaDesarrollo{}
	if base == nil || !ctdomain.InstanteUTCCanonico(ahora) {
		return vacio, errAutoridadesAuditoriaConsultaDesarrollo
	}
	base.mu.Lock()
	principalID, certificado := base.principalID, base.certificadoSHA256
	contextoBase, sello, reloj := base.contexto, base.sello, base.reloj
	base.mu.Unlock()
	if sello == nil || !identificadorSesionDesarrolloValido(principalID) ||
		!contextoSinteticoCTConsistenteParaBorradorBolsa(principalID, certificado, contextoBase) {
		return vacio, errAutoridadesAuditoriaConsultaDesarrollo
	}
	principal := vecdomain.Principal{
		ID: principalID, Roles: []string{rolTecnicoRRHHContratacionTemporalDesarrollo},
		AuthMethod: vecdomain.AuthMethodCertificate, AuthAssurance: vecdomain.AuthAssuranceHigh,
		Attributes: map[string]string{
			"autoridad": AutoridadNoAutoritativa, "perfil_ejecucion": config.ExecutionProfileDevelopment,
			"certificate_sha256": certificado,
		},
	}
	contextoCT, errCT := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, ahora, discriminadorContextoAuditoriaConsultaDesarrollo("ct"))
	contextoBolsa, errBolsa := nuevoContextoSinteticoContratacionTemporalDesarrolloConDiscriminador(
		principal, ahora, discriminadorContextoAuditoriaConsultaDesarrollo("bolsa"))
	if errCT != nil || errBolsa != nil ||
		!contextoSinteticoBolsaSeparadoDeCT(contextoBase, contextoCT) ||
		!contextoSinteticoBolsaSeparadoDeCT(contextoBase, contextoBolsa) ||
		!contextoSinteticoBolsaSeparadoDeCT(contextoCT, contextoBolsa) {
		return vacio, errAutoridadesAuditoriaConsultaDesarrollo
	}
	perfilCT := contextoCT.Resultado.Contexto.PerfilActivoRef
	perfilBolsa := contextoBolsa.Resultado.Contexto.PerfilActivoRef
	if !perfilActivoSeguridadComunValido(perfilCT) || !perfilActivoSeguridadComunValido(perfilBolsa) || perfilCT == perfilBolsa {
		return vacio, errAutoridadesAuditoriaConsultaDesarrollo
	}
	nuevoCanal := func(contexto ctports.ContextoAutorizacionAltaV3) *soporteAltaContratacionTemporalDesarrollo {
		return &soporteAltaContratacionTemporalDesarrollo{
			sello: sello, principalID: principalID, certificadoSHA256: certificado,
			contexto: contexto, reloj: reloj,
		}
	}
	baseOperacion := principalID + "\x00" + certificado + "\x00registro-contexto-auditoria-consulta-"
	return soportesAuditoriaConsultaDesarrollo{
		CT: nuevoCanal(contextoCT), Bolsa: nuevoCanal(contextoBolsa),
		PerfilCT: perfilCT, PerfilBolsa: perfilBolsa,
		OperacionContextoCT:    referenciaAltaContratacionTemporalDesarrollo("oca_", baseOperacion+"ct-v1"),
		OperacionContextoBolsa: referenciaAltaContratacionTemporalDesarrollo("oca_", baseOperacion+"bolsa-v1"),
	}, nil
}

// La raíz llama a esta función solo tras el selector explícito y la doble
// llave. El paquete sigue siendo DEMO; la referencia se valida además contra
// la publicación vigente de la autoridad PostgreSQL, no contra el fichero.
func nuevoProveedorOpcionesAuditoriaConsultaDesarrollo(
	ctx context.Context, cfg config.Config, reloj relojContratacionTemporalDesarrollo,
	validador vecports.ValidadorReferenciaMotivoAutorizacionV2,
) (*auditoria.OpcionesCatalogo, auditoria.Opciones, error) {
	if ctx == nil || ctx.Err() != nil || dependenciaAuditoriaConsultaNula(validador) {
		return nil, auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	ruta, err := cfg.RutaCatalogoAuditoriaConsultaDesarrollo()
	if err != nil {
		return nil, auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	consulta, err := fichero.NuevaConsultaCatalogos(ruta)
	if err != nil {
		return nil, auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	lector, err := reglas.NuevoResolutor(reglas.Configuracion{
		Consulta: consulta, Metadatos: consulta, CatalogoID: catalogoMotivosAuditoriaConsultaDesarrollo,
		ModuloID: auditoria.ModuloAutorizacion, Reloj: reloj,
	})
	if err != nil {
		return nil, auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	proveedor, err := auditoria.NuevoProveedorOpcionesCatalogo(lector, validador)
	if err != nil {
		return nil, auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	opciones, err := proveedor.Actuales(ctx)
	if err != nil || !opciones.EsEjemplo || opciones.PermisoRequerido != auditoria.AccionConsultar {
		return nil, auditoria.Opciones{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	return proveedor, opciones, nil
}

// Un perfil V3 tiene una sola fuente y una lista explícita de expedientes.
// El filtro de fechas, actor y página queda ligado adicionalmente por la
// huella de RecursoFiltro; la decisión solo puede exponer CamposPermitidos.
func instantaneaAuditoriaConsultaNominalDesarrollo(
	principalID, perfilRef, fuente, expedienteRef, finalidad string, ahora time.Time,
) (vecdomain.InstantaneaAutorizacion, error) {
	if (fuente != "ct" && fuente != "bolsa") || expedienteRef == "" || finalidad == "" ||
		!perfilActivoSeguridadComunValido(perfilRef) {
		return vecdomain.InstantaneaAutorizacion{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	instantanea, err := nuevaInstantaneaAutorizacionContratacionTemporalDesarrollo(
		principalID, perfilRef, ahora, "consulta_auditoria_"+fuente+"_desarrollo",
		"Consulta de auditoria "+fuente+" en desarrollo", "consulta-auditoria-"+fuente,
		[]vecdomain.ConcesionRol{{
			Accion: auditoria.AccionConsultar, ModuloID: auditoria.ModuloAutorizacion,
			TipoRecurso: auditoria.TipoRecurso, Finalidades: []string{finalidad},
			GarantiaMinima: vecdomain.AuthAssuranceHigh, CamposPermitidos: auditoria.CamposPermitidos(),
		}},
		[]vecdomain.AmbitoPerfil{
			{Clave: "expediente_ref", Valores: []string{expedienteRef}},
			{Clave: "fuente", Valores: []string{fuente}},
		},
	)
	if err != nil || instantanea.Validar() != nil {
		return vecdomain.InstantaneaAutorizacion{}, errAutoridadesAuditoriaConsultaDesarrollo
	}
	return instantanea, nil
}

// El publicador de la raíz debe instalar ambas instantáneas en la autoridad
// VEC antes de exponer rutas. Esta fábrica usa exclusivamente los lectores y
// registros PostgreSQL centrales y dos proveedores materiales nominales.
type dependenciasAutoridadesAuditoriaConsultaDesarrollo struct {
	FuenteCT, FuenteBolsa         *pgxpool.Pool
	RegistroCT, RegistroBolsa     *pgxpool.Pool
	MotivosCT, MotivosBolsa       *pgxpool.Pool
	MaterialCT, MaterialBolsa     *proveedorMaterialAltaContratacionTemporalDesarrollo
	Reloj                         relojContratacionTemporalDesarrollo
	Opciones                      auditoria.Opciones
	PerfilCT, PerfilBolsa         string
	ExpedienteCT, ExpedienteBolsa string
}

func nuevosEmisoresAuditoriaConsultaDesarrollo(d dependenciasAutoridadesAuditoriaConsultaDesarrollo) (auditoria.EmisorMaterialV3, auditoria.EmisorMaterialV3, error) {
	if d.FuenteCT == nil || d.FuenteBolsa == nil || d.RegistroCT == nil || d.RegistroBolsa == nil ||
		d.MotivosCT == nil || d.MotivosBolsa == nil || d.MaterialCT == nil || d.MaterialBolsa == nil ||
		d.FuenteCT == d.RegistroCT || d.FuenteBolsa == d.RegistroBolsa ||
		!perfilActivoSeguridadComunValido(d.PerfilCT) || !perfilActivoSeguridadComunValido(d.PerfilBolsa) || d.PerfilCT == d.PerfilBolsa ||
		d.Opciones.FinalidadRef == "" || d.Opciones.Motivo.Validar() != nil ||
		d.Opciones.MotivoRef != d.Opciones.Motivo.Referencia() ||
		!d.Opciones.EsEjemplo || d.Opciones.PermisoRequerido != auditoria.AccionConsultar {
		return nil, nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	ct, err := nuevoEmisorAuditoriaConsultaNominalDesarrollo("ct", d.PerfilCT, d.ExpedienteCT,
		d.FuenteCT, d.RegistroCT, d.MotivosCT, d.MaterialCT, d.Opciones, d.Reloj)
	if err != nil {
		return nil, nil, err
	}
	bolsa, err := nuevoEmisorAuditoriaConsultaNominalDesarrollo("bolsa", d.PerfilBolsa, d.ExpedienteBolsa,
		d.FuenteBolsa, d.RegistroBolsa, d.MotivosBolsa, d.MaterialBolsa, d.Opciones, d.Reloj)
	if err != nil {
		return nil, nil, err
	}
	return ct, bolsa, nil
}

func nuevoEmisorAuditoriaConsultaNominalDesarrollo(
	fuente, perfil, expediente string, poolFuente, poolRegistro, poolMotivos *pgxpool.Pool,
	material *proveedorMaterialAltaContratacionTemporalDesarrollo, opciones auditoria.Opciones,
	reloj relojContratacionTemporalDesarrollo,
) (auditoria.EmisorMaterialV3, error) {
	if expediente == "" || material == nil || poolFuente == nil || poolRegistro == nil || poolMotivos == nil ||
		material.emisor == nil || material.atestador == nil || material.confianza == nil {
		return nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	fuenteCentral, err := postgresvec.NuevoAlmacenAutorizacion(poolFuente)
	if err != nil {
		return nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	registroCentral, err := postgresvec.NuevoAlmacenAutorizacion(poolRegistro)
	if err != nil {
		return nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	validador, err := postgresvec.NuevoValidadorReferenciaMotivoPostgreSQLV2(poolMotivos, catalogoMotivosAuditoriaConsultaDesarrollo)
	if err != nil {
		return nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	servicio, err := aplicacionvec.NuevoServicioAutorizacionSolicitudLigadaV3(
		fuenteCentral, registroCentral, registroCentral, validador, reloj,
		seguridadvec.GeneradorReferenciasCriptograficas{},
		aplicacionvec.ConfiguracionServicioAutorizacion{VigenciaDecision: 90 * time.Second},
	)
	if err != nil {
		return nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	guardia := autorizadorAuditoriaConsultaNominalDesarrollo{
		delegado: servicio, fuente: fuente, perfil: perfil, expediente: expediente,
		finalidad: opciones.FinalidadRef, motivo: opciones.Motivo,
	}
	emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(guardia, material)
	if err != nil {
		return nil, errAutoridadesAuditoriaConsultaDesarrollo
	}
	return emisor, nil
}

// La comprobación nominal precede al PDP; la instantánea publicada y el CAS
// centrales vuelven a restringir exactamente esas mismas dimensiones.
type autorizadorAuditoriaConsultaNominalDesarrollo struct {
	delegado                              vecports.AutorizadorSolicitudLigadaV3
	fuente, perfil, expediente, finalidad string
	motivo                                vecdomain.ReferenciaEntradaCatalogo
}

func (a autorizadorAuditoriaConsultaNominalDesarrollo) ExigirSolicitudLigadaV3(
	ctx context.Context, solicitud vecdomain.SolicitudAutorizacionLigadaV3,
	resultado vecdomain.ResultadoContextoActorRegistradoV2,
) (vecdomain.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	vacia := vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}
	datos, err := solicitud.Datos()
	if ctx == nil || ctx.Err() != nil || dependenciaAuditoriaConsultaNula(a.delegado) || err != nil ||
		resultado.Validar() != nil || resultado.Contexto.PerfilActivoRef != a.perfil ||
		datos.Accion != auditoria.AccionConsultar || datos.Recurso.ModuloID != auditoria.ModuloAutorizacion ||
		datos.Recurso.Tipo != auditoria.TipoRecurso || datos.Recurso.Referencia != a.expediente ||
		len(datos.Recurso.Ambitos) != 2 || datos.Recurso.Ambitos["fuente"] != a.fuente ||
		datos.Recurso.Ambitos["expediente_ref"] != a.expediente ||
		len(datos.Recurso.Atributos) != 1 || !huellaAuditoriaConsultaValida(datos.Recurso.Atributos["filtro_sha256"]) ||
		datos.Finalidad != a.finalidad || datos.ReferenciaMotivo != a.motivo {
		return vecdomain.DecisionAutorizacionLigadaV3{}, vacia, auditoria.ErrDenegada
	}
	return a.delegado.ExigirSolicitudLigadaV3(ctx, solicitud, resultado)
}

const envRRHHAuditoriaEnabled = "VEC_RRHH_AUDITORIA_ENABLED"

const (
	claveFronteraOpcionesAuditoriaRRHH = "rrhh-auditoria-opciones"
	claveFronteraConsultaAuditoriaRRHH = "rrhh-auditoria-consultar"
	clavePoliticaAuditoriaRRHH         = "politica-rrhh-auditoria"
	claveCapacidadOpcionesAuditoria    = "capacidad-rrhh-auditoria-opciones"
	claveCapacidadConsultaAuditoria    = "capacidad-rrhh-auditoria-consultar"
)

// GET expone solo los parámetros del catálogo. POST admite dos perfiles
// nominales distintos; la fuente tipada determina qué identidad y emisor V3
// se usan antes de consultar el almacenamiento propietario.
func descriptoresFronterasAuditoriaRRHHDesarrollo(perfilCT, perfilBolsa string) ([]descriptorFronteraComunDesarrollo, error) {
	if !perfilActivoSeguridadComunValido(perfilCT) || !perfilActivoSeguridadComunValido(perfilBolsa) || perfilCT == perfilBolsa {
		return nil, ErrSeguridadComunDesarrolloDenegada
	}
	return []descriptorFronteraComunDesarrollo{
		{Clave: claveFronteraOpcionesAuditoriaRRHH, Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodGet, Ruta: auditoria.RutaOpciones, PerfilesActivosRef: []string{perfilCT},
			ClavePolitica: clavePoliticaAuditoriaRRHH, ClaveCapacidad: claveCapacidadOpcionesAuditoria},
		{Clave: claveFronteraConsultaAuditoriaRRHH, Superficie: superficieInternaSeguridadComunDesarrollo,
			Metodo: http.MethodPost, Ruta: auditoria.RutaConsulta, PerfilesActivosRef: []string{perfilCT, perfilBolsa},
			ClavePolitica: clavePoliticaAuditoriaRRHH, ClaveCapacidad: claveCapacidadConsultaAuditoria},
	}, nil
}
