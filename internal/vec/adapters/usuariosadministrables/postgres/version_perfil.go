package postgres

import "vec-diputacion-granada/internal/vec/domain"

// Sólo versiones del rol fijo de Aplicación, sin fijar el número (AUT48).
func versionRolUsuariosAdmitida(v string) bool { return domain.VersionRolAplicacionAdmitida(v) }
