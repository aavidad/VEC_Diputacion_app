package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	almacenvec "vec-diputacion-granada/internal/vec/adapters/almacen"
	"vec-diputacion-granada/internal/vec/adapters/conservacion"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	"vec-diputacion-granada/internal/vec/adapters/observabilidad"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	dochttp "vec-diputacion-granada/internal/vec/documentos/adapters/httpinterno"
	docpg "vec-diputacion-granada/internal/vec/documentos/adapters/postgres"
	docapp "vec-diputacion-granada/internal/vec/documentos/application"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrComposicionDocumentosNoDisponible = errors.New("bootstrap: Documentos no disponible")

const (
	prefijoRutasDocumentos     = "/api/vec/documentos/"
	ficheroMaterialDocumentos  = "documentos.json"
	finalidadListarDocumentos  = "listar_documentos_expediente"
	finalidadDescargaDocumento = "descargar_documento_original"
	rolEjecutorDocumentos      = "vec_documentos_ejecutor"
	rolAuditorDocumentos       = "vec_documentos_auditor"
)

// configuracionDocumentosDesarrollo es el material privado del montaje
// (identidad/documentos.json, fuera de Git): cuentas ya registradas, DSN
// nominales, motivos del catálogo de autorización y almacén de originales.
// Las claves V3 no viven aquí: las publica el gobierno único de desarrollo.
type configuracionDocumentosDesarrollo struct {
	redaccionMaterialRutasDietas
	Version                  int                           `json:"version"`
	Autoridad                string                        `json:"autoridad"`
	Cuentas                  []cuentaRutasDietasDesarrollo `json:"cuentas"`
	DSNRegistroIdentidad     string                        `json:"dsn_registro_identidad"`
	DSNRevalidacionIdentidad string                        `json:"dsn_revalidacion_identidad"`
	DSNContexto              string                        `json:"dsn_contexto"`
	DSNFuenteAutorizacion    string                        `json:"dsn_fuente_autorizacion"`
	DSNRegistroAutorizacion  string                        `json:"dsn_registro_autorizacion"`
	DSNMotivos               string                        `json:"dsn_motivos"`
	DSNDocumentos            string                        `json:"dsn_documentos"`
	DSNDocumentosAuditor     string                        `json:"dsn_documentos_auditor"`
	Motivos                  struct {
		Listar core.ReferenciaEntradaCatalogo `json:"listar"`
	} `json:"motivos"`
	Almacen almacenDocumentosDesarrollo `json:"almacen"`
}

// almacenDocumentosDesarrollo selecciona el conector de originales. El
// predeterminado es "ficheros" (no depende de otra aplicación); "s3" queda
// como alternativa y sólo se usa si el material privado lo pide.
type almacenDocumentosDesarrollo struct {
	Tipo                string            `json:"tipo"`
	Directorio          string            `json:"directorio"`
	TamanoMaximo        int64             `json:"tamano_maximo"`
	RetencionMinimaDias int64             `json:"retencion_minima_dias"`
	S3                  map[string]string `json:"s3"`
}

// autoridadDocumentosDesarrollo es la frontera mTLS de /api/vec/documentos/.
// Resuelve identidad y sesión registradas, audita toda denegación con el
// LOGIN auditor propio y deja la autorización de cada efecto al PDP V3.
type autoridadDocumentosDesarrollo struct {
	base        *autoridadRutasDietasDesarrollo
	reloj       vecports.Reloj
	cuentas     map[string]cuentaRutasDietasDesarrollo
	rutas       []vechttp.RutaExacta
	publicadas  map[string]bool
	registrador registradorDenegacionesDocumentos
	incidencias vecports.EmisorIncidenciasTecnicas
	cerrar      func()
}

type registradorDenegacionesDocumentos interface {
	RegistrarDenegacion(context.Context, docpg.OrdenDenegacionFrontera) error
}

type claveContextoDocumentos struct{}

type contextoDocumentos struct {
	autoridad *autoridadDocumentosDesarrollo
	ruta      string
	seguridad contextoSeguridadComunDesarrollo
}

func esRutaDocumentos(ruta string) bool {
	return ruta+"/" == prefijoRutasDocumentos || strings.HasPrefix(ruta, prefijoRutasDocumentos)
}

func (a *autoridadDocumentosDesarrollo) proteger(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			responderDenegacionDocumentos(w, http.StatusServiceUnavailable, "servicio_no_disponible")
			return
		}
		if !esRutaDocumentos(r.URL.Path) {
			siguiente.ServeHTTP(w, r)
			return
		}
		publicada := a != nil && a.publicadas[r.URL.Path] && r.URL.RawPath == ""
		// Sin cadena mTLS verificada no hay identidad: se responde sin
		// escritura durable para que un anónimo no amplifique la bitácora.
		if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.VerifiedChains[0]) == 0 {
			if !publicada {
				responderDenegacionDocumentos(w, http.StatusNotFound, "recurso_no_encontrado")
				return
			}
			responderDenegacionDocumentos(w, http.StatusUnauthorized, "autenticacion_requerida")
			return
		}
		if a == nil || a.base == nil || cabeceraLibreComisionesDietas(r.Header) {
			a.denegar(w, r, http.StatusUnauthorized, docpg.MotivoFronteraAutenticacion, "autenticacion_requerida", "")
			return
		}
		if !publicada {
			a.denegar(w, r, http.StatusNotFound, docpg.MotivoFronteraDenegado, "recurso_no_encontrado", "")
			return
		}
		principal, err := a.base.resolvedor.ResolveDemoIdentity(r.Context(), r)
		ahora := a.reloj.Ahora()
		cert := r.TLS.VerifiedChains[0][0]
		if err != nil || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh ||
			ahora.Before(cert.NotBefore) || !ahora.Before(cert.NotAfter) {
			a.denegar(w, r, http.StatusUnauthorized, docpg.MotivoFronteraAutenticacion, "autenticacion_requerida", "")
			return
		}
		cuenta, ok := a.cuentas[principal.Attributes["certificate_sha256"]]
		if !ok || cuenta.Sujeto != principal.ID {
			a.denegar(w, r, http.StatusForbidden, docpg.MotivoFronteraDenegado, "acceso_denegado", "")
			return
		}
		vinculo, resultado, err := a.base.resolverSesion(r.Context(), r, &capsulaRutasDietasDesarrollo{autoridad: a.base, peticion: r, cuenta: cuenta, instante: ahora})
		if err != nil {
			if errors.Is(err, vechttp.ErrRutaDietasDenegada) {
				a.denegar(w, r, http.StatusForbidden, docpg.MotivoFronteraDenegado, "acceso_denegado", "")
				return
			}
			a.denegar(w, r, http.StatusServiceUnavailable, docpg.MotivoFronteraDependencia, "servicio_no_disponible", "")
			return
		}
		actorRef := actorVerificadoComisionesDietas(vinculo, resultado, a.reloj.Ahora())
		if actorRef == "" {
			a.denegar(w, r, http.StatusServiceUnavailable, docpg.MotivoFronteraDependencia, "servicio_no_disponible", "")
			return
		}
		ctx := context.WithValue(r.Context(), claveContextoDocumentos{}, contextoDocumentos{autoridad: a, ruta: r.URL.Path,
			seguridad: contextoSeguridadComunDesarrollo{Vinculo: vinculo, Resultado: resultado}})
		siguiente.ServeHTTP(w, r.WithContext(ctx))
	})
}

// denegar registra la denegación con el LOGIN auditor antes de responder. Si
// el registro no se confirma, responde indisponibilidad y lo declara como
// incidencia técnica: ninguna denegación queda sin rastro.
func (a *autoridadDocumentosDesarrollo) denegar(w http.ResponseWriter, r *http.Request, estado int, motivo, codigo, actorRef string) {
	if a == nil || a.registrador == nil {
		responderDenegacionDocumentos(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	ruta := docpg.RutaFronteraOtra
	if a.publicadas[r.URL.Path] && r.URL.RawPath == "" {
		ruta = r.URL.Path
	}
	metodo := "otro"
	if r.Method == http.MethodPost {
		metodo = http.MethodPost
	}
	var aleatorio [16]byte
	correlacion := "corr_no_disponible"
	if _, err := rand.Read(aleatorio[:]); err == nil {
		correlacion = "corr_" + hex.EncodeToString(aleatorio[:])
	}
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
	defer cancelar()
	if err := a.registrador.RegistrarDenegacion(ctx, docpg.OrdenDenegacionFrontera{CorrelacionRef: correlacion, Motivo: motivo,
		Ruta: ruta, Metodo: metodo, ActorRef: actorRef}); err != nil {
		a.incidencia(core.IncidenciaAuditoriaNoRegistrada, core.ComponenteIncidenciaAuditoria, core.EtapaIncidenciaRegistro)
		responderDenegacionDocumentos(w, http.StatusServiceUnavailable, "servicio_no_disponible")
		return
	}
	if estado >= http.StatusInternalServerError {
		a.incidencia(core.IncidenciaHTTPInternoFallido, core.ComponenteIncidenciaHTTP, core.EtapaIncidenciaPeticion)
	}
	responderDenegacionDocumentos(w, estado, codigo)
}

func (a *autoridadDocumentosDesarrollo) incidencia(codigo core.CodigoIncidenciaTecnica, componente core.ComponenteIncidenciaTecnica, etapa core.EtapaIncidenciaTecnica) {
	if a != nil && a.incidencias != nil {
		a.incidencias.Emitir(core.SolicitudIncidenciaTecnica{Codigo: codigo, Componente: componente, Etapa: etapa})
	}
}

// responderDenegacionDocumentos usa el mismo sobre de error que el adaptador
// HTTP documental; nunca incluye detalle interno.
func responderDenegacionDocumentos(w http.ResponseWriter, estado int, codigo string) {
	contenido, _ := json.Marshal(map[string]any{"error": map[string]string{"codigo": codigo, "clave_i18n": "api.documentos.error." + codigo}})
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(contenido)))
	w.Header().Set("Cache-Control", "no-store, no-transform")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'")
	w.WriteHeader(estado)
	_, _ = w.Write(contenido)
}

// autoridadExactasConDocumentos encadena la autoridad del despachador único:
// las rutas documentales sólo pasan con el contexto que esta frontera fijó
// para la misma petición y ruta.
type autoridadExactasConDocumentos struct {
	delegada   vechttp.AutoridadRutasExactas
	documentos *autoridadDocumentosDesarrollo
}

func (a autoridadExactasConDocumentos) AutorizarRutaExacta(ctx context.Context, ruta string) error {
	if !esRutaDocumentos(ruta) {
		if a.delegada == nil {
			return vechttp.ErrAutenticacionRutaExactaRequerida
		}
		return a.delegada.AutorizarRutaExacta(ctx, ruta)
	}
	if ctx == nil || a.documentos == nil {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	c, ok := ctx.Value(claveContextoDocumentos{}).(contextoDocumentos)
	if !ok || c.autoridad != a.documentos || c.ruta != ruta || c.seguridad.Resultado.Validar() != nil ||
		!c.seguridad.Vinculo.VigenteEn(a.documentos.reloj.Ahora(), c.seguridad.Resultado) {
		return vechttp.ErrAccesoRutaExactaDenegado
	}
	return nil
}

// autoridadConsultaDocumentos obtiene del PDP V3 la concesión ligada a la
// preimagen exacta de cada consulta, con la identidad que la frontera
// registró para la misma petición.
type autoridadConsultaDocumentos struct {
	autoridad *autoridadDocumentosDesarrollo
	emisor    emisorMaterialDietasDesarrollo
	motivo    core.ReferenciaEntradaCatalogo
}

func (a autoridadConsultaDocumentos) ResolverConsultaExpediente(ctx context.Context, c docports.ConsultaExpediente) (docports.AutorizacionV3, error) {
	preimagen, err := c.PreimagenListar()
	if err != nil {
		return docports.AutorizacionV3{}, docports.ErrSolicitudInvalida
	}
	return a.autorizar(ctx, docports.AccionListar, finalidadListarDocumentos, c.ExpedienteRef, c.ExpedienteRef, preimagen)
}

// ResolverDescargaOriginal existe por contrato, pero la composición no
// publica la descarga mientras no haya autoridad de lectura del almacén.
func (a autoridadConsultaDocumentos) ResolverDescargaOriginal(ctx context.Context, c docports.ConsultaDocumento, expediente string) (docports.AutorizacionV3, error) {
	preimagen, err := c.PreimagenDescargar()
	if err != nil {
		return docports.AutorizacionV3{}, docports.ErrSolicitudInvalida
	}
	return a.autorizar(ctx, docports.AccionDescargar, finalidadDescargaDocumento, c.DocumentoID, expediente, preimagen)
}

func (a autoridadConsultaDocumentos) autorizar(ctx context.Context, accion, finalidad, recursoRef, ambito string, preimagen []byte) (docports.AutorizacionV3, error) {
	vacia := docports.AutorizacionV3{}
	if ctx == nil || a.autoridad == nil || a.emisor == nil {
		return vacia, docports.ErrCapacidadNoDisponible
	}
	c, ok := ctx.Value(claveContextoDocumentos{}).(contextoDocumentos)
	if !ok || c.autoridad != a.autoridad || c.seguridad.Resultado.Validar() != nil ||
		!c.seguridad.Vinculo.VigenteEn(a.autoridad.reloj.Ahora(), c.seguridad.Resultado) {
		return vacia, docports.ErrAccesoDenegado
	}
	datos, err := c.seguridad.Vinculo.Datos()
	if err != nil {
		return vacia, docports.ErrAccesoDenegado
	}
	recurso, err := docports.RecursoV3(accion, recursoRef, preimagen)
	if err != nil {
		return vacia, docports.ErrSolicitudInvalida
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return vacia, fmt.Errorf("%w: correlación", docports.ErrCapacidadNoDisponible)
	}
	correlacionRef, err := correlacion.ValorCanonico()
	if err != nil {
		return vacia, fmt.Errorf("%w: correlación", docports.ErrCapacidadNoDisponible)
	}
	solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: c.seguridad.Vinculo, ReferenciaMotivo: a.motivo, Accion: accion,
		Recurso: recurso, Finalidad: finalidad, Correlacion: correlacion,
	})
	if err != nil {
		return vacia, docports.ErrAccesoDenegado
	}
	decision, confirmacion, exportador, err := a.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, c.seguridad.Resultado)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return vacia, err
		}
		if errors.Is(err, core.ErrAutorizacionDenegada) {
			return vacia, docports.ErrAccesoDenegado
		}
		a.autoridad.incidencia(core.IncidenciaGobiernoV3NoDisponible, core.ComponenteIncidenciaGobiernoV3, core.EtapaIncidenciaConsulta)
		return vacia, fmt.Errorf("%w: emisión V3", docports.ErrCapacidadNoDisponible)
	}
	if decision.ValidarPara(solicitud) != nil || exportador == nil {
		return vacia, docports.ErrAccesoDenegado
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(solicitud, decision, confirmacion, c.seguridad.Resultado, a.motivo, material, docports.AudienciaV3) {
		return vacia, docports.ErrAccesoDenegado
	}
	return docports.AutorizacionV3{Material: material, Accion: accion, Finalidad: finalidad, RecursoRef: recursoRef,
		AmbitoRef: ambito, PrincipalID: datos.PrincipalID, PerfilActivoRef: datos.PerfilActivoRef, CorrelacionRef: correlacionRef}, nil
}

// servicioLecturaVigilado declara la indisponibilidad de PostgreSQL como
// incidencia técnica sin alterar el error devuelto.
type servicioLecturaVigilado struct {
	servicio    dochttp.ServicioLectura
	incidencias vecports.EmisorIncidenciasTecnicas
}

func (s servicioLecturaVigilado) vigilar(err error) {
	if err != nil && s.incidencias != nil && errors.Is(err, docpg.ErrRepositorioNoDisponible) &&
		!errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		s.incidencias.Emitir(core.SolicitudIncidenciaTecnica{Codigo: core.IncidenciaPostgresNoDisponible,
			Componente: core.ComponenteIncidenciaPostgreSQL, Etapa: core.EtapaIncidenciaConsulta})
	}
}

func (s servicioLecturaVigilado) ListarExpediente(ctx context.Context, c docports.ConsultaExpediente) (docports.PaginaDocumentos, error) {
	p, err := s.servicio.ListarExpediente(ctx, c)
	s.vigilar(err)
	return p, err
}

func (s servicioLecturaVigilado) DescargarOriginal(ctx context.Context, c docports.ConsultaDocumento) (docports.Original, error) {
	o, err := s.servicio.DescargarOriginal(ctx, c)
	s.vigilar(err)
	return o, err
}

// abrirPoolDocumentos exige un LOGIN nominal con una única membresía, la del
// rol técnico indicado, sin privilegios ni SET/ADMIN, y TLS válido.
func abrirPoolDocumentos(ctx context.Context, dsn, rol string) (*pgxpool.Pool, string, error) {
	if dsn == "" || (rol != rolEjecutorDocumentos && rol != rolAuditorDocumentos) {
		return nil, "", ErrComposicionDocumentosNoDisponible
	}
	c, err := pgxpool.ParseConfig(dsn)
	if err != nil || c.ConnConfig.User == "" || validarTLSPostgreSQLBorradores(&c.ConnConfig.Config, true) != nil {
		return nil, "", ErrComposicionDocumentosNoDisponible
	}
	c.MaxConns = 4
	c.MinConns = 0
	c.ConnConfig.ConnectTimeout = 5 * time.Second
	if c.ConnConfig.RuntimeParams == nil {
		c.ConnConfig.RuntimeParams = map[string]string{}
	}
	for k, v := range map[string]string{"application_name": "vec-documentos-desarrollo", "timezone": "UTC", "search_path": "pg_catalog",
		"statement_timeout": "10s", "lock_timeout": "2s", "idle_in_transaction_session_timeout": "15s"} {
		c.ConnConfig.RuntimeParams[k] = v
	}
	pool, err := pgxpool.NewWithConfig(ctx, c)
	if err != nil {
		return nil, "", ErrComposicionDocumentosNoDisponible
	}
	var usuario string
	var valido bool
	err = pool.QueryRow(ctx, `SELECT session_user::text,
 session_user=current_user AND r.rolcanlogin AND r.rolinherit AND NOT(r.rolsuper OR r.rolcreatedb OR r.rolcreaterole OR r.rolreplication OR r.rolbypassrls)
 AND (SELECT count(*)=1 FROM pg_catalog.pg_auth_members m WHERE m.member=r.oid)
 AND EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members m JOIN pg_catalog.pg_roles g ON g.oid=m.roleid WHERE m.member=r.oid AND g.rolname=$1
   AND m.inherit_option AND NOT m.set_option AND NOT m.admin_option
   AND NOT(g.rolcanlogin OR g.rolsuper OR g.rolcreatedb OR g.rolcreaterole OR g.rolreplication OR g.rolbypassrls)
   AND NOT EXISTS(SELECT 1 FROM pg_catalog.pg_auth_members superior WHERE superior.member=g.oid))
 FROM pg_catalog.pg_roles r WHERE r.rolname=session_user`, rol).Scan(&usuario, &valido)
	if err != nil || !valido {
		pool.Close()
		return nil, "", ErrComposicionDocumentosNoDisponible
	}
	return pool, usuario, nil
}

// preflightDocumentos comprueba con el LOGIN ejecutor que las fachadas de
// Documentos 1–4 existen y le están concedidas, y que no alcanza el registro
// de denegaciones. Una función ausente hace fallar la consulta.
func preflightDocumentos(ctx context.Context, ejecutor *pgxpool.Pool) error {
	var ok bool
	if ejecutor.QueryRow(ctx, `SELECT bool_and(has_function_privilege(f,'EXECUTE')) FROM unnest(ARRAY[
  'vec_documentos.listar_expediente_v2(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_documentos.obtener_original_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_documentos.confirmar_alta_v2(bytea,jsonb,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_documentos.registrar_referencia_externa_v1(bytea,jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)']) f
 WHERE to_regprocedure('vec_documentos.huella_efecto_v1(bytea)') IS NOT NULL
   AND NOT has_function_privilege('vec_documentos.registrar_denegacion_frontera_v1(text,text,text,text,text)','EXECUTE')`).Scan(&ok) != nil || !ok {
		return ErrComposicionDocumentosNoDisponible
	}
	return nil
}

// nuevoAlmacenDocumentos compone el conector de originales elegido por el
// material privado y verifica sus capacidades mínimas al arrancar.
func nuevoAlmacenDocumentos(ctx context.Context, a almacenDocumentosDesarrollo, reloj vecports.Reloj) (vecports.AlmacenObjetos, func(), error) {
	registro := almacenvec.NuevoRegistroConectoresAlmacen()
	requisitos := vecports.RequisitosAlmacenObjetos{EscrituraEnFlujo: true, LecturaEnFlujo: true, ReferenciasOpacas: true,
		IntegridadSHA256: true, Retencion: true}
	var (
		identificador string
		valores       almacenvec.ConfiguracionConectorAlmacen
	)
	switch a.Tipo {
	case "", "ficheros":
		if len(a.S3) != 0 || a.TamanoMaximo < 1 || a.RetencionMinimaDias < 1 {
			return nil, nil, ErrComposicionDocumentosNoDisponible
		}
		identificador = "ficheros-local"
		if almacenvec.RegistrarFicheros(registro, identificador, reloj) != nil {
			return nil, nil, ErrComposicionDocumentosNoDisponible
		}
		valores = almacenvec.ConfiguracionConectorAlmacen{"directorio": a.Directorio,
			"tamano_maximo": strconv.FormatInt(a.TamanoMaximo, 10), "retencion_minima_dias": strconv.FormatInt(a.RetencionMinimaDias, 10)}
	case "s3":
		if a.Directorio != "" || len(a.S3) == 0 {
			return nil, nil, ErrComposicionDocumentosNoDisponible
		}
		identificador = "s3-documentos"
		if almacenvec.RegistrarS3Compatible(registro, identificador) != nil {
			return nil, nil, ErrComposicionDocumentosNoDisponible
		}
		valores = almacenvec.ConfiguracionConectorAlmacen(a.S3)
	default:
		return nil, nil, ErrComposicionDocumentosNoDisponible
	}
	almacen, err := registro.Crear(ctx, identificador, valores, requisitos)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: almacén de originales", ErrComposicionDocumentosNoDisponible)
	}
	cerrar := func() {}
	if cerrable, ok := almacen.(interface{ Cerrar() error }); ok {
		cerrar = func() { _ = cerrable.Cerrar() }
	}
	return almacen, cerrar, nil
}

// nuevosDocumentosDesarrollo devuelve nil con el selector apagado. Con él
// activado, cualquier pieza ausente o incoherente impide arrancar.
func nuevosDocumentosDesarrollo(cfg config.Config, resolvedor vechttp.DemoIdentityResolver, derivador *derivadorIdentidadOperacionDesarrollo,
	material *proveedorMaterialAltaContratacionTemporalDesarrollo, registroIncidencias io.Writer,
) (*autoridadDocumentosDesarrollo, error) {
	activo, err := cfg.DocumentosDesarrolloActivo()
	if err != nil {
		return nil, err
	}
	if !activo {
		return nil, nil
	}
	identidad, ok := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !ok || identidad == nil || derivador == nil || !derivador.valido() || material == nil {
		return nil, errDocumentosEn()
	}
	contenido, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", ficheroMaterialDocumentos), 256<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return nil, errDocumentosEn()
	}
	defer borrarBytes(contenido)
	var c configuracionDocumentosDesarrollo
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa ||
		len(c.Cuentas) == 0 || len(c.Cuentas) > 64 || c.Motivos.Listar.Validar() != nil {
		return nil, errDocumentosEn()
	}
	cuentas := map[string]cuentaRutasDietasDesarrollo{}
	for _, cuenta := range c.Cuentas {
		b, e := hex.DecodeString(cuenta.CertificadoSHA256)
		if e != nil || len(b) != sha256.Size || hex.EncodeToString(b) != cuenta.CertificadoSHA256 || cuenta.Sujeto == "" || cuenta.CuentaRef == "" || cuenta.PerfilRef == "" {
			return nil, errDocumentosEn()
		}
		var digest [32]byte
		copy(digest[:], b)
		principal, existe := identidad.porHuella[digest]
		if !existe || principal.ID != cuenta.Sujeto || principal.AuthMethod != core.AuthMethodCertificate ||
			principal.AuthAssurance != core.AuthAssuranceHigh || cuentas[cuenta.CertificadoSHA256].Sujeto != "" {
			return nil, errDocumentosEn()
		}
		cuentas[cuenta.CertificadoSHA256] = cuenta
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	var pools []*pgxpool.Pool
	var cierres []func()
	var unaVez sync.Once
	cerrar := func() {
		unaVez.Do(func() {
			for i := len(cierres) - 1; i >= 0; i-- {
				cierres[i]()
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
	usuarios := map[string]bool{}
	for _, entrada := range []struct{ dsn, rol string }{
		{c.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"}, {c.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"},
		{c.DSNContexto, "vec_contexto_actor_v1_runtime"}, {c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"},
		{c.DSNRegistroAutorizacion, "vec_autorizacion_registro"}, {c.DSNMotivos, "vec_autorizacion_motivos_evaluador"},
	} {
		pool, usuario, e := abrirPoolRutasDietas(ctx, entrada.dsn, entrada.rol)
		if e != nil {
			return nil, errDocumentosEn()
		}
		pools = append(pools, pool)
		if usuarios[usuario] {
			return nil, errDocumentosEn()
		}
		usuarios[usuario] = true
	}
	for _, entrada := range []struct{ dsn, rol string }{{c.DSNDocumentos, rolEjecutorDocumentos}, {c.DSNDocumentosAuditor, rolAuditorDocumentos}} {
		pool, usuario, e := abrirPoolDocumentos(ctx, entrada.dsn, entrada.rol)
		if e != nil {
			return nil, errDocumentosEn()
		}
		pools = append(pools, pool)
		if usuarios[usuario] {
			return nil, errDocumentosEn()
		}
		usuarios[usuario] = true
	}
	ejecutor, auditor := pools[6], pools[7]
	if preflightDocumentos(ctx, ejecutor) != nil {
		return nil, errDocumentosEn()
	}
	registrador, err := docpg.NuevoRegistradorFrontera(auditor)
	if err != nil || registrador.Preflight(ctx) != nil {
		return nil, errDocumentosEn()
	}
	repositorio, err := docpg.NuevoRepositorio(ejecutor)
	if err != nil {
		return nil, errDocumentosEn()
	}
	reloj := relojRutasDietas{}
	almacen, cerrarAlmacen, err := nuevoAlmacenDocumentos(ctx, c.Almacen, reloj)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errDocumentosEn(), err)
	}
	cierres = append(cierres, cerrarAlmacen)
	politicas, err := conservacion.NuevoCatalogoProvisional(reloj)
	if err != nil {
		return nil, errDocumentosEn()
	}
	registroSesiones, err := identidadpg.NuevoRegistroSesionesPostgreSQL(ctx, pools[0], pools[1], &seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
	if err != nil {
		return nil, errDocumentosEn()
	}
	revalidador, err := identidadpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, pools[1])
	if err != nil {
		return nil, errDocumentosEn()
	}
	resolutor, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pools[2])
	if err != nil {
		return nil, errDocumentosEn()
	}
	// RRHH consulta documentos sin exigir proyección de empleado.
	servicioContexto, err := vecapp.NuevoServicioContextoActorProductivoV2(resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj)
	if err != nil {
		return nil, errDocumentosEn()
	}
	contextos, err := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if err != nil {
		return nil, errDocumentosEn()
	}
	fuente, err := vecpg.NuevoAlmacenAutorizacion(pools[3])
	if err != nil {
		return nil, errDocumentosEn()
	}
	registroAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(pools[4])
	if err != nil {
		return nil, errDocumentosEn()
	}
	validadorMotivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools[5], c.Motivos.Listar.CatalogoID)
	if err != nil {
		return nil, errDocumentosEn()
	}
	autorizador, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registroAutorizacion, registroAutorizacion, validadorMotivos, reloj,
		seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if err != nil {
		return nil, errDocumentosEn()
	}
	emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, material)
	if err != nil {
		return nil, errDocumentosEn()
	}
	nonce, err := nonceRutasDietas()
	if err != nil {
		return nil, errDocumentosEn()
	}
	incidencias, err := observabilidad.NuevoEmisorJSONLines(observabilidad.OpcionesEmisor{Destino: registroIncidenciasSeguro(registroIncidencias),
		Capacidad: 256, Entorno: os.Getenv("VEC_ENTORNO")})
	if err != nil {
		return nil, errDocumentosEn()
	}
	cierres = append(cierres, func() {
		ctxCierre, cancelarCierre := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelarCierre()
		_ = incidencias.Cerrar(ctxCierre)
	})
	base := &autoridadRutasDietasDesarrollo{resolvedor: identidad, cuentas: cuentas, registro: registroSesiones, revalidador: revalidador, contextos: contextos, reloj: reloj, instancia: nonce}
	a := &autoridadDocumentosDesarrollo{base: base, reloj: reloj, cuentas: cuentas, registrador: registrador, incidencias: incidencias, cerrar: cerrar}
	servicio := &docapp.Servicio{Repositorio: repositorio, Almacen: almacen, Politicas: politicas, Reloj: reloj}
	rutas, err := dochttp.NuevasRutas(dochttp.Configuracion{
		Servicio:  servicioLecturaVigilado{servicio: servicio, incidencias: incidencias},
		Autoridad: autoridadConsultaDocumentos{autoridad: a, emisor: emisor, motivo: c.Motivos.Listar},
		// La descarga exige una decisión de almacén propia para leer el
		// original; su autoridad aún no está compuesta en la raíz.
		Incidencias: incidencias, Tipos: politicas, DescargaDisponible: false,
	})
	if err != nil {
		return nil, errDocumentosEn()
	}
	a.rutas = rutas
	a.publicadas = map[string]bool{}
	for _, ruta := range rutas {
		a.publicadas[ruta.Ruta] = true
	}
	completa = true
	return a, nil
}

// registroIncidenciasSeguro garantiza un destino: sin registro de la
// composición, las incidencias van a la salida estándar del proceso.
func registroIncidenciasSeguro(w io.Writer) io.Writer {
	if w == nil {
		return os.Stdout
	}
	return w
}

// errDocumentosEn añade sólo fichero y línea del rechazo, sin DSN,
// identidades ni material.
func errDocumentosEn() error {
	_, fichero, linea, ok := runtime.Caller(1)
	if !ok {
		return ErrComposicionDocumentosNoDisponible
	}
	return fmt.Errorf("%w (%s:%d)", ErrComposicionDocumentosNoDisponible, filepath.Base(fichero), linea)
}
