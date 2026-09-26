package domain

import "errors"

// ErrPoliticaInformeTrasSubsanacionInvalida: el catálogo pide informe nuevo
// con un documento de firma que no es una clave del circuito.
var ErrPoliticaInformeTrasSubsanacionInvalida = errors.New(
	"contratacion temporal: politica de informe tras subsanacion invalida",
)

// PoliticaInformeTrasSubsanacion dice, según el catálogo (duda 5 de RRHH), si
// tras subsanar un reparo hace falta un informe jurídico nuevo antes de
// fiscalizar otra vez y qué documento del circuito de firma se vuelve a
// firmar por ello. La política cero es la conducta de siempre: se fiscaliza
// de nuevo con el mismo informe.
type PoliticaInformeTrasSubsanacion struct {
	ExigeInformeNuevo bool
	// DocumentoFirma es la clave del circuito de firma que abre una ronda
	// nueva con el informe nuevo; vacía, ningún documento se firma de nuevo.
	DocumentoFirma string
}

// Validar exige que el documento, si se declara, sea una clave del circuito y
// que solo se declare cuando se exige informe nuevo.
func (p PoliticaInformeTrasSubsanacion) Validar() error {
	if p.DocumentoFirma != "" && (!p.ExigeInformeNuevo || !ClaveDocumentoFirmaValida(p.DocumentoFirma)) {
		return ErrPoliticaInformeTrasSubsanacionInvalida
	}
	return nil
}
