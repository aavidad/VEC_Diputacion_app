package seudonimizacionpkcs11

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/miekg/pkcs11"

	registro "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

func TestConfiguracionRechazaDependenciasIncompletas(t *testing.T) {
	c := Configuracion{
		Modulo: "/modulo/inexistente", TokenLabel: "vec-prueba", TokenSerial: "serial-prueba",
		ObjetoID: []byte{1}, ClaveID: "clave-prueba", ClaveVersion: 1,
		DominioRef: "idh_0123456789012345678901", EspacioIdentidad: "https://idp.example.test",
		PINFichero: "/pin/inexistente",
	}
	if _, err := Abrir(c); err == nil {
		t.Fatal("se admitio modulo PKCS#11 ausente")
	}
	casos := []func(*Configuracion){
		func(c *Configuracion) { c.ObjetoID = nil },
		func(c *Configuracion) { c.ClaveVersion = 0 },
		func(c *Configuracion) { c.DominioRef = "idh_corto" },
		func(c *Configuracion) { c.EspacioIdentidad = "http://idp.example.test" },
	}
	for _, cambiar := range casos {
		invalida := c
		cambiar(&invalida)
		if _, err := Abrir(invalida); err == nil {
			t.Fatal("se admitio configuracion PKCS#11 invalida")
		}
	}
}

// Esta prueba usa el modulo SoftHSM real indicado por el ejecutor. El token,
// los PIN y la clave se crean dentro de t.TempDir y no salen del proceso.
func TestSoftHSMSeudonimizaCincoPropositos(t *testing.T) {
	modulo := os.Getenv("VEC_TEST_SOFTHSM_MODULE")
	if modulo == "" {
		t.Skip("SoftHSM no configurado para prueba PKCS#11")
	}
	directorio := t.TempDir()
	tokens := filepath.Join(directorio, "tokens")
	if err := os.Mkdir(tokens, 0700); err != nil {
		t.Fatal(err)
	}
	conf := filepath.Join(directorio, "softhsm2.conf")
	if err := os.WriteFile(conf, []byte("directories.tokendir = "+tokens+"\nobjectstore.backend = file\nlog.level = ERROR\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOFTHSM2_CONF", conf)
	soPIN := pinAleatorio(t)
	userPIN := pinAleatorio(t)
	ctx := pkcs11.New(modulo)
	if ctx == nil {
		t.Fatal("SoftHSM no disponible")
	}
	if err := ctx.Initialize(); err != nil {
		t.Fatal(err)
	}
	slots, err := ctx.GetSlotList(false)
	if err != nil || len(slots) == 0 {
		t.Fatalf("sin slot SoftHSM: %v", err)
	}
	const label = "vec-hmac-prueba"
	if err := ctx.InitToken(slots[0], soPIN, label); err != nil {
		t.Fatal(err)
	}
	activos, err := ctx.GetSlotList(true)
	if err != nil {
		t.Fatal(err)
	}
	var slot uint
	var encontrado bool
	for _, candidato := range activos {
		i, err := ctx.GetTokenInfo(candidato)
		if err == nil && strings.TrimSpace(i.Label) == label {
			slot, encontrado = candidato, true
		}
	}
	if !encontrado {
		t.Fatal("token SoftHSM generado ausente")
	}
	sesion, err := ctx.OpenSession(slot, pkcs11.CKF_SERIAL_SESSION|pkcs11.CKF_RW_SESSION)
	if err != nil {
		t.Fatal(err)
	}
	if err := ctx.Login(sesion, pkcs11.CKU_SO, soPIN); err != nil {
		t.Fatal(err)
	}
	if err := ctx.InitPIN(sesion, userPIN); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Logout(sesion); err != nil {
		t.Fatal(err)
	}
	if err := ctx.Login(sesion, pkcs11.CKU_USER, userPIN); err != nil {
		t.Fatal(err)
	}
	id := []byte{0x27, 0x01}
	_, err = ctx.GenerateKey(sesion,
		[]*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_GENERIC_SECRET_KEY_GEN, nil)},
		[]*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_SECRET_KEY),
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_GENERIC_SECRET),
			pkcs11.NewAttribute(pkcs11.CKA_ID, id),
			pkcs11.NewAttribute(pkcs11.CKA_LABEL, "vec-hmac-prueba-v1"),
			pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
			pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
			pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
			pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, false),
			pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
			pkcs11.NewAttribute(pkcs11.CKA_VALUE_LEN, 32),
		})
	if err != nil {
		t.Fatalf("no se pudo generar clave HMAC dentro de SoftHSM: %v", err)
	}
	idExportable := []byte{0x27, 0x02}
	_, err = ctx.GenerateKey(sesion,
		[]*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_GENERIC_SECRET_KEY_GEN, nil)},
		[]*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_SECRET_KEY),
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_GENERIC_SECRET),
			pkcs11.NewAttribute(pkcs11.CKA_ID, idExportable),
			pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
			pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
			pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
			pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, true),
			pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
			pkcs11.NewAttribute(pkcs11.CKA_VALUE_LEN, 32),
		})
	if err != nil {
		t.Fatalf("no se pudo preparar clave exportable para rechazo: %v", err)
	}
	info, err := ctx.GetTokenInfo(slot)
	if err != nil {
		t.Fatal(err)
	}
	_ = ctx.Logout(sesion)
	_ = ctx.CloseSession(sesion)
	_ = ctx.Finalize()
	ctx.Destroy()

	pinRuta := filepath.Join(directorio, "pin")
	if err := os.WriteFile(pinRuta, []byte(userPIN), 0600); err != nil {
		t.Fatal(err)
	}
	config := Configuracion{
		Modulo: modulo, TokenLabel: label, TokenSerial: strings.TrimSpace(info.SerialNumber),
		ObjetoID: id, ClaveID: "vec-hmac-prueba-v1", ClaveVersion: 1,
		DominioRef: "idh_0123456789012345678901", EspacioIdentidad: "https://idp.example.test",
		PINFichero: pinRuta,
	}
	conector, err := Abrir(config)
	if err != nil {
		t.Fatal(err)
	}
	entrada := registro.IdentificadoresAlta{
		EspacioIdentidad: config.EspacioIdentidad,
		AsercionID:       "mismo", SesionID: "mismo", SujetoID: "mismo",
		CuentaID: "mismo", CuentaOrdinariaID: "mismo",
	}
	resultado, err := conector.SeudonimizarAlta(context.Background(), entrada)
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Esquema != registro.EsquemaHMACSHA256V1 || resultado.DominioRef != config.DominioRef ||
		resultado.ClaveID != config.ClaveID || resultado.ClaveVersion != 1 {
		t.Fatal("coordenadas HMAC incorrectas")
	}
	huellas := [][32]byte{resultado.AsercionIDHMAC, resultado.SesionIDHMAC,
		resultado.SujetoIDHMAC, resultado.CuentaIDHMAC, resultado.CuentaOrdinariaIDHMAC}
	for i := range huellas {
		if huellas[i] == [32]byte{} {
			t.Fatal("HMAC nulo")
		}
		for j := i + 1; j < len(huellas); j++ {
			if huellas[i] == huellas[j] {
				t.Fatal("propositos HMAC no separados")
			}
		}
	}
	repetido, err := conector.SeudonimizarAlta(context.Background(), entrada)
	if err != nil || resultado != repetido {
		t.Fatal("HMAC inestable")
	}
	entrada.CuentaOrdinariaID = ""
	sinOrdinaria, err := conector.SeudonimizarAlta(context.Background(), entrada)
	if err != nil || sinOrdinaria.CuentaOrdinariaIDHMAC != [32]byte{} {
		t.Fatal("cuenta ordinaria opcional mal seudonimizada")
	}
	entrada.EspacioIdentidad = "https://otro.example.test"
	if _, err := conector.SeudonimizarAlta(context.Background(), entrada); err == nil {
		t.Fatal("se admitio otro espacio de identidad")
	}
	if err := conector.Cerrar(); err != nil {
		t.Fatal(err)
	}
	if _, err := conector.SeudonimizarAlta(context.Background(), entrada); err == nil {
		t.Fatal("se admitio conector cerrado")
	}
	config.ObjetoID = []byte{0xff}
	if _, err := Abrir(config); err == nil {
		t.Fatal("se admitio objeto de clave ausente")
	}
	config.ObjetoID = idExportable
	if _, err := Abrir(config); err == nil {
		t.Fatal("se admitio clave exportable")
	}
	config.ObjetoID = id
	config.TokenSerial = "serial-ajeno"
	if _, err := Abrir(config); err == nil {
		t.Fatal("se admitio token ajeno")
	}
}

func pinAleatorio(t *testing.T) string {
	t.Helper()
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(b[:])
}
