package bootstrap

import (
	"context"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type contextoAutoridadOfertaCT interface {
	ContextoOferta(context.Context) (context.Context, ct.ContextoAutorizacionAltaV3, error)
	MotivoOferta() core.ReferenciaEntradaCatalogo
}

type preparadorSeleccionOfertaCTDesarrollo struct {
	puente    *puenteBolsaLlamamientoDesarrollo
	autoridad contextoAutoridadOfertaCT
}

type contextoConsultaEstadoOfertaCT interface {
	ContextoConsultaEstadoOferta(context.Context) (context.Context, ct.ContextoAutorizacionAltaV3, error)
	MotivoOferta() core.ReferenciaEntradaCatalogo
}

type preparadorConsultaEstadoSeleccionOfertaCTDesarrollo struct {
	autoridad contextoConsultaEstadoOfertaCT
	lector    ct.LectorContextoExpedienteSeleccionOfertaCT
}

var _ ct.PreparadorConsultaEstadoSeleccionOfertaCT = preparadorConsultaEstadoSeleccionOfertaCTDesarrollo{}

func (p preparadorConsultaEstadoSeleccionOfertaCTDesarrollo) PrepararConsultaEstadoSeleccionOfertaCT(
	ctx context.Context, expediente string,
) (ct.SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT, error) {
	vacia := ct.SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT{}
	if ctx == nil || ctx.Err() != nil || dependenciaBootstrapNula(p.autoridad) ||
		dependenciaBootstrapNula(p.lector) {
		return vacia, ct.ErrIntegracionBolsaNoDisponible
	}
	_, actor, err := p.autoridad.ContextoConsultaEstadoOferta(ctx)
	if err != nil {
		return vacia, err
	}
	// La consulta preparatoria procede de la cabecera CT actual. O6 solo se
	// consulta después de autorizar y consumir la decisión V3 nueva.
	actual, err := p.lector.LeerContextoExpedienteSeleccionOfertaCT(ctx,
		organizacionAltaContratacionTemporalDesarrollo, expediente)
	if err != nil {
		return vacia, err
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacia, err
	}
	q := ct.SolicitudAutorizacionConsultaEstadoSeleccionOfertaCT{
		OrganizacionRef: organizacionAltaContratacionTemporalDesarrollo,
		ExpedienteRef:   expediente, CategoriaRef: actual.CategoriaRef,
		CentroRef: actual.CentroRef, VersionObservada: actual.VersionActual,
		ActorRef:  actor.Resultado.Contexto.PersonaRef,
		PerfilRef: actor.Resultado.Contexto.PerfilActivoRef,
		Vinculo:   actor.Vinculo, ResultadoContexto: actor.Resultado,
		Correlacion: correlacion, MotivoAutorizacion: p.autoridad.MotivoOferta(),
	}
	return q, q.Validar()
}

var _ ct.PreparadorSeleccionOfertaCT = preparadorSeleccionOfertaCTDesarrollo{}

func (p preparadorSeleccionOfertaCTDesarrollo) PrepararIntencion(ctx context.Context,
	expediente string, version uint64, clave string,
) (ct.IntencionSeleccionOfertaCT, error) {
	vacia := ct.IntencionSeleccionOfertaCT{}
	if p.puente == nil || dependenciaBootstrapNula(p.autoridad) {
		return vacia, ct.ErrIntegracionBolsaNoDisponible
	}
	d, err := p.puente.preparacion(ctx, clave)
	if err != nil {
		return vacia, err
	}
	e := d.expediente.Fiscalizado
	if e.Referencia != expediente || e.Version != version || e.Analisis == nil ||
		e.Solicitud.NumeroPersonas < 2 || e.Solicitud.NumeroPersonas > 100 {
		return vacia, ct.ErrSeleccionOfertaCTInvalida
	}
	_, actor, err := p.autoridad.ContextoOferta(ctx)
	if err != nil {
		return vacia, err
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacia, err
	}
	i := ct.IntencionSeleccionOfertaCT{OrganizacionRef: e.OrganizacionRef,
		ExpedienteRef: e.Referencia, VersionExpediente: e.Version, ClaveIdempotencia: clave,
		ActorRef: actor.Resultado.Contexto.PersonaRef, PerfilRef: actor.Resultado.Contexto.PerfilActivoRef,
		CategoriaRef: e.Analisis.CategoriaRef, CentroRef: e.Solicitud.CentroRef,
		Vinculo: actor.Vinculo, ResultadoContexto: actor.Resultado,
		Correlacion: correlacion, MotivoAutorizacion: p.autoridad.MotivoOferta()}
	i.HuellaIntencion = ct.HuellaIntencionSeleccionOfertaCT(i)
	return i, i.Validar()
}

func (p preparadorSeleccionOfertaCTDesarrollo) PrepararComando(ctx context.Context,
	i ct.IntencionSeleccionOfertaCT,
) (ct.ComandoOfertaBolsaCT, error) {
	vacio := ct.ComandoOfertaBolsaCT{}
	if p.puente == nil || i.Validar() != nil || p.puente.configuracionNativa == nil {
		return vacio, ct.ErrIntegracionBolsaNoDisponible
	}
	d, err := p.puente.preparacion(ctx, i.ClaveIdempotencia)
	if err != nil {
		return vacio, err
	}
	e := d.expediente.Fiscalizado
	if e.Referencia != i.ExpedienteRef || e.Version != i.VersionExpediente ||
		e.OrganizacionRef != i.OrganizacionRef || d.expediente.VersionActual != e.Version ||
		e.Analisis == nil || e.Analisis.CategoriaRef != i.CategoriaRef || e.Solicitud.CentroRef != i.CentroRef {
		return vacio, ct.ErrSeleccionOfertaCTInvalida
	}
	// El proveedor existente comprueba la necesidad y su publicación RPT; sus
	// etiquetas proceden del catálogo, no de referencias escritas por el canal.
	configuracion, _, _, err := p.puente.configuracionNativa(ctx, d)
	if err != nil {
		return vacio, err
	}
	if configuracion.Referencia != e.Referencia || configuracion.Descripcion != e.Solicitud.Detalle {
		return vacio, ct.ErrSeleccionOfertaCTInvalida
	}
	verificado, err := p.puente.cargarOrigenConstituidoCT(ctx, d)
	if err != nil {
		return vacio, err
	}
	o := verificado.registro
	origen := bolsa.OrigenOfertaCT{OrganizacionRef: e.OrganizacionRef, UnidadRef: d.unidad,
		ExpedienteRef: e.Referencia, VersionExpediente: e.Version, ClaveCT: i.ClaveIdempotencia,
		NecesidadRef: d.necesidad, ActaRef: o.ActaRef, CategoriaRef: o.CategoriaRef,
		BolsaRef: o.BolsaRef, VersionBolsa: o.VersionBolsa, HuellaBolsaSHA256: o.HuellaBolsaSHA256,
		InstantaneaRef: o.InstantaneaRef, VersionInstantanea: o.VersionInstantanea,
		HuellaInstantaneaSHA256: o.HuellaInstantaneaSHA256, PoliticaRef: o.PoliticaRef,
		VersionPolitica: o.VersionPolitica, HuellaPoliticaSHA256: o.HuellaPoliticaSHA256, Lectura: o}
	numero := int(e.Solicitud.NumeroPersonas)
	if origen.ValidarPara(o.BolsaRef, numero) != nil {
		return vacio, ct.ErrSeleccionOfertaCTInvalida
	}
	contenido, err := origen.JSONCanonico()
	if err != nil {
		return vacio, err
	}
	datos := ct.DatosOfertaSeleccionCT{Categoria: configuracion.Categoria, Centro: configuracion.Centro,
		FechaInicio: e.Analisis.Periodo.Inicio.UTC().Format(time.DateOnly), Descripcion: configuracion.Descripcion}
	if !e.Analisis.Periodo.Fin.IsZero() {
		datos.FechaFin = e.Analisis.Periodo.Fin.UTC().Format(time.DateOnly)
	}
	c := ct.ComandoOfertaBolsaCT{OrganizacionRef: i.OrganizacionRef, ExpedienteRef: i.ExpedienteRef,
		VersionExpediente: i.VersionExpediente, ActorRef: i.ActorRef, BolsaRef: o.BolsaRef,
		NecesidadRef: d.necesidad, NumeroPlazas: numero, NumeroElegibles: origen.NumeroElegibles(),
		Datos: datos, OrigenCT: contenido, ClaveBolsa: ct.ClaveBolsaSeleccionOfertaCT(i),
		InicioPlazo: ct.InicioSeleccionOfertaPublicacion}
	return c, c.ValidarPara(i)
}
