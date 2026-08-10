//go:build ignore && linux && amd64

package main

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func autoridadTicketO3bPruebaM38(t *testing.T, ticket string) (*autoridadCapturaO3bM38, *os.File) {
	t.Helper()
	c := custodiaNominalO3bPruebaM38(t)
	lector, escritor, err := os.Pipe()
	if err != nil {
		t.Fatalf("crear ticket: %v", err)
	}
	t.Cleanup(func() { _ = lector.Close(); _ = escritor.Close() })
	c.ticketEscritor = escritor
	c.ppid = os.Getpid()
	c.control.recepcion = &receptorSobreS0M38{fase: recepcionSobreS1M38, sobre: sobreRetenidoM38{ticket: ticket}}
	fisico, err := snapshotActualO3aM38()
	if err != nil {
		t.Fatalf("sellar inventario: %v", err)
	}
	c.lease.fisico = fisico
	a := &autoridadCapturaO3bM38{
		estado:     capturaB0RecibidoM38,
		custodia:   c,
		fdsBarrera: [3]int{int(c.controlFD.Fd()), int(c.terminal.Fd()), int(escritor.Fd())},
	}
	if err := prepararTicketO3bM38(a); err != nil {
		t.Fatalf("preparar ticket: %v", err)
	}
	permiso, valido := a.custodia.lease.comenzar(operacionEscribirTicketO3bM38, 0, [2]int{a.ticket.fd, -1})
	if !valido {
		t.Fatal("preparar permiso de primera escritura")
	}
	a.ticket.primerPermiso, a.ticket.permisoPrimero = permiso, true
	a.estado = capturaB1BarreraVerdeM38
	return a, lector
}

func TestTicketO3bByteExactoYUsoUnico(t *testing.T) {
	for _, ticket := range []string{"x", "uno|dos|tres", strings.Repeat("z", maximoTicketO3bM38)} {
		t.Run(strconv.Itoa(len(ticket)), func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			a, lector := autoridadTicketO3bPruebaM38(t, ticket)
			fd := a.ticket.fd
			if err := emitirYCerrarTicketO3bM38(a); err != nil || !a.es(capturaB2TicketCerradoM38) {
				t.Fatalf("emisión nominal: estado=%d err=%v", a.estado, err)
			}
			recibido, err := io.ReadAll(lector)
			esperado := strconv.Itoa(os.Getpid()) + "|" + ticket + "\n"
			if err != nil || string(recibido) != esperado {
				t.Fatalf("trama distinta: longitud=%d err=%v", len(recibido), err)
			}
			if a.custodia.ticketEscritor != nil || !a.ticket.cierreIntentado {
				t.Fatal("el escritor conservó propiedad tras el cierre")
			}
			if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0); errno != syscall.EBADF {
				t.Fatalf("el FD ticket sigue vivo: %v", errno)
			}
			if err := emitirYCerrarTicketO3bM38(a); err == nil || !a.es(capturaB7RetirandoM38) {
				t.Fatalf("segundo uso aceptado: estado=%d err=%v", a.estado, err)
			}
		})
	}
}

func TestTicketO3bPreparacionCerrada(t *testing.T) {
	for _, ticket := range []string{"", strings.Repeat("x", maximoTicketO3bM38+1), "linea\n", string([]byte{0x1f})} {
		t.Run(strconv.Itoa(len(ticket)), func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			c := custodiaNominalO3bPruebaM38(t)
			c.ppid = os.Getpid()
			c.control.recepcion = &receptorSobreS0M38{fase: recepcionSobreS1M38, sobre: sobreRetenidoM38{ticket: ticket}}
			fisico, err := snapshotActualO3aM38()
			if err != nil {
				t.Fatal(err)
			}
			c.lease.fisico = fisico
			a := &autoridadCapturaO3bM38{estado: capturaB0RecibidoM38, custodia: c,
				fdsBarrera: [3]int{int(c.controlFD.Fd()), int(c.terminal.Fd()), int(c.ticketEscritor.Fd())}}
			fd := int(c.ticketEscritor.Fd())
			if err := prepararTicketO3bM38(a); !errors.Is(err, errTicketO3bM38) || a.ticket == nil {
				t.Fatalf("ticket adverso aceptado: %v", err)
			}
			if err := retirarTicketO3bM38(a, errTicketO3bM38); !errors.Is(err, errTicketO3bM38) ||
				a.estado != capturaB7RetirandoM38 || c.ticketEscritor != nil || !a.ticket.cierreIntentado {
				t.Fatalf("preparación adversa no convergió: estado=%d err=%v", a.estado, err)
			}
			if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0); errno != syscall.EBADF {
				t.Fatalf("la retirada dejó vivo el escritor: %v", errno)
			}
		})
	}
}

func TestTicketO3bPreparadoAusenteFatal(t *testing.T) {
	if os.Getenv("O3B_P3_FATAL_SIN_PREPARADO") == "1" {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		a, _ := autoridadTicketO3bPruebaM38(t, "x")
		a.ticket = nil
		_ = emitirYCerrarTicketO3bM38(a)
		os.Exit(99)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestTicketO3bPreparadoAusenteFatal$")
	cmd.Env = append(os.Environ(), "O3B_P3_FATAL_SIN_PREPARADO=1")
	err := cmd.Run()
	if salida, ok := err.(*exec.ExitError); !ok || salida.ExitCode() != estadoFallo {
		t.Fatalf("combinación imposible no fatalizó: %v", err)
	}
}

func TestTicketO3bEscrituraParcialYEINTRAcotado(t *testing.T) {
	restantes, interrupciones := 7, 0
	for _, n := range []int{1, 2, 4} {
		avance, nuevas, err := aplicarResultadoEscrituraO3bM38(restantes, n, interrupciones, nil)
		if err != nil || avance != n || nuevas != interrupciones {
			t.Fatalf("parcial %d no avanzó: avance=%d interrupciones=%d err=%v", n, avance, nuevas, err)
		}
		restantes -= avance
	}
	if restantes != 0 {
		t.Fatalf("escritura parcial quedó corta: %d", restantes)
	}
	if avance, nuevas, err := aplicarResultadoEscrituraO3bM38(3, 2, 0, syscall.EINTR); err != nil || avance != 2 || nuevas != 1 {
		t.Fatalf("EINTR con bytes perdió avance: avance=%d nuevas=%d err=%v", avance, nuevas, err)
	}
	for i := 1; i <= 8; i++ {
		avance, nuevas, err := aplicarResultadoEscrituraO3bM38(1, 0, interrupciones, syscall.EINTR)
		if err != nil || avance != 0 || nuevas != i {
			t.Fatalf("EINTR %d rechazado: avance=%d nuevas=%d err=%v", i, avance, nuevas, err)
		}
		interrupciones = nuevas
	}
	if _, nuevas, err := aplicarResultadoEscrituraO3bM38(1, 0, interrupciones, syscall.EINTR); !errors.Is(err, syscall.EINTR) || nuevas != 9 {
		t.Fatalf("noveno EINTR aceptado: interrupciones=%d err=%v", nuevas, err)
	}
	for _, caso := range []struct {
		n   int
		err error
	}{{0, nil}, {2, nil}, {0, syscall.EPIPE}, {0, syscall.EAGAIN}} {
		if _, _, err := aplicarResultadoEscrituraO3bM38(1, caso.n, 0, caso.err); err == nil {
			t.Fatalf("resultado adverso aceptado: n=%d err=%v", caso.n, caso.err)
		}
	}
}

func TestTicketO3bFalloCierraUnaVezSinSegundaEmision(t *testing.T) {
	for _, caso := range []string{"epipe", "ebadf"} {
		t.Run(caso, func(t *testing.T) {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			a, lector := autoridadTicketO3bPruebaM38(t, "sensible|opaco")
			fd := a.ticket.fd
			switch caso {
			case "epipe":
				if err := lector.Close(); err != nil {
					t.Fatal(err)
				}
			case "ebadf":
				if err := syscall.Close(fd); err != nil {
					t.Fatal(err)
				}
			}
			err := emitirYCerrarTicketO3bM38(a)
			if err == nil || !a.es(capturaB7RetirandoM38) || a.custodia.ticketEscritor != nil || !a.ticket.cierreIntentado {
				t.Fatalf("fallo no convergió: estado=%d writer=%v err=%v", a.estado, a.custodia.ticketEscritor, err)
			}
			if segundo := emitirYCerrarTicketO3bM38(a); segundo == nil {
				t.Fatal("fallo permitió segunda emisión")
			}
		})
	}
}

func TestTicketO3bCierreFallidoNoSeReintenta(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	a, _ := autoridadTicketO3bPruebaM38(t, "x")
	fd := a.ticket.fd
	if !a.custodia.lease.consolidarCritico(a.ticket.primerPermiso) {
		t.Fatal("limpiar permiso test-only")
	}
	a.ticket.permisoPrimero = false
	if err := syscall.Close(fd); err != nil {
		t.Fatal(err)
	}
	if err := cerrarTicketO3bM38(a); !errors.Is(err, syscall.EBADF) {
		t.Fatalf("EBADF no fue cierre fallido: %v", err)
	}
	if a.custodia.ticketEscritor != nil || !a.ticket.cierreIntentado {
		t.Fatal("cierre fallido conservó propiedad")
	}
	reutilizado, err := os.Open("/dev/null")
	if err != nil {
		t.Fatal(err)
	}
	defer reutilizado.Close()
	if int(reutilizado.Fd()) != fd {
		t.Fatalf("el kernel no reutilizó el FD cerrado: anterior=%d nuevo=%d", fd, reutilizado.Fd())
	}
	primera := errors.New("primera")
	if err = retirarTicketO3bM38(a, primera); err != primera || a.estado != capturaB7RetirandoM38 {
		t.Fatalf("retirada sustituyó causa: estado=%d err=%v", a.estado, err)
	}
	if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, reutilizado.Fd(), syscall.F_GETFD, 0); errno != 0 {
		t.Fatalf("retirada cerró el FD reutilizado: %v", errno)
	}
}
