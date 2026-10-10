package domain

import "regexp"

var patronClaveVinculoEmisionBolsa = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var patronLlamamientoEmisionBolsa = regexp.MustCompile(`^llamamiento:[0-9a-f]{64}$`)

type DatosVinculoEmisionBolsa struct {
	OrganizacionRef   string
	ExpedienteRef     string
	VersionEsperada   uint64
	BolsaRef          string
	LlamamientoRef    string
	ReciboEmisionRef  string
	ClaveIdempotencia string
}

func (d DatosVinculoEmisionBolsa) Validar() error {
	if !ReferenciaOpacaValida(d.OrganizacionRef) || !ReferenciaOpacaValida(d.ExpedienteRef) ||
		!ReferenciaOpacaValida(d.BolsaRef) || !patronLlamamientoEmisionBolsa.MatchString(d.LlamamientoRef) ||
		d.ReciboEmisionRef != "recibo:"+d.LlamamientoRef ||
		d.VersionEsperada == 0 || d.VersionEsperada > 9_007_199_254_740_991 ||
		!patronClaveVinculoEmisionBolsa.MatchString(d.ClaveIdempotencia) ||
		d.ClaveIdempotencia == "00000000-0000-4000-8000-000000000000" {
		return ErrDatoInvalido
	}
	return nil
}

func LlamamientoEmisionBolsaValido(ref string) bool {
	return patronLlamamientoEmisionBolsa.MatchString(ref)
}
