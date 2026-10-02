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
			PerfilClave:      "tecnico_solicitante",
			FirmasRequeridas: []domain.ClaveCatalogo{"tecnico_solicitante", "delegacion"},
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

func TestCatalogoRRHHV2PublicaVistosBuenosAlternativos(t *testing.T) {
	d, err := cargarDefinicionCircuitoRRHH("../../../config/contratacion_temporal_circuito_rrhh_v2.json")
	if err != nil {
		t.Fatal(err)
	}
	if d.Flujo.HuellaSHA256 != "1721c3a66576b21163b590602589f1627095bd6b6bfa37c79862af62775146e2" {
		t.Fatalf("huella v2 inesperada: %s", d.Flujo.HuellaSHA256)
	}
	for _, base := range []string{
		"contratacion_temporal.circuito.resolucion_firmada",
		"contratacion_temporal.circuito.resolucion_sin_cambio",
	} {
		jefatura, direccion := false, false
		for _, tr := range d.Transiciones {
			switch tr.Clave {
			case domain.ClaveCatalogo(base):
				jefatura = len(tr.FirmasRequeridas) == 2 && tr.FirmasRequeridas[0] == "jefatura_servicio_rrhh" && tr.FirmasRequeridas[1] == "diputacion_delegada_rrhh"
			case domain.ClaveCatalogo(base + "_visto_bueno_direccion"):
				direccion = len(tr.FirmasRequeridas) == 2 && tr.FirmasRequeridas[0] == "direccion_rrhh" && tr.FirmasRequeridas[1] == "diputacion_delegada_rrhh"
			}
		}
		if !jefatura || !direccion {
			t.Fatalf("faltan alternativas ordenadas en %s", base)
		}
	}
}
