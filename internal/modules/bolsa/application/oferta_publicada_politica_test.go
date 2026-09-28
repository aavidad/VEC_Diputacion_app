package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type plazoOfertaVersionadaPrueba struct {
	bolsa  string
	legacy int
}

func (p *plazoOfertaVersionadaPrueba) PlazoDisposicion(context.Context, time.Time) (ports.PlazoOferta, time.Time, error) {
	p.legacy++
	return ports.PlazoOferta{}, time.Time{}, ports.ErrPlazoOfertaNoConfigurado
}
func (p *plazoOfertaVersionadaPrueba) PlazoDisposicionBolsa(_ context.Context, bolsa string, desde time.Time) (ports.PlazoOferta, time.Time, error) {
	p.bolsa = bolsa
	return ports.PlazoOferta{ReglaRef: "politica-ofertas:" + bolsa + ":3", HuellaCatalogo: strings.Repeat("a", 64), PoliticaVersion: 3, MunicipioSede: "18087", Unidad: "dias_habiles", Cantidad: 2, Computo: "administrativo", UltimoDia: "2026-09-29", Ejemplo: true, Calendarios: []string{"calendario:2026:1"}}, desde.Add(72 * time.Hour), nil
}

func TestPublicarOfertaUsaPoliticaVersionadaDeBolsa(t *testing.T) {
	ahora := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	repo := &repositorioOfertasPrueba{}
	plazo := &plazoOfertaVersionadaPrueba{}
	s, err := NuevoServicioOfertasPublicadas(contextoContactoPrueba{}, &autorizadorBorradorPrueba{t: t, instante: ahora}, repo, plazo, func() time.Time { return ahora })
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.PublicarOferta(t.Context(), solicitudPublicarOfertaPrueba(t, ahora))
	if err != nil || repo.publicado == nil || plazo.bolsa != "bolsa:of" || plazo.legacy != 0 ||
		repo.publicado.Plazo.PoliticaVersion != 3 || repo.publicado.Plazo.MunicipioSede != "18087" ||
		len(repo.publicado.Plazo.Calendarios) != 1 {
		t.Fatalf("err=%v bolsa=%q legacy=%d comando=%+v", err, plazo.bolsa, plazo.legacy, repo.publicado)
	}
}

func TestHuellaMaterialPlazoOfertaCoincideConContextoV3(t *testing.T) {
	publicada := time.Date(2026, 9, 25, 10, 0, 0, 123456000, time.UTC)
	vence := time.Date(2026, 9, 29, 22, 0, 0, 0, time.UTC)
	plazo := ports.PlazoOferta{
		ReglaRef: "politica-ofertas:bolsa:of:1:1", HuellaCatalogo: strings.Repeat("a", 64),
		Unidad: "dias_habiles", Cantidad: 2, Computo: "administrativo", MunicipioSede: "18087",
		UltimoDia: "2026-09-29", PoliticaVersion: 1, Calendarios: []string{"cal:1", "cal:2"},
	}
	h := huellaMaterialPlazoOferta("bolsa:of:1", publicada, vence, plazo)
	if h != "2c8e98be8253c155f74ad913cdb6da08f46548e46c1f3cdfe1ed7a26670cc350" {
		t.Fatalf("huella de plazo divergente: %s", h)
	}
	r := dominiovec.RecursoAutorizable{Referencia: "bolsa:of:1", ModuloID: "bolsa", Tipo: "bolsa_constituida",
		Ambitos:   map[string]string{"unidad_ref": "unidad:rrhh", "ambito_ref": "ambito:bolsa"},
		Atributos: map[string]string{"material_sha256": h}}
	contexto, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil || contexto != "064fb6a33c88bb3ae1c4f43b51baaf4bc56404c7a39b1c83a71d931aa8e578f1" {
		t.Fatalf("contexto V3 divergente: %s %v", contexto, err)
	}
}

func TestHuellaMaterialHorasIncluyeAperturaYVencimiento(t *testing.T) {
	apertura := time.Date(2026, 3, 28, 12, 17, 13, 123456000, time.UTC)
	vence := apertura.Add(48 * time.Hour)
	plazo := ports.PlazoOferta{
		ReglaRef: "politica-ofertas:bolsa:of:1:2", HuellaCatalogo: strings.Repeat("a", 64),
		Unidad: "horas_naturales", Cantidad: 48, Computo: "continuo_utc", MunicipioSede: "18087",
		UltimoDia: "2026-03-30", PoliticaVersion: 2, Calendarios: []string{"calendario:utc-continuo:v1"},
		AperturaEn: apertura.Format(formatoInstanteMaterialOferta), VenceEn: vence.Format(formatoInstanteMaterialOferta),
	}
	h := huellaMaterialPlazoOferta("bolsa:of:1", apertura, vence, plazo)
	if h != "fa8852b8fc417012ce19880e4525e2d57247ef78ef33f8d0da4568b58e735255" {
		t.Fatalf("vector Go/SQL B54 divergente: %s", h)
	}
	plazo.VenceEn = vence.Add(90 * 24 * time.Hour).Format(formatoInstanteMaterialOferta)
	if huellaMaterialPlazoOferta("bolsa:of:1", apertura, vence, plazo) == h {
		t.Fatal("la huella V3 no liga el vencimiento del recibo")
	}
}
