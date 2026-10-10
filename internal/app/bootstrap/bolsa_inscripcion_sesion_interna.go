package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	httpinscripcion "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinscripcion"
	"vec-diputacion-granada/internal/modules/bolsa/application/inscripcion"
	vechttp "vec-diputacion-granada/internal/vec/adapters/httpapi"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Esta adaptación consume el registro, la revalidación y el resolutor V2
// comunes de la superficie interna. Las cuentas/perfiles llegan del gobierno
// nominal antes del arranque; aquí no se crean perfiles ni concesiones. La
// primera versión sólo atiende a RRHH: el canal de empleado está en
// docs/estudio_requisitos/pendientes_v2.md.
type sesionInternaInscripcionBolsa struct {
	comun   *autoridadRutasDietasDesarrollo
	cuentas map[string]cuentaRutasDietasDesarrollo
}

var _ SesionInscripcionBolsa = (*sesionInternaInscripcionBolsa)(nil)

var referenciaCuentaInscripcionInterna = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9:._/#-]{2,255}$`)

func nuevaSesionInternaInscripcionBolsa(comun *autoridadRutasDietasDesarrollo,
	cuentas []cuentaRutasDietasDesarrollo) (SesionInscripcionBolsa, error) {
	if comun == nil || comun.resolvedor == nil || comun.registro == nil || comun.revalidador == nil ||
		comun.contextos == nil || comun.reloj == nil || len(cuentas) == 0 {
		return nil, inscripcion.ErrNoDisponible
	}
	porHuella := make(map[string]cuentaRutasDietasDesarrollo, len(cuentas))
	for _, cuenta := range cuentas {
		if !huellaCertificadoInscripcionValida(cuenta.CertificadoSHA256) || cuenta.Sujeto == "" ||
			!referenciaCuentaInscripcionInterna.MatchString(cuenta.CuentaRef) ||
			!referenciaCuentaInscripcionInterna.MatchString(cuenta.PerfilRef) {
			return nil, inscripcion.ErrNoDisponible
		}
		if _, existe := porHuella[cuenta.CertificadoSHA256]; existe {
			return nil, inscripcion.ErrNoDisponible
		}
		porHuella[cuenta.CertificadoSHA256] = cuenta
	}
	return &sesionInternaInscripcionBolsa{comun: comun, cuentas: porHuella}, nil
}

func (s *sesionInternaInscripcionBolsa) ResolverInscripcion(r *http.Request) (contextoSeguridadComunDesarrollo, AcreditacionSesionInscripcionBolsa, error) {
	var vacio contextoSeguridadComunDesarrollo
	var sin AcreditacionSesionInscripcionBolsa
	if s == nil || s.comun == nil || r == nil || r.URL == nil || r.Context().Err() != nil ||
		r.URL.RawPath != "" || r.URL.EscapedPath() != r.URL.Path || r.URL.ForceQuery ||
		(r.Method != http.MethodGet && r.Method != http.MethodPost) || !rutaInscripcionInterna(r.URL.Path) ||
		r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" ||
		r.Header.Get("X-Vec-Persona") != "" || r.Header.Get("X-Vec-Perfil") != "" || r.Header.Get("X-Vec-Actor") != "" ||
		r.TLS == nil || len(r.TLS.VerifiedChains) == 0 || len(r.TLS.VerifiedChains[0]) == 0 {
		return vacio, sin, inscripcion.ErrSesionAusente
	}
	peticion := peticionIdentidadConsultasContratacionTemporalDesarrollo(r)
	if peticion == nil || peticion.TLS == nil || len(peticion.TLS.VerifiedChains) == 0 || len(peticion.TLS.VerifiedChains[0]) == 0 {
		return vacio, sin, inscripcion.ErrSesionAusente
	}
	certificado := peticion.TLS.VerifiedChains[0][0]
	if certificado == nil || len(certificado.Raw) == 0 {
		return vacio, sin, inscripcion.ErrSesionAusente
	}
	huella := sha256.Sum256(certificado.Raw)
	certificadoSHA256 := hex.EncodeToString(huella[:])
	cuenta, presente := s.cuentas[certificadoSHA256]
	ahora := s.comun.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if !presente || ahora.Before(certificado.NotBefore) || !ahora.Before(certificado.NotAfter) {
		return vacio, sin, inscripcion.ErrSesionAusente
	}
	principal, err := s.comun.resolvedor.ResolveDemoIdentity(peticion.Context(), peticion)
	if err != nil || principal.ID != cuenta.Sujeto || principal.AuthMethod != core.AuthMethodCertificate ||
		principal.AuthAssurance != core.AuthAssuranceHigh || principal.Attributes["certificate_sha256"] != certificadoSHA256 {
		return vacio, sin, inscripcion.ErrSesionAusente
	}
	vinculo, resultado, err := s.comun.resolverSesion(peticion.Context(), peticion,
		&capsulaRutasDietasDesarrollo{autoridad: s.comun, peticion: peticion, cuenta: cuenta, instante: ahora})
	if err != nil {
		// Sólo una denegación de sesión es 401; un fallo del registro o del
		// contexto se presenta como servicio no disponible.
		if errors.Is(err, vechttp.ErrRutaDietasDenegada) {
			return vacio, sin, inscripcion.ErrSesionAusente
		}
		return vacio, sin, inscripcion.ErrNoDisponible
	}
	datos, err := vinculo.Datos()
	ahora = s.comun.reloj.Ahora().UTC().Truncate(time.Microsecond)
	if err != nil || resultado.Validar() != nil || vinculo.ValidarPara(resultado) != nil ||
		!vinculo.VigenteEn(ahora, resultado) || datos.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 ||
		datos.CuentaPrivilegiada || datos.CuentaRef != cuenta.CuentaRef || datos.PerfilActivoRef != cuenta.PerfilRef ||
		resultado.Contexto.PerfilActivoRef != cuenta.PerfilRef || resultado.Contexto.Instantanea.CuentaRef != cuenta.CuentaRef ||
		resultado.Contexto.PersonaRef != datos.PrincipalID {
		return vacio, sin, inscripcion.ErrSesionAusente
	}
	hasta := datos.SesionValidaHasta
	if certificado.NotAfter.Before(hasta) {
		hasta = certificado.NotAfter
	}
	if resultado.Contexto.Instantanea.VigenteHasta.Before(hasta) {
		hasta = resultado.Contexto.Instantanea.VigenteHasta
	}
	if !ahora.Before(hasta) {
		return vacio, sin, inscripcion.ErrSesionAusente
	}
	return contextoSeguridadComunDesarrollo{Vinculo: vinculo, Resultado: resultado}, AcreditacionSesionInscripcionBolsa{
		CertificadoHuellaSHA256: certificadoSHA256, Canal: string(core.SuperficieAutenticacionInternaCorporativaV1),
		PersonaRef: resultado.Contexto.PersonaRef, PerfilRef: cuenta.PerfilRef, CuentaRef: cuenta.CuentaRef,
		SesionRef: datos.SesionRef, AutenticacionRef: datos.AutenticacionRef,
		VerificadaEn: datos.AutenticacionVerificadaEn, ValidaHasta: hasta,
	}, nil
}

func rutaInscripcionInterna(path string) bool {
	return path == httpinscripcion.RutaRRHH || strings.HasPrefix(path, httpinscripcion.RutaRRHH+"/")
}
