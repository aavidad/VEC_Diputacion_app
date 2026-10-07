package bootstrap

import (
	"bytes"
	"crypto/ed25519"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type relojFirmantePrivadoPrueba struct{}

func (*relojFirmantePrivadoPrueba) Ahora() time.Time {
	return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
}

func TestFirmantePrivadoCargaSoloSemillaConPublicaExacta(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "semilla.bin")
	semilla := bytes.Repeat([]byte{17}, ed25519.SeedSize)
	if err := os.WriteFile(ruta, semilla, 0600); err != nil {
		t.Fatal(err)
	}
	publica := ed25519.NewKeyFromSeed(semilla).Public().(ed25519.PublicKey)
	cfg := ConfiguracionFirmanteAtestacionV3Privado{ClaveID: "clave:admin:prueba", Audiencia: "vec:admin:prueba:atestacion:v3", PrefijoEvidencia: "evidencia:firma:admin:prueba:", ArchivoSemilla: ruta, PublicaEsperada: publica}
	f, cerrar, err := NuevoFirmanteAtestacionV3DesdeArchivo(cfg, &relojFirmantePrivadoPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	datos := f.(*firmanteAtestacionAltaContratacionTemporalDesarrollo)
	if datos.audiencia != cfg.Audiencia || datos.prefijoEvidencia != cfg.PrefijoEvidencia || !bytes.Equal(datos.privada.Public().(ed25519.PublicKey), publica) {
		t.Fatal("material sustituido")
	}
	cerrar()
	if len(datos.privada) != 0 {
		t.Fatal("clave no borrada")
	}
	cfg.PublicaEsperada = ed25519.NewKeyFromSeed(bytes.Repeat([]byte{18}, ed25519.SeedSize)).Public().(ed25519.PublicKey)
	if _, _, err := NuevoFirmanteAtestacionV3DesdeArchivo(cfg, &relojFirmantePrivadoPrueba{}); err == nil {
		t.Fatal("publica ajena aceptada")
	}
	cfg.PublicaEsperada = publica
	if err := os.Chmod(ruta, 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := NuevoFirmanteAtestacionV3DesdeArchivo(cfg, &relojFirmantePrivadoPrueba{}); err == nil {
		t.Fatal("archivo legible por otros aceptado")
	}
	if _, _, err := NuevoFirmanteAtestacionV3DesdeArchivo(cfg, (*relojFirmantePrivadoPrueba)(nil)); err == nil {
		t.Fatal("reloj nulo tipado aceptado")
	}
}
