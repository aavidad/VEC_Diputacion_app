package postgres

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestContextoB95RechazaVistaPreviaYRepresentacionesAlteradas(t *testing.T) {
	confirmacion := []byte(`{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:seleccion"},"atributos":{}}`)
	suma := sha256.Sum256(confirmacion)
	huella := hex.EncodeToString(suma[:])
	if !contextoRecursoCargaConvocaValido(confirmacion, huella) {
		t.Fatal("contexto nominal de confirmación rechazado")
	}
	for _, raw := range [][]byte{
		[]byte(`{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:seleccion"},"atributos":{"fase":"vista_previa"}}`),
		[]byte(`{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:seleccion"},"atributos":null}`),
		[]byte(`{"atributos":{},"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:seleccion"}}`),
		[]byte(`{"ambitos":{"ambito_ref":"ambito:bolsa","unidad_ref":"unidad:seleccion"},"atributos":{},"extra":1}`),
	} {
		if contextoRecursoCargaConvocaValido(raw, huella) {
			t.Fatalf("contexto no canónico o ajeno aceptado: %s", raw)
		}
	}
}
