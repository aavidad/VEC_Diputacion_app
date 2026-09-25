package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

func huellaContratoPrueba(contenido string) string {
	s := sha256.Sum256([]byte(contenido))
	return hex.EncodeToString(s[:])
}

func eventoRefPrueba(tipo, origen string) string {
	s := sha256.Sum256([]byte(tipo + "\x1f" + origen))
	return prefijoEventoContratoParticipacion + hex.EncodeToString(s[:])
}

// eventoContratoPrueba reproduce la forma de CT113 (jsonb::text).
func eventoContratoPrueba(reemplazos ...string) string {
	e := `{"tipo": "incorporacion", "inicio": "2027-01-04T00:00:00.000000Z", "esquema": "vec.contratacion-temporal.contrato-bolsa.v1", "evento_ref": "` + eventoRefPrueba("incorporacion", "ref:outbox:1") + `", "origen_ref": "ref:outbox:1", "causa_clave": "necesidad_temporal", "ocurrido_en": "2027-01-02T09:00:01.000000Z", "fin_previsto": "2027-03-31T00:00:00.000000Z", "categoria_ref": "categoria:desarrollo:c2", "expediente_ref": "expediente:ct:1", "llamamiento_ref": "llamamiento:1", "modalidad_clave": "sustitucion", "organizacion_ref": "organizacion:desarrollo:dipgra"}`
	for i := 0; i+1 < len(reemplazos); i += 2 {
		e = strings.Replace(e, reemplazos[i], reemplazos[i+1], 1)
	}
	return e
}

func TestDecodificarEventoContratoParticipacionValido(t *testing.T) {
	c := eventoContratoPrueba()
	e, err := DecodificarEventoContratoParticipacion([]byte(c), huellaContratoPrueba(c))
	if err != nil {
		t.Fatal(err)
	}
	if e.Tipo != "incorporacion" || e.LlamamientoRef != "llamamiento:1" || e.Inicio == nil || e.FinPrevisto == nil ||
		e.Inicio.Day() != 4 || e.FinPrevisto.Month() != 3 || e.ModalidadClave != "sustitucion" || e.OcurridoEn.IsZero() {
		t.Fatalf("evento=%+v", e)
	}
	sinFechas := eventoContratoPrueba(`"inicio": "2027-01-04T00:00:00.000000Z"`, `"inicio": null`,
		`"fin_previsto": "2027-03-31T00:00:00.000000Z"`, `"fin_previsto": null`, `"causa_clave": "necesidad_temporal"`, `"causa_clave": null`)
	e, err = DecodificarEventoContratoParticipacion([]byte(sinFechas), huellaContratoPrueba(sinFechas))
	if err != nil || e.Inicio != nil || e.FinPrevisto != nil || e.CausaClave != "" {
		t.Fatalf("nulos admitidos: %+v %v", e, err)
	}
}

func TestDecodificarEventoContratoParticipacionRechazos(t *testing.T) {
	casos := map[string]string{
		"campo desconocido": eventoContratoPrueba(`"tipo": "incorporacion",`, `"tipo": "incorporacion", "dni": "x",`),
		"campo ausente":     eventoContratoPrueba(`"causa_clave": "necesidad_temporal", `, ``),
		"clave duplicada":   eventoContratoPrueba(`"tipo": "incorporacion",`, `"tipo": "incorporacion", "tipo": "incorporacion",`),
		"esquema ajeno":     eventoContratoPrueba(`contrato-bolsa.v1`, `contrato-bolsa.v2`),
		"evento_ref libre":  eventoContratoPrueba(`"origen_ref": "ref:outbox:1"`, `"origen_ref": "ref:outbox:2"`),
		"fecha sin hora":    eventoContratoPrueba(`"inicio": "2027-01-04T00:00:00.000000Z"`, `"inicio": "2027-01-04"`),
		"fin antes":         eventoContratoPrueba(`"fin_previsto": "2027-03-31T00:00:00.000000Z"`, `"fin_previsto": "2026-03-31T00:00:00.000000Z"`),
		"sin ocurrido":      eventoContratoPrueba(`"ocurrido_en": "2027-01-02T09:00:01.000000Z"`, `"ocurrido_en": null`),
		"clave vacía":       eventoContratoPrueba(`"modalidad_clave": "sustitucion"`, `"modalidad_clave": ""`),
		"clave inválida":    eventoContratoPrueba(`"modalidad_clave": "sustitucion"`, `"modalidad_clave": "Sustitución"`),
		"ref no opaca":      eventoContratoPrueba(`"expediente_ref": "expediente:ct:1"`, `"expediente_ref": "expediente ct 1"`),
		"no objeto":         `[]`,
	}
	for nombre, c := range casos {
		if _, err := DecodificarEventoContratoParticipacion([]byte(c), huellaContratoPrueba(c)); !errors.Is(err, ErrEventoContratoParticipacionInvalido) {
			t.Errorf("%s: aceptado (%v)", nombre, err)
		}
	}
	c := eventoContratoPrueba()
	if _, err := DecodificarEventoContratoParticipacion([]byte(c), strings.Repeat("0", 64)); err == nil {
		t.Error("huella falsa aceptada")
	}
	if _, err := DecodificarEventoContratoParticipacion([]byte(strings.Repeat(" ", maximoEventoContratoParticipacion+1)), huellaContratoPrueba(strings.Repeat(" ", maximoEventoContratoParticipacion+1))); err == nil {
		t.Error("evento sobredimensionado aceptado")
	}
}
