// vec-copias-ensayar-logica es un consumidor offline de ensayos sintéticos.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vec-diputacion-granada/internal/modules/administracion/adapters/ensayologicopg"
)

type configuracionCLI struct {
	ImagenSHA256         string `json:"imagen_sha256"`
	VersionPostgreSQL    string `json:"version_postgresql"`
	UsuarioBootstrap     string `json:"usuario_bootstrap"`
	LimiteArchivoBytes   int64  `json:"limite_archivo_bytes"`
	CPUs                 int    `json:"cpus"`
	MemoriaBytes         int64  `json:"memoria_bytes"`
	TiempoLimiteSegundos int64  `json:"tiempo_limite_segundos"`
}

type salida struct {
	Alcance   string                   `json:"alcance"`
	Mensaje   string                   `json:"mensaje"`
	Resultado ensayologicopg.Resultado `json:"resultado"`
}

func leer(ruta string, destino any) error {
	f, err := os.OpenFile(ruta, os.O_RDONLY|syscall.O_NONBLOCK, 0) // #nosec G304 G703 -- fichero local explícito; fstat rechaza FIFO/dispositivos sin bloquear la lectura.
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return io.ErrUnexpectedEOF
	}
	d := json.NewDecoder(io.LimitReader(f, 1<<20))
	d.DisallowUnknownFields()
	if err = d.Decode(destino); err != nil {
		return err
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}

func diagnosticar(w io.Writer, clave string) {
	_ = json.NewEncoder(w).Encode(struct {
		ErrorClave string `json:"error_clave"`
	}{clave})
}

func run(args []string, out, diag io.Writer) int {
	return runContext(context.Background(), args, out, diag)
}

func runContext(ctx context.Context, args []string, out, diag io.Writer) int {
	flags := flag.NewFlagSet("vec-copias-ensayar-logica", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var config, solicitud, catalogo string
	flags.StringVar(&config, "configuracion", "", "")
	flags.StringVar(&solicitud, "solicitud", "", "")
	flags.StringVar(&catalogo, "catalogo", "", "")
	if flags.Parse(args) != nil || flags.NArg() != 0 || config == "" || solicitud == "" || catalogo == "" {
		diagnosticar(diag, "copias_ensayo_logico_error_argumentos")
		return 2
	}
	var textos map[string]string
	if leer(catalogo, &textos) != nil || textos["alcance"] == "" || textos["restauracion_logica_completada"] == "" || textos["restauracion_logica_fallida"] == "" {
		diagnosticar(diag, "copias_ensayo_logico_error_catalogo")
		return 2
	}
	var c configuracionCLI
	var s ensayologicopg.Solicitud
	if leer(config, &c) != nil || leer(solicitud, &s) != nil || c.TiempoLimiteSegundos < 1 || c.TiempoLimiteSegundos > 1800 {
		diagnosticar(diag, "copias_ensayo_logico_error_entrada")
		return 2
	}
	e := ensayologicopg.Ensayador{Configuracion: ensayologicopg.Configuracion{
		ImagenSHA256: c.ImagenSHA256, VersionPostgreSQL: c.VersionPostgreSQL, UsuarioBootstrap: c.UsuarioBootstrap,
		LimiteArchivoBytes: c.LimiteArchivoBytes, CPUs: c.CPUs, MemoriaBytes: c.MemoriaBytes,
		TiempoLimite: time.Duration(c.TiempoLimiteSegundos) * time.Second,
	}}
	r := e.Ensayar(ctx, s)
	if json.NewEncoder(out).Encode(salida{Alcance: textos["alcance"], Mensaje: textos[r.Estado], Resultado: r}) != nil {
		diagnosticar(diag, "copias_ensayo_logico_error_salida")
		return 2
	}
	if r.Estado != "restauracion_logica_completada" || !r.LimpiezaCompletada {
		return 1
	}
	return 0
}

func main() {
	ctx, cancelar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	codigo := runContext(ctx, os.Args[1:], os.Stdout, os.Stderr)
	cancelar()
	os.Exit(codigo)
}
