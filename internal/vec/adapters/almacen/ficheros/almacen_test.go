package ficheros

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
	pruebasvec "vec-diputacion-granada/internal/vec/pruebas"
)

type relojFijo struct{ t time.Time }

func (r relojFijo) Ahora() time.Time { return r.t }

func instante() time.Time { return time.Date(2026, time.September, 25, 10, 0, 0, 0, time.UTC) }

func directorioPrueba(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "originales")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func nuevoPrueba(t *testing.T, dir string) *Almacen {
	t.Helper()
	a, err := Nuevo(Configuracion{Directorio: dir, TamanoMaximo: 1 << 20, RetencionMinimaAdmitida: 24 * time.Hour}, relojFijo{instante()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Cerrar() })
	return a
}

func contexto(t *testing.T, sufijo, accion string, objeto ports.ReferenciaObjetoAlmacen) ports.ContextoOperacionAlmacen {
	t.Helper()
	c, err := pruebasvec.NuevoContextoAlmacen(instante(), sufijo, accion, objeto)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func escritura(t *testing.T, sufijo, clave string, zona ports.ZonaAlmacen, contenido []byte) ports.SolicitudEscribirObjeto {
	t.Helper()
	suma := sha256.Sum256(contenido)
	return ports.SolicitudEscribirObjeto{
		Contexto:          contexto(t, sufijo, ports.AccionAlmacenEscribir, ports.ReferenciaObjetoAlmacen{}),
		ClaveIdempotencia: clave, Zona: zona, MIME: "application/pdf", Tamano: int64(len(contenido)),
		HuellaSHA256: hex.EncodeToString(suma[:]), Contenido: bytes.NewReader(contenido),
	}
}

func TestNuevoRechazaDirectoriosInseguros(t *testing.T) {
	base := t.TempDir()
	abierto := filepath.Join(base, "abierto")
	if err := os.Mkdir(abierto, 0o755); err != nil {
		t.Fatal(err)
	}
	privado := filepath.Join(base, "privado")
	if err := os.Mkdir(privado, 0o700); err != nil {
		t.Fatal(err)
	}
	enlace := filepath.Join(base, "enlace")
	if err := os.Symlink(privado, enlace); err != nil {
		t.Fatal(err)
	}
	reloj := relojFijo{instante()}
	for nombre, cfg := range map[string]Configuracion{
		"relativo":           {Directorio: "originales", TamanoMaximo: 10, RetencionMinimaAdmitida: time.Hour},
		"no canónico":        {Directorio: privado + "/../privado", TamanoMaximo: 10, RetencionMinimaAdmitida: time.Hour},
		"permisos":           {Directorio: abierto, TamanoMaximo: 10, RetencionMinimaAdmitida: time.Hour},
		"enlace":             {Directorio: enlace, TamanoMaximo: 10, RetencionMinimaAdmitida: time.Hour},
		"inexistente":        {Directorio: filepath.Join(base, "no"), TamanoMaximo: 10, RetencionMinimaAdmitida: time.Hour},
		"sin tamaño":         {Directorio: privado, RetencionMinimaAdmitida: time.Hour},
		"tamaño enorme":      {Directorio: privado, TamanoMaximo: 1 << 40, RetencionMinimaAdmitida: time.Hour},
		"límite negativo":    {Directorio: privado, TamanoMaximo: 10, RetencionMinimaAdmitida: time.Hour, MaximoVolcadosConcurrentes: -1},
		"límite excesivo":    {Directorio: privado, TamanoMaximo: 10, RetencionMinimaAdmitida: time.Hour, MaximoVolcadosConcurrentes: 17},
		"retención negativa": {Directorio: privado, TamanoMaximo: 10, RetencionMinimaAdmitida: -time.Hour},
	} {
		if _, err := Nuevo(cfg, reloj); err == nil {
			t.Fatalf("%s: configuración insegura aceptada", nombre)
		}
	}
	if _, err := Nuevo(Configuracion{Directorio: privado, TamanoMaximo: 10, RetencionMinimaAdmitida: time.Hour}, nil); err == nil {
		t.Fatal("sin reloj aceptado")
	}
}

type infoConUID struct {
	os.FileInfo
	uid uint32
}

func (i infoConUID) Sys() any { return &syscall.Stat_t{Uid: i.uid} }

func TestCerrojoExigePropietarioDelProceso(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "cerrojo")
	if err := os.WriteFile(ruta, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(ruta)
	if err != nil || !ficheroPrivado(info) || ficheroPrivado(infoConUID{info, uint32(os.Geteuid() + 1)}) {
		t.Fatalf("validación de propietario: %v", err)
	}
}

func TestNuevoLimpiaTemporalesHuerfanosSinSeguirEnlaces(t *testing.T) {
	dir := directorioPrueba(t)
	a := nuevoPrueba(t, dir)
	temporal, err := a.temporal()
	if err != nil {
		t.Fatal(err)
	}
	ruta := temporal.Name()
	if err := temporal.Close(); err != nil {
		t.Fatal(err)
	}
	ajeno := filepath.Join(dir, "ajeno")
	if err := os.WriteFile(ajeno, []byte("intacto"), 0o600); err != nil {
		t.Fatal(err)
	}
	enlace := filepath.Join(dir, dirTemporal, "tmp_"+strings.Repeat("a", 32))
	if err := os.Symlink(ajeno, enlace); err != nil {
		t.Fatal(err)
	}
	if err := a.Cerrar(); err != nil {
		t.Fatal(err)
	}
	b := nuevoPrueba(t, dir)
	if _, err := os.Lstat(ruta); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temporal huérfano conservado: %v", err)
	}
	if _, err := os.Lstat(enlace); err != nil {
		t.Fatalf("enlace alterado: %v", err)
	}
	if contenido, err := os.ReadFile(ajeno); err != nil || string(contenido) != "intacto" {
		t.Fatalf("destino ajeno alterado: %v", err)
	}
	_ = b
}

type lectorBloqueado struct {
	origen io.Reader
	inicio chan<- struct{}
	libre  <-chan struct{}
	unaVez sync.Once
}

func (l *lectorBloqueado) Read(p []byte) (int, error) {
	l.unaVez.Do(func() { l.inicio <- struct{}{}; <-l.libre })
	return l.origen.Read(p)
}

func TestVolcadosLimitadosPorSemaforo(t *testing.T) {
	a, err := Nuevo(Configuracion{Directorio: directorioPrueba(t), TamanoMaximo: 1 << 20,
		RetencionMinimaAdmitida: time.Hour, MaximoVolcadosConcurrentes: 1}, relojFijo{instante()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Cerrar() })
	libre := make(chan struct{})
	inicios := make(chan struct{}, 2)
	errores := make(chan error, 2)
	for i := 0; i < 2; i++ {
		contenido := []byte(fmt.Sprintf("contenido %d", i))
		s := escritura(t, fmt.Sprintf("limite%d", i), fmt.Sprintf("clave:limite:%d", i), ports.ZonaAlmacenAdmitida, contenido)
		s.Contenido = &lectorBloqueado{origen: bytes.NewReader(contenido), inicio: inicios, libre: libre}
		go func() { _, err := a.Escribir(context.Background(), s); errores <- err }()
	}
	select {
	case <-inicios:
	case <-time.After(2 * time.Second):
		t.Fatal("ningún volcado comenzó")
	}
	select {
	case <-inicios:
		t.Fatal("dos volcados comenzaron con límite uno")
	case <-time.After(50 * time.Millisecond):
	}
	close(libre)
	for i := 0; i < 2; i++ {
		if err := <-errores; err != nil {
			t.Fatal(err)
		}
	}
}

func TestNuevoEsperaUnVolcadoActivoAntesDeLimpiar(t *testing.T) {
	dir := directorioPrueba(t)
	a := nuevoPrueba(t, dir)
	inicio := make(chan struct{}, 1)
	libre := make(chan struct{})
	contenido := []byte("contenido activo")
	s := escritura(t, "activo", "clave:activo", ports.ZonaAlmacenAdmitida, contenido)
	s.Contenido = &lectorBloqueado{origen: bytes.NewReader(contenido), inicio: inicio, libre: libre}
	escrituraTerminada := make(chan error, 1)
	go func() { _, err := a.Escribir(context.Background(), s); escrituraTerminada <- err }()
	select {
	case <-inicio:
	case <-time.After(2 * time.Second):
		t.Fatal("el volcado no comenzó")
	}
	arranqueTerminado := make(chan error, 1)
	go func() {
		b, err := Nuevo(Configuracion{Directorio: dir, TamanoMaximo: 1 << 20,
			RetencionMinimaAdmitida: time.Hour}, relojFijo{instante()})
		if b != nil {
			_ = b.Cerrar()
		}
		arranqueTerminado <- err
	}()
	select {
	case err := <-arranqueTerminado:
		t.Fatalf("Nuevo limpió durante un volcado: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(libre)
	if err := <-escrituraTerminada; err != nil {
		t.Fatal(err)
	}
	if err := <-arranqueTerminado; err != nil {
		t.Fatal(err)
	}
}

func TestAutorizacionCaducadaNoLeeContenido(t *testing.T) {
	a, err := Nuevo(Configuracion{Directorio: directorioPrueba(t), TamanoMaximo: 1 << 20,
		RetencionMinimaAdmitida: time.Hour}, relojFijo{instante().Add(2 * time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Cerrar() })
	s := escritura(t, "caducada", "clave:caducada", ports.ZonaAlmacenAdmitida, []byte("contenido"))
	s.Contenido = lectorQueFalla{}
	if _, err := a.Escribir(context.Background(), s); err == nil {
		t.Fatal("autorización caducada aceptada")
	}
}

// relojMovil devuelve el instante vigente hasta que la lectura del contenido
// lo adelanta más allá de la vigencia de la concesión.
type relojMovil struct {
	mu sync.Mutex
	t  time.Time
}

func (r *relojMovil) Ahora() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.t
}

func (r *relojMovil) fijar(t time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.t = t
}

type lectorQueCaduca struct {
	origen io.Reader
	reloj  *relojMovil
	unaVez sync.Once
}

func (l *lectorQueCaduca) Read(p []byte) (int, error) {
	l.unaVez.Do(func() { l.reloj.fijar(instante().Add(2 * time.Hour)) })
	return l.origen.Read(p)
}

// La concesión vence mientras se vuelca el contenido: la comprobación
// posterior al volcado debe rechazar la escritura sin dejar rastro.
func TestAutorizacionCaducaDuranteElVolcado(t *testing.T) {
	dir := directorioPrueba(t)
	reloj := &relojMovil{t: instante()}
	a, err := Nuevo(Configuracion{Directorio: dir, TamanoMaximo: 1 << 20,
		RetencionMinimaAdmitida: time.Hour}, reloj)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Cerrar() })
	contenido := []byte("contenido que caduca")
	s := escritura(t, "caduca", "clave:caduca", ports.ZonaAlmacenAdmitida, contenido)
	lector := &lectorQueCaduca{origen: bytes.NewReader(contenido), reloj: reloj}
	s.Contenido = lector
	if _, err := a.Escribir(context.Background(), s); err == nil {
		t.Fatal("escritura aceptada con la concesión vencida durante el volcado")
	}
	if reloj.Ahora().Equal(instante()) {
		t.Fatal("el contenido no llegó a leerse")
	}
	for _, sub := range []string{dirTemporal, dirObjetos, dirIdempotencia} {
		entradas, err := os.ReadDir(filepath.Join(dir, sub))
		if err != nil {
			t.Fatal(err)
		}
		if len(entradas) != 0 {
			t.Fatalf("%s conserva %d entradas tras el rechazo", sub, len(entradas))
		}
	}
}

type lectorQueFalla struct{}

func (lectorQueFalla) Read([]byte) (int, error) { panic("contenido leído sin autorización vigente") }

func TestEscribirLeerVerificaHuellaPermisosYReinicio(t *testing.T) {
	dir := directorioPrueba(t)
	a := nuevoPrueba(t, dir)
	ctx := context.Background()
	original := []byte("%PDF-1.7\noriginal sintético")
	resultado, err := a.Escribir(ctx, escritura(t, "a", "clave:escritura:a", ports.ZonaAlmacenAdmitida, original))
	if err != nil {
		t.Fatal(err)
	}
	if resultado.Objeto.RetenidoHasta != instante().Add(24*time.Hour) || resultado.Evidencia.ReintentoIdempotente ||
		!strings.HasPrefix(resultado.Objeto.Objeto.Referencia, "obj_") {
		t.Fatalf("resultado inesperado: %+v", resultado.Objeto)
	}
	for _, ruta := range []string{
		filepath.Join(dir, dirObjetos, resultado.Objeto.Objeto.Referencia),
		filepath.Join(dir, dirObjetos, resultado.Objeto.Objeto.Referencia+".json"),
	} {
		info, err := os.Stat(ruta)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("%s: permisos %v %v", ruta, info, err)
		}
	}
	for _, sub := range []string{dirObjetos, dirIdempotencia, dirTemporal} {
		info, err := os.Stat(filepath.Join(dir, sub))
		if err != nil || info.Mode().Perm() != 0o700 {
			t.Fatalf("%s: permisos %v", sub, err)
		}
	}
	if restos, _ := os.ReadDir(filepath.Join(dir, dirTemporal)); len(restos) != 0 {
		t.Fatalf("temporales residuales: %d", len(restos))
	}

	// Un proceso nuevo sobre el mismo directorio lee el mismo original.
	if err := a.Cerrar(); err != nil {
		t.Fatal(err)
	}
	b := nuevoPrueba(t, dir)
	abrir := ports.SolicitudAbrirObjeto{Contexto: contexto(t, "leer", ports.AccionAlmacenLeer, resultado.Objeto.Objeto),
		Objeto: resultado.Objeto.Objeto, Zona: ports.ZonaAlmacenAdmitida, Limite: int64(len(original))}
	lectura, err := b.Abrir(ctx, abrir)
	if err != nil {
		t.Fatal(err)
	}
	leido, _ := io.ReadAll(lectura.Contenido)
	if !bytes.Equal(leido, original) || lectura.Objeto.HuellaSHA256 != resultado.Objeto.HuellaSHA256 {
		t.Fatal("lectura distinta del original")
	}

	// Límite de lectura inferior al tamaño.
	corto := abrir
	corto.Limite = 3
	if _, err := b.Abrir(ctx, corto); !errors.Is(err, ports.ErrLimiteObjetoAlmacenExcedido) {
		t.Fatalf("límite: %v", err)
	}
	// Alteración del contenido en disco: la huella lo detecta.
	ruta := filepath.Join(dir, dirObjetos, resultado.Objeto.Objeto.Referencia)
	alterado := append([]byte(nil), original...)
	alterado[0] = 'X'
	if err := os.WriteFile(ruta, alterado, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Abrir(ctx, abrir); !errors.Is(err, ports.ErrIntegridadObjetoAlmacen) {
		t.Fatalf("alteración no detectada: %v", err)
	}
	// Un objeto que no existe no revela rutas.
	ajeno := ports.ReferenciaObjetoAlmacen{Referencia: "obj_" + strings.Repeat("0", 32), Version: "1"}
	if _, err := b.Abrir(ctx, ports.SolicitudAbrirObjeto{Contexto: contexto(t, "ajeno", ports.AccionAlmacenLeer, ajeno),
		Objeto: ajeno, Zona: ports.ZonaAlmacenAdmitida, Limite: 10}); !errors.Is(err, ports.ErrObjetoAlmacenNoEncontrado) {
		t.Fatalf("ajeno: %v", err)
	}
	traversal := ports.ReferenciaObjetoAlmacen{Referencia: "../../etc/passwd", Version: "1"}
	if c, err := pruebasvec.NuevoContextoAlmacen(instante(), "tr", ports.AccionAlmacenLeer, traversal); err == nil {
		if _, err := b.Abrir(ctx, ports.SolicitudAbrirObjeto{Contexto: c, Objeto: traversal, Zona: ports.ZonaAlmacenAdmitida, Limite: 10}); err == nil {
			t.Fatal("referencia con ruta aceptada")
		}
	}
}

func TestEscribirIdempotenteYRechazos(t *testing.T) {
	a := nuevoPrueba(t, directorioPrueba(t))
	ctx := context.Background()
	contenido := []byte("%PDF-1.7\nidempotente")
	primera, err := a.Escribir(ctx, escritura(t, "i", "clave:idem", ports.ZonaAlmacenAdmitida, contenido))
	if err != nil {
		t.Fatal(err)
	}
	segunda, err := a.Escribir(ctx, escritura(t, "i", "clave:idem", ports.ZonaAlmacenAdmitida, contenido))
	if err != nil || segunda.Objeto.Objeto != primera.Objeto.Objeto || !segunda.Evidencia.ReintentoIdempotente {
		t.Fatalf("reintento: %v %+v", err, segunda.Objeto)
	}
	if _, err := a.Escribir(ctx, escritura(t, "i", "clave:idem", ports.ZonaAlmacenAdmitida, []byte("%PDF-1.7\notro"))); !errors.Is(err, ports.ErrIdempotenciaAlmacenReutilizada) {
		t.Fatalf("clave reutilizada: %v", err)
	}
	falsa := escritura(t, "f", "clave:falsa", ports.ZonaAlmacenAdmitida, contenido)
	falsa.Contenido = bytes.NewReader([]byte("%PDF-1.7\nidempotentX"))
	if _, err := a.Escribir(ctx, falsa); !errors.Is(err, ports.ErrIntegridadObjetoAlmacen) {
		t.Fatalf("huella falsa: %v", err)
	}
	grande := escritura(t, "g", "clave:grande", ports.ZonaAlmacenAdmitida, make([]byte, 2<<20))
	if _, err := a.Escribir(ctx, grande); !errors.Is(err, ports.ErrLimiteObjetoAlmacenExcedido) {
		t.Fatalf("tamaño: %v", err)
	}
	cancelado, cancelar := context.WithCancel(ctx)
	cancelar()
	if _, err := a.Escribir(cancelado, escritura(t, "c", "clave:cancelada", ports.ZonaAlmacenAdmitida, contenido)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelación: %v", err)
	}
}

func TestPromocionYRetencion(t *testing.T) {
	a := nuevoPrueba(t, directorioPrueba(t))
	ctx := context.Background()
	contenido := []byte("%PDF-1.7\ncuarentena")
	cuarentena, err := a.Escribir(ctx, escritura(t, "q", "clave:cuarentena", ports.ZonaAlmacenCuarentena, contenido))
	if err != nil || !cuarentena.Objeto.RetenidoHasta.IsZero() {
		t.Fatalf("cuarentena: %v %+v", err, cuarentena.Objeto)
	}
	promover := ports.SolicitudPromoverObjeto{Contexto: contexto(t, "p", ports.AccionAlmacenPromover, cuarentena.Objeto.Objeto),
		ClaveIdempotencia: "clave:promocion", Origen: cuarentena.Objeto.Objeto, EvidenciaAnalisisRef: "analisis:limpio:1"}
	admitido, err := a.Promover(ctx, promover)
	if err != nil || admitido.Objeto.Zona != ports.ZonaAlmacenAdmitida || admitido.Objeto.Objeto == cuarentena.Objeto.Objeto ||
		admitido.Objeto.HuellaSHA256 != cuarentena.Objeto.HuellaSHA256 {
		t.Fatalf("promoción: %v %+v", err, admitido.Objeto)
	}
	repetida, err := a.Promover(ctx, promover)
	if err != nil || repetida.Objeto.Objeto != admitido.Objeto.Objeto || !repetida.Evidencia.ReintentoIdempotente {
		t.Fatalf("promoción repetida: %v", err)
	}
	hasta := instante().Add(10 * 365 * 24 * time.Hour)
	retener := ports.SolicitudRetenerObjeto{Contexto: contexto(t, "r", ports.AccionAlmacenAplicarRetencion, admitido.Objeto.Objeto),
		Objeto: admitido.Objeto.Objeto, PoliticaRef: "politica:conservacion:1", Hasta: hasta}
	retenido, err := a.AplicarRetencion(ctx, retener)
	if err != nil || !retenido.Objeto.RetenidoHasta.Equal(hasta) {
		t.Fatalf("retención: %v", err)
	}
	acortar := retener
	acortar.Contexto = contexto(t, "r2", ports.AccionAlmacenAplicarRetencion, admitido.Objeto.Objeto)
	acortar.Hasta = instante().Add(48 * time.Hour)
	if _, err := a.AplicarRetencion(ctx, acortar); !errors.Is(err, ports.ErrRetencionObjetoAlmacenVigente) {
		t.Fatalf("retención acortada: %v", err)
	}
}

func TestEscriturasConcurrentes(t *testing.T) {
	a := nuevoPrueba(t, directorioPrueba(t))
	var wg sync.WaitGroup
	errores := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			contenido := []byte(fmt.Sprintf("%%PDF-1.7\nconcurrente %d", i))
			if _, err := a.Escribir(context.Background(), escritura(t, fmt.Sprintf("c%d", i), fmt.Sprintf("clave:concurrente:%d", i), ports.ZonaAlmacenAdmitida, contenido)); err != nil {
				errores <- err
			}
		}(i)
	}
	wg.Wait()
	close(errores)
	for err := range errores {
		t.Fatal(err)
	}
	if a.String() != "almacen-ficheros[ficheros-local]" {
		t.Fatalf("String revela datos: %s", a.String())
	}
}

// Con retención mínima cero el almacén no fija retención al escribir ni al
// promover (política de conservación provisional); se aplica después.
func TestSinRetencionAlEscribirLaAplicaDespues(t *testing.T) {
	a, err := Nuevo(Configuracion{Directorio: directorioPrueba(t), TamanoMaximo: 1 << 20}, relojFijo{instante()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Cerrar() })
	if a.RetencionAlEscribir() || !nuevoPrueba(t, directorioPrueba(t)).RetencionAlEscribir() {
		t.Fatal("declaración de retención al escribir incoherente")
	}
	ctx := context.Background()
	caps, err := a.Capacidades(ctx)
	if err != nil || !caps.Retencion || caps.RetencionAtomicaEnPromocion {
		t.Fatalf("capacidades sin retención al escribir: %v %+v", err, caps)
	}
	admitido, err := a.Escribir(ctx, escritura(t, "sin", "clave:sin:retencion", ports.ZonaAlmacenAdmitida, []byte("%PDF-1.7\nsin retención")))
	if err != nil || !admitido.Objeto.RetenidoHasta.IsZero() {
		t.Fatalf("escritura sin retención: %v %+v", err, admitido.Objeto)
	}
	cuarentena, err := a.Escribir(ctx, escritura(t, "q", "clave:sin:cuarentena", ports.ZonaAlmacenCuarentena, []byte("%PDF-1.7\ncuarentena")))
	if err != nil {
		t.Fatal(err)
	}
	promovido, err := a.Promover(ctx, ports.SolicitudPromoverObjeto{Contexto: contexto(t, "p", ports.AccionAlmacenPromover, cuarentena.Objeto.Objeto),
		ClaveIdempotencia: "clave:sin:promocion", Origen: cuarentena.Objeto.Objeto, EvidenciaAnalisisRef: "analisis:limpio:1"})
	if err != nil || !promovido.Objeto.RetenidoHasta.IsZero() {
		t.Fatalf("promoción sin retención: %v %+v", err, promovido.Objeto)
	}
	hasta := instante().Add(6 * 365 * 24 * time.Hour)
	retenido, err := a.AplicarRetencion(ctx, ports.SolicitudRetenerObjeto{Contexto: contexto(t, "r", ports.AccionAlmacenAplicarRetencion, admitido.Objeto.Objeto),
		Objeto: admitido.Objeto.Objeto, PoliticaRef: "politica:conservacion:definitiva", Hasta: hasta})
	if err != nil || !retenido.Objeto.RetenidoHasta.Equal(hasta) {
		t.Fatalf("retención posterior: %v", err)
	}
}
