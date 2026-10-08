package inscripcion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// RecursoLectura fija la referencia opaca de auditoría y permiso para el
// filtro completo de una petición. Preparador y aplicación usan esta función;
// la lista y el total se consultan bajo la misma captura.
func RecursoLectura(accion, personaRef, idioma string, filtro Filtro, ref string) (string, error) {
	if !referenciaOpaca.MatchString(personaRef) || (idioma != "es" && idioma != "en") {
		return "", ErrSolicitudInvalida
	}
	switch accion {
	case AccionDetalleAbierta:
		if !convocatoriaRefValida(ref) || filtro != (Filtro{}) {
			return "", ErrSolicitudInvalida
		}
		return ref, nil
	case AccionDetallePropia, AccionDetalleRRHH:
		if !solicitudRefValida(ref) || filtro != (Filtro{}) {
			return "", ErrSolicitudInvalida
		}
		return ref, nil
	case AccionListarAbiertas, AccionListarPropias, AccionListarRRHH, AccionMotivosRRHH:
		if accion == AccionMotivosRRHH {
			if ref != "admitir" && ref != "rechazar" {
				return "", ErrSolicitudInvalida
			}
		} else if ref != "" || filtro.Validar() != nil ||
			(accion == AccionListarAbiertas && filtro.Cursor != "" && !convocatoriaRefValida(filtro.Cursor)) ||
			(accion != AccionListarAbiertas && filtro.Cursor != "" && !solicitudRefValida(filtro.Cursor)) {
			return "", ErrSolicitudInvalida
		}
	default:
		return "", ErrSolicitudInvalida
	}
	contenido, err := json.Marshal(struct {
		Accion          string `json:"accion"`
		PersonaRef      string `json:"persona_ref"`
		Idioma          string `json:"idioma"`
		Estado          string `json:"estado"`
		ConvocatoriaRef string `json:"convocatoria_ref"`
		Limite          int    `json:"limite"`
		Cursor          string `json:"cursor"`
		Ref             string `json:"ref"`
	}{accion, personaRef, idioma, filtro.Estado, filtro.ConvocatoriaRef, filtro.Limite, filtro.Cursor, ref})
	if err != nil {
		return "", ErrSolicitudInvalida
	}
	huella := sha256.Sum256(contenido)
	prefijo := ""
	switch accion {
	case AccionListarAbiertas:
		prefijo = "inscripciones_abiertas_"
	case AccionListarPropias:
		prefijo = "inscripciones_propias_"
	case AccionListarRRHH:
		prefijo = "inscripciones_rrhh_"
	case AccionMotivosRRHH:
		prefijo = "motivos_inscripcion_"
	}
	return prefijo + hex.EncodeToString(huella[:]), nil
}
