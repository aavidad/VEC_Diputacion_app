package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/vec/domain"
)

func planFixture(t *testing.T) documento {
	t.Helper()
	b, e := os.ReadFile("testdata/fuente.sintetica.json")
	if e != nil {
		t.Fatal(e)
	}
	var p domain.PlanFuentesInicialesAdminV1
	if decodificarEstricto(b, &p) != nil {
		t.Fatal("fixture")
	}
	_, sha, e := p.CanonicoYHuella()
	if e != nil {
		t.Fatal(e)
	}
	return documento{p, sha}
}
func envelopeFixture(t *testing.T, p documento, estado string, replay bool) []byte {
	t.Helper()
	h := strings.Repeat("e", 64)
	ref := func(prefix, c string) string { return prefix + strings.Repeat(c, 32) }
	fecha := "2026-10-03T12:30:00.123456+00:00"
	e := envoltura{Estado: estado, Replay: replay, AuditoriaIntento: auditoriaIntento{ref("aud_v3_fi_", "a"), 3, h, ref("correlacion_", "a"), fecha}}
	if estado == "permitido" {
		r := &reciboFuentes{Esquema: "vec.admin.fuentes-confirmadas.v1", ReciboRef: ref("recibo_fuentes:", "b"), OperacionRef: p.Plan.OperacionRef, PlanSHA256: p.HuellaPlanSHA256, PreimagenSHA256: h, ConfiguracionSHA256: h, OperadorLogin: "operador_sintetico", AprobacionRef: "aprobacion_ensayo", AuditoriaRef: ref("aud_v3_f_", "c"), AuditoriaSecuencia: 2, AuditoriaHuellaSHA256: h, RegistradaEn: fecha}
		r.CA = reciboCA{Esquema: "vec.ca.fuentes-iniciales-admin.v1", Version: 1, ReciboRef: ref("recibo_ca_fuentes:", "d"), OperacionRef: p.Plan.OperacionRef, PlanSHA256: p.HuellaPlanSHA256, AprobacionRef: r.AprobacionRef, AlcanceFuente: p.Plan.AlcanceFuente, RegistradaEn: fecha, HuellaSHA256: h, Datos: datosCA{OrganizacionRef: p.Plan.Organizacion.OrganizacionRef, OrganizacionVersion: 1}}
		r.IS = reciboIS{Esquema: "vec.is.fuentes-iniciales-admin.v1", Version: 1, ReciboRef: ref("recibo_is_fuentes:", "e"), OperacionRef: p.Plan.OperacionRef, PlanSHA256: p.HuellaPlanSHA256, AprobacionRef: r.AprobacionRef, AlcanceFuente: p.Plan.AlcanceFuente, RegistradaEn: fecha, HuellaSHA256: h, Datos: datosIS{PoliticaRef: p.Plan.PoliticaADMIN.PoliticaRef}}
		for i, c := range []string{"a", "b"} {
			r.IS.Datos.Personas[i] = cuentaPersona{p.Plan.Personas[i].PersonaRef, "cta_" + strings.Repeat(c, 22) + "o", "cta_" + strings.Repeat(c, 22) + "p", 1}
			x := r.IS.Datos.Personas[i]
			r.CA.Datos.Personas[i] = cuentaPersonaCA{x.PersonaRef, x.CuentaOrdinariaRef, x.CuentaPrivilegiadaRef, 1, 1, 1}
		}
		e.Recibo = r
	} else {
		codigo := "fuentes_rechazadas"
		if estado == "error" {
			codigo = "fuentes_no_disponibles"
		}
		e.Codigo = &codigo
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type txFixture struct {
	pasos             []string
	respuesta         []byte
	fallar            string
	canon, aprobacion string
}

func (t *txFixture) PrepararUTC(context.Context) error {
	t.pasos = append(t.pasos, "utc")
	if t.fallar == "utc" {
		return errors.New("secreto")
	}
	return nil
}
func (t *txFixture) Provisionar(_ context.Context, c, a string) ([]byte, error) {
	t.pasos = append(t.pasos, "consulta")
	t.canon = c
	t.aprobacion = a
	if t.fallar == "consulta" {
		return nil, errors.New("secreto")
	}
	return append([]byte{}, t.respuesta...), nil
}
func (t *txFixture) Confirmar(context.Context) error {
	t.pasos = append(t.pasos, "commit")
	if t.fallar == "commit" {
		return errors.New("secreto")
	}
	return nil
}
func (t *txFixture) Cerrar(context.Context) { t.pasos = append(t.pasos, "cerrar") }
func TestCommitEnTodosLosEstados(t *testing.T) {
	p := planFixture(t)
	canon, _, _ := p.Plan.CanonicoYHuella()
	for _, estado := range []string{"permitido", "denegado", "error"} {
		t.Run(estado, func(t *testing.T) {
			tx := &txFixture{respuesta: envelopeFixture(t, p, estado, false)}
			abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
			b, e, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, canon, strings.Repeat("f", 64), p, abrir)
			if err != nil || e.Estado != estado || len(b) == 0 || !reflect.DeepEqual(tx.pasos, []string{"utc", "consulta", "commit", "cerrar"}) {
				t.Fatal("protocolo incorrecto", err, tx.pasos)
			}
			if tx.aprobacion != strings.Repeat("f", 64) || tx.canon != string(canon) {
				t.Fatal("sustituye aprobación externa o canon")
			}
		})
	}
}
func TestFalloCommitNoEntregaRecibo(t *testing.T) {
	p := planFixture(t)
	canon, _, _ := p.Plan.CanonicoYHuella()
	tx := &txFixture{respuesta: envelopeFixture(t, p, "permitido", false), fallar: "commit"}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	b, e, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, canon, p.HuellaPlanSHA256, p, abrir)
	if err != errCommit || b != nil || e.Recibo != nil {
		t.Fatal("presenta efecto incierto")
	}
	if !reflect.DeepEqual(tx.pasos, []string{"utc", "consulta", "commit", "cerrar"}) {
		t.Fatal("reintenta")
	}
}
func TestEnvolturaInvalidaNoConfirma(t *testing.T) {
	p := planFixture(t)
	tx := &txFixture{respuesta: []byte(`{"estado":"permitido","secreto":"no mostrar"}`)}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	_, _, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, nil, p.HuellaPlanSHA256, p, abrir)
	if err != errEnvoltura || !reflect.DeepEqual(tx.pasos, []string{"utc", "consulta", "cerrar"}) {
		t.Fatal("confirma respuesta inválida")
	}
}
func TestCanalesPG(t *testing.T) {
	tlsSeguro := &tls.Config{ServerName: "database.example.invalid", MinVersion: tls.VersionTLS12}
	casos := []struct {
		host   string
		tls    *tls.Config
		socket bool
		ok     bool
	}{{"/ruta/socket", nil, true, true}, {"/ruta/socket", nil, false, false}, {"database.example.invalid", nil, false, false}, {"database.example.invalid", tlsSeguro, false, true}, {"database.example.invalid", &tls.Config{InsecureSkipVerify: true}, false, false}}
	for _, c := range casos {
		cfg := &pgconn.Config{Host: c.host, TLSConfig: c.tls, User: "operador", Database: "sintetica"}
		if canalValido(cfg, c.socket) != c.ok {
			t.Fatal("canal divergente")
		}
		if c.ok {
			cfg.Fallbacks = []*pgconn.FallbackConfig{{Host: "database.example.invalid"}}
			if canalValido(cfg, c.socket) {
				t.Fatal("fallback sin verificar")
			}
		}
	}
}
