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
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/config"
	cronoscomp "vec-diputacion-granada/internal/modules/cronos/adapters/composicion"
	cronoshttp "vec-diputacion-granada/internal/modules/cronos/adapters/httpinterno"
	cronospg "vec-diputacion-granada/internal/modules/cronos/adapters/postgres"
	cronosapp "vec-diputacion-granada/internal/modules/cronos/application"
	cronosdomain "vec-diputacion-granada/internal/modules/cronos/domain"
	cronosports "vec-diputacion-granada/internal/modules/cronos/ports"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrComposicionCronosEmpleadoNoDisponible = errors.New("bootstrap: Cronos para la persona empleada no disponible")

const prefijoRutasCronosEmpleado = "/api/interna/cronos/"

// El archivo privado sólo selecciona cuentas registradas, conexiones
// nominales, motivos y la política del canal remoto. Las claves V3 son las del
// gobierno único que publica Contratación. Con el selector activado, su
// ausencia o cualquier dependencia incompleta impide arrancar.
type configuracionCronosEmpleadoDesarrollo struct {
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
	DSNCronos                string                        `json:"dsn_cronos"`
	DSNCronosAuditor         string                        `json:"dsn_cronos_auditor"`
	ZonaHoraria              string                        `json:"zona_horaria"`
	Motivos                  struct {
		Saldo          core.ReferenciaEntradaCatalogo `json:"saldo"`
		Marcaje        core.ReferenciaEntradaCatalogo `json:"marcaje"`
		Disponibilidad core.ReferenciaEntradaCatalogo `json:"disponibilidad"`
		Recuperacion   core.ReferenciaEntradaCatalogo `json:"recuperacion"`
		Movimientos    core.ReferenciaEntradaCatalogo `json:"movimientos"`
		Correccion     core.ReferenciaEntradaCatalogo `json:"correccion"`
		Permisos       core.ReferenciaEntradaCatalogo `json:"permisos"`
		Permiso        core.ReferenciaEntradaCatalogo `json:"permiso"`
		// Resolución de permisos y avisos: sólo con su selector activo.
		Bandeja      core.ReferenciaEntradaCatalogo `json:"bandeja"`
		Resolucion   core.ReferenciaEntradaCatalogo `json:"resolucion"`
		Avisos       core.ReferenciaEntradaCatalogo `json:"avisos"`
		ArchivoAviso core.ReferenciaEntradaCatalogo `json:"archivo_aviso"`
		// Notificaciones a RRHH: sólo con su selector activo.
		Notificacion          core.ReferenciaEntradaCatalogo `json:"notificacion"`
		Notificaciones        core.ReferenciaEntradaCatalogo `json:"notificaciones"`
		BandejaNotificaciones core.ReferenciaEntradaCatalogo `json:"bandeja_notificaciones"`
		AtencionNotificacion  core.ReferenciaEntradaCatalogo `json:"atencion_notificacion"`
	} `json:"motivos"`
	CanalRemoto struct {
		PoliticaVersionRef string `json:"politica_version_ref"`
		CanalRef           string `json:"canal_ref"`
		CalidadRef         string `json:"calidad_ref"`
	} `json:"canal_remoto"`
}

// autoridadCronosEmpleadoDesarrollo es la frontera de /api/interna/cronos/.
// Sólo publica las rutas exactas (ocho, más cuatro con la resolución de
// permisos) cuando todas sus dependencias están compuestas; toda otra ruta
// bajo el prefijo se deniega.
type autoridadCronosEmpleadoDesarrollo struct {
	base        *autoridadRutasDietasDesarrollo
	reloj       vecports.Reloj
	cuentas     map[string]cuentaRutasDietasDesarrollo
	rutas       map[string]http.Handler
	registrador cronosports.RegistroDenegacionFronteraCronos
	cerrar      func()
}

type claveContextoCronosEmpleado struct{}

type contextoCronosEmpleado struct {
	autoridad *autoridadCronosEmpleadoDesarrollo
	seguridad contextoSeguridadComunDesarrollo
}

// seguridadCronosEmpleadoDesarrollo entrega a la composición de Cronos la
// identidad que esta frontera registró para la misma petición.
type seguridadCronosEmpleadoDesarrollo struct {
	autoridad *autoridadCronosEmpleadoDesarrollo
}

func (s seguridadCronosEmpleadoDesarrollo) ResolverIdentidadRegistradaCronos(ctx context.Context) (cronoscomp.IdentidadRegistradaCronos, error) {
	if ctx == nil || s.autoridad == nil || s.autoridad.reloj == nil {
		return cronoscomp.IdentidadRegistradaCronos{}, ErrComposicionCronosEmpleadoNoDisponible
	}
	c, ok := ctx.Value(claveContextoCronosEmpleado{}).(contextoCronosEmpleado)
	if !ok || c.autoridad != s.autoridad || c.seguridad.Resultado.Validar() != nil || c.seguridad.Vinculo.ValidarPara(c.seguridad.Resultado) != nil ||
		!c.seguridad.Vinculo.VigenteEn(s.autoridad.reloj.Ahora(), c.seguridad.Resultado) {
		return cronoscomp.IdentidadRegistradaCronos{}, ErrComposicionCronosEmpleadoNoDisponible
	}
	clon, err := c.seguridad.Resultado.Clonar()
	if err != nil {
		return cronoscomp.IdentidadRegistradaCronos{}, ErrComposicionCronosEmpleadoNoDisponible
	}
	return cronoscomp.IdentidadRegistradaCronos{Contexto: clon, Vinculo: c.seguridad.Vinculo}, nil
}

func (a *autoridadCronosEmpleadoDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a == nil || r == nil || r.URL == nil {
		responderDenegacionCronosEmpleado(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	manejador, publicada := a.rutas[r.URL.Path]
	publicada = publicada && r.URL.RawPath == ""
	// Sin cadena mTLS verificada no hay identidad acreditada: se responde sin
	// auditoría durable para que un anónimo no pueda amplificar escrituras en
	// denegacion_frontera. Solo se auditan denegaciones con identidad TLS.
	if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.VerifiedChains[0]) == 0 {
		if !publicada {
			responderDenegacionCronosEmpleado(w, http.StatusNotFound, "no_disponible")
			return
		}
		responderDenegacionCronosEmpleado(w, http.StatusUnauthorized, "autenticacion_requerida")
		return
	}
	if !publicada {
		a.denegar(w, r, http.StatusNotFound, cronosports.MotivoFronteraAccesoDenegado, "no_disponible", "")
		return
	}
	if a.base == nil || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
		a.denegar(w, r, http.StatusUnauthorized, cronosports.MotivoFronteraAutenticacion, "autenticacion_requerida", "")
		return
	}
	principal, err := a.base.resolvedor.ResolveDemoIdentity(r.Context(), r)
	ahora := a.reloj.Ahora()
	cert := r.TLS.VerifiedChains[0][0]
	if err != nil || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh || ahora.Before(cert.NotBefore) || !ahora.Before(cert.NotAfter) {
		a.denegar(w, r, http.StatusUnauthorized, cronosports.MotivoFronteraAutenticacion, "autenticacion_requerida", "")
		return
	}
	cuenta, ok := a.cuentas[principal.Attributes["certificate_sha256"]]
	if !ok || cuenta.Sujeto != principal.ID {
		a.denegar(w, r, http.StatusForbidden, cronosports.MotivoFronteraAccesoDenegado, "acceso_denegado", "")
		return
	}
	vinculo, resultado, err := a.base.resolverSesion(r.Context(), r, &capsulaRutasDietasDesarrollo{autoridad: a.base, peticion: r, cuenta: cuenta, instante: ahora})
	switch {
	case errors.Is(err, vecports.ErrProyeccionEmpleadoContextoActorAusente):
		a.denegar(w, r, http.StatusForbidden, cronosports.MotivoFronteraSinEmpleado, "sin_empleado", "")
		return
	case errors.Is(err, vecports.ErrProyeccionEmpleadoContextoActorAmbigua):
		a.denegar(w, r, http.StatusForbidden, cronosports.MotivoFronteraEmpleadoAmbiguo, "empleado_ambiguo", "")
		return
	case err != nil:
		a.denegar(w, r, http.StatusServiceUnavailable, cronosports.MotivoFronteraDependencia, "no_disponible", "")
		return
	}
	if actorVerificadoComisionesDietas(vinculo, resultado, a.reloj.Ahora()) == "" {
		a.denegar(w, r, http.StatusServiceUnavailable, cronosports.MotivoFronteraDependencia, "no_disponible", "")
		return
	}
	ctx := context.WithValue(r.Context(), claveContextoCronosEmpleado{}, contextoCronosEmpleado{autoridad: a, seguridad: contextoSeguridadComunDesarrollo{Vinculo: vinculo, Resultado: resultado}})
	ctx = cronospg.ContextoConEstadoRemoto(ctx)
	manejador.ServeHTTP(w, r.WithContext(ctx))
}

// denegar registra la denegación con el LOGIN auditor antes de responder; si
// el registro no se confirma, la respuesta es dependencia no disponible.
func (a *autoridadCronosEmpleadoDesarrollo) denegar(w http.ResponseWriter, r *http.Request, estado int, motivo, codigo, actorRef string) {
	ruta := "otra"
	if _, ok := a.rutas[r.URL.Path]; ok && r.URL.RawPath == "" {
		ruta = r.URL.Path
	}
	metodo := "otro"
	if r.Method == http.MethodGet || r.Method == http.MethodPost {
		metodo = r.Method
	}
	var aleatorio [16]byte
	correlacion := "corr_no_disponible"
	if _, err := rand.Read(aleatorio[:]); err == nil {
		correlacion = "corr_" + hex.EncodeToString(aleatorio[:])
	}
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
	defer cancelar()
	if a.registrador == nil || a.registrador.RegistrarDenegacionFronteraCronos(ctx, cronosports.OrdenDenegacionFronteraCronos{CorrelacionRef: correlacion, Motivo: motivo, Ruta: ruta, Metodo: metodo, ActorRef: actorRef}) != nil {
		responderDenegacionCronosEmpleado(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	responderDenegacionCronosEmpleado(w, estado, codigo)
}

func responderDenegacionCronosEmpleado(w http.ResponseWriter, estado int, codigo string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": codigo})
}

// emisorCronosEmpleadoDesarrollo elige el emisor renovable de la audiencia
// que corresponde a cada acción; una acción desconocida no tiene emisor.
type emisorCronosEmpleadoDesarrollo struct {
	porAccion map[string]emisorMaterialDietasDesarrollo
}

func (e emisorCronosEmpleadoDesarrollo) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, solicitud core.SolicitudAutorizacionLigadaV3, resultado core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	datos, err := solicitud.Datos()
	elegido := e.porAccion[datos.Accion]
	if err != nil || elegido == nil {
		return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ErrComposicionCronosEmpleadoNoDisponible
	}
	return elegido.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
}

type relojCronosEmpleado struct{}

func (relojCronosEmpleado) AhoraUTC() time.Time { return time.Now().UTC().Truncate(time.Microsecond) }

// servicioContextoActorCronos compone ContextoActor con el alcance
// {empleado}: Cronos exige el empleado canónico que proyecta Personal
// (ContextoActor 000007) y deniega sin él o con varios.
func servicioContextoActorCronos(resolutor vecports.ResolutorRegistroContextoActorV2, reloj vecports.Reloj) (*vecapp.ServicioContextoActor, error) {
	alcance, err := core.NuevoAlcanceProyeccionesContextoActor(core.ProyeccionContextoActorEmpleado)
	if err != nil {
		return nil, err
	}
	return vecapp.NuevoServicioContextoActorProductivoV2ConAlcance(resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj, alcance)
}

// preflightCronosEmpleado comprueba con cada LOGIN nominal que las funciones
// de 000007 y 000008 existen (exigen AD3-53 y AD3-70) y que cada uno sólo tiene
// las que le corresponden. Una función ausente hace fallar la consulta.
func preflightCronosEmpleado(ctx context.Context, ejecutor, auditor *pgxpool.Pool, resolucion, notificaciones bool) error {
	var ok bool
	// Con la resolución activa se exigen también las cuatro funciones de
	// 000009 (AD3-57) y el circuito J-A de 000010, cuya bandeja devuelve lo
	// pendiente de asignación; sin ella no se consultan.
	if resolucion && (ejecutor.QueryRow(ctx, `SELECT bool_and(has_function_privilege(f,'EXECUTE')) AND to_regclass('vec_cronos_v1.permiso_circuito_directo') IS NOT NULL FROM unnest(ARRAY[
  'vec_cronos_v1.consultar_bandeja_permisos_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.resolver_permiso_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.consultar_avisos_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.archivar_aviso_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)']) f`).Scan(&ok) != nil || !ok) {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	// Con las notificaciones activas, las cuatro funciones de 000010 (AD3-58).
	if notificaciones && (ejecutor.QueryRow(ctx, `SELECT bool_and(has_function_privilege(f,'EXECUTE')) FROM unnest(ARRAY[
  'vec_cronos_v1.registrar_notificacion_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.consultar_notificaciones_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.consultar_bandeja_notificaciones_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.atender_notificacion_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)']) f`).Scan(&ok) != nil || !ok) {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	if ejecutor.QueryRow(ctx, `SELECT bool_and(has_function_privilege(f,'EXECUTE')) FROM unnest(ARRAY[
  'vec_cronos_v1.consultar_saldo_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.consultar_estado_remoto_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.registrar_marcaje_remoto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.recuperar_marcaje_remoto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.consultar_movimientos_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.solicitar_correccion_propia_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.consultar_permisos_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)',
  'vec_cronos_v1.solicitar_permiso_propio_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)']) f`).Scan(&ok) != nil || !ok {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	if auditor.QueryRow(ctx, `SELECT has_function_privilege('vec_cronos_v1.registrar_denegacion_frontera_v1(text,text,text,text,text)','EXECUTE')
 AND has_function_privilege('vec_cronos_v1.registrar_resultado_ejecucion_marcaje_v1(text,text,text,text,text,text,text,text,timestamptz)','EXECUTE')
 AND NOT has_function_privilege('vec_cronos_v1.registrar_marcaje_remoto_v1(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)','EXECUTE')`).Scan(&ok) != nil || !ok {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	return nil
}

// nuevasRutasCronosEmpleadoDesarrollo devuelve nil con el selector apagado.
// Con él activado, cualquier pieza ausente o incoherente impide arrancar.
func nuevasRutasCronosEmpleadoDesarrollo(cfg config.Config, resolvedor vechttp.DemoIdentityResolver, derivador *derivadorIdentidadOperacionDesarrollo, material materialCronosDesdeCTDesarrollo) (*autoridadCronosEmpleadoDesarrollo, error) {
	activo, err := cfg.CronosEmpleadoDesarrolloActivo()
	if err != nil {
		return nil, err
	}
	if !activo {
		return nil, nil
	}
	resolucion, err := cfg.CronosResolucionDesarrolloActiva()
	if err != nil {
		return nil, err
	}
	notificaciones, err := cfg.CronosNotificacionesDesarrolloActivas()
	if err != nil {
		return nil, err
	}
	identidad, ok := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !ok || identidad == nil || derivador == nil || !derivador.valido() || !material.completo() || (resolucion && !material.completoResolucion()) ||
		(notificaciones && !material.completoNotificaciones()) {
		return nil, errCronosEmpleadoEn()
	}
	contenido, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "cronos-empleado.json"), 256<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return nil, errCronosEmpleadoEn()
	}
	defer borrarBytes(contenido)
	var c configuracionCronosEmpleadoDesarrollo
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa || len(c.Cuentas) == 0 || len(c.Cuentas) > 64 ||
		(c.ZonaHoraria != cronosdomain.ZonaSaldoPeninsula && c.ZonaHoraria != cronosdomain.ZonaSaldoCanarias) {
		return nil, errCronosEmpleadoEn()
	}
	motivos := cronoscomp.MotivosCronos{Saldo: c.Motivos.Saldo, Marcaje: c.Motivos.Marcaje, Disponibilidad: c.Motivos.Disponibilidad, Recuperacion: c.Motivos.Recuperacion,
		Movimientos: c.Motivos.Movimientos, Correccion: c.Motivos.Correccion, Permisos: c.Motivos.Permisos, Permiso: c.Motivos.Permiso}
	comunes := []core.ReferenciaEntradaCatalogo{motivos.Marcaje, motivos.Disponibilidad, motivos.Recuperacion, motivos.Movimientos, motivos.Correccion, motivos.Permisos, motivos.Permiso}
	// La resolución exige sus cuatro motivos con su selector; sin él se
	// ignoran para no emitir decisiones de una capacidad no compuesta.
	if resolucion {
		motivos.Bandeja, motivos.Resolucion, motivos.Avisos, motivos.ArchivoAviso = c.Motivos.Bandeja, c.Motivos.Resolucion, c.Motivos.Avisos, c.Motivos.ArchivoAviso
		comunes = append(comunes, motivos.Bandeja, motivos.Resolucion, motivos.Avisos, motivos.ArchivoAviso)
	}
	// Igual para las notificaciones a RRHH.
	if notificaciones {
		motivos.Notificacion, motivos.Notificaciones = c.Motivos.Notificacion, c.Motivos.Notificaciones
		motivos.BandejaNotificaciones, motivos.AtencionNotificacion = c.Motivos.BandejaNotificaciones, c.Motivos.AtencionNotificacion
		comunes = append(comunes, motivos.Notificacion, motivos.Notificaciones, motivos.BandejaNotificaciones, motivos.AtencionNotificacion)
	}
	for _, m := range comunes {
		if m.CatalogoID != motivos.Saldo.CatalogoID {
			return nil, errCronosEmpleadoEn()
		}
	}
	canal, err := cronosdomain.NuevaAcreditacionCanalMarcaje(cronosdomain.DatosAcreditacionCanalMarcaje{PoliticaVersionRef: c.CanalRemoto.PoliticaVersionRef, CanalRef: c.CanalRemoto.CanalRef, OrigenRef: cronosdomain.OrigenMarcajeRemoto, CalidadRef: c.CanalRemoto.CalidadRef})
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	zona, err := time.LoadLocation(c.ZonaHoraria)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	cuentas := map[string]cuentaRutasDietasDesarrollo{}
	for _, cuenta := range c.Cuentas {
		b, e := hex.DecodeString(cuenta.CertificadoSHA256)
		if e != nil || len(b) != sha256.Size || hex.EncodeToString(b) != cuenta.CertificadoSHA256 || cuenta.Sujeto == "" || cuenta.CuentaRef == "" || cuenta.PerfilRef == "" {
			return nil, errCronosEmpleadoEn()
		}
		var digest [32]byte
		copy(digest[:], b)
		principal, existe := identidad.porHuella[digest]
		if !existe || principal.ID != cuenta.Sujeto || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh || cuentas[cuenta.CertificadoSHA256].Sujeto != "" {
			return nil, errCronosEmpleadoEn()
		}
		cuentas[cuenta.CertificadoSHA256] = cuenta
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	entradas := []struct{ dsn, rol string }{
		{c.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"}, {c.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"},
		{c.DSNContexto, "vec_contexto_actor_v1_runtime"}, {c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"},
		{c.DSNRegistroAutorizacion, "vec_autorizacion_registro"}, {c.DSNMotivos, "vec_autorizacion_motivos_evaluador"},
		{c.DSNCronos, "vec_cronos_v1_ejecutor"}, {c.DSNCronosAuditor, "vec_cronos_v1_auditor"},
	}
	var pools []*pgxpool.Pool
	var unaVez sync.Once
	cerrar := func() {
		unaVez.Do(func() {
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
	for _, entrada := range entradas {
		pool, usuario, e := abrirPoolRutasDietas(ctx, entrada.dsn, entrada.rol)
		if e != nil {
			return nil, errCronosEmpleadoEn()
		}
		pools = append(pools, pool)
		if usuarios[usuario] {
			return nil, errCronosEmpleadoEn()
		}
		usuarios[usuario] = true
	}
	if preflightCronosEmpleado(ctx, pools[6], pools[7], resolucion, notificaciones) != nil {
		return nil, errCronosEmpleadoEn()
	}
	registro, err := identidadpg.NuevoRegistroSesionesPostgreSQL(ctx, pools[0], pools[1], &seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	revalidador, err := identidadpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, pools[1])
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	resolutor, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pools[2])
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	reloj := relojRutasDietas{}
	servicioContexto, err := servicioContextoActorCronos(resolutor, reloj)
	if err != nil || !servicioContexto.Alcance().IncluyeEmpleado() {
		return nil, errCronosEmpleadoEn()
	}
	contextos, err := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	fuente, err := vecpg.NuevoAlmacenAutorizacion(pools[3])
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	registroAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(pools[4])
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	validadorMotivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools[5], motivos.Saldo.CatalogoID)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	autorizador, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registroAutorizacion, registroAutorizacion, validadorMotivos, reloj, seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	emisor := emisorCronosEmpleadoDesarrollo{porAccion: map[string]emisorMaterialDietasDesarrollo{}}
	proveedoresPorAccion := map[string]*proveedorMaterialAltaContratacionTemporalDesarrollo{
		cronosapp.AccionRegistrarMarcajePropio:        material.marcaje,
		cronosapp.AccionConsultarDisponibilidadRemota: material.disponibilidad,
		cronosapp.AccionRecuperarMarcajeRemoto:        material.recibo,
		cronosapp.AccionConsultarSaldoPropio:          material.saldo,
		cronosapp.AccionConsultarMovimientosPropios:   material.movimientos,
		cronosapp.AccionSolicitarCorreccion:           material.correccion,
		cronosapp.AccionConsultarPermisosPropios:      material.permisos,
		cronosapp.AccionSolicitarPermisoPropio:        material.solicitudPermiso,
	}
	if resolucion {
		proveedoresPorAccion[cronosapp.AccionConsultarBandeja] = material.bandeja
		proveedoresPorAccion[cronosapp.AccionResolverPermiso] = material.resolucion
		proveedoresPorAccion[cronosapp.AccionConsultarAvisosPropios] = material.avisos
		proveedoresPorAccion[cronosapp.AccionArchivarAvisoPropio] = material.archivoAviso
	}
	if notificaciones {
		proveedoresPorAccion[cronosapp.AccionRegistrarNotificacion] = material.notificacion
		proveedoresPorAccion[cronosapp.AccionConsultarNotificacionesPropias] = material.notificaciones
		proveedoresPorAccion[cronosapp.AccionConsultarBandejaNotif] = material.bandejaNotificaciones
		proveedoresPorAccion[cronosapp.AccionAtenderNotificacion] = material.atencionNotificacion
	}
	for accion, proveedor := range proveedoresPorAccion {
		e, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, proveedor)
		if err != nil {
			return nil, errCronosEmpleadoEn()
		}
		emisor.porAccion[accion] = e
	}
	autorizadorCronos, err := cronoscomp.NuevoAutorizadorCronos(emisor, motivos)
	if err != nil || autorizadorCronos.ResolucionConfigurada() != resolucion || autorizadorCronos.NotificacionesConfiguradas() != notificaciones {
		return nil, errCronosEmpleadoEn()
	}
	nonce, err := nonceRutasDietas()
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	registrador, err := cronospg.NuevoRegistroDenegacionFronteraPostgreSQL(pools[7])
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	base := &autoridadRutasDietasDesarrollo{resolvedor: identidad, cuentas: cuentas, registro: registro, revalidador: revalidador, contextos: contextos, reloj: reloj, instancia: nonce}
	a := &autoridadCronosEmpleadoDesarrollo{base: base, reloj: reloj, cuentas: cuentas, registrador: registrador, cerrar: cerrar}
	rutas, err := componerManejadoresCronosEmpleado(dependenciasCronosEmpleado{
		ejecutor: pools[6], auditor: pools[7], identidad: seguridadCronosEmpleadoDesarrollo{autoridad: a},
		autorizador: autorizadorCronos, canal: canal, zona: zona,
	})
	if err != nil {
		return nil, err
	}
	a.rutas = rutas
	completa = true
	return a, nil
}

type dependenciasCronosEmpleado struct {
	ejecutor, auditor *pgxpool.Pool
	identidad         cronoscomp.ResolutorIdentidadRegistradaCronos
	autorizador       *cronoscomp.AutorizadorCronos
	canal             cronosdomain.AcreditacionCanalMarcaje
	zona              *time.Location
}

// componerManejadoresCronosEmpleado une repositorios, casos de uso y
// manejadores. Devuelve las ocho rutas exactas, más las cuatro de la
// resolución de permisos si el autorizador la tiene configurada, o ninguna.
func componerManejadoresCronosEmpleado(d dependenciasCronosEmpleado) (map[string]http.Handler, error) {
	if d.ejecutor == nil || d.auditor == nil || dependenciaDietasNula(d.identidad) || d.autorizador == nil || d.zona == nil {
		return nil, errCronosEmpleadoEn()
	}
	resolutor, err := cronoscomp.NuevoResolutorPeticionCronos(d.identidad, d.autorizador, d.canal)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	disponibilidad, err := cronoscomp.NuevoProveedorDisponibilidadRemota(d.autorizador, d.identidad)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	resultados, err := cronospg.NuevoRegistroResultadoEjecucionMarcajePostgreSQL(d.auditor)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	remotos, err := cronospg.NuevoRepositorioMarcajesRemotos(d.ejecutor, resultados, disponibilidad, d.canal)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	saldos, err := cronospg.NuevoRepositorioConsultaSaldo(d.ejecutor)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	reloj := relojCronosEmpleado{}
	consultaSaldo, err := cronosapp.NuevoServicioConsultaSaldo(saldos, reloj, d.zona)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	marcajes, err := cronosapp.NuevoServicioMarcajesRemotos(remotos, remotos, reloj)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	movimientosRepo, err := cronospg.NuevoRepositorioConsultaMovimientos(d.ejecutor)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	movimientos, err := cronosapp.NuevoServicioConsultaMovimientos(movimientosRepo, reloj, d.zona)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	correccionesRepo, err := cronospg.NuevoRepositorioCorreccionesPropias(d.ejecutor, d.zona.String())
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	correcciones, err := cronosapp.NuevoServicioCorrecciones(correccionesRepo, reloj)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	permisosRepo, err := cronospg.NuevoRepositorioPermisosPropios(d.ejecutor)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	permisos, err := cronosapp.NuevoServicioPermisosPropios(permisosRepo, reloj, d.zona)
	if err != nil {
		return nil, errCronosEmpleadoEn()
	}
	deps := DependenciasManejadoresCronos{
		ConsultaSaldo: consultaSaldo, ResolverSaldo: resolutor,
		MarcajesRemotos: marcajes, ResolverMarcajeRemoto: resolutor, ResolverRecuperacionRemota: resolutor,
		Movimientos: movimientos, ResolverMovimientos: resolutor,
		Correcciones: correcciones, ResolverCorreccion: resolutor,
		Permisos: permisos, ResolverPermisos: resolutor,
	}
	resolucion := d.autorizador.ResolucionConfigurada()
	if resolucion {
		resolucionRepo, err := cronospg.NuevoRepositorioResolucionPermisos(d.ejecutor)
		if err != nil {
			return nil, errCronosEmpleadoEn()
		}
		avisosRepo, err := cronospg.NuevoRepositorioAvisosPropios(d.ejecutor)
		if err != nil {
			return nil, errCronosEmpleadoEn()
		}
		if deps.Resolucion, err = cronosapp.NuevoServicioResolucionPermisos(resolucionRepo, reloj, d.zona); err != nil {
			return nil, errCronosEmpleadoEn()
		}
		if deps.Avisos, err = cronosapp.NuevoServicioAvisosPropios(avisosRepo, reloj, d.zona); err != nil {
			return nil, errCronosEmpleadoEn()
		}
		deps.ResolverResolucion, deps.ResolverAvisos = resolutor, resolutor
	}
	notificaciones := d.autorizador.NotificacionesConfiguradas()
	if notificaciones {
		propiasRepo, err := cronospg.NuevoRepositorioNotificacionesPropias(d.ejecutor)
		if err != nil {
			return nil, errCronosEmpleadoEn()
		}
		bandejaRepo, err := cronospg.NuevoRepositorioBandejaNotificaciones(d.ejecutor)
		if err != nil {
			return nil, errCronosEmpleadoEn()
		}
		if deps.NotificacionesPropias, err = cronosapp.NuevoServicioNotificacionesPropias(propiasRepo, reloj, d.zona); err != nil {
			return nil, errCronosEmpleadoEn()
		}
		if deps.BandejaNotificaciones, err = cronosapp.NuevoServicioBandejaNotificaciones(bandejaRepo, reloj, d.zona); err != nil {
			return nil, errCronosEmpleadoEn()
		}
		deps.ResolverNotificacionesPropias, deps.ResolverBandejaNotificaciones = resolutor, resolutor
	}
	m, err := PrepararManejadoresCronos(deps)
	if err != nil || (m.Resolucion != nil) != resolucion || (m.Avisos != nil) != resolucion ||
		(m.NotificacionesPropias != nil) != notificaciones || (m.BandejaNotificaciones != nil) != notificaciones {
		return nil, errCronosEmpleadoEn()
	}
	rutas := map[string]http.Handler{
		cronoshttp.RutaConsultarSaldoPropio:         m.SaldoPropio,
		cronoshttp.RutaRegistrarMarcajeRemoto:       m.MarcajeRemoto,
		cronoshttp.RutaDisponibilidadMarcajeRemoto:  m.MarcajeRemoto,
		cronoshttp.RutaRecuperarReciboMarcajeRemoto: m.RecuperacionRemota,
		cronoshttp.RutaConsultarMovimientosPropios:  m.Movimientos,
		cronoshttp.RutaSolicitarCorreccionPropia:    m.CorreccionPropia,
		cronoshttp.RutaConsultarPermisosPropios:     m.PermisosPropios,
		cronoshttp.RutaSolicitarPermisoPropio:       m.PermisosPropios,
	}
	if resolucion {
		rutas[cronoshttp.RutaBandejaPermisos] = m.Resolucion
		rutas[cronoshttp.RutaResolverPermiso] = m.Resolucion
		rutas[cronoshttp.RutaAvisosPropios] = m.Avisos
		rutas[cronoshttp.RutaArchivarAviso] = m.Avisos
	}
	if notificaciones {
		rutas[cronoshttp.RutaNotificacionesPropias] = m.NotificacionesPropias
		rutas[cronoshttp.RutaRegistrarNotificacion] = m.NotificacionesPropias
		rutas[cronoshttp.RutaBandejaNotificaciones] = m.BandejaNotificaciones
		rutas[cronoshttp.RutaAtenderNotificacion] = m.BandejaNotificaciones
	}
	return rutas, nil
}

// componerRaizConCronosEmpleado monta el prefijo interno de Cronos delante de
// la raíz existente. Sin autoridad compuesta la raíz queda como estaba.
func componerRaizConCronosEmpleado(raiz http.Handler, cronos *autoridadCronosEmpleadoDesarrollo) http.Handler {
	if cronos == nil {
		return raiz
	}
	mux := http.NewServeMux()
	mux.Handle(prefijoRutasCronosEmpleado, cronos)
	mux.Handle("/", raiz)
	return mux
}

// errCronosEmpleadoEn añade sólo fichero y línea del rechazo, sin DSN,
// identidades ni material.
func errCronosEmpleadoEn() error {
	_, fichero, linea, ok := runtime.Caller(1)
	if !ok {
		return ErrComposicionCronosEmpleadoNoDisponible
	}
	return fmt.Errorf("%w (%s:%d)", ErrComposicionCronosEmpleadoNoDisponible, filepath.Base(fichero), linea)
}
