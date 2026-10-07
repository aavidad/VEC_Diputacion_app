package observabilidad

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/catalogoincidencias"
	"vec-diputacion-granada/internal/vec/domain"
)

func configRecolectorPrueba(t *testing.T) ConfiguracionRecolector {
	t.Helper()
	directorio := t.TempDir()
	if err := os.Chmod(directorio, 0700); err != nil {
		t.Fatal(err)
	}
	return ConfiguracionRecolector{Directorio: directorio, MaxLineaBytes: 512, MaxArchivoBytes: 513, MaxArchivos: 3, RetencionSegundos: 3600, VentanaSegundos: 60, Umbrales: map[domain.CodigoIncidenciaTecnica]uint64{domain.IncidenciaArranqueFallido: 2}}
}

func lineaRecolectorPrueba(t *testing.T) []byte {
	t.Helper()
	clasificacion, _ := domain.ClasificarIncidenciaTecnica(domain.SolicitudIncidenciaTecnica{Codigo: domain.IncidenciaArranqueFallido, Componente: domain.ComponenteIncidenciaServidor, Etapa: domain.EtapaIncidenciaEscucha})
	inc := domain.NuevaIncidenciaTecnica(clasificacion, time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), domain.EntornoIncidenciaPruebas, "936aac665", "0123456789abcdef0123456789abcdef")
	catalogo, err := catalogoincidencias.Predeterminado()
	if err != nil {
		t.Fatal(err)
	}
	inc.Mensaje, _ = catalogo.Plantilla(inc.Codigo)
	b, err := json.Marshal(lineaIncidencia{inc.Esquema, inc.Instante.Format(formatoInstante), string(inc.Codigo), string(inc.Severidad), string(inc.Componente), string(inc.Etapa), string(inc.Entorno), inc.VersionBinario, inc.Correlacion, inc.Recuento, inc.Mensaje})
	if err != nil || len(b) > 512 {
		t.Fatal("fixture de incidencia no conforme", err, len(b))
	}
	return append(b, '\n')
}

func TestRecolectorDescartaDatosAjenosYSigueTrasLineaLarga(t *testing.T) {
	cfg := configRecolectorPrueba(t)
	valida := lineaRecolectorPrueba(t)
	var entrada bytes.Buffer
	entrada.WriteString(strings.Repeat("dato-personal-sintetico ", 1000) + "\n")
	entrada.Write(bytes.Replace(valida, []byte(`"codigo":`), []byte(`"secreto":"dato-personal-sintetico","codigo":`), 1))
	entrada.Write(bytes.Replace(valida, []byte(`"critica"`), []byte(`"error"`), 1))
	entrada.Write(bytes.Replace(valida, []byte(`El servidor no ha podido arrancar.`), []byte(`dato-personal-sintetico`), 1))
	entrada.Write(valida)
	entrada.Write(valida)
	var alertas bytes.Buffer
	m, err := RecolectarIncidencias(&entrada, &alertas, cfg)
	if err != nil || m.Recibidas != 6 || m.Rechazadas != 4 || m.Escritas != 2 || m.Alertas != 1 {
		t.Fatalf("metricas %+v, error %v", m, err)
	}
	archivos, err := os.ReadDir(cfg.Directorio)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range archivos {
		b, err := os.ReadFile(filepath.Join(cfg.Directorio, f.Name()))
		if err != nil || bytes.Contains(b, []byte("dato-personal-sintetico")) {
			t.Fatal("entrada rechazada conservada", err)
		}
		if !strings.HasSuffix(f.Name(), ".jsonl") {
			continue
		}
		if !bytes.Equal(b, valida) {
			t.Fatalf("salida sin proyeccion canonica: %s", b)
		}
	}
	if strings.Contains(alertas.String(), "dato-personal-sintetico") || !strings.Contains(alertas.String(), "ARRANQUE_FALLIDO") {
		t.Fatal("alerta no cerrada")
	}
}

func TestRecolectorRotacionRetencionYRecuperacion(t *testing.T) {
	cfg := configRecolectorPrueba(t)
	valida := lineaRecolectorPrueba(t)
	m, err := RecolectarIncidencias(bytes.NewReader(bytes.Repeat(valida, 5)), new(bytes.Buffer), cfg)
	if err != nil || m.Escritas != 5 || m.Retirados != 2 {
		t.Fatalf("primera pasada %+v %v", m, err)
	}
	m, err = RecolectarIncidencias(bytes.NewReader(valida), new(bytes.Buffer), cfg)
	if err != nil || m.Escritas != 1 || m.Retirados != 1 {
		t.Fatalf("recuperacion %+v %v", m, err)
	}
	archivos, _ := os.ReadDir(cfg.Directorio)
	var retenidos int
	for _, f := range archivos {
		if !strings.HasSuffix(f.Name(), ".jsonl") {
			continue
		}
		info, _ := f.Info()
		if info.Size() > cfg.MaxArchivoBytes || info.Mode().Perm() != 0600 {
			t.Fatal("archivo sin limites o privacidad")
		}
		retenidos++
	}
	if retenidos != cfg.MaxArchivos {
		t.Fatal("retencion por cantidad", retenidos)
	}
	antiguo := filepath.Join(cfg.Directorio, "incidencias-00000000000000000004.jsonl")
	fecha := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(antiguo, fecha, fecha); err != nil {
		t.Fatal(err)
	}
	m, err = RecolectarIncidencias(strings.NewReader(""), new(bytes.Buffer), cfg)
	if err != nil || m.Retirados != 1 {
		t.Fatalf("retencion temporal %+v %v", m, err)
	}
}

func TestRecolectorRespetaArchivosAjenosYRechazaEnlaces(t *testing.T) {
	for _, enlace := range []string{"simbolico", "duro"} {
		t.Run(enlace, func(t *testing.T) {
			cfg := configRecolectorPrueba(t)
			ajeno := filepath.Join(t.TempDir(), "ajeno")
			if err := os.WriteFile(ajeno, []byte("intacto"), 0600); err != nil {
				t.Fatal(err)
			}
			activo := filepath.Join(cfg.Directorio, archivoActivoRecolector)
			var err error
			if enlace == "simbolico" {
				err = os.Symlink(ajeno, activo)
			} else {
				err = os.Link(ajeno, activo)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := RecolectarIncidencias(bytes.NewReader(lineaRecolectorPrueba(t)), new(bytes.Buffer), cfg); err == nil {
				t.Fatal("enlace aceptado")
			}
			b, err := os.ReadFile(ajeno)
			if err != nil || string(b) != "intacto" {
				t.Fatal("archivo ajeno alterado")
			}
		})
	}
	truncado := configRecolectorPrueba(t)
	if err := os.WriteFile(filepath.Join(truncado.Directorio, archivoActivoRecolector), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := RecolectarIncidencias(strings.NewReader(""), new(bytes.Buffer), truncado); err == nil {
		t.Fatal("archivo truncado aceptado")
	}
}

func TestRecolectorBloqueaSegundoProcesoYUmbralesPorVentana(t *testing.T) {
	cfg := configRecolectorPrueba(t)
	a, err := abrirAlmacenRecolector(cfg, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	defer a.cerrar()
	if b, err := abrirAlmacenRecolector(cfg, time.Now); err == nil {
		_ = b.cerrar()
		t.Fatal("segundo propietario aceptado")
	}
	l, err := validarLineaRecolector(lineaRecolectorPrueba(t))
	if err != nil {
		t.Fatal("fixture invalida")
	}
	inicio := time.Now()
	c := contadorAlertas{cfg: cfg, inicio: inicio, cantidades: make(map[domain.CodigoIncidenciaTecnica]uint64), avisados: make(map[domain.CodigoIncidenciaTecnica]bool)}
	var salida bytes.Buffer
	var m MetricasRecolector
	for i := 0; i < 3; i++ {
		if err := c.registrar(l, inicio, &salida, &m); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		if err := c.registrar(l, inicio.Add(time.Minute), &salida, &m); err != nil {
			t.Fatal(err)
		}
	}
	if m.Alertas != 2 {
		t.Fatal("alertas por ventana", m)
	}
}

func TestRecolectorRechazaClavesAmbiguas(t *testing.T) {
	valida := lineaRecolectorPrueba(t)
	for _, entrada := range [][]byte{
		bytes.Replace(valida, []byte(`"codigo":`), []byte(`"codigo":"dato-personal-sintetico","codigo":`), 1),
		bytes.Replace(valida, []byte(`"codigo":`), []byte(`"CODIGO":`), 1),
		bytes.Replace(valida, []byte(`"codigo":`), []byte(`"CODIGO":"dato-personal-sintetico","codigo":`), 1),
	} {
		if _, err := validarLineaRecolector(entrada); err == nil {
			t.Fatal("entrada ambigua aceptada")
		}
	}
}
