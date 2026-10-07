package domain

import (
	"errors"
	"strings"
	"testing"
)

func consultaRecibosValida() ConsultaFuenteRecibos {
	return ConsultaFuenteRecibos{
		PersonaRef: "persona:uno", RelacionRef: "relacion:uno",
		EntidadPagadoraRef: "entidad:pagadora", PeriodoNomina: "periodo:origen:1",
		Limite: 2,
	}
}

func paginaRecibosValida(c ConsultaFuenteRecibos) PaginaFuenteRecibos {
	pagador := c.EntidadPagadoraRef
	if pagador == "" {
		pagador = "entidad:aportada-por-fuente"
	}
	return PaginaFuenteRecibos{
		Consulta: c, VersionFuente: "fuente:v1", CoberturaRef: "cobertura:1", Total: 1,
		Recibos: []DescriptorRecibo{{
			PersonaRef: c.PersonaRef, RelacionRef: c.RelacionRef,
			EntidadPagadoraRef: pagador, PeriodoNomina: c.PeriodoNomina,
			OrigenRef: "origen:recibo", ReciboRef: "original:1", VersionRecibo: "v1", CustodioRef: "custodio:1",
		}},
	}
}

func TestFuenteRecibosRechazaCrucesYLímites(t *testing.T) {
	c := consultaRecibosValida()
	p := paginaRecibosValida(c)
	if err := ValidarPaginaFuenteRecibos(c, p); err != nil {
		t.Fatalf("pagina valida: %v", err)
	}
	casos := []struct {
		nombre string
		c      ConsultaFuenteRecibos
		p      PaginaFuenteRecibos
	}{
		{"otra persona", c, cambiarRecibo(p, func(r *DescriptorRecibo) { r.PersonaRef = "persona:otra" })},
		{"otra relacion", c, cambiarRecibo(p, func(r *DescriptorRecibo) { r.RelacionRef = "relacion:otra" })},
		{"otra entidad", c, cambiarRecibo(p, func(r *DescriptorRecibo) { r.EntidadPagadoraRef = "entidad:otra" })},
		{"otro periodo", c, cambiarRecibo(p, func(r *DescriptorRecibo) { r.PeriodoNomina = "periodo:otro" })},
		{"filtro distinto", c, func() PaginaFuenteRecibos { x := p; x.Consulta.RelacionRef = "relacion:otra"; return x }()},
		{"total menor", c, func() PaginaFuenteRecibos { x := p; x.Total = 0; return x }()},
		{"total sin continuacion", c, func() PaginaFuenteRecibos { x := p; x.Total = 2; return x }()},
		{"recibo duplicado", c, func() PaginaFuenteRecibos {
			x := p
			x.Total = 2
			x.Recibos = append(append([]DescriptorRecibo{}, p.Recibos...), p.Recibos[0])
			return x
		}()},
		{"pagina mayor que limite", func() ConsultaFuenteRecibos { x := c; x.Limite = 1; return x }(), func() PaginaFuenteRecibos {
			x := p
			x.Consulta.Limite = 1
			x.Total = 2
			x.Recibos = append([]DescriptorRecibo{}, p.Recibos...)
			otro := p.Recibos[0]
			otro.ReciboRef = "original:2"
			x.Recibos = append(x.Recibos, otro)
			return x
		}()},
		{"referencia con control", c, cambiarRecibo(p, func(r *DescriptorRecibo) { r.ReciboRef = "original:\n1" })},
		{"referencia excesiva", c, cambiarRecibo(p, func(r *DescriptorRecibo) { r.CustodioRef = strings.Repeat("x", 257) })},
		{"url en referencia", c, cambiarRecibo(p, func(r *DescriptorRecibo) { r.ReciboRef = "https://origen.invalid/r?id=1" })},
		{"esquema url sin barras", c, cambiarRecibo(p, func(r *DescriptorRecibo) { r.ReciboRef = "javascript:alert" })},
		{"ruta en referencia", c, cambiarRecibo(p, func(r *DescriptorRecibo) { r.CustodioRef = "ruta\\privada" })},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			if err := ValidarPaginaFuenteRecibos(tc.c, tc.p); !errors.Is(err, ErrFuenteRecibosInvalida) {
				t.Fatalf("esperado rechazo, recibido %v", err)
			}
		})
	}
}

func TestFuenteRecibosPrimeraPaginaSinFiltroPagador(t *testing.T) {
	c := consultaRecibosValida()
	c.EntidadPagadoraRef = ""
	p := paginaRecibosValida(c)
	if err := ValidarPaginaFuenteRecibos(c, p); err != nil {
		t.Fatalf("primera pagina sin pagador: %v", err)
	}
	p.Recibos[0].EntidadPagadoraRef = ""
	if err := ValidarPaginaFuenteRecibos(c, p); !errors.Is(err, ErrFuenteRecibosInvalida) {
		t.Fatalf("pagador ausente no rechazado: %v", err)
	}
	p = paginaRecibosValida(c)
	p.Total = 2
	p.CursorSiguiente = "cursor:2"
	if err := ValidarPaginaFuenteRecibos(c, p); err != nil {
		t.Fatalf("primera pagina con continuacion: %v", err)
	}
	c.VersionFuente, c.CoberturaRef, c.TotalEsperado, c.Cursor = p.VersionFuente, p.CoberturaRef, p.Total, p.CursorSiguiente
	p = paginaRecibosValida(c)
	p.Total = 2
	p.Recibos[0].EntidadPagadoraRef = "entidad:segunda"
	if err := ValidarPaginaFuenteRecibos(c, p); err != nil {
		t.Fatalf("continuacion sin filtro pagador: %v", err)
	}
	p.Consulta.EntidadPagadoraRef = "entidad:segunda"
	if err := ValidarPaginaFuenteRecibos(c, p); !errors.Is(err, ErrFuenteRecibosInvalida) {
		t.Fatalf("cambio de filtro al continuar no rechazado: %v", err)
	}
	c = consultaRecibosValida()
	c.EntidadPagadoraRef = "entidad:filtrada"
	p = paginaRecibosValida(c)
	p.Recibos[0].EntidadPagadoraRef = "entidad:otra"
	if err := ValidarPaginaFuenteRecibos(c, p); !errors.Is(err, ErrFuenteRecibosInvalida) {
		t.Fatalf("pagador distinto del filtro no rechazado: %v", err)
	}
}

func TestFuenteRecibosPaginaSiguienteConVistaFijada(t *testing.T) {
	c := consultaRecibosValida()
	p := paginaRecibosValida(c)
	p.Total = 3
	p.CursorSiguiente = "cursor:2"
	if err := ValidarPaginaFuenteRecibos(c, p); err != nil {
		t.Fatalf("primera pagina: %v", err)
	}
	c.VersionFuente, c.CoberturaRef, c.TotalEsperado, c.Cursor = p.VersionFuente, p.CoberturaRef, p.Total, p.CursorSiguiente
	p = paginaRecibosValida(c)
	p.Total = 3
	p.Recibos[0].ReciboRef = "original:2"
	if err := ValidarPaginaFuenteRecibos(c, p); err != nil {
		t.Fatalf("pagina siguiente: %v", err)
	}
	p.VersionFuente = "fuente:v2"
	if err := ValidarPaginaFuenteRecibos(c, p); !errors.Is(err, ErrFuenteRecibosInvalida) {
		t.Fatalf("cambio de version no rechazado: %v", err)
	}
	p.VersionFuente = c.VersionFuente
	p.CoberturaRef = "cobertura:2"
	if err := ValidarPaginaFuenteRecibos(c, p); !errors.Is(err, ErrFuenteRecibosInvalida) {
		t.Fatalf("cambio de cobertura no rechazado: %v", err)
	}
	p.CoberturaRef = c.CoberturaRef
	p.Total = 4
	if err := ValidarPaginaFuenteRecibos(c, p); !errors.Is(err, ErrFuenteRecibosInvalida) {
		t.Fatalf("cambio de total no rechazado: %v", err)
	}
	c.Limite = LimitePaginaRecibos + 1
	if err := ValidarConsultaFuenteRecibos(c); !errors.Is(err, ErrFuenteRecibosInvalida) {
		t.Fatalf("limite excesivo no rechazado: %v", err)
	}
}

func cambiarRecibo(p PaginaFuenteRecibos, cambiar func(*DescriptorRecibo)) PaginaFuenteRecibos {
	p.Recibos = append([]DescriptorRecibo(nil), p.Recibos...)
	cambiar(&p.Recibos[0])
	return p
}
