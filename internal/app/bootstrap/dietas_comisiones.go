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
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/config"
	dietas "vec-diputacion-granada/internal/modules/dietas"
	dietashttp "vec-diputacion-granada/internal/modules/dietas/adapters/httpinterno"
	dietaspg "vec-diputacion-granada/internal/modules/dietas/adapters/postgres"
	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	personalhttp "vec-diputacion-granada/internal/modules/personal/adapters/httpinterno"
	personalpg "vec-diputacion-granada/internal/modules/personal/adapters/postgres"
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

// El archivo privado sólo selecciona cuentas, conexiones y motivos. Las claves
// V3 no viven aquí: Dietas firma con el gobierno único de desarrollo que
// publica Contratación (ver dietas_material_ct_desarrollo.go). Su ausencia con
// el selector activado impide montar la ruta; no genera claves.
type configuracionComisionesDietasDesarrollo struct {
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
	DSNAuditoriaFrontera     string                         `json:"dsn_auditoria_frontera"`
	MotivoPersonal           core.ReferenciaEntradaCatalogo `json:"motivo_personal"`
	MotivoCrear              core.ReferenciaEntradaCatalogo `json:"motivo_crear"`
	MotivoConsultar          core.ReferenciaEntradaCatalogo `json:"motivo_consultar"`
}

type claveContextoComisionesDietas struct{}
type contextoComisionesDietas struct {
	autoridad    *autoridadComisionesDietasDesarrollo
	ruta, metodo string
	seguridad    contextoSeguridadComunDesarrollo
}

type seguridadComisionesDietasDesarrollo struct {
	autoridad *autoridadComisionesDietasDesarrollo
}

func (s seguridadComisionesDietasDesarrollo) ResolverContexto(ctx context.Context) (contextoSeguridadComunDesarrollo, error) {
	if ctx == nil || s.autoridad == nil {
		return contextoSeguridadComunDesarrollo{}, ErrIdentidadPersonalDietasNoDisponible
	}
	c, ok := ctx.Value(claveContextoComisionesDietas{}).(contextoComisionesDietas)
	if !ok || c.autoridad != s.autoridad || c.seguridad.Resultado.Validar() != nil || c.seguridad.Vinculo.ValidarPara(c.seguridad.Resultado) != nil || !c.seguridad.Vinculo.VigenteEn(s.autoridad.reloj.Ahora(), c.seguridad.Resultado) {
		return contextoSeguridadComunDesarrollo{}, ErrIdentidadPersonalDietasNoDisponible
	}
	clon, err := c.seguridad.Resultado.Clonar()
	if err != nil {
		return contextoSeguridadComunDesarrollo{}, ErrIdentidadPersonalDietasNoDisponible
	}
	return contextoSeguridadComunDesarrollo{Vinculo: c.seguridad.Vinculo, Resultado: clon}, nil
}

// emisorMaterialDietasDesarrollo es lo único que Dietas necesita de un emisor
// V3: el renovable de Contratación sigue la configuración vigente cada día.
type emisorMaterialDietasDesarrollo interface {
	EmitirMaterialAutorizacionAtestadaV3(context.Context, core.SolicitudAutorizacionLigadaV3, core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error)
}

type emisorComisionesDietasDesarrollo struct {
	personal, crear, consultar emisorMaterialDietasDesarrollo
	adicionales                map[string]emisorMaterialDietasDesarrollo
}

func (e *emisorComisionesDietasDesarrollo) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, solicitud core.SolicitudAutorizacionLigadaV3, resultado core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	var elegido emisorMaterialDietasDesarrollo
	datos, err := solicitud.Datos()
	if err == nil && e != nil {
		switch datos.Accion {
		case "personal.relacion.propia.consultar_dietas":
			elegido = e.personal
		case "dietas.borrador.propio.crear":
			elegido = e.crear
		case "dietas.borrador.propio.consultar":
			elegido = e.consultar
		default:
			elegido = e.adicionales[datos.Accion]
		}
	}
	if elegido == nil {
		return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, ErrComposicionBorradoresDietasNoDisponible
	}
	return elegido.EmitirMaterialAutorizacionAtestadaV3(ctx, solicitud, resultado)
}

type autoridadComisionesDietasDesarrollo struct {
	base                *autoridadRutasDietasDesarrollo
	reloj               vecports.Reloj
	cuentas             map[string]cuentaRutasDietasDesarrollo
	rutas               []vechttp.RutaExacta
	colecciones         []vechttp.RutaColeccion
	registrador         dietasports.RegistradorAuditoriaFronteraComision
	registradorPersonal personalports.RegistradorAuditoriaFronteraAsignacionDietas
	cerrar              func()
}

func esRutaComisionesDietas(ruta string) bool {
	return ruta == dietashttp.RutaBorradores || strings.HasPrefix(ruta, dietashttp.RutaBorradores+"/") || ruta == personalhttp.RutaRelacionesDietas || strings.HasPrefix(ruta, personalhttp.RutaRelacionesDietas+"/") || ruta == personalhttp.RutaAsignacionesDietas || strings.HasPrefix(ruta, personalhttp.RutaAsignacionesDietas+"/")
}

func cabeceraLibreComisionesDietas(cabeceras http.Header) bool {
	for nombre := range cabeceras {
		switch strings.ToLower(nombre) {
		case "cookie", "authorization", "x-vec-actor", "x-vec-persona", "x-vec-perfil":
			return true
		}
	}
	return false
}

var referenciaRutaComisionDietas = regexp.MustCompile(`^dco_[A-Za-z0-9_-]{22,128}$`)
var referenciaRutaRelacionPersonalDietas = regexp.MustCompile(`^rel_[A-Za-z0-9_-]{22,128}$`)

func metodoComisionesDietasValido(ruta, metodo string) bool {
	if escrituraAsignacionDietas(ruta, metodo) && !escrituraAsignacionDietasAbierta(catalogoValidadoresCompetentesAsignacionDietas) {
		return false
	}
	if ruta == personalhttp.RutaRelacionesDietas {
		return metodo == http.MethodGet
	}
	if ruta == personalhttp.RutaAsignacionesDietas {
		return metodo == http.MethodPost
	}
	if resto, ok := strings.CutPrefix(ruta, personalhttp.RutaAsignacionesDietas+"/"); ok {
		if relacion, grupo := strings.CutSuffix(resto, "/grupo"); grupo {
			return metodo == http.MethodPut && referenciaRutaRelacionPersonalDietas.MatchString(relacion)
		}
		return referenciaRutaRelacionPersonalDietas.MatchString(resto) && (metodo == http.MethodGet || metodo == http.MethodPut)
	}
	if ruta == dietashttp.RutaBorradores {
		return metodo == http.MethodGet || metodo == http.MethodPost
	}
	if ruta == dietashttp.RutaBorradores+"/circuito" {
		return metodo == http.MethodGet
	}
	if resto, ok := strings.CutPrefix(ruta, dietashttp.RutaBorradores+"/circuito/"); ok {
		referencia, decision := strings.CutSuffix(resto, "/decisiones")
		return decision && metodo == http.MethodPost && referenciaRutaComisionDietas.MatchString(referencia)
	}
	resto, ok := strings.CutPrefix(ruta, dietashttp.RutaBorradores+"/")
	if !ok {
		return false
	}
	if referenciaRutaComisionDietas.MatchString(resto) {
		return metodo == http.MethodGet || metodo == http.MethodPut || metodo == http.MethodDelete
	}
	referencia, enviar := strings.CutSuffix(resto, "/enviar")
	return enviar && metodo == http.MethodPost && referenciaRutaComisionDietas.MatchString(referencia)
}

func accionFronteraComisionesDietas(ruta, metodo string) string {
	if !metodoComisionesDietasValido(ruta, metodo) {
		return dietasports.AccionFronteraMetodoNoAdmitido
	}
	if ruta != dietashttp.RutaBorradores {
		if strings.HasPrefix(ruta, dietashttp.RutaBorradores+"/circuito") {
			if metodo == http.MethodGet {
				return dietasports.AccionFronteraConsultarBandeja
			}
			return dietasports.AccionFronteraDecidir
		}
		if strings.HasSuffix(ruta, "/enviar") {
			return dietasports.AccionFronteraEnviar
		}
		switch metodo {
		case http.MethodPut:
			return dietasports.AccionFronteraEditar
		case http.MethodDelete:
			return dietasports.AccionFronteraBorrar
		default:
			return dietasports.AccionFronteraConsultarDetalle
		}
	}
	if metodo == http.MethodPost {
		return dietasports.AccionFronteraCrear
	}
	return dietasports.AccionFronteraListar
}

func (a *autoridadComisionesDietasDesarrollo) proteger(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil {
			responderDenegacionComisionesDietas(w, http.StatusServiceUnavailable)
			return
		}
		if !esRutaComisionesDietas(r.URL.Path) {
			siguiente.ServeHTTP(w, r)
			return
		}
		if a == nil || a.base == nil || r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.VerifiedChains[0]) == 0 || cabeceraLibreComisionesDietas(r.Header) {
			a.denegar(w, r, http.StatusUnauthorized, "")
			return
		}
		principal, err := a.base.resolvedor.ResolveDemoIdentity(r.Context(), r)
		ahora := a.reloj.Ahora()
		cert := r.TLS.VerifiedChains[0][0]
		if err != nil || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh || ahora.Before(cert.NotBefore) || !ahora.Before(cert.NotAfter) {
			a.denegar(w, r, http.StatusUnauthorized, "")
			return
		}
		cuenta, ok := a.cuentas[principal.Attributes["certificate_sha256"]]
		if !ok || cuenta.Sujeto != principal.ID {
			a.denegar(w, r, http.StatusForbidden, "")
			return
		}
		vinculo, resultado, err := a.base.resolverSesion(r.Context(), r, &capsulaRutasDietasDesarrollo{autoridad: a.base, peticion: r, cuenta: cuenta, instante: ahora})
		if motivo := motivoEmpleadoDietas(err); motivo != "" {
			a.denegarConMotivo(w, r, motivo)
			return
		}
		if err != nil {
			a.denegar(w, r, http.StatusServiceUnavailable, "")
			return
		}
		actorRef := actorVerificadoComisionesDietas(vinculo, resultado, a.reloj.Ahora())
		if actorRef == "" {
			a.denegar(w, r, http.StatusServiceUnavailable, "")
			return
		}
		if r.URL.RawPath != "" || r.URL.EscapedPath() != r.URL.Path {
			if esRutaPersonalDietas(r.URL.Path) {
				a.denegarPersonalEstructural(w, r, actorRef)
				return
			}
			a.denegar(w, r, http.StatusForbidden, actorRef)
			return
		}
		if !metodoComisionesDietasValido(r.URL.Path, r.Method) {
			a.denegar(w, r, http.StatusForbidden, actorRef)
			return
		}
		ctx := context.WithValue(r.Context(), claveContextoComisionesDietas{}, contextoComisionesDietas{autoridad: a, ruta: r.URL.Path, metodo: r.Method, seguridad: contextoSeguridadComunDesarrollo{Vinculo: vinculo, Resultado: resultado}})
		siguiente.ServeHTTP(w, r.WithContext(ctx))
	})
}

func esRutaPersonalDietas(ruta string) bool {
	return ruta == personalhttp.RutaRelacionesDietas || strings.HasPrefix(ruta, personalhttp.RutaRelacionesDietas+"/") || ruta == personalhttp.RutaAsignacionesDietas || strings.HasPrefix(ruta, personalhttp.RutaAsignacionesDietas+"/")
}

func (a *autoridadComisionesDietasDesarrollo) denegarPersonalEstructural(w http.ResponseWriter, r *http.Request, actorRef string) {
	if a.registrarDenegacionPersonal(r.Context(), r.URL.Path, r.Method, http.StatusBadRequest, actorRef, true) != nil {
		responderDenegacionComisionesDietas(w, http.StatusServiceUnavailable)
		return
	}
	responderDenegacionComisionesDietas(w, http.StatusBadRequest)
}

// Sólo el vínculo V2 ligado al contexto registrado puede aportar identidad a
// la auditoría. Ni la petición ni un principal aún no revalidado son fuente.
func actorVerificadoComisionesDietas(vinculo core.VinculoAutenticacionActorV2, resultado core.ResultadoContextoActorRegistradoV2, ahora time.Time) string {
	if !vinculo.VigenteEn(ahora, resultado) {
		return ""
	}
	datos, err := vinculo.Datos()
	if err != nil {
		return ""
	}
	return datos.PrincipalID
}

func (a *autoridadComisionesDietasDesarrollo) denegar(w http.ResponseWriter, r *http.Request, estado int, actorRef string) {
	if r == nil || r.URL == nil {
		responderDenegacionComisionesDietas(w, http.StatusServiceUnavailable)
		return
	}
	if a.registrarDenegacion(r.Context(), r.URL.Path, r.Method, estado, actorRef) != nil {
		responderDenegacionComisionesDietas(w, http.StatusServiceUnavailable)
		return
	}
	responderDenegacionComisionesDietas(w, estado)
}

func (a *autoridadComisionesDietasDesarrollo) registrarDenegacion(ctx context.Context, rutaPeticion, metodo string, estado int, actorRef string) error {
	if rutaPeticion == personalhttp.RutaRelacionesDietas || strings.HasPrefix(rutaPeticion, personalhttp.RutaRelacionesDietas+"/") || rutaPeticion == personalhttp.RutaAsignacionesDietas || strings.HasPrefix(rutaPeticion, personalhttp.RutaAsignacionesDietas+"/") {
		return a.registrarDenegacionPersonal(ctx, rutaPeticion, metodo, estado, actorRef, false)
	}
	if a == nil || a.registrador == nil || ctx == nil {
		return ErrComposicionBorradoresDietasNoDisponible
	}
	motivo := dietasports.MotivoFronteraDependencia
	switch estado {
	case http.StatusUnauthorized:
		motivo = dietasports.MotivoFronteraAutenticacion
	case http.StatusForbidden:
		motivo = dietasports.MotivoFronteraAccesoDenegado
	}
	ruta := dietasports.RutaAuditoriaFronteraComision
	if strings.HasPrefix(rutaPeticion, dietashttp.RutaBorradores+"/circuito") {
		ruta = dietasports.RutaAuditoriaFronteraCircuito
	} else if rutaPeticion != dietashttp.RutaBorradores {
		ruta = dietasports.RutaAuditoriaFronteraDetalle
	}
	var aleatorio [16]byte
	correlacion := "corr_no_disponible"
	if _, err := rand.Read(aleatorio[:]); err == nil {
		correlacion = "corr_" + hex.EncodeToString(aleatorio[:])
	}
	orden := dietasports.OrdenAuditoriaFronteraComision{CorrelacionRef: correlacion, Motivo: motivo, Ruta: ruta, Accion: accionFronteraComisionesDietas(rutaPeticion, metodo), ActorRef: actorRef}
	if orden.Validar() != nil {
		return ErrComposicionBorradoresDietasNoDisponible
	}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancelar()
	return a.registrador.RegistrarAuditoriaFronteraComision(ctxAuditoria, orden)
}

func (a *autoridadComisionesDietasDesarrollo) registrarDenegacionPersonal(ctx context.Context, rutaPeticion, metodo string, estado int, actorRef string, estructural bool) error {
	if a == nil || a.registradorPersonal == nil || ctx == nil {
		return ErrComposicionBorradoresDietasNoDisponible
	}
	ruta, accion := personalports.RutaFronteraRelacionesDietas, "consultar_relaciones"
	if rutaPeticion == personalhttp.RutaAsignacionesDietas || strings.HasPrefix(rutaPeticion, personalhttp.RutaAsignacionesDietas+"/") {
		ruta, accion = personalports.RutaFronteraAsignacionesDietas, "registrar_inicial"
		if rutaPeticion != personalhttp.RutaAsignacionesDietas {
			ruta, accion = personalports.RutaFronteraAsignacionDetalle, "consultar"
			if strings.HasSuffix(rutaPeticion, "/grupo") {
				ruta, accion = personalports.RutaFronteraAsignacionGrupo, "grupo_corregir"
			} else if metodo == http.MethodPut {
				accion = "corregir"
			}
		}
	}
	if estructural || !metodoComisionesDietasValido(rutaPeticion, metodo) {
		accion = "metodo_no_admitido"
	}
	motivo := personalports.MotivoFronteraPersonalDependencia
	switch estado {
	case http.StatusBadRequest:
		motivo = personalports.MotivoFronteraPersonalPeticion
	case http.StatusUnauthorized:
		motivo = personalports.MotivoFronteraPersonalAutenticacion
		actorRef = ""
	case http.StatusForbidden:
		motivo = personalports.MotivoFronteraPersonalDenegado
	case http.StatusNotFound:
		motivo = personalports.MotivoFronteraPersonalNoEncontrada
	case http.StatusMethodNotAllowed:
		motivo = personalports.MotivoFronteraPersonalMetodo
	case http.StatusNotAcceptable:
		motivo = personalports.MotivoFronteraPersonalRepresentacion
	case http.StatusConflict:
		motivo = personalports.MotivoFronteraPersonalConflicto
	}
	recursoRef := ""
	if !estructural && actorRef != "" && strings.HasPrefix(rutaPeticion, personalhttp.RutaAsignacionesDietas+"/") {
		resto := strings.TrimPrefix(rutaPeticion, personalhttp.RutaAsignacionesDietas+"/")
		if relacion, grupo := strings.CutSuffix(resto, "/grupo"); grupo {
			resto = relacion
		}
		if referenciaRutaRelacionPersonalDietas.MatchString(resto) {
			recursoRef = resto
		}
	}
	var aleatorio [16]byte
	correlacion := "corr_no_disponible"
	if _, err := rand.Read(aleatorio[:]); err == nil {
		correlacion = "corr_" + hex.EncodeToString(aleatorio[:])
	}
	orden := personalports.OrdenAuditoriaFronteraAsignacionDietas{CorrelacionRef: correlacion, Motivo: motivo, Ruta: ruta, Accion: accion, ActorRef: actorRef, RecursoRef: recursoRef, EstadoHTTP: estado}
	if orden.Validar() != nil {
		return ErrComposicionBorradoresDietasNoDisponible
	}
	ctxAuditoria, cancelar := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancelar()
	return a.registradorPersonal.RegistrarAuditoriaFronteraAsignacionDietas(ctxAuditoria, orden)
}

// denegarConMotivo audita el 403 y devuelve solo el código cerrado del
// motivo; la sesión no llegó a acreditar actor, así que no se atribuye.
func (a *autoridadComisionesDietasDesarrollo) denegarConMotivo(w http.ResponseWriter, r *http.Request, motivo string) {
	if a.registrarDenegacion(r.Context(), r.URL.Path, r.Method, http.StatusForbidden, "") != nil {
		responderDenegacionComisionesDietas(w, http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	responderDenegacionComisionesDietas(w, http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(struct {
		Codigo string `json:"codigo"`
	}{Codigo: motivo})
}

func responderDenegacionComisionesDietas(w http.ResponseWriter, estado int) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(estado)
}

type autoridadExactasConDietas struct {
	delegada vechttp.AutoridadRutasExactas
	dietas   *autoridadComisionesDietasDesarrollo
}

func (a autoridadExactasConDietas) AutorizarRutaExacta(ctx context.Context, ruta string) error {
	if !esRutaComisionesDietas(ruta) {
		return a.delegada.AutorizarRutaExacta(ctx, ruta)
	}
	if ctx == nil {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	c, ok := ctx.Value(claveContextoComisionesDietas{}).(contextoComisionesDietas)
	if !ok {
		return vechttp.ErrAutenticacionRutaExactaRequerida
	}
	if a.dietas == nil || c.autoridad != a.dietas || c.ruta != ruta || !metodoComisionesDietasValido(ruta, c.metodo) || c.seguridad.Resultado.Validar() != nil || !c.seguridad.Vinculo.VigenteEn(a.dietas.reloj.Ahora(), c.seguridad.Resultado) {
		actorRef := ""
		if a.dietas != nil && a.dietas.reloj != nil && c.autoridad == a.dietas {
			actorRef = actorVerificadoComisionesDietas(c.seguridad.Vinculo, c.seguridad.Resultado, a.dietas.reloj.Ahora())
		}
		if a.dietas != nil && a.dietas.registrarDenegacion(ctx, ruta, c.metodo, http.StatusForbidden, actorRef) != nil {
			return ErrComposicionBorradoresDietasNoDisponible
		}
		return vechttp.ErrAccesoRutaExactaDenegado
	}
	return nil
}

func nuevasComisionesDietasDesarrollo(cfg config.Config, resolvedor vechttp.DemoIdentityResolver, derivador *derivadorIdentidadOperacionDesarrollo, material materialDietasDesdeCTDesarrollo) (*autoridadComisionesDietasDesarrollo, error) {
	activo, err := cfg.DietasBorradoresDesarrolloActivos()
	if err != nil {
		return nil, err
	}
	if !activo {
		return nil, nil
	}
	identidad, ok := resolvedor.(*resolvedorIdentidadDesarrollo)
	if !ok || identidad == nil || derivador == nil || !derivador.valido() || !material.completo() {
		return nil, errComposicionDietasEn()
	}
	ruta := filepath.Join(cfg.DevelopmentMaterialDir, "identidad", "dietas-comisiones.json")
	contenido, err := leerFicheroMaterialSeguro(ruta, 256<<10)
	if err != nil || validarClavesJSONUnicas(contenido) != nil {
		return nil, errComposicionDietasEn()
	}
	defer borrarBytes(contenido)
	var c configuracionComisionesDietasDesarrollo
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa || len(c.Cuentas) == 0 || len(c.Cuentas) > 64 || c.MotivoPersonal.Validar() != nil || c.MotivoCrear.Validar() != nil || c.MotivoConsultar != c.MotivoCrear || c.MotivoPersonal.CatalogoID != c.MotivoCrear.CatalogoID {
		return nil, errComposicionDietasEn()
	}
	cuentas := map[string]cuentaRutasDietasDesarrollo{}
	for _, cuenta := range c.Cuentas {
		b, e := hex.DecodeString(cuenta.CertificadoSHA256)
		if e != nil || len(b) != sha256.Size || hex.EncodeToString(b) != cuenta.CertificadoSHA256 || cuenta.Sujeto == "" || cuenta.CuentaRef == "" || cuenta.PerfilRef == "" {
			return nil, errComposicionDietasEn()
		}
		var digest [32]byte
		copy(digest[:], b)
		principal, existe := identidad.porHuella[digest]
		if !existe || principal.ID != cuenta.Sujeto || principal.AuthMethod != core.AuthMethodCertificate || principal.AuthAssurance != core.AuthAssuranceHigh || cuentas[cuenta.CertificadoSHA256].Sujeto != "" {
			return nil, errComposicionDietasEn()
		}
		cuentas[cuenta.CertificadoSHA256] = cuenta
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	entradas := []struct{ dsn, rol string }{{c.DSNRegistroIdentidad, "vec_identidad_sesiones_v1_registrador"}, {c.DSNRevalidacionIdentidad, "vec_identidad_sesiones_v1_revalidador"}, {c.DSNContexto, "vec_contexto_actor_v1_runtime"}, {c.DSNFuenteAutorizacion, "vec_autorizacion_fuente"}, {c.DSNRegistroAutorizacion, "vec_autorizacion_registro"}, {c.DSNMotivos, "vec_autorizacion_motivos_evaluador"}}
	var pools []*pgxpool.Pool
	var propios *poolsPostgreSQLDietasDesarrollo
	var auditoria *pgxpool.Pool
	var asignacionPersonal, auditoriaPersonal *pgxpool.Pool
	completa := false
	cerrar := func() {
		if propios != nil {
			propios.Cerrar()
		}
		if auditoria != nil {
			auditoria.Close()
		}
		if asignacionPersonal != nil {
			asignacionPersonal.Close()
		}
		if auditoriaPersonal != nil {
			auditoriaPersonal.Close()
		}
		for _, p := range pools {
			p.Close()
		}
	}
	defer func() {
		if !completa {
			cerrar()
		}
	}()
	usuarios := map[string]bool{}
	for _, entrada := range entradas {
		pool, usuario, e := abrirPoolRutasDietas(ctx, entrada.dsn, entrada.rol)
		if e != nil || usuarios[usuario] {
			if pool != nil {
				pool.Close()
			}
			return nil, errComposicionDietasEn()
		}
		pools = append(pools, pool)
		usuarios[usuario] = true
	}
	propios, err = nuevosPoolsPostgreSQLDietasDesarrollo(ctx, cfg)
	if err != nil {
		// Sus errores son centinelas fijos (conexión, identidad, configuración),
		// sin DSN ni identidades: se conservan para diagnosticar el arranque.
		return nil, fmt.Errorf("%w: %w", errComposicionDietasEn(), err)
	}
	if err := acreditarPostimagenPersonalDietas(ctx, propios.Personal()); err != nil {
		return nil, fmt.Errorf("%w: %w", errComposicionDietasEn(), err)
	}
	for _, pool := range []*pgxpool.Pool{propios.Dietas(), propios.Personal()} {
		var usuario string
		if pool == nil || pool.QueryRow(ctx, `SELECT session_user::text`).Scan(&usuario) != nil || usuarios[usuario] {
			return nil, errComposicionDietasEn()
		}
		usuarios[usuario] = true
	}
	_, topologiaDietas, err := acreditarPoolPostgreSQLDietasDesarrollo(ctx, propios.Dietas(), rolDietasBorradoresPostgreSQLDesarrollo)
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	dsnAsignacion, err := cfg.DietasBorradoresPostgreSQL.DSNAsignacionPersonal()
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	usuarioAsignacion := ""
	var topologiaAsignacion topologiaPostgreSQLDietasDesarrollo
	asignacionPersonal, usuarioAsignacion, topologiaAsignacion, err = abrirPoolPersonalAsignacionDietas(ctx, dsnAsignacion, perfilesPersonalAsignacionDietas[0])
	if err != nil || usuarios[usuarioAsignacion] || !topologiaDietas.igual(topologiaAsignacion) {
		return nil, errComposicionDietasEn()
	}
	if acreditarFuncionesAsignacionPersonalDietas(ctx, asignacionPersonal) != nil {
		return nil, errComposicionDietasEn()
	}
	usuarios[usuarioAsignacion] = true
	dsnAuditoriaPersonal, err := cfg.DietasBorradoresPostgreSQL.DSNAuditoriaPersonal()
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	usuarioAuditoriaPersonal := ""
	var topologiaAuditoriaPersonal topologiaPostgreSQLDietasDesarrollo
	auditoriaPersonal, usuarioAuditoriaPersonal, topologiaAuditoriaPersonal, err = abrirPoolPersonalAsignacionDietas(ctx, dsnAuditoriaPersonal, perfilesPersonalAsignacionDietas[1])
	if err != nil || usuarios[usuarioAuditoriaPersonal] || !topologiaDietas.igual(topologiaAuditoriaPersonal) {
		return nil, errComposicionDietasEn()
	}
	usuarios[usuarioAuditoriaPersonal] = true
	registradorPersonal, err := personalpg.NuevoRegistradorAuditoriaFronteraAsignacionPostgreSQL(auditoriaPersonal)
	if err != nil || registradorPersonal.Preflight(ctx) != nil {
		return nil, errComposicionDietasEn()
	}
	auditoria, usuarioAuditoria, err := abrirPoolAuditoriaFronteraDietasDesarrollo(ctx, c.DSNAuditoriaFrontera)
	if err != nil || usuarios[usuarioAuditoria] {
		return nil, errComposicionDietasEn()
	}
	registrador, err := dietaspg.NuevoRegistradorAuditoriaFronteraComisionPostgreSQL(auditoria)
	if err != nil || registrador.Preflight(ctx) != nil {
		return nil, errComposicionDietasEn()
	}
	registro, err := identidadpg.NuevoRegistroSesionesPostgreSQL(ctx, pools[0], pools[1], &seudonimizadorSesionDesarrollo{derivador: derivador}, espacioIdentidadSesionDesarrollo, dominioIdentidadSesionDesarrollo)
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	revalidador, err := identidadpg.NuevoRevalidadorAutenticacionActorPostgreSQL(ctx, pools[1])
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	resolutor, err := contextopg.NuevoResolutorRegistroContextoActorPostgreSQLV2(ctx, pools[2])
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	reloj := relojRutasDietas{}
	servicioContexto, err := servicioContextoActorDietas(resolutor, reloj)
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	contextos, err := vecapp.NuevaAutoridadContextoActorRegistradoV2(servicioContexto)
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	fuente, err := vecpg.NuevoAlmacenAutorizacion(pools[3])
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	registroAutorizacion, err := vecpg.NuevoAlmacenAutorizacion(pools[4])
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	motivos, err := vecpg.NuevoValidadorReferenciaMotivoPostgreSQLV2(pools[5], c.MotivoPersonal.CatalogoID)
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	autorizador, err := vecapp.NuevoServicioAutorizacionSolicitudLigadaV3(fuente, registroAutorizacion, registroAutorizacion, motivos, reloj, seguridad.GeneradorReferenciasCriptograficas{}, vecapp.ConfiguracionServicioAutorizacion{VigenciaDecision: 30 * time.Second})
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	emisores := make(map[string]emisorMaterialDietasDesarrollo)
	for _, descriptor := range descriptoresMaterialDietasDesarrollo() {
		if !emisorAsignacionDietasMontable(descriptor.Audiencia, catalogoValidadoresCompetentesAsignacionDietas) {
			continue
		}
		p := material.adicionales[descriptor.Audiencia]
		emisor, e := nuevoEmisorMaterialRenovableCTDesarrollo(autorizador, p)
		if e != nil {
			return nil, errComposicionDietasEn()
		}
		emisores[descriptor.Audiencia] = emisor
	}
	emisoresPorAccion := map[string]emisorMaterialDietasDesarrollo{
		"dietas.borrador.propio.editar":                emisores[audienciaConsumoEditarDietasDesarrollo],
		"dietas.borrador.propio.borrar":                emisores[audienciaConsumoBorrarDietasDesarrollo],
		"dietas.borrador.propio.enviar":                emisores[audienciaConsumoEnviarDietasDesarrollo],
		"dietas.documento.propio.consultar":            emisores[audienciaConsumoDocumentoDietasDesarrollo],
		"personal.asignacion_dietas.consultar":         emisores[audienciaConsumoConsultarAsignacionDietas],
		"personal.asignacion_dietas.registrar_inicial": emisores[audienciaConsumoRegistrarAsignacionDietas],
		"personal.asignacion_dietas.corregir":          emisores[audienciaConsumoCorregirAsignacionDietas],
		"personal.asignacion_dietas.grupo_corregir":    emisores[audienciaConsumoCorregirGrupoDietas],
	}
	// Alta y corrección completa D7 solo tienen emisor con el catálogo abierto.
	for accion, emisor := range emisoresPorAccion {
		if emisor == nil {
			delete(emisoresPorAccion, accion)
		}
	}
	nonce, err := nonceRutasDietas()
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	base := &autoridadRutasDietasDesarrollo{resolvedor: identidad, cuentas: cuentas, registro: registro, revalidador: revalidador, contextos: contextos, reloj: reloj, instancia: nonce}
	a := &autoridadComisionesDietasDesarrollo{base: base, reloj: reloj, cuentas: cuentas, registrador: registrador, registradorPersonal: registradorPersonal, cerrar: cerrar}
	seguridadComisiones := seguridadComisionesDietasDesarrollo{autoridad: a}
	calculador, err := nuevoCasoUsoCalculoRutas(cfg)
	if err != nil || calculador == nil {
		return nil, errComposicionDietasEn()
	}
	tarifas, err := dietaspg.NuevoRepositorioTarifasProvisionales(propios.Dietas())
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	puntos := map[string]dietasports.CoordenadaRuta{}
	for _, punto := range dietas.ProvinceRoutePoints() {
		puntos[punto.Code] = dietasports.CoordenadaRuta{Latitud: punto.Latitude, Longitud: punto.Longitude, Nombre: punto.Name}
	}
	preparador, err := dietasapp.NuevoPreparadorComision(puntos, calculador, tarifas)
	if err != nil {
		return nil, errComposicionDietasEn()
	}
	rutas, colecciones, err := componerBorradoresDietas(dependenciasBorradoresDietas{personal: propios.Personal(), personalAsignacion: asignacionPersonal, dietas: propios.Dietas(), auditoriaPersonal: registradorPersonal, seguridad: seguridadComisiones, reloj: reloj, emisorPersonal: &emisorComisionesDietasDesarrollo{personal: emisores[audienciaConsumoPersonalDietasDesarrollo], adicionales: emisoresPorAccion}, emisorDietas: &emisorComisionesDietasDesarrollo{crear: emisores[audienciaConsumoCrearDietasDesarrollo], consultar: emisores[audienciaConsumoConsultarDietasDesarrollo], adicionales: emisoresPorAccion}, motivoPersonal: c.MotivoPersonal, motivoDietas: c.MotivoCrear, preparador: preparador})
	if err != nil {
		return nil, err
	}
	a.rutas, a.colecciones = rutas, colecciones
	completa = true
	return a, nil
}

// errComposicionDietasEn conserva ErrComposicionBorradoresDietasNoDisponible y
// añade solo fichero y línea del rechazo: sin DSN, identidades ni material.
func errComposicionDietasEn() error {
	_, fichero, linea, ok := runtime.Caller(1)
	if !ok {
		return ErrComposicionBorradoresDietasNoDisponible
	}
	return fmt.Errorf("%w (%s:%d)", ErrComposicionBorradoresDietasNoDisponible, filepath.Base(fichero), linea)
}
