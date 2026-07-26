package ports_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

var (
	_ fmt.Formatter  = ports.SolicitudCuadroRRHH{}
	_ fmt.Formatter  = ports.PaginaCuadroRRHH{}
	_ slog.LogValuer = ports.SolicitudCuadroRRHH{}
	_ slog.LogValuer = ports.PaginaCuadroRRHH{}
)

func TestSolicitudCuadroRRHHValidaLimitesYCursorSinPanico(t *testing.T) {
	t.Parallel()
	cursorValido := base64.RawURLEncoding.EncodeToString(make([]byte, 32))
	for nombre, cursor := range map[string]string{
		"ceros": cursorValido,
		"url": base64.RawURLEncoding.EncodeToString(
			bytes.Repeat([]byte{0xff}, 32),
		),
	} {
		if _, err := ports.NuevaSolicitudCuadroRRHH(
			"2026/CT", domain.EstadoEnCurso, "solicitud", 100, cursor,
		); err != nil {
			t.Fatalf("cursor válido %s rechazado: %v", nombre, err)
		}
	}
	if _, err := ports.NuevaSolicitudCuadroRRHH(
		"", "", "", 1, "",
	); err != nil {
		t.Fatalf("primera página sin cursor rechazada: %v", err)
	}
	for nombre, cursor := range map[string]string{
		"42_caracteres": strings.Repeat("A", 42),
		"44_caracteres": strings.Repeat("A", 44),
		"31_bytes_mas_salto": base64.RawURLEncoding.EncodeToString(
			make([]byte, 31),
		) + "\n",
		"33_bytes": base64.RawURLEncoding.EncodeToString(
			make([]byte, 33),
		),
		"relleno":        cursorValido[:42] + "=",
		"alfabeto_mas":   cursorValido[:42] + "+",
		"alfabeto_barra": cursorValido[:42] + "/",
		"unicode":        cursorValido[:41] + "é",
		"bits_relleno":   cursorValido[:42] + "B",
	} {
		if _, err := ports.NuevaSolicitudCuadroRRHH(
			"", "", "", 1, cursor,
		); !errors.Is(err, ports.ErrSolicitudConsultaRRHHInvalida) {
			t.Fatalf("%s aceptado: longitud=%d error=%v", nombre, len(cursor), err)
		} else if strings.Contains(err.Error(), cursor) {
			t.Fatalf("%s filtrado en error: %v", nombre, err)
		}
	}
	for _, limite := range []uint16{0, 101} {
		if _, err := ports.NuevaSolicitudCuadroRRHH(
			"", "", "", limite, "",
		); !errors.Is(err, ports.ErrSolicitudConsultaRRHHInvalida) {
			t.Fatalf("límite %d aceptado: %v", limite, err)
		}
	}
	solicitud, err := ports.NuevaSolicitudCuadroRRHH(
		"", "", "", 1, cursorValido,
	)
	if err != nil {
		t.Fatal(err)
	}
	for _, representacion := range []string{
		solicitud.String(), solicitud.GoString(),
	} {
		if strings.Contains(representacion, cursorValido) {
			t.Fatalf("solicitud filtró cursor: %q", representacion)
		}
	}
	if contenido, err := solicitud.MarshalJSON(); contenido != nil ||
		!errors.Is(err, ports.ErrMaterialConsultaRRHHSensible) ||
		strings.Contains(err.Error(), cursorValido) {
		t.Fatalf("MarshalJSON filtró solicitud: %q, %v", contenido, err)
	}

	ahora := instantePuertosRRHH()
	autoridad, contexto := autoridadYContextoPuertosRRHH(t, ahora)
	primera, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	capacidad := capacidadCuadroPuertosRRHH(t, autoridad, contexto, primera, ahora)
	orden, err := ports.NuevaOrdenConsultaCuadroRRHH(
		contexto, capacidad, primera, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	lectura, err := ports.NuevoReciboLecturaRRHH(
		"lectura:rrhh:cursor", "auditoria:rrhh:cursor",
		contexto, capacidad, "", 0, 0, ahora,
	)
	if err != nil {
		t.Fatal(err)
	}
	pagina := ports.PaginaCuadroRRHH{
		GeneradaEn: ahora, HayMas: true,
		CursorSiguiente: cursorValido, Lectura: lectura,
	}
	if err := pagina.ValidarPara(orden); err != nil {
		t.Fatalf("cursor siguiente válido rechazado: %v", err)
	}
	pagina.CursorSiguiente = cursorValido[:42] + "B"
	if err := pagina.ValidarPara(orden); !errors.Is(
		err, ports.ErrResultadoConsultaRRHHNoConfiable,
	) || strings.Contains(err.Error(), pagina.CursorSiguiente) {
		t.Fatalf("cursor siguiente no canónico aceptado: %v", err)
	}
}

func TestCursorRRHHNoSeFiltraEnSlogNiFmt(t *testing.T) {
	t.Parallel()
	cursor := base64.RawURLEncoding.EncodeToString(
		bytes.Repeat([]byte{0xff}, 32),
	)
	solicitud, err := ports.NuevaSolicitudCuadroRRHH("", "", "", 1, cursor)
	if err != nil {
		t.Fatal(err)
	}
	pagina := ports.PaginaCuadroRRHH{
		HayMas: true, CursorSiguiente: cursor,
	}

	formato := fmt.Sprintf(
		"%v %#v %+v %#v", solicitud, solicitud, pagina, pagina,
	)
	if strings.Contains(formato, cursor) ||
		strings.Contains(formato, "CursorSiguiente") {
		t.Fatalf("fmt filtró el cursor o campos: %q", formato)
	}

	for _, caso := range []struct {
		nombre  string
		handler func(*bytes.Buffer) slog.Handler
	}{
		{"json", func(salida *bytes.Buffer) slog.Handler {
			return slog.NewJSONHandler(salida, nil)
		}},
		{"texto", func(salida *bytes.Buffer) slog.Handler {
			return slog.NewTextHandler(salida, nil)
		}},
	} {
		var salida bytes.Buffer
		slog.New(caso.handler(&salida)).Info(
			"consulta", "solicitud", solicitud, "pagina", pagina,
		)
		contenido := salida.String()
		for _, sensible := range []string{
			cursor, "CursorSiguiente", "cursor_siguiente",
		} {
			if strings.Contains(contenido, sensible) {
				t.Fatalf("%s filtró %q: %s", caso.nombre, sensible, contenido)
			}
		}
		for _, redaccion := range []string{
			solicitud.String(), pagina.String(),
		} {
			if !strings.Contains(contenido, redaccion) {
				t.Fatalf("%s omitió redacción %q: %s", caso.nombre, redaccion, contenido)
			}
		}
	}
}
