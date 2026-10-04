package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
)

func entradaS2Prueba(t *testing.T) string {
	t.Helper()
	return `{"material_propuesto":` + string(ejemplo(t, "material-completo")) + `,"esperada":{"preparacion_ref":"prep:original","revision":3,"huella_material_sha256":"` + strings.Repeat("b", 64) + `"},"clave_operacion":"operacion:original"}`
}

func TestSalidaS2EsPeticionDeGuardadoConPreimagenExplicita(t *testing.T) {
	var out, errOut bytes.Buffer
	entrada := strings.Replace(entradaS2Prueba(t), `"version_material": 1`, `"version_material": 99`, 1)
	if ejecutar(context.Background(), []string{"--catalogos-dir", "../../web/static/textos", "--salida", "solicitud-s2"}, strings.NewReader(entrada), &out, &errOut) != 0 || errOut.Len() != 0 {
		t.Fatal(errOut.String())
	}
	var s solicitudS2JSON
	d := json.NewDecoder(bytes.NewReader(out.Bytes()))
	d.DisallowUnknownFields()
	if d.Decode(&s) != nil || s.Esperada.Revision != 3 || s.Esperada.PreparacionRef != "prep:original" || s.Esperada.HuellaMaterialSHA256 != strings.Repeat("b", 64) || s.ClaveOperacion != "operacion:original" {
		t.Fatalf("peticion incompatible: %s", out.String())
	}
	pendientes, err := s.Material.Pendientes()
	if err != nil || len(pendientes) != 14 || len(s.Material.Referencias) != 10 {
		t.Fatal("material o pendientes alterados", err)
	}
	for _, ref := range s.Material.Referencias {
		if ref.Referencia.ID != "ejemplo:"+ref.Campo || ref.Referencia.Version != 1 || ref.Referencia.HuellaContenidoSHA256 != strings.Repeat("a", 64) {
			t.Fatalf("referencia exacta alterada: %+v", ref)
		}
	}
	for _, p := range pendientes {
		if p.Codigo != "referencia_no_verificada" && p.Codigo != "circuito_pendiente" {
			t.Fatalf("referencia promovida: %+v", p)
		}
	}
	for _, prohibido := range []string{"actor", "ambito", "perfil", "autorizacion", "version_material", "identidad_material", "contenido_canonico_bolsa"} {
		if bytes.Contains(out.Bytes(), []byte(`"`+prohibido+`"`)) {
			t.Fatal(prohibido)
		}
	}
}

func TestSalidaS2ExigeWrapperCompletoYNoEmiteMaterialParcial(t *testing.T) {
	base := entradaS2Prueba(t)
	for nombre, entrada := range map[string]string{
		"material local":   string(ejemplo(t, "material-completo")),
		"esperada ausente": `{"material_propuesto":` + string(ejemplo(t, "material-completo")) + `,"clave_operacion":"op:1"}`,
		"esperada null":    `{"material_propuesto":` + string(ejemplo(t, "material-completo")) + `,"esperada":null,"clave_operacion":"op:1"}`,
		"clave ausente":    strings.Replace(base, `,"clave_operacion":"operacion:original"`, "", 1),
		"revision omitida": strings.Replace(base, `"revision":3,`, "", 1),
		"huella omitida":   strings.Replace(base, `,"huella_material_sha256":"`+strings.Repeat("b", 64)+`"`, "", 1),
		"huella ausente":   strings.Replace(base, strings.Repeat("b", 64), "", 1),
		"actor aportado":   strings.Replace(base, `"clave_operacion":`, `"actor":"persona:1","clave_operacion":`, 1),
		"clave repetida":   strings.Replace(base, `"clave_operacion":`, `"clave_operacion":"otra","clave_operacion":`, 1),
	} {
		t.Run(nombre, func(t *testing.T) {
			var out, errOut bytes.Buffer
			if ejecutar(context.Background(), []string{"--catalogos-dir", "../../web/static/textos", "--salida", "solicitud-s2"}, strings.NewReader(entrada), &out, &errOut) == 0 || out.Len() != 0 || errOut.Len() == 0 {
				t.Fatalf("salida parcial: %s / %s", out.String(), errOut.String())
			}
		})
	}
	var out, errOut bytes.Buffer
	if ejecutar(context.Background(), []string{"--salida", "desconocida"}, strings.NewReader(base), &out, &errOut) == 0 || out.Len() != 0 {
		t.Fatal("modo desconocido admitido")
	}
}

func TestSalidaS2AceptaAltaExplicitaSinRellenarReferencias(t *testing.T) {
	entrada := `{"material_propuesto":` + string(ejemplo(t, "material-incompleto")) + `,"esperada":{"preparacion_ref":"prep:nueva","revision":0,"huella_material_sha256":""},"clave_operacion":"operacion:nueva"}`
	var out, errOut bytes.Buffer
	if ejecutar(context.Background(), []string{"--catalogos-dir", "../../web/static/textos", "--salida", "solicitud-s2"}, strings.NewReader(entrada), &out, &errOut) != 0 {
		t.Fatal(errOut.String())
	}
	var s solicitudS2JSON
	if json.Unmarshal(out.Bytes(), &s) != nil || s.Esperada != (prep.Esperada{PreparacionRef: "prep:nueva"}) || len(s.Material.Contenido.Documentos) != 0 {
		t.Fatal("alta rellenada", out.String())
	}
}
