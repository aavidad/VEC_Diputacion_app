package httpcopias

import (
	"net/http"
	"strings"

	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
)

// La clasificación describe el intento; nunca se utiliza para conceder permiso.
// No conserva URL, query ni contenido del formulario en la auditoría.
func accionAuditoria(r *http.Request, accion, recurso string) (string, string) {
	if r.Method == http.MethodGet {
		accion = string(p.Consultar)
		if recurso == "" {
			ref := strings.TrimPrefix(r.URL.Path, PrefijoV1+"/")
			switch ref {
			case "capacidades", "calendario", "retencion", "propuestas", "opciones-restauracion":
			default:
				if strings.HasPrefix(r.URL.Path, PrefijoV1+"/") && referencia(ref) {
					recurso = ref
				}
			}
		}
	} else if r.Method == http.MethodPost {
		switch r.URL.Path {
		case PrefijoV1 + "/lanzamientos":
			accion = string(p.Lanzar)
		case PrefijoV1 + "/calendario":
			accion = string(p.ConfigurarCalendario)
		case PrefijoV1 + "/retencion":
			accion = string(p.ConfigurarRetencion)
		case PrefijoV1 + "/propuestas":
			accion = string(p.Proponer)
		default:
			partes := strings.Split(strings.TrimPrefix(r.URL.Path, PrefijoV1+"/propuestas/"), "/")
			accion = ""
			if strings.HasPrefix(r.URL.Path, PrefijoV1+"/propuestas/") && len(partes) == 2 && referencia(partes[0]) {
				switch partes[1] {
				case "revision":
					accion, recurso = string(p.Revisar), partes[0]
				case "ejecucion":
					accion, recurso = string(p.Ejecutar), partes[0]
				}
			}
		}
	} else {
		accion = ""
	}
	if !referencia(recurso) {
		recurso = ""
	}
	return accion, recurso
}
