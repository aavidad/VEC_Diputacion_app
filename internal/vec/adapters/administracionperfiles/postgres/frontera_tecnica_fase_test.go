package postgres

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	api "vec-diputacion-granada/internal/vec/adapters/httpapi/administracionperfiles"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestFronteraSesionResueltaIncompatibleNuncaEsTecnica(t *testing.T) {
	p := &poolTecnicoPrueba{}
	nominal := &nominalTecnicoPrueba{}
	a, err := NuevoAuditorFronteraCompuesto(nominal, registradorTecnicoPrueba(p))
	if err != nil {
		t.Fatal(err)
	}
	err = a.RegistrarDenegacionADMIN(context.Background(), api.DenegacionADMIN{SesionResuelta: true, Codigo: "respuesta_incompatible"})
	if err == nil || len(p.eventos) != 0 || nominal.llamadas != 0 {
		t.Fatal("sesion_resuelta_downgrade")
	}
}

type sesionIncompatibleFrontera struct{ s api.SesionConfiable }

func (s sesionIncompatibleFrontera) ResolverSesionADMIN(context.Context, *http.Request) (api.SesionConfiable, error) {
	return s.s, nil
}

type lecturasNoInvocadasFrontera struct{ api.FuenteLecturas }

type registroAntesRespuestaFrontera struct {
	*registroFronteraPrueba
	w                *httptest.ResponseRecorder
	respuestaEmitida bool
}

func (r *registroAntesRespuestaFrontera) AppendIntentoAuditoria(ctx context.Context, o ports.OrdenIntentoAuditoria) (ports.AcuseIntentoAuditoria, error) {
	r.respuestaEmitida = r.w.Code != http.StatusOK || r.w.Body.Len() != 0
	return r.registroFronteraPrueba.AppendIntentoAuditoria(ctx, o)
}

func TestFronteraSesionIncompatibleAppendNominalAntes503(t *testing.T) {
	for _, metodo := range []string{http.MethodGet, http.MethodPost} {
		t.Run(metodo, func(t *testing.T) {
			ahora := time.Now().UTC().Truncate(time.Microsecond)
			actor, evidencia := sesionFronteraPrueba(t, ahora)
			// La evidencia es original y usable. La instantánea del resolutor es
			// incoherente y su correlación inválida; ninguna sustituye el contexto V2.
			s := api.SesionConfiable{Actor: actor, Evidencia: evidencia, CorrelacionRef: "SECRET_correlacion_sesion_invalida"}
			w := httptest.NewRecorder()
			reg := &registroAntesRespuestaFrontera{registroFronteraPrueba: &registroFronteraPrueba{ahora: ahora}, w: w}
			nominal, err := NuevaAuditorFronteraNominal(reg, configFronteraPrueba())
			if err != nil {
				t.Fatal(err)
			}
			pool := &poolTecnicoPrueba{}
			compuesto, err := NuevoAuditorFronteraCompuesto(nominal, registradorTecnicoPrueba(pool))
			if err != nil {
				t.Fatal(err)
			}
			h, err := api.NuevoHandlerLecturas("https://admin.example.test", sesionIncompatibleFrontera{s}, &lecturasNoInvocadasFrontera{}, compuesto)
			if err != nil {
				t.Fatal(err)
			}
			ctx, err := ports.ConCorrelacionIncidenciasPeticion(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			privada, err := ports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
			if err != nil {
				t.Fatal(err)
			}
			ref, err := privada.ValorCanonico()
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(metodo, "https://admin.example.test"+api.PrefijoV1+"/personas", nil).WithContext(ctx)
			r.TLS = &tls.ConnectionState{VerifiedChains: [][]*x509.Certificate{{{}}}}
			r.Header.Set("X-Correlation-ID", "SECRET_cabecera")
			if metodo == http.MethodPost {
				r.Header.Set("Origin", "https://admin.example.test")
				r.Header.Set("Sec-Fetch-Site", "same-origin")
				r.Header.Set("Sec-Fetch-Mode", "cors")
				r.Header.Set("Sec-Fetch-Dest", "empty")
			}
			accion := "consultar"
			if metodo == http.MethodPost {
				accion = "escribir"
			}
			h.ServeHTTP(w, r)
			if w.Code != 503 || reg.llamadas != 1 || reg.respuestaEmitida || len(pool.eventos) != 0 || strings.Contains(w.Body.String(), "SECRET") {
				t.Fatal("sin_append_nominal_o_downgrade")
			}
			orden := reg.ordenes[0]
			if orden.Datos.Resultado != "error" || orden.Datos.CorrelacionRef != ref || orden.Datos.Accion != configFronteraPrueba().Destinos[accion].Accion || orden.ResultadoContexto.Contexto.PersonaRef != actor.PersonaRef || orden.ResultadoContexto.RegistroContextoRef != evidencia.ResultadoContexto.RegistroContextoRef {
				t.Fatal("evidencia_accion_o_correlacion_sustituidas")
			}

		})
	}
}
