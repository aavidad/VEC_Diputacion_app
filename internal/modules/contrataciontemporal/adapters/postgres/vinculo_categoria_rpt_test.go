package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestDecodificarConsultaVinculoRPTConservaAnclajeSQL(t *testing.T) {
	c, e := ports.NuevaConsultaVinculoCategoriaRPT("organizacion:uno", "expediente:uno")
	if e != nil {
		t.Fatal(e)
	}
	h := strings.Repeat("a", 64)
	raw := `{"encontrado":true,"version_expediente":7,"analisis":{"version":2,"recibo_ref":"recibo:analisis","huella_sha256":"` + h + `","categoria_ref":"categoria:tecnica"},"vinculo":null}`
	l, ok, e := decodificarConsultaVinculoRPT(raw, c)
	if e != nil || !ok || l.Analisis.AnalisisHuellaSHA256 != h || l.Analisis.VersionExpediente != 7 || l.Vinculo != nil {
		t.Fatalf("lectura SQL perdida: %+v %v %v", l, ok, e)
	}
	if _, _, e = decodificarConsultaVinculoRPT(strings.Replace(raw, "categoria:tecnica", "", 1), c); e == nil {
		t.Fatal("analisis incompleto admitido")
	}
	vinculo := `{"revision":1,"recibo_ref":"recibo:vinculo","catalogo_id":"rpt-categorias","modulo_id":"personal","catalogo_version":1,"catalogo_huella_sha256":"` + strings.Repeat("b", 64) + `","categoria_id":"categoria:tecnica","fuente_ref":"fuente:expediente","motivo_ref":"motivo:uno","aprobacion_ref":"aprobacion:uno","prospectivo":true,"acredita_procedencia_historica":false}`
	conVinculo := strings.Replace(raw, `"vinculo":null`, `"vinculo":`+vinculo, 1)
	l, ok, e = decodificarConsultaVinculoRPT(conVinculo, c)
	if e != nil || !ok || l.Vinculo == nil || l.Vinculo.MotivoRef != "motivo:uno" || l.Vinculo.AprobacionRef != "aprobacion:uno" {
		t.Fatalf("vinculo confirmado ilegible: %+v %v %v", l, ok, e)
	}
	if _, _, e = decodificarConsultaVinculoRPT(strings.Replace(conVinculo, `"motivo_ref":"motivo:uno",`, "", 1), c); e == nil {
		t.Fatal("vinculo sin motivo admitido")
	}
}

func TestRegistroVinculoRPTSoloDevuelveReciboExacto(t *testing.T) {
	m := ports.RegistroVinculoCategoriaRPT{Esquema: ports.EsquemaRegistroVinculoCategoriaRPT,
		OrganizacionRef: "organizacion:uno", ExpedienteRef: "expediente:uno", VersionExpedienteEsperada: 7,
		AnalisisVersion: 2, AnalisisReciboRef: "recibo:analisis", AnalisisHuellaSHA256: strings.Repeat("a", 64),
		CategoriaRef: "categoria:tecnica", CatalogoID: "rpt-categorias", ModuloID: "personal", CatalogoVersion: 1,
		CatalogoHuellaSHA256: strings.Repeat("b", 64), CategoriaID: "categoria:tecnica", FuenteRef: "fuente:expediente",
		MotivoRef: "motivo:uno", AprobacionRef: "aprobacion:uno", ClaveIdempotencia: "11111111-1111-4111-8111-111111111111"}
	h, e := m.SHA256()
	if e != nil {
		t.Fatal(e)
	}
	vinculo := ports.EstadoVinculoCategoriaRPT{Revision: 1, ReciboRef: "recibo:vinculo", CatalogoID: m.CatalogoID, ModuloID: m.ModuloID,
		CatalogoVersion: m.CatalogoVersion, CatalogoHuellaSHA256: m.CatalogoHuellaSHA256, CategoriaID: m.CategoriaID,
		FuenteRef: m.FuenteRef, MotivoRef: m.MotivoRef, AprobacionRef: m.AprobacionRef, Prospectivo: true}
	recibo := ports.ReciboVinculoCategoriaRPT{ReciboRef: "recibo:vinculo", RegistradoEn: time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC), Revision: 1, MaterialSHA256: h, Prospectivo: true,
		DecisionRef: "decision:ct", AuditoriaRef: "auditoria:ct", ConsumoHuellaSHA256: strings.Repeat("c", 64),
		RPTDecisionRef: "decision:rpt", RPTPerfilRef: "perfil:rpt", RPTAuditoriaRef: "auditoria:rpt", RPTConsumoHuellaSHA256: strings.Repeat("d", 64), Vinculo: vinculo}
	respuesta := map[string]any{"replay": false, "recibo": recibo, "vinculo": vinculo}
	b, _ := json.Marshal(respuesta)
	if _, e := decodificarRegistroVinculoRPT(string(b), m); e != nil {
		t.Fatal(e)
	}
	recibo.MaterialSHA256 = strings.Repeat("c", 64)
	respuesta["recibo"] = recibo
	b, _ = json.Marshal(respuesta)
	if _, e := decodificarRegistroVinculoRPT(string(b), m); e == nil {
		t.Fatal("huella de recibo ajena admitida")
	}
}

func TestLecturaHistoricaRPTCompruebaDocumentoYClave(t *testing.T) {
	doc := `{"id":"rpt-categorias","modulo_id":"personal","version":1,"estado":"publicado","entradas":[{"clave":"categoria:tecnica","etiqueta":"Tecnica"}]}`
	h := sha256.Sum256([]byte(doc))
	p := domain.PublicacionCategoriaRPT{CatalogoID: "rpt-categorias", ModuloID: "personal", CatalogoVersion: 1, CatalogoHuella: hex.EncodeToString(h[:]), CategoriaID: "categoria:tecnica", CategoriaClave: "categoria:tecnica"}
	respuesta := struct {
		Encontrado   bool `json:"encontrado"`
		ConsumoNuevo bool `json:"consumo_nuevo"`
		Datos        struct {
			Publicacion struct {
				CatalogoID        string `json:"catalogo_id"`
				Version           uint64 `json:"version"`
				HuellaSHA256      string `json:"huella_sha256"`
				DocumentoCanonico string `json:"documento_canonico"`
			} `json:"publicacion"`
			Entrada struct {
				Clave string `json:"clave"`
			} `json:"entrada"`
		} `json:"datos"`
	}{Encontrado: true, ConsumoNuevo: true}
	respuesta.Datos.Publicacion.CatalogoID = p.CatalogoID
	respuesta.Datos.Publicacion.Version = p.CatalogoVersion
	respuesta.Datos.Publicacion.HuellaSHA256 = p.CatalogoHuella
	respuesta.Datos.Publicacion.DocumentoCanonico = doc
	respuesta.Datos.Entrada.Clave = p.CategoriaID
	b, e := json.Marshal(respuesta)
	if e != nil {
		t.Fatal(e)
	}
	if ok, e := validarLecturaPublicacionRPT(string(b), p); e != nil || !ok {
		t.Fatalf("publicacion exacta rechazada: %v", e)
	}
	respuesta.Datos.Entrada.Clave = "categoria:otra"
	b, _ = json.Marshal(respuesta)
	if _, e := validarLecturaPublicacionRPT(string(b), p); e == nil {
		t.Fatal("entrada ajena admitida")
	}
}

func TestHuellaContextoVinculoRPTCoincideConContextoSQL(t *testing.T) {
	material := []byte(`{"esquema":"vec.ct.vinculo-categoria-rpt.consulta.v1","organizacion_ref":"organizacion:uno","expediente_ref":"expediente:uno"}`)
	h := sha256.Sum256(material)
	canon := `{"ambitos":{"organizacion_ref":"organizacion:uno"},"atributos":{"material_sha256":"` + hex.EncodeToString(h[:]) + `"}}`
	esperado := sha256.Sum256([]byte(canon))
	actual, e := huellaContextoVinculoRPT("expediente:uno", "contratacion_temporal", "vinculo_categoria_rpt_ct", map[string]string{"organizacion_ref": "organizacion:uno"}, material)
	if e != nil || actual != hex.EncodeToString(esperado[:]) {
		t.Fatalf("contexto CT divergente de AD3-127: %s %v", actual, e)
	}
	canonRPT := `{"ambitos":{"catalogo_id":"rpt-categorias","modulo_id":"personal"},"atributos":{"material_sha256":"` + hex.EncodeToString(h[:]) + `"}}`
	esperadoRPT := sha256.Sum256([]byte(canonRPT))
	actual, e = huellaContextoVinculoRPT("rpt-categorias", "personal", "catalogo_configurable", map[string]string{"catalogo_id": "rpt-categorias", "modulo_id": "personal"}, material)
	if e != nil || actual != hex.EncodeToString(esperadoRPT[:]) {
		t.Fatalf("contexto RPT divergente de AD3-117: %s %v", actual, e)
	}
}
