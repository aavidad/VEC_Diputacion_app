package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	ErrSolicitudSubsanacionReparoInvalida    = errors.New("contratacion temporal: solicitud de subsanacion de reparo invalida")
	ErrSubsanacionReparoDenegada             = ports.ErrAutorizacionDenegada
	ErrResultadoSubsanacionReparoNoConfiable = errors.New("contratacion temporal: resultado de subsanacion de reparo no confiable")
)

type SolicitudRegistrarSubsanacionReparo struct {
	AutenticacionRef, SesionRef, PerfilRef, OrganizacionRef string
	ExpedienteRef                                           string
	VersionEsperada                                         uint64
	ClaveIdempotencia                                       string
	Observaciones                                           string
}

func (s SolicitudRegistrarSubsanacionReparo) Validar() error {
	if (ports.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: s.AutenticacionRef, SesionRef: s.SesionRef, PerfilRef: s.PerfilRef}).Validar() != nil ||
		!domain.ReferenciaOpacaValida(s.OrganizacionRef) || !domain.ReferenciaOpacaValida(s.ExpedienteRef) ||
		!ports.VersionOperacionAnalisisConIncrementoValida(s.VersionEsperada) ||
		!ports.ClaveIdempotenciaValida(s.ClaveIdempotencia) ||
		(domain.DatosSubsanacionReparo{RetornoRef: "retorno:pendiente", Observaciones: s.Observaciones}).Validar() != nil {
		return ErrSolicitudSubsanacionReparoInvalida
	}
	return nil
}

type ServicioSubsanacionReparos struct {
	contextos     ports.ResolutorContextoAutorizacionAltaV3
	ambitos       ports.SelladorAmbitoSubsanacionReparo
	huellas       ports.DerivadorHuellaSubsanacionReparo
	preparador    ports.PreparadorSubsanacionReparo
	confirmador   ports.ConfirmadorSubsanacionReparo
	politicas     ports.ResolutorPoliticaSubsanacionReparo
	correlaciones vp.GeneradorReferenciasAutorizacionV2
	autorizador   vp.AutorizadorSolicitudLigadaV3
	reloj         ports.Reloj
}

func NuevoServicioSubsanacionReparos(contextos ports.ResolutorContextoAutorizacionAltaV3, ambitos ports.SelladorAmbitoSubsanacionReparo, huellas ports.DerivadorHuellaSubsanacionReparo, preparador ports.PreparadorSubsanacionReparo, confirmador ports.ConfirmadorSubsanacionReparo, politicas ports.ResolutorPoliticaSubsanacionReparo, correlaciones vp.GeneradorReferenciasAutorizacionV2, autorizador vp.AutorizadorSolicitudLigadaV3, reloj ports.Reloj) (*ServicioSubsanacionReparos, error) {
	if dependenciaNula(contextos) || dependenciaNula(ambitos) || dependenciaNula(huellas) || dependenciaNula(preparador) || dependenciaNula(confirmador) || dependenciaNula(politicas) || dependenciaNula(correlaciones) || dependenciaNula(autorizador) || dependenciaNula(reloj) {
		return nil, ErrSolicitudSubsanacionReparoInvalida
	}
	return &ServicioSubsanacionReparos{contextos: contextos, ambitos: ambitos, huellas: huellas, preparador: preparador, confirmador: confirmador, politicas: politicas, correlaciones: correlaciones, autorizador: autorizador, reloj: reloj}, nil
}

func (s *ServicioSubsanacionReparos) RegistrarSubsanacionReparo(ctx context.Context, solicitud SolicitudRegistrarSubsanacionReparo) (ports.ReciboSubsanacionReparo, error) {
	if s == nil || ctx == nil || solicitud.Validar() != nil {
		return ports.ReciboSubsanacionReparo{}, ErrSolicitudSubsanacionReparoInvalida
	}
	contexto, err := s.contextos.ResolverContextoAutorizacionAltaV3(ctx, ports.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: solicitud.AutenticacionRef, SesionRef: solicitud.SesionRef, PerfilRef: solicitud.PerfilRef})
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, ErrSubsanacionReparoDenegada
	}
	if contexto.ValidarPara(ports.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: solicitud.AutenticacionRef, SesionRef: solicitud.SesionRef, PerfilRef: solicitud.PerfilRef}, instanteCanonico(s.reloj.Ahora())) != nil {
		return ports.ReciboSubsanacionReparo{}, ErrSubsanacionReparoDenegada
	}
	vinculo, err := contexto.Vinculo.Datos()
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, ErrSubsanacionReparoDenegada
	}
	material := ports.MaterialSubsanacionReparo{OrganizacionRef: solicitud.OrganizacionRef, ExpedienteRef: solicitud.ExpedienteRef, VersionEsperada: solicitud.VersionEsperada, ClaveIdempotencia: solicitud.ClaveIdempotencia, Observaciones: solicitud.Observaciones, ActorRef: vinculo.PrincipalID, PerfilRef: vinculo.PerfilActivoRef}
	if !material.Valido() {
		return ports.ReciboSubsanacionReparo{}, ErrSubsanacionReparoDenegada
	}
	ambitos, err := s.ambitos.SellarAmbitoSubsanacionReparo(ctx, ports.SolicitudSellarAmbitoIdempotencia{ClaveIdempotencia: material.ClaveIdempotencia, OrganizacionRef: material.OrganizacionRef, ActorRef: material.ActorRef, PerfilRef: material.PerfilRef})
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, ErrSubsanacionReparoDenegada
	}
	huellas, err := s.huellas.DerivarHuellaSubsanacionReparo(ctx, material)
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, ErrSubsanacionReparoDenegada
	}
	preparacion, err := s.preparador.PrepararSubsanacionReparo(ctx, ports.SolicitudPrepararSubsanacionReparo{Material: material, AmbitosHMAC: ambitos, HuellasPeticionHMAC: huellas})
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, err
	}
	versionPreparada := solicitud.VersionEsperada
	if preparacion.Confirmada {
		versionPreparada++
	}
	if preparacion.Material != material || preparacion.Expediente.Validar() != nil || preparacion.Expediente.Referencia != solicitud.ExpedienteRef ||
		preparacion.Expediente.OrganizacionRef != solicitud.OrganizacionRef ||
		preparacion.Expediente.Version != versionPreparada || preparacion.Expediente.Asignacion == nil ||
		preparacion.Expediente.Fiscalizacion == nil || preparacion.Expediente.Fiscalizacion.Retorno == nil ||
		preparacion.RetornoRef != preparacion.Expediente.Fiscalizacion.Retorno.RetornoRef ||
		!domain.ReferenciaOpacaValida(preparacion.RetornoRef) || !domain.ReferenciaOpacaValida(preparacion.ReciboRef) ||
		!domain.ReferenciaOpacaValida(preparacion.EventoRef) ||
		!ports.ColeccionesHMACContienenPar(ambitos, ports.DominioAmbitoIdempotenciaSubsanacionReparo, huellas, ports.DominioHuellaPeticionSubsanacionReparo, preparacion.AmbitoIdempotenciaHMAC, preparacion.HuellaPeticionHMAC) {
		return ports.ReciboSubsanacionReparo{}, ErrResultadoSubsanacionReparoNoConfiable
	}
	ahora := instanteCanonico(s.reloj.Ahora())
	ps := ports.SolicitudResolverPoliticaSubsanacionReparo{OrganizacionRef: solicitud.OrganizacionRef, ExpedienteRef: solicitud.ExpedienteRef, ActorRef: vinculo.PrincipalID, PerfilRef: vinculo.PerfilActivoRef, RetornoRef: preparacion.RetornoRef, VersionEsperada: solicitud.VersionEsperada, Instante: ahora}
	politica, err := s.politicas.ResolverPoliticaSubsanacionReparo(ctx, ps)
	if err != nil || !politica.ValidaPara(ps, ahora) {
		return ports.ReciboSubsanacionReparo{}, ErrSubsanacionReparoDenegada
	}
	corr, err := vd.GenerarReferenciaCorrelacionAutorizacionV2(ctx, s.correlaciones)
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, ErrSubsanacionReparoDenegada
	}
	h := sha256.Sum256([]byte(solicitud.Observaciones))
	sv3, err := vd.NuevaSolicitudAutorizacionLigadaV3(vd.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: contexto.Vinculo, ReferenciaMotivo: politica.MotivoAutorizacion, Accion: string(domain.AccionRegistrarSubsanacionReparo), Finalidad: ports.FinalidadRegistrarSubsanacionReparo, Correlacion: corr, Recurso: vd.RecursoAutorizable{Referencia: solicitud.ExpedienteRef, ModuloID: ports.ModuloContratacion, Tipo: ports.TipoRecursoSubsanacionReparo, Ambitos: map[string]string{"organizacion_ref": solicitud.OrganizacionRef, "expediente_ref": solicitud.ExpedienteRef, "fase_previa": string(preparacion.Expediente.FaseActual), "estado_previo": string(preparacion.Expediente.EstadoActual)}, Atributos: map[string]string{"version_expediente": strconv.FormatUint(solicitud.VersionEsperada, 10), "retorno_ref": preparacion.RetornoRef, "observaciones_huella_sha256": hex.EncodeToString(h[:]), "unidad_asignada_ref": preparacion.Expediente.Asignacion.UnidadRef, "responsable_asignado_ref": preparacion.Expediente.Asignacion.ResponsableRef, "politica_ref": politica.DefinicionRef, "politica_version": strconv.FormatUint(politica.DefinicionVersion, 10), "politica_huella_sha256": politica.DefinicionHuellaSHA256, "ambito_idempotencia_hmac": preparacion.AmbitoIdempotenciaHMAC, "huella_peticion_hmac": preparacion.HuellaPeticionHMAC}}})
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, ErrSubsanacionReparoDenegada
	}
	decision, confirmacion, err := s.autorizador.ExigirSolicitudLigadaV3(ctx, sv3, contexto.Resultado)
	efecto := instanteCanonico(s.reloj.Ahora())
	if err != nil || contexto.ValidarPara(ports.SolicitudResolverContextoAutorizacionAltaV3{AutenticacionRef: solicitud.AutenticacionRef, SesionRef: solicitud.SesionRef, PerfilRef: solicitud.PerfilRef}, efecto) != nil || !politica.ValidaPara(ps, efecto) || !autorizacionV3ValidaEn(sv3, decision, confirmacion, efecto) {
		return ports.ReciboSubsanacionReparo{}, ErrSubsanacionReparoDenegada
	}
	if preparacion.Confirmada {
		if preparacion.ReciboConfirmado == nil || !preparacion.ReciboConfirmado.ValidarPara(ports.OrdenConfirmarSubsanacionReparo{OrganizacionRef: solicitud.OrganizacionRef, Expediente: preparacion.Expediente, VersionAnterior: solicitud.VersionEsperada, ClaveIdempotencia: solicitud.ClaveIdempotencia, ActorRef: vinculo.PrincipalID, UnidadRef: preparacion.Expediente.Asignacion.UnidadRef, ReciboRef: preparacion.ReciboRef, EventoRef: preparacion.EventoRef, RegistradaEn: preparacion.ReciboConfirmado.RegistradaEn, Material: material, Preparacion: preparacion, Politica: politica}) {
			return ports.ReciboSubsanacionReparo{}, ErrResultadoSubsanacionReparoNoConfiable
		}
		return *preparacion.ReciboConfirmado, nil
	}
	actuacion := domain.DatosActuacion{AccionClave: domain.AccionRegistrarSubsanacionReparo, ActorRef: vinculo.PrincipalID, UnidadRef: preparacion.Expediente.Asignacion.UnidadRef, ReciboRef: preparacion.ReciboRef, RealizadaEn: efecto, FaseDestino: domain.FaseSubsanacionUnidad, EstadoDestino: domain.EstadoIncidencia, Observaciones: solicitud.Observaciones, RetornoRef: preparacion.RetornoRef}
	siguiente, err := preparacion.Expediente.RegistrarSubsanacionReparo(solicitud.VersionEsperada, domain.DatosSubsanacionReparo{RetornoRef: preparacion.RetornoRef, Observaciones: solicitud.Observaciones}, actuacion)
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, err
	}
	orden := ports.OrdenConfirmarSubsanacionReparo{OrganizacionRef: solicitud.OrganizacionRef, Expediente: siguiente, VersionAnterior: solicitud.VersionEsperada, ClaveIdempotencia: solicitud.ClaveIdempotencia, ActorRef: vinculo.PrincipalID, UnidadRef: actuacion.UnidadRef, ReciboRef: preparacion.ReciboRef, EventoRef: preparacion.EventoRef, RegistradaEn: efecto, Material: material, Preparacion: preparacion, Politica: politica, Evidencia: ports.EvidenciaAutorizacionSubsanacionReparo{Contexto: contexto, SolicitudV3: sv3, DecisionV3: decision, ConfirmacionV3: confirmacion}}
	recibo, err := s.confirmador.ConfirmarSubsanacionReparo(ctx, orden)
	if err != nil {
		return ports.ReciboSubsanacionReparo{}, err
	}
	if !recibo.ValidarPara(orden) {
		return ports.ReciboSubsanacionReparo{}, ErrResultadoSubsanacionReparoNoConfiable
	}
	return recibo, nil
}
