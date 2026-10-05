package capturacopias

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

var ErrProceso = errors.New("copias_captura_proceso")

// Comando pertenece a configuración privada del operador, nunca a peticiones.
type Comando struct {
	Ejecutable string   `json:"ejecutable"`
	Argumentos []string `json:"argumentos"`
}

type Ejecutor struct {
	Entorno []string
	Limite  time.Duration
}

// Ejecutar no utiliza shell, no hereda secretos del proceso y no comunica el
// stderr del proveedor. La cancelación termina el grupo, incluidos descendientes.
func (e Ejecutor) Ejecutar(ctx context.Context, c Comando, destino io.Writer) error {
	if !filepath.IsAbs(c.Ejecutable) || e.Limite <= 0 {
		return ErrProceso
	}
	switch filepath.Base(c.Ejecutable) {
	case "sh", "bash", "dash", "zsh", "fish", "cmd", "powershell":
		return ErrProceso
	}
	ctx, cancel := context.WithTimeout(ctx, e.Limite)
	defer cancel()
	// #nosec G204 -- executable and arguments come only from private operator config, no HTTP or manifest input.
	cmd := exec.CommandContext(ctx, c.Ejecutable, c.Argumentos...)
	cmd.Env = append([]string{"LANG=C", "LC_ALL=C"}, e.Entorno...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = destino
	cmd.Stderr = io.Discard
	cmd.WaitDelay = time.Second
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	if cmd.Run() != nil {
		return ErrProceso
	}
	return nil
}

type Escritor struct {
	ID        string  `json:"id"`
	Cerrar    Comando `json:"cerrar"`
	Drenar    Comando `json:"drenar"`
	Comprobar Comando `json:"comprobar"`
	Reabrir   Comando `json:"reabrir"`
}

type Control struct {
	Ejecutor   Ejecutor
	Escritores []Escritor
}

func (c Control) todos(ctx context.Context, seleccionar func(Escritor) Comando, continuar bool) error {
	if len(c.Escritores) == 0 {
		return ErrProceso
	}
	var fallo error
	for _, w := range c.Escritores {
		if c.Ejecutor.Ejecutar(ctx, seleccionar(w), io.Discard) != nil {
			fallo = ErrProceso
			if !continuar {
				return fallo
			}
		}
	}
	return fallo
}
func (c Control) CerrarAdmision(ctx context.Context) error {
	return c.todos(ctx, func(w Escritor) Comando { return w.Cerrar }, false)
}
func (c Control) Drenar(ctx context.Context) error {
	return c.todos(ctx, func(w Escritor) Comando { return w.Drenar }, false)
}
func (c Control) ComprobarExclusion(ctx context.Context) error {
	return c.todos(ctx, func(w Escritor) Comando { return w.Comprobar }, false)
}
func (c Control) Reabrir(ctx context.Context) error {
	return c.todos(ctx, func(w Escritor) Comando { return w.Reabrir }, true)
}
