package administracionperfiles

import (
	"errors"
	"net/http"

	"vec-diputacion-granada/internal/vec/domain"
)

func (h *Handler) postLoteOrdinario(w http.ResponseWriter, r *http.Request, s SesionConfiable) {
	if h.organizacionLote == "" {
		h.denegarActor(w, r, s, http.StatusServiceUnavailable, "servicio_no_disponible", "aplicar_lote_ordinario", "")
		return
	}
	var servicio ServicioLotes = h.lotes
	if h.lotes == nil {
		servicio, _ = h.actos.(ServicioLotes)
	}
	if dependenciaNula(servicio) {
		h.denegarActor(w, r, s, http.StatusServiceUnavailable, "servicio_no_disponible", "aplicar_lote_ordinario", "")
		return
	}
	var dto SolicitudLote
	if err := decodificarLimitado(w, r, &dto, 64*1024); err != nil {
		estado := http.StatusBadRequest
		if errors.Is(err, errCuerpoExcesivo) {
			estado = http.StatusRequestEntityTooLarge
		}
		h.denegarActor(w, r, s, estado, "solicitud_invalida", "aplicar_lote_ordinario", "")
		return
	}
	if len(dto.Cambios) == 0 || len(dto.Cambios) > domain.MaximoCambiosLoteAdministracionPerfiles {
		h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "aplicar_lote_ordinario", "")
		return
	}
	solicitud := domain.SolicitudLoteAdministracionPerfiles{OperacionRef: dto.OperacionRef,
		OrganizacionRef: h.organizacionLote,
		Actor:           s.Actor, Evidencia: s.Evidencia, InstantaneaAutorizacion: s.InstantaneaAutorizacion,
		Motivo: dto.Motivo.dominio(), ReferenciaActo: dto.ReferenciaActo, CorrelacionRef: s.CorrelacionRef}
	for _, cambio := range dto.Cambios {
		solicitud.Cambios = append(solicitud.Cambios, domain.CambioPerfilAdministracion{
			Operacion:      domain.OperacionAdministracionPerfiles(cambio.Operacion),
			InicioVigencia: domain.InicioVigenciaLoteAdministracion(cambio.InicioVigencia),
			RolVersionRef:  cambio.RolVersionRef, Objetivo: cambio.Objetivo.dominio()})
	}
	_, huella, err := solicitud.CanonicoYHuella()
	if err != nil {
		h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "aplicar_lote_ordinario", "")
		return
	}
	solicitud.HuellaSolicitudSHA256 = huella
	for _, cambio := range solicitud.Cambios {
		if cambio.Operacion == domain.OperacionRevocarPerfil {
			continue
		}
		rol, err := h.catalogo.ResolverRolAdministrable(r.Context(), cambio.RolVersionRef)
		if err != nil {
			h.denegarLoteError(w, r, s, err)
			return
		}
		if rol.VersionRef != cambio.RolVersionRef || rol.Clase != domain.ClaseControlPerfilOrdinario ||
			(rol.UnidadRequerida && cambio.Objetivo.UnidadRef == "") {
			h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", "aplicar_lote_ordinario", "")
			return
		}
	}
	recibo, err := servicio.AplicarLoteOrdinario(r.Context(), solicitud)
	if err != nil {
		h.denegarLoteError(w, r, s, err)
		return
	}
	if recibo.ValidarPara(solicitud) != nil {
		h.denegarActor(w, r, s, http.StatusServiceUnavailable, "servicio_no_disponible", "aplicar_lote_ordinario", "")
		return
	}
	dtoRecibo := ReciboLote{OperacionRef: recibo.OperacionRef, ActoRef: recibo.ActoRef, ReciboRef: recibo.ReciboRef,
		AuditoriaRef: recibo.AuditoriaRef, HuellaSolicitudSHA256: recibo.HuellaSolicitudSHA256,
		FuentesSHA256: recibo.FuentesSHA256, ConfirmadoEn: recibo.ConfirmadoEn,
		Cambios: make([]Recibo, 0, len(recibo.Cambios))}
	for _, cambio := range recibo.Cambios {
		dtoRecibo.Cambios = append(dtoRecibo.Cambios, reciboDTO(cambio))
	}
	for _, inicio := range recibo.Inicios {
		dto := InicioEfectivoLote{Modo: string(inicio.Modo)}
		if !inicio.VigenteDesde.IsZero() {
			desde := inicio.VigenteDesde
			dto.VigenteDesde = &desde
		}
		dtoRecibo.Inicios = append(dtoRecibo.Inicios, dto)
	}
	jsonRespuesta(w, http.StatusOK, struct {
		Recibo ReciboLote `json:"recibo"`
	}{dtoRecibo})
}

func (h *Handler) denegarLoteError(w http.ResponseWriter, r *http.Request, s SesionConfiable, err error) {
	h.denegarErrorLote(w, r, s, err, "aplicar_lote_ordinario", "")
}

// denegarErrorLote traduce igual los errores de aplicar y de preparar. Un
// administrador sin la concesión del lote (ErrControlAdministracionPerfilesInvalido)
// recibe 403 en los dos casos.
func (h *Handler) denegarErrorLote(w http.ResponseWriter, r *http.Request, s SesionConfiable, err error, accion, recurso string) {
	estado, codigo := http.StatusServiceUnavailable, "servicio_no_disponible"
	switch {
	case errors.Is(err, ErrAutenticacionRequerida):
		estado, codigo = http.StatusUnauthorized, "autenticacion_requerida"
	case errors.Is(err, ErrAccesoDenegado), errors.Is(err, domain.ErrAutorizacionDenegada),
		errors.Is(err, domain.ErrControlAdministracionPerfilesInvalido):
		estado, codigo = http.StatusForbidden, "acceso_denegado"
	case errors.Is(err, ErrConflictoEstado):
		estado, codigo = http.StatusConflict, "conflicto_estado"
	case errors.Is(err, ErrRecursoNoEncontrado):
		estado, codigo = http.StatusNotFound, "recurso_no_encontrado"
	}
	h.denegarActor(w, r, s, estado, codigo, accion, recurso)
}
