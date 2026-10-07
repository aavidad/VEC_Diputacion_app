package ports

import (
	"context"
	"errors"
	"time"

	bolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	prep "vec-diputacion-granada/internal/modules/bolsa/domain/preparacionbases"
	vec "vec-diputacion-granada/internal/vec/domain"
	core "vec-diputacion-granada/internal/vec/ports"
)

const (
	AccionGuardarPreparacionBases   = "bolsa.preparacion_bases.guardar"
	AccionConsultarPreparacionBases = "bolsa.preparacion_bases.consultar"
	TipoRecursoPreparacionBases     = "preparacion_bases"
	FinalidadPreparacionBases       = "preparacion_bases"
)

var (
	ErrPreparacionBasesInvalida          = errors.New("bolsa.preparacion_bases.solicitud_invalida")
	ErrPreparacionBasesConflicto         = errors.New("bolsa.preparacion_bases.version_en_conflicto")
	ErrPreparacionBasesClaveReutilizada  = errors.New("bolsa.preparacion_bases.clave_reutilizada")
	ErrPreparacionBasesNoEncontrada      = errors.New("bolsa.preparacion_bases.no_encontrada")
	ErrPreparacionBasesNoDisponible      = errors.New("bolsa.preparacion_bases.no_disponible")
	ErrResultadoPreparacionBasesInvalido = errors.New("bolsa.preparacion_bases.resultado_no_confiable")
)

// Estos contratos internos requieren capacidad del nucleo; no son cuerpos
// JSON ni aceptan actor, perfil o permiso declarado por el cliente.
// Ambito se resuelve en la frontera confiable y el repositorio lo compara
// con el de la cadena guardada; no puede cambiar al corregir ni al recuperar.
type GuardarPreparacionBases struct {
	Ambito bolsa.AmbitoOrganizativoConvocatoria
	bloqueoSerializacionGobiernoConvocatoria
	Esperada       prep.Esperada
	Material       prep.Material
	ClaveOperacion string
	Autorizacion   core.EvidenciaUsoDecisionAutorizacion
	SolicitadaEn   time.Time
}

type ConsultarPreparacionBases struct {
	Ambito bolsa.AmbitoOrganizativoConvocatoria
	bloqueoSerializacionGobiernoConvocatoria
	Exacta       prep.Esperada
	Autorizacion core.EvidenciaUsoDecisionAutorizacion
	SolicitadaEn time.Time
}

func (s GuardarPreparacionBases) RecursoAutorizable() (vec.RecursoAutorizable, error) {
	h, err := prep.HuellaIntencion(s.Esperada, s.Material, s.Ambito)
	if err != nil || !prep.IdentificadorValido(s.ClaveOperacion) || len(s.ClaveOperacion) > 128 {
		return vec.RecursoAutorizable{}, ErrPreparacionBasesInvalida
	}
	return vec.RecursoAutorizable{Referencia: s.Esperada.PreparacionRef, ModuloID: "bolsa", Tipo: TipoRecursoPreparacionBases,
		Ambitos: ambitosPreparacion(s.Ambito), Atributos: map[string]string{"huella_intencion_sha256": h}}, nil
}

func (s GuardarPreparacionBases) Validar() error {
	r, err := s.RecursoAutorizable()
	if err != nil || validarAutorizacionPreparacionBases(s.Autorizacion, r, AccionGuardarPreparacionBases,
		[]string{"auditoria", "evento_outbox", "historia", "material_preparacion"}, s.SolicitadaEn) != nil {
		return ErrPreparacionBasesInvalida
	}
	return nil
}

func (s ConsultarPreparacionBases) RecursoAutorizable() (vec.RecursoAutorizable, error) {
	if s.Ambito.Validar() != nil || s.Exacta.Validar(false) != nil {
		return vec.RecursoAutorizable{}, ErrPreparacionBasesInvalida
	}
	return vec.RecursoAutorizable{Referencia: s.Exacta.PreparacionRef, ModuloID: "bolsa", Tipo: TipoRecursoPreparacionBases,
		Ambitos: ambitosPreparacion(s.Ambito), Atributos: map[string]string{"revision": numeroDecimalConvocatoria(s.Exacta.Revision), "huella_material_sha256": s.Exacta.HuellaMaterialSHA256}}, nil
}

func (s ConsultarPreparacionBases) Validar() error {
	r, err := s.RecursoAutorizable()
	if err != nil || validarAutorizacionPreparacionBases(s.Autorizacion, r, AccionConsultarPreparacionBases,
		[]string{"material_preparacion"}, s.SolicitadaEn) != nil {
		return ErrPreparacionBasesInvalida
	}
	return nil
}

func ambitosPreparacion(a bolsa.AmbitoOrganizativoConvocatoria) map[string]string {
	m := map[string]string{"organizacion_ref": a.OrganizacionRef()}
	if a.UnidadGestionRef() != "" {
		m["unidad_gestion_ref"] = a.UnidadGestionRef()
	}
	return m
}

// La evidencia funcional NO acredita consumo V3. El siguiente adaptador SQL
// requiere material V3 nominal/atestacion central registrada y consumo conjunto
// con auditoria; este corte no implementa ni simula esa infraestructura.
// Esta validacion no concede acceso. El adaptador debe releer PDP registrado,
// perfiles y ambitos autorizados y consumir la decision dentro de su transaccion.
func validarAutorizacionPreparacionBases(e core.EvidenciaUsoDecisionAutorizacion, r vec.RecursoAutorizable, accion string, campos []string, instante time.Time) error {
	d, err := e.Datos()
	h, errHuella := r.HuellaContextoAutorizacionSHA256()
	v, errVinculo := d.Decision.VinculoAutenticacionActor.Datos()
	interna := v.Superficie == vec.SuperficieAutenticacionInternaCorporativaV1 && !v.CuentaPrivilegiada
	privilegiada := v.Superficie == vec.SuperficieAutenticacionAdministracionPrivilegiadaV1 && v.CuentaPrivilegiada
	if err != nil || errHuella != nil || errVinculo != nil || !instanteGobiernoConvocatoriaCanonico(instante) || e.ValidarEn(instante) != nil ||
		instante.Before(d.VerificadaEn) || instante.Sub(d.VerificadaEn) > VentanaMaximaUsoAutorizacionConvocatoria ||
		d.Decision.Accion != accion || d.Decision.RecursoRef != r.Referencia || d.Decision.ModuloID != r.ModuloID ||
		d.Decision.TipoRecurso != r.Tipo || d.Decision.ContextoRecursoHuellaSHA256 != h || d.Decision.Finalidad != FinalidadPreparacionBases ||
		!mismosCamposConvocatoria(d.Decision.CamposPermitidos, campos) || len(d.Decision.Obligaciones) != 0 ||
		d.Decision.GarantiaMinima != vec.AuthAssuranceHigh || v.GarantiaObservada != vec.AuthAssuranceHigh ||
		v.MetodoObservado == vec.AuthMethodDemo || (!interna && !privilegiada) {
		return ErrPreparacionBasesInvalida
	}
	return nil
}

// ReciboPreparacionBases permanece identico en replay; el acceso actual
// tiene consumo y auditoria nuevos, separados de este efecto historico.
type ReciboPreparacionBases struct {
	ReciboRef             string
	HistoriaRef           string
	AuditoriaRef          string
	EventoRef             string
	HuellaIntencionSHA256 string
	ConfirmadaEn          time.Time
}

type ResultadoPreparacionBases struct {
	Version                  prep.Version
	Recibo                   ReciboPreparacionBases
	AutorizacionRef          string
	HuellaAutorizacionSHA256 string
	ConsumoAutorizacionRef   string
	AuditoriaAccesoRef       string
	AccedidaEn               time.Time
}

// RepositorioPreparacionBases es una barrera atomica: CAS revision+huella,
// clave estable por actor y huella de intencion, decision registrada vigente,
// version inmutable, historia, auditoria y outbox en un solo COMMIT.
// Busca replay ANTES del CAS: misma clave+intencion recupera version/recibo;
// clave reusada con otra intencion falla. Todo replay y lectura reautoriza y
// audita. Nunca devuelve material antes de confirmar su transaccion de acceso.
// Ningun adaptador puede escribir las tablas formales de GobiernoBorradores.
type RepositorioPreparacionBases interface {
	GuardarPreparacionBases(context.Context, GuardarPreparacionBases) (ResultadoPreparacionBases, error)
	ConsultarPreparacionBases(context.Context, ConsultarPreparacionBases) (ResultadoPreparacionBases, error)
}
