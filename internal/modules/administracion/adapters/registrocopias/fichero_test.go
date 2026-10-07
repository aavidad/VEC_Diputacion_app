package registrocopias

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	unix "syscall"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	port "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

var declaracion = port.Declaracion{Actor: "actor:sintetico", Correlacion: "correlacion:prueba"}

func solicitud() operacionescopias.Solicitud {
	return operacionescopias.Solicitud{Operacion: "op:uno", Clave: "clave:uno", SHA256: strings.Repeat("a", 64), Conjunto: "conjunto:uno", Destino: "destino:uno", Politica: "politica:uno"}
}
func pruebaConfig(t *testing.T) Config {
	t.Helper()
	d := t.TempDir()
	r := t.TempDir()
	if err := os.Chmod(d, 0700); err != nil {
		t.Fatal(err)
	}
	return Config{Directorio: d, RaicesRestauradas: []string{r}, LimiteListado: 2}
}
func abrir(t *testing.T, c Config) *Fichero {
	t.Helper()
	f, e := Abrir(c)
	if e != nil {
		t.Fatal(e)
	}
	return f
}
func reservar(t *testing.T, f *Fichero) port.Resultado {
	t.Helper()
	r, e := f.Reservar(context.Background(), declaracion, solicitud())
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func leerTramas(t *testing.T, c Config) []trama {
	t.Helper()
	b, e := os.ReadFile(filepath.Join(c.Directorio, "operaciones.jsonl"))
	if e != nil {
		t.Fatal(e)
	}
	var r []trama
	for _, l := range bytes.Split(bytes.TrimSuffix(b, []byte{'\n'}), []byte{'\n'}) {
		var v trama
		if json.Unmarshal(l, &v) != nil {
			t.Fatal("json")
		}
		r = append(r, v)
	}
	return r
}
func TestReabrirTodasTransicionesReplayYAuditoria(t *testing.T) {
	cfg := pruebaConfig(t)
	s := solicitud()
	reserva := reservar(t, abrir(t, cfg))
	replay, e := abrir(t, cfg).Reservar(context.Background(), declaracion, s)
	if e != nil || !replay.Replay || replay.Recibo != reserva.Recibo || replay.Auditoria.Referencia == reserva.Auditoria.Referencia {
		t.Fatalf("reserva replay %+v %v", replay, e)
	}
	h := strings.Repeat("b", 64)
	comandos := []operacionescopias.Comando{
		{Clave: "c:0", VersionEsperada: 0, SolicitudSHA256: s.SHA256, Accion: "iniciar_captura"},
		{Clave: "c:1", VersionEsperada: 1, SolicitudSHA256: s.SHA256, Accion: "confirmar_captura", ManifiestoSHA256: h},
		{Clave: "c:2", VersionEsperada: 2, SolicitudSHA256: s.SHA256, Accion: "iniciar_verificacion", ManifiestoSHA256: h, Ejecucion: "ensayo:uno"},
		{Clave: "c:3", VersionEsperada: 3, SolicitudSHA256: s.SHA256, Accion: "declarar_ensayo", Evidencia: &operacionescopias.Evidencia{Modo: "fisico", Conjunto: s.Conjunto, ManifiestoSHA256: h, Ejecucion: "ensayo:uno", Referencia: "evidencia:fisica", SHA256: h, Resultado: "satisfactorio"}},
		{Clave: "c:4", VersionEsperada: 4, SolicitudSHA256: s.SHA256, Accion: "declarar_ensayo", Evidencia: &operacionescopias.Evidencia{Modo: "logico", Conjunto: s.Conjunto, ManifiestoSHA256: h, Ejecucion: "ensayo:uno", Referencia: "evidencia:logica", SHA256: h, Resultado: "satisfactorio"}},
	}
	for _, cmd := range comandos {
		r, e := abrir(t, cfg).Aplicar(context.Background(), declaracion, s.Operacion, cmd)
		if e != nil {
			t.Fatal(e)
		}
		q, e := abrir(t, cfg).Consultar(context.Background(), declaracion, s.Operacion)
		if e != nil || q.Recibo != r.Recibo || !reflect.DeepEqual(q.Historia, r.Historia) {
			t.Fatal("reopen", e)
		}
		rp, e := abrir(t, cfg).Aplicar(context.Background(), declaracion, s.Operacion, cmd)
		if e != nil || !rp.Replay || rp.Recibo != r.Recibo || len(rp.Historia) != len(r.Historia) {
			t.Fatal("replay", e)
		}
	}
	q, e := abrir(t, cfg).Consultar(context.Background(), declaracion, s.Operacion)
	if e != nil || q.Recibo.Estado != operacionescopias.VerificadaDeclarada || q.Reconciliacion != "autenticar_evidencias_antes_de_uso" {
		t.Fatal("declaration promoted", e)
	}
	tramas := leerTramas(t, cfg)
	n := 0
	for _, v := range tramas {
		if v.Registro.Evento != nil {
			n++
		}
		if v.Registro.Auditoria.Autoridad != "actor_declarado_sin_autorizacion" {
			t.Fatal("authority")
		}
	}
	if n != 5 || len(tramas) != 18 {
		t.Fatal(n, len(tramas))
	}
}
func TestConflictoGlobalCASRechazoAuditado(t *testing.T) {
	cfg := pruebaConfig(t)
	f := abrir(t, cfg)
	reservar(t, f)
	s := solicitud()
	s.Operacion = "op:otra"
	if _, e := f.Reservar(context.Background(), declaracion, s); !errors.Is(e, operacionescopias.ErrConflicto) {
		t.Fatal(e)
	}
	c := operacionescopias.Comando{Clave: "cmd:uno", VersionEsperada: 1, SolicitudSHA256: s.SHA256, Accion: "iniciar_captura"}
	r, e := f.Aplicar(context.Background(), declaracion, "op:uno", c)
	if !errors.Is(e, operacionescopias.ErrVersion) || r.Auditoria.Resultado != operacionescopias.ErrVersion.Error() {
		t.Fatal(e, r)
	}
	c.VersionEsperada = 0
	r, e = f.Aplicar(context.Background(), declaracion, "op:uno", c)
	if e != nil {
		t.Fatal(e)
	}
	c.SolicitudSHA256 = strings.Repeat("c", 64)
	if _, e = f.Aplicar(context.Background(), declaracion, "op:uno", c); !errors.Is(e, operacionescopias.ErrConflicto) {
		t.Fatal(e)
	}
	q, e := abrir(t, cfg).Consultar(context.Background(), declaracion, "op:uno")
	if e != nil || len(q.Historia) != 1 || q.Recibo != r.Recibo {
		t.Fatal(e)
	}
	if len(leerTramas(t, cfg)) != 6 {
		t.Fatal("audit missing")
	}
}
func TestConcurrentReservationAndCAS(t *testing.T) {
	cfg := pruebaConfig(t)
	const n = 12
	var wg sync.WaitGroup
	res := make(chan port.Resultado, n)
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := abrir(t, cfg).Reservar(context.Background(), declaracion, solicitud())
			res <- r
			errs <- e
		}()
	}
	wg.Wait()
	close(res)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	newCount := 0
	var recibo port.Recibo
	for r := range res {
		if !r.Replay {
			newCount++
		}
		if recibo.Referencia == "" {
			recibo = r.Recibo
		}
		if r.Recibo != recibo {
			t.Fatal("duplicate receipt")
		}
	}
	if newCount != 1 {
		t.Fatal(newCount)
	}
	wins := make(chan error, 2)
	for _, key := range []string{"a", "b"} {
		wg.Add(1)
		go func(key string) {
			defer wg.Done()
			_, e := abrir(t, cfg).Aplicar(context.Background(), declaracion, "op:uno", operacionescopias.Comando{Clave: key, VersionEsperada: 0, SolicitudSHA256: solicitud().SHA256, Accion: "iniciar_captura"})
			wins <- e
		}(key)
	}
	wg.Wait()
	a, b := <-wins, <-wins
	if (a == nil) == (b == nil) || (a != nil && !errors.Is(a, operacionescopias.ErrVersion)) || (b != nil && !errors.Is(b, operacionescopias.ErrVersion)) {
		t.Fatal(a, b)
	}
}

func TestConcurrentDestinationExclusion(t *testing.T) {
	cfg := pruebaConfig(t)
	var wg sync.WaitGroup
	out := make(chan error, 2)
	for _, op := range []string{"op:a", "op:b"} {
		wg.Add(1)
		go func(op string) {
			defer wg.Done()
			s := solicitud()
			s.Operacion, s.Clave = op, op
			_, e := abrir(t, cfg).Reservar(context.Background(), declaracion, s)
			out <- e
		}(op)
	}
	wg.Wait()
	a, b := <-out, <-out
	if (a == nil) == (b == nil) || (a != nil && !errors.Is(a, port.ErrDestinoOcupado)) || (b != nil && !errors.Is(b, port.ErrDestinoOcupado)) {
		t.Fatal(a, b)
	}
	f := abrir(t, cfg)
	r, e := f.Listar(context.Background(), declaracion, port.Consulta{Limite: 2})
	if e != nil || len(r.Operaciones) != 1 {
		t.Fatal(r, e)
	}
	winner := r.Operaciones[0].Solicitud
	if r, e := f.Reservar(context.Background(), declaracion, winner); e != nil || !r.Replay {
		t.Fatal("active replay", e)
	}
}
func TestSubprocessReservation(t *testing.T) {
	if os.Getenv("VEC_CS07_HELPER") == "1" {
		f, e := Abrir(Config{Directorio: os.Getenv("VEC_CS07_CONTROL"), RaicesRestauradas: []string{os.Getenv("VEC_CS07_RESTORED")}})
		if e != nil {
			t.Fatal(e)
		}
		if _, e = f.Reservar(context.Background(), declaracion, solicitud()); e != nil {
			t.Fatal(e)
		}
		return
	}
	cfg := pruebaConfig(t)
	const n = 4
	cmds := make([]*exec.Cmd, n)
	for i := range cmds {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSubprocessReservation$") // #nosec G204 -- This test binary with fixed test selector, no shell.
		cmd.Env = []string{"VEC_CS07_HELPER=1", "VEC_CS07_CONTROL=" + cfg.Directorio, "VEC_CS07_RESTORED=" + cfg.RaicesRestauradas[0]}
		cmds[i] = cmd
		if e := cmd.Start(); e != nil {
			t.Fatal(e)
		}
	}
	for _, cmd := range cmds {
		if e := cmd.Wait(); e != nil {
			t.Fatal(e)
		}
	}
	tramas := leerTramas(t, cfg)
	if len(tramas) != n {
		t.Fatal(len(tramas))
	}
	for _, v := range tramas {
		if v.Registro.Recibo != tramas[0].Registro.Recibo {
			t.Fatal("duplicated reservation")
		}
	}
}
func TestTornWritesAndWholeFrameTruncationFailClosed(t *testing.T) {
	for _, mode := range []string{"partial", "newline", "boundary", "marker", "missing_marker", "tamper", "duplicate_json"} {
		t.Run(mode, func(t *testing.T) {
			cfg := pruebaConfig(t)
			f := abrir(t, cfg)
			reservar(t, f)
			if _, e := f.Consultar(context.Background(), declaracion, "op:uno"); e != nil {
				t.Fatal(e)
			}
			p := filepath.Join(cfg.Directorio, "operaciones.jsonl")
			b, e := os.ReadFile(p)
			if e != nil {
				t.Fatal(e)
			}
			marker := filepath.Join(cfg.Directorio, "confirmaciones.sha256")
			switch mode {
			case "partial":
				b = b[:len(b)-20]
			case "newline":
				b = b[:len(b)-1]
			case "boundary":
				b = b[:bytes.IndexByte(b, '\n')+1]
			case "marker":
				m, e := os.ReadFile(marker)
				if e != nil {
					t.Fatal(e)
				}
				if os.WriteFile(marker, m[:len(m)-1], 0600) != nil {
					t.Fatal("write marker")
				}
			case "missing_marker":
				if os.Remove(marker) != nil {
					t.Fatal("remove")
				}
			case "tamper":
				b = bytes.Replace(b, []byte("actor:sintetico"), []byte("actor:alterado"), 1)
			case "duplicate_json":
				b = bytes.Replace(b, []byte(`"secuencia":1`), []byte(`"secuencia":1,"secuencia":1`), 1)
			}
			if e := os.WriteFile(p, b, 0600); e != nil {
				t.Fatal(e)
			}
			before, _ := os.ReadFile(p)
			_, e = abrir(t, cfg).Consultar(context.Background(), declaracion, "op:uno")
			after, _ := os.ReadFile(p)
			if !errors.Is(e, port.ErrCorrupto) || !bytes.Equal(before, after) {
				t.Fatal("accepted/changed torn history", e)
			}
		})
	}
}
func TestRollbackIndependentAndList(t *testing.T) {
	cfg := pruebaConfig(t)
	f := abrir(t, cfg)
	reservar(t, f)
	for _, op := range []string{"op:dos", "op:tres"} {
		s := solicitud()
		s.Operacion, s.Clave, s.Destino = op, op, op
		if _, e := f.Reservar(context.Background(), declaracion, s); e != nil {
			t.Fatal(e)
		}
	}
	if e := os.RemoveAll(cfg.RaicesRestauradas[0]); e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(cfg.RaicesRestauradas[0], 0700); e != nil {
		t.Fatal(e)
	}
	f = abrir(t, cfg)
	r, e := f.Listar(context.Background(), declaracion, port.Consulta{})
	if e != nil || len(r.Operaciones) != 2 || r.Siguiente != "op:tres" {
		t.Fatal(r, e)
	}
	r, e = f.Listar(context.Background(), declaracion, port.Consulta{Despues: r.Siguiente})
	if e != nil || len(r.Operaciones) != 1 || r.Operaciones[0].Solicitud.Operacion != "op:uno" || r.Siguiente != "" {
		t.Fatal(r, e)
	}
	if _, e = f.Listar(context.Background(), declaracion, port.Consulta{Limite: 3}); !errors.Is(e, port.ErrEntrada) {
		t.Fatal(e)
	}
}
func TestConfigurationSymlinkAndCancellation(t *testing.T) {
	cfg := pruebaConfig(t)
	bad := cfg
	bad.Directorio = cfg.RaicesRestauradas[0]
	if _, e := Abrir(bad); !errors.Is(e, port.ErrConfiguracion) {
		t.Fatal(e)
	}
	f := abrir(t, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := f.Reservar(ctx, declaracion, solicitud()); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	if e := os.Symlink(filepath.Join(cfg.RaicesRestauradas[0], "victim"), filepath.Join(cfg.Directorio, "operaciones.jsonl")); e != nil {
		t.Fatal(e)
	}
	if _, e := f.Reservar(context.Background(), declaracion, solicitud()); !errors.Is(e, port.ErrIO) {
		t.Fatal(e)
	}
	if _, e := os.Stat(filepath.Join(cfg.RaicesRestauradas[0], "victim")); !os.IsNotExist(e) {
		t.Fatal("followed symlink")
	}
	if e := os.Remove(filepath.Join(cfg.Directorio, "operaciones.jsonl")); e != nil {
		t.Fatal(e)
	}
	fd, e := unix.Open(filepath.Join(cfg.Directorio, "operaciones.jsonl"), unix.O_RDWR|unix.O_CREAT, 0600)
	if e != nil {
		t.Fatal(e)
	}
	defer func() { _ = unix.Close(fd) }()
	if e := unix.Flock(fd, unix.LOCK_EX); e != nil {
		t.Fatal(e)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, e = f.Reservar(ctx, declaracion, solicitud()); !errors.Is(e, context.DeadlineExceeded) {
		t.Fatal(e)
	}
}
