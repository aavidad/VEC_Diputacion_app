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
	"fmt"
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
	// La generación activa del material sintético es la 2 (la primera de la
	// configuración), como en el generador de credenciales de desarrollo.
	emisorPrueba = "emisor:ct:desarrollo:v2"
	raizPrueba   = "clave:atestacion:ct:desarrollo:v2"
	audienciaCT  = "vec:desarrollo:contratacion-temporal:atestacion:v3"
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
	dir, inventarioCT, idempotencia, motivos, salida, dsnArchivo string
	// secretos reúne el material de idempotencia y las claves B2 derivadas:
	// ninguno puede aparecer en la salida de la herramienta.
	secretos  [][]byte
	gobierno  *gobiernoFalso
	huellasB2 [8]string
}

// configuracionIdempotencia reproduce idempotencia/configuracion.json de
// scripts/generar_credenciales_desarrollo.sh para las generaciones dadas
// (la primera es la activa).
func configuracionIdempotencia(generaciones ...int) []byte {
	var g []string
	for _, n := range generaciones {
		g = append(g, fmt.Sprintf(`{"generacion":%d,"referencia_localizador":"clave:hmac:convocatorias:localizador:desarrollo:v%d","referencia_huella_solicitud":"clave:hmac:convocatorias:huella:desarrollo:v%d"}`, n, n, n))
	}
	return []byte(`{"version":1,"esquema":"vec.bolsa.convocatoria.idempotencia-hmac.desarrollo.v1","autoridad":"no_autoritativo","version_esquema_hmac":2,"generaciones":[` + strings.Join(g, ",") + `]}`)
}

// escribirIdempotencia deja en dir un material sintético de idempotencia
// con el formato del generador de desarrollo y devuelve sus secretos.
func escribirIdempotencia(t *testing.T, dir string, generaciones ...int) [][]byte {
	t.Helper()
	escribir(t, filepath.Join(dir, "configuracion.json"), configuracionIdempotencia(generaciones...), 0600)
	var secretos [][]byte
	for _, n := range generaciones {
		for _, dominio := range []string{"localizador", "huella-solicitud"} {
			b := make([]byte, 32)
			if _, err := rand.Read(b); err != nil {
				t.Fatal(err)
			}
			escribir(t, filepath.Join(dir, fmt.Sprintf("g%d-%s.bin", n, dominio)), b, 0600)
			secretos = append(secretos, b)
		}
	}
	return secretos
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
	// Material de desarrollo de vec-server sintético: sólo su subdirectorio
	// de idempotencia, que es lo único que lee la herramienta.
	materialVecServer := filepath.Join(dir, "vec-server")
	e.idempotencia = filepath.Join(materialVecServer, "idempotencia")
	for _, d := range []string{materialVecServer, e.idempotencia} {
		if err := os.Mkdir(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	e.secretos = escribirIdempotencia(t, e.idempotencia, 2, 1)
	e.motivos = filepath.Join(privado, "motivos_b2.json")
	escribir(t, e.motivos, jsonDe(t, motivosJSON()), 0600)
	e.dsnArchivo = filepath.Join(privado, "gobierno.dsn")
	escribir(t, e.dsnArchivo, []byte("postgres://lector:SECRETO_DSN_PRUEBA@127.0.0.1:1/vec\n"), 0600)
	e.salida = filepath.Join(dir, "salida")
	e.gobierno = &gobiernoFalso{claves: map[string]filaClave{}, raiz: filaRaiz{ClaveID: raizPrueba, Version: 1, Audiencia: audienciaCT, Vigente: true}}
	e.publicar(t)
	return e
}

// publicar simula el gobierno que deja vec-server a partir del material
// actual: una fila por clave B2 obtenida por la ruta de vec-server.
func (e *escenario) publicar(t *testing.T) {
	t.Helper()
	claves, err := bootstrap.DerivarClavesPersonalB2V3DesdeMaterialDesarrollo(e.idempotencia, ahoraPrueba)
	if err != nil {
		t.Fatal(err)
	}
	for i := range claves {
		c := claves[i]
		e.huellasB2[i] = c.SHA256
		e.secretos = append(e.secretos, claves[i].CopiarSecreto())
		e.gobierno.claves[c.SHA256] = filaClave{ClaveID: c.ClaveID, Version: uint64(40 + i), Revision: uint64(60 + i), HuellaGobierno: c.HuellaGobierno,
			EmisorID: c.EmisorID, Audiencia: c.Audiencia, Desde: c.Desde, Hasta: c.Hasta, ActoPropio: true, Vigente: true, PunteroVigente: true}
		claves[i].Borrar()
	}
}

func (e *escenario) args() []string {
	return []string{"-inventario-ct", e.inventarioCT, "-material-idempotencia", e.idempotencia, "-motivos", e.motivos, "-salida", e.salida, "-dsn-archivo", e.dsnArchivo}
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
	prohibidos := []string{"SECRETO_DSN_PRUEBA", "DSN_PRIVADO_PRUEBA", e.dir}
	for _, s := range e.secretos {
		prohibidos = append(prohibidos, hex.EncodeToString(s), base64.StdEncoding.EncodeToString(s))
	}
	for _, prohibido := range prohibidos {
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
			f.Audiencia = "vec_contratacion_temporal.confirmar_alta_atestada.v1"
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
		"clave_publicada_otro_id": func(e *escenario) {
			f := e.gobierno.claves[e.huellasB2[5]]
			f.ClaveID = "clave:capacidad:personal-b2-catalogo-publicar:desarrollo:v9"
			e.gobierno.claves[e.huellasB2[5]] = f
		},
		"emisor_publicado_distinto": func(e *escenario) {
			f := e.gobierno.claves[e.huellasB2[2]]
			f.EmisorID = "emisor:otro"
			e.gobierno.claves[e.huellasB2[2]] = f
		},
		"raiz_distinta":   func(e *escenario) { e.gobierno.raiz.ClaveID = "clave:atestacion:ct:desarrollo:v3" },
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
			for _, m := range []errorPropio{errClaveB2, errRaiz} {
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
	// El material de idempotencia cambió después de que vec-server publicara
	// (misma generación, otro secreto): las huellas derivadas no casan.
	e := nuevoEscenario(t)
	otra := make([]byte, 32)
	if _, err := rand.Read(otra); err != nil {
		t.Fatal(err)
	}
	escribir(t, filepath.Join(e.idempotencia, "g2-huella-solicitud.bin"), otra, 0600)
	codigo, texto := e.ejecutar(t, e.dep())
	if codigo != 1 || !strings.Contains(texto, string(errClaveB2)) {
		t.Fatalf("material divergente aceptado: %d %s", codigo, texto)
	}
	e.sinResiduos(t)

	// Material rotado a otra generación que el gobierno aún no conoce: el
	// emisor derivado deja de ser el de ct_v3.json.
	e = nuevoEscenario(t)
	e.secretos = append(e.secretos, escribirIdempotencia(t, e.idempotencia, 3, 2)...)
	codigo, texto = e.ejecutar(t, e.dep())
	if codigo != 1 || !strings.Contains(texto, string(errEmisorDistinto)) {
		t.Fatalf("generación no publicada aceptada: %d %s", codigo, texto)
	}
	e.sinResiduos(t)

	// Y aunque el gobierno la publicara, ct_v3.json sigue con el emisor v2.
	e.publicar(t)
	if codigo, _ := e.ejecutar(t, e.dep()); codigo != 1 {
		t.Fatal("emisor distinto del de ct_v3.json aceptado")
	}
	e.sinResiduos(t)
}

func TestPermisosInsegurosRechazados(t *testing.T) {
	casos := map[string]func(t *testing.T, e *escenario){
		"idempotencia_fichero_0640": func(t *testing.T, e *escenario) {
			must(t, os.Chmod(filepath.Join(e.idempotencia, "g1-localizador.bin"), 0640))
		},
		"idempotencia_configuracion_0644": func(t *testing.T, e *escenario) {
			must(t, os.Chmod(filepath.Join(e.idempotencia, "configuracion.json"), 0644))
		},
		"idempotencia_dir_0750":   func(t *testing.T, e *escenario) { must(t, os.Chmod(e.idempotencia, 0750)) },
		"material_vecserver_0755": func(t *testing.T, e *escenario) { must(t, os.Chmod(filepath.Dir(e.idempotencia), 0755)) },
		"motivos_0640":            func(t *testing.T, e *escenario) { must(t, os.Chmod(e.motivos, 0640)) },
		"dsn_0644":                func(t *testing.T, e *escenario) { must(t, os.Chmod(e.dsnArchivo, 0644)) },
		"ct_dir_0755":             func(t *testing.T, e *escenario) { must(t, os.Chmod(filepath.Dir(e.inventarioCT), 0755)) },
		"ct_json_0644":            func(t *testing.T, e *escenario) { must(t, os.Chmod(e.inventarioCT, 0644)) },
		"salida_0755":             func(t *testing.T, e *escenario) { must(t, os.Mkdir(e.salida, 0755)); must(t, os.Chmod(e.salida, 0755)) },
		"padre_0777":              func(t *testing.T, e *escenario) { must(t, os.Chmod(e.dir, 0777)) },
		"idempotencia_corta": func(t *testing.T, e *escenario) {
			escribir(t, filepath.Join(e.idempotencia, "g2-localizador.bin"), make([]byte, 16), 0600)
		},
		"idempotencia_otro_nombre": func(t *testing.T, e *escenario) {
			otro := filepath.Join(filepath.Dir(e.idempotencia), "idempotencia-copia")
			must(t, os.Rename(e.idempotencia, otro))
			e.idempotencia = otro
		},
		"idempotencia_raiz_material": func(t *testing.T, e *escenario) { e.idempotencia = filepath.Dir(e.idempotencia) },
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
		"directorio_idempotencia": func(t *testing.T, e *escenario) {
			real := filepath.Join(e.dir, "idempotencia-real")
			must(t, os.Rename(e.idempotencia, real))
			must(t, os.Symlink(real, e.idempotencia))
		},
		"material_vecserver": func(t *testing.T, e *escenario) {
			material := filepath.Dir(e.idempotencia)
			real := filepath.Join(e.dir, "vec-server-real")
			must(t, os.Rename(material, real))
			must(t, os.Symlink(real, material))
		},
		"fichero_idempotencia": func(t *testing.T, e *escenario) {
			ruta := filepath.Join(e.idempotencia, "g2-huella-solicitud.bin")
			real := filepath.Join(e.dir, "privado", "real.bin")
			must(t, os.Rename(ruta, real))
			must(t, os.Symlink(real, ruta))
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

func TestIdempotenciaDentroDeRepositorioRechazada(t *testing.T) {
	e := nuevoEscenario(t)
	must(t, os.Mkdir(filepath.Join(filepath.Dir(e.idempotencia), ".git"), 0700))
	if err := comprobarIdempotencia(e.idempotencia); err != errIdempotencia {
		t.Fatal("material de idempotencia dentro de un repositorio aceptado")
	}
}

// Tras el rename el material ya está activado: un fallo del fsync del padre
// se informa como tal, no como un error que invite a repetir la preparación.
func TestFsyncDelPadreNoConfirmadoTrasActivar(t *testing.T) {
	e := nuevoEscenario(t)
	d := e.dep()
	d.sincronizarPadre = func(*os.File) error { return errors.New("EIO simulado") }
	codigo, texto := e.ejecutar(t, d)
	if codigo != 0 || !strings.Contains(texto, "activado; fsync del padre no confirmado") {
		t.Fatalf("fsync fallido tras activar mal informado: %d %s", codigo, texto)
	}
	if _, err := internactproveedores.CargarMaterialPersonalB2(e.salida); err != nil {
		t.Fatal("el material activado no es cargable")
	}
}

// Si la ruta del padre pasa a designar otro directorio entre la composición
// y la activación, no se activa en ninguno de los dos y el temporal se
// retira del directorio original por su descriptor.
func TestPadreSustituidoAntesDeActivarNoActiva(t *testing.T) {
	e := nuevoEscenario(t)
	movido := e.dir + "-movido"
	d := e.dep()
	d.antesDeActivar = func() error {
		must(t, os.Rename(e.dir, movido))
		must(t, os.Mkdir(e.dir, 0700))
		return nil
	}
	t.Cleanup(func() { _ = os.RemoveAll(movido) })
	codigo, texto := e.ejecutar(t, d)
	if codigo != 1 || !strings.Contains(texto, string(errActivacion)) {
		t.Fatalf("activó con el padre sustituido: %d %s", codigo, texto)
	}
	e.sinResiduos(t)
	if _, err := os.Lstat(filepath.Join(movido, "salida")); !os.IsNotExist(err) {
		t.Fatal("activó en el directorio original movido")
	}
	entradas, err := os.ReadDir(movido)
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range entradas {
		if strings.HasPrefix(x.Name(), prefijoTemporal) {
			t.Fatal("quedó el temporal en el directorio original")
		}
	}
}

func TestIdentidadDeGobiernoSePropagaSinDetalles(t *testing.T) {
	e := nuevoEscenario(t)
	d := e.dep()
	d.abrirGobierno = func(context.Context, string) (fuenteGobierno, error) { return nil, errIdentidad }
	if codigo, texto := e.ejecutar(t, d); codigo != 1 || !strings.Contains(texto, string(errIdentidad)) {
		t.Fatalf("identidad rechazada mal informada: %s", texto)
	}
	e.sinResiduos(t)
	for _, clausula := range []string{"NOT i.rolsuper", "NOT i.rolbypassrls", "NOT i.rolcreaterole", "NOT d.admin_option", "rol_id NOT IN"} {
		if !strings.Contains(sqlIdentidadGobierno, clausula) {
			t.Fatalf("la comprobación de identidad no exige %s", clausula)
		}
	}
}

// Como la sonda AD3-69, el cotejo no compara la revisión de la clave con
// checkpoint_gobierno.revision (escalas distintas), pero sí conserva los
// mínimos de configuración y raíz del checkpoint como anti-retroceso.
func TestGobiernoSinComparacionEntreEscalasConservaMinimos(t *testing.T) {
	if strings.Contains(sqlClavePorHuella, "checkpoint") {
		t.Fatal("la consulta de clave vuelve a comparar con el checkpoint")
	}
	for _, clausula := range []string{"cfg.secuencia >= ck.configuracion_secuencia_minima", "r.version >= ck.raiz_version_minima"} {
		if !strings.Contains(sqlRaizVigente, clausula) {
			t.Fatalf("la raíz vigente no exige %s", clausula)
		}
	}
}
