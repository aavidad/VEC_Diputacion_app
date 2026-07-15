package ports

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func solicitudAnalisisContenidoValida(t *testing.T) SolicitudAnalizarContenido {
	t.Helper()
	objeto := ReferenciaObjetoAlmacen{Referencia: "objeto:opaco:uno", Version: "version:uno"}
	decision, recurso, vinculos, instante := autorizacionAlmacenPrueba(
		t, AccionNegocioAnalizarCargaDocumental, []string{"analisis_seguridad", "estado"}, true,
	)
	vinculos.ObjetoVinculado = objeto
	recurso.Atributos[AtributoAlmacenObjetoRef] = objeto.Referencia
	recurso.Atributos[AtributoAlmacenObjetoVersion] = objeto.Version
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decision.ContextoRecursoHuellaSHA256 = huella
	lectura, err := NuevoContextoAnalizarCargaDocumentalAlmacen(decision, recurso, vinculos, instante)
	if err != nil {
		t.Fatal(err)
	}
	contexto, err := lectura.DerivarPaso(PasoAlmacenAnalizarContenido)
	if err != nil {
		t.Fatal(err)
	}
	return SolicitudAnalizarContenido{
		Contexto:          contexto,
		Objeto:            objeto,
		ConectorAlmacenID: "almacen_s3_corporativo",
		Zona:              ZonaAlmacenCuarentena,
		MIME:              "application/pdf",
		Tamano:            9,
		HuellaSHA256:      strings.Repeat("a", 64),
		Contenido:         bytes.NewReader([]byte("contenido")),
	}
}

func resultadoAnalisisContenidoLimpio(t *testing.T, contexto ContextoOperacionAlmacen) ResultadoAnalisisContenido {
	t.Helper()
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		t.Fatal(err)
	}
	inicio := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	return ResultadoAnalisisContenido{
		Objeto:            ReferenciaObjetoAlmacen{Referencia: "objeto:opaco:uno", Version: "version:uno"},
		ConectorAlmacenID: "almacen_s3_corporativo", HuellaObjetoSHA256: strings.Repeat("a", 64),
		TamanoObjeto: 9, MIMEDeclarado: "application/pdf", MIMEDetectado: "application/pdf",
		ConectorAnalizadorID: "antivirus_corporativo", VersionConector: 2,
		MotorRef: "motor:antivirus:corporativo", VersionMotor: "2.4.1", FirmasRef: "firmas:20260715:1200",
		Estado: EstadoAnalisisContenidoLimpio, CodigoResultado: "sin_detecciones", BytesAnalizados: 9,
		EvidenciaRef: "evidencia:antivirus:uno", HuellaEvidenciaSHA256: strings.Repeat("b", 64),
		AnalisisIniciadoEn: inicio, AnalisisCompletadoEn: inicio.Add(time.Second),
		CorrelacionRef: proyeccion.CorrelacionRef, AutorizacionRef: proyeccion.AutorizacionRef,
		Finalidad: proyeccion.Finalidad, Clasificacion: proyeccion.Clasificacion,
	}
}

func TestAnalizadorContenidoFallaCerradoYVinculaElObjetoExacto(t *testing.T) {
	solicitud := solicitudAnalisisContenidoValida(t)
	resultado := resultadoAnalisisContenidoLimpio(t, solicitud.Contexto)
	if err := solicitud.Validar(); err != nil {
		t.Fatalf("solicitud valida: %v", err)
	}
	if err := resultado.ValidarContra(solicitud); err != nil {
		t.Fatalf("resultado valido: %v", err)
	}

	pruebas := []struct {
		nombre string
		muta   func(*ResultadoAnalisisContenido)
	}{
		{"otro objeto", func(r *ResultadoAnalisisContenido) { r.Objeto.Referencia = "objeto:opaco:otro" }},
		{"otra huella", func(r *ResultadoAnalisisContenido) { r.HuellaObjetoSHA256 = strings.Repeat("c", 64) }},
		{"tamano parcial", func(r *ResultadoAnalisisContenido) { r.BytesAnalizados-- }},
		{"otra autorizacion", func(r *ResultadoAnalisisContenido) { r.AutorizacionRef = "autorizacion:otra" }},
		{"limpio con deteccion", func(r *ResultadoAnalisisContenido) {
			r.Detecciones = []DeteccionContenido{{Clase: ClaseDeteccionMalware, Codigo: "firma_detectada", FirmaRef: "firma:uno"}}
		}},
		{"malicioso sin deteccion", func(r *ResultadoAnalisisContenido) { r.Estado = EstadoAnalisisContenidoMalicioso }},
		{"estado libre", func(r *ResultadoAnalisisContenido) { r.Estado = "omitido" }},
		{"mime detectado no canonico", func(r *ResultadoAnalisisContenido) { r.MIMEDetectado = "Application/PDF" }},
		{"limpio con mime distinto", func(r *ResultadoAnalisisContenido) { r.MIMEDetectado = "application/zip" }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			alterado := resultado
			alterado.Detecciones = append([]DeteccionContenido(nil), resultado.Detecciones...)
			prueba.muta(&alterado)
			if !errors.Is(alterado.ValidarContra(solicitud), ErrResultadoAnalisisContenidoInvalido) {
				t.Fatalf("la alteracion %q fue aceptada", prueba.nombre)
			}
		})
	}

	noConcluyente := resultado
	noConcluyente.Estado = EstadoAnalisisContenidoNoConcluyente
	noConcluyente.CodigoResultado = "motor_no_disponible"
	noConcluyente.MotorRef = ""
	noConcluyente.VersionMotor = ""
	noConcluyente.FirmasRef = ""
	noConcluyente.BytesAnalizados = 0
	if err := noConcluyente.Validar(); err != nil {
		t.Fatalf("un fallo debe conservarse como no concluyente valido: %v", err)
	}
	if noConcluyente.Estado == EstadoAnalisisContenidoLimpio {
		t.Fatal("un fallo del motor nunca equivale a limpio")
	}
}

func TestSolicitudAnalisisSoloAdmiteAccionExplicitaDeAnalisis(t *testing.T) {
	solicitud := solicitudAnalisisContenidoValida(t)
	var ausente ContextoOperacionAlmacen
	solicitud.Contexto = ausente
	if !errors.Is(solicitud.Validar(), ErrAutorizacionAlmacenInvalida) {
		t.Fatal("se acepto una capacidad ausente")
	}

	decision, recurso, vinculos, instante := autorizacionAlmacenPrueba(
		t, AccionNegocioAnalizarCargaDocumental, []string{"analisis_seguridad", "estado"}, true,
	)
	vinculos.ObjetoVinculado = solicitud.Objeto
	recurso.Atributos[AtributoAlmacenObjetoRef] = solicitud.Objeto.Referencia
	recurso.Atributos[AtributoAlmacenObjetoVersion] = solicitud.Objeto.Version
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	decision.ContextoRecursoHuellaSHA256 = huella
	lectura, err := NuevoContextoAnalizarCargaDocumentalAlmacen(decision, recurso, vinculos, instante)
	if err != nil {
		t.Fatal(err)
	}
	solicitud.Contexto = lectura
	if !errors.Is(solicitud.Validar(), ErrAutorizacionAlmacenInvalida) {
		t.Fatal("se reutilizo el paso de lectura para analizar")
	}
}

func TestCapacidadesAnalizadorContenidoSonExigibles(t *testing.T) {
	capacidades := CapacidadesAnalizadorContenido{
		ConectorID: "antivirus_corporativo", VersionConector: 2, AnalisisEnFlujo: true,
		CanalAutenticado: true, CifradoEnTransito: true, IdentidadMutua: true,
		ActualizacionFirmas: true, DetectaMalware: true, DetectaContenidoActivo: true,
		TamanoMaximo: 64 << 20,
	}
	requisitos := RequisitosAnalizadorContenido{
		AnalisisEnFlujo: true, CanalAutenticado: true, CifradoEnTransito: true,
		IdentidadMutua: true, ActualizacionFirmas: true, DetectaMalware: true,
		DetectaContenidoActivo: true, TamanoMinimo: 32 << 20,
	}
	if err := VerificarCapacidadesAnalizadorContenido(capacidades, requisitos); err != nil {
		t.Fatalf("capacidades validas: %v", err)
	}
	capacidades.CanalAutenticado = false
	if !errors.Is(VerificarCapacidadesAnalizadorContenido(capacidades, requisitos), ErrCapacidadAnalisisContenidoNoDisponible) {
		t.Fatal("se acepto un motor sin canal autenticado")
	}
}

func TestSolicitudAnalisisRechazaLectorTipadoNulo(t *testing.T) {
	solicitud := solicitudAnalisisContenidoValida(t)
	var lector *bytes.Reader
	solicitud.Contenido = lector
	if !errors.Is(solicitud.Validar(), ErrSolicitudAnalisisContenidoInvalida) {
		t.Fatal("se acepto un io.Reader que contenia un puntero nulo")
	}
}
