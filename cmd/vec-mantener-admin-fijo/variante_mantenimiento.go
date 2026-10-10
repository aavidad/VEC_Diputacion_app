package main

type varianteMantenimientoPrivada struct {
	QuerySQL, Esquema, OrigenRef, DestinoRef, OrigenSufijo, DestinoSufijo string
	VersionRolDestino                                                     int
}

// Sólo el plan privado aprobado elige una de las operaciones conocidas:
// 1 = AUT42 (Rol4→Rol5), 2 = AUT45 (Rol5→Rol6, lote), 3 = AUT51 (Rol6→Rol7,
// gobierno del plan nominal de firma), 4 = AUT59 (Rol7→Rol8, gobierno de
// definiciones), 5 = AUT64 (Rol8→Rol9, versión de Bolsa).
// Ninguna variante instala SQL ni crea la aprobación externa.
func varianteMantenimiento(v uint64) (varianteMantenimientoPrivada, bool) {
	switch v {
	case 1:
		return varianteMantenimientoPrivada{"SELECT vec_autorizacion.mantener_version_perfil_fijo_admin_v1($1::text,$2::text)", "vec.admin.mantenimiento-fijo.v1", "rol:administracion_perfiles:v4", "rol:administracion_perfiles:v5", ":v1", ":v2", 5}, true
	case 2:
		return varianteMantenimientoPrivada{"SELECT vec_autorizacion.mantener_version_perfil_fijo_lote_admin_v1($1::text,$2::text)", "vec.admin.mantenimiento-fijo.v2", "rol:administracion_perfiles:v5", "rol:administracion_perfiles:v6", ":v2", ":v3", 6}, true
	case 3:
		return varianteMantenimientoPrivada{"SELECT vec_autorizacion.mantener_version_perfil_fijo_plan_firma_admin_v1($1::text,$2::text)", "vec.admin.mantenimiento-fijo.v3", "rol:administracion_perfiles:v6", "rol:administracion_perfiles:v7", ":v3", ":v4", 7}, true
	case 4:
		return varianteMantenimientoPrivada{"SELECT vec_autorizacion.mantener_version_perfil_fijo_gobierno_definiciones_admin_v1($1::text,$2::text)", "vec.admin.mantenimiento-fijo.v4", "rol:administracion_perfiles:v7", "rol:administracion_perfiles:v8", ":v4", ":v5", 8}, true
	case 5:
		return varianteMantenimientoPrivada{"SELECT vec_autorizacion.mantener_version_perfil_fijo_version_bolsa_admin_v1($1::text,$2::text)", "vec.admin.mantenimiento-fijo.v5", "rol:administracion_perfiles:v8", "rol:administracion_perfiles:v9", ":v5", ":v6", 9}, true
	default:
		return varianteMantenimientoPrivada{}, false
	}
}
func versionMantenimientoAdmitida(v uint64) bool { _, ok := varianteMantenimiento(v); return ok }
