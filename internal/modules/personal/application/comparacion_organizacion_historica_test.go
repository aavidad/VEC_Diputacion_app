package application

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorComparacionPrueba struct {
	t         *testing.T
	n         int
	denegarEn int
}

func (a *autorizadorComparacionPrueba) AutorizarConsultaOrganizacionHistorica(_ context.Context, m domain.MaterialConsultaOrganizacionHistorica) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.n++
	if a.n == a.denegarEn {
		return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, domain.ErrConsultaOrganizacionHistoricaDenegada
	}
	return autorizacionHistoricaPrueba(a.t, m, domain.AccionConsultaOrganizacionHistorica), nil
}

type repositorioComparacionPrueba struct {
	n                int
	cursores         []string
	cambiarCobertura bool
	cancelar         context.CancelFunc
}

func (r *repositorioComparacionPrueba) ConsultarOrganizacionHistorica(_ context.Context, o ports.OrdenConsultaOrganizacionHistorica) (ports.ResultadoConsultaOrganizacionHistorica, error) {
	r.n++
	s := o.Material.Solicitud().Selector
	r.cursores = append(r.cursores, s.Cursor)
	p := paginaHistoricaVaciaValida(s, o.Autorizacion)
	p.Pagina.Cobertura.Unidades = "parcial"
	if s.Cursor == "" {
		p.Pagina.CursorSiguiente = "pagina_2"
		p.Pagina.Unidades = []domain.UnidadOrganizacionHistorica{{Traza: domain.TrazaOrganizacionHistorica{ID: "unidad:uno", Version: 1, FuenteRef: "fuente:ejemplo", ActoRef: "acto:ejemplo", HuellaSHA256: strings.Repeat("a", 64), EfectosDesde: "2024-01-01", ConocidoDesde: s.ConocidoEn}, CatalogoID: ports.IDCatalogoOrganizacion, CatalogoVersion: 1, CatalogoRevision: 1, ClaveCatalogo: "centro:uno", Tipo: "centro", Etiqueta: "Centro sintético"}}
	} else if r.cambiarCobertura {
		p.Pagina.Cobertura.Unidades = "completa"
	}
	p.Evidencia.ReciboRef = fmt.Sprintf("recibo:pagina:%d", r.n)
	p.Evidencia.AuditoriaRef = fmt.Sprintf("auditoria:pagina:%d", r.n)
	if r.cancelar != nil {
		r.cancelar()
	}
	return p, nil
}
func TestComparacionHistoricaAutorizaCadaPaginaYConservaEvidencias(t *testing.T) {
	a := &autorizadorComparacionPrueba{t: t}
	r := &repositorioComparacionPrueba{}
	c, _ := NuevoServicioConsultaOrganizacionHistorica(a, r, &intentosHistoricosPrueba{})
	s, _ := NuevoServicioComparacionOrganizacionHistorica(c)
	solicitud := solicitudHistoricaPrueba(t)
	got, err := s.Comparar(context.Background(), solicitud, solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if a.n != 4 || r.n != 4 || !reflect.DeepEqual(r.cursores, []string{"", "pagina_2", "", "pagina_2"}) {
		t.Fatalf("authorization bypass: %d %+v", a.n, r)
	}
	if len(got.LecturasAntes) != 2 || len(got.LecturasDespues) != 2 || got.LecturasAntes[1].Evidencia.ReciboRef != "recibo:pagina:2" || got.LecturasDespues[0].Evidencia.AuditoriaRef != "auditoria:pagina:3" || got.LecturasAntes[1].Selector.Cursor != "pagina_2" {
		t.Fatalf("evidence lost: %+v", got)
	}
	if got.Comparacion.Unidades.Antes != 1 || got.Comparacion.Unidades.Estado != "parcial" || len(got.Comparacion.Unidades.Cambios) != 0 {
		t.Fatalf("coverage lost: %+v", got.Comparacion)
	}
}
func TestComparacionHistoricaFallaSinResultadoParcial(t *testing.T) {
	for _, caso := range []string{"denegacion", "cobertura", "cancelacion", "selector inicial", "segunda solicitud invalida", "ambito distinto"} {
		t.Run(caso, func(t *testing.T) {
			a := &autorizadorComparacionPrueba{t: t}
			r := &repositorioComparacionPrueba{}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c, _ := NuevoServicioConsultaOrganizacionHistorica(a, r, &intentosHistoricosPrueba{})
			s, _ := NuevoServicioComparacionOrganizacionHistorica(c)
			antes := solicitudHistoricaPrueba(t)
			despues := antes
			esperado := domain.ErrOrganizacionHistoricaNoDisponible
			switch caso {
			case "denegacion":
				a.denegarEn = 2
				esperado = domain.ErrConsultaOrganizacionHistoricaDenegada
			case "cobertura":
				r.cambiarCobertura = true
			case "cancelacion":
				r.cancelar = cancel
				esperado = context.Canceled
			case "selector inicial":
				antes.Selector.Cursor = "pagina_2"
				esperado = domain.ErrComparacionOrganizacionHistoricaInvalida
			case "segunda solicitud invalida":
				despues.Selector.Limite = 101
				esperado = domain.ErrConsultaOrganizacionHistoricaInvalida
			case "ambito distinto":
				despues.Selector.OrganismoRef = "organismo:otro"
				esperado = domain.ErrComparacionOrganizacionHistoricaInvalida
			}
			got, err := s.Comparar(ctx, antes, despues)
			if err != esperado || !reflect.DeepEqual(got, ResultadoComparacionOrganizacionHistorica{}) {
				t.Fatalf("result=%+v err=%v", got, err)
			}
			if (caso == "selector inicial" || caso == "segunda solicitud invalida" || caso == "ambito distinto") && a.n != 0 {
				t.Fatal("consumed authorization before validation")
			}
		})
	}
}
func TestComparacionHistoricaDependenciasNulas(t *testing.T) {
	for _, c := range []*ServicioConsultaOrganizacionHistorica{nil, {}} {
		if _, err := NuevoServicioComparacionOrganizacionHistorica(c); err == nil {
			t.Fatal("nil dependency accepted")
		}
	}
	var s *ServicioComparacionOrganizacionHistorica
	if _, err := s.Comparar(context.Background(), domain.SolicitudConsultaOrganizacionHistorica{}, domain.SolicitudConsultaOrganizacionHistorica{}); err == nil {
		t.Fatal("nil service accepted")
	}
}
