package telemetria

import "strings"

const (
	// marcadorValor sustituye a cualquier tramo del camino que pueda ser un
	// valor (referencias, números, fechas, nombres de fichero con versión).
	marcadorValor = "{valor}"
	// rutaNoEncontrada se usa para un 404 sin patrón: el camino lo eligió
	// quien hizo la petición y no se copia al registro.
	rutaNoEncontrada = "{no_encontrada}"
	// rutaSinPlantilla se usa para otros 4xx sin patrón, por el mismo motivo.
	rutaSinPlantilla = "{sin_plantilla}"
	maxTramosRuta    = 16
	maxTramoRuta     = 40
	maxLongitudRuta  = 200
)

// plantillaDePatron convierte un patrón de http.ServeMux en la plantilla de
// la ruta: quita el método y el host, que se registran aparte o no aplican.
// Un patrón de subárbol ("/api/vec/") no identifica la ruta y devuelve "";
// entonces manda el camino normalizado.
func plantillaDePatron(patron string) string {
	patron = strings.TrimSpace(patron)
	if i := strings.IndexByte(patron, ' '); i >= 0 {
		patron = strings.TrimSpace(patron[i+1:])
	}
	if i := strings.IndexByte(patron, '/'); i > 0 {
		patron = patron[i:]
	}
	if patron == "" || patron[0] != '/' || len(patron) > maxLongitudRuta {
		return ""
	}
	exacto := strings.HasSuffix(patron, "{$}")
	patron = strings.TrimSuffix(patron, "{$}")
	if !exacto && strings.HasSuffix(patron, "/") {
		// Subárbol: "/x/" atiende cualquier camino por debajo.
		return ""
	}
	for i := 0; i < len(patron); i++ {
		if c := patron[i]; c < 0x21 || c > 0x7e || c == '"' || c == '\\' {
			return ""
		}
	}
	return patron
}

// normalizarCamino conserva solo los tramos que tienen forma de nombre de
// ruta (minúsculas, guion, guion bajo y punto) y sustituye los demás por
// {valor}. Así ningún identificador, número o dato escrito por la persona
// llega al registro, aunque el enrutador no haya anotado su plantilla.
func normalizarCamino(camino string) string {
	if camino == "" || camino[0] != '/' {
		return "/"
	}
	var b strings.Builder
	b.Grow(min(len(camino), maxLongitudRuta))
	tramos := strings.Split(camino[1:], "/")
	for i, tramo := range tramos {
		if i >= maxTramosRuta {
			b.WriteString("/" + marcadorValor)
			break
		}
		b.WriteByte('/')
		if esTramoDeRuta(tramo) {
			b.WriteString(tramo)
		} else {
			b.WriteString(marcadorValor)
		}
		if b.Len() > maxLongitudRuta {
			break
		}
	}
	salida := b.String()
	if len(salida) > maxLongitudRuta {
		salida = salida[:maxLongitudRuta]
	}
	return salida
}

func esTramoDeRuta(tramo string) bool {
	if tramo == "" {
		return true
	}
	if len(tramo) > maxTramoRuta {
		return false
	}
	if esVersionDeRuta(tramo) {
		return true
	}
	for i := 0; i < len(tramo); i++ {
		c := tramo[i]
		if (c < 'a' || c > 'z') && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	return true
}

// esVersionDeRuta admite tramos "v1" a "v99", habituales en las API.
func esVersionDeRuta(tramo string) bool {
	if len(tramo) < 2 || len(tramo) > 3 || tramo[0] != 'v' {
		return false
	}
	for i := 1; i < len(tramo); i++ {
		if tramo[i] < '0' || tramo[i] > '9' {
			return false
		}
	}
	return true
}
