//go:build ignore && linux && amd64

package main

import "errors"

const (
	signoIntRawO4aM38  uint8 = 2
	signoTermRawO4aM38 uint8 = 15
)

const (
	causaCancelado65O4aM38 causaPrimariaO4aM38 = iota + 1
	causaProtocolo65O4aM38
	causaSenalInt130O4aM38
	causaSenalTerm143O4aM38
	causaPlazo65O4aM38
	causaSalidaO4aM38
	causaIncidente65O4aM38
)

var errSemillaConsumidaO4aM38 = errors.New("semilla O4a ya consumida")

func causaPrimariaValidaO4aM38(c causaPrimariaO4aM38) bool {
	return c >= causaCancelado65O4aM38 && c <= causaIncidente65O4aM38
}

func causaControlSelladaO4aM38(c canonControlRawO4aM38) (causaPrimariaO4aM38, bool) {
	switch c {
	case controlRawCancelado65O4aM38:
		return causaCancelado65O4aM38, true
	case controlRawProtocolo65O4aM38:
		return causaProtocolo65O4aM38, true
	case controlRawSenalInt130O4aM38:
		return causaSenalInt130O4aM38, true
	case controlRawSenalTerm143O4aM38:
		return causaSenalTerm143O4aM38, true
	}
	return causaVaciaO4aM38, false
}

func causaSenalSelladaO4aM38(baseline, palabra uint64) causaPrimariaO4aM38 {
	contadorBase, contador := baseline>>10, palabra>>10
	if baseline&mascaraEstadoObservadorO3aM38 != 2 || uint8(baseline>>2) != 0 ||
		palabra&mascaraEstadoObservadorO3aM38 != 2 || contador < contadorBase {
		fatalO3cM38()
	}
	if contador != contadorBase+1 {
		return causaIncidente65O4aM38
	}
	switch uint8(palabra >> 2) {
	case signoIntRawO4aM38:
		return causaSenalInt130O4aM38
	case signoTermRawO4aM38:
		return causaSenalTerm143O4aM38
	}
	return causaIncidente65O4aM38
}

func traducirSemillaSelladaO4aM38(s sellosO4aM38) (causaPrimariaO4aM38, bool) {
	if s.retornoCont < 0 {
		fatalO3cM38()
	}
	if s.retornoCont != 0 {
		return causaIncidente65O4aM38, true
	}
	if s.baselineSenal&mascaraEstadoObservadorO3aM38 != 2 || uint8(s.baselineSenal>>2) != 0 {
		fatalO3cM38()
	}
	d := discriminanteObservacionO3cM38(s.primera)
	if !discriminanteObservacionValidoO3cM38(d) {
		fatalO3cM38()
	}
	if d == observacionControlRawO3cM38 {
		c, valida := causaControlSelladaO4aM38(s.canonControlRaw)
		if !valida || s.palabraObservada != s.baselineSenal {
			fatalO3cM38()
		}
		return c, true
	}
	if s.canonControlRaw != controlRawVacioO4aM38 {
		fatalO3cM38()
	}
	switch d {
	case observacionSenalRawO3cM38:
		return causaSenalSelladaO4aM38(s.baselineSenal, s.palabraObservada), true
	case observacionPidfdTerminalNaturalO3cM38:
		if s.palabraObservada != s.baselineSenal {
			fatalO3cM38()
		}
		return causaSalidaO4aM38, true
	case observacionPidfdInfraestructuraO3cM38:
		if s.palabraObservada != s.baselineSenal {
			fatalO3cM38()
		}
		return causaIncidente65O4aM38, true
	case observacionPidfdVacioO3cM38:
		if s.palabraObservada != s.baselineSenal {
			fatalO3cM38()
		}
		return causaVaciaO4aM38, false
	}
	fatalO3cM38()
	return causaVaciaO4aM38, false
}

func sembrarCausaO4aM38(a *autoridadCausaO4aM38) error {
	if a == nil {
		return errSemillaConsumidaO4aM38
	}
	if a.auto != a {
		fatalO3cM38()
	}
	previa := causaPrimariaO4aM38(a.causa.Load())
	if previa != causaVaciaO4aM38 {
		if !causaPrimariaValidaO4aM38(previa) {
			fatalO3cM38()
		}
		return errSemillaConsumidaO4aM38
	}
	if !a.estado.CompareAndSwap(uint32(causaA1ValidadoM38), uint32(causaA2ObservandoM38)) {
		estado := estadoCausaO4aM38(a.estado.Load())
		if estado == causaA2ObservandoM38 || estado == causaA3CausaFijadaM38 {
			return errSemillaConsumidaO4aM38
		}
		fatalO3cM38()
	}
	causa, enclavar := traducirSemillaSelladaO4aM38(a.sellos)
	if !enclavar {
		return nil
	}
	if !a.causa.CompareAndSwap(uint32(causaVaciaO4aM38), uint32(causa)) {
		a.estado.Store(uint32(causaAFFatalM38))
		fatalO3cM38()
	}
	if !a.estado.CompareAndSwap(uint32(causaA2ObservandoM38), uint32(causaA3CausaFijadaM38)) {
		a.estado.Store(uint32(causaAFFatalM38))
		fatalO3cM38()
	}
	return nil
}
