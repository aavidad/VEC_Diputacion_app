package firmavec

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"reflect"
	"sync"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/firmaemisorv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmasv2"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// La autoridad implementada por CT liga una cápsula a esta misma petición.
// El puerto no admite un actor libre ni acepta datos del cuerpo HTTP.
type AutoridadSesionFirmanteV2 interface {
	AcreditarSesionFirmaVecV2(*http.Request, ports.SeleccionFirmanteV2, string, string) (SesionFirmanteAcreditadaV2, error)
}

type SesionFirmanteAcreditadaV2 interface {
	AbrirSesionFirmaVecV2(context.Context, *http.Request) (core.VinculoAutenticacionActorV2, core.ResultadoContextoActorRegistradoV2, error)
	RevalidarSesionFirmaVecV2(context.Context, *http.Request) (core.VinculoAutenticacionActorV2, core.ResultadoContextoActorRegistradoV2, error)
}

type claveCompetenciaFirmaVecV2 struct{}

type errorActorAUT56Opaco struct{ causa error }

func (e *errorActorAUT56Opaco) Error() string              { return ports.ErrFirmaDocumentoDenegada.Error() }
func (e *errorActorAUT56Opaco) String() string             { return e.Error() }
func (e *errorActorAUT56Opaco) GoString() string           { return e.Error() }
func (e *errorActorAUT56Opaco) Format(s fmt.State, _ rune) { _, _ = s.Write([]byte(e.Error())) }
func (e *errorActorAUT56Opaco) Unwrap() []error {
	return []error{ports.ErrFirmaDocumentoDenegada, e.causa}
}

func falloActorAUT56(causa error) error {
	if causa == nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	return &errorActorAUT56Opaco{causa: causa}
}

// CompetenciaFirmaVecV2 pertenece a una petición POST exacta. AUT56 escribe
// una sola selección y CA25/sesión común la cotejan antes de usar el actor.
type CompetenciaFirmaVecV2 struct {
	peticion  *http.Request
	mu        sync.Mutex
	evidencia ports.EvidenciaCompetenciaFirmante
	presente  bool
	ambigua   bool
	sesion    SesionFirmanteAcreditadaV2
	autoridad AutoridadSesionFirmanteV2
	fallida   bool
}

// PrepararPeticionFirmaVecV2 instala el contenedor antes de ejecutar AUT56.
// La normalización TLS sólo retiene la hoja si coincide exactamente con la
// cadena verificada recibida del servidor TLS.
func PrepararPeticionFirmaVecV2(r *http.Request) (*http.Request, error) {
	if r == nil || r.URL == nil || r.Context().Err() != nil || r.Method != http.MethodPost ||
		r.URL.Path != httpinterno.RutaRegistroFirmaVec || r.URL.RawPath != "" || r.URL.RawQuery != "" ||
		r.URL.ForceQuery || r.URL.EscapedPath() != r.URL.Path {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 1 && len(r.TLS.VerifiedChains) == 1 &&
		len(r.TLS.PeerCertificates) <= len(r.TLS.VerifiedChains[0]) {
		verificada := r.TLS.VerifiedChains[0]
		coincide := true
		for i, cert := range r.TLS.PeerCertificates {
			if cert == nil || verificada[i] == nil || !bytes.Equal(cert.Raw, verificada[i].Raw) {
				coincide = false
				break
			}
		}
		if coincide {
			copia := new(http.Request)
			*copia = *r
			tls := *r.TLS
			tls.PeerCertificates = tls.PeerCertificates[:1:1]
			copia.TLS = &tls
			r = copia
		}
	}
	c := &CompetenciaFirmaVecV2{}
	peticion := r.WithContext(context.WithValue(r.Context(), claveCompetenciaFirmaVecV2{}, c))
	c.peticion = peticion
	return peticion, nil
}

func CompetenciaDesdeContextoFirmaVecV2(ctx context.Context) (*CompetenciaFirmaVecV2, bool) {
	if ctx == nil || ctx.Err() != nil {
		return nil, false
	}
	c, ok := ctx.Value(claveCompetenciaFirmaVecV2{}).(*CompetenciaFirmaVecV2)
	return c, ok && c != nil && c.peticion != nil
}

func (c *CompetenciaFirmaVecV2) guardar(e ports.EvidenciaCompetenciaFirmante) error {
	if c == nil {
		return ports.ErrCompetenciaFirmanteNoAcreditada
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ambigua || (c.presente && c.evidencia != e) {
		c.ambigua = true
		return ports.ErrCompetenciaFirmanteNoDisponible
	}
	c.evidencia, c.presente = e, true
	return nil
}

func (c *CompetenciaFirmaVecV2) Leer() (ports.EvidenciaCompetenciaFirmante, *http.Request, bool) {
	if c == nil {
		return ports.EvidenciaCompetenciaFirmante{}, nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.presente || c.ambigua || c.peticion == nil {
		return ports.EvidenciaCompetenciaFirmante{}, nil, false
	}
	return c.evidencia, c.peticion, true
}

func (c *CompetenciaFirmaVecV2) resolver(ctx context.Context, a AutoridadSesionFirmanteV2) (firmaemisorv2.ContextoActorFirmaV2, error) {
	var cero firmaemisorv2.ContextoActorFirmaV2
	if ctx != nil && ctx.Err() != nil {
		return cero, falloActorAUT56(ctx.Err())
	}
	if c == nil || dependenciaNula(a) || ctx == nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.presente || c.ambigua || c.fallida || c.peticion == nil || c.peticion.Context().Err() != nil {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	if c.sesion != nil && !mismaAutoridadSesionFirmaVecV2(c.autoridad, a) {
		c.fallida = true
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	var vinculo core.VinculoAutenticacionActorV2
	var resultado core.ResultadoContextoActorRegistradoV2
	var err error
	if c.sesion == nil {
		e := c.evidencia
		seleccion := ports.SeleccionFirmanteV2{PersonaRef: e.FirmantePrincipalRef,
			CuentaRef: e.CuentaFirmanteRef, PerfilActivoRef: e.PerfilActivoFirmanteRef, RolID: e.RolIDFirmante,
			VinculoCertificado: ports.ReferenciaVersionadaFirmanteV2{Referencia: e.VinculoCredencialFirmanteRef,
				Version: e.VinculoCredencialFirmanteRevision, HuellaSHA256: e.VinculoCredencialFirmanteHuella}}
		c.sesion, err = a.AcreditarSesionFirmaVecV2(c.peticion, seleccion, e.FirmantePrincipalRef, e.Solicitud.CertificadoHuella)
		if err == nil && !dependenciaNula(c.sesion) {
			c.autoridad = a
			vinculo, resultado, err = c.sesion.AbrirSesionFirmaVecV2(ctx, c.peticion)
		}
	} else {
		vinculo, resultado, err = c.sesion.RevalidarSesionFirmaVecV2(ctx, c.peticion)
	}
	if err != nil || dependenciaNula(c.sesion) {
		c.fallida = true
		return cero, falloActorAUT56(err)
	}
	return firmaemisorv2.ContextoActorFirmaV2{Vinculo: vinculo, Resultado: resultado,
		CertificadoCanalSHA256: c.evidencia.Solicitud.CertificadoHuella}, nil
}

// La cápsula acreditada nunca puede pasar a otra fuente nominal aun cuando
// ambas reciban el mismo context.Context. La autoridad de sesión es estatal.
func mismaAutoridadSesionFirmaVecV2(primera, segunda AutoridadSesionFirmanteV2) bool {
	if dependenciaNula(primera) || dependenciaNula(segunda) {
		return false
	}
	a, b := reflect.ValueOf(primera), reflect.ValueOf(segunda)
	return a.Kind() == reflect.Pointer && b.Kind() == reflect.Pointer &&
		a.Type() == b.Type() && a.Pointer() == b.Pointer()
}

type FuenteNominalFirmaVecV2 struct{ autoridad AutoridadSesionFirmanteV2 }

var _ firmaemisorv2.FuenteContextoActorFirmaV2 = (*FuenteNominalFirmaVecV2)(nil)
var _ consultafirmasv2.FuenteContexto = (*FuenteNominalFirmaVecV2)(nil)

func NuevaFuenteNominalFirmaVecV2(a AutoridadSesionFirmanteV2) (*FuenteNominalFirmaVecV2, error) {
	if dependenciaNula(a) || reflect.ValueOf(a).Kind() != reflect.Pointer {
		return nil, ports.ErrFirmaDocumentoDenegada
	}
	return &FuenteNominalFirmaVecV2{autoridad: a}, nil
}

func (f *FuenteNominalFirmaVecV2) RevalidarContextoActorFirmaV2(ctx context.Context) (firmaemisorv2.ContextoActorFirmaV2, error) {
	var cero firmaemisorv2.ContextoActorFirmaV2
	if ctx != nil && ctx.Err() != nil {
		return cero, falloActorAUT56(ctx.Err())
	}
	if f == nil || dependenciaNula(f.autoridad) {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	c, ok := CompetenciaDesdeContextoFirmaVecV2(ctx)
	if !ok {
		return cero, ports.ErrFirmaDocumentoDenegada
	}
	return c.resolver(ctx, f.autoridad)
}

func (f *FuenteNominalFirmaVecV2) ResolverContextoConsultaFirmasR5V2(ctx context.Context) (consultafirmasv2.Contexto, error) {
	actor, err := f.RevalidarContextoActorFirmaV2(ctx)
	if err != nil {
		return consultafirmasv2.Contexto{}, err
	}
	c, ok := CompetenciaDesdeContextoFirmaVecV2(ctx)
	if !ok {
		return consultafirmasv2.Contexto{}, ports.ErrFirmaDocumentoDenegada
	}
	e, _, presente := c.Leer()
	if !presente || !domain.ReferenciaOpacaValida(e.Solicitud.OrganizacionRef) {
		return consultafirmasv2.Contexto{}, ports.ErrFirmaDocumentoDenegada
	}
	return consultafirmasv2.Contexto{OrganizacionRef: e.Solicitud.OrganizacionRef,
		FirmantePrincipalCandidatoRef: actor.Resultado.Contexto.PersonaRef}, nil
}

type FuenteCompetenciaFirmaVecV2 struct {
	delegada   ports.FuenteCompetenciaFirmante
	rolExterno string
}

var _ ports.FuenteCompetenciaFirmante = (*FuenteCompetenciaFirmaVecV2)(nil)

func NuevaFuenteCompetenciaFirmaVecV2(delegada ports.FuenteCompetenciaFirmante, rolExterno string) (*FuenteCompetenciaFirmaVecV2, error) {
	if dependenciaNula(delegada) || rolExterno == "" {
		return nil, ports.ErrCompetenciaFirmanteNoDisponible
	}
	return &FuenteCompetenciaFirmaVecV2{delegada: delegada, rolExterno: rolExterno}, nil
}

func (f *FuenteCompetenciaFirmaVecV2) AcreditarCompetenciaFirmante(ctx context.Context, q ports.SolicitudCompetenciaFirmante) (ports.EvidenciaCompetenciaFirmante, error) {
	var cero ports.EvidenciaCompetenciaFirmante
	if f == nil || dependenciaNula(f.delegada) || ctx == nil || ctx.Err() != nil {
		return cero, ports.ErrCompetenciaFirmanteNoDisponible
	}
	c, vec := CompetenciaDesdeContextoFirmaVecV2(ctx)
	if vec && (!domain.HuellaSHA256FirmaValida(q.CertificadoHuella) || q.FirmanteRef != "ref:"+q.CertificadoHuella) {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	e, err := f.delegada.AcreditarCompetenciaFirmante(ctx, q)
	if err != nil || !vec {
		return e, err
	}
	if e.Solicitud != q || !e.Vigente || !domain.ReferenciaOpacaValida(e.FirmantePrincipalRef) ||
		!domain.ReferenciaOpacaValida(e.CuentaFirmanteRef) || !domain.ReferenciaOpacaValida(e.PerfilActivoFirmanteRef) ||
		!domain.ReferenciaOpacaValida(e.VinculoCredencialFirmanteRef) ||
		e.VinculoCredencialFirmanteRevision == 0 || !domain.HuellaSHA256FirmaValida(e.VinculoCredencialFirmanteHuella) ||
		e.RolIDFirmante == "" || e.RolIDFirmante == f.rolExterno {
		return cero, ports.ErrCompetenciaFirmanteNoAcreditada
	}
	if err := c.guardar(e); err != nil {
		return cero, err
	}
	return e, nil
}
