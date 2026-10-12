package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	httpct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	domct "vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	ct "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ctConfirmadaB2Prueba struct {
	*servicioCTFacadeB2Prueba
	origen     ct.OrigenIncorporacionPersonalB2
	encontrado bool
	errOrigen  error
	fuentes    *fuentesIncorporacionPersonalB2
}

func (s *ctConfirmadaB2Prueba) LeerContratoPlanNominal(ctx context.Context, org, exp string) (ct.ContratoPlanNominalB2, error) {
	if s.fuentes != nil {
		if _, err := s.fuentes.ResolverUnidadPlanNominalB2(ctx, org, exp); err != nil {
			return ct.ContratoPlanNominalB2{}, err
		}
	}
	return s.servicioCTFacadeB2Prueba.LeerContratoPlanNominal(ctx, org, exp)
}

func (s *ctConfirmadaB2Prueba) LeerOrigenIncorporacionB2(ctx context.Context, org, exp string) (ct.OrigenIncorporacionPersonalB2, bool, error) {
	if s.fuentes != nil {
		if _, err := s.fuentes.ResolverUnidadPlanNominalB2(ctx, org, exp); err != nil {
			return ct.OrigenIncorporacionPersonalB2{}, false, err
		}
	}
	s.lecturas++
	return s.origen, s.encontrado, s.errOrigen
}

func escenarioConfirmadaB2Prueba(t *testing.T) (*fachadaIncorporacionPersonalB2, *ctConfirmadaB2Prueba, *opcionesFacadeB2Prueba, *lectorDetalleIncorporacionB2Prueba) {
	t.Helper()
	m := materialPreparacionB2Prueba(t)
	clave := "44444444-4444-4444-8444-444444444444"
	c := ct.ContratoPlanNominalB2{Protocolo: ct.ProtocoloIncorporacionPersonalB2, PlanRef: "plan:prueba", PlanVersion: 1, PlanSHA256: strings.Repeat("b", 64), Material: m,
		Solicitud: ct.SolicitudPlanNominalB2{OrganizacionRef: m.OrganizacionRef, ClaveIdempotencia: clave}}
	o := ct.OrigenIncorporacionPersonalB2{Protocolo: ct.ProtocoloIncorporacionPersonalB2,
		Confirmacion: ct.ConfirmacionOrigenIncorporacionB2{OrganizacionRef: m.OrganizacionRef, UnidadCTRef: m.UnidadCTRef, ExpedienteRef: m.ExpedienteRef, PlanRef: c.PlanRef, PlanVersion: c.PlanVersion, PlanSHA256: c.PlanSHA256,
			Hechos: ct.HechosPersonalIncorporacionB2{ModoPersonal: "alta_empleado", PersonalPlanRef: "perplan:prueba", PersonalPlanVersion: 1,
				PersonalPlanReciboRef: "perplanrec:prueba", PersonalPlanSHA256: strings.Repeat("c", 64), EmpleadoRef: "empleado:prueba",
				AltaReciboRef: "perrec:relacion", RelacionRef: "relacion:prueba", RelacionVersion: 1, RelacionReciboRef: "perrec:relacion",
				OcupacionRef: "ocupacion:prueba", OcupacionVersion: 1, OcupacionReciboRef: "perrec:ocupacion",
				RPTConfirmacionRef: "rpt:confirmacion", RPTReciboRef: "rpt:recibo", SeguimientoRef: "perrec:ejecucion"}},
		ReciboRef: "recibo:incorporacion-personal-b2:prueba", AuditoriaRef: "auditoria:prueba", OutboxRef: "outbox:prueba", RegistradoEn: time.Date(2026, 10, 12, 16, 0, 0, 0, time.UTC)}
	ctFake := &ctConfirmadaB2Prueba{servicioCTFacadeB2Prueba: &servicioCTFacadeB2Prueba{contrato: c}, origen: o, encontrado: true}
	op := &opcionesFacadeB2Prueba{err: errors.New("RPT actual no admite el plan histórico")}
	instante := time.Date(2026, 10, 12, 15, 0, 0, 0, time.UTC)
	detalle := &lectorDetalleIncorporacionB2Prueba{detalle: ct.DetalleExpedienteRRHH{Resumen: ct.ResumenExpedienteRRHH{
		ExpedienteRef: m.ExpedienteRef, OrganizacionRef: m.OrganizacionRef, NumeroVisible: "2026/CT-0001", Version: 9,
		FlujoRef: "flujo:ct:prueba", FlujoVersion: 1, FlujoHuella: strings.Repeat("a", 64), FaseClave: domct.FaseNombramiento,
		EstadoClave: domct.EstadoCompletado, CentroRef: "centro:prueba", CategoriaRef: m.CategoriaRef, UnidadRef: m.UnidadCTRef,
		CreadoEn: instante, ActualizadoEn: instante.Add(time.Hour),
	}, Asignacion: &ct.AsignacionOperativaRRHH{UnidadRef: m.UnidadCTRef, AsignadaEn: instante}}}
	f := &fachadaIncorporacionPersonalB2{ct: ctFake, opciones: op, organizacionRef: m.OrganizacionRef,
		fuentes: &fuentesIncorporacionPersonalB2{organizacionRef: m.OrganizacionRef, detalle: detalle}}
	if detalle.detalle.Resumen.Validar() != nil {
		t.Fatal("fixture de detalle inválido")
	}
	return f, ctFake, op, detalle
}

func TestIncorporacionB2GETComparteSoloDetalleDeLaPeticion(t *testing.T) {
	f, repo, op, detalle := escenarioConfirmadaB2Prueba(t)
	repo.fuentes = f.fuentes
	ctx := context.WithValue(context.Background(), claveRutaPeticionIncorporacionB2{}, rutaPeticionIncorporacionB2{metodo: http.MethodGet, ruta: httpct.RutaPlanB2})
	if p, err := f.Consultar(ctx, repo.contrato.Material.ExpedienteRef); err != nil || p.Estado != "incorporacion_confirmada" || detalle.llamadas != 1 || repo.lecturas != 2 || op.lecturas != 0 {
		t.Fatalf("GET duplicó detalle o perdió recibo: %+v; err=%v detalle=%d ct=%d opciones=%d", p, err, detalle.llamadas, repo.lecturas, op.lecturas)
	}
	if _, err := f.Consultar(ctx, repo.contrato.Material.ExpedienteRef); err != nil || detalle.llamadas != 2 {
		t.Fatalf("la captura cruzó peticiones: err=%v detalle=%d", err, detalle.llamadas)
	}
}

func TestIncorporacionB2GETConfirmadaTrasCeseConservaReciboSinFuentesActuales(t *testing.T) {
	f, repo, opciones, detalle := escenarioConfirmadaB2Prueba(t)
	for range 2 {
		p, err := f.Consultar(context.Background(), repo.contrato.Material.ExpedienteRef)
		if err != nil || p.Estado != "incorporacion_confirmada" || p.VersionExpedienteActual != 9 || p.Recibo == nil || p.Recibo.ReciboRef != repo.origen.ReciboRef || p.Plan == nil || p.Plan.Intencion.VersionExpediente != 8 {
			t.Fatalf("GET histórico perdió recibo o versión actual: %+v; %v", p, err)
		}
		b, err := json.Marshal(p)
		if err != nil || strings.Contains(string(b), `:null`) {
			t.Fatalf("GET confirmó con opciones nulas: %s; %v", b, err)
		}
	}
	if opciones.lecturas != 0 || detalle.llamadas != 2 || repo.lecturas != 4 || repo.mutaciones != 0 {
		t.Fatalf("GET histórico releyó fuentes cambiantes o tuvo efecto: opciones=%d detalle=%d ct=%d escrituras=%d", opciones.lecturas, detalle.llamadas, repo.lecturas, repo.mutaciones)
	}
}

func TestIncorporacionB2GETConfirmadaRechazaCrucesYDenegacionActual(t *testing.T) {
	for nombre, alterar := range map[string]func(*fachadaIncorporacionPersonalB2, *ctConfirmadaB2Prueba, *lectorDetalleIncorporacionB2Prueba){
		"organizacion": func(_ *fachadaIncorporacionPersonalB2, s *ctConfirmadaB2Prueba, _ *lectorDetalleIncorporacionB2Prueba) {
			s.origen.Confirmacion.OrganizacionRef = "org:ajena"
		},
		"unidad": func(_ *fachadaIncorporacionPersonalB2, s *ctConfirmadaB2Prueba, _ *lectorDetalleIncorporacionB2Prueba) {
			s.origen.Confirmacion.UnidadCTRef = "unidad:ajena"
		},
		"plan": func(_ *fachadaIncorporacionPersonalB2, s *ctConfirmadaB2Prueba, _ *lectorDetalleIncorporacionB2Prueba) {
			s.origen.Confirmacion.PlanRef = "plan:ajeno"
		},
		"huella": func(_ *fachadaIncorporacionPersonalB2, s *ctConfirmadaB2Prueba, _ *lectorDetalleIncorporacionB2Prueba) {
			s.origen.Confirmacion.PlanSHA256 = strings.Repeat("c", 64)
		},
		"version": func(_ *fachadaIncorporacionPersonalB2, s *ctConfirmadaB2Prueba, _ *lectorDetalleIncorporacionB2Prueba) {
			s.origen.Confirmacion.PlanVersion = 2
		},
		"expediente": func(_ *fachadaIncorporacionPersonalB2, s *ctConfirmadaB2Prueba, _ *lectorDetalleIncorporacionB2Prueba) {
			s.origen.Confirmacion.ExpedienteRef = "expediente:ajeno"
		},
		"version_actual": func(_ *fachadaIncorporacionPersonalB2, _ *ctConfirmadaB2Prueba, d *lectorDetalleIncorporacionB2Prueba) {
			d.detalle.Resumen.Version = 7
		},
		"detalle_ajeno": func(_ *fachadaIncorporacionPersonalB2, _ *ctConfirmadaB2Prueba, d *lectorDetalleIncorporacionB2Prueba) {
			d.detalle.Resumen.OrganizacionRef = "org:ajena"
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			f, repo, op, detalle := escenarioConfirmadaB2Prueba(t)
			alterar(f, repo, detalle)
			if p, err := f.Consultar(context.Background(), repo.contrato.Material.ExpedienteRef); !errors.Is(err, httpct.ErrManejadorIncorporacionPersonalB2) || p.Recibo != nil || op.lecturas != 0 {
				t.Fatalf("cruce expuso recibo: %+v; %v", p, err)
			}
		})
	}
	f, repo, op, detalle := escenarioConfirmadaB2Prueba(t)
	repo.err = ct.ErrPlanNominalB2Denegado
	if p, err := f.Consultar(context.Background(), repo.contrato.Material.ExpedienteRef); !errors.Is(err, httpct.ErrDenegadaIncorporacionPersonalB2) || p.Recibo != nil || op.lecturas != 0 || detalle.llamadas != 0 {
		t.Fatalf("denegación actual omitida: %+v; %v", p, err)
	}
}

func TestIncorporacionB2GETSinOrigenAunExigeFuentesActuales(t *testing.T) {
	f, repo, op, _ := escenarioConfirmadaB2Prueba(t)
	repo.encontrado = false
	if _, err := f.Consultar(context.Background(), repo.contrato.Material.ExpedienteRef); err == nil || op.lecturas != 1 {
		t.Fatalf("sin origen se saltó la preparación prospectiva: %v; opciones=%d", err, op.lecturas)
	}
}
