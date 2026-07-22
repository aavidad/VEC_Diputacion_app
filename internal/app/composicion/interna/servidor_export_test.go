package interna

import (
	"net/http"
	"testing"
)

// NuevaCapsulaOpacaParaPrueba solo existe en el binario de pruebas para poder
// verificar desde el paquete consumidor que una copia por valor falla cerrada.
func NuevaCapsulaOpacaParaPrueba(t *testing.T) *ServidorInterno {
	t.Helper()
	material := materialTLSMutuoPrueba(t, opcionesCertificadoServidor{})
	servidor, err := construirServidorInternoPrueba(t, material.cfg, http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	return servidor
}
