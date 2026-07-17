package calculoexperiencia

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"unicode/utf8"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

// RestaurarResultadoExperienciaV1 acepta exclusivamente los mismos bytes que
// produce RepresentacionCanonica. La restauracion estructural no autentica la
// ejecucion; el caso oficial debe exigir ademas huella y prueba confiables.
func RestaurarResultadoExperienciaV1(contenido []byte) (ResultadoExperienciaV1, error) {
	if len(contenido) == 0 || len(contenido) > maximoBytesResultadoV1 {
		return ResultadoExperienciaV1{},
			nuevoError("resultado.representacion_canonica", CodigoFueraDeLimites)
	}
	if !utf8.Valid(contenido) {
		return ResultadoExperienciaV1{},
			nuevoError("resultado.representacion_canonica", CodigoValorNoCanonico)
	}
	if err := comprobarJSONEntradaSinClavesDuplicadas(contenido); err != nil {
		return ResultadoExperienciaV1{},
			nuevoError("resultado.representacion_canonica", codigoError(err))
	}
	var material materialResultadoExperienciaV1
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&material); err != nil {
		return ResultadoExperienciaV1{}, clasificarErrorRestauracionResultadoV1(err)
	}
	var sobrante struct{}
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return ResultadoExperienciaV1{},
			nuevoError("resultado.representacion_canonica", CodigoValorNoCanonico)
	}
	if material.Esquema != esquemaResultadoExperienciaV1 {
		return ResultadoExperienciaV1{}, nuevoError("resultado.esquema", CodigoEsquemaIncompatible)
	}
	resultado, err := reconstruirResultadoExperienciaV1(material)
	if err != nil {
		return ResultadoExperienciaV1{}, err
	}
	canonico, err := resultado.RepresentacionCanonica()
	if err != nil || !bytes.Equal(canonico, contenido) {
		return ResultadoExperienciaV1{},
			nuevoError("resultado.representacion_canonica", CodigoValorNoCanonico)
	}
	return resultado, nil
}

func RestaurarResultadoExperienciaV1ConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (ResultadoExperienciaV1, error) {
	if !huellaSHA256Canonica(huellaEsperada) {
		return ResultadoExperienciaV1{},
			nuevoError("resultado.huella_esperada_sha256", CodigoValorNoCanonico)
	}
	resultado, err := RestaurarResultadoExperienciaV1(contenido)
	if err != nil {
		return ResultadoExperienciaV1{}, err
	}
	huellaReal, err := resultado.HuellaSHA256()
	if err != nil || subtle.ConstantTimeCompare(
		[]byte(huellaReal), []byte(huellaEsperada),
	) != 1 {
		return ResultadoExperienciaV1{},
			nuevoError("resultado.huella_esperada_sha256", CodigoHuellaNoCoincide)
	}
	return resultado, nil
}

func clasificarErrorRestauracionResultadoV1(err error) error {
	if errors.Is(err, ErrFueraDeLimites) {
		return nuevoError("resultado.representacion_canonica", CodigoFueraDeLimites)
	}
	return nuevoError("resultado.representacion_canonica", CodigoValorNoCanonico)
}

func reconstruirResultadoExperienciaV1(
	material materialResultadoExperienciaV1,
) (ResultadoExperienciaV1, error) {
	vinculos, err := reconstruirVinculosResultadoV1(material.Vinculos)
	if err != nil {
		return ResultadoExperienciaV1{}, err
	}
	seleccion, err := reconstruirSeleccionResultadoV1(material.Seleccion)
	if err != nil {
		return ResultadoExperienciaV1{}, err
	}
	resultado := ResultadoExperienciaV1{
		estado: material.Estado, fase: material.Fase, vinculos: vinculos,
		seleccion:    seleccion,
		intervalos:   make([]IntervaloAplicacionResultadoExperienciaV1, len(material.Intervalos)),
		aplicaciones: make([]AplicacionCalculadaResultadoExperienciaV1, len(material.Aplicaciones)),
		reglas:       make([]ResultadoReglaExperienciaV1, len(material.Reglas)),
		secciones:    make([]SubtotalSeccionResultadoExperienciaV1, len(material.Secciones)),
		bloqueos:     make([]BloqueoResultadoExperienciaV1, len(material.Bloqueos)),
	}
	if material.Total != nil {
		resultado.total = *material.Total
		resultado.tieneTotal = true
	}
	for indice, origen := range material.Intervalos {
		resultado.intervalos[indice], err = reconstruirIntervaloResultadoV1(origen)
		if err != nil {
			return ResultadoExperienciaV1{}, err
		}
	}
	for indice, origen := range material.Aplicaciones {
		resultado.aplicaciones[indice], err = reconstruirAplicacionResultadoV1(origen)
		if err != nil {
			return ResultadoExperienciaV1{}, err
		}
	}
	for indice, origen := range material.Reglas {
		resultado.reglas[indice], err = reconstruirReglaResultadoV1(origen)
		if err != nil {
			return ResultadoExperienciaV1{}, err
		}
	}
	for indice, origen := range material.Secciones {
		resultado.secciones[indice], err = reconstruirSeccionResultadoV1(origen)
		if err != nil {
			return ResultadoExperienciaV1{}, err
		}
	}
	for indice, origen := range material.Bloqueos {
		resultado.bloqueos[indice], err = reconstruirBloqueoResultadoV1(origen)
		if err != nil {
			return ResultadoExperienciaV1{}, err
		}
	}
	if err := resultado.Validar(); err != nil {
		return ResultadoExperienciaV1{}, err
	}
	return resultado, nil
}

func reconstruirVinculosResultadoV1(
	m materialVinculosResultadoV1,
) (VinculosResultadoExperienciaV1, error) {
	conjunto, err := reconstruirReferenciaResultadoV1(m.Conjunto)
	if err != nil {
		return VinculosResultadoExperienciaV1{}, err
	}
	instantanea, err := reconstruirReferenciaResultadoV1(m.Entrada.Instantanea)
	if err != nil {
		return VinculosResultadoExperienciaV1{}, err
	}
	resultado := VinculosResultadoExperienciaV1{
		motor: VinculoMotorResultadoExperienciaV1{
			contrato: m.Motor.Contrato, version: m.Motor.Version,
			huellaContratoSHA256: m.Motor.HuellaContratoSHA256,
		},
		plan: VinculoPlanResultadoExperienciaV1{
			esquema: m.Plan.Esquema, huellaSHA256: m.Plan.HuellaSHA256,
		},
		conjunto: conjunto,
		entrada: VinculoEntradaResultadoExperienciaV1{
			instantanea: instantanea, huellaContenidoSHA256: m.Entrada.HuellaContenidoSHA256,
		},
		fechaCorte: m.FechaCorte,
	}
	if err := validarVinculosResultadoV1(resultado); err != nil {
		return VinculosResultadoExperienciaV1{}, err
	}
	return resultado, nil
}

func reconstruirSeleccionResultadoV1(
	m materialSeleccionResultadoV1,
) (SeleccionResultadoExperienciaV1, error) {
	resultado := SeleccionResultadoExperienciaV1{
		aplicaciones:    make([]AplicacionSeleccionResultadoExperienciaV1, len(m.Aplicaciones)),
		descartes:       make([]DescarteSeleccionResultadoExperienciaV1, len(m.Descartes)),
		sinCoincidencia: make([]SinCoincidenciaResultadoExperienciaV1, len(m.SinCoincidencia)),
		evaluaciones:    m.Evaluaciones,
	}
	for indice, origen := range m.Aplicaciones {
		tramo, err := reconstruirReferenciaResultadoV1(origen.Tramo)
		if err != nil {
			return SeleccionResultadoExperienciaV1{}, err
		}
		resultado.aplicaciones[indice] = AplicacionSeleccionResultadoExperienciaV1{
			tramo: tramo, reglaClave: origen.Regla, grupoClave: origen.Grupo,
			seccionClave: origen.Seccion, prioridad: origen.Prioridad, razon: origen.Razon,
		}
	}
	for indice, origen := range m.Descartes {
		tramo, err := reconstruirReferenciaResultadoV1(origen.Tramo)
		if err != nil {
			return SeleccionResultadoExperienciaV1{}, err
		}
		resultado.descartes[indice] = DescarteSeleccionResultadoExperienciaV1{
			tramo: tramo, reglaClave: origen.Regla, grupoClave: origen.Grupo,
			reglaSeleccionada: origen.ReglaSeleccionada, razon: origen.Razon,
		}
	}
	for indice, origen := range m.SinCoincidencia {
		tramo, err := reconstruirReferenciaResultadoV1(origen.Tramo)
		if err != nil {
			return SeleccionResultadoExperienciaV1{}, err
		}
		resultado.sinCoincidencia[indice] = SinCoincidenciaResultadoExperienciaV1{
			tramo: tramo, razon: origen.Razon,
		}
	}
	if err := validarSeleccionResultadoV1(resultado); err != nil {
		return SeleccionResultadoExperienciaV1{}, err
	}
	return resultado, nil
}

func reconstruirReferenciaResultadoV1(m materialReferencia) (reglasbaremo.ReferenciaVersionada, error) {
	referencia, err := reglasbaremo.NuevaReferenciaVersionada(m.Referencia, m.Version, m.HuellaSHA256)
	if err != nil {
		return reglasbaremo.ReferenciaVersionada{},
			nuevoError("resultado.referencia", CodigoValorNoCanonico)
	}
	return referencia, nil
}
