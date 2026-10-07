package ensayofisicopg

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

type observadorReal struct {
	t            *testing.T
	nombre, raiz string
	observado    bool
}

func (o *observadorReal) Observar(ctx context.Context, entorno puertos.Entorno) (puertos.Observacion, error) {
	o.t.Helper()
	r := entorno.PostgreSQL.(RuntimeObservacion)
	o.nombre, o.raiz = r.Nombre, r.Raiz
	if !entorno.Aislamiento.RuntimeInspeccionado || !entorno.Aislamiento.ConfiguracionOrigenExcluida {
		o.t.Fatal("aislamiento no observado")
	}
	antes, err := r.ComprobarExclusion(ctx)
	if err != nil {
		return puertos.Observacion{}, err
	}
	b, err := r.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-At", "-U", "cs06_fixture", "-d", "postgres", "-c", "SELECT codigo FROM prueba ORDER BY codigo"}, nil, 4096)
	if err != nil || strings.TrimSpace(string(b)) != "dato_sintetico" {
		o.t.Logf("dato testigo: %q error %v", b, err)
		return puertos.Observacion{}, fmt.Errorf("dato_restaurado_no_comprobable")
	}
	if err := comprobarPlantillasReales(ctx, entorno, o.t); err != nil {
		o.t.Logf("plantillas error %v", err)
		return puertos.Observacion{}, err
	}
	// Testigo real archivado: psql de la misma imagen, recuperado desde su TAR.
	// No se presenta como binario VEC ni como consulta funcional autorizada.
	b, err = r.EjecutarArchivado(ctx, "fisica:testigo", []string{"--version"}, nil, 4096)
	if err != nil || !strings.Contains(string(b), "18.4") {
		o.t.Logf("bin testigo: %q error %v", b, err)
		return puertos.Observacion{}, fmt.Errorf("testigo_archivado_no_comprobable")
	}
	despues, err := r.ComprobarExclusion(ctx)
	if err != nil || despues != antes {
		o.t.Logf("sello runtime: before %s after %s error %v", antes, despues, err)
		return puertos.Observacion{}, fmt.Errorf("exclusion_cambiada")
	}
	if err := comprobarProceso(ctx, entorno); err != nil {
		o.t.Logf("testigo proceso error %v", err)
		return puertos.Observacion{}, err
	}
	o.observado = true
	return puertos.Observacion{ContrasteEstado: "no_comprobable", ArranqueEstado: "no_comprobable"}, nil
}
func TestFisicaPG18RealAislada(t *testing.T) {
	imagen := os.Getenv("VEC_CS06F_IMAGEN_SHA256")
	if imagen == "" {
		t.Skip("requiere imagen PG18.4 local fijada en VEC_CS06F_IMAGEN_SHA256")
	}
	if !huellaValida.MatchString(imagen) {
		t.Fatal("imagen no admitida")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	raiz, err := os.MkdirTemp("/var/tmp", "vec-cs06f-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	nombre := filepath.Base(raiz)
	t.Cleanup(func() {
		if !eliminarContenedor(nombre) {
			t.Error("fixture container pendiente")
		}
		if os.RemoveAll(raiz) != nil {
			t.Error("fixture scratch pendiente")
		}
	})
	datos := filepath.Join(raiz, "pgdata")
	if os.Mkdir(datos, 0700) != nil {
		t.Fatal("pgdata")
	}
	c := Configuracion{ImagenSHA256: imagen, VersionPostgreSQL: "18.4", UsuarioBootstrap: "cs06_fixture", LimiteArchivoBytes: 128 << 20, LimiteExtraidoBytes: 128 << 20, LimiteEntradas: 10000, CPUs: 1, MemoriaBytes: 2 << 30, TiempoLimite: 90 * time.Second}
	e := Ensayador{Configuracion: c}
	opciones := e.opcionesAisladas(nombre)
	cmd := append(opciones, "-d", "-v", datos+":/data:rw", "-e", "PGDATA=/data", "-e", "POSTGRES_USER=cs06_fixture", "-e", "POSTGRES_HOST_AUTH_METHOD=trust", "sha256:"+imagen, "postgres", "-c", "listen_addresses=", "-c", "unix_socket_directories=/var/run/postgresql")
	if _, err = docker(ctx, nil, 4096, cmd...); err != nil {
		t.Fatal("iniciar fixture", err)
	}
	inicio, cerrarInicio := context.WithTimeout(ctx, 30*time.Second)
	defer cerrarInicio()
	for inicio.Err() == nil {
		estado, _ := docker(inicio, nil, 4096, "inspect", "--format", "{{.State.Running}}", nombre)
		if strings.TrimSpace(string(estado)) != "true" {
			estado, _ := docker(inicio, nil, 4096, "inspect", "--format", "{{json .State}}", nombre)
			t.Logf("fixture state %s", estado)
			logs := exec.CommandContext(inicio, dockerLocal, "--host", "unix:///var/run/docker.sock", "logs", nombre)
			logs.Env = []string{"PATH=/usr/bin:/bin", "HOME=/nonexistent", "DOCKER_CONFIG=/nonexistent"}
			b, _ := logs.CombinedOutput()
			t.Fatalf("fixture no arrancó: %s", b)
		}
		b, err := docker(inicio, nil, 4096, "exec", nombre, "cat", "/proc/1/comm")
		if err == nil && strings.TrimSpace(string(b)) == "postgres" && e.esperar(inicio, nombre) {
			break
		}
		select {
		case <-inicio.Done():
			t.Fatal("fixture timeout")
		case <-time.After(100 * time.Millisecond):
		}
	}
	if _, err = docker(ctx, nil, 4096, "exec", nombre, "psql", "-X", "-U", "cs06_fixture", "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-c", "CREATE TABLE prueba(codigo text PRIMARY KEY CHECK(codigo <> '')); INSERT INTO prueba VALUES ('dato_sintetico');"); err != nil {
		t.Fatal("fixture SQL", err)
	}
	testigo := filepath.Join(raiz, "testigo")
	b, err := docker(ctx, nil, 4<<20, "exec", nombre, "cat", "/usr/lib/postgresql/18/bin/psql")
	if err != nil || os.WriteFile(testigo, b, 0700) != nil {
		t.Fatal("fixture testigo")
	}
	if _, err = docker(ctx, nil, 4096, "stop", "--time", "10", nombre); err != nil {
		t.Fatal("parada fría", err)
	}
	pg := empaquetar(t, datos)
	bin := empaquetar(t, testigo)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cert, key := archivosTLS(t)
	s := Solicitud{Sintetica: true, Componentes: []Componente{{ID: "fisica:pgdata", Tipo: "base_fisica", Tar: pg}, {ID: "fisica:testigo", Tipo: "binario", Tar: bin}, {ID: "fisica:servidor", Tipo: "binario", Tar: empaquetar(t, exe)}, {ID: "fisica:certificado", Tipo: "configuracion", Tar: empaquetar(t, cert)}, {ID: "fisica:clave", Tipo: "configuracion", Tar: empaquetar(t, key)}}}
	cuenta := &cuentaTar{}
	for _, comp := range s.Componentes {
		if err := revisarTar(ctx, comp.Tar.Ruta, c, cuenta); err != nil {
			t.Fatalf("TAR fixture %s no admitido bytes=%d entradas=%d: %v", comp.ID, cuenta.bytes, cuenta.entradas, err)
		}
	}
	o := &observadorReal{t: t}
	e.Observador = o
	r := e.Ensayar(ctx, s)
	if r.Estado != "restauracion_fisica_completada" || !r.LimpiezaCompletada || !o.observado || r.HabilitaRestauracion || r.Observacion.ArranqueEstado != "no_comprobable" {
		t.Fatalf("restore/testigo: %+v observado %v", r, o.observado)
	}
	if _, err := os.Stat(o.raiz); !os.IsNotExist(err) {
		t.Fatal("scratch ensayo sigue vivo")
	}
	if !eliminarContenedor(o.nombre) {
		t.Fatal("container ensayo sigue vivo")
	}
	c.VersionPostgreSQL = "18.3"
	r = (Ensayador{Configuracion: c}).Ensayar(ctx, s)
	if r.Estado != "restauracion_fisica_fallida" || r.Etapa != "versiones" || !r.LimpiezaCompletada {
		t.Fatalf("version mismatch: %+v", r)
	}
}
func empaquetar(t *testing.T, origen string) Archivo {
	t.Helper()
	destino := filepath.Join(t.TempDir(), "componente.tar")
	f, err := os.Create(destino)
	if err != nil {
		t.Fatal(err)
	}
	w := tar.NewWriter(f)
	err = filepath.Walk(origen, func(p string, i os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		rel, e := filepath.Rel(origen, p)
		if e != nil {
			return e
		}
		n := "contenido"
		if rel != "." {
			n += "/" + filepath.ToSlash(rel)
		}
		h, e := tar.FileInfoHeader(i, "")
		if e != nil {
			return e
		}
		h.Name = n
		h.Uid, h.Gid = 0, 0
		h.Uname, h.Gname = "", ""
		if e = w.WriteHeader(h); e != nil {
			return e
		}
		if i.Mode().IsRegular() {
			g, e := os.Open(p)
			if e != nil {
				return e
			}
			_, e = io.Copy(w, g)
			cerrar := g.Close()
			if e != nil {
				return e
			}
			return cerrar
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if w.Close() != nil || f.Close() != nil {
		t.Fatal("tar cerrar")
	}
	b, err := os.ReadFile(destino)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(b)
	return Archivo{destino, hex.EncodeToString(h[:])}
}
