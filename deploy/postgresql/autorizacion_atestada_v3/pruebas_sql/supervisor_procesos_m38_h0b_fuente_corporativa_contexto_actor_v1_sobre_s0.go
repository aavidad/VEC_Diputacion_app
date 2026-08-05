//go:build ignore && linux && amd64

package main

import (
	"errors"
	"fmt"
	"strings"
)

type resultadoRecepcionSobreM38 uint8

const (
	recepcionSobreNecesitaDatosM38 resultadoRecepcionSobreM38 = iota
	recepcionSobreConfirmadaM38
)

type faseRecepcionSobreM38 uint8

const (
	recepcionSobreS0M38 faseRecepcionSobreM38 = iota
	recepcionSobreS1M38
)

type sobreRetenidoM38 struct {
	nonce          string
	pidRunner      string
	selector       string
	identidad      string
	longitudTicket string
	ticket         string
}

type receptorSobreS0M38 struct {
	lector *lectorTramaM38
	fase   faseRecepcionSobreM38
	sobre  sobreRetenidoM38
	fallo  error
}

var (
	errReceptorSobreNuloM38        = errors.New("receptor de sobre M38 nulo")
	errInvarianteRecepcionSobreM38 = errors.New("invariante de recepción de sobre M38 incumplida")
	errUsoPosteriorSobreM38        = errors.New("uso posterior a recepción de sobre M38")
)

func nuevoReceptorSobreS0M38() (*receptorSobreS0M38, error) {
	lector, err := nuevoLectorTramaM38("SOBRE")
	if err != nil {
		return nil, err
	}
	return &receptorSobreS0M38{lector: lector}, nil
}

func sobreVacioS0M38(s sobreRetenidoM38) bool {
	return s.nonce == "" && s.pidRunner == "" && s.selector == "" &&
		s.identidad == "" && s.longitudTicket == "" && s.ticket == ""
}

func limpiarSobreS0M38(s *sobreRetenidoM38) {
	s.nonce = ""
	s.pidRunner = ""
	s.selector = ""
	s.identidad = ""
	s.longitudTicket = ""
	s.ticket = ""
}

func fallarRecepcionSobreS0M38(
	r *receptorSobreS0M38,
	causa error,
) (int, resultadoRecepcionSobreM38, error) {
	if r.lector != nil {
		r.lector.limpiarBuffer()
	}
	limpiarSobreS0M38(&r.sobre)
	r.fase, r.fallo = recepcionSobreS0M38, causa
	return 0, 0, r.fallo
}

func (r *receptorSobreS0M38) consumir(
	fragmento []byte,
	fin bool,
) (consumidos int, resultado resultadoRecepcionSobreM38, err error) {
	if r == nil {
		return 0, 0, errReceptorSobreNuloM38
	}
	if r.fallo != nil {
		return 0, 0, r.fallo
	}
	if r.fase == recepcionSobreS1M38 {
		return 0, 0, errUsoPosteriorSobreM38
	}
	if r.fase != recepcionSobreS0M38 || r.lector == nil || !sobreVacioS0M38(r.sobre) {
		return fallarRecepcionSobreS0M38(r, errInvarianteRecepcionSobreM38)
	}

	trama, consumidos, lectura, err := r.lector.consumir(fragmento, fin)
	if err != nil {
		return fallarRecepcionSobreS0M38(r, err)
	}

	switch lectura {
	case lecturaNecesitaDatosM38:
		compatible := r.lector.estado == lectorAbiertoVacioM38 ||
			r.lector.estado == lectorAbiertoParcialM38 ||
			r.lector.estado == lectorMonotramaEsperandoEOFM38
		if !compatible || trama.clase != "" || trama.campos != nil || trama.ticket != "" {
			return fallarRecepcionSobreS0M38(r, errInvarianteRecepcionSobreM38)
		}
		return consumidos, recepcionSobreNecesitaDatosM38, nil
	case lecturaTramaFinalM38:
		valida := r.lector.estado == lectorEOFLimpioM38 &&
			r.lector.longitud == 0 && r.lector.buffer == [4096]byte{} &&
			trama.clase == "SOBRE" && len(trama.campos) == 5 && trama.ticket != ""
		if !valida {
			return fallarRecepcionSobreS0M38(r, errInvarianteRecepcionSobreM38)
		}
		retenido := sobreRetenidoM38{
			nonce:          trama.campos[0],
			pidRunner:      trama.campos[1],
			selector:       trama.campos[2],
			identidad:      trama.campos[3],
			longitudTicket: trama.campos[4],
			ticket:         trama.ticket,
		}
		r.sobre, r.fase = retenido, recepcionSobreS1M38
		return consumidos, recepcionSobreConfirmadaM38, nil
	default:
		return fallarRecepcionSobreS0M38(r, errInvarianteRecepcionSobreM38)
	}
}

func textoSobreS0M38(pid, selector, ticket string) string {
	h := strings.Repeat("a", 64)
	return tramaCrudaSobreS0M38(h, pid, selector, h, fmt.Sprint(len(ticket)), ticket)
}

func tramaCrudaSobreS0M38(nonce, pid, selector, identidad, longitud, ticket string) string {
	return "V1|SOBRE|" + nonce + "|" + pid + "|" + selector + "|" + identidad + "|" + longitud + "|" + ticket + "\n"
}

func receptorNuevoExactoS0M38() (*receptorSobreS0M38, error) {
	r, err := nuevoReceptorSobreS0M38()
	if err != nil || r == nil || r.lector == nil || r.lector.clase != "SOBRE" ||
		r.lector.limite != 4096 || r.lector.estado != lectorAbiertoVacioM38 ||
		r.fase != recepcionSobreS0M38 || r.sobre != (sobreRetenidoM38{}) || r.fallo != nil {
		return nil, errors.New("receptor inicial discrepante")
	}
	return r, nil
}

func sobreRetenidoExactoS0M38(r *receptorSobreS0M38, pid, selector, ticket string) bool {
	h := strings.Repeat("a", 64)
	esperado := sobreRetenidoM38{
		nonce:          h,
		pidRunner:      pid,
		selector:       selector,
		identidad:      h,
		longitudTicket: fmt.Sprint(len(ticket)),
		ticket:         ticket,
	}
	return r != nil && r.fase == recepcionSobreS1M38 && r.fallo == nil &&
		r.sobre == esperado && r.lector.estado == lectorEOFLimpioM38 && lectorLimpioM38(r.lector)
}

func exigirFalloDirectoSobreS0M38(entrada []byte, fin bool, objetivo error) error {
	r, err := receptorNuevoExactoS0M38()
	if err != nil {
		return err
	}
	n, resultado, primerError := r.consumir(entrada, fin)
	if primerError == nil || objetivo != nil && !errors.Is(primerError, objetivo) ||
		n != 0 || resultado != recepcionSobreNecesitaDatosM38 ||
		r.fase != recepcionSobreS0M38 || r.sobre != (sobreRetenidoM38{}) || r.fallo != primerError {
		return errors.New("fallo directo de sobre discrepante")
	}
	n, resultado, repetido := r.consumir([]byte(textoSobreS0M38("1", "A00", "x")), true)
	if repetido != primerError || n != 0 || resultado != recepcionSobreNecesitaDatosM38 ||
		r.fase != recepcionSobreS0M38 || r.sobre != (sobreRetenidoM38{}) {
		return errors.New("fallo de sobre no quedó enclavado")
	}
	return nil
}

func probarRecepcionesValidasSobreS0M38() error {
	r, err := receptorNuevoExactoS0M38()
	if err != nil {
		return err
	}
	if n, resultado, err := r.consumir(nil, false); err != nil || n != 0 ||
		resultado != recepcionSobreNecesitaDatosM38 || r.fase != recepcionSobreS0M38 {
		return errors.New("entrada vacía sin EOF discrepante")
	}

	ticket := "x|y|z"
	texto := textoSobreS0M38("2147483647", "NOMINAL", ticket)
	entrada := []byte(texto)
	r, err = receptorNuevoExactoS0M38()
	if err != nil {
		return err
	}
	n, resultado, err := r.consumir(entrada, true)
	entrada[0] ^= 1
	if err != nil || n != len(texto) || resultado != recepcionSobreConfirmadaM38 ||
		!sobreRetenidoExactoS0M38(r, "2147483647", "NOMINAL", ticket) {
		return errors.New("recepción directa de sobre discrepante")
	}
	primero := r.sobre
	estado := r.lector.estado
	if n, resultado, err = r.consumir([]byte(strings.Repeat("z", 8192)), false); err != errUsoPosteriorSobreM38 ||
		n != 0 || resultado != recepcionSobreNecesitaDatosM38 || r.sobre != primero ||
		r.fase != recepcionSobreS1M38 || r.lector.estado != estado {
		return errors.New("uso posterior alteró el sobre")
	}

	for corte := 0; corte <= len(texto); corte++ {
		r, err = receptorNuevoExactoS0M38()
		if err != nil {
			return err
		}
		prefijo := []byte(texto[:corte])
		n, resultado, err = r.consumir(prefijo, false)
		if err != nil || n != corte || resultado != recepcionSobreNecesitaDatosM38 ||
			r.fase != recepcionSobreS0M38 || r.sobre != (sobreRetenidoM38{}) {
			return fmt.Errorf("corte inicial de sobre %d discrepante", corte)
		}
		if corte > 0 {
			prefijo[0] ^= 1
		}
		n, resultado, err = r.consumir([]byte(texto[corte:]), true)
		if err != nil || n != len(texto)-corte || resultado != recepcionSobreConfirmadaM38 ||
			!sobreRetenidoExactoS0M38(r, "2147483647", "NOMINAL", ticket) {
			return fmt.Errorf("reensamblado de sobre %d discrepante", corte)
		}
	}

	r, err = receptorNuevoExactoS0M38()
	if err != nil {
		return err
	}
	for i := range len(texto) {
		n, resultado, err = r.consumir([]byte{texto[i]}, i == len(texto)-1)
		if err != nil || n != 1 || i < len(texto)-1 &&
			(resultado != recepcionSobreNecesitaDatosM38 || r.fase != recepcionSobreS0M38) ||
			i == len(texto)-1 && (resultado != recepcionSobreConfirmadaM38 ||
				!sobreRetenidoExactoS0M38(r, "2147483647", "NOMINAL", ticket)) {
			return fmt.Errorf("byte de sobre %d discrepante", i)
		}
	}

	casos := []struct {
		pid, selector, ticket string
		longitud              int
	}{
		{"1", "A00", "x", 149},
		{"2147483647", "NOMINAL", strings.Repeat("x", 2048), 2212},
	}
	for _, caso := range casos {
		texto = textoSobreS0M38(caso.pid, caso.selector, caso.ticket)
		if len(texto) != caso.longitud {
			return errors.New("frontera canónica de sobre discrepante")
		}
		r, err = receptorNuevoExactoS0M38()
		if err != nil {
			return err
		}
		n, resultado, err = r.consumir([]byte(texto), true)
		if err != nil || n != len(texto) || resultado != recepcionSobreConfirmadaM38 ||
			!sobreRetenidoExactoS0M38(r, caso.pid, caso.selector, caso.ticket) {
			return errors.New("frontera canónica de sobre rechazada")
		}
	}
	return nil
}

func probarErroresFlujoSobreS0M38() error {
	h := strings.Repeat("a", 64)
	valido := textoSobreS0M38("1", "A00", "x")
	for _, caso := range []struct {
		entrada  []byte
		fin      bool
		objetivo error
	}{
		{nil, true, errEOFSinMonotramaM38},
		{[]byte("V1"), true, errEOFParcialM38},
		{[]byte("\n"), true, errTramaFlujoM38},
		{[]byte(valido + "x"), false, errDatosPosterioresM38},
		{[]byte(valido + "x"), true, errDatosPosterioresM38},
		{[]byte(valido + "\x00"), false, errDatosPosterioresM38},
		{[]byte(valido + "\x00"), true, errDatosPosterioresM38},
		{[]byte(valido + "\n"), false, errDatosPosterioresM38},
		{[]byte(valido + "\n"), true, errDatosPosterioresM38},
		{[]byte(valido + valido), false, errDatosPosterioresM38},
		{[]byte(valido + valido), true, errDatosPosterioresM38},
		{[]byte(strings.Repeat("x", 4095) + "\n"), true, errTramaFlujoM38},
		{[]byte(strings.Repeat("x", 4096)), false, errExcesoFlujoM38},
		{[]byte(strings.Repeat("x", 4097)), false, errExcesoFlujoM38},
		{[]byte(valido + strings.Repeat("x", 1<<16)), false, errDatosPosterioresM38},
		{[]byte(strings.Repeat("x", 1<<16)), false, errExcesoFlujoM38},
	} {
		if err := exigirFalloDirectoSobreS0M38(caso.entrada, caso.fin, caso.objetivo); err != nil {
			return err
		}
	}
	for _, hostil := range []byte{0, '\r', '\t', 0x1f, 0x7f, 0x80} {
		if err := exigirFalloDirectoSobreS0M38([]byte{'V', hostil}, false, errByteFlujoM38); err != nil {
			return err
		}
	}

	r, err := receptorNuevoExactoS0M38()
	if err != nil {
		return err
	}
	if n, resultado, err := r.consumir([]byte("V"), false); err != nil || n != 1 ||
		resultado != recepcionSobreNecesitaDatosM38 || r.fase != recepcionSobreS0M38 {
		return errors.New("parcial inicial de sobre discrepante")
	}
	if n, resultado, primerError := r.consumir([]byte("1"), true); !errors.Is(primerError, errEOFParcialM38) ||
		n != 0 || resultado != recepcionSobreNecesitaDatosM38 || r.fallo != primerError {
		return errors.New("EOF parcial fragmentado discrepante")
	}

	r, err = receptorNuevoExactoS0M38()
	if err != nil {
		return err
	}
	if n, resultado, err := r.consumir([]byte("\n"), false); err != nil || n != 1 ||
		resultado != recepcionSobreNecesitaDatosM38 || r.fase != recepcionSobreS0M38 ||
		r.lector.estado != lectorMonotramaEsperandoEOFM38 {
		return errors.New("LF aislado no permaneció en S0/L2")
	}
	if n, resultado, primerError := r.consumir(nil, true); !errors.Is(primerError, errTramaFlujoM38) ||
		n != 0 || resultado != recepcionSobreNecesitaDatosM38 || r.fallo != primerError {
		return errors.New("EOF posterior a LF aislado discrepante")
	}

	r, err = receptorNuevoExactoS0M38()
	if err != nil {
		return err
	}
	if n, resultado, err := r.consumir([]byte(valido), false); err != nil || n != len(valido) ||
		resultado != recepcionSobreNecesitaDatosM38 || r.lector.estado != lectorMonotramaEsperandoEOFM38 {
		return errors.New("sobre válido no alcanzó L2")
	}
	if n, resultado, primerError := r.consumir([]byte{'x'}, false); !errors.Is(primerError, errDatosPosterioresM38) ||
		n != 0 || resultado != recepcionSobreNecesitaDatosM38 || r.fallo != primerError {
		return errors.New("dato posterior desde L2 discrepante")
	}

	invalidas := []string{
		tramaCrudaSobreS0M38(h, "1", "A00", h, "1", ""),
		tramaCrudaSobreS0M38("g"+h[1:], "1", "A00", h, "1", "x"),
		tramaCrudaSobreS0M38(strings.ToUpper(h), "1", "A00", h, "1", "x"),
		tramaCrudaSobreS0M38(h[:63], "1", "A00", h, "1", "x"),
		tramaCrudaSobreS0M38(h, "0", "A00", h, "1", "x"),
		tramaCrudaSobreS0M38(h, "01", "A00", h, "1", "x"),
		tramaCrudaSobreS0M38(h, "2147483648", "A00", h, "1", "x"),
		tramaCrudaSobreS0M38(h, "1", "A0", h, "1", "x"),
		tramaCrudaSobreS0M38(h, "1", "a00", h, "1", "x"),
		tramaCrudaSobreS0M38(h, "1", "A00", strings.ToUpper(h), "1", "x"),
		tramaCrudaSobreS0M38(h, "1", "A00", "g"+h[1:], "1", "x"),
		tramaCrudaSobreS0M38(h, "1", "A00", h[:63], "1", "x"),
		tramaCrudaSobreS0M38(h, "1", "A00", h, "0", "x"),
		tramaCrudaSobreS0M38(h, "1", "A00", h, "01", "x"),
		tramaCrudaSobreS0M38(h, "1", "A00", h, "2", "x"),
		tramaCrudaSobreS0M38(h, "1", "A00", h, "2049", "x"),
		strings.Replace(valido, "V1|", "V2|", 1),
		strings.Replace(valido, "|SOBRE|", "|OTRA|", 1),
		strings.Replace(valido, "|1|x\n", "|x\n", 1),
	}
	for _, invalida := range invalidas {
		if err := exigirFalloDirectoSobreS0M38([]byte(invalida), true, errTramaFlujoM38); err != nil {
			return err
		}
	}
	for _, invalida := range []string{
		tramaCrudaSobreS0M38(h, "1", "A00", h, "3", "x\ty"),
		tramaCrudaSobreS0M38(h, "1", "A00", h, "3", "x\ry"),
		tramaCrudaSobreS0M38(h, "1", "A00", h, "3", "x\x00y"),
	} {
		if err := exigirFalloDirectoSobreS0M38([]byte(invalida), true, errByteFlujoM38); err != nil {
			return err
		}
	}
	return nil
}

func probarInvariantesSobreS0M38() error {
	var nulo *receptorSobreS0M38
	if n, resultado, err := nulo.consumir(nil, false); err != errReceptorSobreNuloM38 ||
		n != 0 || resultado != recepcionSobreNecesitaDatosM38 {
		return errors.New("receptor nulo no falló cerrado")
	}
	for _, r := range []*receptorSobreS0M38{
		{},
		{lector: &lectorTramaM38{clase: "SOBRE", limite: 4096}, fase: faseRecepcionSobreM38(2)},
		{lector: &lectorTramaM38{clase: "SOBRE", limite: 4096}, sobre: sobreRetenidoM38{ticket: "x"}},
		{lector: &lectorTramaM38{clase: "SOBRE", limite: 4096, estado: lectorAbiertoParcialM38,
			longitud: 1, buffer: [4096]byte{'V'}}, sobre: sobreRetenidoM38{ticket: "sensible"}},
		{lector: &lectorTramaM38{clase: "SOBRE", limite: 4096, estado: lectorEOFLimpioM38}},
	} {
		n, resultado, primerError := r.consumir(nil, true)
		if primerError != errInvarianteRecepcionSobreM38 || n != 0 ||
			resultado != recepcionSobreNecesitaDatosM38 || r.sobre != (sobreRetenidoM38{}) ||
			r.fallo != primerError || r.fase != recepcionSobreS0M38 ||
			r.lector != nil && !lectorLimpioM38(r.lector) {
			return errors.New("invariante de receptor no falló cerrado")
		}
		if n, resultado, err := r.consumir([]byte(textoSobreS0M38("1", "A00", "x")), true); err != primerError ||
			n != 0 || resultado != recepcionSobreNecesitaDatosM38 || r.sobre != (sobreRetenidoM38{}) ||
			r.lector != nil && !lectorLimpioM38(r.lector) {
			return errors.New("invariante de receptor no quedó enclavada")
		}
	}
	return nil
}

func autoprobarSobreS0M38() error {
	if err := probarRecepcionesValidasSobreS0M38(); err != nil {
		return fmt.Errorf("recepciones válidas S0: %w", err)
	}
	if err := probarErroresFlujoSobreS0M38(); err != nil {
		return fmt.Errorf("errores de flujo S0: %w", err)
	}
	if err := probarInvariantesSobreS0M38(); err != nil {
		return fmt.Errorf("invariantes S0: %w", err)
	}
	return nil
}
