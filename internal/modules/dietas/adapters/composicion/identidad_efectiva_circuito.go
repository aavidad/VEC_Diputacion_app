package composicion

import (
	"context"
	"errors"

	dietasapp "vec-diputacion-granada/internal/modules/dietas/application"
	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

var ErrIdentidadCircuitoNoDisponible = errors.New("dietas: identidad del circuito no disponible")

// FuenteCompetenciaCircuitoSinCatalogo es la fuente vigente mientras no se
// publique el catálogo gobernado de validadores competentes (duda 40): no
// acredita a nadie en ninguna etapa. No deduce competencia de cargos, perfiles
// ni de las referencias de la asignación D7.
type FuenteCompetenciaCircuitoSinCatalogo struct{}

func (FuenteCompetenciaCircuitoSinCatalogo) EstadoCompetencias(context.Context, vecdomain.ResultadoContextoActorRegistradoV2) (dietasports.EstadoCompetenciasCircuito, error) {
	return dietasports.EstadoCompetenciasCircuito{Fuente: dietasports.FuenteCompetenciaSinFuente, Etapas: []domain.EtapaCircuito{}}, nil
}

func (FuenteCompetenciaCircuitoSinCatalogo) UnidadCompetente(context.Context, vecdomain.ResultadoContextoActorRegistradoV2, domain.EtapaCircuito) (string, error) {
	return "", dietasports.ErrCompetenciaCircuitoSinFuente
}

// audienciasCircuito fija la audiencia V3 propia de cada acción del circuito.
var audienciasCircuito = map[string]string{
	"dietas.documento.revisar":                 "vec_dietas.documento.revisar.v1",
	"dietas.documento.autorizar":               "vec_dietas.documento.autorizar.v1",
	"dietas.documento.liquidar":                "vec_dietas.documento.liquidar.v1",
	"dietas.documento.fiscalizar":              "vec_dietas.documento.fiscalizar.v1",
	"dietas.bandeja.revision.consultar":        "vec_dietas.bandeja.revision.consultar.v1",
	"dietas.bandeja.autorizacion.consultar":    "vec_dietas.bandeja.autorizacion.consultar.v1",
	"dietas.bandeja.liquidacion.consultar":     "vec_dietas.bandeja.liquidacion.consultar.v1",
	"dietas.bandeja.fiscalizacion.consultar":   "vec_dietas.bandeja.fiscalizacion.consultar.v1",
	dietasapp.AccionConsultarDocumentoCircuito: "vec_dietas.circuito.documento.consultar.v1",
}

// AudienciaCircuito devuelve la audiencia V3 de una acción del circuito.
func AudienciaCircuito(accion string) (string, bool) {
	a, ok := audienciasCircuito[accion]
	return a, ok
}

// ResolutorIdentidadEfectivaCircuito compone la sesión corporativa, la
// competencia acreditada por la fuente gobernada y el material V3 de la
// acción exacta. La unidad sale siempre de la fuente, nunca del cliente.
type ResolutorIdentidadEfectivaCircuito struct {
	identidad ResolutorIdentidadRegistradaBorrador
	fuente    dietasports.FuenteCompetenciaCircuito
	emisor    EmisorMaterialBorradorV3
	motivo    vecdomain.ReferenciaEntradaCatalogo
}

func NuevoResolutorIdentidadEfectivaCircuito(i ResolutorIdentidadRegistradaBorrador, f dietasports.FuenteCompetenciaCircuito, e EmisorMaterialBorradorV3, motivo vecdomain.ReferenciaEntradaCatalogo) (*ResolutorIdentidadEfectivaCircuito, error) {
	if nulo(i) || nulo(f) || nulo(e) || motivo.Validar() != nil {
		return nil, ErrIdentidadCircuitoNoDisponible
	}
	return &ResolutorIdentidadEfectivaCircuito{identidad: i, fuente: f, emisor: e, motivo: motivo}, nil
}

func (r *ResolutorIdentidadEfectivaCircuito) base(ctx context.Context) (IdentidadRegistradaBorrador, error) {
	if r == nil || ctx == nil || nulo(r.identidad) || nulo(r.fuente) || nulo(r.emisor) {
		return IdentidadRegistradaBorrador{}, dietasports.ErrCircuitoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return IdentidadRegistradaBorrador{}, err
	}
	base, err := r.identidad.ResolverIdentidadRegistradaBorrador(ctx)
	if err != nil {
		return IdentidadRegistradaBorrador{}, opacoCircuito(ctx, err)
	}
	if base.Contexto.Validar() != nil || base.Vinculo.ValidarPara(base.Contexto) != nil {
		return IdentidadRegistradaBorrador{}, dietasports.ErrAccesoCircuitoDenegado
	}
	return base, nil
}

func (r *ResolutorIdentidadEfectivaCircuito) EstadoCompetenciasCircuito(ctx context.Context) (dietasports.EstadoCompetenciasCircuito, error) {
	base, err := r.base(ctx)
	if err != nil {
		return dietasports.EstadoCompetenciasCircuito{}, err
	}
	estado, err := r.fuente.EstadoCompetencias(ctx, base.Contexto)
	if err != nil {
		return dietasports.EstadoCompetenciasCircuito{}, opacoCircuito(ctx, err)
	}
	return estado, nil
}

func (r *ResolutorIdentidadEfectivaCircuito) ResolverIdentidadEfectivaCircuito(ctx context.Context, solicitud dietasports.SolicitudOperacionCircuito) (dietasports.IdentidadEfectivaCircuito, error) {
	var cero dietasports.IdentidadEfectivaCircuito
	base, err := r.base(ctx)
	if err != nil {
		return cero, err
	}
	etapa := solicitud.Etapa()
	if etapa.EstadoPendiente() == "" {
		return cero, domain.ErrDecisionCircuitoInvalida
	}
	unidad, err := r.fuente.UnidadCompetente(ctx, base.Contexto, etapa)
	if errors.Is(err, dietasports.ErrCompetenciaCircuitoSinFuente) {
		return cero, err
	}
	if err != nil || unidad == "" {
		return cero, opacoCircuito(ctx, err)
	}
	switch solicitud.Operacion {
	case dietasports.OperacionDecidirCircuito:
		solicitud.Decision.UnidadRef = unidad
	case dietasports.OperacionListarBandeja:
		solicitud.Consulta.UnidadRef = unidad
		if solicitud.Consulta.Limite == 0 {
			solicitud.Consulta.Limite = 20
		}
	case dietasports.OperacionConsultarDocumentoCircuito:
		solicitud.Documento.UnidadRef = unidad
	default:
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	accion, recurso, finalidad := dietasapp.ContratoCircuito(solicitud)
	audiencia, ok := AudienciaCircuito(accion)
	if accion == "" || !ok {
		return cero, domain.ErrDecisionCircuitoInvalida
	}
	efecto, err := dietasapp.ConstruirEfectoAutorizacionCircuito(base.Contexto, unidad, solicitud)
	if err != nil || efecto.Recurso.Referencia != recurso {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	correlacion, err := vecdomain.GenerarReferenciaCorrelacionAutorizacionV2(ctx, seguridadvec.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	peticion, err := vecdomain.NuevaSolicitudAutorizacionLigadaV3(vecdomain.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: base.Vinculo,
		ReferenciaMotivo:          r.motivo,
		Accion:                    accion,
		Recurso:                   efecto.Recurso,
		Finalidad:                 finalidad,
		Correlacion:               correlacion,
	})
	if err != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	decision, confirmacion, exportador, err := r.emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, peticion, base.Contexto)
	if err != nil || decision.ValidarPara(peticion) != nil || nulo(exportador) {
		return cero, opacoCircuito(ctx, err)
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(peticion, decision, confirmacion, base.Contexto, r.motivo, material, audiencia) {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	if material.PersonaVersion() != base.Contexto.Contexto.Instantanea.PersonaVersion || material.PerfilVersion() != base.Contexto.Contexto.Instantanea.PerfilVersion {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	return dietasports.IdentidadEfectivaCircuito{
		Vinculo:              base.Vinculo,
		ContextoRegistrado:   base.Contexto,
		Autorizacion:         dietasports.AutorizacionCircuitoDurable{Material: material, Accion: accion, RecursoRef: recurso, Finalidad: finalidad},
		UnidadCompetenciaRef: unidad,
	}, nil
}

func opacoCircuito(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return dietasports.ErrAccesoCircuitoDenegado
}

var _ dietasports.ResolutorIdentidadEfectivaCircuito = (*ResolutorIdentidadEfectivaCircuito)(nil)
var _ dietasports.FuenteCompetenciaCircuito = FuenteCompetenciaCircuitoSinCatalogo{}
