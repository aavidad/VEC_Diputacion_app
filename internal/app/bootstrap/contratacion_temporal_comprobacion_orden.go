package bootstrap

import (
	"encoding/json"
	"strings"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

// La misma orden de origen lleva el corte fiscalizado que B137 coteja con
// CT233. No crea otra lectura ni modifica la captura compartida del permiso.
func ordenOrigenCTConContextoComprobacion(
	original json.RawMessage, d preparacionLlamamientoDesarrollo, ambitoEsperado string,
) (json.RawMessage, error) {
	vacia := json.RawMessage(nil)
	e := d.expediente.Fiscalizado
	if e.Analisis == nil || e.Asignacion == nil || e.Version == 0 ||
		e.OrganizacionRef == "" || e.Referencia == "" || d.necesidad == "" ||
		d.categoria != e.Analisis.CategoriaRef || d.unidad != e.Asignacion.UnidadRef ||
		d.necesidad != referenciaPuenteLlamamientoDesarrollo("necesidad", e.OrganizacionRef, e.Referencia, "6") ||
		string(e.Analisis.ModalidadClave) == "" || string(e.Analisis.CausaClave) == "" ||
		validarClavesJSONUnicas(original) != nil {
		return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	ambito, err := ambitoOrdenOrigenConstituidoCT(original, d.unidad)
	if err != nil || ambito != ambitoEsperado {
		return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	var campos map[string]json.RawMessage
	if json.Unmarshal(original, &campos) != nil || campos == nil {
		return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	for clave := range campos {
		if strings.EqualFold(clave, "comprobacion_contexto") {
			return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
		}
	}
	contexto, err := json.Marshal(struct {
		AmbitoRef         string `json:"ambito_ref"`
		CategoriaRef      string `json:"categoria_ref"`
		CausaOfrecida     string `json:"causa_ofrecida"`
		ExpedienteRef     string `json:"expediente_ref"`
		ModalidadOfrecida string `json:"modalidad_ofrecida"`
		NecesidadRef      string `json:"necesidad_ref"`
		OrganizacionRef   string `json:"organizacion_ref"`
		UnidadRef         string `json:"unidad_ref"`
		VersionExpediente uint64 `json:"version_expediente"`
	}{ambito, d.categoria, string(e.Analisis.CausaClave), e.Referencia,
		string(e.Analisis.ModalidadClave), d.necesidad, e.OrganizacionRef, d.unidad, e.Version})
	if err != nil {
		return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	campos["comprobacion_contexto"] = contexto
	nueva, err := json.Marshal(campos)
	if err != nil {
		return vacia, ports.ErrOrigenLlamamientoCTNoDisponible
	}
	return nueva, nil
}
