package administracion

import "strings"

// hostAdmin es la forma cerrada de VEC_ADMIN_HOST. Admite «nombre» (puerto
// 443) o «nombre:puerto». La autoridad es lo único que se acepta en la
// cabecera Host y lo que forma el origen HTTPS: el navegador omite el puerto
// por defecto, así que con 443 la autoridad es el nombre solo y cualquier
// variante con puerto (también «:443») se rechaza. El nombre, sin puerto, es
// lo que se contrasta con host_admin de la política de certificado (IS15).
type hostAdmin struct {
	nombre, autoridad string
}

// analizarHostAdmin rechaza mayúsculas, espacios, corchetes, punto final,
// puertos fuera de 1..65535, ceros a la izquierda y cualquier otro adorno.
func analizarHostAdmin(valor string) (hostAdmin, bool) {
	nombre, puerto, conPuerto := strings.Cut(valor, ":")
	if !nombreHostAdminValido(nombre) {
		return hostAdmin{}, false
	}
	if !conPuerto || puerto == "443" {
		return hostAdmin{nombre: nombre, autoridad: nombre}, true
	}
	if !puertoCanonico(puerto) {
		return hostAdmin{}, false
	}
	return hostAdmin{nombre: nombre, autoridad: nombre + ":" + puerto}, true
}

// puertoCanonico admite 1..65535 en decimal, sin signo ni ceros a la izquierda.
func puertoCanonico(puerto string) bool {
	if puerto == "" || len(puerto) > 5 || puerto[0] == '0' {
		return false
	}
	n := 0
	for _, c := range puerto {
		if c < '0' || c > '9' {
			return false
		}
		n = n*10 + int(c-'0')
	}
	return n <= 65535
}

func (h hostAdmin) origen() string { return "https://" + h.autoridad }

// nombreHostAdminValido sigue la regla de host_admin de la migración 000016 de
// identidad_sesiones_v1 (etiquetas DNS en minúscula y al menos un punto), más
// estricta que el CHECK de 000009.
func nombreHostAdminValido(nombre string) bool {
	if len(nombre) < 4 || len(nombre) > 253 || !strings.Contains(nombre, ".") {
		return false
	}
	for _, etiqueta := range strings.Split(nombre, ".") {
		if etiqueta == "" || len(etiqueta) > 63 || etiqueta[0] == '-' || etiqueta[len(etiqueta)-1] == '-' {
			return false
		}
		for _, c := range etiqueta {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}
