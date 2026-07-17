package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

var instanteFuenteAutoridadPrueba = time.Date(2026, time.July, 17, 9, 30, 0, 123_456_000, time.UTC)

func contenidoFuenteAutoridadPrueba() ContenidoFuenteAutoridad {
	return ContenidoFuenteAutoridad{
		MateriaClave: "seleccion_temporal",
		Nombre:       "Reglamento de selección temporal y bolsas",
		Ambitos: []AmbitoFuenteAutoridad{
			{DimensionClave: "entidad", ValoresClave: []string{"diputacion_granada"}},
			{DimensionClave: "colectivo", ValoresClave: []string{"personal_laboral", "personal_funcionario"}},
		},
		Documento: DocumentoFuenteAutoridad{
			DocumentoID: "doc:autoridad:reglamento:2026", DocumentoVersion: 1,
			RepresentacionRef: "rep:pdfa:reglamento:2026:v1", HuellaContenidoSHA256: strings.Repeat("a", 64),
			PublicacionOficialRef: "bop:granada:2026:112", ActoOrigenRef: "acto:pleno:2026:reglamento_bolsas",
			OrganoEmisorRef: "organo:diputacion:pleno",
		},
		Preceptos: []PreceptoFuenteAutoridad{
			{Clave: "articulo_12_3", Cita: "Artículo 12.3"},
			{Clave: "disposicion_final_2", Cita: "Disposición final segunda"},
		},
		Vigencia: PeriodoFuenteAutoridad{
			Desde: time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC),
		},
		Efectos: PeriodoFuenteAutoridad{
			Desde: time.Date(2026, time.July, 1, 0, 0, 0, 0, time.UTC),
		},
		ConocidaEn: time.Date(2026, time.July, 16, 12, 0, 0, 0, time.UTC),
	}
}

func nuevaFuenteAutoridadPrueba(t *testing.T) FuenteAutoridadVersionada {
	t.Helper()
	fuente, err := NuevaFuenteAutoridadBorradorV1(DatosAltaFuenteAutoridadV1{
		ID: "reglamento_bolsas_2026", Contenido: contenidoFuenteAutoridadPrueba(),
		CreadaPor: "per_actor_creador_000000000001", CreadaEn: instanteFuenteAutoridadPrueba,
		MotivoCreacionCodigo: "alta_documento_oficial",
	})
	if err != nil {
		t.Fatalf("NuevaFuenteAutoridadBorrador() error = %v", err)
	}
	return fuente
}

func solicitudYEvidenciaActoFuenteAutoridadPrueba(
	t *testing.T,
	fuente FuenteAutoridadVersionada,
	estadoNuevo EstadoFuenteAutoridad,
	actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	caracter string,
	registradaEn time.Time,
	preparadaEn time.Time,
) (SolicitudTransicionFuenteAutoridadV1, EvidenciaActoFuenteAutoridad) {
	t.Helper()
	solicitud, err := fuente.PrepararSolicitudTransicionV1(DatosPreparacionTransicionFuenteAutoridadV1{
		EstadoNuevo: estadoNuevo, ActorRef: actorRef, MotivoCodigo: motivoCodigo,
		SolicitudRef: "solicitud:acto:" + caracter, PreparadaEn: preparadaEn,
		ExpiraEn: registradaEn.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("PrepararSolicitudTransicionV1() error = %v", err)
	}
	mensaje, err := PrepararMensajeAtestacionActoFuenteAutoridadV1(
		solicitud, DatosMensajeAtestacionActoFuenteAutoridadV1{
			EvidenciaRef: "evidencia:acto:" + caracter,
			ActoRef:      "acto:autoridad:" + caracter, DocumentoRef: "doc:acto:" + caracter,
			RepresentacionRef: "rep:acto:" + caracter, HuellaDocumentoSHA256: strings.Repeat(caracter, 64),
			OrganoRef:      "organo:competente:" + caracter,
			FirmasRefs:     []string{"firma:secretaria:" + caracter, "firma:presidencia:" + caracter},
			ComprobadorRef: "conector:firma:" + caracter,
			ActoOcurridoEn: preparadaEn.Add(-time.Minute), ComprobadaEn: registradaEn,
		})
	if err != nil {
		t.Fatalf("PrepararMensajeAtestacionActoFuenteAutoridadV1() error = %v", err)
	}
	evidencia, err := mensaje.ConstituirEvidenciaAtestadaV1(DatosSobreAtestacionActoFuenteAutoridadV1{
		AtestacionRef: "atestacion:acto:" + caracter, HuellaAtestacionSHA256: strings.Repeat(caracter, 64),
		FirmaAtestacionRef: "firma:atestacion:" + caracter,
	})
	if err != nil {
		t.Fatalf("ConstituirEvidenciaAtestadaV1() error = %v", err)
	}
	return solicitud, evidencia
}

func datosPreparacionTransicionFuenteAutoridadPrueba(
	estadoNuevo EstadoFuenteAutoridad,
	actorRef string,
	motivo CodigoMotivoFuenteAutoridad,
	solicitudRef string,
	preparadaEn time.Time,
) DatosPreparacionTransicionFuenteAutoridadV1 {
	return DatosPreparacionTransicionFuenteAutoridadV1{
		EstadoNuevo: estadoNuevo, ActorRef: actorRef, MotivoCodigo: motivo,
		SolicitudRef: solicitudRef, PreparadaEn: preparadaEn, ExpiraEn: preparadaEn.Add(time.Hour),
	}
}

func evidenciaDesdeCompromisoFuenteAutoridadPrueba(
	t testing.TB,
	compromiso CompromisoTransicionFuenteAutoridadV1,
	indice int,
) EvidenciaActoFuenteAutoridad {
	t.Helper()
	huellaCompromiso, err := compromiso.HuellaSHA256()
	if err != nil {
		t.Fatalf("Huella compromiso %d: %v", indice, err)
	}
	referencia := fmt.Sprintf("%03d", indice)
	huella := fmt.Sprintf("%064x", indice+1)
	evidencia := EvidenciaActoFuenteAutoridad{
		EvidenciaRef: "evidencia:acto:" + referencia, Accion: compromiso.Accion,
		FuenteID: compromiso.Fuente.FuenteID, FuenteVersion: compromiso.Fuente.Version,
		HuellaContenidoSHA256: compromiso.Fuente.HuellaContenidoSHA256,
		ActoRef:               "acto:autoridad:" + referencia, DocumentoRef: "doc:acto:" + referencia,
		RepresentacionRef: "rep:acto:" + referencia, HuellaDocumentoSHA256: huella,
		OrganoRef:      "organo:competente:" + referencia,
		FirmasRefs:     []string{"firma:secretaria:" + referencia, "firma:presidencia:" + referencia},
		ComprobadorRef: "conector:firma:" + referencia,
		AtestacionRef:  "atestacion:acto:" + referencia, HuellaAtestacionSHA256: huella,
		FirmaAtestacionRef: "firma:atestacion:" + referencia, HuellaCompromisoSHA256: huellaCompromiso,
		ActoOcurridoEn: compromiso.PreparadaEn.Add(-time.Minute),
		ComprobadaEn:   compromiso.PreparadaEn.Add(time.Minute),
	}
	huellaMensaje, err := mensajeAtestacionActoFuenteAutoridad(compromiso, evidencia).HuellaSHA256()
	if err != nil {
		t.Fatalf("Huella mensaje atestado %d: %v", indice, err)
	}
	evidencia.HuellaMensajeAtestadoSHA256 = huellaMensaje
	return evidencia
}

func contenidoFuenteAutoridadVoluminosoPrueba(valoresPorAmbito int) ContenidoFuenteAutoridad {
	contenido := contenidoFuenteAutoridadPrueba()
	contenido.Preceptos = make([]PreceptoFuenteAutoridad, maximoPreceptosFuenteAutoridad)
	for indice := range contenido.Preceptos {
		contenido.Preceptos[indice] = PreceptoFuenteAutoridad{
			Clave: fmt.Sprintf("precepto_%04d", indice),
			Cita:  "A" + strings.Repeat("x", maximoCaracteresCitaAutoridad-1),
		}
	}
	contenido.Ambitos = make([]AmbitoFuenteAutoridad, maximoAmbitosFuenteAutoridad)
	for ambito := range contenido.Ambitos {
		valores := make([]string, valoresPorAmbito)
		for indice := range valores {
			valores[indice] = fmt.Sprintf("v%s_%03d_%03d", strings.Repeat("x", 96), ambito, indice)
		}
		contenido.Ambitos[ambito] = AmbitoFuenteAutoridad{
			DimensionClave: fmt.Sprintf("dimension_%03d", ambito),
			ValoresClave:   valores,
		}
	}
	return contenido
}

func fuenteAutoridadVoluminosaConHistoriaPrueba(t testing.TB, transiciones int) FuenteAutoridadVersionada {
	t.Helper()
	fuente, err := NuevaFuenteAutoridadBorradorV1(DatosAltaFuenteAutoridadV1{
		ID: "fuente_voluminosa_prueba", Contenido: contenidoFuenteAutoridadVoluminosoPrueba(120),
		CreadaPor: "per_actor_creador_volumen_000001", CreadaEn: instanteFuenteAutoridadPrueba,
		MotivoCreacionCodigo: "alta_prueba_volumen",
	})
	if err != nil {
		t.Fatalf("crear fuente voluminosa: %v", err)
	}
	huella, err := fuente.HuellaContenidoSHA256()
	if err != nil {
		t.Fatalf("huella fuente voluminosa: %v", err)
	}
	estado := EstadoFuenteAutoridadBorrador
	for indice := 0; indice < transiciones; indice++ {
		estadoNuevo := EstadoFuenteAutoridadPublicada
		if estado == EstadoFuenteAutoridadPublicada {
			estadoNuevo = EstadoFuenteAutoridadSuspendida
		}
		actor := fmt.Sprintf("per_actor_transicion_%022d", indice)
		motivo := CodigoMotivoFuenteAutoridad(fmt.Sprintf("transicion_%03d", indice))
		registradaEn := fuente.CreadaEn.Add(time.Duration(indice+1) * time.Hour)
		compromiso, err := construirCompromisoTransicionFuenteAutoridad(
			fuente.ID, fuente.Version, huella, fuente.Revision, uint64(indice+1),
			estado, estadoNuevo, actor, motivo, fuente.huellaHistoriaActual(),
			"solicitud:volumen:"+fmt.Sprintf("%03d", indice), registradaEn.Add(-2*time.Minute), registradaEn.Add(time.Hour),
		)
		if err != nil {
			t.Fatalf("compromiso %d: %v", indice, err)
		}
		transicion := TransicionFuenteAutoridad{
			Secuencia: uint64(indice + 1), EstadoAnterior: estado, EstadoNuevo: estadoNuevo,
			ActorRef: actor, MotivoCodigo: motivo, RegistradaEn: registradaEn,
			SolicitudRef: compromiso.SolicitudRef, PreparadaEn: compromiso.PreparadaEn, ExpiraEn: compromiso.ExpiraEn,
			Evidencia:                    evidenciaDesdeCompromisoFuenteAutoridadPrueba(t, compromiso, indice),
			HuellaHistoriaAnteriorSHA256: fuente.huellaHistoriaActual(),
		}
		huellaHistoriaNueva, err := huellaHistoriaTransicionFuenteAutoridad(transicion, compromiso)
		if err != nil {
			t.Fatalf("huella de historia %d: %v", indice, err)
		}
		transicion.HuellaHistoriaNuevaSHA256 = huellaHistoriaNueva
		fuente.Transiciones = append(fuente.Transiciones, transicion)
		fuente.Revision++
		fuente.Estado = estadoNuevo
		estado = estadoNuevo
	}
	return fuente
}

func TestFuenteAutoridadConservaHistoriaYSeparaContenidoDeEstado(t *testing.T) {
	borrador := nuevaFuenteAutoridadPrueba(t)
	huellaContenidoInicial, err := borrador.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaEstadoInicial, err := borrador.HuellaEstadoSHA256()
	if err != nil {
		t.Fatal(err)
	}

	contenidoActualizado := contenidoFuenteAutoridadPrueba()
	contenidoActualizado.Preceptos = append(contenidoActualizado.Preceptos,
		PreceptoFuenteAutoridad{Clave: "anexo_1", Cita: "Anexo I"})
	actualizado, err := borrador.ActualizarBorrador(
		borrador.Revision, contenidoActualizado, "per_actor_editor_0000000000001",
		"anadir_anexo_citado", instanteFuenteAutoridadPrueba.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("ActualizarBorrador() error = %v", err)
	}
	huellaContenidoActualizada, err := actualizado.HuellaContenidoSHA256()
	if err != nil || huellaContenidoActualizada == huellaContenidoInicial {
		t.Fatalf("la edición no cambió la huella de contenido: inicial=%s actual=%s error=%v", huellaContenidoInicial, huellaContenidoActualizada, err)
	}

	publicadaEn := instanteFuenteAutoridadPrueba.Add(2 * time.Hour)
	actorPublicador := "per_actor_publicador_0000000001"
	motivoPublicacion := CodigoMotivoFuenteAutoridad("publicacion_aprobada")
	solicitudPublicacion, evidenciaPublicacion := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, actualizado, EstadoFuenteAutoridadPublicada, actorPublicador, motivoPublicacion,
		"b", publicadaEn, publicadaEn.Add(-time.Minute),
	)
	publicada, err := actualizado.AplicarTransicionV1(solicitudPublicacion, evidenciaPublicacion, evidenciaPublicacion.ComprobadaEn)
	if err != nil {
		t.Fatalf("Publicar() error = %v", err)
	}
	if publicada.Estado != EstadoFuenteAutoridadPublicada || publicada.Revision != 3 || len(publicada.Transiciones) != 1 {
		t.Fatalf("publicada = %+v", publicada)
	}
	if _, err := publicada.ActualizarBorrador(publicada.Revision, publicada.Contenido,
		"per_actor_editor_0000000000002", "edicion_prohibida", publicadaEn.Add(time.Minute)); !errors.Is(err, ErrTransicionAutoridadInvalida) {
		t.Fatalf("editar publicada error = %v", err)
	}

	suspendidaEn := publicadaEn.Add(time.Hour)
	actorSuspensor := "per_actor_suspensor_0000000001"
	motivoSuspension := CodigoMotivoFuenteAutoridad("suspension_por_resolucion")
	solicitudSuspension, evidenciaSuspension := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, publicada, EstadoFuenteAutoridadSuspendida, actorSuspensor, motivoSuspension,
		"c", suspendidaEn, suspendidaEn.Add(-time.Minute),
	)
	suspendida, err := publicada.AplicarTransicionV1(solicitudSuspension, evidenciaSuspension, evidenciaSuspension.ComprobadaEn)
	if err != nil {
		t.Fatalf("Suspender() error = %v", err)
	}
	levantadaEn := suspendidaEn.Add(time.Hour)
	actorLevantamiento := "per_actor_levantamiento_00000001"
	motivoLevantamiento := CodigoMotivoFuenteAutoridad("levantamiento_por_nuevo_acto")
	solicitudLevantamiento, evidenciaLevantamiento := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, suspendida, EstadoFuenteAutoridadPublicada, actorLevantamiento, motivoLevantamiento,
		"d", levantadaEn, levantadaEn.Add(-time.Minute),
	)
	levantada, err := suspendida.AplicarTransicionV1(solicitudLevantamiento, evidenciaLevantamiento, evidenciaLevantamiento.ComprobadaEn)
	if err != nil {
		t.Fatalf("LevantarSuspension() error = %v", err)
	}
	derogadaEn := levantadaEn.Add(time.Hour)
	actorDerogacion := "per_actor_derogacion_0000000001"
	motivoDerogacion := CodigoMotivoFuenteAutoridad("derogacion_por_acto_posterior")
	solicitudDerogacion, evidenciaDerogacion := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, levantada, EstadoFuenteAutoridadDerogada, actorDerogacion, motivoDerogacion,
		"e", derogadaEn, derogadaEn.Add(-time.Minute),
	)
	derogada, err := levantada.AplicarTransicionV1(solicitudDerogacion, evidenciaDerogacion, evidenciaDerogacion.ComprobadaEn)
	if err != nil {
		t.Fatalf("Derogar() error = %v", err)
	}
	if derogada.Estado != EstadoFuenteAutoridadDerogada || derogada.Revision != 6 || len(derogada.Transiciones) != 4 {
		t.Fatalf("derogada = %+v", derogada)
	}
	if _, err := derogada.PrepararSolicitudTransicionV1(datosPreparacionTransicionFuenteAutoridadPrueba(
		EstadoFuenteAutoridadSuspendida, "per_actor_suspensor_0000000002", "estado_terminal",
		"solicitud:estado_terminal", derogadaEn.Add(time.Hour),
	)); !errors.Is(err, ErrTransicionAutoridadInvalida) {
		t.Fatalf("transición desde derogada error = %v", err)
	}

	for nombre, fuente := range map[string]FuenteAutoridadVersionada{
		"publicada": publicada, "suspendida": suspendida, "levantada": levantada, "derogada": derogada,
	} {
		huella, err := fuente.HuellaContenidoSHA256()
		if err != nil || huella != huellaContenidoActualizada {
			t.Fatalf("%s alteró contenido: huella=%s error=%v", nombre, huella, err)
		}
		if huellaEstado, err := fuente.HuellaEstadoSHA256(); err != nil || !esSHA256Autoridad(huellaEstado) {
			t.Fatalf("%s no produjo huella de estado: huella=%s error=%v", nombre, huellaEstado, err)
		}
	}
	huellaEstadoFinal, err := derogada.HuellaEstadoSHA256()
	if err != nil || huellaEstadoFinal == huellaEstadoInicial {
		t.Fatalf("la historia no cambió la huella de estado: inicial=%s final=%s error=%v", huellaEstadoInicial, huellaEstadoFinal, err)
	}

	contenidoV2 := contenidoFuenteAutoridadPrueba()
	contenidoV2.Nombre = "Reglamento de selección temporal y bolsas corregido"
	v2, err := derogada.NuevaVersionV1(contenidoV2, "per_actor_creador_v2_0000000001",
		"correccion_material_aprobada", derogadaEn.Add(time.Hour))
	if err != nil {
		t.Fatalf("NuevaVersion() error = %v", err)
	}
	referenciaV1, err := derogada.ReferenciaLinajeExacta()
	if err != nil || v2.VersionAnterior != referenciaV1 || v2.Estado != EstadoFuenteAutoridadBorrador {
		t.Fatalf("v2 no fijó predecesora exacta: v2=%+v ref=%+v error=%v", v2, referenciaV1, err)
	}
	alterada := v2
	alterada.VersionAnterior.HuellaHistoriaSHA256 = strings.Repeat("f", 64)
	if err := alterada.Validar(); !errors.Is(err, ErrFuenteAutoridadInvalida) {
		t.Fatalf("linaje de predecesora manipulado: %v", err)
	}
	if _, err := derogada.NuevaVersionV1(
		derogada.Contenido, "per_actor_creador_v2_igual_00001", "version_sin_cambios", derogadaEn.Add(2*time.Hour),
	); !errors.Is(err, ErrFuenteAutoridadInvalida) {
		t.Fatalf("sucesora sin cambios aceptada: %v", err)
	}
}

func TestFuenteAutoridadCanonizaOrdenSinCambiarHuellas(t *testing.T) {
	primera := nuevaFuenteAutoridadPrueba(t)
	contenido := contenidoFuenteAutoridadPrueba()
	contenido.Ambitos[1], contenido.Ambitos[0] = contenido.Ambitos[0], contenido.Ambitos[1]
	contenido.Ambitos[0].ValoresClave[0], contenido.Ambitos[0].ValoresClave[1] =
		contenido.Ambitos[0].ValoresClave[1], contenido.Ambitos[0].ValoresClave[0]
	contenido.Preceptos[0], contenido.Preceptos[1] = contenido.Preceptos[1], contenido.Preceptos[0]
	segunda, err := NuevaFuenteAutoridadBorradorV1(DatosAltaFuenteAutoridadV1{
		ID: primera.ID, Contenido: contenido,
		CreadaPor: primera.CreadaPor, CreadaEn: primera.CreadaEn,
		MotivoCreacionCodigo: primera.MotivoCreacionCodigo,
	})
	if err != nil {
		t.Fatal(err)
	}
	huellaContenidoPrimera, _ := primera.HuellaContenidoSHA256()
	huellaContenidoSegunda, _ := segunda.HuellaContenidoSHA256()
	huellaEstadoPrimera, _ := primera.HuellaEstadoSHA256()
	huellaEstadoSegunda, _ := segunda.HuellaEstadoSHA256()
	if huellaContenidoPrimera != huellaContenidoSegunda || huellaEstadoPrimera != huellaEstadoSegunda {
		t.Fatalf("orden alteró huellas: contenido %s/%s estado %s/%s",
			huellaContenidoPrimera, huellaContenidoSegunda, huellaEstadoPrimera, huellaEstadoSegunda)
	}

	publicadaEn := instanteFuenteAutoridadPrueba.Add(time.Hour)
	actorPublicador := "per_actor_publicador_0000000001"
	motivoPublicacion := CodigoMotivoFuenteAutoridad("publicacion_aprobada")
	solicitud1, evidencia1 := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, primera, EstadoFuenteAutoridadPublicada, actorPublicador, motivoPublicacion,
		"b", publicadaEn, publicadaEn.Add(-time.Minute),
	)
	solicitud2, _ := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, segunda, EstadoFuenteAutoridadPublicada, actorPublicador, motivoPublicacion,
		"b", publicadaEn, publicadaEn.Add(-time.Minute),
	)
	evidencia2 := evidencia1
	evidencia2.FirmasRefs = append([]string(nil), evidencia1.FirmasRefs...)
	evidencia2.FirmasRefs[0], evidencia2.FirmasRefs[1] = evidencia2.FirmasRefs[1], evidencia2.FirmasRefs[0]
	publicada1, err1 := primera.AplicarTransicionV1(solicitud1, evidencia1, evidencia1.ComprobadaEn)
	publicada2, err2 := segunda.AplicarTransicionV1(solicitud2, evidencia2, evidencia2.ComprobadaEn)
	if err1 != nil || err2 != nil {
		t.Fatalf("publicar variantes: %v / %v", err1, err2)
	}
	huellaPublicada1, _ := publicada1.HuellaEstadoSHA256()
	huellaPublicada2, _ := publicada2.HuellaEstadoSHA256()
	if huellaPublicada1 != huellaPublicada2 {
		t.Fatalf("orden de firmas alteró huella: %s / %s", huellaPublicada1, huellaPublicada2)
	}
}

func TestFuenteAutoridadRechazaEstructurasAmbiguasONoCanonicas(t *testing.T) {
	base := nuevaFuenteAutoridadPrueba(t)
	zonaNoUTC := time.FixedZone("UTC-falso", 0)
	pruebas := []struct {
		nombre string
		mutar  func(*FuenteAutoridadVersionada)
	}{
		{"id no canónico", func(f *FuenteAutoridadVersionada) { f.ID = "Reglamento Bolsas" }},
		{"versión sin predecesora", func(f *FuenteAutoridadVersionada) { f.Version = 2 }},
		{"dimensión repetida", func(f *FuenteAutoridadVersionada) {
			f.Contenido.Ambitos = append(f.Contenido.Ambitos, f.Contenido.Ambitos[0])
		}},
		{"valor repetido", func(f *FuenteAutoridadVersionada) {
			f.Contenido.Ambitos[0].ValoresClave = append(f.Contenido.Ambitos[0].ValoresClave, f.Contenido.Ambitos[0].ValoresClave[0])
		}},
		{"precepto repetido", func(f *FuenteAutoridadVersionada) {
			f.Contenido.Preceptos = append(f.Contenido.Preceptos, f.Contenido.Preceptos[0])
		}},
		{"periodo vacío", func(f *FuenteAutoridadVersionada) { f.Contenido.Vigencia.Hasta = f.Contenido.Vigencia.Desde }},
		{"instante no UTC", func(f *FuenteAutoridadVersionada) { f.Contenido.ConocidaEn = f.Contenido.ConocidaEn.In(zonaNoUTC) }},
		{"precisión no canónica", func(f *FuenteAutoridadVersionada) { f.CreadaEn = f.CreadaEn.Add(time.Nanosecond) }},
		{"año no serializable", func(f *FuenteAutoridadVersionada) { f.CreadaEn = time.Date(10_000, 1, 1, 0, 0, 0, 0, time.UTC) }},
		{"conocida después de alta", func(f *FuenteAutoridadVersionada) { f.Contenido.ConocidaEn = f.CreadaEn.Add(time.Microsecond) }},
		{"control en cita", func(f *FuenteAutoridadVersionada) { f.Contenido.Preceptos[0].Cita = "Artículo\n12" }},
		{"formato invisible en actor", func(f *FuenteAutoridadVersionada) { f.CreadaPor += "\u200b" }},
		{"texto no NFC", func(f *FuenteAutoridadVersionada) { f.Contenido.Nombre = "Seleccio\u0301n" }},
		{"motivo libre", func(f *FuenteAutoridadVersionada) { f.MotivoCreacionCodigo = "incluye texto libre" }},
		{"huella mayúscula", func(f *FuenteAutoridadVersionada) {
			f.Contenido.Documento.HuellaContenidoSHA256 = strings.Repeat("A", 64)
		}},
		{"huella nula", func(f *FuenteAutoridadVersionada) {
			f.Contenido.Documento.HuellaContenidoSHA256 = huellaSHA256Nula
		}},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			alterada := base
			alterada.Contenido.Ambitos = append([]AmbitoFuenteAutoridad(nil), base.Contenido.Ambitos...)
			for indice := range alterada.Contenido.Ambitos {
				alterada.Contenido.Ambitos[indice].ValoresClave = append([]string(nil), base.Contenido.Ambitos[indice].ValoresClave...)
			}
			alterada.Contenido.Preceptos = append([]PreceptoFuenteAutoridad(nil), base.Contenido.Preceptos...)
			prueba.mutar(&alterada)
			if err := alterada.Validar(); !errors.Is(err, ErrFuenteAutoridadInvalida) {
				t.Fatalf("Validar() error = %v", err)
			}
		})
	}
}

func TestFuenteAutoridadExigeSegregacionYEvidenciaExacta(t *testing.T) {
	borrador := nuevaFuenteAutoridadPrueba(t)
	editorRef := "per_actor_editor_0000000000001"
	contenidoEditado := borrador.Contenido
	contenidoEditado.Nombre = "Reglamento de selección temporal y bolsas revisado"
	editado, err := borrador.ActualizarBorrador(borrador.Revision, contenidoEditado,
		editorRef, "revision_formal", instanteFuenteAutoridadPrueba.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	publicadaEn := instanteFuenteAutoridadPrueba.Add(2 * time.Hour)
	actorPublicador := "per_actor_publicador_0000000001"
	motivoPublicacion := CodigoMotivoFuenteAutoridad("publicacion_aprobada")
	solicitud, evidencia := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, editado, EstadoFuenteAutoridadPublicada, actorPublicador, motivoPublicacion,
		"b", publicadaEn, publicadaEn.Add(-time.Minute),
	)
	if _, err := editado.PrepararSolicitudTransicionV1(datosPreparacionTransicionFuenteAutoridadPrueba(
		EstadoFuenteAutoridadPublicada, editado.CreadaPor, "autopublicacion",
		"solicitud:autopublicacion:creador", publicadaEn,
	)); !errors.Is(err, ErrTransicionAutoridadInvalida) {
		t.Fatalf("publicar por creador error = %v", err)
	}
	if _, err := editado.PrepararSolicitudTransicionV1(datosPreparacionTransicionFuenteAutoridadPrueba(
		EstadoFuenteAutoridadPublicada, editorRef, "autopublicacion",
		"solicitud:autopublicacion:editor", publicadaEn,
	)); !errors.Is(err, ErrTransicionAutoridadInvalida) {
		t.Fatalf("publicar por editor error = %v", err)
	}

	pruebas := []struct {
		nombre string
		mutar  func(*EvidenciaActoFuenteAutoridad)
	}{
		{"acción distinta", func(e *EvidenciaActoFuenteAutoridad) { e.Accion = AccionActoDerogarFuenteAutoridad }},
		{"fuente distinta", func(e *EvidenciaActoFuenteAutoridad) { e.FuenteID = "otra_fuente" }},
		{"versión distinta", func(e *EvidenciaActoFuenteAutoridad) { e.FuenteVersion++ }},
		{"huella distinta", func(e *EvidenciaActoFuenteAutoridad) { e.HuellaContenidoSHA256 = strings.Repeat("f", 64) }},
		{"compromiso distinto", func(e *EvidenciaActoFuenteAutoridad) { e.HuellaCompromisoSHA256 = strings.Repeat("e", 64) }},
		{"atestación nula", func(e *EvidenciaActoFuenteAutoridad) { e.HuellaAtestacionSHA256 = huellaSHA256Nula }},
		{"acto distinto", func(e *EvidenciaActoFuenteAutoridad) { e.ActoRef = "acto:autoridad:alterado" }},
		{"documento distinto", func(e *EvidenciaActoFuenteAutoridad) { e.DocumentoRef = "doc:acto:alterado" }},
		{"representación distinta", func(e *EvidenciaActoFuenteAutoridad) { e.RepresentacionRef = "rep:acto:alterada" }},
		{"huella documental distinta", func(e *EvidenciaActoFuenteAutoridad) { e.HuellaDocumentoSHA256 = strings.Repeat("d", 64) }},
		{"órgano distinto", func(e *EvidenciaActoFuenteAutoridad) { e.OrganoRef = "organo:competente:alterado" }},
		{"comprobador distinto", func(e *EvidenciaActoFuenteAutoridad) { e.ComprobadorRef = "conector:firma:alterado" }},
		{"fecha del acto distinta", func(e *EvidenciaActoFuenteAutoridad) { e.ActoOcurridoEn = e.ActoOcurridoEn.Add(-time.Hour) }},
		{"firma repetida", func(e *EvidenciaActoFuenteAutoridad) { e.FirmasRefs = []string{e.FirmasRefs[0], e.FirmasRefs[0]} }},
		{"firma válida pero distinta", func(e *EvidenciaActoFuenteAutoridad) { e.FirmasRefs[0] = "firma:secretaria:alterada" }},
		{"comprobación posterior", func(e *EvidenciaActoFuenteAutoridad) { e.ComprobadaEn = publicadaEn.Add(time.Microsecond) }},
	}
	for _, prueba := range pruebas {
		t.Run(prueba.nombre, func(t *testing.T) {
			alterada := evidencia
			alterada.FirmasRefs = append([]string(nil), evidencia.FirmasRefs...)
			prueba.mutar(&alterada)
			_, err := editado.AplicarTransicionV1(solicitud, alterada, evidencia.ComprobadaEn)
			if !errors.Is(err, ErrEvidenciaActoAutoridadInvalida) {
				t.Fatalf("Publicar() error = %v", err)
			}
		})
	}
}

func TestFuenteAutoridadDevuelveCopiasDefensivasYCitasExactas(t *testing.T) {
	fuente := nuevaFuenteAutoridadPrueba(t)
	clon, err := fuente.ClonarCanonica()
	if err != nil {
		t.Fatal(err)
	}
	clon.Contenido.Ambitos[0].ValoresClave[0] = "alterado"
	clon.Contenido.Preceptos[0].Cita = "Alterada"
	if fuente.Contenido.Ambitos[0].ValoresClave[0] == "alterado" || fuente.Contenido.Preceptos[0].Cita == "Alterada" {
		t.Fatal("ClonarCanonica compartió memoria")
	}
	referencia, err := fuente.ReferenciaExacta()
	if err != nil {
		t.Fatal(err)
	}
	cita := CitaFuenteAutoridad{Fuente: referencia, Preceptos: []string{"disposicion_final_2", "articulo_12_3"}}
	canonica, err := cita.ClonarCanonica()
	if err != nil || canonica.Preceptos[0] != "articulo_12_3" {
		t.Fatalf("cita canónica = %+v, error = %v", canonica, err)
	}
	if err := (CitaFuenteAutoridad{Fuente: referencia}).Validar(); !errors.Is(err, ErrReferenciaAutoridadInvalida) {
		t.Fatalf("cita sin preceptos error = %v", err)
	}
	if _, err := fuente.Citar("articulo_12_3"); !errors.Is(err, ErrReferenciaAutoridadInvalida) {
		t.Fatalf("citar borrador error = %v", err)
	}
	publicadaEn := instanteFuenteAutoridadPrueba.Add(time.Hour)
	actor := "per_actor_publicador_citas_000001"
	motivo := CodigoMotivoFuenteAutoridad("publicacion_para_cita")
	solicitud, evidencia := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, fuente, EstadoFuenteAutoridadPublicada, actor, motivo,
		"b", publicadaEn, publicadaEn.Add(-time.Minute),
	)
	publicada, err := fuente.AplicarTransicionV1(solicitud, evidencia, evidencia.ComprobadaEn)
	if err != nil {
		t.Fatal(err)
	}
	evidencia.FirmasRefs[0] = "firma:entrada:alterada"
	clonPublicada, err := publicada.ClonarCanonica()
	if err != nil {
		t.Fatal(err)
	}
	clonPublicada.Transiciones[0].Evidencia.FirmasRefs[0] = "firma:clon:alterada"
	if publicada.Transiciones[0].Evidencia.FirmasRefs[0] == "firma:entrada:alterada" ||
		publicada.Transiciones[0].Evidencia.FirmasRefs[0] == "firma:clon:alterada" {
		t.Fatal("las firmas de evidencia compartieron memoria")
	}
	citaPublicada, err := publicada.Citar("disposicion_final_2", "articulo_12_3")
	if err != nil || citaPublicada.Preceptos[0] != "articulo_12_3" {
		t.Fatalf("cita publicada = %+v, error = %v", citaPublicada, err)
	}
	if _, err := publicada.Citar("precepto_inexistente"); !errors.Is(err, ErrReferenciaAutoridadInvalida) {
		t.Fatalf("citar precepto inexistente error = %v", err)
	}
}

func TestPeriodoFuenteAutoridadUsaExtremosSemiabiertos(t *testing.T) {
	desde := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	hasta := desde.Add(24 * time.Hour)
	periodo := PeriodoFuenteAutoridad{Desde: desde, Hasta: hasta}
	if !periodo.Contiene(desde) || !periodo.Contiene(hasta.Add(-time.Microsecond)) || periodo.Contiene(hasta) {
		t.Fatalf("intervalo semiabierto incorrecto: %+v", periodo)
	}
	mismaHoraOtraZona := desde.Add(12*time.Hour + 999*time.Nanosecond).In(time.FixedZone("UTC+2", 2*60*60))
	if !periodo.Contiene(mismaHoraOtraZona) {
		t.Fatal("el instante equivalente debe normalizarse a UTC y microsegundos")
	}
}

func TestHuellaFuenteAutoridadTieneVectorDorado(t *testing.T) {
	fuente := nuevaFuenteAutoridadPrueba(t)
	huella, err := fuente.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	const esperada = "d2821789a5f1e641d1dc29934c2676e1cc274ac58570290d581a25dfa8817e0e"
	if huella != esperada {
		t.Fatalf("huella = %s, want %s", huella, esperada)
	}
	huellaEstado, err := fuente.HuellaEstadoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	const estadoEsperado = "38d5659fdca0e871464bf3d64d1872aadf90f6d5a1abe885e0dbc231f8ea5f2d"
	if huellaEstado != estadoEsperado {
		t.Fatalf("huella estado = %s, want %s", huellaEstado, estadoEsperado)
	}
}

func TestReferenciaFuenteAutoridadNuncaEmiteUnaReferenciaInvalida(t *testing.T) {
	if referencia, err := (ReferenciaFuenteAutoridad{}).Referencia(); !errors.Is(err, ErrReferenciaAutoridadInvalida) || referencia != "" {
		t.Fatalf("referencia inválida emitida: %q, error=%v", referencia, err)
	}
	exacta, err := nuevaFuenteAutoridadPrueba(t).ReferenciaExacta()
	if err != nil {
		t.Fatal(err)
	}
	referencia, err := exacta.Referencia()
	if err != nil || !strings.HasPrefix(referencia, "fuente:reglamento_bolsas_2026:v1:sha256:") {
		t.Fatalf("referencia exacta = %q, error=%v", referencia, err)
	}
}

func TestSolicitudTransicionFuenteAutoridadV1EsDefensiva(t *testing.T) {
	fuente := nuevaFuenteAutoridadPrueba(t)
	solicitud, _ := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, fuente, EstadoFuenteAutoridadPublicada, "per_publicador_solicitud_defensiva_01",
		"publicacion_solicitud_defensiva", "b", fuente.CreadaEn.Add(time.Hour),
		fuente.CreadaEn.Add(59*time.Minute),
	)
	bytesPrimeraLectura, err := solicitud.BytesCanonicos()
	if err != nil {
		t.Fatal(err)
	}
	bytesPrimeraLectura[0] = '['
	bytesSegundaLectura, err := solicitud.BytesCanonicos()
	if err != nil || len(bytesSegundaLectura) == 0 || bytesSegundaLectura[0] != '{' {
		t.Fatalf("la lectura compartió memoria: %q, error=%v", bytesSegundaLectura, err)
	}

	alterada := solicitud
	alterada.bytesCanonicos = append([]byte(nil), solicitud.bytesCanonicos...)
	alterada.bytesCanonicos[0] = '['
	if !errors.Is(alterada.Validar(), ErrTransicionAutoridadInvalida) {
		t.Fatal("una solicitud con bytes manipulados fue aceptada")
	}
	alterada = solicitud
	alterada.compromiso.ActorRef = "per_actor_sustituido_00000000001"
	if !errors.Is(alterada.Validar(), ErrTransicionAutoridadInvalida) {
		t.Fatal("una solicitud cuyo compromiso fue sustituido fue aceptada")
	}
}

func TestSobresFirmablesFuenteAutoridadV1TienenVectoresDorados(t *testing.T) {
	fuente := nuevaFuenteAutoridadPrueba(t)
	solicitud, evidencia := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, fuente, EstadoFuenteAutoridadPublicada, "per_publicador_vector_dorado_00001",
		"publicacion_vector_dorado", "b", fuente.CreadaEn.Add(time.Hour),
		fuente.CreadaEn.Add(59*time.Minute),
	)
	compromiso, err := solicitud.Compromiso()
	if err != nil {
		t.Fatal(err)
	}
	bytesCompromiso, err := compromiso.BytesCanonicos()
	if err != nil {
		t.Fatal(err)
	}
	huellaCompromiso, err := compromiso.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	mensaje := mensajeAtestacionActoFuenteAutoridad(compromiso, evidencia)
	bytesMensaje, err := mensaje.BytesCanonicos()
	if err != nil {
		t.Fatal(err)
	}
	huellaMensaje, err := mensaje.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	const compromisoEsperado = "bcb4633735b7126174aa5bd371356184885bac3d20e5044796ee12c31db10f04"
	const mensajeEsperado = "9d8a4aff85bfefedfdc673a3e2a103c17a961bb35c29c7fbeb91962866db0477"
	const bytesCompromisoEsperados = `{"esquema":"vec.fuente_autoridad.compromiso_transicion.v1","solicitud_ref":"solicitud:acto:b","fuente":{"fuente_id":"reglamento_bolsas_2026","version":1,"huella_contenido_sha256":"d2821789a5f1e641d1dc29934c2676e1cc274ac58570290d581a25dfa8817e0e"},"revision_previa":1,"secuencia":1,"estado_anterior":"borrador","estado_nuevo":"publicada","accion":"publicar","actor_ref":"per_publicador_vector_dorado_00001","motivo_codigo":"publicacion_vector_dorado","huella_historia_previa_sha256":"67686c345a2dcf4f415ea04742562d420272198da0eb50d4da5add566724b687","preparada_en":"2026-07-17T10:29:00.123456Z","expira_en":"2026-07-17T11:30:00.123456Z"}`
	const bytesMensajeEsperados = `{"esquema":"vec.fuente_autoridad.mensaje_atestacion_acto.v1","compromiso":{"esquema":"vec.fuente_autoridad.compromiso_transicion.v1","solicitud_ref":"solicitud:acto:b","fuente":{"fuente_id":"reglamento_bolsas_2026","version":1,"huella_contenido_sha256":"d2821789a5f1e641d1dc29934c2676e1cc274ac58570290d581a25dfa8817e0e"},"revision_previa":1,"secuencia":1,"estado_anterior":"borrador","estado_nuevo":"publicada","accion":"publicar","actor_ref":"per_publicador_vector_dorado_00001","motivo_codigo":"publicacion_vector_dorado","huella_historia_previa_sha256":"67686c345a2dcf4f415ea04742562d420272198da0eb50d4da5add566724b687","preparada_en":"2026-07-17T10:29:00.123456Z","expira_en":"2026-07-17T11:30:00.123456Z"},"evidencia_ref":"evidencia:acto:b","acto_ref":"acto:autoridad:b","documento_ref":"doc:acto:b","representacion_ref":"rep:acto:b","huella_documento_sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","organo_ref":"organo:competente:b","firmas_refs":["firma:presidencia:b","firma:secretaria:b"],"comprobador_ref":"conector:firma:b","acto_ocurrido_en":"2026-07-17T10:28:00.123456Z","comprobada_en":"2026-07-17T10:30:00.123456Z"}`
	if string(bytesCompromiso) != bytesCompromisoEsperados {
		t.Errorf("bytes compromiso = %s", bytesCompromiso)
	}
	if string(bytesMensaje) != bytesMensajeEsperados {
		t.Errorf("bytes mensaje = %s", bytesMensaje)
	}
	jsonCompromiso, err := json.Marshal(compromiso)
	if err != nil || !bytes.Equal(jsonCompromiso, bytesCompromiso) {
		t.Fatalf("json.Marshal alteró el compromiso canónico: %s / %s, error=%v",
			jsonCompromiso, bytesCompromiso, err)
	}
	jsonMensaje, err := json.Marshal(mensaje)
	if err != nil || !bytes.Equal(jsonMensaje, bytesMensaje) {
		t.Fatalf("json.Marshal alteró el mensaje canónico: %s / %s, error=%v",
			jsonMensaje, bytesMensaje, err)
	}
	if huellaCompromiso != compromisoEsperado {
		t.Errorf("huella compromiso = %s, want %s", huellaCompromiso, compromisoEsperado)
	}
	if huellaMensaje != mensajeEsperado {
		t.Errorf("huella mensaje = %s, want %s", huellaMensaje, mensajeEsperado)
	}

	bytesCanonicos, err := mensaje.BytesCanonicos()
	if err != nil {
		t.Fatal(err)
	}
	mensaje.FirmasRefs[0], mensaje.FirmasRefs[1] = mensaje.FirmasRefs[1], mensaje.FirmasRefs[0]
	bytesReordenados, err := mensaje.BytesCanonicos()
	if err != nil || !bytes.Equal(bytesCanonicos, bytesReordenados) {
		t.Fatalf("el orden equivalente de firmas alteró el mensaje: %v", err)
	}
	jsonReordenado, err := json.Marshal(mensaje)
	if err != nil || !bytes.Equal(bytesCanonicos, jsonReordenado) {
		t.Fatalf("json.Marshal conservó el orden no canónico de firmas: %v", err)
	}
}

func TestFuenteAutoridadSeparaFlujoTecnicoDeActoJuridicoHistorico(t *testing.T) {
	borrador := nuevaFuenteAutoridadPrueba(t)
	preparadaEn := borrador.CreadaEn.Add(time.Hour)
	registradaEn := preparadaEn.Add(time.Hour)
	solicitud, evidencia := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, borrador, EstadoFuenteAutoridadPublicada, "per_publicador_historico_0000001",
		"incorporacion_historica", "b", registradaEn, preparadaEn,
	)
	compromiso, err := solicitud.Compromiso()
	if err != nil {
		t.Fatal(err)
	}
	evidencia.ActoOcurridoEn = time.Date(2020, time.January, 15, 12, 0, 0, 0, time.UTC)
	huellaMensaje, err := mensajeAtestacionActoFuenteAutoridad(compromiso, evidencia).HuellaSHA256()
	if err != nil {
		t.Fatalf("acto histórico no firmable: %v", err)
	}
	evidencia.HuellaMensajeAtestadoSHA256 = huellaMensaje
	publicada, err := borrador.AplicarTransicionV1(solicitud, evidencia, registradaEn)
	if err != nil {
		t.Fatalf("incorporación histórica rechazada: %v", err)
	}
	transicion := publicada.Transiciones[0]
	if !transicion.PreparadaEn.Equal(preparadaEn) || !transicion.RegistradaEn.Equal(registradaEn) ||
		!transicion.Evidencia.ActoOcurridoEn.Equal(evidencia.ActoOcurridoEn) {
		t.Fatalf("ejes temporales mezclados: %+v", transicion)
	}

	evidenciaTemprana := evidencia
	evidenciaTemprana.ComprobadaEn = preparadaEn.Add(-time.Microsecond)
	evidenciaTemprana.ActoOcurridoEn = evidenciaTemprana.ComprobadaEn.Add(-time.Hour)
	if _, err := mensajeAtestacionActoFuenteAutoridad(compromiso, evidenciaTemprana).BytesCanonicos(); !errors.Is(err, ErrEvidenciaActoAutoridadInvalida) {
		t.Fatalf("comprobación anterior a la solicitud aceptada: %v", err)
	}
	if _, err := borrador.AplicarTransicionV1(
		solicitud, evidencia, evidencia.ComprobadaEn.Add(-time.Microsecond),
	); !errors.Is(err, ErrTransicionAutoridadInvalida) {
		t.Fatalf("registro anterior a la comprobación aceptado: %v", err)
	}
	if _, err := borrador.AplicarTransicionV1(
		solicitud, evidencia, compromiso.ExpiraEn.Add(time.Microsecond),
	); !errors.Is(err, ErrSolicitudAutoridadExpirada) {
		t.Fatalf("solicitud expirada aceptada: %v", err)
	}
	if _, err := borrador.AplicarTransicionV1(
		solicitud, evidencia, compromiso.ExpiraEn,
	); !errors.Is(err, ErrSolicitudAutoridadExpirada) {
		t.Fatalf("limite exclusivo de expiración aceptado: %v", err)
	}
	evidenciaEnLimite := evidencia
	evidenciaEnLimite.ComprobadaEn = compromiso.ExpiraEn
	if _, err := mensajeAtestacionActoFuenteAutoridad(compromiso, evidenciaEnLimite).BytesCanonicos(); !errors.Is(err, ErrEvidenciaActoAutoridadInvalida) {
		t.Fatalf("comprobación en el límite exclusivo aceptada: %v", err)
	}
}

func TestFuenteAutoridadAnclaLaHistoriaPreviaEnElActoFirmado(t *testing.T) {
	borrador := nuevaFuenteAutoridadPrueba(t)
	contenido := borrador.Contenido
	contenido.Nombre += " con historial"
	editado, err := borrador.ActualizarBorrador(
		borrador.Revision, contenido, "per_editor_historia_original_0001", "revision_historia",
		borrador.CreadaEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	registradaEn := editado.ultimaMutacionEn().Add(time.Hour)
	solicitud, evidencia := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, editado, EstadoFuenteAutoridadPublicada, "per_publicador_historia_00000001",
		"publicacion_historia", "b", registradaEn, registradaEn.Add(-time.Minute),
	)
	publicada, err := editado.AplicarTransicionV1(solicitud, evidencia, evidencia.ComprobadaEn)
	if err != nil {
		t.Fatal(err)
	}
	alterada := publicada
	alterada.EdicionesBorrador = append([]EdicionBorradorFuenteAutoridad(nil), publicada.EdicionesBorrador...)
	alterada.Transiciones = append([]TransicionFuenteAutoridad(nil), publicada.Transiciones...)
	alterada.Transiciones[0].Evidencia.FirmasRefs = append(
		[]string(nil), publicada.Transiciones[0].Evidencia.FirmasRefs...,
	)
	alterada.EdicionesBorrador[0].ActorRef = "per_editor_historia_sustituido_001"
	huellaHistoria, err := huellaHistoriaEdicionBorradorFuenteAutoridad(alterada.EdicionesBorrador[0])
	if err != nil {
		t.Fatal(err)
	}
	alterada.EdicionesBorrador[0].HuellaHistoriaNuevaSHA256 = huellaHistoria
	alterada.Transiciones[0].HuellaHistoriaAnteriorSHA256 = huellaHistoria
	if err := alterada.Validar(); !errors.Is(err, ErrFuenteAutoridadInvalida) {
		t.Fatalf("la historia reescrita conservó válido el acto firmado: %v", err)
	}
}

func TestFuenteAutoridadRechazaSucesoraAlDesbordarVersion(t *testing.T) {
	maximaVersion := ^uint64(0)
	borrador, err := nuevaFuenteAutoridadBorradorVersionada(
		"fuente_version_maxima", maximaVersion,
		ReferenciaLinajeFuenteAutoridad{
			Fuente: ReferenciaFuenteAutoridad{
				FuenteID: "fuente_version_maxima", Version: maximaVersion - 1,
				HuellaContenidoSHA256: strings.Repeat("a", 64),
			},
			Revision: 1, Estado: EstadoFuenteAutoridadPublicada,
			HuellaHistoriaSHA256: strings.Repeat("b", 64), HuellaEstadoSHA256: strings.Repeat("c", 64),
		},
		contenidoFuenteAutoridadPrueba(), "per_creador_version_maxima_00001", "alta_version_maxima",
		instanteFuenteAutoridadPrueba,
	)
	if err != nil {
		t.Fatal(err)
	}
	registradaEn := borrador.CreadaEn.Add(time.Hour)
	solicitud, evidencia := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, borrador, EstadoFuenteAutoridadPublicada, "per_publicador_version_maxima_001",
		"publicacion_version_maxima", "b", registradaEn, registradaEn.Add(-time.Minute),
	)
	publicada, err := borrador.AplicarTransicionV1(solicitud, evidencia, evidencia.ComprobadaEn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := publicada.NuevaVersionV1(
		contenidoFuenteAutoridadPrueba(), "per_creador_sucesora_maxima_0001", "sucesora_maxima",
		registradaEn.Add(time.Hour),
	); !errors.Is(err, ErrTransicionAutoridadInvalida) {
		t.Fatalf("sucesora con desbordamiento aceptada: %v", err)
	}
}

func TestFuenteAutoridadImpidePublicarATodosLosEditoresDelBorrador(t *testing.T) {
	fuente := nuevaFuenteAutoridadPrueba(t)
	primerEditor := "per_editor_primero_00000000000001"
	segundoEditor := "per_editor_segundo_00000000000001"
	contenido := fuente.Contenido
	contenido.Nombre += " revisado"
	primera, err := fuente.ActualizarBorrador(
		fuente.Revision, contenido, primerEditor, "primera_revision", fuente.CreadaEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido = primera.Contenido
	contenido.Preceptos = append(contenido.Preceptos, PreceptoFuenteAutoridad{Clave: "anexo_2", Cita: "Anexo II"})
	segunda, err := primera.ActualizarBorrador(
		primera.Revision, contenido, segundoEditor, "segunda_revision", fuente.CreadaEn.Add(2*time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	publicador := "per_publicador_independiente_000001"
	motivo := CodigoMotivoFuenteAutoridad("publicacion_independiente")
	registradaEn := fuente.CreadaEn.Add(3 * time.Hour)
	solicitud, evidencia := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, segunda, EstadoFuenteAutoridadPublicada, publicador, motivo, "b", registradaEn, registradaEn.Add(-time.Minute),
	)
	for _, actor := range []string{fuente.CreadaPor, primerEditor, segundoEditor} {
		if _, err := segunda.PrepararSolicitudTransicionV1(datosPreparacionTransicionFuenteAutoridadPrueba(
			EstadoFuenteAutoridadPublicada, actor, motivo, "solicitud:segregacion:"+actor, registradaEn,
		)); !errors.Is(err, ErrTransicionAutoridadInvalida) {
			t.Fatalf("el actor que intervino en el borrador %q pudo publicar: %v", actor, err)
		}
	}
	if _, err := segunda.AplicarTransicionV1(solicitud, evidencia, evidencia.ComprobadaEn); err != nil {
		t.Fatalf("publicador independiente rechazado: %v", err)
	}
}

func TestFuenteAutoridadExigeCronologiaEstricta(t *testing.T) {
	fuente := nuevaFuenteAutoridadPrueba(t)
	contenido := fuente.Contenido
	contenido.Nombre += " primera"
	primera, err := fuente.ActualizarBorrador(
		fuente.Revision, contenido, "per_editor_cronologia_primero_001", "primera_edicion",
		fuente.CreadaEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	contenido = primera.Contenido
	contenido.Nombre += " segunda"
	for _, instante := range []time.Time{primera.ultimaMutacionEn(), primera.ultimaMutacionEn().Add(-time.Microsecond)} {
		if _, err := primera.ActualizarBorrador(
			primera.Revision, contenido, "per_editor_cronologia_segundo_001", "segunda_edicion", instante,
		); !errors.Is(err, ErrFuenteAutoridadInvalida) {
			t.Fatalf("edición con tiempo no posterior %s: %v", instante, err)
		}
	}
	segunda, err := primera.ActualizarBorrador(
		primera.Revision, contenido, "per_editor_cronologia_segundo_001", "segunda_edicion",
		primera.ultimaMutacionEn().Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := segunda.PrepararSolicitudTransicionV1(datosPreparacionTransicionFuenteAutoridadPrueba(
		EstadoFuenteAutoridadPublicada, "per_publicador_cronologia_000001", "publicacion_cronologica",
		"solicitud:cronologia:simultanea", segunda.ultimaMutacionEn(),
	)); !errors.Is(err, ErrTransicionAutoridadInvalida) {
		t.Fatalf("publicación simultánea a última edición: %v", err)
	}
	publicadaEn := segunda.ultimaMutacionEn().Add(time.Hour)
	actor := "per_publicador_cronologia_000001"
	motivo := CodigoMotivoFuenteAutoridad("publicacion_cronologica")
	solicitud, evidencia := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, segunda, EstadoFuenteAutoridadPublicada, actor, motivo,
		"b", publicadaEn, publicadaEn.Add(-time.Minute),
	)
	publicada, err := segunda.AplicarTransicionV1(solicitud, evidencia, evidencia.ComprobadaEn)
	if err != nil {
		t.Fatal(err)
	}
	contenidoV2 := contenidoFuenteAutoridadPrueba()
	contenidoV2.Nombre += " versión sucesora"
	if _, err := publicada.NuevaVersionV1(
		contenidoV2, "per_creador_version_sucesora_00001", "nueva_version", publicadaEn,
	); !errors.Is(err, ErrTransicionAutoridadInvalida) {
		t.Fatalf("versión simultánea a su predecesora: %v", err)
	}
	if _, err := publicada.NuevaVersionV1(
		contenidoV2, "per_creador_version_sucesora_00001", "nueva_version", publicadaEn.Add(time.Hour),
	); err != nil {
		t.Fatalf("versión posterior rechazada: %v", err)
	}
}

func TestFuenteAutoridadNoReutilizaEvidenciaEntreCiclos(t *testing.T) {
	fuente := nuevaFuenteAutoridadPrueba(t)
	publicadaEn := fuente.CreadaEn.Add(time.Hour)
	actorPublica := "per_actor_publica_ciclos_0000001"
	motivoPublica := CodigoMotivoFuenteAutoridad("publicacion_ciclos")
	solicitudPublicacion, evidenciaPublicacion := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, fuente, EstadoFuenteAutoridadPublicada, actorPublica, motivoPublica,
		"b", publicadaEn, publicadaEn.Add(-time.Minute),
	)
	publicada, err := fuente.AplicarTransicionV1(solicitudPublicacion, evidenciaPublicacion, evidenciaPublicacion.ComprobadaEn)
	if err != nil {
		t.Fatal(err)
	}
	suspendidaEn := publicadaEn.Add(time.Hour)
	actorSuspende := "per_actor_suspende_ciclos_000001"
	motivoSuspende := CodigoMotivoFuenteAutoridad("suspension_ciclos")
	solicitudSuspension, evidenciaSuspension := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, publicada, EstadoFuenteAutoridadSuspendida, actorSuspende, motivoSuspende,
		"c", suspendidaEn, suspendidaEn.Add(-time.Minute),
	)
	suspendida, err := publicada.AplicarTransicionV1(solicitudSuspension, evidenciaSuspension, evidenciaSuspension.ComprobadaEn)
	if err != nil {
		t.Fatal(err)
	}
	levantadaEn := suspendidaEn.Add(time.Hour)
	actorLevanta := "per_actor_levanta_ciclos_00000001"
	motivoLevanta := CodigoMotivoFuenteAutoridad("levantamiento_ciclos")
	solicitudLevantamiento, evidenciaLevantamiento := solicitudYEvidenciaActoFuenteAutoridadPrueba(
		t, suspendida, EstadoFuenteAutoridadPublicada, actorLevanta, motivoLevanta,
		"d", levantadaEn, levantadaEn.Add(-time.Minute),
	)
	levantada, err := suspendida.AplicarTransicionV1(solicitudLevantamiento, evidenciaLevantamiento, evidenciaLevantamiento.ComprobadaEn)
	if err != nil {
		t.Fatal(err)
	}
	segundaSuspensionEn := levantadaEn.Add(time.Hour)
	segundaSolicitud, err := levantada.PrepararSolicitudTransicionV1(datosPreparacionTransicionFuenteAutoridadPrueba(
		EstadoFuenteAutoridadSuspendida, actorSuspende, motivoSuspende,
		"solicitud:segunda_suspension", segundaSuspensionEn,
	))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := levantada.AplicarTransicionV1(
		segundaSolicitud, evidenciaSuspension, segundaSuspensionEn,
	); !errors.Is(err, ErrEvidenciaActoAutoridadInvalida) {
		t.Fatalf("evidencia reutilizada en otro ciclo: %v", err)
	}
}

func TestFuenteAutoridadValidaSiemprePuedeProducirHuellasEnLimites(t *testing.T) {
	fuente := fuenteAutoridadVoluminosaConHistoriaPrueba(t, maximoTransicionesFuenteAutoridad)
	if err := fuente.Validar(); err != nil {
		t.Fatalf("fuente máxima válida: %v", err)
	}
	for nombre, obtener := range map[string]func() (string, error){
		"contenido": fuente.HuellaContenidoSHA256,
		"estado":    fuente.HuellaEstadoSHA256,
	} {
		huella, err := obtener()
		if err != nil || !esSHA256Autoridad(huella) {
			t.Fatalf("huella %s = %s, error=%v", nombre, huella, err)
		}
	}
	if err := contenidoFuenteAutoridadVoluminosoPrueba(180).Validar(); !errors.Is(err, ErrFuenteAutoridadInvalida) {
		t.Fatalf("contenido por encima del límite: %v", err)
	}
}

func BenchmarkValidarFuenteAutoridadConContenidoEHistoriaMaximos(b *testing.B) {
	fuente := fuenteAutoridadVoluminosaConHistoriaPrueba(b, maximoTransicionesFuenteAutoridad)
	if err := fuente.Validar(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.SetBytes(maximoBytesContenidoAutoridad)
	b.ResetTimer()
	for indice := 0; indice < b.N; indice++ {
		if err := fuente.Validar(); err != nil {
			b.Fatal(err)
		}
	}
}
