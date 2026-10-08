package firmavec

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"reflect"
	"time"

	"vec-diputacion-granada/internal/app/composicion/identidadordinaria"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

var ErrSesionCertificadoDenegada = errors.New("contratacion temporal: sesion certificado denegada")

type errorSesionOpaco struct{ causa error }

func (e *errorSesionOpaco) Error() string              { return ErrSesionCertificadoDenegada.Error() }
func (e *errorSesionOpaco) String() string             { return e.Error() }
func (e *errorSesionOpaco) GoString() string           { return e.Error() }
func (e *errorSesionOpaco) Format(s fmt.State, _ rune) { _, _ = s.Write([]byte(e.Error())) }
func (e *errorSesionOpaco) Unwrap() []error            { return []error{ErrSesionCertificadoDenegada, e.causa} }

func falloSesion(causa error) error {
	if causa == nil {
		return ErrSesionCertificadoDenegada
	}
	return &errorSesionOpaco{causa: causa}
}

// FuenteCertificadoTemporal usa sólo la sesión común ya ligada a la petición.
// AUT56 y CA25 aportan cotejos esperados; nunca eligen identidad o perfil.
type FuenteCertificadoTemporal struct {
	comun *identidadordinaria.FuenteCertificadoTemporal
	reloj vp.Reloj
}

var _ ports.FuenteSesionFirmanteV2 = (*FuenteCertificadoTemporal)(nil)

func NuevaFuenteCertificadoTemporal(comun *identidadordinaria.FuenteCertificadoTemporal, reloj vp.Reloj) (*FuenteCertificadoTemporal, error) {
	if comun == nil || dependenciaNula(reloj) {
		return nil, ErrSesionCertificadoDenegada
	}
	return &FuenteCertificadoTemporal{comun: comun, reloj: reloj}, nil
}

func (f *FuenteCertificadoTemporal) AbrirSesionFirmanteV2(ctx context.Context, q ports.SolicitudSesionFirmanteV2) (ports.SesionFirmanteV2, error) {
	if f == nil || f.comun == nil || dependenciaNula(f.reloj) || ctx == nil {
		return nil, ErrSesionCertificadoDenegada
	}
	if err := ctx.Err(); err != nil {
		return nil, falloSesion(err)
	}
	if !SolicitudValida(q, f.reloj.Ahora()) {
		return nil, ErrSesionCertificadoDenegada
	}
	registrada, err := f.comun.Abrir(ctx)
	if err != nil {
		return nil, falloSesion(err)
	}
	esperados, err := registrada.Esperados()
	if err != nil {
		return nil, ErrSesionCertificadoDenegada
	}
	s := &sesionCertificado{fuente: f, solicitud: q, esperados: esperados}
	if _, err := s.cotejar(registrada, f.reloj.Ahora()); err != nil {
		return nil, err
	}
	return s, nil
}

type sesionCertificado struct {
	fuente    *FuenteCertificadoTemporal
	solicitud ports.SolicitudSesionFirmanteV2
	esperados identidadordinaria.EsperadosSesion
}

func (s *sesionCertificado) RevalidarSesionFirmanteV2(ctx context.Context) (ports.EvidenciaSesionFirmanteV2, error) {
	if s == nil || s.fuente == nil || s.fuente.comun == nil || ctx == nil {
		return ports.EvidenciaSesionFirmanteV2{}, ErrSesionCertificadoDenegada
	}
	if err := ctx.Err(); err != nil {
		return ports.EvidenciaSesionFirmanteV2{}, falloSesion(err)
	}
	if !SolicitudValida(s.solicitud, s.fuente.reloj.Ahora()) {
		return ports.EvidenciaSesionFirmanteV2{}, ErrSesionCertificadoDenegada
	}
	registrada, err := s.fuente.comun.Revalidar(ctx, s.esperados)
	if err != nil {
		return ports.EvidenciaSesionFirmanteV2{}, falloSesion(err)
	}
	return s.cotejar(registrada, s.fuente.reloj.Ahora())
}

func (s *sesionCertificado) cotejar(registrada identidadordinaria.SesionRegistrada, ahora time.Time) (ports.EvidenciaSesionFirmanteV2, error) {
	var cero ports.EvidenciaSesionFirmanteV2
	if s == nil || s.fuente == nil || !SolicitudValida(s.solicitud, ahora) ||
		registrada.CanalSHA256() == "" || subtle.ConstantTimeCompare([]byte(registrada.CanalSHA256()), []byte(s.solicitud.CanalTLSVinculadoSHA256)) != 1 ||
		registrada.EmpleadoRef() == "" || !ahora.Before(registrada.VigenteHasta()) {
		return cero, ErrSesionCertificadoDenegada
	}
	vinculo, resultado, snapshot, err := registrada.Contexto()
	if err != nil || vinculo.ValidarPara(resultado) != nil || snapshot.Validar() != nil || !vinculo.VigenteEn(ahora, resultado) ||
		resultado.Contexto.PersonaRef != s.solicitud.PersonaEsperadaRef || resultado.Contexto.PerfilActivoRef != s.solicitud.PerfilEsperadoRef ||
		snapshot.VersionRol.RolID != s.solicitud.RolEsperadoID || snapshot.AsignacionPerfil.PerfilActivoRef != s.solicitud.PerfilEsperadoRef ||
		!resultado.Contexto.AlcanceProyecciones().IncluyeEmpleado() {
		return cero, ErrSesionCertificadoDenegada
	}
	datos, err := vinculo.Datos()
	if err != nil || datos.PrincipalID != s.solicitud.PersonaEsperadaRef || datos.CuentaRef != s.solicitud.CuentaEsperadaRef ||
		datos.PerfilActivoRef != s.solicitud.PerfilEsperadoRef || datos.MetodoObservado != core.AuthMethodCertificate ||
		datos.GarantiaObservada != core.AuthAssuranceSubstantial || datos.CuentaPrivilegiada ||
		datos.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 {
		return cero, ErrSesionCertificadoDenegada
	}
	hasta := registrada.VigenteHasta()
	if s.solicitud.CertificadoTLSValidoHasta.Before(hasta) {
		hasta = s.solicitud.CertificadoTLSValidoHasta
	}
	if !ahora.Before(hasta) {
		return cero, ErrSesionCertificadoDenegada
	}
	return ports.EvidenciaSesionFirmanteV2{Vinculo: vinculo, Resultado: resultado,
		CertificadoCanalSHA256: s.solicitud.CertificadoCanalSHA256, CertificadoValidoHasta: hasta}, nil
}

func SolicitudValida(q ports.SolicitudSesionFirmanteV2, ahora time.Time) bool {
	return domain.HuellaSHA256FirmaValida(q.CertificadoCanalSHA256) && domain.HuellaSHA256FirmaValida(q.CanalTLSVinculadoSHA256) &&
		q.PersonaEsperadaRef != "" && q.CuentaEsperadaRef != "" && q.PerfilEsperadoRef != "" && q.RolEsperadoID != "" &&
		!q.CertificadoVerificadoEn.IsZero() && !q.CertificadoVerificadoEn.After(ahora) &&
		!q.CertificadoTLSValidoHasta.IsZero() && ahora.Before(q.CertificadoTLSValidoHasta)
}

func dependenciaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	default:
		return false
	}
}
