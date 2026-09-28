package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type sellosReincorporacionPrueba struct{}

type lecturaReincorporacionPrueba struct {
	consultas                int
	organizacion, expediente string
	resultado                string
	err                      error
}

func (l *lecturaReincorporacionPrueba) LeerAntecedenteReincorporacionTitular(_ context.Context, m ports.MaterialReincorporacionTitular, _ vp.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.AntecedenteReincorporacionTitular, error) {
	l.consultas++
	l.organizacion, l.expediente = m.OrganizacionRef, m.ExpedienteRef
	resultado := l.resultado
	if resultado == "" {
		resultado = ports.ResultadoAntecedenteCoincide
	}
	antecedente := ports.AntecedenteReincorporacionTitular{Resultado: resultado,
		ExpedienteRef: m.ExpedienteRef, CeseEventoRef: "evento:cese:prueba", CeseReciboRef: "recibo:cese:prueba",
		LecturaRef: "lectura:" + strings.Repeat("c", 64), AuditoriaRef: "aud_v3_" + strings.Repeat("1", 32),
		ConsumoHuellaSHA256: strings.Repeat("a", 64), RegistradaEn: time.Date(2027, 2, 15, 10, 0, 0, 0, time.UTC)}
	if resultado == ports.ResultadoAntecedenteNoCoincide {
		antecedente.CeseEventoRef, antecedente.CeseReciboRef = "", ""
	}
	return antecedente, l.err
}

type politicaLecturaReincorporacionPrueba struct{ reglas *reglasSeguimientoDoble }

func (p politicaLecturaReincorporacionPrueba) PoliticaLecturaReincorporacionTitular(ctx context.Context, ahora time.Time) (ports.PoliticaOperacionSeguimiento, error) {
	_, politica, err := p.reglas.CausasCese(ctx, ahora)
	return politica, err
}

func (sellosReincorporacionPrueba) SellarAmbitoReincorporacionTitular(context.Context, ports.SolicitudSellarAmbitoIdempotencia) (ports.ColeccionSellosHMAC, error) {
	return ports.NuevaColeccionSellosHMAC("hmac-sha256:"+ports.DominioAmbitoReincorporacionTitular+"/v1:"+strings.Repeat("a", 64), nil)
}
func (sellosReincorporacionPrueba) DerivarHuellaReincorporacionTitular(context.Context, []byte) (ports.ColeccionSellosHMAC, error) {
	return ports.NuevaColeccionSellosHMAC("hmac-sha256:"+ports.DominioHuellaReincorporacionTitular+"/v1:"+strings.Repeat("b", 64), nil)
}

type repositorioReincorporacionPrueba struct {
	expediente                    domain.Expediente
	confirmada                    bool
	preparaciones, confirmaciones int
	recibo                        ports.ReciboReincorporacionTitular
}

func (r *repositorioReincorporacionPrueba) PrepararReincorporacionTitular(_ context.Context, m ports.MaterialReincorporacionTitular, lectura ports.AntecedenteReincorporacionTitular, s ports.SellosOperacionSeguimiento, refs ports.ReferenciasEfectoSeguimiento) (ports.PreparacionReincorporacionTitular, error) {
	if !lectura.ValidoPara(m) || lectura.Resultado != ports.ResultadoAntecedenteCoincide {
		return ports.PreparacionReincorporacionTitular{}, ports.ErrAutorizacionDenegada
	}
	r.preparaciones++
	a, _ := s.Ambitos.Datos()
	h, _ := s.Huellas.Datos()
	p := ports.PreparacionReincorporacionTitular{Expediente: r.expediente.Clonar(), Referencias: refs,
		AmbitoIdempotenciaHMAC: a.Activo.Valor, HuellaPeticionHMAC: h.Activo.Valor,
		CeseEventoRef: "evento:cese:prueba", CeseReciboRef: "recibo:cese:prueba", Confirmada: r.confirmada}
	if r.confirmada {
		p.Recibo = &r.recibo
	}
	return p, nil
}
func (r *repositorioReincorporacionPrueba) ConfirmarReincorporacionTitular(_ context.Context, o ports.OrdenConfirmarReincorporacionTitular) (ports.ReciboReincorporacionTitular, error) {
	if !o.Lectura.ValidoPara(o.Material) || o.Lectura.Resultado != ports.ResultadoAntecedenteCoincide {
		return ports.ReciboReincorporacionTitular{}, ports.ErrAutorizacionDenegada
	}
	r.confirmaciones++
	r.recibo = ports.ReciboReincorporacionTitular{Operacion: ports.OperacionRegistrarReincorporacionTitular,
		OrganizacionRef: o.Material.OrganizacionRef, ExpedienteRef: o.Material.ExpedienteRef, RelacionRef: o.Material.RelacionRef,
		FechaEfectiva: o.Material.FechaEfectiva.Format(time.DateOnly), CeseEventoRef: o.Preparacion.CeseEventoRef,
		CeseReciboRef: o.Preparacion.CeseReciboRef, VersionAnterior: o.Material.VersionEsperada,
		VersionResultante: o.Siguiente.Version, ReciboRef: o.Preparacion.Referencias.ReciboRef,
		AuditoriaRef: "aud_v3_" + strings.Repeat("0", 32), EventoRef: o.Preparacion.Referencias.EventoRef,
		ActorRef: o.Material.ActorRef, RegistradaEn: o.InstanteEfecto}
	r.confirmada = true
	return r.recibo, nil
}

func TestReincorporacionExigeCanalYReautorizaReplay(t *testing.T) {
	esc := nuevoEscenarioSeguimiento(t, 1)
	actual := esc.repo.expediente
	fecha := time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC)
	ref := "documento:ct:justificante:1"
	sha := strings.Repeat("b", 64)
	cesado, err := actual.RegistrarCese(actual.Version, domain.DatosCese{CausaClave: "fin_sustitucion", FechaEfecto: fecha,
		JustificanteTipo: "comunicacion_reincorporacion", JustificanteRef: ref, JustificanteSHA256: sha},
		domain.DatosActuacion{AccionClave: domain.AccionCesarNombramiento, ActorRef: "per_prueba", UnidadRef: actual.Asignacion.UnidadRef,
			ReciboRef: "recibo:cese:prueba", RealizadaEn: actual.ActualizadoEn.Add(time.Minute), FaseDestino: domain.FaseNombramiento,
			EstadoDestino: domain.EstadoEnCurso, DocumentosRef: []string{ref}})
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioReincorporacionPrueba{expediente: cesado}
	lectura := &lecturaReincorporacionPrueba{}
	s, err := NuevoServicioReincorporacionTitular(DependenciasReincorporacionTitular{Contextos: esc.servicio.contextos,
		Lector: lectura, PoliticaLectura: politicaLecturaReincorporacionPrueba{esc.reglas},
		Sellos: sellosReincorporacionPrueba{}, Repositorio: repo, Reglas: esc.reglas, Autorizador: esc.autorizador,
		Referencias: referenciasSeguimientoDoble{}, Reloj: esc.servicio.reloj})
	if err != nil {
		t.Fatal(err)
	}
	sol := SolicitudRegistrarReincorporacionTitular{Canal: esc.canal, ExpedienteRef: cesado.Referencia,
		RelacionRef: "relacion:personal:1", FechaEfectiva: fecha, DocumentoRef: ref, DocumentoSHA256: sha,
		VersionEsperada: cesado.Version, ClaveIdempotencia: "88888888-8888-4888-8888-888888888888"}
	if _, err = s.RegistrarReincorporacionTitular(context.Background(), sol); err != nil {
		t.Fatal(err)
	}
	if repo.confirmaciones != 1 || lectura.consultas != 1 || lectura.organizacion != sol.Canal.OrganizacionRef || lectura.expediente != sol.ExpedienteRef || len(esc.autorizador.solicitudes) != 2 ||
		esc.autorizador.solicitudes[0].Audiencia != ports.AudienciaLecturaReincorporacionTitularV1 ||
		esc.autorizador.solicitudes[0].Accion != ports.AccionConsultarAntecedenteReincorporacionTitular ||
		esc.autorizador.solicitudes[1].Audiencia != ports.AudienciaConsumoReincorporacionTitularV1 ||
		esc.autorizador.solicitudes[1].Accion != domain.AccionRegistrarReincorporacionTitular {
		t.Fatal("efecto o decisión V3 incorrectos")
	}
	if _, err = s.RegistrarReincorporacionTitular(context.Background(), sol); err != nil {
		t.Fatal(err)
	}
	if repo.confirmaciones != 1 || lectura.consultas != 2 || len(esc.autorizador.solicitudes) != 4 {
		t.Fatal("replay duplicó efecto o no reautorizó")
	}
	lectura.resultado = ports.ResultadoAntecedenteNoCoincide
	if _, err = s.RegistrarReincorporacionTitular(context.Background(), sol); !errors.Is(err, ports.ErrReincorporacionCeseNoCoincide) ||
		repo.preparaciones != 2 || len(esc.autorizador.solicitudes) != 5 {
		t.Fatal("lectura auditada sin coincidencia permitió preparar el expediente")
	}
	lectura.err = ports.ErrAutorizacionDenegada
	if _, err = s.RegistrarReincorporacionTitular(context.Background(), sol); !errors.Is(err, ports.ErrAutorizacionDenegada) ||
		repo.preparaciones != 2 || len(esc.autorizador.solicitudes) != 6 {
		t.Fatal("la lectura denegada permitió preparar o distinguir estado del expediente")
	}
	sol.Canal.AutenticacionRef = ""
	if _, err = s.RegistrarReincorporacionTitular(context.Background(), sol); !errors.Is(err, ErrSolicitudReincorporacionTitularInvalida) {
		t.Fatalf("canal: %v", err)
	}
}

type autorizadorReincorporacionDenegadorPrueba struct{ llamadas int }

func (a *autorizadorReincorporacionDenegadorPrueba) AutorizarOperacionSeguimiento(
	context.Context, ports.SolicitudAutorizarOperacionSeguimiento,
) (vp.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	a.llamadas++
	return vp.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrAutorizacionDenegada
}

func TestReincorporacionDenegadaAntesDeLectorYRepositorio(t *testing.T) {
	esc := nuevoEscenarioSeguimiento(t, 1)
	fecha := time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC)
	ref := "documento:ct:justificante:1"
	sha := strings.Repeat("b", 64)
	cesado, err := esc.repo.expediente.RegistrarCese(esc.repo.expediente.Version, domain.DatosCese{
		CausaClave: "fin_sustitucion", FechaEfecto: fecha, JustificanteTipo: "comunicacion_reincorporacion",
		JustificanteRef: ref, JustificanteSHA256: sha,
	}, domain.DatosActuacion{
		AccionClave: domain.AccionCesarNombramiento, ActorRef: "per_prueba", UnidadRef: esc.repo.expediente.Asignacion.UnidadRef,
		ReciboRef: "recibo:cese:prueba", RealizadaEn: esc.repo.expediente.ActualizadoEn.Add(time.Minute),
		FaseDestino: domain.FaseNombramiento, EstadoDestino: domain.EstadoEnCurso, DocumentosRef: []string{ref},
	})
	if err != nil {
		t.Fatal(err)
	}
	repo := &repositorioReincorporacionPrueba{expediente: cesado}
	lector := &lecturaReincorporacionPrueba{}
	denegador := &autorizadorReincorporacionDenegadorPrueba{}
	servicio, err := NuevoServicioReincorporacionTitular(DependenciasReincorporacionTitular{
		Contextos: esc.servicio.contextos, Lector: lector, PoliticaLectura: politicaLecturaReincorporacionPrueba{esc.reglas},
		Sellos: sellosReincorporacionPrueba{}, Repositorio: repo, Reglas: esc.reglas, Autorizador: denegador,
		Referencias: referenciasSeguimientoDoble{}, Reloj: esc.servicio.reloj,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = servicio.RegistrarReincorporacionTitular(context.Background(), SolicitudRegistrarReincorporacionTitular{
		Canal: esc.canal, ExpedienteRef: cesado.Referencia, RelacionRef: "relacion:personal:1",
		FechaEfectiva: fecha, DocumentoRef: ref, DocumentoSHA256: sha, VersionEsperada: cesado.Version,
		ClaveIdempotencia: "88888888-8888-4888-8888-888888888888",
	})
	if !errors.Is(err, ports.ErrAutorizacionDenegada) || denegador.llamadas != 1 ||
		lector.consultas != 0 || repo.preparaciones != 0 || repo.confirmaciones != 0 {
		t.Fatalf("denegación permitió lectura o efecto CT: err=%v autorización=%d lectura=%d preparación=%d confirmación=%d",
			err, denegador.llamadas, lector.consultas, repo.preparaciones, repo.confirmaciones)
	}
}
