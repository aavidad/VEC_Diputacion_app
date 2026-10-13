package bootstrap

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestMaterialPublicacionExternaV3CompletoYPropio(t *testing.T) {
	materiales := materialesPublicadosPrueba(t, consumidoresPortalExternoV3...)
	defer func() {
		for _, lista := range materiales {
			borrarMaterialesV3PortalExterno(lista)
		}
	}()
	lista := ordenarMaterialExternoV3(materiales)
	if len(lista) != 23 {
		t.Fatalf("audiencias publicables = %d, se esperan 23", len(lista))
	}
	coordenadas := make([]coordenadaClaveExternaV3, len(lista))
	for i, m := range lista {
		coordenadas[i] = coordenadaClaveExternaV3{m.audienciaConsumo, uint64(100 + i), uint64(100 + i), uint64(100 + i)}
	}
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	if err := coordinarMaterialExternoV3(lista, "confianza:atestacion:externo:27", 27,
		ahora.Add(-time.Minute), ahora.Add(time.Hour), lista[0].validaDesde, lista[0].validaHasta, coordenadas); err != nil {
		t.Fatal(err)
	}
	propuesta, err := serializarMaterialExternoV3(lista)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(propuesta, []byte("clave:atestacion:ct:")) || bytes.Contains(propuesta, []byte("emisor:ct:")) {
		t.Fatal("la publicación conserva un identificador interno")
	}
	if !strings.HasPrefix(lista[0].claveID, "clave:atestacion:externo:") {
		t.Fatal("la raíz no es externa")
	}
	var contenido map[string]any
	if err := json.Unmarshal(propuesta, &contenido); err != nil {
		t.Fatal(err)
	}
	claves := contenido["claves"].([]any)
	for _, valor := range claves {
		delete(valor.(map[string]any), "secreto_hmac_hex")
	}
	publico, err := json.Marshal(contenido)
	if err != nil || !publicoIgualExternoV3(propuesta, publico) {
		t.Fatal("la publicación pública no coincide")
	}
	claves[0].(map[string]any)["huella_secreto_sha256"] = strings.Repeat("0", 64)
	alterado, err := json.Marshal(contenido)
	if err != nil || publicoIgualExternoV3(propuesta, alterado) {
		t.Fatal("replay divergente aceptado")
	}
}
