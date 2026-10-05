package postgres

// AUT45 conserva exactamente las dos concesiones de usuarios de Rol5.
func versionRolUsuariosAdmitida(v string) bool {
	return v == "rol:administracion_perfiles:v5" || v == "rol:administracion_perfiles:v6"
}
