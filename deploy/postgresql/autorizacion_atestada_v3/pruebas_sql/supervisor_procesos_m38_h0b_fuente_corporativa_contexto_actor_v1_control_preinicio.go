//go:build ignore && linux && amd64

package main

import "errors"

type resultadoControlPreinicioM38 uint8

const (
	controlPreinicioNecesitaDatosM38 resultadoControlPreinicioM38 = iota
	controlPreinicioArmadoM38
	controlPreinicioInicioPendienteM38
	controlPreinicioCausaEnclavadaM38
)

type faseControlPreinicioM38 uint8

const (
	controlPreinicioS1M38 faseControlPreinicioM38 = iota
	controlPreinicioS2M38
	controlPreinicioS3M38
	controlPreinicioS5M38
)

type causaPreinicioM38 struct {
	faseOrigen faseControlPreinicioM38
	causa      string
	estado     string
}

type controladorPreinicioM38 struct {
	recepcion *receptorSobreS0M38
	lector    *lectorTramaM38
	nonce     [64]byte
	fase      faseControlPreinicioM38
	causa     causaPreinicioM38
	fallo     error
}

var (
	errControlPreinicioNuloM38         = errors.New("controlador previo M38 nulo")
	errSobreNoConfirmadoPreinicioM38   = errors.New("sobre M38 no confirmado para control previo")
	errInvarianteControlPreinicioM38   = errors.New("invariante del control previo M38 incumplida")
	errUsoPosteriorControlPreinicioM38 = errors.New("uso posterior al control previo M38")
)

func receptorPristinoPreinicioM38(r *receptorSobreS0M38) bool {
	return r.lector != nil && r.lector.clase == "SOBRE" && r.lector.limite == 4096 &&
		r.lector.estado == lectorAbiertoVacioM38 && r.lector.err == nil &&
		lectorLimpioM38(r.lector) && r.sobre.nonce == "" && r.sobre.pidRunner == "" &&
		r.sobre.selector == "" && r.sobre.identidad == "" &&
		r.sobre.longitudTicket == "" && r.sobre.ticket == ""
}

func sobrePresentePreinicioM38(s *sobreRetenidoM38) bool {
	return len(s.nonce) == 64 && s.pidRunner != "" && s.selector != "" &&
		s.identidad != "" && s.longitudTicket != "" && s.ticket != ""
}

func receptorConfirmadoPreinicioM38(r *receptorSobreS0M38) bool {
	return r.lector != nil && r.lector.clase == "SOBRE" && r.lector.limite == 4096 &&
		r.lector.estado == lectorEOFLimpioM38 && r.lector.err == nil &&
		lectorLimpioM38(r.lector) && sobrePresentePreinicioM38(&r.sobre)
}

func retirarReceptorPreinicioM38(r *receptorSobreS0M38, fallo error) {
	if r.lector != nil {
		r.lector.limpiarBuffer()
	}
	limpiarSobreS0M38(&r.sobre)
	r.fase, r.fallo = recepcionSobreS0M38, fallo
}

func nuevoControladorPreinicioM38(
	recepcion *receptorSobreS0M38,
) (*controladorPreinicioM38, error) {
	if recepcion == nil {
		return nil, errSobreNoConfirmadoPreinicioM38
	}
	if recepcion.fallo != nil {
		fallo := recepcion.fallo
		retirarReceptorPreinicioM38(recepcion, fallo)
		return nil, fallo
	}
	if recepcion.fase == recepcionSobreS0M38 && receptorPristinoPreinicioM38(recepcion) {
		retirarReceptorPreinicioM38(recepcion, errSobreNoConfirmadoPreinicioM38)
		return nil, errSobreNoConfirmadoPreinicioM38
	}
	if recepcion.fase != recepcionSobreS1M38 || !receptorConfirmadoPreinicioM38(recepcion) {
		retirarReceptorPreinicioM38(recepcion, errInvarianteControlPreinicioM38)
		return nil, errInvarianteControlPreinicioM38
	}
	controlador := &controladorPreinicioM38{recepcion: recepcion, fase: controlPreinicioS1M38}
	copy(controlador.nonce[:], recepcion.sobre.nonce)
	lector, err := nuevoLectorTramaM38("CONTROL")
	if err != nil {
		retirarReceptorPreinicioM38(recepcion, err)
		return nil, err
	}
	controlador.lector = lector
	return controlador, nil
}

func nonceCoincidePreinicioM38(esperado *[64]byte, recibido string) bool {
	if len(recibido) != len(esperado) {
		return false
	}
	for i := range esperado {
		if esperado[i] != recibido[i] {
			return false
		}
	}
	return true
}

func lectorActivoPreinicioM38(l *lectorTramaM38) bool {
	if l == nil || l.clase != "CONTROL" || l.limite != 1024 || l.err != nil {
		return false
	}
	if l.estado == lectorAbiertoVacioM38 {
		return lectorLimpioM38(l)
	}
	if l.estado != lectorAbiertoParcialM38 || l.longitud <= 0 || l.longitud >= l.limite {
		return false
	}
	for i := 0; i < len(l.buffer); i++ {
		b := l.buffer[i]
		if i < l.longitud && (b < 0x20 || b > 0x7e) || i >= l.longitud && b != 0 {
			return false
		}
	}
	return true
}

func falloInvarianteActivoPreinicioM38(c *controladorPreinicioM38) error {
	if c.fase != controlPreinicioS1M38 && c.fase != controlPreinicioS2M38 && c.fase != controlPreinicioS3M38 ||
		c.causa != (causaPreinicioM38{}) || c.recepcion == nil || c.lector == nil {
		return errInvarianteControlPreinicioM38
	}
	if c.lector.estado == lectorErrorTerminalM38 && c.lector.err != nil {
		return c.lector.err
	}
	if c.recepcion.fallo != nil {
		return c.recepcion.fallo
	}
	if c.recepcion.fase != recepcionSobreS1M38 || c.recepcion.lector == nil ||
		c.recepcion.lector.clase != "SOBRE" || c.recepcion.lector.limite != 4096 ||
		c.recepcion.lector.estado != lectorEOFLimpioM38 || c.recepcion.lector.err != nil ||
		!lectorLimpioM38(c.recepcion.lector) || !sobrePresentePreinicioM38(&c.recepcion.sobre) ||
		!nonceCoincidePreinicioM38(&c.nonce, c.recepcion.sobre.nonce) ||
		!lectorActivoPreinicioM38(c.lector) {
		return errInvarianteControlPreinicioM38
	}
	return nil
}

func (c *controladorPreinicioM38) enclavarCausa(causa, estado string) {
	origen := c.fase
	if c.lector != nil {
		c.lector.limpiarBuffer()
	}
	if c.recepcion != nil {
		retirarReceptorPreinicioM38(c.recepcion, errUsoPosteriorSobreM38)
	}
	c.fase = controlPreinicioS5M38
	c.causa = causaPreinicioM38{faseOrigen: origen, causa: causa, estado: estado}
	c.fallo, c.recepcion, c.lector = nil, nil, nil
}

func (c *controladorPreinicioM38) limpiarConFallo(fallo error) {
	if c.fallo != nil {
		fallo = c.fallo
	}
	if fallo == nil {
		fallo = errInvarianteControlPreinicioM38
	}
	if c.lector != nil {
		c.lector.limpiarBuffer()
	}
	if c.recepcion != nil {
		retirarReceptorPreinicioM38(c.recepcion, fallo)
	}
	c.fase, c.causa, c.fallo = controlPreinicioS5M38, causaPreinicioM38{}, fallo
	c.recepcion, c.lector = nil, nil
}

func terminalInternoValidoPreinicioM38(c *controladorPreinicioM38) bool {
	return c.fase == controlPreinicioS5M38 && c.fallo != nil &&
		c.causa == (causaPreinicioM38{}) && c.recepcion == nil && c.lector == nil
}
func terminalFuncionalValidoPreinicioM38(c *controladorPreinicioM38) bool {
	if c.fase != controlPreinicioS5M38 || c.fallo != nil || c.recepcion != nil || c.lector != nil ||
		(c.causa.faseOrigen != controlPreinicioS1M38 && c.causa.faseOrigen != controlPreinicioS2M38 &&
			c.causa.faseOrigen != controlPreinicioS3M38) {
		return false
	}
	_, _, valida := causaTransportadaPreinicioM38(c.causa.causa, c.causa.estado)
	return valida && (c.causa.faseOrigen != controlPreinicioS1M38 ||
		c.causa.causa == "CANCELADO" || c.causa.causa == "PROTOCOLO")
}

func (c *controladorPreinicioM38) retornarTerminal(consumidos int, posterior bool) (int, resultadoControlPreinicioM38, error) {
	if c.fallo != nil {
		fallo := c.fallo
		if !terminalInternoValidoPreinicioM38(c) {
			c.limpiarConFallo(fallo)
		}
		return 0, 0, fallo
	}
	if terminalFuncionalValidoPreinicioM38(c) {
		if posterior {
			return 0, 0, errUsoPosteriorControlPreinicioM38
		}
		return consumidos, controlPreinicioCausaEnclavadaM38, nil
	}
	c.limpiarConFallo(errInvarianteControlPreinicioM38)
	return 0, 0, c.fallo
}

func (c *controladorPreinicioM38) invalidar(fallo error) (int, resultadoControlPreinicioM38, error) {
	c.limpiarConFallo(fallo)
	return c.retornarTerminal(0, false)
}

func errorControlNormalizablePreinicioM38(err error) bool {
	return errors.Is(err, errByteFlujoM38) || errors.Is(err, errExcesoFlujoM38) ||
		errors.Is(err, errTramaFlujoM38) || errors.Is(err, errEOFParcialM38)
}

func causaTransportadaPreinicioM38(causa, estado string) (string, string, bool) {
	switch causa {
	case "CANCELADO":
		return "CANCELADO", "65", estado == "65"
	case "PROTOCOLO":
		return "PROTOCOLO", "65", estado == "65"
	case "SENAL_INT":
		return "SENAL_INT", "130", estado == "130"
	case "SENAL_TERM":
		return "SENAL_TERM", "143", estado == "143"
	}
	return "", "", false
}

func tramaControlEstructuralPreinicioM38(t tramaM38) bool {
	if t.clase != "CONTROL" || len(t.campos) == 0 || t.ticket != "" {
		return false
	}
	return t.campos[0] == "ARMAR" && len(t.campos) == 3 ||
		t.campos[0] == "INICIAR" && len(t.campos) == 2 ||
		t.campos[0] == "CANCELAR" && len(t.campos) == 4
}

func (c *controladorPreinicioM38) aplicar(t tramaM38) (terminal bool, fallo error) {
	if !tramaControlEstructuralPreinicioM38(t) {
		return false, errInvarianteControlPreinicioM38
	}
	validaNonce := nonceCoincidePreinicioM38(&c.nonce, t.campos[1])
	switch c.fase {
	case controlPreinicioS1M38:
		if t.campos[0] == "ARMAR" && validaNonce && t.campos[2] == c.recepcion.sobre.pidRunner {
			c.fase = controlPreinicioS2M38
			return false, nil
		}
	case controlPreinicioS2M38:
		if t.campos[0] == "INICIAR" && validaNonce {
			c.fase = controlPreinicioS3M38
			return false, nil
		}
		if t.campos[0] == "CANCELAR" && validaNonce {
			causa, estado, valida := causaTransportadaPreinicioM38(t.campos[2], t.campos[3])
			if valida {
				c.enclavarCausa(causa, estado)
				return true, nil
			}
		}
	case controlPreinicioS3M38:
		if t.campos[0] == "CANCELAR" && validaNonce {
			causa, estado, valida := causaTransportadaPreinicioM38(t.campos[2], t.campos[3])
			if valida {
				c.enclavarCausa(causa, estado)
				return true, nil
			}
		}
	default:
		return false, errInvarianteControlPreinicioM38
	}
	c.enclavarCausa("PROTOCOLO", "65")
	return true, nil
}

func resultadoActivoPreinicioM38(c *controladorPreinicioM38) resultadoControlPreinicioM38 {
	if c.lector.estado == lectorAbiertoParcialM38 || c.fase == controlPreinicioS1M38 {
		return controlPreinicioNecesitaDatosM38
	}
	if c.fase == controlPreinicioS2M38 {
		return controlPreinicioArmadoM38
	}
	return controlPreinicioInicioPendienteM38
}

func tuplaLectorValidaPreinicioM38(
	l *lectorTramaM38, sufijo []byte, fin bool,
	trama tramaM38, n int, lectura resultadoLecturaM38, fallo error,
) bool {
	if l == nil || l.clase != "CONTROL" || l.limite != 1024 ||
		n < 0 || n > len(sufijo) || n > l.limite {
		return false
	}
	if fallo != nil {
		return n == 0 && lectura == lecturaNecesitaDatosM38 && tramaCeroM38(trama) &&
			l.estado == lectorErrorTerminalM38 && l.err == fallo && lectorLimpioM38(l)
	}
	if l.err != nil {
		return false
	}
	switch lectura {
	case lecturaNecesitaDatosM38:
		return !fin && n == len(sufijo) && tramaCeroM38(trama) && lectorActivoPreinicioM38(l) &&
			(n == 0 || l.estado == lectorAbiertoParcialM38 && n <= l.longitud)
	case lecturaTramaM38:
		ultimoEOF := fin && n == len(sufijo)
		estadoValido := !ultimoEOF && l.estado == lectorAbiertoVacioM38 ||
			ultimoEOF && l.estado == lectorEOFLimpioM38
		return n > 0 && tramaControlEstructuralPreinicioM38(trama) &&
			estadoValido && lectorLimpioM38(l)
	case lecturaEOFLimpioM38:
		return n == 0 && len(sufijo) == 0 && fin && tramaCeroM38(trama) &&
			l.estado == lectorEOFLimpioM38 && lectorLimpioM38(l)
	}
	return false
}

func (c *controladorPreinicioM38) consumir(
	fragmento []byte,
	fin bool,
) (consumidos int, resultado resultadoControlPreinicioM38, err error) {
	if c == nil {
		return 0, 0, errControlPreinicioNuloM38
	}
	if c.fallo != nil || c.fase == controlPreinicioS5M38 {
		return c.retornarTerminal(0, true)
	}
	if fallo := falloInvarianteActivoPreinicioM38(c); fallo != nil {
		return c.invalidar(fallo)
	}
	tramas := 0
	for {
		if tramas == 3 {
			return c.invalidar(errInvarianteControlPreinicioM38)
		}
		sufijo := fragmento[consumidos:]
		trama, n, lectura, fallo := c.lector.consumir(sufijo, fin)
		if !tuplaLectorValidaPreinicioM38(c.lector, sufijo, fin, trama, n, lectura, fallo) {
			return c.invalidar(errInvarianteControlPreinicioM38)
		}
		if fallo != nil {
			if errorControlNormalizablePreinicioM38(fallo) {
				c.enclavarCausa("PROTOCOLO", "65")
				return c.retornarTerminal(consumidos, false)
			}
			return c.invalidar(fallo)
		}
		if consumidos+n > len(fragmento) || consumidos+n > 3072 {
			return c.invalidar(errInvarianteControlPreinicioM38)
		}
		switch lectura {
		case lecturaNecesitaDatosM38:
			consumidos += n
			return consumidos, resultadoActivoPreinicioM38(c), nil
		case lecturaTramaM38:
			consumidos, tramas = consumidos+n, tramas+1
			terminal, fallo := c.aplicar(trama)
			if fallo != nil {
				return c.invalidar(fallo)
			}
			if terminal {
				return c.retornarTerminal(consumidos, false)
			}
			if c.lector.estado == lectorEOFLimpioM38 {
				trama, n, lectura, fallo = c.lector.consumir(nil, true)
				if fallo != nil || !tuplaLectorValidaPreinicioM38(c.lector, nil, true, trama, n, lectura, fallo) ||
					consumidos != len(fragmento) || !fin {
					return c.invalidar(errInvarianteControlPreinicioM38)
				}
				c.enclavarCausa("CANCELADO", "65")
				return c.retornarTerminal(consumidos, false)
			}
			if c.lector.estado != lectorAbiertoVacioM38 || !lectorLimpioM38(c.lector) {
				return c.invalidar(errInvarianteControlPreinicioM38)
			}
			if consumidos == len(fragmento) {
				if fin {
					return c.invalidar(errInvarianteControlPreinicioM38)
				}
				return consumidos, resultadoActivoPreinicioM38(c), nil
			}
		case lecturaEOFLimpioM38:
			c.enclavarCausa("CANCELADO", "65")
			return c.retornarTerminal(consumidos, false)
		default:
			return c.invalidar(errInvarianteControlPreinicioM38)
		}
	}
}
