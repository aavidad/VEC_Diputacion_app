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
	personalcomp "vec-diputacion-granada/internal/modules/personal/adapters/composicion"
	personalhttp "vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
	personalapp "vec-diputacion-granada/internal/modules/personal/application"
	personaldomain "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	contextopg "vec-diputacion-granada/internal/vec/adapters/contextoactor/postgres"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	identidadpg "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
	vecpg "vec-diputacion-granada/internal/vec/adapters/postgres"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecapp "vec-diputacion-granada/internal/vec/application"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrComposicionPersonalEmpleadoNoDisponible = errors.New("bootstrap: ficha propia de Personal no disponible")

const prefijoRutasPersonalEmpleado = "/api/interna/personal/"

// Personal se integra como Cronos y Dietas en el gobierno V3 único de
// desarrollo: una clave derivada para la audiencia de la ficha propia
// (AD3-74), con dominio y prefijo propios, bajo la raíz de Contratación.
func descriptorMaterialFichaPropiaPersonalDesarrollo() descriptorMaterialConsumidorV3Desarrollo {
	return descriptorMaterialConsumidorV3Desarrollo{
		Audiencia: personaldomain.AudienciaFichaPropia, Dominio: "vec.personal.ficha-propia.desarrollo.capacidad-v3",
		Prefijo: "clave:capacidad:personal-ficha-propia:", ProveedorNominal: "proveedor-material-personal-ficha-propia",
	}
}

// personalEmpleadoSolicitado sólo refleja el selector; la validación completa
// (doble llave de desarrollo) la hace la composición de la ruta.
func personalEmpleadoSolicitado(selector string) bool { return selector == "true" }

// El archivo privado identidad/personal-empleado.json sólo selecciona
// cuentas ya registradas, conexiones nominales, el motivo y la zona de la
// fecha de efectos. No crea identidades, perfiles ni concesiones.
type configuracionPersonalEmpleadoDesarrollo struct {
	redaccionMaterialRutasDietas
	Version                  int                            `json:"version"`
	Autoridad                string                         `json:"autoridad"`
	Cuentas                  []cuentaRutasDietasDesarrollo  `json:"cuentas"`
	DSNRegistroIdentidad     string                         `json:"dsn_registro_identidad"`
	DSNRevalidacionIdentidad string                         `json:"dsn_revalidacion_identidad"`
	DSNContexto              string                         `json:"dsn_contexto"`
	DSNFuenteAutorizacion    string                         `json:"dsn_fuente_autorizacion"`
	DSNRegistroAutorizacion  string                         `json:"dsn_registro_autorizacion"`
	DSNMotivos               string                         `json:"dsn_motivos"`
	DSNPersonal              string                         `json:"dsn_personal"`
	DSNPersonalFrontera      string                         `json:"dsn_personal_frontera"`
	ZonaHoraria              string                         `json:"zona_horaria"`
	MotivoFichaPropia        core.ReferenciaEntradaCatalogo `json:"motivo_ficha_propia"`
}

// autoridadPersonalEmpleadoDesarrollo es la frontera de /api/interna/personal/.
// Sólo publica la ruta exacta de la ficha propia; toda otra ruta bajo el
// prefijo se deniega.
type autoridadPersonalEmpleadoDesarrollo struct {
	base     *autoridadRutasDietasDesarrollo
	reloj    vecports.Reloj
	cuentas  map[string]cuentaRutasDietasDesarrollo
	rutas    map[string]http.Handler
	registro personalports.RegistroDenegacionFichaPropia
	cerrar   func()
}

type claveContextoPersonalEmpleado struct{}

type contextoPersonalEmpleado struct {
	autoridad *autoridadPersonalEmpleadoDesarrollo
	seguridad contextoSeguridadComunDesarrollo
}

// seguridadPersonalEmpleadoDesarrollo entrega a Personal la identidad que
// esta frontera registró para la misma petición: al proveedor V3 el vínculo
// y el resultado, y al manejador el contexto de actor.
type seguridadPersonalEmpleadoDesarrollo struct {
	autoridad *autoridadPersonalEmpleadoDesarrollo
}

func (s seguridadPersonalEmpleadoDesarrollo) identidad(ctx context.Context) (contextoSeguridadComunDesarrollo, error) {
	if ctx == nil || s.autoridad == nil || s.autoridad.reloj == nil {
		return contextoSeguridadComunDesarrollo{}, ErrComposicionPersonalEmpleadoNoDisponible
	}
	c, ok := ctx.Value(claveContextoPersonalEmpleado{}).(contextoPersonalEmpleado)
	if !ok || c.autoridad != s.autoridad || c.seguridad.Resultado.Validar() != nil || c.seguridad.Vinculo.ValidarPara(c.seguridad.Resultado) != nil ||
		!c.seguridad.Vinculo.VigenteEn(s.autoridad.reloj.Ahora(), c.seguridad.Resultado) {
		return contextoSeguridadComunDesarrollo{}, ErrComposicionPersonalEmpleadoNoDisponible
	}
	clon, err := c.seguridad.Resultado.Clonar()
	if err != nil {
		return contextoSeguridadComunDesarrollo{}, ErrComposicionPersonalEmpleadoNoDisponible
	}
	return contextoSeguridadComunDesarrollo{Vinculo: c.seguridad.Vinculo, Resultado: clon}, nil
}

func (s seguridadPersonalEmpleadoDesarrollo) ResolverIdentidadFichaPropia(ctx context.Context) (personalcomp.IdentidadRegistradaFichaPropia, error) {
	c, err := s.identidad(ctx)
	if err != nil {
		return personalcomp.IdentidadRegistradaFichaPropia{}, err
	}
	return personalcomp.IdentidadRegistradaFichaPropia{Vinculo: c.Vinculo, Resultado: c.Resultado}, nil
}

func (s seguridadPersonalEmpleadoDesarrollo) ResolverActorFichaPropia(ctx context.Context) (core.ContextoActor, error) {
	c, err := s.identidad(ctx)
	if err != nil {
		return core.ContextoActor{}, err
	}
	return c.Resultado.Contexto.Clonar()
}

func (a *autoridadPersonalEmpleadoDesarrollo) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if a == nil || r == nil || r.URL == nil {
		responderDenegacionCronosEmpleado(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	manejador, publicada := a.rutas[r.URL.Path]
	publicada = publicada && r.URL.RawPath == ""
	// Sin cadena mTLS verificada no hay identidad: se responde sin auditoría
	// durable para que un anónimo no pueda amplificar escrituras.
	if r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.VerifiedChains[0]) == 0 {
		if !publicada {
			responderDenegacionCronosEmpleado(w, http.StatusNotFound, "no_disponible")
			return
		}
		responderDenegacionCronosEmpleado(w, http.StatusUnauthorized, "autenticacion_requerida")
		return
	}
	if !publicada {
		a.denegar(w, r, http.StatusNotFound, "no_encontrada", "no_disponible")
		return
	}
	if a.base == nil || r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
		a.denegar(w, r, http.StatusUnauthorized, "autenticacion_requerida", "autenticacion_requerida")
		return
	}
	principal, err := a.base.resolvedor.ResolveDemoIdentity(r.Context(), r)
	ahora := a.reloj.Ahora()
	cert := r.TLS.VerifiedChains[0][0]
	if err != nil || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh || ahora.Before(cert.NotBefore) || !ahora.Before(cert.NotAfter) {
		a.denegar(w, r, http.StatusUnauthorized, "autenticacion_requerida", "autenticacion_requerida")
		return
	}
	cuenta, ok := a.cuentas[principal.Attributes["certificate_sha256"]]
	if !ok || cuenta.Sujeto != principal.ID {
		a.denegar(w, r, http.StatusForbidden, "acceso_denegado", "acceso_denegado")
		return
	}
	vinculo, resultado, err := a.base.resolverSesion(r.Context(), r, &capsulaRutasDietasDesarrollo{autoridad: a.base, peticion: r, cuenta: cuenta, instante: ahora})
	switch {
	case errors.Is(err, vecports.ErrProyeccionEmpleadoContextoActorAusente):
		a.denegar(w, r, http.StatusForbidden, "sin_empleado", "sin_empleado")
		return
	case errors.Is(err, vecports.ErrProyeccionEmpleadoContextoActorAmbigua):
		a.denegar(w, r, http.StatusForbidden, "empleado_ambiguo", "empleado_ambiguo")
		return
	case err != nil:
		a.denegar(w, r, http.StatusServiceUnavailable, "dependencia_no_disponible", "no_disponible")
		return
	}
	if actorVerificadoComisionesDietas(vinculo, resultado, a.reloj.Ahora()) == "" {
		a.denegar(w, r, http.StatusServiceUnavailable, "dependencia_no_disponible", "no_disponible")
		return
	}
	ctx := context.WithValue(r.Context(), claveContextoPersonalEmpleado{}, contextoPersonalEmpleado{autoridad: a, seguridad: contextoSeguridadComunDesarrollo{Vinculo: vinculo, Resultado: resultado}})
	manejador.ServeHTTP(w, r.WithContext(ctx))
}

// denegar registra la denegación con el LOGIN registrador de frontera antes
// de responder; si no se confirma, responde dependencia no disponible.
func (a *autoridadPersonalEmpleadoDesarrollo) denegar(w http.ResponseWriter, r *http.Request, estado int, motivo, codigo string) {
	var aleatorio [16]byte
	correlacion := "corr_no_disponible"
	if _, err := rand.Read(aleatorio[:]); err == nil {
		correlacion = "corr_" + hex.EncodeToString(aleatorio[:])
	}
	ctx, cancelar := context.WithTimeout(context.WithoutCancel(r.Context()), 2*time.Second)
	defer cancelar()
	if a.registro == nil || a.registro.RegistrarDenegacionFichaPropia(ctx, personalports.DenegacionFichaPropia{CorrelacionRef: correlacion, Motivo: motivo, EstadoHTTP: estado}) != nil {
		responderDenegacionCronosEmpleado(w, http.StatusServiceUnavailable, "no_disponible")
		return
	}
	responderDenegacionCronosEmpleado(w, estado, codigo)
}

// servicioContextoActorPersonalEmpleado compone ContextoActor con el alcance
// {empleado}: la ficha propia exige el empleado canónico que proyecta
// Personal y deniega sin él o con varios.
func servicioContextoActorPersonalEmpleado(resolutor vecports.ResolutorRegistroContextoActorV2, reloj vecports.Reloj) (*vecapp.ServicioContextoActor, error) {
	alcance, err := core.NuevoAlcanceProyeccionesContextoActor(core.ProyeccionContextoActorEmpleado)
	if err != nil {
		return nil, err
	}
	return vecapp.NuevoServicioContextoActorProductivoV2ConAlcance(resolutor, contextopg.NuevoGeneradorOperacionContextoActorV2Criptografico(), reloj, alcance)
}

// nuevasRutasPersonalEmpleadoDesarrollo devuelve nil con el selector apagado.
// Con él activado, cualquier pieza ausente o incoherente impide arrancar.
func nuevasRutasPersonalEmpleadoDesarrollo(cfg config.Config, resolvedor vechttp.DemoIdentityResolver, derivador *derivadorIdentidadOperacionDesarrollo, material *proveedorMaterialAltaContratacionTemporalDesarrollo) (*autoridadPersonalEmpleadoDesarrollo, error) {
	activo, err := cfg.PersonalEmpleadoDesarrolloActivo()
	if err != nil {
		return nil, err
	}
	if !activo {
		return nil, nil
	}
	identidad, ok := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !ok || identidad == nil || derivador == nil || !derivador.valido() || material == nil {
		return nil, errPersonalEmpleadoEn()
	}
	contenido, err := leerFicheroMaterialSeguro(filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "personal-empleado.json"), 256<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return nil, errPersonalEmpleadoEn()
	}
	defer borrarBytes(contenido)
	var c configuracionPersonalEmpleadoDesarrollo
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa ||
		len(c.Cuentas) == 0 || len(c.Cuentas) > 64 || !core.ReferenciaMotivoAutorizacionV2Valida(c.MotivoFichaPropia) {
		return nil, errPersonalEmpleadoEn()
	}
	zona, err := time.LoadLocation(c.ZonaHoraria)
	if err != nil || c.ZonaHoraria == "" || c.ZonaHoraria == "Local" {
		return nil, errPersonalEmpleadoEn()
	}
	cuentas := map[string]cuentaRutasDietasDesarrollo{}
	for _, cuenta := range c.Cuentas {
		b, e := hex.DecodeString(cuenta.CertificadoSHA256)
		if e != nil || len(b) != sha256.Size || hex.EncodeToString(b) != cuenta.CertificadoSHA256 || cuenta.Sujeto == "" || cuenta.CuentaRef == "" || cuenta.PerfilRef == "" {
			return nil, errPersonalEmpleadoEn()
		}
		var digest [32]byte
		copy(digest[:], b)
		principal, existe := identidad.porHuella[digest]
		if !existe || principal.ID != cuenta.Sujeto || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh || cuentas[cuenta.CertificadoSHA256].Sujeto != "" {
			return nil, errPersonalEmpleadoEn()
		}
		cuentas[cuenta.CertificadoSHA256] = cuenta
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	entradas := []struct{ dsn, rol string }{
		{c.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"}, {c.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"},
		{c.DSNContexto, "vec_contexto_actor_v1_runtime"}, {c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"},
		{c.DSNRegistroAutorizacion, "vec_autorizacion_registro"}, {c.DSNMotivos, "vec_autorizacion_motivos_evaluador"},
		{c.DSNPersonal, "vec_personal_ejecutor"}, {c.DSNPersonalFrontera, "vec_personal_registrador_frontera"},
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
			return nil, errPersonalEmpleadoEn()
		}
		pools = append(pools, pool)
		if usuarios[usuario] {
			return nil, errPersonalEmpleadoEn()
		}
		usuarios[usuario] = true
	}
	registroDenegaciones, err := personalpg.NuevoRegistroDenegacionFichaPropiaPostgreSQL(pools[7])
	if err != nil || personalpg.PreflightEjecutorFichaPropia(ctx, pools[6]) != nil || registroDenegaciones.PreflightFichaPropia(ctx) != nil {
		return nil, errPersonalEmpleadoEn()
	}
	registro, err := identidadpg.NuevoRegistroSesionesPostgreSQL(ctx, pools[0], pools[1], &seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	revalidador, err := identidadpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, pools[1])
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	resolutor, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pools[2])
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	reloj := relojRutasDietas{}
	servicioContexto, err := servicioContextoActorPersonalEmpleado(resolutor, reloj)
	if err != nil || !servicioContexto.Alcance().IncluyeEmpleado() {
		return nil, errPersonalEmpleadoEn()
	}
	contextos, err := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	fuente, err := vecpg.NuevoAlmacenAutorizacion(pools[3])
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	registroAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(pools[4])
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	validadorMotivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools[5], c.MotivoFichaPropia.CatalogoID)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	autorizador, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registroAutorizacion, registroAutorizacion, validadorMotivos, reloj, seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	emisor, err := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, material)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	nonce, err := nonceRutasDietas()
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	base := &autoridadRutasDietasDesarrollo{resolvedor: identidad, cuentas: cuentas, registro: registro, revalidador: revalidador, contextos: contextos, reloj: reloj, instancia: nonce}
	a := &autoridadPersonalEmpleadoDesarrollo{base: base, reloj: reloj, cuentas: cuentas, registro: registroDenegaciones, cerrar: cerrar}
	manejador, err := componerManejadorFichaPropia(pools[6], seguridadPersonalEmpleadoDesarrollo{autoridad: a}, emisor, c.MotivoFichaPropia, registroDenegaciones, zona)
	if err != nil {
		return nil, err
	}
	a.rutas = map[string]http.Handler{personalhttp.RutaFichaPropia: manejador}
	completa = true
	return a, nil
}

// componerManejadorFichaPropia une repositorio, proveedor V3, caso de uso y
// manejador. Devuelve la ruta completa o ninguna.
func componerManejadorFichaPropia(ejecutor *pgxpool.Pool, identidad seguridadPersonalEmpleadoDesarrollo, emisor emisorMaterialDietasDesarrollo, motivo core.ReferenciaEntradaCatalogo, registro personalports.RegistroDenegacionFichaPropia, zona *time.Location) (http.Handler, error) {
	if ejecutor == nil || identidad.autoridad == nil || dependenciaDietasNula(emisor) || dependenciaDietasNula(registro) || zona == nil {
		return nil, errPersonalEmpleadoEn()
	}
	repositorio, err := personalpg.NuevoRepositorioRegistroEmpleadoB2PostgreSQL(ejecutor)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	proveedor, err := personalcomp.NuevoProveedorAutorizacionFichaPropia(identidad, emisor, motivo)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	servicio, err := personalapp.NuevoServicioFichaPropia(proveedor, repositorio)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	manejador, err := personalhttp.NuevoManejadorFichaPropia(identidad, servicio, registro, time.Now, zona)
	if err != nil {
		return nil, errPersonalEmpleadoEn()
	}
	return manejador, nil
}

// componerRaizConPersonalEmpleado monta el prefijo interno de Personal
// delante de la raíz existente. Sin autoridad compuesta la raíz no cambia.
func componerRaizConPersonalEmpleado(raiz http.Handler, personal *autoridadPersonalEmpleadoDesarrollo) http.Handler {
	if personal == nil {
		return raiz
	}
	mux := http.NewServeMux()
	mux.Handle(prefijoRutasPersonalEmpleado, personal)
	mux.Handle("/", raiz)
	return mux
}

// errPersonalEmpleadoEn añade sólo fichero y línea del rechazo, sin DSN,
// identidades ni material.
func errPersonalEmpleadoEn() error {
	_, fichero, linea, ok := runtime.Caller(1)
	if !ok {
		return ErrComposicionPersonalEmpleadoNoDisponible
	}
	return fmt.Errorf("%w (%s:%d)", ErrComposicionPersonalEmpleadoNoDisponible, filepath.Base(fichero), linea)
}
