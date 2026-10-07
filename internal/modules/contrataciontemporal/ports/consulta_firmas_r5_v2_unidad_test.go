package ports

import (
	"bytes"
	"testing"
)

// AD210 y CT186 distinguen una consulta sin unidad (UnidadRef null) de una
// con unidad (texto); "" no es ninguna de las dos y se denegaría.
func TestMaterialConsultaFirmasR5V2UnidadNullOTexto(t *testing.T) {
	m := MaterialConsultaFirmasR5V2{Via: ViaFirmaCertificadoVEC, MaterialConsultaFirmasR5: MaterialConsultaFirmasR5{
		OrganizacionRef: "org_consulta", ExpedienteRef: "exp:consulta", VersionExpediente: 1, Documento: "informe_definitivo",
		FirmantePrincipalCandidatoRef: "per_consulta", ClaveIdempotencia: "clave-consulta-000001", PasoOrden: 1,
		CatalogoHuella: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}
	sin, err := m.Canonico()
	if err != nil || !bytes.Contains(sin, []byte(`"UnidadRef":null`)) {
		t.Fatalf("sin unidad: %s %v", sin, err)
	}
	m.UnidadRef = "unidad:del:paso"
	con, err := m.Canonico()
	if err != nil || !bytes.Contains(con, []byte(`"UnidadRef":"unidad:del:paso"`)) {
		t.Fatalf("con unidad: %s %v", con, err)
	}
}
