package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestEnvelopeLiteralAUT57LlegaACommit(t *testing.T) {
	b, err := os.ReadFile("testdata/recibo_aut57_permitido.json")
	if err != nil {
		t.Fatal(err)
	}
	h := strings.Repeat("a", 64)
	p := documento{HuellaPlanSHA256: h, Plan: domain.PlanIdentidadInternaSinteticaV1{
		OperacionRef: "piis_" + strings.Repeat("a", 22), AlcanceFuente: "sintetico_declarado",
		Persona:      domain.PersonaIdentidadInternaSintetica{PersonaRef: "per_" + strings.Repeat("b", 22)},
		Organizacion: domain.OrganizacionIdentidadInternaSintetica{OrganizacionRef: "org_" + strings.Repeat("c", 16), VersionEsperada: 2},
	}}
	cfg := conexionPrivada{LoginEsperado: "vec_ensayo_identidad", PreimagenSHA256: h, ConfiguracionSHA256: h, AprobacionRef: "aprobacion_ensayo"}
	for nombre, payload := range map[string][]byte{
		"literal_sql":   b,
		"sin_esquema":   bytes.Replace(b, []byte("\"esquema\": \"vec.aut.fuentes-identidad-interna-sintetica.v1\","), nil, 1),
		"esquema_ajeno": bytes.Replace(b, []byte("vec.aut.fuentes-identidad-interna-sintetica.v1"), []byte("vec.aut.fuentes-admin.v1"), 1),
	} {
		t.Run(nombre, func(t *testing.T) {
			tx := &txDoble{salida: payload}
			abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
			_, _, err := ejecutarOperacion(context.Background(), cfg, time.Second, nil, h, p, "apply", abrir)
			if nombre == "literal_sql" && (err != nil || tx.commits != 1) {
				t.Fatalf("recibo literal rechazado antes de confirmar: %v", err)
			}
			if nombre != "literal_sql" && (err == nil || tx.commits != 0) {
				t.Fatal("esquema ausente o ajeno aceptado")
			}
		})
	}
}

type txDoble struct {
	commitErr error
	commits   int
	cerradas  int
	apply     int
	recover   int
	salida    []byte
}

func (t *txDoble) PrepararUTC(context.Context) error { return nil }
func (t *txDoble) Provisionar(context.Context, string, string) ([]byte, error) {
	t.apply++
	return append([]byte(nil), t.salida...), nil
}
func (t *txDoble) Recuperar(context.Context, string, string) ([]byte, error) {
	t.recover++
	return append([]byte(nil), t.salida...), nil
}
func (t *txDoble) Confirmar(context.Context) error { t.commits++; return t.commitErr }
func (t *txDoble) Cerrar(context.Context)          { t.cerradas++ }
func txDenegada() *txDoble {
	codigo := "identidad_rechazada"
	b, _ := json.Marshal(envoltura{Estado: "denegado", Codigo: &codigo, AuditoriaIntento: auditoriaIntento{AuditoriaRef: "aud_1", Secuencia: 1, HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: "cor_1", RegistradaEn: "2026-10-07T00:00:00Z"}})
	return &txDoble{salida: b}
}
func TestDenegacionSeConfirmaYRecuperacionUsaFachada(t *testing.T) {
	for _, modo := range []string{"apply", "reconcile"} {
		t.Run(modo, func(t *testing.T) {
			tx := txDenegada()
			abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
			_, e, err := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, nil, "", documento{}, modo, abrir)
			if err != nil || e.Estado != "denegado" || tx.commits != 1 || tx.cerradas != 1 {
				t.Fatal("audit not committed")
			}
			if modo == "reconcile" && (tx.recover != 1 || tx.apply != 0) {
				t.Fatal("recovery created effect")
			}
		})
	}
}
func TestCommitInciertoNoEntregaReciboNiReintenta(t *testing.T) {
	tx := txDenegada()
	tx.commitErr = errors.New("private_error")
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	b, _, e := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, nil, "", documento{}, "apply", abrir)
	if e != errCommit || b != nil || tx.apply != 1 || tx.commits != 1 || tx.cerradas != 1 {
		t.Fatal("uncertain commit")
	}
}
func TestEnvolturaExtraSeRechazaAntesDeCommit(t *testing.T) {
	tx := txDenegada()
	tx.salida = append(tx.salida[:len(tx.salida)-1], []byte(`,"secret":"must_not_be_saved"}`)...)
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	if _, _, e := ejecutarOperacion(context.Background(), conexionPrivada{}, time.Second, nil, "", documento{}, "apply", abrir); e == nil || tx.commits != 0 {
		t.Fatal("untrusted receipt accepted")
	}
}
func TestJSONDuplicadoYCamposPrivilegiados(t *testing.T) {
	var a aprobacionPrivada
	if decodificarEstricto([]byte(`{"huella_plan_sha256":"a","huella_plan_sha256":"b"}`), &a) == nil {
		t.Fatal("duplicate")
	}
	var d datosIS
	if decodificarEstricto([]byte(`{"persona_ref":"p","cuenta_ordinaria_ref":"c","version_titularidad":1,"cuenta_privilegiada_ref":"x"}`), &d) == nil {
		t.Fatal("privileged account")
	}
}
func TestDestinoPrivadoNoAbreRedAjenaNiElevaRol(t *testing.T) {
	for _, dsn := range []string{"host=example.invalid dbname=synthetic user=nominal sslmode=verify-full", "host=127.0.0.1 dbname=synthetic user=nominal sslmode=disable", "host=127.0.0.1 dbname=synthetic user=nominal sslmode=verify-full role=privileged", "host=127.0.0.1 dbname=synthetic user=nominal sslmode=verify-full options='-c role=privileged'"} {
		cfg := conexionPrivada{DSN: dsn, HostEsperado: "127.0.0.1", PuertoEsperado: 5432, BaseEsperada: "synthetic", LoginEsperado: "nominal"}
		if tx, e := nuevaTransaccionPG(context.Background(), cfg, time.Millisecond); e != errConexion || tx != nil {
			t.Fatal("unsafe destination accepted")
		}
	}
}
func TestReciboPermitidoCotejaDestinoYVersionOrganizacion(t *testing.T) {
	h := strings.Repeat("a", 64)
	fecha := "2026-10-07T00:00:00Z"
	cuenta := "cta_" + strings.Repeat("a", 22)
	p := domain.PlanIdentidadInternaSinteticaV1{OperacionRef: "operation", AlcanceFuente: "sintetico_declarado", Persona: domain.PersonaIdentidadInternaSintetica{PersonaRef: "persona"}, Organizacion: domain.OrganizacionIdentidadInternaSintetica{OrganizacionRef: "organizacion", VersionEsperada: 2, ProcedenciaHuellaSHA256: h}}
	cfg := conexionPrivada{LoginEsperado: "nominal", PreimagenSHA256: h, ConfiguracionSHA256: h, AprobacionRef: "approval"}
	r := reciboIdentidad{Esquema: "vec.aut.fuentes-identidad-interna-sintetica.v1", ReciboRef: "receipt", OperacionRef: p.OperacionRef, PlanSHA256: h, PreimagenSHA256: h, ConfiguracionSHA256: h, OperadorLogin: cfg.LoginEsperado, AprobacionRef: cfg.AprobacionRef, AuditoriaRef: "audit", AuditoriaSecuencia: 1, AuditoriaHuellaSHA256: h, RegistradaEn: fecha}
	r.IS = reciboIS{Esquema: "vec.is.identidad-interna-sintetica.v1", Version: 1, ReciboRef: "is", OperacionRef: p.OperacionRef, PlanSHA256: h, AprobacionRef: cfg.AprobacionRef, AlcanceFuente: p.AlcanceFuente, RegistradaEn: fecha, HuellaSHA256: h, Datos: datosIS{PersonaRef: p.Persona.PersonaRef, CuentaOrdinariaRef: cuenta, VersionTitularidad: 1}}
	r.CA = reciboCA{Esquema: "vec.ca.identidad-interna-sintetica.v1", Version: 1, ReciboRef: "ca", OperacionRef: p.OperacionRef, PlanSHA256: h, AprobacionRef: cfg.AprobacionRef, AlcanceFuente: p.AlcanceFuente, RegistradaEn: fecha, HuellaSHA256: h, Datos: datosCA{OrganizacionRef: p.Organizacion.OrganizacionRef, OrganizacionVersion: 2, PersonaRef: p.Persona.PersonaRef, PersonaVersion: 1, CuentaOrdinariaRef: cuenta, ProyeccionCuentaVersion: 1}}
	e := envoltura{Estado: "permitido", Recibo: &r, Replay: true, AuditoriaIntento: auditoriaIntento{AuditoriaRef: "audit", Secuencia: 1, HuellaSHA256: h, CorrelacionRef: "cor", RegistradaEn: fecha}}
	b, _ := json.Marshal(e)
	if _, err := validarEnvoltura(b, p, h, cfg, "reconcile"); err != nil {
		t.Fatal(err)
	}
	cfg.LoginEsperado = "different"
	if _, err := validarEnvoltura(b, p, h, cfg, "reconcile"); err == nil {
		t.Fatal("login mismatch")
	}
	cfg.LoginEsperado = "nominal"
	r.CA.Datos.OrganizacionVersion = 1
	b, _ = json.Marshal(e)
	if _, err := validarEnvoltura(b, p, h, cfg, "reconcile"); err == nil {
		t.Fatal("organization version mismatch")
	}
}
