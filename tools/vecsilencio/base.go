package main

import (
	"bufio"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// AnalizarArbol recorre los directorios bajo raiz y analiza cada fichero Go
// de producción (sin _test.go, testdata ni código generado).
func AnalizarArbol(raiz string, directorios []string) ([]Hallazgo, error) {
	var hallazgos []Hallazgo
	for _, dir := range directorios {
		inicio := filepath.Join(raiz, dir)
		err := filepath.WalkDir(inicio, func(ruta string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				nombre := d.Name()
				if ruta != inicio && (nombre == "testdata" || nombre == "vendor" || strings.HasPrefix(nombre, ".")) {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(ruta, ".go") || strings.HasSuffix(ruta, "_test.go") {
				return nil
			}
			relativa, err := filepath.Rel(raiz, ruta)
			if err != nil {
				return err
			}
			fset := token.NewFileSet()
			fichero, err := parser.ParseFile(fset, ruta, nil, parser.ParseComments)
			if err != nil {
				return fmt.Errorf("analizar %s: %w", filepath.ToSlash(relativa), err)
			}
			if ast.IsGenerated(fichero) {
				return nil
			}
			hallazgos = append(hallazgos, AnalizarFichero(fset, fichero, filepath.ToSlash(relativa))...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return hallazgos, nil
}

// Recontar agrupa los hallazgos por huella.
func Recontar(hallazgos []Hallazgo) map[string]int {
	r := map[string]int{}
	for _, h := range hallazgos {
		r[h.Huella()]++
	}
	return r
}

func reglaDeHuella(huella string) string {
	if i := strings.LastIndexByte(huella, ':'); i >= 0 {
		return huella[i+1:]
	}
	return ""
}

// reglaBloqueante indica si la regla cuenta para la línea base. VS003 es
// solo informe.
func reglaBloqueante(regla string) bool {
	return regla != Regla5xxSinEmisor
}

// Diferencia es una huella cuyo recuento difiere de la base.
type Diferencia struct {
	Huella  string
	Base    int
	Vigente int
}

// Comparacion separa lo nuevo (falla) de lo reducible (aviso).
type Comparacion struct {
	Nuevas     []string
	Reducibles []Diferencia
}

// Comparar contrasta los recuentos vigentes con la base. VS000 nunca se
// hereda aunque aparezca en la base.
func Comparar(base, vigentes map[string]int) Comparacion {
	var c Comparacion
	for huella, n := range vigentes {
		regla := reglaDeHuella(huella)
		if !reglaBloqueante(regla) {
			continue
		}
		if regla == ReglaJustificacionInvalida || n > base[huella] {
			c.Nuevas = append(c.Nuevas, huella)
		}
	}
	for huella, n := range base {
		if vigentes[huella] < n {
			c.Reducibles = append(c.Reducibles, Diferencia{Huella: huella, Base: n, Vigente: vigentes[huella]})
		}
	}
	sort.Strings(c.Nuevas)
	sort.Slice(c.Reducibles, func(i, j int) bool { return c.Reducibles[i].Huella < c.Reducibles[j].Huella })
	return c
}

// Reducir devuelve la base limitada a lo vigente: nunca añade huellas ni
// aumenta recuentos.
func Reducir(base, vigentes map[string]int) map[string]int {
	r := map[string]int{}
	for huella, n := range base {
		if v := vigentes[huella]; v > 0 {
			r[huella] = min(n, v)
		}
	}
	return r
}

const cabeceraBase = `# Línea base de tools/vecsilencio: fallos silenciosos heredados.
# Formato: fichero:función:regla recuento. Solo puede decrecer; no editar a
# mano para añadir. Regenerar con -actualizar-base tras corregir.
`

// EscribirBase guarda las huellas bloqueantes ordenadas.
func EscribirBase(ruta string, recuentos map[string]int) error {
	huellas := make([]string, 0, len(recuentos))
	for huella, n := range recuentos {
		regla := reglaDeHuella(huella)
		if n > 0 && reglaBloqueante(regla) && regla != ReglaJustificacionInvalida {
			huellas = append(huellas, huella)
		}
	}
	sort.Strings(huellas)
	var b strings.Builder
	b.WriteString(cabeceraBase)
	for _, huella := range huellas {
		b.WriteString(huella + " " + strconv.Itoa(recuentos[huella]) + "\n")
	}
	return os.WriteFile(ruta, []byte(b.String()), 0o644)
}

// LeerBase lee la línea base. Un fichero ausente equivale a base vacía.
func LeerBase(ruta string) (map[string]int, error) {
	base := map[string]int{}
	f, err := os.Open(ruta)
	if errors.Is(err, fs.ErrNotExist) {
		return base, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	lector := bufio.NewScanner(f)
	numero := 0
	for lector.Scan() {
		numero++
		linea := strings.TrimSpace(lector.Text())
		if linea == "" || strings.HasPrefix(linea, "#") {
			continue
		}
		huella, recuento, ok := strings.Cut(linea, " ")
		n, errN := strconv.Atoi(strings.TrimSpace(recuento))
		if !ok || errN != nil || n <= 0 || strings.Count(huella, ":") < 2 {
			return nil, fmt.Errorf("línea base %s:%d mal formada", ruta, numero)
		}
		base[huella] = n
	}
	return base, lector.Err()
}
