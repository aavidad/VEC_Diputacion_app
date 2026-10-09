package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	bolsapersonal "vec-diputacion-granada/internal/modules/bolsa/adapters/httppersonal"
	mibolsa "vec-diputacion-granada/internal/modules/bolsa/application/mibolsa"
	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	puertosvec "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/reglas"
)

type preparadorRetiradaRRHH18 struct{}

func (preparadorRetiradaRRHH18) PrepararMiBolsa(*http.Request) (mibolsa.Orden, error) {
	return mibolsa.Orden{}, nil
}

type registroRetiradaRRHH18 struct {
	puertosbolsa.RegistroPortalCandidato
	llamadas int
}

func (r *registroRetiradaRRHH18) SolicitarPortal(context.Context, puertosbolsa.SolicitudPortalCandidato) (puertosbolsa.ReciboSolicitudPortal, error) {
	r.llamadas++
	return puertosbolsa.ReciboSolicitudPortal{}, errors.New("efecto inesperado")
}

type autorizadorRetiradaRRHH18 struct {
	puertosvec.AutorizadorSolicitudLigadaV3
	llamadas int
}

func (a *autorizadorRetiradaRRHH18) ExigirSolicitudLigadaV3(context.Context, dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3, error) {
	a.llamadas++
	return dominiovec.DecisionAutorizacionLigadaV3{}, puertosvec.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, dominiovec.ErrAutorizacionDenegada
}

type proveedorRetiradaRRHH18 struct {
	puertosbolsa.ProveedorMaterialPortalCandidato
}

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
	for _, tipo := range []string{puertosbolsa.SolicitudPortalPausa, puertosbolsa.SolicitudPortalReactivacion} {
		if _, _, err := sinRegla.SituacionesAdmitidas(t.Context(), tipo); err != puertosbolsa.ErrPausaPortalNoConfigurada {
			t.Fatalf("sin b29 no hay acción %s: %v", tipo, err)
		}
	}
	if _, err := listaAtributoRegla(reglas.Regla{Atributos: map[string]string{"pausa_desde": " disponible"}}, "pausa_desde"); !errors.Is(err, puertosbolsa.ErrPortalCandidatoNoDisponible) || errors.Is(err, puertosbolsa.ErrPausaPortalNoConfigurada) {
		t.Fatalf("atributo malformado convertido en ausencia: %v", err)
	}
}

func TestReglasRRHH18SinPausaConservanConsultaYDenieganPlazo(t *testing.T) {
	for _, ruta := range []string{
		"../../../data/demo/reglas/bolsa_reglas.rrhh-20261002.v4.json",
		"../../../data/demo/reglas/bolsa_reglas.rrhh-20261008.v5.json",
	} {
		t.Run(filepath.Base(ruta), func(t *testing.T) {
			resolutor, err := nuevoResolutorReglasEjemplo(ruta, reglas.CatalogoBolsa, reglas.ModuloBolsa,
				new(calculadoraPortalPrueba), relojReglasEjemploPrueba{ahora: time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)})
			if err != nil {
				t.Fatal(err)
			}
			r := reglasPortalCandidatoDesarrollo{resolutor: resolutor}
			efectivos, err := r.ResultadosContactoEfectivo(t.Context())
			if err != nil || !slices.Equal(efectivos, []string{"contactado"}) {
				t.Fatalf("contacto efectivo vigente: %v %v", efectivos, err)
			}
			if modo, _, err := r.ModoRespuesta(t.Context()); err != nil || modo == "" {
				t.Fatalf("modo vigente: %q %v", modo, err)
			}
			if _, _, err := r.PausaMaxima(t.Context(), time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)); err != puertosbolsa.ErrPausaPortalNoConfigurada {
				t.Fatalf("la pausa retirada no debe tener plazo: %v", err)
			}
			for _, tipo := range []string{puertosbolsa.SolicitudPortalPausa, puertosbolsa.SolicitudPortalReactivacion} {
				if _, _, err := r.SituacionesAdmitidas(t.Context(), tipo); err != puertosbolsa.ErrPausaPortalNoConfigurada {
					t.Fatalf("la acción %s retirada no debe admitir situaciones: %v", tipo, err)
				}
			}
		})
	}
}

func TestPortalRRHH18RetiradaPausaYReactivacionDa409SinEfecto(t *testing.T) {
	for _, ruta := range []string{
		"../../../data/demo/reglas/bolsa_reglas.rrhh-20261002.v4.json",
		"../../../data/demo/reglas/bolsa_reglas.rrhh-20261008.v5.json",
	} {
		for _, caso := range []struct{ nombre, cuerpo string }{
			{"pausa", `{"tipo":"pausa","bolsa":"bolsa:auxiliar","pausa_hasta":"2026-10-10T00:00:00Z","clave":"clave-pausa-retirada"}`},
			{"reactivacion", `{"tipo":"reactivacion","bolsa":"bolsa:auxiliar","clave":"clave-reactiva-retirada"}`},
		} {
			t.Run(filepath.Base(ruta)+"/"+caso.nombre, func(t *testing.T) {
				reloj := relojReglasEjemploPrueba{ahora: time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)}
				resolutor, err := nuevoResolutorReglasEjemplo(ruta, reglas.CatalogoBolsa, reglas.ModuloBolsa,
					new(calculadoraPortalPrueba), reloj)
				if err != nil {
					t.Fatal(err)
				}
				registro, autorizador := new(registroRetiradaRRHH18), new(autorizadorRetiradaRRHH18)
				portal, err := mibolsa.NuevoPortal(registro, autorizador, new(proveedorRetiradaRRHH18),
					reglasPortalCandidatoDesarrollo{resolutor: resolutor}, reloj)
				if err != nil {
					t.Fatal(err)
				}
				h, err := bolsapersonal.NuevoPortal(bolsapersonal.RutaMiBolsaSolicitudes, preparadorRetiradaRRHH18{}, portal)
				if err != nil {
					t.Fatal(err)
				}
				peticion := httptest.NewRequest(http.MethodPost, bolsapersonal.RutaMiBolsaSolicitudes, strings.NewReader(caso.cuerpo))
				peticion.Header.Set("Content-Type", "application/json")
				peticion.Header.Set("Accept", "application/json")
				respuesta := httptest.NewRecorder()
				h.ServeHTTP(respuesta, peticion)
				if respuesta.Code != http.StatusConflict ||
					respuesta.Body.String() != `{"error":{"codigo":"pausa_no_disponible"}}` ||
					autorizador.llamadas != 0 || registro.llamadas != 0 {
					t.Fatalf("acción retirada: estado=%d permiso=%d efecto=%d", respuesta.Code,
						autorizador.llamadas, registro.llamadas)
				}
			})
		}
	}
}

func TestReglasRRHHVersion3ConservanPausaConfigurada(t *testing.T) {
	reloj := relojReglasEjemploPrueba{ahora: time.Date(2026, 10, 9, 8, 0, 0, 0, time.UTC)}
	resolutor, err := nuevoResolutorReglasEjemplo("../../../data/demo/reglas/bolsa_reglas.rrhh-20261002.v3.json",
		reglas.CatalogoBolsa, reglas.ModuloBolsa, new(calculadoraPortalPrueba), reloj)
	if err != nil {
		t.Fatal(err)
	}
	r := reglasPortalCandidatoDesarrollo{resolutor: resolutor}
	for _, tipo := range []string{puertosbolsa.SolicitudPortalPausa, puertosbolsa.SolicitudPortalReactivacion} {
		situaciones, _, err := r.SituacionesAdmitidas(t.Context(), tipo)
		if err != nil || len(situaciones) == 0 {
			t.Fatalf("la regla anterior de %s no quedó disponible: %v", tipo, err)
		}
	}
	if fin, ref, err := r.PausaMaxima(t.Context(), reloj.Ahora()); err != nil ||
		!fin.After(reloj.Ahora()) || !strings.Contains(ref, reglas.BolsaPausaVoluntaria) {
		t.Fatalf("la pausa anterior perdió su fecha catalogada: %v", err)
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
	if con.VersionRol.RolID == sin.VersionRol.RolID || len(con.VersionRol.Concesiones) != 2+len(accionesPropiasPortalDesarrollo()) {
		t.Fatal("el portal debe usar un rol propio con la consulta y cada acción propia")
	}
	for _, c := range con.VersionRol.Concesiones[2:] {
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
	if len(descriptoresMaterialPortalCandidatoDesarrollo()) != len(puertosbolsa.AccionesPortalCandidato()) {
		t.Fatal("audiencias del portal incompletas")
	}
}
