package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"time"

	"vec-diputacion-granada/config"
	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	adminhttp "vec-diputacion-granada/internal/modules/administracion/adapters/http"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const exportadorAdministracionDesarrollo = "EXPORTER-VEC-ADMIN-DESARROLLO-v1"

// El proveedor consulta sesión/cuenta/perfil y permiso vigentes. El
// certificado nominal no concede permisos y no sustituye ese proveedor.
type revalidadorAccesoAdministracionDesarrollo interface {
	PrincipalAdministracionDesarrollo(context.Context) (vecdomain.Principal, error)
}

type autoridadAdministracionDesarrollo struct {
	identidad *identidadAdministracionDesarrollo
	proveedor revalidadorAccesoAdministracionDesarrollo
	direccion string
	vinculada atomic.Bool
}

type claveConexionAdministracionDesarrollo struct{}
type claveCapacidadAdministracionDesarrollo struct{}

type conexionAdministracionDesarrollo struct {
	autoridad *autoridadAdministracionDesarrollo
	conexion  *tls.Conn
}

// Solo la frontera de transporte puede crear esta capacidad. Caduca al
// terminar ServeHTTP y no sale del contexto de la petición que la originó.
type capacidadAdministracionDesarrollo struct {
	autoridad               *autoridadAdministracionDesarrollo
	identidad               *identidadAdministracionDesarrollo
	conexion                *tls.Conn
	peticion                context.Context
	ruta                    string
	metodo                  string
	certificadoVerificadoEn time.Time
	certificadoValidoHasta  time.Time
	activa                  atomic.Bool
}

func nuevaAutoridadAdministracionDesarrollo(cfg config.Config, resolvedor *resolvedorIdentidadDesarrollo, proveedor revalidadorAccesoAdministracionDesarrollo) (*autoridadAdministracionDesarrollo, error) {
	cfg = cfg.Normalize()
	if !cfg.DevelopmentEnabledByDoubleKey() || validarRedLocalDesarrollo(cfg) != nil || len(cfg.HTTPAllowedCIDRs) == 0 ||
		resolvedor == nil || resolvedor.administracion == nil || len(resolvedor.porHuella) != 1 || len(resolvedor.porSujeto) != 0 || dependenciaAdministracionNula(proveedor) {
		return nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	identidad := *resolvedor.administracion
	identidad.identidad.principal = clonarPrincipalDesarrollo(identidad.identidad.principal)
	registrada, existe := resolvedor.porHuella[identidad.identidad.huella]
	if !existe || !reflect.DeepEqual(registrada, identidad.identidad.principal) ||
		registrada.Validate() != nil || registrada.AuthMethod != vecdomain.AuthMethodCertificate ||
		registrada.AuthAssurance != vecdomain.AuthAssuranceHigh || len(registrada.Roles) != 1 || registrada.Roles[0] != "administrador" ||
		len(registrada.Permissions) != 0 || registrada.Attributes["autoridad"] != AutoridadNoAutoritativa ||
		registrada.Attributes["perfil_ejecucion"] != config.ExecutionProfileDevelopment ||
		!referenciasIdentidadAdministracionDesarrolloValidas(&identidad) ||
		registrada.Attributes["certificate_sha256"] != hex.EncodeToString(identidad.identidad.huella[:]) {
		return nil, ErrConfiguracionCorreoAdministracionNoDisponible
	}
	return &autoridadAdministracionDesarrollo{identidad: &identidad, proveedor: proveedor, direccion: cfg.Address}, nil
}

// Se llama desde la composición sobre el servidor exclusivo ADMIN después de
// instalar su TLS. Sólo acredita la ruta administrativa exacta.
// Un Handler invocado fuera de este http.Server no puede emitir capacidades.
func (a *autoridadAdministracionDesarrollo) vincularServidor(servidor *http.Server) error {
	if a == nil || servidor == nil || servidor.Handler == nil || servidor.ConnContext != nil || servidor.Addr != a.direccion ||
		servidor.TLSConfig == nil || servidor.TLSConfig.ClientAuth != tls.RequireAndVerifyClientCert ||
		servidor.TLSConfig.ClientCAs == nil || servidor.TLSConfig.MinVersion != tls.VersionTLS13 || servidor.TLSConfig.MaxVersion != tls.VersionTLS13 ||
		!a.vinculada.CompareAndSwap(false, true) {
		return ErrConfiguracionCorreoAdministracionNoDisponible
	}
	servidor.ConnContext = func(ctx context.Context, conexion net.Conn) context.Context {
		tlsConexion, ok := conexion.(*tls.Conn)
		if !ok {
			return ctx
		}
		return context.WithValue(ctx, claveConexionAdministracionDesarrollo{}, conexionAdministracionDesarrollo{autoridad: a, conexion: tlsConexion})
	}
	servidor.Handler = a.proteger(servidor.Handler)
	return nil
}

func (a *autoridadAdministracionDesarrollo) proteger(siguiente http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil || r.URL == nil || siguiente == nil {
			http.Error(w, "servicio no disponible", http.StatusServiceUnavailable)
			return
		}
		lectura := rutaLecturaAdministracionDesarrollo(r.URL.Path, r.Method)
		if r.URL.Path != adminhttp.RutaConfiguracionCorreo && !lectura {
			if r.URL.Path == "/administracion" || strings.HasPrefix(r.URL.Path, "/administracion/") || r.URL.Path == "/portal-empleado/portal-i18n.js" {
				http.NotFound(w, r)
				return
			}
			siguiente.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store, no-transform")
		capacidad, ok := a.acreditarPeticion(r)
		if lectura {
			capacidad, ok = a.acreditarCanalCertificado(r)
		}
		if !ok {
			http.Error(w, "acceso denegado", http.StatusForbidden)
			return
		}
		capacidad.activa.Store(true)
		defer capacidad.activa.Store(false)
		ctx := context.WithValue(r.Context(), claveCapacidadAdministracionDesarrollo{}, capacidad)
		if lectura {
			if _, err := a.principalAdministracion(ctx); err != nil {
				http.Error(w, "acceso denegado", http.StatusForbidden)
				return
			}
		}
		siguiente.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Lectura de la superficie ADMIN: la ruta y el método originales quedan en la
// cápsula; no se convierten en una petición API ni en autorización de escritura.
func rutaLecturaAdministracionDesarrollo(ruta, metodo string) bool {
	if metodo != http.MethodGet && metodo != http.MethodHead {
		return false
	}
	switch ruta {
	case "/administracion", "/administracion/", "/administracion/index.html",
		"/administracion/administracion.js", "/administracion/configuracion-correo.js",
		"/administracion/configuracion-correo.css", "/administracion/tema.css", "/administracion/i18n.js",
		"/portal-empleado/portal-i18n.js":
		return true
	default:
		return false
	}
}

func (a *autoridadAdministracionDesarrollo) acreditarPeticion(r *http.Request) (*capacidadAdministracionDesarrollo, bool) {
	if r == nil || r.URL == nil || r.URL.Path != adminhttp.RutaConfiguracionCorreo || (r.Method != http.MethodGet && r.Method != http.MethodPut) {
		return nil, false
	}
	return a.acreditarCanalCertificado(r)
}

func (a *autoridadAdministracionDesarrollo) acreditarCanalCertificado(r *http.Request) (*capacidadAdministracionDesarrollo, bool) {
	if a == nil || a.identidad == nil || !a.vinculada.Load() || r == nil || r.URL == nil || r.TLS == nil ||
		r.Context().Err() != nil || r.URL.RawPath != "" || r.URL.RawQuery != "" || r.URL.ForceQuery ||
		r.URL.Opaque != "" || r.URL.Fragment != "" || r.URL.RawFragment != "" || r.URL.EscapedPath() != r.URL.Path ||
		!direccionRemotaLoopback(r.RemoteAddr) ||
		cabeceraIdentidadAmbientalPresente(r.Header) || r.Header.Get("Authorization") != "" || r.Header.Get("Cookie") != "" {
		return nil, false
	}
	conexion, ok := r.Context().Value(claveConexionAdministracionDesarrollo{}).(conexionAdministracionDesarrollo)
	if !ok || conexion.autoridad != a || conexion.conexion == nil {
		return nil, false
	}
	estado, remoto, ok := observarConexionAdministracionDesarrollo(conexion.conexion)
	if !ok || remoto != r.RemoteAddr {
		return nil, false
	}
	if !estado.HandshakeComplete || estado.Version != tls.VersionTLS13 || len(estado.VerifiedChains) != 1 ||
		len(estado.VerifiedChains[0]) < 2 || len(estado.PeerCertificates) == 0 ||
		!r.TLS.HandshakeComplete || r.TLS.Version != estado.Version || len(r.TLS.PeerCertificates) == 0 {
		return nil, false
	}
	certificado := estado.VerifiedChains[0][0]
	if certificado == nil || estado.PeerCertificates[0] == nil || r.TLS.PeerCertificates[0] == nil || certificado.IsCA ||
		!bytes.Equal(certificado.Raw, estado.PeerCertificates[0].Raw) || !bytes.Equal(certificado.Raw, r.TLS.PeerCertificates[0].Raw) ||
		sha256.Sum256(certificado.Raw) != a.identidad.identidad.huella {
		return nil, false
	}
	exportadorReal, valido := exportarVinculoAdministracionDesarrollo(&estado)
	if !valido {
		return nil, false
	}
	exportadorPeticion, valido := exportarVinculoAdministracionDesarrollo(r.TLS)
	if !valido || subtle.ConstantTimeCompare(exportadorReal, exportadorPeticion) != 1 {
		return nil, false
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	validoHasta := certificado.NotAfter.UTC()
	for _, eslabon := range estado.VerifiedChains[0] {
		if eslabon == nil || ahora.Before(eslabon.NotBefore) || !ahora.Before(eslabon.NotAfter) {
			return nil, false
		}
		if eslabon.NotAfter.Before(validoHasta) {
			validoHasta = eslabon.NotAfter.UTC()
		}
	}
	return &capacidadAdministracionDesarrollo{autoridad: a, identidad: a.identidad, conexion: conexion.conexion, peticion: r.Context(), ruta: r.URL.Path, metodo: r.Method, certificadoVerificadoEn: ahora, certificadoValidoHasta: validoHasta}, true
}

func observarConexionAdministracionDesarrollo(conexion *tls.Conn) (estado tls.ConnectionState, remoto string, valido bool) {
	// El valor cero de tls.Conn nunca procede de ConnContext y no tiene
	// transporte subyacente. Las pruebas de trasplante deben denegarse.
	defer func() {
		if recover() != nil {
			estado, remoto, valido = tls.ConnectionState{}, "", false
		}
	}()
	if conexion == nil || conexion.RemoteAddr() == nil {
		return tls.ConnectionState{}, "", false
	}
	return conexion.ConnectionState(), conexion.RemoteAddr().String(), true
}

func exportarVinculoAdministracionDesarrollo(estado *tls.ConnectionState) (material []byte, valido bool) {
	// Un estado fabricado carece del exportador privado de crypto/tls.
	// Su posible pánico se convierte únicamente aquí en denegación.
	defer func() {
		if recover() != nil {
			material, valido = nil, false
		}
	}()
	if estado == nil {
		return nil, false
	}
	material, err := estado.ExportKeyingMaterial(exportadorAdministracionDesarrollo, nil, sha256.Size)
	return material, err == nil && len(material) == sha256.Size
}

func (a *autoridadAdministracionDesarrollo) capacidadValida(ctx context.Context) (*capacidadAdministracionDesarrollo, bool) {
	if a == nil || a.identidad == nil || ctx == nil || ctx.Err() != nil || !a.vinculada.Load() {
		return nil, false
	}
	c, ok := ctx.Value(claveCapacidadAdministracionDesarrollo{}).(*capacidadAdministracionDesarrollo)
	if !ok || c == nil || c.autoridad != a || c.identidad != a.identidad || !c.activa.Load() ||
		c.peticion == nil || c.peticion.Err() != nil || c.conexion == nil ||
		!((c.ruta == adminhttp.RutaConfiguracionCorreo && (c.metodo == http.MethodGet || c.metodo == http.MethodPut)) || rutaLecturaAdministracionDesarrollo(c.ruta, c.metodo)) {
		return nil, false
	}
	conexion, ok := ctx.Value(claveConexionAdministracionDesarrollo{}).(conexionAdministracionDesarrollo)
	if !ok || conexion.autoridad != a || conexion.conexion != c.conexion {
		return nil, false
	}
	ahora := time.Now().UTC()
	if ahora.Before(c.certificadoVerificadoEn) || !ahora.Before(c.certificadoValidoHasta) || !ahora.Before(c.certificadoVerificadoEn.Add(2*time.Minute)) {
		return nil, false
	}
	return c, true
}

func capacidadAdministracionDesdeContexto(ctx context.Context) (*capacidadAdministracionDesarrollo, bool) {
	if ctx == nil || ctx.Err() != nil {
		return nil, false
	}
	c, ok := ctx.Value(claveCapacidadAdministracionDesarrollo{}).(*capacidadAdministracionDesarrollo)
	if !ok || c == nil || c.autoridad == nil {
		return nil, false
	}
	return c.autoridad.capacidadValida(ctx)
}

func (a *autoridadAdministracionDesarrollo) PrincipalConfiguracionCorreo(ctx context.Context) (vecdomain.Principal, error) {
	c, ok := a.capacidadValida(ctx)
	if !ok || c.ruta != adminhttp.RutaConfiguracionCorreo {
		return vecdomain.Principal{}, vechttp.ErrAccesoRutaExactaDenegado
	}
	return a.principalAdministracion(ctx)
}

func (a *autoridadAdministracionDesarrollo) principalAdministracion(ctx context.Context) (vecdomain.Principal, error) {
	c, ok := a.capacidadValida(ctx)
	if !ok || dependenciaAdministracionNula(a.proveedor) {
		return vecdomain.Principal{}, vechttp.ErrAccesoRutaExactaDenegado
	}
	principal, err := a.proveedor.PrincipalAdministracionDesarrollo(ctx)
	if _, vigente := a.capacidadValida(ctx); err != nil || !vigente || principal.Validate() != nil || principal.ID != c.identidad.personaRef ||
		principal.AuthMethod != vecdomain.AuthMethodCertificate || principal.AuthAssurance != vecdomain.AuthAssuranceHigh ||
		!principal.HasPermission(adminmodule.PermissionIntegrationsManage) {
		return vecdomain.Principal{}, vechttp.ErrAccesoRutaExactaDenegado
	}
	return clonarPrincipalDesarrollo(principal), nil
}

func (a *autoridadAdministracionDesarrollo) AutorizarRutaExacta(ctx context.Context, ruta string) error {
	if ruta != adminhttp.RutaConfiguracionCorreo {
		return vechttp.ErrAccesoRutaExactaDenegado
	}
	_, err := a.PrincipalConfiguracionCorreo(ctx)
	return err
}

func (a *autoridadAdministracionDesarrollo) VerificarAccesoConfiguracionCorreo(ctx context.Context, principal vecdomain.Principal) error {
	actual, err := a.PrincipalConfiguracionCorreo(ctx)
	if err != nil || !reflect.DeepEqual(actual, principal) {
		return vechttp.ErrAccesoRutaExactaDenegado
	}
	return nil
}
