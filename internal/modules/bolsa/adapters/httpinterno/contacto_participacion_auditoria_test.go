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
	dominiobolsa "vec-diputacion-granada/internal/modules/bolsa/domain"
	"vec-diputacion-granada/internal/modules/bolsa/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
	"vec-diputacion-granada/internal/vec/pruebas"
)

// Teléfono sintético que nunca debe salir en una respuesta de error.
const telefonoContactoAuditadoHTTP = "600123456"

var ofertaContactoAuditadoHTTP = "oferta:" + strings.Repeat("d", 64)

type preparadorContactosAuditadoHTTP struct {
	q  ports.ConsultaContactosParticipacion
	qb ports.ConsultaContactosBolsa
}

func (preparadorContactosAuditadoHTTP) PrepararSolicitudRegistrarContacto(context.Context, EntradaRegistrarContactoParticipacion) (ports.SolicitudRegistrarContactoParticipacion, error) {
	return ports.SolicitudRegistrarContactoParticipacion{}, errors.New("no usado")
}
func (p preparadorContactosAuditadoHTTP) PrepararConsultaContactos(_ context.Context, _, _, cursor string, limite int) (ports.ConsultaContactosParticipacion, error) {
	q := p.q
	q.Cursor, q.Limite = cursor, limite
	return q, nil
}
func (p preparadorContactosAuditadoHTTP) PrepararConsultaContactosBolsa(_ context.Context, _, cursor string, limite int) (ports.ConsultaContactosBolsa, error) {
	q := p.qb
	q.Cursor, q.Limite = cursor, limite
	return q, nil
}

type operadorContactosAuditadoHTTP struct {
	pagina ports.PaginaContactosParticipacion
	err    error
}

func (*operadorContactosAuditadoHTTP) RegistrarContactoParticipacion(context.Context, ports.SolicitudRegistrarContactoParticipacion) (ports.RegistroContactoParticipacion, error) {
	return ports.RegistroContactoParticipacion{}, errors.New("no usado")
}
func (o *operadorContactosAuditadoHTTP) ListarContactosParticipacion(context.Context, ports.ConsultaContactosParticipacion) (ports.PaginaContactosParticipacion, error) {
	return o.pagina, o.err
}
func (o *operadorContactosAuditadoHTTP) ListarContactosBolsa(context.Context, ports.ConsultaContactosBolsa) (ports.PaginaContactosParticipacion, error) {
	return o.pagina, o.err
}

type operadorContactosConIntentosHTTP struct {
	*operadorContactosAuditadoHTTP
	intentos *operadorIntentosPrueba
}

func (o operadorContactosConIntentosHTTP) EstadoIntentosTelefonicos(ctx context.Context, ll string, c []dominiobolsa.ContactoParticipacion, completo bool) (ports.EstadoIntentosContacto, error) {
	return o.intentos.EstadoIntentosTelefonicos(ctx, ll, c, completo)
}

func consultasContactosAuditadasHTTP(t *testing.T, ctx context.Context) (preparadorContactosAuditadoHTTP, string) {
	t.Helper()
	correlacion, err := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	actor, vinculo, err := pruebas.NuevoContextoRegistradoYVinculoV2(time.Now().UTC(), "per_0123456789abcdefghijkl", "prf_0123456789abcdefghijkl", dominiovec.AuthMethodCertificate, dominiovec.AuthAssuranceHigh)
	if err != nil {
		t.Fatal(err)
	}
	motivo := dominiovec.ReferenciaEntradaCatalogo{CatalogoID: "motivos_contacto_participacion_bolsa_v2", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("b", 64), EntradaClave: "motivo_0123456789abcdef0123456789abcdef"}
	ref, _ := correlacion.ValorCanonico()
	return preparadorContactosAuditadoHTTP{
		q:  ports.ConsultaContactosParticipacion{Vinculo: vinculo, ResultadoContexto: actor, BolsaRef: "bolsa:1", ParticipacionRef: "participacion:1", Correlacion: correlacion, MotivoAutorizacion: motivo},
		qb: ports.ConsultaContactosBolsa{Vinculo: vinculo, ResultadoContexto: actor, BolsaRef: "bolsa:1", Correlacion: correlacion, MotivoAutorizacion: motivo},
	}, ref
}

func contactoValidoAuditadoHTTP() dominiobolsa.ContactoParticipacion {
	return dominiobolsa.ContactoParticipacion{ContactoRef: "contacto:" + strings.Repeat("e", 64), BolsaRef: "bolsa:1", ParticipacionRef: "participacion:1",
		LlamamientoRef: "llamamiento:1", Canal: dominiobolsa.CanalContactoTelefono, Instante: time.Date(2026, 10, 5, 8, 0, 0, 0, time.UTC),
		Actor: "per_0123456789abcdefghijkl", Resultado: dominiobolsa.ResultadoContactoNoContesta, Anotacion: "Sin respuesta"}
}

func pedirContactosAuditados(ctx context.Context, h http.Handler, destino string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, destino, nil).WithContext(ctx)
	r.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestContactosRRHHCabecerasExigenAcuseNominal(t *testing.T) {
	destinos := []string{rutaContactosPrueba, RutaContactosOferta + "?bolsa_ref=bolsa%3A1&oferta_ref=" + ofertaContactoAuditadoHTTP}
	for _, caso := range []string{"exito", "denegado", "error", "parcial", "acuse_invalido", "registro_fallido"} {
		for _, destino := range destinos {
			t.Run(caso+destino[:12], func(t *testing.T) {
				ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				preparador, correlacion := consultasContactosAuditadasHTTP(t, ctx)
				o := &operadorContactosAuditadoHTTP{err: dominiovec.ErrAutorizacionDenegada}
				esperado := http.StatusForbidden
				switch caso {
				case "exito":
					o.err, esperado = nil, http.StatusOK
				case "error":
					o.err, esperado = ports.ErrContactoParticipacionNoDisponible, http.StatusServiceUnavailable
				case "parcial":
					contacto := contactoValidoAuditadoHTTP()
					contacto.Anotacion = "Llamar al " + telefonoContactoAuditadoHTTP
					o.pagina, esperado = ports.PaginaContactosParticipacion{Contactos: []dominiobolsa.ContactoParticipacion{contacto}}, http.StatusServiceUnavailable
				case "acuse_invalido", "registro_fallido":
					esperado = http.StatusServiceUnavailable
				}
				r := &registradorDocumentalesHTTPPrueba{t: t, fallo: caso == "registro_fallido", invalido: caso == "acuse_invalido"}
				auditados, err := application.NuevosContactosRRHHAuditados(o, r, "vec-bolsa-prueba")
				if err != nil {
					t.Fatal(err)
				}
				h, err := NuevoHandlerContactoParticipacion(preparador, auditados)
				if err != nil {
					t.Fatal(err)
				}
				w := pedirContactosAuditados(ctx, h, destino)
				cuerpo := w.Body.String()
				if w.Code != esperado || strings.Contains(cuerpo, telefonoContactoAuditadoHTTP) || strings.Contains(cuerpo, "per_") || w.Header().Get("Set-Cookie") != "" {
					t.Fatalf("respuesta no cerrada: %d %s", w.Code, cuerpo)
				}
				confirmado := caso == "denegado" || caso == "error" || caso == "parcial"
				if confirmado {
					if w.Header().Get("X-Audit-Ref") != "aud_documental_http_sintetica" || w.Header().Get("X-Correlation-Ref") != correlacion || len(r.ordenes) != 1 {
						t.Fatal("acuse o correlación exacta perdidos")
					}
				} else if w.Header().Get("X-Audit-Ref") != "" || w.Header().Get("X-Correlation-Ref") != "" {
					t.Fatal("acuse falso")
				}
				if caso == "exito" && len(r.ordenes) != 0 {
					t.Fatal("lectura correcta registrada como intento fallido")
				}
			})
		}
	}
}

func TestContactosRRHHAuditadosConservanControlDeIntentos(t *testing.T) {
	ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	preparador, _ := consultasContactosAuditadasHTTP(t, ctx)
	contacto := contactoValidoAuditadoHTTP()
	intentos := &operadorIntentosPrueba{}
	o := operadorContactosConIntentosHTTP{&operadorContactosAuditadoHTTP{pagina: ports.PaginaContactosParticipacion{Contactos: []dominiobolsa.ContactoParticipacion{contacto}, CursorSiguiente: contacto.ContactoRef}}, intentos}
	auditados, err := application.NuevosContactosRRHHAuditados(o, &registradorDocumentalesHTTPPrueba{t: t}, "vec-bolsa-prueba")
	if err != nil {
		t.Fatal(err)
	}
	h, err := NuevoHandlerContactoParticipacion(preparador, auditados)
	if err != nil {
		t.Fatal(err)
	}
	w := pedirContactosAuditados(ctx, h, rutaContactosPrueba+"?llamamiento_ref=llamamiento:1")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"configurado":true`) || intentos.llamamiento != "llamamiento:1" {
		t.Fatalf("control de intentos perdido tras el decorador: %d %s", w.Code, w.Body.String())
	}
}

type resolutorContactosAuditadoHTTP struct{}

func (resolutorContactosAuditadoHTTP) ResolverContextoSituacionParticipacion(context.Context, dominiovec.ContextoActor, string, string) (ports.ContextoSituacionParticipacionResuelto, error) {
	return ports.ContextoSituacionParticipacionResuelto{UnidadRef: "unidad:seleccion", AmbitoRef: "ambito:bolsa"}, nil
}
func (resolutorContactosAuditadoHTTP) ResolverContextoContactosBolsa(context.Context, dominiovec.ContextoActor, string) (ports.ContextoSituacionParticipacionResuelto, error) {
	return ports.ContextoSituacionParticipacionResuelto{UnidadRef: "unidad:seleccion", AmbitoRef: "ambito:bolsa"}, nil
}

type repositorioContactosAuditadoHTTP struct {
	ports.RepositorioContactoParticipacion
	lecturas int
}

func (r *repositorioContactosAuditadoHTTP) ListarContactosParticipacion(context.Context, ports.ConsultaContactosParticipacion) (ports.PaginaContactosParticipacion, error) {
	r.lecturas++
	return ports.PaginaContactosParticipacion{}, nil
}
func (r *repositorioContactosAuditadoHTTP) ListarContactosBolsa(context.Context, ports.ConsultaContactosBolsa) (ports.PaginaContactosParticipacion, error) {
	r.lecturas++
	return ports.PaginaContactosParticipacion{}, nil
}

func TestContactosRRHHConservanDenegacionDelContratoEmisorV3(t *testing.T) {
	destinos := []string{rutaContactosPrueba, RutaContactosOferta + "?bolsa_ref=bolsa%3A1&oferta_ref=" + ofertaContactoAuditadoHTTP}
	for _, denegado := range []bool{false, true} {
		for _, destino := range destinos {
			ctx, err := vecports.ConCorrelacionIncidenciasPeticion(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			preparador, correlacion := consultasContactosAuditadasHTTP(t, ctx)
			// Contrato del emisor común: el marcador solo acompaña a una
			// denegación nominal ya registrada; un fallo técnico no lo lleva.
			e := &emisorDocumentalesHTTPPrueba{err: errors.New("emision material: fallo privado sintetico")}
			esperado, resultado := http.StatusServiceUnavailable, dominiovec.ResultadoIntentoAuditoriaError
			if denegado {
				e.err = errors.Join(e.err, vecports.ErrDenegacionExplicitaAutorizacionLigadaV3)
				esperado, resultado = http.StatusForbidden, dominiovec.ResultadoIntentoAuditoriaDenegado
			}
			repo := &repositorioContactosAuditadoHTTP{}
			servicio, err := application.NuevoServicioContactoParticipacion(resolutorContactosAuditadoHTTP{}, e, repo)
			if err != nil {
				t.Fatal(err)
			}
			r := &registradorDocumentalesHTTPPrueba{t: t}
			auditados, err := application.NuevosContactosRRHHAuditados(servicio, r, "vec-bolsa-prueba")
			if err != nil {
				t.Fatal(err)
			}
			h, err := NuevoHandlerContactoParticipacion(preparador, auditados)
			if err != nil {
				t.Fatal(err)
			}
			w := pedirContactosAuditados(ctx, h, destino)
			if w.Code != esperado || e.llamadas != 1 || repo.lecturas != 0 || len(r.ordenes) != 1 || r.ordenes[0].Datos.Resultado != resultado ||
				w.Header().Get("X-Audit-Ref") != "aud_documental_http_sintetica" || w.Header().Get("X-Correlation-Ref") != correlacion ||
				strings.Contains(w.Body.String(), "privado") {
				t.Fatalf("contrato emisor perdido en %s: %d %s", destino, w.Code, w.Body.String())
			}
		}
	}
}
