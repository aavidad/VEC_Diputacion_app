package reglasbaremo

func reconstruirAprobacionGobierno(
	material materialRestauracionAprobacion,
) (AtestacionAprobacionFirmadaReglasBaremo, error) {
	atestacion, err := reconstruirReferenciaGobierno(material.Atestacion)
	if err != nil {
		return AtestacionAprobacionFirmadaReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	vinculo, err := reconstruirVinculoGobierno(material.Vinculo)
	if err != nil {
		return AtestacionAprobacionFirmadaReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	firma, err := reconstruirReferenciaGobierno(material.Firma)
	if err != nil {
		return AtestacionAprobacionFirmadaReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	politica, err := reconstruirReferenciaGobierno(material.PoliticaFirma)
	if err != nil {
		return AtestacionAprobacionFirmadaReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	return NuevaAtestacionAprobacionFirmadaReglasBaremo(
		DatosAtestacionAprobacionFirmadaReglasBaremo{
			Atestacion: atestacion, Vinculo: vinculo, Firma: firma,
			PoliticaFirma: politica,
			Firmantes:     append([]string(nil), material.Firmantes...),
			FirmadaEn:     material.FirmadaEn,
			VerificadaEn:  material.VerificadaEn,
			ValidaHasta:   material.ValidaHasta,
		},
	)
}

func reconstruirDependenciasGobierno(
	material materialRestauracionDependencias,
) (AtestacionDependenciasVigentesReglasBaremo, error) {
	atestacion, err := reconstruirReferenciaGobierno(material.Atestacion)
	if err != nil {
		return AtestacionDependenciasVigentesReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	vinculo, err := reconstruirVinculoGobierno(material.Vinculo)
	if err != nil {
		return AtestacionDependenciasVigentesReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	convocatoria, err := reconstruirReferenciaGobierno(material.Convocatoria)
	if err != nil {
		return AtestacionDependenciasVigentesReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	bases, err := reconstruirReferenciaGobierno(material.Bases)
	if err != nil {
		return AtestacionDependenciasVigentesReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	if len(material.Dependencias) == 0 ||
		len(material.Dependencias) > maximoDependenciasReglasBaremo {
		return AtestacionDependenciasVigentesReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	dependencias := make([]ReferenciaVersionada, len(material.Dependencias))
	for indice := range material.Dependencias {
		dependencias[indice], err = reconstruirReferenciaGobierno(material.Dependencias[indice])
		if err != nil {
			return AtestacionDependenciasVigentesReglasBaremo{}, ErrGobiernoEvidenciaInvalida
		}
	}
	return NuevaAtestacionDependenciasVigentesReglasBaremo(
		DatosAtestacionDependenciasVigentesReglasBaremo{
			Atestacion: atestacion, Vinculo: vinculo, Convocatoria: convocatoria,
			Bases: bases, Dependencias: dependencias, VerificadorRef: material.VerificadorRef,
			VerificadaEn: material.VerificadaEn, ValidaHasta: material.ValidaHasta,
		},
	)
}

func reconstruirAutoridadGobierno(
	material materialAutoridadGobiernoReglas,
) (AtestacionAutoridadReglasBaremo, error) {
	atestacion, err := reconstruirReferenciaGobierno(material.Atestacion)
	if err != nil {
		return AtestacionAutoridadReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	vinculo, err := reconstruirVinculoGobierno(material.Vinculo)
	if err != nil {
		return AtestacionAutoridadReglasBaremo{}, ErrGobiernoEvidenciaInvalida
	}
	var relacionada *ReferenciaVersionada
	if material.Relacionada != nil {
		valor, err := reconstruirReferenciaGobierno(*material.Relacionada)
		if err != nil {
			return AtestacionAutoridadReglasBaremo{}, ErrGobiernoEvidenciaInvalida
		}
		relacionada = &valor
	}
	return NuevaAtestacionAutoridadReglasBaremo(
		DatosAtestacionAutoridadReglasBaremo{
			Atestacion: atestacion, Vinculo: vinculo, Accion: material.Accion,
			PrincipalRef: material.PrincipalRef, Relacionada: relacionada,
			EmitidaEn: material.EmitidaEn, ValidaHasta: material.ValidaHasta,
		},
	)
}
