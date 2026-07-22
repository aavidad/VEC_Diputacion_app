package interna

import (
	"bytes"
	"os"
	"testing"
)

func TestLimpiarBytesPropiosSobrescribeBuffer(t *testing.T) {
	contenido := []byte("PRIVATE-KEY-SERIALIZADA")
	limpiarBytesPropios(contenido)
	if !bytes.Equal(contenido, make([]byte, len(contenido))) {
		t.Fatalf("buffer propietario no sobrescrito: %x", contenido)
	}
	limpiarBytesPropios(nil)
}

func TestMaterializarTLSNoLimpiaBuffersAjenos(t *testing.T) {
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	leer := func(ruta string) []byte {
		t.Helper()
		contenido, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		return contenido
	}
	certPEM := leer(material.cfg.CertificadoServidorTLS)
	clavePEM := leer(material.cfg.ClaveServidorTLS)
	caPEM := leer(material.cfg.AutoridadClientesTLS)
	certOriginal := append([]byte(nil), certPEM...)
	claveOriginal := append([]byte(nil), clavePEM...)
	caOriginal := append([]byte(nil), caPEM...)
	if _, err := materializarTLS(material.cfg, certPEM, clavePEM, caPEM); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(certPEM, certOriginal) || !bytes.Equal(clavePEM, claveOriginal) ||
		!bytes.Equal(caPEM, caOriginal) {
		t.Fatal("materializarTLS modifico buffers propiedad del llamador")
	}
}
