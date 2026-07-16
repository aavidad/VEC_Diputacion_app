package domain

func (f FirmaDecisionTecnica) evidenciaSelloCoherente() bool {
	presente := f.SelloTiempoRef != "" || f.HuellaSelloTiempoSHA256 != "" || f.PoliticaSelloTiempoRef != "" ||
		f.VinculoRevisionSelladaRef != "" || f.HuellaVinculoRevisionSelladaSHA256 != "" ||
		f.PoliticaSelloTiempoVersion != 0 || f.HuellaPoliticaSelloTiempoSHA256 != "" ||
		f.ValidacionSelloTiempoRef != "" || f.HuellaValidacionSelloTiempoSHA256 != "" || !f.SelladaEn.IsZero() ||
		f.ValidacionDocumentoSelladoRef != "" || f.HuellaValidacionDocumentoSelladoSHA256 != "" ||
		!f.ValidadoDocumentoSelladoEn.IsZero()
	if !f.RequiereSelloTiempo {
		return !presente
	}
	return referenciaOpacaValida(f.SelloTiempoRef) && huellaSHA256Valida(f.HuellaSelloTiempoSHA256) &&
		referenciaOpacaValida(f.VinculoRevisionSelladaRef) && huellaSHA256Valida(f.HuellaVinculoRevisionSelladaSHA256) &&
		referenciaOpacaValida(f.PoliticaSelloTiempoRef) && f.PoliticaSelloTiempoVersion > 0 &&
		huellaSHA256Valida(f.HuellaPoliticaSelloTiempoSHA256) &&
		referenciaOpacaValida(f.ValidacionSelloTiempoRef) &&
		huellaSHA256Valida(f.HuellaValidacionSelloTiempoSHA256) && !f.SelladaEn.IsZero() &&
		!f.SelladaEn.Before(f.ValidadaInicialEn) &&
		referenciaOpacaValida(f.ValidacionDocumentoSelladoRef) &&
		huellaSHA256Valida(f.HuellaValidacionDocumentoSelladoSHA256) &&
		!f.ValidadoDocumentoSelladoEn.IsZero() && !f.ValidadoDocumentoSelladoEn.Before(f.SelladaEn) &&
		!f.ValidadaEn.Before(f.ValidadoDocumentoSelladoEn) &&
		(f.RequiereAumentoLongevidad ||
			(f.ValidacionDocumentoSelladoRef == f.ValidacionFirmaRef &&
				f.HuellaValidacionDocumentoSelladoSHA256 == f.HuellaValidacionSHA256 &&
				f.ValidadoDocumentoSelladoEn.Equal(f.ValidadaEn)))
}

func (f FirmaDecisionTecnica) evidenciaLongevidadCoherente() bool {
	presente := f.NivelLongevidadClave != "" || f.AumentoLongevidadRef != "" ||
		f.HuellaAumentoLongevidadSHA256 != "" || f.VinculoRevisionLongevaRef != "" ||
		f.HuellaVinculoRevisionLongevaSHA256 != "" || f.PoliticaLongevidadRef != "" ||
		f.PoliticaLongevidadVersion != 0 || f.HuellaPoliticaLongevidadSHA256 != "" ||
		f.ValidacionLongevidadRef != "" || f.HuellaValidacionLongevidadSHA256 != "" || !f.AumentadaEn.IsZero()
	if !f.RequiereAumentoLongevidad {
		return !presente
	}
	return claveNegocioValida(f.NivelLongevidadClave) && referenciaOpacaValida(f.AumentoLongevidadRef) &&
		huellaSHA256Valida(f.HuellaAumentoLongevidadSHA256) && referenciaOpacaValida(f.VinculoRevisionLongevaRef) &&
		huellaSHA256Valida(f.HuellaVinculoRevisionLongevaSHA256) && referenciaOpacaValida(f.PoliticaLongevidadRef) &&
		f.PoliticaLongevidadVersion > 0 && huellaSHA256Valida(f.HuellaPoliticaLongevidadSHA256) &&
		referenciaOpacaValida(f.ValidacionLongevidadRef) &&
		huellaSHA256Valida(f.HuellaValidacionLongevidadSHA256) && !f.AumentadaEn.IsZero() &&
		!f.AumentadaEn.Before(f.ValidadoDocumentoSelladoEn)
}

func perfilFirmaDecisionValido(perfil string) bool {
	switch perfil {
	case "pades_baseline_b", "pades_baseline_t", "pades_baseline_lta":
		return true
	default:
		return false
	}
}

func (f FirmaDecisionTecnica) perfilFirmaCoherente() bool {
	switch f.PerfilFirmaAlcanzadoClave {
	case "pades_baseline_b":
		return !f.RequiereSelloTiempo && !f.RequiereAumentoLongevidad
	case "pades_baseline_t":
		return f.RequiereSelloTiempo && !f.RequiereAumentoLongevidad
	case "pades_baseline_lta":
		return f.RequiereSelloTiempo && f.RequiereAumentoLongevidad
	default:
		return false
	}
}
