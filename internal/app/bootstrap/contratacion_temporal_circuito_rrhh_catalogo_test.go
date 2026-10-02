package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

func TestCargarDefinicionCircuitoRRHHExigePublicacionIntacta(t *testing.T) {
	definicion, err := domain.NuevaDefinicionCircuitoRRHH(
		"flujo:ct:rrhh:prueba", 2, "solicitud",
		[]domain.TransicionCircuitoRRHH{{
			Clave: "peticion_firmada", Tipo: domain.HitoPeticionFirmada,
			Origen: "solicitud", Destino: "peticion_firmada",
			RequiereDocumento: true, RequiereFirma: true,
			PerfilClave: "tecnico_solicitante",
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(definicion)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "circuito.json")
	escribir := func(datos []byte) {
		t.Helper()
		if err := os.WriteFile(ruta, datos, 0600); err != nil {
			t.Fatal(err)
		}
	}
	escribir(contenido)
	leida, err := cargarDefinicionCircuitoRRHH(ruta)
	if err != nil || leida.Flujo != definicion.Flujo || len(leida.Transiciones) != 1 {
		t.Fatalf("publicacion valida: definicion=%+v error=%v", leida.Flujo, err)
	}

	alterada := strings.Replace(string(contenido), "tecnico_solicitante", "otro_perfil", 1)
	escribir([]byte(alterada))
	if _, err := cargarDefinicionCircuitoRRHH(ruta); err == nil {
		t.Fatal("acepto contenido distinto de la huella publicada")
	}
	escribir(append(contenido, []byte(` {"fuente":"cliente"}`)...))
	if _, err := cargarDefinicionCircuitoRRHH(ruta); err == nil {
		t.Fatal("acepto un segundo documento JSON")
	}
	escribir([]byte(strings.Replace(string(contenido), `"transiciones":`, `"fuente":"cliente","transiciones":`, 1)))
	if _, err := cargarDefinicionCircuitoRRHH(ruta); err == nil {
		t.Fatal("acepto un campo ajeno a la definicion")
	}
}
