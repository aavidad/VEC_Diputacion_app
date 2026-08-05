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

var errPrevalidacionM38 = errors.New("prevalidación de salida M38")
var errLimiteFisicoM38 = errors.New("límite físico de entrada M38")

type resultadoLecturaM38 uint8

const (
	lecturaNecesitaDatosM38 resultadoLecturaM38 = iota
	lecturaTramaM38
	lecturaTramaFinalM38
	lecturaEOFLimpioM38
)

type estadoLectorM38 uint8

const (
	lectorAbiertoVacioM38 estadoLectorM38 = iota
	lectorAbiertoParcialM38
	lectorMonotramaEsperandoEOFM38
	lectorEOFLimpioM38
	lectorErrorTerminalM38
)

var (
	errClaseLectorM38      = errors.New("clase del lector M38 inválida")
	errByteFlujoM38        = errors.New("byte de flujo M38 inválido")
	errExcesoFlujoM38      = errors.New("exceso físico del flujo M38")
	errTramaFlujoM38       = errors.New("trama del flujo M38 inválida")
	errEOFParcialM38       = errors.New("EOF con trama M38 parcial")
	errEOFSinMonotramaM38  = errors.New("EOF sin monotrama M38")
	errDatosPosterioresM38 = errors.New("datos posteriores a monotrama M38")
	errUsoPosteriorEOFM38  = errors.New("uso posterior a EOF M38")
)

type lectorTramaM38 struct {
	clase    string
	limite   int
	estado   estadoLectorM38
	longitud int
	buffer   [4096]byte
	err      error
}

func nuevoLectorTramaM38(clase string) (*lectorTramaM38, error) {
	limite := limiteTramaM38(clase)
	if limite == 0 {
		return nil, errClaseLectorM38
	}
	return &lectorTramaM38{clase: clase, limite: limite}, nil
}

func (l *lectorTramaM38) limpiarBuffer() {
	clear(l.buffer[:])
	l.longitud = 0
}

func (l *lectorTramaM38) fallar(err error) (tramaM38, int, resultadoLecturaM38, error) {
	l.limpiarBuffer()
	l.estado, l.err = lectorErrorTerminalM38, err
	return tramaM38{}, 0, lecturaNecesitaDatosM38, err
}

func (l *lectorTramaM38) consumir(fragmento []byte, fin bool) (tramaM38, int, resultadoLecturaM38, error) {
	if l.estado == lectorErrorTerminalM38 {
		return tramaM38{}, 0, lecturaNecesitaDatosM38, l.err
	}
	if l.estado == lectorEOFLimpioM38 {
		if len(fragmento) == 0 && fin {
			return tramaM38{}, 0, lecturaEOFLimpioM38, nil
		}
		return l.fallar(errUsoPosteriorEOFM38)
	}
	if l.estado == lectorMonotramaEsperandoEOFM38 {
		if len(fragmento) != 0 {
			return l.fallar(errDatosPosterioresM38)
		}
		if !fin {
			return tramaM38{}, 0, lecturaNecesitaDatosM38, nil
		}
		return l.entregarMonotrama(0)
	}
	for i, b := range fragmento {
		if b != '\n' {
			if b < 0x20 || b > 0x7e {
				return l.fallar(errByteFlujoM38)
			}
			if l.longitud == l.limite-1 {
				return l.fallar(errExcesoFlujoM38)
			}
			l.buffer[l.longitud], l.longitud = b, l.longitud+1
			continue
		}
		l.buffer[l.longitud], l.longitud = b, l.longitud+1
		consumidos := i + 1
		if l.clase != "CONTROL" {
			if consumidos != len(fragmento) {
				return l.fallar(errDatosPosterioresM38)
			}
			if fin {
				return l.entregarMonotrama(consumidos)
			}
			l.estado = lectorMonotramaEsperandoEOFM38
			return tramaM38{}, consumidos, lecturaNecesitaDatosM38, nil
		}
		trama, err := decodificarTramaM38(l.clase, l.buffer[:l.longitud])
		if err != nil {
			return l.fallar(fmt.Errorf("%w: %w", errTramaFlujoM38, err))
		}
		l.limpiarBuffer()
		l.estado = lectorAbiertoVacioM38
		if fin && consumidos == len(fragmento) {
			l.estado = lectorEOFLimpioM38
		}
		return trama, consumidos, lecturaTramaM38, nil
	}
	if fin {
		if l.longitud != 0 {
			return l.fallar(errEOFParcialM38)
		}
		if l.clase != "CONTROL" {
			return l.fallar(errEOFSinMonotramaM38)
		}
		l.estado = lectorEOFLimpioM38
		return tramaM38{}, 0, lecturaEOFLimpioM38, nil
	}
	if l.longitud != 0 {
		l.estado = lectorAbiertoParcialM38
	}
	return tramaM38{}, len(fragmento), lecturaNecesitaDatosM38, nil
}

func (l *lectorTramaM38) entregarMonotrama(consumidos int) (tramaM38, int, resultadoLecturaM38, error) {
	trama, err := decodificarTramaM38(l.clase, l.buffer[:l.longitud])
	if err != nil {
		return l.fallar(fmt.Errorf("%w: %w", errTramaFlujoM38, err))
	}
	l.limpiarBuffer()
	l.estado = lectorEOFLimpioM38
	return trama, consumidos, lecturaTramaFinalM38, nil
}

func limiteTramaM38(clase string) int {
	switch clase {
	case "SOBRE":
		return 4096
	case "CONTROL", "TERMINAL":
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
		return errPrevalidacionM38
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
			cardinalidadValida = len(t.campos) == map[string]int{"ARMAR": 3, "INICIAR": 2, "CANCELAR": 4}[t.campos[0]]
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
			return errPrevalidacionM38
		}
		restante -= len(campo)
	}
	return nil
}

func autoprobarLectorTramaM38() error {
	h := strings.Repeat("a", 64)
	type muestra struct {
		clase, texto string
		maximo       int
	}
	muestras := []muestra{
		{"SOBRE", "V1|SOBRE|" + h + "|2147483647|NOMINAL|" + h + "|2048|" + strings.Repeat("x", 2048) + "\n", 2212},
		{"CONTROL", "V1|CONTROL|CANCELAR|" + h + "|SENAL_TERM|143\n", 100},
		{"TERMINAL", "V1|TERMINAL|" + h + "|2147483647|S3|143|SENAL_TERM|2147483647|2147483647|2147483647|2147483647|18446744073709551615|1|1|0|1\n", 179},
		{"TICKET", "2147483647|" + strings.Repeat("x", 2048) + "\n", 2060},
	}
	fresco := func(m muestra) *lectorTramaM38 {
		return &lectorTramaM38{clase: m.clase, limite: limiteTramaM38(m.clase)}
	}
	for _, m := range muestras {
		lector, err := nuevoLectorTramaM38(m.clase)
		if err != nil || lector == nil || lector.clase != m.clase || lector.limite != limiteTramaM38(m.clase) || lector.estado != lectorAbiertoVacioM38 || !lectorLimpioM38(lector) || lector.err != nil || len(m.texto) != m.maximo {
			return errors.New("construcción o máximo canónico discrepante")
		}
	}
	if lector, err := nuevoLectorTramaM38("DESCONOCIDA"); lector != nil || err != errClaseLectorM38 {
		return errors.New("constructor aceptó una clase inválida")
	}
	for _, m := range muestras {
		clase, texto := m.clase, m.texto
		for corte := 0; corte <= len(texto); corte++ {
			lector := fresco(m)
			prefijo := []byte(texto[:corte])
			trama, consumidos, resultado, err := lector.consumir(prefijo, false)
			esTrama := clase == "CONTROL" && corte == len(texto)
			esperado := lecturaNecesitaDatosM38
			if esTrama {
				esperado = lecturaTramaM38
			}
			if err != nil || consumidos != corte || resultado != esperado || esTrama != tramaExactaM38(trama, texto) || !esTrama && !tramaCeroM38(trama) || esTrama && lector.estado != lectorAbiertoVacioM38 {
				return errors.New("corte inicial de trama discrepante")
			}
			if corte > 0 && corte < len(texto) {
				prefijo[0] ^= 1
			}
			trama, consumidos, resultado, err = lector.consumir([]byte(texto[corte:]), true)
			esperado = lecturaTramaFinalM38
			if clase == "CONTROL" {
				esperado = lecturaTramaM38
				if esTrama {
					esperado = lecturaEOFLimpioM38
				}
			}
			if err != nil || consumidos != len(texto)-corte || resultado != esperado || esperado == lecturaEOFLimpioM38 && !tramaCeroM38(trama) || esperado != lecturaEOFLimpioM38 && !tramaExactaM38(trama, texto) || !lectorLimpioM38(lector) || lector.estado != lectorEOFLimpioM38 {
				return errors.New("reensamblado de trama discrepante")
			}
		}
		lector := fresco(m)
		for i := range len(texto) {
			trama, consumidos, resultado, err := lector.consumir([]byte{texto[i]}, i == len(texto)-1)
			final := clase == "CONTROL" && resultado == lecturaTramaM38 || clase != "CONTROL" && resultado == lecturaTramaFinalM38
			if err != nil || consumidos != 1 || i < len(texto)-1 && (resultado != lecturaNecesitaDatosM38 || !tramaCeroM38(trama)) || i == len(texto)-1 && (!final || !tramaExactaM38(trama, texto) || lector.estado != lectorEOFLimpioM38) {
				return errors.New("fragmentación byte a byte discrepante")
			}
		}
	}
	controlA := "V1|CONTROL|INICIAR|" + h + "\n"
	controlB := "V1|CONTROL|CANCELAR|" + h + "|PROTOCOLO|65\n"
	controlC := muestras[1].texto
	lector := fresco(muestras[1])
	sobrante := []byte(controlA + controlB + controlC)
	for _, esperado := range []string{controlA, controlB, controlC} {
		trama, consumidos, resultado, err := lector.consumir(sobrante, true)
		if err != nil || resultado != lecturaTramaM38 || consumidos != len(esperado) || string(sobrante[:consumidos]) != esperado || !tramaExactaM38(trama, esperado) {
			return errors.New("coalescencia de controles discrepante")
		}
		sobrante = sobrante[consumidos:]
	}
	if _, n, r, err := lector.consumir(nil, true); err != nil || n != 0 || r != lecturaEOFLimpioM38 {
		return errors.New("EOF coalescido no enclavado")
	}
	lector = fresco(muestras[1])
	if trama, n, r, err := lector.consumir([]byte(controlA+controlB[:8]), false); err != nil || n != len(controlA) || r != lecturaTramaM38 || !tramaExactaM38(trama, controlA) || lector.estado != lectorAbiertoVacioM38 {
		return errors.New("control completo más parcial discrepante")
	}
	if trama, n, r, err := lector.consumir([]byte(controlB[:8]), false); err != nil || n != 8 || r != lecturaNecesitaDatosM38 || !tramaCeroM38(trama) || lector.estado != lectorAbiertoParcialM38 {
		return errors.New("parcial de control no conservada")
	}
	if trama, n, r, err := lector.consumir([]byte(controlB[8:]+controlC), false); err != nil || n != len(controlB)-8 || r != lecturaTramaM38 || !tramaExactaM38(trama, controlB) || lector.estado != lectorAbiertoVacioM38 {
		return errors.New("parcial más controles coalescidos discrepante")
	}
	if trama, n, r, err := lector.consumir([]byte(controlC), false); err != nil || n != len(controlC) || r != lecturaTramaM38 || !tramaExactaM38(trama, controlC) || lector.estado != lectorAbiertoVacioM38 {
		return errors.New("segundo control coalescido no consumido")
	}
	for _, m := range []muestra{muestras[0], muestras[2], muestras[3]} {
		for _, cola := range [][]byte{[]byte(m.texto), {'x'}, {0}, {'\n'}} {
			for _, fin := range []bool{false, true} {
				lector = fresco(m)
				if err := exigirErrorLectorM38(lector, append([]byte(m.texto), cola...), fin, errDatosPosterioresM38); err != nil {
					return err
				}
			}
		}
		lector = fresco(m)
		if err := exigirErrorLectorM38(lector, nil, true, errEOFSinMonotramaM38); err != nil {
			return err
		}
		lector = fresco(m)
		original := []byte(m.texto)
		if trama, n, r, err := lector.consumir(original, false); err != nil || n != len(original) || r != lecturaNecesitaDatosM38 || !tramaCeroM38(trama) || lector.estado != lectorMonotramaEsperandoEOFM38 {
			return errors.New("monotrama expuesta antes de EOF")
		}
		original[0] ^= 1
		if trama, n, r, err := lector.consumir(nil, false); err != nil || n != 0 || r != lecturaNecesitaDatosM38 || !tramaCeroM38(trama) || lector.estado != lectorMonotramaEsperandoEOFM38 {
			return errors.New("L2 vacío sin EOF discrepante")
		}
		lectorConCola := *lector
		if err := exigirErrorLectorM38(&lectorConCola, []byte{'x'}, false, errDatosPosterioresM38); err != nil {
			return err
		}
		if trama, n, r, err := lector.consumir(nil, true); err != nil || n != 0 || r != lecturaTramaFinalM38 || !tramaExactaM38(trama, m.texto) || !lectorLimpioM38(lector) || lector.estado != lectorEOFLimpioM38 {
			return errors.New("copia defensiva L2 discrepante")
		}
	}
	lector = fresco(muestras[0])
	if trama, n, r, err := lector.consumir([]byte{'\n'}, false); err != nil || n != 1 || r != lecturaNecesitaDatosM38 || !tramaCeroM38(trama) || lector.estado != lectorMonotramaEsperandoEOFM38 || lector.longitud != 1 || lector.buffer[0] != '\n' {
		return errors.New("monotrama inválida no retenida en L2")
	}
	lectorInvalidoConDato := *lector
	if err := exigirErrorLectorM38(&lectorInvalidoConDato, []byte{'x'}, true, errDatosPosterioresM38); err != nil {
		return err
	}
	if err := exigirErrorLectorM38(lector, nil, true, errTramaFlujoM38); err != nil {
		return err
	}
	for _, fin := range []bool{false, true} {
		lector = fresco(muestras[0])
		if err := exigirErrorLectorM38(lector, []byte("\nx"), fin, errDatosPosterioresM38); err != nil {
			return err
		}
	}
	lector = fresco(muestras[1])
	if trama, n, r, err := lector.consumir(nil, false); err != nil || n != 0 || r != lecturaNecesitaDatosM38 || !tramaCeroM38(trama) || lector.estado != lectorAbiertoVacioM38 {
		return errors.New("L0 vacío sin EOF discrepante")
	}
	if _, _, _, err := lector.consumir([]byte("V"), false); err != nil {
		return err
	}
	if trama, n, r, err := lector.consumir(nil, false); err != nil || n != 0 || r != lecturaNecesitaDatosM38 || !tramaCeroM38(trama) || lector.estado != lectorAbiertoParcialM38 || lector.longitud != 1 || lector.buffer[0] != 'V' {
		return errors.New("L1 vacío alteró la parcial")
	}
	if _, _, _, err := lector.consumir([]byte("1"), false); err != nil {
		return err
	}
	if err := exigirErrorLectorM38(lector, nil, true, errEOFParcialM38); err != nil {
		return err
	}
	lector = fresco(muestras[1])
	if err := exigirErrorLectorM38(lector, []byte("V1"), true, errEOFParcialM38); err != nil {
		return err
	}
	for _, m := range muestras {
		limite := limiteTramaM38(m.clase)
		lector = fresco(m)
		if err := exigirErrorLectorM38(lector, []byte(strings.Repeat("x", limite-1)+"\n"), true, errTramaFlujoM38); err != nil || errors.Is(lector.err, errExcesoFlujoM38) {
			return errors.New("frontera física no alcanzó O1a")
		}
		lector = fresco(m)
		if err := exigirErrorLectorM38(lector, []byte(strings.Repeat("x", limite)), false, errExcesoFlujoM38); err != nil {
			return err
		}
		lector = fresco(m)
		if _, _, _, err := lector.consumir([]byte(strings.Repeat("x", limite-1)), false); err != nil {
			return err
		}
		frontera := *lector
		if err := exigirErrorLectorM38(lector, []byte{'x'}, false, errExcesoFlujoM38); err != nil {
			return err
		}
		if err := exigirErrorLectorM38(&frontera, []byte{0}, false, errByteFlujoM38); err != nil {
			return errors.New("byte inválido no prevaleció sobre exceso")
		}
	}
	for _, hostil := range []byte{0, '\r', '\t', 0x1f, 0x7f, 0x80} {
		for _, prefijo := range [][]byte{nil, []byte("V1")} {
			lector = fresco(muestras[1])
			if _, _, _, err := lector.consumir(prefijo, false); err != nil {
				return err
			}
			if err := exigirErrorLectorM38(lector, []byte{hostil}, false, errByteFlujoM38); err != nil {
				return err
			}
		}
	}
	for _, adversa := range []string{"\n", "V2|CONTROL|INICIAR|" + h + "\n", "V1|TERMINAL|INICIAR|" + h + "\n", "V1|CONTROL|ARMAR|" + h + "\n", "V1|CONTROL|CANCELAR|" + h + "|SALIDA|0\n"} {
		lector = fresco(muestras[1])
		if err := exigirErrorLectorM38(lector, []byte(adversa), true, errTramaFlujoM38); err != nil {
			return err
		}
	}
	grande := []byte(controlA + strings.Repeat("x", 1<<20))
	lector = fresco(muestras[1])
	trama, n, r, err := lector.consumir(grande, false)
	if err != nil || n != len(controlA) || r != lecturaTramaM38 || !tramaExactaM38(trama, controlA) || len(grande[n:]) != 1<<20 || !lectorLimpioM38(lector) || lector.estado != lectorAbiertoVacioM38 {
		return errors.New("fragmento enorme con LF temprano discrepante")
	}
	grande = []byte(strings.Repeat("x", 1<<20))
	lector = fresco(muestras[1])
	if err := exigirErrorLectorM38(lector, grande, false, errExcesoFlujoM38); err != nil {
		return err
	}
	entrada := []byte(controlA)
	lector = fresco(muestras[1])
	trama, n, r, err = lector.consumir(entrada, false)
	entrada[0] ^= 1
	if err != nil || n != len(entrada) || r != lecturaTramaM38 || !tramaExactaM38(trama, controlA) || !lectorLimpioM38(lector) || lector.estado != lectorAbiertoVacioM38 {
		return errors.New("trama entregada dependía del fragmento")
	}
	lector = &lectorTramaM38{clase: "CONTROL", limite: limiteTramaM38("CONTROL")}
	for repeticion := range 2 {
		if _, n, r, err := lector.consumir(nil, true); err != nil || n != 0 || r != lecturaEOFLimpioM38 || lector.estado != lectorEOFLimpioM38 {
			return fmt.Errorf("EOF limpio número %d rechazado", repeticion+1)
		}
	}
	if !lectorLimpioM38(lector) {
		return errors.New("EOF limpio no borró el buffer")
	}
	if err := exigirErrorLectorM38(lector, []byte(controlA), true, errUsoPosteriorEOFM38); err != nil {
		return err
	}
	lector = &lectorTramaM38{clase: "CONTROL", limite: limiteTramaM38("CONTROL")}
	if trama, n, r, err := lector.consumir([]byte(controlA), true); err != nil || n != len(controlA) || r != lecturaTramaM38 || !tramaExactaM38(trama, controlA) || lector.estado != lectorEOFLimpioM38 {
		return errors.New("CONTROL con EOF no enclavó L3")
	}
	lectorL3 := *lector
	if err := exigirErrorLectorM38(&lectorL3, []byte{'x'}, false, errUsoPosteriorEOFM38); err != nil {
		return err
	}
	return exigirErrorLectorM38(lector, nil, false, errUsoPosteriorEOFM38)
}

func exigirErrorLectorM38(lector *lectorTramaM38, fragmento []byte, fin bool, objetivo error) error {
	trama, consumidos, resultado, err := lector.consumir(fragmento, fin)
	if !errors.Is(err, objetivo) || consumidos != 0 || resultado != lecturaNecesitaDatosM38 || !tramaCeroM38(trama) || !lectorLimpioM38(lector) || lector.estado != lectorErrorTerminalM38 {
		return fmt.Errorf("tupla de error del lector discrepante: %w", objetivo)
	}
	trama, consumidos, resultado, repetido := lector.consumir([]byte("V1|CONTROL|INICIAR|aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n"), true)
	if repetido != err || consumidos != 0 || resultado != lecturaNecesitaDatosM38 || !tramaCeroM38(trama) || lector.estado != lectorErrorTerminalM38 {
		return fmt.Errorf("error del lector no pegajoso: %w", objetivo)
	}
	return nil
}

func tramaCeroM38(trama tramaM38) bool {
	return trama.clase == "" && trama.campos == nil && trama.ticket == ""
}

func tramaExactaM38(trama tramaM38, texto string) bool {
	salida, err := codificarTramaM38(trama)
	return err == nil && string(salida) == texto
}
func lectorLimpioM38(lector *lectorTramaM38) bool {
	return lector.longitud == 0 && lector.buffer == [4096]byte{}
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
	limite := limiteTramaM38(clase)
	if limite > 0 && len(entrada) > limite {
		return tramaM38{}, errLimiteFisicoM38
	}
	if limite == 0 || len(entrada) == 0 || entrada[len(entrada)-1] != '\n' {
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
	if len(p) != 16 || !hexM38(p[2]) || !decimalM38(p[3], 1, 2147483647) || (p[4] != "S1" && p[4] != "S2" && p[4] != "S3" && p[4] != "S4") || !causaEstadoM38(p[6], p[5], true) || !decimalM38(p[12], 0, 1) || !decimalM38(p[13], 0, 1) || !decimalM38(p[14], 0, 2147483647) || !decimalM38(p[15], 0, 1) {
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
		{"SOBRE", strings.Replace(sobre("A00", "1", "x"), h, "g"+h[1:], 1)}, {"SOBRE", strings.Replace(sobre("A00", "1", "x"), h, strings.ToUpper(h), 1)},
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
	if _, err = codificarTramaM38(t); !errors.Is(err, errPrevalidacionM38) {
		return errors.New("encoder reservó una trama sobredimensionada")
	}
	if _, err = codificarTramaM38(tramaM38{clase: "CONTROL"}); err == nil {
		return errors.New("encoder aceptó un control vacío")
	}
	for clase, limite := range map[string]int{"SOBRE": 4096, "CONTROL": 1024, "TERMINAL": 1024, "TICKET": 2060} {
		for _, delta := range []int{-1, 0, 1} {
			entrada := []byte(strings.Repeat("x", limite+delta-1) + "\n")
			_, err := decodificarTramaM38(clase, entrada)
			if delta == 1 && !errors.Is(err, errLimiteFisicoM38) {
				return errors.New("exceso sin centinela de límite")
			}
			if delta < 1 && (err == nil || errors.Is(err, errLimiteFisicoM38)) {
				return errors.New("frontera gramatical confundida con límite")
			}
		}
	}
	return autoprobarLectorTramaM38()
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
