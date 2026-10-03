package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorHistoricoPrueba struct {
	a   vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3
	n   int
	err error
}

func (p *autorizadorHistoricoPrueba) AutorizarConsultaOrganizacionHistorica(_ context.Context, _ domain.MaterialConsultaOrganizacionHistorica) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.n++
	return p.a, p.err
}

type repositorioHistoricoPrueba struct {
	r   ports.ResultadoConsultaOrganizacionHistorica
	n   int
	err error
}

func (p *repositorioHistoricoPrueba) ConsultarOrganizacionHistorica(_ context.Context, _ ports.OrdenConsultaOrganizacionHistorica) (ports.ResultadoConsultaOrganizacionHistorica, error) {
	p.n++
	return p.r, p.err
}

func solicitudHistoricaPrueba(t *testing.T) domain.SolicitudConsultaOrganizacionHistorica {
	t.Helper()
	s := solicitudP(t)
	fecha, err := domain.NuevaFechaCivil("2024-12-31")
	if err != nil {
		t.Fatal(err)
	}
	return domain.SolicitudConsultaOrganizacionHistorica{Actor: s.Actor, Selector: domain.SelectorOrganizacionHistorica{
		OrganismoRef: "organismo:dipgra", VigenteEn: fecha, ConocidoEn: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC), Limite: 10,
	}}
}

func autorizacionHistoricaPrueba(t *testing.T, m domain.MaterialConsultaOrganizacionHistorica, operacion string) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	r := m.Recurso()
	n, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), operacion, r.Referencia, h, domain.AudienciaConsultaOrganizacionHistorica, time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC), time.Date(2025, 1, 15, 12, 0, 3, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	actorCanonico, err := m.Solicitud().Actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), n, []byte("d"), []byte("m"), actorCanonico, 1, 1, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func paginaHistoricaVaciaValida(s domain.SelectorOrganizacionHistorica, a vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) ports.ResultadoConsultaOrganizacionHistorica {
	x := a.ResumenCapacidad()
	return ports.ResultadoConsultaOrganizacionHistorica{
		Pagina: ports.PaginaOrganizacionHistorica{Selector: s, Cobertura: ports.CoberturaFuentesOrganizacionHistorica{
			Unidades: "sin_datos", PuestosTipo: "sin_datos", Dotaciones: "sin_datos", Plazas: "sin_datos", PuestosIndividuales: "sin_datos", Vinculos: "sin_datos",
		}},
		Evidencia: ports.EvidenciaConsultaOrganizacionHistorica{ReciboRef: "recibo:uno", DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("a", 64), AuditoriaRef: "auditoria:uno", ConsultadaEn: x.EmitidaEn().Add(time.Microsecond)},
	}
}

func TestConsultaHistoricaFallaCerradoAntesDeLeer(t *testing.T) {
	s := solicitudHistoricaPrueba(t)
	m, err := domain.NuevoMaterialConsultaOrganizacionHistorica(s)
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioHistoricoPrueba{}
	a := &autorizadorHistoricoPrueba{a: autorizacionHistoricaPrueba(t, m, "accion:otra")}
	servicio, err := NuevoServicioConsultaOrganizacionHistorica(a, repo, &intentosHistoricosPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.Consultar(context.Background(), s)
	if !errors.Is(err, domain.ErrConsultaOrganizacionHistoricaDenegada) || repo.n != 0 {
		t.Fatalf("err=%v repo=%d", err, repo.n)
	}
	s.Selector.Limite = 101
	_, err = servicio.Consultar(context.Background(), s)
	if !errors.Is(err, domain.ErrConsultaOrganizacionHistoricaInvalida) || a.n != 1 {
		t.Fatalf("err=%v autorizaciones=%d", err, a.n)
	}
}

func TestConsultaHistoricaDistingueSinDatosYRechazaFilasNoCubiertas(t *testing.T) {
	s := solicitudHistoricaPrueba(t)
	m, err := domain.NuevoMaterialConsultaOrganizacionHistorica(s)
	if err != nil {
		t.Fatal(err)
	}
	aut := autorizacionHistoricaPrueba(t, m, domain.AccionConsultaOrganizacionHistorica)
	resultado := paginaHistoricaVaciaValida(s.Selector, aut)
	repo := &repositorioHistoricoPrueba{r: resultado}
	servicio, _ := NuevoServicioConsultaOrganizacionHistorica(&autorizadorHistoricoPrueba{a: aut}, repo, &intentosHistoricosPrueba{})
	got, err := servicio.Consultar(context.Background(), s)
	if err != nil || got.Pagina.Cobertura.Plazas != "sin_datos" || repo.n != 1 {
		t.Fatalf("resultado=%+v err=%v", got, err)
	}
	resultado.Pagina.Plazas = []domain.PlazaOrganizacionHistorica{{}}
	repo.r = resultado
	_, err = servicio.Consultar(context.Background(), s)
	if !errors.Is(err, domain.ErrOrganizacionHistoricaNoDisponible) {
		t.Fatalf("acepto fila sin evidencia: %v", err)
	}
	resultado.Pagina.Plazas = nil
	resultado.Pagina.Cobertura.Plazas = "desconocida"
	repo.r = resultado
	_, err = servicio.Consultar(context.Background(), s)
	if !errors.Is(err, domain.ErrOrganizacionHistoricaNoDisponible) {
		t.Fatalf("acepto cobertura desconocida: %v", err)
	}
}

func TestConsultaHistoricaConservaSoloDenegacionNominal(t *testing.T) {
	s := solicitudHistoricaPrueba(t)
	m, _ := domain.NuevoMaterialConsultaOrganizacionHistorica(s)
	a := autorizacionHistoricaPrueba(t, m, domain.AccionConsultaOrganizacionHistorica)
	for _, caso := range []struct {
		nombre   string
		fallo    error
		esperado error
	}{
		{"denegacion", domain.ErrConsultaOrganizacionHistoricaDenegada, domain.ErrConsultaOrganizacionHistoricaDenegada},
		{"infraestructura", errors.New("conexion privada"), domain.ErrOrganizacionHistoricaNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			repo := &repositorioHistoricoPrueba{}
			autorizador := &autorizadorHistoricoPrueba{a: a, err: caso.fallo}
			servicio, _ := NuevoServicioConsultaOrganizacionHistorica(autorizador, repo, &intentosHistoricosPrueba{})
			_, err := servicio.Consultar(context.Background(), s)
			if !errors.Is(err, caso.esperado) || repo.n != 0 {
				t.Fatalf("err=%v repo=%d", err, repo.n)
			}
			repo.err = caso.fallo
			autorizador.err = nil
			_, err = servicio.Consultar(context.Background(), s)
			if !errors.Is(err, caso.esperado) || repo.n != 1 {
				t.Fatalf("err=%v repo=%d", err, repo.n)
			}
		})
	}
}

// Dobles focales: verifican orden y propagación; no acreditan persistencia.
type intentosHistoricosPrueba struct {
	motivos   []string
	antes     func()
	err       error
	preflight error
}

func (i *intentosHistoricosPrueba) VerificarRegistroConsultaOrganizacionHistorica(context.Context) error {
	return i.preflight
}
func (i *intentosHistoricosPrueba) RegistrarIntentoConsultaOrganizacionHistorica(ctx context.Context, in ports.IntentoConsultaOrganizacionHistorica) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if i.antes != nil {
		i.antes()
	}
	i.motivos = append(i.motivos, in.Motivo)
	return i.err
}
func TestConsultaHistoricaExigeIntentosYNoDuplicaExito(t *testing.T) {
	s := solicitudHistoricaPrueba(t)
	m, _ := domain.NuevoMaterialConsultaOrganizacionHistorica(s)
	a := autorizacionHistoricaPrueba(t, m, domain.AccionConsultaOrganizacionHistorica)
	repo := &repositorioHistoricoPrueba{r: paginaHistoricaVaciaValida(s.Selector, a)}
	aut := &autorizadorHistoricoPrueba{a: a}
	if _, err := NuevoServicioConsultaOrganizacionHistorica(aut, repo, nil); err == nil {
		t.Fatal("sin registro")
	}
	i := &intentosHistoricosPrueba{}
	svc, _ := NuevoServicioConsultaOrganizacionHistorica(aut, repo, i)
	if _, err := svc.Consultar(context.Background(), s); err != nil || len(i.motivos) != 0 {
		t.Fatal("éxito añade intento o falla")
	}
	repo.err = domain.ErrConsultaOrganizacionHistoricaDenegada
	i.antes = func() {
		if repo.n != 2 {
			t.Fatal("registro antes de repositorio")
		}
	}
	if _, err := svc.Consultar(context.Background(), s); !errors.Is(err, domain.ErrConsultaOrganizacionHistoricaDenegada) || len(i.motivos) != 1 || i.motivos[0] != "denegado" {
		t.Fatal("denegación sin intento")
	}
	i.antes = nil
	i.err = errors.New("destino privado")
	if _, err := svc.Consultar(context.Background(), s); !errors.Is(err, domain.ErrOrganizacionHistoricaNoDisponible) {
		t.Fatal("fallo auditor permite confirmar denegación")
	}
}
func TestConsultaHistoricaRegistraCancelacionYEntradaInvalida(t *testing.T) {
	s := solicitudHistoricaPrueba(t)
	i := &intentosHistoricosPrueba{}
	aut := &autorizadorHistoricoPrueba{}
	repo := &repositorioHistoricoPrueba{}
	svc, _ := NuevoServicioConsultaOrganizacionHistorica(aut, repo, i)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.Consultar(ctx, s); !errors.Is(err, context.Canceled) || len(i.motivos) != 1 || i.motivos[0] != "no_disponible" {
		t.Fatal("cancelación sin registro independiente")
	}
	s.Selector.Limite = 101
	if _, err := svc.Consultar(context.Background(), s); !errors.Is(err, domain.ErrConsultaOrganizacionHistoricaInvalida) || len(i.motivos) != 2 || i.motivos[1] != "entrada_invalida" || aut.n != 0 || repo.n != 0 {
		t.Fatal("entrada inválida alcanzó negocio")
	}
	i.preflight = errors.New("sin ACL")
	s.Selector.Limite = 10
	if _, err := svc.Consultar(context.Background(), s); !errors.Is(err, domain.ErrOrganizacionHistoricaNoDisponible) || aut.n != 0 || repo.n != 0 {
		t.Fatal("preflight abierto")
	}
}
