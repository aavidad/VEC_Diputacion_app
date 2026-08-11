package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

type loteFusion struct {
	desde, hasta          int
	cat, res, fuente, man []byte
}

func valorManifest(b []byte, clave string) string {
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		p := strings.SplitN(l, "\t", 2)
		if len(p) == 2 && p[0] == clave {
			return p[1]
		}
	}
	return ""
}

func leerRango(man []byte) (int, int, error) {
	p := strings.Split(valorManifest(man, "rango"), "-")
	if len(p) != 2 {
		return 0, 0, errors.New("rango ausente")
	}
	d, e1 := strconv.Atoi(p[0])
	h, e2 := strconv.Atoi(p[1])
	if e1 != nil || e2 != nil || d < 1 || h < d {
		return 0, 0, errors.New("rango invalido")
	}
	return d, h, nil
}

func verificarSumas(dir string) error {
	b, err := os.ReadFile(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		return err
	}
	s := bufio.NewScanner(strings.NewReader(string(b)))
	vistos := map[string]bool{}
	for s.Scan() {
		p := strings.Fields(s.Text())
		if len(p) != 2 {
			return errors.New("checksum malformado")
		}
		n := filepath.Base(p[1])
		if vistos[n] {
			return errors.New("checksum duplicado")
		}
		contenido, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil || sha(contenido) != p[0] {
			return fmt.Errorf("checksum invalido %s", n)
		}
		vistos[n] = true
	}
	for _, n := range []string{"catalogo.tsv", "fuentes.tsv", "manifiesto.tsv", "resultados.tsv"} {
		if !vistos[n] {
			return fmt.Errorf("checksum ausente %s", n)
		}
	}
	return s.Err()
}

func sellosActuales(repo string) (map[string]string, error) {
	rutas := map[string]string{
		"runner_sha256": "tools/o3c_p6_mutantes/main.go", "runner_fusion_sha256": "tools/o3c_p6_mutantes/fusion.go",
		"runner_test_sha256": "tools/o3c_p6_mutantes/main_test.go", "runner_readme_sha256": "tools/o3c_p6_mutantes/README.md",
		"ast_main_sha256":        "tools/o3c_p6_ast/main.go",
		"ast_invariantes_sha256": "tools/o3c_p6_ast/invariantes.go", "ast_retirada_sha256": "tools/o3c_p6_ast/retirada.go",
		"ast_seguridad_sha256": "tools/o3c_p6_ast/seguridad.go", "ast_test_sha256": "tools/o3c_p6_ast/main_test.go",
		"ast_readme_sha256": "tools/o3c_p6_ast/README.md", "conductor_sha256": fuentes["Q"],
	}
	ejecutable, err := os.Executable()
	if err != nil {
		return nil, err
	}
	binario, err := os.ReadFile(ejecutable)
	if err != nil {
		return nil, err
	}
	r := map[string]string{"runner_bin_sha256": sha(binario)}
	for k, ruta := range rutas {
		b, err := os.ReadFile(filepath.Join(repo, ruta))
		if err != nil {
			return nil, err
		}
		r[k] = sha(b)
	}
	return r, nil
}

func fuentesEsperadas(repo string) ([]byte, error) {
	keys := make([]string, 0, len(fuentes))
	for k := range fuentes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out strings.Builder
	out.WriteString("clave\tarchivo\tsha256\n")
	for _, k := range keys {
		var b []byte
		var err error
		if k == "Q" {
			b, err = os.ReadFile(filepath.Join(repo, fuentes[k]))
		} else {
			b, err = gitMostrar(repo, base+":"+paquete+"/"+fuentes[k])
		}
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&out, "%s\t%s\t%s\n", k, fuentes[k], sha(b))
	}
	return []byte(out.String()), nil
}

func gitMostrar(repo, objeto string) ([]byte, error) {
	return ejecutarSalida(repo, "git", "-C", repo, "show", objeto)
}

func ejecutarSalida(dir string, argv ...string) ([]byte, error) {
	if len(argv) == 0 {
		return nil, errors.New("comando vacio")
	}
	c := exec.Command(argv[0], argv[1:]...)
	c.Dir = dir
	return c.Output()
}

func validarResultados(cat, res []byte) (map[string]bool, error) {
	catalogoL := strings.Split(strings.TrimSpace(string(cat)), "\n")
	resultadoL := strings.Split(strings.TrimSpace(string(res)), "\n")
	if len(catalogoL) < 2 || len(resultadoL) != len(catalogoL) {
		return nil, errors.New("cardinalidad catalogo/resultados")
	}
	esperados := map[string][2]string{}
	for _, l := range catalogoL[1:] {
		p := strings.Split(l, "\t")
		if len(p) != 7 {
			return nil, errors.New("catalogo columnas")
		}
		esperados[p[0]] = [2]string{p[1], p[2]}
	}
	vistos := map[string]bool{}
	for _, l := range resultadoL[1:] {
		p := strings.Split(l, "\t")
		if len(p) != 7 {
			return nil, errors.New("resultado columnas")
		}
		e, ok := esperados[p[0]]
		if !ok || vistos[p[0]] || e != [2]string{p[1], p[2]} {
			return nil, errors.New("resultado ID/familia/alternativa")
		}
		muertes := map[string]bool{"MUERTO-PRUEBA-CAUSAL": true, "MUERTO-AST-TIPOS-DAG-CAUSAL": true, "MUERTO-META-EVIDENCIA-NUEVA-CAUSAL": true, "MUERTO-META-CONDUCTOR-CAUSAL": true, "MUERTO-CONDUCTOR-SIN-SKIP-CAUSAL": true}
		if p[3] != "COMPILA" || !muertes[p[4]] {
			return nil, errors.New("resultado no muerto")
		}
		duracion, err := strconv.ParseUint(p[5], 10, 64)
		if err != nil || duracion == 0 || len(p[6]) != 64 {
			return nil, errors.New("duracion/SHA invalido")
		}
		if _, err := hex.DecodeString(p[6]); err != nil {
			return nil, errors.New("SHA invalido")
		}
		vistos[p[0]] = true
	}
	return vistos, nil
}

func fusionar(repo, root, out string) error {
	entradas, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	var lotes []loteFusion
	for _, e := range entradas {
		if !e.IsDir() {
			continue
		}
		d := filepath.Join(root, e.Name())
		man, err := os.ReadFile(filepath.Join(d, "manifiesto.tsv"))
		if err != nil {
			continue
		}
		if err := verificarSumas(d); err != nil {
			return fmt.Errorf("lote %s: %w", e.Name(), err)
		}
		desde, hasta, err := leerRango(man)
		if err != nil {
			return err
		}
		cat, ec := os.ReadFile(filepath.Join(d, "catalogo.tsv"))
		res, er := os.ReadFile(filepath.Join(d, "resultados.tsv"))
		f, ef := os.ReadFile(filepath.Join(d, "fuentes.tsv"))
		if ec != nil || er != nil || ef != nil {
			return errors.New("lote incompleto")
		}
		if hasta-desde+1 != len(strings.Split(strings.TrimSpace(string(cat)), "\n"))-1 {
			return errors.New("rango/cardinalidad")
		}
		lotes = append(lotes, loteFusion{desde, hasta, cat, res, f, man})
	}
	sort.Slice(lotes, func(i, j int) bool { return lotes[i].desde < lotes[j].desde })
	if len(lotes) == 0 {
		return errors.New("sin lotes")
	}
	actuales, err := sellosActuales(repo)
	if err != nil {
		return err
	}
	fuenteEsperada, err := fuentesEsperadas(repo)
	if err != nil {
		return err
	}
	primeraFuente := string(fuenteEsperada)
	siguiente := 1
	firma := ""
	ids, familias := map[string]bool{}, map[string]bool{}
	var cat, res strings.Builder
	cat.WriteString("id\tfamilia\talternativa\tarchivo\tantes_hex\tdespues_hex\toraculo\n")
	res.WriteString("id\tfamilia\talternativa\tcompilacion\tmuerte_causal\tduracion_ns\tsalida_sha256\n")
	for _, l := range lotes {
		if l.desde != siguiente {
			return errors.New("rangos con hueco/solape")
		}
		siguiente = l.hasta + 1
		if string(l.fuente) != primeraFuente {
			return errors.New("fuentes distintas")
		}
		if valorManifest(l.man, "base") != base || valorManifest(l.man, "toolchain") != runtime.Version() || valorManifest(l.man, "catalogo_total") != strconv.Itoa(len(catalogo())) {
			return errors.New("base/toolchain/catalogo")
		}
		if valorManifest(l.man, "modo") != "mutantes" {
			return errors.New("lote preflight no fusionable")
		}
		var f strings.Builder
		for _, k := range []string{"runner_sha256", "runner_fusion_sha256", "runner_test_sha256", "runner_readme_sha256", "runner_bin_sha256", "ast_main_sha256", "ast_invariantes_sha256", "ast_retirada_sha256", "ast_seguridad_sha256", "ast_test_sha256", "ast_readme_sha256", "conductor_sha256"} {
			if valorManifest(l.man, k) != actuales[k] {
				return fmt.Errorf("sello actual invalido %s", k)
			}
			f.WriteString(actuales[k])
		}
		if firma == "" {
			firma = f.String()
		} else if firma != f.String() {
			return errors.New("sellos distintos")
		}
		vistosL, err := validarResultados(l.cat, l.res)
		if err != nil {
			return err
		}
		for _, line := range strings.Split(strings.TrimSpace(string(l.cat)), "\n")[1:] {
			p := strings.Split(line, "\t")
			if ids[p[0]] || !vistosL[p[0]] {
				return errors.New("ID duplicado/sin resultado")
			}
			ids[p[0]] = true
			familias[p[1]] = true
			cat.WriteString(line + "\n")
		}
		for _, line := range strings.Split(strings.TrimSpace(string(l.res)), "\n")[1:] {
			res.WriteString(line + "\n")
		}
	}
	if siguiente != len(catalogo())+1 || len(ids) != len(catalogo()) || len(familias) != 24 {
		return errors.New("cobertura incompleta")
	}
	for _, x := range catalogo() {
		if !ids[x.id] {
			return fmt.Errorf("falta %s", x.id)
		}
	}
	if err = os.MkdirAll(out, 0755); err != nil {
		return err
	}
	archivos := map[string][]byte{"catalogo.tsv": []byte(cat.String()), "resultados.tsv": []byte(res.String()), "fuentes.tsv": lotes[0].fuente}
	for n, b := range archivos {
		if err = os.WriteFile(filepath.Join(out, n), b, 0644); err != nil {
			return err
		}
	}
	man := fmt.Sprintf("base\t%s\ntoolchain\t%s\ncatalogo_total\t%d\nmutantes\t%d/%d compilables-y-muertos\nfamilias\t24/24\nrunner_sha256\t%s\nrunner_fusion_sha256\t%s\nrunner_test_sha256\t%s\nrunner_readme_sha256\t%s\nrunner_bin_sha256\t%s\nast_main_sha256\t%s\nast_invariantes_sha256\t%s\nast_retirada_sha256\t%s\nast_seguridad_sha256\t%s\nast_test_sha256\t%s\nast_readme_sha256\t%s\nconductor_sha256\t%s\nfuentes_sha256\t%s\nresiduos\tcero\n", base, runtime.Version(), len(catalogo()), len(ids), len(catalogo()), actuales["runner_sha256"], actuales["runner_fusion_sha256"], actuales["runner_test_sha256"], actuales["runner_readme_sha256"], actuales["runner_bin_sha256"], actuales["ast_main_sha256"], actuales["ast_invariantes_sha256"], actuales["ast_retirada_sha256"], actuales["ast_seguridad_sha256"], actuales["ast_test_sha256"], actuales["ast_readme_sha256"], actuales["conductor_sha256"], sha(fuenteEsperada))
	if err = os.WriteFile(filepath.Join(out, "manifiesto.tsv"), []byte(man), 0644); err != nil {
		return err
	}
	var sums strings.Builder
	for _, n := range []string{"catalogo.tsv", "fuentes.tsv", "manifiesto.tsv", "resultados.tsv"} {
		b, _ := os.ReadFile(filepath.Join(out, n))
		fmt.Fprintf(&sums, "%s  tools/o3c_p6_mutantes/evidencia/%s\n", sha(b), n)
	}
	return os.WriteFile(filepath.Join(out, "SHA256SUMS"), []byte(sums.String()), 0644)
}
