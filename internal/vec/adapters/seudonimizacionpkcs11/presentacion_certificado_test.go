package seudonimizacionpkcs11

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/miekg/pkcs11"

	registro "vec-diputacion-granada/internal/vec/adapters/httpseguridad/postgres"
)

// Los digestos se calcularon con un generador Python independiente de este
// paquete: HMAC-SHA-256 sobre seis campos UTF-8 con longitud uint32 BE.
func TestPreimagenPresentacionConservaAltaYSeparaPropositos(t *testing.T) {
	config := Configuracion{
		DominioRef: "idh_0123456789012345678901", EspacioIdentidad: "https://idp.example.test",
		ClaveVersion: 7,
	}
	clave := make([]byte, 32)
	for i := range clave {
		clave[i] = 0x42 // material sintetico exclusivo de esta prueba
	}
	casos := []struct {
		proposito, valor, esperado string
	}{
		{"asercion", "asr-v-01", "119d43c3a9da7613d7fd8dcc1b6d8720e5dbf153587220bc54dad5f211391ffb"},
		{"sesion", "ses-v-01", "e51be0568b523615e64c4c00ee274b152492b88290d4468510de7444e793c088"},
		{"sujeto", "suj-v-01", "5e89bdeef2fbdc09af6d1d315ab4e3cb096b48d314a514d7656186698d77a47b"},
		{"cuenta", "cue-v-01", "fc4e89ad0299af8fc2d4331e5d1631afd8a181858c3e4900d2c29c081585739e"},
		{propositoCertificadoDER, "sha256:" + strings.Repeat("1", 64), "79f8221a8e3aee887050abac79b937a0969248a2cb7b81037dc5eb0ba259ce1a"},
		{propositoCADer, "sha256:" + strings.Repeat("2", 64), "8a14c8b483eb031ed9cfd3759a14d736bae0c4c9322cb111c0533f9556f112be"},
		{propositoPresentacionID, "asr-v-01", "edb2869270f5816924695b0d21bab328dfea22a8ba10d0ae20bd8a123fc01356"},
	}
	for _, caso := range casos {
		t.Run(caso.proposito, func(t *testing.T) {
			mensaje := mensajeCanonico(config, caso.proposito, caso.valor)
			mac := hmac.New(sha256.New, clave)
			_, _ = mac.Write(mensaje)
			if got := hex.EncodeToString(mac.Sum(nil)); got != caso.esperado {
				t.Fatalf("preimagen del proposito %s diverge del vector independiente", caso.proposito)
			}
			if got := mensajeManual(config, caso.proposito, caso.valor); !hmac.Equal(got, mensaje) {
				t.Fatalf("encuadre de longitud divergente en %s", caso.proposito)
			}
		})
	}
	clear(clave)
}

// Esta construcción de preimagen pertenece solo al vector de prueba. No se
// usa en el conector ni comparte su helper de serialización.
func mensajeManual(c Configuracion, proposito, valor string) []byte {
	campos := []string{registro.EsquemaHMACSHA256V1, c.DominioRef, c.EspacioIdentidad,
		"7", proposito, valor}
	var salida []byte
	for _, campo := range campos {
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(campo)))
		salida = append(salida, n[:]...)
		salida = append(salida, campo...)
	}
	return salida
}

func TestPresentacionCertificadoRechazaEntradaAntesDeUsarHSM(t *testing.T) {
	c := &Conector{config: Configuracion{EspacioIdentidad: "https://idp.example.test"}}
	ids := registro.IdentificadoresPresentacionCertificado{
		EspacioIdentidad: c.config.EspacioIdentidad,
		AsercionID:       "asr-v-01", SesionIDAfirmada: "ses-v-01",
		SujetoID: "suj-v-01", CuentaID: "cue-v-01", NonceID: "asr-v-01",
		CertificadoSHA256: "sha256:" + strings.Repeat("1", 64),
		CASHA256:          "sha256:" + strings.Repeat("2", 64),
	}
	casos := []struct {
		nombre string
		mutar  func(*registro.IdentificadoresPresentacionCertificado)
	}{
		{"espacio", func(i *registro.IdentificadoresPresentacionCertificado) {
			i.EspacioIdentidad = "https://otro.example.test"
		}},
		{"nonce_ajeno", func(i *registro.IdentificadoresPresentacionCertificado) { i.NonceID = "otro" }},
		{"certificado_no_canonico", func(i *registro.IdentificadoresPresentacionCertificado) {
			i.CertificadoSHA256 = "sha256:" + strings.Repeat("A", 64)
		}},
		{"certificado_nulo", func(i *registro.IdentificadoresPresentacionCertificado) {
			i.CertificadoSHA256 = "sha256:" + strings.Repeat("0", 64)
		}},
		{"ca_igual", func(i *registro.IdentificadoresPresentacionCertificado) { i.CASHA256 = i.CertificadoSHA256 }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			copia := ids
			caso.mutar(&copia)
			if salida, err := c.SeudonimizarPresentacionCertificado(context.Background(), copia); err == nil || salida != (registro.SeudonimosPresentacionCertificado{}) {
				t.Fatalf("entrada no acreditada alcanzo HSM: %v", err)
			}
		})
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := c.SeudonimizarPresentacionCertificado(ctx, ids); err != context.Canceled {
		t.Fatalf("cancelacion no conservada: %v", err)
	}
}

// SoftHSM es opcional en el ejecutor. Cuando está disponible, esta prueba usa
// una clave no extraíble generada dentro del token y compara las cuatro
// huellas con SeudonimizarAlta; las tres nuevas se contrastan con mensajes de
// propósito construidos independientemente y firmados por ese mismo token.
func TestSoftHSMPresentacionComparteEpochYSeparaSietePropositos(t *testing.T) {
	modulo := os.Getenv("VEC_TEST_SOFTHSM_MODULE")
	if modulo == "" {
		t.Skip("SoftHSM no configurado para prueba PKCS#11")
	}
	conector := abrirTokenPresentacionPrueba(t, modulo)
	t.Cleanup(func() { _ = conector.Cerrar() })
	ids := registro.IdentificadoresPresentacionCertificado{
		EspacioIdentidad: conector.config.EspacioIdentidad,
		AsercionID:       "asr-v-01", SesionIDAfirmada: "ses-v-01",
		SujetoID: "suj-v-01", CuentaID: "cue-v-01", NonceID: "asr-v-01",
		CertificadoSHA256: "sha256:" + strings.Repeat("1", 64),
		CASHA256:          "sha256:" + strings.Repeat("2", 64),
	}
	alto, err := conector.SeudonimizarAlta(context.Background(), registro.IdentificadoresAlta{
		EspacioIdentidad: ids.EspacioIdentidad, AsercionID: ids.AsercionID,
		SesionID: ids.SesionIDAfirmada, SujetoID: ids.SujetoID, CuentaID: ids.CuentaID,
	})
	if err != nil {
		t.Fatal(err)
	}
	actual, err := conector.SeudonimizarPresentacionCertificado(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	if actual.Esquema != alto.Esquema || actual.DominioRef != alto.DominioRef ||
		actual.EspacioIdentidad != alto.EspacioIdentidad || actual.ClaveID != alto.ClaveID ||
		actual.ClaveVersion != alto.ClaveVersion ||
		actual.AsercionIDHMAC != alto.AsercionIDHMAC || actual.SesionIDHMAC != alto.SesionIDHMAC ||
		actual.SujetoIDHMAC != alto.SujetoIDHMAC || actual.CuentaIDHMAC != alto.CuentaIDHMAC {
		t.Fatal("los cuatro HMAC de alta no coinciden con presentacion")
	}
	nuevas := []struct {
		proposito, valor string
		huella           [32]byte
	}{
		{propositoCertificadoDER, ids.CertificadoSHA256, actual.CertificadoDERHMAC},
		{propositoCADer, ids.CASHA256, actual.CAHMAC},
		{propositoPresentacionID, ids.NonceID, actual.NonceHMAC},
	}
	for _, nueva := range nuevas {
		mensaje := mensajeManual(conector.config, nueva.proposito, nueva.valor)
		esperada, err := conector.firmar(mensaje)
		clear(mensaje)
		if err != nil || nueva.huella == [32]byte{} || nueva.huella != esperada {
			t.Fatalf("HMAC PKCS#11 incompatible para %s: %v", nueva.proposito, err)
		}
	}
	huellaTodas := [][32]byte{actual.AsercionIDHMAC, actual.SesionIDHMAC,
		actual.SujetoIDHMAC, actual.CuentaIDHMAC, actual.CertificadoDERHMAC,
		actual.CAHMAC, actual.NonceHMAC}
	for i := range huellaTodas {
		for j := i + 1; j < len(huellaTodas); j++ {
			if huellaTodas[i] == huellaTodas[j] {
				t.Fatal("dos propositos compartieron HMAC")
			}
		}
	}
	repetida, err := conector.SeudonimizarPresentacionCertificado(context.Background(), ids)
	if err != nil || actual != repetida {
		t.Fatal("presentacion PKCS#11 inestable")
	}
}

func abrirTokenPresentacionPrueba(t *testing.T, modulo string) *Conector {
	t.Helper()
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
	ctx := pkcs11.New(modulo)
	if ctx == nil || ctx.Initialize() != nil {
		t.Fatal("SoftHSM de prueba no disponible")
	}
	slots, err := ctx.GetSlotList(false)
	if err != nil || len(slots) == 0 {
		t.Fatalf("sin slot SoftHSM: %v", err)
	}
	soPIN, userPIN := pinAleatorio(t), pinAleatorio(t)
	const label = "vec-presentacion-prueba"
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
		info, err := ctx.GetTokenInfo(candidato)
		if err == nil && strings.TrimSpace(info.Label) == label {
			slot, encontrado = candidato, true
		}
	}
	if !encontrado {
		t.Fatal("token SoftHSM de prueba ausente")
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
	id := []byte{0x27, 0x18}
	_, err = ctx.GenerateKey(sesion,
		[]*pkcs11.Mechanism{pkcs11.NewMechanism(pkcs11.CKM_GENERIC_SECRET_KEY_GEN, nil)},
		[]*pkcs11.Attribute{
			pkcs11.NewAttribute(pkcs11.CKA_CLASS, pkcs11.CKO_SECRET_KEY),
			pkcs11.NewAttribute(pkcs11.CKA_KEY_TYPE, pkcs11.CKK_GENERIC_SECRET),
			pkcs11.NewAttribute(pkcs11.CKA_ID, id),
			pkcs11.NewAttribute(pkcs11.CKA_TOKEN, true),
			pkcs11.NewAttribute(pkcs11.CKA_PRIVATE, true),
			pkcs11.NewAttribute(pkcs11.CKA_SENSITIVE, true),
			pkcs11.NewAttribute(pkcs11.CKA_EXTRACTABLE, false),
			pkcs11.NewAttribute(pkcs11.CKA_SIGN, true),
			pkcs11.NewAttribute(pkcs11.CKA_VALUE_LEN, 32),
		})
	if err != nil {
		t.Fatalf("clave HMAC de prueba no provisionada: %v", err)
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
	conector, err := Abrir(Configuracion{
		Modulo: modulo, TokenLabel: label, TokenSerial: strings.TrimSpace(info.SerialNumber),
		ObjetoID: id, ClaveID: "vec-hmac-presentacion-prueba-v1", ClaveVersion: 7,
		DominioRef: "idh_0123456789012345678901", EspacioIdentidad: "https://idp.example.test",
		PINFichero: pinRuta,
	})
	if err != nil {
		t.Fatal(err)
	}
	return conector
}
