package ports

import (
	"encoding/json"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// ConsultaPersonalLlamamiento identifica una página leída, sin copiar su contenido.
type ConsultaPersonalLlamamiento struct {
	ReciboRef    string `json:"recibo_ref"`
	AuditoriaRef string `json:"auditoria_ref"`
}

// ConsultasPersonalLlamamiento conserva un valor inmutable y comparable, como
// los recibos que lo contienen. En transporte es una lista JSON, no texto libre.
// El valor vacío solo corresponde a recibos anteriores sin esta procedencia.
type ConsultasPersonalLlamamiento string

func NuevasConsultasPersonalLlamamiento(paginas []ConsultaPersonalLlamamiento) (ConsultasPersonalLlamamiento, error) {
	if len(paginas) == 0 || len(paginas) > 64 {
		return "", ErrRespuestaBolsaNoConfiable
	}
	for _, pagina := range paginas {
		if !domain.ReferenciaOpacaValida(pagina.ReciboRef) || !domain.ReferenciaOpacaValida(pagina.AuditoriaRef) {
			return "", ErrRespuestaBolsaNoConfiable
		}
	}
	canon, err := json.Marshal(paginas)
	if err != nil {
		return "", err
	}
	return ConsultasPersonalLlamamiento(canon), nil
}

func (c ConsultasPersonalLlamamiento) MarshalJSON() ([]byte, error) {
	var validada ConsultasPersonalLlamamiento
	if err := validada.UnmarshalJSON([]byte(c)); err != nil || validada != c {
		return nil, ErrRespuestaBolsaNoConfiable
	}
	return []byte(c), nil
}

func (c *ConsultasPersonalLlamamiento) UnmarshalJSON(contenido []byte) error {
	if c == nil {
		return ErrRespuestaBolsaNoConfiable
	}
	var paginas []ConsultaPersonalLlamamiento
	if err := decodificarArtefactoCerradoBolsa(contenido, &paginas); err != nil {
		return ErrRespuestaBolsaNoConfiable
	}
	valor, err := NuevasConsultasPersonalLlamamiento(paginas)
	if err != nil {
		return err
	}
	*c = valor
	return nil
}
