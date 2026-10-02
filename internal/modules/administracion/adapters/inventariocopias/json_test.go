package inventariocopias

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestJSONEstrictoYAcotado(t *testing.T) {
	for _, entrada := range []string{
		`{"inventario":{},"inventario":{}}`,
		`{"inventario":{},"INVENTARIO":{}}`,
		`{"inventario":{"ref":"a","ref":"b"}}`,
		`{"inventario":{},"desconocido":true}`,
		`{} {}`,
		`{"inventario":`,
		`null`,
		strings.Repeat(" ", MaxDocumentoBytes+1),
		strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34),
	} {
		if _, err := LeerDescriptor(strings.NewReader(entrada)); err != ErrDocumento {
			t.Fatalf("error=%v longitud=%d", err, len(entrada))
		}
	}
	_, descriptor, _ := fixture(t)
	datos, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LeerDescriptor(bytes.NewReader(datos)); err != nil {
		t.Fatal(err)
	}
}

func TestJSONExigeInventariosCompletosSinNull(t *testing.T) {
	_, descriptor, observado := fixture(t)
	for _, esDescriptor := range []bool{false, true} {
		var original any = observado
		if esDescriptor {
			original = descriptor
		}
		datos, err := json.Marshal(original)
		if err != nil {
			t.Fatal(err)
		}
		for _, campo := range []string{"almacenes", "extensiones", "migraciones"} {
			for _, usarNull := range []bool{false, true} {
				var documento map[string]any
				if err := json.Unmarshal(datos, &documento); err != nil {
					t.Fatal(err)
				}
				inventario := documento
				if esDescriptor {
					inventario = documento["inventario"].(map[string]any)
				}
				objeto := inventario["postgresql"].(map[string]any)
				if campo == "migraciones" {
					objeto = inventario["modulos"].([]any)[0].(map[string]any)
				}
				if usarNull {
					objeto[campo] = nil
				} else {
					delete(objeto, campo)
				}
				entrada, err := json.Marshal(documento)
				if err != nil {
					t.Fatal(err)
				}
				if esDescriptor {
					_, err = LeerDescriptor(bytes.NewReader(entrada))
				} else {
					_, err = LeerInventario(bytes.NewReader(entrada))
				}
				if err != ErrDocumento {
					t.Fatalf("descriptor=%v campo=%s null=%v error=%v", esDescriptor, campo, usarNull, err)
				}
			}
		}
	}
}
