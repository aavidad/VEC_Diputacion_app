package ports

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestEstadoProtegidoFlujoFirmaBloqueaSerializacionYFormateo(t *testing.T) {
	estado, err := NuevoEstadoProtegidoFlujoFirmaBaremacion(
		AlgoritmoProteccionEstadoAES256GCM,
		"clave-estado-flujo-firma-v1",
		bytes.Repeat([]byte{0x11}, 12),
		bytes.Repeat([]byte{0x22}, 32),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := json.Marshal(estado); !errors.Is(err, ErrSerializacionEstadoFlujoProhibida) {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if _, err := estado.MarshalText(); !errors.Is(err, ErrSerializacionEstadoFlujoProhibida) {
		t.Fatalf("MarshalText() error = %v", err)
	}
	formateado := fmt.Sprintf("%v|%#v|%+v", estado, estado, estado)
	if strings.Contains(formateado, "111111") || strings.Contains(formateado, "222222") ||
		strings.Count(formateado, "[ESTADO-FLUJO-FIRMA-PROTEGIDO]") != 3 {
		t.Fatalf("el formateo filtro el sobre: %q", formateado)
	}

	datos, err := estado.DatosPersistencia()
	if err != nil {
		t.Fatal(err)
	}
	datos.Nonce[0] ^= 0xff
	datos.Cifrado[0] ^= 0xff
	if estado.Validar() != nil {
		t.Fatal("DatosPersistencia devolvio alias mutables sobre el estado")
	}
}

func TestRepresentacionCanonicaFlujoFirmaLigaNonceAEAD(t *testing.T) {
	cifrado := bytes.Repeat([]byte{0x44}, 32)
	estadoA, err := NuevoEstadoProtegidoFlujoFirmaBaremacion(
		AlgoritmoProteccionEstadoAES256GCM,
		"clave-estado-flujo-firma-v1",
		bytes.Repeat([]byte{0x01}, 12),
		cifrado,
	)
	if err != nil {
		t.Fatal(err)
	}
	estadoB, err := NuevoEstadoProtegidoFlujoFirmaBaremacion(
		AlgoritmoProteccionEstadoAES256GCM,
		"clave-estado-flujo-firma-v1",
		bytes.Repeat([]byte{0x02}, 12),
		cifrado,
	)
	if err != nil {
		t.Fatal(err)
	}
	instante := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	expediente := ExpedienteFlujoFirmaBaremacion{
		FlujoRef: "flujo-firma-001", Version: 1,
		IndiceIdempotenciaHMAC: hmacFlujoFirmaPuertosPrueba("1"),
		HuellaSolicitudHMAC:    hmacFlujoFirmaPuertosPrueba("2"),
		VinculoActorHMAC:       hmacFlujoFirmaPuertosPrueba("3"),
		PerfilActorClave:       "perfil_rrhh",
		ProcesoRef:             "proceso-001",
		SolicitudRef:           "solicitud-001",
		BaremacionMeritoRef:    "baremacion-001",
		DecisionRef:            "decision-001",
		Estado:                 EstadoExpedienteFirmaPreparando,
		EstadoProtegido:        estadoA,
		CreadoEn:               instante,
		ActualizadoEn:          instante,
	}
	canonicaA, err := RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(expediente)
	if err != nil {
		t.Fatal(err)
	}
	expediente.EstadoProtegido = estadoB
	canonicaB, err := RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(expediente)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(canonicaA.Revelar(), canonicaB.Revelar()) {
		t.Fatal("la representacion sellada no ligo el nonce AEAD")
	}
}

func hmacFlujoFirmaPuertosPrueba(caracter string) string {
	return "hmac-sha256:flujo_firma_prueba_v1:" + strings.Repeat(caracter, 64)
}
