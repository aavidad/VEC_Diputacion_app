package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
)

func planFixture(t *testing.T) documento {
	t.Helper()
	b, e := os.ReadFile("testdata/material.sintetico.json")
	if e != nil {
		t.Fatal(e)
	}
	var p domain.PlanUnidadInicialAdminV1
	if decodificarEstricto(b, &p) != nil {
		t.Fatal("material")
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
	fecha := "2026-10-04T08:30:00.123456+00:00"
	e := envoltura{Estado: estado, Replay: replay, AuditoriaIntento: auditoriaIntento{ref("aud_v3_ui_", "a"), 3, h, ref("correlacion_", "a"), fecha}}
	if estado == "permitido" {
		u := p.Plan.Unidad
		r := &reciboUnidad{Esquema: "vec.personal.unidad-inicial.v1", Version: 1, ReciboRef: ref("recibo_unidad:", "b"), OperacionRef: p.Plan.OperacionRef, PlanSHA256: p.HuellaPlanSHA256, FuenteRef: p.Plan.Fuente.Referencia, FuenteVersion: 1, FuenteSHA256: p.Plan.Fuente.HuellaSHA256, PreimagenSHA256: h, ConfiguracionSHA256: h, AprobacionRef: "aprobacion_ensayo", AlcanceFuente: p.Plan.AlcanceFuente, RegistradaEn: fecha, ReciboSHA256: h, AuditoriaRef: ref("aud_v3_u_", "c"), AuditoriaSecuencia: 2, AuditoriaHuellaSHA256: h, AuditoriaRegistradaEn: fecha}
		r.Unidad = nodoHistoria{NodoRef: u.NodoRef, Revision: 1, OrganismoRef: u.OrganizacionRef, UnidadRef: u.UnidadRef, Clase: u.Clase, CatalogoRef: u.CatalogoRef, CatalogoVersion: 1, CatalogoRevision: 1, CatalogoEntradaClave: u.CatalogoEntradaClave, Denominacion: u.Denominacion, VigenteDesde: u.VigenteDesde, VigenteHasta: u.VigenteHasta, ConocidoDesde: fecha, FuenteRef: p.Plan.Fuente.Referencia, ActoRef: p.Plan.ActoTecnicoRef, HuellaFuenteSHA256: p.Plan.Fuente.HuellaSHA256}
		e.Recibo = r
	} else {
		codigo := "unidad_rechazada"
		if estado == "error" {
			codigo = "unidad_no_disponible"
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
	pasos                     []string
	respuesta                 []byte
	fallar                    string
	canon, aprobacion, fuente string
}

func (t *txFixture) PrepararUTC(context.Context) error { t.pasos = append(t.pasos, "utc"); return nil }
func (t *txFixture) Provisionar(_ context.Context, c, a, f string) ([]byte, error) {
	t.pasos = append(t.pasos, "consulta")
	t.canon = c
	t.aprobacion = a
	t.fuente = f
	if t.fallar == "consulta" {
		return nil, errors.New("error_no_mostrar")
	}
	return append([]byte{}, t.respuesta...), nil
}
func (t *txFixture) Confirmar(context.Context) error {
	t.pasos = append(t.pasos, "commit")
	if t.fallar == "commit" {
		return errors.New("error_no_mostrar")
	}
	return nil
}
func (t *txFixture) Cerrar(context.Context) { t.pasos = append(t.pasos, "cerrar") }
func argsAplicacionFixture(t *testing.T) ([]string, string, documento) {
	t.Helper()
	args, path, _, _ := prepararFixture(t)
	var salida, errores bytes.Buffer
	if ejecutar(args, &salida, &errores, relojFixture) != 0 {
		t.Fatal(errores.String())
	}
	write := func(name string, v any) string {
		t.Helper()
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		p := filepath.Join(filepath.Dir(path), name)
		if os.WriteFile(p, b, 0600) != nil {
			t.Fatal("archivo")
		}
		return p
	}
	con := write("conexion.json", conexionPrivada{DSN: "postgres://operador@database.example.invalid/sintetica?sslmode=verify-full"})
	// Una SHA distinta evidencia que la CLI NO la sustituye por su SHA calculada.
	approval := write("aprobacion.json", aprobacionPrivada{strings.Repeat("f", 64)})
	ack := filepath.Join(filepath.Dir(path), "acuse.json")
	args = append(args, "--cotejar", "--aplicar", "--conexion", con, "--aprobacion", approval, "--acuse", ack, "--timeout", "5s")
	return args, ack, planFixture(t)
}
func TestUnidadAplicacionConfirmaTresEstados(t *testing.T) {
	for _, estado := range []string{"permitido", "denegado", "error"} {
		t.Run(estado, func(t *testing.T) {
			args, acuse, p := argsAplicacionFixture(t)
			tx := &txFixture{respuesta: envelopeFixture(t, p, estado, estado == "permitido")}
			llamadas := 0
			abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) {
				llamadas++
				b, e := os.ReadFile(acuse)
				if e != nil || len(b) != 0 {
					t.Fatal("acuse no reservado antesDB")
				}
				return tx, nil
			}
			var salida, errores bytes.Buffer
			exit := ejecutarConProveedor(args, &salida, &errores, relojFixture, abrir)
			expected := 0
			if estado != "permitido" {
				expected = 1
			}
			if exit != expected || errores.Len() != 0 || llamadas != 1 || !reflect.DeepEqual(tx.pasos, []string{"utc", "consulta", "commit", "cerrar"}) {
				t.Fatal("protocolo", exit, errores.String(), tx.pasos)
			}
			b, e := os.ReadFile(acuse)
			if e != nil || !bytes.Equal(b, append(append([]byte{}, tx.respuesta...), '\n')) {
				t.Fatal("acuse cambia respuesta")
			}
			if tx.aprobacion != strings.Repeat("f", 64) {
				t.Fatal("autoaprueba huella")
			}
			var fuente domain.FuenteUnidadInicialAdminV1
			if decodificarEstricto([]byte(tx.fuente), &fuente) != nil || p.Plan.ValidarConFuente(fuente) != nil {
				t.Fatal("fuente no real/canónica")
			}
			if strings.Contains(salida.String(), p.Plan.Unidad.Denominacion) || strings.Contains(salida.String(), "nodo_ref") {
				t.Fatal("expone metadatos")
			}
			if ejecutarConProveedor(args, &salida, &errores, relojFixture, abrir) == 0 || llamadas != 1 {
				t.Fatal("envía antesrechazaracuseexistente")
			}
		})
	}
}
func TestUnidadCommitInciertoSinRecibo(t *testing.T) {
	args, acuse, p := argsAplicacionFixture(t)
	tx := &txFixture{respuesta: envelopeFixture(t, p, "permitido", false), fallar: "commit"}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	var salida, errores bytes.Buffer
	if ejecutarConProveedor(args, &salida, &errores, relojFixture, abrir) != 2 || salida.Len() != 0 {
		t.Fatal("éxito incierto")
	}
	var d diagnosticoAplicacion
	if json.Unmarshal(errores.Bytes(), &d) != nil || d.Estado != "indeterminado" || d.Confirmado || d.AcuseGuardado {
		t.Fatal("no indica incertidumbre")
	}
	if _, e := os.Stat(acuse); !os.IsNotExist(e) {
		t.Fatal("guarda recibo sin commit")
	}
	if strings.Contains(errores.String(), "error_no_mostrar") {
		t.Fatal("expone error original")
	}
}
func TestUnidadRespuestaInvalidaNoConfirma(t *testing.T) {
	args, acuse, _ := argsAplicacionFixture(t)
	tx := &txFixture{respuesta: []byte(`{"estado":"permitido","secreto":"no mostrar"}`)}
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) { return tx, nil }
	var salida, errores bytes.Buffer
	if ejecutarConProveedor(args, &salida, &errores, relojFixture, abrir) != 1 || !reflect.DeepEqual(tx.pasos, []string{"utc", "consulta", "cerrar"}) {
		t.Fatal("confirma respuesta abierta")
	}
	if _, e := os.Stat(acuse); !os.IsNotExist(e) {
		t.Fatal("acuse de respuesta no confirmada")
	}
}
func TestUnidadAplicarNecesitaCotejo(t *testing.T) {
	args, _, _ := argsAplicacionFixture(t)
	filtrados := []string{}
	for _, a := range args {
		if a != "--cotejar" {
			filtrados = append(filtrados, a)
		}
	}
	llamadas := 0
	abrir := func(context.Context, conexionPrivada, time.Duration) (transaccion, error) {
		llamadas++
		return nil, nil
	}
	var salida, errores bytes.Buffer
	if ejecutarConProveedor(filtrados, &salida, &errores, relojFixture, abrir) != 1 || llamadas != 0 {
		t.Fatal("aplica sin cotejo")
	}
}
