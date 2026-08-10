//go:build ignore && linux && amd64

package main

import (
	"os"
	"os/exec"
	"runtime"
	"syscall"
	"testing"
	"time"
)

func custodiaNominalO3bPruebaM38(t *testing.T) *custodiaO3aM38 {
	t.Helper()
	archivos := make([]*os.File, 0, 3)
	for range 3 {
		lector, escritor, err := os.Pipe()
		if err != nil {
			t.Fatalf("crear FD de prueba: %v", err)
		}
		archivos = append(archivos, escritor)
		t.Cleanup(func() { _ = lector.Close(); _ = escritor.Close() })
	}
	tid := syscall.Gettid()
	r := nuevoRegistroAutoridadO3aM38()
	l := nuevaLeaseGuardiaO3aM38(r)
	l.estado.Store(3)
	o := nuevoObservadorSenalO3aM38(r)
	o.palabra.Store(2)
	a := nuevaAutoridadO3aM38()
	a.estado = arranqueA6EntregadoM38
	identidad := identidadFDO3aM38{dev: 1, ino: 2, modo: syscall.S_IFREG, enlaces: 1}
	var suma [32]byte
	suma[0] = 1
	return &custodiaO3aM38{
		autoridad: a,
		control: &controladorPreinicioM38{
			fase:   controlPreinicioS3M38,
			lector: &lectorTramaM38{clase: "CONTROL", limite: 1024},
		},
		controlFD: archivos[0], terminal: archivos[1], ticketEscritor: archivos[2],
		lease: l, observador: o, baselineSenal: 2,
		finBootstrap: time.Now().Add(time.Second), tid: tid, ppid: os.Getppid(),
		cmd:           &exec.Cmd{Process: &os.Process{Pid: os.Getpid()}},
		pidfdPrimario: 1_000_000, pidfdReserva: 1_000_001, pidfdOpaco: 1_000_002,
		snapshot:    snapshotFDO3aM38{limite: 13, mapa: map[int]huellaFDO3aM38{}},
		baseline:    snapshotFDO3aM38{limite: 13, mapa: map[int]huellaFDO3aM38{}},
		formaRaiz:   formaRaizM38{identidad: identidad},
		formaRunner: formaRunnerM38{identidad: identidad, sha256: suma},
	}
}

func agregadoNominalO3bPruebaM38(t *testing.T) *agregadoO3aM38 {
	c := custodiaNominalO3bPruebaM38(t)
	c.consumida.Store(1)
	return &agregadoO3aM38{custodia: c}
}

func TestConsumirAutoridadO3bNominalYAlias(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	agregado := agregadoNominalO3bPruebaM38(t)
	alias := agregado
	escritor := agregado.custodia.ticketEscritor
	estadoLease := agregado.custodia.lease.estado.Load()
	palabra := agregado.custodia.observador.palabra.Load()
	autoridad, err := consumirAutoridadO3bM38(&agregado)
	if err != nil || agregado != nil || !autoridad.es(capturaB0RecibidoM38) {
		t.Fatalf("entrada nominal no produjo B0: autoridad=%v err=%v", autoridad, err)
	}
	if autoridad.custodia.ticketEscritor != escritor || autoridad.custodia.lease.estado.Load() != estadoLease ||
		autoridad.custodia.observador.palabra.Load() != palabra {
		t.Fatal("P1 alteró ticket, lease u observador")
	}
	if segunda, err := consumirAutoridadO3bM38(&alias); err != errEntradaO3bM38 || alias != nil || !segunda.es(capturaB8RetiradoM38) {
		t.Fatalf("alias consumido aceptado: autoridad=%v err=%v", segunda, err)
	}
	clon := &agregadoO3aM38{custodia: autoridad.custodia}
	if segunda, err := consumirAutoridadO3bM38(&clon); err != errEntradaO3bM38 || clon != nil || !segunda.es(capturaB8RetiradoM38) {
		t.Fatalf("clon consumido aceptado: autoridad=%v err=%v", segunda, err)
	}
}

func TestConsumirAutoridadO3bRechazaEntradaYAutoridad(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if a, err := consumirAutoridadO3bM38(nil); a != nil || err != errEntradaO3bM38 {
		t.Fatal("puntero nulo aceptado")
	}
	var nulo *agregadoO3aM38
	if a, err := consumirAutoridadO3bM38(&nulo); a != nil || err != errEntradaO3bM38 {
		t.Fatal("agregado nulo aceptado")
	}
	envoltorioNulo := &agregadoO3aM38{}
	if a, err := consumirAutoridadO3bM38(&envoltorioNulo); err != errEntradaO3bM38 || envoltorioNulo != nil || !a.es(capturaB8RetiradoM38) {
		t.Fatal("custodia nula no fue consumida de forma terminal")
	}
	casos := []struct {
		nombre string
		mutar  func(*custodiaO3aM38)
	}{
		{"estado", func(c *custodiaO3aM38) { c.autoridad.estado = arranqueA5PidfdTresM38 }},
		{"lease_alias", func(c *custodiaO3aM38) { c.lease.auto = &leaseGuardiaO3aM38{} }},
		{"lease_generacion", func(c *custodiaO3aM38) { c.lease.generacion++ }},
		{"observador_alias", func(c *custodiaO3aM38) { c.observador.auto = &observadorSenalO3aM38{} }},
		{"observador_generacion", func(c *custodiaO3aM38) { c.observador.generacion++ }},
		{"registro_distinto", func(c *custodiaO3aM38) {
			r := nuevoRegistroAutoridadO3aM38()
			delete(c.observador.registro.observadores, c.observador)
			c.observador.registro = r
			r.observadores[c.observador] = c.observador.generacion
		}},
		{"tid", func(c *custodiaO3aM38) { c.tid++ }},
		{"contador", func(c *custodiaO3aM38) { c.observador.palabra.Add(1 << 10) }},
		{"pidfd_alias", func(c *custodiaO3aM38) { c.pidfdOpaco = c.pidfdPrimario }},
		{"fd_alias", func(c *custodiaO3aM38) { c.terminal = c.controlFD }},
		{"fd_pidfd_alias", func(c *custodiaO3aM38) { c.pidfdPrimario = int(c.controlFD.Fd()) }},
		{"lector_no_limpio", func(c *custodiaO3aM38) { c.control.lector.buffer[0] = 'x' }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			agregado := agregadoNominalO3bPruebaM38(t)
			escritor := agregado.custodia.ticketEscritor
			caso.mutar(agregado.custodia)
			custodia := agregado.custodia
			if a, err := consumirAutoridadO3bM38(&agregado); err != errAutoridadO3bM38 || agregado != nil || !a.es(capturaB7RetirandoM38) || a.custodia != custodia {
				t.Fatalf("autoridad adversa aceptada: autoridad=%v err=%v", a, err)
			}
			if custodia.consumida.Load() != 2 || custodia.ticketEscritor != escritor {
				t.Fatal("rechazo no fue lineal o tocó ticket")
			}
		})
	}
}

func TestTransicionesCapturaO3bTotales(t *testing.T) {
	verdes := map[[2]estadoCapturaO3bM38]bool{
		{capturaB0RecibidoM38, capturaB1BarreraVerdeM38}:              true,
		{capturaB1BarreraVerdeM38, capturaB2TicketCerradoM38}:         true,
		{capturaB2TicketCerradoM38, capturaB3StopObservadoM38}:        true,
		{capturaB3StopObservadoM38, capturaB4IdentidadAcreditadaM38}:  true,
		{capturaB4IdentidadAcreditadaM38, capturaB4TTransfiriendoM38}: true,
		{capturaB4TTransfiriendoM38, capturaB5CapturadoM38}:           true,
		{capturaB7RetirandoM38, capturaB8RetiradoM38}:                 true,
	}
	for desde := capturaB0RecibidoM38; desde <= capturaBFFatalM38; desde++ {
		for hacia := capturaB0RecibidoM38; hacia <= capturaBFFatalM38; hacia++ {
			esperada := verdes[[2]estadoCapturaO3bM38{desde, hacia}] ||
				desde <= capturaB4IdentidadAcreditadaM38 && (hacia == capturaB7RetirandoM38 || hacia == capturaBFFatalM38) ||
				desde == capturaB4TTransfiriendoM38 && hacia == capturaBFFatalM38 ||
				desde == capturaB7RetirandoM38 && hacia == capturaBFFatalM38
			if transicionCapturaO3bM38(desde, hacia) != esperada {
				t.Fatalf("transición %d->%d incorrecta", desde, hacia)
			}
		}
	}
}
