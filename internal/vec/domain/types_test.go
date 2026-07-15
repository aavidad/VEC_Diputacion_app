package domain

import (
	"strings"
	"testing"
)

func TestPrincipalDeniegaPermisoVacio(t *testing.T) {
	principal := Principal{
		ID: "persona-1", Permissions: []string{"documentos.generar"},
		AuthMethod: AuthMethodCertificate, AuthAssurance: AuthAssuranceHigh,
	}
	if !principal.HasPermission("documentos.generar") {
		t.Fatal("un permiso exacto de un principal valido debe reconocerse")
	}
	if principal.HasPermission("") || principal.HasPermission("   ") {
		t.Fatal("un permiso vacio nunca debe autorizar una operacion")
	}
	if principal.HasPermission(" documentos.generar ") {
		t.Fatal("el permiso solicitado no debe normalizarse para conceder acceso")
	}
	principal.Permissions = append(principal.Permissions, " permiso-invalido ")
	if principal.HasPermission("documentos.generar") {
		t.Fatal("un principal parcialmente invalido no debe conservar otros permisos")
	}
	if principal.HasAllPermissions(nil) || principal.HasAllPermissions([]string{}) {
		t.Fatal("una lista de permisos vacia debe denegar por defecto")
	}
}

func TestPrincipalNoInfierePermisosDesdeRoles(t *testing.T) {
	principal := Principal{
		ID:            "persona-1",
		Roles:         []string{"administrador", "tecnico_rrhh"},
		AuthMethod:    AuthMethodCertificate,
		AuthAssurance: AuthAssuranceHigh,
	}
	if err := principal.Validate(); err != nil {
		t.Fatalf("principal valido: %v", err)
	}
	for _, permiso := range []string{"vec.roles.manage", "bolsa.baremacion.decision.inicial.adoptar"} {
		if principal.HasPermission(permiso) {
			t.Fatalf("el rol informativo concedio el permiso no declarado %q", permiso)
		}
	}
}

func TestManifestExigePermisosPositivosConcretosYDeclarados(t *testing.T) {
	base := ModuleManifest{
		ID:      "vec.module.prueba",
		NameKey: "ui.module.prueba",
		Permissions: []Permission{
			{Key: "prueba.consultar", LabelKey: "ui.permission.prueba.consultar"},
		},
		Menu: []MenuEntry{{
			ID: "prueba.inicio", ModuleID: "vec.module.prueba", LabelKey: "ui.menu.prueba.inicio",
			Path: "/modules/prueba", RequiredPermissions: []string{"prueba.consultar"},
		}},
	}
	if err := base.Validate(); err != nil {
		t.Fatalf("manifest valido: %v", err)
	}

	pruebas := []struct {
		nombre    string
		modificar func(*ModuleManifest)
	}{
		{"catalogo_vacio", func(m *ModuleManifest) { m.Permissions = nil }},
		{"menu_sin_permiso", func(m *ModuleManifest) { m.Menu[0].RequiredPermissions = nil }},
		{"permiso_no_declarado", func(m *ModuleManifest) { m.Menu[0].RequiredPermissions = []string{"prueba.administrar"} }},
		{"comodin_catalogo", func(m *ModuleManifest) { m.Permissions[0].Key = "prueba.*" }},
		{"comodin_menu", func(m *ModuleManifest) { m.Menu[0].RequiredPermissions = []string{"*"} }},
		{"modulo_cruzado", func(m *ModuleManifest) { m.Menu[0].ModuleID = "vec.module.otro" }},
		{"ruta_externa", func(m *ModuleManifest) { m.Menu[0].Path = "//sitio.example" }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			manifest := base
			manifest.Permissions = append([]Permission(nil), base.Permissions...)
			manifest.Menu = append([]MenuEntry(nil), base.Menu...)
			manifest.Menu[0].RequiredPermissions = append([]string(nil), base.Menu[0].RequiredPermissions...)
			prueba.modificar(&manifest)
			if err := manifest.Validate(); err == nil {
				t.Fatal("el manifiesto inseguro fue aceptado")
			}
		})
	}
}

func TestPrincipalValidaMetodoYGarantiaConocidos(t *testing.T) {
	valido := Principal{ID: "persona-1", AuthMethod: AuthMethodCertificate, AuthAssurance: AuthAssuranceHigh}
	if err := valido.Validate(); err != nil {
		t.Fatalf("principal valido: %v", err)
	}

	pruebas := []Principal{
		{ID: "persona-1", AuthMethod: "inventado", AuthAssurance: AuthAssuranceHigh},
		{ID: "persona-1", AuthMethod: AuthMethodCertificate, AuthAssurance: "inventada"},
		{ID: " persona-1", AuthMethod: AuthMethodCertificate, AuthAssurance: AuthAssuranceHigh},
		{ID: "persona-1", Roles: []string{"tecnico", "tecnico"}, AuthMethod: AuthMethodCertificate, AuthAssurance: AuthAssuranceHigh},
		{ID: "persona-1", Roles: []string{"*"}, AuthMethod: AuthMethodCertificate, AuthAssurance: AuthAssuranceHigh},
		{ID: "persona-1", Permissions: []string{"bolsa.*"}, AuthMethod: AuthMethodCertificate, AuthAssurance: AuthAssuranceHigh},
		{ID: "persona-1", Attributes: map[string]string{"dato": "valor\x00oculto"}, AuthMethod: AuthMethodCertificate, AuthAssurance: AuthAssuranceHigh},
		{ID: "persona-1", Attributes: map[string]string{"dato": strings.Repeat("x", 1025)}, AuthMethod: AuthMethodCertificate, AuthAssurance: AuthAssuranceHigh},
	}
	for _, principal := range pruebas {
		if err := principal.Validate(); err != ErrPrincipalInvalid {
			t.Fatalf("Validate() = %v; esperado ErrPrincipalInvalid para %+v", err, principal)
		}
	}
}
