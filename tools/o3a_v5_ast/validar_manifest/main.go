package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

type mutante struct {
	ID, Familia, Alternativa, Ruta, Anterior, Posterior, Oraculo string
}
type manifiesto struct {
	Estado                    string
	CoberturaFamiliasCompleta bool `json:"cobertura_familias_completa"`
	Mutantes                  []mutante
}

var archivo = map[string]string{
	"G1":  "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go",
	"G4":  "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio.go",
	"G6a": "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go",
	"G6b": "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go",
	"G6c": "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go",
	"G7a": "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go",
	"G7b": "supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go",
}
var idValido = regexp.MustCompile(`^M[0-9]{3}$`)
var familiaValida = regexp.MustCompile(`^M(?:0[1-9]|[1-5][0-9]|6[0-6])$`)

const baseAutorizada = "1e4bb36b193795688be7575d0ea872fe37033b170bc59561bd1ed33f2a92f8b8"
const recetaAutorizada = "57f4dfb0bd1c5a6602c311a04a8999a35b1cef2ca77147fe1b4ecdf01ec7c706"
const goAutorizado = "8da5fd321795754b994c64e3eb8a5a14ff47bd285559a7e876f3c79abafc67f9"
const versionGoAutorizada = "11b4fb14680701f98ca60fd8464a836ca4374f17896a125f4510ab6ae8cecc9b"
const gotoolAutorizado = "1061bd99d16310f8f549e375a5c0cb18a79d66441ca0ed4dee60f70fde633f9b"
const gorootAutorizado = "b53ebeab1542ea933c6f995a2bcf862d505cb8343ad2b0d1f7a7de3238157ae6"

var fuentesBase = []string{
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_control_preinicio_pruebas.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_operativo.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_sobre_s0.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_autoridad.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_preparacion.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas.go",
	"supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_pruebas_adversas.go",
}

func fallo(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "manifest_o3a_v5=fallo "+f+"\n", a...)
	os.Exit(1)
}

func main() {
	if len(os.Args) != 3 && len(os.Args) != 4 && len(os.Args) != 6 {
		fallo("uso: validar_manifest MANIFEST PRUEBAS_SQL [CATALOGO [LEDGER SEC_LEDGER]]")
	}
	datos, err := os.ReadFile(os.Args[1])
	if err != nil {
		fallo("lectura: %v", err)
	}
	var m manifiesto
	if err = json.Unmarshal(datos, &m); err != nil {
		fallo("json: %v", err)
	}
	if m.Estado != "COMPLETO_EJECUTADO" && m.Estado != "COMPLETO_NO_EJECUTADO" && m.Estado != "PARCIAL_NO_GO" {
		fallo("estado de manifest inválido: %s", m.Estado)
	}
	vistas, familias := map[string]bool{}, map[string]int{}
	alternativas := map[string][]string{}
	for i, x := range m.Mutantes {
		esperado := fmt.Sprintf("M%03d", i+1)
		if !idValido.MatchString(x.ID) || x.ID != esperado || vistas[x.ID] {
			fallo("ID no consecutivo/único en %d: %s", i, x.ID)
		}
		vistas[x.ID] = true
		if !familiaValida.MatchString(x.Familia) {
			fallo("familia inválida %s", x.Familia)
		}
		familias[x.Familia]++
		if x.Alternativa != "" {
			alternativas[x.Familia] = append(alternativas[x.Familia], x.Alternativa)
		}
		nombre, ok := archivo[x.Ruta]
		if !ok {
			fallo("ruta lógica inválida %s", x.Ruta)
		}
		if x.Anterior == "" || x.Posterior == "" || x.Anterior == x.Posterior || x.Oraculo == "" {
			fallo("campos causales incompletos %s", x.ID)
		}
		fuente, err := os.ReadFile(filepath.Join(os.Args[2], nombre))
		if err != nil {
			fallo("fuente %s: %v", x.Ruta, err)
		}
		if n := bytes.Count(fuente, []byte(x.Anterior)); n != 1 {
			fallo("%s patrón anterior cardinalidad=%d", x.ID, n)
		}
	}
	if len(os.Args) >= 4 {
		b, err := os.ReadFile(os.Args[3])
		if err != nil {
			fallo("catálogo: %v", err)
		}
		var esperado map[string][]string
		if err = json.Unmarshal(b, &esperado); err != nil {
			fallo("json catálogo: %v", err)
		}
		for familia, lista := range esperado {
			obtenido := alternativas[familia]
			sort.Strings(lista)
			sort.Strings(obtenido)
			if !reflect.DeepEqual(obtenido, lista) {
				fallo("alternativas %s discrepantes obtenido=%v esperado=%v", familia, obtenido, lista)
			}
		}
	}
	var faltan []string
	for i := 1; i <= 66; i++ {
		f := fmt.Sprintf("M%02d", i)
		if familias[f] == 0 {
			faltan = append(faltan, f)
		}
	}
	if len(faltan) > 0 || !m.CoberturaFamiliasCompleta || (m.Estado != "COMPLETO_EJECUTADO" && m.Estado != "COMPLETO_NO_EJECUTADO") {
		sort.Strings(faltan)
		fallo("expansión incompleta atomicos=%d faltan=%v", len(m.Mutantes), faltan)
	}
	ejecutados := 0
	if m.Estado == "COMPLETO_EJECUTADO" {
		if len(os.Args) != 6 {
			fallo("COMPLETO_EJECUTADO exige catálogo y dos ledgers")
		}
		validarLedgerPrincipal(os.Args[4], os.Args[2], m)
		validarLedgerSeguridad(os.Args[5], os.Args[2])
		ejecutados = len(m.Mutantes)
	}
	fmt.Printf("manifest_o3a_v5=ok atomicos=%d familias=66 ejecutados=%d estado=%s\n", len(m.Mutantes), ejecutados, m.Estado)
}

func leerLedger(ruta string) [][]string {
	b, err := os.ReadFile(ruta)
	if err != nil {
		fallo("ledger %s: %v", ruta, err)
	}
	lineas := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lineas) == 0 || lineas[0] != "base_sha\tid\tcompilo\toraculo\testado\tclasificacion" {
		fallo("cabecera ledger inválida: %s", ruta)
	}
	var filas [][]string
	for _, linea := range lineas[1:] {
		campos := strings.Split(linea, "\t")
		if len(campos) != 6 {
			fallo("fila ledger inválida %s: %q", ruta, linea)
		}
		filas = append(filas, campos)
	}
	return filas
}

func validarBase(f []string) string {
	if f[1] != "BASE" || f[2] != "si" || f[3] != "autoprueba" || f[4] != "0" || f[5] != "verde" {
		fallo("control BASE inválido: %v", f)
	}
	if f[0] != baseAutorizada {
		fallo("SHA BASE no autorizado: %s", f[0])
	}
	return f[0]
}

func validarFuentes(rutaLedger, dir, base string) {
	ruta := strings.TrimSuffix(rutaLedger, ".tsv") + "_fuentes.tsv"
	b, err := os.ReadFile(ruta)
	if err != nil {
		fallo("huellas fuente %s: %v", ruta, err)
	}
	lineas := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lineas) != len(fuentesBase) {
		fallo("huellas fuente filas=%d", len(lineas))
	}
	for i, nombre := range fuentesBase {
		campos := strings.Split(lineas[i], "\t")
		if len(campos) != 2 || campos[1] != nombre {
			fallo("huella fuente orden inválido: %q", lineas[i])
		}
		fuente, err := os.ReadFile(filepath.Join(dir, nombre))
		if err != nil {
			fallo("fuente base %s: %v", nombre, err)
		}
		h := fmt.Sprintf("%x", sha256.Sum256(fuente))
		if campos[0] != h {
			fallo("fuente base cambió %s", nombre)
		}
	}
	if h := fmt.Sprintf("%x", sha256.Sum256(b)); h != base || h != baseAutorizada {
		fallo("conjunto BASE no autorizado: %s", h)
	}
}

func validarHerramientas(rutaLedger string, actual manifiesto) {
	ruta := strings.TrimSuffix(rutaLedger, ".tsv") + "_herramientas.tsv"
	b, err := os.ReadFile(ruta)
	if err != nil {
		fallo("metadato herramientas %s: %v", ruta, err)
	}
	lineas := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lineas) != 14 || lineas[0] != "tipo\treferencia\tsha256\tversion" {
		fallo("metadato herramientas inválido: %s", ruta)
	}
	esperados := []string{"runner", "checker_fuente", "aplicador_fuente", "go_mod", "go_sum", "receta", "manifest", "checker_bin", "aplicador_bin", "gotooldir", "goroot_tree", "go", "goroot"}
	repo := raizRepositorio(filepath.Dir(rutaLedger))
	binarios, limpiar := reconstruirHerramientas(repo)
	defer limpiar()
	for i, tipo := range esperados {
		campos := strings.Split(lineas[i+1], "\t")
		if len(campos) != 4 || campos[0] != tipo {
			fallo("herramienta orden inválido: %q", lineas[i+1])
		}
		rutaActual := ""
		switch tipo {
		case "checker_bin", "aplicador_bin":
			referencia := map[string]string{"checker_bin": "STAGING/o3a_v5_ast_checker", "aplicador_bin": "STAGING/o3a_v5_aplicador"}[tipo]
			if campos[1] != referencia || campos[3] != "reconstruido" {
				fallo("binario reconstruido inválido: %v", campos)
			}
			rutaActual = binarios[tipo]
		case "gotooldir":
			referencia := "GOROOT/pkg/tool/" + runtime.GOOS + "_" + runtime.GOARCH
			if campos[1] != referencia {
				fallo("GOTOOLDIR no portable: %v", campos)
			}
			rutaActual = strings.TrimSuffix(rutaLedger, ".tsv") + "_gotool.tsv"
			validarInventarioGoTool(rutaActual, campos[2], campos[3])
		case "goroot_tree":
			if campos[1] != "GOROOT/" {
				fallo("árbol GOROOT no portable: %v", campos)
			}
			rutaActual = strings.TrimSuffix(rutaLedger, ".tsv") + "_goroot.tsv"
			validarInventarioGOROOT(rutaActual, campos[2], campos[3])
		case "go":
			if campos[1] != "GOROOT/bin/go" || campos[3] != "go version "+runtime.Version()+" "+runtime.GOOS+"/"+runtime.GOARCH {
				fallo("Go efectivo no portable: %v", campos)
			}
			rutaActual = filepath.Join(runtime.GOROOT(), "bin", "go")
			if campos[2] != goAutorizado {
				fallo("binario Go no autorizado")
			}
		case "goroot":
			if campos[1] != "GOROOT/VERSION" || !strings.HasPrefix(campos[3], runtime.Version()) {
				fallo("GOROOT efectivo no portable: %v", campos)
			}
			rutaActual = filepath.Join(runtime.GOROOT(), "VERSION")
			if campos[2] != versionGoAutorizada {
				fallo("VERSION Go no autorizada")
			}
		default:
			if filepath.IsAbs(campos[1]) || strings.Contains(campos[1], "..") {
				fallo("referencia privada/no portable: %s", campos[1])
			}
			rutaActual = filepath.Join(repo, filepath.FromSlash(campos[1]))
		}
		datos, err := os.ReadFile(rutaActual)
		if err != nil {
			fallo("herramienta %s: %v", tipo, err)
		}
		if h := fmt.Sprintf("%x", sha256.Sum256(datos)); h != campos[2] {
			fallo("herramienta cambió: %s", tipo)
		}
		if tipo == "receta" && campos[2] != recetaAutorizada {
			fallo("receta no autorizada: %s", campos[2])
		}
		if tipo == "manifest" {
			var snap manifiesto
			if err = json.Unmarshal(datos, &snap); err != nil {
				fallo("snapshot manifest: %v", err)
			}
			if snap.Estado != "EJECUCION_AUTORIZADA" || !snap.CoberturaFamiliasCompleta {
				fallo("snapshot manifest no era ejecutable: estado=%s cobertura=%v", snap.Estado, snap.CoberturaFamiliasCompleta)
			}
			if actual.Estado != "COMPLETO_EJECUTADO" || !actual.CoberturaFamiliasCompleta {
				fallo("transición manifest no cerrada: estado=%s cobertura=%v", actual.Estado, actual.CoberturaFamiliasCompleta)
			}
			if !reflect.DeepEqual(snap.Mutantes, actual.Mutantes) {
				fallo("snapshot manifest no corresponde a mutantes vigentes")
			}
		}
	}
}

func reconstruirHerramientas(repo string) (map[string]string, func()) {
	tmp, err := os.MkdirTemp("", "o3a-v5-validar-")
	if err != nil {
		fallo("staging validador: %v", err)
	}
	for _, d := range []string{"tmp", "cache", "mod"} {
		if err = os.Mkdir(filepath.Join(tmp, d), 0700); err != nil {
			fallo("staging %s: %v", d, err)
		}
	}
	goBin := filepath.Join(runtime.GOROOT(), "bin", "go")
	entorno := []string{"HOME=" + tmp, "PATH=/usr/bin:/bin", "TMPDIR=" + filepath.Join(tmp, "tmp"), "GOROOT=" + runtime.GOROOT(), "GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "CGO_ENABLED=0", "GOCACHE=" + filepath.Join(tmp, "cache"), "GOMODCACHE=" + filepath.Join(tmp, "mod")}
	salidas := map[string]string{"checker_bin": filepath.Join(tmp, "o3a_v5_ast_checker"), "aplicador_bin": filepath.Join(tmp, "o3a_v5_aplicador")}
	for tipo, modulo := range map[string]string{"checker_bin": "./tools/o3a_v5_ast", "aplicador_bin": "./tools/o3a_v5_ast/aplicar_manifest"} {
		cmd := exec.Command(goBin, "build", "-trimpath", "-o", salidas[tipo], modulo)
		cmd.Dir = repo
		cmd.Env = entorno
		if out, err := cmd.CombinedOutput(); err != nil {
			fallo("reconstrucción %s: %v: %s", tipo, err, out)
		}
	}
	return salidas, func() { _ = os.RemoveAll(tmp) }
}

func validarInventarioGoTool(ruta, hashEsperado, version string) {
	if hashEsperado != gotoolAutorizado {
		fallo("GOTOOLDIR no autorizado: %s", hashEsperado)
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		fallo("inventario GOTOOLDIR: %v", err)
	}
	lineas := strings.Split(strings.TrimSpace(string(b)), "\n")
	if version != fmt.Sprintf("archivos=%d", len(lineas)) {
		fallo("cardinalidad GOTOOLDIR: %s", version)
	}
	if h := fmt.Sprintf("%x", sha256.Sum256(b)); h != hashEsperado {
		fallo("hash inventario GOTOOLDIR")
	}
	dir := filepath.Join(runtime.GOROOT(), "pkg", "tool", runtime.GOOS+"_"+runtime.GOARCH)
	entradas, err := os.ReadDir(dir)
	if err != nil {
		fallo("GOTOOLDIR: %v", err)
	}
	var nombres []string
	for _, e := range entradas {
		if !e.IsDir() {
			nombres = append(nombres, e.Name())
		}
	}
	sort.Strings(nombres)
	if len(nombres) != len(lineas) {
		fallo("GOTOOLDIR cardinalidad actual=%d", len(nombres))
	}
	for i, n := range nombres {
		campos := strings.Split(lineas[i], "\t")
		if len(campos) != 2 || campos[1] != n {
			fallo("GOTOOLDIR orden %d", i)
		}
		d, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			fallo("GOTOOL %s: %v", n, err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(d)) != campos[0] {
			fallo("GOTOOL alterado: %s", n)
		}
	}
}

func validarInventarioGOROOT(ruta, hashEsperado, version string) {
	if hashEsperado != gorootAutorizado {
		fallo("árbol GOROOT no autorizado: %s", hashEsperado)
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		fallo("inventario GOROOT: %v", err)
	}
	lineas := strings.Split(strings.TrimSpace(string(b)), "\n")
	if version != fmt.Sprintf("archivos=%d", len(lineas)) || len(lineas) != 11536 {
		fallo("cardinalidad árbol GOROOT: %s/%d", version, len(lineas))
	}
	if h := fmt.Sprintf("%x", sha256.Sum256(b)); h != hashEsperado {
		fallo("hash inventario árbol GOROOT")
	}
	var nombres []string
	err = filepath.WalkDir(runtime.GOROOT(), func(r string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.Type().IsRegular() {
			rel, e := filepath.Rel(runtime.GOROOT(), r)
			if e != nil {
				return e
			}
			nombres = append(nombres, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		fallo("recorrido GOROOT: %v", err)
	}
	sort.Strings(nombres)
	if len(nombres) != len(lineas) {
		fallo("GOROOT actual cardinalidad=%d", len(nombres))
	}
	for i, n := range nombres {
		campos := strings.Split(lineas[i], "\t")
		if len(campos) != 2 || campos[1] != n {
			fallo("GOROOT orden %d", i)
		}
		d, e := os.ReadFile(filepath.Join(runtime.GOROOT(), filepath.FromSlash(n)))
		if e != nil {
			fallo("GOROOT %s: %v", n, e)
		}
		if fmt.Sprintf("%x", sha256.Sum256(d)) != campos[0] {
			fallo("stdlib/toolchain alterado: %s", n)
		}
	}
}

func raizRepositorio(desde string) string {
	actual, err := filepath.Abs(desde)
	if err != nil {
		fallo("raíz repositorio: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(actual, ".git")); err == nil {
			return actual
		}
		padre := filepath.Dir(actual)
		if padre == actual {
			fallo(".git no encontrado desde %s", desde)
		}
		actual = padre
	}
}

func validarLedgerPrincipal(ruta, dir string, actual manifiesto) {
	filas := leerLedger(ruta)
	if len(filas) != len(actual.Mutantes)+1 {
		fallo("ledger principal filas=%d esperado=%d", len(filas), len(actual.Mutantes)+1)
	}
	base := validarBase(filas[0])
	validarFuentes(ruta, dir, base)
	validarHerramientas(ruta, actual)
	for i, m := range actual.Mutantes {
		f := filas[i+1]
		if f[0] != base || f[1] != m.ID || f[2] != "si" || (f[5] != "muerto" && f[5] != "muerto_ast") {
			fallo("ledger principal no cerrado %s: %v", m.ID, f)
		}
		if f[3] != "autoprueba" && f[3] != "ast_tipado" {
			fallo("oráculo ledger inválido %s: %s", m.ID, f[3])
		}
	}
}

func validarLedgerSeguridad(ruta, dir string) {
	esperados := []string{"SEC-M63-LEASE-REGISTRO", "SEC-M63-LEASE-TID", "SEC-M63-OBSERVADOR-REGISTRO", "SEC-M63-OBSERVADOR-TID"}
	filas := leerLedger(ruta)
	if len(filas) != len(esperados)+1 {
		fallo("ledger SEC filas=%d esperado=%d", len(filas), len(esperados)+1)
	}
	base := validarBase(filas[0])
	validarFuentes(ruta, dir, base)
	repo := raizRepositorio(filepath.Dir(ruta))
	b, err := os.ReadFile(filepath.Join(repo, "tools/o3a_v5_ast/manifest_seguridad_pendiente_target.json"))
	if err != nil {
		fallo("manifest SEC vigente: %v", err)
	}
	var sec manifiesto
	if err = json.Unmarshal(b, &sec); err != nil {
		fallo("manifest SEC vigente JSON: %v", err)
	}
	if len(sec.Mutantes) != len(esperados) {
		fallo("manifest SEC cardinalidad=%d", len(sec.Mutantes))
	}
	for i, id := range esperados {
		if sec.Mutantes[i].ID != id {
			fallo("manifest SEC ID %d=%s", i, sec.Mutantes[i].ID)
		}
	}
	validarHerramientas(ruta, sec)
	for i, id := range esperados {
		f := filas[i+1]
		if f[0] != base || f[1] != id || f[2] != "si" || f[3] != "autoprueba" || f[4] != "65" || f[5] != "muerto" {
			fallo("ledger SEC no cerrado %s: %v", id, f)
		}
	}
}
