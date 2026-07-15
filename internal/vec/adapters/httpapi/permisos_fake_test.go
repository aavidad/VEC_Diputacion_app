package httpapi

import (
	"testing"

	adminmodule "vec-diputacion-granada/internal/modules/administracion"
	bolsamodule "vec-diputacion-granada/internal/modules/bolsa"
	cronosmodule "vec-diputacion-granada/internal/modules/cronos"
	dietasmodule "vec-diputacion-granada/internal/modules/dietas"
	personalmodule "vec-diputacion-granada/internal/modules/personal"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestPermissionsForRolesMatrizPositivaExacta(t *testing.T) {
	carcasa := []string{"vec.session.read", "vec.modules.read", "vec.menu.read"}
	ciudadano := []string{
		"vec.session.read",
		"vec.menu.read",
		bolsamodule.PermissionRead,
		bolsamodule.PermissionDocument,
		bolsamodule.PermissionClaim,
		bolsamodule.PermissionNotification,
	}
	administradorTecnico := append(append([]string(nil), carcasa...),
		adminmodule.PermissionRolesManage,
		adminmodule.PermissionCatalogsManage,
		adminmodule.PermissionIntegrationsManage,
		adminmodule.PermissionMonitoringRead,
	)

	casos := map[string][]string{
		"ciudadano":        ciudadano,
		"candidate":        ciudadano,
		"administrador":    administradorTecnico,
		"system_admin":     administradorTecnico,
		"jefatura_rrhh":    carcasa,
		"tecnico_rrhh":     carcasa,
		"validator_l2":     carcasa,
		"administrativo":   carcasa,
		"validator_l1":     carcasa,
		"personal_interno": carcasa,
		"jefe_servicio":    carcasa,
		"jefe_seccion":     carcasa,
	}

	for rol, esperados := range casos {
		rol, esperados := rol, esperados
		t.Run(rol, func(t *testing.T) {
			comprobarPermisosExactos(t, permissionsForRoles([]string{rol}), esperados)
		})
	}
}

func TestPermissionsForRolesMatrizNegativaRolPorPermisoCritico(t *testing.T) {
	permisosCriticos := permisosDeManifiestos(
		personalmodule.Manifest(),
		cronosmodule.Manifest(),
		dietasmodule.Manifest(),
		bolsamodule.Manifest(),
		adminmodule.Manifest(),
	)
	permisosCriticos = append(permisosCriticos, "vec.workspace.read")
	permitidosPorRol := map[string]map[string]bool{
		"ciudadano": {
			bolsamodule.PermissionRead:         true,
			bolsamodule.PermissionDocument:     true,
			bolsamodule.PermissionClaim:        true,
			bolsamodule.PermissionNotification: true,
		},
		"candidate": {
			bolsamodule.PermissionRead:         true,
			bolsamodule.PermissionDocument:     true,
			bolsamodule.PermissionClaim:        true,
			bolsamodule.PermissionNotification: true,
		},
		"administrador": {
			adminmodule.PermissionRolesManage:        true,
			adminmodule.PermissionCatalogsManage:     true,
			adminmodule.PermissionIntegrationsManage: true,
			adminmodule.PermissionMonitoringRead:     true,
		},
		"system_admin": {
			adminmodule.PermissionRolesManage:        true,
			adminmodule.PermissionCatalogsManage:     true,
			adminmodule.PermissionIntegrationsManage: true,
			adminmodule.PermissionMonitoringRead:     true,
		},
		"jefatura_rrhh":    {},
		"tecnico_rrhh":     {},
		"validator_l2":     {},
		"administrativo":   {},
		"validator_l1":     {},
		"personal_interno": {},
		"jefe_servicio":    {},
		"jefe_seccion":     {},
	}

	for rol, permitidos := range permitidosPorRol {
		concedidos := conjuntoPermisos(t, permissionsForRoles([]string{rol}))
		for _, permiso := range permisosCriticos {
			_, obtenido := concedidos[permiso]
			esperado := permitidos[permiso]
			if obtenido != esperado {
				t.Fatalf("rol %q, permiso critico %q: concedido=%t, esperado=%t", rol, permiso, obtenido, esperado)
			}
		}
	}
}

func TestPermissionsForRolesAmbiguedadOValorNoCanonicoDeniegaTodo(t *testing.T) {
	casos := map[string][]string{
		"ausente":               nil,
		"lista vacia":           {},
		"desconocido":           {"superadministrador"},
		"espacios":              {" administrador "},
		"mayusculas":            {"ADMINISTRADOR"},
		"dos perfiles":          {"administrador", "tecnico_rrhh"},
		"perfil repetido":       {"administrador", "administrador"},
		"canonico mas alias":    {"tecnico_rrhh", "validator_l2"},
		"ciudadano mas interno": {"ciudadano", "personal_interno"},
	}

	for nombre, roles := range casos {
		t.Run(nombre, func(t *testing.T) {
			if obtenidos := permissionsForRoles(roles); len(obtenidos) != 0 {
				t.Fatalf("permissionsForRoles(%#v) = %#v; la ambiguedad debe denegar todo", roles, obtenidos)
			}
		})
	}
}

func comprobarPermisosExactos(t *testing.T, obtenidos, esperados []string) {
	t.Helper()
	conjuntoObtenido := conjuntoPermisos(t, obtenidos)
	conjuntoEsperado := conjuntoPermisos(t, esperados)
	if len(conjuntoObtenido) != len(conjuntoEsperado) {
		t.Fatalf("permisos = %#v, esperados %#v", obtenidos, esperados)
	}
	for permiso := range conjuntoEsperado {
		if _, ok := conjuntoObtenido[permiso]; !ok {
			t.Fatalf("falta permiso exacto %q en %#v", permiso, obtenidos)
		}
	}
}

func conjuntoPermisos(t *testing.T, permisos []string) map[string]struct{} {
	t.Helper()
	resultado := make(map[string]struct{}, len(permisos))
	for _, permiso := range permisos {
		if permiso == "" {
			t.Fatal("la lista contiene un permiso vacio")
		}
		if _, repetido := resultado[permiso]; repetido {
			t.Fatalf("la lista contiene el permiso repetido %q", permiso)
		}
		resultado[permiso] = struct{}{}
	}
	return resultado
}

func permisosDeManifiestos(manifiestos ...domain.ModuleManifest) []string {
	var permisos []string
	for _, manifiesto := range manifiestos {
		for _, permiso := range manifiesto.Permissions {
			permisos = append(permisos, permiso.Key)
		}
	}
	return permisos
}

func principalConPermisosExpresosPrueba(permisos ...string) domain.Principal {
	return domain.Principal{
		ID:            "actor-capacidad-expresa-prueba",
		Roles:         []string{"capacidad-expresa-prueba"},
		Permissions:   append([]string(nil), permisos...),
		AuthMethod:    domain.AuthMethodDemo,
		AuthAssurance: domain.AuthAssuranceHigh,
	}
}
