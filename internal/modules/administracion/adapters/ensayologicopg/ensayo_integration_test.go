package ensayologicopg

import (
	"bytes"
	"context"
	"crypto/rand"
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
)

const limiteFixture = 4 << 20

func TestEnsayoLogicoPostgreSQL18Aislado(t *testing.T) {
	imagen := os.Getenv("VEC_CS06L_IMAGEN_SHA256")
	if imagen == "" {
		t.Skip("ensayo real omitido: requiere VEC_CS06L_IMAGEN_SHA256 de una imagen PostgreSQL 18.4 local")
	}
	if len(imagen) != 64 {
		t.Fatal("VEC_CS06L_IMAGEN_SHA256 requiere 64 caracteres hexadecimales sin prefijo")
	}
	if _, err := hex.DecodeString(imagen); err != nil {
		t.Fatal("VEC_CS06L_IMAGEN_SHA256 no es hexadecimal")
	}
	origen := nuevoOrigenSintetico(t, imagen)
	origen.fixture(t, "origen_valido.sql")
	dumpValido, globals := origen.capturar(t)
	config := Configuracion{
		ImagenSHA256:       imagen,
		VersionPostgreSQL:  "18.4",
		UsuarioBootstrap:   "cs06_destino",
		LimiteArchivoBytes: limiteFixture,
		CPUs:               1,
		MemoriaBytes:       512 << 20,
		TiempoLimite:       90 * time.Second,
	}
	t.Run("dump_y_globals_con_propietarios_y_acl", func(t *testing.T) {
		resultado := (Ensayador{Configuracion: config}).Ensayar(context.Background(),
			Solicitud{Sintetica: true, Dump: dumpValido, Globals: globals})
		if resultado.Estado != "restauracion_logica_completada" || resultado.Etapa != "completado" {
			t.Fatalf("restauración lógica válida: estado=%q etapa=%q razones=%v", resultado.Estado, resultado.Etapa, resultado.Razones)
		}
		if resultado.VersionPostgreSQL != "18.4" || len(resultado.Razones) != 0 {
			t.Fatalf("evidencia del ensayo válido: versión=%q razones=%v", resultado.VersionPostgreSQL, resultado.Razones)
		}
		comprobarLimitesResultado(t, resultado)
	})
	t.Run("observador_antes_de_limpieza", func(t *testing.T) { verificarObservadores(t, context.Background(), config, dumpValido, globals) })
	t.Run("version_distinta_rechazada_antes_de_sql", func(t *testing.T) {
		otraVersion := config
		otraVersion.VersionPostgreSQL = "18.3"
		resultado := (Ensayador{Configuracion: otraVersion}).Ensayar(context.Background(),
			Solicitud{Sintetica: true, Dump: dumpValido, Globals: globals})
		if resultado.Estado != "restauracion_logica_fallida" || resultado.Etapa != "versiones" || len(resultado.Razones) == 0 {
			t.Fatalf("rechazo anterior al SQL por versión: estado=%q etapa=%q razones=%v", resultado.Estado, resultado.Etapa, resultado.Razones)
		}
		comprobarLimitesResultado(t, resultado)
	})
	t.Run("acl_no_se_omite_si_falta_su_rol", func(t *testing.T) {
		contenido, err := os.ReadFile(globals.Ruta)
		if err != nil {
			t.Fatal(err)
		}
		var sinLector []string
		for _, linea := range strings.Split(string(contenido), "\n") {
			if !strings.Contains(linea, "cs06l_lector") {
				sinLector = append(sinLector, linea)
			}
		}
		globalsIncompletos := archivoFixture(t, "globals-sin-lector.sql", []byte(strings.Join(sinLector, "\n")))
		resultado := (Ensayador{Configuracion: config}).Ensayar(context.Background(),
			Solicitud{Sintetica: true, Dump: dumpValido, Globals: globalsIncompletos})
		if resultado.Estado != "restauracion_logica_fallida" || resultado.Etapa != "restauracion" || len(resultado.Razones) == 0 {
			t.Fatalf("ACL con rol ausente: estado=%q etapa=%q razones=%v", resultado.Estado, resultado.Etapa, resultado.Razones)
		}
		comprobarLimitesResultado(t, resultado)
	})
	origen.fixture(t, "validador_cambiado.sql")
	if observado := strings.TrimSpace(string(origen.psql(t, "SELECT count(*) FROM cs06l_sintetico.registros WHERE NOT cs06l_sintetico.admite_codigo(codigo);"))); observado != "1" {
		t.Fatalf("la regresión CHECK no está reproducida: esperado 1 dato inválido, obtenido %q", observado)
	}
	dumpRegresivo, globalsRegresivos := origen.capturar(t)
	t.Run("check_a_funcion_cambiada_impide_restaurar", func(t *testing.T) {
		resultado := (Ensayador{Configuracion: config}).Ensayar(context.Background(),
			Solicitud{Sintetica: true, Dump: dumpRegresivo, Globals: globalsRegresivos})
		if resultado.Estado != "restauracion_logica_fallida" || resultado.Etapa != "restauracion" || len(resultado.Razones) == 0 {
			t.Fatalf("CHECK regresiva: estado=%q etapa=%q razones=%v", resultado.Estado, resultado.Etapa, resultado.Razones)
		}
		comprobarLimitesResultado(t, resultado)
	})
}

func comprobarLimitesResultado(t *testing.T, resultado Resultado) {
	t.Helper()
	if resultado.HabilitaRestauracion {
		t.Error("un ensayo lógico parcial no habilita la restauración del conjunto")
	}
	if !resultado.LimpiezaCompletada {
		t.Error("el ensayo no acredita la limpieza de sus recursos")
	}
}

// El origen pertenece exclusivamente a estas pruebas: nunca acepta una base,
// DSN, puerto o SQL del operador. Los dumps conservan propietarios y ACL.
type origenSintetico struct {
	contenedor   string
	configDocker string
}

func nuevoOrigenSintetico(t *testing.T, imagen string) *origenSintetico {
	t.Helper()
	var aleatorio [12]byte
	if _, err := rand.Read(aleatorio[:]); err != nil {
		t.Fatal(err)
	}
	datos, err := os.MkdirTemp("/var/tmp", "vec-cs06l-fuente-")
	if err != nil {
		t.Fatalf("crear PGDATA propio de la fuente: %v", err)
	}
	origen := &origenSintetico{
		contenedor:   "vec-cs06l-fixture-" + hex.EncodeToString(aleatorio[:]),
		configDocker: t.TempDir(),
	}
	t.Cleanup(func() {
		ctx, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelar()
		if _, err := origen.docker(ctx, nil, "rm", "--force", origen.contenedor); err != nil {
			t.Errorf("limpieza del origen sintético: %v", err)
		}
		if err := os.RemoveAll(datos); err != nil { // #nosec G703 -- directorio propio generado por MkdirTemp; no procede del operador.
			t.Errorf("limpieza del PGDATA propio de la fuente: %v", err)
		}
	})
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	uid, gid := os.Getuid(), os.Getgid()
	opcionesTmpfs := fmt.Sprintf("rw,nosuid,nodev,uid=%d,gid=%d,mode=700", uid, gid)
	_, err = origen.docker(ctx, nil,
		"run", "--detach", "--name", origen.contenedor,
		"--pull=never", "--network=none", "--read-only", "--rm",
		"--user", fmt.Sprintf("%d:%d", uid, gid), "--cpus=1", "--memory=512m", "--memory-swap=512m", "--pids-limit=96",
		"--cap-drop=ALL", "--security-opt=no-new-privileges",
		"--tmpfs", "/tmp:"+opcionesTmpfs+",size=16m",
		"--tmpfs", "/var/run/postgresql:"+opcionesTmpfs+",size=8m",
		"--tmpfs", "/var/lib/postgresql:"+opcionesTmpfs+",size=8m",
		"-v", datos+":/data:rw",
		"--env", "PGDATA=/data",
		"--env", "POSTGRES_USER=cs06l_origen_bootstrap",
		"--env", "POSTGRES_DB=cs06l_sintetica",
		"--env", "POSTGRES_HOST_AUTH_METHOD=trust",
		"sha256:"+imagen, "postgres", "-c", "listen_addresses=", "-c", "unix_socket_directories=/var/run/postgresql")
	if err != nil {
		t.Fatalf("crear origen PostgreSQL sintético: %v", err)
	}
	listo := false
	for ctx.Err() == nil {
		if _, err := origen.docker(ctx, nil, "exec", origen.contenedor,
			"pg_isready", "-h", "/var/run/postgresql", "-U", "cs06l_origen_bootstrap", "-d", "cs06l_sintetica"); err == nil {
			proceso, err := origen.docker(ctx, nil, "exec", origen.contenedor, "cat", "/proc/1/comm")
			if err == nil && strings.TrimSpace(string(proceso)) == "postgres" {
				listo = true
				break
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !listo {
		t.Fatal("el origen PostgreSQL sintético no arrancó dentro del límite")
	}
	version := strings.TrimSpace(string(origen.psql(t, "SHOW server_version;")))
	if version != "18.4" && !strings.HasPrefix(version, "18.4 ") {
		t.Fatalf("versión del origen: esperado 18.4, obtenido %q", version)
	}
	return origen
}

func (o *origenSintetico) psql(t *testing.T, sql string) []byte {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelar()
	salida, err := o.docker(ctx, strings.NewReader(sql), "exec", "--interactive", o.contenedor,
		"psql", "--no-psqlrc", "--set=ON_ERROR_STOP=1", "--tuples-only", "--no-align",
		"--host=/var/run/postgresql", "--username=cs06l_origen_bootstrap", "--dbname=cs06l_sintetica")
	if err != nil {
		t.Fatalf("preparar origen sintético: %v", err)
	}
	return salida
}

func (o *origenSintetico) fixture(t *testing.T, nombre string) {
	t.Helper()
	contenido, err := os.ReadFile(filepath.Join("testdata", nombre))
	if err != nil {
		t.Fatal(err)
	}
	o.psql(t, string(contenido))
}

func (o *origenSintetico) capturar(t *testing.T) (Archivo, Archivo) {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelar()
	dump, err := o.docker(ctx, nil, "exec", o.contenedor, "pg_dump",
		"--host=/var/run/postgresql", "--username=cs06l_origen_bootstrap", "--dbname=cs06l_sintetica", "--format=custom")
	if err != nil {
		t.Fatalf("capturar dump custom: %v", err)
	}
	if !bytes.HasPrefix(dump, []byte("PGDMP")) {
		t.Fatal("la captura no es un dump custom")
	}
	toc, err := o.docker(ctx, bytes.NewReader(dump), "exec", "--interactive", o.contenedor, "pg_restore", "--list")
	if err != nil {
		t.Fatalf("inspeccionar dump custom sintético: %v", err)
	}
	for _, linea := range strings.Split(string(toc), "\n") {
		if strings.Contains(linea, "Dumped from database version:") || strings.Contains(linea, "Dumped by pg_dump version:") {
			t.Logf("versión en cabecera sintética: %s", linea)
		}
	}
	for _, objeto := range []string{"cs06l_propietario", "TABLE cs06l_sintetico registros", "ACL", "SEQUENCE SET"} {
		if !bytes.Contains(toc, []byte(objeto)) {
			t.Fatalf("el dump no conserva %q", objeto)
		}
	}
	globals, err := o.docker(ctx, nil, "exec", o.contenedor, "pg_dumpall",
		"--host=/var/run/postgresql", "--username=cs06l_origen_bootstrap", "--globals-only", "--no-role-passwords")
	if err != nil {
		t.Fatalf("capturar globals: %v", err)
	}
	for _, rol := range []string{"CREATE ROLE cs06l_propietario;", "CREATE ROLE cs06l_lector;"} {
		if !bytes.Contains(globals, []byte(rol)) {
			t.Fatalf("globals no conserva el rol sintético %q", rol)
		}
	}
	return archivoFixture(t, "base.dump", dump), archivoFixture(t, "globals.sql", globals)
}

func archivoFixture(t *testing.T, nombre string, contenido []byte) Archivo {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), nombre)
	if err := os.WriteFile(ruta, contenido, 0600); err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256(contenido)
	return Archivo{Ruta: ruta, SHA256: hex.EncodeToString(huella[:])}
}

func (o *origenSintetico) docker(ctx context.Context, entrada io.Reader, argumentos ...string) ([]byte, error) {
	args := append([]string{"--host=unix:///var/run/docker.sock", "--config", o.configDocker}, argumentos...)
	cmd := exec.CommandContext(ctx, "/usr/bin/docker", args...)
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	cmd.Stdin = entrada
	var salida salidaLimitadaFixture
	cmd.Stdout = &salida
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return salida.Bytes(), nil
}

type salidaLimitadaFixture struct{ bytes.Buffer }

func (s *salidaLimitadaFixture) Write(p []byte) (int, error) {
	if len(p) > limiteFixture-s.Len() {
		return 0, fmt.Errorf("la salida de la fixture supera %d bytes", limiteFixture)
	}
	return s.Buffer.Write(p)
}
