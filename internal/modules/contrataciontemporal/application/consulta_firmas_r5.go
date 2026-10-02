package application

import (
	"slices"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/application/consultafirmas"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

// CamposConsultaFirmasR5 amplía CT152 con procedencia, original, cabeza
// histórica y dos indicadores mínimos. Nunca devuelve principal,
// certificado, actor ni perfil de la firma.
func CamposConsultaFirmasR5() []string {
	campos := append(consultafirmas.CamposConsultaFirmasDocumento(),
		"Via", "OriginalRef", "OriginalVersion",
		"ReferenciaPortafirmasDeclarada", "FechaPortafirmasDeclarada",
		"FirmantePrincipalAcreditado", "CoincideFirmanteCandidato",
		"HistoriaRevision", "HistoriaHuella",
		"CoincideFirmanteEnOtroPaso", "HistoriaSeparacionAcreditada")
	slices.Sort(campos)
	return campos
}

func RecursoConsultaFirmasR5(m ports.MaterialConsultaFirmasR5) (vecdomain.RecursoAutorizable, error) {
	h, err := m.HuellaSHA256()
	if err != nil {
		return vecdomain.RecursoAutorizable{}, ports.ErrSolicitudFirmaDocumentoInvalida
	}
	return vecdomain.RecursoAutorizable{
		Referencia: m.ExpedienteRef, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoConsultaFirmasR5,
		Ambitos:   map[string]string{"organizacion_ref": m.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": h},
	}, nil
}

func ValidarCapacidadConsultaFirmasR5(c ports.CapacidadConsultaFirmasR5, m ports.MaterialConsultaFirmasR5) error {
	r, err := RecursoConsultaFirmasR5(m)
	if err != nil {
		return ports.ErrFirmaDocumentoDenegada
	}
	h, err := r.HuellaContextoAutorizacionSHA256()
	a := c.ExportarMaterialParaConsumidor()
	resumen := a.ResumenCapacidad()
	if err != nil || a.ValidarEstructura() != nil ||
		resumen.Operacion() != ports.AccionConsultarFirmasR5 || resumen.EfectoRef() != m.ExpedienteRef ||
		resumen.EfectoHuellaSHA256() != h || resumen.AudienciaConsumo() != ports.AudienciaConsultaFirmasR5V3 {
		return ports.ErrFirmaDocumentoDenegada
	}
	return nil
}

// Una fachada SQL que añadiese identidad/certificado a esta lectura violaría
// la proyección concedida. Las filas CT118 conservan procedencia vacía.
func validarProyeccionFirmasR5(m ports.MaterialConsultaFirmasR5, lectura ports.LecturaFirmasR5) error {
	if lectura.HistoriaRevision > 9007199254740991 ||
		!domain.HuellaSHA256FirmaValida(lectura.HistoriaHuella) {
		return ports.ErrResultadoFirmaDocumentoInvalido
	}
	for _, f := range lectura.Firmas {
		if f.Documento != m.Documento || f.ExpedienteVersion == 0 || f.ExpedienteVersion > m.VersionExpediente ||
			(f.CoincideFirmanteCandidato && !f.FirmantePrincipalAcreditado) ||
			f.CertificadoHuella != "" || f.FirmanteRef != "" || f.ActorRef != "" || f.PerfilRef != "" ||
			!domain.ReferenciaOpacaValida(f.FirmaRef) || !domain.ReferenciaOpacaValida(f.ReciboRef) {
			return ports.ErrResultadoFirmaDocumentoInvalido
		}
		switch f.Via {
		case "":
			if f.FirmantePrincipalAcreditado || f.CoincideFirmanteCandidato ||
				f.HistoriaRevision != 0 || f.HistoriaHuella != "" ||
				f.OriginalRef != "" || f.OriginalVersion != 0 ||
				f.ReferenciaPortafirmasDeclarada != "" || f.FechaPortafirmasDeclarada != "" {
				return ports.ErrResultadoFirmaDocumentoInvalido
			}
		case ports.ViaFirmaCertificadoVEC:
			if f.Resultado != domain.ResultadoFirmaFirmado || !f.FirmantePrincipalAcreditado ||
				f.HistoriaRevision > lectura.HistoriaRevision ||
				!domain.HuellaSHA256FirmaValida(f.HistoriaHuella) ||
				!domain.ReferenciaOpacaValida(f.OriginalRef) || f.OriginalVersion == 0 ||
				f.ReferenciaPortafirmasDeclarada != "" || f.FechaPortafirmasDeclarada != "" {
				return ports.ErrResultadoFirmaDocumentoInvalido
			}
		case ports.ViaFirmaExternaPortafirmas:
			if f.Resultado != domain.ResultadoFirmaFirmado || !f.FirmantePrincipalAcreditado ||
				f.HistoriaRevision > lectura.HistoriaRevision ||
				!domain.HuellaSHA256FirmaValida(f.HistoriaHuella) ||
				!domain.ReferenciaOpacaValida(f.OriginalRef) || f.OriginalVersion == 0 ||
				!ports.ReferenciaPortafirmasDeclaradaValida(f.ReferenciaPortafirmasDeclarada) {
				return ports.ErrResultadoFirmaDocumentoInvalido
			}
			if _, ok := ports.FechaFirmaExternaCanonica(f.FechaPortafirmasDeclarada); !ok {
				return ports.ErrResultadoFirmaDocumentoInvalido
			}
		default:
			return ports.ErrResultadoFirmaDocumentoInvalido
		}
	}
	return nil
}
