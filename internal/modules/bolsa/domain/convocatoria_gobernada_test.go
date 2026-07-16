package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	dominiovec "vec-diputacion-granada/internal/vec/domain"
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

func definicionFlujoConvocatoriaGobernadaPrueba(t *testing.T) dominiovec.DefinicionFlujo {
	t.Helper()
	fecha := time.Date(2026, time.July, 1, 8, 0, 0, 0, time.UTC)
	referenciaEstado := func(clave string) dominiovec.ReferenciaEntradaCatalogo {
		return dominiovec.ReferenciaEntradaCatalogo{
			CatalogoID: "estados-convocatoria-bolsa", CatalogoVersion: 1,
			CatalogoHuellaSHA256: cadenaRepetidaConvocatoria('9'), EntradaClave: clave,
		}
	}
	borrador := dominiovec.DefinicionFlujo{
		ID: "convocatoria-bolsa", Version: 4, Revision: 1,
		VersionAnteriorRef: "convocatoria-bolsa:3", ModuloID: "bolsa",
		TipoEntidad: TipoEntidadFlujoConvocatoriaBolsa, Nombre: "Procedimiento de convocatoria de bolsa",
		Descripcion: "Fases publicables de una convocatoria de seleccion.",
		FuenteRef:   "procedimiento:seleccion-externa:v1", MotivoCreacion: "Version gobernada para pruebas.",
		EstadoInicial: "inscripcion", AccionInicio: "bolsa.convocatoria.iniciar",
		GarantiaInicio: dominiovec.AuthAssuranceHigh,
		Estados: []dominiovec.EstadoFlujoConfigurable{
			{Clave: "inscripcion", Catalogo: referenciaEstado("inscripcion"), Orden: 10},
			{Clave: "alegaciones", Catalogo: referenciaEstado("alegaciones"), Orden: 20, Terminal: true},
		},
		Transiciones: []dominiovec.TransicionFlujoConfigurable{{
			Clave: "abrir_alegaciones", Desde: []string{"inscripcion"}, Hacia: "alegaciones",
			Accion: "bolsa.convocatoria.abrir_alegaciones", ReglaRef: "regla:convocatoria:alegaciones:v1",
			Prioridad: 10, GarantiaMinima: dominiovec.AuthAssuranceHigh,
		}},
		Estado: dominiovec.EstadoDefinicionFlujoBorrador, CreadaPor: "persona:flujos:001", CreadaEn: fecha,
	}
	publicada, err := borrador.Publicar(
		"persona:flujos:002", "aprobacion:flujo:convocatoria:004",
		"Definicion revisada para el procedimiento.", fecha.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("publicar definicion de flujo de prueba: %v", err)
	}
	return publicada
}

func configuracionConvocatoriaGobernadaPrueba(t *testing.T) ConfiguracionFijadaConvocatoria {
	t.Helper()
	referencia := func(id string, version int, marca byte) ReferenciaConfiguracionConvocatoria {
		return ReferenciaConfiguracionConvocatoria{
			ID: id, Version: version, HuellaContenidoSHA256: cadenaRepetidaConvocatoria(marca),
		}
	}
	definicion := definicionFlujoConvocatoriaGobernadaPrueba(t)
	huellaFlujo, err := definicion.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return ConfiguracionFijadaConvocatoria{
		Catalogos:        referencia("catalogos:bolsa", 3, '1'),
		Calendario:       referencia("calendario:auxiliar-2026", 2, '2'),
		ReglasBaremacion: referencia("baremo:auxiliar-2026", 5, '3'),
		FlujoProceso: ReferenciaConfiguracionConvocatoria{
			ID: definicion.ID, Version: definicion.Version, HuellaContenidoSHA256: huellaFlujo,
		},
		FlujoSolicitud: referencia("solicitud-bolsa", 7, '5'),
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
		InstanciaFlujoRef: "instancia:flujo:convocatoria:001",
		Contenido:         contenidoConvocatoriaGobernadaPrueba(),
		Configuracion:     configuracionConvocatoriaGobernadaPrueba(t),
		ExpedienteRef:     "expediente:seleccion:2026-001",
		Motivo:            "Preparacion de la convocatoria aprobada por el servicio.",
		ActorID:           "persona:tecnica:001",
		Instante:          time.Date(2026, time.July, 2, 10, 30, 0, 999, time.FixedZone("CEST", 2*60*60)),
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
	huellaEstado, err := version.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	return EvidenciaAprobacionConvocatoria{
			Accion: "publicar", Referencia: "aprobacion:publicacion:001",
			HuellaEvidenciaSHA256: cadenaRepetidaConvocatoria('a'),
			ConvocatoriaRef:       version.Referencia(), Revision: version.Revision,
			HuellaContenidoSHA256: huella, HuellaEstadoSHA256: huellaEstado,
			AprobadaPor: "persona:supervisora:001", AprobadaEn: fecha.Add(-2 * time.Minute),
		}, EvidenciaDependenciasConvocatoria{
			Referencia:            "comprobacion:dependencias:001",
			HuellaEvidenciaSHA256: cadenaRepetidaConvocatoria('b'),
			ConvocatoriaRef:       version.Referencia(), Revision: version.Revision,
			HuellaContenidoSHA256: huella, HuellaEstadoSHA256: huellaEstado,
			VerificadaEn: fecha.Add(-time.Minute),
		}
}

func publicarVersionConvocatoriaPrueba(
	t *testing.T,
	version VersionConvocatoriaGobernada,
	fecha time.Time,
) VersionConvocatoriaGobernada {
	t.Helper()
	if version.Secuencia != 1 {
		t.Fatalf("el auxiliar solo publica la version inicial: %s", version.Referencia())
	}
	aprobacion, dependencias := evidenciasPublicacionPrueba(t, version, fecha)
	publicada, err := version.PublicarInicial(
		"persona:gestora:001", aprobacion, dependencias, "Publicacion aprobada.", fecha,
	)
	if err != nil {
		t.Fatalf("Publicar() error = %v", err)
	}
	return publicada
}

func instanciaFlujoConvocatoriaPrueba(
	version VersionConvocatoriaGobernada,
	estado string,
	actualizadaEn time.Time,
) dominiovec.InstanciaFlujo {
	instancia := dominiovec.InstanciaFlujo{
		ID: "instancia:flujo:convocatoria:001", TipoEntidad: TipoEntidadFlujoConvocatoriaBolsa,
		EntidadRef:                      version.ID,
		DefinicionRef:                   version.Configuracion.FlujoProceso.ReferenciaVersionada(),
		DefinicionContenidoHuellaSHA256: version.Configuracion.FlujoProceso.HuellaContenidoSHA256,
		EstadoActual:                    "inscripcion", Revision: 1, CreadaPor: "sistema:bolsa",
		CreadaEn: version.PublicadaEn,
	}
	if estado == "inscripcion" {
		return instancia
	}
	instancia.EstadoActual = estado
	instancia.Revision = 2
	instancia.UltimaTransicionClave = "abrir_alegaciones"
	instancia.UltimaDecisionReglaRef = "decision:regla:001"
	instancia.UltimaAutorizacionRef = "decision:autorizacion:001"
	instancia.UltimaCorrelacionRef = "correlacion:001"
	instancia.UltimoMotivo = "Apertura de la fase siguiente."
	instancia.ActualizadaPor = "persona:gestora:001"
	instancia.ActualizadaEn = actualizadaEn
	return instancia
}

func TestBorradorNoFingePublicacionNiCongelaFase(t *testing.T) {
	version := versionConvocatoriaGobernadaPrueba(t)
	if version.EstadoGobierno != EstadoGobiernoConvocatoriaBorrador || version.Referencia() != "proceso:bolsa:auxiliar-2026#1" {
		t.Fatalf("identidad o gobierno incorrectos: %+v", version)
	}
	if _, err := version.ProyectarPublica(
		dominiovec.InstanciaFlujo{}, dominiovec.DefinicionFlujo{},
	); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
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
	if _, err := version.PublicarInicial(
		aprobacion.AprobadaPor, aprobacion, dependencias, "Publicacion.", fecha,
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("el aprobador pudo materializar su aprobacion: %v", err)
	}
	if _, err := version.PublicarInicial(
		version.CreadaPor, aprobacion, dependencias, "Publicacion.", fecha,
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("el creador pudo publicar: %v", err)
	}
	publicada := publicarVersionConvocatoriaPrueba(t, version, fecha)
	definicion := definicionFlujoConvocatoriaGobernadaPrueba(t)
	huellaPublicada, err := publicada.HuellaContenidoSHA256()
	if err != nil || huellaPublicada != huellaBorrador {
		t.Fatalf("publicar altero contenido: antes=%q despues=%q error=%v", huellaBorrador, huellaPublicada, err)
	}
	inscripcion, err := publicada.ProyectarPublica(
		instanciaFlujoConvocatoriaPrueba(publicada, "inscripcion", fecha), definicion,
	)
	if err != nil {
		t.Fatal(err)
	}
	alegaciones, err := publicada.ProyectarPublica(
		instanciaFlujoConvocatoriaPrueba(publicada, "alegaciones", fecha.Add(time.Hour)), definicion,
	)
	if err != nil {
		t.Fatal(err)
	}
	if inscripcion.Estado != "inscripcion" || alegaciones.Estado != "alegaciones" ||
		!inscripcion.DatosPublicos.PublicadaEn.Equal(fecha) ||
		inscripcion.DatosPublicos.Documentos[0].PublicadoEn != fecha ||
		publicada.EstadoGobierno != EstadoGobiernoConvocatoriaPublicada {
		t.Fatalf("proyeccion incoherente: inscripcion=%+v alegaciones=%+v", inscripcion, alegaciones)
	}
	instanciaAjena := instanciaFlujoConvocatoriaPrueba(publicada, "inscripcion", fecha)
	instanciaAjena.DefinicionContenidoHuellaSHA256 = cadenaRepetidaConvocatoria('e')
	if _, err := publicada.ProyectarPublica(
		instanciaAjena, definicion,
	); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("acepto una fase sin vinculo con el flujo exacto: %v", err)
	}
}

func TestProyeccionExigeInstanciaReservadaCronologiaYEstadoDefinido(t *testing.T) {
	version := versionConvocatoriaGobernadaPrueba(t)
	publicada := publicarVersionConvocatoriaPrueba(t, version, version.CreadaEn.Add(time.Hour))
	definicion := definicionFlujoConvocatoriaGobernadaPrueba(t)

	gemela := instanciaFlujoConvocatoriaPrueba(publicada, "inscripcion", publicada.PublicadaEn)
	gemela.ID = "instancia:flujo:convocatoria:gemela"
	if _, err := publicada.ProyectarPublica(gemela, definicion); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("acepto una instancia gemela no reservada: %v", err)
	}

	anterior := instanciaFlujoConvocatoriaPrueba(publicada, "alegaciones", publicada.PublicadaEn.Add(time.Hour))
	anterior.CreadaEn = publicada.PublicadaEn.Add(-time.Minute)
	if _, err := publicada.ProyectarPublica(anterior, definicion); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("acepto un flujo iniciado antes de publicar pese a transicionar despues: %v", err)
	}

	desconocido := instanciaFlujoConvocatoriaPrueba(publicada, "inscripcion", publicada.PublicadaEn)
	desconocido.EstadoActual = "estado_ajeno"
	if _, err := publicada.ProyectarPublica(desconocido, definicion); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("acepto un estado ausente de la definicion exacta: %v", err)
	}
}

func TestContenidoAdmiteLasCincuentaYOchoCategoriasYRechazaTextoNoCanonico(t *testing.T) {
	contenido := contenidoConvocatoriaGobernadaPrueba()
	contenido.Categorias = make([]string, 58)
	for indice := range contenido.Categorias {
		contenido.Categorias[indice] = fmt.Sprintf("categoria_%02d", indice+1)
	}
	if err := contenido.Validar(); err != nil {
		t.Fatalf("rechazo 58 categorias: %v", err)
	}
	contenido.Titulo = "Ti\u0301tulo no NFC"
	if err := contenido.Validar(); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("acepto texto Unicode no canonico: %v", err)
	}
	contenido = contenidoConvocatoriaGobernadaPrueba()
	contenido.Descripcion = strings.Repeat("a", 48_001)
	if err := contenido.Validar(); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("acepto texto desproporcionado: %v", err)
	}
}

func TestConfiguracionExigeDocumentoPublicoFirmadoYCustodiadoUnoAUno(t *testing.T) {
	contenido := contenidoConvocatoriaGobernadaPrueba()
	configuracion := configuracionConvocatoriaGobernadaPrueba(t)
	configuracion.Documentos[0].FirmaValidadaRef = ""
	if err := configuracion.ValidarPara(contenido); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("acepto documento sin firma: %v", err)
	}
	configuracion = configuracionConvocatoriaGobernadaPrueba(t)
	configuracion.Documentos[0].PublicacionRef = "doc:ajeno"
	if err := configuracion.ValidarPara(contenido); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("acepto documento ajeno: %v", err)
	}
	for nombre, identificador := range map[string]string{
		"mayusculas":              "Solicitud-Bolsa",
		"separador de referencia": "solicitud:bolsa",
	} {
		t.Run(nombre, func(t *testing.T) {
			configuracion := configuracionConvocatoriaGobernadaPrueba(t)
			configuracion.FlujoSolicitud.ID = identificador
			if err := configuracion.ValidarPara(contenido); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
				t.Fatalf("acepto identificador imposible para DefinicionFlujo: %v", err)
			}
		})
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
	if _, err := actualizada.PublicarInicial(
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
	if _, err := version.PublicarInicial(
		"persona:gestora:001", aprobacion, dependencias, "Publicacion.", fecha,
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("acepto comprobacion caducada: %v", err)
	}
}

func TestAprobacionNoSeReutilizaTrasCambioAYVueltaA(t *testing.T) {
	inicial := versionConvocatoriaGobernadaPrueba(t)
	fechaPublicacion := inicial.CreadaEn.Add(time.Hour)
	aprobacionAntigua, dependenciasAntiguas := evidenciasPublicacionPrueba(t, inicial, fechaPublicacion)
	configuracionB := inicial.Configuracion
	configuracionB.Calendario.Version++
	configuracionB.Calendario.HuellaContenidoSHA256 = cadenaRepetidaConvocatoria('e')
	versionB, err := inicial.ActualizarBorrador(
		1, inicial.Contenido, configuracionB, "persona:tecnica:002", "Cambio B.", inicial.CreadaEn.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	vueltaA, err := versionB.ActualizarBorrador(
		2, inicial.Contenido, inicial.Configuracion, "persona:tecnica:002", "Vuelta a A.", inicial.CreadaEn.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	huellaInicial, _ := inicial.HuellaContenidoSHA256()
	huellaVuelta, _ := vueltaA.HuellaContenidoSHA256()
	if huellaInicial != huellaVuelta || vueltaA.Revision != 3 {
		t.Fatalf("la prueba no reconstruyo A-B-A: %q != %q, revision=%d", huellaInicial, huellaVuelta, vueltaA.Revision)
	}
	if _, err := vueltaA.PublicarInicial(
		"persona:gestora:001", aprobacionAntigua, dependenciasAntiguas, "Intento de replay.", fechaPublicacion,
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("reutilizo aprobacion de revision 1 en revision 3: %v", err)
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
	huellaEstado, err := publicada.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	aprobacion := EvidenciaAprobacionConvocatoria{
		Accion: "retirar", Referencia: "aprobacion:retirada:001",
		HuellaEvidenciaSHA256: cadenaRepetidaConvocatoria('d'),
		ConvocatoriaRef:       publicada.Referencia(), Revision: publicada.Revision,
		HuellaContenidoSHA256: huellaAntes, HuellaEstadoSHA256: huellaEstado,
		AprobadaPor: "persona:inspectora:001", AprobadaEn: fecha.Add(-time.Minute),
	}
	if _, err := publicada.Retirar(
		publicada.PublicadaPor, aprobacion, "Retirada.", fecha,
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("quien publico pudo retirar: %v", err)
	}
	if _, err := publicada.Retirar(
		aprobacion.AprobadaPor, aprobacion, "Retirada.", fecha,
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("el aprobador pudo materializar la retirada: %v", err)
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
	manipulada := segunda
	manipulada.VersionAnteriorRef = "proceso:ajeno#1"
	if err := manipulada.Validar(); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("acepto predecesora ajena: %v", err)
	}
	fechaPublicacion := segunda.CreadaEn.Add(time.Hour)
	aprobacion, dependencias := evidenciasPublicacionPrueba(t, segunda, fechaPublicacion)
	resultado, err := segunda.PublicarSucesora(
		primera, "persona:gestora:002", aprobacion, dependencias, "Publicacion sustitutiva.", fechaPublicacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	segunda = resultado.Publicada
	sustituida := resultado.Predecesora
	if sustituida.EstadoGobierno != EstadoGobiernoConvocatoriaSustituida ||
		sustituida.SustituidaPorRef != segunda.Referencia() || sustituida.SustituidaEn != segunda.PublicadaEn {
		t.Fatalf("sustitucion incorrecta: %+v", sustituida)
	}
	instancia := instanciaFlujoConvocatoriaPrueba(sustituida, "inscripcion", sustituida.SustituidaEn)
	definicion := definicionFlujoConvocatoriaGobernadaPrueba(t)
	if _, err := sustituida.ProyectarPublica(
		instancia, definicion,
	); !errors.Is(err, ErrVersionConvocatoriaGobernadaInvalida) {
		t.Fatalf("una version sustituida aparecio como activa: %v", err)
	}
	instanciaAnterior := instanciaFlujoConvocatoriaPrueba(primera, "inscripcion", primera.PublicadaEn)
	proyeccionSucesora, err := segunda.ProyectarPublica(instanciaAnterior, definicion)
	if err != nil || proyeccionSucesora.DatosPublicos == nil ||
		proyeccionSucesora.DatosPublicos.ActualizadaEn != segunda.PublicadaEn {
		t.Fatalf("la sucesora no reutilizo el flujo estable: proyeccion=%+v error=%v", proyeccionSucesora, err)
	}

	conOtroFlujo, err := primera.NuevaVersion(
		"v3", primera.Contenido, primera.Configuracion, "expediente:seleccion:2026-003",
		"persona:tecnica:004", "Intento de migracion implicita.", segunda.PublicadaEn.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	conOtroFlujo.Configuracion.FlujoProceso.Version++
	conOtroFlujo.Configuracion.FlujoProceso.HuellaContenidoSHA256 = cadenaRepetidaConvocatoria('f')
	conOtroFlujo, err = conOtroFlujo.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	fechaOtroFlujo := conOtroFlujo.CreadaEn.Add(time.Hour)
	aprobacionOtroFlujo, dependenciasOtroFlujo := evidenciasPublicacionPrueba(t, conOtroFlujo, fechaOtroFlujo)
	if _, err := conOtroFlujo.PublicarSucesora(
		primera, "persona:gestora:002", aprobacionOtroFlujo, dependenciasOtroFlujo,
		"Migracion implicita.", fechaOtroFlujo,
	); !errors.Is(err, ErrTransicionGobiernoConvocatoria) {
		t.Fatalf("acepto cambiar el flujo de una cadena iniciada: %v", err)
	}
}

func TestPublicarSucesoraDesdeRetiradaClonaPredecesora(t *testing.T) {
	inicial := versionConvocatoriaGobernadaPrueba(t)
	publicada := publicarVersionConvocatoriaPrueba(t, inicial, inicial.CreadaEn.Add(time.Hour))
	fechaRetirada := publicada.PublicadaEn.Add(time.Hour)
	huellaContenido, err := publicada.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaEstado, err := publicada.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	aprobacionRetirada := EvidenciaAprobacionConvocatoria{
		Accion: "retirar", Referencia: "aprobacion:retirada:clon",
		HuellaEvidenciaSHA256: cadenaRepetidaConvocatoria('d'), ConvocatoriaRef: publicada.Referencia(),
		Revision: publicada.Revision, HuellaContenidoSHA256: huellaContenido, HuellaEstadoSHA256: huellaEstado,
		AprobadaPor: "persona:inspectora:002", AprobadaEn: fechaRetirada.Add(-time.Minute),
	}
	retirada, err := publicada.Retirar(
		"persona:gestora:003", aprobacionRetirada, "Retirada para corregir las bases.", fechaRetirada,
	)
	if err != nil {
		t.Fatal(err)
	}
	sucesora, err := retirada.NuevaVersion(
		"v2", retirada.Contenido, retirada.Configuracion, "expediente:seleccion:2026-002",
		"persona:tecnica:004", "Nueva version tras retirada.", fechaRetirada.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	fechaPublicacion := sucesora.CreadaEn.Add(time.Hour)
	aprobacion, dependencias := evidenciasPublicacionPrueba(t, sucesora, fechaPublicacion)
	resultado, err := sucesora.PublicarSucesora(
		retirada, "persona:gestora:004", aprobacion, dependencias,
		"Publicacion posterior a retirada.", fechaPublicacion,
	)
	if err != nil {
		t.Fatal(err)
	}
	resultado.Predecesora.Contenido.Documentos[0].Titulo = "Alterado"
	resultado.Predecesora.Configuracion.Documentos[0].DocumentoRef = "documento:alterado"
	resultado.Predecesora.AprobacionPublicacion.Referencia = "aprobacion:alterada"
	if retirada.Contenido.Documentos[0].Titulo == "Alterado" ||
		retirada.Configuracion.Documentos[0].DocumentoRef == "documento:alterado" ||
		retirada.AprobacionPublicacion.Referencia == "aprobacion:alterada" {
		t.Fatal("la predecesora retirada comparte memoria con el resultado")
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

func TestRepresentacionesCanonicasCoincidenConSusHuellasYDevuelvenCopias(t *testing.T) {
	version := versionConvocatoriaGobernadaPrueba(t)
	representacion, err := version.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	huella, err := version.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	suma := sha256.Sum256(representacion)
	if hex.EncodeToString(suma[:]) != huella {
		t.Fatalf("huella de estado no coincide: %q", huella)
	}
	representacionContenido, err := version.RepresentacionContenidoCanonica()
	if err != nil {
		t.Fatal(err)
	}
	huellaContenido, err := version.HuellaContenidoSHA256()
	if err != nil {
		t.Fatal(err)
	}
	sumaContenido := sha256.Sum256(representacionContenido)
	if hex.EncodeToString(sumaContenido[:]) != huellaContenido {
		t.Fatalf("huella de contenido no coincide: %q", huellaContenido)
	}
	representacion[0] ^= 0xff
	repetida, err := version.RepresentacionCanonica()
	if err != nil || len(repetida) == 0 || repetida[0] == representacion[0] {
		t.Fatalf("la representacion comparte memoria: error=%v", err)
	}
}

func TestReferenciaDeInstanciaFormaParteDeAmbasHuellas(t *testing.T) {
	original := versionConvocatoriaGobernadaPrueba(t)
	manipulada := original
	manipulada.InstanciaFlujoRef = "instancia:flujo:convocatoria:002"
	manipulada, err := manipulada.ClonarCanonico()
	if err != nil {
		t.Fatal(err)
	}
	huellaContenidoOriginal, _ := original.HuellaContenidoSHA256()
	huellaContenidoManipulada, _ := manipulada.HuellaContenidoSHA256()
	huellaEstadoOriginal, _ := original.HuellaSHA256()
	huellaEstadoManipulada, _ := manipulada.HuellaSHA256()
	if huellaContenidoOriginal == huellaContenidoManipulada || huellaEstadoOriginal == huellaEstadoManipulada {
		t.Fatal("la identidad reservada de la instancia no quedo comprometida en ambas huellas")
	}
}

func TestVectoresGoldenDeRepresentacionesCanonicas(t *testing.T) {
	version := versionConvocatoriaGobernadaPrueba(t)
	estado, err := version.RepresentacionCanonica()
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := version.RepresentacionContenidoCanonica()
	if err != nil {
		t.Fatal(err)
	}
	huellaEstado := sha256.Sum256(estado)
	huellaContenido := sha256.Sum256(contenido)
	const esperadaEstado = "b5287ba99b36840ae3c1c99beeba50e0c73014fb51a33984d73017c0837e9f56"
	const esperadaContenido = "3aa5032b7d6ce8d15aa8ec9c90b3007477c185dce0ad10899ce2a4aaf1d71ba0"
	if obtenida := hex.EncodeToString(huellaEstado[:]); obtenida != esperadaEstado {
		t.Errorf("vector golden de estado cambiado: %s", obtenida)
	}
	if obtenida := hex.EncodeToString(huellaContenido[:]); obtenida != esperadaContenido {
		t.Errorf("vector golden de contenido cambiado: %s", obtenida)
	}
	if !strings.HasPrefix(string(estado), `{"esquema":"`+esquemaEstadoVersionConvocatoria+`"`) ||
		!strings.HasPrefix(string(contenido), `{"esquema":"`+esquemaContenidoVersionConvocatoria+`"`) {
		t.Fatal("las representaciones no declaran su esquema antes del material")
	}
}

func TestRepresentacionesCanonicasNoDependenDelOrdenDeColecciones(t *testing.T) {
	primera := versionConvocatoriaGobernadaPrueba(t)
	primera.Contenido.Categorias = []string{"tecnico_gestion", "auxiliar_administrativo"}
	segunda := primera
	segunda.Contenido.Categorias = []string{"auxiliar_administrativo", "tecnico_gestion"}
	estadoPrimera, errPrimera := primera.RepresentacionCanonica()
	estadoSegunda, errSegunda := segunda.RepresentacionCanonica()
	contenidoPrimera, errContenidoPrimera := primera.RepresentacionContenidoCanonica()
	contenidoSegunda, errContenidoSegunda := segunda.RepresentacionContenidoCanonica()
	if errPrimera != nil || errSegunda != nil || errContenidoPrimera != nil || errContenidoSegunda != nil {
		t.Fatalf("representar permutaciones: %v %v %v %v",
			errPrimera, errSegunda, errContenidoPrimera, errContenidoSegunda)
	}
	if !bytes.Equal(estadoPrimera, estadoSegunda) || !bytes.Equal(contenidoPrimera, contenidoSegunda) {
		t.Fatal("el orden de entrada altero los bytes canonicos")
	}
}
