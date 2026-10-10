package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/bootstrap"
	core "vec-diputacion-granada/internal/vec/domain"
)

func escenarioIncorporacion(t *testing.T) (*escenario, string, dependencias) {
	t.Helper()
	e := nuevoEscenario(t)
	claves, err := bootstrap.DerivarClavesIncorporacionB2DesdeMaterialDesarrollo(e.idempotencia, ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		for i := range claves {
			claves[i].Borrar()
		}
	}()
	ops := map[string]any{}
	for _, c := range claves {
		ops[c.Capacidad] = map[string]any{"motivo": referencia(1), "capacidad": map[string]any{"file": c.Capacidad + ".cap"}}
		e.gobierno.claves[c.SHA256] = filaClave{ClaveID: c.ClaveID, HuellaGobierno: c.HuellaGobierno, EmisorID: c.EmisorID,
			Audiencia: c.Audiencia, Version: 57, Revision: 83, Desde: c.Desde, Hasta: c.Hasta, ActoPropio: true, Vigente: true, PunteroVigente: true}
	}
	base := map[string]string{}
	anidado := map[string]string{}
	for _, grupo := range []struct {
		m       map[string]string
		nombres []string
	}{{base, []string{"fuente_autorizacion", "motivos_autorizacion", "registro_ct"}}, {anidado, []string{"personal_planes", "personal_actos", "personal_consultas", "bolsa_persona", "rpt"}}} {
		for _, n := range grupo.nombres {
			grupo.m[n] = n + ".dsn"
			escribir(t, filepath.Join(e.dir, n+".dsn"), []byte("DSN_PRIVADO_"+n), 0600)
		}
	}
	c := map[string]any{"esquema": "vec.contratacion-temporal.incorporacion-servidor.v2", "referencias": map[string]any{
		"PrincipalV3Ref": "principal:prueba", "PerfilV3Ref": "perfil:prueba", "OrganizacionRef": "ref:" + strings.Repeat("a", 64),
		"UnidadRef": "ref:" + strings.Repeat("b", 64), "ActorRef": "ref:" + strings.Repeat("c", 64)}, "dsn_files": base,
		"personal_b2": map[string]any{"protocolo": "personal_b2_v1", "organismo_ref": "organismo:prueba", "catalogo_rpt_id": "categorias_rpt", "modulo_rpt_id": "personal", "dsn_files": anidado, "operaciones": ops}}
	ruta := filepath.Join(e.dir, "incorporacion.json")
	escribir(t, ruta, jsonDe(t, c), 0600)
	d := e.dep()
	d.validarMotivosIncorporacion = func(context.Context, bootstrap.ConfiguracionPreparacionIncorporacionB2, *os.Root, time.Time) error {
		return nil
	}
	d.resolverMotivoDetalle = func(context.Context, string, time.Time) (core.ReferenciaEntradaCatalogo, error) {
		var ref core.ReferenciaEntradaCatalogo
		if err := json.Unmarshal(jsonDe(t, referencia(1)), &ref); err != nil {
			t.Fatal(err)
		}
		return ref, nil
	}
	return e, ruta, d
}

func TestPrepararIncorporacionGobiernoExactoYPublicacionAtomica(t *testing.T) {
	e, ruta, d := escenarioIncorporacion(t)
	args := []string{"-inventario-ct", e.inventarioCT, "-material-idempotencia", e.idempotencia, "-incorporacion-config", ruta, "-dsn-archivo", e.dsnArchivo, "-salida", e.salida}
	var out, errOut bytes.Buffer
	if rc := ejecutar(context.Background(), args, "", false, &out, &errOut, d); rc != 0 {
		t.Fatalf("rc=%d: %s", rc, errOut.String())
	}
	c, r, err := bootstrap.LeerConfiguracionPreparacionIncorporacionB2(filepath.Join(e.salida, "servidor.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	if len(c.PersonalB2.Operaciones) != 24 {
		t.Fatal("operaciones incompletas")
	}
	for _, op := range c.PersonalB2.Operaciones {
		if op.Capacidad.Version != 57 || op.Capacidad.RevisionGobierno != 83 {
			t.Fatal("no tomó gobierno publicado")
		}
	}
	if bootstrap.ValidarMaterialPreparadoIncorporacionB2(filepath.Join(e.salida, "servidor.json"), ahoraPrueba) != nil {
		t.Fatal("runtime rechazó capacidades")
	}
}

func TestPrepararIncorporacionDesdeServidorConRaizPublicadaRotada(t *testing.T) {
	e, ruta, d := escenarioIncorporacion(t)
	coordenadas, err := bootstrap.DerivarCoordenadasCTPreparacion(e.idempotencia, ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	e.gobierno.raiz = filaRaiz{ClaveID: coordenadas.RaizID, Version: 7, Audiencia: coordenadas.Audiencia,
		HuellaSPKI: coordenadas.HuellaSPKI, Vigente: true}
	motivosRRHH := filepath.Join(e.dir, "motivos_rrhh.dsn")
	escribir(t, motivosRRHH, []byte("postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1:1/vec\n"), 0600)
	args := []string{"-incorporacion-motivos-rrhh-dsn", motivosRRHH, "-material-idempotencia", e.idempotencia,
		"-incorporacion-config", ruta, "-dsn-archivo", e.dsnArchivo, "-salida", e.salida}
	var out, errOut bytes.Buffer
	if rc := ejecutar(context.Background(), args, "", false, &out, &errOut, d); rc != 0 {
		t.Fatalf("rc=%d: %s", rc, errOut.String())
	}
	if bootstrap.ValidarMaterialPreparadoIncorporacionB2(filepath.Join(e.salida, "servidor.json"), ahoraPrueba) != nil {
		t.Fatal("salida B2 rechazada")
	}
}

func TestPrepararIncorporacionLoginRRHHYErroresNoRevelanDSN(t *testing.T) {
	e, ruta, d := escenarioIncorporacion(t)
	coordenadas, err := bootstrap.DerivarCoordenadasCTPreparacion(e.idempotencia, ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	e.gobierno.raiz = filaRaiz{ClaveID: coordenadas.RaizID, Version: 72, Audiencia: coordenadas.Audiencia,
		HuellaSPKI: coordenadas.HuellaSPKI, Vigente: true}
	motivosRRHH := filepath.Join(e.dir, "motivos_rrhh.dsn")
	escribir(t, motivosRRHH, []byte("postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1:1/vec\n"), 0600)
	args := []string{"-incorporacion-motivos-rrhh-dsn", motivosRRHH, "-incorporacion-motivos-rrhh-login", "otro_login",
		"-material-idempotencia", e.idempotencia, "-incorporacion-config", ruta,
		"-dsn-archivo", e.dsnArchivo, "-salida", e.salida}
	d.resolverMotivoDetalle = nil
	var out, errOut bytes.Buffer
	if rc := ejecutar(context.Background(), args, "", false, &out, &errOut, d); rc != 1 ||
		strings.Contains(out.String()+errOut.String(), "SECRETO_DSN_PRUEBA") || !strings.Contains(errOut.String(), "ct_detalle") {
		t.Fatal("LOGIN RRHH distinto admitido o secreto expuesto")
	}
	e.sinResiduos(t)
	args[3] = "lector"
	d.abrirGobierno = func(context.Context, string) (fuenteGobierno, error) {
		return nil, errors.New("SECRETO_DSN_PRUEBA: fallo de conexión sintético")
	}
	out.Reset()
	errOut.Reset()
	if rc := ejecutar(context.Background(), args, "", false, &out, &errOut, d); rc != 1 ||
		strings.Contains(out.String()+errOut.String(), "SECRETO_DSN_PRUEBA") {
		t.Fatal("error de gobierno reveló el DSN")
	}
	e.sinResiduos(t)
	args = append(args, "-inventario-ct", e.inventarioCT)
	out.Reset()
	errOut.Reset()
	if rc := ejecutar(context.Background(), args, "", false, &out, &errOut, d); rc != 2 {
		t.Fatal("LOGIN RRHH con inventario CT ambiguo admitido")
	}
	args = []string{"-inventario-ct", e.inventarioCT, "-incorporacion-motivos-rrhh-login", "lector",
		"-material-idempotencia", e.idempotencia, "-incorporacion-config", ruta,
		"-dsn-archivo", e.dsnArchivo, "-salida", e.salida}
	out.Reset()
	errOut.Reset()
	if rc := ejecutar(context.Background(), args, "", false, &out, &errOut, d); rc != 2 {
		t.Fatal("LOGIN RRHH aplicado al modo legado")
	}
	args = []string{"-incorporacion-motivos-rrhh-dsn", motivosRRHH, "-incorporacion-motivos-rrhh-login", "",
		"-material-idempotencia", e.idempotencia, "-incorporacion-config", ruta,
		"-dsn-archivo", e.dsnArchivo, "-salida", e.salida}
	out.Reset()
	errOut.Reset()
	if rc := ejecutar(context.Background(), args, "", false, &out, &errOut, d); rc != 2 {
		t.Fatal("LOGIN RRHH explícito vacío admitido")
	}
}

func TestPrepararIncorporacionRechazaAmbosDSNBajoGitAntesDeConectar(t *testing.T) {
	e, ruta, d := escenarioIncorporacion(t)
	git := filepath.Join(e.dir, "arbol-git")
	if err := os.Mkdir(git, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(git, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	privado := filepath.Join(git, "privado")
	if err := os.Mkdir(privado, 0700); err != nil {
		t.Fatal(err)
	}
	rutaGit := filepath.Join(privado, "dsn")
	escribir(t, rutaGit, []byte("postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1:1/vec\n"), 0600)
	abiertas := 0
	d.abrirGobierno = func(context.Context, string) (fuenteGobierno, error) {
		abiertas++
		return e.gobierno, nil
	}
	for _, opciones := range []opciones{
		{incorporacionMotivosRRHHDSN: rutaGit, idempotencia: e.idempotencia, incorporacionConfig: ruta, dsnArchivo: e.dsnArchivo, salida: e.salida},
		{incorporacionMotivosRRHHDSN: e.dsnArchivo, idempotencia: e.idempotencia, incorporacionConfig: ruta, dsnArchivo: rutaGit, salida: e.salida},
	} {
		p := preparacion{opciones: opciones, dep: d}
		if _, err := p.prepararIncorporacion(context.Background()); err == nil || abiertas != 0 {
			t.Fatal("DSN bajo Git admitido o gobierno conectado")
		}
		e.sinResiduos(t)
	}
}

func TestPrepararIncorporacionDesdeServidorRechazaRaizSPKIYModoAmbiguo(t *testing.T) {
	e, ruta, d := escenarioIncorporacion(t)
	coordenadas, err := bootstrap.DerivarCoordenadasCTPreparacion(e.idempotencia, ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	e.gobierno.raiz = filaRaiz{ClaveID: coordenadas.RaizID, Version: 7, Audiencia: coordenadas.Audiencia,
		HuellaSPKI: strings.Repeat("f", 64), Vigente: true}
	motivosRRHH := filepath.Join(e.dir, "motivos_rrhh.dsn")
	escribir(t, motivosRRHH, []byte("postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1:1/vec\n"), 0600)
	args := []string{"-incorporacion-motivos-rrhh-dsn", motivosRRHH, "-material-idempotencia", e.idempotencia,
		"-incorporacion-config", ruta, "-dsn-archivo", e.dsnArchivo, "-salida", e.salida}
	var out, errOut bytes.Buffer
	if rc := ejecutar(context.Background(), args, "", false, &out, &errOut, d); rc != 1 || !strings.Contains(errOut.String(), "raíz vigente") {
		t.Fatalf("raíz distinta aceptada rc=%d: %s", rc, errOut.String())
	}
	e.sinResiduos(t)
	e.gobierno.raiz.HuellaSPKI = coordenadas.HuellaSPKI
	d.resolverMotivoDetalle = func(context.Context, string, time.Time) (core.ReferenciaEntradaCatalogo, error) {
		return core.ReferenciaEntradaCatalogo{}, nil
	}
	out.Reset()
	errOut.Reset()
	if rc := ejecutar(context.Background(), args, "", false, &out, &errOut, d); rc != 1 || !strings.Contains(errOut.String(), "ct_detalle") {
		t.Fatalf("motivo de detalle distinto aceptado rc=%d: %s", rc, errOut.String())
	}
	e.sinResiduos(t)
	args = append(args, "-inventario-ct", e.inventarioCT)
	out.Reset()
	errOut.Reset()
	if rc := ejecutar(context.Background(), args, "", false, &out, &errOut, d); rc != 2 {
		t.Fatalf("fuentes CT ambiguas aceptadas rc=%d", rc)
	}
	e.sinResiduos(t)
}

func TestPrepararIncorporacionGobiernoAusenteNoEscribe(t *testing.T) {
	e, ruta, d := escenarioIncorporacion(t)
	e.gobierno.claves = map[string]filaClave{}
	p := preparacion{opciones: opciones{inventarioCT: e.inventarioCT, idempotencia: e.idempotencia, incorporacionConfig: ruta, dsnArchivo: e.dsnArchivo, salida: e.salida}, dep: d}
	if _, err := p.prepararIncorporacion(context.Background()); err == nil {
		t.Fatal("aceptó gobierno ausente")
	}
	e.sinResiduos(t)
}

func TestPrepararIncorporacionMotivoDetalleDistintoNoEscribe(t *testing.T) {
	e, ruta, d := escenarioIncorporacion(t)
	d.resolverMotivoDetalle = func(context.Context, string, time.Time) (core.ReferenciaEntradaCatalogo, error) {
		return core.ReferenciaEntradaCatalogo{}, nil
	}
	p := preparacion{opciones: opciones{inventarioCT: e.inventarioCT, idempotencia: e.idempotencia, incorporacionConfig: ruta, dsnArchivo: e.dsnArchivo, salida: e.salida}, dep: d}
	if _, err := p.prepararIncorporacion(context.Background()); err == nil {
		t.Fatal("aceptó detalle distinto")
	}
	e.sinResiduos(t)
}

func TestPrepararIncorporacionMotivoRechazadoIndicaClaveSinDatos(t *testing.T) {
	e, ruta, d := escenarioIncorporacion(t)
	d.validarMotivosIncorporacion = func(ctx context.Context, c bootstrap.ConfiguracionPreparacionIncorporacionB2, r *os.Root, ahora time.Time) error {
		return bootstrap.ValidarMotivosPreparacionIncorporacionB2(ctx, c, nil, ahora)
	}
	p := preparacion{opciones: opciones{inventarioCT: e.inventarioCT, idempotencia: e.idempotencia, incorporacionConfig: ruta, dsnArchivo: e.dsnArchivo, salida: e.salida}, dep: d}
	_, err := p.prepararIncorporacion(context.Background())
	if err == nil || mensajeSeguro(err) != "motivo B2 motivos_autorizacion: esperado=motivo publicado vigente observado=no disponible o distinto" {
		t.Fatalf("diagnóstico no nominal: %v", err)
	}
	e.sinResiduos(t)
}

func TestPrepararIncorporacionRechazaMaterialNoPublicadoPostgreSQL18(t *testing.T) {
	if os.Getenv("VEC_KITB2_TEST_DESECHABLE") != "si" {
		t.Skip("requiere clon PostgreSQL 18 desechable de main")
	}
	e, ruta, _ := escenarioIncorporacion(t)
	d := dependencias{abrirGobierno: abrirGobiernoPostgreSQL, reloj: func() time.Time { return ahoraPrueba }}
	p := preparacion{opciones: opciones{inventarioCT: e.inventarioCT, idempotencia: e.idempotencia, incorporacionConfig: ruta, salida: e.salida}, dsnEntorno: os.Getenv("VEC_KITB2_TEST_GOBIERNO_DSN"), dep: d}
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	_, err := p.prepararIncorporacion(ctx)
	if err == nil || !strings.Contains(mensajeSeguro(err), "clave B2 bolsa_anclaje:") {
		t.Fatalf("material sin publicación aceptado o fallo ajeno: %v", err)
	}
	e.sinResiduos(t)
}
