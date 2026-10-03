package postgres

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func relacionUnidadCTSQLPrueba(t *testing.T) ([]byte, ports.SolicitudRelacionUnidadExpedienteCT) {
	t.Helper()
	q := ports.SolicitudRelacionUnidadExpedienteCT{OrganizacionRef: "organizacion:ct:sintetica", ExpedienteRef: "expediente:ct:sintetico", UnidadRefEsperada: "unidad:ct:gestora", VersionExpedienteSolicitada: 7}
	s := relacionUnidadExpedienteCTSQL{Esquema: esquemaRelacionUnidadExpedienteCT, OrganizacionRef: q.OrganizacionRef, ExpedienteRef: q.ExpedienteRef,
		UnidadRefEsperada: q.UnidadRefEsperada, VersionExpedienteSolicitada: q.VersionExpedienteSolicitada, UnidadRef: q.UnidadRefEsperada,
		UnidadSnapshotSolicitadoRef: q.UnidadRefEsperada, VersionExpedienteActual: 9, VersionOrigenVinculo: 4,
		OperacionOrigenRef: "reserva:ct:asignacion", ReservaAsignacionRef: "reserva:ct:asignacion", ReciboAsignacionRef: "recibo:ct:asignacion",
		PruebaSnapshotOrigenHuellaSHA256: strings.Repeat("a", 64), TipoEventoOrigen: ports.TipoEventoOrigenRelacionUnidadCT,
		EventoAsignacionRef: "evento:ct:asignacion", EventoPayloadHuellaSHA256: strings.Repeat("b", 64), AsignacionConfirmadaEn: "2026-10-01T10:00:00.123456Z"}
	contenido, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return contenido, q
}

func TestDecodificarRelacionUnidadCTConservaFuenteYVersiones(t *testing.T) {
	contenido, q := relacionUnidadCTSQLPrueba(t)
	r, err := DecodificarRelacionUnidadExpedienteCT(contenido, q)
	if err != nil {
		t.Fatal(err)
	}
	if r.Solicitud != q || r.VersionExpedienteActual != 9 || r.VersionOrigenVinculo != 4 || r.UnidadSnapshotSolicitadoRef != q.UnidadRefEsperada || r.UnidadRef != q.UnidadRefEsperada || r.AsignacionConfirmadaEn.Nanosecond() != 123456000 || r.AsignacionConfirmadaEn.Location() != time.UTC || r.PruebaSnapshotOrigenHuellaSHA256 != strings.Repeat("a", 64) {
		t.Fatalf("datos fuente alterados: %+v", r)
	}
}

func TestDecodificarRelacionUnidadCTRechazaJSONAjenoSinDatos(t *testing.T) {
	contenido, q := relacionUnidadCTSQLPrueba(t)
	for nombre, mutar := range map[string]func([]byte) []byte{
		"array":           func(c []byte) []byte { return append(append([]byte("["), c...), ']') },
		"campo_extra":     func(c []byte) []byte { return append([]byte(`{"campo_privado":"valor_privado",`), c[1:]...) },
		"clave_duplicada": func(c []byte) []byte { return append([]byte(`{"unidad_ref":"unidad:ct:gestora",`), c[1:]...) },
		"alias_mayuscula": func(c []byte) []byte { return bytes.Replace(c, []byte(`"unidad_ref"`), []byte(`"UNIDAD_REF"`), 1) },
		"campo_ausente":   func(c []byte) []byte { return bytes.Replace(c, []byte(`"version_origen_vinculo":4,`), nil, 1) },
		"esquema_ajeno": func(c []byte) []byte {
			return bytes.Replace(c, []byte(esquemaRelacionUnidadExpedienteCT), []byte("vec.otra.relacion.v1"), 1)
		},
		"valor_null": func(c []byte) []byte {
			return bytes.Replace(c, []byte(`"version_origen_vinculo":4`), []byte(`"version_origen_vinculo":null`), 1)
		},
		"sufijo": func(c []byte) []byte { return append(c, []byte(` {}`)...) },
		"unidad_actor": func(c []byte) []byte {
			return bytes.Replace(c, []byte(`"unidad_ref":"unidad:ct:gestora"`), []byte(`"unidad_ref":"unidad:actor"`), 1)
		},
		"historica_otra": func(c []byte) []byte {
			return bytes.Replace(c, []byte(`"version_expediente_solicitada":7`), []byte(`"version_expediente_solicitada":8`), 1)
		},
		"fecha_sin_micro":  func(c []byte) []byte { return bytes.Replace(c, []byte(".123456Z"), []byte("Z"), 1) },
		"fecha_con_offset": func(c []byte) []byte { return bytes.Replace(c, []byte(".123456Z"), []byte(".123456+00:00"), 1) },
		"tamano":           func(c []byte) []byte { return append(c, bytes.Repeat([]byte(" "), maximaRelacionUnidadCTBytes)...) },
	} {
		t.Run(nombre, func(t *testing.T) {
			c := mutar(bytes.Clone(contenido))
			r, err := DecodificarRelacionUnidadExpedienteCT(c, q)
			if !errors.Is(err, ports.ErrRelacionUnidadExpedienteCTInvalida) || r != (ports.RelacionUnidadExpedienteCT{}) {
				t.Fatalf("cuerpo ajeno aceptado: %+v,%v", r, err)
			}
			if err.Error() != ports.ErrRelacionUnidadExpedienteCTInvalida.Error() {
				t.Fatalf("detalle privado en error: %v", err)
			}
		})
	}
}

func TestDecodificarRelacionUnidadCTConservaCausasSinExponerValores(t *testing.T) {
	contenido, q := relacionUnidadCTSQLPrueba(t)
	_, err := DecodificarRelacionUnidadExpedienteCT(nil, q)
	if !errors.Is(err, io.ErrUnexpectedEOF) || !errors.Is(err, ports.ErrRelacionUnidadExpedienteCTInvalida) {
		t.Fatalf("causa EOF perdida: %v", err)
	}
	_, err = DecodificarRelacionUnidadExpedienteCT([]byte(`{"esquema":!}`), q)
	var sintaxis *json.SyntaxError
	if !errors.As(err, &sintaxis) || err.Error() != ports.ErrRelacionUnidadExpedienteCTInvalida.Error() {
		t.Fatalf("causa JSON perdida/expuesta: %v", err)
	}
	for _, numero := range []string{"-1", "7.5", "18446744073709551616"} {
		c := bytes.Replace(contenido, []byte(`"version_expediente_solicitada":7`), []byte(`"version_expediente_solicitada":`+numero), 1)
		_, err = DecodificarRelacionUnidadExpedienteCT(c, q)
		var tipo *json.UnmarshalTypeError
		if !errors.As(err, &tipo) || err.Error() != ports.ErrRelacionUnidadExpedienteCTInvalida.Error() {
			t.Fatalf("causa numérica perdida/expuesta: %v", err)
		}
	}
	c := bytes.Replace(contenido, []byte("2026-10-01T10:00:00.123456Z"), []byte("valor_fecha_privado"), 1)
	_, err = DecodificarRelacionUnidadExpedienteCT(c, q)
	var fecha *time.ParseError
	if !errors.As(err, &fecha) || fecha.Value != "valor_fecha_privado" || err.Error() != ports.ErrRelacionUnidadExpedienteCTInvalida.Error() {
		t.Fatalf("causa fecha perdida/expuesta: %v", err)
	}
}
