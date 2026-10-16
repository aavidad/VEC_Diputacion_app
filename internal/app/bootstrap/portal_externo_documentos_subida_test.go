package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfiguracionSubidaLocalFallaSoloSuFuncion(t *testing.T) {
	raiz := t.TempDir()
	if _, err := leerConfiguracionSubidaDocumentosExterna(raiz); err == nil {
		t.Fatal("aceptó subida sin material privado")
	}
	material := filepath.Join(raiz, "externo", "documentos")
	if err := os.MkdirAll(material, 0o700); err != nil {
		t.Fatal(err)
	}
	directorio := filepath.Join(raiz, "custodia")
	if err := os.Mkdir(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	retencionDeclarada := int64(0)
	c := configuracionSubidaDocumentosExterna{
		Version: 1, DSNDatos: "doc-datos", DSNFrontera: "doc-frontera",
		Directorio: directorio, RetencionMinimaDias: &retencionDeclarada,
		ClaveCorrelacionArchivo: "correlacion.key",
		ClaveSeudonimoArchivo:   "seudonimo.key", ClaveSolicitudArchivo: "solicitud.key",
		Analizador: configuracionAnalizadorSubidaExterna{Endpoint: "https://analizador.invalid",
			NombreServidor: "analizador.invalid", RutaCA: "ca.pem", RutaCertificado: "cliente.pem",
			RutaClave: "cliente.key", ConectorID: "analizador_pdf"},
	}
	ruta := filepath.Join(material, "subidas.json")
	escribir := func() {
		b, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	escribir()
	if _, err := leerConfiguracionSubidaDocumentosExterna(raiz); err != nil {
		t.Fatalf("rechazó material local válido: %v", err)
	}
	c.RetencionMinimaDias = nil
	escribir()
	if _, err := leerConfiguracionSubidaDocumentosExterna(raiz); err == nil {
		t.Fatal("aceptó retención no declarada")
	}
	c.RetencionMinimaDias = &retencionDeclarada
	escribir()
	if err := os.Chmod(ruta, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := leerConfiguracionSubidaDocumentosExterna(raiz); err == nil {
		t.Fatal("aceptó material legible por terceros")
	}
	if err := os.Chmod(ruta, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(directorio, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := leerConfiguracionSubidaDocumentosExterna(raiz); err == nil {
		t.Fatal("aceptó custodia accesible por terceros")
	}
	if err := os.Chmod(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	c.ClaveSolicitudArchivo = "../otra.key"
	escribir()
	if _, err := leerConfiguracionSubidaDocumentosExterna(raiz); err == nil {
		t.Fatal("aceptó clave fuera del directorio privado")
	}
}

func TestClavesHMACSubidaLocalExigenTreintaYDosBytesPrivados(t *testing.T) {
	raiz := t.TempDir()
	material := filepath.Join(raiz, "externo", "documentos")
	if err := os.MkdirAll(material, 0o700); err != nil {
		t.Fatal(err)
	}
	c := configuracionSubidaDocumentosExterna{ClaveCorrelacionArchivo: "correlacion.key",
		ClaveSeudonimoArchivo: "seudonimo.key", ClaveSolicitudArchivo: "solicitud.key"}
	ruta := filepath.Join(material, "solicitud.key")
	if _, err := leerClavesSubidaDocumentosExterna(raiz, c); err == nil {
		t.Fatal("aceptó clave ausente")
	}
	for nombre, valor := range map[string]string{
		"correlacion.key": "12345678901234567890123456789012",
		"seudonimo.key":   "22345678901234567890123456789012",
		"solicitud.key":   "32345678901234567890123456789012",
	} {
		if err := os.WriteFile(filepath.Join(material, nombre), []byte(valor), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	claves, err := leerClavesSubidaDocumentosExterna(raiz, c)
	if err != nil || claves == (clavesSubidaDocumentosExterna{}) {
		t.Fatalf("rechazó clave privada válida: %v", err)
	}
	correlaciones, seudonimos, sellador, err := nuevasClavesAplicacionSubidaLocal(&claves)
	if err != nil || correlaciones == nil || seudonimos == nil || sellador == nil {
		t.Fatalf("no compuso tres finalidades separadas: %v", err)
	}
	if claves != (clavesSubidaDocumentosExterna{}) {
		t.Fatal("no borró copia temporal")
	}
	if err := os.Chmod(ruta, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := leerClavesSubidaDocumentosExterna(raiz, c); err == nil {
		t.Fatal("aceptó clave legible por terceros")
	}
}
