package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func muestra() (opciones, destino) {
	ref := func(p string) string { return p + strings.Repeat("a", 24) }
	o := opciones{Registro: ref("rca_"), Vinculo: ref("vcc_"), Revision: 1, Estado: "activo", Desde: "2026-10-03T10:00:00.123456Z", Hasta: "2027-10-03T10:00:00Z", Evidencia: ref("evi_"), EvidenciaSHA: strings.Repeat("a", 64), SoloKit: true}
	d := destino{CuentaRef: ref("cta_"), CuentaVersion: 2, PersonaRef: ref("per_"), PersonaVersion: 3, VinculoRef: ref("vca_"), VinculoVersion: 4}
	return o, d
}

func TestDescriptorCanonico(t *testing.T) {
	o, d := muestra()
	x, err := construir(o, d, strings.Repeat("b", 64))
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(x)
	if err != nil {
		t.Fatal(err)
	}
	esperado := `{"esquema":"vec.certificado-cuenta.binding.v1","vinculo_ref":"` + o.Vinculo + `","revision":1,"certificado_der_sha256":"` + strings.Repeat("b", 64) + `","cuenta_ref":"` + d.CuentaRef + `","cuenta_version":2,"persona_ref":"` + d.PersonaRef + `","persona_version":3,"vinculo_cuenta_persona_ref":"` + d.VinculoRef + `","vinculo_cuenta_persona_version":4,"estado":"activo","vigente_desde":"` + o.Desde + `","vigente_hasta":"` + o.Hasta + `","evidencia_ref":"` + o.Evidencia + `","evidencia_sha256":"` + o.EvidenciaSHA + `","preimagen_ref":"","preimagen_revision":0,"preimagen_sha256":""}`
	if string(b) != esperado {
		t.Fatalf("descriptor distinto")
	}
	o.Preimagen = o.Vinculo
	o.PreimagenRevision = 1
	o.PreimagenSHA = strings.Repeat("c", 64)
	o.Revision = 2
	if _, err = construir(o, d, strings.Repeat("b", 64)); err != nil {
		t.Fatal(err)
	}
	o.PreimagenSHA = ""
	if _, err = construir(o, d, strings.Repeat("b", 64)); err == nil {
		t.Fatal("CAS incompleto")
	}
}

func TestValidacion(t *testing.T) {
	o, d := muestra()
	sha := strings.Repeat("b", 64)
	for nombre, cambiar := range map[string]func(*opciones){
		"sin marca":    func(o *opciones) { o.SoloKit = false },
		"registro":     func(o *opciones) { o.Registro = "rca_corto" },
		"evidencia":    func(o *opciones) { o.EvidenciaSHA = strings.Repeat("0", 64) },
		"fecha":        func(o *opciones) { o.Desde = "2026-10-03T10:00:00.1234567Z" },
		"intervalo":    func(o *opciones) { o.Hasta = o.Desde },
		"estado nuevo": func(o *opciones) { o.Estado = "revocado" },
	} {
		t.Run(nombre, func(t *testing.T) {
			a := o
			cambiar(&a)
			if _, err := construir(a, d, sha); err == nil {
				t.Fatal("aceptado")
			}
		})
	}
	if hostLoopback("db.ejemplo") || !hostLoopback("localhost") || !hostLoopback("::1") {
		t.Fatal("frontera de red")
	}
	if _, err := destinoJSON([]byte(`{"cuenta_ref":"x"}`)); err == nil {
		t.Fatal("metadata incompleta")
	}
}

func certificadoPrueba(t *testing.T) (string, string) {
	t.Helper()
	clave, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	plantilla := &x509.Certificate{SerialNumber: big.NewInt(1), NotBefore: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), NotAfter: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC), BasicConstraintsValid: true}
	der, err := x509.CreateCertificate(rand.Reader, plantilla, plantilla, &clave.PublicKey, clave)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(t.TempDir(), "muestra.der")
	if err = os.WriteFile(ruta, der, 0600); err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(der)
	return ruta, hex.EncodeToString(h[:])
}

func TestCertificadoYSalida(t *testing.T) {
	ruta, sha := certificadoPrueba(t)
	obtenida, err := certificadoSHA(ruta)
	if err != nil || obtenida != sha {
		t.Fatal("hash DER")
	}
	o, d := muestra()
	args := []string{"--solo-kit-sintetico", "--registro-destino-ref", o.Registro, "--certificado-der", ruta, "--vinculo-ref", o.Vinculo, "--revision", "1", "--estado", o.Estado, "--vigente-desde", o.Desde, "--vigente-hasta", o.Hasta, "--evidencia-ref", o.Evidencia, "--evidencia-sha256", o.EvidenciaSHA}
	var salida, diagnostico bytes.Buffer
	llamadas := 0
	consulta := func(string) (destino, error) { llamadas++; return d, nil }
	if codigo := ejecutar(args, &salida, &diagnostico, consulta); codigo != 0 || llamadas != 1 {
		t.Fatal("kit")
	}
	if !bytes.HasSuffix(salida.Bytes(), []byte{'\n'}) || bytes.Contains(salida.Bytes(), []byte("CN")) || bytes.Contains(salida.Bytes(), []byte("SPKI")) {
		t.Fatal("salida")
	}
	var objeto map[string]any
	if err = json.Unmarshal(bytes.TrimSuffix(salida.Bytes(), []byte{'\n'}), &objeto); err != nil || len(objeto) != 18 || objeto["certificado_der_sha256"] != sha {
		t.Fatal("forma JSON")
	}
	salida.Reset()
	diagnostico.Reset()
	llamadas = 0
	if codigo := ejecutar(args[:len(args)-2], &salida, &diagnostico, consulta); codigo == 0 || salida.Len() != 0 || llamadas != 0 {
		t.Fatal("fallo abierto")
	}
	salida.Reset()
	diagnostico.Reset()
	if codigo := ejecutar(args, &salida, &diagnostico, func(string) (destino, error) { return destino{}, os.ErrPermission }); codigo == 0 || salida.Len() != 0 {
		t.Fatal("fallo de consulta abierto")
	}
}
