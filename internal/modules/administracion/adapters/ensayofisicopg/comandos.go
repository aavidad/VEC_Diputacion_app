package ensayofisicopg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"time"
	"vec-diputacion-granada/internal/shared/plazoarranque"
)

const dockerLocal = "/usr/bin/docker"

// El endpoint es local y fijo: DOCKER_HOST, contextos del operador y plugins
// de su HOME no forman parte de esta frontera. No se invoca ningún shell.
func docker(ctx context.Context, entrada io.Reader, limite int, argumentos ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, dockerLocal, append([]string{"--host", "unix:///var/run/docker.sock"}, argumentos...)...) // #nosec G204 G702 -- ejecutable fijo, endpoint Unix local y argumentos cerrados de este adaptador.
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=/nonexistent", "DOCKER_CONFIG=/nonexistent", "LANG=C", "LC_ALL=C"}
	cmd.Stdin, cmd.Stderr = entrada, io.Discard
	out := &salidaAcotada{limite: limite}
	cmd.Stdout = out
	cmd.WaitDelay = time.Second
	err := cmd.Run()
	return out.Bytes(), err
}

type salidaAcotada struct {
	bytes.Buffer
	limite int
}

func (s *salidaAcotada) Write(p []byte) (int, error) {
	if len(p) > s.limite-s.Len() {
		return 0, fmt.Errorf("ensayo_fisico_salida_excedida")
	}
	return s.Buffer.Write(p)
}

func (e Ensayador) opcionesAisladas(nombre string) []string {
	c := e.Configuracion
	return []string{"run", "--name", nombre, "--rm", "--pull", "never", "--network", "none", "--read-only",
		"--user", fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid()), "--cap-drop", "ALL",
		"--security-opt", "no-new-privileges:true", "--cpus", strconv.Itoa(c.CPUs),
		"--memory", strconv.FormatInt(c.MemoriaBytes, 10), "--memory-swap", strconv.FormatInt(c.MemoriaBytes, 10),
		"--pids-limit", "128", "--tmpfs", "/tmp:rw,nosuid,nodev,size=67108864",
		"--tmpfs", "/var/run/postgresql:rw,nosuid,nodev,size=16777216,mode=1777", "--tmpfs", "/var/lib/postgresql:rw,nosuid,nodev,size=16777216,mode=1777"}
}

func (e Ensayador) herramienta(ctx context.Context, nombre, programa string, args ...string) ([]byte, error) {
	cmd := append(e.opcionesAisladas(nombre), "--entrypoint", programa, "sha256:"+e.Configuracion.ImagenSHA256)
	return docker(ctx, nil, 4096, append(cmd, args...)...)
}

func eliminarContenedor(nombre string) bool {
	ctx, cancelar := context.WithTimeout(context.Background(), plazoarranque.Ampliar(15*time.Second))
	defer cancelar()
	_, _ = docker(ctx, nil, 4096, "container", "rm", "--force", nombre)
	b, err := docker(ctx, nil, 4096, "container", "ls", "--all", "--filter", "name=^/"+nombre+"$", "--format", "{{.ID}}")
	return err == nil && len(bytes.TrimSpace(b)) == 0
}
