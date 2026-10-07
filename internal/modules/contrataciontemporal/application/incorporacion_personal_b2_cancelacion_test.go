package application

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

// Cada puerto devuelve éxito aunque cancele el contexto. El servicio debe
// detener la secuencia sin depender de que el puerto posterior respete ctx.
type puertosPlanB2Cancelacion struct {
	cancelar   context.CancelFunc
	cancelarEn string
	llamadas   []string
	contrato   ports.ContratoPlanNominalB2
	origen     ports.OrigenIncorporacionPersonalB2
}

func (p *puertosPlanB2Cancelacion) visitar(paso string) {
	p.llamadas = append(p.llamadas, paso)
	if p.cancelarEn == paso {
		p.cancelar()
	}
}
func (p *puertosPlanB2Cancelacion) ResolverPlanNominalB2(context.Context, ports.SolicitudPlanNominalB2, ports.ActorIncorporacionPersonalB2) (domain.PlanIncorporacionPersonalB2, error) {
	p.visitar("resolver_plan")
	return p.contrato.Material, nil
}
func (p *puertosPlanB2Cancelacion) ResolverUnidadPlanNominalB2(context.Context, string, string) (string, error) {
	p.visitar("resolver_unidad")
	return p.contrato.Material.UnidadCTRef, nil
}
func (p *puertosPlanB2Cancelacion) VerificarHechosPersonalB2(_ context.Context, _ ports.ContratoPlanNominalB2, h ports.HechosPersonalIncorporacionB2, _ ports.ActorIncorporacionPersonalB2) (ports.HechosPersonalIncorporacionB2, error) {
	p.visitar("verificar_hechos")
	return h, nil
}
func (p *puertosPlanB2Cancelacion) AutorizarPlanNominalB2(_ context.Context, accion string, _ []byte, _ ports.ActorIncorporacionPersonalB2) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.visitar(accion)
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}
func (p *puertosPlanB2Cancelacion) RegistrarPlanNominalB2(context.Context, ports.RegistroPlanNominalB2, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ContratoPlanNominalB2, error) {
	p.visitar("registrar_plan")
	return p.contrato, nil
}
func (p *puertosPlanB2Cancelacion) LeerContratoPlanNominal(context.Context, string, string, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, ...string) (ports.ContratoPlanNominalB2, error) {
	p.visitar("leer_contrato")
	return p.contrato, nil
}
func (p *puertosPlanB2Cancelacion) ConfirmarOrigenIncorporacionB2(_ context.Context, m ports.ConfirmacionOrigenIncorporacionB2, _ vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.OrigenIncorporacionPersonalB2, error) {
	p.visitar("confirmar_origen")
	o := p.origen
	o.Confirmacion = m
	return o, nil
}
func (p *puertosPlanB2Cancelacion) LeerOrigenIncorporacionB2(context.Context, string, string, vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, ...string) (ports.OrigenIncorporacionPersonalB2, bool, error) {
	p.visitar("leer_origen")
	return p.origen, true, nil
}

func materialPlanB2Cancelacion(t *testing.T) (ports.ContratoPlanNominalB2, ports.OrigenIncorporacionPersonalB2, ports.ActorIncorporacionPersonalB2) {
	t.Helper()
	h := strings.Repeat("a", 64)
	fuente := domain.FuentePlanPersonalB2{Ref: "fuente:original", Version: 1, SHA256: h}
	p := domain.PlanIncorporacionPersonalB2{
		OrganizacionRef: "org:uno", UnidadCTRef: "unidad:rrhh", ExpedienteRef: "expediente:uno", VersionExpediente: 8,
		AnalisisVersion: 2, AnalisisReciboRef: "recibo:analisis", AnalisisSHA256: h, PropuestaReciboRef: "recibo:propuesta",
		AceptacionRef: "aceptacion:uno", AceptacionReciboRef: "recibo:aceptacion",
		Bolsa: domain.SelectorBolsaPlanB2{UnidadRef: "unidad:uno", CategoriaRef: "categoria:uno", NecesidadRef: "necesidad:uno",
			AceptacionOperacionRef: "operacion:aceptacion", AceptacionRegistroSHA256: h, AperturaOperacionRef: "operacion:apertura",
			AperturaRegistroSHA256: h, LlamamientoRef: "llamamiento:uno", PropuestaRef: "propuesta:uno"},
		PersonaRef: "per_aaaaaaaaaaaaaaaaaaaaaaaa", PersonaVersion: 1, PersonaFuente: fuente, PersonaReciboBolsaRef: "recibo:bolsa",
		OrganismoRef: "organismo:uno", UnidadRef: "unidad:uno", FuenteOrganizacion: domain.FuenteSinVersionPlanB2{Ref: fuente.Ref, SHA256: h},
		FuentePlantilla: domain.InstrumentoPlanPersonalB2{Ref: "11111111-1111-4111-8111-111111111111", Revision: 1, FuenteRef: fuente.Ref, FuenteSHA256: h},
		FuenteRPT:       domain.InstrumentoPlanPersonalB2{Ref: "22222222-2222-4222-8222-222222222222", Revision: 1, FuenteRef: fuente.Ref, FuenteSHA256: h},
		VersionPlazaRef: "plantilla:uno", VersionPuestoRef: "rpt:uno", RevisionPlaza: 1, RevisionPuesto: 1,
		PuestoRef: "puesto:uno", PlazaRef: "plaza:uno", CatalogoRPTID: "catalogo:rpt", CatalogoRPTModulo: "personal",
		CatalogoRPTVersion: 1, CatalogoRPTSHA256: h, CategoriaRef: "categoria:uno", VinculoRevision: 1, VinculoReciboRef: "recibo:vinculo",
		Regimen: domain.EntradaPlanPersonalB2{Ref: "regimen:uno", Version: 1}, Modalidad: domain.EntradaPlanPersonalB2{Ref: "modalidad:uno", Version: 1},
		ClaseOcupacion: "temporal", Desde: "2026-09-30", MotivoClave: "incorporacion", DocumentoRef: "documento:uno", DocumentoSHA256: h,
		EjercicioSintetico: true,
	}
	sol := ports.SolicitudPlanNominalB2{OrganizacionRef: p.OrganizacionRef, ExpedienteRef: p.ExpedienteRef,
		VersionExpediente: p.VersionExpediente, PuestoRef: p.PuestoRef, PlazaRef: p.PlazaRef,
		VersionPlantillaRef: p.VersionPlazaRef, VersionRPTRef: p.VersionPuestoRef,
		Regimen: p.Regimen, Modalidad: p.Modalidad, ClaseOcupacion: p.ClaseOcupacion,
		Desde: p.Desde, MotivoClave: p.MotivoClave, DocumentoRef: p.DocumentoRef, DocumentoSHA256: p.DocumentoSHA256,
		ClaveIdempotencia: "33333333-3333-4333-8333-333333333333"}
	sha, err := domain.SHA256PlanPersonalB2(ports.RegistroPlanNominalB2{Solicitud: sol, Material: p})
	if err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	c := ports.ContratoPlanNominalB2{Protocolo: ports.ProtocoloIncorporacionPersonalB2, PlanRef: "plan:uno", PlanVersion: 1,
		PlanReciboRef: "recibo:plan", PlanSHA256: sha, IntencionRef: "intencion:uno", IntencionReciboRef: "recibo:intencion",
		IntencionVersion: 1, SolicitudPersonalRef: "solicitud:personal", IdempotenciaPersonalUUID: sol.ClaveIdempotencia,
		Solicitud: sol, Material: p, RegistradoEn: ahora}
	if !contratoPlanB2Valido(c) {
		t.Fatal("contrato de prueba inválido")
	}
	o := ports.OrigenIncorporacionPersonalB2{Protocolo: ports.ProtocoloIncorporacionPersonalB2,
		Confirmacion: ports.ConfirmacionOrigenIncorporacionB2{OrganizacionRef: p.OrganizacionRef, ExpedienteRef: p.ExpedienteRef,
			UnidadCTRef: p.UnidadCTRef, PlanRef: c.PlanRef, PlanVersion: c.PlanVersion, PlanSHA256: c.PlanSHA256},
		ReciboRef: "recibo:origen", AuditoriaRef: "auditoria:origen", OutboxRef: "outbox:origen", RegistradoEn: ahora}
	cuenta := vd.CuentaAutenticadaContextoActor{CuentaRef: "cta_0123456789abcdefghijkl", Metodo: vd.AuthMethodCertificate, Garantia: vd.AuthAssuranceHigh}
	actor, err := vd.NuevoContextoActor(cuenta, vd.InstantaneaContextoActor{
		VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: cuenta.CuentaRef, CuentaVersion: 1,
		PersonaRef: "per_0123456789abcdefghijkl", PersonaVersion: 1, PerfilActivoRef: "prf_0123456789abcdefghijkl", PerfilVersion: 1,
		Estado: vd.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour),
	}, ahora)
	if err != nil {
		t.Fatal(err)
	}
	return c, o, actor
}

func TestPlanPersonalB2CancelacionDetienePuertoPosterior(t *testing.T) {
	contrato, origen, actor := materialPlanB2Cancelacion(t)
	operaciones := []struct {
		nombre   string
		pasos    []string
		ejecutar func(context.Context, *ServicioPlanNominalB2) error
	}{
		{"registrar", []string{"resolver_plan", ports.AccionRegistrarPlanNominalB2, "registrar_plan"}, func(ctx context.Context, s *ServicioPlanNominalB2) error {
			_, err := s.RegistrarPlanNominalB2(ctx, contrato.Solicitud, actor)
			return err
		}},
		{"leer_contrato", []string{"resolver_unidad", ports.AccionLeerPlanNominalB2, "leer_contrato"}, func(ctx context.Context, s *ServicioPlanNominalB2) error {
			_, err := s.LeerContratoPlanNominal(ctx, contrato.Material.OrganizacionRef, contrato.Material.ExpedienteRef)
			return err
		}},
		{"confirmar_origen", []string{"resolver_unidad", ports.AccionLeerPlanNominalB2, "leer_contrato", "verificar_hechos", ports.AccionConfirmarOrigenB2, "confirmar_origen"}, func(ctx context.Context, s *ServicioPlanNominalB2) error {
			_, err := s.ConfirmarOrigenIncorporacionB2(ctx, origen.Confirmacion, actor)
			return err
		}},
		{"leer_origen", []string{"resolver_unidad", ports.AccionLeerPlanNominalB2, "leer_origen"}, func(ctx context.Context, s *ServicioPlanNominalB2) error {
			_, _, err := s.LeerOrigenIncorporacionB2(ctx, contrato.Material.OrganizacionRef, contrato.Material.ExpedienteRef)
			return err
		}},
	}
	for _, op := range operaciones {
		t.Run(op.nombre, func(t *testing.T) {
			// -1 cancela antes de entrar; cada índice restante cancela en un
			// puerto previo al último; len(pasos)-1 recorre el control sin cancelar.
			for i := -1; i < len(op.pasos); i++ {
				nombre := "sin_cancelar"
				if i == -1 {
					nombre = "entrada_cancelada"
				} else if i < len(op.pasos)-1 {
					nombre = op.pasos[i]
				}
				t.Run(nombre, func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					p := &puertosPlanB2Cancelacion{cancelar: cancel, contrato: contrato, origen: origen}
					if i == -1 {
						cancel()
					} else if i < len(op.pasos)-1 {
						p.cancelarEn = op.pasos[i]
					}
					s, err := NuevoServicioPlanNominalB2(p, p, p)
					if err != nil {
						t.Fatal(err)
					}
					err = op.ejecutar(ctx, s)
					if i < len(op.pasos)-1 && !errors.Is(err, context.Canceled) {
						t.Fatalf("cancelación ignorada: %v", err)
					}
					if i == len(op.pasos)-1 && err != nil {
						t.Fatalf("control sin cancelación: %v", err)
					}
					esperadas := op.pasos[:i+1]
					if len(p.llamadas) != len(esperadas) || len(esperadas) > 0 && !reflect.DeepEqual(p.llamadas, esperadas) {
						t.Fatalf("puertos invocados: %v; esperados: %v", p.llamadas, esperadas)
					}
				})
			}
		})
	}
}
