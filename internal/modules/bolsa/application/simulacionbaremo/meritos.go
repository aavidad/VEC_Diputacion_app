package simulacionbaremo

import (
	"encoding/json"

	"vec-diputacion-granada/internal/modules/bolsa/domain/calculomeritos"
)

// ServicioMeritos conserva el contrato de solicitud común, con esquemas propios.
// SimularMeritos no resuelve autoridades ni escribe valoraciones oficiales.
type ServicioMeritos struct{}
type SimulacionMeritos struct {
	canonico        []byte
	huellaResultado string
}

func (s SimulacionMeritos) RepresentacionCanonica() []byte { return append([]byte(nil), s.canonico...) }
func (s SimulacionMeritos) HuellaResultadoSHA256() string  { return s.huellaResultado }
func (ServicioMeritos) SimularMeritos(s Solicitud) (SimulacionMeritos, error) {
	fallo := func(fase string, err error) (SimulacionMeritos, error) {
		return SimulacionMeritos{}, &Error{Fase: fase, causa: err}
	}
	c, err := calculomeritos.RestaurarConjunto(s.ConjuntoCanonico, s.HuellaConjuntoSHA256)
	if err != nil {
		return fallo("conjunto", err)
	}
	e, err := calculomeritos.RestaurarEntrada(s.EntradaCanonica, s.HuellaEntradaSHA256)
	if err != nil {
		return fallo("entrada", err)
	}
	r, err := calculomeritos.Calcular(c, e)
	if err != nil {
		return fallo("calculo", err)
	}
	canonico, err := r.RepresentacionCanonica()
	if err != nil {
		return fallo("resultado", err)
	}
	huella, err := r.HuellaSHA256()
	if err != nil {
		return fallo("resultado", err)
	}
	contenido, err := json.Marshal(struct {
		Esquema               string          `json:"esquema"`
		Alcance               string          `json:"alcance"`
		ConvocatoriaRef       string          `json:"convocatoria_ref"`
		HuellaResultadoSHA256 string          `json:"huella_resultado_sha256"`
		Resultado             json.RawMessage `json:"resultado"`
	}{"vec.bolsa.simulacion_meritos.v1", "simulacion", c.ConvocatoriaRef, huella, canonico})
	if err != nil {
		return fallo("resultado", err)
	}
	return SimulacionMeritos{canonico: contenido, huellaResultado: huella}, nil
}
