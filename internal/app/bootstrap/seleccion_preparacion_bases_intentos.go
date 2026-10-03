package bootstrap

import (
	"bytes"
	"context"
	"errors"

	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Todo consumidor de producción usa este puerto decorado. El servicio V3
// conserva las transacciones de negocio; sus intentos fallidos se registran
// después del retorno, con la identidad capturada antes de la operación.
type preparadorBasesAuditadoV3 struct {
	servicio               bolsaports.PreparadorBasesDurableV3
	broker                 *proveedorPreparacionBasesV3
	registrador            vecports.RegistradorIntentosAuditoria
	proceso                string
	denegado, errorTecnico core.ReferenciaEntradaCatalogo
}

func nuevoPreparadorBasesAuditadoV3(s bolsaports.PreparadorBasesDurableV3, b *proveedorPreparacionBasesV3, r vecports.RegistradorIntentosAuditoria,
	proceso string, denegado, errorTecnico core.ReferenciaEntradaCatalogo) (*preparadorBasesAuditadoV3, error) {
	if dependenciaEsNulaContratacionTemporalDesarrollo(s) || b == nil || dependenciaEsNulaContratacionTemporalDesarrollo(r) ||
		!procesoAuditoriaIntentosConfigurado(proceso) || !core.ReferenciaMotivoAutorizacionV2Valida(denegado) || !core.ReferenciaMotivoAutorizacionV2Valida(errorTecnico) {
		return nil, bolsaports.ErrPreparacionBasesNoDisponible
	}
	return &preparadorBasesAuditadoV3{s, b, r, proceso, denegado, errorTecnico}, nil
}

func (p *preparadorBasesAuditadoV3) Guardar(ctx context.Context, q bolsaports.SolicitudGuardarPreparacionBasesV3) (bolsaports.ResultadoPreparacionBasesV3, error) {
	z, err := p.capturar(ctx, q.Actor, bolsaports.AccionGuardarPreparacionBases)
	if err != nil {
		return bolsaports.ResultadoPreparacionBasesV3{}, err
	}
	preparacion, err := bolsaapp.PrepararGuardadoPreparacionBasesV3(q)
	if err != nil {
		return bolsaports.ResultadoPreparacionBasesV3{}, p.registrar(ctx, z, q.Correlacion, "", bolsaports.AccionGuardarPreparacionBases, err)
	}
	resultado, err := p.servicio.Guardar(ctx, q)
	if err != nil {
		return bolsaports.ResultadoPreparacionBasesV3{}, p.registrar(ctx, z, q.Correlacion, preparacion.Recurso.Referencia, bolsaports.AccionGuardarPreparacionBases, err)
	}
	return resultado, nil
}

func (p *preparadorBasesAuditadoV3) Consultar(ctx context.Context, q bolsaports.SolicitudConsultarPreparacionBasesV3) (bolsaports.ResultadoPreparacionBasesV3, error) {
	z, err := p.capturar(ctx, q.Actor, bolsaports.AccionConsultarPreparacionBases)
	if err != nil {
		return bolsaports.ResultadoPreparacionBasesV3{}, err
	}
	preparacion, err := bolsaapp.PrepararConsultaPreparacionBasesV3(q)
	if err != nil {
		return bolsaports.ResultadoPreparacionBasesV3{}, p.registrar(ctx, z, q.Correlacion, "", bolsaports.AccionConsultarPreparacionBases, err)
	}
	resultado, err := p.servicio.Consultar(ctx, q)
	if err != nil {
		return bolsaports.ResultadoPreparacionBasesV3{}, p.registrar(ctx, z, q.Correlacion, preparacion.Recurso.Referencia, bolsaports.AccionConsultarPreparacionBases, err)
	}
	return resultado, nil
}

func (p *preparadorBasesAuditadoV3) capturar(ctx context.Context, actor core.ContextoActor, accion string) (contextoSeguridadComunDesarrollo, error) {
	var cero contextoSeguridadComunDesarrollo
	if p == nil || p.broker == nil {
		return cero, bolsaports.ErrPreparacionBasesNoDisponible
	}
	z, i, err := p.broker.contexto(ctx)
	if err != nil {
		return cero, err
	}
	b, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || !bytes.Equal(b, z.Resultado.RepresentacionCanonica) || p.broker.perfiles[i].accion != accion {
		return cero, bolsaports.ErrPreparacionBasesNoDisponible
	}
	return z, nil
}

func (p *preparadorBasesAuditadoV3) registrar(ctx context.Context, z contextoSeguridadComunDesarrollo, c core.ReferenciaCorrelacionAutorizacionV2,
	recurso, accion string, err error) error {
	// Ausencias y conflictos son resultados autorizados con auditoría de acceso
	// en la transacción de Bolsa. No son denegaciones ni errores técnicos.
	if err == nil {
		return err
	}
	tecnico := errorTecnicoPDPPreparacionBasesV3(ctx, err)
	denegacion := errors.Is(err, bolsaports.ErrPreparacionBasesDenegada)
	if !tecnico && !denegacion && bolsaapp.AccesoConfirmadoPreparacionBasesV3(err) {
		return err
	}
	resultado, motivo := core.ResultadoIntentoAuditoriaError, p.errorTecnico
	if denegacion && !tecnico {
		resultado, motivo = core.ResultadoIntentoAuditoriaDenegado, p.denegado
	}
	correlacion, e := c.ValorCanonico()
	if e != nil {
		return bolsaports.ErrPreparacionBasesNoDisponible
	}
	vinculo, e := z.Vinculo.Datos()
	if e != nil {
		return bolsaports.ErrPreparacionBasesNoDisponible
	}
	ref, e := vecports.NuevaReferenciaIntentoAuditoria()
	if e != nil {
		return bolsaports.ErrPreparacionBasesNoDisponible
	}
	if recurso == "" {
		recurso = ref
	}
	orden, e := vecports.NuevaOrdenIntentoAuditoria(ref, z.Resultado, z.Vinculo, core.DatosIntentoAuditoria{Accion: accion, ModuloID: "bolsa",
		RecursoRef: recurso, FinalidadRef: bolsaports.FinalidadPreparacionBases, Resultado: resultado, Motivo: motivo,
		Proceso: p.proceso, Canal: string(vinculo.Superficie), CorrelacionRef: correlacion})
	if e != nil {
		return bolsaports.ErrPreparacionBasesNoDisponible
	}
	// La cuenta dedicada de L impone el plazo de configuración en su propio
	// adaptador. La desconexión HTTP no cancela el registro del hecho observado.
	acuse, e := p.registrador.AppendIntentoAuditoria(context.WithoutCancel(ctx), orden)
	if e != nil || acuse.ValidarPara(orden) != nil {
		return bolsaports.ErrPreparacionBasesNoDisponible
	}
	return err
}

var _ bolsaports.PreparadorBasesDurableV3 = (*preparadorBasesAuditadoV3)(nil)
