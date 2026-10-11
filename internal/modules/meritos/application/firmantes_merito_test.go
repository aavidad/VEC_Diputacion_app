package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/meritos/datos"
	"vec-diputacion-granada/internal/modules/meritos/domain"
	"vec-diputacion-granada/internal/modules/meritos/ports"
	vec "vec-diputacion-granada/internal/vec/domain"
)

type autorizadorFirmantesPrueba struct {
	llamadas int
	ajeno    bool
}

func (a *autorizadorFirmantesPrueba) AutorizarPreparacionFirmantes(_ context.Context, q ports.SolicitudAutoridadFirmantes) (ports.AutoridadFirmantes, error) {
	a.llamadas++
	actor := persona
	if a.ajeno {
		actor = "persona:ajena"
	}
	return ports.AutoridadFirmantes{Accion: q.Accion, ActorRef: actor, PerfilRef: perfil,
		HechoRef: q.HechoRef, DecisionRef: q.DecisionRef, VersionRosterEsperada: q.VersionRosterEsperada,
		HuellaSolicitudSHA256: q.HuellaSolicitudSHA256, PermisoRef: "permiso:rrhh:firmantes",
		VigenteHasta: instante.Add(10 * time.Second)}, nil
}

type designacionesFirmantesPrueba struct {
	llamadas int
	err      error
}

type decisionesFirmantesPrueba struct {
	err     error
	alterar bool
}

func (f *decisionesFirmantesPrueba) ObtenerMaterialExactoAutorizado(_ context.Context, _ ports.AutoridadFirmantes, m domain.MaterialDecisionFirmantes) (domain.MaterialDecisionFirmantes, error) {
	if f.err != nil {
		return domain.MaterialDecisionFirmantes{}, f.err
	}
	if f.alterar {
		m.MotivoRef = "motivo:ajeno"
	}
	return m, nil
}

func (d *designacionesFirmantesPrueba) ComprobarDesignacionesExactas(_ context.Context, p domain.PlanFirmantesMerito) (ports.ComprobacionDesignaciones, error) {
	d.llamadas++
	if d.err != nil {
		return ports.ComprobacionDesignaciones{}, d.err
	}
	return ports.ComprobacionDesignaciones{HuellaRosterSHA256: p.HuellaRosterSHA256,
		FuenteRef: "fuente:designaciones", FuenteVersion: 1, CompetenciaRef: "competencia:bases:1",
		VigenteHasta: instante.Add(10 * time.Second)}, nil
}

type repositorioFirmantesPrueba struct {
	plan           domain.PlanFirmantesMerito
	recibo         string
	clave          string
	operaciones    map[string]ports.RecuperacionPlanFirmantes
	confirmaciones int
	lecturas       int
	recuperaciones int
}

func (r *repositorioFirmantesPrueba) RecuperarConfirmadaAutorizada(_ context.Context, a ports.AutoridadFirmantes, clave string) (ports.RecuperacionPlanFirmantes, bool, error) {
	r.recuperaciones++
	v, ok := r.operaciones[a.ActorRef+":"+clave]
	if ok {
		v.Resultado.Repetida = true
	}
	return v, ok, nil
}

func (r *repositorioFirmantesPrueba) ObtenerActualAutorizado(_ context.Context, _ ports.AutoridadFirmantes, ref string) (domain.PlanFirmantesMerito, error) {
	r.lecturas++
	if r.plan.Material.DecisionRef != ref {
		return domain.PlanFirmantesMerito{}, ports.ErrPlanFirmantesNoDisponible
	}
	return r.plan, nil
}

func (r *repositorioFirmantesPrueba) GuardarVersion(_ context.Context, o ports.OrdenGuardarPlanFirmantes) (ports.ResultadoGuardarPlanFirmantes, error) {
	if r.clave == o.ClaveIdempotencia {
		if !reflect.DeepEqual(r.plan, o.Plan) {
			return ports.ResultadoGuardarPlanFirmantes{}, ports.ErrPlanFirmantesConflicto
		}
		return ports.ResultadoGuardarPlanFirmantes{Plan: r.plan, ReciboRef: r.recibo, Repetida: true}, nil
	}
	if r.plan.VersionRoster != o.VersionRosterEsperada || o.VersionRosterEsperada > 0 && r.plan.HuellaRosterSHA256 != o.HuellaAnteriorSHA256 {
		return ports.ResultadoGuardarPlanFirmantes{}, ports.ErrPlanFirmantesConflicto
	}
	r.confirmaciones++
	r.plan, r.clave, r.recibo = o.Plan, o.ClaveIdempotencia, "recibo:firmantes:prueba"
	if r.operaciones == nil {
		r.operaciones = map[string]ports.RecuperacionPlanFirmantes{}
	}
	r.operaciones[o.Autoridad.ActorRef+":"+o.ClaveIdempotencia] = ports.RecuperacionPlanFirmantes{
		Accion: o.Autoridad.Accion, ActorRef: o.Autoridad.ActorRef, ClaveIdempotencia: o.ClaveIdempotencia,
		HuellaSolicitudSHA256: o.Autoridad.HuellaSolicitudSHA256, VersionRosterEsperada: o.VersionRosterEsperada,
		Resultado: ports.ResultadoGuardarPlanFirmantes{Plan: o.Plan, ReciboRef: r.recibo},
	}
	return ports.ResultadoGuardarPlanFirmantes{Plan: r.plan, ReciboRef: r.recibo}, nil
}

func solicitudFirmantesPrueba(t *testing.T) SolicitudFirmantesMerito {
	t.Helper()
	_, base, _, _, _ := escenario(t)
	evidencias := append([]vec.ReferenciaDocumento(nil), base.Hecho.Evidencias...)
	b, err := json.Marshal(evidencias)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(b)
	m := domain.MaterialDecisionFirmantes{HechoRef: base.Hecho.Referencia, HechoVersion: base.Hecho.Version,
		DecisionRef: "decision:merito:1", DecisionVersion: 1,
		Evidencias: evidencias, EvidenciasSHA256: hex.EncodeToString(h[:]), MotivoRef: "motivo:revision:1", PuntosMicropuntos: "0"}
	m.ContenidoSHA256, _ = domain.HuellaContenidoDecisionFirmantes(m)
	return SolicitudFirmantesMerito{Contexto: base.Contexto, Vinculo: base.Vinculo,
		Accion: ports.AccionPrepararPlanFirmantes, ClaveIdempotencia: "clave:firmantes:1",
		Material: m,
		Firmantes: []domain.FirmanteDesignado{{PersonaRef: "persona:revisor", RolClave: "tecnico_rrhh", OrganoRef: "organo:rrhh",
			DesignacionRef: "designacion:revisor", DesignacionVersion: 1, Orden: 1, Obligatorio: true}}}
}

func servicioFirmantesPrueba(t *testing.T) (*ServicioFirmantesMerito, *autorizadorFirmantesPrueba, *designacionesFirmantesPrueba, *repositorioFirmantesPrueba) {
	t.Helper()
	c, err := datos.CatalogoRolesFirmantesV1()
	if err != nil {
		t.Fatal(err)
	}
	a, d, r := &autorizadorFirmantesPrueba{}, &designacionesFirmantesPrueba{}, &repositorioFirmantesPrueba{}
	s, err := NuevoServicioFirmantesMerito(c, a, &decisionesFirmantesPrueba{}, d, r, relojPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	return s, a, d, r
}

func TestFirmantesPrepararYReplayConservaSnapshot(t *testing.T) {
	s, a, d, r := servicioFirmantesPrueba(t)
	q := solicitudFirmantesPrueba(t)
	primero, err := s.Preparar(context.Background(), q)
	if err != nil || primero.ReciboRef == "" || r.confirmaciones != 1 || a.llamadas != 1 || d.llamadas != 1 {
		t.Fatalf("preparacion sin autoridades: %+v, %v; auth=%d design=%d repo=%d", primero, err, a.llamadas, d.llamadas, r.confirmaciones)
	}
	segundo, err := s.Preparar(context.Background(), q)
	if err != nil || !segundo.Repetida || segundo.ReciboRef != primero.ReciboRef ||
		segundo.Plan.HuellaRosterSHA256 != primero.Plan.HuellaRosterSHA256 || r.confirmaciones != 1 || a.llamadas != 2 ||
		r.recuperaciones != 2 || d.llamadas != 1 {
		t.Fatalf("replay creo nueva version o evito autoridad: %+v, %v", segundo, err)
	}
}

func TestFirmantesDeniegaSinDesignacionOConActorAjeno(t *testing.T) {
	for _, caso := range []string{"designacion", "actor", "rol", "material"} {
		t.Run(caso, func(t *testing.T) {
			s, a, d, r := servicioFirmantesPrueba(t)
			q := solicitudFirmantesPrueba(t)
			switch caso {
			case "designacion":
				d.err = errors.New("designacion sin fuente")
			case "actor":
				a.ajeno = true
			case "rol":
				q.Firmantes[0].RolClave = "sindicato_aprueba"
			case "material":
				s.decisiones.(*decisionesFirmantesPrueba).alterar = true
			}
			out, err := s.Preparar(context.Background(), q)
			if err == nil || out.ReciboRef != "" || r.confirmaciones != 0 {
				t.Fatalf("autoridad ausente produjo roster: %+v, %v", out, err)
			}
		})
	}
}

func TestFirmantesCambioTrasFirmaReiniciaSinPuntosDeBolsa(t *testing.T) {
	s, _, _, r := servicioFirmantesPrueba(t)
	q := solicitudFirmantesPrueba(t)
	primero, err := s.Preparar(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	r.plan.Firmas = []domain.FirmaConstatada{{DesignacionRef: "designacion:revisor", PersonaRef: "persona:revisor",
		DocumentoRef: "documento:decision", Revision: 1, HuellaSHA256: strings.Repeat("c", 64), Sentido: "conformidad"}}
	q.Accion, q.VersionRosterEsperada, q.HuellaAnteriorSHA256 = ports.AccionCambiarPlanFirmantes, 1, primero.Plan.HuellaRosterSHA256
	q.ClaveIdempotencia = "clave:firmantes:2"
	q.Material.DecisionVersion, q.Material.MotivoRef = 2, "motivo:revision:2"
	q.Material.ContenidoSHA256, _ = domain.HuellaContenidoDecisionFirmantes(q.Material)
	segundo, err := s.Preparar(context.Background(), q)
	if err != nil || segundo.Plan.VersionRoster != 2 || len(segundo.Plan.Firmas) != 0 ||
		segundo.Plan.Material.MotivoRef != q.Material.MotivoRef || r.confirmaciones != 2 {
		t.Fatalf("firma antigua sobrevivio al cambio: %+v, %v", segundo, err)
	}
}

func TestFirmantesCambioConfirmadoRecuperaAntesDelCAS(t *testing.T) {
	s, a, _, r := servicioFirmantesPrueba(t)
	q := solicitudFirmantesPrueba(t)
	primero, err := s.Preparar(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	q.Accion, q.VersionRosterEsperada, q.HuellaAnteriorSHA256 = ports.AccionCambiarPlanFirmantes, 1, primero.Plan.HuellaRosterSHA256
	q.ClaveIdempotencia = "clave:firmantes:cambio"
	q.Material.DecisionVersion, q.Material.MotivoRef = 2, "motivo:revision:2"
	q.Material.ContenidoSHA256, _ = domain.HuellaContenidoDecisionFirmantes(q.Material)
	confirmado, err := s.Preparar(context.Background(), q)
	if err != nil || confirmado.Plan.VersionRoster != 2 || r.lecturas != 1 {
		t.Fatalf("cambio no confirmado: %+v %v", confirmado, err)
	}
	repetido, err := s.Preparar(context.Background(), q)
	if err != nil || !repetido.Repetida || repetido.ReciboRef != confirmado.ReciboRef ||
		repetido.Plan.HuellaRosterSHA256 != confirmado.Plan.HuellaRosterSHA256 || r.lecturas != 1 ||
		r.confirmaciones != 2 || a.llamadas != 3 {
		t.Fatalf("replay consulto version actual antes de recuperar: %+v %v", repetido, err)
	}
	q.Material.PuntosMicropuntos = "1000000"
	q.Material.ContenidoSHA256, _ = domain.HuellaContenidoDecisionFirmantes(q.Material)
	if out, err := s.Preparar(context.Background(), q); !errors.Is(err, ports.ErrPlanFirmantesConflicto) ||
		out.ReciboRef != "" || r.lecturas != 1 || r.confirmaciones != 2 {
		t.Fatalf("clave reutilizada con otra preimagen recupero datos: %+v %v", out, err)
	}
}

func TestFirmantesSustitucionConfirmadaRecuperaAntesDelCAS(t *testing.T) {
	s, _, _, r := servicioFirmantesPrueba(t)
	q := solicitudFirmantesPrueba(t)
	primero, err := s.Preparar(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	q.Accion, q.VersionRosterEsperada, q.HuellaAnteriorSHA256 = ports.AccionSustituirPlanFirmantes, 1, primero.Plan.HuellaRosterSHA256
	q.ClaveIdempotencia, q.Firmantes, q.SustituyeRef = "clave:firmantes:sustitucion", nil, "designacion:revisor"
	q.Sustituto = domain.FirmanteDesignado{PersonaRef: "persona:sustituto", RolClave: "tecnico_rrhh",
		OrganoRef: "organo:rrhh", DesignacionRef: "designacion:sustituto", DesignacionVersion: 1,
		Orden: 1, Obligatorio: true, SustituyeRef: q.SustituyeRef}
	confirmado, err := s.Preparar(context.Background(), q)
	if err != nil || confirmado.Plan.VersionRoster != 2 || r.lecturas != 1 {
		t.Fatalf("sustitucion no confirmada: %+v %v", confirmado, err)
	}
	repetido, err := s.Preparar(context.Background(), q)
	if err != nil || !repetido.Repetida || repetido.ReciboRef != confirmado.ReciboRef ||
		r.lecturas != 1 || r.confirmaciones != 2 {
		t.Fatalf("sustitucion repetida alcanzo CAS: %+v %v", repetido, err)
	}
	q.Sustituto.PersonaRef = "persona:otro"
	if out, err := s.Preparar(context.Background(), q); !errors.Is(err, ports.ErrPlanFirmantesConflicto) ||
		out.ReciboRef != "" || r.confirmaciones != 2 {
		t.Fatalf("sustituto distinto reutilizo clave: %+v %v", out, err)
	}
}
