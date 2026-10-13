package bootstrap

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/shared/telemetria"
)

func TestSelectorCTArranqueConservaCausaSinImprimirSecreto(t *testing.T) {
	var registro bytes.Buffer
	anterior := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&registro, nil)))
	t.Cleanup(func() { slog.SetDefault(anterior) })
	centinela := errors.New("capacidad no disponible")
	causa := errors.New("postgres://usuario:secreto@servidor/base?identidad=persona")
	err := diagnosticarSelectorCTArranque(envCTFirmasR5V2Enabled, "ct_firmas_r5_v2", "auditoria_configuracion",
		"VEC_DEVELOPMENT_MATERIAL_DIR: auditoria-intentos.json", centinela, causa)
	if !errors.Is(err, centinela) || !errors.Is(err, causa) {
		t.Fatal("se ha perdido la causa o el centinela")
	}
	if ComponenteFalloArranque(err) != "ct_firmas_r5_v2_auditoria_configuracion" {
		t.Fatal("se ha perdido la etapa de composición")
	}
	if telemetria.ClaseErrorArranque(err) != "configuracion" {
		t.Fatal("el error de configuración no está clasificado")
	}
	if telemetria.MensajeErrorArranque(err) != mensajeFuenteConfiguracionArranque {
		t.Fatal("el mensaje raíz no es el diagnóstico seguro catalogado")
	}
	for _, dato := range []string{"postgres://", "secreto", "persona", "usuario"} {
		if strings.Contains(registro.String(), dato) || strings.Contains(err.Error(), dato) {
			t.Fatalf("el diagnóstico revela %s", dato)
		}
	}
	if !strings.Contains(registro.String(), "auditoria-intentos.json") || !strings.Contains(registro.String(), envCTFirmasR5V2Enabled) {
		t.Fatal("el registro no identifica el selector y su requisito")
	}
}
