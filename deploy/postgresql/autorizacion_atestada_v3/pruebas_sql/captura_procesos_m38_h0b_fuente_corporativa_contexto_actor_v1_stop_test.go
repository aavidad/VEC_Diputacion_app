//go:build ignore && linux && amd64

package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func ticketRealStopO3bPruebaM38(a *autoridadCapturaO3bM38) string {
	h := strings.Repeat("a", 64)
	r := a.custodia.formaRunner
	z := a.custodia.formaRaiz.identidad
	forma := fmt.Sprintf("1|1|%d|directory|700|2", os.Geteuid())
	runner := fmt.Sprintf("%d|%d|%d|regular file|%o|%d|%d|%s", r.identidad.dev, r.identidad.ino,
		r.identidad.uid, r.identidad.modo&0777, 0, r.identidad.tamano, hex.EncodeToString(r.sha256[:]))
	raiz := fmt.Sprintf("%d|%d|%d|directory|%o|%d", z.dev, z.ino, z.uid, z.modo&0777, z.enlaces)
	return "NOMINAL|" + h + "|" + h + "|" + forma + "|" + runner + "|" + raiz
}

func autoridadTicketCerradoStopO3bPruebaM38(t *testing.T) *autoridadCapturaO3bM38 {
	t.Helper()
	_, a := autoridadRealBarreraO3bPruebaM38(t)
	if err := ejecutarBarreraO3bM38(a); err != nil {
		t.Fatalf("barrera: %v", err)
	}
	if err := emitirYCerrarTicketO3bM38(a); err != nil || !a.es(capturaB2TicketCerradoM38) {
		t.Fatalf("ticket: estado=%d err=%v", a.estado, err)
	}
	return a
}

func TestStopO3bAutoStopRealSinSenalGo(t *testing.T) {
	if os.Getenv("O3B_P4_HIJO") == "1" {
		_, a := autoridadRealBarreraO3bPruebaM38(t)
		if err := sondaGrupoCeroO3bM38(a.custodia, a.custodia.pidfdPrimario); err != nil {
			os.Exit(5)
		}
		antes, err := leerStatStopO3bM38(a.custodia)
		if err != nil || muestraTStopO3bM38(antes) {
			os.Exit(6)
		}
		a.custodia.control.recepcion.sobre.ticket = ticketRealStopO3bPruebaM38(a)
		if err := ejecutarBarreraO3bM38(a); err != nil || emitirYCerrarTicketO3bM38(a) != nil {
			os.Exit(2)
		}
		baseline := a.custodia.baselineSenal
		if err := observarStopO3bM38(a); err != nil || !a.es(capturaB3StopObservadoM38) {
			os.Exit(3)
		}
		actual, senal, valido := a.custodia.observador.observar()
		if !valido || actual != baseline || senal != 0 {
			os.Exit(4)
		}
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestStopO3bAutoStopRealSinSenalGo$")
	cmd.Env = append(os.Environ(), "O3B_P4_HIJO=1")
	if salida, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("auto-STOP real: %v: %s", err, salida)
	}
}

func TestStopO3bMuestrasYPlazo(t *testing.T) {
	if !muestraTStopO3bM38([]byte("7 (a ) b) T 1\n")) || muestraTStopO3bM38([]byte("7 (a) T")) {
		t.Fatal("clasificación T no cerrada")
	}
	for _, estado := range []byte{'R', 'S', 'D', 'I', 'Z', 'X'} {
		if muestraTStopO3bM38([]byte(fmt.Sprintf("7 (a) %c 1\n", estado))) {
			t.Fatalf("estado adverso aceptado: %c", estado)
		}
	}
	ahora := time.Now()
	if finStopO3bM38(ahora, ahora.Add(2*time.Second)) != ahora.Add(time.Second) ||
		finStopO3bM38(ahora, ahora.Add(time.Second/2)) != ahora.Add(time.Second/2) {
		t.Fatal("finStop no respeta el mínimo")
	}
}

func TestStopO3bNegativosCeroTicket(t *testing.T) {
	casoHijo := os.Getenv("O3B_P4_NEGATIVO")
	if casoHijo != "" {
		_, a := autoridadRealBarreraO3bPruebaM38(t)
		a.custodia.control.recepcion.sobre.ticket = ticketRealStopO3bPruebaM38(a)
		if ejecutarBarreraO3bM38(a) != nil || emitirYCerrarTicketO3bM38(a) != nil {
			os.Exit(2)
		}
		if casoHijo == "bootstrap" {
			a.custodia.finBootstrap = time.Now()
		} else {
			a.custodia.observador.anotar(syscall.SIGTERM)
		}
		err := observarStopO3bM38(a)
		if err == nil || !a.es(capturaB7RetirandoM38) || a.custodia.ticketEscritor != nil ||
			!errors.As(err, new(*falloBarreraO3bM38)) {
			os.Exit(3)
		}
		os.Exit(0)
	}
	for _, caso := range []string{"bootstrap", "observador"} {
		t.Run(caso, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestStopO3bNegativosCeroTicket$")
			cmd.Env = append(os.Environ(), "O3B_P4_NEGATIVO="+caso)
			if salida, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("negativo %s: %v: %s", caso, err, salida)
			}
		})
	}
}
