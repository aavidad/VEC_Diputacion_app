package bootstrap

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"reflect"

	bolsaapp "vec-diputacion-granada/internal/modules/bolsa/application"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	selauth "vec-diputacion-granada/internal/modules/seleccion/adapters/autorizacion"
	selhttp "vec-diputacion-granada/internal/modules/seleccion/adapters/http"
	"vec-diputacion-granada/internal/vec/adapters/seguridad"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El broker sólo consume la sesión y el PDP existentes. No dispone de
// publicadores ni de un actor o perfil que pueda elegir el cuerpo HTTP.
type proveedorPreparacionBasesV3 struct {
	perfiles         [2]*perfilPreparacionBasesV3
	sesiones         [2]*proveedorSesionConsultaRRHHDesarrollo
	materiales       [2]*proveedorMaterialAltaContratacionTemporalDesarrollo
	pdp              vecports.AutorizadorSolicitudLigadaV3
	reloj            relojContratacionTemporalDesarrollo
	registrarRechazo func(*http.Request) error
	registrarEntrada func(context.Context, contextoSeguridadComunDesarrollo, int, core.ReferenciaCorrelacionAutorizacionV2, error) error
}

func nuevoProveedorPreparacionBasesV3(ps [2]*perfilPreparacionBasesV3, ss [2]*proveedorSesionConsultaRRHHDesarrollo,
	ms [2]*proveedorMaterialAltaContratacionTemporalDesarrollo, pdp vecports.AutorizadorSolicitudLigadaV3,
	reloj relojContratacionTemporalDesarrollo, registrarRechazo func(*http.Request) error) (*proveedorPreparacionBasesV3, error) {
	if registrarRechazo == nil || dependenciaEsNulaContratacionTemporalDesarrollo(pdp) || ps[0] == nil || ps[1] == nil || ps[0].perfilRef() == ps[1].perfilRef() || ms[0] == ms[1] {
		return nil, bolsaports.ErrPreparacionBasesNoDisponible
	}
	for i, p := range ps {
		par := paresPreparacionBasesHTTPV3()[i]
		if p.soporte == nil || ss[i] == nil || ss[i].soporte != p.soporte || ss[i].base.Contexto.PerfilActivoRef != p.perfilRef() ||
			ms[i] == nil || ms[i].atestador == nil || ms[i].confianza == nil || ms[i].emisor == nil ||
			p.plantilla.Validar() != nil || p.accion != par.accion || p.ruta != par.ruta || p.audiencia != DescriptoresMaterialPreparacionBasesV3()[i].Audiencia {
			return nil, bolsaports.ErrPreparacionBasesNoDisponible
		}
	}
	if !ss[0].fronteras.mismaInstancia(ss[1].fronteras) {
		return nil, bolsaports.ErrPreparacionBasesNoDisponible
	}
	return &proveedorPreparacionBasesV3{perfiles: ps, sesiones: ss, materiales: ms, pdp: pdp, reloj: reloj, registrarRechazo: registrarRechazo}, nil
}

func (p *proveedorPreparacionBasesV3) indice(ctx context.Context) (int, error) {
	if p == nil || ctx == nil {
		return 0, bolsaports.ErrPreparacionBasesNoDisponible
	}
	c, ok := ctx.Value(claveCapacidadConsultasContratacionTemporalDesarrollo{}).(capacidadConsultaContratacionTemporalDesarrollo)
	if !ok {
		return 0, bolsaports.ErrPreparacionBasesDenegada
	}
	for i, perfil := range p.perfiles {
		if perfil == nil || p.sesiones[i] == nil || c.ruta != perfil.ruta {
			continue
		}
		if !p.sesiones[i].sesionPreparacionBasesHTTPV3(ctx, c.ruta) || c.contextoOperacion == nil {
			return 0, bolsaports.ErrPreparacionBasesDenegada
		}
		if ctx.Err() != nil {
			return 0, bolsaports.ErrPreparacionBasesNoDisponible
		}
		return i, nil
	}
	return 0, bolsaports.ErrPreparacionBasesDenegada
}

func (p *proveedorPreparacionBasesV3) contexto(ctx context.Context) (contextoSeguridadComunDesarrollo, int, error) {
	var cero contextoSeguridadComunDesarrollo
	i, err := p.indice(ctx)
	if err != nil {
		return cero, 0, err
	}
	perfil := p.perfiles[i]
	c, valida := perfil.soporte.capacidadValida(ctx)
	if !valida {
		return cero, 0, bolsaports.ErrPreparacionBasesDenegada
	}
	holder := c.contextoOperacion
	holder.mu.Lock()
	defer holder.mu.Unlock()
	if holder.soporte == nil {
		holder.soporte = perfil.soporte
		actual, err := p.sesiones[i].ResolverContexto(ctx)
		if err == nil && actual.Resultado.Validar() == nil && actual.Vinculo.ValidarPara(actual.Resultado) == nil &&
			mismoContextoEsperadoRegistradoDesarrollo(perfil.soporte.contextoEsperadoRegistrado, actual.Resultado) {
			holder.contexto.Vinculo, holder.contexto.Resultado = actual.Vinculo, actual.Resultado
		} else if errors.Is(err, ctports.ErrConsultaRRHHNoDisponible) || ctx.Err() != nil {
			holder.err = bolsaports.ErrPreparacionBasesNoDisponible
		} else {
			holder.err = bolsaports.ErrPreparacionBasesDenegada
		}
	}
	if holder.err != nil {
		return cero, 0, holder.err
	}
	if holder.soporte != perfil.soporte || holder.contexto.Resultado.Contexto.PerfilActivoRef != perfil.perfilRef() ||
		holder.contexto.Vinculo.ValidarPara(holder.contexto.Resultado) != nil ||
		!holder.contexto.Vinculo.VigenteEn(p.reloj.Ahora(), holder.contexto.Resultado) ||
		holder.contexto.Resultado.Contexto.Principal.AuthMethod != core.AuthMethodCertificate ||
		holder.contexto.Resultado.Contexto.Principal.AuthAssurance != core.AuthAssuranceHigh {
		return cero, 0, bolsaports.ErrPreparacionBasesDenegada
	}
	r, err := holder.contexto.Resultado.Clonar()
	if err != nil {
		return cero, 0, bolsaports.ErrPreparacionBasesNoDisponible
	}
	return contextoSeguridadComunDesarrollo{Vinculo: holder.contexto.Vinculo, Resultado: r}, i, nil
}

func (p *proveedorPreparacionBasesV3) ResolverIdentidadConvocatoria(ctx context.Context) (selauth.IdentidadRegistrada, error) {
	z, _, err := p.contexto(ctx)
	if err != nil {
		return selauth.IdentidadRegistrada{}, err
	}
	return selauth.IdentidadRegistrada{Vinculo: z.Vinculo, Resultado: z.Resultado}, nil
}

func (p *proveedorPreparacionBasesV3) ResolverContextoHTTP(r *http.Request) (selhttp.ContextoPreparacionBases, error) {
	if r == nil {
		return selhttp.ContextoPreparacionBases{}, bolsaports.ErrPreparacionBasesDenegada
	}
	z, i, err := p.contexto(r.Context())
	if err != nil {
		if errors.Is(err, bolsaports.ErrPreparacionBasesDenegada) && !errors.Is(err, bolsaports.ErrPreparacionBasesNoDisponible) {
			if p.registrarRechazo == nil || p.registrarRechazo(r) != nil {
				return selhttp.ContextoPreparacionBases{}, bolsaports.ErrPreparacionBasesNoDisponible
			}
		}
		return selhttp.ContextoPreparacionBases{}, err
	}
	c, err := core.GenerarReferenciaCorrelacionAutorizacionV2(r.Context(), seguridad.GeneradorReferenciasCriptograficas{})
	if err != nil {
		return selhttp.ContextoPreparacionBases{}, bolsaports.ErrPreparacionBasesNoDisponible
	}
	auditar := p.registrarEntrada
	return selhttp.ContextoPreparacionBases{Actor: z.Resultado.Contexto, Correlacion: c, Ambito: p.perfiles[i].ambito,
		RegistrarErrorEntrada: func(ctx context.Context, err error) error {
			if auditar == nil {
				return bolsaports.ErrPreparacionBasesNoDisponible
			}
			return auditar(ctx, z, i, c, err)
		}}, nil
}

func (p *proveedorPreparacionBasesV3) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s core.SolicitudAutorizacionLigadaV3,
	r core.ResultadoContextoActorRegistradoV2) (core.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	var d core.DecisionAutorizacionLigadaV3
	var confirmacion vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3
	z, i, err := p.contexto(ctx)
	if err != nil {
		return d, confirmacion, nil, errorEmisorPreparacionBasesV3(err)
	}
	perfil := p.perfiles[i]
	if !bytes.Equal(r.RepresentacionCanonica, z.Resultado.RepresentacionCanonica) || r.Validar() != nil ||
		validarSolicitudMaterialPreparacionBasesV3(s, z, perfil) != nil {
		return d, confirmacion, nil, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3
	}
	d, confirmacion, err = p.pdp.ExigirSolicitudLigadaV3(ctx, s, z.Resultado)
	if err != nil {
		if errorTecnicoPDPPreparacionBasesV3(ctx, err) {
			return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, bolsaports.ErrPreparacionBasesNoDisponible
		}
		if errors.Is(err, core.ErrAutorizacionDenegada) || errors.Is(err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3) {
			return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3
		}
		return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, bolsaports.ErrPreparacionBasesNoDisponible
	}
	campos := bolsaapp.CamposPreparacionBasesV3(perfil.accion)
	if d.ValidarPara(s) != nil || d.ExigirProyeccionPara(s, campos, nil) != nil {
		return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3
	}
	e, err := p.materiales[i].proveerMaterialConfirmacion(ctx, s, d, confirmacion, perfil.motivo, z.Resultado)
	if err != nil || !vecports.MaterialAtestadoLigadoV3(s, d, confirmacion, z.Resultado, perfil.motivo, e, perfil.audiencia) {
		return core.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, bolsaports.ErrPreparacionBasesNoDisponible
	}
	return d, confirmacion, exportadorPreparacionBasesV3{e}, nil
}

func errorTecnicoPDPPreparacionBasesV3(ctx context.Context, err error) bool {
	if ctx != nil && ctx.Err() != nil {
		return true
	}
	for _, causa := range []error{vecports.ErrFuenteAutorizacionNoDisponible, vecports.ErrRegistroConcesionAutorizacionLigadaV3NoDisponible,
		vecports.ErrRegistroDenegacionAutorizacionLigadaV3NoDisponible, errAutorizacionComunDesarrolloNoDisponible,
		bolsaports.ErrPreparacionBasesNoDisponible, context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, causa) {
			return true
		}
	}
	return false
}

func validarSolicitudMaterialPreparacionBasesV3(s core.SolicitudAutorizacionLigadaV3, z contextoSeguridadComunDesarrollo, p *perfilPreparacionBasesV3) error {
	d, err := s.Datos()
	if p == nil || err != nil || d.ReferenciaMotivo != p.motivo || d.Accion != p.accion ||
		d.Finalidad != bolsaports.FinalidadPreparacionBases || d.Recurso.ModuloID != "bolsa" ||
		d.Recurso.Tipo != bolsaports.TipoRecursoPreparacionBases || !prep.IdentificadorValido(d.Recurso.Referencia) || len(d.Recurso.Referencia) > 128 ||
		d.Recurso.Validar() != nil || len(d.Recurso.Atributos) != 1 || d.Correlacion.Validar() != nil {
		return bolsaports.ErrPreparacionBasesDenegada
	}
	ambitos := map[string]string{"organizacion_ref": p.ambito.OrganizacionRef()}
	if p.ambito.UnidadGestionRef() != "" {
		ambitos["unidad_gestion_ref"] = p.ambito.UnidadGestionRef()
	}
	h, err := hex.DecodeString(d.Recurso.Atributos["material_sha256"])
	a, errActual := z.Vinculo.Datos()
	b, errSolicitado := d.VinculoAutenticacionActor.Datos()
	if err != nil || len(h) != 32 || hex.EncodeToString(h) != d.Recurso.Atributos["material_sha256"] ||
		!reflect.DeepEqual(ambitos, d.Recurso.Ambitos) || errActual != nil || errSolicitado != nil || a != b ||
		d.VinculoAutenticacionActor.ValidarPara(z.Resultado) != nil {
		return bolsaports.ErrPreparacionBasesDenegada
	}
	return nil
}

func errorEmisorPreparacionBasesV3(err error) error {
	if errors.Is(err, bolsaports.ErrPreparacionBasesDenegada) {
		return vecports.ErrDenegacionExplicitaAutorizacionLigadaV3
	}
	return bolsaports.ErrPreparacionBasesNoDisponible
}

type exportadorPreparacionBasesV3 struct {
	vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
}

func (e exportadorPreparacionBasesV3) ExportarMaterialParaConsumidor() (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	if e.ValidarEstructura() != nil {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, bolsaports.ErrPreparacionBasesNoDisponible
	}
	return e.ExportacionMaterialConsumoAutorizacionAtestadaV3, nil
}

var _ selauth.ResolutorIdentidad = (*proveedorPreparacionBasesV3)(nil)
var _ selauth.EmisorV3 = (*proveedorPreparacionBasesV3)(nil)
