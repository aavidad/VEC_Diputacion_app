package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var instanteConsultaInternaAutoridadPrueba = time.Date(
	2026, time.July, 17, 12, 0, 0, 123_456_000, time.UTC,
)

type consultaInternaGobernadaAutoridadPrueba struct {
	encontrada bool
	fuente     domain.FuenteAutoridadVersionada
	error      error
	alterar    func(*ports.ResultadoConsultaInternaFuenteAutoridad)
	antes      func(ports.SolicitudConsultaInternaGobernadaFuenteAutoridad)
	llamadas   int
	solicitud  ports.SolicitudConsultaInternaGobernadaFuenteAutoridad
}

func (c *consultaInternaGobernadaAutoridadPrueba) ConsultarVersionExacta(
	ctx context.Context,
	solicitud ports.SolicitudConsultaInternaGobernadaFuenteAutoridad,
) (ports.ResultadoConsultaInternaFuenteAutoridad, error) {
	c.llamadas++
	c.solicitud = solicitud
	if c.antes != nil {
		c.antes(solicitud)
	}
	if c.error != nil {
		return ports.ResultadoConsultaInternaFuenteAutoridad{}, c.error
	}
	if err := ctx.Err(); err != nil {
		return ports.ResultadoConsultaInternaFuenteAutoridad{}, err
	}
	selector, errSelector := solicitud.Selector()
	autorizacion, errAutorizacion := solicitud.Autorizacion()
	datosAutorizacion, errDatos := autorizacion.Datos()
	solicitadaEn, errInstante := solicitud.SolicitadaEn()
	if errSelector != nil || errAutorizacion != nil || errDatos != nil || errInstante != nil {
		return ports.ResultadoConsultaInternaFuenteAutoridad{}, ports.ErrConsultaInternaFuenteAutoridadInvalida
	}
	resultadoRecibo := ports.ResultadoConsultaFuenteNoEncontrada
	if c.encontrada {
		resultadoRecibo = ports.ResultadoConsultaFuenteEncontrada
	}
	estado := ports.ReferenciaEstadoFuenteAutoridad{}
	if c.encontrada {
		estado, _ = ports.EstadoExactoFuenteAutoridad(c.fuente)
	}
	auditoriaConfirmada, err := ports.PrepararAuditoriaResultadoConsultaInternaFuenteAutoridad(
		solicitud, resultadoRecibo, estado,
	)
	if err != nil {
		return ports.ResultadoConsultaInternaFuenteAutoridad{}, err
	}
	auditoriaConfirmada.ID = "auditoria:consulta:fuente:00000001"
	auditoriaConfirmada.Seq = 17
	auditoriaConfirmada.IntegrityAlgorithm = "sha256-chain-v1"
	auditoriaConfirmada.PrevSignature = "firma:auditoria:fuente:anterior"
	auditoriaConfirmada.Signature = "firma:auditoria:fuente:actual"
	recibo, err := ports.NuevoReciboConsultaInternaFuenteAutoridad(
		solicitud,
		ports.DatosReciboConsultaInternaFuenteAutoridad{
			TransaccionRef:                 "transaccion:consulta:fuente:00000001",
			Selector:                       selector,
			Resultado:                      resultadoRecibo,
			Estado:                         estado,
			DecisionRef:                    datosAutorizacion.Decision.DecisionRef,
			HuellaDecisionSHA256:           datosAutorizacion.HuellaDecisionSHA256,
			AuditoriaRef:                   auditoriaConfirmada.ID,
			AuditoriaSecuencia:             auditoriaConfirmada.Seq,
			AuditoriaAlgoritmoIntegridad:   auditoriaConfirmada.IntegrityAlgorithm,
			AuditoriaEncadenadoAnteriorRef: auditoriaConfirmada.PrevSignature,
			AuditoriaFirmaRef:              auditoriaConfirmada.Signature,
			AuditoriaConfirmada:            auditoriaConfirmada,
			ConfirmadaEn:                   solicitadaEn.Add(time.Microsecond),
		},
	)
	if err != nil {
		return ports.ResultadoConsultaInternaFuenteAutoridad{}, err
	}
	resultado := ports.ResultadoConsultaInternaFuenteAutoridad{
		Encontrada: c.encontrada, Estado: estado, Recibo: recibo,
	}
	if c.encontrada {
		resultado.Fuente = c.fuente
	}
	if c.alterar != nil {
		c.alterar(&resultado)
	}
	return resultado, nil
}

type exigidorConsultaInternaAutoridadPrueba struct {
	error    error
	llamadas int
}

type autorizadorConsultaInternaAutoridadV2Prueba struct {
	ahora          time.Time
	campos         []string
	obligaciones   []string
	garantiaMinima domain.AuthAssurance
	mutar          func(*domain.DecisionAutorizacion)
	observar       func(domain.DatosSolicitudAutorizacionLigadaV2)
	despues        func()
	llamadas       int
}

func (a *autorizadorConsultaInternaAutoridadV2Prueba) ExigirSolicitudLigadaV2(
	ctx context.Context,
	solicitud domain.SolicitudAutorizacionLigadaV2,
) (domain.DecisionAutorizacion, error) {
	if err := ctx.Err(); err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	datos, err := solicitud.Datos()
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	datosVinculo, err := datos.VinculoAutenticacionActor.Datos()
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	huellaContexto, err := datos.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	huellaCatalogoPoliticas, err := domain.HuellaEvidenciasCatalogoPoliticasAutorizacion(nil, nil)
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	huellaSolicitud, err := domain.HuellaSHA256SolicitudAutorizacionV2(solicitud)
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	huellaMotivo, err := domain.HuellaSHA256MotivoAutorizacionV2(datos.ReferenciaMotivo)
	if err != nil {
		return domain.DecisionAutorizacion{}, err
	}
	a.llamadas++
	decision := domain.DecisionAutorizacion{
		DecisionRef: fmt.Sprintf("decision:consulta-autoridad:%03d", a.llamadas),
		Concedida:   true,
		Codigo:      "concedida",
		PrincipalID: datosVinculo.PrincipalID, PerfilActivoRef: datosVinculo.PerfilActivoRef,
		Accion: datos.Accion, RecursoRef: datos.Recurso.Referencia,
		ModuloID: datos.Recurso.ModuloID, TipoRecurso: datos.Recurso.Tipo,
		ContextoRecursoHuellaSHA256: huellaContexto,
		Finalidad:                   datos.Finalidad, CorrelacionRef: datos.CorrelacionRef,
		EsquemaHuellaSolicitud:                domain.EsquemaHuellaSolicitudAutorizacionV2,
		SolicitudHuellaSHA256:                 huellaSolicitud,
		EsquemaHuellaMotivo:                   domain.EsquemaHuellaMotivoAutorizacionV2,
		MotivoHuellaSHA256:                    huellaMotivo,
		VinculoAutenticacionActor:             datos.VinculoAutenticacionActor,
		AsignacionRef:                         "asignacion:consulta-autoridad:v1",
		AsignacionHuellaSHA256:                strings.Repeat("a", 64),
		VersionRolRef:                         "rol:consulta-autoridad:v1",
		VersionRolHuellaSHA256:                strings.Repeat("b", 64),
		ControlVigenciaVersionRolRef:          "rol:consulta-autoridad:v1",
		ControlVigenciaVersionRolRevision:     1,
		ControlVigenciaVersionRolHuellaSHA256: strings.Repeat("c", 64),
		RevisionCatalogoPoliticas:             1,
		CatalogoPoliticasHuellaSHA256:         huellaCatalogoPoliticas,
		GarantiaMinima:                        a.garantiaMinima,
		CamposPermitidos:                      append([]string(nil), a.campos...),
		Obligaciones:                          append([]string(nil), a.obligaciones...),
		EmitidaEn:                             a.ahora.Add(-time.Second),
		ValidaHasta:                           a.ahora.Add(time.Minute),
	}
	if a.observar != nil {
		a.observar(datos)
	}
	if a.mutar != nil {
		a.mutar(&decision)
	}
	if a.despues != nil {
		a.despues()
	}
	return decision, nil
}

func (e *exigidorConsultaInternaAutoridadPrueba) ExigirEvidenciaUsoDecisionAutorizacionSolicitudLigadaV2(
	context.Context,
	domain.ContextoActor,
	domain.VinculoAutenticacionActorV1,
	domain.RecursoAutorizable,
	string,
	domain.ReferenciaEntradaCatalogo,
	PoliticaUsoDecisionAutorizacion,
) (ports.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2, error) {
	e.llamadas++
	return ports.EvidenciaUsoDecisionAutorizacionSolicitudLigadaV2{}, e.error
}

func TestConsultaInternaFuenteAutoridadAutorizaAntesDeRevelarYConservaEvidencia(t *testing.T) {
	servicio, repositorio, autorizador, orden := escenarioConsultaInternaAutoridadPrueba(t, true)
	autorizador.observar = func(datos domain.DatosSolicitudAutorizacionLigadaV2) {
		if datos.Accion != ports.AccionConsultarFuenteAutoridadInterna ||
			datos.Recurso.ModuloID != ports.ModuloFuentesAutoridad ||
			datos.Recurso.Tipo != ports.TipoRecursoFuenteAutoridad ||
			datos.Finalidad != FinalidadConsultaInternaFuenteAutoridad ||
			datos.ReferenciaMotivo != orden.MotivoCatalogo ||
			datos.CorrelacionRef != orden.CorrelacionRef ||
			datos.Recurso.Atributos["fuente_id"] != orden.Selector.FuenteID ||
			datos.Recurso.Atributos[ports.AtributoMotivoCatalogoIDConsultaAutoridad] != orden.MotivoCatalogo.CatalogoID ||
			datos.Recurso.Atributos[ports.AtributoMotivoCatalogoVersionConsultaAutoridad] != fmt.Sprint(orden.MotivoCatalogo.CatalogoVersion) ||
			datos.Recurso.Atributos[ports.AtributoMotivoCatalogoHuellaConsultaAutoridad] != orden.MotivoCatalogo.CatalogoHuellaSHA256 ||
			datos.Recurso.Atributos[ports.AtributoMotivoEntradaClaveConsultaAutoridad] != orden.MotivoCatalogo.EntradaClave {
			t.Fatalf("solicitud PDP fuera de politica: %v", datos)
		}
	}
	repositorio.antes = func(solicitud ports.SolicitudConsultaInternaGobernadaFuenteAutoridad) {
		if autorizador.llamadas != 1 {
			t.Fatalf("repositorio invocado antes del PDP: llamadas=%d", autorizador.llamadas)
		}
		evidencia, err := solicitud.Autorizacion()
		motivoCatalogo, errMotivo := solicitud.MotivoCatalogo()
		if err != nil || errMotivo != nil || motivoCatalogo != orden.MotivoCatalogo ||
			evidencia.ValidarEn(instanteConsultaInternaAutoridadPrueba) != nil ||
			evidencia.ValidarMotivo(motivoCatalogo) != nil {
			t.Fatalf("se descarto la evidencia: evidencia=%v error=%v", evidencia, err)
		}
		auditoria, err := solicitud.Auditoria()
		if err != nil || auditoria.ActorID != orden.ContextoActor.PersonaRef ||
			auditoria.Action != ports.AccionConsultarFuenteAutoridadInterna ||
			auditoria.Purpose != FinalidadConsultaInternaFuenteAutoridad ||
			auditoria.Result != "" || auditoria.Reason != orden.MotivoCatalogo.EntradaClave ||
			auditoria.Metadata[ports.AtributoMotivoCatalogoIDConsultaAutoridad] != orden.MotivoCatalogo.CatalogoID ||
			auditoria.Metadata[ports.AtributoMotivoCatalogoHuellaConsultaAutoridad] != orden.MotivoCatalogo.CatalogoHuellaSHA256 ||
			len(auditoria.ActorRoles) != 0 {
			t.Fatalf("auditoria previa incoherente: %+v, %v", auditoria, err)
		}
	}

	resultado, err := servicio.ConsultarExacta(context.Background(), orden)
	if err != nil || !resultado.Encontrada || repositorio.llamadas != 1 || autorizador.llamadas != 1 {
		t.Fatalf("consulta gobernada: resultado=%v repo=%d PDP=%d error=%v",
			resultado, repositorio.llamadas, autorizador.llamadas, err)
	}
	esperado, err := ports.EstadoExactoFuenteAutoridad(resultado.Fuente)
	datosRecibo, errRecibo := resultado.Recibo.Datos()
	if err != nil || errRecibo != nil || resultado.EstadoExacto != esperado ||
		datosRecibo.Resultado != ports.ResultadoConsultaFuenteEncontrada ||
		datosRecibo.Estado != esperado {
		t.Fatalf("resultado exacto invalido: estado=%+v recibo=%v error=%v",
			resultado.EstadoExacto, resultado.Recibo, errors.Join(err, errRecibo))
	}

	clon, err := resultado.Clonar()
	if err != nil {
		t.Fatal(err)
	}
	clon.Fuente.Contenido.Preceptos[0].Cita = "mutada en clon"
	if resultado.Fuente.Contenido.Preceptos[0].Cita == "mutada en clon" {
		t.Fatal("Clonar compartio memoria mutable")
	}
	valorRepositorio := repositorio.fuente.Contenido.Ambitos[0].ValoresClave[0]
	resultado.Fuente.Contenido.Ambitos[0].ValoresClave[0] = "mutada_fuera"
	if repositorio.fuente.Contenido.Ambitos[0].ValoresClave[0] != valorRepositorio {
		t.Fatal("el resultado compartio memoria con el repositorio")
	}
}

func TestConsultaInternaFuenteAutoridadExpresaAusenciaSoloTrasRecibo(t *testing.T) {
	servicio, repositorio, autorizador, orden := escenarioConsultaInternaAutoridadPrueba(t, false)
	resultado, err := servicio.ConsultarExacta(context.Background(), orden)
	datosRecibo, errRecibo := resultado.Recibo.Datos()
	if err != nil || errRecibo != nil || resultado.Encontrada || repositorio.llamadas != 1 || autorizador.llamadas != 1 ||
		datosRecibo.Resultado != ports.ResultadoConsultaFuenteNoEncontrada ||
		datosRecibo.Estado != (ports.ReferenciaEstadoFuenteAutoridad{}) ||
		resultado.Fuente.Validar() == nil || resultado.EstadoExacto.Validar() == nil {
		t.Fatalf("ausencia no explicita: resultado=%v repo=%d PDP=%d error=%v",
			resultado, repositorio.llamadas, autorizador.llamadas, err)
	}
	if _, err := resultado.Clonar(); err != nil {
		t.Fatalf("resultado ausente no clonable: %v", err)
	}
}

func TestConsultaInternaFuenteAutoridadNoTieneOraculoAntesDeAutorizar(t *testing.T) {
	for _, existe := range []bool{false, true} {
		t.Run(fmt.Sprintf("existe_%v", existe), func(t *testing.T) {
			repositorio := &consultaInternaGobernadaAutoridadPrueba{
				encontrada: existe, fuente: fuenteConsultaInternaAutoridadPrueba(t),
			}
			exigidor := &exigidorConsultaInternaAutoridadPrueba{error: domain.ErrAutorizacionDenegada}
			servicio, err := NuevoServicioConsultaInternaFuentesAutoridad(
				repositorio, exigidor, relojUsoAutorizacionPrueba{ahora: instanteConsultaInternaAutoridadPrueba},
			)
			if err != nil {
				t.Fatal(err)
			}
			orden := ordenConsultaInternaAutoridadPrueba()
			_, err = servicio.ConsultarExacta(context.Background(), orden)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || repositorio.llamadas != 0 ||
				exigidor.llamadas != 1 {
				t.Fatalf("oraculo previo: repo=%d PEP=%d error=%v",
					repositorio.llamadas, exigidor.llamadas, err)
			}
		})
	}
}

func TestConsultaInternaFuenteAutoridadFallaAntesDelRepositorio(t *testing.T) {
	servicio, repositorio, autorizador, base := escenarioConsultaInternaAutoridadPrueba(t, true)
	cancelado, cancelar := context.WithCancel(context.Background())
	cancelar()
	casos := []struct {
		nombre string
		ctx    context.Context
		mutar  func(*OrdenConsultaInternaExactaFuenteAutoridad)
	}{
		{"contexto nulo", nil, func(*OrdenConsultaInternaExactaFuenteAutoridad) {}},
		{"contexto cancelado", cancelado, func(*OrdenConsultaInternaExactaFuenteAutoridad) {}},
		{"actor cero", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) { o.ContextoActor = domain.ContextoActor{} }},
		{"rol declarado", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.ContextoActor.Principal.Roles = []string{"administrador"}
		}},
		{"vinculo cero", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.VinculoAutenticacionActor = domain.VinculoAutenticacionActorV1{}
		}},
		{"selector cero", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.Selector = ports.SelectorVersionFuenteAutoridad{}
		}},
		{"selector no canonico", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) { o.Selector.FuenteID = "RPT_2026" }},
		{"referencia motivo cero", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.MotivoCatalogo = domain.ReferenciaEntradaCatalogo{}
		}},
		{"clave catalogada semantica", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.MotivoCatalogo.EntradaClave = "consulta_tecnica"
		}},
		{"catalogo con texto libre", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.MotivoCatalogo.CatalogoID = "Motivos de RRHH"
		}},
		{"clave con posible PII", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.MotivoCatalogo.EntradaClave = "dni_12345678z"
		}},
		{"clave opaca corta", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.MotivoCatalogo.EntradaClave = "motivo_0123456789abcdef0123456789abcde"
		}},
		{"clave opaca no hexadecimal", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.MotivoCatalogo.EntradaClave = "motivo_0123456789abcdef0123456789abcdeg"
		}},
		{"huella catalogo nula", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.MotivoCatalogo.CatalogoHuellaSHA256 = ""
		}},
		{"correlacion vacia", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) { o.CorrelacionRef = "" }},
		{"correlacion semantica anterior", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.CorrelacionRef = "correlacion:autoridad:consulta:0001"
		}},
		{"correlacion opaca corta", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.CorrelacionRef = "correlacion_0123456789abcdef0123456789abcde"
		}},
		{"correlacion opaca mayuscula", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.CorrelacionRef = "correlacion_0123456789ABCDEF0123456789abcdef"
		}},
		{"correlacion opaca no hexadecimal", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) {
			o.CorrelacionRef = "correlacion_0123456789abcdef0123456789abcdeg"
		}},
		{"correlacion con espacio", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) { o.CorrelacionRef = "correlacion no segura" }},
		{"correlacion con control", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) { o.CorrelacionRef = "correlacion:\n" }},
		{"correlacion unicode", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) { o.CorrelacionRef = "correlacion:á" }},
		{"correlacion comodin", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) { o.CorrelacionRef = "correlacion:*" }},
		{"correlacion excesiva", context.Background(), func(o *OrdenConsultaInternaExactaFuenteAutoridad) { o.CorrelacionRef = strings.Repeat("a", 513) }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			repositorio.llamadas, autorizador.llamadas = 0, 0
			orden := base
			caso.mutar(&orden)
			_, err := servicio.ConsultarExacta(caso.ctx, orden)
			if err == nil || repositorio.llamadas != 0 || autorizador.llamadas != 0 {
				t.Fatalf("precondicion aceptada: repo=%d PDP=%d error=%v",
					repositorio.llamadas, autorizador.llamadas, err)
			}
		})
	}
}

func escenarioConsultaInternaAutoridadPrueba(
	t *testing.T,
	encontrada bool,
) (*ServicioConsultaInternaFuentesAutoridad, *consultaInternaGobernadaAutoridadPrueba,
	*autorizadorConsultaInternaAutoridadV2Prueba, OrdenConsultaInternaExactaFuenteAutoridad,
) {
	t.Helper()
	fuente := fuenteConsultaInternaAutoridadPrueba(t)
	repositorio := &consultaInternaGobernadaAutoridadPrueba{encontrada: encontrada, fuente: fuente}
	autorizador := &autorizadorConsultaInternaAutoridadV2Prueba{
		ahora:          instanteConsultaInternaAutoridadPrueba,
		campos:         []string{ports.CampoConsultaInternaFuenteAutoridad},
		garantiaMinima: domain.AuthAssuranceHigh,
	}
	fachada := nuevaFachadaUsoAutorizacionV2AutoridadPrueba(
		t, autorizador, relojUsoAutorizacionPrueba{ahora: instanteConsultaInternaAutoridadPrueba},
	)
	servicio, err := NuevoServicioConsultaInternaFuentesAutoridad(
		repositorio, fachada, relojUsoAutorizacionPrueba{ahora: instanteConsultaInternaAutoridadPrueba},
	)
	if err != nil {
		t.Fatal(err)
	}
	return servicio, repositorio, autorizador, ordenConsultaInternaAutoridadPrueba()
}

func nuevaFachadaUsoAutorizacionV2AutoridadPrueba(
	t testing.TB,
	autorizador ports.AutorizadorSolicitudLigadaV2,
	reloj ports.Reloj,
) *FachadaUsoDecisionAutorizacionSolicitudLigadaV2 {
	t.Helper()
	fachada, err := NuevaFachadaUsoDecisionAutorizacionSolicitudLigadaV2(autorizador, reloj)
	if err != nil {
		t.Fatalf("crear fachada V2 de autoridad: %v", err)
	}
	return fachada
}

func ordenConsultaInternaAutoridadPrueba() OrdenConsultaInternaExactaFuenteAutoridad {
	actor, vinculo := contextoYVinculoAutenticacionAplicacionPrueba(instanteConsultaInternaAutoridadPrueba)
	return OrdenConsultaInternaExactaFuenteAutoridad{
		ContextoActor: actor, VinculoAutenticacionActor: vinculo,
		Selector: ports.SelectorVersionFuenteAutoridad{FuenteID: "rpt_historica_2026", Version: 1},
		MotivoCatalogo: domain.ReferenciaEntradaCatalogo{
			CatalogoID: "motivos_consulta_fuentes_autoridad", CatalogoVersion: 3,
			CatalogoHuellaSHA256: strings.Repeat("d", 64),
			EntradaClave:         "motivo_0123456789abcdef0123456789abcdef",
		},
		CorrelacionRef: "correlacion_0123456789abcdef0123456789abcdef",
	}
}

func fuenteConsultaInternaAutoridadPrueba(t testing.TB) domain.FuenteAutoridadVersionada {
	t.Helper()
	creadaEn := instanteConsultaInternaAutoridadPrueba.Add(-time.Hour)
	fuente, err := domain.NuevaFuenteAutoridadBorradorV1(domain.DatosAltaFuenteAutoridadV1{
		ID: "rpt_historica_2026",
		Contenido: domain.ContenidoFuenteAutoridad{
			MateriaClave: "plantilla_rpt", Nombre: "Relacion de puestos historica",
			Ambitos: []domain.AmbitoFuenteAutoridad{{
				DimensionClave: "entidad", ValoresClave: []string{"diputacion_granada"},
			}},
			Documento: domain.DocumentoFuenteAutoridad{
				DocumentoID: "doc:rpt:2026", DocumentoVersion: 1,
				RepresentacionRef: "rep:pdfa:rpt:2026", HuellaContenidoSHA256: strings.Repeat("a", 64),
				PublicacionOficialRef: "bop:granada:2026:10", ActoOrigenRef: "acto:pleno:rpt:2026",
				OrganoEmisorRef: "organo:diputacion:pleno",
			},
			Preceptos:  []domain.PreceptoFuenteAutoridad{{Clave: "anexo_rpt", Cita: "Anexo RPT"}},
			Vigencia:   domain.PeriodoFuenteAutoridad{Desde: creadaEn.Add(-24 * time.Hour)},
			Efectos:    domain.PeriodoFuenteAutoridad{Desde: creadaEn.Add(-24 * time.Hour)},
			ConocidaEn: creadaEn.Add(-time.Hour),
		},
		CreadaPor: "per_creador_consulta_00000000001", CreadaEn: creadaEn,
		MotivoCreacionCodigo: "incorporacion_historica",
	})
	if err != nil {
		t.Fatalf("crear fuente: %v", err)
	}
	return fuente
}
