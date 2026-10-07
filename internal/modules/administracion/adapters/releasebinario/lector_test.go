package releasebinario

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

func TestInspeccionDeBinariosGoReales(t *testing.T) {
	raiz := t.TempDir()
	repo := filepath.Join(raiz, "fuentes")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	for nombre, contenido := range map[string]string{"go.mod": "module ejemplo.invalid/material-sintetico\n\ngo 1.25.12\n", "main.go": "package main\nvar valor string\nfunc main() {}\n"} {
		if err := os.WriteFile(filepath.Join(repo, nombre), []byte(contenido), 0600); err != nil {
			t.Fatal(err)
		}
	}
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	entorno := []string{"PATH=" + filepath.Dir(git) + ":" + filepath.Join(runtime.GOROOT(), "bin") + ":/usr/bin:/bin", "GOROOT=" + runtime.GOROOT(), "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "GOWORK=off", "CGO_ENABLED=0", "GOCACHE=/dev/shm/go-build", "TMPDIR=" + raiz, "GIT_AUTHOR_DATE=2026-10-01T12:00:00Z", "GIT_COMMITTER_DATE=2026-10-01T12:00:00Z"}
	ejecutar := func(programa string, argumentos ...string) string {
		t.Helper()
		ctx, cancelar := context.WithTimeout(context.Background(), time.Minute)
		defer cancelar()
		cmd := exec.CommandContext(ctx, programa, argumentos...)
		cmd.Dir, cmd.Env = repo, entorno
		salida, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("fixture fallo: %v (%s)", err, salida)
		}
		return strings.TrimSpace(string(salida))
	}
	ejecutar(git, "init", "--quiet")
	ejecutar(git, "add", "go.mod", "main.go")
	ejecutar(git, "-c", "user.name=Ana Martin", "-c", "user.email=ana@example.invalid", "commit", "--quiet", "-m", "fixture sintetica")
	revision := ejecutar(git, "rev-parse", "HEAD")
	goBin := filepath.Join(runtime.GOROOT(), "bin", "go")
	construir := func(nombre, vcs string) {
		t.Helper()
		ejecutar(goBin, "build", "-p", "8", "-buildvcs="+vcs, "-ldflags=-X main.valor=token-sintetico-no-publicar", "-o", filepath.Join(raiz, nombre), ".")
	}
	construir("limpio", "true")
	limpio := Inspeccionar(raiz, "limpio", 16<<20)
	datos, err := os.ReadFile(filepath.Join(raiz, "limpio"))
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(datos)
	if limpio.Estado != "datos_observados" || limpio.Observado.RevisionVCS != revision || limpio.Observado.SHA256 != hex.EncodeToString(h[:]) || limpio.Observado.Modificado == nil || *limpio.Observado.Modificado || limpio.Observado.GOOS != runtime.GOOS || limpio.Observado.GOARCH != runtime.GOARCH {
		t.Fatalf("informe=%+v", limpio)
	}
	comprobarPrivacidad(t, limpio, raiz)
	if err := os.Symlink("limpio", filepath.Join(raiz, "actual")); err != nil {
		t.Fatal(err)
	}
	if actual := Inspeccionar(raiz, "actual", 16<<20); actual.Estado != "datos_observados" || actual.Observado.SHA256 != limpio.Observado.SHA256 {
		t.Fatalf("enlace interno=%+v", actual)
	}
	archivoExterior := filepath.Join(t.TempDir(), "fuera")
	if err := os.WriteFile(archivoExterior, datos, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(archivoExterior, filepath.Join(raiz, "fuera")); err != nil {
		t.Fatal(err)
	}
	construir("sin-vcs", "false")
	if informe := Inspeccionar(raiz, "sin-vcs", 16<<20); informe.Estado != "no_comprobable" || informe.Observado.RevisionVCS != "" {
		t.Fatalf("sin vcs: %+v", informe)
	}
	if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte("package main\nvar valor string\nfunc main() { _ = valor }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	construir("modificado", "true")
	if informe := Inspeccionar(raiz, "modificado", 16<<20); informe.Estado != "no_comprobable" || informe.Observado.Modificado == nil || !*informe.Observado.Modificado {
		t.Fatalf("modificado: %+v", informe)
	}
	if err := os.WriteFile(filepath.Join(raiz, "truncado"), datos[:64], 0600); err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		ruta string
		max  int64
	}{{"truncado", 1024}, {"limpio", int64(len(datos) - 1)}, {"ausente", 1024}, {"../fuera", 1024}, {"fuera", 1024}, {"limpio", 0}, {"limpio", MaxSnapshotBytes + 1}} {
		informe := Inspeccionar(raiz, caso.ruta, caso.max)
		if informe.Estado != "no_comprobable" || len(informe.Razones) == 0 {
			t.Fatalf("caso=%+v informe=%+v", caso, informe)
		}
		comprobarPrivacidad(t, informe, raiz)
	}
}

func TestMetadatosAmbiguosYNoPermitidos(t *testing.T) {
	for _, settings := range [][]debug.BuildSetting{
		{{Key: "GOOS", Value: "linux"}, {Key: "GOOS", Value: "otro"}},
		{{Key: "vcs", Value: "hg"}},
	} {
		informe := Informe{Estado: "datos_observados", Autenticidad: "no_comprobada"}
		extraer(&informe, &debug.BuildInfo{GoVersion: runtime.Version(), Settings: settings})
		if informe.Estado != "no_comprobable" {
			t.Fatalf("informe=%+v", informe)
		}
		comprobarPrivacidad(t, informe, "")
	}
}

func TestIgnoraMetadatosAjenosSinRomperVersionesFuturas(t *testing.T) {
	settings := []debug.BuildSetting{{Key: "GOOS", Value: runtime.GOOS}, {Key: "GOARCH", Value: runtime.GOARCH}, {Key: "vcs", Value: "git"}, {Key: "vcs.revision", Value: strings.Repeat("a", 40)}, {Key: "vcs.time", Value: "2026-10-01T12:00:00Z"}, {Key: "vcs.modified", Value: "false"}}
	informe := Informe{Estado: "datos_observados", Autenticidad: "no_comprobada"}
	extraer(&informe, &debug.BuildInfo{GoVersion: runtime.Version(), Settings: append(settings, debug.BuildSetting{Key: "vcs.futuro", Value: "token-sintetico-no-publicar"})})
	if informe.Estado != "datos_observados" || informe.Observado.VCS != "git" {
		t.Fatalf("informe=%+v", informe)
	}
	comprobarPrivacidad(t, informe, "")
}

func comprobarPrivacidad(t *testing.T, informe Informe, raiz string) {
	t.Helper()
	salida, err := json.Marshal(informe)
	if err != nil {
		t.Fatal(err)
	}
	if informe.HabilitaCopia || informe.HabilitaRestauracion || informe.Autenticidad != "no_comprobada" || strings.Contains(string(salida), "token-sintetico-no-publicar") || strings.Contains(string(salida), "material-sintetico") || raiz != "" && strings.Contains(string(salida), raiz) {
		t.Fatalf("salida no minimizada: %s", salida)
	}
}
