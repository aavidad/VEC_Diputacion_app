package ports

import (
	"context"
	"time"
)

func perfilFirmaAdmitidoEnPolitica(p PoliticaFirmaBaremacion, perfil string) bool {
	if p.Validar() != nil || !perfilFirmaPermitido(perfil) {
		return false
	}
	switch p.PerfilFirmaClave {
	case PerfilFirmaPAdESBaselineB:
		return perfil == PerfilFirmaPAdESBaselineB
	case PerfilFirmaPAdESBaselineT:
		return perfil == PerfilFirmaPAdESBaselineB || perfil == PerfilFirmaPAdESBaselineT
	case PerfilFirmaPAdESBaselineLTA:
		return perfil == PerfilFirmaPAdESBaselineB || perfil == PerfilFirmaPAdESBaselineT ||
			perfil == PerfilFirmaPAdESBaselineLTA
	default:
		return false
	}
}

func evidenciasEmbebidasPerfilFirmaValidas(
	perfil, selloRef, huellaSello, aumentoRef, huellaAumento string,
) bool {
	selloPresente := referenciaValida(selloRef, 512) && huellaSHA256Valida(huellaSello)
	aumentoPresente := referenciaValida(aumentoRef, 512) && huellaSHA256Valida(huellaAumento)
	switch perfil {
	case PerfilFirmaPAdESBaselineB:
		return selloRef == "" && huellaSello == "" && aumentoRef == "" && huellaAumento == ""
	case PerfilFirmaPAdESBaselineT:
		return selloPresente && aumentoRef == "" && huellaAumento == ""
	case PerfilFirmaPAdESBaselineLTA:
		return selloPresente && aumentoPresente
	default:
		return false
	}
}

// ValidarRevisionPAdESDe exige una nueva revision fisica del mismo PDF
// firmado. FirmaRef y HuellaFirmaSHA256 identifican la firma criptografica
// base y permanecen estables; la referencia y la huella del contenedor PDF
// tienen que cambiar al incorporar atributos PAdES no firmados.
func (a ArtefactoFirma) ValidarRevisionPAdESDe(origen ArtefactoFirma) error {
	if a.Validar() != nil || origen.Validar() != nil ||
		a.DocumentoFirmadoRef == origen.DocumentoFirmadoRef ||
		a.HuellaDocumentoSHA256 == origen.HuellaDocumentoSHA256 {
		return ErrRevisionPDFFirmaNoConfiable
	}
	esperado := origen
	esperado.DocumentoFirmadoRef = a.DocumentoFirmadoRef
	esperado.HuellaDocumentoSHA256 = a.HuellaDocumentoSHA256
	if a != esperado {
		return ErrRevisionPDFFirmaNoConfiable
	}
	return nil
}

type SolicitudSellarTiempoFirma struct {
	Contexto          ContextoOperacionFirma
	ClaveIdempotencia string
	ArtefactoOrigen   ArtefactoFirma
	ValidacionOrigen  ValidacionFirmaServidor
	Politica          PoliticaFirmaBaremacion
	SolicitadaEn      time.Time
}

func (s SolicitudSellarTiempoFirma) Validar() error {
	if s.Contexto.ContextoOperacionBaremacion.ValidarPara(AccionSellarTiempoDecisionBaremacion, ClaseRecursoArtefactoFirma, s.ArtefactoOrigen.FirmaRef) != nil ||
		s.Contexto.Validar() != nil || !referenciaValida(s.ClaveIdempotencia, 512) ||
		s.ArtefactoOrigen.Validar() != nil || s.ValidacionOrigen.Validar() != nil ||
		s.ValidacionOrigen.Artefacto != s.ArtefactoOrigen ||
		!s.ValidacionOrigen.AptaParaPerfil(s.Politica, PerfilFirmaPAdESBaselineB) ||
		s.Politica.Validar() != nil || !s.Politica.RequiereSelloTiempo || s.SolicitadaEn.IsZero() ||
		s.SolicitadaEn.Before(s.ValidacionOrigen.ValidadaEn) {
		return ErrSolicitudBaremacionInvalida
	}
	return nil
}

type SelloTiempoFirma struct {
	SelloTiempoRef                  string
	HuellaSelloTiempoSHA256         string
	ArtefactoOrigen                 ArtefactoFirma
	ArtefactoSellado                ArtefactoFirma
	PoliticaSelloTiempoRef          string
	PoliticaSelloTiempoVersion      int
	HuellaPoliticaSelloTiempoSHA256 string
	ValidacionSelloRef              string
	HuellaValidacionSHA256          string
	SelladoEn                       time.Time
}

func (s SelloTiempoFirma) Validar() error {
	if !referenciaValida(s.SelloTiempoRef, 512) || !huellaSHA256Valida(s.HuellaSelloTiempoSHA256) ||
		s.ArtefactoSellado.ValidarRevisionPAdESDe(s.ArtefactoOrigen) != nil ||
		!referenciaValida(s.PoliticaSelloTiempoRef, 512) || s.PoliticaSelloTiempoVersion < 1 ||
		!huellaSHA256Valida(s.HuellaPoliticaSelloTiempoSHA256) || !referenciaValida(s.ValidacionSelloRef, 512) ||
		!huellaSHA256Valida(s.HuellaValidacionSHA256) || s.SelladoEn.IsZero() ||
		s.SelladoEn.Before(s.ArtefactoOrigen.FirmadaEn) {
		return ErrSelloTiempoNoDisponible
	}
	return nil
}

func (s SelloTiempoFirma) ValidarPara(sol SolicitudSellarTiempoFirma) error {
	if sol.Validar() != nil || s.Validar() != nil || s.ArtefactoOrigen != sol.ArtefactoOrigen ||
		s.PoliticaSelloTiempoRef != sol.Politica.PoliticaSelloTiempoRef ||
		s.PoliticaSelloTiempoVersion != sol.Politica.PoliticaSelloTiempoVersion ||
		s.HuellaPoliticaSelloTiempoSHA256 != sol.Politica.HuellaPoliticaSelloTiempoSHA256 ||
		s.SelladoEn.Before(sol.SolicitadaEn) {
		return ErrSelloTiempoNoDisponible
	}
	return nil
}

type SelladorTiempoFirma interface {
	SellarTiempoFirma(context.Context, SolicitudSellarTiempoFirma) (SelloTiempoFirma, error)
}
