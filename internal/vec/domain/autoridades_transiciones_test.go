package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

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
