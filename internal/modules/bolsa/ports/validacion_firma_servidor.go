package ports

import (
	"context"
	"sort"
	"time"
)

// SolicitudValidarFirmaServidor exige que el conector ateste tanto la
// revision exacta del PDF como las evidencias embebidas del perfil solicitado.
type SolicitudValidarFirmaServidor struct {
	Contexto                              ContextoOperacionFirma
	Artefacto                             ArtefactoFirma
	Politica                              PoliticaFirmaBaremacion
	FirmanteEsperadoRef                   string
	PerfilEsperadoClave                   string
	PerfilFirmaEsperadoClave              string
	SelloTiempoEsperadoRef                string
	HuellaSelloTiempoEsperadaSHA256       string
	AumentoLongevidadEsperadoRef          string
	HuellaAumentoLongevidadEsperadaSHA256 string
	SolicitadaEn                          time.Time
}

func (s SolicitudValidarFirmaServidor) Validar() error {
	if s.Contexto.ContextoOperacionBaremacion.ValidarPara(AccionValidarFirmaDecisionBaremacion, ClaseRecursoArtefactoFirma, s.Artefacto.FirmaRef) != nil ||
		s.Contexto.Validar() != nil || s.Artefacto.Validar() != nil || s.Politica.Validar() != nil ||
		!referenciaValida(s.FirmanteEsperadoRef, 512) || !claveValida(s.PerfilEsperadoClave) ||
		!perfilFirmaAdmitidoEnPolitica(s.Politica, s.PerfilFirmaEsperadoClave) ||
		!evidenciasEmbebidasPerfilFirmaValidas(s.PerfilFirmaEsperadoClave,
			s.SelloTiempoEsperadoRef, s.HuellaSelloTiempoEsperadaSHA256,
			s.AumentoLongevidadEsperadoRef, s.HuellaAumentoLongevidadEsperadaSHA256) ||
		s.Artefacto.PoliticaFirmaRef != s.Politica.Referencia || s.Artefacto.PoliticaFirmaVersion != s.Politica.Version ||
		s.Artefacto.HuellaPoliticaFirmaSHA256 != s.Politica.HuellaSHA256 ||
		s.Artefacto.FirmanteRef != s.FirmanteEsperadoRef || s.Artefacto.PerfilFirmanteClave != s.PerfilEsperadoClave ||
		!s.Politica.VigenteEn(s.Artefacto.FirmadaEn.UTC()) ||
		s.SolicitadaEn.IsZero() || s.SolicitadaEn.Before(s.Artefacto.FirmadaEn) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

// ValidacionFirmaServidor conserva la atestacion del conector sobre una
// revision PAdES y las referencias exactas de las evidencias que encontro.
type ValidacionFirmaServidor struct {
	Estado                                  EstadoValidacionFirma
	Artefacto                               ArtefactoFirma
	ValidacionRef                           string
	HuellaValidacionSHA256                  string
	FirmanteVerificadoRef                   string
	PerfilVerificadoClave                   string
	PerfilFirmaVerificadoClave              string
	SelloTiempoVerificadoRef                string
	HuellaSelloTiempoVerificadaSHA256       string
	AumentoLongevidadVerificadoRef          string
	HuellaAumentoLongevidadVerificadaSHA256 string
	Comprobaciones                          []ComprobacionFirma
	ValidadaEn                              time.Time
}

func (v ValidacionFirmaServidor) Validar() error {
	if !v.Estado.Valido() || v.Artefacto.Validar() != nil || !referenciaValida(v.ValidacionRef, 512) ||
		!huellaSHA256Valida(v.HuellaValidacionSHA256) || !referenciaValida(v.FirmanteVerificadoRef, 512) ||
		!claveValida(v.PerfilVerificadoClave) || !perfilFirmaPermitido(v.PerfilFirmaVerificadoClave) ||
		!evidenciasEmbebidasPerfilFirmaValidas(v.PerfilFirmaVerificadoClave,
			v.SelloTiempoVerificadoRef, v.HuellaSelloTiempoVerificadaSHA256,
			v.AumentoLongevidadVerificadoRef, v.HuellaAumentoLongevidadVerificadaSHA256) ||
		v.ValidadaEn.IsZero() || v.ValidadaEn.Before(v.Artefacto.FirmadaEn) ||
		len(v.Comprobaciones) == 0 || len(v.Comprobaciones) > maximoComprobacionesFirma {
		return ErrValidacionFirmaNoConcluyente
	}
	claves := make(map[string]struct{}, len(v.Comprobaciones))
	for _, comprobacion := range v.Comprobaciones {
		if comprobacion.Validar() != nil {
			return ErrValidacionFirmaNoConcluyente
		}
		if _, existe := claves[comprobacion.Clave]; existe {
			return ErrValidacionFirmaNoConcluyente
		}
		claves[comprobacion.Clave] = struct{}{}
	}
	return nil
}

func (v ValidacionFirmaServidor) AptaParaDecision() bool {
	if v.Validar() != nil || v.Estado != EstadoValidacionFirmaValida ||
		v.FirmanteVerificadoRef != v.Artefacto.FirmanteRef ||
		v.PerfilVerificadoClave != v.Artefacto.PerfilFirmanteClave {
		return false
	}
	for _, comprobacion := range v.Comprobaciones {
		if comprobacion.Estado != EstadoComprobacionSuperada {
			return false
		}
	}
	return true
}

func (v ValidacionFirmaServidor) AptaParaPolitica(p PoliticaFirmaBaremacion) bool {
	return v.AptaParaPerfil(p, p.PerfilFirmaClave)
}

// AptaParaPerfil permite verificar cada etapa material del flujo B -> T ->
// LTA sin confundir el perfil intermedio con el objetivo final de la politica.
func (v ValidacionFirmaServidor) AptaParaPerfil(p PoliticaFirmaBaremacion, perfil string) bool {
	if p.Validar() != nil || !v.AptaParaDecision() ||
		!perfilFirmaAdmitidoEnPolitica(p, perfil) || v.PerfilFirmaVerificadoClave != perfil ||
		v.Artefacto.PoliticaFirmaRef != p.Referencia ||
		v.Artefacto.PoliticaFirmaVersion != p.Version ||
		v.Artefacto.HuellaPoliticaFirmaSHA256 != p.HuellaSHA256 {
		return false
	}
	claves := make([]string, len(v.Comprobaciones))
	for indice := range v.Comprobaciones {
		claves[indice] = v.Comprobaciones[indice].Clave
	}
	return mismoConjuntoClaves(claves, p.ComprobacionesObligatorias)
}

func (v ValidacionFirmaServidor) ValidarPara(s SolicitudValidarFirmaServidor) error {
	if s.Validar() != nil || v.Validar() != nil || !v.AptaParaPerfil(s.Politica, s.PerfilFirmaEsperadoClave) ||
		v.Artefacto != s.Artefacto ||
		v.FirmanteVerificadoRef != s.FirmanteEsperadoRef || v.PerfilVerificadoClave != s.PerfilEsperadoClave ||
		v.SelloTiempoVerificadoRef != s.SelloTiempoEsperadoRef ||
		v.HuellaSelloTiempoVerificadaSHA256 != s.HuellaSelloTiempoEsperadaSHA256 ||
		v.AumentoLongevidadVerificadoRef != s.AumentoLongevidadEsperadoRef ||
		v.HuellaAumentoLongevidadVerificadaSHA256 != s.HuellaAumentoLongevidadEsperadaSHA256 ||
		v.ValidadaEn.Before(s.SolicitadaEn) {
		return ErrValidacionFirmaNoConcluyente
	}
	return nil
}

func (v ValidacionFirmaServidor) Clonar() (ValidacionFirmaServidor, error) {
	clon := v
	clon.Comprobaciones = append([]ComprobacionFirma(nil), v.Comprobaciones...)
	sort.Slice(clon.Comprobaciones, func(i, j int) bool { return clon.Comprobaciones[i].Clave < clon.Comprobaciones[j].Clave })
	return clon, clon.Validar()
}

type ValidadorFirmaServidor interface {
	ValidarFirmaServidor(context.Context, SolicitudValidarFirmaServidor) (ValidacionFirmaServidor, error)
}
