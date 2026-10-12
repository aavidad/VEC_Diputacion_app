package analisiscontenido

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/ports"
)

var errParserPDF = errors.New("analisiscontenido: parser pdf no concluyente")

type configuracionQPDF struct {
	Bwrap, QPDF, Prlimit string
}

const limiteSalidaQPDF = 2 << 20

//go:embed testdata/pasivo.pdf
var pdfSonda []byte

// inspeccionarPDF aplica una política deliberadamente estrecha: un solo EOF,
// estructura íntegra, sin cifrado y sin diccionarios de acciones o adjuntos.
// qpdf corre en un espacio de montaje mínimo, sin red y con límites de recursos.
func inspeccionarPDF(ctx context.Context, contenido []byte, cfg configuracionQPDF) (codigo string, deteccion *ports.DeteccionContenido, err error) {
	if len(contenido) < 16 || !bytes.HasPrefix(contenido, []byte("%PDF-")) {
		return "PDF_FORMATO", &ports.DeteccionContenido{Clase: ports.ClaseDeteccionPolitica, Codigo: "PDF_FORMATO"}, nil
	}
	final := bytes.TrimRight(contenido, "\r\n\t ")
	if !bytes.HasSuffix(final, []byte("%%EOF")) || bytes.Count(contenido, []byte("%%EOF")) != 1 {
		return "PDF_DANADO", &ports.DeteccionContenido{Clase: ports.ClaseDeteccionArchivoDanado, Codigo: "PDF_DANADO"}, nil
	}
	dir, err := os.MkdirTemp("/tmp", "vec-analizador-pdf-")
	if err != nil {
		return "", nil, errParserPDF
	}
	defer func() {
		if os.RemoveAll(dir) != nil {
			codigo, deteccion, err = "", nil, errParserPDF
		}
	}()
	archivo, err := os.CreateTemp(dir, "original-*.pdf")
	if err != nil {
		return "", nil, errParserPDF
	}
	ruta := archivo.Name()
	if err = escribirTodo(archivo, contenido); err != nil {
		_ = archivo.Close()
		return "", nil, errParserPDF
	}
	if err = archivo.Close(); err != nil {
		return "", nil, errParserPDF
	}
	salidaQPDF, _, err := ejecutarQPDF(ctx, cfg, dir, "--is-encrypted", ruta)
	if err != nil {
		return "", nil, errParserPDF
	}
	if salidaQPDF == 0 {
		return "PDF_CIFRADO", &ports.DeteccionContenido{Clase: ports.ClaseDeteccionPolitica, Codigo: "PDF_CIFRADO"}, nil
	}
	if salidaQPDF != 2 {
		return "", nil, errParserPDF
	}
	salidaQPDF, _, err = ejecutarQPDF(ctx, cfg, dir, "--check", ruta)
	if err != nil {
		return "", nil, errParserPDF
	}
	if salidaQPDF != 0 {
		return "PDF_DANADO", &ports.DeteccionContenido{Clase: ports.ClaseDeteccionArchivoDanado, Codigo: "PDF_DANADO"}, nil
	}
	salidaQPDF, datos, err := ejecutarQPDF(ctx, cfg, dir, "--json=1", "--json-key=objects", "--json-stream-data=none", ruta)
	if err != nil || salidaQPDF != 0 {
		return "", nil, errParserPDF
	}
	var estructura struct {
		Version int            `json:"version"`
		Objects map[string]any `json:"objects"`
	}
	if json.Unmarshal(datos, &estructura) != nil || estructura.Version != 1 || len(estructura.Objects) == 0 {
		return "", nil, errParserPDF
	}
	if contenidoActivo(estructura.Objects) {
		return "PDF_ACTIVO", &ports.DeteccionContenido{Clase: ports.ClaseDeteccionContenidoActivo, Codigo: "PDF_ACTIVO"}, nil
	}
	return "PDF_PASIVO", nil, nil
}

var nombresPDFProhibidos = map[string]struct{}{
	"/A": {}, "/AA": {}, "/Action": {}, "/Annots": {}, "/OpenAction": {}, "/JS": {}, "/JavaScript": {},
	"/AcroForm": {}, "/XFA": {}, "/EmbeddedFiles": {}, "/EF": {}, "/Filespec": {}, "/AF": {}, "/Collection": {},
	"/RichMedia": {}, "/Rendition": {}, "/Movie": {}, "/Sound": {}, "/3D": {}, "/Launch": {}, "/SubmitForm": {},
	"/GoToR": {}, "/GoToE": {}, "/URI": {}, "/Names": {}, "/Encrypt": {}, "/Screen": {}, "/Widget": {},
}

func contenidoActivo(valor any) bool {
	switch v := valor.(type) {
	case map[string]any:
		for k, x := range v {
			if _, prohibido := nombresPDFProhibidos[k]; prohibido || contenidoActivo(x) {
				return true
			}
		}
	case []any:
		for _, x := range v {
			if contenidoActivo(x) {
				return true
			}
		}
	case string:
		if strings.HasPrefix(v, "/") {
			_, prohibido := nombresPDFProhibidos[v]
			return prohibido
		}
	}
	return false
}

type salidaAcotada struct {
	datos    []byte
	limite   int
	excedida bool
}

func (s *salidaAcotada) Write(p []byte) (int, error) {
	if len(s.datos)+len(p) > s.limite {
		s.excedida = true
		return 0, errParserPDF
	}
	s.datos = append(s.datos, p...)
	return len(p), nil
}

func ejecutarQPDF(ctx context.Context, cfg configuracionQPDF, dir string, args ...string) (int, []byte, error) {
	if cfg.Bwrap == "" || cfg.QPDF == "" || cfg.Prlimit == "" || !strings.HasPrefix(dir, "/tmp/vec-analizador-pdf-") {
		return 0, nil, errParserPDF
	}
	limite, cancelar := context.WithTimeout(ctx, 5*time.Second)
	defer cancelar()
	opciones := []string{
		"--unshare-net", "--unshare-pid", "--die-with-parent", "--clearenv", "--setenv", "PATH", "/usr/bin", "--setenv", "HOME", "/tmp",
		"--ro-bind", "/usr", "/usr", "--ro-bind", "/lib", "/lib", "--ro-bind", "/lib64", "/lib64",
		"--tmpfs", "/tmp", "--ro-bind", dir, dir, "--dev", "/dev", "--proc", "/proc", "--chdir", "/tmp",
		"--", cfg.Prlimit, "--cpu=4", "--as=536870912", "--fsize=0", "--", cfg.QPDF,
	}
	opciones = append(opciones, args...)
	// #nosec G204 -- ejecutables validados en NuevoServicio, argumentos fijos y ruta temporal privada; qpdf corre sin red.
	cmd := exec.CommandContext(limite, cfg.Bwrap, opciones...)
	var salida salidaAcotada
	salida.limite = limiteSalidaQPDF
	var errores salidaAcotada
	errores.limite = 4 << 10
	cmd.Stdout = &salida
	cmd.Stderr = &errores
	err := cmd.Run()
	if limite.Err() != nil || salida.excedida || errores.excedida {
		return 0, nil, errParserPDF
	}
	if err == nil {
		return 0, salida.datos, nil
	}
	var salidaError *exec.ExitError
	if errors.As(err, &salidaError) {
		return salidaError.ExitCode(), salida.datos, nil
	}
	return 0, nil, errParserPDF
}

var _ io.Writer = (*salidaAcotada)(nil)
