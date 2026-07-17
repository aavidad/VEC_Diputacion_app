package calculoexperiencia

import (
	"bytes"
	"encoding/json"

	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
	"vec-diputacion-granada/internal/shared/baremacion"
)

type materialResultadoExperienciaV1 struct {
	Esquema      string                          `json:"esquema"`
	Estado       EstadoResultadoExperienciaV1    `json:"estado"`
	Fase         FaseResultadoExperienciaV1      `json:"fase"`
	Vinculos     materialVinculosResultadoV1     `json:"vinculos"`
	Seleccion    materialSeleccionResultadoV1    `json:"seleccion"`
	Intervalos   materialesIntervalosResultadoV1 `json:"intervalos"`
	Aplicaciones materialesCalculosResultadoV1   `json:"aplicaciones"`
	Reglas       materialesReglasResultadoV1     `json:"reglas"`
	Secciones    materialesSeccionesResultadoV1  `json:"secciones"`
	Total        *baremacion.Puntos              `json:"total,omitempty"`
	Bloqueos     materialesBloqueosResultadoV1   `json:"bloqueos"`
}

type materialVinculosResultadoV1 struct {
	Motor      materialMotorResultadoV1   `json:"motor"`
	Plan       materialPlanResultadoV1    `json:"plan"`
	Conjunto   materialReferencia         `json:"conjunto"`
	Entrada    materialEntradaResultadoV1 `json:"entrada"`
	FechaCorte baremacion.FechaCivil      `json:"fecha_corte"`
}

type materialMotorResultadoV1 struct {
	Contrato             string `json:"contrato"`
	Version              uint64 `json:"version"`
	HuellaContratoSHA256 string `json:"huella_contrato_sha256"`
}

type materialPlanResultadoV1 struct {
	Esquema      string `json:"esquema"`
	HuellaSHA256 string `json:"huella_sha256"`
}

type materialEntradaResultadoV1 struct {
	Instantanea           materialReferencia `json:"instantanea"`
	HuellaContenidoSHA256 string             `json:"huella_contenido_sha256"`
}

type materialSeleccionResultadoV1 struct {
	Aplicaciones    materialesSeleccionAplicacionesV1 `json:"aplicaciones"`
	Descartes       materialesSeleccionDescartesV1    `json:"descartes"`
	SinCoincidencia materialesSinCoincidenciaV1       `json:"sin_coincidencia"`
	Evaluaciones    uint64                            `json:"evaluaciones"`
}

type materialAplicacionSeleccionV1 struct {
	Tramo     materialReferencia                `json:"tramo"`
	Regla     string                            `json:"regla"`
	Grupo     string                            `json:"grupo"`
	Seccion   string                            `json:"seccion"`
	Prioridad uint32                            `json:"prioridad"`
	Razon     CodigoRazonResultadoExperienciaV1 `json:"razon"`
}

type materialDescarteSeleccionV1 struct {
	Tramo             materialReferencia                `json:"tramo"`
	Regla             string                            `json:"regla"`
	Grupo             string                            `json:"grupo"`
	ReglaSeleccionada string                            `json:"regla_seleccionada"`
	Razon             CodigoRazonResultadoExperienciaV1 `json:"razon"`
}

type materialSinCoincidenciaV1 struct {
	Tramo materialReferencia                `json:"tramo"`
	Razon CodigoRazonResultadoExperienciaV1 `json:"razon"`
}

type materialIntervaloAplicacionV1 struct {
	Tramo    materialReferencia                   `json:"tramo"`
	Regla    string                               `json:"regla"`
	Periodo  materialPeriodo                      `json:"periodo"`
	Extremo  reglasbaremo.TratamientoExtremoFinal `json:"extremo"`
	Efectivo *materialIntervaloEfectivoV1         `json:"efectivo,omitempty"`
	Razon    CodigoRazonResultadoExperienciaV1    `json:"razon,omitempty"`
}

type materialIntervaloEfectivoV1 struct {
	Desde          baremacion.FechaCivil `json:"desde"`
	HastaExclusivo baremacion.FechaCivil `json:"hasta_exclusivo"`
	Dias           uint64                `json:"dias"`
}

type materialAplicacionCalculadaV1 struct {
	Tramo      materialReferencia   `json:"tramo"`
	Regla      string               `json:"regla"`
	Jornada    materialJornadaV1    `json:"jornada"`
	Unidades   materialUnidadesV1   `json:"unidades"`
	Puntuacion materialPuntuacionV1 `json:"puntuacion"`
}

type materialJornadaV1 struct {
	Origen             baremacion.FraccionJornada        `json:"origen"`
	Modo               reglasbaremo.ModoJornada          `json:"modo"`
	Factor             string                            `json:"factor"`
	AtestacionPresente bool                              `json:"atestacion_presente"`
	AtestacionUsada    bool                              `json:"atestacion_usada"`
	Razon              CodigoRazonResultadoExperienciaV1 `json:"razon"`
}

type materialUnidadesV1 struct {
	Exactas   string                               `json:"exactas"`
	Aportadas string                               `json:"aportadas"`
	Resto     string                               `json:"resto"`
	Frontera  FronteraRestosResultadoExperienciaV1 `json:"frontera"`
}

type materialPuntuacionV1 struct {
	Bruto      string  `json:"bruto"`
	Redondeado *string `json:"redondeado,omitempty"`
}

type materialTopeV1 struct {
	Antes    string  `json:"antes"`
	Limite   *string `json:"limite,omitempty"`
	Despues  string  `json:"despues"`
	Aplicado bool    `json:"aplicado"`
}

type materialRedondeoV1 struct {
	Momento reglasbaremo.MomentoRedondeo `json:"momento"`
	Modo    baremacion.ModoRedondeo      `json:"modo"`
	Entrada string                       `json:"entrada"`
	Salida  string                       `json:"salida"`
}

type materialReglaResultadoV1 struct {
	Seccion            string             `json:"seccion"`
	Regla              string             `json:"regla"`
	UnidadesAgregadas  string             `json:"unidades_agregadas"`
	UnidadesTrasRestos string             `json:"unidades_tras_restos"`
	RestoRegla         string             `json:"resto_regla"`
	TopeUnidades       materialTopeV1     `json:"tope_unidades"`
	Coeficiente        baremacion.Puntos  `json:"coeficiente_micropuntos_por_unidad"`
	Bruto              string             `json:"bruto"`
	Redondeo           materialRedondeoV1 `json:"redondeo"`
	TopePuntos         materialTopeV1     `json:"tope_puntos"`
	PuntosFinales      string             `json:"puntos_finales"`
}

type materialSeccionResultadoV1 struct {
	Seccion       string            `json:"seccion"`
	AntesTope     string            `json:"antes_tope"`
	Tope          materialTopeV1    `json:"tope"`
	PuntosFinales baremacion.Puntos `json:"puntos_finales"`
}

type materialBloqueoResultadoV1 struct {
	Codigo         CodigoBloqueoResultadoExperienciaV1 `json:"codigo"`
	Tramos         materialesReferenciasBloqueoV1      `json:"tramos"`
	Reglas         materialesReglasBloqueoV1           `json:"reglas"`
	Grupo          string                              `json:"grupo,omitempty"`
	Seccion        string                              `json:"seccion,omitempty"`
	ClaveGobernada string                              `json:"clave_gobernada,omitempty"`
	ValorExacto    *string                             `json:"valor_exacto,omitempty"`
}

type materialesSeleccionAplicacionesV1 []materialAplicacionSeleccionV1
type materialesSeleccionDescartesV1 []materialDescarteSeleccionV1
type materialesSinCoincidenciaV1 []materialSinCoincidenciaV1
type materialesIntervalosResultadoV1 []materialIntervaloAplicacionV1
type materialesCalculosResultadoV1 []materialAplicacionCalculadaV1
type materialesReglasResultadoV1 []materialReglaResultadoV1
type materialesSeccionesResultadoV1 []materialSeccionResultadoV1
type materialesBloqueosResultadoV1 []materialBloqueoResultadoV1
type materialesReferenciasBloqueoV1 []materialReferencia
type materialesReglasBloqueoV1 []string

type elementoColeccionResultadoV1 interface {
	materialAplicacionSeleccionV1 | materialDescarteSeleccionV1 |
		materialSinCoincidenciaV1 | materialIntervaloAplicacionV1 |
		materialAplicacionCalculadaV1 | materialReglaResultadoV1 |
		materialSeccionResultadoV1 | materialBloqueoResultadoV1 |
		materialReferencia | string
}

func decodificarColeccionResultadoV1[T elementoColeccionResultadoV1](
	datos []byte,
	limite int,
	campo string,
) ([]T, error) {
	decodificador := json.NewDecoder(bytes.NewReader(datos))
	decodificador.DisallowUnknownFields()
	inicio, err := decodificador.Token()
	if err != nil || inicio != json.Delim('[') {
		return nil, nuevoError(campo, CodigoValorNoCanonico)
	}
	resultado := make([]T, 0)
	for decodificador.More() {
		if len(resultado) >= limite {
			return nil, nuevoError(campo, CodigoFueraDeLimites)
		}
		var elemento T
		if err := decodificador.Decode(&elemento); err != nil {
			return nil, nuevoError(campo, CodigoValorNoCanonico)
		}
		resultado = append(resultado, elemento)
	}
	fin, err := decodificador.Token()
	if err != nil || fin != json.Delim(']') || exigirFinJSON(decodificador) != nil {
		return nil, nuevoError(campo, CodigoValorNoCanonico)
	}
	return resultado, nil
}
