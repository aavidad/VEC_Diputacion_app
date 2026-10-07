package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const AudienciaConsultaConvocatoriaV3 = "vec_bolsa_convocatorias.version.consultar.v1"

// PreparacionConsultaConvocatoria liga selector, contexto y correlación al
// recurso exacto. No contiene una concesión ni consulta fuentes persistentes.
type PreparacionConsultaConvocatoria struct {
	Canonico []byte
	Recurso  vecdomain.RecursoAutorizable
}

func PrepararConsultaConvocatoria(s ports.SolicitudConsultaConvocatoria) (PreparacionConsultaConvocatoria, error) {
	var cero PreparacionConsultaConvocatoria
	if s.Actor.Validar() != nil || s.Selector.Validar() != nil || s.Correlacion.Validar() != nil {
		return cero, ports.ErrConsultaConvocatoriaInvalida
	}
	material := struct {
		Esquema         string `json:"esquema"`
		ConvocatoriaID  string `json:"convocatoria_id"`
		Secuencia       int    `json:"secuencia"`
		ActorRef        string `json:"actor_ref"`
		ContextoRef     string `json:"contexto_actor_ref"`
		ContextoVersion uint64 `json:"contexto_version"`
		PersonaVersion  uint64 `json:"persona_version"`
		PerfilRef       string `json:"perfil_ref"`
		PerfilVersion   uint64 `json:"perfil_version"`
		CorrelacionRef  string `json:"correlacion_ref"`
	}{"vec.bolsa.convocatoria.consulta-version.v3", s.Selector.ID, s.Selector.Secuencia,
		s.Actor.PersonaRef, s.Actor.Instantanea.VinculoRef, s.Actor.Instantanea.VinculoVersion,
		s.Actor.Instantanea.PersonaVersion, s.Actor.PerfilActivoRef, s.Actor.Instantanea.PerfilVersion, correlacionConvocatoria(s)}
	canon, err := json.Marshal(material)
	if err != nil {
		return cero, ports.ErrConsultaConvocatoriaInvalida
	}
	h := sha256.Sum256(canon)
	recurso := vecdomain.RecursoAutorizable{Referencia: s.Selector.Referencia(), ModuloID: bolsaports.ModuloGobiernoConvocatorias, Tipo: bolsaports.TipoRecursoVersionConvocatoriaGobernada,
		Ambitos:   map[string]string{"convocatoria_id": s.Selector.ID, "secuencia": strconv.Itoa(s.Selector.Secuencia)},
		Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if recurso.Validar() != nil {
		return cero, ports.ErrConsultaConvocatoriaInvalida
	}
	return PreparacionConsultaConvocatoria{Canonico: canon, Recurso: recurso}, nil
}

// ValidarMaterialConsultaConvocatoria solo verifica la ligadura de la
// exportación nominal. La función SQL debe verificar de nuevo V3 y consumirlo.
func ValidarMaterialConsultaConvocatoria(o ports.OrdenConsultaConvocatoria) error {
	p, err := PrepararConsultaConvocatoria(o.Solicitud)
	if err != nil {
		return err
	}
	a := o.Autorizacion
	x := a.ResumenCapacidad()
	h, err := p.Recurso.HuellaContextoAutorizacionSHA256()
	actor, errActor := o.Solicitud.Actor.RepresentacionCanonicaVinculadaV2()
	if err != nil || errActor != nil || a.ValidarEstructura() != nil ||
		x.Operacion() != bolsaports.AccionConsultarVersionConvocatoria || x.AudienciaConsumo() != AudienciaConsultaConvocatoriaV3 ||
		x.EfectoRef() != p.Recurso.Referencia || x.EfectoHuellaSHA256() != h ||
		a.PersonaVersion() != o.Solicitud.Actor.Instantanea.PersonaVersion || a.PerfilVersion() != o.Solicitud.Actor.Instantanea.PerfilVersion ||
		!bytes.Equal(actor, a.ContextoActorCanonico()) {
		return ports.ErrConvocatoriaNoDisponible
	}
	return nil
}
