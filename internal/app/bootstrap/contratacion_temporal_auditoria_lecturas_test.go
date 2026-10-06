package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
	postgresct "vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/postgres"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/application"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	core "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type lectorAuditoriaLecturasCTPrueba struct {
	cerrado  bool
	err      error
	parcial  bool
	despues  func()
	contexto ports.ContextoAutorizacionAltaV3
	soporte  *soporteAltaContratacionTemporalDesarrollo
}

func (l *lectorAuditoriaLecturasCTPrueba) terminar(ctx context.Context) error {
	var err error
	l.contexto, err = l.soporte.contextoOperativoDesarrollo(ctx)
	if err != nil {
		return err
	}
	l.cerrado = true
	if l.despues != nil {
		l.despues()
	}
	return l.err
}

func (l *lectorAuditoriaLecturasCTPrueba) ConsultarReciboRespuesta(ctx context.Context, s ports.SolicitudConsultaReciboRespuesta) (ports.ReciboRespuestaConsultado, error) {
	err := l.terminar(ctx)
	if err != nil && !l.parcial {
		return ports.ReciboRespuestaConsultado{}, err
	}
	return ports.ReciboRespuestaConsultado{OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef,
		ComunicacionRef: s.ComunicacionRef, Respuesta: ports.RespuestaLlamamientoAceptada,
		JustificanteRef: "justificante:sintetico:001", ReciboRef: "recibo:sintetico:001", AuditoriaRef: "auditoria:sintetica:001",
		RegistradaEn: time.Now().UTC().Truncate(time.Microsecond), Estado: ports.EstadoRespuestaRecibidaRegistrada}, err
}

func (l *lectorAuditoriaLecturasCTPrueba) ConsultarComunicacionesExpediente(ctx context.Context, c ports.ConsultaComunicacionesExpediente) (ports.PaginaComunicacionesExpediente, error) {
	err := l.terminar(ctx)
	if err != nil && !l.parcial {
		return ports.PaginaComunicacionesExpediente{}, err
	}
	return ports.PaginaComunicacionesExpediente{ExpedienteRef: c.ExpedienteRef, Comunicaciones: []ports.ComunicacionExpediente{}}, err
}

type registradorAuditoriaLecturasCTPrueba struct {
	lector        *lectorAuditoriaLecturasCTPrueba
	ordenes       []vecports.DatosOrdenIntentoAuditoria
	fallo         error
	acuseInvalido bool
	recuperar     bool
}

func (r *registradorAuditoriaLecturasCTPrueba) AppendIntentoAuditoria(ctx context.Context, orden vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	if !r.lector.cerrado || ctx.Err() != nil {
		return vecports.AcuseIntentoAuditoria{}, errors.New("lector abierto o registro cancelado")
	}
	datos, err := orden.Datos()
	if err != nil {
		return vecports.AcuseIntentoAuditoria{}, err
	}
	r.ordenes = append(r.ordenes, datos)
	if r.fallo != nil && (!r.recuperar || len(r.ordenes) == 1) {
		return vecports.AcuseIntentoAuditoria{}, r.fallo
	}
	if r.acuseInvalido {
		return vecports.AcuseIntentoAuditoria{}, nil
	}
	return vecports.AcuseIntentoAuditoria{AuditoriaRef: "auditoria_sintetica", Secuencia: 1,
		HuellaSHA256: strings.Repeat("a", 64), CorrelacionRef: datos.Datos.CorrelacionRef, RegistradaEn: time.Now().UTC()}, nil
}

func escenarioAuditoriaLecturasCT(t *testing.T, familia string, fallo error) (context.Context, *lectorAuditoriaLecturasCTPrueba, *registradorAuditoriaLecturasCTPrueba, func(context.Context) (bool, error)) {
	t.Helper()
	soporte, _, principal := escenarioAutorizacionCoberturaDesarrolloPrueba(t)
	ruta := httpinterno.RutaConsultaReciboRespuesta
	if familia == "comunicaciones" {
		ruta = httpinterno.RutaConsultaComunicacionesExpediente
	}
	ctx, err := vecports.ConCorrelacionIncidenciasPeticion(contextoRutaCoberturaDesarrolloPrueba(soporte, principal, ruta))
	if err != nil {
		t.Fatal(err)
	}
	l := &lectorAuditoriaLecturasCTPrueba{err: fallo, soporte: soporte}
	r := &registradorAuditoriaLecturasCTPrueba{lector: l}
	cfg := configuracionAuditoriaLecturasCT{Proceso: "vec-server", Canal: "interna_corporativa"}
	if familia == "comunicaciones" {
		a, err := nuevoLectorComunicacionesAuditadoCT(l, soporte, r, cfg)
		if err != nil {
			t.Fatal(err)
		}
		return ctx, l, r, func(ctx context.Context) (bool, error) {
			p, err := a.ConsultarComunicacionesExpediente(ctx, ports.ConsultaComunicacionesExpediente{ExpedienteRef: "expediente:sintetico:001", Limite: 10})
			return p.ExpedienteRef != "" || p.Comunicaciones != nil || p.SiguienteCursor != "", err
		}
	}
	a, err := nuevoLectorReciboRespuestaAuditadoCT(l, soporte, r, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, l, r, func(ctx context.Context) (bool, error) {
		p, err := a.ConsultarReciboRespuesta(ctx, ports.SolicitudConsultaReciboRespuesta{OrganizacionRef: "organizacion:sintetica:001", ExpedienteRef: "expediente:sintetico:001", ComunicacionRef: "comunicacion:sintetica:001"})
		return p != (ports.ReciboRespuestaConsultado{}), err
	}
}

func TestAuditoriaLecturasCTIntentosNominalesTrasRetorno(t *testing.T) {
	for _, familia := range []string{"recibo", "comunicaciones"} {
		denegado, tecnico := ports.ErrConsultaReciboRespuestaDenegada, ports.ErrConsultaReciboRespuestaFallo
		accion, recurso := postgresct.AccionConsultaReciboRespuesta, "comunicacion:sintetica:001"
		if familia == "comunicaciones" {
			denegado, tecnico = ports.ErrConsultaComunicacionesExpedienteDenegada, ports.ErrConsultaComunicacionesExpedienteNoDisponible
			accion, recurso = postgresct.AccionConsultaComunicacionesExpediente, "expediente:sintetico:001"
		}
		for _, fallo := range []error{denegado, tecnico, errors.Join(denegado, tecnico), context.Canceled} {
			for _, registro := range []string{"confirmado", "fallo", "acuse_invalido", "commit_incierto"} {
				t.Run(familia+"/"+fallo.Error()+"/"+registro, func(t *testing.T) {
					ctx, lector, registrador, consultar := escenarioAuditoriaLecturasCT(t, familia, fallo)
					ctx, cancelar := context.WithCancel(ctx)
					defer cancelar()
					lector.despues = func() {
						lector.soporte.sesionOperativa = nil
						if fallo == context.Canceled {
							cancelar()
						}
					}
					switch registro {
					case "fallo":
						registrador.fallo = vecports.ErrIntentoAuditoriaConflicto
					case "acuse_invalido":
						registrador.acuseInvalido = true
					case "commit_incierto":
						registrador.fallo, registrador.recuperar = vecports.ErrIntentoAuditoriaNoDisponible, true
					}
					conDatos, err := consultar(ctx)
					if conDatos || err == nil || len(registrador.ordenes) == 0 {
						t.Fatal("datos o éxito sin intento nominal confirmado")
					}
					if registro == "fallo" || registro == "acuse_invalido" {
						if !errors.Is(err, tecnico) {
							t.Fatal("fallo del registro comunicado como denegación confirmada")
						}
					} else if !errors.Is(err, fallo) {
						t.Fatal("resultado original sustituido")
					}
					var auditado ports.FalloLecturaAuditado
					if errors.As(err, &auditado) != (registro == "confirmado" || registro == "commit_incierto") {
						t.Fatal("acuse perdido o atribuido a un intento sin confirmar")
					}
					orden := registrador.ordenes[0]
					resultado := core.ResultadoIntentoAuditoriaError
					if fallo == denegado {
						resultado = core.ResultadoIntentoAuditoriaDenegado
					}
					correlacion, _ := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
					ref, _ := correlacion.ValorCanonico()
					if orden.Datos.Resultado != resultado || orden.Datos.Accion != accion || orden.Datos.RecursoRef != recurso || orden.Datos.CorrelacionRef != ref ||
						!reflect.DeepEqual(orden.ResultadoContexto, lector.contexto.Resultado) {
						t.Fatal("acción, recurso, correlación o contexto histórico distintos del lector")
					}
					if registro == "commit_incierto" && (len(registrador.ordenes) != 2 || !reflect.DeepEqual(orden, registrador.ordenes[1])) {
						t.Fatal("recuperación con otra orden de auditoría")
					}
				})
			}
		}
	}
}

type autoridadComunicacionesLecturaCTPrueba struct {
	soporte *soporteAltaContratacionTemporalDesarrollo
}

func (a autoridadComunicacionesLecturaCTPrueba) ResolverContextoConsultaComunicacionesExpediente(ctx context.Context) error {
	_, err := a.soporte.contextoOperativoDesarrollo(ctx)
	return err
}

func TestAuditoriaLecturasCTCabeceraHTTPExigeAcuseConfirmado(t *testing.T) {
	for _, familia := range []string{"recibo", "comunicaciones"} {
		for _, estado := range []string{"denegado", "cancelado", "fallo_registrador", "acuse_invalido"} {
			t.Run(familia+"/"+estado, func(t *testing.T) {
				fallo := ports.ErrConsultaReciboRespuestaDenegada
				if familia == "comunicaciones" {
					fallo = ports.ErrConsultaComunicacionesExpedienteDenegada
				}
				ctx, l, r, _ := escenarioAuditoriaLecturasCT(t, familia, fallo)
				ctx, cancelar := context.WithCancel(ctx)
				defer cancelar()
				esperado := http.StatusForbidden
				switch estado {
				case "cancelado":
					l.despues = cancelar
					esperado = http.StatusRequestTimeout
				case "fallo_registrador":
					r.fallo = vecports.ErrIntentoAuditoriaConflicto
					esperado = http.StatusServiceUnavailable
				case "acuse_invalido":
					r.acuseInvalido = true
					esperado = http.StatusServiceUnavailable
				}
				cfg := configuracionAuditoriaLecturasCT{Proceso: "vec-server", Canal: "interna_corporativa"}
				var h http.Handler
				var err error
				url := httpinterno.RutaConsultaReciboRespuesta + "?organizacion_ref=organizacion:sintetica:001&expediente_ref=expediente:sintetico:001&comunicacion_ref=comunicacion:sintetica:001"
				if familia == "comunicaciones" {
					lector, e := nuevoLectorComunicacionesAuditadoCT(l, l.soporte, r, cfg)
					if e != nil {
						t.Fatal(e)
					}
					servicio, e := application.NuevoServicioConsultaComunicacionesExpediente(lector)
					if e != nil {
						t.Fatal(e)
					}
					h, err = httpinterno.NuevoManejadorConsultaComunicacionesExpediente(autoridadComunicacionesLecturaCTPrueba{l.soporte}, servicio)
					url = httpinterno.RutaConsultaComunicacionesExpediente + "?expediente_ref=expediente:sintetico:001"
				} else {
					lector, e := nuevoLectorReciboRespuestaAuditadoCT(l, l.soporte, r, cfg)
					if e != nil {
						t.Fatal(e)
					}
					servicio, e := application.NuevoServicioConsultaReciboRespuesta(lector)
					if e != nil {
						t.Fatal(e)
					}
					h, err = httpinterno.NuevoManejadorConsultaReciboRespuesta(servicio)
				}
				if err != nil {
					t.Fatal(err)
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil).WithContext(ctx))
				if w.Code != esperado {
					t.Fatalf("estado=%d esperado=%d", w.Code, esperado)
				}
				if (w.Header().Get("X-Audit-Ref") == "auditoria_sintetica") != (estado == "denegado" || estado == "cancelado") {
					t.Fatal("cabecera de auditoría perdida o emitida sin acuse")
				}
				correlacion := ""
				if estado == "denegado" || estado == "cancelado" {
					ref, e := vecports.ReferenciaCorrelacionAutorizacionV2DePeticion(ctx)
					if e != nil {
						t.Fatal(e)
					}
					correlacion, e = ref.ValorCanonico()
					if e != nil {
						t.Fatal(e)
					}
				} else if w.Header().Get("X-Audit-Ref") != "" {
					t.Fatal("referencia de auditoría falsa cuando falló el registro")
				}
				if w.Header().Get("X-Correlation-Ref") != correlacion {
					t.Fatal("correlación del acuse perdida o emitida sin confirmar")
				}
			})
		}
	}
}

func TestAuditoriaLecturasCTConstructorExigeRegistradorYOrigenConfigurado(t *testing.T) {
	_, l, r, _ := escenarioAuditoriaLecturasCT(t, "recibo", nil)
	for _, caso := range []struct {
		registrador vecports.RegistradorIntentosAuditoria
		cfg         configuracionAuditoriaLecturasCT
	}{
		{nil, configuracionAuditoriaLecturasCT{Proceso: "vec-server", Canal: "interna_corporativa"}},
		{(*registradorAuditoriaLecturasCTPrueba)(nil), configuracionAuditoriaLecturasCT{Proceso: "vec-server", Canal: "interna_corporativa"}},
		{r, configuracionAuditoriaLecturasCT{Canal: "interna_corporativa"}},
		{r, configuracionAuditoriaLecturasCT{Proceso: "vec-server", Canal: "administracion_privilegiada"}},
	} {
		if _, err := nuevoLectorReciboRespuestaAuditadoCT(l, l.soporte, caso.registrador, caso.cfg); err == nil {
			t.Fatal("recibo sin registrador u origen configurado")
		}
		if _, err := nuevoLectorComunicacionesAuditadoCT(l, l.soporte, caso.registrador, caso.cfg); err == nil {
			t.Fatal("comunicaciones sin registrador u origen configurado")
		}
	}
}

func TestAuditoriaLecturasCTPermitidoYAusenciaConservanConsumoSQL(t *testing.T) {
	for _, familia := range []string{"recibo", "comunicaciones"} {
		ausencia := ports.ErrReciboRespuestaNoEncontrado
		if familia == "comunicaciones" {
			ausencia = ports.ErrConsultaComunicacionesExpedienteNoEncontrado
		}
		for _, fallo := range []error{nil, ausencia} {
			ctx, _, r, consultar := escenarioAuditoriaLecturasCT(t, familia, fallo)
			conDatos, err := consultar(ctx)
			if err != fallo || conDatos != (fallo == nil) || len(r.ordenes) != 0 {
				t.Fatal("lectura autorizada duplicada o alterada")
			}
		}
	}
}

func TestAuditoriaLecturasCTRechazaContextoInvalidoYDatosParciales(t *testing.T) {
	for _, familia := range []string{"recibo", "comunicaciones"} {
		for _, condicion := range []string{"sin_contexto", "sin_correlacion", "perfil_distinto", "datos_parciales"} {
			t.Run(familia+"/"+condicion, func(t *testing.T) {
				ctx, lector, r, consultar := escenarioAuditoriaLecturasCT(t, familia, ports.ErrConsultaReciboRespuestaDenegada)
				switch condicion {
				case "sin_contexto":
					ctx, _ = vecports.ConCorrelacionIncidenciasPeticion(context.Background())
				case "sin_correlacion":
					ctx = contextoRutaCoberturaDesarrolloPrueba(lector.soporte, lector.soporte.contexto.Resultado.Contexto.Principal, httpinterno.RutaConsultaReciboRespuesta)
				case "perfil_distinto":
					lector.soporte.contextoEsperadoRegistrado = core.ResultadoContextoActorRegistradoV2{}
				case "datos_parciales":
					lector.parcial = true
				}
				conDatos, err := consultar(ctx)
				if conDatos || err == nil {
					t.Fatal("contexto o datos no confiables devueltos")
				}
				if condicion != "datos_parciales" && (lector.cerrado || len(r.ordenes) != 0) {
					t.Fatal("lectura o identidad inventada antes de acreditar contexto")
				}
				if condicion == "datos_parciales" && (len(r.ordenes) != 1 || r.ordenes[0].Datos.Resultado != core.ResultadoIntentoAuditoriaError) {
					t.Fatal("salida parcial no auditada como error")
				}
			})
		}
	}
}
