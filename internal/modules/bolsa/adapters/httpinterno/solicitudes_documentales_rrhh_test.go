package httpinterno

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/bolsa/application"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

type consultorDocumentalesPrueba struct {
	items    []ports.SolicitudDocumentalPendienteRRHH
	err      error
	llamadas int
}

type contextoEmisionDocumentalHTTPPrueba struct{}

func (contextoEmisionDocumentalHTTPPrueba) ResolverContextoSituacionParticipacion(context.Context, dominiovec.ContextoActor, string, string) (ports.ContextoSituacionParticipacionResuelto, error) {
	return ports.ContextoSituacionParticipacionResuelto{UnidadRef: "unidad:seleccion", AmbitoRef: "ambito:bolsa"}, nil
}

type repositorioEmisionDocumentalHTTPPrueba struct {
	ports.RepositorioSituacionParticipacion
	lecturas int
}

func (*repositorioEmisionDocumentalHTTPPrueba) ParticipacionPerteneceABolsa(context.Context, string, string) (bool, error) {
	return true, nil
}

func (r *repositorioEmisionDocumentalHTTPPrueba) ListarSolicitudesDocumentalesPendientes(context.Context, string, string, string, time.Time, vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) ([]ports.SolicitudDocumentalPendienteRRHH, error) {
	r.lecturas++
	return nil, nil
}

type emisorDocumentalesHTTPPrueba struct {
	err      error
	llamadas int
}

func (e *emisorDocumentalesHTTPPrueba) EmitirMaterialAutorizacionAtestadaV3(context.Context, dominiovec.SolicitudAutorizacionLigadaV3, dominiovec.ResultadoContextoActorRegistradoV2) (dominiovec.DecisionAutorizacionLigadaV3, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, vecports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	return dominiovec.DecisionAutorizacionLigadaV3{}, vecports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, e.err
}

func TestSolicitudesDocumentalesRRHHConservaDenegacionDelContratoEmisorV3(t *testing.T) {
	for _, denegado := range []bool{false, true} {
		ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
		if err != nil {
			t.Fatal(err)
		}
		actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(time.Now().UTC(), "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
		if err != nil {
			t.Fatal(err)
		}
		q := ports.SolicitudCambiarSituacionParticipacion{ResultadoContexto: actor, Vinculo: vinculo, BolsaRef: "bolsa:01", ParticipacionRef: "participacion:01", Destino: "disponible", Motivo: "consulta", ClaveIdempotencia: "consulta", Correlacion: correlacion,
			MotivoAutorizacion: dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("b", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"}}
		// Es el contrato del emisor común: error privado más marcador sólo
		// cuando ya ha validado la denegación nominal. Un fallo técnico carece
		// de ese marcador y no debe convertirse en autorización denegada.
		e := &emisorDocumentalesHTTPPrueba{err: errors.New("emision material: fallo privado sintetico")}
		esperado := http.StatusServiceUnavailable
		resultado := dominiovec.ResultadoIntentoAuditoriaError
		if denegado {
			e.err = errors.Join(e.err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3)
			esperado = http.StatusForbidden
			resultado = dominiovec.ResultadoIntentoAuditoriaDenegado
		}
		repo := &repositorioEmisionDocumentalHTTPPrueba{}
		servicio, err := application.NuevoServicioSituacionParticipacion(contextoEmisionDocumentalHTTPPrueba{}, e, repo, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		r := &registradorDocumentalesHTTPPrueba{t: t}
		auditada, err := application.NuevaConsultaSolicitudesDocumentalesAuditada(servicio, r, "vec-bolsa-prueba")
		if err != nil {
			t.Fatal(err)
		}
		h, err := NuevoHandlerSolicitudesDocumentalesRRHH(preparadorDocumentalesAuditadoHTTP{q: q}, auditada)
		if err != nil {
			t.Fatal(err)
		}
		peticion := httptest.NewRequest(http.MethodGet, RutaSolicitudesDocumentalesPendientesRRHH+"?bolsa_ref=bolsa%3A01&participacion_ref=participacion%3A01", nil).WithContext(ctx)
		peticion.Header.Set("Accept", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, peticion)
		ref, _ := correlacion.ValorCanonico()
		if w.Code != esperado || e.llamadas != 1 || repo.lecturas != 0 || len(r.ordenes) != 1 || r.ordenes[0].Datos.Resultado != resultado ||
			w.Header().Get("X-Audit-Ref") != "aud_documental_http_sintetica" || w.Header().Get("X-Correlation-Ref") != ref || strings.Contains(w.Body.String(), "privado") {
			t.Fatalf("contrato emisor perdido: %d %s", w.Code, w.Body.String())
		}
	}
}

type preparadorDocumentalesAuditadoHTTP struct {
	q ports.SolicitudCambiarSituacionParticipacion
}

func (p preparadorDocumentalesAuditadoHTTP) PrepararSolicitudCambiarSituacion(context.Context, EntradaCambiarSituacionParticipacion) (ports.SolicitudCambiarSituacionParticipacion, error) {
	return p.q, nil
}

type registradorDocumentalesHTTPPrueba struct {
	t        *testing.T
	fallo    bool
	invalido bool
	ordenes  []vecports.DatosOrdenIntentoAuditoria
}

func (r *registradorDocumentalesHTTPPrueba) AppendIntentoAuditoria(_ context.Context, o vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	r.t.Helper()
	d, err := o.Datos()
	if err != nil {
		r.t.Fatal(err)
	}
	r.ordenes = append(r.ordenes, d)
	if r.fallo {
		return vecports.AcuseIntentoAuditoria{}, vecports.ErrIntentoAuditoriaNoDisponible
	}
	if r.invalido {
		return vecports.AcuseIntentoAuditoria{}, nil
	}
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "aud_documental_http_sintetica", Secuencia: 1,
		HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: d.Datos.CorrelacionRef, RegistradaEn: time.Now().UTC()}, nil
}

func TestSolicitudesDocumentalesRRHHCabecerasExigenAcuseNominal(t *testing.T) {
	for _, caso := range []string{"denegado", "error", "parcial", "acuse_invalido", "registro_fallido", "sin_contexto"} {
		t.Run(caso, func(t *testing.T) {
			ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
			if err != nil {
				t.Fatal(err)
			}
			actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(time.Now().UTC(), "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
			if err != nil {
				t.Fatal(err)
			}
			q := ports.SolicitudCambiarSituacionParticipacion{ResultadoContexto: actor, Vinculo: vinculo,
				BolsaRef: "bolsa:01", ParticipacionRef: "participacion:01", Destino: "disponible", Motivo: "consulta", ClaveIdempotencia: "consulta", Correlacion: correlacion,
				MotivoAutorizacion: dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("b", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"}}
			c := &consultorDocumentalesPrueba{err: dominiovec.ErrAutorizacionDenegada}
			r := &registradorDocumentalesHTTPPrueba{t: t, fallo: caso == "registro_fallido", invalido: caso == "acuse_invalido"}
			esperado := http.StatusForbidden
			if caso != "denegado" {
				esperado = http.StatusServiceUnavailable
			}
			if caso == "error" {
				c.err = ports.ErrSituacionParticipacionNoDisponible
			}
			if caso == "parcial" {
				c.items = []ports.SolicitudDocumentalPendienteRRHH{{DocumentoRef: "documento:no_publicar"}}
			}
			if caso == "sin_contexto" {
				q.ResultadoContexto = dominiovec.ResultadoContextoActorRegistradoV2{}
			}
			a, err := application.NuevaConsultaSolicitudesDocumentalesAuditada(c, r, "vec-bolsa-prueba")
			if err != nil {
				t.Fatal(err)
			}
			h, err := NuevoHandlerSolicitudesDocumentalesRRHH(preparadorDocumentalesAuditadoHTTP{q: q}, a)
			if err != nil {
				t.Fatal(err)
			}
			peticion := httptest.NewRequest(http.MethodGet, RutaSolicitudesDocumentalesPendientesRRHH+"?bolsa_ref=bolsa%3A01&participacion_ref=participacion%3A01", nil).WithContext(ctx)
			peticion.Header.Set("Accept", "application/json")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, peticion)
			if w.Code != esperado || strings.Contains(w.Body.String(), "documento:no_publicar") || strings.Contains(w.Body.String(), "per_") || w.Header().Get("Set-Cookie") != "" {
				t.Fatalf("resultadoHTTP no cerrado: %d %s", w.Code, w.Body.String())
			}
			confirmado := caso == "denegado" || caso == "error" || caso == "parcial"
			if confirmado {
				ref, _ := correlacion.ValorCanonico()
				if w.Header().Get("X-Audit-Ref") != "aud_documental_http_sintetica" || w.Header().Get("X-Correlation-Ref") != ref || len(r.ordenes) != 1 {
					t.Fatal("acuse o correlación exacta perdidos")
				}
			} else if w.Header().Get("X-Audit-Ref") != "" || w.Header().Get("X-Correlation-Ref") != "" {
				t.Fatal("acuse falso")
			}
		})
	}
}

func (c *consultorDocumentalesPrueba) ListarSolicitudesDocumentalesRRHH(_ context.Context, q ports.SolicitudCambiarSituacionParticipacion) ([]ports.SolicitudDocumentalPendienteRRHH, error) {
	c.llamadas++
	if q.BolsaRef != "bolsa:01" || q.ParticipacionRef != "participacion:01" {
		return nil, dominiovec.ErrAutorizacionDenegada
	}
	return c.items, c.err
}

func TestSolicitudesDocumentalesRRHHFiltraYSoloMuestraPendientes(t *testing.T) {
	consulta := &consultorDocumentalesPrueba{items: []ports.SolicitudDocumentalPendienteRRHH{{
		SolicitudRef: "solicitud-documental:" + strings.Repeat("a", 64), Version: 1, ContenidoSHA256: strings.Repeat("b", 64),
		DocumentoRef: "documento:parte-1", DocumentoSHA256: strings.Repeat("c", 64), FechaFinCausa: "2026-10-02",
		Estado: "pendiente_rrhh", ReciboRef: "recibo:solicitud-documental:" + strings.Repeat("d", 64), RegistradaEn: time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC),
	}}}
	h, err := NuevoHandlerSolicitudesDocumentalesRRHH(preparadorSituacionHTTPPrueba{}, consulta)
	if err != nil {
		t.Fatal(err)
	}
	ruta := RutaSolicitudesDocumentalesPendientesRRHH + "?bolsa_ref=bolsa%3A01&participacion_ref=participacion%3A01"
	r := httptest.NewRequest(http.MethodGet, ruta, nil)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || consulta.llamadas != 1 || !strings.Contains(w.Body.String(), `"esquema":"vec.bolsa.rrhh.solicitudes_documentales.v1"`) ||
		!strings.Contains(w.Body.String(), `"documento_ref":"documento:parte-1"`) || strings.Contains(w.Body.String(), `"candidato_ref"`) {
		t.Fatalf("lectura focal: %d %s", w.Code, w.Body.String())
	}
	for _, rutaMala := range []string{RutaSolicitudesDocumentalesPendientesRRHH + "?bolsa_ref=bolsa%3A01", ruta + "&candidato_ref=can_ajeno", ruta + "&bolsa_ref=bolsa%3A02"} {
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, rutaMala, nil)
		r.Header.Set("Accept", "application/json")
		h.ServeHTTP(w, r)
		if w.Code != http.StatusBadRequest || consulta.llamadas != 1 {
			t.Fatalf("filtro alterado admitido: %d", w.Code)
		}
	}
	consulta.items[0].FechaFinCausa = ""
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, ruta, nil)
	r.Header.Set("Accept", "application/json")
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"fecha_fin_causa":null`) {
		t.Fatalf("fecha no acreditada inventada: %d %s", w.Code, w.Body.String())
	}
	consulta.err = dominiovec.ErrAutorizacionDenegada
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, ruta, nil)
	r.Header.Set("Accept", "application/json")
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || strings.Contains(w.Body.String(), "documento:parte-1") {
		t.Fatalf("denegación con datos: %d %s", w.Code, w.Body.String())
	}
}
