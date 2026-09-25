package documentos

import "testing"

func TestManifestDocumentosDeclaraLecturaPeroNoFirmaNoAcreditada(t *testing.T) {
	m := Manifest()
	if err := m.Validate(); err != nil {
		t.Fatalf("manifest invalido: %v", err)
	}
	if m.ID != "vec.module.documentos" || len(m.Menu) != 1 ||
		m.Menu[0].RequiredPermissions[0] != "documentos.expediente.listar" {
		t.Fatal("menu documental inesperado")
	}
	for _, p := range m.Permissions {
		if p.Key == "documentos.firma.confirmar" {
			t.Fatal("anuncia transicion sin recibo del verificador")
		}
	}
}
