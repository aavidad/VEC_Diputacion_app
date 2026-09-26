package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vd "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type relojSeguimientoFijo struct{ t time.Time }

func (r relojSeguimientoFijo) Ahora() time.Time { return r.t }

type contextosSeguimientoDoble struct {
	c ports.ContextoAutorizacionAltaV3
}

func (d contextosSeguimientoDoble) ResolverContextoAutorizacionAltaV3(context.Context, ports.SolicitudResolverContextoAutorizacionAltaV3) (ports.ContextoAutorizacionAltaV3, error) {
	return d.c, nil
}

type reglasSeguimientoDoble struct {
	causas      []ports.CausaCese
	condiciones []string
	fase        domain.ClaveFase
	motivos     []ports.OpcionMotivoSeguimiento
	ahora       time.Time
}

func (r reglasSeguimientoDoble) politica(ref string) ports.PoliticaOperacionSeguimiento {
	return ports.PoliticaOperacionSeguimiento{DefinicionRef: ref, DefinicionVersion: 1, DefinicionHuellaSHA256: strings.Repeat("c", 64),
		MotivoAutorizacion: vd.ReferenciaEntradaCatalogo{CatalogoID: "motivos_seguimiento_ct", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("d", 64), EntradaClave: "motivo_" + strings.Repeat("1", 32)},
		EvaluadaEn: r.ahora, ValidaHasta: r.ahora.Add(time.Minute)}
}
func (r reglasSeguimientoDoble) CausasCese(context.Context, time.Time) ([]ports.CausaCese, ports.PoliticaOperacionSeguimiento, error) {
	return r.causas, r.politica("causas_cese_contratacion_temporal"), nil
}
func (r reglasSeguimientoDoble) ReglaCierre(context.Context, time.Time) (ports.ReglaCierreExpediente, ports.PoliticaOperacionSeguimiento, error) {
	return ports.ReglaCierreExpediente{Condiciones: r.condiciones}, r.politica("vec.contratacion_temporal.reglas"), nil
}
func (r reglasSeguimientoDoble) ReglaModificacion(context.Context, time.Time) (ports.ReglaModificacionNombramiento, ports.PoliticaOperacionSeguimiento, error) {
	return ports.ReglaModificacionNombramiento{FaseRetorno: r.fase, Motivos: r.motivos}, r.politica("vec.contratacion_temporal.reglas"), nil
}

type sellosSeguimientoDoble struct{}

func (sellosSeguimientoDoble) coleccion(operacion string, ambito bool) ports.ColeccionSellosHMAC {
	a, h, _ := ports.DominiosHMACOperacionSeguimiento(operacion)
	d := h
	if ambito {
		d = a
	}
	c, _ := ports.NuevaColeccionSellosHMAC("hmac-sha256:"+d+"/v1:"+strings.Repeat("a", 64), nil)
	return c
}
func (s sellosSeguimientoDoble) SellarAmbitoOperacionSeguimiento(_ context.Context, operacion string, _ ports.SolicitudSellarAmbitoIdempotencia) (ports.ColeccionSellosHMAC, error) {
	return s.coleccion(operacion, true), nil
}
func (s sellosSeguimientoDoble) DerivarHuellaOperacionSeguimiento(_ context.Context, operacion string, _ []byte) (ports.ColeccionSellosHMAC, error) {
	return s.coleccion(operacion, false), nil
}

type referenciasSeguimientoDoble struct{}

func (referenciasSeguimientoDoble) GenerarReferenciasSeguimiento(context.Context) (ports.ReferenciasEfectoSeguimiento, error) {
	return ports.ReferenciasEfectoSeguimiento{ReservaRef: "reserva:prueba", ReciboRef: "recibo:prueba", EventoRef: "evento:prueba"}, nil
}

type repositorioSeguimientoDoble struct {
	expediente    domain.Expediente
	preparaciones int
	ordenes       []ports.OrdenConfirmarOperacionSeguimiento
}

func (r *repositorioSeguimientoDoble) PrepararOperacionSeguimiento(_ context.Context, operacion string, _ any, s ports.SellosOperacionSeguimiento, refs ports.ReferenciasEfectoSeguimiento) (ports.PreparacionOperacionSeguimiento, error) {
	r.preparaciones++
	a, _ := s.Ambitos.Datos()
	h, _ := s.Huellas.Datos()
	return ports.PreparacionOperacionSeguimiento{Expediente: r.expediente.Clonar(), Referencias: refs, AmbitoIdempotenciaHMAC: a.Activo.Valor,
		HuellaPeticionHMAC: h.Activo.Valor, IncorporacionRef: "ref:incorporacion:prueba", CeseReciboRef: "recibo:cese:prueba",
		AceptacionRef: "resolucion:aceptacion:prueba"}, nil
}
func (r *repositorioSeguimientoDoble) ConfirmarOperacionSeguimiento(_ context.Context, o ports.OrdenConfirmarOperacionSeguimiento) (ports.ReciboOperacionSeguimiento, error) {
	r.ordenes = append(r.ordenes, o)
	return ports.ReciboOperacionSeguimiento{Operacion: o.Operacion, OrganizacionRef: o.Siguiente.OrganizacionRef, ExpedienteRef: o.Siguiente.Referencia,
		VersionAnterior: o.Preparacion.Expediente.Version, VersionResultante: o.Siguiente.Version, FaseResultante: o.Siguiente.FaseActual,
		EstadoResultante: o.Siguiente.EstadoActual, ReciboRef: "recibo:prueba", AuditoriaRef: "aud_v3_" + strings.Repeat("0", 32),
		EventoRef: "evento:prueba", ActorRef: "per_prueba", RegistradaEn: o.InstanteEfecto}, nil
}
func (r *repositorioSeguimientoDoble) ConsultarEstadoSeguimiento(context.Context, string, string) (ports.EstadoSeguimientoExpediente, error) {
	return ports.EstadoSeguimientoExpediente{}, nil
}

type autorizadorSeguimientoDoble struct {
	solicitudes []ports.SolicitudAutorizarOperacionSeguimiento
}

func (a *autorizadorSeguimientoDoble) AutorizarOperacionSeguimiento(_ context.Context, s ports.SolicitudAutorizarOperacionSeguimiento) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.solicitudes = append(a.solicitudes, s)
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
}

type costeSeguimientoDoble struct{ centimos int64 }

func (c costeSeguimientoDoble) CalcularCosteModificacion(context.Context, domain.Expediente, domain.PeriodoPrevisto, domain.JornadaDiezmilesimas) (domain.Importe, string, error) {
	return domain.Importe{Moneda: "EUR", Centimos: c.centimos}, "autoridad:ct:desarrollo:calculo-coste", nil
}

type escenarioSeguimiento struct {
	servicio    *ServicioOperacionesSeguimiento
	repo        *repositorioSeguimientoDoble
	autorizador *autorizadorSeguimientoDoble
	canal       ContextoCanalSeguimiento
	reglas      *reglasSeguimientoDoble
}

func nuevoEscenarioSeguimiento(t *testing.T, coste int64) escenarioSeguimiento {
	t.Helper()
	contenido, err := os.ReadFile("../domain/testdata/expediente_nombramiento_v7.json")
	if err != nil {
		t.Fatal(err)
	}
	var e domain.Expediente
	dec := json.NewDecoder(bytes.NewReader(contenido))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&e); err != nil {
		t.Fatal(err)
	}
	ahora := e.ActualizadoEn.Add(time.Hour)
	contexto := contextoAutorizacionAltaV3Prueba(t, ahora)
	v, err := contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	reglas := &reglasSeguimientoDoble{ahora: ahora, condiciones: []string{domain.CondicionCeseRegistrado},
		causas: []ports.CausaCese{{Clave: "fin_sustitucion", Etiqueta: "Fin de la sustitución", JustificanteTipo: "comunicacion_reincorporacion"}},
		fase:   domain.FaseFiscalizacion, motivos: []ports.OpcionMotivoSeguimiento{{Clave: "cambio_jornada", Etiqueta: "Cambio de jornada"}}}
	repo := &repositorioSeguimientoDoble{expediente: e}
	autorizador := &autorizadorSeguimientoDoble{}
	s, err := NuevoServicioOperacionesSeguimiento(DependenciasOperacionesSeguimiento{Contextos: contextosSeguimientoDoble{contexto}, Sellos: sellosSeguimientoDoble{},
		Repositorio: repo, Reglas: reglas, Autorizador: autorizador, Referencias: referenciasSeguimientoDoble{}, Coste: costeSeguimientoDoble{coste},
		Lector: repo, Reloj: relojSeguimientoFijo{ahora}})
	if err != nil {
		t.Fatal(err)
	}
	return escenarioSeguimiento{servicio: s, repo: repo, autorizador: autorizador, reglas: reglas,
		canal: ContextoCanalSeguimiento{AutenticacionRef: v.AutenticacionRef, SesionRef: v.SesionRef, PerfilRef: v.PerfilActivoRef, OrganizacionRef: e.OrganizacionRef}}
}

func TestCeseTomaDelCatalogoLaCausaYElJustificante(t *testing.T) {
	e := nuevoEscenarioSeguimiento(t, 1)
	sol := SolicitudRegistrarCese{Canal: e.canal, ExpedienteRef: e.repo.expediente.Referencia, VersionEsperada: 7,
		ClaveIdempotencia: "11111111-1111-4111-8111-111111111111", CausaClave: "fin_sustitucion", FechaEfecto: time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC),
		JustificanteRef: "documento:justificante", JustificanteSHA256: strings.Repeat("b", 64)}
	recibo, err := e.servicio.RegistrarCese(context.Background(), sol)
	if err != nil || recibo.VersionResultante != 8 {
		t.Fatalf("cese: %+v %v", recibo, err)
	}
	orden := e.repo.ordenes[0]
	if orden.Contexto.Atributos["justificante_tipo"] != "comunicacion_reincorporacion" || orden.Contexto.Atributos["causa_clave"] != "fin_sustitucion" ||
		orden.Contexto.Atributos["incorporacion_ref"] != "ref:incorporacion:prueba" || e.autorizador.solicitudes[0].Audiencia != ports.AudienciaConsumoCeseV1 {
		t.Fatalf("contexto autorizado inesperado: %+v", orden.Contexto.Atributos)
	}
	sol.CausaClave = "causa_inexistente"
	if _, err := e.servicio.RegistrarCese(context.Background(), sol); !errors.Is(err, ErrSolicitudSeguimientoInvalida) {
		t.Fatalf("causa fuera del catálogo: %v", err)
	}
}

func TestCierreAplicaLasCondicionesDeLaRegla(t *testing.T) {
	e := nuevoEscenarioSeguimiento(t, 1)
	cesado, err := e.repo.expediente.RegistrarCese(7, domain.DatosCese{CausaClave: "renuncia", FechaEfecto: time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC),
		JustificanteTipo: "escrito_renuncia", JustificanteRef: "documento:renuncia", JustificanteSHA256: strings.Repeat("b", 64)},
		domain.DatosActuacion{AccionClave: domain.AccionCesarNombramiento, ActorRef: "per_prueba", UnidadRef: e.repo.expediente.Asignacion.UnidadRef,
			ReciboRef: "recibo:cese:prueba", RealizadaEn: e.repo.expediente.ActualizadoEn.Add(time.Minute), FaseDestino: domain.FaseNombramiento,
			EstadoDestino: domain.EstadoEnCurso, DocumentosRef: []string{"documento:renuncia"}})
	if err != nil {
		t.Fatal(err)
	}
	e.repo.expediente = cesado
	sol := SolicitudCerrarExpediente{Canal: e.canal, ExpedienteRef: cesado.Referencia, VersionEsperada: 8, ClaveIdempotencia: "22222222-2222-4222-8222-222222222222"}
	recibo, err := e.servicio.CerrarExpediente(context.Background(), sol)
	if err != nil || recibo.EstadoResultante != domain.EstadoCompletado {
		t.Fatalf("cierre solo con cese: %+v %v", recibo, err)
	}
	e.reglas.condiciones = []string{domain.CondicionCeseRegistrado, domain.CondicionGINPIXConfirmado}
	if _, err := e.servicio.CerrarExpediente(context.Background(), sol); !errors.Is(err, ErrSolicitudSeguimientoInvalida) {
		t.Fatalf("la regla exige GINPIX: %v", err)
	}
	fecha := time.Date(2027, 2, 2, 0, 0, 0, 0, time.UTC)
	sol.GINPIXNumero, sol.GINPIXConfirmadaEn = "GX-1", &fecha
	if _, err := e.servicio.CerrarExpediente(context.Background(), sol); err != nil {
		t.Fatalf("cierre con GINPIX: %v", err)
	}
	if got := e.repo.ordenes[len(e.repo.ordenes)-1].Contexto.Atributos["condiciones"]; got != "cese_registrado,ginpix_confirmado" {
		t.Fatalf("condiciones autorizadas: %q", got)
	}
}

func TestModificacionUsaElCosteDeLaFuenteYLaFaseDeLaRegla(t *testing.T) {
	e := nuevoEscenarioSeguimiento(t, 2000000)
	sol := SolicitudModificarTrasNombramiento{Canal: e.canal, ExpedienteRef: e.repo.expediente.Referencia, VersionEsperada: 7,
		ClaveIdempotencia: "33333333-3333-4333-8333-333333333333", MotivoClave: "cambio_jornada", Periodo: e.repo.expediente.Analisis.Periodo,
		Jornada: 5000, Observaciones: "Reducción de jornada"}
	recibo, err := e.servicio.ModificarTrasNombramiento(context.Background(), sol)
	if err != nil || recibo.FaseResultante != domain.FaseFiscalizacion {
		t.Fatalf("modificación: %+v %v", recibo, err)
	}
	orden := e.repo.ordenes[0]
	if orden.Contexto.Atributos["coste_centimos"] != "2000000" || orden.Siguiente.Analisis.CostePrevisto.Centimos != 2000000 ||
		orden.Contexto.Atributos["fase_retorno"] != "fiscalizacion" {
		t.Fatalf("coste o fase inesperados: %+v", orden.Contexto.Atributos)
	}
	sol.MotivoClave = "motivo_no_admitido"
	if _, err := e.servicio.ModificarTrasNombramiento(context.Background(), sol); !errors.Is(err, ErrSolicitudSeguimientoInvalida) {
		t.Fatalf("motivo fuera de la regla: %v", err)
	}
	caro := nuevoEscenarioSeguimiento(t, 9000000)
	sol.MotivoClave, sol.Canal = "cambio_jornada", caro.canal
	if _, err := caro.servicio.ModificarTrasNombramiento(context.Background(), sol); !errors.Is(err, ports.ErrModificacionCreditoInsuficiente) || len(caro.autorizador.solicitudes) != 0 {
		t.Fatalf("crédito insuficiente sin autorizar: %v", err)
	}
}
