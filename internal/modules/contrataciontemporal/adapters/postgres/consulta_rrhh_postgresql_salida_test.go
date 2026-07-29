package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestAnalizadorCuadroRRHHPostgreSQLValidaMaterialRealDelCursor(
	t *testing.T,
) {
	t.Parallel()
	material := bytes.Repeat([]byte{0x6d}, sha256.Size)
	cursor := base64.RawURLEncoding.EncodeToString(material)
	pagina := ports.PaginaCuadroRRHH{
		GeneradaEn: instanteCanonRRHHPostgreSQLPrueba(),
		Expedientes: []ports.ResumenExpedienteRRHH{
			resumenCanonRRHHPostgreSQLPrueba(
				1,
				instanteCanonRRHHPostgreSQLPrueba(),
			),
		},
		HayMas:          true,
		CursorSiguiente: cursor,
	}
	exportacion, err := pagina.ExportarContenidoCanonicoParaSQL()
	if err != nil {
		t.Fatal(err)
	}
	analizada, err := (analizadorCanonConsultaRRHHPostgreSQL{}).analizarCuadro(
		exportacion.BytesCanonicos(),
		cursor,
		pagina.GeneradaEn,
		1,
	)
	if err != nil {
		t.Fatalf("cursor Base64URL válido rechazado: %v", err)
	}
	if analizada.CursorSiguiente != cursor {
		t.Fatal("el analizador no repuso el cursor validado")
	}

	cursorDeHuellaEquivocada := base64.RawURLEncoding.EncodeToString(
		bytes.Repeat([]byte{0x6e}, sha256.Size),
	)
	if _, err := (analizadorCanonConsultaRRHHPostgreSQL{}).analizarCuadro(
		exportacion.BytesCanonicos(),
		cursorDeHuellaEquivocada,
		pagina.GeneradaEn,
		1,
	); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("cursor de otro material aceptado: %v", err)
	}
	if _, err := (analizadorCanonConsultaRRHHPostgreSQL{}).analizarCuadro(
		exportacion.BytesCanonicos(),
		cursor+"=",
		pagina.GeneradaEn,
		1,
	); !errors.Is(err, ports.ErrResultadoConsultaRRHHNoConfiable) {
		t.Fatalf("cursor Base64URL no canónico aceptado: %v", err)
	}
}

func TestNormalizarErrorFilaConsultaRRHHNoExponeDetalles(t *testing.T) {
	t.Parallel()
	casos := []struct {
		codigo   string
		esperado error
	}{
		{codigo: "42501", esperado: ports.ErrConsultaRRHHNoObservable},
		{codigo: "40001", esperado: ports.ErrConsultaRRHHNoDisponible},
		{codigo: "40P01", esperado: ports.ErrConsultaRRHHNoDisponible},
		{codigo: "55P03", esperado: ports.ErrConsultaRRHHNoDisponible},
		{codigo: "57014", esperado: ports.ErrConsultaRRHHNoDisponible},
		{codigo: "55000", esperado: ports.ErrConsultaRRHHNoDisponible},
	}
	for _, caso := range casos {
		caso := caso
		t.Run(caso.codigo, func(t *testing.T) {
			t.Parallel()
			err := normalizarErrorFilaConsultaRRHH(
				context.Background(),
				&pgconn.PgError{
					Code: caso.codigo, Message: "expediente secreto",
					Detail: "persona:privada",
				},
			)
			if !errors.Is(err, caso.esperado) {
				t.Fatalf("error normalizado = %v", err)
			}
			if strings.Contains(err.Error(), "secreto") ||
				strings.Contains(err.Error(), "privada") {
				t.Fatalf("se filtró detalle PostgreSQL: %v", err)
			}
		})
	}
}

func TestNormalizarErrorTransaccionConsultaRRHHNoExponeDetalles(t *testing.T) {
	t.Parallel()
	err := normalizarErrorConsultaRRHH(
		context.Background(),
		&pgconn.PgError{
			Code: "42501", Message: "expediente secreto",
			Detail: "persona:privada",
		},
	)
	if !errors.Is(err, ports.ErrConsultaRRHHNoObservable) {
		t.Fatalf("error normalizado = %v", err)
	}
	if strings.Contains(err.Error(), "secreto") ||
		strings.Contains(err.Error(), "privada") {
		t.Fatalf("se filtró detalle PostgreSQL: %v", err)
	}
}

func TestContratoSQLConsultaRRHHTieneLigadurasYSalidasExactas(t *testing.T) {
	t.Parallel()
	comprobarSecuenciaLigadurasConsultaRRHH(
		t, consultaCuadroRRHHPostgreSQL, 18,
	)
	comprobarSecuenciaLigadurasConsultaRRHH(
		t, consultaDetalleRRHHPostgreSQL, 15,
	)
	var cuadro salidaCuadroConsultaRRHH
	var detalle salidaDetalleConsultaRRHH
	if total := len(destinosCuadroConsultaRRHH(&cuadro)); total != 21 {
		t.Fatalf("salidas cuadro = %d; se esperaban 21", total)
	}
	if total := len(destinosDetalleConsultaRRHH(&detalle)); total != 20 {
		t.Fatalf("salidas detalle = %d; se esperaban 20", total)
	}
}

func comprobarSecuenciaLigadurasConsultaRRHH(
	t *testing.T,
	consulta string,
	total int,
) {
	t.Helper()
	patron := regexp.MustCompile(`\$([0-9]+)`)
	coincidencias := patron.FindAllStringSubmatch(consulta, -1)
	if len(coincidencias) != total {
		t.Fatalf("ligaduras = %d; se esperaban %d", len(coincidencias), total)
	}
	for indice, coincidencia := range coincidencias {
		numero, err := strconv.Atoi(coincidencia[1])
		if err != nil || numero != indice+1 {
			t.Fatalf(
				"ligadura %d = %q; secuencia no exacta",
				indice+1,
				coincidencia[0],
			)
		}
	}
}

func TestLoginNominalConsultaRRHHRechazaGrupoTecnico(t *testing.T) {
	t.Parallel()
	if loginNominalConsultaRRHHValido("") ||
		loginNominalConsultaRRHHValido(rolConsultorRRHHPostgreSQL) ||
		loginNominalConsultaRRHHValido(" login_rrhh") ||
		loginNominalConsultaRRHHValido("login rrhh") ||
		loginNominalConsultaRRHHValido("Login_rrhh") ||
		loginNominalConsultaRRHHValido("login-rrhh") ||
		loginNominalConsultaRRHHValido(`"login_rrhh"`) ||
		loginNominalConsultaRRHHValido("login_\nrrhh") {
		t.Fatal("se aceptó una identidad no nominal")
	}
	if !loginNominalConsultaRRHHValido("vec_ct_rrhh_servicio_01") {
		t.Fatal("se rechazó un LOGIN nominal dedicado")
	}
}
