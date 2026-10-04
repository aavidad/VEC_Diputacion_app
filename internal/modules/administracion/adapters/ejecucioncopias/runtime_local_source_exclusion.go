package ejecucioncopias

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// EjecutarPostgreSQL satisface el lector CS06 sobre la fuente real. Sólo psql
// puede observar el cluster; el adaptador nunca acepta otro contenedor o DSN.
func (o *OrigenLocal) EjecutarPostgreSQL(ctx context.Context, herramienta string, args []string, entrada []byte, limite int) ([]byte, error) {
	if herramienta != "psql" || len(args) > 64 || len(entrada) > 8<<20 || limite < 1 || limite > 64<<20 {
		return nil, ErrOrigenLocal
	}
	if _, err := o.ComprobarExclusion(ctx); err != nil {
		return nil, err
	}
	for _, a := range args {
		if len(a) > 8<<20 || strings.ContainsRune(a, 0) {
			return nil, ErrOrigenLocal
		}
	}
	cmd := []string{"exec", "--interactive", o.c.Contenedor, "env", "-i", "PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "PGHOST=/var/run/postgresql", "PGUSER=" + o.c.Usuario, "PGOPTIONS=-c default_transaction_read_only=on", o.c.Herramientas["psql"]}
	return dockerCS11(ctx, bytes.NewReader(entrada), limite, append(cmd, args...)...)
}

// Adquirir conserva un flock local propio. El cluster carece de red y se
// comprueban sesiones cliente para evitar medir durante otro escritor.
func (o *OrigenLocal) Adquirir(ctx context.Context, ref string) (func() error, error) {
	if o == nil || ctx == nil || ref != o.c.OrigenRef {
		return nil, ErrOrigenLocal
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.lease != nil {
		return nil, ErrOrigenLocal
	}
	f, err := os.OpenFile(filepath.Join(o.c.RaizPropia, "captura.lock"), os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0600) // #nosec G304 G703 -- ruta fija bajo raíz propia validada.
	if err != nil || syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) != nil {                                // #nosec G115 -- descriptor creado por os.OpenFile dentro del proceso Linux.
		if f != nil {
			_ = f.Close()
		}
		return nil, ErrOrigenLocal
	}
	o.lease = f
	return func() error {
		o.mu.Lock()
		defer o.mu.Unlock()
		if o.lease != f {
			return ErrOrigenLocal
		}
		o.lease = nil
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_UN) // #nosec G115 -- mismo descriptor abierto y retenido por el proceso.
		if f.Close() != nil {
			return ErrOrigenLocal
		}
		return err
	}, nil
}

func (o *OrigenLocal) leaseVigente() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.lease != nil
}
func (o *OrigenLocal) sesionesCliente(ctx context.Context) (bool, error) {
	b, err := dockerCS11(ctx, nil, 1024, "exec", o.c.Contenedor, "env", "-i", "PATH=/usr/bin:/bin", "LANG=C", "LC_ALL=C", "PGHOST=/var/run/postgresql", "PGUSER="+o.c.Usuario, o.c.Herramientas["psql"], "-X", "-At", "-d", o.c.Base, "-c", "SELECT count(*) FROM pg_stat_activity WHERE backend_type='client backend' AND pid<>pg_backend_pid()")
	return err == nil && strings.TrimSpace(string(b)) == "0", err
}
func (o *OrigenLocal) ComprobarExclusion(ctx context.Context) (string, error) {
	if !o.leaseVigente() {
		return "", ErrOrigenLocal
	}
	id, err := o.observarContenedor(ctx, true)
	if err != nil {
		return "", err
	}
	libre, err := o.sesionesCliente(ctx)
	if err != nil || !libre {
		return "", ErrOrigenLocal
	}
	return id, nil
}

// Los cuatro pasos de CS04 usan la misma observación. No hay app/SMTP en el
// cluster fuente aislado; su único proceso propio es postgres.
type EscritoresOrigenLocal struct{ Origen *OrigenLocal }

func (x EscritoresOrigenLocal) CerrarAdmision(ctx context.Context) error {
	_, err := x.Origen.ComprobarExclusion(ctx)
	return err
}
func (x EscritoresOrigenLocal) Drenar(ctx context.Context) error {
	_, err := x.Origen.ComprobarExclusion(ctx)
	return err
}
func (x EscritoresOrigenLocal) ComprobarExclusion(ctx context.Context) error {
	_, err := x.Origen.ComprobarExclusion(ctx)
	return err
}
func (x EscritoresOrigenLocal) Reabrir(context.Context) error { return nil }
func (o *OrigenLocal) Escritores() EscritoresOrigenLocal      { return EscritoresOrigenLocal{o} }

// ControlFisicoOrigenLocal implementa el contrato CS05 sobre el mismo cluster.
type ControlFisicoOrigenLocal struct{ Origen *OrigenLocal }

func (x ControlFisicoOrigenLocal) ComprobarExclusion(ctx context.Context) error {
	if !x.Origen.leaseVigente() {
		return ErrOrigenLocal
	}
	if _, err := x.Origen.observarContenedor(ctx, false); err == nil {
		return nil
	}
	_, err := x.Origen.ComprobarExclusion(ctx)
	return err
}
func (x ControlFisicoOrigenLocal) DetenerPostgreSQL(ctx context.Context) error {
	if !x.Origen.leaseVigente() {
		return ErrOrigenLocal
	}
	if _, err := x.Origen.observarContenedor(ctx, true); err != nil {
		return err
	}
	_, err := dockerCS11(ctx, nil, 2048, "stop", "--time", "30", x.Origen.c.Contenedor)
	return err
}
func (x ControlFisicoOrigenLocal) ComprobarFrio(ctx context.Context) error {
	if _, err := x.Origen.observarContenedor(ctx, false); err != nil {
		return err
	}
	for _, name := range []string{"postmaster.pid", "standby.signal", "recovery.signal"} {
		if _, err := os.Stat(filepath.Join(x.Origen.c.PGDATA, name)); !os.IsNotExist(err) {
			return ErrOrigenLocal
		}
	}
	// pg_controldata lee la copia detenida en un contenedor temporal sin red,
	// con el mismo digest de herramientas y el PGDATA propio montado ro.
	b, err := dockerCS11(ctx, nil, 1<<16, "run", "--rm", "--network", "none", "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--mount", "type=bind,src="+x.Origen.c.PGDATA+",dst=/data,readonly", "--entrypoint", "pg_controldata", "sha256:"+x.Origen.c.ImagenSHA256, "/data")
	if err != nil || !regexp.MustCompile(`(?m)^Database cluster state:\s+shut down\s*$`).Match(b) {
		return ErrOrigenLocal
	}
	return nil
}
func (x ControlFisicoOrigenLocal) ReanudarPostgreSQL(ctx context.Context) error {
	if !x.Origen.leaseVigente() {
		return ErrOrigenLocal
	}
	if _, err := x.Origen.observarContenedor(ctx, true); err != nil {
		if _, err := x.Origen.observarContenedor(ctx, false); err != nil {
			return err
		}
		if _, err := dockerCS11(ctx, nil, 2048, "start", x.Origen.c.Contenedor); err != nil {
			return err
		}
	}
	for n := 0; n < 100; n++ {
		if _, err := x.Origen.observarContenedor(ctx, true); err == nil {
			b, err := dockerCS11(ctx, nil, 1024, "exec", x.Origen.c.Contenedor, "pg_isready", "-h", "/var/run/postgresql", "-U", x.Origen.c.Usuario, "-d", x.Origen.c.Base)
			if err == nil && bytes.Contains(b, []byte("accepting connections")) {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return ErrOrigenLocal
		case <-time.After(100 * time.Millisecond):
		}
	}
	return ErrOrigenLocal
}
func (o *OrigenLocal) ControlFisico() ControlFisicoOrigenLocal { return ControlFisicoOrigenLocal{o} }
