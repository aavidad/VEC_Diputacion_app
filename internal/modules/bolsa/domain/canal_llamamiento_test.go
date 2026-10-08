package domain

import (
	"os"
	"strings"
	"testing"
)

func TestCatalogoCanalesVersionadoCorreoYTelefonoActivosSMSYTelegramApagados(t *testing.T) {
	datos, err := os.ReadFile("../../../../config/bolsa_canales_llamamiento_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParsearCatalogoCanalesLlamamiento(datos)
	if err != nil {
		t.Fatalf("catálogo versionado: %v", err)
	}
	activos := map[string]bool{}
	for _, canal := range c.Canales {
		activos[canal.Canal] = canal.Activo
	}
	if !activos[CanalAvisoCorreo] || !activos[CanalAvisoTelefono] || activos[CanalAvisoSMS] || activos[CanalAvisoTelegram] || len(activos) != 4 {
		t.Fatalf("activación por defecto: %v", activos)
	}
}

func TestCatalogoCanalesRechazaGarantiasRotas(t *testing.T) {
	base := `{"version":"bolsa-canales-llamamiento-v1","canales":[{"canal":"correo","modo":"automatico","al_emitir":true,"seguimiento":false,"acuse":false,"admite_respuesta":false,"resultados":["enviado"],"activo":true}%s]}`
	malos := []string{
		`,{"canal":"telegram","modo":"automatico","al_emitir":false,"seguimiento":true,"acuse":true,"admite_respuesta":false,"resultados":["enviado"],"limite_caracteres":100,"activo":false}`,
		`,{"canal":"telefono","modo":"automatico","al_emitir":false,"seguimiento":true,"acuse":false,"admite_respuesta":true,"resultados":["contactado"],"activo":true}`,
		`,{"canal":"sms","modo":"automatico","al_emitir":false,"seguimiento":true,"acuse":true,"admite_respuesta":false,"resultados":["enviado"],"activo":false}`,
		`,{"canal":"telefono","modo":"manual","al_emitir":false,"seguimiento":true,"acuse":false,"admite_respuesta":true,"resultados":["contactado"],"resultados_cierre":["acepta"],"activo":true}`,
		`,{"canal":"fax","modo":"manual","al_emitir":false,"seguimiento":true,"acuse":false,"admite_respuesta":false,"resultados":["enviado"],"activo":true}`,
		`,{"canal":"correo","modo":"automatico","al_emitir":true,"seguimiento":false,"acuse":false,"admite_respuesta":false,"resultados":["enviado"],"activo":true}`,
		`,{"canal":"telefono","modo":"manual","al_emitir":false,"seguimiento":true,"resultados":["inventado"],"activo":true}`,
	}
	if _, err := ParsearCatalogoCanalesLlamamiento([]byte(strings.Replace(base, "%s", "", 1))); err != nil {
		t.Fatalf("catálogo mínimo: %v", err)
	}
	for i, extra := range malos {
		if _, err := ParsearCatalogoCanalesLlamamiento([]byte(strings.Replace(base, "%s", extra, 1))); err == nil {
			t.Errorf("caso %d admitido", i)
		}
	}
	if _, err := ParsearCatalogoCanalesLlamamiento([]byte(`{"version":"bolsa-canales-llamamiento-v1","canales":[],"extra":1}`)); err == nil {
		t.Error("campo desconocido admitido")
	}
}
