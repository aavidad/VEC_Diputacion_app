package ports

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io"

	"vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// La relación entre la renuncia, la bolsa y el titular se obtiene de Bolsa;
// Documentos la comprueba en su fachada nominal antes de entregar la postimagen.
type SolicitudOriginalSubidaLocalRRHH struct {
	ActorRegistradoV2 domain.ResultadoContextoActorRegistradoV2
	VinculoV2         domain.VinculoAutenticacionActorV2
	CargaID           string
	DocumentoID       string
	HuellaPDFSHA256   string
	RecursoRef        string
	PersonaTitularRef string
	BolsaRef          string
	ParticipacionRef  string
	CorrelacionRef    string
}

type ConsultaPersistenciaSubidaLocalRRHH struct {
	SolicitudOriginalSubidaLocalRRHH
	AutorizacionV3 AutorizacionActoSubidaLocalV3
}

func PreimagenConsultaSubidaLocalRRHH(q SolicitudOriginalSubidaLocalRRHH) ([]byte, error) {
	if q.ActorRegistradoV2.Validar() != nil || q.VinculoV2.ValidarPara(q.ActorRegistradoV2) != nil ||
		q.DocumentoID == "" || len(q.DocumentoID) > 256 || len(q.CargaID) > 256 ||
		len(q.HuellaPDFSHA256) != 64 || q.RecursoRef == "" || len(q.RecursoRef) > 256 ||
		q.PersonaTitularRef == "" || len(q.PersonaTitularRef) > 256 ||
		q.BolsaRef == "" || len(q.BolsaRef) > 256 || q.ParticipacionRef == "" || len(q.ParticipacionRef) > 256 ||
		!domain.ReferenciaCorrelacionAutorizacionV2Valida(q.CorrelacionRef) {
		return nil, ErrSubidaLocalDenegada
	}
	if _, err := hex.DecodeString(q.HuellaPDFSHA256); err != nil {
		return nil, ErrSubidaLocalDenegada
	}
	return json.Marshal(struct {
		Accion               string `json:"accion"`
		Fase                 string `json:"fase"`
		CargaID              string `json:"carga_id"`
		DocumentoID          string `json:"documento_id"`
		RecursoRef           string `json:"recurso_ref"`
		PersonaRef           string `json:"persona_ref"`
		ActorRef             string `json:"actor_ref"`
		PerfilActivoRef      string `json:"perfil_activo_ref"`
		BolsaRef             string `json:"bolsa_ref"`
		ParticipacionRef     string `json:"participacion_ref"`
		RegistroContextoRef  string `json:"registro_contexto_ref"`
		HuellaContextoSHA256 string `json:"huella_contexto_sha256"`
		CorrelacionRef       string `json:"correlacion_ref"`
	}{AccionBolsaJustificanteRenunciaDescargarRRHH, "abrir_rrhh", q.CargaID, q.DocumentoID,
		q.RecursoRef, q.PersonaTitularRef, q.ActorRegistradoV2.Contexto.PersonaRef,
		q.ActorRegistradoV2.Contexto.PerfilActivoRef, q.BolsaRef, q.ParticipacionRef,
		q.ActorRegistradoV2.RegistroContextoRef, q.ActorRegistradoV2.HuellaSHA256, q.CorrelacionRef})
}

type RegistroOriginalSubidaLocalRRHH interface {
	ConsultarAdmitidoRRHH(context.Context, ConsultaPersistenciaSubidaLocalRRHH) (InstantaneaSubidaLocal, error)
}

type AutoridadActoSubidaLocalRRHHV3 interface {
	AutorizarConsultaRRHH(context.Context, []byte, SolicitudOriginalSubidaLocalRRHH) (AutorizacionActoSubidaLocalV3, error)
	AutorizarAlmacenRRHH(context.Context, SolicitudOriginalSubidaLocalRRHH, InstantaneaSubidaLocal,
		vecports.VinculosOperacionAlmacen) (vecports.ContextoOperacionAlmacen, error)
}

type LectorOriginalSubidaLocalRRHH interface {
	AbrirOriginalRRHH(context.Context, SolicitudOriginalSubidaLocalRRHH) (io.ReadCloser, error)
}
