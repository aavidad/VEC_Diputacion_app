package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

type ginpixSeguimientoDoble struct{ r *reglasSeguimientoDoble }

func (g ginpixSeguimientoDoble) PoliticaConfirmacionGINPIX(context.Context, time.Time) (ports.PoliticaOperacionSeguimiento, error) {
	return g.r.politica("vec.contratacion_temporal.reglas"), nil
}

type acreditadaDoble struct {
	estado ports.EstadoIncorporacionAcreditada
	err    error
}

func (a *acreditadaDoble) ConsultarIncorporacionAcreditada(context.Context, string, string) (ports.EstadoIncorporacionAcreditada, error) {
	return a.estado, a.err
}

func escenarioAcreditada(t *testing.T) (escenarioSeguimiento, *acreditadaDoble) {
	t.Helper()
	e := nuevoEscenarioSeguimiento(t, 1)
	acreditada := &acreditadaDoble{}
	s, err := NuevoServicioOperacionesSeguimiento(DependenciasOperacionesSeguimiento{Contextos: e.servicio.contextos, Sellos: e.servicio.sellos,
		Repositorio: e.repo, Reglas: e.reglas, Autorizador: e.autorizador, Referencias: referenciasSeguimientoDoble{}, Coste: costeSeguimientoDoble{1},
		Lector: e.repo, Reloj: e.servicio.reloj, GINPIX: ginpixSeguimientoDoble{e.reglas}, Acreditada: acreditada})
	if err != nil {
		t.Fatal(err)
	}
	e.servicio = s
	return e, acreditada
}

func TestIncorporacionAcreditadaExigeAmbasDependencias(t *testing.T) {
	e := nuevoEscenarioSeguimiento(t, 1)
	_, err := NuevoServicioOperacionesSeguimiento(DependenciasOperacionesSeguimiento{Contextos: e.servicio.contextos, Sellos: e.servicio.sellos,
		Repositorio: e.repo, Reglas: e.reglas, Autorizador: e.autorizador, Referencias: referenciasSeguimientoDoble{}, Coste: costeSeguimientoDoble{1},
		Lector: e.repo, Reloj: e.servicio.reloj, GINPIX: ginpixSeguimientoDoble{e.reglas}})
	if !errors.Is(err, ErrServicioSeguimientoInvalido) {
		t.Fatalf("GINPIX sin lector de la incorporación acreditada: %v", err)
	}
	if _, err := e.servicio.ConfirmarGINPIX(context.Background(), SolicitudConfirmarGINPIX{Canal: e.canal}); !errors.Is(err, ports.ErrOperacionSeguimientoNoDisponible) {
		t.Fatalf("sin incorporación acreditada la confirmación no existe: %v", err)
	}
	if o, err := e.servicio.Opciones(context.Background()); err != nil || o.ConfirmacionGINPIX {
		t.Fatalf("opciones sin incorporación acreditada: %+v %v", o, err)
	}
}

func TestConfirmacionGINPIXLigaNumeroFechaEIncorporacion(t *testing.T) {
	e, _ := escenarioAcreditada(t)
	sol := SolicitudConfirmarGINPIX{Canal: e.canal, ExpedienteRef: e.repo.expediente.Referencia, VersionEsperada: 7,
		ClaveIdempotencia: "44444444-4444-4444-8444-444444444444", GINPIXNumero: "GX-2027-0042", GINPIXConfirmada: time.Date(2027, 2, 10, 0, 0, 0, 0, time.UTC)}
	recibo, err := e.servicio.ConfirmarGINPIX(context.Background(), sol)
	if err != nil || recibo.VersionResultante != 8 || recibo.EstadoResultante != domain.EstadoEnCurso {
		t.Fatalf("confirmación: %+v %v", recibo, err)
	}
	orden := e.repo.ordenes[0]
	a := orden.Contexto.Atributos
	if orden.Operacion != ports.OperacionConfirmarGINPIX || a["ginpix_numero"] != "GX-2027-0042" || a["ginpix_confirmada_en"] != "2027-02-10" ||
		a["incorporacion_ref"] != "ref:incorporacion:prueba" || e.autorizador.solicitudes[0].Audiencia != ports.AudienciaConsumoConfirmacionGINPIXV1 ||
		e.autorizador.solicitudes[0].Recurso.Tipo != ports.TipoRecursoConfirmacionGINPIX {
		t.Fatalf("contexto autorizado inesperado: %+v", a)
	}
	ultima := orden.Siguiente.Actuaciones[len(orden.Siguiente.Actuaciones)-1]
	if ultima.AccionClave != domain.AccionConfirmarGINPIX || len(ultima.DocumentosRef) != 1 || ultima.DocumentosRef[0] != "ginpix:GX-2027-0042" {
		t.Fatalf("actuación: %+v", ultima)
	}
	sol.GINPIXNumero = "-malo"
	if _, err := e.servicio.ConfirmarGINPIX(context.Background(), sol); !errors.Is(err, ErrSolicitudSeguimientoInvalida) {
		t.Fatalf("número inválido: %v", err)
	}
	if o, err := e.servicio.Opciones(context.Background()); err != nil || !o.ConfirmacionGINPIX {
		t.Fatalf("opciones con incorporación acreditada: %+v %v", o, err)
	}
}

func TestCierreTomaElNumeroDeLaConfirmacionDeGINPIX(t *testing.T) {
	e, acreditada := escenarioAcreditada(t)
	cesado, err := e.repo.expediente.RegistrarCese(7, domain.DatosCese{CausaClave: "renuncia", FechaEfecto: time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC),
		JustificanteTipo: "escrito_renuncia", JustificanteRef: "documento:renuncia", JustificanteSHA256: strings.Repeat("b", 64)},
		domain.DatosActuacion{AccionClave: domain.AccionCesarNombramiento, ActorRef: "per_prueba", UnidadRef: e.repo.expediente.Asignacion.UnidadRef,
			ReciboRef: "recibo:cese:prueba", RealizadaEn: e.repo.expediente.ActualizadoEn.Add(time.Minute), FaseDestino: domain.FaseNombramiento,
			EstadoDestino: domain.EstadoEnCurso, DocumentosRef: []string{"documento:renuncia"}})
	if err != nil {
		t.Fatal(err)
	}
	e.repo.expediente = cesado
	e.reglas.condiciones = []string{domain.CondicionCeseRegistrado, domain.CondicionGINPIXConfirmado}
	sol := SolicitudCerrarExpediente{Canal: e.canal, ExpedienteRef: cesado.Referencia, VersionEsperada: 8, ClaveIdempotencia: "22222222-2222-4222-8222-222222222222"}
	if _, err := e.servicio.CerrarExpediente(context.Background(), sol); !errors.Is(err, ports.ErrGINPIXNoConfirmado) || len(e.autorizador.solicitudes) != 0 {
		t.Fatalf("sin confirmación de GINPIX no hay cierre: %v", err)
	}
	acreditada.estado.GINPIX = &ports.EstadoGINPIXConfirmado{Numero: "GX-2027-0042", ConfirmadaEn: "2027-02-10", ReciboRef: "recibo:ginpix:prueba",
		RegistradaEn: time.Date(2027, 2, 10, 9, 0, 0, 0, time.UTC)}
	if _, err := e.servicio.CerrarExpediente(context.Background(), sol); err != nil {
		t.Fatalf("cierre con la confirmación: %v", err)
	}
	a := e.repo.ordenes[len(e.repo.ordenes)-1].Contexto.Atributos
	if a["ginpix_numero"] != "GX-2027-0042" || a["ginpix_confirmada_en"] != "2027-02-10" {
		t.Fatalf("el cierre no usa el número confirmado: %+v", a)
	}
	sol.GINPIXNumero = "GX-OTRO"
	fecha := time.Date(2027, 2, 10, 0, 0, 0, 0, time.UTC)
	sol.GINPIXConfirmadaEn = &fecha
	if _, err := e.servicio.CerrarExpediente(context.Background(), sol); !errors.Is(err, ports.ErrGINPIXDistinto) {
		t.Fatalf("número distinto del confirmado: %v", err)
	}
	sol.GINPIXNumero = "GX-2027-0042"
	otra := fecha.AddDate(0, 0, 1)
	sol.GINPIXConfirmadaEn = &otra
	if _, err := e.servicio.CerrarExpediente(context.Background(), sol); !errors.Is(err, ports.ErrGINPIXDistinto) {
		t.Fatalf("fecha distinta de la confirmada: %v", err)
	}
	acreditada.estado.GINPIX.Numero = "-malo"
	sol.GINPIXNumero, sol.GINPIXConfirmadaEn = "", nil
	if _, err := e.servicio.CerrarExpediente(context.Background(), sol); !errors.Is(err, ports.ErrResultadoSeguimientoNoConfiable) {
		t.Fatalf("lectura no confiable: %v", err)
	}
}

func TestEstadoAnadeLaIncorporacionAcreditada(t *testing.T) {
	e, acreditada := escenarioAcreditada(t)
	acreditada.estado.Centro = &ports.EstadoConfirmacionCentro{FechaIncorporacion: "2026-09-01", DocumentoTipo: "toma_posesion",
		DocumentoRef: "registro:centro:1", DocumentoSHA256: strings.Repeat("f", 64), ReciboRef: "recibo:centro:1", RegistradaEn: time.Date(2026, 9, 1, 8, 0, 0, 0, time.UTC)}
	estado, err := e.servicio.Estado(context.Background(), e.canal.OrganizacionRef, e.repo.expediente.Referencia)
	if err != nil || estado.Acreditada == nil || estado.Acreditada.Centro == nil || estado.Acreditada.GINPIX != nil {
		t.Fatalf("estado: %+v %v", estado, err)
	}
	acreditada.estado.Centro.DocumentoSHA256 = "no"
	if _, err := e.servicio.Estado(context.Background(), e.canal.OrganizacionRef, e.repo.expediente.Referencia); !errors.Is(err, ports.ErrResultadoSeguimientoNoConfiable) {
		t.Fatalf("estado no confiable: %v", err)
	}
}

type repositorioCentroDoble struct {
	filas       []ports.ExpedienteIncorporacionCentro
	materiales  []ports.MaterialIncorporacionCentro
	consultas   int
	errConsulta error
}

func (r *repositorioCentroDoble) ListarIncorporacionesCentro(_ context.Context, c ports.ConsultaIncorporacionesCentro) ([]ports.ExpedienteIncorporacionCentro, error) {
	r.consultas++
	if c.Validar() != nil {
		return nil, ports.ErrIncorporacionCentroInvalida
	}
	return r.filas, r.errConsulta
}

func (r *repositorioCentroDoble) ConfirmarIncorporacionCentro(_ context.Context, m ports.MaterialIncorporacionCentro) (ports.ReciboIncorporacionCentro, error) {
	r.materiales = append(r.materiales, m)
	return ports.ReciboIncorporacionCentro{ReciboRef: "recibo:centro:1", ConfirmacionRef: "confirmacion:centro:1", PeticionRef: m.PeticionRef,
		ExpedienteRef: m.ExpedienteRef, FechaIncorporacion: m.FechaIncorporacion, DocumentoTipo: m.Documento.Tipo, ActorRef: m.Actor.ActorRef,
		RegistradoEn: time.Now().UTC(), EstadoLocal: "registrado"}, nil
}

type reglaAcreditacionDoble struct{}

func (reglaAcreditacionDoble) DocumentoAcreditativo(_ context.Context, modalidad domain.ClaveCatalogo) (domain.ClaveCatalogo, ports.ReglaDocumentoIncorporacion, error) {
	tipo := domain.ClaveCatalogo("toma_posesion")
	if modalidad == "relevo" {
		tipo = "contrato_firmado"
	}
	return tipo, ports.ReglaDocumentoIncorporacion{Referencia: "vec.contratacion_temporal.reglas:1:c21.acreditacion_incorporacion",
		HuellaSHA256: strings.Repeat("a", 64), ModalidadClave: string(modalidad)}, nil
}

func TestConfirmacionDelCentroTomaElDocumentoDelCatalogo(t *testing.T) {
	actor := domain.ActorPeticionCentro{ActorRef: "per_centro_1", PerfilRef: "prf_centro_1", CentroRef: "centro-520", PuestoRef: "puesto-1"}
	repo := &repositorioCentroDoble{filas: []ports.ExpedienteIncorporacionCentro{
		{PeticionRef: "peticion:centro:1", ExpedienteRef: "expediente:ct:1", NumeroVisible: "2026/1", Version: 7, Fase: "nombramiento", Estado: "en_curso", ModalidadClave: "relevo"},
		{PeticionRef: "peticion:centro:2", ExpedienteRef: "expediente:ct:2", NumeroVisible: "2026/2", Version: 3, Fase: "fiscalizacion", Estado: "en_curso", ModalidadClave: "sustitucion"},
	}}
	s, err := NuevoServicioIncorporacionCentro(repo, reglaAcreditacionDoble{}, relojSeguimientoFijo{time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	bandeja, err := s.Bandeja(context.Background(), "organizacion:desarrollo:dipgra", actor)
	if err != nil || len(bandeja) != 2 || bandeja[0].DocumentoExigido != "contrato_firmado" || bandeja[1].DocumentoExigido != "" {
		t.Fatalf("bandeja: %+v %v", bandeja, err)
	}
	sol := SolicitudConfirmarIncorporacionCentro{OrganizacionRef: "organizacion:desarrollo:dipgra", Actor: actor,
		ClaveIdempotencia: "77777777-7777-4777-8777-777777777777", PeticionRef: "peticion:centro:1", ExpedienteRef: "expediente:ct:1",
		FechaIncorporacion: "2026-09-25", DocumentoRef: "registro:centro:1", DocumentoSHA256: strings.Repeat("b", 64)}
	recibo, err := s.Confirmar(context.Background(), sol)
	if err != nil || recibo.DocumentoTipo != "contrato_firmado" || repo.materiales[0].Regla.ModalidadClave != "relevo" {
		t.Fatalf("confirmación: %+v %v", recibo, err)
	}
	futura := sol
	futura.FechaIncorporacion = "2026-09-27"
	if _, err := s.Confirmar(context.Background(), futura); !errors.Is(err, ports.ErrIncorporacionCentroInvalida) {
		t.Fatalf("incorporación futura: %v", err)
	}
	otra := sol
	otra.ExpedienteRef, otra.PeticionRef = "expediente:ct:2", "peticion:centro:2"
	if _, err := s.Confirmar(context.Background(), otra); !errors.Is(err, ports.ErrIncorporacionCentroNoAdmitida) {
		t.Fatalf("expediente sin nombramiento: %v", err)
	}
	ajena := sol
	ajena.PeticionRef = "peticion:centro:2"
	if _, err := s.Confirmar(context.Background(), ajena); !errors.Is(err, ports.ErrIncorporacionCentroNoAdmitida) {
		t.Fatalf("expediente de otra petición: %v", err)
	}
	sinHuella := sol
	sinHuella.DocumentoSHA256 = strings.Repeat("0", 64)
	if _, err := s.Confirmar(context.Background(), sinHuella); !errors.Is(err, ports.ErrIncorporacionCentroInvalida) {
		t.Fatalf("huella nula: %v", err)
	}
	antes := repo.consultas
	if _, err := s.Confirmar(context.Background(), sinHuella); err == nil || repo.consultas != antes {
		t.Fatal("una solicitud inválida no consume la lectura autorizada")
	}
	if len(repo.materiales) != 1 {
		t.Fatalf("solo una confirmación llega al repositorio: %d", len(repo.materiales))
	}
}
