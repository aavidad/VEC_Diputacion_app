package confianzadocumental

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// suiteAtestacionAutorizacionPDPCOSEEdDSAV1 identifica sin ambiguedad el perfil
// sombra que firma la Sig_structure de COSE Sign1 con external_aad. No equivale
// a VEC-AD-ED25519-1, que el estudio previo definio como firma separada sobre
// VEC-AD-1, ni constituye por si solo una aprobacion productiva de Seguridad.
// Que el verificador COSE generico admita ES256 tampoco aprueba una suite PDP
// ES256.
const suiteAtestacionAutorizacionPDPCOSEEdDSAV1 = "VEC-AD-COSE-EDDSA-1"

// comprobacionAtestacionAutorizacionPDPV4 solo puede resultar util si contiene
// una PruebaCOSESign1DocumentalVerificada valida. El literal cero y cualquier
// mezcla entre solicitud, sobre, cabecera o vinculo se deniegan.
type comprobacionAtestacionAutorizacionPDPV4 struct {
	prueba    PruebaCOSESign1DocumentalVerificada
	cabecera  domain.CabeceraAtestacionAutorizacionV1
	solicitud SolicitudVerificacionCOSESign1
	sobre     ports.SobreCriptograficoDocumentalCrudoV4
}

// EmitirAutoridadInternaEjecucionDocumentalV4 es la unica composicion publica
// capaz de devolver la autoridad V4. La solicitud vinculada sigue sin ser una
// credencial: el metodo reconstruye desde ella la decision completa, produce
// exactamente VEC-AD-1 y exige una firma COSE de una raiz PDP fijada.
//
// cabecera es seleccion de configuracion, no autoridad. Suite, kid y audiencia
// se cotejan de nuevo contra la raiz usada realmente por COSE. El llamador no
// aporta ningun instante. Una lectura interna preliminar permite reconstruir
// el mensaje; la lectura realizada por VerificarCOSESign1 fija el instante de
// emision y provoca una revalidacion completa posterior. Un retroceso entre
// ambas lecturas se deniega.
func (s *Servicio) EmitirAutoridadInternaEjecucionDocumentalV4(
	ctx context.Context,
	vinculo ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	sobre ports.SobreCriptograficoDocumentalCrudoV4,
) (AutoridadInternaEjecucionDocumentalV4, error) {
	preparadaEn, err := s.capturarInstanteAtestacionPDP(ctx)
	if err != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, denegarAutoridadAtestacionPDP(err)
	}
	if vinculo.ValidarEn(preparadaEn) != nil ||
		s.validarCabeceraAtestacionPDP(cabecera) != nil ||
		sobre.ValidarSintaxis() != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, denegarAutoridadAtestacionPDP(nil)
	}

	decision, err := decisionDesdeSolicitudVinculadaAtestacionPDP(vinculo, preparadaEn)
	if err != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, denegarAutoridadAtestacionPDP(nil)
	}
	mensaje, err := domain.SerializarMensajeAtestacionAutorizacionV1(cabecera, decision)
	if err != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, denegarAutoridadAtestacionPDP(nil)
	}
	solicitudCOSE, err := NuevaSolicitudVerificacionCOSESign1(
		mensaje,
		AudienciaCOSEAtestacionAutorizacionPDP,
	)
	if err != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, denegarAutoridadAtestacionPDP(nil)
	}

	prueba, err := s.VerificarCOSESign1(ctx, solicitudCOSE, sobre)
	if err != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, denegarAutoridadAtestacionPDP(err)
	}
	emitidaEn, err := prueba.VerificadaEn()
	if err != nil || emitidaEn.Before(preparadaEn) {
		return AutoridadInternaEjecucionDocumentalV4{}, denegarAutoridadAtestacionPDP(nil)
	}
	comprobacion := comprobacionAtestacionAutorizacionPDPV4{
		prueba: prueba, cabecera: cabecera, solicitud: solicitudCOSE, sobre: sobre,
	}
	if comprobacion.validarPara(vinculo, emitidaEn) != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, denegarAutoridadAtestacionPDP(nil)
	}
	if err := ctx.Err(); err != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, denegarAutoridadAtestacionPDP(err)
	}

	autoridad, err := emitirAutoridadInternaEjecucionDocumentalV4(
		vinculo,
		comprobacion,
		emitidaEn,
	)
	if err != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, denegarAutoridadAtestacionPDP(nil)
	}
	if err := ctx.Err(); err != nil {
		return AutoridadInternaEjecucionDocumentalV4{}, denegarAutoridadAtestacionPDP(err)
	}
	return autoridad, nil
}

func (s *Servicio) capturarInstanteAtestacionPDP(ctx context.Context) (time.Time, error) {
	if ctx == nil || s == nil || s.reloj == nil || len(s.raices) == 0 {
		return time.Time{}, ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	if err := ctx.Err(); err != nil {
		return time.Time{}, err
	}
	instante := s.reloj.Ahora()
	if !instanteCanonicoDocumental(instante) ||
		instante.Before(s.configuracionPublicadaEn) ||
		!instante.Before(s.configuracionExpiraEn) {
		return time.Time{}, ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	return instante, nil
}

func (s *Servicio) validarCabeceraAtestacionPDP(
	cabecera domain.CabeceraAtestacionAutorizacionV1,
) error {
	if s == nil || cabecera.Validar() != nil ||
		cabecera.FormatoVersion != domain.VersionFormatoAtestacionAutorizacionV1 {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	raiz, existe := s.raices[cabecera.ClaveID]
	suite, aprobada := suiteAtestacionAutorizacionPDP(raiz.algoritmo)
	if !existe || !aprobada || raiz.audiencia != AudienciaCOSEAtestacionAutorizacionPDP ||
		cabecera.Suite != suite || cabecera.Suite != raiz.suiteAtestacionPDP ||
		cabecera.Audiencia != raiz.audienciaDespliegueAtestacionPDP {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	return nil
}

func (c comprobacionAtestacionAutorizacionPDPV4) validarPara(
	vinculo ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	instante time.Time,
) error {
	if c.cabecera.Validar() != nil || c.solicitud.Validar() != nil ||
		c.sobre.ValidarSintaxis() != nil || c.prueba.Validar() != nil ||
		c.prueba.ValidarPara(c.solicitud, c.sobre) != nil ||
		verificarPruebaAtestacionPDPContraVinculo(c.prueba, c.cabecera, vinculo, instante) != nil {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}

	if validarCabeceraAtestacionPDPContraPrueba(c.cabecera, c.prueba) != nil {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	audiencia, err := c.solicitud.Audiencia()
	if err != nil || audiencia != AudienciaCOSEAtestacionAutorizacionPDP {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	decision, err := decisionDesdeSolicitudVinculadaAtestacionPDP(vinculo, instante)
	if err != nil {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	mensaje, err := domain.SerializarMensajeAtestacionAutorizacionV1(c.cabecera, decision)
	if err != nil {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	payload, err := c.solicitud.PayloadEsperado()
	if err != nil || !bytes.Equal(payload, mensaje) {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	return nil
}

func verificarPruebaAtestacionPDPContraVinculo(
	prueba PruebaCOSESign1DocumentalVerificada,
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	vinculo ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	instante time.Time,
) error {
	if prueba.Validar() != nil || !instanteCanonicoDocumental(instante) ||
		vinculo.ValidarEn(instante) != nil ||
		prueba.audiencia != AudienciaCOSEAtestacionAutorizacionPDP ||
		!prueba.verificadaEn.Equal(instante) {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	if validarCabeceraAtestacionPDPContraPrueba(cabecera, prueba) != nil {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	decision, err := decisionDesdeSolicitudVinculadaAtestacionPDP(vinculo, instante)
	if err != nil {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	mensaje, err := domain.SerializarMensajeAtestacionAutorizacionV1(cabecera, decision)
	if err != nil || prueba.huellaPayloadSHA256 != huellaBytesDocumentales(mensaje) {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	return nil
}

func validarCabeceraAtestacionPDPContraPrueba(
	cabecera domain.CabeceraAtestacionAutorizacionV1,
	prueba PruebaCOSESign1DocumentalVerificada,
) error {
	if prueba.Validar() != nil || prueba.audiencia != AudienciaCOSEAtestacionAutorizacionPDP ||
		cabecera.Validar() != nil || !audienciaDespliegueAtestacionPDPValida(cabecera.Audiencia) {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	suite, aprobada := suiteAtestacionAutorizacionPDP(prueba.algoritmo)
	if !aprobada || cabecera.FormatoVersion != domain.VersionFormatoAtestacionAutorizacionV1 ||
		cabecera.Suite != suite || !bytes.Equal([]byte(cabecera.ClaveID), prueba.claveID) {
		return ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	return nil
}

func decisionDesdeSolicitudVinculadaAtestacionPDP(
	vinculo ports.SolicitudVinculadaAutorizacionEjecucionDocumentalV4,
	instante time.Time,
) (domain.DecisionAutorizacion, error) {
	if !instanteCanonicoDocumental(instante) || vinculo.ValidarEn(instante) != nil {
		return domain.DecisionAutorizacion{}, ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	solicitud, err := vinculo.PrepararSolicitudAplicacionEn(instante)
	if err != nil {
		return domain.DecisionAutorizacion{}, ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	evidencia, err := solicitud.EvidenciaEstructural()
	if err != nil {
		return domain.DecisionAutorizacion{}, ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	datos, err := evidencia.Datos()
	if err != nil || evidencia.ValidarEn(instante) != nil ||
		datos.Decision.ValidarEvidenciaInstantanea() != nil {
		return domain.DecisionAutorizacion{}, ErrAutoridadInternaEjecucionDocumentalV4Invalida
	}
	return datos.Decision, nil
}

func suiteAtestacionAutorizacionPDP(
	algoritmo AlgoritmoCOSEDocumental,
) (string, bool) {
	if algoritmo != AlgoritmoCOSEDocumentalEdDSA {
		return "", false
	}
	return suiteAtestacionAutorizacionPDPCOSEEdDSAV1, true
}

func audienciaDespliegueAtestacionPDPValida(audiencia string) bool {
	partes := strings.Split(audiencia, "/")
	if len(partes) != 4 || partes[0] != "vec-diputacion" {
		return false
	}
	cabecera := domain.CabeceraAtestacionAutorizacionV1{
		FormatoVersion: domain.VersionFormatoAtestacionAutorizacionV1,
		Suite:          suiteAtestacionAutorizacionPDPCOSEEdDSAV1,
		ClaveID:        "clave-validacion-audiencia",
		Audiencia:      audiencia,
	}
	if cabecera.Validar() != nil {
		return false
	}
	for _, parte := range partes[1:] {
		if parte == "" || parte == "." || parte == ".." {
			return false
		}
	}
	return true
}

func denegarAutoridadAtestacionPDP(causa error) error {
	base := errorAutoridadInternaEjecucionDocumentalV4()
	if errors.Is(causa, context.Canceled) || errors.Is(causa, context.DeadlineExceeded) {
		return errors.Join(base, causa)
	}
	return base
}
