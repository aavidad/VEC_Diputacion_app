package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/app/separacionportales"
)

func TestTareasInternasNoSeEjecutanEnElProcesoExterno(t *testing.T) {
	for nombre := range subcomandosSoloInterno {
		if err := comprobarSubcomandoEnPortal([]string{"vec-server", nombre}, "externo"); err == nil {
			t.Fatalf("%s no debe ejecutarse en el proceso externo", nombre)
		}
		for _, portal := range []string{"", "interno"} {
			if err := comprobarSubcomandoEnPortal([]string{"vec-server", nombre}, portal); err != nil {
				t.Fatalf("%s en %q: %v", nombre, portal, err)
			}
		}
		if err := comprobarSubcomandoEnPortal([]string{"vec-server", nombre}, "ambos"); !errors.Is(err, separacionportales.ErrPortalDesconocido) {
			t.Fatalf("%s con portal no valido: %v", nombre, err)
		}
	}
	if err := comprobarSubcomandoEnPortal([]string{"vec-server"}, "externo"); err != nil {
		t.Fatalf("el servidor externo debe poder arrancar: %v", err)
	}
	if err := comprobarSubcomandoEnPortal([]string{"vec-server", "tarea-nueva"}, "externo"); err == nil {
		t.Fatal("una tarea no admitida no debe ejecutarse en el proceso externo")
	}
}

func materialPrueba(t *testing.T, portal, clave, huellaCAInterna string) (string, string) {
	t.Helper()
	raiz := filepath.Join(t.TempDir(), "material")
	claveCA, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	certificado := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "CA sintetica del portal " + portal},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, certificado, certificado, &claveCA.PublicKey, claveCA)
	if err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(der)
	version := 4
	manifiesto := map[string]any{}
	if portal == "externo" {
		version = 2
		manifiesto["huella_ca_interna_sha256"] = huellaCAInterna
	}
	manifiesto["version"] = version
	contenidoManifiesto, err := json.Marshal(manifiesto)
	if err != nil {
		t.Fatal(err)
	}
	for relativa, contenido := range map[string]string{
		separacionportales.FicheroMarcaPortal: `{"version":1,"portal":"` + portal + `"}`,
		"idempotencia/clave.bin":              clave,
		"ca/ca.crt":                           string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		"manifiesto.json":                     string(contenidoManifiesto),
	} {
		ruta := filepath.Join(raiz, filepath.FromSlash(relativa))
		if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, []byte(contenido), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return raiz, hex.EncodeToString(huella[:])
}

func TestComprobarSeparacionPortalesInformaSinValores(t *testing.T) {
	interno, huellaCAInterna := materialPrueba(t, "interno", "clave sintetica interna", "")
	externo, _ := materialPrueba(t, "externo", "clave sintetica externa", huellaCAInterna)
	guion := filepath.Join(t.TempDir(), "arrancar.sh")
	if err := os.WriteFile(guion, []byte("export VEC_CT_DATABASE_URL=\"postgresql://vec_ct_sintetico:secreta@127.0.0.1/vec\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	err := ejecutarComprobacionSeparacion([]string{"--material-interno", interno, "--material-externo", externo, "--entorno-interno", guion}, &salida)
	if err != nil || !strings.HasPrefix(salida.String(), "separacion_portales=correcta") || strings.Contains(salida.String(), "secreta") {
		t.Fatalf("salida %q, %v", salida.String(), err)
	}
	salida.Reset()
	err = ejecutarComprobacionSeparacion([]string{"--material-interno", interno, "--material-externo", externo, "--entorno-externo", guion}, &salida)
	if !errors.Is(err, separacionportales.ErrSeparacionPortales) || strings.Contains(err.Error(), "secreta") {
		t.Fatalf("una conexion interna en el externo debe rechazarse sin mostrarla: %v", err)
	}
	copia, _ := materialPrueba(t, "externo", "clave sintetica interna", huellaCAInterna)
	err = ejecutarComprobacionSeparacion([]string{"--material-interno", interno, "--material-externo", copia}, &salida)
	var rechazo *separacionportales.ErrorSeparacion
	if !errors.Is(err, separacionportales.ErrSeparacionPortales) || !errors.As(err, &rechazo) ||
		rechazo.Elemento != "idempotencia/clave.bin = idempotencia/clave.bin" {
		t.Fatalf("una clave de idempotencia copiada debe rechazarse por su contenido: %v", err)
	}
	if err := ejecutarComprobacionSeparacion([]string{"--material-interno", interno}, &salida); !errors.Is(err, errArgumentosSeparacion) {
		t.Fatalf("faltan argumentos: %v", err)
	}
}

func TestProcesoExternoSoloAdmiteSuListaPositivaDeTareas(t *testing.T) {
	if err := comprobarSubcomandoEnPortal([]string{"vec-server", subcomandoExportarSeudonimosPortalExterno}, "externo"); err != nil {
		t.Fatalf("la exportacion de alias debe admitirse en el externo: %v", err)
	}
	for _, tarea := range []string{subcomandoPrepararPortalExterno, subcomandoComprobarSeparacion, "importar-convoca"} {
		if err := comprobarSubcomandoEnPortal([]string{"vec-server", tarea}, "externo"); err == nil {
			t.Fatalf("%s no debe ejecutarse en el externo", tarea)
		}
	}
}

func TestPrepararConAliasExigeListaDeCuentas(t *testing.T) {
	llamada := false
	preparar := func(context.Context, config.Config, bootstrap.OpcionesPreparacionPortalExterno) (bootstrap.ResumenPreparacionPortalExterno, error) {
		llamada = true
		return bootstrap.ResumenPreparacionPortalExterno{}, nil
	}
	alias := filepath.Join(t.TempDir(), "alias.json")
	if err := os.WriteFile(alias, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	err := ejecutarPreparacionPortalExterno(context.Background(), []string{"--material-externo", "/x", "--consumidores", "mi_bolsa", "--seudonimos", alias}, io.Discard, config.Config{}, preparar)
	if !errors.Is(err, errArgumentosPreparacion) || llamada {
		t.Fatalf("alias sin lista de cuentas aceptados: %v", err)
	}
}
