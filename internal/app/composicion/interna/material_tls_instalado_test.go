package interna

import (
	"os"
	"testing"
)

// El runner de aprovisionamiento ejecuta esta sonda como usuario no
// privilegiado sobre la copia instalada de la PKI de desarrollo. Sin esa
// instalación, la prueba local ordinaria no inventa material TLS.
func TestTLSInstaladoDesdeAprovisionamiento(t *testing.T) {
	certificado := os.Getenv("VEC_PRUEBA_TLS_INSTALADO_CERT")
	clave := os.Getenv("VEC_PRUEBA_TLS_INSTALADO_KEY")
	autoridad := os.Getenv("VEC_PRUEBA_TLS_INSTALADO_CA")
	nombre := os.Getenv("VEC_PRUEBA_TLS_INSTALADO_NOMBRE")
	if certificado == "" && clave == "" && autoridad == "" && nombre == "" {
		t.Skip("requiere PKI de desarrollo instalada por el aprovisionador")
	}
	if certificado == "" || clave == "" || autoridad == "" || nombre == "" || os.Geteuid() == 0 {
		t.Fatal("material TLS de instalación incompleto o proceso privilegiado")
	}
	cfg := configuracionInternaValidaPrueba()
	cfg.CertificadoServidorTLS = certificado
	cfg.ClaveServidorTLS = clave
	cfg.AutoridadClientesTLS = autoridad
	cfg.NombreServidorTLS = nombre
	if cfg.Validar() != nil {
		t.Fatal("configuración de superficie instalada inválida")
	}
	material, err := cargarMaterialTLS(cfg)
	if err != nil || material.configuracion == nil ||
		validarTLSMutuo(material.configuracion) != nil {
		t.Fatalf("la PKI instalada no carga bajo contrato TLS: %v", err)
	}
}
