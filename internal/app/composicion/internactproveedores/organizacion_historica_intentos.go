package internactproveedores

import (
	"context"
	"reflect"
	"sync"
	"time"

	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	personal "vec-diputacion-granada/internal/modules/personal/domain"
	personalports "vec-diputacion-granada/internal/modules/personal/ports"
	"vec-diputacion-granada/internal/shared/plazoarranque"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type RegistradorIntentosOrganizacionHistorica interface {
	vecports.RegistradorIntentosAuditoria
	PreflightIntentoAuditoria(context.Context) error
}

// Valores publicados por el gobierno del proceso; no se obtienen de HTTP.
type ConfiguracionIntentosOrganizacionHistorica struct {
	Proceso               string
	Canal                 string
	MotivoDenegado        core.ReferenciaEntradaCatalogo
	MotivoEntradaInvalida core.ReferenciaEntradaCatalogo
	MotivoNoDisponible    core.ReferenciaEntradaCatalogo
}

type DependenciasIntentosOrganizacionHistorica struct {
	Registrador   RegistradorIntentosOrganizacionHistorica
	Configuracion ConfiguracionIntentosOrganizacionHistorica
}

type RegistroIntentosOrganizacionHistorica struct {
	destino RegistradorIntentosOrganizacionHistorica
	config  ConfiguracionIntentosOrganizacionHistorica
}

func NuevoRegistroIntentosOrganizacionHistorica(d DependenciasIntentosOrganizacionHistorica) (*RegistroIntentosOrganizacionHistorica, error) {
	if interfazNula(d.Registrador) {
		return nil, personal.ErrOrganizacionHistoricaNoDisponible
	}
	c := d.Configuracion
	for _, motivo := range []core.ReferenciaEntradaCatalogo{c.MotivoDenegado, c.MotivoEntradaInvalida, c.MotivoNoDisponible} {
		datos := core.DatosIntentoAuditoria{Accion: personal.AccionConsultaOrganizacionHistorica, ModuloID: "personal", RecursoRef: "organizacion:validacion", FinalidadRef: "consultar_organizacion_historica", Resultado: core.ResultadoIntentoAuditoriaError, Motivo: motivo, Proceso: c.Proceso, Canal: c.Canal, CorrelacionRef: "correlacion_00000000000000000000000000000000"}
		if datos.Validar() != nil || c.Canal != string(core.SuperficieAutenticacionInternaCorporativaV1) {
			return nil, personal.ErrOrganizacionHistoricaNoDisponible
		}
	}
	return &RegistroIntentosOrganizacionHistorica{d.Registrador, c}, nil
}

func (r *RegistroIntentosOrganizacionHistorica) VerificarRegistroConsultaOrganizacionHistorica(ctx context.Context) error {
	if _, err := intentoOriginalOrganizacionHistorica(ctx); err != nil {
		return personal.ErrOrganizacionHistoricaNoDisponible
	}
	return r.PreflightIntentosOrganizacionHistorica(ctx)
}

func (r *RegistroIntentosOrganizacionHistorica) PreflightIntentosOrganizacionHistorica(ctx context.Context) error {
	if r == nil || ctx == nil || ctx.Err() != nil || interfazNula(r.destino) || r.destino.PreflightIntentoAuditoria(ctx) != nil {
		return personal.ErrOrganizacionHistoricaNoDisponible
	}
	return nil
}

type claveIntentoOrganizacionHistorica struct{}
type intentoOrganizacionHistorica struct {
	identidad                                  ct.ContextoAutorizacionAltaV3
	organismo, unidad, referencia, correlacion string
	mu                                         sync.Mutex
	orden                                      *vecports.OrdenIntentoAuditoria
	acuse                                      *vecports.AcuseIntentoAuditoria
}

func intentoOriginalOrganizacionHistorica(ctx context.Context) (*intentoOrganizacionHistorica, error) {
	if ctx != nil {
		if in, ok := ctx.Value(claveIntentoOrganizacionHistorica{}).(*intentoOrganizacionHistorica); ok && in != nil {
			return in, nil
		}
	}
	return nil, personal.ErrOrganizacionHistoricaNoDisponible
}

func (r *RegistroIntentosOrganizacionHistorica) RegistrarIntentoConsultaOrganizacionHistorica(ctx context.Context, in personalports.IntentoConsultaOrganizacionHistorica) error {
	if r == nil || ctx == nil || interfazNula(r.destino) {
		return personal.ErrOrganizacionHistoricaNoDisponible
	}
	i, err := intentoOriginalOrganizacionHistorica(ctx)
	if err != nil {
		return err
	}
	motivo, resultado := r.config.MotivoNoDisponible, core.ResultadoIntentoAuditoriaError
	switch in.Motivo {
	case "denegado":
		motivo, resultado = r.config.MotivoDenegado, core.ResultadoIntentoAuditoriaDenegado
	case "entrada_invalida":
		motivo = r.config.MotivoEntradaInvalida
	case "no_disponible":
	default:
		return personal.ErrOrganizacionHistoricaNoDisponible
	}
	datos := core.DatosIntentoAuditoria{Accion: personal.AccionConsultaOrganizacionHistorica, ModuloID: "personal", RecursoRef: "personal:organizacion_historica:" + i.organismo, FinalidadRef: "consultar_organizacion_historica", Resultado: resultado, Motivo: motivo, Proceso: r.config.Proceso, Canal: r.config.Canal, CorrelacionRef: i.correlacion}
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.orden == nil {
		o, err := vecports.NuevaOrdenIntentoAuditoria(i.referencia, i.identidad.Resultado, i.identidad.Vinculo, datos)
		if err != nil {
			return personal.ErrOrganizacionHistoricaNoDisponible
		}
		i.orden = &o
	} else {
		o, err := i.orden.Datos()
		if err != nil || o.Datos != datos {
			return personal.ErrOrganizacionHistoricaNoDisponible
		}
	}
	if i.acuse != nil {
		return nil
	}
	for range 2 {
		acuse, err := r.destino.AppendIntentoAuditoria(ctx, *i.orden)
		if err == nil && acuse.ValidarPara(*i.orden) == nil {
			i.acuse = &acuse
			return nil
		}
	}
	return personal.ErrOrganizacionHistoricaNoDisponible
}

type consultaOrganizacionHistorica interface {
	Consultar(context.Context, personal.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error)
}

type fuenteOriginalOrganizacionHistorica interface {
	ContextoOriginalOrganizacionHistoricaParaAuditoria(context.Context) (ct.ContextoAutorizacionAltaV3, string, string, error)
}

type ConsultaOrganizacionHistoricaConIntentos struct {
	consulta consultaOrganizacionHistorica
	fuente   fuenteOriginalOrganizacionHistorica
	registro *RegistroIntentosOrganizacionHistorica
}

func NuevaConsultaOrganizacionHistoricaConIntentos(c consultaOrganizacionHistorica, f fuenteOriginalOrganizacionHistorica, r *RegistroIntentosOrganizacionHistorica) (*ConsultaOrganizacionHistoricaConIntentos, error) {
	if interfazNula(c) || interfazNula(f) || r == nil || interfazNula(r.destino) {
		return nil, personal.ErrOrganizacionHistoricaNoDisponible
	}
	return &ConsultaOrganizacionHistoricaConIntentos{c, f, r}, nil
}

func (c *ConsultaOrganizacionHistoricaConIntentos) Consultar(ctx context.Context, s personal.SolicitudConsultaOrganizacionHistorica) (personalports.ResultadoConsultaOrganizacionHistorica, error) {
	var vacio personalports.ResultadoConsultaOrganizacionHistorica
	if c == nil || ctx == nil || interfazNula(c.consulta) || interfazNula(c.fuente) || c.registro == nil {
		return vacio, personal.ErrOrganizacionHistoricaNoDisponible
	}
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		return vacio, personal.ErrOrganizacionHistoricaNoDisponible
	}
	refCorrelacion, err := correlacion.ValorCanonico()
	if err != nil {
		return vacio, personal.ErrOrganizacionHistoricaNoDisponible
	}
	capturaCtx, cancelar := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
	defer cancelar()
	identidad, org, unidad, err := c.fuente.ContextoOriginalOrganizacionHistoricaParaAuditoria(capturaCtx)
	if err != nil || identidad.Resultado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.Resultado) != nil {
		return vacio, personal.ErrOrganizacionHistoricaNoDisponible
	}
	copia, err := identidad.Resultado.Clonar()
	if err != nil {
		return vacio, personal.ErrOrganizacionHistoricaNoDisponible
	}
	identidad.Resultado = copia
	ref, err := vecports.NuevaReferenciaIntentoAuditoria()
	if err != nil {
		return vacio, personal.ErrOrganizacionHistoricaNoDisponible
	}
	ctx = context.WithValue(ctx, claveIntentoOrganizacionHistorica{}, &intentoOrganizacionHistorica{identidad: identidad, organismo: org, unidad: unidad, referencia: ref, correlacion: refCorrelacion})
	if !reflect.DeepEqual(s.Actor, copia.Contexto) || s.Selector.OrganismoRef != org || (unidad != "" && s.Selector.UnidadClave != unidad) {
		auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), plazoarranque.Ampliar(2*time.Second))
		defer cancel()
		if c.registro.RegistrarIntentoConsultaOrganizacionHistorica(auditCtx, personalports.IntentoConsultaOrganizacionHistorica{Motivo: "denegado"}) != nil {
			return vacio, personal.ErrOrganizacionHistoricaNoDisponible
		}
		return vacio, personal.ErrConsultaOrganizacionHistoricaDenegada
	}
	return c.consulta.Consultar(ctx, s)
}

var _ personalports.RegistroIntentosConsultaOrganizacionHistorica = (*RegistroIntentosOrganizacionHistorica)(nil)
