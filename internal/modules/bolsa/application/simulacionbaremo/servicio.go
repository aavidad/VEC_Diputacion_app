// Package simulacionbaremo coordina el motor puro sin efectos administrativos.
package simulacionbaremo

import (
	"encoding/json"
	"fmt"

	"vec-diputacion-granada/internal/modules/bolsa/domain/calculoexperiencia"
	"vec-diputacion-granada/internal/modules/bolsa/domain/reglasbaremo"
)

// Solicitud fija los bytes y huellas de ambas instantáneas, sin buscar una
// versión vigente ni consultar datos personales o autoridades externas.
type Solicitud struct {
	ConjuntoCanonico     []byte
	HuellaConjuntoSHA256 string
	EntradaCanonica      []byte
	HuellaEntradaSHA256  string
}

// Simulador es el contrato común que pueden consumir CLI y otros adaptadores.
type Simulador interface {
	Simular(Solicitud) (Simulacion, error)
}

type Servicio struct{}

var _ Simulador = Servicio{}

// Simulacion conserva el resultado semántico y su envoltorio de simulación.
// Las huellas permiten reproducir el cálculo; no acreditan aprobación ni auditoría.
type Simulacion struct {
	resultado       calculoexperiencia.ResultadoExperienciaV1
	canonico        []byte
	huellaResultado string
}

func (s Simulacion) Resultado() calculoexperiencia.ResultadoExperienciaV1 { return s.resultado }
func (s Simulacion) RepresentacionCanonica() []byte                       { return append([]byte(nil), s.canonico...) }
func (s Simulacion) HuellaResultadoSHA256() string                        { return s.huellaResultado }

// Error identifica la fase técnica fallida y conserva el error de dominio.
// No contiene los bytes, las rutas ni los valores recibidos.
type Error struct {
	Fase  string
	causa error
}

func (e *Error) Error() string { return fmt.Sprintf("simulacion_baremo:%s", e.Fase) }
func (e *Error) Unwrap() error { return e.causa }

func fallo(fase string, causa error) (Simulacion, error) {
	return Simulacion{}, &Error{Fase: fase, causa: causa}
}

// Simular admite conjuntos en borrador. Un impedimento de negocio conserva
// un resultado bloqueado sin total; un error técnico nunca devuelve resultado.
func (Servicio) Simular(s Solicitud) (Simulacion, error) {
	conjunto, err := reglasbaremo.RestaurarConjuntoReglasBaremoConHuellaSHA256(s.ConjuntoCanonico, s.HuellaConjuntoSHA256)
	if err != nil {
		return fallo("conjunto", err)
	}
	plan, err := calculoexperiencia.Compilar(conjunto)
	if err != nil {
		return fallo("compilacion", err)
	}
	entrada, err := calculoexperiencia.RestaurarEntradaExperienciaConHuellaSHA256(s.EntradaCanonica, s.HuellaEntradaSHA256)
	if err != nil {
		return fallo("entrada", err)
	}
	resultado, err := calculoexperiencia.CalcularExperienciaV1(plan, entrada)
	if err != nil {
		return fallo("calculo", err)
	}
	canonico, err := resultado.RepresentacionCanonica()
	if err != nil {
		return fallo("resultado", err)
	}
	huella, err := resultado.HuellaSHA256()
	if err != nil {
		return fallo("resultado", err)
	}
	envoltorio, err := json.Marshal(struct {
		Esquema               string          `json:"esquema"`
		Alcance               string          `json:"alcance"`
		ConvocatoriaRef       string          `json:"convocatoria_ref"`
		HuellaResultadoSHA256 string          `json:"huella_resultado_sha256"`
		Resultado             json.RawMessage `json:"resultado"`
	}{"vec.bolsa.simulacion_experiencia.v1", "simulacion", conjunto.Identidad().ConvocatoriaRef(), huella, canonico})
	if err != nil {
		return fallo("resultado", err)
	}
	return Simulacion{resultado: resultado, canonico: envoltorio, huellaResultado: huella}, nil
}
