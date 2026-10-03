package application

import (
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	docports "vec-diputacion-granada/internal/vec/documentos/ports"
)

// Los constructores V2 no requieren autoridades V1: el dictamen acumulado,
// la consulta y el efecto deben proceder de sus contratos nominales V2.
func NuevoServicioFirmaExternaV2(
	base *ServicioFirmaDocumento, verificador docports.VerificadorFirmasDocumento,
	registro ports.RegistroFirmasVerificadasV2, autorizador ports.AutorizadorFirmaVerificadaV2,
	consulta ports.AutorizadorConsultaFirmasR5V2, pdfAnterior ports.FuentePDFFirmaAnterior,
	competencia ports.FuenteCompetenciaFirmante, politica ports.FuentePoliticaMismaPersonaEnPasos,
) (*ServicioFirmaExterna, error) {
	d, err := dependenciasServicioFirmaV2(base, verificador, registro, autorizador, consulta, pdfAnterior, competencia)
	if err != nil {
		return nil, err
	}
	if nula(politica) {
		politica = nil
	}
	return &ServicioFirmaExterna{base: base, competencia: competencia, politica: politica, multiple: d}, nil
}

func NuevoServicioFirmaVecV2(
	base *ServicioFirmaDocumento, verificador docports.VerificadorFirmasDocumento,
	registro ports.RegistroFirmasVerificadasV2, autorizador ports.AutorizadorFirmaVerificadaV2,
	consulta ports.AutorizadorConsultaFirmasR5V2, pdfAnterior ports.FuentePDFFirmaAnterior,
	competencia ports.FuenteCompetenciaFirmante, politica ports.FuentePoliticaMismaPersonaEnPasos,
) (*ServicioFirmaVec, error) {
	d, err := dependenciasServicioFirmaV2(base, verificador, registro, autorizador, consulta, pdfAnterior, competencia)
	if err != nil {
		return nil, err
	}
	if nula(politica) {
		politica = nil
	}
	return &ServicioFirmaVec{base: base, competencia: competencia, politica: politica, multiple: d}, nil
}

func dependenciasServicioFirmaV2(
	base *ServicioFirmaDocumento, verificador docports.VerificadorFirmasDocumento,
	registro ports.RegistroFirmasVerificadasV2, autorizador ports.AutorizadorFirmaVerificadaV2,
	consulta ports.AutorizadorConsultaFirmasR5V2, pdfAnterior ports.FuentePDFFirmaAnterior,
	competencia ports.FuenteCompetenciaFirmante,
) (*dependenciasFirmaMultipleR5, error) {
	if base == nil || nula(base.circuito) || nula(base.original) || nula(base.custodio) || len(base.tiposCustodia) == 0 || nula(competencia) {
		return nil, ErrCircuitoFirmaNoDisponible
	}
	return nuevasDependenciasFirmaMultipleR5(verificador, registro, autorizador, consulta, pdfAnterior)
}
