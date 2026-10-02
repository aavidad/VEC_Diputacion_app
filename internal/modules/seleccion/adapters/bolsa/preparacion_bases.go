package bolsa

import (
	"context"
	"errors"

	dominio "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
)

type CanonizadorBases struct{}

var _ ports.CanonizadorBases = CanonizadorBases{}

func (CanonizadorBases) ComprobarReferencia(r dominio.ReferenciaConfiguracionConvocatoria) error {
	return r.Validar()
}

// EvaluarMaterialBases traduce el contrato de Seleccion a la unica evaluacion
// de preparacion del dominio Bolsa. Las referencias siguen siendo propuestas.
func (CanonizadorBases) EvaluarMaterialBases(ctx context.Context, material ports.MaterialBasesPropuesto) (ports.EvaluacionMaterialBases, error) {
	var vacio ports.EvaluacionMaterialBases
	if ctx == nil {
		return vacio, ports.ErrPreparadorBasesNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return vacio, err
	}
	r := material.Referencias
	propuesta := prep.Material{Contenido: material.Contenido, Referencias: []prep.ReferenciaPropuesta{
		{Campo: "fuente_bases", Referencia: r.FuenteBases}, {Campo: "catalogos", Referencia: r.Catalogos},
		{Campo: "calendario", Referencia: r.Calendario}, {Campo: "reglas_baremacion", Referencia: r.ReglasBaremacion},
		{Campo: "flujo_proceso", Referencia: r.FlujoProceso}, {Campo: "flujo_solicitud", Referencia: r.FlujoSolicitud},
		{Campo: "plantilla", Referencia: r.Plantilla}, {Campo: "plaza", Referencia: r.Plaza},
		{Campo: "oep", Referencia: r.OEP}, {Campo: "rpt", Referencia: r.RPT},
	}}
	evaluacion, err := prep.EvaluarMaterialBases(propuesta)
	if cancelacion := ctx.Err(); cancelacion != nil {
		return vacio, cancelacion
	}
	if errors.Is(err, prep.ErrMaterialInvalido) {
		return vacio, ports.ErrMaterialBasesInvalido
	}
	if err != nil {
		return vacio, ports.ErrPreparadorBasesNoDisponible
	}
	resultado := ports.EvaluacionMaterialBases{ContenidoCanonicoBolsa: evaluacion.ContenidoCanonico}
	for _, p := range evaluacion.Pendientes {
		resultado.Pendientes = append(resultado.Pendientes, ports.PendientePreparacionBases{Campo: p.Campo, Codigo: p.Codigo})
	}
	return resultado, nil
}

func (CanonizadorBases) CanonizarContenido(ctx context.Context, c dominio.ContenidoPublicableConvocatoria) (dominio.ContenidoPublicableConvocatoria, error) {
	if err := ctx.Err(); err != nil {
		return dominio.ContenidoPublicableConvocatoria{}, err
	}
	return c.ClonarCanonico()
}
