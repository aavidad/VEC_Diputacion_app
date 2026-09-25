package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/bootstrap"
	"vec-diputacion-granada/internal/app/composicion/internactproveedores"
)

const (
	catalogoPrueba = "motivos.b2"
	emisorPrueba   = "emisor:ct:desarrollo:v1"
	raizPrueba     = "clave:atestacion:ct:desarrollo:v1"
	baseIDPrueba   = "clave:capacidad:ct:desarrollo:v1"
	audienciaCT    = "vec:desarrollo:contratacion-temporal:atestacion:v3"
)

var (
	desdePrueba = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	hastaPrueba = time.Date(2036, 1, 1, 0, 0, 0, 0, time.UTC)
	ahoraPrueba = time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
)

// gobiernoFalso es una instantánea en memoria indexada por huella del
// secreto, como la consulta real.
type gobiernoFalso struct {
	claves  map[string]filaClave
	raiz    filaRaiz
	errRaiz error
	cerrado bool
}

func (g *gobiernoFalso) clavePorHuellaSecreto(_ context.Context, h string) (filaClave, error) {
	f, ok := g.claves[h]
	if !ok {
		return filaClave{}, errSinFila
	}
	return f, nil
}
func (g *gobiernoFalso) raizVigente(context.Context) (filaRaiz, error) { return g.raiz, g.errRaiz }
func (g *gobiernoFalso) cerrar()                                       { g.cerrado = true }

type escenario struct {
	dir, inventarioCT, claveBase, motivos, salida, dsnArchivo string
	base                                                      []byte
	gobierno                                                  *gobiernoFalso
	huellasB2                                                 [8]string
}

func huellaHex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func escribir(t *testing.T, ruta string, b []byte, modo os.FileMode) {
	t.Helper()
	if err := os.WriteFile(ruta, b, modo); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(ruta, modo); err != nil {
		t.Fatal(err)
	}
}

func referencia(n int) map[string]any {
	return map[string]any{"catalogo_id": catalogoPrueba, "catalogo_version": 3, "catalogo_huella_sha256": strings.Repeat("b", 64),
		"entrada_clave": "motivo_" + strings.Repeat(string("0123456789abcdef"[n]), 32)}
}

func motivosJSON() map[string]any {
	m := map[string]any{}
	for i, d := range bootstrap.DescriptoresCapacidadPersonalB2V3Desarrollo() {
		m[d.Capacidad] = referencia(i + 1)
	}
	return m
}

func capacidadCT() map[string]any {
	return map[string]any{"clave_id": "clave:capacidad:ct-alta:desarrollo:v1", "version": 1, "archivo": "ct_alta.hmac", "sha256": strings.Repeat("c", 64),
		"emisor_id": emisorPrueba, "desde": desdePrueba, "hasta": hastaPrueba, "revision_gobierno": 1, "huella_gobierno": strings.Repeat("d", 64)}
}

func inventarioCTJSON() map[string]any {
	pools := map[string]any{}
	for i, n := range []string{"fuente_autorizacion", "registro_autorizacion", "motivos_autorizacion", "motivos_rrhh", "consulta_rrhh", "gobierno_v3"} {
		pools[n] = map[string]any{"dsn": "DSN_PRIVADO_PRUEBA_" + n, "login": "login_" + string(rune('a'+i))}
	}
	ref := referencia(0)
	return map[string]any{
		"version": 1, "pools": pools, "planes_archivo": "planes.json",
		"terna_planes":     map[string]any{"referencia": "planes:ensamblaje:v2", "version": 2, "huella_sha256": strings.Repeat("e", 64)},
		"personal_archivo": "personal.json",
		"terna_personal":   map[string]any{"referencia": "personal:fuente:v1", "version": 1, "huella_sha256": strings.Repeat("f", 64)},
		"catalogo_motivos": catalogoPrueba, "motivo_alta": ref, "motivo_lectura": ref,
		"v3": map[string]any{"clave_id": raizPrueba, "clave_version": 1, "clave_archivo": "raiz.ed25519", "clave_sha256": strings.Repeat("1", 64),
			"audiencia": audienciaCT, "raiz_desde": desdePrueba, "raiz_hasta": hastaPrueba,
			"configuracion_ref": "confianza:atestacion:ct:desarrollo:2026-09-24", "configuracion_orden": 20260924,
			"configuracion_publicada": ahoraPrueba.Add(-36 * time.Hour), "configuracion_expira": ahoraPrueba.Add(-12 * time.Hour),
			"configuracion_sha256": strings.Repeat("2", 64),
			"capacidades":          map[string]any{"alta": capacidadCT(), "lectura": capacidadCT(), "ct": capacidadCT(), "cuadro": capacidadCT(), "detalle": capacidadCT()}},
	}
}

func jsonDe(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// nuevoEscenario prepara material privado sintético y un gobierno publicado
// coherente con la derivación única de T3.
func nuevoEscenario(t *testing.T) *escenario {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	e := &escenario{dir: dir}
	ctDir := filepath.Join(dir, "ct")
	privado := filepath.Join(dir, "privado")
	for _, d := range []string{ctDir, privado} {
		if err := os.Mkdir(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	e.inventarioCT = filepath.Join(ctDir, "ct_v3.json")
	escribir(t, e.inventarioCT, jsonDe(t, inventarioCTJSON()), 0600)
	e.base = make([]byte, 32)
	if _, err := rand.Read(e.base); err != nil {
		t.Fatal(err)
	}
	e.claveBase = filepath.Join(privado, "clave_base.hmac")
	escribir(t, e.claveBase, e.base, 0600)
	e.motivos = filepath.Join(privado, "motivos_b2.json")
	escribir(t, e.motivos, jsonDe(t, motivosJSON()), 0600)
	e.dsnArchivo = filepath.Join(privado, "gobierno.dsn")
	escribir(t, e.dsnArchivo, []byte("postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1:1/vec\n"), 0600)
	e.salida = filepath.Join(dir, "salida")
	e.gobierno = &gobiernoFalso{claves: map[string]filaClave{}, raiz: filaRaiz{ClaveID: raizPrueba, Version: 1, Audiencia: audienciaCT, Vigente: true}}
	e.gobierno.claves[huellaHex(e.base)] = filaClave{ClaveID: baseIDPrueba, Version: 1, Revision: 1, HuellaGobierno: strings.Repeat("9", 64),
		EmisorID: emisorPrueba, Audiencia: audienciaClaveBaseCT, Desde: desdePrueba, Hasta: hastaPrueba, ActoPropio: true, Vigente: true, DentroCheckpoint: true, PunteroVigente: true}
	claves, err := bootstrap.DerivarClavesPersonalB2V3Desarrollo(e.base, baseIDPrueba, emisorPrueba, desdePrueba, hastaPrueba)
	if err != nil {
		t.Fatal(err)
	}
	for i := range claves {
		c := claves[i]
		e.huellasB2[i] = c.SHA256
		e.gobierno.claves[c.SHA256] = filaClave{ClaveID: c.ClaveID, Version: uint64(40 + i), Revision: uint64(60 + i), HuellaGobierno: c.HuellaGobierno,
			EmisorID: c.EmisorID, Audiencia: c.Audiencia, Desde: c.Desde, Hasta: c.Hasta, ActoPropio: true, Vigente: true, DentroCheckpoint: true, PunteroVigente: true}
		claves[i].Borrar()
	}
	return e
}

func (e *escenario) args() []string {
	return []string{"-inventario-ct", e.inventarioCT, "-clave-base", e.claveBase, "-motivos", e.motivos, "-salida", e.salida, "-dsn-archivo", e.dsnArchivo}
}

func (e *escenario) dep() dependencias {
	return dependencias{
		abrirGobierno: func(context.Context, string) (fuenteGobierno, error) { return e.gobierno, nil },
		reloj:         func() time.Time { return ahoraPrueba },
	}
}

func (e *escenario) ejecutar(t *testing.T, d dependencias) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	codigo := ejecutar(context.Background(), e.args(), "", false, &out, &errOut, d)
	texto := out.String() + errOut.String()
	for _, prohibido := range []string{hex.EncodeToString(e.base), base64.StdEncoding.EncodeToString(e.base), "SECRETO_DSN_PRUEBA", "DSN_PRIVADO_PRUEBA", e.dir} {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("la salida expone un dato privado")
		}
	}
	return codigo, texto
}

// sinResiduos comprueba que no queda salida ni directorio temporal.
func (e *escenario) sinResiduos(t *testing.T) {
	t.Helper()
	if _, err := os.Lstat(e.salida); !os.IsNotExist(err) {
		t.Fatal("quedó material de salida tras un fallo")
	}
	entradas, err := os.ReadDir(e.dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range entradas {
		if strings.HasPrefix(x.Name(), ".vec-preparar-material-") {
			t.Fatal("quedó un directorio temporal")
		}
	}
}

func TestPreparaYActivaMaterialCargableConLosCargadoresReales(t *testing.T) {
	e := nuevoEscenario(t)
	codigo, texto := e.ejecutar(t, e.dep())
	if codigo != 0 {
		t.Fatalf("código %d: %s", codigo, texto)
	}
	if !e.gobierno.cerrado {
		t.Fatal("la instantánea de gobierno no se cerró")
	}
	info, err := os.Lstat(e.salida)
	if err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		t.Fatal("la salida no es un directorio 0700")
	}
	entradas, _ := os.ReadDir(e.salida)
	if len(entradas) != 9 {
		t.Fatalf("se esperaban 9 ficheros, hay %d", len(entradas))
	}
	for _, x := range entradas {
		i, err := x.Info()
		if err != nil || !i.Mode().IsRegular() || i.Mode().Perm() != 0600 {
			t.Fatalf("%s no es regular 0600", x.Name())
		}
	}
	m, err := internactproveedores.CargarMaterialPersonalB2(e.salida)
	if err != nil {
		t.Fatalf("cargador real: %v", err)
	}
	defer m.Cerrar()
	if m.Version != 4 || m.CatalogoMotivos != catalogoPrueba || m.V3.Capacidades.Empleados.Version != 47 || m.V3.Capacidades.Ficha.RevisionGobierno != 60 {
		t.Fatal("coordenadas no tomadas del gobierno")
	}
	if m.V3.Capacidades.Hecho.SHA256 != e.huellasB2[3] || m.Motivos.Empleados.EntradaClave != referencia(8)["entrada_clave"] {
		t.Fatal("huella o motivo no corresponden a su capacidad")
	}
	b, err := os.ReadFile(filepath.Join(e.salida, m.V3.Capacidades.Alta.Archivo))
	if err != nil || huellaHex(b) != e.huellasB2[2] {
		t.Fatal("el fichero de la clave no tiene la huella publicada")
	}
	inventario, _ := os.ReadFile(filepath.Join(e.salida, nombreInventarioB2))
	if bytes.Contains(inventario, []byte(hex.EncodeToString(b))) || bytes.Contains(inventario, []byte("dsn")) {
		t.Fatal("el inventario contiene secretos")
	}
}

func TestSalidaVaciaExistenteSeSustituye(t *testing.T) {
	e := nuevoEscenario(t)
	if err := os.Mkdir(e.salida, 0700); err != nil {
		t.Fatal(err)
	}
	if codigo, texto := e.ejecutar(t, e.dep()); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, texto)
	}
	if _, err := internactproveedores.CargarMaterialPersonalB2(e.salida); err != nil {
		t.Fatal("material no cargable")
	}
}

func TestGobiernoInconsistenteFallaCerrado(t *testing.T) {
	casos := map[string]func(e *escenario){
		"clave_b2_ausente": func(e *escenario) { delete(e.gobierno.claves, e.huellasB2[5]) },
		"puntero_no_vigente": func(e *escenario) {
			f := e.gobierno.claves[e.huellasB2[0]]
			f.PunteroVigente = false
			e.gobierno.claves[e.huellasB2[0]] = f
		},
		"clave_revocada": func(e *escenario) {
			f := e.gobierno.claves[e.huellasB2[7]]
			f.Revocada = true
			e.gobierno.claves[e.huellasB2[7]] = f
		},
		"fuera_de_checkpoint": func(e *escenario) {
			f := e.gobierno.claves[e.huellasB2[1]]
			f.DentroCheckpoint = false
			e.gobierno.claves[e.huellasB2[1]] = f
		},
		"clave_caducada": func(e *escenario) {
			f := e.gobierno.claves[e.huellasB2[2]]
			f.Vigente = false
			e.gobierno.claves[e.huellasB2[2]] = f
		},
		"huella_gobierno_ajena": func(e *escenario) {
			f := e.gobierno.claves[e.huellasB2[3]]
			f.HuellaGobierno = strings.Repeat("7", 64)
			e.gobierno.claves[e.huellasB2[3]] = f
		},
		"audiencia_cruzada": func(e *escenario) {
			f := e.gobierno.claves[e.huellasB2[4]]
			f.Audiencia = audienciaClaveBaseCT
			e.gobierno.claves[e.huellasB2[4]] = f
		},
		"vigencia_distinta": func(e *escenario) {
			f := e.gobierno.claves[e.huellasB2[6]]
			f.Hasta = hastaPrueba.Add(time.Hour)
			e.gobierno.claves[e.huellasB2[6]] = f
		},
		"acto_ajeno": func(e *escenario) {
			f := e.gobierno.claves[e.huellasB2[6]]
			f.ActoPropio = false
			e.gobierno.claves[e.huellasB2[6]] = f
		},
		"base_no_propia": func(e *escenario) {
			h := huellaHex(e.base)
			f := e.gobierno.claves[h]
			f.ActoPropio = false
			e.gobierno.claves[h] = f
		},
		"base_otra_audiencia": func(e *escenario) {
			h := huellaHex(e.base)
			f := e.gobierno.claves[h]
			f.Audiencia = "otra.v1"
			e.gobierno.claves[h] = f
		},
		"emisor_base_distinto": func(e *escenario) {
			h := huellaHex(e.base)
			f := e.gobierno.claves[h]
			f.EmisorID = "emisor:otro"
			e.gobierno.claves[h] = f
		},
		"raiz_distinta":   func(e *escenario) { e.gobierno.raiz.ClaveID = "clave:atestacion:ct:desarrollo:v2" },
		"raiz_no_vigente": func(e *escenario) { e.gobierno.raiz.Vigente = false },
		"raiz_ilegible":   func(e *escenario) { e.gobierno.errRaiz = errRaiz },
		"version_publicada_cero": func(e *escenario) {
			f := e.gobierno.claves[e.huellasB2[0]]
			f.Version = 0
			e.gobierno.claves[e.huellasB2[0]] = f
		},
		"revision_publicada_cero": func(e *escenario) {
			f := e.gobierno.claves[e.huellasB2[0]]
			f.Revision = 0
			e.gobierno.claves[e.huellasB2[0]] = f
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEscenario(t)
			alterar(e)
			codigo, texto := e.ejecutar(t, e.dep())
			if codigo != 1 {
				t.Fatalf("aceptó un gobierno inconsistente: %d", codigo)
			}
			motivoCotejo := false
			for _, m := range []errorPropio{errClaveB2, errRaiz, errClaveBaseNoPub, errEmisorDistinto} {
				motivoCotejo = motivoCotejo || strings.Contains(texto, string(m))
			}
			if !motivoCotejo {
				t.Fatalf("falló por otra causa: %s", texto)
			}
			e.sinResiduos(t)
		})
	}
}

func TestMaterialPrivadoDivergenteFallaCerrado(t *testing.T) {
	e := nuevoEscenario(t)
	otra := make([]byte, 32)
	if _, err := rand.Read(otra); err != nil {
		t.Fatal(err)
	}
	escribir(t, e.claveBase, otra, 0600)
	codigo, texto := e.ejecutar(t, e.dep())
	if codigo != 1 || !strings.Contains(texto, string(errClaveBaseNoPub)) {
		t.Fatalf("clave base errónea aceptada: %d %s", codigo, texto)
	}
	e.sinResiduos(t)

	// Clave base publicada, pero el gobierno B2 procede de otra base: las
	// huellas derivadas no casan.
	e = nuevoEscenario(t)
	f := e.gobierno.claves[huellaHex(e.base)]
	delete(e.gobierno.claves, huellaHex(e.base))
	e.gobierno.claves[huellaHex(otra)] = f
	escribir(t, e.claveBase, otra, 0600)
	if codigo, _ := e.ejecutar(t, e.dep()); codigo != 1 {
		t.Fatal("derivación de otra base aceptada")
	}
	e.sinResiduos(t)
}

func TestPermisosInsegurosRechazados(t *testing.T) {
	casos := map[string]func(t *testing.T, e *escenario){
		"clave_base_0644": func(t *testing.T, e *escenario) { must(t, os.Chmod(e.claveBase, 0644)) },
		"clave_base_0400": func(t *testing.T, e *escenario) { must(t, os.Chmod(e.claveBase, 0400)) },
		"motivos_0640":    func(t *testing.T, e *escenario) { must(t, os.Chmod(e.motivos, 0640)) },
		"dsn_0644":        func(t *testing.T, e *escenario) { must(t, os.Chmod(e.dsnArchivo, 0644)) },
		"ct_dir_0755":     func(t *testing.T, e *escenario) { must(t, os.Chmod(filepath.Dir(e.inventarioCT), 0755)) },
		"ct_json_0644":    func(t *testing.T, e *escenario) { must(t, os.Chmod(e.inventarioCT, 0644)) },
		"salida_0755":     func(t *testing.T, e *escenario) { must(t, os.Mkdir(e.salida, 0755)); must(t, os.Chmod(e.salida, 0755)) },
		"padre_0777":      func(t *testing.T, e *escenario) { must(t, os.Chmod(e.dir, 0777)) },
		"clave_base_corta": func(t *testing.T, e *escenario) {
			escribir(t, e.claveBase, e.base[:16], 0600)
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEscenario(t)
			alterar(t, e)
			if codigo, _ := e.ejecutar(t, e.dep()); codigo != 1 {
				t.Fatal("permisos inseguros aceptados")
			}
			if nombre != "salida_0755" {
				e.sinResiduos(t)
			}
			must(t, os.Chmod(e.dir, 0700))
		})
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestEnlacesSimbolicosRechazados(t *testing.T) {
	casos := map[string]func(t *testing.T, e *escenario){
		"clave_base": func(t *testing.T, e *escenario) {
			real := filepath.Join(e.dir, "privado", "real.hmac")
			must(t, os.Rename(e.claveBase, real))
			must(t, os.Symlink(real, e.claveBase))
		},
		"motivos": func(t *testing.T, e *escenario) {
			real := filepath.Join(e.dir, "privado", "real.json")
			must(t, os.Rename(e.motivos, real))
			must(t, os.Symlink(real, e.motivos))
		},
		"dsn": func(t *testing.T, e *escenario) {
			real := filepath.Join(e.dir, "privado", "real.dsn")
			must(t, os.Rename(e.dsnArchivo, real))
			must(t, os.Symlink(real, e.dsnArchivo))
		},
		"directorio_ct": func(t *testing.T, e *escenario) {
			enlace := filepath.Join(e.dir, "ct-enlace")
			must(t, os.Symlink(filepath.Dir(e.inventarioCT), enlace))
			e.inventarioCT = filepath.Join(enlace, "ct_v3.json")
		},
		"salida": func(t *testing.T, e *escenario) {
			real := filepath.Join(e.dir, "destino-real")
			must(t, os.Mkdir(real, 0700))
			must(t, os.Symlink(real, e.salida))
		},
		"componente_intermedio_salida": func(t *testing.T, e *escenario) {
			real := filepath.Join(e.dir, "padre-real")
			must(t, os.Mkdir(real, 0700))
			enlace := filepath.Join(e.dir, "padre-enlace")
			must(t, os.Symlink(real, enlace))
			e.salida = filepath.Join(enlace, "salida")
		},
	}
	for nombre, alterar := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEscenario(t)
			alterar(t, e)
			if codigo, _ := e.ejecutar(t, e.dep()); codigo != 1 {
				t.Fatal("enlace simbólico aceptado")
			}
			if nombre != "salida" {
				e.sinResiduos(t)
			}
		})
	}
}

func TestSalidaConContenidoNoSeTocaNiSeSustituye(t *testing.T) {
	e := nuevoEscenario(t)
	must(t, os.Mkdir(e.salida, 0700))
	previo := filepath.Join(e.salida, "personal_b2_v3.json")
	escribir(t, previo, []byte("previo"), 0600)
	if codigo, _ := e.ejecutar(t, e.dep()); codigo != 1 {
		t.Fatal("aceptó una salida con contenido")
	}
	b, err := os.ReadFile(previo)
	if err != nil || string(b) != "previo" {
		t.Fatal("se alteró el contenido previo")
	}
	entradas, _ := os.ReadDir(e.salida)
	if len(entradas) != 1 {
		t.Fatal("se añadieron ficheros a la salida")
	}
}

func TestInterrupcionAntesDeActivarNoDejaMaterialParcial(t *testing.T) {
	e := nuevoEscenario(t)
	d := e.dep()
	var temporales []string
	d.antesDeActivar = func() error {
		// En este punto el temporal ya está completo y validado.
		entradas, _ := os.ReadDir(e.dir)
		for _, x := range entradas {
			if strings.HasPrefix(x.Name(), ".vec-preparar-material-") {
				temporales = append(temporales, x.Name())
			}
		}
		return errors.New("interrupción simulada")
	}
	if codigo, _ := e.ejecutar(t, d); codigo != 1 {
		t.Fatal("la interrupción no hizo fallar la preparación")
	}
	if len(temporales) != 1 {
		t.Fatal("no se compuso en un temporal hermano")
	}
	e.sinResiduos(t)

	// Contexto cancelado (SIGINT/SIGTERM): tampoco se abre el gobierno.
	e = nuevoEscenario(t)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	abierto := false
	d = e.dep()
	d.abrirGobierno = func(context.Context, string) (fuenteGobierno, error) { abierto = true; return e.gobierno, nil }
	var out, errOut bytes.Buffer
	if codigo := ejecutar(ctx, e.args(), "", false, &out, &errOut, d); codigo != 1 || abierto {
		t.Fatal("contexto cancelado no detuvo la preparación")
	}
	e.sinResiduos(t)
}

func TestSalidaVaciaReaparecidaConContenidoFallaSinPerdida(t *testing.T) {
	e := nuevoEscenario(t)
	d := e.dep()
	d.antesDeActivar = func() error {
		// Carrera: otro proceso crea la salida con contenido tras la comprobación.
		must(t, os.Mkdir(e.salida, 0700))
		escribir(t, filepath.Join(e.salida, "ajeno"), []byte("x"), 0600)
		return nil
	}
	if codigo, _ := e.ejecutar(t, d); codigo != 1 {
		t.Fatal("activó sobre una salida ajena")
	}
	if b, err := os.ReadFile(filepath.Join(e.salida, "ajeno")); err != nil || string(b) != "x" {
		t.Fatal("se alteró la salida ajena")
	}
}

func TestMotivosInvalidos(t *testing.T) {
	casos := map[string]func(m map[string]any) []byte{
		"campo_desconocido": func(m map[string]any) []byte { m["extra"] = referencia(1); b, _ := json.Marshal(m); return b },
		"falta_uno":         func(m map[string]any) []byte { delete(m, "empleados"); b, _ := json.Marshal(m); return b },
		"catalogo_distinto": func(m map[string]any) []byte {
			r := referencia(2)
			r["catalogo_id"] = "motivos.otros"
			m["alta"] = r
			b, _ := json.Marshal(m)
			return b
		},
		"clave_no_opaca": func(m map[string]any) []byte {
			r := referencia(2)
			r["entrada_clave"] = "motivo_legible"
			m["hecho"] = r
			b, _ := json.Marshal(m)
			return b
		},
		"clave_duplicada": func(m map[string]any) []byte {
			b, _ := json.Marshal(m)
			r, _ := json.Marshal(referencia(3))
			return append(append([]byte(`{"ficha":`), r...), append([]byte(","), b[1:]...)...)
		},
		"dos_documentos": func(m map[string]any) []byte { b, _ := json.Marshal(m); return append(b, b...) },
	}
	for nombre, construir := range casos {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEscenario(t)
			escribir(t, e.motivos, construir(motivosJSON()), 0600)
			codigo, texto := e.ejecutar(t, e.dep())
			if codigo != 1 || !strings.Contains(texto, string(errMotivos)) {
				t.Fatalf("motivos inválidos aceptados: %s", texto)
			}
			e.sinResiduos(t)
		})
	}
}

func TestDSNSoloPorFicheroOEntornoYNuncaEnErrores(t *testing.T) {
	e := nuevoEscenario(t)
	var out, errOut bytes.Buffer
	if codigo := ejecutar(context.Background(), e.args(), "postgres://x:SECRETO_DSN_PRUEBA@h/d", true, &out, &errOut, e.dep()); codigo != 2 {
		t.Fatal("aceptó dos orígenes de DSN")
	}
	sinDSN := e.args()[:len(e.args())-2]
	if codigo := ejecutar(context.Background(), sinDSN, "", false, &out, &errOut, e.dep()); codigo != 2 {
		t.Fatal("aceptó ausencia de DSN")
	}
	if codigo := ejecutar(context.Background(), append(sinDSN, "postgres://posicional"), "", false, &out, &errOut, e.dep()); codigo != 2 {
		t.Fatal("aceptó argumentos posicionales")
	}
	d := e.dep()
	var recibido string
	d.abrirGobierno = func(_ context.Context, dsn string) (fuenteGobierno, error) {
		recibido = dsn
		return nil, errors.New("fallo con postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1")
	}
	if codigo := ejecutar(context.Background(), sinDSN, "postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1:1/vec", true, &out, &errOut, d); codigo != 1 || recibido == "" {
		t.Fatal("el DSN de entorno no llegó a la conexión")
	}
	if strings.Contains(out.String()+errOut.String(), "SECRETO_DSN_PRUEBA") {
		t.Fatal("el DSN aparece en la salida")
	}
	e.sinResiduos(t)
}

func TestConsultasNoLeenSecretosDelGobierno(t *testing.T) {
	for _, q := range []string{sqlClavePorHuella, sqlRaizVigente} {
		if strings.Contains(q, "secreto_hmac") || strings.Contains(strings.ToUpper(q), "INSERT") || strings.Contains(strings.ToUpper(q), "UPDATE") {
			t.Fatal("una consulta lee secretos o escribe")
		}
	}
}

func TestInventarioCTConEmisoresDistintosRechazado(t *testing.T) {
	e := nuevoEscenario(t)
	inv := inventarioCTJSON()
	c := capacidadCT()
	c["emisor_id"] = "emisor:otro"
	inv["v3"].(map[string]any)["capacidades"].(map[string]any)["detalle"] = c
	escribir(t, e.inventarioCT, jsonDe(t, inv), 0600)
	if codigo, texto := e.ejecutar(t, e.dep()); codigo != 1 || !strings.Contains(texto, string(errEmisorCT)) {
		t.Fatalf("emisores distintos aceptados: %s", texto)
	}
	e.sinResiduos(t)
}
