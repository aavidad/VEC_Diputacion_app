package observabilidad

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

// destinoSeguro acumula líneas de forma concurrente y segura.
type destinoSeguro struct {
	mu    sync.Mutex
	datos bytes.Buffer
}

func (d *destinoSeguro) Write(p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.datos.Write(p)
}

func (d *destinoSeguro) texto() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.datos.String()
}

func (d *destinoSeguro) lineas(t *testing.T) []map[string]any {
	t.Helper()
	var resultado []map[string]any
	escaner := bufio.NewScanner(strings.NewReader(d.texto()))
	for escaner.Scan() {
		linea := escaner.Bytes()
		if !utf8.Valid(linea) {
			t.Fatalf("linea no UTF-8: %q", linea)
		}
		var campos map[string]any
		decodificador := json.NewDecoder(bytes.NewReader(linea))
		decodificador.UseNumber()
		if err := decodificador.Decode(&campos); err != nil {
			t.Fatalf("linea no JSON: %q: %v", linea, err)
		}
		resultado = append(resultado, campos)
	}
	return resultado
}

// destinoBloqueado no devuelve hasta que se libera.
type destinoBloqueado struct {
	liberar chan struct{}
	entrado chan struct{}
	una     sync.Once
}

func nuevoDestinoBloqueado() *destinoBloqueado {
	return &destinoBloqueado{liberar: make(chan struct{}), entrado: make(chan struct{})}
}

func (d *destinoBloqueado) Write(p []byte) (int, error) {
	d.una.Do(func() { close(d.entrado) })
	<-d.liberar
	return len(p), nil
}

type destinoFallido struct{}

func (destinoFallido) Write([]byte) (int, error) { return 0, errors.New("disco lleno en /home/x") }

type aleatorioFallido struct{}

func (aleatorioFallido) Read([]byte) (int, error) { return 0, errors.New("sin entropia") }

var camposPermitidos = []string{
	"codigo", "componente", "correlacion", "entorno", "esquema", "etapa",
	"instante", "mensaje", "recuento", "severidad", "version_binario",
}

func nuevoEmisor(t *testing.T, o OpcionesEmisor) *EmisorJSONLines {
	t.Helper()
	e, err := NuevoEmisorJSONLines(o)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func cerrar(t *testing.T, e *EmisorJSONLines) {
	t.Helper()
	ctx, cancelar := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelar()
	if err := e.Cerrar(ctx); err != nil {
		t.Fatalf("cerrar: %v", err)
	}
}

func solicitudValida() domain.SolicitudIncidenciaTecnica {
	return domain.SolicitudIncidenciaTecnica{
		Codigo: domain.IncidenciaArranqueFallido, Componente: domain.ComponenteIncidenciaComposicion, Etapa: domain.EtapaIncidenciaComposicion,
	}
}

func TestEmisorEscribeListaBlancaExacta(t *testing.T) {
	destino := &destinoSeguro{}
	instante := time.Date(2026, 9, 25, 8, 0, 0, 987654321, time.UTC)
	e := nuevoEmisor(t, OpcionesEmisor{Destino: destino, Entorno: "produccion", VersionBinario: "1f5222d7", Reloj: func() time.Time { return instante }})
	e.Emitir(solicitudValida())
	e.Emitir(solicitudValida())
	cerrar(t, e)
	lineas := destino.lineas(t)
	if len(lineas) != 2 {
		t.Fatalf("se esperaban 2 lineas, hay %d", len(lineas))
	}
	for _, campos := range lineas {
		var claves []string
		for clave := range campos {
			claves = append(claves, clave)
		}
		sort.Strings(claves)
		if strings.Join(claves, ",") != strings.Join(camposPermitidos, ",") {
			t.Fatalf("campos fuera de lista blanca: %v", claves)
		}
		esperado := map[string]string{
			"esquema": domain.EsquemaIncidenciaTecnica, "instante": "2026-09-25T08:00:00.987Z",
			"codigo": "ARRANQUE_FALLIDO", "severidad": "critica", "componente": "composicion",
			"etapa": "composicion", "entorno": "produccion", "version_binario": "1f5222d7",
			"recuento": "1", "mensaje": "El servidor no ha podido arrancar.",
		}
		for clave, valor := range esperado {
			if got := campos[clave]; got == nil || jsonTexto(got) != valor {
				t.Fatalf("%s = %v, se esperaba %q", clave, got, valor)
			}
		}
		if !domain.EsCorrelacionTecnicaValida(jsonTexto(campos["correlacion"])) {
			t.Fatalf("correlacion no aleatoria hexadecimal: %v", campos["correlacion"])
		}
	}
	if lineas[0]["correlacion"] == lineas[1]["correlacion"] {
		t.Fatal("dos incidencias comparten correlacion")
	}
	m := e.MetricasEmision()
	if m.Aceptadas != 2 || m.Escritas != 2 || m.Descartadas != 0 || m.Saneadas != 0 || m.FallosEscritura != 0 || m.PendientesEnCola != 0 {
		t.Fatalf("metricas inesperadas: %+v", m)
	}
}

func jsonTexto(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case json.Number:
		return x.String()
	default:
		return ""
	}
}

func TestEmisorSaneaCodigosYCamposDesconocidos(t *testing.T) {
	destino := &destinoSeguro{}
	e := nuevoEmisor(t, OpcionesEmisor{Destino: destino})
	e.Emitir(domain.SolicitudIncidenciaTecnica{Codigo: "FALLO_INVENTADO", Componente: domain.ComponenteIncidenciaHTTP, Etapa: domain.EtapaIncidenciaPeticion})
	e.Emitir(domain.SolicitudIncidenciaTecnica{Codigo: domain.IncidenciaHTTPInternoFallido, Componente: "bolsa", Etapa: domain.EtapaIncidenciaPeticion})
	cerrar(t, e)
	for _, campos := range destino.lineas(t) {
		if campos["codigo"] != "RECOLECCION_DEGRADADA" || campos["componente"] != "supervision" || campos["etapa"] != "validacion" {
			t.Fatalf("no saneado: %v", campos)
		}
		if campos["entorno"] != "desconocido" || campos["version_binario"] != "desconocida" {
			t.Fatalf("metadatos no saneados: %v", campos)
		}
	}
	if m := e.MetricasEmision(); m.Saneadas != 2 || m.Escritas != 2 {
		t.Fatalf("metricas de saneamiento: %+v", m)
	}
}

func TestEmisorNuncaEmiteMarcadoresSinteticosSensibles(t *testing.T) {
	marcadores := []string{
		"12345678Z", "persona.sintetica@example.org", "Juan Sintetico Perez",
		"10.1.2.3", "per_0123456789abcdef", "/home/persona/vec", "exp_ct_0001",
		"SELECT * FROM personas", "goroutine 1 [running]",
	}
	destino := &destinoSeguro{}
	e := nuevoEmisor(t, OpcionesEmisor{
		Destino:        destino,
		Entorno:        strings.Join(marcadores, " "),
		VersionBinario: marcadores[0],
		Aleatorio:      strings.NewReader(strings.Repeat("per_0123456789ab", 64)),
	})
	for _, m := range marcadores {
		e.Emitir(domain.SolicitudIncidenciaTecnica{Codigo: domain.CodigoIncidenciaTecnica(m), Componente: domain.ComponenteIncidenciaHTTP, Etapa: domain.EtapaIncidenciaPeticion})
		e.Emitir(domain.SolicitudIncidenciaTecnica{Codigo: domain.IncidenciaHTTPInternoFallido, Componente: domain.ComponenteIncidenciaTecnica(m), Etapa: domain.EtapaIncidenciaPeticion})
		e.Emitir(domain.SolicitudIncidenciaTecnica{Codigo: domain.IncidenciaHTTPInternoFallido, Componente: domain.ComponenteIncidenciaHTTP, Etapa: domain.EtapaIncidenciaTecnica(m)})
		e.Emitir(domain.SolicitudIncidenciaTecnica{Codigo: domain.CodigoIncidenciaTecnica("ARRANQUE_FALLIDO " + m), Componente: domain.ComponenteIncidenciaServidor, Etapa: domain.EtapaIncidenciaEscucha})
	}
	cerrar(t, e)
	salida := destino.texto()
	if strings.Count(salida, "\n") != 4*len(marcadores) {
		t.Fatalf("lineas inesperadas: %d", strings.Count(salida, "\n"))
	}
	for _, m := range marcadores {
		for _, trozo := range []string{m, strings.ToLower(m)} {
			if strings.Contains(salida, trozo) {
				t.Fatalf("marcador sensible %q presente en la salida", trozo)
			}
		}
	}
	for _, trozo := range []string{"@", "/home", "per_", "10.1.", "12345678"} {
		if strings.Contains(salida, trozo) {
			t.Fatalf("fragmento sensible %q presente en la salida", trozo)
		}
	}
}

func TestEmisorColaLlenaDescartaYContabilizaSinBloquear(t *testing.T) {
	destino := nuevoDestinoBloqueado()
	e := nuevoEmisor(t, OpcionesEmisor{Destino: destino, Capacidad: 4})
	e.Emitir(solicitudValida())
	<-destino.entrado // el trabajador queda bloqueado en la primera escritura
	const total = 1000
	listo := make(chan struct{})
	go func() {
		defer close(listo)
		for i := 0; i < total; i++ {
			e.Emitir(solicitudValida())
		}
	}()
	select {
	case <-listo:
	case <-time.After(5 * time.Second):
		t.Fatal("Emitir bloqueo con la cola llena")
	}
	m := e.MetricasEmision()
	if m.Aceptadas != 5 || m.Descartadas != total-4 || m.PendientesEnCola != 4 {
		t.Fatalf("metricas con cola llena: %+v", m)
	}
	close(destino.liberar)
	cerrar(t, e)
	if m := e.MetricasEmision(); m.Escritas != 6 { // 5 aceptadas + 1 declaración de descartes
		t.Fatalf("escritas tras liberar: %+v", m)
	}
}

func TestEmisorDeclaraDescartesComoRecoleccionDegradada(t *testing.T) {
	destino := &destinoSeguro{}
	bloqueo := nuevoDestinoBloqueado()
	cambio := &destinoConmutado{bloqueo: bloqueo, final: destino}
	e := nuevoEmisor(t, OpcionesEmisor{Destino: cambio, Capacidad: 1, PeriodoInformeDescartes: 10 * time.Millisecond})
	e.Emitir(solicitudValida())
	<-bloqueo.entrado
	e.Emitir(solicitudValida()) // ocupa la cola
	for i := 0; i < 7; i++ {
		e.Emitir(solicitudValida()) // descartadas
	}
	close(bloqueo.liberar)
	limite := time.Now().Add(5 * time.Second)
	for !strings.Contains(destino.texto(), `"etapa":"emision"`) {
		if time.Now().After(limite) {
			t.Fatalf("no se declararon los descartes: %s", destino.texto())
		}
		time.Sleep(5 * time.Millisecond)
	}
	cerrar(t, e)
	var declaraciones int
	for _, campos := range destino.lineas(t) {
		if campos["etapa"] == "emision" {
			declaraciones++
			if campos["codigo"] != "RECOLECCION_DEGRADADA" || jsonTexto(campos["recuento"]) != "7" {
				t.Fatalf("declaracion de descartes inesperada: %v", campos)
			}
		}
	}
	if declaraciones != 1 {
		t.Fatalf("se esperaba una declaracion, hay %d", declaraciones)
	}
}

// destinoConmutado bloquea la primera escritura y envía el resto a final.
type destinoConmutado struct {
	bloqueo *destinoBloqueado
	final   io.Writer
	mu      sync.Mutex
	primera bool
}

func (d *destinoConmutado) Write(p []byte) (int, error) {
	d.mu.Lock()
	primera := !d.primera
	d.primera = true
	d.mu.Unlock()
	if primera {
		return d.bloqueo.Write(p)
	}
	return d.final.Write(p)
}

func TestEmisorConDestinoBloqueadoRetornaEnMenosDeUnMilisegundo(t *testing.T) {
	destino := nuevoDestinoBloqueado()
	e := nuevoEmisor(t, OpcionesEmisor{Destino: destino, Capacidad: 64})
	defer close(destino.liberar)
	e.Emitir(solicitudValida())
	<-destino.entrado
	const n = 20000
	duraciones := make([]time.Duration, n)
	for i := range duraciones {
		inicio := time.Now()
		e.Emitir(solicitudValida())
		duraciones[i] = time.Since(inicio)
	}
	sort.Slice(duraciones, func(i, j int) bool { return duraciones[i] < duraciones[j] })
	// Percentil 99 para tolerar desalojos del planificador ajenos al emisor;
	// la mediana debe quedar órdenes de magnitud por debajo del milisegundo.
	if p99 := duraciones[n*99/100]; p99 >= time.Millisecond {
		t.Fatalf("p99 de Emitir con destino bloqueado = %v", p99)
	}
	if p50 := duraciones[n/2]; p50 >= 100*time.Microsecond {
		t.Fatalf("mediana de Emitir con destino bloqueado = %v", p50)
	}
	if m := e.MetricasEmision(); m.Descartadas < n-64 {
		t.Fatalf("descartes insuficientes: %+v", m)
	}
}

func TestEmisorRafagasConcurrentes(t *testing.T) {
	destino := &destinoSeguro{}
	e := nuevoEmisor(t, OpcionesEmisor{Destino: destino, Capacidad: 128, PeriodoInformeDescartes: time.Hour})
	const productores, porProductor = 16, 2000
	var grupo sync.WaitGroup
	for p := 0; p < productores; p++ {
		grupo.Add(1)
		go func(p int) {
			defer grupo.Done()
			for i := 0; i < porProductor; i++ {
				s := solicitudValida()
				if i%3 == 0 {
					s.Codigo = "INVENTADO"
				}
				e.Emitir(s)
				_ = e.MetricasEmision()
			}
		}(p)
	}
	grupo.Wait()
	cerrar(t, e)
	m := e.MetricasEmision()
	if m.Aceptadas+m.Descartadas != productores*porProductor {
		t.Fatalf("aceptadas+descartadas = %d", m.Aceptadas+m.Descartadas)
	}
	declaracion := uint64(0)
	if m.Descartadas > 0 {
		declaracion = 1 // el cierre declara los descartes pendientes
	}
	lineas := destino.lineas(t)
	if uint64(len(lineas)) != m.Escritas || m.Escritas != m.Aceptadas+declaracion || m.PendientesEnCola != 0 {
		t.Fatalf("lineas=%d metricas=%+v", len(lineas), m)
	}
}

func TestEmisorNoRetieneMemoriaDelLlamante(t *testing.T) {
	destino := nuevoDestinoBloqueado()
	e := nuevoEmisor(t, OpcionesEmisor{Destino: destino, Capacidad: CapacidadMaxima})
	defer close(destino.liberar)
	e.Emitir(solicitudValida())
	<-destino.entrado
	runtime.GC()
	var antes runtime.MemStats
	runtime.ReadMemStats(&antes)
	// 64 cadenas distintas de 1 MiB: si la cola las retuviera, serían 64 MiB.
	for i := 0; i < 64; i++ {
		enorme := strings.Repeat(string(rune('a'+i%26)), 1<<20)
		e.Emitir(domain.SolicitudIncidenciaTecnica{Codigo: domain.CodigoIncidenciaTecnica(enorme), Componente: domain.ComponenteIncidenciaTecnica(enorme), Etapa: domain.EtapaIncidenciaTecnica(enorme)})
	}
	for i := 0; i < 3*CapacidadMaxima; i++ {
		e.Emitir(solicitudValida())
	}
	runtime.GC()
	var despues runtime.MemStats
	runtime.ReadMemStats(&despues)
	if crecimiento := int64(despues.HeapAlloc) - int64(antes.HeapAlloc); crecimiento > 8<<20 {
		t.Fatalf("la cola retiene %d bytes", crecimiento)
	}
	if m := e.MetricasEmision(); m.PendientesEnCola != CapacidadMaxima {
		t.Fatalf("cola no acotada a su capacidad: %+v", m)
	}
}

// Un código VÁLIDO obtenido como subcadena de un búfer grande no debe retener
// el búfer mientras la incidencia espera en la cola: el catálogo guarda sus
// propias constantes, no la cadena del llamante.
func TestEmisorNoRetieneBuferDeCodigoValido(t *testing.T) {
	destino := nuevoDestinoBloqueado()
	e := nuevoEmisor(t, OpcionesEmisor{Destino: destino, Capacidad: 128})
	defer close(destino.liberar)
	e.Emitir(solicitudValida())
	<-destino.entrado
	runtime.GC()
	var antes runtime.MemStats
	runtime.ReadMemStats(&antes)
	base := solicitudValida()
	for i := 0; i < 64; i++ {
		bufer := string(base.Codigo) + strings.Repeat(string(rune('a'+i%26)), 1<<20)
		s := base
		s.Codigo = domain.CodigoIncidenciaTecnica(bufer[:len(base.Codigo)])
		e.Emitir(s)
	}
	runtime.GC()
	var despues runtime.MemStats
	runtime.ReadMemStats(&despues)
	if crecimiento := int64(despues.HeapAlloc) - int64(antes.HeapAlloc); crecimiento > 8<<20 {
		t.Fatalf("la cola retiene %d bytes de búferes del llamante", crecimiento)
	}
	if m := e.MetricasEmision(); m.PendientesEnCola == 0 || m.Saneadas != 0 {
		t.Fatalf("los códigos válidos debían aceptarse sin sanear: %+v", m)
	}
}

func TestEmisorCierre(t *testing.T) {
	t.Run("vacia pendientes y rechaza despues", func(t *testing.T) {
		destino := &destinoSeguro{}
		e := nuevoEmisor(t, OpcionesEmisor{Destino: destino, Capacidad: 256, PeriodoInformeDescartes: time.Hour})
		for i := 0; i < 200; i++ {
			e.Emitir(solicitudValida())
		}
		cerrar(t, e)
		cerrar(t, e) // idempotente
		e.Emitir(solicitudValida())
		m := e.MetricasEmision()
		if m.Escritas != 200 || m.Descartadas != 1 || len(destino.lineas(t)) != 200 {
			t.Fatalf("cierre no vacio la cola o acepto despues: %+v", m)
		}
	})
	t.Run("destino bloqueado respeta el contexto", func(t *testing.T) {
		destino := nuevoDestinoBloqueado()
		defer close(destino.liberar)
		e := nuevoEmisor(t, OpcionesEmisor{Destino: destino})
		e.Emitir(solicitudValida())
		<-destino.entrado
		ctx, cancelar := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancelar()
		inicio := time.Now()
		if err := e.Cerrar(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("cerrar con destino bloqueado: %v", err)
		}
		if time.Since(inicio) > 2*time.Second {
			t.Fatal("cerrar no respeto el contexto")
		}
	})
	t.Run("emision concurrente con cierre", func(t *testing.T) {
		destino := &destinoSeguro{}
		e := nuevoEmisor(t, OpcionesEmisor{Destino: destino, Capacidad: 64, PeriodoInformeDescartes: time.Hour})
		var grupo sync.WaitGroup
		for p := 0; p < 8; p++ {
			grupo.Add(1)
			go func() {
				defer grupo.Done()
				for i := 0; i < 500; i++ {
					e.Emitir(solicitudValida())
				}
			}()
		}
		cerrar(t, e)
		grupo.Wait()
		m := e.MetricasEmision()
		if m.Aceptadas+m.Descartadas != 4000 || m.PendientesEnCola != 0 {
			t.Fatalf("contabilidad tras cierre concurrente: %+v", m)
		}
		// Toda incidencia aceptada se escribió: ninguna quedó en la cola.
		escritasIncidencias := uint64(0)
		for _, campos := range destino.lineas(t) {
			if campos["codigo"] == "ARRANQUE_FALLIDO" {
				escritasIncidencias++
			}
		}
		if escritasIncidencias != m.Aceptadas {
			t.Fatalf("aceptadas %d, escritas %d", m.Aceptadas, escritasIncidencias)
		}
	})
}

func TestEmisorToleraFallosDeDestinoYEntropia(t *testing.T) {
	e := nuevoEmisor(t, OpcionesEmisor{Destino: destinoFallido{}})
	e.Emitir(solicitudValida())
	cerrar(t, e)
	if m := e.MetricasEmision(); m.FallosEscritura != 1 || m.Escritas != 0 {
		t.Fatalf("fallo de escritura no contabilizado: %+v", m)
	}
	destino := &destinoSeguro{}
	e = nuevoEmisor(t, OpcionesEmisor{Destino: destino, Aleatorio: aleatorioFallido{}})
	e.Emitir(solicitudValida())
	cerrar(t, e)
	lineas := destino.lineas(t)
	if len(lineas) != 1 || lineas[0]["correlacion"] != strings.Repeat("0", 32) {
		t.Fatalf("correlacion sin entropia: %v", lineas)
	}
}

func TestEmisorConfiguracion(t *testing.T) {
	if _, err := NuevoEmisorJSONLines(OpcionesEmisor{}); !errors.Is(err, ErrDestinoIncidenciasAusente) {
		t.Fatalf("destino ausente: %v", err)
	}
	e := nuevoEmisor(t, OpcionesEmisor{Destino: io.Discard, Capacidad: 1 << 30})
	if cap(e.cola) != CapacidadMaxima {
		t.Fatalf("capacidad no acotada: %d", cap(e.cola))
	}
	cerrar(t, e)
	var inerte *EmisorJSONLines
	inerte.Emitir(solicitudValida())
	if inerte.MetricasEmision() != (ports.MetricasEmisionIncidencias{}) || inerte.Cerrar(context.Background()) != nil {
		t.Fatal("emisor nil no inerte")
	}
}

func BenchmarkEmitirConDestinoBloqueado(b *testing.B) {
	destino := nuevoDestinoBloqueado()
	e, err := NuevoEmisorJSONLines(OpcionesEmisor{Destino: destino, Capacidad: 64})
	if err != nil {
		b.Fatal(err)
	}
	defer close(destino.liberar)
	s := solicitudValida()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.Emitir(s)
	}
}
