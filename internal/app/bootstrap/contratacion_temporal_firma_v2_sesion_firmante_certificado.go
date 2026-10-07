package bootstrap

import (
	"context"
	"crypto/subtle"
	"fmt"
	"time"

	"vec-diputacion-granada/internal/app/composicion/identidadordinaria"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// La raíz proporciona una fuente con el mismo ServicioIdentidad que vinculó
// la cápsula al contexto de la petición. CT sólo coteja la selección AUT56.
type fuenteCertificadoFirmaVecV2 struct {
	comun *identidadordinaria.FuenteCertificadoTemporal
	reloj vp.Reloj
}

var _ ports.FuenteSesionFirmanteV2 = (*fuenteCertificadoFirmaVecV2)(nil)

func nuevaAutoridadSesionFirmanteV2Certificado(
	comun *identidadordinaria.FuenteCertificadoTemporal, reloj vp.Reloj,
) (*autoridadSesionFirmanteV2, error) {
	if comun == nil || dependenciaEsNulaContratacionTemporalDesarrollo(reloj) {
		return nil, errSesionFirmanteV2NoDisponible
	}
	a, err := nuevaAutoridadSesionFirmanteV2ConFuente(&fuenteCertificadoFirmaVecV2{comun: comun, reloj: reloj}, reloj, nil)
	if err != nil {
		return nil, err
	}
	a.garantia = core.AuthAssuranceSubstantial
	a.exigirCanalTLS = true
	return a, nil
}

func (f *fuenteCertificadoFirmaVecV2) AbrirSesionFirmanteV2(ctx context.Context,
	q ports.SolicitudSesionFirmanteV2,
) (ports.SesionFirmanteV2, error) {
	if f == nil || f.comun == nil || dependenciaEsNulaContratacionTemporalDesarrollo(f.reloj) || ctx == nil {
		return nil, errSesionFirmanteV2Denegada
	}
	if err := ctx.Err(); err != nil {
		return nil, falloSesionFirmanteV2(err)
	}
	if !solicitudCertificadoFirmaVecV2Valida(q, f.reloj.Ahora()) {
		return nil, errSesionFirmanteV2Denegada
	}
	registrada, err := f.comun.Abrir(ctx)
	if err != nil {
		return nil, falloSesionFirmanteV2(err)
	}
	esperados, err := registrada.Esperados()
	if err != nil {
		return nil, errSesionFirmanteV2Denegada
	}
	s := &sesionCertificadoFirmaVecV2{fuente: f, solicitud: q, esperados: esperados}
	if _, err := s.cotejar(registrada, f.reloj.Ahora()); err != nil {
		return nil, err
	}
	return s, nil
}

type sesionCertificadoFirmaVecV2 struct {
	fuente    *fuenteCertificadoFirmaVecV2
	solicitud ports.SolicitudSesionFirmanteV2
	esperados identidadordinaria.EsperadosSesion
}

func (s *sesionCertificadoFirmaVecV2) RevalidarSesionFirmanteV2(ctx context.Context) (ports.EvidenciaSesionFirmanteV2, error) {
	if s == nil || s.fuente == nil || s.fuente.comun == nil || ctx == nil {
		return ports.EvidenciaSesionFirmanteV2{}, errSesionFirmanteV2Denegada
	}
	if err := ctx.Err(); err != nil {
		return ports.EvidenciaSesionFirmanteV2{}, falloSesionFirmanteV2(err)
	}
	ahora := s.fuente.reloj.Ahora()
	if !solicitudCertificadoFirmaVecV2Valida(s.solicitud, ahora) {
		return ports.EvidenciaSesionFirmanteV2{}, errSesionFirmanteV2Denegada
	}
	registrada, err := s.fuente.comun.Revalidar(ctx, s.esperados)
	if err != nil {
		return ports.EvidenciaSesionFirmanteV2{}, falloSesionFirmanteV2(err)
	}
	return s.cotejar(registrada, s.fuente.reloj.Ahora())
}

func (s *sesionCertificadoFirmaVecV2) cotejar(registrada identidadordinaria.SesionRegistrada,
	ahora time.Time,
) (ports.EvidenciaSesionFirmanteV2, error) {
	var cero ports.EvidenciaSesionFirmanteV2
	if s == nil || s.fuente == nil || !solicitudCertificadoFirmaVecV2Valida(s.solicitud, ahora) ||
		registrada.CanalSHA256() == "" ||
		subtle.ConstantTimeCompare([]byte(registrada.CanalSHA256()), []byte(s.solicitud.CanalTLSVinculadoSHA256)) != 1 ||
		registrada.EmpleadoRef() == "" || !ahora.Before(registrada.VigenteHasta()) {
		return cero, errSesionFirmanteV2Denegada
	}
	vinculo, resultado, snapshot, err := registrada.Contexto()
	if err != nil || vinculo.ValidarPara(resultado) != nil || snapshot.Validar() != nil ||
		!vinculo.VigenteEn(ahora, resultado) ||
		resultado.Contexto.PersonaRef != s.solicitud.PersonaEsperadaRef ||
		resultado.Contexto.PerfilActivoRef != s.solicitud.PerfilEsperadoRef ||
		snapshot.VersionRol.RolID != s.solicitud.RolEsperadoID ||
		snapshot.AsignacionPerfil.PerfilActivoRef != s.solicitud.PerfilEsperadoRef ||
		!resultado.Contexto.AlcanceProyecciones().IncluyeEmpleado() {
		return cero, errSesionFirmanteV2Denegada
	}
	datos, err := vinculo.Datos()
	if err != nil || datos.PrincipalID != s.solicitud.PersonaEsperadaRef ||
		datos.CuentaRef != s.solicitud.CuentaEsperadaRef ||
		datos.PerfilActivoRef != s.solicitud.PerfilEsperadoRef ||
		datos.MetodoObservado != core.AuthMethodCertificate ||
		datos.GarantiaObservada != core.AuthAssuranceSubstantial ||
		datos.CuentaPrivilegiada ||
		datos.Superficie != core.SuperficieAutenticacionInternaCorporativaV1 {
		return cero, errSesionFirmanteV2Denegada
	}
	hasta := registrada.VigenteHasta()
	if s.solicitud.CertificadoTLSValidoHasta.Before(hasta) {
		hasta = s.solicitud.CertificadoTLSValidoHasta
	}
	if !ahora.Before(hasta) {
		return cero, errSesionFirmanteV2Denegada
	}
	return ports.EvidenciaSesionFirmanteV2{Vinculo: vinculo, Resultado: resultado,
		CertificadoCanalSHA256: s.solicitud.CertificadoCanalSHA256,
		CertificadoValidoHasta: hasta}, nil
}

func solicitudCertificadoFirmaVecV2Valida(q ports.SolicitudSesionFirmanteV2, ahora time.Time) bool {
	return huellaSHA256ValidaContratacionTemporalDesarrollo(q.CertificadoCanalSHA256) &&
		huellaSHA256ValidaContratacionTemporalDesarrollo(q.CanalTLSVinculadoSHA256) &&
		q.PersonaEsperadaRef != "" && q.CuentaEsperadaRef != "" &&
		q.PerfilEsperadoRef != "" && q.RolEsperadoID != "" &&
		!q.CertificadoVerificadoEn.IsZero() && !q.CertificadoVerificadoEn.After(ahora) &&
		!q.CertificadoTLSValidoHasta.IsZero() && ahora.Before(q.CertificadoTLSValidoHasta)
}

type errorSesionFirmanteV2Opaco struct{ causa error }

func (e *errorSesionFirmanteV2Opaco) Error() string    { return errSesionFirmanteV2Denegada.Error() }
func (e *errorSesionFirmanteV2Opaco) String() string   { return e.Error() }
func (e *errorSesionFirmanteV2Opaco) GoString() string { return e.Error() }
func (e *errorSesionFirmanteV2Opaco) Format(estado fmt.State, _ rune) {
	_, _ = estado.Write([]byte(e.Error()))
}
func (e *errorSesionFirmanteV2Opaco) Unwrap() []error {
	return []error{errSesionFirmanteV2Denegada, e.causa}
}

func falloSesionFirmanteV2(causa error) error {
	if causa == nil {
		return errSesionFirmanteV2Denegada
	}
	return &errorSesionFirmanteV2Opaco{causa: causa}
}
