package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	httpinscripcion "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinscripcion"
	usuarioshttp "vec-diputacion-granada/internal/modules/usuarios/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
)

var (
	ErrSesionExternaInscripcionNoDisponible  = errors.New("bootstrap: sesion externa de inscripcion no disponible")
	ErrSesionExternaInscripcionNoAutenticada = errors.New("bootstrap: sesion externa de inscripcion no autenticada")
)

// AcreditacionSesionExternaInscripcion describe la misma resolución nominal
// que devuelve el vínculo. Es identidad para esta petición, no un permiso de
// lectura ni de escritura sobre una convocatoria.
type AcreditacionSesionExternaInscripcion struct {
	CertificadoHuellaSHA256 string
	Canal                   string
	PersonaRef              string
	PerfilRef               string
	CuentaRef               string
	SesionRef               string
	AutenticacionRef        string
	VerificadaEn            time.Time
	ValidaHasta             time.Time
}

const textoSesionInscripcionOculto = "[DATOS DE SESION OCULTOS]"

func (AcreditacionSesionExternaInscripcion) String() string {
	return textoSesionInscripcionOculto
}
func (AcreditacionSesionExternaInscripcion) GoString() string {
	return textoSesionInscripcionOculto
}
func (AcreditacionSesionExternaInscripcion) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(textoSesionInscripcionOculto))
}
func (AcreditacionSesionExternaInscripcion) MarshalJSON() ([]byte, error) {
	return []byte(`{"acreditacion_sesion_externa":"[OCULTA]"}`), nil
}
func (AcreditacionSesionExternaInscripcion) LogValue() slog.Value {
	return slog.StringValue(textoSesionInscripcionOculto)
}

// SesionExternaInscripcion reutiliza la autoridad, las conexiones y el
// registro de sesión del Área personal. No conserva una identidad entre
// peticiones ni construye un segundo servicio de autenticación.
type SesionExternaInscripcion struct {
	preferencias *autoridadPreferenciasUsuariosDesarrollo
}

// La causa sigue disponible para errors.Is/As internos. Ninguna representación
// textual del error incorpora el mensaje del registro, del SQL o del actor.
type falloSesionExternaInscripcion struct{ causa error }

func (falloSesionExternaInscripcion) Error() string {
	return ErrSesionExternaInscripcionNoDisponible.Error()
}
func (falloSesionExternaInscripcion) String() string {
	return ErrSesionExternaInscripcionNoDisponible.Error()
}
func (falloSesionExternaInscripcion) GoString() string {
	return ErrSesionExternaInscripcionNoDisponible.Error()
}
func (falloSesionExternaInscripcion) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(ErrSesionExternaInscripcionNoDisponible.Error()))
}
func (falloSesionExternaInscripcion) MarshalJSON() ([]byte, error) {
	return []byte(`{"error":"sesion_no_disponible"}`), nil
}
func (falloSesionExternaInscripcion) LogValue() slog.Value {
	return slog.StringValue(ErrSesionExternaInscripcionNoDisponible.Error())
}
func (e falloSesionExternaInscripcion) Unwrap() []error {
	return []error{ErrSesionExternaInscripcionNoDisponible, e.causa}
}

func NuevaSesionExternaInscripcion(preferencias *autoridadPreferenciasUsuariosDesarrollo) (*SesionExternaInscripcion, error) {
	if preferencias == nil || preferencias.base == nil || preferencias.base.resolvedor == nil ||
		dependenciaAutorizacionComunDesarrolloNula(preferencias.base.registro) ||
		dependenciaAutorizacionComunDesarrolloNula(preferencias.base.revalidador) ||
		dependenciaAutorizacionComunDesarrolloNula(preferencias.base.contextos) ||
		preferencias.superficie != core.SuperficieAutenticacionExternaPersonalV1 ||
		preferencias.ruta != usuarioshttp.RutaMisPreferenciasAreaPersonal || len(preferencias.cuentas) == 0 {
		return nil, ErrSesionExternaInscripcionNoDisponible
	}
	return &SesionExternaInscripcion{preferencias: preferencias}, nil
}

// ResolverSesionExterna exige la misma prueba mTLS y la misma cuenta
// configurada que Preferencias, y vuelve a registrar y revalidar la sesión.
// El consumidor debe solicitar aparte la autorización de su acción exacta.
func (s *SesionExternaInscripcion) ResolverSesionExterna(r *http.Request) (contextoSeguridadComunDesarrollo, AcreditacionSesionExternaInscripcion, error) {
	var vacio contextoSeguridadComunDesarrollo
	var sin AcreditacionSesionExternaInscripcion
	if s == nil || s.preferencias == nil || r == nil || r.URL == nil || r.Context().Err() != nil ||
		r.URL.RawPath != "" || r.URL.EscapedPath() != r.URL.Path || r.URL.ForceQuery ||
		(r.Method != http.MethodGet && r.Method != http.MethodPost) || !rutaInscripcionExterna(r.URL.Path) {
		return vacio, sin, ErrSesionExternaInscripcionNoAutenticada
	}
	segura, cuenta, certificado, ahora, err := s.preferencias.identificarCuentaCertificada(r)
	if err != nil {
		if errors.Is(err, ErrMaterialDesarrolloInvalido) {
			return vacio, sin, ErrSesionExternaInscripcionNoAutenticada
		}
		return vacio, sin, falloSesionExternaInscripcion{err}
	}
	if certificado == nil || len(certificado.Raw) == 0 {
		return vacio, sin, ErrSesionExternaInscripcionNoAutenticada
	}
	huella := sha256.Sum256(certificado.Raw)
	certificadoSHA256 := hex.EncodeToString(huella[:])
	if certificadoSHA256 != cuenta.CertificadoSHA256 {
		return vacio, sin, ErrSesionExternaInscripcionNoAutenticada
	}
	vinculo, resultado, err := s.preferencias.resolverSesion(segura, cuenta, ahora)
	if err != nil {
		return vacio, sin, falloSesionExternaInscripcion{err}
	}
	ahora = s.preferencias.reloj.Ahora().UTC().Truncate(time.Microsecond)
	datos, err := vinculo.Datos()
	if err != nil || resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil ||
		!vinculo.VigenteEn(ahora, resultado) || ahora.Before(certificado.NotBefore) || !ahora.Before(certificado.NotAfter) ||
		datos.Superficie != core.SuperficieAutenticacionExternaPersonalV1 ||
		datos.MetodoObservado != core.AuthMethodCertificate || datos.GarantiaObservada != core.AuthAssuranceHigh ||
		datos.CuentaPrivilegiada || datos.CuentaRef != cuenta.CuentaRef || datos.PerfilActivoRef != cuenta.PerfilRef ||
		resultado.Contexto.PersonaRef == "" || resultado.Contexto.PersonaRef != datos.PrincipalID ||
		resultado.Contexto.Instantanea.CuentaRef != cuenta.CuentaRef || resultado.Contexto.PerfilActivoRef != cuenta.PerfilRef {
		return vacio, sin, ErrSesionExternaInscripcionNoAutenticada
	}
	hasta := datos.SesionValidaHasta
	if certificado.NotAfter.Before(hasta) {
		hasta = certificado.NotAfter
	}
	instantanea := resultado.Contexto.Instantanea
	if instantanea.VigenteHasta.Before(hasta) {
		hasta = instantanea.VigenteHasta
	}
	for _, vinculoContexto := range instantanea.Vinculos {
		if vinculoContexto.VigenteHasta.Before(hasta) {
			hasta = vinculoContexto.VigenteHasta
		}
	}
	if !ahora.Before(hasta) || datos.AutenticacionVerificadaEn.After(ahora) {
		return vacio, sin, ErrSesionExternaInscripcionNoAutenticada
	}
	return contextoSeguridadComunDesarrollo{Vinculo: vinculo, Resultado: resultado}, AcreditacionSesionExternaInscripcion{
		CertificadoHuellaSHA256: certificadoSHA256, Canal: string(core.SuperficieAutenticacionExternaPersonalV1),
		PersonaRef: resultado.Contexto.PersonaRef, PerfilRef: datos.PerfilActivoRef, CuentaRef: datos.CuentaRef,
		SesionRef: datos.SesionRef, AutenticacionRef: datos.AutenticacionRef,
		VerificadaEn: datos.AutenticacionVerificadaEn, ValidaHasta: hasta,
	}, nil
}

// Las rutas de la persona aspirante tienen prefijo propio, fuera de Mi Bolsa.
func rutaInscripcionExterna(ruta string) bool {
	return ruta == httpinscripcion.RutaAbiertas || strings.HasPrefix(ruta, httpinscripcion.RutaAbiertas+"/") ||
		ruta == httpinscripcion.RutaPropias || strings.HasPrefix(ruta, httpinscripcion.RutaPropias+"/")
}
