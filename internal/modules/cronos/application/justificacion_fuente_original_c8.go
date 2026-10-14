package application

import (
	"context"
	"encoding/json"
	"regexp"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
)

type FuenteSolicitudOriginalC8 struct {
	capturador ports.CapturadorLecturaOriginalC8
	lector     ports.LectorOriginalC8
}

var contextoOriginalC8 = regexp.MustCompile(`^rca_[-A-Za-z0-9_]{22,128}$`)

func NuevaFuenteSolicitudOriginalC8(c ports.CapturadorLecturaOriginalC8, l ports.LectorOriginalC8) (*FuenteSolicitudOriginalC8, error) {
	if dependenciaRemotaNula(c) || dependenciaRemotaNula(l) {
		return nil, ports.ErrJustificacionNoDisponible
	}
	return &FuenteSolicitudOriginalC8{c, l}, nil
}

func (f *FuenteSolicitudOriginalC8) PrepararOriginal(ctx context.Context, orden ports.OrdenOriginalC8,
	solicitudRef string, modo ports.ModoLecturaOriginalC8) (ports.SolicitudOriginalC8, error) {
	var cero ports.SolicitudOriginalC8
	if f == nil || dependenciaRemotaNula(f.capturador) || dependenciaRemotaNula(f.lector) || ctx == nil || ctx.Err() != nil ||
		!domain.SolicitudPermisoRefValida(solicitudRef) {
		return cero, ports.ErrJustificacionNoDisponible
	}
	actor, err := orden.ContextoActor()
	if err != nil {
		return cero, err
	}
	var captura ports.CapturaLecturaOriginalC8
	switch modo {
	case ports.LecturaOriginalPropia:
		captura, err = f.capturador.CapturarLecturaOriginalPropia(ctx, orden, solicitudRef)
	case ports.LecturaOriginalRRHH:
		captura, err = f.capturador.CapturarLecturaOriginalRRHH(ctx, orden, solicitudRef)
	default:
		return cero, ports.ErrJustificacionNoDisponible
	}
	if err != nil {
		return cero, err
	}
	var ordenCapturada map[string]json.RawMessage
	if json.Unmarshal(captura.Orden, &ordenCapturada) != nil {
		return cero, ports.ErrJustificacionNoDisponible
	}
	var accion, contextoRef string
	if json.Unmarshal(ordenCapturada["accion"], &accion) != nil ||
		json.Unmarshal(ordenCapturada["registro_contexto_ref"], &contextoRef) != nil ||
		!contextoOriginalC8.MatchString(contextoRef) {
		return cero, ports.ErrJustificacionNoDisponible
	}
	lectura, err := f.lector.ConsultarOriginal(ctx, captura, solicitudRef)
	if err != nil || lectura.SolicitudRef != solicitudRef || lectura.EstadoPermiso != domain.EstadoPermisoConcedido ||
		!lectura.PendienteJustificar || !lectura.JustificanteExigido {
		return cero, ports.ErrJustificacionNoDisponible
	}
	if (accion == "cronos.justificacion.propia.consultar") != (lectura.PersonaTitularRef == actor.PersonaRef) ||
		(modo == ports.LecturaOriginalPropia && accion != "cronos.justificacion.propia.consultar") ||
		(modo == ports.LecturaOriginalRRHH && accion != "cronos.justificacion.rrhh.consultar") {
		return cero, ports.ErrJustificacionNoDisponible
	}
	resultado := ports.SolicitudOriginalC8{SolicitudRef: lectura.SolicitudRef, EmpleadoRef: lectura.EmpleadoRef,
		PersonaTitularRef: lectura.PersonaTitularRef, EstadoPermiso: lectura.EstadoPermiso, VersionPermiso: lectura.VersionPermiso,
		PendienteJustificar: lectura.PendienteJustificar, JustificanteExigido: lectura.JustificanteExigido, ContextoRef: contextoRef}
	for i, h := range lectura.Historia {
		if h.Version != int64(i+1) || !reciboCronosOriginalC8.MatchString(h.ReciboRef) ||
			h.InstanteUTC.IsZero() || h.InstanteUTC.Nanosecond()%1000 != 0 ||
			(i == 0 && h.Estado != domain.JustificacionPendiente) {
			return cero, ports.ErrJustificacionNoDisponible
		}
		original := domain.OriginalJustificacionC8V2{DocumentoID: h.DocumentoID, DocumentoVersion: h.DocumentoVersion,
			HuellaSHA256Bytes: h.SHA256, ReciboDocumentoRef: h.ReciboDocumentoRef, RecursoRef: solicitudRef,
			PersonaTitularRef: lectura.PersonaTitularRef, PersonaAportanteRef: lectura.PersonaTitularRef,
			CustodioID: h.CustodioID, ModuloID: "cronos", TipoRecurso: "justificacion",
			Finalidad: "custodiar_justificacion_cronos", EstadoAdmision: "admitido"}
		if original.ValidarPara(solicitudRef, lectura.PersonaTitularRef) != nil ||
			(h.Estado != domain.JustificacionPendiente && h.Estado != domain.JustificacionAceptada && h.Estado != domain.JustificacionRechazada) {
			return cero, ports.ErrJustificacionNoDisponible
		}
		resultado.Actual = &domain.JustificacionOriginalC8V2{SolicitudRef: solicitudRef, EmpleadoRef: lectura.EmpleadoRef,
			Original: original, Version: h.Version, Estado: h.Estado, MotivoRef: h.MotivoRef}
	}
	return resultado, nil
}
