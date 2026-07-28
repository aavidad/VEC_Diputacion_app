package ports_test

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

const (
	huellaContenidoDetalleMinimo   = "8e63fb2710c43306f709e8236715537526b5831ba3db95470280cb069cbf136a"
	huellaResultadoDetalleMinimo   = "0b7d78f6d34cd87f3da98fc32a0830ba953c83f36c2a7423fba9810baff78e31"
	huellaContenidoDetalleCompleto = "97b2d440c764090e452e51fb3623900a2cac78d97f337a42437b599ec6335e9b"
	huellaResultadoDetalleCompleto = "9126d2df02a909685878ce93681be8364ee0ab50eff2e7d43cf16a89bd84d0dd"
)

func TestCanonContenidoDetalleConservaVectoresMinimoYCompleto(t *testing.T) {
	t.Parallel()
	casos := []struct {
		nombre, huellaContenido, huellaResultado string
		bloques, bytesContenido                  int
	}{
		{
			nombre: "minimo", bloques: 0, bytesContenido: 577,
			huellaContenido: huellaContenidoDetalleMinimo,
			huellaResultado: huellaResultadoDetalleMinimo,
		},
		{
			nombre: "completo", bloques: 3, bytesContenido: 1342,
			huellaContenido: huellaContenidoDetalleCompleto,
			huellaResultado: huellaResultadoDetalleCompleto,
		},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.nombre, func(t *testing.T) {
			t.Parallel()
			entrada, _ := entradaDetalleMinimizadaPrueba(t, caso.bloques)
			generadaEn := instantePuertosRRHH().Add(time.Minute)
			contenido, err := entrada.ExportarContenidoCanonicoParaSQL(
				generadaEn,
			)
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := contenido.ExportarResultadoCanonicoParaSQL()
			if err != nil {
				t.Fatal(err)
			}
			if contenido.HuellaSHA256() != caso.huellaContenido ||
				len(contenido.BytesCanonicos()) != caso.bytesContenido ||
				resultado.HuellaSHA256() != caso.huellaResultado ||
				len(resultado.BytesCanonicos()) != 150 {
				t.Fatalf(
					"vector divergente: contenido=%s/%d resultado=%s/%d",
					contenido.HuellaSHA256(), len(contenido.BytesCanonicos()),
					resultado.HuellaSHA256(), len(resultado.BytesCanonicos()),
				)
			}
			if contenido.Dominio() != ports.DominioCanonContenidoDetalleRRHH ||
				contenido.Version() !=
					ports.VersionCanonContenidoResultadoRRHH ||
				resultado.TipoConsulta() != "detalle" ||
				!resultado.GeneradaEn().Equal(generadaEn) ||
				resultado.Total() != 1 ||
				resultado.ContenidoHuellaSHA256() !=
					contenido.HuellaSHA256() ||
				resultado.CursorHuellaSHA256() != "" {
				t.Fatalf("metadatos divergentes: %#v", resultado)
			}
			canonResultado := "" +
				"VEC-CT-RESULTADO-CONSULTA-RRHH-V1\n" +
				"7:detalle\n" +
				"27:2026-07-26T08:01:00.000000Z\n" +
				"1:1\n" +
				"64:" + caso.huellaContenido + "\n" +
				"0:\n"
			comprobarVectorCanonSQL(
				t, resultado, canonResultado, caso.huellaResultado,
			)
			canon := contenido.BytesCanonicos()
			if !bytes.HasPrefix(
				canon,
				[]byte("VEC-CT-CONTENIDO-DETALLE-RRHH-V1\n"),
			) || bytes.Contains(canon, []byte("lectura:rrhh:minimizada")) ||
				bytes.Contains(canon, []byte("auditoria:rrhh:minimizada")) {
				t.Fatal("cabecera incorrecta o recibo filtrado en el canon")
			}
		})
	}
}

func TestCanonContenidoDetalleRechazaAusenciaYCorteTemporalInvalido(
	t *testing.T,
) {
	t.Parallel()
	var vacia ports.EntradaDetalleExpedienteRRHHMinimizada
	if _, err := vacia.ExportarContenidoCanonicoParaSQL(
		instantePuertosRRHH(),
	); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("entrada ausente aceptada: %v", err)
	}

	entrada, _ := entradaDetalleMinimizadaPrueba(t, 3)
	for nombre, instante := range map[string]time.Time{
		"anterior": instantePuertosRRHH().Add(-time.Microsecond),
		"nanosegundos": instantePuertosRRHH().
			Add(time.Minute).
			Add(time.Nanosecond),
		"año_cero": time.Date(
			0, time.January, 1, 0, 0, 0, 0, time.UTC,
		),
	} {
		if _, err := entrada.ExportarContenidoCanonicoParaSQL(instante); !errors.Is(
			err, ports.ErrResultadoConsultaRRHHNoConfiable,
		) {
			t.Fatalf("%s aceptado: %v", nombre, err)
		}
	}
}

func TestCanonContenidoDetalleEsOpacoYDefensivo(t *testing.T) {
	t.Parallel()
	entrada, _ := entradaDetalleMinimizadaPrueba(t, 3)
	contenido, err := entrada.ExportarContenidoCanonicoParaSQL(
		instantePuertosRRHH().Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := contenido.ExportarResultadoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	original := contenido.BytesCanonicos()
	copia := contenido.BytesCanonicos()
	copia[0] ^= 0xff
	if !bytes.Equal(original, contenido.BytesCanonicos()) {
		t.Fatal("BytesCanonicos conserva un alias mutable")
	}
	for nombre, exportacion := range map[string]canonRRHHParaSQL{
		"contenido_detalle": contenido,
		"resultado_detalle": resultado,
	} {
		comprobarOpacidadCanonSQL(t, nombre, exportacion, []string{
			"expediente:rrhh:minimizado",
			"lectura:rrhh:minimizada",
			"auditoria:rrhh:minimizada",
			contenido.HuellaSHA256(),
		})
		for _, formato := range []string{
			fmt.Sprintf("%v", exportacion),
			fmt.Sprintf("%+v", exportacion),
			fmt.Sprintf("%#v", exportacion),
		} {
			if strings.Contains(formato, "expediente:rrhh:minimizado") {
				t.Fatalf("%s filtró detalle en fmt: %q", nombre, formato)
			}
		}
	}

	var cero ports.ExportacionCanonicaContenidoDetalleRRHH
	if _, err := cero.ExportarResultadoCanonicoParaSQL(); !errors.Is(
		err, ports.ErrResultadoConsultaRRHHNoConfiable,
	) {
		t.Fatalf("exportación cero aceptada: %v", err)
	}
}
