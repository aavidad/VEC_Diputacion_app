package domain

import "time"

// EvidenciaSesionAdministracionPerfiles transporta el par emitido por las
// autoridades de sesión y contexto. Es un dato interno: no procede de JSON
// HTTP ni se serializa como contexto V3. El consumidor V3 selecciona sus
// campos explícitos al preparar la autorización ligada.
type EvidenciaSesionAdministracionPerfiles struct {
	ResultadoContexto ResultadoContextoActorRegistradoV2 `json:"-"`
	Vinculo           VinculoAutenticacionActorV2        `json:"-"`
}

func (e EvidenciaSesionAdministracionPerfiles) ValidarPara(actor ContextoActor) error {
	if actor.Validar() != nil || e.Vinculo.ValidarPara(e.ResultadoContexto) != nil {
		return ErrVinculoAutenticacionActorV2Invalido
	}
	huella, err := actor.HuellaSHA256VinculadaV2()
	if err != nil || huella != e.ResultadoContexto.HuellaSHA256 {
		return ErrVinculoAutenticacionActorV2Invalido
	}
	return nil
}

// ValidarEn evita consumir evidencia caducada aunque su estructura siga intacta.
func (e EvidenciaSesionAdministracionPerfiles) ValidarEn(actor ContextoActor, instante time.Time) error {
	if e.ValidarPara(actor) != nil || !e.Vinculo.VigenteEn(instante, e.ResultadoContexto) {
		return ErrVinculoAutenticacionActorV2Invalido
	}
	return nil
}
