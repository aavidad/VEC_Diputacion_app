package application

import (
	"context"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

// ValidarResultadoPreflightFirmaR5V2 protege la proyección de metadatos. La
// autorización y el vínculo con los bytes se comprueban antes en el servicio.
func ValidarResultadoPreflightFirmaR5V2(r ports.ResultadoPreflightFirmaR5V2, q ports.SolicitudPreflightFirmaR5) error {
	if ValidarResultadoPreflightFirmaR5(r.ResultadoPreflightFirmaR5, q) != nil || r.PasoPendiente > 2 {
		return ports.ErrPreflightFirmaR5NoConfiable
	}
	if r.PasoPendiente == 0 {
		if r.EntradaDocumentoRef != "" || r.EntradaDocumentoVersion != 0 || r.EntradaDocumentoHuella != "" {
			return ports.ErrPreflightFirmaR5NoConfiable
		}
		return nil
	}
	if !domain.ReferenciaOpacaValida(r.EntradaDocumentoRef) || r.EntradaDocumentoVersion == 0 ||
		r.EntradaDocumentoVersion > 9007199254740991 || !domain.HuellaSHA256FirmaValida(r.EntradaDocumentoHuella) {
		return ports.ErrPreflightFirmaR5NoConfiable
	}
	if (r.PasoPendiente == 1 && (r.EntradaDocumentoRef != r.OriginalRef || r.EntradaDocumentoVersion != r.OriginalVersion)) ||
		(r.PasoPendiente > 1 && r.EntradaDocumentoRef == r.OriginalRef) {
		return ports.ErrPreflightFirmaR5NoConfiable
	}
	return nil
}

func (s *ServicioPreflightFirmaR5) validarCabezaPreflightConEntrada(
	ctx context.Context, q ports.SolicitudPreflightFirmaR5, actor, clave string, orden int,
	catalogo domain.CircuitoFirma, lectura ports.LecturaFirmasR5,
) error {
	actual, _, cabeza, err := s.firmas.leerHistoriaPreflightFirmaR5(ctx, q.Canal.OrganizacionRef,
		q.Canal.ExpedienteRef, q.Documento, q.Canal.VersionObservada, actor, clave, orden)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil || actual.CatalogoRef != catalogo.CatalogoRef || actual.HuellaCatalogo != catalogo.HuellaCatalogo ||
		cabeza.HistoriaRevision != lectura.HistoriaRevision || cabeza.HistoriaHuella != lectura.HistoriaHuella {
		return ports.ErrPreflightFirmaR5NoDisponible
	}
	return nil
}

// La preparación utiliza la autoridad de lectura del registrador RRHH. Las
// comprobaciones nominales de cada vía siguen perteneciendo a disponibilidad.
func (s *ServicioFirmaExterna) leerHistoriaPreflightFirmaR5(ctx context.Context, org, exp, documento string, version uint64, candidato, clave string, orden int) (domain.CircuitoFirma, domain.CircuitoFirmaDocumento, ports.LecturaFirmasR5, error) {
	if s.multiple == nil {
		return s.leerHistoria(ctx, org, exp, documento, version, candidato, clave, orden)
	}
	c, err := s.base.circuitoValido(ctx)
	if err != nil {
		return domain.CircuitoFirma{}, domain.CircuitoFirmaDocumento{}, ports.LecturaFirmasR5{}, err
	}
	doc, ok := c.Documento(documento)
	if !ok || orden < 1 || orden > len(doc.Pasos) {
		return c, doc, ports.LecturaFirmasR5{}, ports.ErrPreflightFirmaR5Invalido
	}
	m := ports.MaterialConsultaFirmasR5V2{Via: ports.ViaFirmaExternaPortafirmas,
		MaterialConsultaFirmasR5: ports.MaterialConsultaFirmasR5{
			OrganizacionRef: org, ExpedienteRef: exp, VersionExpediente: version,
			Documento: documento, FirmantePrincipalCandidatoRef: candidato,
			ClaveIdempotencia: clave, PasoOrden: orden, CatalogoHuella: c.HuellaCatalogo,
		}}
	capacidad, err := s.multiple.consulta.AutorizarConsultaFirmasR5V2(ctx, m)
	if err != nil {
		if ctx.Err() != nil {
			return c, doc, ports.LecturaFirmasR5{}, ctx.Err()
		}
		return c, doc, ports.LecturaFirmasR5{}, ports.ErrFirmaDocumentoDenegada
	}
	if err := ValidarCapacidadConsultaFirmasR5V2(capacidad, m); err != nil {
		return c, doc, ports.LecturaFirmasR5{}, err
	}
	l, err := s.multiple.registro.ConsultarFirmasAutorizadasV2(ctx, m, capacidad)
	if err != nil {
		return c, doc, ports.LecturaFirmasR5{}, err
	}
	if err := validarHistoriaMultipleR5(m.MaterialConsultaFirmasR5, l); err != nil {
		return c, doc, ports.LecturaFirmasR5{}, err
	}
	l.Firmas = enriquecerFirmasRevisionMultiple(l)
	return c, doc, l.LecturaFirmasR5, nil
}

type consultaPreflightR5V2ContextoActual struct {
	origen                      ports.AutorizadorConsultaFirmasR5V2
	contextoRef, contextoHuella string
}

func (a consultaPreflightR5V2ContextoActual) AutorizarConsultaFirmasR5V2(ctx context.Context, m ports.MaterialConsultaFirmasR5V2) (ports.CapacidadConsultaFirmasR5V2, error) {
	c, err := a.origen.AutorizarConsultaFirmasR5V2(ctx, m)
	if err != nil {
		return ports.CapacidadConsultaFirmasR5V2{}, err
	}
	material := c.ExportarMaterialParaConsumidor()
	r := material.ResumenCapacidad()
	if material.ValidarEstructura() != nil || r.ContextoRef() != a.contextoRef || r.ContextoHuellaSHA256() != a.contextoHuella {
		return ports.CapacidadConsultaFirmasR5V2{}, ports.ErrFirmaDocumentoDenegada
	}
	return c, nil
}
