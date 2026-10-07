package capturafisica

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

// Herramienta procede exclusivamente de configuración local protegida, nunca
// del inventario archivado ni de una petición web. No se interpreta un shell.
type Herramienta struct {
	Ejecutable string   `json:"ejecutable"`
	Argumentos []string `json:"argumentos"`
}
type ConfigControl struct {
	PGDATA         string                 `json:"pgdata"`
	Herramientas   map[string]Herramienta `json:"herramientas"`
	TiempoSegundos int                    `json:"tiempo_segundos"`
}
type ControlComandos struct{ Config ConfigControl }

func (c ControlComandos) ComprobarExclusion(ctx context.Context) error {
	out, e := c.ejecutar(ctx, "escritores_observar", 0)
	if e != nil || strings.TrimSpace(out) != "excluded" {
		return ErrControl
	}
	return nil
}
func (c ControlComandos) DetenerPostgreSQL(ctx context.Context) error {
	_, e := c.ejecutar(ctx, "pg_detener", 0)
	return e
}
func (c ControlComandos) ComprobarFrio(ctx context.Context) error {
	if _, e := c.ejecutar(ctx, "pg_estado", 3); e != nil {
		return ErrControl
	}
	out, e := c.ejecutar(ctx, "pg_control", 0)
	if e != nil {
		return ErrControl
	}
	limpio := false
	for _, linea := range strings.Split(out, "\n") {
		clave, valor, ok := strings.Cut(linea, ":")
		if ok && strings.TrimSpace(clave) == "Database cluster state" {
			limpio = strings.TrimSpace(valor) == "shut down"
		}
	}
	if !limpio {
		return ErrControl
	}
	pg, e := os.OpenRoot(c.Config.PGDATA)
	if e != nil {
		return ErrControl
	}
	defer pg.Close()
	if _, e = pg.Lstat("postmaster.pid"); !os.IsNotExist(e) {
		return ErrControl
	}
	return nil
}
func (c ControlComandos) ReanudarPostgreSQL(ctx context.Context) error {
	if _, e := c.ejecutar(ctx, "pg_estado", 0); e == nil {
		return nil
	}
	if _, e := c.ejecutar(ctx, "pg_estado", 3); e != nil {
		return ErrControl
	}
	if _, e := c.ejecutar(ctx, "pg_reanudar", 0); e != nil {
		return e
	}
	_, e := c.ejecutar(ctx, "pg_estado", 0)
	return e
}
func (c ControlComandos) CerrarEscritores(ctx context.Context) error {
	_, e := c.ejecutar(ctx, "escritores_detener", 0)
	return e
}
func (c ControlComandos) ReabrirEscritores(ctx context.Context) error {
	if _, e := c.ejecutar(ctx, "pg_estado", 0); e != nil {
		return ErrControl
	}
	_, e := c.ejecutar(ctx, "escritores_reanudar", 0)
	return e
}

func (c ControlComandos) ejecutar(ctx context.Context, alias string, esperado int) (string, error) {
	cfg := c.Config
	h, ok := cfg.Herramientas[alias]
	if !ok || cfg.TiempoSegundos < 1 || cfg.TiempoSegundos > 3600 || !filepath.IsAbs(h.Ejecutable) || sinEnlaces(h.Ejecutable) != nil {
		return "", ErrConfiguracion
	}
	switch filepath.Base(h.Ejecutable) {
	case "sh", "bash", "dash", "zsh", "fish", "powershell", "cmd", "env":
		return "", ErrConfiguracion
	}
	s, e := os.Stat(h.Ejecutable)
	if e != nil || !s.Mode().IsRegular() || s.Mode().Perm()&0022 != 0 || s.Mode().Perm()&0111 == 0 {
		return "", ErrConfiguracion
	}
	limitado, cancel := context.WithTimeout(ctx, plazoarranque.Ampliar(time.Duration(cfg.TiempoSegundos)*time.Second))
	defer cancel()
	// El alias usa exclusivamente configuración local protegida y no interpreta shell.
	cmd := exec.CommandContext(limitado, h.Ejecutable, h.Argumentos...) // #nosec G204 -- configured local platform authority, fixed aliases and no shell
	cmd.Env = []string{"LC_ALL=C", "LANG=C", "PATH=/usr/bin:/bin"}
	var out limiteSalida
	cmd.Stdout = &out
	cmd.Stderr = &out
	e = cmd.Run()
	obtenido := 0
	if e != nil {
		ex, ok := e.(*exec.ExitError)
		if !ok {
			return "", ErrControl
		}
		obtenido = ex.ExitCode()
	}
	if obtenido != esperado || out.exceso {
		return "", ErrControl
	}
	return out.String(), nil
}

type limiteSalida struct {
	bytes.Buffer
	exceso bool
}

func (l *limiteSalida) Write(p []byte) (int, error) {
	n := len(p)
	if l.Len()+n > 65536 {
		l.exceso = true
		return n, nil
	}
	_, _ = l.Buffer.Write(p)
	return n, nil
}
