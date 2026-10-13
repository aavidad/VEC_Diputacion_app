package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	vec "vec-diputacion-granada/internal/vec/domain"
)

func TestMotivoCarteraExternaExigeReferenciaPublicadaEnMaterialPrivado(t *testing.T) {
	raiz := t.TempDir()
	ruta := filepath.Join(raiz, rutaMotivoCarteraExterna)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	referencia := vec.ReferenciaEntradaCatalogo{
		CatalogoID: "motivos_autorizacion_meritos_cartera", CatalogoVersion: 2,
		CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		EntradaClave:         "motivo_00000000000000000000000000000001",
	}
	cuenta := cuentaCarteraExterna{CertificadoSHA256: strings.Repeat("a", 64),
		CuentaRef: "cta_bbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_bbbbbbbbbbbbbbbbbbbbbb"}
	b, err := json.Marshal(configuracionMotivoCarteraExterna{Version: 1, MotivoRetirada: referencia,
		Cuentas: []cuentaCarteraExterna{cuenta}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0o600); err != nil {
		t.Fatal(err)
	}
	obtenida, err := leerConfiguracionCarteraExterna(raiz)
	if err != nil || obtenida.MotivoRetirada != referencia || len(obtenida.Cuentas) != 1 || obtenida.Cuentas[0] != cuenta {
		t.Fatalf("configuracion privada válida: %v %#v", err, obtenida)
	}
	if err := os.Chmod(ruta, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := leerConfiguracionCarteraExterna(raiz); err == nil {
		t.Fatal("aceptó material legible por terceros")
	}
	if err := os.Chmod(ruta, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(`{"version":1,"version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := leerConfiguracionCarteraExterna(raiz); err == nil {
		t.Fatal("aceptó claves duplicadas")
	}
}

func TestCuentaCarteraExternaConservaPersonaPeroSeparaCuentaYPerfil(t *testing.T) {
	huella := strings.Repeat("a", 64)
	usuarios := map[string]cuentaUsuariosPreferenciasDesarrollo{huella: {
		cuentaRutasDietasDesarrollo: cuentaRutasDietasDesarrollo{CertificadoSHA256: huella,
			Sujeto: "per_ejemplo", CuentaRef: "cta_aaaaaaaaaaaaaaaaaaaaaa",
			PerfilRef: "prf_aaaaaaaaaaaaaaaaaaaaaa"}}}
	configuradas := []cuentaCarteraExterna{{CertificadoSHA256: huella,
		CuentaRef: "cta_bbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_bbbbbbbbbbbbbbbbbbbbbb"}}
	meritos, err := cuentasNominalesCarteraExterna(usuarios, configuradas)
	if err != nil || len(meritos) != 1 || meritos[huella].Sujeto != usuarios[huella].Sujeto ||
		meritos[huella].CuentaRef == usuarios[huella].CuentaRef ||
		meritos[huella].PerfilRef == usuarios[huella].PerfilRef {
		t.Fatalf("identidades externas mezcladas: %v %#v", err, meritos)
	}
	configuradas[0].CuentaRef = usuarios[huella].CuentaRef
	if _, err := cuentasNominalesCarteraExterna(usuarios, configuradas); err == nil {
		t.Fatal("reutilizó cuenta Usuarios para Méritos")
	}
	configuradas[0].CuentaRef = "cta_bbbbbbbbbbbbbbbbbbbbbb"
	configuradas[0].PerfilRef = usuarios[huella].PerfilRef
	if _, err := cuentasNominalesCarteraExterna(usuarios, configuradas); err == nil {
		t.Fatal("reutilizó perfil Usuarios para Méritos")
	}
}
