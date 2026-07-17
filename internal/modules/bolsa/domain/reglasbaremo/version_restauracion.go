package reglasbaremo

import (
	"bytes"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"time"
	"unicode/utf8"
)

// materialRestauracionVersionGobernada conserva el contenido como bytes para
// someterlo a su propio restaurador canonico antes de construir el gobierno.
// El resto reutiliza exactamente el contrato que emite RepresentacionCanonica.
type materialRestauracionVersionGobernada struct {
	Esquema             string                           `json:"esquema"`
	Contenido           fragmentoJSONGobierno            `json:"contenido"`
	ReferenciaContenido materialReferenciaGobiernoReglas `json:"referencia_contenido"`
	Revision            uint64                           `json:"revision"`
	Estado              EstadoGobiernoReglasBaremo       `json:"estado"`
	CreadaPor           string                           `json:"creada_por"`
	CreadaEn            time.Time                        `json:"creada_en"`
	MotivoCreacion      materialMotivoGobiernoReglas     `json:"motivo_creacion"`
	Publicacion         *materialRestauracionPublicacion `json:"publicacion,omitempty"`
	Activacion          *materialRestauracionActivacion  `json:"activacion,omitempty"`
	Terminal            *materialTerminalGobiernoReglas  `json:"terminal,omitempty"`
}

type materialRestauracionPublicacion struct {
	ActorRef   string                         `json:"actor_ref"`
	Motivo     materialMotivoGobiernoReglas   `json:"motivo"`
	Aprobacion materialRestauracionAprobacion `json:"aprobacion"`
	Instante   time.Time                      `json:"instante"`
}

type materialRestauracionAprobacion struct {
	Atestacion    materialReferenciaGobiernoReglas `json:"atestacion"`
	Vinculo       materialVinculoGobiernoReglas    `json:"vinculo"`
	Firma         materialReferenciaGobiernoReglas `json:"firma"`
	PoliticaFirma materialReferenciaGobiernoReglas `json:"politica_firma"`
	Firmantes     listaFirmantesGobierno           `json:"firmantes"`
	FirmadaEn     time.Time                        `json:"firmada_en"`
	VerificadaEn  time.Time                        `json:"verificada_en"`
	ValidaHasta   time.Time                        `json:"valida_hasta"`
}

type materialRestauracionActivacion struct {
	ActorRef     string                           `json:"actor_ref"`
	Motivo       materialMotivoGobiernoReglas     `json:"motivo"`
	Dependencias materialRestauracionDependencias `json:"dependencias"`
	Instante     time.Time                        `json:"instante"`
}

type materialRestauracionDependencias struct {
	Atestacion     materialReferenciaGobiernoReglas `json:"atestacion"`
	Vinculo        materialVinculoGobiernoReglas    `json:"vinculo"`
	Convocatoria   materialReferenciaGobiernoReglas `json:"convocatoria"`
	Bases          materialReferenciaGobiernoReglas `json:"bases"`
	Dependencias   listaReferenciasGobierno         `json:"dependencias"`
	VerificadorRef string                           `json:"verificador_ref"`
	VerificadaEn   time.Time                        `json:"verificada_en"`
	ValidaHasta    time.Time                        `json:"valida_hasta"`
}

// fragmentoJSONGobierno permite entregar el objeto contenido a su restaurador
// cerrado sin introducir representaciones abiertas en el modelo de dominio.
type fragmentoJSONGobierno []byte

func (f *fragmentoJSONGobierno) UnmarshalJSON(contenido []byte) error {
	*f = append((*f)[:0], contenido...)
	return nil
}

type listaFirmantesGobierno []string

func (l *listaFirmantesGobierno) UnmarshalJSON(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	inicio, err := decodificador.Token()
	if err != nil || inicio != json.Delim('[') {
		return errorCanonGobierno()
	}
	resultado := make([]string, 0, 2)
	for decodificador.More() {
		if len(resultado) >= maximoFirmantesAprobacionReglasBaremo {
			return ErrGobiernoEvidenciaInvalida
		}
		var firmante string
		if err := decodificador.Decode(&firmante); err != nil {
			return errorCanonGobierno()
		}
		resultado = append(resultado, firmante)
	}
	fin, err := decodificador.Token()
	if err != nil || fin != json.Delim(']') {
		return errorCanonGobierno()
	}
	*l = resultado
	return nil
}

type listaReferenciasGobierno []materialReferenciaGobiernoReglas

func (l *listaReferenciasGobierno) UnmarshalJSON(contenido []byte) error {
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	inicio, err := decodificador.Token()
	if err != nil || inicio != json.Delim('[') {
		return errorCanonGobierno()
	}
	resultado := make([]materialReferenciaGobiernoReglas, 0, 8)
	for decodificador.More() {
		if len(resultado) >= maximoDependenciasReglasBaremo {
			return ErrGobiernoEvidenciaInvalida
		}
		var referencia materialReferenciaGobiernoReglas
		if err := decodificador.Decode(&referencia); err != nil {
			return errorCanonGobierno()
		}
		resultado = append(resultado, referencia)
	}
	fin, err := decodificador.Token()
	if err != nil || fin != json.Delim(']') {
		return errorCanonGobierno()
	}
	*l = resultado
	return nil
}

// RestaurarVersionGobernadaReglasBaremo reconstruye exclusivamente la
// representacion canonica V1. Ademas de validar el JSON, reproduce el ciclo de
// gobierno mediante sus constructores y transiciones publicas.
func RestaurarVersionGobernadaReglasBaremo(
	contenido []byte,
) (VersionGobernadaReglasBaremo, error) {
	if len(contenido) == 0 || len(contenido) > maximoBytesGobiernoReglasBaremo {
		return VersionGobernadaReglasBaremo{}, nuevoError(
			"representacion_canonica_gobierno", CodigoFueraDeLimites,
		)
	}
	if !utf8.Valid(contenido) {
		return VersionGobernadaReglasBaremo{}, errorCanonGobierno()
	}

	material, err := decodificarMaterialVersionGobernada(contenido)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	if material.Esquema != esquemaGobiernoReglasBaremo {
		return VersionGobernadaReglasBaremo{}, nuevoError("esquema", CodigoEsquemaIncompatible)
	}
	if err := validarFormaMaterialVersionGobernada(material); err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	if err := validarVolumenEvidenciasGobierno(material); err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}

	version, err := reconstruirVersionGobernada(material)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	canonico, err := version.RepresentacionCanonica()
	if err != nil || !bytes.Equal(canonico, contenido) {
		return VersionGobernadaReglasBaremo{}, errorCanonGobierno()
	}
	return version, nil
}

// RestaurarVersionGobernadaReglasBaremoConHuellaSHA256 exige tambien la
// huella canonica esperada y la compara en tiempo constante.
func RestaurarVersionGobernadaReglasBaremoConHuellaSHA256(
	contenido []byte,
	huellaEsperada string,
) (VersionGobernadaReglasBaremo, error) {
	if !huellaSHA256Valida(huellaEsperada) {
		return VersionGobernadaReglasBaremo{}, nuevoError(
			"huella_esperada_sha256", CodigoValorNoCanonico,
		)
	}
	version, err := RestaurarVersionGobernadaReglasBaremo(contenido)
	if err != nil {
		return VersionGobernadaReglasBaremo{}, err
	}
	huellaReal, err := version.HuellaSHA256()
	if err != nil || subtle.ConstantTimeCompare(
		[]byte(huellaReal), []byte(huellaEsperada),
	) != 1 {
		return VersionGobernadaReglasBaremo{}, nuevoError(
			"huella_esperada_sha256", CodigoHuellaNoCoincide,
		)
	}
	return version, nil
}

func decodificarMaterialVersionGobernada(
	contenido []byte,
) (materialRestauracionVersionGobernada, error) {
	var material materialRestauracionVersionGobernada
	decodificador := json.NewDecoder(bytes.NewReader(contenido))
	decodificador.DisallowUnknownFields()
	if err := decodificador.Decode(&material); err != nil {
		if errors.Is(err, ErrGobiernoEvidenciaInvalida) {
			return materialRestauracionVersionGobernada{}, ErrGobiernoEvidenciaInvalida
		}
		return materialRestauracionVersionGobernada{}, errorCanonGobierno()
	}
	var sobrante struct{}
	if err := decodificador.Decode(&sobrante); !errors.Is(err, io.EOF) {
		return materialRestauracionVersionGobernada{}, errorCanonGobierno()
	}
	return material, nil
}

func errorCanonGobierno() error {
	return nuevoError("representacion_canonica_gobierno", CodigoValorNoCanonico)
}

func validarFormaMaterialVersionGobernada(
	material materialRestauracionVersionGobernada,
) error {
	if !material.Estado.Valido() {
		return ErrGobiernoEstadoInvalido
	}
	valida := false
	switch material.Estado {
	case EstadoReglasBaremoBorrador:
		valida = material.Revision == 1 && material.Publicacion == nil &&
			material.Activacion == nil && material.Terminal == nil
	case EstadoReglasBaremoPublicada:
		valida = material.Revision == 2 && material.Publicacion != nil &&
			material.Activacion == nil && material.Terminal == nil
	case EstadoReglasBaremoActiva:
		valida = material.Revision == 3 && material.Publicacion != nil &&
			material.Activacion != nil && material.Terminal == nil
	case EstadoReglasBaremoSustituida:
		valida = material.Revision == 4 && material.Publicacion != nil &&
			material.Activacion != nil && material.Terminal != nil &&
			material.Terminal.Accion == AccionSustituirReglasBaremo
	case EstadoReglasBaremoRetirada:
		valida = material.Revision == 4 && material.Publicacion != nil &&
			material.Activacion != nil && material.Terminal != nil &&
			material.Terminal.Accion == AccionRetirarReglasBaremo
	case EstadoReglasBaremoDescartada:
		valida = material.Revision == 2 && material.Publicacion == nil &&
			material.Activacion == nil && material.Terminal != nil &&
			material.Terminal.Accion == AccionDescartarReglasBaremo
	}
	if !valida {
		return ErrGobiernoInvarianteQuebrada
	}
	return nil
}

func validarVolumenEvidenciasGobierno(
	material materialRestauracionVersionGobernada,
) error {
	if material.Publicacion != nil {
		cantidad := len(material.Publicacion.Aprobacion.Firmantes)
		if cantidad == 0 || cantidad > maximoFirmantesAprobacionReglasBaremo {
			return ErrGobiernoEvidenciaInvalida
		}
	}
	if material.Activacion != nil {
		cantidad := len(material.Activacion.Dependencias.Dependencias)
		if cantidad == 0 || cantidad > maximoDependenciasReglasBaremo {
			return ErrGobiernoEvidenciaInvalida
		}
	}
	return nil
}
