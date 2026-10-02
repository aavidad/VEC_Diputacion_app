package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func materialRevisionPrueba172() MaterialFirmaVerificadaV2 {
	h := func(c string) string { return strings.Repeat(c, 64) }
	evidence := json.RawMessage(`[{"Orden":1}]`)
	hash := sha256.Sum256(evidence)
	return MaterialFirmaVerificadaV2{
		MaterialFirmaExterna: MaterialFirmaExterna{Via: ViaFirmaCertificadoVEC, OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba", VersionExpediente: 7,
			Documento: "resolucion", CatalogoRef: "catalogo:circuito:1", CatalogoHuella: h("a"), PasoRef: "catalogo:circuito:resolucion:p1", PasoOrden: 1, Secuencia: 1, HistoriaHuella: h("b"), OriginalRef: "ref:" + h("c"), OriginalVersion: 1, OriginalHuella: h("c"), FirmadoHuella: h("d"), CertificadoHuella: h("e"), FirmanteRef: "ref:" + h("e"), FirmantePrincipalRef: "per_firmante_prueba", PerfilFirmanteRef: "perfil:jefatura", CargoFirmante: "ct_cargo_jefatura", UnidadFirmanteRef: "unidad:rrhh", PerfilActivoFirmanteRef: "perfil:activo", AsignacionFirmanteRef: "asignacion:prueba", AsignacionFirmanteVersion: 1, AsignacionFirmanteHuella: h("f"), VersionRolFirmanteRef: "rol:jefatura:1", VersionRolFirmanteHuella: h("1"), ControlVigenciaFirmanteRef: "rol:jefatura:1", ControlVigenciaFirmanteRevision: 1, ControlVigenciaFirmanteHuella: h("2"), AsignacionVigenteDesde: "2026-01-01T00:00:00Z", AsignacionVigenteHasta: "2027-01-01T00:00:00Z", ActoCompetenciaRef: "acto:asignacion:prueba", PoliticaVerificacion: "politica:vec:firma:verificacion-autonoma:v2", RevocacionEstado: "vigente", SelloTiempoEstado: "no_presente", ClaveIdempotencia: "clave-firma-prueba-000001", DocumentoCustodiaRef: "documento:firmado:prueba", DocumentoCustodiaVersion: VersionDocumentoCustodiado},
		RolIDFirmante: "ct_cargo_jefatura", CatalogoVersion: 1, CuentaFirmanteRef: "cuenta:firmante:prueba", VinculoCredencialFirmanteRef: "vinculo:credencial:prueba", VinculoCredencialFirmanteRevision: 1, VinculoCredencialFirmanteHuella: h("3"), EntradaDocumentoRef: "ref:" + h("c"), EntradaDocumentoVersion: 1, EntradaDocumentoLongitud: 1000, EntradaDocumentoHuella: h("c"), OrdenFirmaPDF: 1, ByteRange: [4]uint64{0, 1050, 1150, 50}, RevisionHuellaSHA256: h("d"), ContenidoFirmadoHuellaSHA256: h("4"), RevisionLongitud: 1200, EvidenciaFirmasCanonica: evidence, EvidenciaFirmasHuellaSHA256: hex.EncodeToString(hash[:]), ComprobadaEn: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC),
	}
}
func TestMaterialFirmaV2ObservacionNoCambiaActo(t *testing.T) {
	m := materialRevisionPrueba172()
	first, e := m.Canonico()
	if e != nil {
		t.Fatal(e)
	}
	m.ComprobadaEn = m.ComprobadaEn.Add(time.Minute)
	second, e := m.Canonico()
	if e != nil || !bytes.Equal(first, second) {
		t.Fatal("la observación temporal cambió la identidad de la firma")
	}
	var wire map[string]json.RawMessage
	_ = json.Unmarshal(first, &wire)
	if _, ok := wire["ComprobadaEn"]; ok {
		t.Fatal("observación incluida en canon")
	}
	if string(wire["PoliticaVerificacion"]) != `"politica:vec:firma:verificacion-autonoma:v2"` {
		t.Fatal("política degradada a V1")
	}
}
func TestMaterialFirmaV2NoAceptaCadenaParcial(t *testing.T) {
	tests := map[string]func(*MaterialFirmaVerificadaV2){
		"hash evidencia":             func(m *MaterialFirmaVerificadaV2) { m.EvidenciaFirmasHuellaSHA256 = strings.Repeat("9", 64) },
		"firma sin antecedente":      func(m *MaterialFirmaVerificadaV2) { m.PasoOrden = 2; m.OrdenFirmaPDF = 2 },
		"entrada ajena":              func(m *MaterialFirmaVerificadaV2) { m.EntradaDocumentoRef = "ref:" + strings.Repeat("8", 64) },
		"revision sin cubrir final":  func(m *MaterialFirmaVerificadaV2) { m.ByteRange[3]-- },
		"longitud desbordada":        func(m *MaterialFirmaVerificadaV2) { m.RevisionLongitud = ^uint64(0) },
		"vinculo credencial ausente": func(m *MaterialFirmaVerificadaV2) { m.VinculoCredencialFirmanteRef = "" },
	}
	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			m := materialRevisionPrueba172()
			change(&m)
			if _, e := m.Canonico(); e == nil {
				t.Fatal("se admitió material parcial")
			}
		})
	}
}
func TestConsultaV2ExigeUnidadVECYConservaRRHH(t *testing.T) {
	base := MaterialConsultaFirmasR5{OrganizacionRef: "organizacion:prueba", ExpedienteRef: "expediente:prueba", VersionExpediente: 7, Documento: "resolucion", FirmantePrincipalCandidatoRef: "per_firmante_prueba", ClaveIdempotencia: "clave-firma-prueba-000001", PasoOrden: 1, CatalogoHuella: strings.Repeat("a", 64)}
	m := MaterialConsultaFirmasR5V2{MaterialConsultaFirmasR5: base, Via: ViaFirmaCertificadoVEC}
	if _, e := m.Canonico(); e == nil {
		t.Fatal("VEC consultó sin unidad del recurso")
	}
	m.UnidadRef = "unidad:rrhh"
	if _, e := m.Canonico(); e != nil {
		t.Fatal(e)
	}
	m.Via = ViaFirmaExternaPortafirmas
	if _, e := m.Canonico(); e == nil {
		t.Fatal("se amplió en silencio el ámbito RRHH")
	}
	m.UnidadRef = ""
	if _, e := m.Canonico(); e != nil {
		t.Fatal(e)
	}
}
