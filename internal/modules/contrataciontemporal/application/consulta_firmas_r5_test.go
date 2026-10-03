package application

import (
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestCamposConsultaFirmasR5ContratoAD159(t *testing.T) {
	esperados := []string{
		"CatalogoHuella", "CatalogoRef", "ClaveIdempotencia", "CoincideFirmanteCandidato", "CoincideFirmanteEnOtroPaso", "ConMotivoDevolucion",
		"Documento", "DocumentoCustodiaRef", "DocumentoCustodiaVersion", "ExpedienteVersion",
		"FechaPortafirmasDeclarada", "FirmaRef", "FirmadoHuella", "FirmantePrincipalAcreditado",
		"HistoriaHuella", "HistoriaRevision", "HistoriaSeparacionAcreditada",
		"OriginalHuella", "OriginalRef", "OriginalVersion", "PasoOrden", "PasoRef", "ReciboRef",
		"ReferenciaPortafirmasDeclarada", "RegistradaEn", "Resultado", "Secuencia",
		"SelloTiempoEstado", "Via",
	}
	if got := CamposConsultaFirmasR5(); !reflect.DeepEqual(got, esperados) {
		t.Fatalf("proyeccion AD159 incompatible: %q", got)
	}
}

func TestConsultaFirmasR5LigaVersionYDocumentoAlRecurso(t *testing.T) {
	m := ports.MaterialConsultaFirmasR5{
		OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:001",
		VersionExpediente: 7, Documento: "informe_definitivo", FirmantePrincipalCandidatoRef: "per_firmante_sintetico_001",
		ClaveIdempotencia: "clave-consulta-r5-0001", PasoOrden: 1, CatalogoHuella: strings.Repeat("a", 64),
	}
	canon, err := m.Canonico()
	esperado := `{"OrganizacionRef":"organizacion:desarrollo:dipgra","ExpedienteRef":"expediente:ct:001","VersionExpediente":7,"Documento":"informe_definitivo","FirmantePrincipalCandidatoRef":"per_firmante_sintetico_001","ClaveIdempotencia":"clave-consulta-r5-0001","PasoOrden":1,"CatalogoHuella":"` + strings.Repeat("a", 64) + `"}`
	if err != nil || string(canon) != esperado {
		t.Fatalf("canon AD159: %s, %v", canon, err)
	}
	recurso, err := RecursoConsultaFirmasR5(m)
	if err != nil || recurso.Referencia != m.ExpedienteRef ||
		recurso.Ambitos["organizacion_ref"] != m.OrganizacionRef ||
		recurso.Atributos["material_sha256"] == "" {
		t.Fatalf("recurso AD159: %+v, %v", recurso, err)
	}
	base := recurso.Atributos["material_sha256"]
	m.VersionExpediente++
	otro, err := RecursoConsultaFirmasR5(m)
	if err != nil || otro.Atributos["material_sha256"] == base {
		t.Fatal("la version no quedo ligada al recurso")
	}
	m.VersionExpediente--
	m.Documento = "resolucion"
	otro, err = RecursoConsultaFirmasR5(m)
	if err != nil || otro.Atributos["material_sha256"] == base {
		t.Fatal("el documento no quedo ligado al recurso")
	}
	m.Documento = "informe_definitivo"
	m.PasoOrden = 2
	otro, err = RecursoConsultaFirmasR5(m)
	if err != nil || otro.Atributos["material_sha256"] == base {
		t.Fatal("el paso no quedo ligado al recurso")
	}
}
