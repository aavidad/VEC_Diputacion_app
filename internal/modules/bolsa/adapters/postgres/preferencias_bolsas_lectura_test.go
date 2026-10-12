package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

func TestErrorLecturaPreferenciasNoConfundeACLConDenegacion(t *testing.T) {
	para := func(mensaje string) error { return &pgconn.PgError{Code: "42501", Message: mensaje} }
	if !errors.Is(errorLecturaPreferencias(context.Background(), para("AD246: autoridad actual de lectura denegada")), dominiovec.ErrAutorizacionDenegada) {
		t.Fatal("denegación nominal perdió clasificación")
	}
	if !errors.Is(errorLecturaPreferencias(context.Background(), para("permission denied for function consultar_preferencias_bolsas_v1")), bolsa.ErrPreferenciasNoDisponibles) {
		t.Fatal("ACL técnica confundida con denegación")
	}
}

func TestDecodificarLecturaPreferenciasExigeAcuseYProyeccionExactos(t *testing.T) {
	fecha := time.Now().UTC().Truncate(time.Microsecond).Format("2006-01-02T15:04:05.000000Z")
	base := fmt.Sprintf(`{"estado":{"version":0,"orden_guardado":[],"bolsas_actuales":["bolsa:propia"],"bolsas_nuevas":["bolsa:propia"],"bolsas_retiradas":[],"revision_requerida":true,"recibo_ref":null,"registrado_en":null},"auditoria_ref":"aud_v3_lp_%s","auditoria_huella_sha256":"%s","correlacion_ref":"correlacion:propia","auditada_en":%q}`, strings.Repeat("a", 32), strings.Repeat("b", 64), fecha)
	estado, err := decodificarLecturaPreferencias([]byte(base), "correlacion:propia")
	if err != nil || estado.Version != 0 || len(estado.BolsasNuevas) != 1 {
		t.Fatalf("lectura legítima: %+v %v", estado, err)
	}
	variantes := []string{
		strings.Replace(base, `"correlacion_ref":"correlacion:propia"`, `"correlacion_ref":"correlacion:ajena"`, 1),
		strings.Replace(base, `"auditoria_ref":"aud_v3_lp_`, `"auditoria_ref":"falsa_`, 1),
		strings.Replace(base, `"revision_requerida":true`, `"revision_requerida":false`, 1),
		strings.Replace(base, `"bolsas_nuevas":["bolsa:propia"]`, `"bolsas_nuevas":[]`, 1),
		strings.Replace(base, `"registrado_en":null`, `"registrado_en":"2026-01-01T00:00:00Z"`, 1),
		strings.Replace(base, `"auditada_en":`, `"candidato_ref":"can_ajeno","auditada_en":`, 1),
		strings.Replace(base, `"version":0`, `"version":0,"version":1`, 1),
	}
	for _, v := range variantes {
		if _, err := decodificarLecturaPreferencias([]byte(v), "correlacion:propia"); !errors.Is(err, bolsa.ErrPreferenciasNoDisponibles) {
			t.Fatalf("respuesta sin autoridad aceptada: %s: %v", v, err)
		}
	}
	malformado := strings.Replace(base, `"version":0`, `"version":?`, 1)
	_, err = decodificarLecturaPreferencias([]byte(malformado), "correlacion:propia")
	var sintaxis *json.SyntaxError
	if !errors.Is(err, bolsa.ErrPreferenciasNoDisponibles) || !errors.As(err, &sintaxis) {
		t.Fatalf("error de JSON no propagado con clasificación segura: %v", err)
	}
}
