package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

var errSesionFirmanteV2NoDisponible = errors.New("contratacion temporal: sesion del firmante no disponible")
var errSesionFirmanteV2Denegada = errors.New("contratacion temporal: sesion del firmante denegada")

// La fuente común decide cómo autenticar, registrar y revalidar la sesión.
// Este consumidor sólo admite el certificado del canal TLS verificado y coteja
// su resultado con la selección AUT56 y la persona del certificado CA25.
type autoridadSesionFirmanteV2 struct {
	fuente       ports.FuenteSesionFirmanteV2
	reloj        vp.Reloj
	preacreditar func(*http.Request) error
}

type capsulaSesionFirmanteV2 struct {
	autoridad    *autoridadSesionFirmanteV2
	peticion     *http.Request
	seleccion    ports.SeleccionFirmanteV2
	personaCA25  string
	huellaAUT56  string
	verificadoEn time.Time
	validoHasta  time.Time
	consumida    atomic.Bool
	mu           sync.Mutex
	sesion       ports.SesionFirmanteV2
	fallida      bool
}

func nuevaAutoridadSesionFirmanteV2ConFuente(fuente ports.FuenteSesionFirmanteV2, reloj vp.Reloj,
	preacreditar func(*http.Request) error,
) (*autoridadSesionFirmanteV2, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(fuente) || dependenciaEsNulaContratacionTemporalDesarrollo(reloj) {
		return nil, errSesionFirmanteV2NoDisponible
	}
	return &autoridadSesionFirmanteV2{fuente: fuente, reloj: reloj, preacreditar: preacreditar}, nil
}

// acreditar no acepta identidad ambiental. La selección y la referencia CA25
// proceden del contenedor privado que construye la fuente competencial 3b.
func (a *autoridadSesionFirmanteV2) acreditar(r *http.Request, seleccion ports.SeleccionFirmanteV2,
	personaCA25, huellaAUT56 string,
) (*capsulaSesionFirmanteV2, error) {
	if a == nil || r == nil || r.URL == nil || r.Context().Err() != nil ||
		r.URL.Path != httpinterno.RutaRegistroFirmaVec || r.URL.RawPath != "" || r.URL.RawQuery != "" ||
		r.URL.ForceQuery || r.URL.EscapedPath() != r.URL.Path || r.URL.Scheme != "" || r.URL.Host != "" ||
		r.URL.User != nil || r.URL.Opaque != "" || r.URL.Fragment != "" || r.URL.RawFragment != "" ||
		r.Method != http.MethodPost ||
		len(r.Header.Values("Cookie")) != 0 || len(r.Header.Values("Authorization")) != 0 ||
		cabeceraIdentidadAmbientalPresente(r.Header) || a.reloj == nil || a.fuente == nil {
		return nil, errSesionFirmanteV2Denegada
	}
	if a.preacreditar != nil && a.preacreditar(r) != nil {
		return nil, errSesionFirmanteV2Denegada
	}
	ahora := a.reloj.Ahora()
	hasta, huella, ok := certificadoSesionFirmanteV2(r, ahora)
	if !ok || huella != huellaAUT56 || seleccion.PersonaRef == "" ||
		seleccion.PersonaRef != personaCA25 || seleccion.CuentaRef == "" || seleccion.PerfilActivoRef == "" ||
		seleccion.RolID == "" ||
		seleccion.VinculoCertificado.Referencia == "" || seleccion.VinculoCertificado.Version == 0 ||
		!huellaSHA256ValidaContratacionTemporalDesarrollo(seleccion.VinculoCertificado.HuellaSHA256) {
		return nil, errSesionFirmanteV2Denegada
	}
	return &capsulaSesionFirmanteV2{autoridad: a, peticion: r, seleccion: seleccion,
		personaCA25: personaCA25, huellaAUT56: huella, verificadoEn: ahora, validoHasta: hasta}, nil
}

// certificadoSesionFirmanteV2 exige una única cadena comprobada por TLS y la
// hoja exacta presentada en el canal. La fuente común debe confirmar además
// revocación, política de garantía y cuenta; mTLS aislado no concede sesión.
func certificadoSesionFirmanteV2(r *http.Request, ahora time.Time) (time.Time, string, bool) {
	// La hoja se devuelve sólo como huella y fecha: no hay clave transportable.
	if r == nil || r.TLS == nil || !r.TLS.HandshakeComplete || r.TLS.Version != tls.VersionTLS13 ||
		len(r.TLS.PeerCertificates) != 1 || len(r.TLS.VerifiedChains) != 1 || len(r.TLS.VerifiedChains[0]) < 2 {
		return time.Time{}, "", false
	}
	hoja, verificada := r.TLS.PeerCertificates[0], r.TLS.VerifiedChains[0][0]
	if hoja == nil || verificada == nil || !bytes.Equal(hoja.Raw, verificada.Raw) ||
		ahora.Before(verificada.NotBefore) || !ahora.Before(verificada.NotAfter) {
		return time.Time{}, "", false
	}
	huella := sha256.Sum256(verificada.Raw)
	return verificada.NotAfter.UTC().Truncate(time.Microsecond), hex.EncodeToString(huella[:]), true
}

func (a *autoridadSesionFirmanteV2) abrir(r *http.Request, c *capsulaSesionFirmanteV2) (core.VinculoAutenticacionActorV2,
	core.ResultadoContextoActorRegistradoV2, error,
) {
	if r == nil {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, errSesionFirmanteV2Denegada
	}
	return a.abrirConContexto(r.Context(), r, c)
}

func (a *autoridadSesionFirmanteV2) abrirConContexto(ctx context.Context, r *http.Request, c *capsulaSesionFirmanteV2) (core.VinculoAutenticacionActorV2,
	core.ResultadoContextoActorRegistradoV2, error,
) {
	var vacio core.VinculoAutenticacionActorV2
	var sinContexto core.ResultadoContextoActorRegistradoV2
	if a == nil || ctx == nil || ctx.Err() != nil || c == nil || c.autoridad != a || c.peticion != r ||
		!c.consumida.CompareAndSwap(false, true) || !a.peticionVigente(r, c) {
		return vacio, sinContexto, errSesionFirmanteV2Denegada
	}
	solicitud := ports.SolicitudSesionFirmanteV2{CertificadoCanalSHA256: c.huellaAUT56,
		PersonaEsperadaRef: c.personaCA25, CuentaEsperadaRef: c.seleccion.CuentaRef,
		PerfilEsperadoRef: c.seleccion.PerfilActivoRef, CertificadoVerificadoEn: c.verificadoEn,
		CertificadoTLSValidoHasta: c.validoHasta}
	sesion, err := a.fuente.AbrirSesionFirmanteV2(ctx, solicitud)
	if err != nil || dependenciaEsNulaContratacionTemporalDesarrollo(sesion) {
		return vacio, sinContexto, errSesionFirmanteV2NoDisponible
	}
	c.mu.Lock()
	c.sesion = sesion
	c.mu.Unlock()
	return a.revalidarConContexto(ctx, r, c)
}

// revalidar consulta el registro común en cada invocación del emisor, sin
// consumir otra cápsula ni crear una segunda sesión para la petición.
func (a *autoridadSesionFirmanteV2) revalidar(r *http.Request, c *capsulaSesionFirmanteV2) (core.VinculoAutenticacionActorV2,
	core.ResultadoContextoActorRegistradoV2, error,
) {
	if r == nil {
		return core.VinculoAutenticacionActorV2{}, core.ResultadoContextoActorRegistradoV2{}, errSesionFirmanteV2Denegada
	}
	return a.revalidarConContexto(r.Context(), r, c)
}

func (a *autoridadSesionFirmanteV2) revalidarConContexto(ctx context.Context, r *http.Request, c *capsulaSesionFirmanteV2) (core.VinculoAutenticacionActorV2,
	core.ResultadoContextoActorRegistradoV2, error,
) {
	var vacio core.VinculoAutenticacionActorV2
	var sinContexto core.ResultadoContextoActorRegistradoV2
	if a == nil || ctx == nil || ctx.Err() != nil || c == nil || c.autoridad != a || c.peticion != r || !c.consumida.Load() || !a.peticionVigente(r, c) {
		return vacio, sinContexto, errSesionFirmanteV2Denegada
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.fallida || dependenciaEsNulaContratacionTemporalDesarrollo(c.sesion) {
		return vacio, sinContexto, errSesionFirmanteV2Denegada
	}
	e, err := c.sesion.RevalidarSesionFirmanteV2(ctx)
	if err != nil {
		c.fallida = true
		return vacio, sinContexto, errSesionFirmanteV2Denegada
	}
	ahora := a.reloj.Ahora()
	datos, err := e.Vinculo.Datos()
	if err != nil || e.Resultado.Validar() != nil || e.Vinculo.ValidarPara(e.Resultado) != nil ||
		!e.Vinculo.VigenteEn(ahora, e.Resultado) || e.CertificadoCanalSHA256 != c.huellaAUT56 ||
		e.CertificadoValidoHasta.IsZero() || ahora.Before(e.Resultado.Contexto.ResueltoEn) ||
		!ahora.Before(e.CertificadoValidoHasta) || e.CertificadoValidoHasta.After(c.validoHasta) ||
		datos.CuentaRef != c.seleccion.CuentaRef || datos.PerfilActivoRef != c.seleccion.PerfilActivoRef ||
		datos.PrincipalID != c.personaCA25 || datos.CuentaPrivilegiada ||
		datos.MetodoObservado != core.AuthMethodCertificate || datos.GarantiaObservada != core.AuthAssuranceHigh ||
		datos.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 ||
		e.Resultado.Contexto.PersonaRef != c.personaCA25 ||
		!e.Resultado.Contexto.AlcanceProyecciones().IncluyeEmpleado() || !a.peticionVigente(r, c) {
		c.fallida = true
		return vacio, sinContexto, errSesionFirmanteV2Denegada
	}
	return e.Vinculo, e.Resultado, nil
}

func (a *autoridadSesionFirmanteV2) peticionVigente(r *http.Request, c *capsulaSesionFirmanteV2) bool {
	if a == nil || r == nil || c == nil || r.Context().Err() != nil || r.URL == nil ||
		r.URL.Path != httpinterno.RutaRegistroFirmaVec || r.Method != http.MethodPost || a.reloj == nil ||
		a.fuente == nil || !a.reloj.Ahora().Before(c.validoHasta) {
		return false
	}
	_, huella, ok := certificadoSesionFirmanteV2(r, a.reloj.Ahora())
	return ok && huella == c.huellaAUT56
}
