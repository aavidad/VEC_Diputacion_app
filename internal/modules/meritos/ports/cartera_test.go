package ports

import (
	"errors"
	"testing"
	"time"

	vec "vec-diputacion-granada/internal/vec/domain"
)

func TestPaginaCarteraNoEntregaOtroFiltroNiCursorSinAcuse(t *testing.T) {
	corte := time.Date(2026, 10, 12, 14, 30, 0, 0, time.UTC)
	base := PaginaCarteraInterna{Hechos: []FichaCartera{{
		HechoRef: "hecho:2", VersionHecho: 1, Tipo: "curso_asistencia",
		Denominacion: "Curso de archivos", Disponibilidad: DisponibilidadCartera{Estado: "disponible"},
	}}, MotivosRetirada: []MotivoRetiradaCartera{{
		Motivo: vec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_meritos_cartera",
			CatalogoVersion: 1, CatalogoHuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			EntradaClave: "motivo_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		Etiqueta: "Documento incorrecto",
	}}, Total: 2, CursorSiguienteHechoRef: "hecho:2", CorteEn: corte,
		ReciboAcceso: ReciboLecturaCartera{AuditoriaRef: "aud_v3_lp_0123456789abcdef0123456789abcdef",
			Secuencia: "42", HuellaSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			CorrelacionRef: "correlacion:1", ConsultadaEn: corte.Add(time.Millisecond)}}
	if err := base.Validar(corte, "correlacion:1", 1, "disponibles", "hecho:1"); err != nil {
		t.Fatalf("página autorizada: %v", err)
	}
	casos := []struct {
		nombre string
		mutar  func(*PaginaCarteraInterna)
		filtro string
		ultimo string
	}{
		{"filtro ajeno", nil, "retirados", "hecho:1"},
		{"cursor posterior", nil, "disponibles", "hecho:3"},
		{"sin auditoría", func(p *PaginaCarteraInterna) { p.ReciboAcceso.AuditoriaRef = "" }, "disponibles", "hecho:1"},
		{"correlación ajena", func(p *PaginaCarteraInterna) { p.ReciboAcceso.CorrelacionRef = "otra" }, "disponibles", "hecho:1"},
		{"huella inválida", func(p *PaginaCarteraInterna) { p.ReciboAcceso.HuellaSHA256 = "0" }, "disponibles", "hecho:1"},
		{"cursor falso", func(p *PaginaCarteraInterna) { p.CursorSiguienteHechoRef = "hecho:5" }, "disponibles", "hecho:1"},
		{"corte distinto", func(p *PaginaCarteraInterna) { p.CorteEn = corte.Add(time.Second) }, "disponibles", "hecho:1"},
		{"sin motivos publicados", func(p *PaginaCarteraInterna) { p.MotivosRetirada = nil }, "disponibles", "hecho:1"},
		{"etiqueta vacía", func(p *PaginaCarteraInterna) { p.MotivosRetirada[0].Etiqueta = " " }, "disponibles", "hecho:1"},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			p := base
			p.Hechos = append([]FichaCartera(nil), base.Hechos...)
			p.MotivosRetirada = append([]MotivoRetiradaCartera(nil), base.MotivosRetirada...)
			if tc.mutar != nil {
				tc.mutar(&p)
			}
			if err := p.Validar(corte, "correlacion:1", 1, tc.filtro, tc.ultimo); !errors.Is(err, ErrCarteraNoDisponible) {
				t.Fatalf("se entregaría página falsa: %v", err)
			}
		})
	}
}
