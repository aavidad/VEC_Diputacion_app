package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	dominiovec "vec-diputacion-granada/internal/vec/domain"
)

type recuperadorPoliticaFinPrueba struct {
	politica   domain.PoliticaFin
	confirmada bool
	err        error
	analisis   ports.ConsultaPoliticaFinAnalisisConfirmada
	alta       ports.ConsultaPoliticaFinAltaConfirmada
	consultas  int
}

func (r *recuperadorPoliticaFinPrueba) ConsultarPoliticaFinAnalisisConfirmada(
	_ context.Context, c ports.ConsultaPoliticaFinAnalisisConfirmada,
) (domain.PoliticaFin, bool, error) {
	r.consultas++
	r.analisis = c
	return r.politica, r.confirmada, r.err
}

func (r *recuperadorPoliticaFinPrueba) ConsultarPoliticaFinAltaConfirmada(
	_ context.Context, c ports.ConsultaPoliticaFinAltaConfirmada,
) (domain.PoliticaFin, bool, error) {
	r.consultas++
	r.alta = c
	return r.politica, r.confirmada, r.err
}

type preparadorPeriodoFinPrueba struct {
	politica  domain.PoliticaFin
	consultas int
	err       error
}

func (p *preparadorPeriodoFinPrueba) PrepararPeriodoModalidad(
	_ context.Context, _ domain.ClaveCatalogo, periodo domain.PeriodoPrevisto,
) (domain.PeriodoPrevisto, error) {
	p.consultas++
	if p.err != nil {
		return domain.PeriodoPrevisto{}, p.err
	}
	periodo.PoliticaFin = p.politica
	return periodo, periodo.Validar()
}

type flujoFinReplayPrueba struct {
	delegado      ports.ResolutorFlujoAlta
	actor, perfil string
	vigente       bool
	llamadas      int
}

func (f *flujoFinReplayPrueba) ResolverFlujoAlta(ctx context.Context, solicitud ports.SolicitudResolverFlujo) (ports.ConfiguracionAltaFlujo, error) {
	f.llamadas++
	if !f.vigente && !PoliticaFinAltaConfirmadaPara(ctx, solicitud.OrganizacionRef, f.actor, f.perfil, solicitud.MotivoClave) {
		return ports.ConfiguracionAltaFlujo{}, ports.ErrFlujoNoDisponible
	}
	return f.delegado.ResolverFlujoAlta(ctx, solicitud)
}

type motivoFinReplayPrueba struct {
	delegado      ports.ResolutorMotivoAutorizacionAltaV3
	actor, perfil string
	vigente       bool
	llamadas      int
}

func (m *motivoFinReplayPrueba) ResolverMotivoAutorizacionAltaV3(ctx context.Context, solicitud ports.SolicitudResolverMotivoAutorizacionAltaV3) (dominiovec.ReferenciaEntradaCatalogo, error) {
	m.llamadas++
	if !m.vigente && !PoliticaFinAltaConfirmadaPara(ctx, solicitud.OrganizacionRef, m.actor, m.perfil, solicitud.MotivoClave) {
		return dominiovec.ReferenciaEntradaCatalogo{}, ports.ErrMotivoAutorizacionNoDisponible
	}
	return m.delegado.ResolverMotivoAutorizacionAltaV3(ctx, solicitud)
}

func politicaFinHistoricaPrueba() domain.PoliticaFin {
	return domain.PoliticaFin{
		ReglaRef: "regla:modalidad-historica-sintetica", CatalogoVersion: 4,
		CatalogoHuellaSHA256: strings.Repeat("a", 64),
		FechaFin:             "opcional", CausaFin: "fin_sustitucion",
	}
}

func TestAnalisisRecuperaPoliticaHistoricaAntesDeSellarSemantica(t *testing.T) {
	escenario := nuevoEscenarioOperacionAnalisisSaneado(t, ports.OperacionRegistrarAnalisis, "-fin-historico")
	escenario.registrar.DatosFuncionales.Periodo.Fin = time.Time{}
	escenario.registrar.DatosFuncionales.Periodo.CausaFin = "fin_sustitucion"
	servicio, dobles := construirServicioOperacionAnalisisSaneado(t, escenario)
	actual := &preparadorPeriodoFinPrueba{politica: domain.PoliticaFin{
		ReglaRef: "regla:modalidad-nueva-sintetica", CatalogoVersion: 5,
		CatalogoHuellaSHA256: strings.Repeat("b", 64), FechaFin: "opcional", CausaFin: "fin_sustitucion",
	}}
	recuperacion := &recuperadorPoliticaFinPrueba{politica: politicaFinHistoricaPrueba(), confirmada: true}
	servicio.periodos = actual
	if err := servicio.ConfigurarRecuperacionPoliticaFin(recuperacion); err != nil {
		t.Fatal(err)
	}
	dobles.preparaciones.errConsulta = errors.New("parada sintética tras contraste semántico")
	_, _ = servicio.Registrar(context.Background(), escenario.registrar)
	if recuperacion.consultas != 1 || recuperacion.analisis.Validar() != nil || actual.consultas != 0 || dobles.preparaciones.consultas != 1 {
		t.Fatal("el replay no buscó la instantánea histórica antes de la consulta confirmada")
	}
	vinculo, err := escenario.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	esperados := ports.DatosPreimagenesConsultaOperacionAnalisis{
		ClaveIdempotencia: escenario.registrar.ClaveIdempotencia,
		Operacion:         ports.OperacionRegistrarAnalisis, OrganizacionRef: escenario.registrar.OrganizacionRef,
		ExpedienteRef: escenario.registrar.ExpedienteRef, VersionExpediente: escenario.registrar.VersionEsperada,
		ActorRef: vinculo.PrincipalID, PerfilRef: vinculo.PerfilActivoRef,
		ArtefactoRef: escenario.registrar.ArtefactoRef, DatosFuncionales: escenario.registrar.DatosFuncionales,
	}
	esperados.DatosFuncionales.Periodo.PoliticaFin = recuperacion.politica
	preimagenes, err := ports.NuevasPreimagenesConsultaOperacionAnalisis(esperados)
	if err != nil {
		t.Fatal(err)
	}
	sellosEsperados, err := dobles.sellador.SellarOperacionAnalisis(context.Background(), preimagenes)
	if err != nil {
		t.Fatal(err)
	}
	sellosConsulta, err := dobles.preparaciones.solicitudConsulta.SellosIdentidadFuncional()
	if err != nil {
		t.Fatal(err)
	}
	_, semanticaEsperada, err := sellosEsperados.ParActivo()
	if err != nil {
		t.Fatal(err)
	}
	_, semanticaConsulta, err := sellosConsulta.ParActivo()
	if err != nil || semanticaConsulta != semanticaEsperada {
		t.Fatal("el contraste del recibo no incluye la política original y los datos cliente")
	}
}

func TestAltaRecuperaPoliticaHistoricaAntesDeDerivarHuella(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	escenario.solicitud.Solicitud.Periodo.Fin = time.Time{}
	escenario.solicitud.Solicitud.Periodo.CausaFin = "fin_sustitucion"
	servicio, dobles := construirServicioRegistro(t, escenario)
	actual := &preparadorPeriodoFinPrueba{politica: domain.PoliticaFin{
		ReglaRef: "regla:modalidad-nueva-sintetica", CatalogoVersion: 5,
		CatalogoHuellaSHA256: strings.Repeat("b", 64), FechaFin: "opcional", CausaFin: "fin_sustitucion",
	}}
	recuperacion := &recuperadorPoliticaFinPrueba{politica: politicaFinHistoricaPrueba(), confirmada: true}
	servicio.periodos = actual
	if err := servicio.ConfigurarRecuperacionPoliticaFin(recuperacion); err != nil {
		t.Fatal(err)
	}
	var huellaPeriodo domain.PeriodoPrevisto
	dobles.huellas.antes = func(material *ports.MaterialHuellaAlta) { huellaPeriodo = material.Solicitud.Periodo }
	dobles.candidaturas.err = errors.New("parada sintética tras derivar huella")
	_, _ = servicio.Registrar(context.Background(), escenario.solicitud)
	if recuperacion.consultas != 1 || recuperacion.alta.Validar() != nil ||
		actual.consultas != 0 || huellaPeriodo.PoliticaFin != recuperacion.politica ||
		dobles.candidaturas.llamadas != 1 {
		t.Fatal("el alta no rehizo la huella con su política confirmada")
	}
}

func TestAnalisisDistingueLegadoConfirmadoDeOperacionNueva(t *testing.T) {
	for _, caso := range []struct {
		nombre       string
		confirmada   bool
		consultasC12 int
	}{
		{nombre: "legado_confirmado", confirmada: true, consultasC12: 0},
		{nombre: "operacion_nueva", confirmada: false, consultasC12: 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			escenario := nuevoEscenarioOperacionAnalisisSaneado(t, ports.OperacionRegistrarAnalisis, "-fin-legacy")
			escenario.registrar.DatosFuncionales.Periodo.Fin = time.Time{}
			escenario.registrar.DatosFuncionales.Periodo.CausaFin = "fin_sustitucion"
			servicio, dobles := construirServicioOperacionAnalisisSaneado(t, escenario)
			actual := &preparadorPeriodoFinPrueba{politica: politicaFinHistoricaPrueba()}
			recuperacion := &recuperadorPoliticaFinPrueba{confirmada: caso.confirmada}
			servicio.periodos = actual
			if err := servicio.ConfigurarRecuperacionPoliticaFin(recuperacion); err != nil {
				t.Fatal(err)
			}
			dobles.preparaciones.errConsulta = errors.New("parada sintética después del contraste")
			_, _ = servicio.Registrar(context.Background(), escenario.registrar)
			if recuperacion.consultas != 1 || actual.consultas != caso.consultasC12 || dobles.preparaciones.consultas != 1 {
				t.Fatal("el estado de confirmación no decide correctamente la política")
			}
			vinculo, err := escenario.contexto.Vinculo.Datos()
			if err != nil {
				t.Fatal(err)
			}
			datos := ports.DatosPreimagenesConsultaOperacionAnalisis{
				ClaveIdempotencia: escenario.registrar.ClaveIdempotencia,
				Operacion:         ports.OperacionRegistrarAnalisis,
				OrganizacionRef:   escenario.registrar.OrganizacionRef,
				ExpedienteRef:     escenario.registrar.ExpedienteRef,
				VersionExpediente: escenario.registrar.VersionEsperada,
				ActorRef:          vinculo.PrincipalID, PerfilRef: vinculo.PerfilActivoRef,
				ArtefactoRef:     escenario.registrar.ArtefactoRef,
				DatosFuncionales: escenario.registrar.DatosFuncionales,
			}
			if !caso.confirmada {
				datos.DatosFuncionales.Periodo.PoliticaFin = actual.politica
			}
			preimagenes, err := ports.NuevasPreimagenesConsultaOperacionAnalisis(datos)
			if err != nil {
				t.Fatal(err)
			}
			sellosEsperados, err := dobles.sellador.SellarOperacionAnalisis(context.Background(), preimagenes)
			if err != nil {
				t.Fatal(err)
			}
			sellosConsulta, err := dobles.preparaciones.solicitudConsulta.SellosIdentidadFuncional()
			if err != nil {
				t.Fatal(err)
			}
			_, semanticaEsperada, err := sellosEsperados.ParActivo()
			if err != nil {
				t.Fatal(err)
			}
			_, semanticaConsulta, err := sellosConsulta.ParActivo()
			if err != nil || semanticaEsperada != semanticaConsulta {
				t.Fatal("la huella de consulta no corresponde a la política aplicable")
			}
		})
	}
}

func TestAltaReplayConfirmadoAdmiteModalidadRetiradaYClaveNuevaSeDeniega(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	escenario.solicitud.Solicitud.Periodo.Fin = time.Time{}
	escenario.solicitud.Solicitud.Periodo.CausaFin = "fin_sustitucion"
	servicio, dobles := construirServicioRegistro(t, escenario)
	vinculo, err := escenario.contexto.Vinculo.Datos()
	if err != nil {
		t.Fatal(err)
	}
	flujo := &flujoFinReplayPrueba{delegado: dobles.flujos, actor: vinculo.PrincipalID, perfil: vinculo.PerfilActivoRef, vigente: true}
	motivo := &motivoFinReplayPrueba{delegado: dobles.motivos, actor: vinculo.PrincipalID, perfil: vinculo.PerfilActivoRef, vigente: true}
	servicio.flujos, servicio.motivos = flujo, motivo
	politica := politicaFinHistoricaPrueba()
	periodos := &preparadorPeriodoFinPrueba{politica: politica}
	recuperacion := &recuperadorPoliticaFinPrueba{}
	servicio.periodos = periodos
	if err := servicio.ConfigurarRecuperacionPoliticaFin(recuperacion); err != nil {
		t.Fatal(err)
	}
	var huellas int
	dobles.huellas.antes = func(*ports.MaterialHuellaAlta) { huellas++ }
	original, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if err != nil {
		t.Fatalf("alta inicial sintética: %v", err)
	}
	if original != escenario.recibo || huellas != 1 || dobles.autorizador.llamadas != 1 || dobles.transaccion.llamadas != 1 {
		t.Fatal("la confirmación inicial no recorrió huella, V3 y transacción")
	}
	flujo.vigente, motivo.vigente = false, false
	periodos.err = errors.New("modalidad retirada de c12")
	recuperacion.confirmada, recuperacion.politica = true, politica
	reintentoPeticion := escenario.solicitud
	reintentoPeticion.Solicitud.Periodo.PoliticaFin = politica
	repetido, err := servicio.Registrar(context.Background(), reintentoPeticion)
	if err != nil || repetido != original || huellas != 2 || dobles.autorizador.llamadas != 2 || dobles.transaccion.llamadas != 2 || periodos.consultas != 1 {
		t.Fatalf("el replay confirmado perdió su recibo o saltó huella/V3: %v", err)
	}
	politicaAjena := politica
	politicaAjena.CatalogoVersion++
	reintentoPeticion.Solicitud.Periodo.PoliticaFin = politicaAjena
	_, err = servicio.Registrar(context.Background(), reintentoPeticion)
	if !errors.Is(err, ports.ErrClaveIdempotenciaUsada) || huellas != 2 || dobles.autorizador.llamadas != 2 || dobles.transaccion.llamadas != 2 {
		t.Fatalf("snapshot distinto reutilizó la clave confirmada: %v", err)
	}
	nueva := escenario.solicitud
	nueva.ClaveIdempotencia = "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	recuperacion.confirmada = false
	recuperacion.politica = domain.PoliticaFin{}
	_, err = servicio.Registrar(context.Background(), nueva)
	if !errors.Is(err, ErrSolicitudRegistroInvalida) || huellas != 2 || dobles.autorizador.llamadas != 2 || dobles.transaccion.llamadas != 2 || periodos.consultas != 2 {
		t.Fatalf("la nueva clave para modalidad retirada llegó al efecto: %v", err)
	}
}

func TestAltaNuevaDePeticionRatificadaConservaSnapshotSinConsultarC12Retirada(t *testing.T) {
	escenario := nuevoEscenarioRegistro(t)
	escenario.solicitud.Solicitud.Periodo.Fin = time.Time{}
	escenario.solicitud.Solicitud.Periodo.CausaFin = "fin_sustitucion"
	escenario.solicitud.Solicitud.Periodo.PoliticaFin = politicaFinHistoricaPrueba()
	servicio, dobles := construirServicioRegistro(t, escenario)
	periodos := &preparadorPeriodoFinPrueba{err: errors.New("modalidad retirada de c12")}
	recuperacion := &recuperadorPoliticaFinPrueba{}
	servicio.periodos = periodos
	if err := servicio.ConfigurarRecuperacionPoliticaFin(recuperacion); err != nil {
		t.Fatal(err)
	}
	var periodoHuella domain.PeriodoPrevisto
	dobles.huellas.antes = func(material *ports.MaterialHuellaAlta) {
		periodoHuella = material.Solicitud.Periodo
	}
	recibo, err := servicio.Registrar(context.Background(), escenario.solicitud)
	if err != nil || recibo != escenario.recibo || recuperacion.consultas != 1 ||
		periodos.consultas != 0 || periodoHuella != escenario.solicitud.Solicitud.Periodo ||
		dobles.autorizador.llamadas != 1 || dobles.transaccion.llamadas != 1 {
		t.Fatalf("alta de petición ratificada no conservó su snapshot y controles: %v", err)
	}
}
