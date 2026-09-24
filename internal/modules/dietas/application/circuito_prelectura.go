package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	dietasports "vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionPreleerCircuito    = "dietas.circuito.preleer"
	FinalidadPreleerCircuito = "preleer_competencia_circuito_dietas"
)

type CasoUsoPrelecturaCircuito interface {
	Preleer(context.Context, dietasports.IdentidadEfectivaPrelecturaCircuito, dietasports.SolicitudPrelecturaCircuito) (dietasports.ContextoComisionCircuito, error)
}

type ServicioPrelecturaCircuito struct {
	repositorio dietasports.RepositorioPrelecturaCircuito
}

func NuevoServicioPrelecturaCircuito(r dietasports.RepositorioPrelecturaCircuito) (*ServicioPrelecturaCircuito, error) {
	if interfazNula(r) {
		return nil, ErrComposicionCircuitoInvalida
	}
	return &ServicioPrelecturaCircuito{repositorio: r}, nil
}

func ValidarSolicitudPrelecturaCircuito(s dietasports.SolicitudPrelecturaCircuito) error {
	if !referenciaComision.MatchString(s.Referencia) || s.Etapa.EstadoPendiente() == "" || !unidadCircuito.MatchString(s.UnidadRef) {
		return domain.ErrDecisionCircuitoInvalida
	}
	return nil
}

func (s *ServicioPrelecturaCircuito) Preleer(ctx context.Context, identidad dietasports.IdentidadEfectivaPrelecturaCircuito, solicitud dietasports.SolicitudPrelecturaCircuito) (dietasports.ContextoComisionCircuito, error) {
	var cero dietasports.ContextoComisionCircuito
	if s == nil || interfazNula(s.repositorio) || ctx == nil {
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return cero, err
	}
	if err := ValidarSolicitudPrelecturaCircuito(solicitud); err != nil {
		return cero, err
	}
	if identidad.UnidadCompetenciaRef != solicitud.UnidadRef || identidad.ContextoRegistrado.Validar() != nil || identidad.Vinculo.ValidarPara(identidad.ContextoRegistrado) != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	a := identidad.Autorizacion
	if a.Material.ValidarEstructura() != nil || a.Accion != AccionPreleerCircuito || a.RecursoRef != solicitud.Referencia || a.Finalidad != FinalidadPreleerCircuito ||
		a.Material.PersonaVersion() != identidad.ContextoRegistrado.Contexto.Instantanea.PersonaVersion || a.Material.PerfilVersion() != identidad.ContextoRegistrado.Contexto.Instantanea.PerfilVersion {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	resultado, err := s.repositorio.Preleer(ctx, identidad, solicitud)
	if err != nil {
		return cero, err
	}
	if resultado.Referencia != solicitud.Referencia || resultado.UnidadRef != solicitud.UnidadRef || resultado.Estado != solicitud.Etapa.EstadoPendiente() || resultado.Version == 0 ||
		!relacionCircuito.MatchString(resultado.RelacionRef) || !asignacionCircuito.MatchString(resultado.AsignacionRef) || resultado.AsignacionVersion == 0 || resultado.GrupoDieta == "" || resultado.CentroRef == "" ||
		!personaCircuito.MatchString(resultado.AdministrativoPersonaRef) || !personaCircuito.MatchString(resultado.ResponsablePersonaRef) {
		return cero, dietasports.ErrCircuitoNoDisponible
	}
	return resultado, nil
}

type materialPrelecturaCircuitoV1 struct {
	Esquema    string               `json:"esquema"`
	Operacion  string               `json:"operacion"`
	RecursoRef string               `json:"recurso_ref"`
	UnidadRef  string               `json:"unidad_ref"`
	Etapa      domain.EtapaCircuito `json:"etapa"`
	Identidad  identidadCircuitoV1  `json:"identidad"`
}

// ConstruirEfectoAutorizacionPrelecturaCircuito solo recibe la unidad ya
// acreditada por una frontera de competencia. No acepta datos HTTP libres.
func ConstruirEfectoAutorizacionPrelecturaCircuito(contexto vecdomain.ResultadoContextoActorRegistradoV2, unidadCompetenciaRef string, solicitud dietasports.SolicitudPrelecturaCircuito) (dietasports.EfectoAutorizacionCircuito, error) {
	var cero dietasports.EfectoAutorizacionCircuito
	if contexto.Validar() != nil || ValidarSolicitudPrelecturaCircuito(solicitud) != nil || unidadCompetenciaRef != solicitud.UnidadRef {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	actor := contexto.Contexto
	i := actor.Instantanea
	m := materialPrelecturaCircuitoV1{Esquema: dietasports.EsquemaEfectoPrelecturaCircuitoV1, Operacion: "preleer", RecursoRef: solicitud.Referencia, UnidadRef: solicitud.UnidadRef, Etapa: solicitud.Etapa,
		Identidad: identidadCircuitoV1{ActorRef: actor.Principal.ID, PerfilRef: actor.PerfilActivoRef, PersonaRef: actor.PersonaRef, ContextoActorRef: i.VinculoRef, ContextoVersion: i.VinculoVersion, CuentaRef: i.CuentaRef, CuentaVersion: i.CuentaVersion, PersonaVersion: i.PersonaVersion, PerfilVersion: i.PerfilVersion}}
	material, err := json.Marshal(m)
	if err != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	h := sha256.Sum256(material)
	atributos := map[string]string{"operacion": "preleer", "recurso_ref": solicitud.Referencia, "material_sha256": hex.EncodeToString(h[:]), "etapa": string(solicitud.Etapa),
		"contexto_actor_ref": i.VinculoRef, "contexto_version": strconv.FormatUint(i.VinculoVersion, 10), "cuenta_ref": i.CuentaRef, "cuenta_version": strconv.FormatUint(i.CuentaVersion, 10), "persona_version": strconv.FormatUint(i.PersonaVersion, 10), "perfil_version": strconv.FormatUint(i.PerfilVersion, 10)}
	recurso := vecdomain.RecursoAutorizable{Referencia: solicitud.Referencia, ModuloID: dietasports.ModuloDietas, Tipo: dietasports.TipoRecursoDocumentoDietas, Ambitos: map[string]string{"persona_ref": actor.PersonaRef, "unidad_ref": solicitud.UnidadRef}, Atributos: atributos}
	if recurso.Validar() != nil {
		return cero, dietasports.ErrAccesoCircuitoDenegado
	}
	return dietasports.EfectoAutorizacionCircuito{Material: material, Recurso: recurso}, nil
}

var _ CasoUsoPrelecturaCircuito = (*ServicioPrelecturaCircuito)(nil)
