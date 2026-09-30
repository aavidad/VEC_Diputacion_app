package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// ContextoParticipacionBolsaExterna es una proyección aprobada en Bolsa de
// una participación externa. Identificar su población no concede permisos:
// cada actuación conserva la autorización V3 y su vínculo propio.
type ContextoParticipacionBolsaExterna struct {
	ParticipacionRef   string `json:"participacion_ref"`
	BolsaRef           string `json:"bolsa_ref"`
	CandidatoRef       string `json:"candidato_ref"`
	PersonaRef         string `json:"persona_ref"`
	PerfilRef          string `json:"perfil_ref"`
	ContextoActorRef   string `json:"contexto_actor_ref"`
	Estado             string `json:"estado"`
	VigenteDesde       string `json:"vigente_desde"`
	VigenteHasta       string `json:"vigente_hasta"`
	FuenteRef          string `json:"fuente_ref"`
	FuenteVersion      int64  `json:"fuente_version"`
	FuenteHuellaSHA256 string `json:"fuente_huella_sha256"`
}

func (ContextoParticipacionBolsaExterna) String() string   { return "[PARTICIPACION EXTERNA PRIVADA]" }
func (ContextoParticipacionBolsaExterna) GoString() string { return "[PARTICIPACION EXTERNA PRIVADA]" }

func preimagenParticipacionBolsaValida(version int64, huella string) bool {
	return version >= 0 && version < math.MaxInt64 &&
		((version == 0 && huella == "") || (version > 0 && huellaPreimagenMiBolsaPortalExterno.MatchString(huella)))
}

func (b ContextoParticipacionBolsaExterna) validaPara(s SnapshotContextoExterno) bool {
	return b.validar() && s.Poblacion == "candidato" && s.VinculoCandidato != nil &&
		b.CandidatoRef == s.VinculoCandidato.CandidatoRef && b.PersonaRef == s.Persona.Referencia &&
		b.PerfilRef == s.Perfil.Referencia && b.ContextoActorRef == s.Contexto.Referencia
}

// El formato común no clasifica una participación: cada herramienta conserva
// su fuente y sus funciones de gobierno propias.
func (b ContextoParticipacionBolsaExterna) validar() bool {
	if !referenciaExterna(b.CandidatoRef, "can_") || !referenciaExterna(b.PersonaRef, "per_") ||
		!referenciaExterna(b.PerfilRef, "prf_") || !referenciaExterna(b.ContextoActorRef, "vca_") ||
		!referenciaExterna(b.FuenteRef, "prc_") || b.FuenteVersion < 1 ||
		!huellaPreimagenMiBolsaPortalExterno.MatchString(b.FuenteHuellaSHA256) ||
		(b.Estado != "activo" && b.Estado != "revocado") {
		return false
	}
	for _, ref := range []string{b.ParticipacionRef, b.BolsaRef} {
		if strings.TrimSpace(ref) != ref || len(ref) == 0 || len(ref) > 512 || strings.ContainsAny(ref, "\x00\r\n\t") {
			return false
		}
	}
	desde, e1 := time.Parse("2006-01-02T15:04:05.000000Z", b.VigenteDesde)
	hasta, e2 := time.Parse("2006-01-02T15:04:05.000000Z", b.VigenteHasta)
	return e1 == nil && e2 == nil && desde.Format("2006-01-02T15:04:05.000000Z") == b.VigenteDesde &&
		hasta.Format("2006-01-02T15:04:05.000000Z") == b.VigenteHasta && hasta.After(desde)
}

func huellaParticipacionBolsaExterna(ctx context.Context, tx consultaSnapshotContextoExterno,
	p PlanProvisionCandidatoExterno,
) (string, error) {
	if ctx == nil || ctx.Err() != nil || tx == nil || p.bolsa == nil ||
		!preimagenParticipacionBolsaValida(p.preimagen.VersionBolsa, p.preimagen.HuellaBolsa) {
		return "", ErrProvisionCandidatoExterno
	}
	var version int64
	var previa string
	err := tx.QueryRow(ctx, `SELECT version,huella_sha256
	 FROM vec_bolsa_llamamientos.preimagen_contexto_participacion_externa_v1($1)`, p.bolsa.ParticipacionRef).Scan(&version, &previa)
	if errors.Is(err, pgx.ErrNoRows) {
		if p.preimagen.VersionBolsa != 0 || p.preimagen.HuellaBolsa != "" {
			return "", ErrProvisionCandidatoExterno
		}
	} else if err != nil || version != p.preimagen.VersionBolsa || previa != p.preimagen.HuellaBolsa {
		return "", ErrProvisionCandidatoExterno
	}
	contenido, err := json.Marshal(p.bolsa)
	if err != nil {
		return "", ErrProvisionCandidatoExterno
	}
	var huella string
	err = tx.QueryRow(ctx, `SELECT vec_bolsa_llamamientos.huella_contexto_participacion_externa_v1($1::jsonb,$2::bigint)`,
		string(contenido), p.preimagen.VersionBolsa+1).Scan(&huella)
	if err != nil || !huellaPreimagenMiBolsaPortalExterno.MatchString(huella) {
		return "", ErrProvisionCandidatoExterno
	}
	return huella, nil
}

func publicarParticipacionBolsaExterna(ctx context.Context, tx consultaSnapshotContextoExterno,
	p PlanProvisionCandidatoExterno,
) error {
	huella, err := huellaParticipacionBolsaExterna(ctx, tx, p)
	if err != nil || huella != p.huellaBolsa {
		return ErrProvisionCandidatoExterno
	}
	contenido, err := json.Marshal(p.bolsa)
	if err != nil {
		return ErrProvisionCandidatoExterno
	}
	var ref, confirmada string
	var version int64
	err = tx.QueryRow(ctx, `SELECT participacion_ref,version,huella_sha256
	 FROM vec_bolsa_llamamientos.publicar_contexto_participacion_externa_v1($1::jsonb,$2::bigint,$3,$4)`,
		string(contenido), p.preimagen.VersionBolsa, huellaNulaProvisionExterna(p.preimagen.HuellaBolsa), huella).Scan(&ref, &version, &confirmada)
	if err != nil || ref != p.bolsa.ParticipacionRef || version != p.preimagen.VersionBolsa+1 || confirmada != huella {
		return ErrProvisionCandidatoExterno
	}
	return nil
}
