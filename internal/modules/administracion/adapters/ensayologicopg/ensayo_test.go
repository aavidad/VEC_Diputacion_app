package ensayologicopg

import (
	"context"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/adapters/ensayofisicopg"
)

func TestEntradasRechazadasAntesDeUsarPostgreSQL(t *testing.T) {
	// La imagen tiene una huella sintácticamente válida y no necesita existir:
	// todos estos casos deben terminar en la frontera de entrada, antes del SQL.
	config := Configuracion{
		ImagenSHA256:       strings.Repeat("a", 64),
		VersionPostgreSQL:  "18.4",
		UsuarioBootstrap:   "cs06_destino",
		LimiteArchivoBytes: limiteFixture,
		CPUs:               1,
		MemoriaBytes:       512 << 20,
		TiempoLimite:       30 * time.Second,
	}
	dump := archivoFixture(t, "base.dump", []byte("PGDMPfixture-sintetica"))
	globals := archivoFixture(t, "globals.sql", []byte("-- globals sinteticos\n"))
	validos := Solicitud{Sintetica: true, Dump: dump, Globals: globals}
	directorio := t.TempDir()
	fifo := filepath.Join(t.TempDir(), "entrada.fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	casos := []struct {
		nombre  string
		clave   string
		cambiar func(*Configuracion, *Solicitud)
	}{
		{"datos_no_sinteticos", "entrada", func(_ *Configuracion, s *Solicitud) { s.Sintetica = false }},
		{"imagen_sin_huella_exacta", "entrada", func(c *Configuracion, _ *Solicitud) { c.ImagenSHA256 = "postgres:18.4" }},
		{"bootstrap_generico", "entrada", func(c *Configuracion, _ *Solicitud) { c.UsuarioBootstrap = "postgres" }},
		{"dump_inexistente", "archivos", func(_ *Configuracion, s *Solicitud) { s.Dump.Ruta = filepath.Join(t.TempDir(), "ausente.dump") }},
		{"dump_huella_distinta", "archivos", func(_ *Configuracion, s *Solicitud) { s.Dump.SHA256 = strings.Repeat("b", 64) }},
		{"globals_huella_distinta", "archivos", func(_ *Configuracion, s *Solicitud) { s.Globals.SHA256 = strings.Repeat("b", 64) }},
		{"dump_tipo_no_regular", "archivos", func(_ *Configuracion, s *Solicitud) { s.Dump.Ruta = directorio }},
		{"fifo_sin_escritor_no_bloquea", "archivos", func(_ *Configuracion, s *Solicitud) { s.Dump.Ruta = fifo }},
		{"limite_archivo_excedido", "archivos", func(c *Configuracion, _ *Solicitud) { c.LimiteArchivoBytes = 8 }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			c, solicitud := config, validos
			caso.cambiar(&c, &solicitud)
			resultado := (Ensayador{Configuracion: c}).Ensayar(context.Background(), solicitud)
			if resultado.Estado != "restauracion_logica_fallida" || resultado.Etapa != "entrada" ||
				!resultado.LimpiezaCompletada || len(resultado.Razones) != 1 || resultado.Razones[0].Clave != caso.clave {
				t.Fatalf("rechazo de entrada: estado=%q etapa=%q razones=%v", resultado.Estado, resultado.Etapa, resultado.Razones)
			}
			comprobarLimitesResultado(t, resultado)
		})
	}
	t.Run("archivado_invalido", func(t *testing.T) {
		resultado := (Ensayador{
			Configuracion: config,
			Observador:    &observadorLecturaReal{},
			ComponentesArchivados: []ensayofisicopg.Componente{{
				ID: "fisica:../escape", Tipo: "documentos",
			}},
		}).Ensayar(context.Background(), validos)
		if resultado.Estado != "restauracion_logica_fallida" || resultado.Etapa != "entrada" ||
			!resultado.LimpiezaCompletada || len(resultado.Razones) != 1 || resultado.Razones[0].Clave != "archivados" {
			t.Fatalf("rechazo de archivado: estado=%q etapa=%q razones=%v", resultado.Estado, resultado.Etapa, resultado.Razones)
		}
		comprobarLimitesResultado(t, resultado)
	})
}
