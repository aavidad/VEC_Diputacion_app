package application

import (
	"context"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type autoridadPropuestaRolRecuperablePrueba struct {
	*autoridadGobiernoPerfilPrueba
	replay, cruzar bool
	caduca         time.Time
}

func (a *autoridadPropuestaRolRecuperablePrueba) ProponerGobiernoRolNuevoRecuperable(ctx context.Context,
	o domain.OrdenPropuestaGobiernoPerfil) (ports.ResultadoPropuestaGobiernoRolNuevo, error) {
	p, err := a.autoridadGobiernoPerfilPrueba.ProponerGobiernoPerfil(ctx, o)
	if err != nil {
		return ports.ResultadoPropuestaGobiernoRolNuevo{}, err
	}
	p.CaducaEn = a.caduca
	if a.cruzar {
		p.Material.Plan.DefinicionNueva.Nombre = "Otro rol"
	}
	return ports.ResultadoPropuestaGobiernoRolNuevo{Propuesta: p, Replay: a.replay,
		AuditoriaAccesoRef: "aud_v3_" + strings.Repeat("a", 32)}, nil
}

func TestGobiernoRolDistinguePropuestaNuevaDeReplayHistorico(t *testing.T) {
	for _, caso := range []struct {
		nombre                   string
		replay, cruzar, admitida bool
	}{
		{"primera_caducada", false, false, false},
		{"replay_historico_exacto", true, false, true},
		{"replay_material_cruzado", true, true, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			servicio, s, anterior, _ := gobiernoPerfilAplicacionPrueba(t)
			autoridad := &autoridadPropuestaRolRecuperablePrueba{autoridadGobiernoPerfilPrueba: anterior,
				replay: caso.replay, cruzar: caso.cruzar, caduca: anterior.ahora.Add(-time.Minute)}
			servicio.actos = autoridad
			r, err := servicio.ProponerGobiernoRolNuevo(context.Background(), s)
			if (err == nil) != caso.admitida || (err == nil && r.Propuesta.CaducaEn.After(anterior.ahora)) {
				t.Fatalf("propuesta/replay: %+v %v", r, err)
			}
		})
	}
}

func TestGobiernoRolRecuperaPropuestaConCatalogoHistoricoExacto(t *testing.T) {
	servicio, s, anterior, _ := gobiernoPerfilAplicacionPrueba(t)
	// La fuente AUT58 conserva la instantánea original, ya fuera de vigencia.
	// La fecha de preparación sale de esa entrada, nunca del body.
	anterior.catalogo.VigenteHasta = anterior.ahora.Add(-time.Minute)
	anterior.catalogo.Entradas[0].VigenteHasta = anterior.catalogo.VigenteHasta
	he, err := anterior.catalogo.Entradas[0].HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	hc, err := anterior.catalogo.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	s.Intencion.Publicacion.CatalogoHuellaSHA256 = hc
	s.Intencion.Publicacion.Selecciones[0].EntradaHuellaSHA256 = he
	plan, instante, err := domain.PrepararPlanGobiernoRolNuevoDesdeCatalogo(anterior.catalogo,
		s.Intencion, anterior.ahora, s.Actor.PersonaRef)
	if err != nil || !instante.Equal(anterior.catalogo.Entradas[0].VigenteDesde) {
		t.Fatalf("instante histórico fuente: %v", err)
	}
	s.HuellaPlanEsperada, err = plan.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	autoridad := &autoridadPropuestaRolRecuperablePrueba{autoridadGobiernoPerfilPrueba: anterior,
		replay: true, caduca: anterior.ahora.Add(-time.Minute)}
	servicio.actos = autoridad
	r, err := servicio.ProponerGobiernoRolNuevo(context.Background(), s)
	if err != nil || !r.Replay || r.Propuesta.HuellaSHA256 == "" {
		t.Fatalf("replay histórico exacto inaccesible: %+v %v", r, err)
	}
	// La misma fuente caducada no convierte un primer efecto en permitido.
	autoridad.replay = false
	if r, err := servicio.ProponerGobiernoRolNuevo(context.Background(), s); err == nil || r.Propuesta.HuellaSHA256 != "" {
		t.Fatal("primera propuesta caducada aceptada")
	}
}

func (a *autoridadGobiernoPerfilPrueba) CerrarGobiernoRolPorReferencia(
	_ context.Context, _ domain.SolicitudCierreGobiernoRolPorReferencia,
) (domain.CierreGobiernoPerfil, error) {
	a.cierres++
	return a.cierre, nil
}

func TestGobiernoRolPorReferenciaRecuperaProponenteDelMaterialDurable(t *testing.T) {
	servicio, completa, autoridad := cierreGobiernoPerfilAplicacionPrueba(t)
	s := domain.SolicitudCierreGobiernoRolPorReferencia{OperacionRef: completa.OperacionRef,
		PropuestaRef: completa.PropuestaRef, PropuestaHuellaSHA256: completa.PropuestaHuellaSHA256,
		Aprobador: completa.Aprobador, Evidencia: completa.Evidencia,
		InstantaneaAutorizacion: completa.InstantaneaAutorizacion,
		Decision:                completa.Decision, Motivo: completa.Motivo, CorrelacionRef: completa.CorrelacionRef}
	cierre, err := servicio.CerrarGobiernoRolPorReferencia(context.Background(), s)
	if err != nil || cierre.ValidarPara(completa) != nil || autoridad.cierres != 1 {
		t.Fatalf("cierre por referencia: %v", err)
	}
	// Si la autoridad devolviera el material de otra propuesta, la aplicación
	// rechaza incluso un recibo que aparenta estar confirmado.
	autoridad.cierre.Material.ProponentePersonaRef = completa.Aprobador.PersonaRef
	if c, err := servicio.CerrarGobiernoRolPorReferencia(context.Background(), s); err == nil || c.Recibo != nil {
		t.Fatal("proponente de otra identidad aceptado en respuesta durable")
	}
	// El selector opaco alterado tampoco puede recibir un cierre anterior.
	s.PropuestaHuellaSHA256 = "0000000000000000000000000000000000000000000000000000000000000000"
	if c, err := servicio.CerrarGobiernoRolPorReferencia(context.Background(), s); err == nil || c.Recibo != nil {
		t.Fatal("huella de propuesta divergente aceptada")
	}
}
