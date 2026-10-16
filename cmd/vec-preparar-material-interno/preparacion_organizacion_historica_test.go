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
			if _, err := p.prepararOrganizacionHistorica(context.Background(), ruta, ""); err == nil {
				t.Fatal("OH inválida admitida")
			}
			e.sinResiduos(t)
		})
	}
}

func escenarioOHServidor(t *testing.T) (*escenario, string, string, dependencias) {
	t.Helper()
	e, ruta, d := escenarioOH(t)
	coordenadas, err := bootstrap.DerivarCoordenadasCTPreparacion(e.idempotencia, ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	e.gobierno.raiz = filaRaiz{ClaveID: coordenadas.RaizID, Version: 7,
		Audiencia: coordenadas.Audiencia, HuellaSPKI: coordenadas.HuellaSPKI, Vigente: true}
	dsn := filepath.Join(e.dir, "motivos-oh.dsn")
	escribir(t, dsn, []byte("postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1:1/vec\n"), 0600)
	d.validarMotivoOH = func(_ context.Context, rutaDSN string, motivo core.ReferenciaEntradaCatalogo, _ time.Time) error {
		if rutaDSN != dsn || motivo.CatalogoID != catalogoPrueba {
			return errMotivoOH
		}
		return nil
	}
	return e, ruta, dsn, d
}

func argsOHServidor(e *escenario, entrada, motivosDSN string) []string {
	return []string{"-material-idempotencia", e.idempotencia, "-organizacion-historica-config", entrada,
		"-organizacion-historica-motivos-dsn", motivosDSN, "-dsn-archivo", e.dsnArchivo, "-salida", e.salida}
}

func TestPreparaOHDesdeServidorSinInventarioCT(t *testing.T) {
	e, entrada, motivosDSN, d := escenarioOHServidor(t)
	var out, stderr bytes.Buffer
	if rc := ejecutar(context.Background(), argsOHServidor(e, entrada, motivosDSN), "", false, &out, &stderr, d); rc != 0 {
		t.Fatalf("rc=%d %s", rc, stderr.String())
	}
	m, err := internactproveedores.CargarMaterialOrganizacionHistorica(e.salida)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Cerrar()
	if m.Capacidad.Version != 71 || m.Capacidad.RevisionGobierno != 83 ||
		m.Contextos["cta_0123456789abcdef0123456789abcdef"].PerfilVersion != 7 {
		t.Fatal("OH sustituyó coordenadas o contexto")
	}
	if f, err := os.ReadDir(e.salida); err != nil || len(f) != 2 {
		t.Fatal("OH escribió material ajeno", err)
	}
}

func TestOHServidorRechazaFuentesAmbiguasYDSNAusente(t *testing.T) {
	e, entrada, motivosDSN, d := escenarioOHServidor(t)
	dsnInseguro := filepath.Join(e.dir, "motivos-inseguro.dsn")
	escribir(t, dsnInseguro, []byte("postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1:1/vec\n"), 0644)
	directorioGit := filepath.Join(e.dir, "arbol-git")
	if err := os.Mkdir(directorioGit, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(directorioGit, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	dsnGit := filepath.Join(directorioGit, "motivos.dsn")
	escribir(t, dsnGit, []byte("postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1:1/vec\n"), 0600)
	for _, caso := range []struct {
		nombre string
		args   []string
		codigo int
	}{
		{"sin_fuente", []string{"-material-idempotencia", e.idempotencia, "-organizacion-historica-config", entrada, "-dsn-archivo", e.dsnArchivo, "-salida", e.salida}, 2},
		{"dos_fuentes", append(argsOHServidor(e, entrada, motivosDSN), "-inventario-ct", e.inventarioCT), 2},
		{"fuente_b2", append(argsOHServidor(e, entrada, motivosDSN), "-incorporacion-motivos-rrhh-dsn", motivosDSN), 2},
		{"dsn_vacio", argsOHServidor(e, entrada, ""), 2},
		{"dsn_ausente", argsOHServidor(e, entrada, filepath.Join(e.dir, "ausente.dsn")), 1},
		{"dsn_inseguro", argsOHServidor(e, entrada, dsnInseguro), 1},
		{"dsn_bajo_git", argsOHServidor(e, entrada, dsnGit), 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			var out, stderr bytes.Buffer
			if rc := ejecutar(context.Background(), caso.args, "", false, &out, &stderr, d); rc != caso.codigo {
				t.Fatalf("rc=%d esperado=%d %s", rc, caso.codigo, stderr.String())
			}
			e.sinResiduos(t)
		})
	}
}

func TestMotivoOHServidorExigeDSNPrivado(t *testing.T) {
	e, _, _, _ := escenarioOHServidor(t)
	for _, caso := range []struct {
		nombre string
		modo   os.FileMode
	}{
		{"lectura_grupo", 0640},
		{"vacio", 0600},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			ruta := filepath.Join(e.dir, "motivo-"+caso.nombre+".dsn")
			contenido := []byte("postgres://lector@127.0.0.1:1/vec\n")
			if caso.nombre == "vacio" {
				contenido = []byte("\n")
			}
			escribir(t, ruta, contenido, caso.modo)
			if err := validarMotivoOrganizacionHistoricaServidor(context.Background(), ruta, core.ReferenciaEntradaCatalogo{}, ahoraPrueba); err == nil {
				t.Fatal("DSN no privado admitido")
			}
		})
	}
	objetivo := filepath.Join(e.dir, "dsn-real")
	escribir(t, objetivo, []byte("postgres://lector@127.0.0.1:1/vec\n"), 0600)
	enlace := filepath.Join(e.dir, "dsn-enlace")
	if err := os.Symlink(objetivo, enlace); err != nil {
		t.Fatal(err)
	}
	if err := validarMotivoOrganizacionHistoricaServidor(context.Background(), enlace, core.ReferenciaEntradaCatalogo{}, ahoraPrueba); err == nil {
		t.Fatal("DSN enlazado admitido")
	}
}

func TestOHServidorRechazaRaizClaveYMotivoAjenos(t *testing.T) {
	for _, caso := range []string{"raiz", "spki", "version", "clave", "revocada", "audiencia", "puntero", "emisor", "motivo"} {
		t.Run(caso, func(t *testing.T) {
			e, entrada, motivosDSN, d := escenarioOHServidor(t)
			switch caso {
			case "raiz":
				e.gobierno.raiz.ClaveID = "clave:ajena"
			case "spki":
				e.gobierno.raiz.HuellaSPKI = strings.Repeat("f", 64)
			case "version":
				e.gobierno.raiz.Version = 0
			case "clave", "revocada", "audiencia", "puntero", "emisor":
				for h, f := range e.gobierno.claves {
					if strings.HasPrefix(f.ClaveID, "clave:capacidad:personal-organizacion-historica-consulta:") {
						if caso == "clave" {
							delete(e.gobierno.claves, h)
						} else {
							switch caso {
							case "revocada":
								f.Revocada = true
							case "audiencia":
								f.Audiencia = "audiencia:ajena"
							case "puntero":
								f.PunteroVigente = false
							case "emisor":
								f.EmisorID = "emisor:ajeno"
							}
							e.gobierno.claves[h] = f
						}
					}
				}
			case "motivo":
				d.validarMotivoOH = func(context.Context, string, core.ReferenciaEntradaCatalogo, time.Time) error { return errMotivoOH }
			}
			var out, stderr bytes.Buffer
			if rc := ejecutar(context.Background(), argsOHServidor(e, entrada, motivosDSN), "", false, &out, &stderr, d); rc != 1 {
				t.Fatalf("OH ajena admitida rc=%d %s", rc, stderr.String())
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
