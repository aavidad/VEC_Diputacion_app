package ports

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
)

func TestManifiestoGeneracionDocumentalEsOpacoCanonicoYDerivaDePlantillaPublicada(t *testing.T) {
	plantilla := plantillaGeneracionDocumentalPrueba()
	declaraciones := declaracionesGeneracionDocumentalPrueba()
	manifiesto, err := NuevoManifiestoGeneracionDocumental(
		plantilla, []DeclaracionRepresentacionGeneracionDocumental{declaraciones[1], declaraciones[0]},
	)
	if err != nil {
		t.Fatalf("crear manifiesto: %v", err)
	}
	mismaEntradaCanonica, err := NuevoManifiestoGeneracionDocumental(plantilla, declaraciones)
	if err != nil {
		t.Fatalf("crear manifiesto canonico: %v", err)
	}
	proyeccion, err := manifiesto.Proyeccion()
	otraProyeccion, otroErr := mismaEntradaCanonica.Proyeccion()
	if err != nil || otroErr != nil ||
		proyeccion.HuellaManifiestoSHA256 != otraProyeccion.HuellaManifiestoSHA256 ||
		proyeccion.PermisoGenerar != plantilla.PermisoGenerar || len(proyeccion.Pasos) != 2 ||
		proyeccion.Pasos[0].ReferenciaLogica != declaraciones[0].ReferenciaLogica ||
		!esSHA256Hexadecimal(proyeccion.Pasos[0].HuellaPasoSHA256) ||
		proyeccion.Pasos[0].PasoRef == proyeccion.Pasos[1].PasoRef {
		t.Fatalf("proyeccion no canonica: %#v / %#v / %v / %v", proyeccion, otraProyeccion, err, otroErr)
	}

	declaraciones[0].ReferenciaLogica = "representacion:mutada"
	posterior, err := manifiesto.Proyeccion()
	if err != nil || posterior.Pasos[0].ReferenciaLogica == declaraciones[0].ReferenciaLogica {
		t.Fatal("el manifiesto conserva un alias mutable de la entrada")
	}
	if _, err := json.Marshal(manifiesto); !errors.Is(err, ErrSerializacionManifiestoProhibida) {
		t.Fatalf("se serializo el manifiesto: %v", err)
	}
	if _, err := json.Marshal(proyeccion); !errors.Is(err, ErrSerializacionManifiestoProhibida) {
		t.Fatalf("se serializo la proyeccion: %v", err)
	}
	simple, err := NuevoManifiestoGeneracionDocumental(
		plantilla, []DeclaracionRepresentacionGeneracionDocumental{declaracionesGeneracionDocumentalPrueba()[0]},
	)
	if err != nil {
		t.Fatalf("una generacion simple debe ser un manifiesto N=1: %v", err)
	}
	proyeccionSimple, err := simple.Proyeccion()
	if err != nil || len(proyeccionSimple.Pasos) != 1 {
		t.Fatalf("manifiesto simple invalido: %#v, %v", proyeccionSimple, err)
	}
	texto := manifiesto.String() + proyeccion.String()
	if strings.Contains(texto, plantilla.PermisoGenerar) || strings.Contains(texto, plantilla.ID) {
		t.Fatalf("el formato revela datos del plan: %s", texto)
	}

	borrador := plantilla
	borrador.Estado = domain.EstadoPlantillaBorrador
	borrador.PublicadaPor = ""
	borrador.PublicadaEn = time.Time{}
	borrador.AprobacionRef = ""
	borrador.MotivoPublicacion = ""
	if _, err := NuevoManifiestoGeneracionDocumental(borrador, declaracionesGeneracionDocumentalPrueba()); !errors.Is(err, ErrManifiestoGeneracionDocumentalInvalido) {
		t.Fatalf("una plantilla no publicada creo manifiesto: %v", err)
	}
	var cero ManifiestoGeneracionDocumental
	if _, err := cero.Proyeccion(); !errors.Is(err, ErrManifiestoGeneracionDocumentalInvalido) {
		t.Fatalf("el valor cero fue valido: %v", err)
	}
}

func TestManifiestoRechazaRepresentacionesAmbiguasONoDeclaradas(t *testing.T) {
	plantilla := plantillaGeneracionDocumentalPrueba()
	base := declaracionesGeneracionDocumentalPrueba()
	casos := []struct {
		nombre string
		mutar  func([]DeclaracionRepresentacionGeneracionDocumental) []DeclaracionRepresentacionGeneracionDocumental
	}{
		{"sin pasos", func([]DeclaracionRepresentacionGeneracionDocumental) []DeclaracionRepresentacionGeneracionDocumental {
			return nil
		}},
		{"referencia duplicada", func(d []DeclaracionRepresentacionGeneracionDocumental) []DeclaracionRepresentacionGeneracionDocumental {
			d[1].ReferenciaLogica = d[0].ReferenciaLogica
			return d
		}},
		{"idempotencia duplicada", func(d []DeclaracionRepresentacionGeneracionDocumental) []DeclaracionRepresentacionGeneracionDocumental {
			d[1].ClaveIdempotencia = d[0].ClaveIdempotencia
			return d
		}},
		{"mime no exacto", func(d []DeclaracionRepresentacionGeneracionDocumental) []DeclaracionRepresentacionGeneracionDocumental {
			d[0].MIME = "Application/PDF"
			return d
		}},
		{"huella no canonica", func(d []DeclaracionRepresentacionGeneracionDocumental) []DeclaracionRepresentacionGeneracionDocumental {
			d[0].HuellaSHA256 = strings.ToUpper(d[0].HuellaSHA256)
			return d
		}},
		{"comodin", func(d []DeclaracionRepresentacionGeneracionDocumental) []DeclaracionRepresentacionGeneracionDocumental {
			d[0].ReferenciaLogica = "representacion:*"
			return d
		}},
		{"formato no admitido", func(d []DeclaracionRepresentacionGeneracionDocumental) []DeclaracionRepresentacionGeneracionDocumental {
			d[0].Formato = domain.FormatoDocumento("odt")
			d[0].MIME = "application/vnd.oasis.opendocument.text"
			return d
		}},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			declaraciones := append([]DeclaracionRepresentacionGeneracionDocumental(nil), base...)
			declaraciones = caso.mutar(declaraciones)
			if _, err := NuevoManifiestoGeneracionDocumental(plantilla, declaraciones); !errors.Is(err, ErrManifiestoGeneracionDocumentalInvalido) {
				t.Fatalf("caso admitido: %v", err)
			}
		})
	}
}

func TestContextoGeneracionDocumentalExigeRecursoPrevioPermisoYDecisionExactos(t *testing.T) {
	manifiesto, contexto, decision, recurso, vinculos, instante := contextoGeneracionDocumentalPrueba(t)
	proyeccion, err := contexto.Proyeccion()
	manifestado, errManifiesto := manifiesto.Proyeccion()
	if err != nil || errManifiesto != nil ||
		proyeccion.AccionNegocio != manifestado.PermisoGenerar ||
		proyeccion.HuellaManifiestoSHA256 != manifestado.HuellaManifiestoSHA256 ||
		proyeccion.HuellaPasoSHA256 != manifestado.Pasos[0].HuellaPasoSHA256 ||
		proyeccion.PasoRef != manifestado.Pasos[0].PasoRef {
		t.Fatalf("contexto documental no ligado: %#v / %#v / %v / %v", proyeccion, manifestado, err, errManifiesto)
	}
	segundo, err := contexto.DerivarPaso(manifestado.Pasos[1].PasoRef)
	if err != nil {
		t.Fatalf("derivar paso declarado: %v", err)
	}
	segundoProyectado, _ := segundo.Proyeccion()
	if segundoProyectado.HuellaPasoSHA256 != manifestado.Pasos[1].HuellaPasoSHA256 ||
		segundoProyectado.HuellaPlanEfectoSHA256 != proyeccion.HuellaPlanEfectoSHA256 {
		t.Fatal("derivar el paso altero o desligo el plan")
	}
	if _, err := contexto.DerivarPaso(PasoOperacionAlmacen("generar_documento_" + strings.Repeat("f", 64))); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("se derivo un paso no declarado: %v", err)
	}

	casos := []struct {
		nombre string
		mutar  func(*domain.DecisionAutorizacion)
	}{
		{"otra accion", func(d *domain.DecisionAutorizacion) { d.Accion = "bolsa.documentos.otra" }},
		{"campo", func(d *domain.DecisionAutorizacion) { d.CamposPermitidos = []string{"contenido"} }},
		{"obligacion", func(d *domain.DecisionAutorizacion) { d.Obligaciones = []string{"doble_control"} }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutada := clonarDecisionAutorizacionCanonica(decision)
			caso.mutar(&mutada)
			if _, err := NuevoContextoGeneracionDocumentalAlmacen(mutada, recurso, manifiesto, vinculos, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
				t.Fatalf("decision ampliada admitida: %v", err)
			}
		})
	}

	otroManifiesto, err := NuevoManifiestoGeneracionDocumental(
		plantillaGeneracionDocumentalPrueba(),
		[]DeclaracionRepresentacionGeneracionDocumental{declaracionesGeneracionDocumentalPrueba()[0]},
	)
	if err != nil {
		t.Fatalf("otro manifiesto: %v", err)
	}
	if _, err := NuevoContextoGeneracionDocumentalAlmacen(decision, recurso, otroManifiesto, vinculos, instante); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("decision reutilizada para otro manifiesto: %v", err)
	}

	base := domain.RecursoAutorizable{
		Referencia: "recurso:documento:otro", ModuloID: "bolsa", Tipo: "documento_bolsa",
		Atributos: map[string]string{AtributoAlmacenEfectoRef: vinculos.EfectoRef},
	}
	if _, err := VincularRecursoGeneracionDocumental(base, manifiesto, vinculos); !errors.Is(err, ErrManifiestoGeneracionDocumentalInvalido) {
		t.Fatalf("se permitio sombrear un atributo reservado: %v", err)
	}
	base.Atributos = map[string]string{"expediente_ref": "expediente:bolsa:001"}
	base.ModuloID = "dietas"
	if _, err := VincularRecursoGeneracionDocumental(base, manifiesto, vinculos); !errors.Is(err, ErrManifiestoGeneracionDocumentalInvalido) {
		t.Fatalf("se cruzo la plantilla a otro modulo: %v", err)
	}

	contextoAntiguo := contextoAlmacenVinculadoPrueba(t, AccionAlmacenEscribir)
	proyeccionAntigua, err := contextoAntiguo.Proyeccion()
	if err != nil || proyeccionAntigua.HuellaManifiestoSHA256 != "" || proyeccionAntigua.HuellaPasoSHA256 != "" {
		t.Fatalf("se rompio la compatibilidad de la fabrica existente: %#v, %v", proyeccionAntigua, err)
	}
	decisionCarga, recursoCarga, vinculosCarga, instanteCarga := autorizacionAlmacenPrueba(
		t, AccionNegocioPrepararCargaDocumental,
		[]string{"clasificacion", "contenido", "huella_sha256", "mime", "tamano"}, false,
	)
	recursoCarga.Atributos[AtributoAlmacenHuellaManifiestoSHA256] = strings.Repeat("e", 64)
	huellaCarga, err := recursoCarga.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("huella de carga cruzada: %v", err)
	}
	decisionCarga.ContextoRecursoHuellaSHA256 = huellaCarga
	if _, err := NuevoContextoPrepararCargaDirectaAlmacen(
		decisionCarga, recursoCarga, vinculosCarga, instanteCarga,
	); !errors.Is(err, ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("una fabrica existente acepto atributos del protocolo documental: %v", err)
	}
}

func TestSolicitudGuardarContenidoSoloAdmiteElPasoYLosBytesComprometidos(t *testing.T) {
	manifiesto, contexto, _, _, _, instante := contextoGeneracionDocumentalPrueba(t)
	proyeccion, _ := manifiesto.Proyeccion()
	solicitud := solicitudGuardarContenidoPasoPrueba(contexto, proyeccion.Pasos[0])
	if err := solicitud.ValidarEn(instante); err != nil {
		t.Fatalf("solicitud exacta denegada: %v", err)
	}

	segundoContexto, err := contexto.DerivarPaso(proyeccion.Pasos[1].PasoRef)
	if err != nil {
		t.Fatalf("derivar segundo paso: %v", err)
	}
	segundo := solicitudGuardarContenidoPasoPrueba(segundoContexto, proyeccion.Pasos[1])
	if err := segundo.Validar(); err != nil {
		t.Fatalf("segundo paso exacto denegado: %v", err)
	}
	cruzada := segundo
	cruzada.Contexto = contexto
	if !errors.Is(cruzada.Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("un paso uso los metadatos de otra representacion")
	}

	casos := []struct {
		nombre string
		mutar  func(*SolicitudGuardarContenido)
	}{
		{"referencia", func(s *SolicitudGuardarContenido) { s.DocumentoID = "representacion:otra" }},
		{"idempotencia", func(s *SolicitudGuardarContenido) { s.ClaveIdempotencia = "generacion:otra" }},
		{"zona", func(s *SolicitudGuardarContenido) { s.Zona = ZonaAlmacenCuarentena }},
		{"mime", func(s *SolicitudGuardarContenido) { s.MIME = "application/octet-stream" }},
		{"tamano", func(s *SolicitudGuardarContenido) { s.Tamano++ }},
		{"huella", func(s *SolicitudGuardarContenido) { s.HuellaSHA256 = strings.Repeat("f", 64) }},
		{"bytes", func(s *SolicitudGuardarContenido) { s.Contenido[0] ^= 1 }},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			mutada := solicitud
			mutada.Contenido = append([]byte(nil), solicitud.Contenido...)
			caso.mutar(&mutada)
			if !errors.Is(mutada.Validar(), ErrSolicitudAlmacenInvalida) {
				t.Fatal("se acepto una solicitud distinta del paso")
			}
		})
	}

	escritura := SolicitudEscribirObjeto{
		Contexto: contexto, ClaveIdempotencia: solicitud.ClaveIdempotencia, Zona: solicitud.Zona,
		MIME: solicitud.MIME, Tamano: solicitud.Tamano, HuellaSHA256: solicitud.HuellaSHA256,
		Contenido: bytes.NewReader(solicitud.Contenido),
	}
	if err := escritura.Validar(); err != nil {
		t.Fatalf("escritura exacta denegada: %v", err)
	}
	escritura.ClaveIdempotencia = "generacion:otra"
	if !errors.Is(escritura.Validar(), ErrAutorizacionAlmacenInvalida) {
		t.Fatalf("el puerto de objetos no cotejo el manifiesto: %v", escritura.Validar())
	}

	contextoGenerico := contextoAlmacenVinculadoPrueba(t, AccionAlmacenEscribir)
	generica := solicitud
	generica.Contexto = contextoGenerico
	if !errors.Is(generica.Validar(), ErrSolicitudAlmacenInvalida) {
		t.Fatal("un contexto generico autorizo generacion documental")
	}
}

func TestEvidenciaYContratoDuraderoConservanManifiestoPasoYEstadoIndeterminado(t *testing.T) {
	manifiesto, contexto, decision, _, _, instante := contextoGeneracionDocumentalPrueba(t)
	proyeccion, _ := manifiesto.Proyeccion()
	solicitud := solicitudGuardarContenidoPasoPrueba(contexto, proyeccion.Pasos[0])
	objeto := ReferenciaObjetoAlmacen{Referencia: "objeto:documento:001", Version: "version:1"}
	evidencia := evidenciaAlmacenVinculadaPrueba(
		contexto, objeto, "evidencia:documento:001", "almacen_s3_corporativo", "", instante,
	)
	guardado := ContenidoDocumentoGuardado{
		ReferenciaLogica: solicitud.DocumentoID, Referencia: objeto.Referencia, Version: objeto.Version,
		ConectorID: evidencia.ConectorID, Zona: solicitud.Zona, MIME: solicitud.MIME,
		HuellaSHA256: solicitud.HuellaSHA256, Tamano: solicitud.Tamano, EvidenciaOperacion: evidencia,
	}
	if err := guardado.ValidarContra(solicitud); err != nil {
		t.Fatalf("resultado exacto denegado: %v", err)
	}
	mutado := guardado
	mutado.EvidenciaOperacion.HuellaPasoSHA256 = strings.Repeat("f", 64)
	if !errors.Is(mutado.ValidarContra(solicitud), ErrSolicitudAlmacenInvalida) {
		t.Fatal("se acepto evidencia de otro paso")
	}

	reserva := SolicitudReservarEfectoGeneracionDocumental{Contexto: contexto, Manifiesto: manifiesto}
	if err := reserva.ValidarEn(instante); err != nil {
		t.Fatalf("reserva exacta denegada: %v", err)
	}
	if !errors.Is(reserva.ValidarEn(decision.ValidaHasta), ErrReservaEfectoGeneracionDocumentalInvalida) {
		t.Fatal("se reservo con la decision caducada")
	}
	estados := make([]EstadoPasoDuraderoGeneracionDocumental, 0, len(proyeccion.Pasos))
	for _, paso := range proyeccion.Pasos {
		estados = append(estados, EstadoPasoDuraderoGeneracionDocumental{
			PasoRef: paso.PasoRef, HuellaPasoSHA256: paso.HuellaPasoSHA256,
			Estado: EstadoPasoEfectoDocumentalReservado,
		})
	}
	contextoProyectado, _ := contexto.Proyeccion()
	resultadoReserva := ResultadoReservaEfectoGeneracionDocumental{
		ReservaRef: "reserva:documental:001", EfectoRef: contextoProyectado.EfectoRef,
		HuellaDecisionSHA256:   contextoProyectado.HuellaDecisionSHA256,
		HuellaPlanEfectoSHA256: contextoProyectado.HuellaPlanEfectoSHA256,
		HuellaManifiestoSHA256: contextoProyectado.HuellaManifiestoSHA256,
		Pasos:                  estados,
	}
	if err := resultadoReserva.ValidarContra(reserva); err != nil {
		t.Fatalf("resultado de reserva exacto denegado: %v", err)
	}
	resultadoMutado := resultadoReserva
	resultadoMutado.Pasos = append([]EstadoPasoDuraderoGeneracionDocumental(nil), estados...)
	resultadoMutado.Pasos[0].Estado = EstadoPasoEfectoDocumentalIndeterminado
	resultadoMutado.Pasos[0].IncidenteRef = "incidente:remoto:001"
	if !errors.Is(resultadoMutado.ValidarContra(reserva), ErrReservaEfectoGeneracionDocumentalInvalida) {
		t.Fatal("una reserva nueva nacio indeterminada")
	}
	resultadoMutado.Repetida = true
	if err := resultadoMutado.ValidarContra(reserva); err != nil {
		t.Fatalf("no se pudo recuperar un estado indeterminado durable: %v", err)
	}
	confirmado := resultadoReserva
	confirmado.Repetida = true
	confirmado.Pasos = append([]EstadoPasoDuraderoGeneracionDocumental(nil), estados...)
	confirmado.Pasos[0] = EstadoPasoDuraderoGeneracionDocumental{
		PasoRef: confirmado.Pasos[0].PasoRef, HuellaPasoSHA256: confirmado.Pasos[0].HuellaPasoSHA256,
		Estado: EstadoPasoEfectoDocumentalConfirmado, Objeto: objeto,
		ConectorID: evidencia.ConectorID, EvidenciaOperacionRef: evidencia.Referencia,
	}
	if err := confirmado.ValidarContra(reserva); err != nil {
		t.Fatalf("no se pudo recuperar un paso confirmado con conector exacto: %v", err)
	}
	confirmado.Pasos[0].ConectorID = ""
	if !errors.Is(confirmado.ValidarContra(reserva), ErrReservaEfectoGeneracionDocumentalInvalida) {
		t.Fatal("un replay confirmado perdio la identidad del conector")
	}
	reordenado := resultadoReserva
	reordenado.Repetida = true
	reordenado.Pasos = []EstadoPasoDuraderoGeneracionDocumental{estados[1], estados[0]}
	if !errors.Is(reordenado.ValidarContra(reserva), ErrReservaEfectoGeneracionDocumentalInvalida) {
		t.Fatal("la reserva acepto pasos reordenados")
	}

	confirmacion := SolicitudConfirmarPasoGeneracionDocumental{
		ReservaRef: resultadoReserva.ReservaRef, Contexto: contexto, Guardado: guardado,
	}
	if err := confirmacion.Validar(); err != nil {
		t.Fatalf("confirmacion exacta denegada: %v", err)
	}
	indeterminado := SolicitudMarcarPasoGeneracionDocumentalIndeterminado{
		ReservaRef: resultadoReserva.ReservaRef, Contexto: contexto, IncidenteRef: "incidente:remoto:001",
	}
	if err := indeterminado.Validar(); err != nil {
		t.Fatalf("marcado indeterminado exacto denegado: %v", err)
	}
	indeterminado.IncidenteRef = "incidente:*"
	if !errors.Is(indeterminado.Validar(), ErrReservaEfectoGeneracionDocumentalInvalida) {
		t.Fatal("se admitio un incidente ambiguo")
	}

	segundoContexto, _ := contexto.DerivarPaso(proyeccion.Pasos[1].PasoRef)
	confirmacion.Contexto = segundoContexto
	if !errors.Is(confirmacion.Validar(), ErrReservaEfectoGeneracionDocumentalInvalida) {
		t.Fatal("se confirmo un resultado con otro paso del mismo plan")
	}
	reserva.Contexto = segundoContexto
	if !errors.Is(reserva.ValidarEn(instante), ErrReservaEfectoGeneracionDocumentalInvalida) {
		t.Fatal("se intento reservar el plan desde un paso derivado")
	}
}

func plantillaGeneracionDocumentalPrueba() domain.PlantillaDocumento {
	fecha := time.Date(2026, time.July, 15, 7, 0, 0, 0, time.UTC)
	return domain.PlantillaDocumento{
		ID: "certificado_bolsa", Version: 3, ModuloID: "bolsa", TipoDocumental: "certificado",
		Nombre: "Certificado de bolsa", Titulo: "Certificado {{numero}}",
		Parrafos: []string{"Se certifica {{contenido}}."},
		Campos: []domain.CampoPlantillaDocumento{
			{Clave: "numero", Etiqueta: "Numero", Obligatorio: true},
			{Clave: "contenido", Etiqueta: "Contenido", Obligatorio: true},
		},
		Formatos:       []domain.FormatoDocumento{domain.FormatoDocumentoDOCX, domain.FormatoDocumentoPDF},
		PermisoGenerar: "bolsa.documentos.generar", GarantiaMinima: domain.AuthAssuranceSubstantial,
		Estado: domain.EstadoPlantillaPublicada, CreadaPor: "tecnico:rrhh:001", CreadaEn: fecha.Add(-time.Hour),
		PublicadaPor: "jefatura:rrhh:001", PublicadaEn: fecha,
		AprobacionRef: "aprobacion:plantilla:003", MotivoPublicacion: "aprobacion_tecnica",
	}
}

func declaracionesGeneracionDocumentalPrueba() []DeclaracionRepresentacionGeneracionDocumental {
	pdf := []byte("%PDF-1.7 contenido de prueba")
	docx := []byte("PK contenido docx de prueba")
	return []DeclaracionRepresentacionGeneracionDocumental{
		{
			ReferenciaLogica: "representacion:documento:docx", ClaveIdempotencia: "generacion:documento:docx",
			Formato: domain.FormatoDocumentoDOCX, Zona: ZonaAlmacenAdmitida, MIME: domain.FormatoDocumentoDOCX.MIME(),
			Tamano: int64(len(docx)), HuellaSHA256: huellaBytesDocumentalesPrueba(docx),
		},
		{
			ReferenciaLogica: "representacion:documento:pdf", ClaveIdempotencia: "generacion:documento:pdf",
			Formato: domain.FormatoDocumentoPDF, Zona: ZonaAlmacenAdmitida, MIME: domain.FormatoDocumentoPDF.MIME(),
			Tamano: int64(len(pdf)), HuellaSHA256: huellaBytesDocumentalesPrueba(pdf),
		},
	}
}

func contextoGeneracionDocumentalPrueba(t *testing.T) (
	ManifiestoGeneracionDocumental,
	ContextoOperacionAlmacen,
	domain.DecisionAutorizacion,
	domain.RecursoAutorizable,
	VinculosOperacionAlmacen,
	time.Time,
) {
	t.Helper()
	plantilla := plantillaGeneracionDocumentalPrueba()
	manifiesto, err := NuevoManifiestoGeneracionDocumental(plantilla, declaracionesGeneracionDocumentalPrueba())
	if err != nil {
		t.Fatalf("manifiesto de prueba: %v", err)
	}
	vinculos := VinculosOperacionAlmacen{
		OperacionRef: "operacion:generacion:001", CargaRef: "generacion:documental:001",
		Clasificacion:       "datos_personales_alta",
		SujetoSeudonimoHMAC: "hmac-sha256:sujeto_documental_v1:" + strings.Repeat("a", 64),
		HuellaSolicitudHMAC: "hmac-sha256:solicitud_documental_v1:" + strings.Repeat("b", 64),
		EfectoRef:           "efecto:generacion:001",
	}
	base := domain.RecursoAutorizable{
		Referencia: "recurso:documento:001", ModuloID: "bolsa", Tipo: "documento_bolsa",
		Ambitos:   map[string]string{"organizacion": "diputacion_granada"},
		Atributos: map[string]string{"expediente_ref": "expediente:bolsa:001"},
	}
	recurso, err := VincularRecursoGeneracionDocumental(base, manifiesto, vinculos)
	if err != nil {
		t.Fatalf("vincular recurso: %v", err)
	}
	decision, instante := decisionAutorizacionReforzadaPrueba(t)
	huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatalf("huella de recurso: %v", err)
	}
	proyeccion, _ := manifiesto.Proyeccion()
	decision.Accion = proyeccion.PermisoGenerar
	decision.RecursoRef = recurso.Referencia
	decision.ModuloID = recurso.ModuloID
	decision.TipoRecurso = recurso.Tipo
	decision.ContextoRecursoHuellaSHA256 = huellaRecurso
	decision.Finalidad = "generar_documento"
	decision.CorrelacionRef = "correlacion:generacion:001"
	decision.CamposPermitidos = nil
	decision.Obligaciones = nil
	if err := decision.ValidarEvidenciaInstantanea(); err != nil {
		t.Fatalf("decision documental: %v", err)
	}
	contexto, err := NuevoContextoGeneracionDocumentalAlmacen(decision, recurso, manifiesto, vinculos, instante)
	if err != nil {
		t.Fatalf("contexto documental: %v", err)
	}
	return manifiesto, contexto, decision, recurso, vinculos, instante
}

func solicitudGuardarContenidoPasoPrueba(
	contexto ContextoOperacionAlmacen,
	paso ProyeccionPasoGeneracionDocumental,
) SolicitudGuardarContenido {
	var contenido []byte
	switch paso.Formato {
	case domain.FormatoDocumentoPDF:
		contenido = []byte("%PDF-1.7 contenido de prueba")
	case domain.FormatoDocumentoDOCX:
		contenido = []byte("PK contenido docx de prueba")
	}
	return SolicitudGuardarContenido{
		Contexto: contexto, ClaveIdempotencia: paso.ClaveIdempotencia,
		DocumentoID: paso.ReferenciaLogica, Zona: paso.Zona, MIME: paso.MIME,
		HuellaSHA256: paso.HuellaSHA256, Tamano: paso.Tamano, Contenido: contenido,
	}
}

func huellaBytesDocumentalesPrueba(contenido []byte) string {
	suma := sha256.Sum256(contenido)
	return hex.EncodeToString(suma[:])
}
