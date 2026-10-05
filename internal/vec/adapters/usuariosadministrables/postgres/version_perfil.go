package postgres

import "strings"

// Sólo versiones del rol fijo de Aplicación, sin fijar el número: la concesión
// exacta de usuarios la comprueban el emisor y la autoridad PostgreSQL (AUT48)
// sobre la versión que señala la asignación actual.
func versionRolUsuariosAdmitida(v string) bool { return VersionRolAplicacionAdmitida(v) }

// VersionRolAplicacionAdmitida acepta «rol:administracion_perfiles:v<N>» con
// N decimal positivo sin ceros a la izquierda. No concede nada por sí misma.
func VersionRolAplicacionAdmitida(v string) bool {
	n, ok := strings.CutPrefix(v, "rol:administracion_perfiles:v")
	if !ok || n == "" || len(n) > 9 || n[0] == '0' {
		return false
	}
	for _, c := range n {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
