package ports

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionReanudacionSeleccionLlamamiento      = "contratacion_temporal.llamamiento.reanudar_orden"
	TipoRecursoReanudacionSeleccionLlamamiento = "reanudacion_seleccion_contratacion_temporal"
	FinalidadReanudacionSeleccionLlamamiento   = "gestionar_contratacion_temporal"
	AudienciaReanudacionSeleccionLlamamiento   = "vec_contratacion_temporal.confirmar_alta_atestada.v1"
)

// NuevoRecursoReanudacionSeleccionLlamamiento no concede permiso. Liga la
// autorización nueva a la intención durable íntegra mediante su huella validada.
// El consumidor SQL contrasta además la solicitud exacta y la ventana caducada.
func NuevoRecursoReanudacionSeleccionLlamamiento(s SolicitudReservaEjecucionSeleccionLlamamiento) (dominiovec.RecursoAutorizable, error) {
	if s.Validar() != nil || s.VersionExpediente != 6 {
		return dominiovec.RecursoAutorizable{}, ErrEjecucionSeleccionLlamamientoInvalida
	}
	// El orden y los nombres son el canon compartido con CT 000055.
	datos := struct {
		OrganizacionRef   string `json:"organizacion_ref"`
		ExpedienteRef     string `json:"expediente_ref"`
		VersionExpediente uint64 `json:"version_expediente"`
		ClaveIdempotencia string `json:"clave_idempotencia"`
		HuellaSemantica   string `json:"huella_semantica"`
	}{s.OrganizacionRef, s.ExpedienteRef, s.VersionExpediente, s.ClaveIdempotencia, s.HuellaSemantica}
	b, err := json.Marshal(datos)
	if err != nil {
		return dominiovec.RecursoAutorizable{}, ErrEjecucionSeleccionLlamamientoInvalida
	}
	h := sha256.Sum256(b)
	return dominiovec.RecursoAutorizable{
		Referencia: s.ExpedienteRef, ModuloID: "contratacion_temporal",
		Tipo:      TipoRecursoReanudacionSeleccionLlamamiento,
		Ambitos:   map[string]string{"organizacion_ref": s.OrganizacionRef},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])},
	}, nil
}
