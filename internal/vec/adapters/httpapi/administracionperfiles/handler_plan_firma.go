package administracionperfiles

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

// RutaGobiernoPlanFirma recibe una operación del kit del plan nominal de firma
// de Contratación temporal (crear, actualizar, publicar o retirar).
const RutaGobiernoPlanFirma = PrefijoV1 + "/plan-firma/gobierno"

// maximoMaterialGobiernoPlanFirma es el límite del material exacto (4 MiB,
// como AD201); el cuerpo lo lleva en base64.
const maximoMaterialGobiernoPlanFirma = 4 << 20

// SolicitudGobiernoPlanFirmaADMIN lleva la sesión ADMIN resuelta por la
// frontera y los bytes exactos del material. Nada de ella concede acceso.
type SolicitudGobiernoPlanFirmaADMIN struct {
	Actor          domain.ContextoActor
	Evidencia      domain.EvidenciaSesionAdministracionPerfiles
	Instantanea    domain.InstantaneaAutorizacion
	Material       []byte
	CorrelacionRef string
}

// ReciboGobiernoPlanFirmaADMIN es el recibo del efecto (o el original en un
// reintento) y la auditoría del consumo de este acceso.
type ReciboGobiernoPlanFirmaADMIN struct {
	Accion              string    `json:"accion"`
	CatalogoRef         string    `json:"catalogo_ref"`
	ReciboRef           string    `json:"recibo_ref"`
	Estado              string    `json:"estado"`
	Revision            int64     `json:"revision"`
	PublicacionSHA256   string    `json:"publicacion_sha256,omitempty"`
	ConfirmadoEn        time.Time `json:"confirmado_en"`
	AuditoriaRef        string    `json:"auditoria_ref"`
	ConsumoAuditoriaRef string    `json:"consumo_auditoria_ref"`
}

// ServicioGobiernoPlanFirmaADMIN lo compone vec-admin sobre la autoridad de
// Contratación temporal. Errores: domain.ErrAutorizacionDenegada (403),
// domain.ErrActoAdministracionPerfilesInvalido (400), ErrConflictoEstado (409);
// cualquier otro es indisponibilidad.
type ServicioGobiernoPlanFirmaADMIN interface {
	GobernarPlanFirma(context.Context, SolicitudGobiernoPlanFirmaADMIN) (ReciboGobiernoPlanFirmaADMIN, error)
}

type solicitudGobiernoPlanFirmaDTO struct {
	MaterialBase64 string `json:"material_base64"`
}

// ConGobiernoPlanFirma abre la ruta del gobierno del plan en el handler de
// usuarios (con o sin lote). Es la única escritura que añade: sin actos ni
// lote, cualquier otro POST no existe. Se llama durante la composición.
func (h *Handler) ConGobiernoPlanFirma(s ServicioGobiernoPlanFirmaADMIN) error {
	if h == nil || dependenciaNula(s) || !h.soloMetadatos || h.gobiernoPlan != nil {
		return ErrConfiguracionIncompleta
	}
	h.gobiernoPlan, h.soloLectura = s, false
	return nil
}

func (h *Handler) postGobiernoPlanFirma(w http.ResponseWriter, r *http.Request, s SesionConfiable) {
	const accion = "gobernar_plan_firma"
	if dependenciaNula(h.gobiernoPlan) {
		h.denegarActor(w, r, s, http.StatusNotFound, "recurso_no_encontrado", accion, "")
		return
	}
	var dto solicitudGobiernoPlanFirmaDTO
	if err := decodificarLimitado(w, r, &dto, maximoMaterialGobiernoPlanFirma*4/3+1024); err != nil {
		estado := http.StatusBadRequest
		if errors.Is(err, errCuerpoExcesivo) {
			estado = http.StatusRequestEntityTooLarge
		}
		h.denegarActor(w, r, s, estado, "solicitud_invalida", accion, "")
		return
	}
	material, err := base64.StdEncoding.Strict().DecodeString(dto.MaterialBase64)
	if err != nil || len(material) < 2 || len(material) > maximoMaterialGobiernoPlanFirma {
		h.denegarActor(w, r, s, http.StatusBadRequest, "solicitud_invalida", accion, "")
		return
	}
	defer clear(material)
	recibo, err := h.gobiernoPlan.GobernarPlanFirma(r.Context(), SolicitudGobiernoPlanFirmaADMIN{
		Actor: s.Actor, Evidencia: s.Evidencia, Instantanea: s.InstantaneaAutorizacion, Material: material,
		CorrelacionRef: s.CorrelacionRef})
	if err != nil {
		// La autoridad ya dejó el intento común con su resultado; aquí sólo se
		// traduce, sin otro registro de frontera.
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
		Recibo ReciboGobiernoPlanFirmaADMIN `json:"recibo"`
	}{recibo})
}
