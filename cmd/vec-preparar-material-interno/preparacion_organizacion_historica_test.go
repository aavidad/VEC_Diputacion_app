package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
	core "vec-diputacion-granada/internal/vec/domain"
)

func escenarioOH(t *testing.T) (*escenario, string, dependencias) {
	t.Helper()
	e := nuevoEscenario(t)
	k, err := bootstrap.DerivarClaveOrganizacionHistoricaDesdeMaterialDesarrollo(e.idempotencia, ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Borrar()
	e.gobierno.claves[k.SHA256] = filaClave{ClaveID: k.ClaveID, Version: 71, Revision: 83, HuellaGobierno: k.HuellaGobierno, EmisorID: k.EmisorID, Audiencia: k.Audiencia, Desde: k.Desde, Hasta: k.Hasta, ActoPropio: true, Vigente: true, PunteroVigente: true}
	c := map[string]any{"version": 1, "catalogo_motivos": catalogoPrueba, "motivo_consulta": referencia(1), "capacidad": map[string]any{"archivo": "pendiente.hmac"}, "contextos": map[string]any{
		"cta_0123456789abcdef0123456789abcdef": map[string]any{"perfil_activo_ref": "prf_0123456789abcdef0123456789abcdef", "perfil_version": 7, "organismo_ref": "organismo:prueba", "unidad_clave": "unidad:prueba"}}}
	dir := filepath.Join(e.dir, "oh")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, inventarioOHNombre)
	escribir(t, ruta, jsonDe(t, c), 0600)
	d := e.dep()
	d.validarMotivoOH = func(context.Context, string, core.ReferenciaEntradaCatalogo, time.Time) error { return nil }
	return e, ruta, d
}

func TestPreparaOHConFilaPublicadaYConservaContextos(t *testing.T) {
	e, ruta, d := escenarioOH(t)
	var out, stderr bytes.Buffer
	args := []string{"-inventario-ct", e.inventarioCT, "-material-idempotencia", e.idempotencia, "-organizacion-historica-config", ruta, "-dsn-archivo", e.dsnArchivo, "-salida", e.salida}
	if rc := ejecutar(context.Background(), args, "", false, &out, &stderr, d); rc != 0 {
		t.Fatalf("rc=%d %s", rc, stderr.String())
	}
	m, err := internactproveedores.CargarMaterialOrganizacionHistorica(e.salida)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cerrar()
	if m.Capacidad.Version != 71 || m.Capacidad.RevisionGobierno != 83 || m.Contextos["cta_0123456789abcdef0123456789abcdef"].PerfilVersion != 7 || m.Contextos["cta_0123456789abcdef0123456789abcdef"].UnidadClave != "unidad:prueba" {
		t.Fatal("gobierno o contexto sustituido")
	}
	f, err := os.ReadDir(e.salida)
	if err != nil || len(f) != 2 {
		t.Fatal("OH escribió material ajeno")
	}
}

func TestPreparaOHRechazaGobiernoAusenteOMotivo(t *testing.T) {
	for _, caso := range []string{"sin_clave", "revocada", "audiencia_b2", "puntero", "motivo"} {
		t.Run(caso, func(t *testing.T) {
			e, ruta, d := escenarioOH(t)
			if caso == "motivo" {
				d.validarMotivoOH = func(context.Context, string, core.ReferenciaEntradaCatalogo, time.Time) error { return errMotivoOH }
			} else {
				for h, f := range e.gobierno.claves {
					if strings.HasPrefix(f.ClaveID, "clave:capacidad:personal-organizacion-historica-consulta:") {
						switch caso {
						case "sin_clave":
							delete(e.gobierno.claves, h)
							continue
						case "revocada":
							f.Revocada = true
						case "audiencia_b2":
							f.Audiencia = "vec_personal.registro_empleado.ficha.v1"
						case "puntero":
							f.PunteroVigente = false
						}
						e.gobierno.claves[h] = f
					}
				}
			}
			p := preparacion{opciones: opciones{inventarioCT: e.inventarioCT, idempotencia: e.idempotencia, dsnArchivo: e.dsnArchivo, salida: e.salida}, dep: d}
			if _, err := p.prepararOrganizacionHistorica(context.Background(), ruta); err == nil {
				t.Fatal("OH inválida admitida")
			}
			e.sinResiduos(t)
		})
	}
}

func TestModosOHYB2NoSeMezclan(t *testing.T) {
	var out, stderr bytes.Buffer
	rc := ejecutar(context.Background(), []string{"-inventario-ct", "ct", "-material-idempotencia", "idem", "-organizacion-historica-config", "oh", "-motivos", "b2", "-salida", "salida"}, "", false, &out, &stderr, dependencias{})
	if rc != 2 {
		t.Fatal("admite modos mezclados")
	}
}
