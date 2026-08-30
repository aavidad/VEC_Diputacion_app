package ginpixfichero

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestExportadorAtomicoEscribeBytesExactos0600YComprobanteSoloLocal(t *testing.T) {
	directorio := t.TempDir()
	exportador := nuevoExportadorAtomicoPrueba(t, directorio)
	preparacion := preparacionExportacionPrueba(t)
	esperado, _ := preparacion.Contenido()
	huellaEsperada, _ := preparacion.HuellaSHA256()
	metadatosEsperados, _ := preparacion.Metadatos()

	comprobante, err := exportador.Exportar(context.Background(), preparacion)
	if err != nil || comprobante.Validar() != nil {
		t.Fatalf("exportar preparacion sintetica: %v", err)
	}
	ruta, _ := comprobante.RutaRelativa()
	if !filepath.IsLocal(ruta) || filepath.Base(ruta) != ruta ||
		strings.Contains(ruta, metadatosEsperados.ExpedienteRef) ||
		strings.ContainsAny(ruta, `/\\`) {
		t.Fatalf("ruta relativa no segura: %q", ruta)
	}
	contenido, err := os.ReadFile(filepath.Join(directorio, ruta))
	if err != nil || !bytes.Equal(contenido, esperado) {
		t.Fatalf("bytes publicados divergentes: %v", err)
	}
	informacion, err := os.Stat(filepath.Join(directorio, ruta))
	if err != nil || !informacion.Mode().IsRegular() || informacion.Mode() != 0o600 {
		t.Fatalf("metadatos de fichero incompatibles: %+v / %v", informacion, err)
	}

	alcance, _ := comprobante.Alcance()
	huella, _ := comprobante.HuellaSHA256()
	metadatos, _ := comprobante.Metadatos()
	tamano, _ := comprobante.TamanoBytes()
	permisos, _ := comprobante.Permisos()
	replay, _ := comprobante.EsReplayLocal()
	if alcance != AlcanceComprobanteExportacionLocal || huella != huellaEsperada ||
		metadatos != metadatosEsperados || tamano != int64(len(esperado)) ||
		permisos != 0o600 || replay {
		t.Fatalf("comprobante local divergente: alcance=%q replay=%v", alcance, replay)
	}
	// LOCAL es el unico alcance acreditado: no hay firma, entrega, acuse ni
	// confirmacion de GINPIX inferibles de este comprobante.
	if alcance == "FIRMA" || alcance == "ENTREGA" || alcance == "ACUSE" ||
		alcance == "CONFIRMACION_GINPIX" {
		t.Fatalf("el comprobante local atribuyo un efecto externo: %q", alcance)
	}
	metadatos.ExpedienteRef = "mutado"
	posteriores, _ := comprobante.Metadatos()
	if posteriores != metadatosEsperados {
		t.Fatal("el getter de metadatos altero el comprobante inmutable")
	}
}

func TestExportadorAtomicoReplayExactoNoReescribe(t *testing.T) {
	directorio := t.TempDir()
	exportador := nuevoExportadorAtomicoPrueba(t, directorio)
	preparacion := preparacionExportacionPrueba(t)

	primero, err := exportador.Exportar(context.Background(), preparacion)
	if err != nil {
		t.Fatalf("primera exportacion: %v", err)
	}
	ruta, _ := primero.RutaRelativa()
	antes, err := os.Stat(filepath.Join(directorio, ruta))
	if err != nil {
		t.Fatalf("stat inicial: %v", err)
	}
	segundo, err := exportador.Exportar(context.Background(), preparacion)
	if err != nil || segundo.Validar() != nil {
		t.Fatalf("replay local exacto: %v", err)
	}
	despues, err := os.Stat(filepath.Join(directorio, ruta))
	if err != nil {
		t.Fatalf("stat posterior: %v", err)
	}
	replayPrimero, _ := primero.EsReplayLocal()
	replaySegundo, _ := segundo.EsReplayLocal()
	rutaSegundo, _ := segundo.RutaRelativa()
	huellaPrimera, _ := primero.HuellaSHA256()
	huellaSegunda, _ := segundo.HuellaSHA256()
	if replayPrimero || !replaySegundo || rutaSegundo != ruta ||
		huellaSegunda != huellaPrimera || !antes.ModTime().Equal(despues.ModTime()) {
		t.Fatal("el replay no conservo identidad y fichero exactos")
	}
	assertEntradasExportacion(t, directorio, ruta)
}

func TestExportadorAtomicoConflictoYMetadatosNoSobrescribenYLimpian(t *testing.T) {
	t.Run("contenido distinto", func(t *testing.T) {
		directorio := t.TempDir()
		exportador := nuevoExportadorAtomicoPrueba(t, directorio)
		preparacion := preparacionExportacionPrueba(t)
		comprobante, err := exportador.Exportar(context.Background(), preparacion)
		if err != nil {
			t.Fatalf("preparar conflicto: %v", err)
		}
		ruta, _ := comprobante.RutaRelativa()
		conflicto, _ := preparacion.Contenido()
		conflicto[len(conflicto)/2] ^= 1
		if err := os.WriteFile(filepath.Join(directorio, ruta), conflicto, 0o600); err != nil {
			t.Fatalf("instalar conflicto: %v", err)
		}
		fallido, err := exportador.Exportar(context.Background(), preparacion)
		if fallido != (ComprobanteExportacionLocal{}) ||
			!errors.Is(err, ErrExportacionLocalGINPIX) {
			t.Fatalf("conflicto aceptado: %+v / %v", fallido, err)
		}
		posterior, _ := os.ReadFile(filepath.Join(directorio, ruta))
		if !bytes.Equal(posterior, conflicto) {
			t.Fatal("el conflicto fue sobrescrito")
		}
		assertEntradasExportacion(t, directorio, ruta)
	})

	t.Run("permisos incompatibles", func(t *testing.T) {
		directorio := t.TempDir()
		exportador := nuevoExportadorAtomicoPrueba(t, directorio)
		preparacion := preparacionExportacionPrueba(t)
		comprobante, err := exportador.Exportar(context.Background(), preparacion)
		if err != nil {
			t.Fatalf("preparar metadatos: %v", err)
		}
		ruta, _ := comprobante.RutaRelativa()
		if err := os.Chmod(filepath.Join(directorio, ruta), 0o640); err != nil {
			t.Fatalf("alterar permisos: %v", err)
		}
		if _, err := exportador.Exportar(context.Background(), preparacion); !errors.Is(
			err,
			ErrExportacionLocalGINPIX,
		) {
			t.Fatalf("metadatos incompatibles aceptados: %v", err)
		}
		assertEntradasExportacion(t, directorio, ruta)
	})
}

func TestExportadorAtomicoRechazaSymlinkYTipoIrregular(t *testing.T) {
	casos := []struct {
		nombre  string
		crear   func(*testing.T, string, string)
		validar func(*testing.T, string)
	}{
		{
			nombre: "symlink",
			crear: func(t *testing.T, directorio, ruta string) {
				t.Helper()
				objetivo := filepath.Join(t.TempDir(), "objetivo.json")
				if err := os.WriteFile(objetivo, []byte("SINTETICO_NO_SECRETO_INTACTO"), 0o600); err != nil {
					t.Fatalf("crear objetivo: %v", err)
				}
				if err := os.Symlink(objetivo, filepath.Join(directorio, ruta)); err != nil {
					t.Fatalf("crear symlink: %v", err)
				}
			},
			validar: func(t *testing.T, ruta string) {
				t.Helper()
				informacion, err := os.Lstat(ruta)
				if err != nil || informacion.Mode()&os.ModeSymlink == 0 {
					t.Fatalf("el symlink fue seguido o sustituido: %v", err)
				}
			},
		},
		{
			nombre: "directorio",
			crear: func(t *testing.T, directorio, ruta string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(directorio, ruta), 0o700); err != nil {
					t.Fatalf("crear tipo irregular: %v", err)
				}
			},
			validar: func(t *testing.T, ruta string) {
				t.Helper()
				informacion, err := os.Stat(ruta)
				if err != nil || !informacion.IsDir() {
					t.Fatalf("el tipo irregular fue sustituido: %v", err)
				}
			},
		},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			directorio := t.TempDir()
			exportador := nuevoExportadorAtomicoPrueba(t, directorio)
			preparacion := preparacionExportacionPrueba(t)
			metadatos, _ := preparacion.Metadatos()
			ruta := nombreFinalExportacion(metadatos)
			caso.crear(t, directorio, ruta)
			if _, err := exportador.Exportar(context.Background(), preparacion); !errors.Is(
				err,
				ErrExportacionLocalGINPIX,
			) {
				t.Fatalf("objetivo hostil aceptado: %v", err)
			}
			caso.validar(t, filepath.Join(directorio, ruta))
			assertEntradasExportacion(t, directorio, ruta)
		})
	}
}

func TestExportadorAtomicoCancelacionAntesDePublicarLimpia(t *testing.T) {
	directorio := t.TempDir()
	exportador := nuevoExportadorAtomicoPrueba(t, directorio)
	ctx := nuevoContextoCancelacionEscalonada(5)
	comprobante, err := exportador.Exportar(ctx, preparacionExportacionPrueba(t))
	if comprobante != (ComprobanteExportacionLocal{}) ||
		!errors.Is(err, ErrExportacionLocalGINPIX) {
		t.Fatalf("cancelacion aceptada: %+v / %v", comprobante, err)
	}
	assertEntradasExportacion(t, directorio)
}

func TestExportadorAtomicoCancelacionDespuesDeLinkConReplaysConservaFinal(t *testing.T) {
	directorio := t.TempDir()
	exportador := nuevoExportadorAtomicoPrueba(t, directorio)
	preparacion := preparacionExportacionPrueba(t)
	ctx := nuevoContextoCancelacionEscalonada(6)

	primero, err := exportador.Exportar(ctx, preparacion)
	if err != nil || primero.Validar() != nil {
		t.Fatalf("publicacion antes de cancelar: %v", err)
	}
	// La sexta consulta era la comprobacion posterior a Link. Exportar ya no
	// debe realizarla: el test cancela aqui, despues del punto de commit.
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Fatal("el contexto no se cancelo despues de publicar")
	}
	ruta, _ := primero.RutaRelativa()

	const concurrentes = 16
	resultados := make(chan error, concurrentes)
	var grupo sync.WaitGroup
	for indice := 0; indice < concurrentes; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			comprobante, err := exportador.Exportar(context.Background(), preparacion)
			if err != nil || comprobante.Validar() != nil {
				resultados <- ErrExportacionLocalGINPIX
				return
			}
			esReplay, _ := comprobante.EsReplayLocal()
			rutaReplay, _ := comprobante.RutaRelativa()
			if !esReplay || rutaReplay != ruta {
				resultados <- ErrExportacionLocalGINPIX
				return
			}
			resultados <- nil
		}()
	}
	grupo.Wait()
	close(resultados)
	for err := range resultados {
		if err != nil {
			t.Fatal("replay concurrente posterior a cancelacion fallido")
		}
	}

	esperado, _ := preparacion.Contenido()
	contenido, err := os.ReadFile(filepath.Join(directorio, ruta))
	if err != nil || !bytes.Equal(contenido, esperado) {
		t.Fatalf("la cancelacion posterior retiro o altero el final: %v", err)
	}
	assertEntradasExportacion(t, directorio, ruta)
}

func TestExportadorAtomicoConcurrenteDejaUnFinalCoherente(t *testing.T) {
	directorio := t.TempDir()
	exportador := nuevoExportadorAtomicoPrueba(t, directorio)
	preparacion := preparacionExportacionPrueba(t)
	const concurrentes = 24
	type resultado struct {
		comprobante ComprobanteExportacionLocal
		err         error
	}
	resultados := make(chan resultado, concurrentes)
	inicio := make(chan struct{})
	var grupo sync.WaitGroup
	for indice := 0; indice < concurrentes; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			<-inicio
			comprobante, err := exportador.Exportar(context.Background(), preparacion)
			resultados <- resultado{comprobante: comprobante, err: err}
		}()
	}
	close(inicio)
	grupo.Wait()
	close(resultados)

	nuevos := 0
	replays := 0
	var ruta string
	for resultado := range resultados {
		if resultado.err != nil || resultado.comprobante.Validar() != nil {
			t.Fatalf("exportacion concurrente fallida: %v", resultado.err)
		}
		esReplay, _ := resultado.comprobante.EsReplayLocal()
		rutaActual, _ := resultado.comprobante.RutaRelativa()
		if ruta == "" {
			ruta = rutaActual
		}
		if rutaActual != ruta {
			t.Fatalf("rutas concurrentes divergentes: %q / %q", ruta, rutaActual)
		}
		if esReplay {
			replays++
		} else {
			nuevos++
		}
	}
	if nuevos != 1 || replays != concurrentes-1 {
		t.Fatalf("resultados concurrentes: nuevos=%d replays=%d", nuevos, replays)
	}
	esperado, _ := preparacion.Contenido()
	contenido, err := os.ReadFile(filepath.Join(directorio, ruta))
	if err != nil || !bytes.Equal(contenido, esperado) {
		t.Fatalf("final concurrente incoherente: %v", err)
	}
	assertEntradasExportacion(t, directorio, ruta)
}

func TestNuevoExportadorAtomicoExigeDirectorioExistenteNoSymlink(t *testing.T) {
	base := t.TempDir()
	inexistente := filepath.Join(base, "ausente")
	if _, err := NuevoExportadorAtomico(inexistente); !errors.Is(err, ErrExportacionLocalGINPIX) {
		t.Fatalf("directorio ausente aceptado: %v", err)
	}
	if _, err := os.Stat(inexistente); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("el constructor creo el directorio: %v", err)
	}
	fichero := filepath.Join(base, "fichero")
	if err := os.WriteFile(fichero, []byte("SINTETICO_NO_SECRETO"), 0o600); err != nil {
		t.Fatalf("crear fichero: %v", err)
	}
	if _, err := NuevoExportadorAtomico(fichero); !errors.Is(err, ErrExportacionLocalGINPIX) {
		t.Fatalf("fichero aceptado como directorio: %v", err)
	}
	symlink := filepath.Join(base, "enlace")
	if err := os.Symlink(base, symlink); err != nil {
		t.Fatalf("crear symlink de directorio: %v", err)
	}
	if _, err := NuevoExportadorAtomico(symlink); !errors.Is(err, ErrExportacionLocalGINPIX) {
		t.Fatalf("symlink aceptado como directorio configurado: %v", err)
	}
}

func TestAbrirRaizInspeccionadaRechazaDirectorioDistinto(t *testing.T) {
	directorioInspeccionado := t.TempDir()
	informacion, err := os.Lstat(directorioInspeccionado)
	if err != nil {
		t.Fatalf("inspeccionar directorio inicial: %v", err)
	}
	directorioSustituto := t.TempDir()
	raiz, err := abrirRaizInspeccionada(directorioSustituto, informacion)
	if raiz != nil {
		_ = raiz.Close()
		t.Fatal("la raiz abierta no corresponde al directorio inspeccionado")
	}
	if !errors.Is(err, ErrExportacionLocalGINPIX) {
		t.Fatalf("discrepancia de raiz aceptada: %v", err)
	}
}

func preparacionExportacionPrueba(t *testing.T) PreparacionExportacion {
	t.Helper()
	preparacion, err := PrepararExportacion(cargaGINPIXFicheroPrueba(t, false))
	if err != nil {
		t.Fatalf("preparar exportacion sintetica: %v", err)
	}
	return preparacion
}

func nuevoExportadorAtomicoPrueba(t *testing.T, directorio string) *ExportadorAtomico {
	t.Helper()
	exportador, err := NuevoExportadorAtomico(directorio)
	if err != nil {
		t.Fatalf("crear exportador: %v", err)
	}
	t.Cleanup(func() {
		if err := exportador.Cerrar(); err != nil {
			t.Errorf("cerrar exportador: %v", err)
		}
	})
	return exportador
}

func assertEntradasExportacion(t *testing.T, directorio string, esperadas ...string) {
	t.Helper()
	entradas, err := os.ReadDir(directorio)
	if err != nil {
		t.Fatalf("listar directorio: %v", err)
	}
	if len(entradas) != len(esperadas) {
		t.Fatalf("residuos o finales inesperados: got=%d want=%d", len(entradas), len(esperadas))
	}
	for indice, esperada := range esperadas {
		if entradas[indice].Name() != esperada ||
			strings.HasPrefix(entradas[indice].Name(), prefijoFicheroTemporal) {
			t.Fatalf("entrada inesperada: %q", entradas[indice].Name())
		}
	}
}

type contextoCancelacionEscalonada struct {
	context.Context
	cancelar   context.CancelFunc
	mu         sync.Mutex
	consultas  int
	cancelarEn int
}

func nuevoContextoCancelacionEscalonada(cancelarEn int) *contextoCancelacionEscalonada {
	ctx, cancelar := context.WithCancel(context.Background())
	return &contextoCancelacionEscalonada{
		Context:    ctx,
		cancelar:   cancelar,
		cancelarEn: cancelarEn,
	}
}

func (c *contextoCancelacionEscalonada) Err() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.consultas++
	if c.consultas == c.cancelarEn {
		c.cancelar()
	}
	return c.Context.Err()
}
