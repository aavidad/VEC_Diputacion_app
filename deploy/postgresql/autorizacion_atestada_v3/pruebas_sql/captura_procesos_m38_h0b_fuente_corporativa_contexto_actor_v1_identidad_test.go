//go:build ignore && linux && amd64

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func statIdentidadPruebaO3bM38(pid, ppid, pgid, sid int, inicio string) []byte {
	campos := make([]string, camposStatO3bM38-3)
	for i := range campos {
		campos[i] = "1"
	}
	campos[0], campos[1], campos[2], campos[18] = fmt.Sprint(ppid), fmt.Sprint(pgid), fmt.Sprint(sid), inicio
	return []byte(fmt.Sprintf("%d (nombre ) con espacios) T %s\n", pid, strings.Join(campos, " ")))
}

func mutarCampoIdentidadStatPruebaO3bM38(datos []byte, campo int, valor string) []byte {
	cierre := strings.LastIndex(string(datos), ") ")
	if campo == 0 {
		apertura := strings.Index(string(datos[:cierre]), " (")
		return []byte(valor + string(datos[apertura:]))
	}
	prefijo := string(datos[:cierre+4])
	resto := strings.Split(string(datos[cierre+4:len(datos)-1]), " ")
	indices := map[int]int{1: 0, 2: 1, 3: 2, 4: 18}
	resto[indices[campo]] = valor
	return []byte(prefijo + strings.Join(resto, " ") + "\n")
}

func TestParsearStatO3bAcotadoYAdversarial(t *testing.T) {
	nominal := statIdentidadPruebaO3bM38(41, 7, 41, 3, "99")
	muestra, err := parsearStatO3bM38(nominal)
	if err != nil || muestra != (muestraStatO3bM38{pid: 41, estado: 'T', ppid: 7, pgid: 41, sid: 3, inicio: 99}) {
		t.Fatalf("stat nominal: muestra=%+v err=%v", muestra, err)
	}
	cortado := append([]byte(nil), nominal[:len(nominal)-3]...)
	extra := append(append([]byte(nil), nominal[:len(nominal)-1]...), []byte(" 1\n")...)
	demasiado := append([]byte(nil), nominal...)
	demasiado = append(demasiado[:len(demasiado)-1], make([]byte, maximoStatO3bM38-len(demasiado)+2)...)
	demasiado = append(demasiado, '\n')
	ceroInicial := statIdentidadPruebaO3bM38(41, 7, 41, 3, "099")
	signo := statIdentidadPruebaO3bM38(41, 7, 41, 3, "+99")
	desborde := statIdentidadPruebaO3bM38(41, 7, 41, 3, "18446744073709551616")
	noDecimal := append([]byte(nil), nominal...)
	partes := strings.Split(string(noDecimal), " ")
	partes[len(partes)-3] = "no_decimal"
	noDecimal = []byte(strings.Join(partes, " "))
	tabulado := []byte(strings.Replace(string(nominal), " 1 1 ", " 1\t1 ", 1))
	dobleEspacio := []byte(strings.Replace(string(nominal), " 1 1 ", " 1  1 ", 1))
	for nombre, datos := range map[string][]byte{
		"truncado": cortado, "extra": extra, "mayor_4096": demasiado,
		"cero_inicial": ceroInicial, "signo": signo, "desborde": desborde, "campo_no_decimal": noDecimal,
		"tabulado": tabulado, "doble_espacio": dobleEspacio,
	} {
		t.Run(nombre, func(t *testing.T) {
			if _, err := parsearStatO3bM38(datos); err == nil {
				t.Fatal("stat adverso aceptado")
			}
		})
	}
	for campo := 0; campo < 5; campo++ {
		for _, adverso := range []string{"-1", "01", "18446744073709551616"} {
			nombre := fmt.Sprintf("identidad_%d_%s", campo, adverso)
			t.Run(nombre, func(t *testing.T) {
				if _, err := parsearStatO3bM38(mutarCampoIdentidadStatPruebaO3bM38(nominal, campo, adverso)); err == nil {
					t.Fatal("campo de identidad adverso aceptado")
				}
			})
		}
	}
}

func TestIdentidadO3bExactaReal(t *testing.T) {
	if os.Getenv("O3B_P5_HIJO") == "1" {
		_, a := autoridadRealBarreraO3bPruebaM38(t)
		a.custodia.control.recepcion.sobre.ticket = ticketRealStopO3bPruebaM38(a)
		if err := ejecutarBarreraO3bM38(a); err != nil || emitirYCerrarTicketO3bM38(a) != nil {
			os.Exit(2)
		}
		if err := observarStopO3bM38(a); err != nil {
			os.Exit(3)
		}
		identidad, err := acreditarIdentidadO3bM38(a)
		if err != nil || identidad == nil || identidad.autoridad != a || identidad.inicio == 0 ||
			identidad.pid != a.custodia.cmd.Process.Pid || identidad.ppid != os.Getpid() ||
			identidad.pgid != identidad.pid || identidad.sid != a.sidSupervisor || !a.es(capturaB4IdentidadAcreditadaM38) {
			os.Exit(4)
		}
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestIdentidadO3bExactaReal$")
	cmd.Env = append(os.Environ(), "O3B_P5_HIJO=1")
	if salida, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("identidad real: %v: %s", err, salida)
	}
}

func TestIdentidadO3bRechazaCampos(t *testing.T) {
	a := &autoridadCapturaO3bM38{sidSupervisor: 9, custodia: &custodiaO3aM38{cmd: &exec.Cmd{Process: &os.Process{Pid: 41}}}}
	nominal := muestraStatO3bM38{pid: 41, estado: 'T', ppid: os.Getpid(), pgid: 41, sid: 9, inicio: 22}
	if !muestraIdentidadEsperadaO3bM38(a, nominal, 22) {
		t.Fatal("identidad nominal rechazada")
	}
	casos := []muestraStatO3bM38{nominal, nominal, nominal, nominal, nominal, nominal}
	casos[0].pid++
	casos[1].estado = 'Z'
	casos[2].ppid++
	casos[3].pgid++
	casos[4].sid++
	casos[5].inicio++
	for i, caso := range casos {
		if muestraIdentidadEsperadaO3bM38(a, caso, 22) {
			t.Fatalf("campo adverso %d aceptado", i)
		}
	}
}
