package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

type calculadoraPortalPrueba struct{ solicitudes []reglas.SolicitudVencimiento }

func (c *calculadoraPortalPrueba) CalcularVencimiento(_ context.Context, s reglas.SolicitudVencimiento) (reglas.Vencimiento, error) {
	c.solicitudes = append(c.solicitudes, s)
	return reglas.Vencimiento{UltimoDia: "2026-10-01", VenceAntesDe: s.Inicio.Add(time.Duration(s.Cantidad) * 24 * time.Hour)}, nil
}

func reglasPortalPrueba(t *testing.T, ruta string) (reglasPortalCandidatoDesarrollo, *calculadoraPortalPrueba) {
	t.Helper()
	calculadora := new(calculadoraPortalPrueba)
	resolutor, err := nuevoResolutorReglasEjemplo(ruta, reglas.CatalogoBolsa, reglas.ModuloBolsa, calculadora, relojPresentacionReglasEjemplo)
	if err != nil {
		t.Fatal(err)
	}
	return reglasPortalCandidatoDesarrollo{resolutor: resolutor}, calculadora
}

func TestReglasPortalCandidatoSalenDelCatalogo(t *testing.T) {
	r, calculadora := reglasPortalPrueba(t, rutaReglasBolsaEjemploPrueba)
	ctx := context.Background()
	modo, ref, err := r.ModoRespuesta(ctx)
	if err != nil || modo != puertosbolsa.ModoRespuestaPortalFirme || !strings.Contains(ref, reglas.BolsaPortalCandidato) {
		t.Fatalf("modo de respuesta: %q %q %v", modo, ref, err)
	}
	if efectivos, err := r.ResultadosContactoEfectivo(ctx); err != nil || !slices.Equal(efectivos, []string{"contactado"}) {
		t.Fatalf("contacto efectivo: %v %v", efectivos, err)
	}
	if s, _, err := r.SituacionesAdmitidas(ctx, "reactivacion"); err != nil || !slices.Equal(s, []string{"no_disponible", "disponible_desde"}) {
		t.Fatalf("situaciones de reactivación: %v %v", s, err)
	}
	if causas, err := r.CausasRenunciaJustificada(ctx); err != nil || !slices.Contains(causas, "enfermedad") || len(causas) != 4 {
		t.Fatalf("causas del art. 10: %v %v", causas, err)
	}
	contacto := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	vence, ref, err := r.VencimientoRespuesta(ctx, contacto)
	if err != nil || !vence.After(contacto) || !strings.Contains(ref, reglas.BolsaPlazoRespuesta) {
		t.Fatalf("plazo de respuesta: %v %q %v", vence, ref, err)
	}
	if _, ref, err := r.PausaMaxima(ctx, contacto); err != nil || !strings.Contains(ref, reglas.BolsaPausaVoluntaria) {
		t.Fatalf("pausa máxima: %q %v", ref, err)
	}
	if len(calculadora.solicitudes) != 2 || calculadora.solicitudes[0].Computo != reglas.ComputoAdministrativo || calculadora.solicitudes[1].Unidad != reglas.UnidadMeses {
		t.Fatalf("cómputos pedidos: %+v", calculadora.solicitudes)
	}
}

func TestReglasPortalCandidatoCambianSinTocarCodigo(t *testing.T) {
	original, err := os.ReadFile(rutaReglasBolsaEjemploPrueba)
	if err != nil {
		t.Fatal(err)
	}
	escribir := func(viejo, nuevo string) string {
		if strings.Count(string(original), viejo) != 1 {
			t.Fatalf("la regla cambió de forma: %s", viejo)
		}
		ruta := filepath.Join(t.TempDir(), "bolsa.demo.json")
		if err := os.WriteFile(ruta, []byte(strings.Replace(string(original), viejo, nuevo, 1)), 0o600); err != nil {
			t.Fatal(err)
		}
		return ruta
	}
	propuesta, _ := reglasPortalPrueba(t, escribir(`"modo_respuesta": "firme"`, `"modo_respuesta": "propuesta_rrhh"`))
	if modo, _, err := propuesta.ModoRespuesta(context.Background()); err != nil || modo != puertosbolsa.ModoRespuestaPortalPropuesta {
		t.Fatalf("modo propuesta: %q %v", modo, err)
	}
	desconocido, _ := reglasPortalPrueba(t, escribir(`"modo_respuesta": "firme"`, `"modo_respuesta": "automatico"`))
	if _, _, err := desconocido.ModoRespuesta(context.Background()); !errors.Is(err, puertosbolsa.ErrPortalCandidatoNoDisponible) {
		t.Fatalf("modo desconocido admitido: %v", err)
	}
	sinRegla, _ := reglasPortalPrueba(t, escribir(`"clave": "b29.portal_candidato"`, `"clave": "b99.otra_regla"`))
	if _, err := sinRegla.ResultadosContactoEfectivo(context.Background()); !errors.Is(err, puertosbolsa.ErrReglasPortalCandidatoAusente) {
		t.Fatalf("sin b29 el portal debe quedar sin acciones: %v", err)
	}
}

func TestMiBolsaConPortalConcedeSoloAccionesPropias(t *testing.T) {
	identidad := &identidadCandidatoBolsaDesarrollo{
		personaRef:   "per_candidato_sintetico_1234567890123456",
		perfilRef:    "prf_candidato_sintetico_1234567890123456",
		candidatoRef: "can_candidato_sintetico_1234567890123456",
	}
	sin, err := nuevaInstantaneaMiBolsaDesarrollo(identidad, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	con, err := nuevaInstantaneaMiBolsaDesarrollo(identidad, time.Now().UTC(), true)
	if err != nil || con.Validar() != nil {
		t.Fatalf("instantánea con portal: %v", err)
	}
	if con.VersionRol.RolID == sin.VersionRol.RolID || len(con.VersionRol.Concesiones) != 5 {
		t.Fatal("el portal debe usar un rol propio con cinco concesiones")
	}
	for _, c := range con.VersionRol.Concesiones[1:] {
		esperado := puertosbolsa.TipoRecursoMiBolsa
		if c.Accion == puertosbolsa.AccionManifestarDisposicionPropia {
			esperado = puertosbolsa.TipoRecursoOfertaBolsa
		}
		if c.TipoRecurso != esperado || len(c.CamposPermitidos) != 0 || len(c.Obligaciones) != 0 ||
			!slices.Equal(c.Finalidades, []string{puertosbolsa.FinalidadPortalCandidato}) {
			t.Fatalf("concesión amplia: %+v", c)
		}
	}
	motivoPortal := motivoPortalMiBolsaDesarrollo()
	politica := &politicaMiBolsaDesarrollo{instantanea: con, motivo: motivoMiBolsaDesarrollo(), motivoPortal: &motivoPortal}
	if m, ok := politica.motivoDe(puertosbolsa.AccionResponderLlamamientoPropio); !ok || m != motivoPortal {
		t.Fatal("la respuesta no usa el motivo del portal")
	}
	if _, ok := (&politicaMiBolsaDesarrollo{instantanea: sin, motivo: motivoMiBolsaDesarrollo()}).motivoDe(puertosbolsa.AccionSolicitarPausaPropia); ok {
		t.Fatal("sin portal compuesto se admite una acción propia")
	}
	if len(descriptoresMaterialPortalCandidatoDesarrollo()) != 4 {
		t.Fatal("audiencias del portal incompletas")
	}
}
