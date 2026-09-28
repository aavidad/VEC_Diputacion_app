package ports

import (
	"errors"
	"regexp"
)

var ErrSelectorCausaParticipacionInvalido = errors.New("bolsa: selector de causa de participacion invalido")

var patronCodigoCausaParticipacion = regexp.MustCompile(`^[a-z][a-z0-9_]{0,127}$`)
var patronHuellaCausaParticipacion = regexp.MustCompile(`^[a-f0-9]{64}$`)

// SelectorCausaParticipacion identifica una entrada inmutable del catálogo.
// La huella es la de esa entrada y versión, no la del catálogo completo.
type SelectorCausaParticipacion struct {
	Codigo       string `json:"codigo"`
	Version      int64  `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (s SelectorCausaParticipacion) Validar() error {
	if !patronCodigoCausaParticipacion.MatchString(s.Codigo) || s.Version < 1 || !patronHuellaCausaParticipacion.MatchString(s.HuellaSHA256) {
		return ErrSelectorCausaParticipacionInvalido
	}
	return nil
}

func (s SelectorCausaParticipacion) Igual(otro SelectorCausaParticipacion) bool {
	return s.Codigo == otro.Codigo && s.Version == otro.Version && s.HuellaSHA256 == otro.HuellaSHA256
}
