package administracionperfiles

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"vec-diputacion-granada/internal/vec/domain"
)

// RutaPublicacionCargoCompetencial recibe el material exacto de publicación de
// un cargo competencial de Personal o de su enlace con quien lo ocupa.
const RutaPublicacionCargoCompetencial = PrefijoV1 + "/personal/cargos-competenciales/publicacion"

// SolicitudEfectoNominalADMIN lleva la sesión ADMIN resuelta por la frontera y
// los bytes exactos del material. Nada de ella concede acceso.
type SolicitudEfectoNominalADMIN struct {
	Actor          domain.ContextoActor
	Evidencia      domain.EvidenciaSesionAdministracionPerfiles
	Instantanea    domain.InstantaneaAutorizacion
	Material       []byte
	CorrelacionRef string
}

// ReciboEfectoNominalADMIN es el recibo validado del módulo (en un reintento,
// el original) y la auditoría del consumo de este acceso.
type ReciboEfectoNominalADMIN struct {
	Cuerpo              json.RawMessage
	ConsumoAuditoriaRef string
}

// ServicioEfectoNominalADMIN lo compone vec-admin sobre la fachada del módulo
// dueño del efecto. Errores: domain.ErrAutorizacionDenegada (403),
// domain.ErrActoAdministracionPerfilesInvalido (400), ErrConflictoEstado (409);
// cualquier otro es indisponibilidad. La autoridad deja su propio intento.
type ServicioEfectoNominalADMIN interface {
	AplicarEfectoNominal(context.Context, SolicitudEfectoNominalADMIN) (ReciboEfectoNominalADMIN, error)
}

type efectoNominal struct {
	servicio ServicioEfectoNominalADMIN
	maximo   int
}

type solicitudEfectoNominalDTO struct {
	MaterialBase64 string `json:"material_base64"`
}

// ConEfectoNominal abre una ruta de efecto nominal en el handler de usuarios,
// con el límite exacto del material de ese efecto. Sólo admite rutas fijas de
// este paquete y una vez cada una. Se llama durante la composición.
func (h *Handler) ConEfectoNominal(ruta string, maximoMaterial int, s ServicioEfectoNominalADMIN) error {
	if h == nil || dependenciaNula(s) || !h.soloMetadatos || ruta != RutaPublicacionCargoCompetencial ||
		maximoMaterial < 2 || maximoMaterial > 4<<20 {
		return ErrConfiguracionIncompleta
	}
	if _, ok := h.efectos[ruta]; ok {
		return ErrConfiguracionIncompleta
	}
	if h.efectos == nil {
		h.efectos = map[string]efectoNominal{}
	}
	h.efectos[ruta], h.soloLectura = efectoNominal{servicio: s, maximo: maximoMaterial}, false
	return nil
}

func (h *Handler) postEfectoNominal(w http.ResponseWriter, r *http.Request, s SesionConfiable, e efectoNominal) {
	// Como en el gobierno del plan: las denegaciones de frontera usan el
	// destino «escribir»; el fallo de la autoridad deja su propio intento.
	const accion = "escribir"
	var dto solicitudEfectoNominalDTO
	if err := decodificarLimitado(w, r, &dto, int64(e.maximo)*4/3+1024); err != nil {
		estado := http.StatusBadRequest
		if errors.Is(err, errCuerpoExcesivo) {
			estado = http.StatusRequestEntityTooLarge
		}
		h.denegarActor(w, r, s, estado, "solicitud_invalida", accion, "")
		return
	}
	material, err := base64.StdEncoding.Strict().DecodeString(dto.MaterialBase64)
	if err != nil || len(material) < 2 || len(material) > e.maximo {
		h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", accion, "")
		return
	}
	defer clear(material)
	recibo, err := e.servicio.AplicarEfectoNominal(r.Context(), SolicitudEfectoNominalADMIN{
		Actor: s.Actor, Evidencia: s.Evidencia, Instantanea: s.InstantaneaAutorizacion, Material: material,
		CorrelacionRef: s.CorrelacionRef})
	if err == nil && (!json.Valid(recibo.Cuerpo) || !strings.HasPrefix(strings.TrimSpace(string(recibo.Cuerpo)), "{") ||
		recibo.ConsumoAuditoriaRef == "") {
		err = errors.New("recibo no válido")
	}
	if err != nil {
		estado, codigo := http.StatusServiceUnavailable, "servicio_no_disponible"
		switch {
		case errors.Is(err, domain.ErrAutorizacionDenegada):
			estado, codigo = http.StatusForbidden, "acceso_denegado"
		case errors.Is(err, domain.ErrActoAdministracionPerfilesInvalido):
			estado, codigo = http.StatusBadRequest, "solicitud_invalida"
		case errors.Is(err, ErrConflictoEstado):
			estado, codigo = http.StatusConflict, "conflicto_estado"
		}
		fallo(w, estado, codigo)
		return
	}
	jsonRespuesta(w, http.StatusOK, struct {
		Recibo              json.RawMessage `json:"recibo"`
		ConsumoAuditoriaRef string          `json:"consumo_auditoria_ref"`
	}{recibo.Cuerpo, recibo.ConsumoAuditoriaRef})
}
