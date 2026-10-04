package application

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
	core "vec-diputacion-granada/internal/vec/ports"
)

const (
	AudienciaGuardarPreparacionBasesV3   = "vec_bolsa_convocatorias.preparacion_bases.guardar.v1"
	AudienciaConsultarPreparacionBasesV3 = "vec_bolsa_convocatorias.preparacion_bases.consultar.v1"
)

type PreparacionOperacionBasesV3 struct {
	Canonico  []byte
	Contenido []byte
	Recurso   vec.RecursoAutorizable
}

type actorPreparacionBasesV3 struct {
	ActorRef        string `json:"actor_ref"`
	ContextoRef     string `json:"contexto_actor_ref"`
	ContextoVersion uint64 `json:"contexto_version"`
	PersonaVersion  uint64 `json:"persona_version"`
	PerfilRef       string `json:"perfil_ref"`
	PerfilVersion   uint64 `json:"perfil_version"`
	CorrelacionRef  string `json:"correlacion_ref"`
}

func PrepararGuardadoPreparacionBasesV3(s ports.SolicitudGuardarPreparacionBasesV3) (PreparacionOperacionBasesV3, error) {
	var cero PreparacionOperacionBasesV3
	a, err := prepararActorBasesV3(s.Actor, s.Correlacion)
	if err != nil || s.Ambito.Validar() != nil || !prep.IdentificadorValido(s.ClaveOperacion) || len(s.ClaveOperacion) > 128 {
		return cero, ports.ErrPreparacionBasesInvalida
	}
	contenido, err := s.Material.RepresentacionCanonica()
	if err != nil {
		return cero, ports.ErrPreparacionBasesInvalida
	}
	h, err := s.Material.HuellaSHA256()
	if err != nil {
		return cero, ports.ErrPreparacionBasesInvalida
	}
	i, err := prep.HuellaIntencion(s.Esperada, s.Material, s.Ambito)
	if err != nil {
		return cero, ports.ErrPreparacionBasesInvalida
	}
	m := struct {
		Esquema         string `json:"esquema"`
		Referencia      string `json:"preparacion_ref"`
		Organizacion    string `json:"organizacion_ref"`
		Unidad          string `json:"unidad_gestion_ref"`
		Revision        int    `json:"revision_esperada"`
		HuellaEsperada  string `json:"huella_esperada_sha256"`
		HuellaMaterial  string `json:"huella_material_sha256"`
		HuellaIntencion string `json:"huella_intencion_sha256"`
		Clave           string `json:"clave_operacion"`
		actorPreparacionBasesV3
	}{"vec.bolsa.preparacion-bases.guardar.v3", s.Esperada.PreparacionRef, s.Ambito.OrganizacionRef(), s.Ambito.UnidadGestionRef(), s.Esperada.Revision, s.Esperada.HuellaMaterialSHA256, h, i, s.ClaveOperacion, a}
	p, err := prepararRecursoBasesV3(m, s.Esperada.PreparacionRef, s.Ambito)
	p.Contenido = contenido
	return p, err
}

func PrepararConsultaPreparacionBasesV3(s ports.SolicitudConsultarPreparacionBasesV3) (PreparacionOperacionBasesV3, error) {
	var cero PreparacionOperacionBasesV3
	a, err := prepararActorBasesV3(s.Actor, s.Correlacion)
	e := s.Selector.Exacta
	actual := s.Selector.Modo == "actual" && e.Validar(true) == nil && e.Revision == 0
	exacta := s.Selector.Modo == "exacta" && e.Validar(false) == nil
	if err != nil || s.Ambito.Validar() != nil || (!actual && !exacta) {
		return cero, ports.ErrPreparacionBasesInvalida
	}
	m := struct {
		Esquema        string `json:"esquema"`
		Referencia     string `json:"preparacion_ref"`
		Organizacion   string `json:"organizacion_ref"`
		Unidad         string `json:"unidad_gestion_ref"`
		Modo           string `json:"modo"`
		Revision       int    `json:"revision"`
		HuellaMaterial string `json:"huella_material_sha256"`
		actorPreparacionBasesV3
	}{"vec.bolsa.preparacion-bases.consultar.v3", e.PreparacionRef, s.Ambito.OrganizacionRef(), s.Ambito.UnidadGestionRef(), s.Selector.Modo, e.Revision, e.HuellaMaterialSHA256, a}
	return prepararRecursoBasesV3(m, e.PreparacionRef, s.Ambito)
}

func prepararActorBasesV3(a vec.ContextoActor, c vec.ReferenciaCorrelacionAutorizacionV2) (actorPreparacionBasesV3, error) {
	var cero actorPreparacionBasesV3
	if _, err := a.RepresentacionCanonicaVinculadaV2(); err != nil || c.Validar() != nil {
		return cero, ports.ErrPreparacionBasesInvalida
	}
	correlacion, err := c.ValorCanonico()
	if err != nil {
		return cero, ports.ErrPreparacionBasesInvalida
	}
	return actorPreparacionBasesV3{a.PersonaRef, a.Instantanea.VinculoRef, a.Instantanea.VinculoVersion, a.Instantanea.PersonaVersion, a.PerfilActivoRef, a.Instantanea.PerfilVersion, correlacion}, nil
}

func prepararRecursoBasesV3(m any, ref string, a bolsa.AmbitoOrganizativoConvocatoria) (PreparacionOperacionBasesV3, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return PreparacionOperacionBasesV3{}, ports.ErrPreparacionBasesInvalida
	}
	h := sha256.Sum256(b)
	ambitos := map[string]string{"organizacion_ref": a.OrganizacionRef()}
	if a.UnidadGestionRef() != "" {
		ambitos["unidad_gestion_ref"] = a.UnidadGestionRef()
	}
	r := vec.RecursoAutorizable{Referencia: ref, ModuloID: "bolsa", Tipo: ports.TipoRecursoPreparacionBases, Ambitos: ambitos, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
	if r.Validar() != nil {
		return PreparacionOperacionBasesV3{}, ports.ErrPreparacionBasesInvalida
	}
	return PreparacionOperacionBasesV3{Canonico: b, Recurso: r}, nil
}

func ValidarMaterialGuardarPreparacionBasesV3(o ports.OrdenGuardarPreparacionBasesV3) error {
	p, err := PrepararGuardadoPreparacionBasesV3(o.Solicitud)
	if err != nil {
		return err
	}
	return ValidarExportacionPreparacionBasesV3(o.Autorizacion, p, o.Solicitud.Actor, ports.AccionGuardarPreparacionBases, AudienciaGuardarPreparacionBasesV3)
}

func ValidarMaterialConsultarPreparacionBasesV3(o ports.OrdenConsultarPreparacionBasesV3) error {
	p, err := PrepararConsultaPreparacionBasesV3(o.Solicitud)
	if err != nil {
		return err
	}
	return ValidarExportacionPreparacionBasesV3(o.Autorizacion, p, o.Solicitud.Actor, ports.AccionConsultarPreparacionBases, AudienciaConsultarPreparacionBasesV3)
}

// Esta ligadura previa no sustituye la verificacion de V3 ni el consumo SQL.
func ValidarExportacionPreparacionBasesV3(e core.ExportacionMaterialConsumoAutorizacionAtestadaV3, p PreparacionOperacionBasesV3, a vec.ContextoActor, accion, audiencia string) error {
	x := e.ResumenCapacidad()
	h, err := p.Recurso.HuellaContextoAutorizacionSHA256()
	actor, errActor := a.RepresentacionCanonicaVinculadaV2()
	if err != nil || errActor != nil || e.ValidarEstructura() != nil || x.Operacion() != accion || x.AudienciaConsumo() != audiencia ||
		x.EfectoRef() != p.Recurso.Referencia || x.EfectoHuellaSHA256() != h ||
		e.PersonaVersion() != a.Instantanea.PersonaVersion || e.PerfilVersion() != a.Instantanea.PerfilVersion || !bytes.Equal(actor, e.ContextoActorCanonico()) ||
		!decisionPreparacionV3Valida(e.DecisionCanonica(), p.Recurso, accion) {
		return ports.ErrPreparacionBasesNoDisponible
	}
	return nil
}

// La proyeccion del recibo no concede lectura de los contenidos referenciados.
func CamposPreparacionBasesV3(accion string) []string {
	switch accion {
	case ports.AccionGuardarPreparacionBases:
		return []string{"auditoria", "evento_outbox", "historia", "material_preparacion", "recibo_preparacion"}
	case ports.AccionConsultarPreparacionBases:
		return []string{"material_preparacion", "recibo_preparacion"}
	default:
		return nil
	}
}

func decisionPreparacionV3Valida(b []byte, r vec.RecursoAutorizable, accion string) bool {
	var d struct {
		Accion       string            `json:"accion"`
		Modulo       string            `json:"modulo_id"`
		Tipo         string            `json:"tipo_recurso"`
		Finalidad    string            `json:"finalidad"`
		Referencia   string            `json:"recurso_ref"`
		Campos       []string          `json:"campos_permitidos"`
		Obligaciones []json.RawMessage `json:"obligaciones"`
		Vinculo      struct {
			Superficie string `json:"superficie"`
		} `json:"vinculo_autenticacion_actor"`
	}
	if json.Unmarshal(b, &d) != nil || d.Accion != accion || d.Modulo != "bolsa" || d.Tipo != ports.TipoRecursoPreparacionBases ||
		d.Finalidad != ports.FinalidadPreparacionBases || d.Referencia != r.Referencia || d.Vinculo.Superficie != "interna_corporativa" ||
		!reflect.DeepEqual(d.Campos, CamposPreparacionBasesV3(accion)) || d.Obligaciones == nil || len(d.Obligaciones) != 0 {
		return false
	}
	return true
}
