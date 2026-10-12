package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/documentos/domain"
	"vec-diputacion-granada/internal/vec/documentos/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type fuentePlanRegistroPrueba struct {
	plan     ports.PlanIngestaMayan
	llamadas int
}

func (f *fuentePlanRegistroPrueba) ConsultarPlanConfirmadoParaRegistro(_ context.Context, _ string,
	_ vecdomain.ResultadoContextoActorRegistradoV2, _ vecdomain.VinculoAutenticacionActorV2) (ports.PlanIngestaMayan, error) {
	f.llamadas++
	return f.plan, nil
}

type politicaPlanRegistroPrueba struct {
	solicitud vecports.SolicitudPoliticaConservacionDocumental
}

func (p politicaPlanRegistroPrueba) SolicitudParaPlan(context.Context, ports.PlanIngestaMayan) (vecports.SolicitudPoliticaConservacionDocumental, error) {
	return p.solicitud, nil
}

type registradorPlanPrueba struct {
	llamadas int
	alta     ports.AltaExterna
	plan     ports.PlanIngestaMayan
}

func (r *registradorPlanPrueba) RegistrarExternoAutorizado(_ context.Context, a ports.AltaExterna,
	_ ports.AutorizadorRegistroExterno) (domain.Documento, error) {
	r.llamadas++
	r.alta = a
	return domain.Documento{ID: a.ID, NumeroVEC: "VEC-2026-7", ModuloID: a.ModuloID,
		ExpedienteRef: a.ExpedienteRef, TipoRef: a.TipoRef, Version: a.Version, MIME: a.MIME,
		Tamano: a.Tamano, HuellaSHA256: a.Custodia.HuellaSHA256, PoliticaRef: r.plan.PoliticaRef,
		VersionPolitica: uint64(r.plan.VersionPolitica), HuellaPoliticaSHA256: r.plan.HuellaPoliticaSHA256,
		ConservacionHasta: r.plan.ConservacionHasta, Proteccion: r.plan.Proteccion,
		EstadoPolitica: r.plan.EstadoPolitica, EstadoFirma: domain.EstadoFirmaPendienteProveedor,
		CreadoEn: r.plan.ActualizadoEn, Custodia: domain.CustodiaExterna, CustodiaExternaRef: a.Custodia}, nil
}

func escenarioRegistroMayan(t *testing.T) (*ServicioRegistroMayan, *fuentePlanRegistroPrueba, *registradorPlanPrueba,
	vecdomain.ResultadoContextoActorRegistradoV2, vecdomain.VinculoAutenticacionActorV2) {
	t.Helper()
	e := nuevoEscenarioOrdenAutorizacionV3Prueba(t)
	datosActor, err := e.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	p, _ := cargaAdmitidaIngestaPrueba(t, e.ahora, e.resultado.Contexto.PersonaRef)
	ref := func(c string) string { return "ref:" + strings.Repeat(c, 64) }
	p.RegistroContextoRef, p.HuellaContextoSHA256 = e.resultado.RegistroContextoRef, e.resultado.HuellaSHA256
	p.TipoRef, p.PoliticaRef, p.VersionPolitica = ref("a"), ref("b"), 1
	p.HuellaPoliticaSHA256 = strings.Repeat("c", 64)
	p.ConservacionHasta = time.Date(2035, 1, 1, 0, 0, 0, 0, time.UTC)
	p.EstadoPolitica, p.Proteccion = "aprobada", "conservacion"
	p.ClaveIdempotencia, p.ClaveReserva, p.HuellaSolicitudSHA256 = ref("c"), ref("d"), strings.Repeat("d", 64)
	p.Estado, p.Version, p.DocumentoID, p.ArchivoID, p.VersionID = ports.EstadoIngestaConfirmada, 3, "21", "34", "55"
	p.CreadoEn, p.ActualizadoEn = e.ahora.Add(-time.Minute), e.ahora
	huella, _ := hex.DecodeString(p.HuellaPoliticaSHA256)
	solicitud, err := vecports.NuevaSolicitudPoliticaConservacionDocumental(
		ref("e"), ref("f"), p.TipoRef, p.ExpedienteRef, p.PoliticaRef,
		uint64(p.VersionPolitica), huella, ref("1"), e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	fuente := &fuentePlanRegistroPrueba{plan: p}
	registro := &registradorPlanPrueba{plan: p}
	s := &ServicioRegistroMayan{Planes: fuente, Politicas: politicaPlanRegistroPrueba{solicitud},
		Registrador: registro, Autoridad: autorizadorRegistroExternoPrueba{}}
	return s, fuente, registro, e.resultado, datosActor.VinculoAutenticacionActor
}

type autorizadorRegistroExternoPrueba struct{}

func (autorizadorRegistroExternoPrueba) AutorizarRegistroExterno(context.Context, []byte, string, string) (ports.AutorizacionV3, error) {
	return ports.AutorizacionV3{}, errors.New("el doble de registro no solicita COSE")
}

func TestRegistroMayanDerivaTuplaDelPlanYReplayConservaRecibo(t *testing.T) {
	s, fuente, registro, actor, vinculo := escenarioRegistroMayan(t)
	plan := fuente.plan
	primero, err := s.Registrar(context.Background(), plan.PlanRef, actor, vinculo)
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := s.Registrar(context.Background(), plan.PlanRef, actor, vinculo)
	if err != nil || primero != segundo || fuente.llamadas != 2 || registro.llamadas != 2 {
		t.Fatalf("replay documental divergente: %v", err)
	}
	h := sha256.Sum256([]byte(dominioReferenciaMeritosMayan + plan.PlanRef))
	if registro.alta.ID != "ref:"+hex.EncodeToString(h[:]) || registro.alta.ClaveIdempotencia != plan.PlanRef ||
		registro.alta.Custodia.Referencia != "mayan:21:34:55" ||
		registro.alta.Custodia.HuellaSHA256 != plan.HuellaSHA256 || registro.alta.ModuloID != "meritos" {
		t.Fatalf("alta no deriva del plan: %+v", registro.alta)
	}
}

func TestRegistroMayanDeniegaPlanInciertoOCruzadoAntesDeD3(t *testing.T) {
	for nombre, alterar := range map[string]func(*ports.PlanIngestaMayan){
		"sin_confirmar":        func(p *ports.PlanIngestaMayan) { p.Estado = ports.EstadoIngestaIndeterminada },
		"persona_ajena":        func(p *ports.PlanIngestaMayan) { p.PersonaRef = "per_abcdefghijklmnopqrstuvwxyz" },
		"version_remota_vacia": func(p *ports.PlanIngestaMayan) { p.VersionID = "" },
		"politica_divergente":  func(p *ports.PlanIngestaMayan) { p.HuellaPoliticaSHA256 = strings.Repeat("e", 64) },
	} {
		t.Run(nombre, func(t *testing.T) {
			s, fuente, registro, actor, vinculo := escenarioRegistroMayan(t)
			alterar(&fuente.plan)
			if _, err := s.Registrar(context.Background(), fuente.plan.PlanRef, actor, vinculo); !errors.Is(err, ports.ErrSolicitudInvalida) || registro.llamadas != 0 {
				t.Fatalf("plan inválido llegó a D3: %v, llamadas=%d", err, registro.llamadas)
			}
		})
	}
}
