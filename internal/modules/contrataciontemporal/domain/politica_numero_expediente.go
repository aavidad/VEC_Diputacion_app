package domain

import "regexp"

// PoliticaNumeroExpediente valida el número externo asignado por MOAD. La
// envolvente técnica de NumeroVisible sigue protegiendo todas las proyecciones.
type PoliticaNumeroExpediente struct {
	Referencia string `json:"referencia"`
	Version    uint64 `json:"version"`
	Patron     string `json:"patron"`
	Ejemplo    string `json:"ejemplo"`
}

func (p PoliticaNumeroExpediente) Validar() error {
	if !ReferenciaOpacaValida(p.Referencia) || p.Version == 0 || len(p.Patron) < 2 || len(p.Patron) > 512 || p.Patron[0] != '^' || p.Patron[len(p.Patron)-1] != '$' || !NumeroExpedienteValido(p.Ejemplo) {
		return ErrDatoInvalido
	}
	r, err := regexp.Compile(p.Patron)
	if err != nil || !r.MatchString(p.Ejemplo) {
		return ErrDatoInvalido
	}
	return nil
}

func (p PoliticaNumeroExpediente) ValidarNumero(numero string) error {
	if p.Validar() != nil || !NumeroExpedienteValido(numero) {
		return ErrDatoInvalido
	}
	r, err := regexp.Compile(p.Patron)
	if err != nil || !r.MatchString(numero) {
		return ErrDatoInvalido
	}
	return nil
}
