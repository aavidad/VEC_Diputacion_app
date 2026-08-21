// Package ginpixfichero codifica cargas canónicas GINPIX sin realizar
// escritura, transporte, firma ni acuse.
package ginpixfichero

import (
	"encoding/json"
	"errors"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

const (
	EsquemaFicheroGINPIXV1        = "vec.dipgra.contratacion-temporal.ginpix.fichero.v1"
	VersionFormatoFicheroGINPIXV1 = uint64(1)
	MaximoBytesFicheroGINPIX      = 512 * 1024

	maximoCamposFicheroGINPIX    = 128
	reservaFijaFicheroGINPIX     = 8 * 1024
	reservaPorCampoFicheroGINPIX = 96
	expansionMaximaJSONGINPIX    = 6
)

var (
	ErrCargaGINPIXInvalida = errors.New(
		"contratacion temporal: carga ginpix para fichero invalida",
	)
	ErrLimiteFicheroGINPIXExcedido = errors.New(
		"contratacion temporal: limite de fichero ginpix excedido",
	)
	ErrCodificacionFicheroGINPIX = errors.New(
		"contratacion temporal: codificacion de fichero ginpix fallida",
	)
)

type ficheroGINPIX struct {
	Esquema   string                 `json:"esquema"`
	Version   uint64                 `json:"version"`
	Metadatos metadatosFicheroGINPIX `json:"metadatos"`
	Campos    []campoFicheroGINPIX   `json:"campos"`
}

type metadatosFicheroGINPIX struct {
	EsquemaModelo        string `json:"esquema_modelo"`
	EsquemaMapeo         string `json:"esquema_mapeo"`
	EsquemaCarga         string `json:"esquema_carga"`
	VersionExpediente    uint64 `json:"version_expediente"`
	ExpedienteRef        string `json:"expediente_ref"`
	IncorporacionRef     string `json:"incorporacion_ref"`
	ProcedenciaModeloRef string `json:"procedencia_modelo_ref"`
	CorrelacionRef       string `json:"correlacion_ref"`
	IdempotenciaRef      string `json:"idempotencia_ref"`
	HuellaModeloSHA256   string `json:"huella_modelo_sha256"`
	MapeoRef             string `json:"mapeo_ref"`
	MapeoVersion         uint64 `json:"mapeo_version"`
	ProcedenciaMapeoRef  string `json:"procedencia_mapeo_ref"`
	HuellaMapeoSHA256    string `json:"huella_mapeo_sha256"`
	HuellaCargaSHA256    string `json:"huella_carga_sha256"`
}

type campoFicheroGINPIX struct {
	Clave  string `json:"clave"`
	Estado string `json:"estado"`
	Valor  string `json:"valor"`
}

// Codificar transforma una carga mapeada válida en JSON determinista. El
// resultado es memoria propiedad del llamador y no activa ningún efecto.
func Codificar(carga domain.CargaMapeadaGINPIX) ([]byte, error) {
	if carga.Validar() != nil {
		return nil, ErrCargaGINPIXInvalida
	}
	datos := carga.Datos()
	presupuesto, err := presupuestoCodificacion(datos)
	if err != nil {
		return nil, err
	}
	campos := make([]campoFicheroGINPIX, len(datos.Campos))
	var claveAnterior domain.ClaveCatalogo
	for indice, campo := range datos.Campos {
		estado, valido := nombreEstadoCampo(campo.Campo.Estado)
		if !valido || !campo.Clave.Valida() ||
			(indice > 0 && campo.Clave <= claveAnterior) {
			return nil, ErrCargaGINPIXInvalida
		}
		claveAnterior = campo.Clave
		campos[indice] = campoFicheroGINPIX{
			Clave:  string(campo.Clave),
			Estado: estado,
			Valor:  campo.Campo.Valor,
		}
	}
	fichero := ficheroGINPIX{
		Esquema: EsquemaFicheroGINPIXV1,
		Version: VersionFormatoFicheroGINPIXV1,
		Metadatos: metadatosFicheroGINPIX{
			EsquemaModelo:        domain.EsquemaModeloCanonicoGINPIXV1,
			EsquemaMapeo:         domain.EsquemaMapeoGINPIXV1,
			EsquemaCarga:         datos.Esquema,
			VersionExpediente:    datos.VersionExpediente,
			ExpedienteRef:        datos.ExpedienteRef,
			IncorporacionRef:     datos.IncorporacionRef,
			ProcedenciaModeloRef: datos.ProcedenciaModeloRef,
			CorrelacionRef:       datos.CorrelacionRef,
			IdempotenciaRef:      datos.IdempotenciaRef,
			HuellaModeloSHA256:   datos.ModeloHuellaSHA256,
			MapeoRef:             datos.MapeoRef,
			MapeoVersion:         datos.MapeoVersion,
			ProcedenciaMapeoRef:  datos.ProcedenciaMapeoRef,
			HuellaMapeoSHA256:    datos.MapeoHuellaSHA256,
			HuellaCargaSHA256:    datos.HuellaSHA256,
		},
		Campos: campos,
	}
	contenido, err := json.Marshal(fichero)
	if err != nil {
		return nil, ErrCodificacionFicheroGINPIX
	}
	if len(contenido) > presupuesto || len(contenido) > MaximoBytesFicheroGINPIX {
		return nil, ErrLimiteFicheroGINPIXExcedido
	}
	return contenido, nil
}

// presupuestoCodificacion aplica el límite con una cota conservadora de
// escape JSON antes de construir el DTO o pedir al codificador su búfer.
func presupuestoCodificacion(datos domain.DatosCargaMapeadaGINPIX) (int, error) {
	if len(datos.Campos) == 0 || len(datos.Campos) > maximoCamposFicheroGINPIX {
		return 0, ErrLimiteFicheroGINPIXExcedido
	}
	restante := MaximoBytesFicheroGINPIX -
		reservaFijaFicheroGINPIX -
		len(datos.Campos)*reservaPorCampoFicheroGINPIX
	textos := []string{
		EsquemaFicheroGINPIXV1,
		domain.EsquemaModeloCanonicoGINPIXV1,
		domain.EsquemaMapeoGINPIXV1,
		datos.Esquema,
		datos.ExpedienteRef,
		datos.IncorporacionRef,
		datos.ProcedenciaModeloRef,
		datos.CorrelacionRef,
		datos.IdempotenciaRef,
		datos.ModeloHuellaSHA256,
		datos.MapeoRef,
		datos.ProcedenciaMapeoRef,
		datos.MapeoHuellaSHA256,
		datos.HuellaSHA256,
	}
	for _, campo := range datos.Campos {
		textos = append(textos, string(campo.Clave), campo.Campo.Valor)
	}
	for _, texto := range textos {
		if len(texto) > restante/expansionMaximaJSONGINPIX {
			return 0, ErrLimiteFicheroGINPIXExcedido
		}
		restante -= len(texto) * expansionMaximaJSONGINPIX
	}
	return MaximoBytesFicheroGINPIX - restante, nil
}

func nombreEstadoCampo(estado domain.EstadoCampoGINPIX) (string, bool) {
	switch estado {
	case domain.EstadoCampoGINPIXAusente:
		return "ausente", true
	case domain.EstadoCampoGINPIXNulo:
		return "nulo", true
	case domain.EstadoCampoGINPIXValor:
		return "valor", true
	default:
		return "", false
	}
}
