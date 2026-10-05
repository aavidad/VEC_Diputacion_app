package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestConfiguracionDenominacionRechazaImplicitudYCaminos(t *testing.T) {
	dir := t.TempDir()
	ruta := filepath.Join(dir, "cfg.json")
	m := metadatosDenominacionPrueba()
	guardarMetadatosDenominacion(t, ruta, m)
	f, e := NuevaFuenteConfiguracionPrivadaDenominacionPersona(ruta)
	if e != nil {
		t.Fatal("cfg_valida")
	}
	if e = os.Chmod(ruta, 0644); e != nil {
		t.Fatal("chmod")
	}
	if _, e = f.CargarMetadatosDenominacionPersona(context.Background()); e == nil {
		t.Fatal("cfg_publica")
	}
	if e = os.Chmod(ruta, 0600); e != nil {
		t.Fatal("chmod")
	}
	casos := [][]byte{
		[]byte(`{"version":1,"version":1,"entorno":"desarrollo"}`),
		[]byte(`{"version":1,"entorno":"produccion"}`),
	}
	for _, b := range casos {
		if e = os.WriteFile(ruta, b, 0600); e != nil {
			t.Fatal("fixture")
		}
		if _, e = f.CargarMetadatosDenominacionPersona(context.Background()); e == nil {
			t.Fatal("campos_no_cerrados")
		}
	}
	guardarMetadatosDenominacion(t, ruta, m)
	enlace := filepath.Join(dir, "enlace.json")
	if e = os.Symlink(ruta, enlace); e != nil {
		t.Fatal("symlink")
	}
	if _, e = NuevaFuenteConfiguracionPrivadaDenominacionPersona(enlace); e == nil {
		t.Fatal("enlace_admitido")
	}
	m.Cifrado.Referencia = "clave:contacto:prueba"
	guardarMetadatosDenominacion(t, ruta, m)
	if _, e = f.CargarMetadatosDenominacionPersona(context.Background()); e == nil {
		t.Fatal("familia_de_clave_ajena")
	}
}
