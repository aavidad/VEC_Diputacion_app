package postgres

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
)

func TestJSONPlazoOfertaUsaFechaCanonicaDelMaterialSQL(t *testing.T) {
	for _, caso := range []struct {
		nanos    int
		esperada string
	}{
		{0, "2026-10-02T09:00:00.000000Z"},
		{123400000, "2026-10-02T09:00:00.123400Z"},
	} {
		notificacion := dominiobolsa.NotificacionOferta{NotificadaEn: time.Date(2026, 10, 2, 9, 0, 0, caso.nanos, time.UTC), ReferenciaCorreo: "correo:extracto:oferta-1", HuellaCorreoSHA256: strings.Repeat("a", 64), Fuente: "correo_externo_declarado_rrhh"}
		p := ports.PlazoOferta{Notificacion: &notificacion, ReglaRef: "politica-ofertas:bolsa:of:1", HuellaCatalogo: strings.Repeat("b", 64), Unidad: "horas_naturales", Cantidad: 48, Computo: "continuo_utc", UltimoDia: "2026-10-04", Ejemplo: true, Calendarios: []string{"calendario:utc-continuo:v1"}, PoliticaVersion: 1, MunicipioSede: "18087", AperturaEn: caso.esperada, VenceEn: notificacion.NotificadaEn.Add(48 * time.Hour).Format("2006-01-02T15:04:05.000000Z")}
		b, err := serializarPlazoOfertaPostgreSQL(p)
		if err != nil {
			t.Fatal(err)
		}
		var datos map[string]json.RawMessage
		if err = json.Unmarshal(b, &datos); err != nil {
			t.Fatal(err)
		}
		var n map[string]string
		if err = json.Unmarshal(datos["notificacion"], &n); err != nil {
			t.Fatal(err)
		}
		if len(datos) != 13 || len(n) != 4 || n["notificada_en"] != caso.esperada || n["notificada_en"] != p.AperturaEn || n["referencia_correo"] != notificacion.ReferenciaCorreo {
			t.Fatalf("JSON SQL divergente: %s", b)
		}
		var recuperado ports.PlazoOferta
		if err = json.Unmarshal(b, &recuperado); err != nil || recuperado.Notificacion == nil || !recuperado.Notificacion.NotificadaEn.Equal(notificacion.NotificadaEn) {
			t.Fatalf("fecha de dominio alterada: %s err=%v", b, err)
		}
		t.Logf("JSON B71 generado por el adaptador: %s", b)
	}
}
