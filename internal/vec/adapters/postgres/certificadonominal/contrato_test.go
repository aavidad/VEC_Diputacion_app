package certificadonominal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	vd "vec-diputacion-granada/internal/vec/domain"
)

const (
	orgPrueba    = "org_fbcc10ed290261ec40d52db731f395c8"
	unidadPrueba = "unidad:sintetica:h10:688c11eaf7ad8c060db295a7c0a22fb0"
)

var derPrueba = strings.Repeat("c", 64)

func asignacionPrueba() vd.AsignacionPerfil {
	return vd.AsignacionPerfil{Ambitos: []vd.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{orgPrueba}},
		{Clave: "unidad_ref", Valores: []string{unidadPrueba}}}}
}

func descriptorPrueba(t *testing.T, cambiar func(map[string]any)) []byte {
	t.Helper()
	d := map[string]any{"esquema": "vec.contexto-actor.certificado-firmante.publicacion.v2", "clave": strings.Repeat("a", 32),
		"vinculo_ref": "vcc_x", "version": 1, "certificado_der_sha256": derPrueba, "cuenta_ref": "cta_x", "persona_ref": "per_x",
		"vinculo_cuenta_persona_ref": "vca_x", "organizacion_ref": orgPrueba, "estado": "vigente",
		"vigente_desde": "2026-10-05T00:00:00.000000Z", "vigente_hasta": "2027-10-05T00:00:00.000000Z", "evidencia_ref": "evi_x",
		"evidencia_sha256": strings.Repeat("e", 64), "preimagen_ref": "", "preimagen_version": 0, "preimagen_sha256": "none"}
	if cambiar != nil {
		cambiar(d)
	}
	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// El recurso y su huella son los que AD205 calcula en SQL con los ámbitos de
// la asignación y el SHA-256 del descriptor.
func TestRecursoCertificadoCoincideConAD205(t *testing.T) {
	material := descriptorPrueba(t, nil)
	accion, r, err := Recurso(material, asignacionPrueba())
	if err != nil || accion != AccionPublicar || r.Referencia != "certificado-nominal:"+derPrueba || r.ModuloID != "administracion" ||
		r.Tipo != "vinculo_certificado_nominal" {
		t.Fatalf("recurso distinto: %v %+v", err, r)
	}
	suma := sha256.Sum256(material)
	canon := `{"ambitos":{"organizacion_ref":"` + orgPrueba + `","unidad_ref":"` + unidadPrueba +
		`"},"atributos":{"descriptor_sha256":"` + hex.EncodeToString(suma[:]) + `"}}`
	esperada := sha256.Sum256([]byte(canon))
	if h, err := r.HuellaContextoAutorizacionSHA256(); err != nil || h != hex.EncodeToString(esperada[:]) {
		t.Fatalf("huella distinta de la de AD205: %s", h)
	}
	if a, _, err := Recurso(descriptorPrueba(t, func(d map[string]any) { d["estado"] = "retirado" }), asignacionPrueba()); err != nil || a != AccionRetirar {
		t.Fatal("retirada sin su acción")
	}
	for nombre, cambiar := range map[string]func(map[string]any){
		"otro_esquema":      func(d map[string]any) { d["esquema"] = "vec.otro.v1" },
		"der_ceros":         func(d map[string]any) { d["certificado_der_sha256"] = strings.Repeat("0", 64) },
		"der_mayusculas":    func(d map[string]any) { d["certificado_der_sha256"] = strings.Repeat("C", 64) },
		"estado_raro":       func(d map[string]any) { d["estado"] = "suspendido" },
		"otra_organizacion": func(d map[string]any) { d["organizacion_ref"] = "org_" + strings.Repeat("d", 32) },
		"clave_de_mas":      func(d map[string]any) { d["extra"] = 1 },
		"clave_de_menos":    func(d map[string]any) { delete(d, "evidencia_ref") },
		"clave_corta":       func(d map[string]any) { d["clave"] = "abc" },
	} {
		if _, _, err := Recurso(descriptorPrueba(t, cambiar), asignacionPrueba()); err == nil {
			t.Fatalf("%s aceptado", nombre)
		}
	}
	for nombre, mutar := range map[string]func(*vd.AsignacionPerfil){
		"sin_unidad":   func(a *vd.AsignacionPerfil) { a.Ambitos = a.Ambitos[:1] },
		"dos_unidades": func(a *vd.AsignacionPerfil) { a.Ambitos[1].Valores = append(a.Ambitos[1].Valores, "unidad:otra") },
		"otra_clave":   func(a *vd.AsignacionPerfil) { a.Ambitos[1].Clave = "centro_ref" },
	} {
		a := asignacionPrueba()
		mutar(&a)
		if _, _, err := Recurso(material, a); err == nil {
			t.Fatalf("asignación %s aceptada", nombre)
		}
	}
	if !Contrato().Valido() {
		t.Fatal("contrato no válido")
	}
}

func respuestaPrueba(t *testing.T, material []byte, decisionRecibo, auditoriaRecibo, decisionConsumo, auditoriaConsumo string, cambiar func(map[string]any)) []byte {
	t.Helper()
	suma := sha256.Sum256(material)
	r := map[string]any{"esquema": "vec.contexto-actor.certificado-firmante.recibo.v2", "clave": strings.Repeat("a", 32), "vinculo_ref": "vcc_x",
		"version": 1, "certificado_der_sha256": derPrueba, "estado": "vigente", "decision_ref": decisionRecibo, "auditoria_ref": auditoriaRecibo,
		"recibo_ref": "recibo_certificado_nominal:" + strings.Repeat("a", 32), "descriptor_sha256": hex.EncodeToString(suma[:]),
		"organizacion_destino": map[string]any{"organizacion_ref": orgPrueba}, "registrada_en": "2026-10-05T20:00:00.000000Z"}
	if cambiar != nil {
		cambiar(r)
	}
	b, _ := json.Marshal(map[string]any{"recibo": r, "consumo": map[string]any{"decision_ref": decisionConsumo, "auditoria_ref": auditoriaConsumo}})
	return b
}

// El recibo de AD205 distingue «nueva» (mismo consumo) de «recuperada» (otra
// decisión y otra auditoría) y rechaza cualquier otra combinación o campo.
func TestReciboCertificado(t *testing.T) {
	material := descriptorPrueba(t, nil)
	accion, recurso, _ := Recurso(material, asignacionPrueba())
	a1, a2 := "aud_v3_"+strings.Repeat("1", 32), "aud_v3_"+strings.Repeat("2", 32)
	for estado, x := range map[string][4]string{"nueva": {"dec1", a1, "dec1", a1}, "recuperada": {"dec1", a1, "dec2", a2}} {
		r, err := recibo(respuestaPrueba(t, material, x[0], x[1], x[2], x[3], nil), accion, recurso)
		var cuerpo map[string]any
		if err != nil || r.ConsumoAuditoriaRef != x[3] || json.Unmarshal(r.Cuerpo, &cuerpo) != nil || cuerpo["estado_replay"] != estado ||
			cuerpo["auditoria_ref"] != a1 {
			t.Fatalf("%s: %v %s", estado, err, r.Cuerpo)
		}
	}
	for nombre, b := range map[string][]byte{
		"mezcla":       respuestaPrueba(t, material, "dec1", a1, "dec1", a2, nil),
		"otro_der":     respuestaPrueba(t, material, "dec1", a1, "dec1", a1, func(r map[string]any) { r["certificado_der_sha256"] = strings.Repeat("d", 64) }),
		"otro_estado":  respuestaPrueba(t, material, "dec1", a1, "dec1", a1, func(r map[string]any) { r["estado"] = "retirado" }),
		"otro_sha":     respuestaPrueba(t, material, "dec1", a1, "dec1", a1, func(r map[string]any) { r["descriptor_sha256"] = strings.Repeat("f", 64) }),
		"campo_de_mas": respuestaPrueba(t, material, "dec1", a1, "dec1", a1, func(r map[string]any) { r["persona_ref"] = "per_x" }),
		"auditoria":    respuestaPrueba(t, material, "dec1", "aud-x", "dec1", "aud-x", nil),
		"recibo_ref":   respuestaPrueba(t, material, "dec1", a1, "dec1", a1, func(r map[string]any) { r["recibo_ref"] = "recibo:x" }),
	} {
		if _, err := recibo(b, accion, recurso); err == nil {
			t.Fatalf("%s aceptado", nombre)
		}
	}
}
