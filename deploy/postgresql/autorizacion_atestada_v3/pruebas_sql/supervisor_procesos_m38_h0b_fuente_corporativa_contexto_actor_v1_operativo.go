//go:build ignore && linux && amd64

// Primitivas Linux compartidas del supervisor probatorio M38.
package main

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// supervisarM38 permanece cerrado hasta que G2-O aporte el protocolo revisado.
func supervisarM38() int {
	return estadoUso
}

// tramaM38 representa una trama completa ya validada; no conserva estado ni FDs.
type tramaM38 struct {
	clase  string
	campos []string
	ticket string
}

func limiteTramaM38(clase string) int {
	switch clase {
	case "SOBRE":
		return 4096
	case "CONTROL", "TERMINAL", "VALIDADA":
		return 1024
	case "TICKET":
		return 2060
	}
	return 0
}

func decimalM38(valor string, minimo, maximo uint64) bool {
	if valor == "" || (len(valor) > 1 && valor[0] == '0') {
		return false
	}
	n, err := strconv.ParseUint(valor, 10, 64)
	return err == nil && n >= minimo && n <= maximo
}

func hexM38(valor string) bool {
	if len(valor) != 64 {
		return false
	}
	for _, b := range []byte(valor) {
		if !(b >= '0' && b <= '9' || b >= 'a' && b <= 'f') {
			return false
		}
	}
	return true
}

func seguroM38(valor string) bool {
	for _, b := range []byte(valor) {
		if b < 0x20 || b > 0x7e || b == '\t' || b == '\r' || b == '\n' || b == 0 {
			return false
		}
	}
	return true
}

func indicadorM38(valor string) bool { return valor == "0" || valor == "1" }

func selectorM38(valor string) bool {
	if valor == "NOMINAL" {
		return true
	}
	return len(valor) == 3 && valor[0] >= 'A' && valor[0] <= 'Z' && valor[1] >= '0' && valor[1] <= '9' && valor[2] >= '0' && valor[2] <= '9'
}

func causaEstadoM38(causa, estado string, terminal bool) bool {
	if !terminal {
		return (causa == "CANCELADO" || causa == "PROTOCOLO") && estado == "65" || causa == "SENAL_INT" && estado == "130" || causa == "SENAL_TERM" && estado == "143"
	}
	if causa == "SALIDA" {
		return estado == "0" || estado == "64" || estado == "65" || estado == "79"
	}
	if causa == "CANCELADO" || causa == "PLAZO" || causa == "PROTOCOLO" || causa == "INCIDENTE" {
		return estado == "65"
	}
	return causa == "SENAL_INT" && estado == "130" || causa == "SENAL_TERM" && estado == "143"
}

func codificarTramaM38(t tramaM38) ([]byte, error) {
	var texto string
	switch t.clase {
	case "SOBRE":
		if len(t.campos) != 5 {
			return nil, errors.New("sobre incompleto")
		}
		texto = "V1|SOBRE|" + strings.Join(t.campos, "|") + "|" + t.ticket
	case "TICKET":
		if len(t.campos) != 1 {
			return nil, errors.New("ticket incompleto")
		}
		texto = t.campos[0] + "|" + t.ticket
	case "CONTROL", "TERMINAL":
		texto = "V1|" + t.clase + "|" + strings.Join(t.campos, "|")
	default:
		return nil, errors.New("clase de trama desconocida")
	}
	b := append([]byte(texto), '\n')
	_, err := decodificarTramaM38(t.clase, b)
	return b, err
}

func decodificarTramaM38(clase string, entrada []byte) (tramaM38, error) {
	if limite := limiteTramaM38(clase); limite == 0 || len(entrada) == 0 || len(entrada) > limite || entrada[len(entrada)-1] != '\n' {
		return tramaM38{}, errors.New("longitud o terminador de trama inválido")
	}
	for _, b := range entrada[:len(entrada)-1] {
		if b < 0x20 || b > 0x7e || b == '\t' || b == '\r' {
			return tramaM38{}, errors.New("byte de trama inválido")
		}
	}
	texto := string(entrada[:len(entrada)-1])
	partes := strings.Split(texto, "|")
	if clase == "SOBRE" {
		return sobreM38(texto)
	}
	if clase == "TICKET" {
		return ticketM38(texto)
	}
	if len(partes) < 2 || partes[0] != "V1" || partes[1] != clase {
		return tramaM38{}, errors.New("versión o clase inválida")
	}
	switch clase {
	case "CONTROL":
		if err := controlM38(partes); err != nil {
			return tramaM38{}, err
		}
	case "TERMINAL":
		if err := terminalM38(partes); err != nil {
			return tramaM38{}, err
		}
	default:
		return tramaM38{}, errors.New("clase no decodificable")
	}
	return tramaM38{clase: clase, campos: partes[2:]}, nil
}

func sobreM38(texto string) (tramaM38, error) {
	p := strings.SplitN(texto, "|", 8)
	if len(p) != 8 || p[0] != "V1" || p[1] != "SOBRE" || !hexM38(p[2]) || !decimalM38(p[3], 1, 2147483647) || !selectorM38(p[4]) || !hexM38(p[5]) || !decimalM38(p[6], 1, 2048) || !seguroM38(p[7]) || uint64(len(p[7])) != numeroM38(p[6]) {
		return tramaM38{}, errors.New("sobre inválido")
	}
	return tramaM38{clase: "SOBRE", campos: p[2:7], ticket: p[7]}, nil
}

func ticketM38(texto string) (tramaM38, error) {
	p := strings.SplitN(texto, "|", 2)
	if len(p) != 2 || !decimalM38(p[0], 1, 2147483647) || !seguroM38(p[1]) || len(p[1]) == 0 || len(p[1]) > 2048 {
		return tramaM38{}, errors.New("ticket inválido")
	}
	return tramaM38{clase: "TICKET", campos: p[:1], ticket: p[1]}, nil
}

func numeroM38(valor string) uint64 { n, _ := strconv.ParseUint(valor, 10, 64); return n }

func controlM38(p []string) error {
	if len(p) == 5 && p[2] == "ARMAR" {
		if hexM38(p[3]) && decimalM38(p[4], 1, 2147483647) {
			return nil
		}
	}
	if len(p) == 4 && p[2] == "INICIAR" && hexM38(p[3]) {
		return nil
	}
	if len(p) == 6 && p[2] == "CANCELAR" && hexM38(p[3]) && causaEstadoM38(p[4], p[5], false) {
		return nil
	}
	return errors.New("control inválido")
}

func terminalM38(p []string) error {
	if len(p) != 16 || !hexM38(p[2]) || !decimalM38(p[3], 1, 2147483647) || (p[4] != "S1" && p[4] != "S2" && p[4] != "S3" && p[4] != "S4") || !causaEstadoM38(p[6], p[5], true) || !indicadorM38(p[12]) || !indicadorM38(p[13]) || !decimalM38(p[14], 0, 2147483647) || !indicadorM38(p[15]) {
		return errors.New("terminal inválido")
	}
	sinBash := p[7] == "-" && p[8] == "-" && p[9] == "-" && p[10] == "-" && p[11] == "-" && p[12] == "0" && p[13] == "0" && p[14] == "0" && p[15] == "1"
	conBash := decimalM38(p[7], 1, 2147483647) && decimalM38(p[8], 1, 2147483647) && decimalM38(p[9], 1, 2147483647) && decimalM38(p[10], 1, 2147483647) && decimalM38(p[11], 1, 18446744073709551615) && p[12] == "1" && p[13] == "1" && p[14] == "0" && p[15] == "1"
	if !(sinBash || conBash) || p[6] == "SALIDA" && (!conBash || (p[4] != "S3" && p[4] != "S4")) || conBash && p[4] != "S3" && p[4] != "S4" || sinBash && p[4] == "S1" && (p[6] == "SENAL_INT" || p[6] == "SENAL_TERM") {
		return errors.New("coherencia terminal inválida")
	}
	return nil
}

func autoprobarTramasM38() error {
	h := strings.Repeat("a", 64)
	sobre := "V1|SOBRE|" + h + "|12|A01|" + h + "|5|x|y|z\n"
	terminal := "V1|TERMINAL|" + h + "|12|S3|0|SALIDA|13|12|12|12|7|1|1|0|1\n"
	validas := []struct{ clase, texto string }{{"SOBRE", sobre}, {"CONTROL", "V1|CONTROL|ARMAR|" + h + "|12\n"}, {"CONTROL", "V1|CONTROL|INICIAR|" + h + "\n"}, {"CONTROL", "V1|CONTROL|CANCELAR|" + h + "|SENAL_INT|130\n"}, {"TERMINAL", terminal}, {"TICKET", "12|x|y\n"}}
	for _, prueba := range validas {
		t, err := decodificarTramaM38(prueba.clase, []byte(prueba.texto))
		if err != nil {
			return err
		}
		salida, err := codificarTramaM38(t)
		if err != nil || string(salida) != prueba.texto {
			return errors.New("ida y vuelta no canónica")
		}
	}
	invalidas := []struct{ clase, texto string }{{"SOBRE", strings.Replace(sobre, "|5|", "|4|", 1)}, {"SOBRE", strings.Replace(sobre, "a", "A", 1)}, {"SOBRE", strings.Replace(sobre, "\n", "\r\n", 1)}, {"CONTROL", "V1|CONTROL|CANCELAR|" + h + "|SALIDA|0\n"}, {"CONTROL", "V1|CONTROL|ARMAR|" + h + "|01\n"}, {"TERMINAL", strings.Replace(terminal, "|S3|0|SALIDA|", "|S1|0|SALIDA|", 1)}, {"TERMINAL", strings.Replace(terminal, "|13|12|12|12|7|1|1|0|1", "|-|-|-|-|-|1|1|0|1", 1)}, {"TICKET", "0|x\n"}, {"TICKET", "12|x\x00\n"}, {"TICKET", "12|x"}}
	for _, prueba := range invalidas {
		if _, err := decodificarTramaM38(prueba.clase, []byte(prueba.texto)); err == nil {
			return errors.New("mutante de trama aceptado")
		}
	}
	for _, clase := range []string{"SOBRE", "CONTROL", "TERMINAL", "TICKET"} {
		limite := limiteTramaM38(clase)
		for _, longitud := range []int{limite - 1, limite, limite + 1} {
			entrada := append([]byte(strings.Repeat("x", longitud-1)), '\n')
			if _, err := decodificarTramaM38(clase, entrada); err == nil {
				return errors.New("frontera de longitud aceptada")
			}
		}
	}
	return nil
}

func duplicarPidfd(pidfd int) (int, error) {
	descriptor, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(pidfd), uintptr(syscall.F_DUPFD_CLOEXEC), 0)
	if errno != 0 {
		return -1, errno
	}
	if descriptor > uintptr(^uint(0)>>1) {
		_ = syscall.Close(int(descriptor))
		return -1, errors.New("duplicado pidfd fuera de rango")
	}
	return int(descriptor), nil
}

func esperarTerminal(pidfd int) error {
	type pollfd struct {
		fd               int32
		eventos, retorno int16
	}
	p := pollfd{fd: int32(pidfd), eventos: 1}
	fin := time.Now().Add(tiempoEspera)
	for {
		restante := time.Until(fin)
		if restante <= 0 {
			return errors.New("plazo de terminalidad agotado")
		}
		milisegundos := restante.Milliseconds()
		if milisegundos == 0 {
			milisegundos = 1
		}
		p.retorno = 0
		n, _, errno := syscall.Syscall(syscall.SYS_POLL, uintptr(unsafe.Pointer(&p)), 1, uintptr(milisegundos))
		if errno == syscall.EINTR {
			continue
		}
		if errno != 0 || n != 1 || p.retorno&1 == 0 {
			return fmt.Errorf("terminalidad no acreditada: n=%d eventos=%x errno=%v", n, p.retorno, errno)
		}
		return nil
	}
}

func activarSubreaper() error {
	if _, _, e := syscall.Syscall6(syscall.SYS_PRCTL, 36, 1, 0, 0, 0, 0); e != 0 {
		return e
	}
	var valor int32
	_, _, e := syscall.Syscall6(syscall.SYS_PRCTL, 37, uintptr(unsafe.Pointer(&valor)), 0, 0, 0, 0)
	if e != 0 || valor != 1 {
		return errors.New("subreaper no acreditado")
	}
	return nil
}

func contarFD() (int, error) {
	entradas, err := os.ReadDir("/proc/self/fd")
	return len(entradas), err
}

func exigirESRCH(pidfd int) error {
	err := enviar(pidfd, 0, pidfdSignalProcessGroup)
	if !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("grupo terminal: se esperaba ESRCH y se obtuvo %v", err)
	}
	return nil
}

func enviar(pidfd int, senal syscall.Signal, banderas uintptr) error {
	_, _, errno := syscall.Syscall6(sysPidfdSendSignal, uintptr(pidfd), uintptr(senal), 0, banderas, 0, 0)
	if errno != 0 {
		return errno
	}
	return nil
}
