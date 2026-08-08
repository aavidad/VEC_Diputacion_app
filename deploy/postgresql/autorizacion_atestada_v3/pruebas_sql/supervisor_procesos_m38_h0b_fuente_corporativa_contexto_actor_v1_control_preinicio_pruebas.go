//go:build ignore && linux && amd64

package main

import (
	"errors"
	"fmt"
	"strings"
)

func receptorConfirmadoPruebaPreinicioM38() (*receptorSobreS0M38, error) {
	receptor, err := nuevoReceptorSobreS0M38()
	if err != nil {
		return nil, err
	}
	texto := textoSobreS0M38("37", "A00", "ticket|sensible")
	if n, resultado, fallo := receptor.consumir([]byte(texto), true); fallo != nil || n != len(texto) || resultado != recepcionSobreConfirmadaM38 {
		return nil, errors.New("no se confirmó el sobre de prueba")
	}
	return receptor, nil
}
func controladorNuevoPruebaPreinicioM38(fase faseControlPreinicioM38) (*controladorPreinicioM38, *receptorSobreS0M38, error) {
	receptor, err := receptorConfirmadoPruebaPreinicioM38()
	if err != nil {
		return nil, nil, err
	}
	controlador, err := nuevoControladorPreinicioM38(receptor)
	if err != nil {
		return nil, receptor, err
	}
	h := strings.Repeat("a", 64)
	if fase >= controlPreinicioS2M38 {
		armar := []byte("V1|CONTROL|ARMAR|" + h + "|37\n")
		if n, resultado, fallo := controlador.consumir(armar, false); fallo != nil || n != len(armar) || resultado != controlPreinicioArmadoM38 {
			return nil, receptor, errors.New("no se alcanzó S2 en la prueba")
		}
	}
	if fase >= controlPreinicioS3M38 {
		iniciar := []byte("V1|CONTROL|INICIAR|" + h + "\n")
		if n, resultado, fallo := controlador.consumir(iniciar, false); fallo != nil || n != len(iniciar) || resultado != controlPreinicioInicioPendienteM38 {
			return nil, receptor, errors.New("no se alcanzó S3 en la prueba")
		}
	}
	return controlador, receptor, nil
}
func exigirCausaPruebaPreinicioM38(c *controladorPreinicioM38, r *receptorSobreS0M38, fase faseControlPreinicioM38, causa, estado string) error {
	if c.fase != controlPreinicioS5M38 || c.causa != (causaPreinicioM38{fase, causa, estado}) || c.fallo != nil ||
		c.recepcion != nil || c.lector != nil || r.fase != recepcionSobreS0M38 || r.fallo != errUsoPosteriorSobreM38 ||
		r.sobre != (sobreRetenidoM38{}) || r.lector == nil || !lectorLimpioM38(r.lector) {
		return errors.New("causa o retirada sensible discrepante")
	}
	return nil
}
func probarConstructoresPreinicioM38() error {
	if c, err := nuevoControladorPreinicioM38(nil); c != nil || err != errSobreNoConfirmadoPreinicioM38 {
		return errors.New("constructor nulo aceptado")
	}
	nuevo, err := nuevoReceptorSobreS0M38()
	if err != nil {
		return err
	}
	if c, fallo := nuevoControladorPreinicioM38(nuevo); c != nil || fallo != errSobreNoConfirmadoPreinicioM38 || nuevo.fallo != fallo || nuevo.fase != recepcionSobreS0M38 {
		return errors.New("receptor S0 aceptado")
	}
	falloPrevio := fmt.Errorf("fallo previo")
	fallido := &receptorSobreS0M38{lector: &lectorTramaM38{buffer: [4096]byte{'x'}, longitud: 1}, sobre: sobreRetenidoM38{ticket: "x"}, fallo: falloPrevio}
	if c, fallo := nuevoControladorPreinicioM38(fallido); c != nil || fallo != falloPrevio || fallido.fallo != falloPrevio || fallido.sobre != (sobreRetenidoM38{}) || !lectorLimpioM38(fallido.lector) {
		return errors.New("fallo previo no se conservó")
	}
	parcial, err := nuevoReceptorSobreS0M38()
	if err != nil {
		return err
	}
	if _, _, err = parcial.consumir([]byte("V"), false); err != nil {
		return err
	}
	if c, fallo := nuevoControladorPreinicioM38(parcial); c != nil || fallo != errInvarianteControlPreinicioM38 || parcial.fallo != fallo || !lectorLimpioM38(parcial.lector) {
		return errors.New("receptor parcial no falló cerrado")
	}
	c, r, err := controladorNuevoPruebaPreinicioM38(controlPreinicioS1M38)
	if err != nil || c.recepcion != r || c.lector == nil || c.lector.clase != "CONTROL" || c.lector.limite != 1024 ||
		c.fase != controlPreinicioS1M38 || c.causa != (causaPreinicioM38{}) || c.fallo != nil || !nonceCoincidePreinicioM38(&c.nonce, strings.Repeat("a", 64)) {
		return errors.New("constructor confirmado discrepante")
	}
	for i, mutar := range []func(*receptorSobreS0M38){
		func(r *receptorSobreS0M38) { r.sobre.nonce = "" }, func(r *receptorSobreS0M38) { r.sobre.nonce = strings.Repeat("a", 63) },
		func(r *receptorSobreS0M38) { r.sobre.pidRunner = "" }, func(r *receptorSobreS0M38) { r.sobre.selector = "" },
		func(r *receptorSobreS0M38) { r.sobre.identidad = "" }, func(r *receptorSobreS0M38) { r.sobre.longitudTicket = "" },
		func(r *receptorSobreS0M38) { r.sobre.ticket = "" }, func(r *receptorSobreS0M38) { r.lector = nil },
		func(r *receptorSobreS0M38) { r.lector.estado = lectorAbiertoVacioM38 }, func(r *receptorSobreS0M38) { r.lector.buffer[0] = 'x' },
		func(r *receptorSobreS0M38) { r.lector.err = errUsoPosteriorEOFM38 },
	} {
		r, err = receptorConfirmadoPruebaPreinicioM38()
		if err != nil {
			return err
		}
		mutar(r)
		if c, fallo := nuevoControladorPreinicioM38(r); c != nil || fallo != errInvarianteControlPreinicioM38 || r.fallo != fallo || r.sobre != (sobreRetenidoM38{}) {
			return fmt.Errorf("receptor incoherente %d aceptado", i)
		}
	}
	return nil
}
func probarFragmentacionPreinicioM38() error {
	h := strings.Repeat("a", 64)
	armar := "V1|CONTROL|ARMAR|" + h + "|37\n"
	for corte := 0; corte <= len(armar); corte++ {
		c, r, err := controladorNuevoPruebaPreinicioM38(controlPreinicioS1M38)
		if err != nil {
			return err
		}
		prefijo := []byte(armar[:corte])
		n, resultado, fallo := c.consumir(prefijo, false)
		esperado := controlPreinicioNecesitaDatosM38
		if corte == len(armar) {
			esperado = controlPreinicioArmadoM38
		}
		if fallo != nil || n != corte || resultado != esperado {
			return fmt.Errorf("corte ARMAR %d discrepante", corte)
		}
		if corte > 0 {
			prefijo[0] ^= 1
		}
		n, resultado, fallo = c.consumir([]byte(armar[corte:]), false)
		if fallo != nil || n != len(armar)-corte || resultado != controlPreinicioArmadoM38 || c.fase != controlPreinicioS2M38 || c.recepcion != r {
			return fmt.Errorf("reensamblado ARMAR %d discrepante", corte)
		}
	}
	c, _, err := controladorNuevoPruebaPreinicioM38(controlPreinicioS1M38)
	if err != nil {
		return err
	}
	for i := range len(armar) {
		n, resultado, fallo := c.consumir([]byte{armar[i]}, false)
		if fallo != nil || n != 1 || i < len(armar)-1 && resultado != controlPreinicioNecesitaDatosM38 || i == len(armar)-1 && resultado != controlPreinicioArmadoM38 {
			return fmt.Errorf("byte ARMAR %d discrepante", i)
		}
	}
	iniciar := "V1|CONTROL|INICIAR|" + h + "\n"
	c, _, err = controladorNuevoPruebaPreinicioM38(controlPreinicioS2M38)
	prefijo := []byte(iniciar[:11])
	if err != nil {
		return err
	}
	if _, _, fallo := c.consumir(prefijo, false); fallo != nil {
		return fallo
	}
	prefijo[0] ^= 1
	if n, resultado, fallo := c.consumir([]byte(iniciar[11:]), false); fallo != nil || n != len(iniciar)-11 || resultado != controlPreinicioInicioPendienteM38 {
		return errors.New("INICIAR dependía del fragmento del llamador")
	}
	return nil
}
func probarEstadosYSecuenciasPreinicioM38() error {
	h, b := strings.Repeat("a", 64), strings.Repeat("b", 64)
	estables := []struct {
		fase      faseControlPreinicioM38
		resultado resultadoControlPreinicioM38
	}{{controlPreinicioS1M38, controlPreinicioNecesitaDatosM38}, {controlPreinicioS2M38, controlPreinicioArmadoM38}, {controlPreinicioS3M38, controlPreinicioInicioPendienteM38}}
	for _, caso := range estables {
		c, r, err := controladorNuevoPruebaPreinicioM38(caso.fase)
		if err != nil {
			return err
		}
		if n, resultado, fallo := c.consumir(nil, false); fallo != nil || n != 0 || resultado != caso.resultado || !sobreRetenidoExactoS0M38(r, "37", "A00", "ticket|sensible") {
			return errors.New("estado vacío activo discrepante")
		}
		if n, resultado, fallo := c.consumir([]byte{'V'}, false); fallo != nil || n != 1 || resultado != controlPreinicioNecesitaDatosM38 {
			return errors.New("L1 activo discrepante")
		}
		if n, resultado, fallo := c.consumir(nil, false); fallo != nil || n != 0 || resultado != controlPreinicioNecesitaDatosM38 {
			return errors.New("L1 vacío alteró la parcial")
		}
	}
	secuencias := []struct {
		fase    faseControlPreinicioM38
		entrada string
	}{
		{controlPreinicioS2M38, "V1|CONTROL|ARMAR|" + h + "|37\n"}, {controlPreinicioS3M38, "V1|CONTROL|ARMAR|" + h + "|37\n"},
		{controlPreinicioS3M38, "V1|CONTROL|INICIAR|" + h + "\n"}, {controlPreinicioS2M38, "V1|CONTROL|INICIAR|" + b + "\n"},
		{controlPreinicioS2M38, "V1|CONTROL|CANCELAR|" + b + "|CANCELADO|65\n"}, {controlPreinicioS3M38, "V1|CONTROL|CANCELAR|" + b + "|CANCELADO|65\n"},
	}
	for _, caso := range secuencias {
		c, r, err := controladorNuevoPruebaPreinicioM38(caso.fase)
		if err != nil {
			return err
		}
		if n, resultado, fallo := c.consumir([]byte(caso.entrada), false); fallo != nil || n != len(caso.entrada) || resultado != controlPreinicioCausaEnclavadaM38 {
			return errors.New("secuencia de S2/S3 no se normalizó")
		}
		if err := exigirCausaPruebaPreinicioM38(c, r, caso.fase, "PROTOCOLO", "65"); err != nil {
			return err
		}
	}
	return nil
}
func probarDrenajePreinicioM38() error {
	h := strings.Repeat("a", 64)
	armar := "V1|CONTROL|ARMAR|" + h + "|37\n"
	iniciar := "V1|CONTROL|INICIAR|" + h + "\n"
	cancelar := "V1|CONTROL|CANCELAR|" + h + "|SENAL_TERM|143\n"
	casos := []struct {
		entrada       string
		consumidos    int
		resultado     resultadoControlPreinicioM38
		fase          faseControlPreinicioM38
		causa, estado string
	}{
		{armar + iniciar, len(armar + iniciar), controlPreinicioInicioPendienteM38, controlPreinicioS3M38, "", ""},
		{armar + cancelar, len(armar + cancelar), controlPreinicioCausaEnclavadaM38, controlPreinicioS2M38, "SENAL_TERM", "143"},
		{armar + iniciar + cancelar, len(armar + iniciar + cancelar), controlPreinicioCausaEnclavadaM38, controlPreinicioS3M38, "SENAL_TERM", "143"},
	}
	for _, caso := range casos {
		c, r, err := controladorNuevoPruebaPreinicioM38(controlPreinicioS1M38)
		if err != nil {
			return err
		}
		n, resultado, fallo := c.consumir([]byte(caso.entrada), false)
		if fallo != nil || n != caso.consumidos || resultado != caso.resultado {
			return errors.New("drenaje coalescido discrepante")
		}
		if caso.causa == "" && (c.fase != caso.fase || c.recepcion != r) {
			return errors.New("inicio pendiente no conservó el sobre")
		}
		if caso.causa != "" {
			if err := exigirCausaPruebaPreinicioM38(c, r, caso.fase, caso.causa, caso.estado); err != nil {
				return err
			}
		}
	}
	c, r, err := controladorNuevoPruebaPreinicioM38(controlPreinicioS1M38)
	if err != nil {
		return err
	}
	grande := []byte(armar + cancelar + strings.Repeat("x", 1<<20))
	n, resultado, fallo := c.consumir(grande, false)
	if fallo != nil || n != len(armar+cancelar) || resultado != controlPreinicioCausaEnclavadaM38 || len(grande[n:]) != 1<<20 {
		return errors.New("S5 recorrió el sufijo posterior")
	}
	if err := exigirCausaPruebaPreinicioM38(c, r, controlPreinicioS2M38, "SENAL_TERM", "143"); err != nil {
		return err
	}
	c, _, err = controladorNuevoPruebaPreinicioM38(controlPreinicioS1M38)
	parcial := cancelar[:17]
	if err != nil {
		return err
	}
	if n, resultado, fallo := c.consumir([]byte(armar+iniciar+parcial), false); fallo != nil || n != len(armar+iniciar+parcial) || resultado != controlPreinicioNecesitaDatosM38 || c.fase != controlPreinicioS3M38 {
		return errors.New("coalescencia con CANCELAR parcial ocultada")
	}
	return nil
}
func probarCausasYEOFPreinicioM38() error {
	h := strings.Repeat("a", 64)
	for _, fase := range []faseControlPreinicioM38{controlPreinicioS2M38, controlPreinicioS3M38} {
		for _, par := range [][2]string{{"CANCELADO", "65"}, {"PROTOCOLO", "65"}, {"SENAL_INT", "130"}, {"SENAL_TERM", "143"}} {
			c, r, err := controladorNuevoPruebaPreinicioM38(fase)
			if err != nil {
				return err
			}
			entrada := []byte("V1|CONTROL|CANCELAR|" + h + "|" + par[0] + "|" + par[1] + "\n")
			if n, resultado, fallo := c.consumir(entrada, true); fallo != nil || n != len(entrada) || resultado != controlPreinicioCausaEnclavadaM38 {
				return errors.New("cancelación canónica rechazada")
			}
			if err := exigirCausaPruebaPreinicioM38(c, r, fase, par[0], par[1]); err != nil {
				return err
			}
			if n, resultado, fallo := c.consumir([]byte(strings.Repeat("x", 1<<20)), true); fallo != errUsoPosteriorControlPreinicioM38 || n != 0 || resultado != 0 {
				return errors.New("uso posterior funcional discrepante")
			}
		}
	}
	for _, fase := range []faseControlPreinicioM38{controlPreinicioS1M38, controlPreinicioS2M38, controlPreinicioS3M38} {
		c, r, err := controladorNuevoPruebaPreinicioM38(fase)
		if err != nil {
			return err
		}
		if n, resultado, fallo := c.consumir(nil, true); fallo != nil || n != 0 || resultado != controlPreinicioCausaEnclavadaM38 {
			return errors.New("EOF limpio rechazado")
		}
		if err := exigirCausaPruebaPreinicioM38(c, r, fase, "CANCELADO", "65"); err != nil {
			return err
		}
	}
	c, r, err := controladorNuevoPruebaPreinicioM38(controlPreinicioS1M38)
	if err != nil {
		return err
	}
	armar := []byte("V1|CONTROL|ARMAR|" + h + "|37\n")
	if n, resultado, fallo := c.consumir(armar, true); fallo != nil || n != len(armar) || resultado != controlPreinicioCausaEnclavadaM38 {
		return errors.New("ARMAR más EOF no canceló")
	}
	if err := exigirCausaPruebaPreinicioM38(c, r, controlPreinicioS2M38, "CANCELADO", "65"); err != nil {
		return err
	}
	c, r, err = controladorNuevoPruebaPreinicioM38(controlPreinicioS2M38)
	iniciar := []byte("V1|CONTROL|INICIAR|" + h + "\n")
	if err != nil {
		return err
	}
	if n, resultado, fallo := c.consumir(iniciar, true); fallo != nil || n != len(iniciar) || resultado != controlPreinicioCausaEnclavadaM38 {
		return errors.New("INICIAR más EOF no canceló")
	}
	return exigirCausaPruebaPreinicioM38(c, r, controlPreinicioS3M38, "CANCELADO", "65")
}
func probarParcialesYErroresPreinicioM38() error {
	h := strings.Repeat("a", 64)
	c, r, err := controladorNuevoPruebaPreinicioM38(controlPreinicioS3M38)
	if err != nil {
		return err
	}
	cancelar := "V1|CONTROL|CANCELAR|" + h + "|PROTOCOLO|65\n"
	parcial := []byte(cancelar[:17])
	if n, resultado, fallo := c.consumir(parcial, false); fallo != nil || n != len(parcial) || resultado != controlPreinicioNecesitaDatosM38 {
		return errors.New("parcial posterior a INICIAR ocultada")
	}
	parcial[0] ^= 1
	if n, resultado, fallo := c.consumir([]byte(cancelar[17:]), false); fallo != nil || n != len(cancelar)-17 || resultado != controlPreinicioCausaEnclavadaM38 {
		return errors.New("cancelación fragmentada rechazada")
	}
	if err := exigirCausaPruebaPreinicioM38(c, r, controlPreinicioS3M38, "PROTOCOLO", "65"); err != nil {
		return err
	}
	for _, entrada := range []string{
		"V1|CONTROL|INICIAR|" + h + "\n",
		"V1|CONTROL|CANCELAR|" + h + "|SENAL_INT|130\n",
		"V1|CONTROL|ARMAR|" + strings.Repeat("b", 64) + "|37\n",
		"V1|CONTROL|ARMAR|" + h + "|38\n",
	} {
		c, r, err = controladorNuevoPruebaPreinicioM38(controlPreinicioS1M38)
		if err != nil {
			return err
		}
		datos := []byte(entrada)
		if n, resultado, fallo := c.consumir(datos, false); fallo != nil || n != len(datos) || resultado != controlPreinicioCausaEnclavadaM38 {
			return errors.New("secuencia inválida no se normalizó")
		}
		if err := exigirCausaPruebaPreinicioM38(c, r, controlPreinicioS1M38, "PROTOCOLO", "65"); err != nil {
			return err
		}
	}
	for _, entrada := range [][]byte{
		[]byte("V2|CONTROL|INICIAR|" + h + "\n"), []byte("V1|OTRA|INICIAR|" + h + "\n"),
		[]byte("V1|CONTROL|INICIAR|" + h + "|extra\n"), []byte("V1|CONTROL|CANCELAR|" + h + "|CANCELADO|64\n"),
		{'V', 0}, {'V', '\r'}, {'V', '\t'}, {'V', 0x80}, []byte(strings.Repeat("x", 1023) + "\n"),
		[]byte(strings.Repeat("x", 1024)), []byte(strings.Repeat("x", 1025)),
		[]byte("V2|CONTROL|INICIAR|" + h + "\nV1|CONTROL|ARMAR|" + h + "|37\n"),
	} {
		c, r, err = controladorNuevoPruebaPreinicioM38(controlPreinicioS1M38)
		if err != nil {
			return err
		}
		if n, resultado, fallo := c.consumir(entrada, false); fallo != nil || n != 0 || resultado != controlPreinicioCausaEnclavadaM38 {
			return errors.New("error físico no se normalizó")
		}
		if err := exigirCausaPruebaPreinicioM38(c, r, controlPreinicioS1M38, "PROTOCOLO", "65"); err != nil {
			return err
		}
	}
	c, r, err = controladorNuevoPruebaPreinicioM38(controlPreinicioS2M38)
	if err != nil {
		return err
	}
	if n, resultado, fallo := c.consumir([]byte("V"), false); fallo != nil || n != 1 || resultado != controlPreinicioNecesitaDatosM38 {
		return errors.New("parcial inicial no retenida")
	}
	if n, resultado, fallo := c.consumir([]byte("1"), true); fallo != nil || n != 0 || resultado != controlPreinicioCausaEnclavadaM38 {
		return errors.New("EOF parcial no se normalizó")
	}
	return exigirCausaPruebaPreinicioM38(c, r, controlPreinicioS2M38, "PROTOCOLO", "65")
}
func probarInvariantesPreinicioM38() error {
	var nulo *controladorPreinicioM38
	if n, resultado, err := nulo.consumir(nil, false); err != errControlPreinicioNuloM38 || n != 0 || resultado != 0 {
		return errors.New("controlador nulo no falló cerrado")
	}
	for _, mutar := range []func(*controladorPreinicioM38){
		func(c *controladorPreinicioM38) { c.fase = faseControlPreinicioM38(9) },
		func(c *controladorPreinicioM38) { c.causa = causaPreinicioM38{causa: "PROTOCOLO"} },
		func(c *controladorPreinicioM38) { c.lector.estado = lectorMonotramaEsperandoEOFM38 },
		func(c *controladorPreinicioM38) { c.lector.estado = lectorEOFLimpioM38 },
		func(c *controladorPreinicioM38) { c.recepcion.lector.buffer[0] = 'x' },
		func(c *controladorPreinicioM38) { c.nonce[0] = 'b' },
		func(c *controladorPreinicioM38) { c.recepcion.sobre.nonce = "" },
		func(c *controladorPreinicioM38) { c.recepcion.sobre.pidRunner = "" },
		func(c *controladorPreinicioM38) { c.recepcion.sobre.selector = "" },
		func(c *controladorPreinicioM38) { c.recepcion.sobre.identidad = "" },
		func(c *controladorPreinicioM38) { c.recepcion.sobre.longitudTicket = "" },
		func(c *controladorPreinicioM38) { c.recepcion.sobre.ticket = "" },
	} {
		c, r, err := controladorNuevoPruebaPreinicioM38(controlPreinicioS1M38)
		if err != nil {
			return err
		}
		mutar(c)
		n, resultado, primerFallo := c.consumir([]byte("adverso"), false)
		if primerFallo == nil || n != 0 || resultado != 0 || c.fase != controlPreinicioS5M38 || c.causa != (causaPreinicioM38{}) || c.fallo != primerFallo || c.recepcion != nil || c.lector != nil || r.sobre != (sobreRetenidoM38{}) || r.fallo != primerFallo {
			return errors.New("invariante no retiró estado sensible")
		}
		if n, resultado, fallo := c.consumir(nil, false); fallo != primerFallo || n != 0 || resultado != 0 {
			return errors.New("fallo interno no quedó pegajoso")
		}
	}
	c, _, err := controladorNuevoPruebaPreinicioM38(controlPreinicioS1M38)
	if err != nil {
		return err
	}
	desconocido := fmt.Errorf("error físico desconocido")
	c.lector.estado, c.lector.err = lectorErrorTerminalM38, desconocido
	if n, resultado, fallo := c.consumir(nil, false); fallo != desconocido || n != 0 || resultado != 0 {
		return errors.New("error desconocido no se conservó")
	}
	return nil
}
func probarTerminalesPreinicioM38() error {
	for _, fase := range []faseControlPreinicioM38{controlPreinicioS1M38, controlPreinicioS2M38, controlPreinicioS3M38, controlPreinicioS5M38} {
		c, r, err := controladorNuevoPruebaPreinicioM38(fase)
		if err != nil {
			return err
		}
		if fase == controlPreinicioS5M38 {
			c.fase, c.causa = fase, causaPreinicioM38{causa: "PROTOCOLO"}
		}
		primero := fmt.Errorf("fallo activo %d", fase)
		c.fallo = primero
		if n, resultado, fallo := c.consumir(nil, false); fallo != primero || n != 0 || resultado != 0 || !terminalInternoValidoPreinicioM38(c) || r.fallo != primero {
			return errors.New("fallo activo no conservó el primero")
		}
	}
	for _, cambio := range []func(*controladorPreinicioM38, *receptorSobreS0M38){
		func(c *controladorPreinicioM38, _ *receptorSobreS0M38) { c.causa.faseOrigen = controlPreinicioS5M38 },
		func(c *controladorPreinicioM38, _ *receptorSobreS0M38) { c.causa.estado = "64" },
		func(c *controladorPreinicioM38, r *receptorSobreS0M38) { c.recepcion = r },
		func(c *controladorPreinicioM38, _ *receptorSobreS0M38) { c.lector = &lectorTramaM38{} },
	} {
		c, r, err := controladorNuevoPruebaPreinicioM38(controlPreinicioS2M38)
		if err != nil {
			return err
		}
		c.enclavarCausa("CANCELADO", "65")
		cambio(c, r)
		if n, resultado, fallo := c.consumir(nil, false); fallo != errInvarianteControlPreinicioM38 || n != 0 || resultado != 0 || !terminalInternoValidoPreinicioM38(c) {
			return errors.New("S5 funcional malformada no se limpió")
		}
	}
	return nil
}
func probarTuplasLectorPreinicioM38() error {
	l0 := &lectorTramaM38{clase: "CONTROL", limite: 1024}
	valida := tramaM38{clase: "CONTROL", campos: []string{"INICIAR", strings.Repeat("a", 64)}}
	conTicket := tramaM38{clase: "CONTROL", campos: valida.campos, ticket: "prohibido"}
	if tuplaLectorValidaPreinicioM38(l0, nil, true, tramaM38{}, 0, lecturaNecesitaDatosM38, nil) ||
		tuplaLectorValidaPreinicioM38(l0, []byte{'x'}, false, tramaM38{}, 1, lecturaNecesitaDatosM38, nil) ||
		tuplaLectorValidaPreinicioM38(l0, nil, false, valida, 0, lecturaTramaM38, nil) ||
		tuplaLectorValidaPreinicioM38(l0, []byte{'x'}, false, conTicket, 1, lecturaTramaM38, nil) {
		return errors.New("tupla O1b imposible aceptada")
	}
	l0Error := &lectorTramaM38{clase: "CONTROL", limite: 1024, err: errInvarianteControlPreinicioM38}
	l0Clase := &lectorTramaM38{clase: "OTRA", limite: 1024}
	l3Limite := &lectorTramaM38{clase: "CONTROL", limite: 2048, estado: lectorEOFLimpioM38}
	if tuplaLectorValidaPreinicioM38(l0Error, nil, false, tramaM38{}, 0, lecturaNecesitaDatosM38, nil) ||
		tuplaLectorValidaPreinicioM38(l0Error, []byte{'x'}, false, valida, 1, lecturaTramaM38, nil) ||
		tuplaLectorValidaPreinicioM38(l0Clase, []byte{'x'}, false, valida, 1, lecturaTramaM38, nil) ||
		tuplaLectorValidaPreinicioM38(l3Limite, nil, true, tramaM38{}, 0, lecturaEOFLimpioM38, nil) {
		return errors.New("tupla O1b con lector incompatible aceptada")
	}
	l1Contador := &lectorTramaM38{clase: "CONTROL", limite: 1024, estado: lectorAbiertoParcialM38, longitud: 1}
	l1Contador.buffer[0] = 'x'
	sufijoLimite := []byte(strings.Repeat("x", 1025))
	if tuplaLectorValidaPreinicioM38(l1Contador, sufijoLimite[:1024], false, tramaM38{}, 1024, lecturaNecesitaDatosM38, nil) ||
		tuplaLectorValidaPreinicioM38(l0, sufijoLimite, false, valida, 1025, lecturaTramaM38, nil) {
		return errors.New("tupla O1b con contador superior al límite aceptada")
	}
	for _, fallo := range []error{errByteFlujoM38, errExcesoFlujoM38, errTramaFlujoM38, errEOFParcialM38} {
		if !errorControlNormalizablePreinicioM38(fallo) || !errorControlNormalizablePreinicioM38(fmt.Errorf("envuelto: %w", fallo)) {
			return errors.New("error normalizable rechazado")
		}
	}
	for _, fallo := range []error{errEOFSinMonotramaM38, errDatosPosterioresM38, errUsoPosteriorEOFM38} {
		if errorControlNormalizablePreinicioM38(fallo) {
			return errors.New("error interno normalizado")
		}
	}
	return nil
}
func autoprobarControlPreinicioM38() error {
	pruebas := []struct {
		nombre string
		fn     func() error
	}{
		{"constructores", probarConstructoresPreinicioM38},
		{"fragmentación", probarFragmentacionPreinicioM38},
		{"estados", probarEstadosYSecuenciasPreinicioM38},
		{"drenaje", probarDrenajePreinicioM38},
		{"causas y EOF", probarCausasYEOFPreinicioM38},
		{"errores", probarParcialesYErroresPreinicioM38},
		{"invariantes", probarInvariantesPreinicioM38},
		{"terminales", probarTerminalesPreinicioM38},
		{"tuplas O1b", probarTuplasLectorPreinicioM38},
	}
	for _, prueba := range pruebas {
		if err := prueba.fn(); err != nil {
			return fmt.Errorf("control previo %s: %w", prueba.nombre, err)
		}
	}
	return nil
}
