package bootstrap

import "context"

// HuellasProvisionLecturasRRHHBolsa identifica la asignación publicada y el
// objetivo exacto calculado desde ella. No contiene identidad ni concesiones.
type HuellasProvisionLecturasRRHHBolsa struct {
	PreimagenSHA256 string `json:"preimagen_sha256"`
	ObjetivoSHA256  string `json:"objetivo_sha256"`
}

// PrepararHuellasProvisionLecturasRRHHBolsa es el punto privado de lectura
// para la futura CLI de provisión. Usa la política Bolsa compuesta y la
// asignación central publicada; no escribe, no emite una aprobación y no
// concede las tres acciones durante el arranque.
func (c *ComposicionSeguridadDesarrollo) PrepararHuellasProvisionLecturasRRHHBolsa(
	ctx context.Context,
) (HuellasProvisionLecturasRRHHBolsa, error) {
	if c == nil || c.politicaRRHHNominalBolsa == nil {
		return HuellasProvisionLecturasRRHHBolsa{}, ErrComposicionDesarrolloIncompleta
	}
	preimagen, objetivo, err := HuellasProvisionLecturasNominalesRRHHBolsa(ctx, c.politicaRRHHNominalBolsa)
	if err != nil {
		return HuellasProvisionLecturasRRHHBolsa{}, err
	}
	return HuellasProvisionLecturasRRHHBolsa{
		PreimagenSHA256: preimagen,
		ObjetivoSHA256:  objetivo,
	}, nil
}
