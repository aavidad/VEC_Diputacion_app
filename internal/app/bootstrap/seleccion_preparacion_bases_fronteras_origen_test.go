package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	selhttp "vec-diputacion-granada/internal/modules/seleccion/adapters/http"
)

type servicioPreparacionFronteraPrueba struct{ efectos int }

func (s *servicioPreparacionFronteraPrueba) Guardar(context.Context, bolsaports.SolicitudGuardarPreparacionBasesV3) (bolsaports.ResultadoPreparacionBasesV3, error) {
	s.efectos++
	return bolsaports.ResultadoPreparacionBasesV3{}, bolsaports.ErrPreparacionBasesNoDisponible
}
func (s *servicioPreparacionFronteraPrueba) Consultar(context.Context, bolsaports.SolicitudConsultarPreparacionBasesV3) (bolsaports.ResultadoPreparacionBasesV3, error) {
	s.efectos++
	return bolsaports.ResultadoPreparacionBasesV3{}, bolsaports.ErrPreparacionBasesNoDisponible
}

func TestPreparacionBasesFronteraBrowserAntesDelServicioYAuditada(t *testing.T) {
	for _, caso := range []struct {
		nombre, site, origin string
		permitido            bool
	}{
		{"same_origin", "same-origin", "https://vec.invalid:8443", true},
		{"cross_site", "cross-site", "https://ajeno.invalid", false},
		{"same_site", "same-site", "https://vec.invalid:8443", false},
		{"origin_null", "", "null", false},
		{"otro_puerto", "", "https://vec.invalid:9443", false},
		{"origin_mismo_host", "", "https://vec.invalid:8443", true},
		{"cli_sin_metadatos", "", "", true},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			for i := range 2 {
				broker, ctx := brokerPreparacionBasesPrueba(t, i)
				auditorias := 0
				frontera, err := nuevaFronteraPreparacionBasesHTTPV3(broker, func(r *http.Request) error {
					if r.Context() != ctx || r.URL.Path != broker.perfiles[i].ruta {
						t.Fatal("auditoría perdió ruta o contexto")
					}
					auditorias++
					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
				servicio := &servicioPreparacionFronteraPrueba{}
				h, err := selhttp.NuevaPreparacionBasesHandler(selhttp.ConfigPreparacionBases{Preparador: servicio,
					ResolverContexto: broker.ResolverContextoHTTP, ValidarFrontera: frontera})
				if err != nil {
					t.Fatal(err)
				}
				entrada := `{"modo":"actual","preparacion_ref":"preparacion:prueba","revision":0,"huella_material_sha256":""}`
				if i == 0 {
					entrada = `{"esperada":{"preparacion_ref":"preparacion:prueba","revision":0,"huella_material_sha256":""},"material":{},"clave_operacion":"clave:prueba"}`
				}
				r := httptest.NewRequest(http.MethodPost, "https://vec.invalid:8443"+broker.perfiles[i].ruta, strings.NewReader(entrada)).WithContext(ctx)
				r.Header.Set("Content-Type", "application/json")
				r.Header.Set("Sec-Fetch-Site", caso.site)
				r.Header.Set("Origin", caso.origin)
				r.Header.Set("Forwarded", "host=ajeno.invalid")
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if caso.permitido {
					if servicio.efectos != 1 || auditorias != 0 {
						t.Fatalf("frontera válida no llegó al servicio: estado=%d", w.Code)
					}
				} else if servicio.efectos != 0 || auditorias != 1 || w.Code != http.StatusForbidden {
					t.Fatalf("rechazo sin barrera o sin auditoría: estado=%d efectos=%d auditorias=%d", w.Code, servicio.efectos, auditorias)
				}
			}
		})
	}
}

func TestPreparacionBasesFronteraAuditoriaCaidaNoAbreEfecto(t *testing.T) {
	b, ctx := brokerPreparacionBasesPrueba(t, 0)
	f, err := nuevaFronteraPreparacionBasesHTTPV3(b, func(*http.Request) error { return errors.New("fallo sintético") })
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "https://vec.invalid:8443"+b.perfiles[0].ruta, nil).WithContext(ctx)
	r.Header.Set("Sec-Fetch-Site", "cross-site")
	if err := f(r); !errors.Is(err, bolsaports.ErrPreparacionBasesNoDisponible) {
		t.Fatal("fallo de auditoría no mantuvo servicio cerrado")
	}
	if _, err := nuevaFronteraPreparacionBasesHTTPV3(b, nil); err == nil {
		t.Fatal("frontera sin auditor admitida")
	}
}
