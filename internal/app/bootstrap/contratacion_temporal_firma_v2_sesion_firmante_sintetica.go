package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/composicion/interna"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Sólo el adaptador opt-in de desarrollo interpreta este fichero. La fuente
// corporativa inyecta su evidencia ya autenticada y registrada por otro canal.
type configuracionSesionFirmanteV2 struct {
	Version                     int                       `json:"version"`
	Autoridad                   string                    `json:"autoridad"`
	PoliticaAutenticacionRef    string                    `json:"politica_autenticacion_ref"`
	PoliticaAutenticacionSHA256 string                    `json:"politica_autenticacion_sha256"`
	RetiradaEn                  string                    `json:"retirada_en"`
	CRLPath                     string                    `json:"crl_path"`
	Firmantes                   []seleccionNominalFirmaCT `json:"firmantes"`
}

func (configuracionSesionFirmanteV2) String() string   { return "[CONFIGURACION PRIVADA FIRMA VEC]" }
func (configuracionSesionFirmanteV2) GoString() string { return "[CONFIGURACION PRIVADA FIRMA VEC]" }

func decodificarConfiguracionSesionFirmanteV2(contenido []byte) (configuracionSesionFirmanteV2, time.Time, error) {
	var c configuracionSesionFirmanteV2
	if validarClavesJSONUnicas(contenido) != nil {
		return c, time.Time{}, errSesionFirmanteV2NoDisponible
	}
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	var extra any
	if dec.Decode(&c) != nil || !errors.Is(dec.Decode(&extra), io.EOF) || c.Version != 1 || c.Autoridad != AutoridadNoAutoritativa ||
		c.PoliticaAutenticacionRef == "" || !huellaSHA256ValidaContratacionTemporalDesarrollo(c.PoliticaAutenticacionSHA256) {
		return configuracionSesionFirmanteV2{}, time.Time{}, errSesionFirmanteV2NoDisponible
	}
	retirada, err := time.Parse(time.RFC3339, c.RetiradaEn)
	if err != nil || retirada.Location() != time.UTC || retirada.Format(time.RFC3339) != c.RetiradaEn {
		return configuracionSesionFirmanteV2{}, time.Time{}, errSesionFirmanteV2NoDisponible
	}
	return c, retirada, nil
}

type fuenteSinteticaSesionFirmanteV2 struct {
	resolvedor                  *resolvedorIdentidadDesarrollo
	registro                    httpseguridad.RegistroSesiones
	revalidador                 core.RevalidadorAutenticacionActorV1
	contextos                   core.ResolutorContextoActorRegistradoV2
	reloj                       vp.Reloj
	retirada                    time.Time
	instancia                   string
	politicaRef, politicaHuella string
	crlPath                     string
	cuentasID                   map[string]string
	selecciones                 map[string]seleccionNominalFirmaCT
	acreditarCA25               func(context.Context, string, seleccionNominalFirmaCT) error
}

var _ ports.FuenteSesionFirmanteV2 = (*fuenteSinteticaSesionFirmanteV2)(nil)

// La garantía sustancial pertenece exclusivamente a esta frontera. El
// predicado general de RRHH continúa exigiendo garantía alta.
func principalCertificadoFirmaVecDesarrolloValido(p core.Principal) bool {
	return p.Validate() == nil && p.AuthMethod == core.AuthMethodCertificate &&
		p.AuthAssurance == core.AuthAssuranceSubstantial && len(p.Permissions) == 0 &&
		p.Attributes["autoridad"] == AutoridadNoAutoritativa &&
		p.Attributes["perfil_ejecucion"] == config.ExecutionProfileDevelopment &&
		huellaSHA256ValidaContratacionTemporalDesarrollo(p.Attributes["certificate_sha256"])
}

func nuevaAutoridadSesionFirmanteV2ConPuertos(resolvedor *resolvedorIdentidadDesarrollo,
	registro httpseguridad.RegistroSesiones, revalidador core.RevalidadorAutenticacionActorV1,
	contextos core.ResolutorContextoActorRegistradoV2, reloj vp.Reloj, retirada time.Time, instancia string, politicaRef, politicaHuella, crlPath string,
) (*autoridadSesionFirmanteV2, error) {
	if resolvedor == nil || dependenciaEsNulaContratacionTemporalDesarrollo(registro) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(revalidador) || dependenciaEsNulaContratacionTemporalDesarrollo(contextos) ||
		dependenciaEsNulaContratacionTemporalDesarrollo(reloj) || retirada.IsZero() || !retirada.After(reloj.Ahora()) ||
		!huellaSHA256ValidaContratacionTemporalDesarrollo(instancia) || politicaRef == "" ||
		!huellaSHA256ValidaContratacionTemporalDesarrollo(politicaHuella) || !filepath.IsAbs(crlPath) || filepath.Clean(crlPath) != crlPath {
		return nil, errSesionFirmanteV2NoDisponible
	}
	f := &fuenteSinteticaSesionFirmanteV2{resolvedor: resolvedor, registro: registro, revalidador: revalidador,
		contextos: contextos, reloj: reloj, retirada: retirada, instancia: instancia,
		politicaRef: politicaRef, politicaHuella: politicaHuella, crlPath: crlPath}
	a, err := nuevaAutoridadSesionFirmanteV2ConFuente(f, reloj, f.preacreditar)
	if err != nil {
		return nil, err
	}
	a.garantia = core.AuthAssuranceSubstantial
	return a, nil
}

func (f *fuenteSinteticaSesionFirmanteV2) preacreditar(r *http.Request) error {
	if f == nil || r == nil || f.reloj == nil || !f.reloj.Ahora().Before(f.retirada) {
		return errSesionFirmanteV2Denegada
	}
	if _, ok := f.principalSeleccionado(r); !ok {
		return errSesionFirmanteV2Denegada
	}
	return nil
}

// La selección sólo identifica una candidata de canal. CA25/IS/F1 y la
// asignación publicada vuelven a contrastarla antes de crear material V3.
func (f *fuenteSinteticaSesionFirmanteV2) principalSeleccionado(r *http.Request) (core.Principal, bool) {
	if f == nil || r == nil || f.reloj == nil || !f.reloj.Ahora().Before(f.retirada) ||
		r.TLS == nil || len(r.TLS.VerifiedChains) != 1 || len(r.TLS.VerifiedChains[0]) < 2 ||
		!direccionRemotaLoopback(r.RemoteAddr) || cabeceraIdentidadAmbientalPresente(r.Header) ||
		len(r.Header.Values("Cookie")) != 0 || len(r.Header.Values("Authorization")) != 0 {
		return core.Principal{}, false
	}
	_, huella, ok := certificadoSesionFirmanteV2(r, f.reloj.Ahora())
	s, presente := f.selecciones[huella]
	if !ok || !presente || f.acreditarCA25 == nil || interna.ComprobarCRLFirmaR5(f.crlPath,
		r.TLS.VerifiedChains[0][0], r.TLS.VerifiedChains[0][1], f.reloj.Ahora()) != nil {
		return core.Principal{}, false
	}
	if f.acreditarCA25(r.Context(), huella, s) != nil {
		return core.Principal{}, false
	}
	p := core.Principal{ID: s.PersonaRef, AuthMethod: core.AuthMethodCertificate,
		AuthAssurance: core.AuthAssuranceSubstantial,
		Attributes: map[string]string{"autoridad": AutoridadNoAutoritativa,
			"perfil_ejecucion":   config.ExecutionProfileDevelopment,
			"certificate_sha256": huella}}
	return p, principalCertificadoFirmaVecDesarrolloValido(p)
}

func (f *fuenteSinteticaSesionFirmanteV2) AbrirSesionFirmanteV2(ctx context.Context,
	q ports.SolicitudSesionFirmanteV2,
) (ports.SesionFirmanteV2, error) {
	if f == nil || ctx == nil || ctx.Err() != nil || f.resolvedor == nil || f.reloj == nil ||
		!f.reloj.Ahora().Before(f.retirada) || !huellaSHA256ValidaContratacionTemporalDesarrollo(q.CertificadoCanalSHA256) ||
		q.CuentaEsperadaRef == "" || q.PerfilEsperadoRef == "" || q.PersonaEsperadaRef == "" {
		return nil, errSesionFirmanteV2Denegada
	}
	if existente, presente, err := f.sesionNominalExistente(ctx, q); presente {
		return existente, err
	}
	seleccion, ok := f.selecciones[q.CertificadoCanalSHA256]
	if !ok || seleccion.PersonaRef != q.PersonaEsperadaRef || seleccion.CuentaRef != q.CuentaEsperadaRef ||
		(q.PerfilEsperadoRef != seleccion.PerfilRef && q.PerfilEsperadoRef != seleccion.PerfilConsultaCTRef &&
			q.PerfilEsperadoRef != seleccion.PerfilDocumentosRef) {
		return nil, errSesionFirmanteV2Denegada
	}
	if f.acreditarCA25 == nil || f.acreditarCA25(ctx, q.CertificadoCanalSHA256, seleccion) != nil {
		return nil, errSesionFirmanteV2Denegada
	}
	ahora := f.reloj.Ahora()
	if q.CertificadoVerificadoEn.IsZero() || q.CertificadoVerificadoEn.After(ahora) ||
		ahora.Sub(q.CertificadoVerificadoEn) > 2*time.Minute {
		return nil, errSesionFirmanteV2Denegada
	}
	hasta := ahora.Add(2 * time.Minute)
	if q.CertificadoTLSValidoHasta.Before(hasta) {
		hasta = q.CertificadoTLSValidoHasta
	}
	if f.retirada.Before(hasta) {
		hasta = f.retirada
	}
	if !hasta.After(ahora) {
		return nil, errSesionFirmanteV2Denegada
	}
	asercion, err := nonceRutasDietas()
	if err != nil {
		return nil, errSesionFirmanteV2NoDisponible
	}
	sesion, err := nonceRutasDietas()
	if err != nil {
		return nil, errSesionFirmanteV2NoDisponible
	}
	cuentaID := "desarrollo:" + q.CuentaEsperadaRef
	if f.cuentasID[q.CertificadoCanalSHA256] != "" {
		cuentaID = f.cuentasID[q.CertificadoCanalSHA256]
	}
	alta := httpseguridad.AltaSesionAtomica{AsercionID: asercion, SesionID: sesion,
		SujetoID: seleccion.PersonaRef, CuentaID: cuentaID,
		Superficie: httpseguridad.SuperficieInternaCorporativa, EspacioIdentidad: espacioIdentidadSesionDesarrollo,
		MetodoObservado: core.AuthMethodCertificate, GarantiaObservada: core.AuthAssuranceSubstantial,
		AutenticacionVerificadaEn: q.CertificadoVerificadoEn, SesionEmitidaEn: ahora, AsercionExpiraEn: hasta,
		PoliticaGarantiaRef:          f.politicaRef,
		PoliticaGarantiaHuellaSHA256: f.politicaHuella,
		AutenticacionHuellaSHA256: huellaRutasDietas(f.instancia + "|" + asercion + "|" + sesion + "|" + q.CertificadoCanalSHA256 + "|" +
			q.CuentaEsperadaRef + "|" + q.PerfilEsperadoRef + "|" + q.PersonaEsperadaRef + "|" + httpinterno.RutaRegistroFirmaVec),
	}
	confirmacion, err := f.registro.ConsumirAsercionYRegistrar(ctx, alta)
	if err != nil || confirmacion.ValidarPara(alta) != nil || confirmacion.CuentaRef != q.CuentaEsperadaRef {
		return nil, errSesionFirmanteV2NoDisponible
	}
	return &sesionSinteticaFirmanteV2{fuente: f, solicitud: q, alta: alta, confirmacion: confirmacion}, nil
}

type sesionSinteticaFirmanteV2 struct {
	fuente       *fuenteSinteticaSesionFirmanteV2
	solicitud    ports.SolicitudSesionFirmanteV2
	alta         httpseguridad.AltaSesionAtomica
	confirmacion httpseguridad.ConfirmacionAltaSesion
}

func (s *sesionSinteticaFirmanteV2) RevalidarSesionFirmanteV2(ctx context.Context) (ports.EvidenciaSesionFirmanteV2, error) {
	var cero ports.EvidenciaSesionFirmanteV2
	if s == nil || s.fuente == nil || ctx == nil || ctx.Err() != nil ||
		!s.fuente.reloj.Ahora().Before(s.fuente.retirada) {
		return cero, errSesionFirmanteV2Denegada
	}
	f := s.fuente
	revalidador := revalidadorSesionConsultaRRHHDesarrollo{delegado: f.revalidador,
		alta: s.alta, confirmacion: s.confirmacion, reloj: f.reloj, superficie: httpseguridad.SuperficieInternaCorporativa}
	vinculo, resultado, err := core.CrearVinculoAutenticacionActorV2ConResultado(ctx, revalidador,
		core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: s.confirmacion.AutenticacionRef,
			SesionRef: s.confirmacion.SesionRef}, f.contextos,
		core.SolicitudContextoActor{Cuenta: core.CuentaAutenticadaContextoActor{
			CuentaRef: s.solicitud.CuentaEsperadaRef, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceSubstantial},
			PerfilActivoRef: s.solicitud.PerfilEsperadoRef}, f.reloj)
	if err != nil {
		return cero, errSesionFirmanteV2Denegada
	}
	hasta := s.solicitud.CertificadoTLSValidoHasta
	if f.retirada.Before(hasta) {
		hasta = f.retirada
	}
	return ports.EvidenciaSesionFirmanteV2{Vinculo: vinculo, Resultado: resultado,
		CertificadoCanalSHA256: s.solicitud.CertificadoCanalSHA256, CertificadoValidoHasta: hasta}, nil
}
