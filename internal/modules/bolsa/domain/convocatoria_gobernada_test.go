package domain

import (
	"errors"
	"testing"
	"time"
)

func contenidoConvocatoriaGobernadaPrueba() ContenidoPublicableConvocatoria {
	publica := convocatoriaPublicaValidaPrueba().DatosPublicos
	documentos := make([]DocumentoPublicableConvocatoria, len(publica.Documentos))
	for indice, documento := range publica.Documentos {
		documentos[indice] = DocumentoPublicableConvocatoria{
			Referencia: documento.Referencia, Tipo: documento.Tipo, Orden: documento.Orden,
			Titulo: documento.Titulo, Descripcion: documento.Descripcion,
			Formato: documento.Formato, URL: documento.URL,
		}
	}
	return ContenidoPublicableConvocatoria{
		IdentificadorPublico: publica.IdentificadorPublico, Tipo: publica.Tipo,
		CatalogoCategorias: publica.CatalogoCategorias,
		Categorias:         append([]string(nil), publica.Categorias...), Titulo: publica.Titulo,
		Resumen: publica.Resumen, Descripcion: publica.Descripcion,
		Plazos:     append([]PlazoConvocatoria(nil), publica.Plazos...),
		Requisitos: append([]RequisitoConvocatoria(nil), publica.Requisitos...),
		Documentos: documentos, Ayuda: append([]AyudaConvocatoria(nil), publica.Ayuda...),
	}
}

func configuracionConvocatoriaGobernadaPrueba() ConfiguracionFijadaConvocatoria {
	referencia := func(id string, version int, marca byte) ReferenciaConfiguracionConvocatoria {
		return ReferenciaConfiguracionConvocatoria{
			ID: id, Version: version, HuellaContenidoSHA256: cadenaRepetidaConvocatoria(marca),
		}
	}
	return ConfiguracionFijadaConvocatoria{
		Catalogos:        referencia("catalogos:bolsa", 3, '1'),
		Calendario:       referencia("calendario:auxiliar-2026", 2, '2'),
		ReglasBaremacion: referencia("baremo:auxiliar-2026", 5, '3'),
		FlujoProceso:     referencia("flujo:convocatoria-bolsa", 4, '4'),
		FlujoSolicitud:   referencia("flujo:solicitud-bolsa", 7, '5'),
		Documentos: []ReferenciaDocumentoOficialConvocatoria{{
			Rol: "bases", PublicacionRef: "doc:bases", DocumentoRef: "documento:logico:bases:001",
			VersionDocumento: 2, RepresentacionRef: "representacion:pdf:bases:002",
			HuellaContenidoSHA256: cadenaRepetidaConvocatoria('6'),
			FirmaValidadaRef:      "firma:validada:bases:002", ReciboCustodiaRef: "custodia:bases:002",
		}},
	}
}

func cadenaRepetidaConvocatoria(marca byte) string {
	bytes := make([]byte, 64)
	for indice := range bytes {
		bytes[indice] = marca
	}
	return string(bytes)
}

func versionConvocatoriaGobernadaPrueba(t *testing.T) VersionConvocatoriaGobernada {
	t.Helper()
	version, err := NuevaVersionConvocatoriaGobernada(DatosNuevaVersionConvocatoriaGobernada{
		ID: "proceso:bolsa:auxiliar-2026", CodigoVersionPublica: "v1",
		Contenido:     contenidoConvocatoriaGobernadaPrueba(),
		Configuracion: configuracionConvocatoriaGobernadaPrueba(),
		ExpedienteRef: "expediente:seleccion:2026-001",
		Motivo:        "Preparacion de la convocatoria aprobada por el servicio.",
		ActorID:       "persona:tecnica:001",
		Instante:      time.Date(2026, time.July, 2, 10, 30, 0, 999, time.FixedZone("CEST", 2*60*60)),
	})
	if err != nil {
		t.Fatalf("NuevaVersionConvocatoriaGobernada() error = %v", err)
	}
	return version
}

func evidenciasPublicacionPrueba(
	t *testing.T,
	version VersionConvocatoriaGobernada,
	fecha time.Time,
) (EvidenciaAprobacionConvocatoria, EvidenciaDependenciasConvocatoria) {
	t.Helper()
	huella, err := version.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return EvidenciaAprobacionConvocatoria{
			Accion: "publicar", Referencia: "aprobacion:publicacion:001",
			HuellaEvidenciaSHA256: cadenaRepetidaConvocatoria('a'),
			ConvocatoriaRef:       version.Referencia(), HuellaContenidoSHA256: huella,
			AprobadaPor: "persona:supervisora:001", AprobadaEn: fecha.Add(-2 * time.Minute),
		}, EvidenciaDependenciasConvocatoria{
			Referencia:            "comprobacion:dependencias:001",
			HuellaEvidenciaSHA256: cadenaRepetidaConvocatoria('b'),
			ConvocatoriaRef:       version.Referencia(), HuellaContenidoSHA256: huella,
			VerificadaEn: fecha.Add(-time.Minute),
		}
}

func publicarVersionConvocatoriaPrueba(
	t *testing.T,
	version VersionConvocatoriaGobernada,
	fecha time.Time,
) VersionConvocatoriaGobernada {
	t.Helper()
	aprobacion, dependencias := evidenciasPublicacionPrueba(t, version, fecha)
	publicada, err := version.Publicar(
		"persona:gestora:001", aprobacion, dependencias, "Publicacion aprobada.", fecha,
	)
	if err != nil {
		t.Fatalf("Publicar() error = %v", err)
	}
	return publicada
}

func TestBorradorNoFingePublicacionNiCongelaFase(t *testing.T) {
	version := versionConvocatoriaGobernadaPrueba(t)
	if version.EstadoGobierno != EstadoGobiernoConvocatoriaBorrador || version.Referencia() != "proceso:bolsa:auxiliar-2026#1" {
		t.Fatalf("identidad o gobierno incorrectos: %+v", version)
	}
	if _, err := version.ProyectarPublica("inscripcion", version.CreadaEn); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("un borrador produjo proyeccion publica: %v", err)
	}
	if version.CreadaEn.Location() != time.UTC || version.CreadaEn.Nanosecond()%1000 != 0 {
		t.Fatalf("instante no canonico: %s", version.CreadaEn)
	}
}

func TestPublicacionMantieneHuellaSemanticaYProyectaFaseDelFlujo(t *testing.T) {
	version := versionConvocatoriaGobernadaPrueba(t)
	fecha := version.CreadaEn.Add(time.Hour)
	huellaBorrador, err := version.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	aprobacion, dependencias := evidenciasPublicacionPrueba(t, version, fecha)
	if _, err := version.Publicar(
		version.CreadaPor, aprobacion, dependencias, "Publicacion.", fecha,
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("el creador pudo publicar: %v", err)
	}
	publicada := publicarVersionConvocatoriaPrueba(t, version, fecha)
	huellaPublicada, err := publicada.HuellaContenidoSHA256()
	if err != nil || huellaPublicada != huellaBorrador {
		t.Fatalf("publicar altero contenido: antes=%q despues=%q error=%v", huellaBorrador, huellaPublicada, err)
	}
	inscripcion, err := publicada.ProyectarPublica("inscripcion", fecha)
	if err != nil {
		t.Fatal(err)
	}
	alegaciones, err := publicada.ProyectarPublica("alegaciones", fecha.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if inscripcion.Estado != "inscripcion" || alegaciones.Estado != "alegaciones" ||
		!inscripcion.DatosPublicos.PublicadaEn.Equal(fecha) ||
		inscripcion.DatosPublicos.Documentos[0].PublicadoEn != fecha ||
		publicada.EstadoGobierno != EstadoGobiernoConvocatoriaPublicada {
		t.Fatalf("proyeccion incoherente: inscripcion=%+v alegaciones=%+v", inscripcion, alegaciones)
	}
}

func TestConfiguracionExigeDocumentoPublicoFirmadoYCustodiadoUnoAUno(t *testing.T) {
	contenido := contenidoConvocatoriaGobernadaPrueba()
	configuracion := configuracionConvocatoriaGobernadaPrueba()
	configuracion.Documentos[0].FirmaValidadaRef = ""
	if err := configuracion.ValidarPara(contenido); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("acepto documento sin firma: %v", err)
	}
	configuracion = configuracionConvocatoriaGobernadaPrueba()
	configuracion.Documentos[0].PublicacionRef = "doc:ajeno"
	if err := configuracion.ValidarPara(contenido); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("acepto documento ajeno: %v", err)
	}
}

func TestActualizarBorradorAplicaCASYComprometeDependenciaExacta(t *testing.T) {
	version := versionConvocatoriaGobernadaPrueba(t)
	huellaAnterior, err := version.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	configuracion := version.Configuracion
	configuracion.ReglasBaremacion.Version++
	configuracion.ReglasBaremacion.HuellaContenidoSHA256 = cadenaRepetidaConvocatoria('c')
	if _, err := version.ActualizarBorrador(
		2, version.Contenido, configuracion, "persona:tecnica:002", "Cambio de baremo.", version.CreadaEn.Add(time.Minute),
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("acepto revision obsoleta: %v", err)
	}
	if _, err := version.ActualizarBorrador(
		1, version.Contenido, version.Configuracion, "persona:tecnica:002", "Cambio sin efecto.", version.CreadaEn.Add(time.Minute),
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("acepto actualizacion sin cambios: %v", err)
	}
	actualizada, err := version.ActualizarBorrador(
		1, version.Contenido, configuracion, "persona:tecnica:002", "Cambio de baremo.", version.CreadaEn.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaPosterior, err := actualizada.HuellaContenidoSHA256()
	if err != nil || huellaPosterior == huellaAnterior || actualizada.Revision != 2 {
		t.Fatalf("cambio no comprometido: antes=%q despues=%q revision=%d error=%v",
			huellaAnterior, huellaPosterior, actualizada.Revision, err)
	}
	fecha := actualizada.UltimaModificacionEn.Add(time.Hour)
	aprobacion, dependencias := evidenciasPublicacionPrueba(t, actualizada, fecha)
	if _, err := actualizada.Publicar(
		actualizada.UltimaModificacionPor, aprobacion, dependencias, "Publicacion.", fecha,
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("el modificador pudo publicar: %v", err)
	}
}

func TestComprobacionDependenciasCaducadaImpidePublicar(t *testing.T) {
	version := versionConvocatoriaGobernadaPrueba(t)
	fecha := version.CreadaEn.Add(time.Hour)
	aprobacion, dependencias := evidenciasPublicacionPrueba(t, version, fecha)
	dependencias.VerificadaEn = fecha.Add(-vigenciaMaximaComprobacionDependencias - time.Microsecond)
	if _, err := version.Publicar(
		"persona:gestora:001", aprobacion, dependencias, "Publicacion.", fecha,
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("acepto comprobacion caducada: %v", err)
	}
}

func TestRetiradaExigeSeparacionYConservaContenido(t *testing.T) {
	version := versionConvocatoriaGobernadaPrueba(t)
	publicada := publicarVersionConvocatoriaPrueba(t, version, version.CreadaEn.Add(time.Hour))
	huellaAntes, err := publicada.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	fecha := publicada.PublicadaEn.Add(time.Hour)
	aprobacion := EvidenciaAprobacionConvocatoria{
		Accion: "retirar", Referencia: "aprobacion:retirada:001",
		HuellaEvidenciaSHA256: cadenaRepetidaConvocatoria('d'),
		ConvocatoriaRef:       publicada.Referencia(), HuellaContenidoSHA256: huellaAntes,
		AprobadaPor: "persona:inspectora:001", AprobadaEn: fecha.Add(-time.Minute),
	}
	if _, err := publicada.Retirar(
		publicada.PublicadaPor, aprobacion, "Retirada.", fecha,
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("quien publico pudo retirar: %v", err)
	}
	retirada, err := publicada.Retirar("persona:gestora:002", aprobacion, "Retirada aprobada.", fecha)
	if err != nil {
		t.Fatal(err)
	}
	huellaDespues, err := retirada.HuellaContenidoSHA256()
	if err != nil || huellaDespues != huellaAntes || retirada.EstadoGobierno != EstadoGobiernoConvocatoriaRetirada {
		t.Fatalf("retirada incoherente: antes=%q despues=%q estado=%q error=%v",
			huellaAntes, huellaDespues, retirada.EstadoGobierno, err)
	}
}

func TestNuevaVersionSustituyeExactamenteLaAnterior(t *testing.T) {
	primera := versionConvocatoriaGobernadaPrueba(t)
	primera = publicarVersionConvocatoriaPrueba(t, primera, primera.CreadaEn.Add(time.Hour))
	segunda, err := primera.NuevaVersion(
		"v2", primera.Contenido, primera.Configuracion, "expediente:seleccion:2026-002",
		"persona:tecnica:003", "Correccion de bases.", primera.PublicadaEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	if segunda.Secuencia != 2 || segunda.VersionAnteriorRef != primera.Referencia() {
		t.Fatalf("sucesion incorrecta: %+v", segunda)
	}
	segunda = publicarVersionConvocatoriaPrueba(t, segunda, segunda.CreadaEn.Add(time.Hour))
	sustituida, err := primera.SustituirPor(segunda)
	if err != nil {
		t.Fatal(err)
	}
	if sustituida.EstadoGobierno != EstadoGobiernoConvocatoriaSustituida ||
		sustituida.SustituidaPorRef != segunda.Referencia() || sustituida.SustituidaEn != segunda.PublicadaEn {
		t.Fatalf("sustitucion incorrecta: %+v", sustituida)
	}
}

func TestClonacionGobernadaNoComparteDocumentos(t *testing.T) {
	version := versionConvocatoriaGobernadaPrueba(t)
	clon, err := version.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	clon.Contenido.Documentos[0].Titulo = "Alterado"
	clon.Configuracion.Documentos[0].DocumentoRef = "documento:alterado"
	if version.Contenido.Documentos[0].Titulo == "Alterado" ||
		version.Configuracion.Documentos[0].DocumentoRef == "documento:alterado" {
		t.Fatal("ClonarCanonico comparte memoria")
	}
}
