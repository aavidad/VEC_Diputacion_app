package postgres

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

const reservaEstructuralCargaDecisionCoberturaO404E = 8 * 1024 * 1024

// codificarCargaDecisionCoberturaO404E escribe los canones sensibles
// directamente en el único buffer final borrable. Solo usa un bloque temporal
// fijo de 4 KiB para hexadecimal, que se pone a cero en cada iteración.
func codificarCargaDecisionCoberturaO404E(
	carga cargaConfirmarDecisionCoberturaO404E,
) ([]byte, error) {
	bytesCanonicos, err := prevalidarCargaDecisionCoberturaO404E(carga)
	if err != nil {
		return nil, err
	}
	var salida bytes.Buffer
	capacidad := bytesCanonicos*2 + 1024*1024
	if capacidad > maximoBytesCargaDecisionCoberturaO404E {
		capacidad = maximoBytesCargaDecisionCoberturaO404E
	}
	salida.Grow(capacidad)
	fallo := func(err error) ([]byte, error) {
		contenido := salida.Bytes()
		borrarBytes(contenido)
		salida.Reset()
		return nil, errors.Join(errSesionDecisionCoberturaO404EInvalida, err)
	}

	salida.WriteByte('{')
	if err := escribirCampoJSONDecisionCoberturaO404E(
		&salida, "esquema", carga.Esquema, false,
	); err != nil {
		return fallo(err)
	}
	if err := escribirCampoJSONDecisionCoberturaO404E(
		&salida, "rama", carga.Rama, true,
	); err != nil {
		return fallo(err)
	}
	if err := escribirCampoJSONDecisionCoberturaO404E(
		&salida, "cabecera", carga.Cabecera, true,
	); err != nil {
		return fallo(err)
	}
	if err := escribirCampoJSONDecisionCoberturaO404E(
		&salida, "gobierno", carga.Gobierno, true,
	); err != nil {
		return fallo(err)
	}
	escribirNombreCampoDecisionCoberturaO404E(&salida, "decision_vec", true)
	if err := escribirDecisionVECDecisionCoberturaO404E(
		&salida,
		carga.DecisionVEC,
	); err != nil {
		return fallo(err)
	}
	escribirNombreCampoDecisionCoberturaO404E(&salida, "consumos_c1", true)
	salida.WriteByte('[')
	for i := range carga.ConsumosC1 {
		if i > 0 {
			salida.WriteByte(',')
		}
		if err := escribirConsumoC1DecisionCoberturaO404E(
			&salida,
			carga.ConsumosC1[i],
		); err != nil {
			return fallo(err)
		}
	}
	salida.WriteByte(']')
	if err := escribirCampoJSONDecisionCoberturaO404E(
		&salida, "concesion", carga.Concesion, true,
	); err != nil {
		return fallo(err)
	}
	if err := escribirCampoJSONDecisionCoberturaO404E(
		&salida, "denegacion", carga.Denegacion, true,
	); err != nil {
		return fallo(err)
	}
	salida.WriteByte('}')
	if salida.Len() == 0 ||
		salida.Len() > maximoBytesCargaDecisionCoberturaO404E {
		return fallo(errSesionDecisionCoberturaO404EInvalida)
	}
	return salida.Bytes(), nil
}

func escribirDecisionVECDecisionCoberturaO404E(
	salida *bytes.Buffer,
	decision decisionVECDecisionCoberturaO404E,
) error {
	metadatos, err := json.Marshal(decision)
	if err != nil || len(metadatos) < 2 ||
		metadatos[0] != '{' || metadatos[len(metadatos)-1] != '}' {
		borrarBytes(metadatos)
		return errors.Join(errSesionDecisionCoberturaO404EInvalida, err)
	}
	defer borrarBytes(metadatos)
	salida.WriteByte('{')
	escribirNombreCampoDecisionCoberturaO404E(
		salida,
		"decision_canonica_hex",
		false,
	)
	escribirHexDecisionCoberturaO404E(salida, decision.DecisionCanonica)
	escribirNombreCampoDecisionCoberturaO404E(
		salida,
		"motivo_canonico_hex",
		true,
	)
	escribirHexDecisionCoberturaO404E(salida, decision.MotivoCanonico)
	if len(metadatos) > 2 {
		salida.WriteByte(',')
		salida.Write(metadatos[1 : len(metadatos)-1])
	}
	salida.WriteByte('}')
	return comprobarLimiteCargaDecisionCoberturaO404E(salida)
}

func escribirConsumoC1DecisionCoberturaO404E(
	salida *bytes.Buffer,
	consumo consumoC1DecisionCoberturaO404E,
) error {
	metadatos, err := json.Marshal(consumo)
	if err != nil || len(metadatos) < 2 ||
		metadatos[0] != '{' || metadatos[len(metadatos)-1] != '}' {
		borrarBytes(metadatos)
		return errors.Join(errSesionDecisionCoberturaO404EInvalida, err)
	}
	defer borrarBytes(metadatos)
	salida.WriteByte('{')
	if len(metadatos) > 2 {
		salida.Write(metadatos[1 : len(metadatos)-1])
		salida.WriteByte(',')
	}
	escribirNombreCampoDecisionCoberturaO404E(
		salida,
		"pruebas_canonicas",
		false,
	)
	escribirPruebasC1DecisionCoberturaO404E(salida, consumo.Pruebas)
	salida.WriteByte('}')
	return comprobarLimiteCargaDecisionCoberturaO404E(salida)
}

func escribirPruebasC1DecisionCoberturaO404E(
	salida *bytes.Buffer,
	p pruebasCanonicasC1DecisionCoberturaO404E,
) {
	salida.WriteByte('{')
	escribirCampoHexDecisionCoberturaO404E(
		salida, "peticion_hex", p.Peticion, false,
	)
	escribirCampoHexDecisionCoberturaO404E(
		salida, "resultado_hex", p.Resultado, true,
	)
	escribirCampoHexDecisionCoberturaO404E(
		salida, "atestacion_hex", p.Atestacion, true,
	)
	escribirCampoHexDecisionCoberturaO404E(
		salida, "confirmacion_tcb_hex", p.ConfirmacionTCB, true,
	)
	escribirCampoHexDecisionCoberturaO404E(
		salida, "catalogo_hex", p.Catalogo, true,
	)
	escribirCampoHexDecisionCoberturaO404E(
		salida, "verificador_hex", p.Verificador, true,
	)
	escribirCampoHexDecisionCoberturaO404E(
		salida, "resumen_hex", p.Resumen, true,
	)
	salida.WriteByte('}')
}

func escribirCampoJSONDecisionCoberturaO404E(
	salida *bytes.Buffer,
	nombre string,
	valor any,
	conComa bool,
) error {
	escribirNombreCampoDecisionCoberturaO404E(salida, nombre, conComa)
	contenido, err := json.Marshal(valor)
	if err != nil {
		borrarBytes(contenido)
		return err
	}
	salida.Write(contenido)
	borrarBytes(contenido)
	return comprobarLimiteCargaDecisionCoberturaO404E(salida)
}

func escribirCampoHexDecisionCoberturaO404E(
	salida *bytes.Buffer,
	nombre string,
	valor []byte,
	conComa bool,
) {
	escribirNombreCampoDecisionCoberturaO404E(salida, nombre, conComa)
	escribirHexDecisionCoberturaO404E(salida, valor)
}

func escribirNombreCampoDecisionCoberturaO404E(
	salida *bytes.Buffer,
	nombre string,
	conComa bool,
) {
	if conComa {
		salida.WriteByte(',')
	}
	salida.WriteByte('"')
	salida.WriteString(nombre)
	salida.WriteString(`":`)
}

func escribirHexDecisionCoberturaO404E(
	salida *bytes.Buffer,
	valor []byte,
) {
	salida.WriteByte('"')
	var temporal [4096]byte
	const bloqueOrigen = len(temporal) / 2
	for inicio := 0; inicio < len(valor); inicio += bloqueOrigen {
		fin := inicio + bloqueOrigen
		if fin > len(valor) {
			fin = len(valor)
		}
		n := hex.Encode(temporal[:], valor[inicio:fin])
		salida.Write(temporal[:n])
		clear(temporal[:n])
	}
	salida.WriteByte('"')
}

func comprobarLimiteCargaDecisionCoberturaO404E(
	salida *bytes.Buffer,
) error {
	if salida.Len() > maximoBytesCargaDecisionCoberturaO404E {
		return errSesionDecisionCoberturaO404EInvalida
	}
	return nil
}

func prevalidarCargaDecisionCoberturaO404E(
	carga cargaConfirmarDecisionCoberturaO404E,
) (int, error) {
	if len(carga.DecisionVEC.DecisionCanonica) == 0 ||
		len(carga.DecisionVEC.DecisionCanonica) >
			maximoBytesCanonVECDecisionCoberturaO404E ||
		len(carga.DecisionVEC.MotivoCanonico) == 0 ||
		len(carga.DecisionVEC.MotivoCanonico) >
			maximoBytesCanonVECDecisionCoberturaO404E ||
		len(carga.ConsumosC1) >
			int(cobertura.MaximoConsumosC1SesionTCBOperacionDecisionCobertura) {
		return 0, errSesionDecisionCoberturaO404EInvalida
	}
	total := len(carga.DecisionVEC.DecisionCanonica) +
		len(carga.DecisionVEC.MotivoCanonico)
	for i := range carga.ConsumosC1 {
		tamanio, err := tamanioPruebasC1DecisionCoberturaO404E(
			carga.ConsumosC1[i].Pruebas,
		)
		if err != nil ||
			total > maximoBytesMaterialCanonicoDecisionCoberturaO404E-tamanio {
			return 0, errSesionDecisionCoberturaO404EInvalida
		}
		total += tamanio
	}
	if total > maximoBytesMaterialCanonicoDecisionCoberturaO404E ||
		total*2 >
			maximoBytesCargaDecisionCoberturaO404E-
				reservaEstructuralCargaDecisionCoberturaO404E {
		return 0, errSesionDecisionCoberturaO404EInvalida
	}
	return total, nil
}

func tamanioPruebasC1DecisionCoberturaO404E(
	p pruebasCanonicasC1DecisionCoberturaO404E,
) (int, error) {
	total := 0
	for _, prueba := range [][]byte{
		p.Peticion, p.Resultado, p.Atestacion, p.ConfirmacionTCB,
		p.Catalogo, p.Verificador, p.Resumen,
	} {
		if len(prueba) == 0 ||
			len(prueba) > maximoBytesPruebaC1DecisionCoberturaO404E {
			return 0, errSesionDecisionCoberturaO404EInvalida
		}
		total += len(prueba)
	}
	return total, nil
}
