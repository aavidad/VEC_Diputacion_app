package bootstrap

import (
	"context"
	"encoding/json"
	"time"

	dietasdomain "vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
)

type preparadorLiquidacionEconomica struct {
	identidad dietasports.ResolutorIdentidadEfectivaCircuito
	circuito  interface {
		ConsultarDocumento(context.Context, dietasports.IdentidadEfectivaCircuito, dietasports.SolicitudDocumentoCircuito) (dietasports.DocumentoCircuito, error)
	}
	catalogo dietasports.FuenteTarifaLiquidacionPublicada
}

func (p *preparadorLiquidacionEconomica) PrepararLiquidacionEconomica(ctx context.Context, ref string, version uint64, revisiones []dietasdomain.RevisionLineaLiquidacion) (*dietasdomain.PreparacionLiquidacionPublicada, error) {
	if p == nil || ctx == nil || p.identidad == nil || p.circuito == nil || p.catalogo == nil || version < 1 || version > 999999999999999999 {
		return nil, errLiquidacionEconomicaNoDisponible
	}
	solicitud := dietasports.SolicitudDocumentoCircuito{Referencia: ref, Etapa: dietasdomain.EtapaLiquidacion}
	identidad, err := p.identidad.ResolverIdentidadEfectivaCircuito(ctx, dietasports.SolicitudOperacionCircuito{Operacion: dietasports.OperacionConsultarDocumentoCircuito, Documento: solicitud})
	if err != nil {
		return nil, err
	}
	solicitud.UnidadRef = identidad.UnidadCompetenciaRef
	vista, err := p.circuito.ConsultarDocumento(ctx, identidad, solicitud)
	if err != nil {
		return nil, err
	}
	if vista.Referencia != ref || vista.Version != version || vista.Estado != dietasdomain.EstadoPendienteLiquidacion || vista.VehiculoPropio == nil {
		return nil, dietasports.ErrEstadoCircuitoConflicto
	}
	var documento dietasdomain.DocumentoComision
	if err = json.Unmarshal(vista.Documento, &documento); err != nil || documento.GrupoDieta < 1 || documento.GrupoDieta > 3 || documento.VehiculoPropio != *vista.VehiculoPropio {
		return nil, dietasdomain.ErrLiquidacionPublicada
	}
	var calculo dietasdomain.CalculoComision
	if err = json.Unmarshal(vista.Calculo, &calculo); err != nil || documento.Validar(calculo, vista.CodigosRuta) != nil {
		return nil, dietasdomain.ErrLiquidacionPublicada
	}
	// El circuito actual sólo publica comisiones nacionales. Una publicación
	// de otro país no se infiere de la ruta ni de una entrada HTTP.
	vehiculo := "sin_vehiculo_propio"
	if documento.VehiculoPropio {
		vehiculo = "propio"
	}
	fecha, err := time.Parse("2006-01-02", vista.FechaInicio)
	if err != nil || fecha.Format("2006-01-02") != vista.FechaInicio {
		return nil, dietasdomain.ErrLiquidacionPublicada
	}
	catalogo, err := p.catalogo.ConsultarTarifaPublicada(ctx, documento.VersionTarifaAceptada, int(documento.GrupoDieta), vehiculo, fecha)
	if err != nil {
		return nil, err
	}
	if catalogo.VersionTarifaRef != documento.VersionTarifaAceptada || catalogo.PaisISO2 != "ES" || catalogo.Grupo != int(documento.GrupoDieta) || catalogo.Vehiculo != vehiculo || catalogo.ValidarForma() != nil {
		return nil, dietasdomain.ErrLiquidacionPublicada
	}
	huella, err := dietasdomain.HuellaDatosLiquidacion(documento)
	if err != nil {
		return nil, dietasdomain.ErrLiquidacionPublicada
	}
	return dietasdomain.PrepararLiquidacionPublicada(ref, int64(version), vista.FechaInicio, vista.HoraInicio, vista.FechaFin, vista.HoraFin, huella, documento, catalogo, revisiones)
}

var _ dietasports.PreparadorLiquidacionEconomicaHTTP = (*preparadorLiquidacionEconomica)(nil)
