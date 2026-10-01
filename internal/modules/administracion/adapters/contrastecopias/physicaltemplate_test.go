package contrastecopias

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	domain "vec-diputacion-granada/internal/modules/administracion/domain/contrastecopias"
)

func TestFuentePlantillaFisicaConfiguracion(t *testing.T) {
	cfg := ConfiguracionPlantillaFisica{NombreClon: "clon_tecnico", BaseControl: "postgres", Lector: limitesPrueba()}
	cfg.Lector.BasesInventariadas = []string{"postgres", "template0", "template1"}
	if _, e := NuevoFuentePlantillaFisica(cfg, nil); e == nil {
		t.Fatal("runtime ausente admitido")
	}
	runtime := &runtimePlantillaInvalido{}
	for _, clone := range []string{"postgres", "template0", "host=privado"} {
		bad := cfg
		bad.NombreClon = clone
		if _, e := NuevoFuentePlantillaFisica(bad, runtime); e == nil {
			t.Fatal("nombre técnico no válido admitido")
		}
	}
	source, e := NuevoFuentePlantillaFisica(cfg, runtime)
	if e != nil {
		t.Fatal(e)
	}
	req := SolicitudBaseNoConectable{Nombre: "template0", VersionPostgreSQL: "18.4", PropiedadesSHA256: strings.Repeat("a", 64), SelloExclusion: strings.Repeat("b", 64), MaxBytes: 1 << 20, MaxFilas: 1000, MaxObjetos: 100}
	if _, e = source.CapturarBase(context.Background(), req); e == nil || runtime.created != 0 {
		t.Fatal("procedencia no observada obtuvo escritura")
	}
}

type runtimePlantillaInvalido struct{ created int }

func (r *runtimePlantillaInvalido) EjecutarPostgreSQL(context.Context, string, []string, []byte, int) ([]byte, error) {
	return nil, errPlantilla
}
func (r *runtimePlantillaInvalido) ComprobarExclusion(context.Context) (string, error) {
	return strings.Repeat("b", 64), nil
}
func (r *runtimePlantillaInvalido) ObservarProcedenciaFisica(context.Context) (ProcedenciaPlantillaFisica, error) {
	return ProcedenciaPlantillaFisica{}, errors.New("dato_privado_no_publicable")
}
func (r *runtimePlantillaInvalido) CrearClonPlantilla(context.Context, string, string) error {
	r.created++
	return errPlantilla
}
func (r *runtimePlantillaInvalido) RetirarClonPlantilla(context.Context, string) error {
	return errPlantilla
}

func TestLimpiezaPlantillaConservaRecursosPendientes(t *testing.T) {
	r := &runtimePlantillaFisicaEnsayo{source: "source-propio", target: "target-propio", scratch: "scratch-propio", madeSource: true, madeTarget: true}
	var names []string
	var contexts []context.Context
	removeContainer := func(ctx context.Context, name string) error {
		names = append(names, name)
		contexts = append(contexts, ctx)
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 5*time.Second {
			t.Error("retirada sin plazo independiente")
		}
		if name == r.source {
			return errors.New("detalle_privado_no_publicable")
		}
		return nil
	}
	removeDirectory := func(ctx context.Context, name string) error {
		contexts = append(contexts, ctx)
		if _, ok := ctx.Deadline(); !ok {
			t.Error("directorio sin plazo")
		}
		return errors.New("detalle_privado_no_publicable")
	}
	e := r.retirarRecursos(removeContainer, removeDirectory)
	if e == nil || strings.Contains(e.Error(), "detalle_privado") || !r.madeSource || r.madeTarget || r.scratch != "scratch-propio" {
		t.Fatal("fallo de limpieza perdió propiedad o divulgó el error")
	}
	if len(names) != 2 || names[0] != r.source || names[1] != r.target || len(contexts) != 3 || contexts[0] == contexts[1] || contexts[1] == contexts[2] {
		t.Fatal("un fallo impidió las otras retiradas")
	}
	names = nil
	if e = r.retirarRecursos(func(_ context.Context, name string) error { names = append(names, name); return nil }, func(context.Context, string) error { return nil }); e != nil || r.madeSource || r.madeTarget || r.scratch != "" || len(names) != 1 || names[0] != r.source {
		t.Fatal("reintento no retiró únicamente recursos pendientes propios")
	}
}

// El ensayo autentica por custodia local privada y verifica un archivo físico
// real obtenido tras parada limpia. No demuestra autenticación criptográfica de
// un conjunto externo; ese contrato corresponde al constructor de plataforma.
func TestFuentePlantillaFisicaPG18(t *testing.T) {
	if os.Getenv("VEC_CS06_PLANTILLA_FISICA_ENSAYO") != "1" {
		t.Skip("ensayo de copia física aislada no solicitado")
	}
	fixture := nuevoRuntimePlantillaFisicaEnsayo(t)
	defer fixture.cerrar(t)
	cfg := ConfiguracionPlantillaFisica{NombreClon: "cs06_clon_tecnico", BaseControl: "postgres", Lector: limitesPrueba()}
	cfg.Lector.BasesInventariadas = []string{"postgres", "template0", "template1", "vec_cs11"}
	cfg.Lector.MaxBytes = 32 << 20
	cfg.Lector.TiempoMaximo = 90 * time.Second
	source, e := NuevoFuentePlantillaFisica(cfg, fixture)
	if e != nil {
		t.Fatal(e)
	}
	reader, e := Nuevo(cfg.Lector)
	if e != nil {
		t.Fatal(e)
	}
	capture := func() domain.Snapshot {
		s, e := reader.CapturarEjecutorConFuente(context.Background(), fixture, "postgres", fixture, source)
		if e != nil {
			t.Fatal(e)
		}
		if !s.Completo || len(domain.Validar(s)) != 0 {
			t.Fatalf("copia física no produjo evidencia lógica completa: %v", s.Motivos)
		}
		return s
	}
	first := capture()
	second := capture()
	if domain.Comparar(first, second).Estado != domain.Igual {
		t.Fatal("mismo punto físico no conservó igualdad lógica")
	}
	req := solicitudPlantillaEnsayo(t, reader, fixture)
	// Cada incumplimiento de procedencia precede al permiso de crear el clon.
	for _, mode := range []string{"artefacto", "imagen", "ventana"} {
		fixture.corruption = mode
		before := fixture.creates
		if _, e = source.CapturarBase(context.Background(), req); e == nil || fixture.creates != before {
			t.Fatal("procedencia discrepante habilitó clon")
		}
	}
	fixture.corruption = ""
	// Un nombre existente no pertenece a esta llamada, aunque esté en el runtime
	// propio. La fuente nunca lo borra para conseguir una captura verde.
	fixture.tecnico(t, `CREATE DATABASE "cs06_clon_tecnico" TEMPLATE "template0";`)
	before := fixture.drops
	if _, e = source.CapturarBase(context.Background(), req); e == nil || fixture.drops != before || !fixture.existeBase(t, "cs06_clon_tecnico") {
		t.Fatal("clon preexistente retirado por la fuente")
	}
	fixture.tecnico(t, `DROP DATABASE "cs06_clon_tecnico";`)
	fixture.failReadClone = true
	if _, e = source.CapturarBase(context.Background(), req); e == nil || fixture.existeBase(t, "cs06_clon_tecnico") {
		t.Fatal("fallo de lectura dejó clon confirmado sin limpiar")
	}
	fixture.failReadClone = false
	fixture.failAckCreate = true
	if _, e = source.CapturarBase(context.Background(), req); e == nil || fixture.existeBase(t, "cs06_clon_tecnico") {
		t.Fatal("CREATE confirmado sin respuesta dejó clon propio")
	}
	fixture.failAckCreate = false
	// Las lecturas ordinarias conservan la configuración RO. La escritura técnica
	// solo está en CREATE/DROP cerrados y no altera datallowconn de template0.
	if fixture.valor(t, "postgres", `SHOW default_transaction_read_only;`) != "on" || fixture.valor(t, "postgres", `SELECT datallowconn::text FROM pg_database WHERE datname='template0';`) != "false" {
		t.Fatal("configuración original cambió durante el clon técnico")
	}
	if fixture.creates < 4 || fixture.drops < 4 {
		t.Fatal("operaciones técnicas reales no acreditadas")
	}
	t.Log("PG18.4 real: archivo físico tras parada limpia, custodia local privada y restauración del mismo punto; proveedor real completo, original no activado, RO conservado, procedencia/imagen/ventana discrepantes bloqueadas, clon ajeno conservado y clon propio limpiado tras fallo o pérdida de respuesta")
}

const imagenPlantillaEnsayo = "postgres:18.4"
const shaImagenPlantillaEnsayo = "sha256:1bf3d6960db467e87a506daef30feb41fecc23b7c5f96b157e873059f2ffb50a"
const dirDatosPlantillaEnsayo = "/var/lib/postgresql/18/docker"

type runtimePlantillaFisicaEnsayo struct {
	source, target, scratch, archive string
	expected, seal                   string
	provenance                       ProcedenciaPlantillaFisica
	corruption                       string
	creates, drops                   int
	failReadClone                    bool
	failAckCreate                    bool
	madeSource, madeTarget           bool
	ownedClones                      map[string]bool
}

func cmdPlantillaEnsayo(ctx context.Context, args []string, input []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "/usr/bin/docker", args...)
	cmd.Stdin = bytes.NewReader(input)
	out, e := cmd.Output()
	if e != nil {
		return nil, errPlantilla
	}
	return out, nil
}
func nuevoRuntimePlantillaFisicaEnsayo(t *testing.T) *runtimePlantillaFisicaEnsayo {
	t.Helper()
	r := &runtimePlantillaFisicaEnsayo{source: "vec-cs06-template-source-sol-20261002", target: "vec-cs06-template-target-sol-20261002", ownedClones: map[string]bool{}}
	scratch, e := os.MkdirTemp("/var/tmp", "vec-cs06-plantilla-fisica-")
	if e != nil {
		t.Fatal("custodia privada no disponible")
	}
	r.scratch = scratch
	t.Cleanup(func() { r.cerrar(t) })
	for _, name := range []string{r.source, r.target} {
		args := []string{"run", "-d", "--name", name, "--network", "none", "--label", "vec.cs06.owner=lector-sol-plantilla", "--cpus", "1", "--memory", "256m", "--pids-limit", "96", "--tmpfs", "/var/lib/postgresql:rw,size=256m", "--tmpfs", "/var/run/postgresql:rw,size=16m", "--entrypoint", "/bin/sh", imagenPlantillaEnsayo, "-c", "chown -R postgres:postgres /var/lib/postgresql /var/run/postgresql; exec tail -f /dev/null"}
		if _, e = cmdPlantillaEnsayo(context.Background(), args, nil); e != nil {
			t.Fatal("runtime sintético no disponible")
		}
		if name == r.source {
			r.madeSource = true
		} else {
			r.madeTarget = true
		}
	}
	if _, e = cmdPlantillaEnsayo(context.Background(), []string{"exec", "--user", "postgres", r.source, "initdb", "-D", dirDatosPlantillaEnsayo, "--auth=trust", "--encoding=UTF8"}, nil); e != nil {
		t.Fatal("inicialización sintética no disponible")
	}
	sourceFacts, e := r.inspeccionarNombre(context.Background(), r.source)
	if e != nil {
		t.Fatal(e)
	}
	r.arrancar(t, r.source, false)
	r.sqlDirecto(t, r.source, "postgres", `CREATE DATABASE vec_cs11;`)
	r.sqlDirecto(t, r.source, "vec_cs11", `CREATE TABLE public.datos(id int4,valor text); INSERT INTO public.datos VALUES(1,'punto físico');`)
	if observed := r.valorContenedor(t, r.source, "postgres", `SHOW data_directory;`); observed != dirDatosPlantillaEnsayo {
		t.Fatal("directorio de datos no observado")
	}
	if _, e = cmdPlantillaEnsayo(context.Background(), []string{"exec", "--user", "postgres", r.source, "pg_ctl", "-D", dirDatosPlantillaEnsayo, "-m", "fast", "-w", "stop"}, nil); e != nil {
		t.Fatal("parada limpia no acreditada")
	}
	data, e := cmdPlantillaEnsayo(context.Background(), []string{"exec", r.source, "tar", "-C", dirDatosPlantillaEnsayo, "-czf", "-", "."}, nil)
	if e != nil || len(data) == 0 || len(data) > 16<<20 {
		t.Fatal("archivo físico sintético no disponible")
	}
	r.archive = filepath.Join(scratch, "punto-fisico.tgz")
	if e = os.WriteFile(r.archive, data, 0600); e != nil {
		t.Fatal("custodia del archivo físico no disponible")
	}
	r.expected = shaBytes(data)
	if _, e = cmdPlantillaEnsayo(context.Background(), []string{"exec", r.target, "mkdir", "-p", dirDatosPlantillaEnsayo}, nil); e != nil {
		t.Fatal("destino físico no disponible")
	}
	if _, e = cmdPlantillaEnsayo(context.Background(), []string{"exec", "-i", r.target, "tar", "-C", dirDatosPlantillaEnsayo, "-xzf", "-"}, data); e != nil {
		t.Fatal("restauración física sintética no disponible")
	}
	r.arrancar(t, r.target, true)
	facts, e := r.inspeccionar(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	binding, _ := json.Marshal([]string{sourceFacts, facts, r.expected, "parada_limpia_archivo_real_restaurado_ventana_propia"})
	r.seal = shaBytes(binding)
	r.provenance = ProcedenciaPlantillaFisica{OperacionRef: "operacion:cs06:fisica:" + r.expected, ConjuntoRef: "conjunto:cs06:fisica:" + r.expected, ArtefactoFisicoSHA256: r.expected, ImagenSHA256: strings.TrimPrefix(shaImagenPlantillaEnsayo, "sha256:"), SelloExclusion: r.seal}
	if _, e = r.ObservarProcedenciaFisica(context.Background()); e != nil {
		t.Fatal(e)
	}
	return r
}
func (r *runtimePlantillaFisicaEnsayo) arrancar(t *testing.T, name string, readonly bool) {
	t.Helper()
	args := []string{"exec", "--user", "postgres", name, "pg_ctl", "-D", dirDatosPlantillaEnsayo, "-l", "/var/lib/postgresql/arranque.log", "-w", "start"}
	if readonly {
		args = append(args, "-o", "-c default_transaction_read_only=on")
	}
	if _, e := cmdPlantillaEnsayo(context.Background(), args, nil); e != nil {
		t.Fatal("arranque PostgreSQL sintético no disponible")
	}
}
func (r *runtimePlantillaFisicaEnsayo) cerrar(t *testing.T) {
	t.Helper()
	err := r.retirarRecursos(func(ctx context.Context, name string) error {
		_, e := cmdPlantillaEnsayo(ctx, []string{"rm", "-f", name}, nil)
		return e
	}, func(_ context.Context, path string) error { return os.RemoveAll(path) })
	if err != nil {
		t.Error("limpieza_recursos_sinteticos_no_confirmada")
	}
}

// Un fallo no borra el registro de propiedad: t.Cleanup puede reintentarlo y
// ninguna retirada pendiente impide intentar las otras. Los errores son nominales.
func (r *runtimePlantillaFisicaEnsayo) retirarRecursos(removeContainer, removeDirectory func(context.Context, string) error) error {
	failed := false
	remove := func(action func(context.Context, string) error, name string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		result := make(chan error, 1)
		go func() { result <- action(ctx, name) }()
		select {
		case e := <-result:
			return e
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if r.madeSource {
		if e := remove(removeContainer, r.source); e != nil {
			failed = true
		} else {
			r.madeSource = false
		}
	}
	if r.madeTarget {
		if e := remove(removeContainer, r.target); e != nil {
			failed = true
		} else {
			r.madeTarget = false
		}
	}
	if r.scratch != "" {
		if e := remove(removeDirectory, r.scratch); e != nil {
			failed = true
		} else {
			r.scratch = ""
		}
	}
	if failed {
		return errors.New("limpieza_recursos_sinteticos_no_confirmada")
	}
	return nil
}

func (r *runtimePlantillaFisicaEnsayo) sqlDirecto(t *testing.T, name, base, sql string) {
	t.Helper()
	if _, e := cmdPlantillaEnsayo(context.Background(), []string{"exec", "-i", "--user", "postgres", name, "psql", "-X", "-q", "-v", "ON_ERROR_STOP=1", "--dbname=" + base}, []byte(sql)); e != nil {
		t.Fatal("preparación sintética no disponible")
	}
}
func (r *runtimePlantillaFisicaEnsayo) valorContenedor(t *testing.T, name, base, sql string) string {
	t.Helper()
	out, e := cmdPlantillaEnsayo(context.Background(), []string{"exec", "-i", "--user", "postgres", name, "psql", "-X", "-q", "-A", "-t", "-v", "ON_ERROR_STOP=1", "--dbname=" + base}, []byte(sql))
	if e != nil {
		t.Fatal("observación sintética no disponible")
	}
	return strings.TrimSpace(string(out))
}
func (r *runtimePlantillaFisicaEnsayo) valor(t *testing.T, base, sql string) string {
	return r.valorContenedor(t, r.target, base, sql)
}
func (r *runtimePlantillaFisicaEnsayo) tecnico(t *testing.T, sql string) {
	t.Helper()
	if e := r.sqlTecnico(context.Background(), sql); e != nil {
		t.Fatal(e)
	}
}
func (r *runtimePlantillaFisicaEnsayo) existeBase(t *testing.T, base string) bool {
	t.Helper()
	return r.valor(t, "postgres", `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname=`+literal(base)+`);`) == "t"
}
func (r *runtimePlantillaFisicaEnsayo) inspeccionar(ctx context.Context) (string, error) {
	return r.inspeccionarNombre(ctx, r.target)
}
func (r *runtimePlantillaFisicaEnsayo) inspeccionarNombre(ctx context.Context, nombre string) (string, error) {
	out, e := cmdPlantillaEnsayo(ctx, []string{"inspect", nombre}, nil)
	if e != nil {
		return "", errPlantilla
	}
	var info []struct {
		Id, Image  string
		Config     struct{ Labels map[string]string }
		HostConfig struct {
			NetworkMode string
			Binds       []string
		}
		Mounts []struct{ Type string }
		State  struct{ Running bool }
	}
	if json.Unmarshal(out, &info) != nil || len(info) != 1 {
		return "", errPlantilla
	}
	i := info[0]
	if !i.State.Running || i.Image != shaImagenPlantillaEnsayo || i.HostConfig.NetworkMode != "none" || len(i.HostConfig.Binds) != 0 || len(i.Mounts) != 0 || i.Config.Labels["vec.cs06.owner"] != "lector-sol-plantilla" {
		return "", errPlantilla
	}
	b, _ := json.Marshal([]string{i.Id, i.Image, i.HostConfig.NetworkMode, i.Config.Labels["vec.cs06.owner"]})
	return shaBytes(b), nil
}
func (r *runtimePlantillaFisicaEnsayo) ComprobarExclusion(ctx context.Context) (string, error) {
	if _, e := r.inspeccionar(ctx); e != nil {
		return "", errPlantilla
	}
	return r.seal, nil
}
func (r *runtimePlantillaFisicaEnsayo) ObservarProcedenciaFisica(ctx context.Context) (ProcedenciaPlantillaFisica, error) {
	if _, e := r.inspeccionar(ctx); e != nil {
		return ProcedenciaPlantillaFisica{}, errPlantilla
	}
	stat, e := os.Stat(r.archive)
	if e != nil || stat.Mode().Perm() != 0600 || stat.Size() > 16<<20 {
		return ProcedenciaPlantillaFisica{}, errPlantilla
	}
	dir, e := os.Stat(r.scratch)
	if e != nil || dir.Mode().Perm() != 0700 {
		return ProcedenciaPlantillaFisica{}, errPlantilla
	}
	data, e := os.ReadFile(r.archive)
	if e != nil || shaBytes(data) != r.expected {
		return ProcedenciaPlantillaFisica{}, errPlantilla
	}
	switch r.corruption {
	case "artefacto":
		return ProcedenciaPlantillaFisica{}, errPlantilla
	case "imagen":
		return ProcedenciaPlantillaFisica{}, errPlantilla
	case "ventana":
		return ProcedenciaPlantillaFisica{}, errPlantilla
	}
	return r.provenance, nil
}
func (r *runtimePlantillaFisicaEnsayo) EjecutarPostgreSQL(ctx context.Context, tool string, args []string, input []byte, max int) ([]byte, error) {
	if tool != "psql" || !bytes.HasPrefix(input, []byte("BEGIN TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY;")) {
		return nil, errPlantilla
	}
	if r.failReadClone {
		for _, arg := range args {
			if arg == "--dbname=cs06_clon_tecnico" {
				return nil, errPlantilla
			}
		}
	}
	out, e := cmdPlantillaEnsayo(ctx, append([]string{"exec", "-i", "--user", "postgres", r.target, tool}, args...), input)
	if e != nil {
		return nil, errPlantilla
	}
	if len(out) > max {
		return nil, errLimite
	}
	return out, nil
}
func (r *runtimePlantillaFisicaEnsayo) sqlTecnico(ctx context.Context, sql string) error {
	// Este proceso técnico usa una opción de sesión, nunca ALTER ROLE/SYSTEM ni
	// configuración persistida. No se concede un transporte de escritura libre.
	_, e := cmdPlantillaEnsayo(ctx, []string{"exec", "-i", "--user", "postgres", "-e", "PGOPTIONS=-c default_transaction_read_only=off", r.target, "psql", "-X", "-q", "-v", "ON_ERROR_STOP=1", "--dbname=postgres"}, []byte(sql))
	return e
}
func (r *runtimePlantillaFisicaEnsayo) CrearClonPlantilla(ctx context.Context, source, clone string) error {
	if !baseAdmitida.MatchString(source) || !baseAdmitida.MatchString(clone) {
		return errPlantilla
	}
	if e := r.sqlTecnico(ctx, `CREATE DATABASE "`+clone+`" TEMPLATE "`+source+`";`); e != nil {
		return errPlantilla
	}
	r.creates++
	r.ownedClones[clone] = true
	if r.failAckCreate {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if e := r.RetirarClonPlantilla(cleanup, clone); e != nil {
			return e
		}
		return errPlantilla
	}
	return nil
}
func (r *runtimePlantillaFisicaEnsayo) RetirarClonPlantilla(ctx context.Context, clone string) error {
	if !baseAdmitida.MatchString(clone) || !r.ownedClones[clone] {
		return errPlantilla
	}
	if e := r.sqlTecnico(ctx, `DROP DATABASE "`+clone+`";`); e != nil {
		return errPlantilla
	}
	r.drops++
	delete(r.ownedClones, clone)
	return nil
}

func solicitudPlantillaEnsayo(t *testing.T, l *Lector, r *runtimePlantillaFisicaEnsayo) SolicitudBaseNoConectable {
	t.Helper()
	tx := &transporteEjecutor{exec: r, base: "postgres", maxBytes: l.limites.MaxBytes, timeout: l.limites.TiempoMaximo.Milliseconds()}
	bases, e := l.leerBases(context.Background(), tx, &presupuestoCaptura{objetos: 8})
	if e != nil {
		t.Fatal(e)
	}
	props := ""
	for _, b := range bases {
		if b.Nombre == "template0" {
			props = shaBytes(b.Propiedades)
		}
	}
	return SolicitudBaseNoConectable{Nombre: "template0", VersionPostgreSQL: "18.4", PropiedadesSHA256: props, SelloExclusion: r.seal, MaxBytes: 32 << 20, MaxFilas: 100000, MaxObjetos: 1000}
}
