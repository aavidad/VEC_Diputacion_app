//go:build ignore && linux && amd64

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func TestMain(m *testing.M) {
	_, fuente, _, ok := runtime.Caller(0)
	if !ok {
		os.Exit(2)
	}
	raiz := filepath.Clean(filepath.Join(filepath.Dir(fuente), "../../../.."))
	if err := os.Chdir(raiz); err != nil {
		os.Exit(2)
	}
	os.Exit(m.Run())
}

func autoridadRealBarreraO3bPruebaM38(t *testing.T) (*fixtureO3aM38, *autoridadCapturaO3bM38) {
	t.Helper()
	f, err := prepararCasoExternoO3aM38()
	if err != nil {
		t.Fatalf("preparar O3a: %v", err)
	}
	if err = avanzarFixtureO3aM38(f); err != nil {
		_ = limpiarFixtureO3aM38(f)
		t.Fatalf("avanzar O3a: %v", err)
	}
	a, err := consumirAutoridadO3bM38(&f.agregado)
	if err != nil || !a.es(capturaB0RecibidoM38) {
		_ = limpiarFixtureO3aM38(f)
		t.Fatalf("consumir P1: %v", err)
	}
	t.Cleanup(func() {
		a.custodia.consumida.Store(1)
		f.agregado = &agregadoO3aM38{custodia: a.custodia}
		if err := limpiarFixtureO3aM38(f); err != nil {
			t.Errorf("limpiar fixture: %v", err)
		}
		runtime.UnlockOSThread()
	})
	return f, a
}

func identidadArchivoBarreraO3bPruebaM38(t *testing.T, fd int) identidadFDO3aM38 {
	t.Helper()
	id, err := identidadPidfdO3aM38(fd)
	if err != nil {
		t.Fatalf("identidad FD: %v", err)
	}
	return id
}

func bytesTicketBarreraO3bPruebaM38(t *testing.T, fd int) int {
	t.Helper()
	var disponibles int32
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), syscall.TIOCINQ, uintptr(unsafe.Pointer(&disponibles)))
	if errno != 0 {
		t.Fatalf("medir ticket: %v", errno)
	}
	return int(disponibles)
}

func limpiarPermisoPrimeraEscrituraBarreraO3bPruebaM38(t *testing.T, a *autoridadCapturaO3bM38) {
	t.Helper()
	if a == nil || a.ticket == nil || !a.ticket.permisoPrimero ||
		!a.custodia.lease.consolidarCritico(a.ticket.primerPermiso) {
		t.Fatal("limpiar permiso test-only")
	}
	a.ticket.permisoPrimero = false
}

func TestBarreraO3bNominalSinEfectoTicket(t *testing.T) {
	_, a := autoridadRealBarreraO3bPruebaM38(t)
	ticketFD := int(a.custodia.ticketEscritor.Fd())
	antes := identidadArchivoBarreraO3bPruebaM38(t, ticketFD)
	if bytesTicketBarreraO3bPruebaM38(t, ticketFD) != 0 {
		t.Fatal("ticket no empezó vacío")
	}
	lease, observador := a.custodia.lease.estado.Load(), a.custodia.observador.palabra.Load()
	if err := ejecutarBarreraO3bM38(a); err != nil || !a.es(capturaB1BarreraVerdeM38) {
		t.Fatalf("barrera nominal causa=%d: %v", causaDelFalloBarreraO3bM38(err), err)
	}
	despues := identidadArchivoBarreraO3bPruebaM38(t, ticketFD)
	if antes != despues || (lease != 1 && lease != 3) || a.custodia.lease.estado.Load() != 2 ||
		a.ticket == nil || !a.ticket.permisoPrimero || a.custodia.observador.palabra.Load() != observador {
		t.Fatal("la barrera no dejó preparado únicamente el permiso del primer Write")
	}
	if bytesTicketBarreraO3bPruebaM38(t, ticketFD) != 0 {
		t.Fatal("la barrera escribió ticket")
	}
	limpiarPermisoPrimeraEscrituraBarreraO3bPruebaM38(t, a)
}

func TestBarreraO3bBordeBootstrap(t *testing.T) {
	ahora := time.Now()
	if !bootstrapDisponibleO3bM38(ahora.Add(time.Second), ahora) {
		t.Fatal("un segundo exacto debe ser admisible")
	}
	if bootstrapDisponibleO3bM38(ahora.Add(time.Second-time.Nanosecond), ahora) {
		t.Fatal("un nanosegundo menos debe retirar")
	}
}

func TestBarreraO3bLimitesControl(t *testing.T) {
	if !dentroPresupuestoControlO3bM38(3, 4095, 8) {
		t.Fatal("el último presupuesto válido fue rechazado")
	}
	for _, limite := range [][3]int{{4, 0, 0}, {0, 4096, 0}, {0, 0, 9}} {
		if dentroPresupuestoControlO3bM38(limite[0], limite[1], limite[2]) {
			t.Fatalf("presupuesto adverso aceptado: %v", limite)
		}
	}
}

func TestBarreraO3bPrecedenciasYNegativos(t *testing.T) {
	casos := []struct {
		nombre string
		causa  causaBarreraO3bM38
		mutar  func(*fixtureO3aM38, *custodiaO3aM38) func()
	}{
		{"control_gana_simultaneo", barreraO3bControlM38, func(f *fixtureO3aM38, c *custodiaO3aM38) func() {
			c.observador.palabra.Add(1 << 10)
			c.finBootstrap = time.Now()
			if err := escribirControlPruebaO3aM38(f, "V1|CONTROL|CANCELAR|"+string(c.control.nonce[:])+"|CANCELADO|65\n"); err != nil {
				panic(err)
			}
			return func() {}
		}},
		{"senal_gana_bootstrap", barreraO3bSenalM38, func(_ *fixtureO3aM38, c *custodiaO3aM38) func() {
			c.observador.palabra.Add(1 << 10)
			c.finBootstrap = time.Now()
			return func() {}
		}},
		{"bootstrap_gana_pidfd", barreraO3bBootstrapM38, func(_ *fixtureO3aM38, c *custodiaO3aM38) func() {
			c.finBootstrap = time.Now().Add(time.Second - time.Nanosecond)
			anterior := c.pidfdReserva
			c.pidfdReserva = -1
			return func() { c.pidfdReserva = anterior }
		}},
		{"reserva_pidfd_ausente", barreraO3bPidfdM38, func(_ *fixtureO3aM38, c *custodiaO3aM38) func() {
			anterior := c.pidfdReserva
			c.pidfdReserva = -1
			return func() { c.pidfdReserva = anterior }
		}},
		{"primario_pidfd_ausente", barreraO3bPidfdM38, func(_ *fixtureO3aM38, c *custodiaO3aM38) func() {
			anterior := c.pidfdPrimario
			c.pidfdPrimario = -1
			return func() { c.pidfdPrimario = anterior }
		}},
		{"inventario", barreraO3bInventarioM38, func(_ *fixtureO3aM38, c *custodiaO3aM38) func() {
			anterior := c.terminal
			c.terminal = c.ticketEscritor
			return func() { c.terminal = anterior }
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			f, a := autoridadRealBarreraO3bPruebaM38(t)
			ticketFD := int(a.custodia.ticketEscritor.Fd())
			antes := identidadArchivoBarreraO3bPruebaM38(t, ticketFD)
			restaurar := caso.mutar(f, a.custodia)
			defer restaurar()
			err := ejecutarBarreraO3bM38(a)
			if causaDelFalloBarreraO3bM38(err) != caso.causa || !a.es(capturaB7RetirandoM38) {
				t.Fatalf("causa=%d esperada=%d estado=%d err=%v", causaDelFalloBarreraO3bM38(err), caso.causa, a.estado, err)
			}
			if identidadArchivoBarreraO3bPruebaM38(t, ticketFD) != antes || bytesTicketBarreraO3bPruebaM38(t, ticketFD) != 0 {
				t.Fatal("camino negativo tocó ticket")
			}
		})
	}
}

func TestBarreraO3bNoDuplicaPidfd(t *testing.T) {
	_, a := autoridadRealBarreraO3bPruebaM38(t)
	antes, err := snapshotActualO3aM38()
	if err != nil {
		t.Fatal(err)
	}
	if err := ejecutarBarreraO3bM38(a); err != nil {
		t.Fatal(err)
	}
	defer limpiarPermisoPrimeraEscrituraBarreraO3bPruebaM38(t, a)
	despues, err := snapshotActualO3aM38()
	if err != nil || !snapshotsIgualesO3aM38(antes, despues) {
		t.Fatalf("inventario cambió: %v", err)
	}
	if a.custodia.pidfdPrimario == a.custodia.pidfdReserva || a.custodia.pidfdPrimario == a.custodia.pidfdOpaco {
		t.Fatal(syscall.EBADF)
	}
}

func TestBarreraO3bControlCerrado(t *testing.T) {
	for _, caso := range []struct {
		nombre string
		datos  string
		eof    bool
	}{
		{"fragmento", "V1|", false},
		{"framing", "trama-invalida\n", false},
		{"eof", "", true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			f, a := autoridadRealBarreraO3bPruebaM38(t)
			ticketFD := int(a.custodia.ticketEscritor.Fd())
			if caso.eof {
				cerrado, err := cerrarUnoConLeaseO3aM38(a.custodia.lease, f.controlEscritor, operacionCerrarDestinosO3aM38)
				if !cerrado || err != nil {
					t.Fatalf("cerrar extremo CONTROL: cerrado=%t err=%v", cerrado, err)
				}
				f.controlEscritor = nil
			} else if err := escribirControlPruebaO3aM38(f, caso.datos); err != nil {
				t.Fatal(err)
			}
			if err := ejecutarBarreraO3bM38(a); causaDelFalloBarreraO3bM38(err) != barreraO3bControlM38 || !a.es(capturaB7RetirandoM38) {
				t.Fatalf("control aceptado: estado=%d err=%v", a.estado, err)
			}
			if bytesTicketBarreraO3bPruebaM38(t, ticketFD) != 0 {
				t.Fatal("control adverso escribió ticket")
			}
		})
	}
}

func TestBarreraO3bRelecturaFinalDetectaSenal(t *testing.T) {
	_, a := autoridadRealBarreraO3bPruebaM38(t)
	c := a.custodia
	if err := leerControlO3bM38(c); err != nil {
		t.Fatal(err)
	}
	if causa, err := revalidarAutoridadBarreraO3bM38(c); err != nil || causa != 0 {
		t.Fatalf("autoridad inicial: causa=%d err=%v", causa, err)
	}
	if causa, fatal, err := acreditarPidfdBarreraO3bM38(c); err != nil || causa != 0 || fatal {
		t.Fatalf("pidfd inicial: causa=%d fatal=%t err=%v", causa, fatal, err)
	}
	if err := acreditarInventarioBarreraO3bM38(a); err != nil {
		t.Fatal(err)
	}
	c.observador.palabra.Add(1 << 10)
	if err := leerControlO3bM38(c); err != nil {
		t.Fatal(err)
	}
	causa, err := revalidarAutoridadBarreraO3bM38(c)
	if err != nil || causa != barreraO3bSenalM38 {
		t.Fatalf("la relectura final no vio señal: causa=%d err=%v", causa, err)
	}
	_ = retirarBarreraO3bM38(a, causa)
}

func TestBarreraO3bRelecturaFinalDetectaControl(t *testing.T) {
	f, a := autoridadRealBarreraO3bPruebaM38(t)
	c := a.custodia
	if err := leerControlO3bM38(c); err != nil {
		t.Fatal(err)
	}
	if causa, err := revalidarAutoridadBarreraO3bM38(c); err != nil || causa != 0 {
		t.Fatalf("autoridad inicial: causa=%d err=%v", causa, err)
	}
	if causa, fatal, err := acreditarPidfdBarreraO3bM38(c); err != nil || causa != 0 || fatal {
		t.Fatalf("pidfd inicial: causa=%d fatal=%t err=%v", causa, fatal, err)
	}
	if err := acreditarInventarioBarreraO3bM38(a); err != nil {
		t.Fatal(err)
	}
	if err := escribirControlPruebaO3aM38(f, "V1|CONTROL|CANCELAR|"+string(c.control.nonce[:])+"|CANCELADO|65\n"); err != nil {
		t.Fatal(err)
	}
	if err := leerControlO3bM38(c); err == nil {
		t.Fatal("la relectura final no vio CONTROL")
	}
	_ = retirarBarreraO3bM38(a, barreraO3bControlM38)
}

func TestBarreraO3bSinReferenciaFiableEsFatal(t *testing.T) {
	if os.Getenv("O3B_P2_FATAL") != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestBarreraO3bSinReferenciaFiableEsFatal$")
		cmd.Env = append(os.Environ(), "O3B_P2_FATAL=1")
		err := cmd.Run()
		var salida *exec.ExitError
		if !errors.As(err, &salida) || salida.ExitCode() != estadoFallo {
			t.Fatalf("BF no terminó en %d: %v", estadoFallo, err)
		}
		return
	}
	_, a := autoridadRealBarreraO3bPruebaM38(t)
	a.custodia.pidfdPrimario, a.custodia.pidfdReserva = -1, -1
	_ = ejecutarBarreraO3bM38(a)
	t.Fatal("BF retornó")
}

func TestBarreraO3bErrorLecturaRetira(t *testing.T) {
	_, a := autoridadRealBarreraO3bPruebaM38(t)
	c := a.custodia
	cerrado, err := cerrarUnoConLeaseO3aM38(c.lease, c.controlFD, operacionCerrarDestinosO3aM38)
	if !cerrado || err != nil {
		t.Fatalf("cerrar CONTROL: cerrado=%t err=%v", cerrado, err)
	}
	defer func() { c.controlFD = nil }()
	err = ejecutarBarreraO3bM38(a)
	if causaDelFalloBarreraO3bM38(err) != barreraO3bControlM38 || !a.es(capturaB7RetirandoM38) {
		t.Fatalf("error de lectura no retiró: estado=%d err=%v", a.estado, err)
	}
}

func TestBarreraO3bRechazaCuartaReferenciaYRecursosCorruptos(t *testing.T) {
	t.Run("cuarta_pidfd", func(t *testing.T) {
		_, a := autoridadRealBarreraO3bPruebaM38(t)
		cuarta, err := syscall.Dup(a.custodia.pidfdPrimario)
		if err != nil {
			t.Fatal(err)
		}
		defer syscall.Close(cuarta)
		err = ejecutarBarreraO3bM38(a)
		if causaDelFalloBarreraO3bM38(err) != barreraO3bInventarioM38 || !a.es(capturaB7RetirandoM38) {
			t.Fatalf("cuarta referencia aceptada: estado=%d err=%v", a.estado, err)
		}
	})
	for _, recurso := range []string{"terminal_alias", "ticket_nulo", "ticket_mismo_tipo"} {
		t.Run(recurso, func(t *testing.T) {
			f, a := autoridadRealBarreraO3bPruebaM38(t)
			if recurso == "terminal_alias" {
				anterior := a.custodia.terminal
				a.custodia.terminal = a.custodia.ticketEscritor
				defer func() { a.custodia.terminal = anterior }()
			} else if recurso == "ticket_nulo" {
				anterior := a.custodia.ticketEscritor
				a.custodia.ticketEscritor = nil
				defer func() { a.custodia.ticketEscritor = anterior }()
			} else {
				anterior := a.custodia.ticketEscritor
				a.custodia.ticketEscritor = f.controlEscritor
				defer func() { a.custodia.ticketEscritor = anterior }()
			}
			err := ejecutarBarreraO3bM38(a)
			if causaDelFalloBarreraO3bM38(err) != barreraO3bInventarioM38 || !a.es(capturaB7RetirandoM38) {
				t.Fatalf("recurso corrupto aceptado: estado=%d err=%v", a.estado, err)
			}
		})
	}
}
