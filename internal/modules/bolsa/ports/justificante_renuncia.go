package ports

import (
	"context"
	"errors"

	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionSubirJustificanteRenunciaPropio     = "bolsa.justificante_renuncia.subir_propio"
	AccionDescargarJustificanteRenunciaPropio = "bolsa.justificante_renuncia.descargar_propio"
	AccionDescargarJustificanteRenunciaRRHH   = "bolsa.justificante_renuncia.descargar_rrhh"
	FinalidadJustificanteRenunciaBolsa        = "custodiar_justificante_renuncia_bolsa"
	TipoRecursoJustificanteRenuncia           = "justificante_renuncia"
)

var ErrJustificanteRenunciaNoDisponible = errors.New("bolsa: justificante de renuncia no disponible")
var ErrJustificanteRenunciaDenegado = errors.New("bolsa: justificante de renuncia denegado")

// La bolsa se selecciona en el servidor; identidad y vínculo llegan del
// contexto registrado y nunca de un campo del formulario.
type SolicitudJustificanteRenuncia struct {
	ActorRegistradoV2 vecdomain.ResultadoContextoActorRegistradoV2
	VinculoV2         vecdomain.VinculoAutenticacionActorV2
	BolsaRef          string
}

type ActoJustificanteRenuncia struct {
	RecursoRef       string `json:"recurso_ref"`
	PersonaRef       string `json:"persona_ref"`
	CandidatoRef     string `json:"candidato_ref"`
	BolsaRef         string `json:"-"`
	ParticipacionRef string `json:"participacion_ref"`
	LlamamientoRef   string `json:"llamamiento_ref"`
	VersionDueno     uint64 `json:"version_dueno"`
}

type ResolutorJustificanteRenuncia interface {
	ResolverJustificanteRenuncia(context.Context, SolicitudJustificanteRenuncia) (ActoJustificanteRenuncia, error)
}

// La autorización técnica previa usa la misma acción de subida y queda
// auditada antes de que Bolsa consulte participación y llamamiento.
type AutoridadPreparacionJustificanteRenuncia interface {
	PrepararJustificanteRenuncia(context.Context, vecdomain.ResultadoContextoActorRegistradoV2,
		vecdomain.VinculoAutenticacionActorV2, string) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}

type SolicitudConsultaJustificanteRenunciaRRHH struct {
	ActorRegistradoV2 vecdomain.ResultadoContextoActorRegistradoV2
	VinculoV2         vecdomain.VinculoAutenticacionActorV2
	BolsaRef          string
	ParticipacionRef  string
}

type JustificanteRenunciaRRHH struct {
	UnidadRef         string `json:"unidad_ref"`
	AmbitoRef         string `json:"ambito_ref"`
	DocumentoRef      string `json:"documento_ref"`
	DocumentoSHA256   string `json:"documento_sha256"`
	RecursoRef        string `json:"recurso_ref"`
	PersonaTitularRef string `json:"persona_titular_ref"`
	BolsaRef          string `json:"-"`
	ParticipacionRef  string `json:"-"`
}

type ResolutorConsultaJustificanteRenunciaRRHH interface {
	ResolverConsultaJustificanteRenunciaRRHH(context.Context, SolicitudConsultaJustificanteRenunciaRRHH) (JustificanteRenunciaRRHH, error)
}

type AutoridadPreparacionConsultaJustificanteRenunciaRRHH interface {
	PrepararConsultaJustificanteRenunciaRRHH(context.Context,
		vecdomain.ResultadoContextoActorRegistradoV2, vecdomain.VinculoAutenticacionActorV2,
		string, string) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)
}
