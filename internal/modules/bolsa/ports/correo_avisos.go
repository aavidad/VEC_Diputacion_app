package ports

import (
	"context"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

// B59: a qué correo va el aviso de un llamamiento. Si la persona candidata
// tiene un correo activo y verificado en «Mis correos» (Usuarios), se usa ese;
// si no, o si Usuarios no responde, el del alta en la bolsa, como siempre.
// La constancia sólo guarda la fuente, el motivo y, con «Mis correos», la
// referencia opaca del correo en Usuarios; nunca la dirección.
const (
	FuenteCorreoMisCorreos = "mis_correos"
	FuenteCorreoAltaBolsa  = "alta_bolsa"

	MotivoFuenteCorreoActivo           = "correo_activo"
	MotivoFuenteSinCorreoActivo        = "sin_correo_activo"
	MotivoFuenteSinPersonaVinculada    = "sin_persona_vinculada"
	MotivoFuenteMisCorreosNoDisponible = "mis_correos_no_disponible"
)

// FuenteCorreoContacto es la constancia de la fuente de un aviso enviado.
type FuenteCorreoContacto struct {
	Fuente    string `json:"fuente"`
	Motivo    string `json:"motivo"`
	CorreoRef string `json:"correo_ref,omitempty"`
}

// Valida admite sólo las combinaciones que PostgreSQL (B59) acepta.
func (f FuenteCorreoContacto) Valida() bool {
	switch f.Fuente {
	case FuenteCorreoMisCorreos:
		return f.Motivo == MotivoFuenteCorreoActivo && correoRefUsuariosValida(f.CorreoRef)
	case FuenteCorreoAltaBolsa:
		return f.CorreoRef == "" && (f.Motivo == MotivoFuenteSinCorreoActivo || f.Motivo == MotivoFuenteSinPersonaVinculada || f.Motivo == MotivoFuenteMisCorreosNoDisponible)
	}
	return false
}

func correoRefUsuariosValida(v string) bool {
	if len(v) != len("correo:")+32 || v[:len("correo:")] != "correo:" {
		return false
	}
	for _, r := range v[len("correo:"):] {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// SolicitudCorreoAvisoPersona lleva la identidad de quien emite (para la
// autorización de la lectura) y las referencias opacas del aviso.
type SolicitudCorreoAvisoPersona struct {
	Vinculo            dominiovec.VinculoAutenticacionActorV2
	ResultadoContexto  dominiovec.ResultadoContextoActorRegistradoV2
	MotivoAutorizacion dominiovec.ReferenciaEntradaCatalogo
	BolsaRef           string
	UnidadRef          string
	AmbitoRef          string
	LlamamientoRef     string
	CandidatoRef       string
}

// FuenteCorreoAvisoPersona consulta a Usuarios el correo activo. Si lo hay,
// llama a usar con la dirección (que no se conserva) y devuelve true y su
// referencia opaca. Sin correo activo devuelve false sin llamar a usar. Un
// error significa que no se pudo consultar y que usar no se ha llamado.
type FuenteCorreoAvisoPersona interface {
	ConCorreoAvisoPersona(context.Context, SolicitudCorreoAvisoPersona, func(string)) (bool, string, error)
}

// LectorCandidatoParticipacion devuelve la referencia de candidato de una
// participación de un llamamiento ya reservado de la bolsa (bolsa,
// llamamiento, participación), o "" si no está vinculada a ninguna persona.
type LectorCandidatoParticipacion interface {
	CandidatoParticipacionAvisos(context.Context, string, string, string) (string, error)
}
