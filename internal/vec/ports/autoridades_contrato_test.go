package ports

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

var instantePuertoAutoridadPrueba = time.Date(2026, time.July, 17, 9, 0, 0, 123_456_000, time.UTC)

type escenarioPuertoAutoridad struct {
	Fuente    domain.FuenteAutoridadVersionada
	Estado    ReferenciaEstadoFuenteAutoridad
	Solicitud domain.SolicitudTransicionFuenteAutoridadV1
	Evidencia domain.EvidenciaActoFuenteAutoridad
	Comprobar SolicitudComprobarActoFuenteAutoridad
	Datos     DatosAtestacionActoFuenteAutoridad
}

func nuevoEscenarioPuertoAutoridad(t testing.TB) escenarioPuertoAutoridad {
	t.Helper()
	creadaEn := instantePuertoAutoridadPrueba
	fuente, err := domain.NuevaFuenteAutoridadBorradorV1(domain.DatosAltaFuenteAutoridadV1{
		ID: "rpt_historica_puerto",
		Contenido: domain.ContenidoFuenteAutoridad{
			MateriaClave: "plantilla_rpt", Nombre: "Relación de puestos histórica",
			Ambitos: []domain.AmbitoFuenteAutoridad{
				{DimensionClave: "entidad", ValoresClave: []string{"diputacion_granada"}},
			},
			Documento: domain.DocumentoFuenteAutoridad{
				DocumentoID: "doc:rpt:2020", DocumentoVersion: 1,
				RepresentacionRef: "rep:pdfa:rpt:2020", HuellaContenidoSHA256: huellaPuertoAutoridadPrueba('a'),
				PublicacionOficialRef: "bop:granada:2020:10", ActoOrigenRef: "acto:pleno:rpt:2020",
				OrganoEmisorRef: "organo:diputacion:pleno",
			},
			Preceptos:  []domain.PreceptoFuenteAutoridad{{Clave: "anexo_rpt", Cita: "Anexo RPT"}},
			Vigencia:   domain.PeriodoFuenteAutoridad{Desde: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)},
			Efectos:    domain.PeriodoFuenteAutoridad{Desde: time.Date(2020, 2, 1, 0, 0, 0, 0, time.UTC)},
			ConocidaEn: creadaEn.Add(-time.Hour),
		},
		CreadaPor: "per_creador_puerto_00000000001", CreadaEn: creadaEn,
		MotivoCreacionCodigo: "incorporacion_historica",
	})
	if err != nil {
		t.Fatalf("crear fuente: %v", err)
	}
	estado, err := EstadoExactoFuenteAutoridad(fuente)
	if err != nil {
		t.Fatalf("crear snapshot OCC: %v", err)
	}
	preparadaEn := creadaEn.Add(time.Hour)
	solicitud, err := fuente.PrepararSolicitudTransicionV1(domain.DatosPreparacionTransicionFuenteAutoridadV1{
		EstadoNuevo: domain.EstadoFuenteAutoridadPublicada,
		ActorRef:    "per_publicador_puerto_000000001", MotivoCodigo: "publicacion_validada",
		SolicitudRef: "solicitud:autoridad:puerto:1", PreparadaEn: preparadaEn,
		ExpiraEn: preparadaEn.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("preparar solicitud: %v", err)
	}
	comprobadaEn := preparadaEn.Add(10 * time.Minute)
	mensaje, err := domain.PrepararMensajeAtestacionActoFuenteAutoridadV1(
		solicitud,
		domain.DatosMensajeAtestacionActoFuenteAutoridadV1{
			EvidenciaRef: "evidencia:autoridad:puerto:1", ActoRef: "acto:pleno:rpt:2020",
			DocumentoRef: "doc:rpt:2020", RepresentacionRef: "rep:pdfa:rpt:2020",
			HuellaDocumentoSHA256: huellaPuertoAutoridadPrueba('b'),
			OrganoRef:             "organo:diputacion:pleno",
			FirmasRefs:            []string{"firma:presidencia:rpt:2020", "firma:secretaria:rpt:2020"},
			ComprobadorRef:        "conector:afirma:produccion",
			ActoOcurridoEn:        time.Date(2020, 1, 15, 12, 0, 0, 0, time.UTC),
			ComprobadaEn:          comprobadaEn,
		},
	)
	if err != nil {
		t.Fatalf("preparar mensaje: %v", err)
	}
	evidencia, err := mensaje.ConstituirEvidenciaAtestadaV1(domain.DatosSobreAtestacionActoFuenteAutoridadV1{
		AtestacionRef:          "atestacion:acto:autoridad:1",
		HuellaAtestacionSHA256: huellaPuertoAutoridadPrueba('c'),
		FirmaAtestacionRef:     "firma:atestacion:autoridad:1",
	})
	if err != nil {
		t.Fatalf("constituir evidencia: %v", err)
	}
	comprobarEn := comprobadaEn.Add(2 * time.Minute)
	comprobar := SolicitudComprobarActoFuenteAutoridad{
		Solicitud: solicitud, EstadoEsperado: estado, ComprobarEn: comprobarEn,
	}
	datos := DatosAtestacionActoFuenteAutoridad{
		Evidencia: evidencia, RevisionEsperada: estado.Revision,
		HuellaEstadoEsperadoSHA256:     estado.HuellaEstadoSHA256,
		VerificadorRef:                 "verificador:firmas:corporativo",
		RegistroAtestacionRef:          "registro:atestacion:autoridad:1",
		HuellaRegistroAtestacionSHA256: huellaPuertoAutoridadPrueba('d'),
		TokenConsumoRef:                "consumo:atestacion:autoridad:1",
		EmitidaEn:                      comprobadaEn.Add(time.Minute), ValidaHasta: comprobadaEn.Add(6 * time.Minute),
	}
	return escenarioPuertoAutoridad{
		Fuente: fuente, Estado: estado, Solicitud: solicitud, Evidencia: evidencia,
		Comprobar: comprobar, Datos: datos,
	}
}

func TestContratoConsultaYEstadoExactoFuenteAutoridad(t *testing.T) {
	escenario := nuevoEscenarioPuertoAutoridad(t)
	if err := (SelectorVersionFuenteAutoridad{
		FuenteID: escenario.Fuente.ID, Version: escenario.Fuente.Version,
	}).Validar(); err != nil {
		t.Fatalf("selector exacto rechazado: %v", err)
	}
	consultaHistoria := ConsultaPaginaHistoriaFuenteAutoridad{
		FuenteID: escenario.Fuente.ID, DesdeVersion: 1, Limite: 1,
	}
	if err := consultaHistoria.Validar(); err != nil {
		t.Fatalf("consulta paginada de historia rechazada: %v", err)
	}
	pagina := PaginaHistoriaFuenteAutoridad{
		Versiones: []domain.FuenteAutoridadVersionada{escenario.Fuente}, HayMas: true, SiguienteVersion: 2,
	}
	clonPagina, err := pagina.ClonarPara(consultaHistoria)
	if err != nil || clonPagina.ValidarPara(consultaHistoria) != nil {
		t.Fatalf("pagina continua rechazada: %+v, %v", clonPagina, err)
	}
	clonPagina.Versiones[0].ID = "alterada"
	if pagina.Versiones[0].ID != escenario.Fuente.ID {
		t.Fatal("la pagina compartio agregados mutables")
	}
	pagina.SiguienteVersion = 3
	if pagina.ValidarPara(consultaHistoria) == nil {
		t.Fatal("pagina con continuidad rota aceptada")
	}
	for _, invalida := range []ConsultaPaginaHistoriaFuenteAutoridad{
		{FuenteID: escenario.Fuente.ID, Limite: 1},
		{FuenteID: escenario.Fuente.ID, DesdeVersion: 1},
		{FuenteID: escenario.Fuente.ID, DesdeVersion: 1, Limite: MaximoVersionesPaginaFuenteAutoridad + 1},
	} {
		if invalida.Validar() == nil {
			t.Fatalf("consulta no acotada aceptada: %+v", invalida)
		}
	}
	for nombre, selector := range map[string]SelectorVersionFuenteAutoridad{
		"id vacio": {}, "version cero": {FuenteID: escenario.Fuente.ID},
		"id no canonico": {FuenteID: "RPT histórica", Version: 1},
	} {
		t.Run(nombre, func(t *testing.T) {
			if !errors.Is(selector.Validar(), ErrConsultaFuenteAutoridadInvalida) {
				t.Fatalf("selector invalido aceptado: %+v", selector)
			}
		})
	}
	if escenario.Estado.Fuente.FuenteID != escenario.Fuente.ID ||
		escenario.Estado.Revision != escenario.Fuente.Revision ||
		escenario.Estado.Estado != escenario.Fuente.Estado || escenario.Estado.Validar() != nil {
		t.Fatalf("snapshot OCC incompleto: %+v", escenario.Estado)
	}
	alterado := escenario.Estado
	alterado.HuellaEstadoSHA256 = strings.Repeat("0", 64)
	if !errors.Is(alterado.Validar(), ErrEstadoFuenteAutoridadInvalido) {
		t.Fatal("se acepto la huella nula")
	}
	alterado = escenario.Estado
	alterado.HuellaHistoriaSHA256 = huellaPuertoAutoridadPrueba('9')
	if alterado.Validar() != nil {
		t.Fatal("la precondicion de historia de prueba debe ser estructuralmente valida")
	}
	if _, err := NuevaOperacionPendienteFuenteAutoridad(
		"operacion:historia:distinta", escenario.Solicitud, alterado,
	); !errors.Is(err, ErrOperacionFuenteAutoridadInvalida) {
		t.Fatalf("operacion ligada a otra cabeza de historia aceptada: %v", err)
	}
	comprobacionAlterada := escenario.Comprobar
	comprobacionAlterada.EstadoEsperado = alterado
	if err := comprobacionAlterada.Validar(); !errors.Is(err, ErrSolicitudComprobacionActoAutoridadInvalida) {
		t.Fatalf("comprobacion ligada a otra cabeza de historia aceptada: %v", err)
	}
	for _, referencia := range []string{"ref:segura:1", "ref/segura@1+v2"} {
		if !referenciaPuertoAutoridadValida(referencia) {
			t.Fatalf("referencia ASCII valida rechazada: %q", referencia)
		}
	}
	for _, referencia := range []string{"ref:ámbito:1", "ref:*", "ref con espacio", "ref:\u202eoculta"} {
		if referenciaPuertoAutoridadValida(referencia) {
			t.Fatalf("referencia ambigua aceptada: %q", referencia)
		}
	}
	if err := (SelectorOperacionFuenteAutoridad{
		OperacionRef: "operacion:autoridad:1",
	}).Validar(); err != nil {
		t.Fatalf("selector de operacion valido rechazado: %v", err)
	}
	for _, referencia := range []string{"", "operacion con espacio", "operacion:\u202eoculta"} {
		if err := (SelectorOperacionFuenteAutoridad{OperacionRef: referencia}).Validar(); !errors.Is(err, ErrOperacionFuenteAutoridadInvalida) {
			t.Fatalf("selector de operacion invalido aceptado: %q", referencia)
		}
	}
}

func TestOperacionFuenteAutoridadCierraEstadosYDefiendeCopias(t *testing.T) {
	escenario := nuevoEscenarioPuertoAutoridad(t)
	operacion, err := NuevaOperacionPendienteFuenteAutoridad(
		"operacion:autoridad:1", escenario.Solicitud, escenario.Estado,
	)
	if err != nil || operacion.Terminal() {
		t.Fatalf("crear pendiente: operacion=%v error=%v", operacion, err)
	}
	datos, err := operacion.Datos()
	if err != nil || datos.Estado != EstadoOperacionFuenteAutoridadPendiente ||
		!datos.PreparadaEn.Equal(datos.ActualizadaEn) {
		t.Fatalf("datos pendientes invalidos: %+v, %v", datos, err)
	}
	datos.Solicitud = domain.SolicitudTransicionFuenteAutoridadV1{}
	segundaLectura, err := operacion.Datos()
	if err != nil || segundaLectura.Solicitud.Validar() != nil {
		t.Fatal("la proyeccion compartio la solicitud mutable")
	}
	if _, err := json.Marshal(operacion); !errors.Is(err, ErrSerializacionOperacionAutoridad) {
		t.Fatalf("JSON de operacion no bloqueado: %v", err)
	}
	if _, err := json.Marshal(segundaLectura); !errors.Is(err, ErrSerializacionOperacionAutoridad) {
		t.Fatalf("JSON de datos no bloqueado: %v", err)
	}
	if _, err := operacion.MarshalBinary(); !errors.Is(err, ErrSerializacionOperacionAutoridad) {
		t.Fatalf("binario de operacion no bloqueado: %v", err)
	}
	if _, err := operacion.GobEncode(); !errors.Is(err, ErrSerializacionOperacionAutoridad) {
		t.Fatalf("gob de operacion no bloqueado: %v", err)
	}
	if _, err := xml.Marshal(operacion); !errors.Is(err, ErrSerializacionOperacionAutoridad) {
		t.Fatalf("XML de operacion no bloqueado: %v", err)
	}
	if texto := fmt.Sprintf("%v %+v %#v", operacion, operacion, operacion); strings.Contains(texto, "operacion:autoridad:1") {
		t.Fatalf("formato expuso referencias: %s", texto)
	}

	compromiso, _ := escenario.Solicitud.Compromiso()
	base := segundaLectura
	casos := []struct {
		nombre   string
		estado   EstadoOperacionFuenteAutoridad
		instante time.Time
		atestar  bool
		resolver bool
		terminal bool
	}{
		{"atestada", EstadoOperacionFuenteAutoridadAtestada, compromiso.PreparadaEn.Add(time.Minute), true, false, false},
		{"confirmada", EstadoOperacionFuenteAutoridadConfirmada, compromiso.PreparadaEn.Add(2 * time.Minute), true, true, true},
		{"cancelada", EstadoOperacionFuenteAutoridadCancelada, compromiso.PreparadaEn.Add(2 * time.Minute), false, true, true},
		{"expirada", EstadoOperacionFuenteAutoridadExpirada, compromiso.ExpiraEn, false, true, true},
		{"obsoleta", EstadoOperacionFuenteAutoridadObsoleta, compromiso.PreparadaEn.Add(2 * time.Minute), false, true, true},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			candidatos := base
			candidatos.Estado, candidatos.ActualizadaEn = caso.estado, caso.instante
			if caso.atestar {
				candidatos.AtestacionRef = "atestacion:operacion:" + caso.nombre
				candidatos.HuellaAtestacionSHA256 = huellaPuertoAutoridadPrueba('e')
			}
			if caso.resolver {
				candidatos.ResolucionRef = "resolucion:operacion:" + caso.nombre
			}
			rehidratada, err := RehidratarOperacionFuenteAutoridad(candidatos)
			if err != nil || rehidratada.Terminal() != caso.terminal {
				t.Fatalf("estado cerrado rechazado: terminal=%v error=%v", rehidratada.Terminal(), err)
			}
		})
	}
	parcial := base
	parcial.Estado = EstadoOperacionFuenteAutoridadAtestada
	parcial.ActualizadaEn = compromiso.PreparadaEn.Add(time.Minute)
	parcial.AtestacionRef = "atestacion:sin:huella"
	if _, err := RehidratarOperacionFuenteAutoridad(parcial); !errors.Is(err, ErrOperacionFuenteAutoridadInvalida) {
		t.Fatalf("atestacion parcial aceptada: %v", err)
	}
}

func TestAtestacionActoAutoridadEsExactaOpacaYDefensiva(t *testing.T) {
	escenario := nuevoEscenarioPuertoAutoridad(t)
	atestacion, err := NuevaAtestacionActoFuenteAutoridad(escenario.Comprobar, escenario.Datos)
	if err != nil {
		t.Fatalf("crear atestacion: %v", err)
	}
	instanteUso := escenario.Datos.EmitidaEn.Add(time.Minute)
	if err := atestacion.ValidarPara(escenario.Comprobar, instanteUso); err != nil {
		t.Fatalf("atestacion exacta rechazada: %v", err)
	}
	datos, err := atestacion.DatosParaConsumo()
	if err != nil {
		t.Fatal(err)
	}
	datos.Evidencia.FirmasRefs[0] = "firma:alterada"
	datosOtraVez, err := atestacion.DatosParaConsumo()
	if err != nil || datosOtraVez.Evidencia.FirmasRefs[0] == "firma:alterada" {
		t.Fatal("la atestacion compartio memoria mutable")
	}
	for nombre, valor := range map[string]any{
		"solicitud": escenario.Comprobar, "atestacion": atestacion, "datos": datosOtraVez,
	} {
		t.Run("serializacion/"+nombre, func(t *testing.T) {
			if _, err := json.Marshal(valor); !errors.Is(err, ErrSerializacionAtestacionActoAutoridad) {
				t.Fatalf("JSON no bloqueado: %v", err)
			}
		})
	}
	if _, err := atestacion.MarshalBinary(); !errors.Is(err, ErrSerializacionAtestacionActoAutoridad) {
		t.Fatalf("binario de atestacion no bloqueado: %v", err)
	}
	if _, err := atestacion.GobEncode(); !errors.Is(err, ErrSerializacionAtestacionActoAutoridad) {
		t.Fatalf("gob de atestacion no bloqueado: %v", err)
	}
	if _, err := xml.Marshal(atestacion); !errors.Is(err, ErrSerializacionAtestacionActoAutoridad) {
		t.Fatalf("XML de atestacion no bloqueado: %v", err)
	}
	texto := fmt.Sprintf("%v %+v %#v", atestacion, atestacion, atestacion)
	if strings.Contains(texto, escenario.Datos.RegistroAtestacionRef) ||
		strings.Contains(texto, escenario.Datos.TokenConsumoRef) {
		t.Fatalf("formato expuso la capacidad: %s", texto)
	}
	if err := atestacion.ValidarPara(escenario.Comprobar, escenario.Datos.ValidaHasta); !errors.Is(err, ErrAtestacionActoAutoridadInvalida) {
		t.Fatalf("limite exclusivo aceptado: %v", err)
	}

	alterada := escenario.Datos
	alterada.Evidencia.HuellaCompromisoSHA256 = huellaPuertoAutoridadPrueba('f')
	if _, err := NuevaAtestacionActoFuenteAutoridad(escenario.Comprobar, alterada); !errors.Is(err, ErrAtestacionActoAutoridadInvalida) {
		t.Fatalf("evidencia para otro compromiso aceptada: %v", err)
	}
	otroEstado := escenario.Comprobar
	otroEstado.EstadoEsperado.HuellaEstadoSHA256 = huellaPuertoAutoridadPrueba('1')
	if _, err := NuevaAtestacionActoFuenteAutoridad(otroEstado, escenario.Datos); !errors.Is(err, ErrAtestacionActoAutoridadInvalida) {
		t.Fatalf("snapshot OCC distinto aceptado: %v", err)
	}
	duplicada := escenario.Datos
	duplicada.TokenConsumoRef = duplicada.RegistroAtestacionRef
	if _, err := NuevaAtestacionActoFuenteAutoridad(escenario.Comprobar, duplicada); !errors.Is(err, ErrAtestacionActoAutoridadInvalida) {
		t.Fatalf("token no separado aceptado: %v", err)
	}
	demasiadoLarga := escenario.Datos
	demasiadoLarga.ValidaHasta = demasiadoLarga.EmitidaEn.Add(
		VigenciaMaximaAtestacionActoFuenteAutoridad + time.Microsecond,
	)
	if _, err := NuevaAtestacionActoFuenteAutoridad(escenario.Comprobar, demasiadoLarga); !errors.Is(err, ErrAtestacionActoAutoridadInvalida) {
		t.Fatalf("atestacion con vigencia excesiva aceptada: %v", err)
	}
	noCanonica := escenario.Comprobar
	noCanonica.ComprobarEn = noCanonica.ComprobarEn.Add(time.Nanosecond)
	if noCanonica.Validar() == nil {
		t.Fatal("instante no canonico aceptado")
	}
}

func TestAtestacionActoAutoridadAdmiteLecturasConcurrentes(t *testing.T) {
	escenario := nuevoEscenarioPuertoAutoridad(t)
	atestacion, err := NuevaAtestacionActoFuenteAutoridad(escenario.Comprobar, escenario.Datos)
	if err != nil {
		t.Fatal(err)
	}
	const lectores = 16
	var grupo sync.WaitGroup
	errores := make(chan error, lectores)
	for indice := 0; indice < lectores; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			datos, err := atestacion.DatosParaConsumo()
			if err == nil {
				datos.Evidencia.FirmasRefs[0] = "firma:local"
			}
			errores <- err
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		if err != nil {
			t.Fatalf("lectura concurrente: %v", err)
		}
	}
}

type consultaVersionFuenteAutoridadContrato struct{}

func (consultaVersionFuenteAutoridadContrato) ObtenerVersion(
	context.Context,
	SelectorVersionFuenteAutoridad,
) (domain.FuenteAutoridadVersionada, error) {
	return domain.FuenteAutoridadVersionada{}, nil
}

func (consultaVersionFuenteAutoridadContrato) ObtenerPorReferencia(
	context.Context,
	domain.ReferenciaFuenteAutoridad,
) (domain.FuenteAutoridadVersionada, error) {
	return domain.FuenteAutoridadVersionada{}, nil
}

func (consultaVersionFuenteAutoridadContrato) ObtenerPorCita(
	context.Context,
	domain.CitaFuenteAutoridad,
) (domain.FuenteAutoridadVersionada, error) {
	return domain.FuenteAutoridadVersionada{}, nil
}

func (consultaVersionFuenteAutoridadContrato) ListarVersiones(
	context.Context,
	ConsultaPaginaHistoriaFuenteAutoridad,
) (PaginaHistoriaFuenteAutoridad, error) {
	return PaginaHistoriaFuenteAutoridad{}, nil
}

func (consultaVersionFuenteAutoridadContrato) ObtenerOperacion(
	context.Context,
	SelectorOperacionFuenteAutoridad,
) (OperacionFuenteAutoridad, error) {
	return OperacionFuenteAutoridad{}, nil
}

type comprobadorActoFuenteAutoridadContrato struct{}

func (comprobadorActoFuenteAutoridadContrato) ComprobarActo(
	context.Context,
	SolicitudComprobarActoFuenteAutoridad,
) (AtestacionActoFuenteAutoridad, error) {
	return AtestacionActoFuenteAutoridad{}, nil
}

var _ ConsultaVersionFuenteAutoridad = consultaVersionFuenteAutoridadContrato{}
var _ ConsultaReferenciaFuenteAutoridad = consultaVersionFuenteAutoridadContrato{}
var _ ConsultaCitaFuenteAutoridad = consultaVersionFuenteAutoridadContrato{}
var _ ConsultaHistoriaFuentesAutoridad = consultaVersionFuenteAutoridadContrato{}
var _ ConsultaOperacionesFuentesAutoridad = consultaVersionFuenteAutoridadContrato{}
var _ ComprobadorActosFuentesAutoridad = comprobadorActoFuenteAutoridadContrato{}

func huellaPuertoAutoridadPrueba(caracter byte) string {
	return strings.Repeat(string(caracter), 64)
}
