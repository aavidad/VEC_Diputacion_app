package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strconv"

	"vec-diputacion-granada/internal/modules/dietas/domain"
	"vec-diputacion-granada/internal/modules/dietas/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
)

const (
	AccionRegistrarLiquidacionEconomica    = "dietas.liquidacion_economica.registrar"
	FinalidadRegistrarLiquidacionEconomica = "registrar_liquidacion_economica_dietas"
	AudienciaRegistrarLiquidacionEconomica = "vec_dietas.liquidacion_economica.registrar.v1"
	EsquemaMaterialLiquidacionEconomica    = "vec.dietas.liquidacion-economica.v1"
)

type ServicioLiquidacionEconomica struct {
	repositorio ports.RepositorioLiquidacionEconomica
	reloj       ports.RelojLiquidacionEconomica
}

func NuevoServicioLiquidacionEconomica(r ports.RepositorioLiquidacionEconomica, reloj ports.RelojLiquidacionEconomica) (*ServicioLiquidacionEconomica, error) {
	if interfazNula(r) || interfazNula(reloj) {
		return nil, ErrComposicionCircuitoInvalida
	}
	return &ServicioLiquidacionEconomica{repositorio: r, reloj: reloj}, nil
}

// RegistrarPreparada es el único caso de escritura. El llamante interno
// construye la preparación con documento y catálogo leídos del propietario;
// SQL consume V3 y vuelve a cotejarlos bajo bloqueo en la misma transacción.
func (s *ServicioLiquidacionEconomica) RegistrarPreparada(ctx context.Context, identidad ports.IdentidadEfectivaCircuito, solicitud ports.SolicitudLiquidacionEconomica) (*domain.LiquidacionEconomica, error) {
	if s == nil || interfazNula(s.repositorio) || interfazNula(s.reloj) || ctx == nil {
		return nil, ports.ErrCircuitoNoDisponible
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	efecto, err := ConstruirEfectoAutorizacionLiquidacionEconomica(identidad.ContextoRegistrado, identidad.UnidadCompetenciaRef, solicitud)
	if err != nil {
		return nil, err
	}
	d := solicitud.Decision
	if !identidadCircuitoValida(identidad, AccionRegistrarLiquidacionEconomica, d.Referencia, FinalidadRegistrarLiquidacionEconomica, d.UnidadRef) {
		return nil, ports.ErrAccesoCircuitoDenegado
	}
	huella, err := efecto.Recurso.HuellaContextoAutorizacionSHA256()
	resumen := identidad.Autorizacion.Material.ResumenCapacidad()
	ahora := s.reloj.AhoraUTC()
	if err != nil || ahora.IsZero() || !identidad.ContextoRegistrado.Contexto.Instantanea.VigenteEn(ahora) ||
		resumen.Operacion() != AccionRegistrarLiquidacionEconomica || resumen.AudienciaConsumo() != AudienciaRegistrarLiquidacionEconomica ||
		resumen.EfectoRef() != d.Referencia || resumen.EfectoHuellaSHA256() != huella ||
		ahora.Before(resumen.EmitidaEn()) || !ahora.Before(resumen.ExpiraEn()) {
		return nil, ports.ErrAccesoCircuitoDenegado
	}
	resultado, err := s.repositorio.Registrar(ctx, identidad, solicitud)
	if err != nil {
		return nil, err
	}
	l, err := domain.RecuperarLiquidacionEconomica(resultado)
	if err != nil || resultado.Constancia.DecisorPersonaRef != identidad.ContextoRegistrado.Contexto.PersonaRef ||
		resultado.Constancia.UnidadRef != d.UnidadRef || !reflect.DeepEqual(resultado.Economia, solicitud.Preparacion.Instantanea()) {
		return nil, ports.ErrCircuitoNoDisponible
	}
	return l, nil
}

type materialLiquidacionEconomica struct {
	Esquema           string                                 `json:"esquema"`
	Operacion         string                                 `json:"operacion"`
	RecursoRef        string                                 `json:"recurso_ref"`
	UnidadRef         string                                 `json:"unidad_ref"`
	Identidad         identidadCircuito                      `json:"identidad"`
	HuellaSemantica   string                                 `json:"huella_semantica"`
	Comando           ports.SolicitudDecisionCircuito        `json:"comando"`
	Economia          domain.InstantaneaLiquidacionPublicada `json:"economia"`
	PreimagenEconomia []byte                                 `json:"preimagen_economia"`
	PreimagenCatalogo []byte                                 `json:"preimagen_catalogo"`
	DocumentoCanonico []byte                                 `json:"documento_canonico"`
	SemanticaCanonica []byte                                 `json:"semantica_canonica"`
}

// ConstruirEfectoAutorizacionLiquidacionEconomica fija el material que la
// autoridad V3 atesta. La huella semántica no incluye versiones renovables
// de contexto/cuenta; sí actor, clave, versión y contenido económico exacto.
func ConstruirEfectoAutorizacionLiquidacionEconomica(contexto vecdomain.ResultadoContextoActorRegistradoV2, unidad string, solicitud ports.SolicitudLiquidacionEconomica) (ports.EfectoAutorizacionCircuito, error) {
	var cero ports.EfectoAutorizacionCircuito
	d := solicitud.Decision
	if contexto.Validar() != nil || unidad != d.UnidadRef || !unidadCircuito.MatchString(unidad) {
		return cero, ports.ErrAccesoCircuitoDenegado
	}
	if ValidarSolicitudDecisionCircuito(d) != nil || d.Etapa != domain.EtapaLiquidacion || d.Decision != domain.DecisionAprobar || solicitud.Preparacion == nil || solicitud.Preparacion.DecisionPendiente() {
		return cero, domain.ErrLiquidacionPublicada
	}
	p, err := domain.RecuperarPreparacionLiquidacionPublicada(solicitud.Preparacion.Instantanea())
	if err != nil {
		return cero, err
	}
	economia := p.Instantanea()
	if economia.ComisionRef != d.Referencia || economia.ComisionVersion < 1 || uint64(economia.ComisionVersion) != d.VersionEsperada {
		return cero, domain.ErrLiquidacionPublicada
	}
	actor := contexto.Contexto
	i := actor.Instantanea
	canon, err := json.Marshal(struct {
		Esquema    string                                 `json:"esquema"`
		PersonaRef string                                 `json:"persona_ref"`
		Comando    ports.SolicitudDecisionCircuito        `json:"comando"`
		Economia   domain.InstantaneaLiquidacionPublicada `json:"economia"`
	}{EsquemaMaterialLiquidacionEconomica, actor.PersonaRef, d, economia})
	if err != nil {
		return cero, domain.ErrLiquidacionPublicada
	}
	semantica := sha256.Sum256(canon)
	preimagenEconomia := economia
	preimagenEconomia.SnapshotSHA256 = ""
	bytesEconomia, err := json.Marshal(preimagenEconomia)
	if err != nil {
		return cero, domain.ErrLiquidacionPublicada
	}
	preimagenCatalogo := economia.Catalogo
	preimagenCatalogo.HuellaSHA256 = ""
	bytesCatalogo, err := json.Marshal(preimagenCatalogo)
	if err != nil {
		return cero, domain.ErrLiquidacionPublicada
	}
	bytesDocumento, err := json.Marshal(economia.Documento)
	if err != nil {
		return cero, domain.ErrLiquidacionPublicada
	}
	m := materialLiquidacionEconomica{Esquema: EsquemaMaterialLiquidacionEconomica, Operacion: AccionRegistrarLiquidacionEconomica,
		RecursoRef: d.Referencia, UnidadRef: unidad, Identidad: identidadCircuito{ActorRef: actor.Principal.ID, PerfilRef: actor.PerfilActivoRef,
			PersonaRef: actor.PersonaRef, ContextoActorRef: i.VinculoRef, ContextoVersion: i.VinculoVersion, CuentaRef: i.CuentaRef,
			CuentaVersion: i.CuentaVersion, PersonaVersion: i.PersonaVersion, PerfilVersion: i.PerfilVersion},
		HuellaSemantica: hex.EncodeToString(semantica[:]), Comando: d, Economia: economia,
		PreimagenEconomia: bytesEconomia, PreimagenCatalogo: bytesCatalogo,
		DocumentoCanonico: bytesDocumento, SemanticaCanonica: canon}
	material, err := json.Marshal(m)
	if err != nil {
		return cero, domain.ErrLiquidacionPublicada
	}
	h := sha256.Sum256(material)
	atributos := map[string]string{"operacion": AccionRegistrarLiquidacionEconomica, "recurso_ref": d.Referencia,
		"material_sha256": hex.EncodeToString(h[:]), "contexto_actor_ref": i.VinculoRef, "contexto_version": strconv.FormatUint(i.VinculoVersion, 10),
		"cuenta_ref": i.CuentaRef, "cuenta_version": strconv.FormatUint(i.CuentaVersion, 10), "persona_version": strconv.FormatUint(i.PersonaVersion, 10),
		"perfil_version": strconv.FormatUint(i.PerfilVersion, 10), "etapa": string(d.Etapa)}
	recurso := vecdomain.RecursoAutorizable{Referencia: d.Referencia, ModuloID: ports.ModuloDietas, Tipo: "liquidacion_economica",
		Ambitos: map[string]string{"persona_ref": actor.PersonaRef, "unidad_ref": unidad}, Atributos: atributos}
	if recurso.Validar() != nil {
		return cero, ports.ErrAccesoCircuitoDenegado
	}
	return ports.EfectoAutorizacionCircuito{Material: material, Recurso: recurso}, nil
}
