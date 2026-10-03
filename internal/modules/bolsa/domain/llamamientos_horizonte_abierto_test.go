package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"
)

func TestNecesidadHorizonteAbiertoSinFechaInventada(t *testing.T) {
	bolsa := bolsaLlamamientoPrueba(t)
	alta := altaNecesidadLlamamientoPrueba(t, bolsa)
	alta.FinPrevisto = time.Time{}
	alta.CausaFinClave = "reincorporacion_titular"
	necesidad, err := NuevaNecesidadCobertura(alta)
	if err != nil {
		t.Fatal(err)
	}
	canonico, err := json.Marshal(necesidad)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(canonico, []byte(`"fin_previsto":null`)) ||
		!bytes.Contains(canonico, []byte(`"causa_fin_clave":"reincorporacion_titular"`)) ||
		bytes.Contains(canonico, []byte("0001-01-01")) {
		t.Fatalf("canon abierto incorrecto: %s", canonico)
	}
	var recuperada NecesidadCobertura
	if err := json.Unmarshal(canonico, &recuperada); err != nil || recuperada.Validar() != nil ||
		!recuperada.HorizonteVigenteEn(instanteLlamamientoPrueba.Add(365*24*time.Hour)) {
		t.Fatalf("necesidad abierta no recuperable: %v", err)
	}
	alta.CausaFinClave = ""
	if _, err := NuevaNecesidadCobertura(alta); err == nil {
		t.Fatal("horizonte sin causa admitido")
	}
	alta.FinPrevisto = instanteLlamamientoPrueba.Add(60 * 24 * time.Hour)
	alta.CausaFinClave = "reincorporacion_titular"
	if _, err := NuevaNecesidadCobertura(alta); err == nil {
		t.Fatal("fecha y causa simultáneas admitidas")
	}
}

func TestNecesidadFechadaConservaCanon(t *testing.T) {
	bolsa := bolsaLlamamientoPrueba(t)
	necesidad, err := NuevaNecesidadCobertura(altaNecesidadLlamamientoPrueba(t, bolsa))
	if err != nil {
		t.Fatal(err)
	}
	type canonAntiguo NecesidadCobertura
	previo, err := json.Marshal(canonAntiguo(necesidad))
	if err != nil {
		t.Fatal(err)
	}
	actual, err := json.Marshal(necesidad)
	if err != nil || !bytes.Equal(actual, previo) || bytes.Contains(actual, []byte("causa_fin_clave")) {
		t.Fatalf("canon fechado alterado: %v", err)
	}
	suma := sha256.Sum256(actual)
	if hex.EncodeToString(suma[:]) != "b227bd350566900a91beb772710328c2e49d6e88c797a0c4a473d73d7be3d8c8" {
		t.Fatal("huella del canon legado alterada")
	}
}
