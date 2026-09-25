package internactproveedores

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// El binario exige el estado exacto posterior a AD3-69: cuatro funciones del
// preflight, ni una más ni una menos. aprovisionar.py debe cotejar lo mismo.
func TestManifiestoPreflightExigeExactamenteV1YV2(t *testing.T) {
	esperadas := []string{
		"vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v1(jsonb)",
		"vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)",
		"vec_autorizacion_atestada_v3.comprobar_material_emision_interna_v2(text,jsonb)",
		"vec_autorizacion_atestada_v3.leer_configuracion_interna_v2(text,jsonb)",
	}
	obtenidas := funcionesEsperadasPerfil(perfilPool{rol: "vec_autorizacion_atestada_v3_preflight_interno"})
	if !slices.Equal(obtenidas, esperadas) {
		t.Fatalf("manifiesto del preflight: %v", obtenidas)
	}
	fuente, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "deploy", "principal", "composicion_interna", "aprovisionar.py"))
	if err != nil {
		t.Fatal(err)
	}
	bloque := regexp.MustCompile(`(?s)"gobierno_v3": \[(.*?)\]`).FindSubmatch(fuente)
	if bloque == nil {
		t.Fatal("aprovisionar.py sin lista gobierno_v3")
	}
	var python []string
	for _, m := range regexp.MustCompile(`"([^"]+)"`).FindAllSubmatch(bloque[1], -1) {
		python = append(python, string(m[1]))
	}
	if !slices.Equal(python, esperadas) {
		t.Fatalf("aprovisionar.py y el binario discrepan: %v", python)
	}
	// La función que acredita el pool es la lectura v1 de CT, presente en ambos.
	if !slices.Contains(esperadas, perfilPreflight.funcion) {
		t.Fatal("la función acreditada del pool no está en el manifiesto")
	}
}

func TestTLSVerificadoRechazaRutaDeFallbackSinVerificacion(t *testing.T) {
	cfg := &pgconn.Config{Host: "db.interno", TLSConfig: &tls.Config{ServerName: "db.interno", MinVersion: tls.VersionTLS12}}
	if !tlsVerificado(cfg) {
		t.Fatal("configuracion TLS verificada rechazada")
	}
	cfg.Fallbacks = []*pgconn.FallbackConfig{{Host: "db.interno"}}
	if tlsVerificado(cfg) {
		t.Fatal("fallback sin TLS admitido")
	}
	cfg.Fallbacks[0].TLSConfig = &tls.Config{ServerName: "db.interno", InsecureSkipVerify: true}
	if tlsVerificado(cfg) {
		t.Fatal("fallback sin verificacion admitido")
	}
	cfg.Fallbacks = nil
	cfg.TLSConfig.ServerName = "otro.interno"
	if tlsVerificado(cfg) {
		t.Fatal("nombre TLS distinto admitido")
	}
}
