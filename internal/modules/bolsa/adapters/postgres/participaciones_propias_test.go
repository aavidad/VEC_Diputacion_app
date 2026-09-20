package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
)

func TestParticipacionesPropiasPostgreSQLInvocaFachadaB11ConAD3YConfirmaTrasDTOEstricto(t *testing.T) {
	consulta := consultaParticipacionesPropiasPostgreSQLPrueba(t)
	material := materialParticipacionesPropiasPostgreSQLPrueba(t, consulta)
	respuesta, err := json.Marshal(resultadoParticipacionesPropiasPostgreSQLPrueba(consulta))
	if err != nil {
		t.Fatal(err)
	}
	tx := &transaccionPanelPostgreSQLPrueba{fila: filaPanelPostgreSQLPrueba{contenido: respuesta}}
	adaptador, err := nuevaConsultaParticipacionesPropiasPostgreSQL(&iniciadorPanelPostgreSQLPrueba{tx: tx})
	if err != nil {
		t.Fatal(err)
	}
	resultado, err := adaptador.ConsultarParticipacionesPropias(context.Background(), consulta, material)
	if err != nil || resultado.Esquema != puertosbolsa.EsquemaParticipacionesPropiasV1 {
		t.Fatalf("consulta B11: resultado=%+v error=%v", resultado, err)
	}
	if tx.confirmaciones != 1 || tx.reversiones != 1 || tx.configuraciones != 1 ||
		!strings.Contains(tx.consulta, funcionConsultarParticipacionesPropiasB11V1) ||
		len(tx.argumentos) != 11 || !strings.Contains(tx.consulta, "$11::bytea") {
		t.Fatalf("frontera B11 divergente: consulta=%q argumentos=%d commit=%d rollback=%d", tx.consulta, len(tx.argumentos), tx.confirmaciones, tx.reversiones)
	}
	if len(tx.argumentos[0].([]byte)) == 0 || string(tx.argumentos[0].([]byte)) != string(consultaCanonicaParticipacionesPropiasB11V1) ||
		tx.argumentos[5] != uint64(1) || tx.argumentos[6] != uint64(1) {
		t.Fatal("consulta canónica o versiones AD3 divergentes")
	}
	for _, indice := range []int{1, 2, 3, 4, 7, 8, 9, 10} {
		if contenido, ok := tx.argumentos[indice].([]byte); !ok || len(contenido) == 0 {
			t.Fatalf("pieza AD3 %d ausente", indice)
		}
	}
}

func TestParticipacionesPropiasPostgreSQLRechazaRespuestaAmbiguaAntesDeCommit(t *testing.T) {
	consulta := consultaParticipacionesPropiasPostgreSQLPrueba(t)
	material := materialParticipacionesPropiasPostgreSQLPrueba(t, consulta)
	valida, err := json.Marshal(resultadoParticipacionesPropiasPostgreSQLPrueba(consulta))
	if err != nil {
		t.Fatal(err)
	}
	for nombre, contenido := range map[string][]byte{
		"duplicada":   append([]byte(`{"esquema":"otra",`), valida[1:]...),
		"desconocida": append([]byte(`{"identidad":"no",`), valida[1:]...),
		"posterior":   append(append([]byte(nil), valida...), []byte(` {}`)...),
	} {
		t.Run(nombre, func(t *testing.T) {
			tx := &transaccionPanelPostgreSQLPrueba{fila: filaPanelPostgreSQLPrueba{contenido: contenido}}
			a, err := nuevaConsultaParticipacionesPropiasPostgreSQL(&iniciadorPanelPostgreSQLPrueba{tx: tx})
			if err != nil {
				t.Fatal(err)
			}
			_, err = a.ConsultarParticipacionesPropias(context.Background(), consulta, material)
			if !errors.Is(err, puertosbolsa.ErrResultadoParticipacionesPropiasInvalido) || tx.confirmaciones != 0 || tx.reversiones != 1 {
				t.Fatalf("respuesta ambigua confirmada: error=%v commit=%d rollback=%d", err, tx.confirmaciones, tx.reversiones)
			}
		})
	}
}

func TestParticipacionesPropiasPostgreSQLFallaCerradoYNoFiltraPostgreSQL(t *testing.T) {
	if a, err := nuevaConsultaParticipacionesPropiasPostgreSQL(nil); a != nil || !errors.Is(err, ErrFuenteParticipacionesPropiasPostgreSQLNoDisponible) {
		t.Fatal("constructor admitió fuente nula")
	}
	consulta := consultaParticipacionesPropiasPostgreSQLPrueba(t)
	tx := &transaccionPanelPostgreSQLPrueba{}
	a, err := nuevaConsultaParticipacionesPropiasPostgreSQL(&iniciadorPanelPostgreSQLPrueba{tx: tx})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.ConsultarParticipacionesPropias(context.Background(), consulta, puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3{})
	if !errors.Is(err, puertosbolsa.ErrConsultaParticipacionesPropiasInvalida) || tx.consulta != "" {
		t.Fatalf("material inválido llegó a SQL: %v", err)
	}
	if err := errorPostgreSQLParticipacionesPropias(context.Background(), &pgconn.PgError{Code: "42501", Message: "interno"}); !errors.Is(err, dominiovec.ErrAutorizacionDenegada) || strings.Contains(err.Error(), "interno") {
		t.Fatal("denegación SQL filtrada o transformada")
	}
	if err := errorPostgreSQLParticipacionesPropias(context.Background(), &pgconn.PgError{Code: "40001"}); !errors.Is(err, ErrConsultaParticipacionesPropiasPostgreSQLEnCurso) {
		t.Fatal("reintento concurrente no normalizado")
	}
}

func consultaParticipacionesPropiasPostgreSQLPrueba(t *testing.T) puertosbolsa.ConsultaParticipacionesPropias {
	t.Helper()
	c, err := puertosbolsa.NuevaConsultaParticipacionesPropias("can_"+strings.Repeat("c", 22), time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func resultadoParticipacionesPropiasPostgreSQLPrueba(c puertosbolsa.ConsultaParticipacionesPropias) puertosbolsa.ResultadoParticipacionesPropias {
	ahora, _ := c.ConsultadaEn()
	return puertosbolsa.ResultadoParticipacionesPropias{Esquema: puertosbolsa.EsquemaParticipacionesPropiasV1, ConsultadaEn: ahora.Add(time.Microsecond), Participaciones: []puertosbolsa.ParticipacionPropia{{BolsaRef: "bol_" + strings.Repeat("b", 22), CategoriaRef: "cat_" + strings.Repeat("a", 22), VersionBolsa: 2, Orden: 3, TotalParticipaciones: 12, EstadoBolsa: "vigente", VigenteDesde: ahora.Add(-time.Hour)}}}
}

func materialParticipacionesPropiasPostgreSQLPrueba(t *testing.T, consulta puertosbolsa.ConsultaParticipacionesPropias) puertosvec.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	recurso, err := puertosbolsa.RecursoAutorizableParticipacionesPropias(consulta)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	ahora, _ := consulta.ConsultadaEn()
	h := strings.Repeat("a", 64)
	resumen, err := puertosvec.NuevoResumenCapacidadAtestacionAutorizacionV3("decision:prueba", h, h, "contexto:prueba", h, puertosbolsa.AccionConsultarParticipacionesPropias, recurso.Referencia, huella, puertosbolsa.AudienciaParticipacionesPropias, ahora, ahora.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	publica, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	material, err := puertosvec.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3([]byte(strings.Repeat("x", 512)), resumen, []byte("{}"), []byte("{}"), []byte("{}"), 1, 1, []byte("a"), []byte("b"), []byte("c"), spki)
	if err != nil {
		t.Fatal(err)
	}
	return material
}
