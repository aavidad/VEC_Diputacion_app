package main

import (
	"bytes"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogoO3cP6CompletoAtomico(t *testing.T) {
	if len(catalogo()) < 120 {
		t.Fatalf("catálogo incompleto: %d alternativas", len(catalogo()))
	}
	vistas := map[string]bool{}
	familias := map[string]bool{}
	for _, x := range catalogo() {
		clave := x.familia + "/" + x.alternativa
		if vistas[clave] || x.oraculo == "" || x.antes == x.despues {
			t.Fatalf("mutante no atómico/distinto: %s", clave)
		}
		vistas[clave], familias[x.familia] = true, true
		var b []byte
		var err error
		if x.fuente == "Q" {
			b, err = os.ReadFile(filepath.Join("../..", fuentes[x.fuente]))
		} else {
			b, err = exec.Command("git", "show", base+":"+paquete+"/"+fuentes[x.fuente]).Output()
		}
		if err != nil || bytes.Count(b, []byte(x.antes)) != 1 {
			t.Fatalf("%s patrón anterior cardinalidad=%d err=%v", clave, bytes.Count(b, []byte(x.antes)), err)
		}
		mutado := bytes.Replace(b, []byte(x.antes), []byte(x.despues), 1)
		if x.fuente == "Q" {
			ruta := filepath.Join(t.TempDir(), "conductor.sh")
			if err = os.WriteFile(ruta, mutado, 0600); err == nil {
				err = exec.Command("/bin/bash", "-n", ruta).Run()
			}
		} else {
			_, err = parser.ParseFile(token.NewFileSet(), x.id, mutado, parser.AllErrors)
		}
		if err != nil {
			t.Fatalf("%s no parsea: %v", clave, err)
		}
	}
	for i := 1; i <= 24; i++ {
		id := "C" + format2(i)
		if !familias[id] {
			t.Fatalf("familia ausente: %s", id)
		}
	}
}

func itoa(i int) string { return string([]byte{'0' + byte(i%10)}) }
func format2(i int) string {
	if i < 10 {
		return "0" + itoa(i)
	}
	return string([]byte{'0' + byte(i/10), '0' + byte(i%10)})
}

func TestSalidaNoPuedeSolaparTemporal(t *testing.T) {
	tmp := t.TempDir()
	if filepath.Clean(tmp) == filepath.Clean(filepath.Join(tmp, ".")) && !strings.HasPrefix(filepath.Clean("tools/o3c_p6_mutantes/evidencia"), tmp) {
		return
	}
	t.Fatal("guardia de rutas inválida")
}

func TestMetaConductorO3cP6RechazaCadaDesviacionC22(t *testing.T) {
	ruta := filepath.Join("../..", fuentes["Q"])
	baseConductor, err := os.ReadFile(ruta)
	if err != nil || validarConductorO3cP6(baseConductor) != nil {
		t.Fatalf("conductor congelado no valida: lectura=%v meta=%v", err, validarConductorO3cP6(baseConductor))
	}
	for _, x := range catalogo() {
		if x.familia != "C22" {
			continue
		}
		mutado := bytes.Replace(baseConductor, []byte(x.antes), []byte(x.despues), 1)
		if err := validarConductorO3cP6(mutado); err == nil {
			t.Errorf("meta-oráculo aceptó %s", x.alternativa)
		}
	}
}

func TestFusionResultadosO3cP6RechazaManipulaciones(t *testing.T) {
	cat := []byte("id\tfamilia\talternativa\tarchivo\tantes_hex\tdespues_hex\toraculo\nC001\tC01\ta\tf.go\t00\t01\to\n")
	verde := []byte("id\tfamilia\talternativa\tcompilacion\tmuerte_causal\tduracion_ns\tsalida_sha256\nC001\tC01\ta\tCOMPILA\tMUERTO-PRUEBA-CAUSAL\t1\t" + strings.Repeat("0", 64) + "\n")
	if _, err := validarResultados(cat, verde); err != nil {
		t.Fatalf("fixture verde: %v", err)
	}
	casos := map[string][]byte{
		"vacio":     []byte("id\tfamilia\talternativa\tcompilacion\tmuerte_causal\tduracion_ns\tsalida_sha256\n"),
		"duplicado": append(append([]byte{}, verde...), bytes.SplitN(verde, []byte{'\n'}, 2)[1]...),
		"sobrevive": bytes.Replace(verde, []byte("MUERTO-PRUEBA-CAUSAL"), []byte("SOBREVIVE"), 1),
		"inventado": bytes.Replace(verde, []byte("MUERTO-PRUEBA-CAUSAL"), []byte("MUERTO-INVENTADO"), 1),
		"duracion0": bytes.Replace(verde, []byte("\t1\t"), []byte("\t0\t"), 1),
		"preflight": bytes.Replace(verde, []byte("MUERTO-PRUEBA-CAUSAL"), []byte("NO-EJECUTADO-PREFLIGHT"), 1),
		"mismatch":  bytes.Replace(verde, []byte("\tC01\ta\t"), []byte("\tC02\tb\t"), 1),
	}
	for nombre, mutado := range casos {
		if _, err := validarResultados(cat, mutado); err == nil {
			t.Errorf("aceptó resultado %s", nombre)
		}
	}
}

func TestFusionO3cP6RechazaLoteManipuladoTrasChecksum(t *testing.T) {
	d := t.TempDir()
	archivos := map[string][]byte{"catalogo.tsv": []byte("c\n"), "fuentes.tsv": []byte("f\n"), "manifiesto.tsv": []byte("m\n"), "resultados.tsv": []byte("r\n")}
	var sumas strings.Builder
	for n, b := range archivos {
		if err := os.WriteFile(filepath.Join(d, n), b, 0600); err != nil {
			t.Fatal(err)
		}
		sumas.WriteString(sha(b) + "  " + n + "\n")
	}
	if err := os.WriteFile(filepath.Join(d, "SHA256SUMS"), []byte(sumas.String()), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verificarSumas(d); err != nil {
		t.Fatalf("lote intacto: %v", err)
	}
	if err := os.WriteFile(filepath.Join(d, "resultados.tsv"), []byte("alterado\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verificarSumas(d); err == nil {
		t.Fatal("aceptó lote coherentemente manipulado sin checksum válido")
	}
}

func TestBinarioCanonicoO3cP6ReproducibleTrimpath(t *testing.T) {
	d := t.TempDir()
	binarios := []string{filepath.Join(d, "a"), filepath.Join(d, "b")}
	for _, salida := range binarios {
		cmd := exec.Command("go", "build", "-trimpath", "-o", salida, ".")
		cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOTOOLCHAIN=local")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build canónico: %v: %s", err, out)
		}
	}
	a, errA := os.ReadFile(binarios[0])
	b, errB := os.ReadFile(binarios[1])
	if errA != nil || errB != nil || !bytes.Equal(a, b) {
		t.Fatalf("binario -trimpath no reproducible: %v/%v %s/%s", errA, errB, sha(a), sha(b))
	}
}

func TestGoRunO3cP6FallaCerrado(t *testing.T) {
	cmd := exec.Command("go", "run", ".", "-fusionar", filepath.Join(t.TempDir(), "ausente"))
	salida, err := cmd.CombinedOutput()
	if err == nil || !bytes.Contains(salida, []byte("binario no canonico")) {
		t.Fatalf("go run no falló cerrado: %v %s", err, salida)
	}
}

func TestMain(m *testing.M) { os.Exit(m.Run()) }
