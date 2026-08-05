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
	if clase == "SOBRE" {
		return 4096
	}
	if clase == "TICKET" {
		return 2060
	}
	if clase == "CONTROL" || clase == "TERMINAL" || clase == "VALIDADA" {
		return 1024
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
	switch causa {
	case "SALIDA":
		return terminal && (estado == "0" || estado == "64" || estado == "65" || estado == "79")
	case "CANCELADO", "PROTOCOLO":
		return estado == "65"
	case "PLAZO", "INCIDENTE":
		return terminal && estado == "65"
	case "SENAL_INT":
		return estado == "130"
	case "SENAL_TERM":
		return estado == "143"
	}
	return false
}

func prevalidarCodificacionM38(t tramaM38) error {
	limite := limiteTramaM38(t.clase)
	if len(t.ticket) > limite {
		return errors.New("trama demasiado grande")
	}
	base, separadores, cardinalidadValida := 1, len(t.campos)-1, false
	switch t.clase {
	case "SOBRE":
		cardinalidadValida = len(t.campos) == 5
		base += len("V1|SOBRE|") + len(t.ticket)
		separadores++
	case "TICKET":
		cardinalidadValida = len(t.campos) == 1
		base += len(t.ticket)
		separadores++
	case "CONTROL":
		if len(t.campos) > 0 {
			esperada := map[string]int{"ARMAR": 3, "INICIAR": 2, "CANCELAR": 4}[t.campos[0]]
			cardinalidadValida = esperada > 0 && len(t.campos) == esperada
		}
		base += len("V1|CONTROL|")
	case "TERMINAL":
		cardinalidadValida = len(t.campos) == 14
		base += len("V1|TERMINAL|")
	}
	if !cardinalidadValida || limite == 0 || separadores < 0 {
		return errors.New("cardinalidad de trama inválida")
	}
	restante := limite - base - separadores
	for _, campo := range t.campos {
		if len(campo) > restante {
			return errors.New("trama demasiado grande")
		}
		restante -= len(campo)
	}
	return nil
}

func codificarTramaM38(t tramaM38) ([]byte, error) {
	if err := prevalidarCodificacionM38(t); err != nil {
		return nil, err
	}
	var texto string
	switch t.clase {
	case "SOBRE":
		texto = "V1|SOBRE|" + strings.Join(t.campos, "|") + "|" + t.ticket
	case "TICKET":
		texto = t.campos[0] + "|" + t.ticket
	case "CONTROL", "TERMINAL":
		texto = "V1|" + t.clase + "|" + strings.Join(t.campos, "|")
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
	var err error
	switch clase {
	case "CONTROL":
		err = controlM38(partes)
	case "TERMINAL":
		err = terminalM38(partes)
	default:
		return tramaM38{}, errors.New("clase no decodificable")
	}
	if err != nil {
		return tramaM38{}, err
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
	if !(sinBash || conBash) || p[6] == "SALIDA" && (!conBash || (p[4] != "S3" && p[4] != "S4")) || conBash && p[4] != "S3" && p[4] != "S4" || sinBash && p[4] == "S4" || sinBash && p[4] == "S1" && (p[6] == "SENAL_INT" || p[6] == "SENAL_TERM") {
		return errors.New("coherencia terminal inválida")
	}
	return nil
}

func autoprobarTramasM38() error {
	h := strings.Repeat("a", 64)
	conBash := "13|12|12|12|7|1|1|0|1"
	sinBash := "-|-|-|-|-|0|0|0|1"
	sobre := func(selector, longitud, ticket string) string {
		return "V1|SOBRE|" + h + "|2147483647|" + selector + "|" + h + "|" + longitud + "|" + ticket + "\n"
	}
	terminal := func(fase, estado, causa, bloque string) string {
		return "V1|TERMINAL|" + h + "|2147483647|" + fase + "|" + estado + "|" + causa + "|" + bloque + "\n"
	}
	maximoInicio := "13|12|12|12|18446744073709551615|1|1|0|1"
	validas := []struct{ clase, texto string }{
		{"SOBRE", sobre("NOMINAL", "5", "x|y|z")}, {"SOBRE", sobre("Z99", "1", "x")},
		{"SOBRE", sobre("A00", "2048", strings.Repeat("x", 2048))},
		{"CONTROL", "V1|CONTROL|ARMAR|" + h + "|2147483647\n"}, {"CONTROL", "V1|CONTROL|INICIAR|" + h + "\n"},
		{"TICKET", "2147483647|" + strings.Repeat("x", 2048) + "\n"},
		{"TERMINAL", terminal("S3", "0", "SALIDA", maximoInicio)},
	}
	terminales := [][2]string{{"SALIDA", "0"}, {"SALIDA", "64"}, {"SALIDA", "65"}, {"SALIDA", "79"}, {"CANCELADO", "65"}, {"PLAZO", "65"}, {"PROTOCOLO", "65"}, {"INCIDENTE", "65"}, {"SENAL_INT", "130"}, {"SENAL_TERM", "143"}}
	for _, par := range terminales {
		for _, fase := range []string{"S3", "S4"} {
			validas = append(validas, struct{ clase, texto string }{"TERMINAL", terminal(fase, par[1], par[0], conBash)})
		}
		if par[0] == "SALIDA" {
			continue
		}
		for _, fase := range []string{"S2", "S3"} {
			validas = append(validas, struct{ clase, texto string }{"TERMINAL", terminal(fase, par[1], par[0], sinBash)})
		}
		if par[0] != "SENAL_INT" && par[0] != "SENAL_TERM" {
			validas = append(validas, struct{ clase, texto string }{"TERMINAL", terminal("S1", par[1], par[0], sinBash)})
		}
	}
	for _, par := range [][2]string{{"CANCELADO", "65"}, {"PROTOCOLO", "65"}, {"SENAL_INT", "130"}, {"SENAL_TERM", "143"}} {
		validas = append(validas, struct{ clase, texto string }{"CONTROL", "V1|CONTROL|CANCELAR|" + h + "|" + par[0] + "|" + par[1] + "\n"})
	}
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
	invalidas := []struct{ clase, texto string }{
		{"SOBRE", sobre("A00", "2048", strings.Repeat("x", 2049))}, {"SOBRE", sobre("A0", "1", "x")},
		{"SOBRE", sobre("a01", "1", "x")}, {"SOBRE", strings.Replace(sobre("A00", "1", "x"), h, strings.ToUpper(h), 1)},
		{"SOBRE", strings.Replace(sobre("A00", "1", "x"), h, strings.Repeat("a", 63), 1)}, {"SOBRE", sobre("A00", "0", "")},
		{"CONTROL", "V2|CONTROL|INICIAR|" + h + "\n"}, {"CONTROL", "V1|TERMINAL|INICIAR|" + h + "\n"},
		{"CONTROL", "V1|CONTROL|ARMAR|" + h + "\n"}, {"CONTROL", "V1|CONTROL|INICIAR|" + h + "|extra\n"},
		{"CONTROL", "V1|CONTROL|ARMAR|" + h + "|01\n"}, {"CONTROL", "V1|CONTROL|ARMAR|" + h + "|2147483648\n"},
		{"CONTROL", "V1|CONTROL|CANCELAR|" + h + "|SALIDA|0\n"}, {"CONTROL", "V1|CONTROL|CANCELAR|" + h + "|PLAZO|65\n"},
		{"TERMINAL", terminal("S4", "65", "CANCELADO", sinBash)}, {"TERMINAL", terminal("S3", "0", "SALIDA", sinBash)},
		{"TERMINAL", terminal("S1", "65", "INCIDENTE", conBash)}, {"TERMINAL", terminal("S2", "65", "PROTOCOLO", conBash)},
		{"TERMINAL", terminal("S1", "130", "SENAL_INT", sinBash)}, {"TERMINAL", terminal("S1", "143", "SENAL_TERM", sinBash)},
		{"TERMINAL", terminal("S3", "65", "SALIDA", "13|-|-|-|-|1|1|0|1")}, {"TERMINAL", terminal("S3", "65", "INCIDENTE", "13|12|12|12|7|1|0|0|1")},
		{"TERMINAL", terminal("S3", "0", "SALIDA", "13|12|12|12|18446744073709551616|1|1|0|1")}, {"TERMINAL", terminal("S3", "0", "SALIDA", "13|12|12|12|7|1|1|2147483648|1")},
		{"TERMINAL", terminal("S3", "130", "SENAL_TERM", conBash)}, {"TERMINAL", terminal("S5", "65", "INCIDENTE", conBash)},
		{"TICKET", "2147483647|" + strings.Repeat("x", 2049) + "\n"}, {"TICKET", "0|x\n"}, {"TICKET", "12|\n"},
		{"TICKET", "12|x\t\n"}, {"TICKET", "12|x\r\n"}, {"TICKET", "12|x\x00\n"}, {"TICKET", "12|x\xc3\n"}, {"TICKET", "12|x\nresto"}, {"TICKET", "12|x"},
	}
	for _, prueba := range invalidas {
		if _, err := decodificarTramaM38(prueba.clase, []byte(prueba.texto)); err == nil {
			return errors.New("mutante de trama aceptado")
		}
	}
	t, err := decodificarTramaM38("SOBRE", []byte(sobre("A00", "1", "x")))
	if err != nil {
		return err
	}
	t.ticket = strings.Repeat("x", limiteTramaM38("SOBRE"))
	if _, err = codificarTramaM38(t); err == nil {
		return errors.New("encoder reservó una trama sobredimensionada")
	}
	if _, err = codificarTramaM38(tramaM38{clase: "CONTROL"}); err == nil {
		return errors.New("encoder aceptó un control vacío")
	}
	for _, prueba := range []struct {
		clase   string
		entrada []byte
	}{{"SOBRE", make([]byte, 4097)}, {"CONTROL", make([]byte, 1025)}, {"TERMINAL", make([]byte, 1025)}, {"TICKET", make([]byte, 2061)}} {
		prueba.entrada[len(prueba.entrada)-1] = '\n'
		if _, err := decodificarTramaM38(prueba.clase, prueba.entrada); err == nil {
			return errors.New("límite físico excedido")
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
