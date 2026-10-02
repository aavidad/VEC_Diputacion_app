package application

import (
	"context"
	"errors"
	"strconv"

	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func finalidadAudiencia(accion string) (string, string) {
	switch accion {
	case accionDeclarar:
		return "declaracion_hecho_propio", "vec_meritos.hecho.declarar.v1"
	case accionVerificar:
		return "verificacion_hecho_merito", "vec_meritos.hecho.verificar.v1"
	case accionRechazar:
		return "revision_hecho_merito", "vec_meritos.hecho.rechazar.v1"
	case accionRectificar:
		return "rectificacion_hecho_propio", "vec_meritos.hecho.rectificar.v1"
	default:
		return "", ""
	}
}

func (s *Servicio) autorizar(ctx context.Context, solicitud Solicitud, orden ports.OrdenOperacion) (ports.AutorizacionOperacion, error) {
	finalidad, audiencia := finalidadAudiencia(orden.Accion)
	auth, err := vec.NuevaSolicitudAutorizacionLigadaV3(vec.DatosSolicitudAutorizacionLigadaV3{
		VinculoAutenticacionActor: solicitud.Vinculo, ReferenciaMotivo: solicitud.Motivo,
		Accion: orden.Accion, Finalidad: finalidad, Correlacion: solicitud.Correlacion,
		Recurso: vec.RecursoAutorizable{Referencia: orden.Hecho.Referencia, ModuloID: "meritos", Tipo: "hecho",
			Ambitos: map[string]string{"persona_ref": orden.Hecho.PersonaRef, "version_esperada": strconv.Itoa(orden.VersionEsperada), "huella_comando_sha256": orden.HuellaComando}},
	})
	if err != nil {
		return ports.AutorizacionOperacion{}, vec.ErrAutorizacionDenegada
	}
	decision, confirmacion, exportador, err := s.autorizador.EmitirMaterialAutorizacionAtestadaV3(ctx, auth, solicitud.Contexto)
	if err != nil || nulo(exportador) {
		return ports.AutorizacionOperacion{}, errorAutoridad(err)
	}
	if decision.ExigirProyeccionPara(auth, []string{"hecho", "declarante_ref", "version", "recibo"}, []string{"auditar"}) != nil {
		return ports.AutorizacionOperacion{}, vec.ErrAutorizacionDenegada
	}
	material, err := exportador.ExportarMaterialParaConsumidor()
	if err != nil || !vecports.MaterialAtestadoLigadoV3(auth, decision, confirmacion, solicitud.Contexto, solicitud.Motivo, material, audiencia) {
		return ports.AutorizacionOperacion{}, ports.ErrRegistroNoDisponible
	}
	resumen := material.ResumenCapacidad()
	// La emisión y su registro pueden ocurrir después del inicio de la petición.
	// Comprobar con ese instante anterior rechazaría material recién emitido.
	ahora := s.reloj.Ahora().UTC()
	if ahora.IsZero() || ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return ports.AutorizacionOperacion{}, vec.ErrAutorizacionDenegada
	}
	contexto, err := solicitud.Contexto.Clonar()
	if err != nil {
		return ports.AutorizacionOperacion{}, ports.ErrRegistroNoDisponible
	}
	return ports.AutorizacionOperacion{Contexto: contexto, Solicitud: auth, Decision: decision, Confirmacion: confirmacion, Material: material}, nil
}

// EspecificacionAutorizacion mantiene el contrato nominal común a servicio y
// adaptador. Publicar o conceder estas acciones pertenece al gobierno central.
func EspecificacionAutorizacion(accion string) (string, string) { return finalidadAudiencia(accion) }

func errorAutoridad(err error) error {
	if errors.Is(err, vec.ErrAutorizacionDenegada) {
		return vec.ErrAutorizacionDenegada
	}
	return ports.ErrRegistroNoDisponible
}
