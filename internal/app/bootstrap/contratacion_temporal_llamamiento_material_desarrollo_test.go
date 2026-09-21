package bootstrap

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestDerivarMaterialConsumidorAplicaDominioYPrefijoNominales(t *testing.T) {
	cfg, _ := generarMaterialDesarrolloPrueba(t)
	composicion, err := NuevaComposicionSeguridadDesarrollo(cfg, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	base, err := nuevoMaterialAtestacionContratacionTemporalDesarrollo(composicion.derivadorIdempotencia, time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	defer base.borrarCopiasEfimeras()
	casos := append([]descriptorMaterialConsumidorV3Desarrollo{
		{Audiencia: puertosbolsa.AudienciaIntegracionLlamamientoDesarrollo, Dominio: "vec.bolsa.desarrollo.capacidad-v3", Prefijo: "clave:capacidad:bolsa:", ProveedorNominal: proveedorMaterialContratacionTemporal},
	}, descriptoresMaterialAutorizacionContratacionTemporalDesarrollo()...)
	if len(casos) != 5 {
		t.Fatalf("audiencias históricas inesperadas: %d", len(casos))
	}
	for _, d := range casos {
		t.Run(d.Audiencia, func(t *testing.T) {
			derivado, err := derivarMaterialConsumidorV3Desarrollo(base, d)
			if err != nil {
				t.Fatal(err)
			}
			defer borrarBytes(derivado.claveHMAC)
			mac := hmac.New(sha256.New, base.claveHMAC)
			_, _ = mac.Write([]byte(d.Dominio + ".v1"))
			esperada := mac.Sum(nil)
			if !bytes.Equal(derivado.claveHMAC, esperada) || derivado.claveHMACID != d.Prefijo+strings.TrimPrefix(base.claveHMACID, "clave:capacidad:ct:") || derivado.audienciaConsumo != d.Audiencia || derivado.emisorID != base.emisorID {
				t.Fatalf("derivación nominal inesperada: id=%q audiencia=%q emisor=%q", derivado.claveHMACID, derivado.audienciaConsumo, derivado.emisorID)
			}
			secreto := sha256.Sum256(esperada)
			if derivado.claveHMACSecreto != hex.EncodeToString(secreto[:]) {
				t.Fatal("secreto derivado no preservado")
			}
			huella := sha256.New()
			_, _ = huella.Write([]byte(d.Dominio + ".gobierno.v1\x00"))
			_, _ = huella.Write(esperada)
			if derivado.claveHMACHuella != hex.EncodeToString(huella.Sum(nil)) {
				t.Fatal("huella de gobierno no incorpora dominio y clave")
			}
		})
	}
}
