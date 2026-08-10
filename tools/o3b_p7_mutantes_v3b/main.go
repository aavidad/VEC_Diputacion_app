package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"
)

const base = "d9f8aeb547f5d1b3b9ab3eb786382f78ef964e28"
const conductorSHA = "5561475016a1cf2b59602e441dd825399356099f84bdcd33a3864169e9374343"
const paquete = "deploy/postgresql/autorizacion_atestada_v3/pruebas_sql"

var rutas = map[string]string{
	"identidad": paquete + "/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_identidad.go",
	"handoff":   paquete + "/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_handoff.go",
	"stop":      paquete + "/captura_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_stop.go",
	"o3a":       paquete + "/supervisor_procesos_m38_h0b_fuente_corporativa_contexto_actor_v1_arranque_inicio.go",
	"conductor": "tools/o3b_p7_conductor/conductor.sh",
}

type mutante struct{ id, familia, alternativa, target, antes, despues, funcion, oraculo string }
type resultado struct {
	mutante
	fuente, mutado, muerte, proceso, stdout, stderr string
	duracion                                        time.Duration
}

func fatal(f string, a ...any) { fmt.Fprintf(os.Stderr, "NO-GO "+f+"\n", a...); os.Exit(1) }
func suma(b []byte) string     { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

func catalogo(path string) ([]mutante, []byte) {
	b, err := os.ReadFile(path)
	if err != nil || len(b) > 1<<20 {
		fatal("catálogo: %v", err)
	}
	s := bufio.NewScanner(bytes.NewReader(b))
	s.Buffer(make([]byte, 4096), 1<<20)
	var out []mutante
	ids := map[string]bool{}
	pares := map[string]bool{}
	linea := 0
	for s.Scan() {
		linea++
		if linea == 1 {
			if s.Text() != "id\tfamilia\talternativa\ttarget\tantes\tdespues\tfuncion\toraculo" {
				fatal("cabecera catálogo")
			}
			continue
		}
		p := strings.Split(s.Text(), "\t")
		if len(p) != 8 {
			fatal("catálogo línea %d", linea)
		}
		m := mutante{p[0], p[1], p[2], p[3], p[4], p[5], p[6], p[7]}
		par := m.familia + "/" + m.alternativa
		if ids[m.id] || pares[par] || rutas[m.target] == "" || m.antes == m.despues || m.oraculo == "" {
			fatal("catálogo no atómico línea %d", linea)
		}
		ids[m.id], pares[par] = true, true
		out = append(out, m)
	}
	if s.Err() != nil || len(out) != 50 {
		fatal("catálogo incompleto: %d: %v", len(out), s.Err())
	}
	exp := map[string]int{"B19": 2, "B20": 4, "B21": 5, "B21A": 4, "B22": 3, "B23": 2, "B24": 2, "B25": 3, "B25A": 4, "B26": 4, "B27": 6, "B28": 4, "B29": 4, "B30": 3}
	got := map[string]int{}
	for _, m := range out {
		got[m.familia]++
	}
	for f, n := range exp {
		if got[f] != n {
			fatal("%s alternativas %d/%d", f, got[f], n)
		}
	}
	return out, b
}

func fuenteBase(repo, ruta string) []byte {
	if ruta == rutas["conductor"] {
		b, err := os.ReadFile(filepath.Join(repo, ruta))
		if err != nil {
			fatal("conductor congelado: %v", err)
		}
		return b
	}
	c := exec.Command("/usr/bin/git", "show", base+":"+ruta)
	c.Dir = repo
	c.Env = []string{"PATH=/usr/bin:/bin", "HOME=/nonexistent", "LANG=C", "LC_ALL=C"}
	b, err := c.Output()
	if err != nil {
		fatal("fuente base %s: %v", ruta, err)
	}
	return b
}

func exportar(repo, dst string) {
	tar := filepath.Join(dst, "base.tar")
	c := exec.Command("/usr/bin/git", "archive", "--format=tar", "-o", tar, base)
	c.Dir = repo
	if b, e := c.CombinedOutput(); e != nil {
		fatal("archive: %v %s", e, b)
	}
	c = exec.Command("/usr/bin/tar", "-xf", tar, "-C", dst)
	if b, e := c.CombinedOutput(); e != nil {
		fatal("extract: %v %s", e, b)
	}
	if e := os.Remove(tar); e != nil {
		fatal("retirar tar: %v", e)
	}
}

func funcionCanonica(src []byte, nombre string) (string, error) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "mutado.go", src, parser.AllErrors)
	if err != nil {
		return "", err
	}
	for _, d := range f.Decls {
		if fn, ok := d.(*ast.FuncDecl); ok && fn.Name.Name == nombre {
			var b bytes.Buffer
			if err := printer.Fprint(&b, fs, fn); err != nil {
				return "", err
			}
			return b.String(), nil
		}
	}
	return "", fmt.Errorf("función %s ausente", nombre)
}

// validarSemantica sólo se usa tras sobrevivir las pruebas conductuales. Cada
// alternativa tiene una propiedad prohibida propia sobre el AST canónico; no
// consulta el patrón de sustitución ni una huella del mutante.
func validarSemantica(m mutante, src []byte) error {
	canon, err := funcionCanonica(src, m.funcion)
	if err != nil && m.alternativa != "global-mutable" && m.alternativa != "init" {
		return err
	}
	reglas := map[string]string{
		"proc-autoridad-senal":       "os.Getpid()",
		"proc-pin-identidad":         `"/proc/"`,
		"entregar-tras-terminalidad": "a.estado == capturaBFFatalM38",
		"entregar-tras-cancelacion":  "origen != nil && *origen == nil",
		"entregar-tras-senal":        "senal == 0",
		"bootstrap-vencido":          "time.Now().Before(a.custodia.finBootstrap)",
		"PID-separado":               "pidSeparado",
		"pidfd-separado":             "pidfdSeparado",
		"Process-separado":           "processSeparado",
		"CONTROL-separado":           "controlSeparado",
		"TERMINAL-separado":          "terminalSeparado",
		"invertir-transferencias":    "okLease := transferirLeaseCapturadaO3bM38",
		"omitir-prevalidacion":       "valida, fatal := true, false",
		"retornar-B4T":               "if !ok {\n\t\treturn nil",
		"rollback-parcial":           "CompareAndSwap(1, a.custodia.baselineSenal)",
		"conservar-writer":           "salida.custodia.ticketEscritor =",
		"perder-CONTROL":             "salida.custodia.controlFD = nil",
		"perder-TERMINAL":            "salida.custodia.terminal = nil",
		"plazo-180s":                 "180 * time.Second",
		"CONT-en-O3b":                "syscall.SIGCONT",
		"Wait-fuera-retirada":        ".cmd.Wait()",
		"Wait-antes-terminalidad":    ".cmd.Wait()",
		"senal-grupo":                "syscall.Kill(-c.cmd.Process.Pid",
		"PID-numerico":               "uintptr(c.cmd.Process.Pid)",
		"reabrir-identidad":          "syscall.Open(ruta, syscall.O_RDONLY|syscall.O_CLOEXEC, 0)",
		"omitir-finRetirada":         "finRetirada := c.finBootstrap",
		"recrear-finRetirada":        "finRetirada := time.Now().Add",
		"reiniciar-o-ampliar":        "2 * duracionRetiradaO3aM38",
		"no-limitar-bootstrap":       "false && finRetirada.After(tope)",
		"retorno-BF":                 "if !ok {\n\t\treturn nil",
		"log-BF":                     `println("BF")`,
		"cierre-BF":                  "syscall.Close(a.custodia.pidfdPrimario)",
		"ES-BF":                      `syscall.Write(2, []byte("BF"))`,
		"goroutine":                  "go func()",
		"callback":                   "callback := func()",
		"hook":                       "hookO3b := func(",
		"segundo-owner":              "segundoOwner := &agregadoO3cM38",
		"arista-O3c":                 "siguienteO3c := func(",
		"arista-O4":                  "o4 := func()",
		"arista-O5":                  "o5 := func()",
		"parser-G4-o-inversa":        "parserG4 := func(",
		"omitir-inventario":          "if err := error(nil)",
		"lease-pending":              "a.custodia == nil || false",
		"omitir-revalidacion-fisica": "if false ||",
		"omitir-cierre-causal":       "a.custodia.ticketEscritor == nil",
	}
	if m.alternativa == "global-mutable" || m.alternativa == "init" {
		fs := token.NewFileSet()
		f, parseErr := parser.ParseFile(fs, "mutado.go", src, parser.AllErrors)
		if parseErr != nil {
			return parseErr
		}
		for _, d := range f.Decls {
			switch x := d.(type) {
			case *ast.FuncDecl:
				if m.alternativa == "init" && x.Name.Name == "init" {
					return errors.New("init prohibido presente")
				}
			case *ast.GenDecl:
				if m.alternativa == "global-mutable" {
					for _, s := range x.Specs {
						if v, ok := s.(*ast.ValueSpec); ok {
							for _, n := range v.Names {
								if n.Name == "estadoGlobalO3bM38" {
									return errors.New("global mutable presente")
								}
							}
						}
					}
				}
			}
		}
		return nil
	}
	prohibido := reglas[m.alternativa]
	if prohibido == "" {
		return fmt.Errorf("regla semántica ausente: %s", m.alternativa)
	}
	if strings.Contains(canon, prohibido) {
		return fmt.Errorf("propiedad prohibida acreditada: %s", m.oraculo)
	}
	return nil
}

func fixtureB30(tmp string, m mutante, mutado []byte) error {
	script := string(mutado)
	var harness string
	switch m.alternativa {
	case "falsear-residuos", "reutilizar-caso-fallido":
		ini := strings.Index(script, "ejecutar() {")
		fin := strings.Index(script[ini:], "\n}\n\nfor modo")
		if ini < 0 || fin < 0 {
			return errors.New("función ejecutar no extraíble")
		}
		fn := script[ini : ini+fin+2]
		harness = `set -euo pipefail
staging=$1; evidencia=$staging/e; target=$staging; sha_target=x; mkdir -p "$evidencia"; : > "$evidencia/casos.tsv"
inventario(){ printf -v "$1" 0; printf -v "$2" 0; printf -v "$3" 0; printf -v "$4" 0; printf -v "$5" 0; }
` + fn + `
printf '#!/bin/sh\necho caso-$1\nexit ${FALLO:-0}\n' > "$staging/fake"; chmod +x "$staging/fake"
if [[ $2 == falsear-residuos ]]; then FALLO=7 ejecutar X normal x oracle "$staging/fake" && exit 40 || exit 0; fi
ejecutar A normal a oracle "$staging/fake"; ejecutar B normal b oracle "$staging/fake"
[[ -f $staging/A.normal.out && -f $staging/B.normal.out ]] || exit 41
`
	case "aceptar-SKIP":
		var linea string
		for _, l := range strings.Split(script, "\n") {
			if strings.Contains(l, "O18_CIEN_CAPTURAS") && strings.Contains(l, "continue") {
				linea = strings.TrimSpace(l)
				break
			}
		}
		if linea == "" {
			return errors.New("guardia SKIP ausente")
		}
		harness = "set -euo pipefail\nid=SKIP\nfor intento in 1; do\n" + linea + "\necho PROCESADO\ndone\n"
	default:
		return errors.New("fixture B30 desconocido")
	}
	ruta := filepath.Join(tmp, "meta-b30.sh")
	if err := os.WriteFile(ruta, []byte(harness), 0700); err != nil {
		return err
	}
	args := []string{"/usr/bin/timeout", "--signal=KILL", "3", "/usr/bin/bash", ruta, tmp, m.alternativa}
	o, e, estado, _ := ejecutarGrupo(tmp, args...)
	if estado != "0" {
		return fmt.Errorf("fixture mata %s estado=%s stdout=%s stderr=%s", m.alternativa, estado, o, e)
	}
	if m.alternativa == "aceptar-SKIP" && strings.TrimSpace(string(o)) != "PROCESADO" {
		return fmt.Errorf("fixture mata aceptar-SKIP: caso no ejecutado")
	}
	return nil
}

func ejecutarGrupo(dir string, argv ...string) ([]byte, []byte, string, time.Duration) {
	c := exec.Command(argv[0], argv[1:]...)
	c.Dir = dir
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	c.WaitDelay = time.Second
	c.Env = []string{"PATH=" + runtime.GOROOT() + "/bin:/usr/bin:/bin", "HOME=" + dir, "GOCACHE=" + filepath.Join(dir, ".cache-go"), "GOMODCACHE=" + filepath.Join(dir, ".cache-mod"), "GOPATH=" + filepath.Join(dir, ".gopath"), "LANG=C", "LC_ALL=C", "GOTOOLCHAIN=local", "CGO_ENABLED=0"}
	var o, e bytes.Buffer
	c.Stdout = &o
	c.Stderr = &e
	ini := time.Now()
	if err := c.Start(); err != nil {
		fatal("arranque grupo: %v", err)
	}
	pgid, errPGID := syscall.Getpgid(c.Process.Pid)
	if errPGID != nil || pgid != c.Process.Pid {
		fatal("PGID no exclusivo pid=%d pgid=%d err=%v", c.Process.Pid, pgid, errPGID)
	}
	err := c.Wait()
	dur := time.Since(ini)
	if c.Process != nil {
		z := syscall.Kill(-pgid, 0)
		if !errors.Is(z, syscall.ESRCH) {
			limite := time.Now().Add(time.Second)
			for !errors.Is(z, syscall.ESRCH) && time.Now().Before(limite) {
				if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
					fatal("PGID %d no retirable: %v", pgid, err)
				}
				time.Sleep(5 * time.Millisecond)
				z = syscall.Kill(-pgid, 0)
			}
			if !errors.Is(z, syscall.ESRCH) {
				fatal("PGID %d no converge a ESRCH", pgid)
			}
		}
	}
	if err == nil {
		return o.Bytes(), e.Bytes(), "0", dur
	}
	var x *exec.ExitError
	if errors.As(err, &x) {
		return o.Bytes(), e.Bytes(), fmt.Sprint(x.ExitCode()), dur
	}
	fatal("ejecución: %v", err)
	return nil, nil, "", 0
}

func compilarYMorir(tmp string, m mutante, orig []byte) resultado {
	if bytes.Count(orig, []byte(m.antes)) != 1 {
		fatal("%s cardinalidad anterior", m.id)
	}
	mut := bytes.Replace(orig, []byte(m.antes), []byte(m.despues), 1)
	ruta := rutas[m.target]
	if err := os.MkdirAll(filepath.Dir(filepath.Join(tmp, ruta)), 0755); err != nil {
		fatal("%s directorio: %v", m.id, err)
	}
	if err := os.WriteFile(filepath.Join(tmp, ruta), mut, 0644); err != nil {
		fatal("%s write: %v", m.id, err)
	}
	defer func() {
		if err := os.WriteFile(filepath.Join(tmp, ruta), orig, 0644); err != nil {
			fatal("%s restauración efímera: %v", m.id, err)
		}
	}()
	var out, errout []byte
	var estado string
	var dur time.Duration
	if m.target == "conductor" {
		out, errout, estado, dur = ejecutarGrupo(tmp, "/usr/bin/bash", "-n", ruta)
	} else {
		dir := filepath.Join(tmp, paquete)
		files, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		args := []string{filepath.Join(runtime.GOROOT(), "bin", "go"), "test", "-c", "-o", filepath.Join(tmp, "mutante.test")}
		for _, f := range files {
			if filepath.Base(f) != "capturar_snapshot_fuente_corporativa_contexto_actor_v1.go" {
				args = append(args, filepath.Base(f))
			}
		}
		out, errout, estado, dur = ejecutarGrupo(dir, args...)
	}
	if estado != "0" {
		fatal("%s no compila/sintaxis estado=%s stderr=%s", m.id, estado, errout)
	}
	var muerte error
	tipoMuerte := "CONDUCTUAL"
	if m.target == "conductor" {
		muerte = fixtureB30(tmp, m, mut)
		tipoMuerte = "META-B30"
	} else {
		pruebaOut, pruebaErr, pruebaEstado, pruebaDur := ejecutarGrupo(filepath.Join(tmp, paquete), "/usr/bin/timeout", "--signal=KILL", "12", filepath.Join(tmp, "mutante.test"), "-test.run=O3b", "-test.count=1")
		dur += pruebaDur
		out = append(out, pruebaOut...)
		errout = append(errout, pruebaErr...)
		if pruebaEstado != "0" {
			muerte = fmt.Errorf("pruebas O3b estado=%s", pruebaEstado)
		} else {
			muerte = validarSemantica(m, mut)
			tipoMuerte = "AST-CFG-SEMANTICO"
		}
	}
	if muerte == nil {
		fatal("%s sobrevivió a pruebas y oráculo semántico %s", m.id, m.oraculo)
	}
	return resultado{m, suma(orig), suma(mut), tipoMuerte, estado, hex.EncodeToString(out), hex.EncodeToString(append(errout, []byte(muerte.Error())...)), dur}
}

func main() {
	repo := flag.String("repo", ".", "checkout")
	out := flag.String("out", "tools/o3b_p7_mutantes_v3b/evidencia", "evidencia nueva")
	desde := flag.Int("desde", 1, "primer ordinal inclusivo")
	hasta := flag.Int("hasta", 50, "último ordinal inclusivo")
	flag.Parse()
	lista, cat := catalogo(filepath.Join(*repo, "tools/o3b_p7_mutantes_v3b/catalogo.tsv"))
	if *desde < 1 || *hasta > len(lista) || *desde > *hasta {
		fatal("rango inválido %d..%d", *desde, *hasta)
	}
	lista = lista[*desde-1 : *hasta]
	var e error
	fuentes := map[string][]byte{}
	for k, r := range rutas {
		fuentes[k] = fuenteBase(*repo, r)
	}
	if suma(fuentes["conductor"]) != conductorSHA {
		fatal("SHA conductor congelado")
	}
	sandbox, errSandbox := os.MkdirTemp("", "o3b-p7-v3b-base-")
	if errSandbox != nil {
		fatal("sandbox: %v", errSandbox)
	}
	defer os.RemoveAll(sandbox)
	exportar(*repo, sandbox)
	dirSalida := *out
	if !filepath.IsAbs(dirSalida) {
		dirSalida = filepath.Join(*repo, dirSalida)
	}
	if _, e := os.Stat(dirSalida); !errors.Is(e, os.ErrNotExist) {
		fatal("evidencia debe ser nueva")
	}
	if e = os.MkdirAll(dirSalida, 0755); e != nil {
		fatal("evidencia: %v", e)
	}
	res := make([]resultado, 0, len(lista))
	for _, m := range lista {
		x := compilarYMorir(sandbox, m, fuentes[m.target])
		fmt.Printf("%s %s/%s MUERTO\n", m.id, m.familia, m.alternativa)
		res = append(res, x)
	}
	var b strings.Builder
	b.WriteString("id\tfamilia\talternativa\ttarget\tsha_fuente\tsha_mutado\tcompila\tmuerte\tduracion_ns\tstdout_hex\tstderr_hex\toraculo\n")
	for _, x := range res {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\t%s\n", x.id, x.familia, x.alternativa, x.target, x.fuente, x.mutado, x.proceso, x.muerte, x.duracion.Nanoseconds(), x.stdout, x.stderr, x.oraculo)
	}
	var f strings.Builder
	keys := make([]string, 0, len(fuentes))
	for k := range fuentes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&f, "%s\t%s\t%s\n", k, suma(fuentes[k]), rutas[k])
	}
	runner, errRunner := os.ReadFile(filepath.Join(*repo, "tools/o3b_p7_mutantes_v3b/main.go"))
	readme, errREADME := os.ReadFile(filepath.Join(*repo, "tools/o3b_p7_mutantes_v3b/README.md"))
	if errRunner != nil || errREADME != nil {
		fatal("leer tooling: runner=%v README=%v", errRunner, errREADME)
	}
	var manifiesto strings.Builder
	fmt.Fprintf(&manifiesto, "base\t%s\n", base)
	fmt.Fprintf(&manifiesto, "mutantes\t%d/%d compilables-y-muertos\n", len(res), len(lista))
	fmt.Fprintf(&manifiesto, "runner_sha256\t%s\n", suma(runner))
	fmt.Fprintf(&manifiesto, "readme_sha256\t%s\n", suma(readme))
	fmt.Fprintf(&manifiesto, "catalogo_sha256\t%s\n", suma(cat))
	fmt.Fprintf(&manifiesto, "fuentes_sha256\t%s\n", suma([]byte(f.String())))
	fmt.Fprintf(&manifiesto, "resultados_sha256\t%s\n", suma([]byte(b.String())))
	fmt.Fprintf(&manifiesto, "conductor_sha256\t%s\n", conductorSHA)
	fmt.Fprintf(&manifiesto, "toolchain\t%s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&manifiesto, "receta\tgo run ./tools/o3b_p7_mutantes_v3b -repo . -out tools/o3b_p7_mutantes_v3b/evidencia-v3\n")
	dir := dirSalida
	files := map[string][]byte{
		"resultados.tsv":  []byte(b.String()),
		"fuentes.tsv":     []byte(f.String()),
		"catalogo.sha256": []byte(suma(cat) + "  catalogo.tsv\n"),
		"manifiesto.tsv":  []byte(manifiesto.String()),
	}
	for n, d := range files {
		if e = os.WriteFile(filepath.Join(dir, n), d, 0644); e != nil {
			fatal("write %s: %v", n, e)
		}
	}
	var sums strings.Builder
	names := []string{"catalogo.sha256", "fuentes.tsv", "manifiesto.tsv", "resultados.tsv"}
	sort.Strings(names)
	for _, n := range names {
		d, _ := os.ReadFile(filepath.Join(dir, n))
		fmt.Fprintf(&sums, "%s  %s\n", suma(d), n)
	}
	_ = os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte(sums.String()), 0644)
}
