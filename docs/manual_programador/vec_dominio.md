# Nucleo VEC: dominio

Parte del [Manual del programador](LEEME.md). Fichero generado con
`scripts/generar_manual_programador.py`; no editar a mano.

## Paquete `internal/vec/domain`

> Tipos puros del shell VEC, sin HTTP ni persistencia concreta.

### Constantes

```go
const (
	// VersionFormatoAtestacionAutorizacionV1 identifica la representacion
	// binaria VEC-AD-1. No identifica ni aprueba un algoritmo de firma.
	VersionFormatoAtestacionAutorizacionV1 uint16 = 1

	// EsquemaMensajeAtestacionAutorizacionV1 separa este mensaje de cualquier
	// otro uso criptografico. El byte cero posterior forma parte del esquema.
	EsquemaMensajeAtestacionAutorizacionV1 = "VEC-AUTORIZACION-ATESTACION-V1-AUTENTICACION-ACTOR"

	// TamanoMaximoMensajeAtestacionAutorizacionV1 mantiene el mensaje como una
	// capacidad breve y coincide con el techo del documento de decision durable.
	TamanoMaximoMensajeAtestacionAutorizacionV1 = 512 * 1024
)
const (
	// VersionFormatoAtestacionAutorizacionV2 identifica exclusivamente la
	// representacion binaria VEC-AD-2. No identifica ni aprueba una suite de
	// firma, un proveedor criptografico o un formato de sobre.
	VersionFormatoAtestacionAutorizacionV2 uint16 = 2

	// EsquemaMensajeAtestacionAutorizacionV2 separa VEC-AD-2 de VEC-AD-1 y de
	// cualquier otro uso criptografico. El byte cero posterior forma parte del
	// dominio binario firmado.
	EsquemaMensajeAtestacionAutorizacionV2 = "VEC-AUTORIZACION-ATESTACION-V2-SOLICITUD-LIGADA-MOTIVO-CATALOGADO"

	// TamanoMaximoMensajeAtestacionAutorizacionV2 conserva el mismo presupuesto
	// acotado de 512 KiB que VEC-AD-1. Se declara por separado para que el perfil
	// V2 sea autocontenido aunque reutilice sus primitivas binarias seguras.
	TamanoMaximoMensajeAtestacionAutorizacionV2 = 512 * 1024
)
const (
	// VersionFormatoAtestacionDenegacionAutorizacionV1 identifica el contrato
	// binario VEC-AD-D-1. Es un espacio de versiones propio: el valor 1 no lo
	// convierte en VEC-AD-1 ni permite tratar una denegacion como concesion.
	VersionFormatoAtestacionDenegacionAutorizacionV1 uint16 = 1

	// EsquemaMensajeAtestacionDenegacionAutorizacionV1 separa de forma
	// criptografica una prueba negativa de VEC-AD-1, VEC-AD-2 y cualquier otro
	// protocolo. El byte cero posterior tambien forma parte del mensaje.
	EsquemaMensajeAtestacionDenegacionAutorizacionV1 = "VEC-AUTORIZACION-DENEGACION-ATESTACION-V1-SOLICITUD-LIGADA-MOTIVO-CATALOGADO"

	// TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1 fija el mismo techo
	// operativo de 512 KiB mediante una constante independiente.
	TamanoMaximoMensajeAtestacionDenegacionAutorizacionV1 = 512 * 1024
)
const (
	AccionFuenteAutoridadBorradorCreado      = "vec.fuentes_autoridad.borrador.creado"
	AccionFuenteAutoridadBorradorActualizado = "vec.fuentes_autoridad.borrador.actualizado"
	AccionFuenteAutoridadPublicada           = "vec.fuentes_autoridad.publicada"
	AccionFuenteAutoridadSuspendida          = "vec.fuentes_autoridad.suspendida"
	AccionFuenteAutoridadSuspensionLevantada = "vec.fuentes_autoridad.suspension_levantada"
	AccionFuenteAutoridadDerogada            = "vec.fuentes_autoridad.derogada"
)
const (
	// VigenciaMaximaDecisionAutorizacion limita la reutilizacion de una
	// decision. Un adaptador puede emitir decisiones con menor vigencia.
	VigenciaMaximaDecisionAutorizacion = 5 * time.Minute
)
const (
	// EsquemaHuellaSolicitudAutorizacionV2 identifica un documento canonico
	// cerrado. La huella resultante acredita solo integridad estructural: no es
	// una firma ni demuestra que la solicitud proceda del PDP o del registro.
	EsquemaHuellaSolicitudAutorizacionV2 = "vec.autorizacion.solicitud.v2.efectiva-minimizada"
	// EsquemaHuellaMotivoAutorizacionV2 compromete una referencia completa a
	// una entrada de catalogo: identificador, version, huella y clave. No
	// constituye una firma ni demuestra por si sola que el catalogo exista,
	// este publicado o proceda de una frontera confiable.
	EsquemaHuellaMotivoAutorizacionV2 = "vec.autorizacion.motivo.v2.referencia-opaca-catalogada"
)
const (
	EsquemaManifiestoPreparacionCargaDirectaV1 = "vec.carga-directa.manifiesto-preparacion.v1"
)
const (
	AccionCatalogoBorradorCreado      = "vec.catalogos.borrador.creado"
	AccionCatalogoBorradorActualizado = "vec.catalogos.borrador.actualizado"
	AccionCatalogoPublicado           = "vec.catalogos.publicado"
	AccionCatalogoRetirado            = "vec.catalogos.retirado"
)
const (
	AccionPoliticaCotejoBorradorCreada      = "vec.documentos.cotejo.politica.borrador.creado"
	AccionPoliticaCotejoBorradorActualizada = "vec.documentos.cotejo.politica.borrador.actualizado"
	AccionPoliticaCotejoPublicada           = "vec.documentos.cotejo.politica.publicada"
	AccionPoliticaCotejoRetirada            = "vec.documentos.cotejo.politica.retirada"
	AccionCodigoCotejoReservado             = "vec.documentos.cotejo.codigo.reservado"
	AccionCodigoCotejoActivado              = "vec.documentos.cotejo.codigo.activado"
	AccionCodigoCotejoRetirado              = "vec.documentos.cotejo.codigo.retirado"
	AccionCodigoCotejoSustituido            = "vec.documentos.cotejo.codigo.sustituido"
	AccionConsultaPublicaCotejo             = "vec.documentos.cotejo.consulta.publica"
	AccionConsultaProtegidaCotejo           = "vec.documentos.cotejo.consulta.protegida"
)
const (
	AccionDefinicionFlujoBorradorCreada      = "vec.flujos.definicion.borrador.creada"
	AccionDefinicionFlujoBorradorActualizada = "vec.flujos.definicion.borrador.actualizada"
	AccionDefinicionFlujoPublicada           = "vec.flujos.definicion.publicada"
	AccionDefinicionFlujoRetirada            = "vec.flujos.definicion.retirada"
	AccionInstanciaFlujoIniciada             = "vec.flujos.instancia.iniciada"
	AccionInstanciaFlujoTransicionada        = "vec.flujos.instancia.transicionada"
	AccionDecisionReglaFlujoRegistrada       = "vec.flujos.regla.decision.registrada"
)
const (
	// VersionVinculoAutenticacionActorV1 identifica el bloque cerrado que liga
	// una decision con la sesion revalidada y el documento de actor resuelto.
	// El valor cero nunca identifica una version compatible.
	VersionVinculoAutenticacionActorV1 uint16 = 1
)
```

### Variables

```go
var (
	// ErrParseoHistoricoAtestacionAutorizacionV1Invalido identifica una
	// representacion VEC-AD-1 que no es exacta, canonica y completa. El parseo
	// solo produce datos nominales: superar esta validacion no prueba una firma
	// ni concede autoridad alguna.
	ErrParseoHistoricoAtestacionAutorizacionV1Invalido = errors.New("vec: parseo historico VEC-AD-1 invalido")

	// ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida
	// evita que una proyeccion con identificadores de persona y sesion termine
	// accidentalmente en registros o serializadores generales. VEC-AD-1 solo se
	// vuelve a emitir dentro del comprobador canonico privado de este archivo.
	ErrSerializacionProyeccionHistoricaAtestacionAutorizacionV1Prohibida = errors.New("vec: serializacion de proyeccion historica VEC-AD-1 prohibida")
)
var (
	// ErrParseoAtestacionAutorizacionV2Invalido identifica un VEC-AD-2 que no
	// es completo, canonico o semanticamente coherente. Parsearlo no verifica
	// una firma y nunca concede autoridad.
	ErrParseoAtestacionAutorizacionV2Invalido = errors.New("vec: parseo no autoritativo VEC-AD-2 invalido")

	// ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida impide que
	// la proyeccion nominal pueda terminar accidentalmente en logs o codecs.
	ErrSerializacionProyeccionAtestacionAutorizacionV2Prohibida = errors.New("vec: serializacion de proyeccion no autoritativa VEC-AD-2 prohibida")
)
var (
	// ErrParseoAtestacionDenegacionAutorizacionV1Invalido identifica un
	// VEC-AD-D-1 no exacto. La proyeccion resultante sigue sin demostrar firma,
	// procedencia ni autoridad para mutar estado.
	ErrParseoAtestacionDenegacionAutorizacionV1Invalido = errors.New("vec: parseo no autoritativo VEC-AD-D-1 invalido")

	// ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida
	// bloquea todos los codecs generales sobre la proyeccion negativa.
	ErrSerializacionProyeccionAtestacionDenegacionAutorizacionV1Prohibida = errors.New("vec: serializacion de proyeccion no autoritativa VEC-AD-D-1 prohibida")
)
var (
	ErrFuenteAutoridadInvalida        = errors.New("vec: fuente de autoridad invalida")
	ErrReferenciaAutoridadInvalida    = errors.New("vec: referencia de autoridad invalida")
	ErrEvidenciaActoAutoridadInvalida = errors.New("vec: evidencia de acto de autoridad invalida")
	ErrTransicionAutoridadInvalida    = errors.New("vec: transicion de fuente de autoridad invalida")
	ErrRevisionAutoridadEnConflicto   = errors.New("vec: revision de fuente de autoridad en conflicto")
	ErrSolicitudAutoridadObsoleta     = errors.New("vec: solicitud de transicion de autoridad obsoleta")
	ErrSolicitudAutoridadExpirada     = errors.New("vec: solicitud de transicion de autoridad expirada")
	ErrLimiteAutoridadAlcanzado       = errors.New("vec: limite de fuente de autoridad alcanzado")
)
var (
	ErrSolicitudAutorizacionInvalida = errors.New("vec: solicitud de autorizacion invalida")
	ErrConfiguracionAccesoInvalida   = errors.New("vec: configuracion de acceso invalida")
	ErrAutorizacionDenegada          = errors.New("vec: autorizacion denegada")
	ErrDecisionAutorizacionInvalida  = errors.New("vec: decision de autorizacion invalida")
)
var (
	ErrReferenciaCorrelacionAutorizacionV2Invalida = errors.New(
		"vec: referencia de correlacion de autorizacion V2 invalida",
	)
	ErrGeneracionReferenciaCorrelacionAutorizacionV2 = errors.New(
		"vec: no se pudo generar la referencia de correlacion de autorizacion V2",
	)
	ErrSerializacionReferenciaCorrelacionAutorizacionV2Prohibida = errors.New(
		"vec: serializacion de referencia de correlacion de autorizacion V2 prohibida",
	)
)
var (
	ErrSolicitudAutorizacionLigadaV2Invalida = errors.New(
		"vec: solicitud de autorizacion ligada V2 invalida",
	)
	ErrSerializacionSolicitudAutorizacionLigadaV2Prohibida = errors.New(
		"vec: serializacion de solicitud de autorizacion ligada V2 prohibida",
	)
)
var (
	ErrCargaDocumentalInvalida        = errors.New("vec: carga documental invalida")
	ErrTransicionCargaNoPermitida     = errors.New("vec: transicion de carga documental no permitida")
	ErrContenidoCargaNoCorresponde    = errors.New("vec: contenido de carga documental no corresponde")
	ErrAnalisisCargaNoCorresponde     = errors.New("vec: analisis de carga documental no corresponde")
	ErrCargaDocumentalFueraDeVigencia = errors.New("vec: carga documental fuera de vigencia")
	ErrManifiestoPreparacionInvalido  = errors.New("vec: manifiesto de preparacion de carga directa invalido")
)
var (
	ErrCatalogoConfigurableInvalido = errors.New("vec: catalogo configurable invalido")
	ErrEntradaCatalogoInvalida      = errors.New("vec: entrada de catalogo invalida")
	ErrEntradaCatalogoDuplicada     = errors.New("vec: entrada de catalogo duplicada")
	ErrEntradaCatalogoNoVigente     = errors.New("vec: entrada de catalogo no vigente")
	ErrCatalogoNoPublicado          = errors.New("vec: catalogo no publicado")
	ErrTransicionCatalogoInvalida   = errors.New("vec: transicion de catalogo invalida")
)
var (
	ErrSolicitudContextoActorInvalida   = errors.New("vec: solicitud de contexto de actor invalida")
	ErrInstantaneaContextoActorInvalida = errors.New("vec: instantanea de contexto de actor invalida")
	ErrContextoActorInvalido            = errors.New("vec: contexto de actor invalido")
	ErrContextoActorNoResuelto          = errors.New("vec: contexto de actor no resuelto")
)
var (
	ErrPoliticaCotejoInvalida       = errors.New("vec: politica de cotejo invalida")
	ErrPoliticaCotejoNoPublicada    = errors.New("vec: politica de cotejo no publicada")
	ErrCodigoCotejoInvalido         = errors.New("vec: codigo de cotejo invalido")
	ErrTransicionCodigoCotejo       = errors.New("vec: transicion de codigo de cotejo invalida")
	ErrVersionCotejoInvalida        = errors.New("vec: version documental de cotejo invalida")
	ErrEvidenciaEmisionInvalida     = errors.New("vec: evidencia de emision invalida")
	ErrDocumentoNoAdmitidoPorCotejo = errors.New("vec: documento no admitido por la politica de cotejo")
)
var (
	ErrPlantillaDocumentoInvalida = errors.New("vec: plantilla documental invalida")
	ErrPlantillaNoPublicada       = errors.New("vec: plantilla documental no publicada")
	ErrCampoPlantillaInvalido     = errors.New("vec: campo de plantilla invalido")
	ErrCampoPlantillaFaltante     = errors.New("vec: falta un campo obligatorio de la plantilla")
	ErrCampoPlantillaDesconocido  = errors.New("vec: dato no declarado por la plantilla")
	ErrFormatoDocumentoInvalido   = errors.New("vec: formato documental invalido")
	ErrDocumentoInvalido          = errors.New("vec: documento invalido")
	ErrGarantiaInsuficiente       = errors.New("vec: nivel de garantia insuficiente")
	ErrContenidoFusionadoExcesivo = errors.New("vec: el contenido fusionado supera el limite seguro")
)
var (
	ErrReferenciaDocumentoInvalida          = errors.New("vec: referencia documental invalida")
	ErrRelacionDocumentoInvalida            = errors.New("vec: relacion documental invalida")
	ErrRelacionDocumentoDuplicada           = errors.New("vec: relacion documental duplicada")
	ErrRequisitoRelacionDocumentoInvalido   = errors.New("vec: requisito de relacion documental invalido")
	ErrRequisitoRelacionDocumentoIncumplido = errors.New("vec: requisito de relacion documental incumplido")
	ErrDocumentoLogicoInvalido              = errors.New("vec: documento logico invalido")
	ErrRepresentacionDocumentoInvalida      = errors.New("vec: representacion documental invalida")
	ErrSolicitudRepresentacionDuplicada     = errors.New("vec: solicitud de representacion documental duplicada")
)
var (
	ErrDefinicionFlujoInvalida    = errors.New("vec: definicion de flujo invalida")
	ErrEstadoFlujoInvalido        = errors.New("vec: estado de flujo invalido")
	ErrTransicionFlujoInvalida    = errors.New("vec: transicion de flujo invalida")
	ErrGrafoFlujoInvalido         = errors.New("vec: grafo de flujo invalido")
	ErrDefinicionFlujoNoPublicada = errors.New("vec: definicion de flujo no publicada")
	ErrInstanciaFlujoInvalida     = errors.New("vec: instancia de flujo invalida")
	ErrDecisionReglaInvalida      = errors.New("vec: decision de regla invalida")
	ErrReglaFlujoDenegada         = errors.New("vec: regla de flujo no satisfecha")
	ErrAprobacionFlujoRequerida   = errors.New("vec: la transicion requiere aprobacion")
	ErrAprobacionFlujoInvalida    = errors.New("vec: aprobacion de flujo invalida")
)
var (
	ErrIdentidadSintacticaDocumentalInvalida  = errors.New("vec: identidad sintactica documental invalida")
	ErrReferenciaPerfilDocumentalInvalida     = errors.New("vec: referencia de perfil documental invalida")
	ErrConformidadDocumentalInvalida          = errors.New("vec: conformidad documental invalida")
	ErrPerfilFormatoDocumentalInvalido        = errors.New("vec: perfil de formato documental invalido")
	ErrPublicacionPerfilDocumentalInvalida    = errors.New("vec: publicacion de perfil documental invalida")
	ErrRevisionCatalogoFormatosInvalida       = errors.New("vec: revision de catalogo de formatos invalida")
	ErrReferenciaComponenteDocumentalInvalida = errors.New("vec: referencia de componente documental invalida")
	ErrReferenciaConectorDocumentalInvalida   = errors.New("vec: referencia de conector documental invalida")
	ErrMarcaInstitucionalDocumentoInvalida    = errors.New("vec: metadato institucional documental invalido")
)
var (
	ErrDineroCobroInvalido                  = errors.New("vec: importe de cobro invalido")
	ErrReferenciaTarifaCobroInvalida        = errors.New("vec: referencia de tarifa de cobro invalida")
	ErrOrdenCobroInvalida                   = errors.New("vec: orden de cobro invalida")
	ErrTransicionCobroInvalida              = errors.New("vec: transicion de cobro invalida")
	ErrEvidenciaCobroInvalida               = errors.New("vec: evidencia de cobro invalida")
	ErrEvidenciaCobroConflictiva            = errors.New("vec: evidencia de cobro conflictiva")
	ErrCoincidenciaCobroInvalida            = errors.New("vec: el resultado no coincide con la orden de cobro")
	ErrDatoTarjetaProhibido                 = errors.New("vec: dato de tarjeta prohibido")
	ErrDevolucionCobroInvalida              = errors.New("vec: devolucion de cobro invalida")
	ErrConciliacionCobroInvalida            = errors.New("vec: conciliacion de cobro invalida")
	ErrSerializacionEvidenciaCobroProhibida = errors.New("vec: serializacion de evidencia interna de cobro prohibida")
	ErrSerializacionOrdenCobroProhibida     = errors.New("vec: serializacion directa de orden de cobro prohibida")
	ErrContextoAutorizacionCobroInvalido    = errors.New("vec: contexto de autorizacion de cobro invalido")
	ErrComandoCobroInvalido                 = errors.New("vec: comando de cobro invalido")
	ErrSerializacionAutorizacionCobro       = errors.New("vec: serializacion directa de autorizacion de cobro prohibida")
)
var (
	ErrPrincipalInvalid = errors.New("vec principal invalid")
	ErrModuleInvalid    = errors.New("vec module invalid")
	ErrMenuEntryInvalid = errors.New("vec menu entry invalid")
	ErrPermissionDenied = errors.New("vec permission denied")
)
var (
	ErrAutenticacionRevalidadaInvalida                  = errors.New("vec: autenticacion revalidada invalida")
	ErrVinculoAutenticacionActorInvalido                = errors.New("vec: vinculo de autenticacion y actor invalido")
	ErrReconstruccionVinculoAutenticacionActorProhibida = errors.New("vec: reconstruccion de vinculo de autenticacion y actor prohibida")
)
var ErrEstadoPersistibleFuenteAutoridadInvalido = errors.New("vec: estado persistible de fuente de autoridad invalido")
var ErrMensajeAtestacionAutorizacionInvalido = errors.New("vec: mensaje de atestacion de autorizacion invalido")
```

### Funciones

```go
func BytesCanonicosIdempotenciaAltaCobro(alta AltaOrdenCobro) ([]byte, error)
```

BytesCanonicosIdempotenciaAltaCobro fija el significado funcional de una
peticion antes de reservarla. Excluye identificadores y tiempos generados,
pero incluye todos los datos que cambiarian el cobro.

```go
func BytesCanonicosIdempotenciaDevolucionCobro(o OrdenCobro, solicitud SolicitudDevolucionOrdenCobro) ([]byte, error)
```

BytesCanonicosIdempotenciaDevolucionCobro fija todos los datos que pueden
cambiar el significado administrativo o economico de una devolucion.
Omite exclusivamente el instante, que procede del servidor.

```go
func CamposRequeridosAccionCobro(accion AccionCobro) ([]string, bool)
```

CamposRequeridosAccionCobro permite configurar el motor de autorizacion con
la misma lista cerrada que aplica el dominio. Devuelve siempre una copia.

```go
func ClaveMotivoAutorizacionV2Valida(clave string) bool
```

ClaveMotivoAutorizacionV2Valida comprueba exclusivamente el perfil opaco de
la clave. No acredita que exista en un catalogo ni que haya sido generada
con entropia suficiente; esas garantias pertenecen al servicio y al
repositorio.

```go
func CumpleGarantiaAutenticacion(actual, minima AuthAssurance) bool
```

CumpleGarantiaAutenticacion conserva el contrato funcional usado por los
adaptadores, pero pertenece al núcleo de identidad y no a un módulo de
autorización o documentos concreto.

```go
func HuellaCatalogoPoliticasAutorizacion(politicas []PoliticaRestrictiva) (string, error)
```

HuellaCatalogoPoliticasAutorizacion calcula una huella determinista del
manifiesto completo referencia+huella. El orden fisico de lectura no forma
parte del significado del catalogo, pero ningun identificador ni valor se
recorta, cambia de caja o corrige para aceptar configuracion no canonica.

```go
func HuellaEvidenciasCatalogoPoliticasAutorizacion(
	referencias []string,
	huellas map[string]string,
) (string, error)
```

HuellaEvidenciasCatalogoPoliticasAutorizacion permite verificar el mismo
manifiesto cuando el adaptador duradero ya ha materializado referencias y
huellas inmutables sin reconstruir los documentos de politica.

```go
func HuellaSHA256MensajeAtestacionAutorizacionV1(
	cabecera CabeceraAtestacionAutorizacionV1,
	decision DecisionAutorizacion,
) (string, error)
```

HuellaSHA256MensajeAtestacionAutorizacionV1 permite publicar vectores de
interoperabilidad sin convertir la huella en firma o autorizacion.

```go
func HuellaSHA256MensajeAtestacionAutorizacionV2(
	cabecera CabeceraAtestacionAutorizacionV2,
	decision DecisionAutorizacion,
	referenciaMotivo ReferenciaEntradaCatalogo,
) (string, error)
```

HuellaSHA256MensajeAtestacionAutorizacionV2 publica un vector de integridad
del mensaje canonico. La huella no constituye firma ni autorizacion.

```go
func HuellaSHA256MensajeAtestacionDenegacionAutorizacionV1(
	cabecera CabeceraAtestacionDenegacionAutorizacionV1,
	decision DecisionAutorizacion,
	referenciaMotivo ReferenciaEntradaCatalogo,
) (string, error)
```

HuellaSHA256MensajeAtestacionDenegacionAutorizacionV1 publica un vector de
integridad. La huella no acredita al PDP y no sustituye la firma del sobre.

```go
func HuellaSHA256MotivoAutorizacionV2(referencia ReferenciaEntradaCatalogo) (string, error)
```

HuellaSHA256MotivoAutorizacionV2 calcula el compromiso que puede cotejar
un adaptador durable. No recibe ni persiste texto libre: compromete la
referencia integra a una entrada catalogada ya resuelta por una frontera
confiable.

```go
func HuellaSHA256SolicitudAutorizacionV2(s SolicitudAutorizacionLigadaV2) (string, error)
func NormalizarValorCodigoCotejo(valor string) (string, error)
```

NormalizarValorCodigoCotejo elimina separadores de lectura. No admite otros
caracteres para evitar distintas representaciones del mismo secreto.

```go
func ReferenciaCorrelacionAutorizacionV2Valida(referencia string) bool
```

ReferenciaCorrelacionAutorizacionV2Valida exige el identificador opaco
de 128 bits reservado a solicitudes V2. La frontera debe generarlo con
crypto/rand; nunca se deriva de datos del usuario o del expediente.

```go
func ReferenciaMotivoAutorizacionV2Valida(referencia ReferenciaEntradaCatalogo) bool
```

ReferenciaMotivoAutorizacionV2Valida aplica el perfil especializado de
motivos de autorizacion V2. Ademas de la referencia de catalogo valida,
exige una version portable, una huella no nula y una clave opaca de 128
bits. La existencia y vigencia de la entrada requieren resolver el catalogo.

```go
func SerializarMensajeAtestacionAutorizacionV1(
	cabecera CabeceraAtestacionAutorizacionV1,
	decision DecisionAutorizacion,
) ([]byte, error)
```

SerializarMensajeAtestacionAutorizacionV1 produce la unica representacion
binaria VEC-AD-1 de una concesion reforzada. No ordena ni corrige las listas
recibidas: una lista que no llegue ya en orden UTF-8 estricto se rechaza.
Los mapas, cuyo orden de iteracion no forma parte del valor Go, se emiten
por clave UTF-8 ascendente.

```go
func SerializarMensajeAtestacionAutorizacionV2(
	cabecera CabeceraAtestacionAutorizacionV2,
	decision DecisionAutorizacion,
	referenciaMotivo ReferenciaEntradaCatalogo,
) ([]byte, error)
```

SerializarMensajeAtestacionAutorizacionV2 produce la unica representacion
binaria VEC-AD-2 de una concesion ligada a su solicitud y a una entrada
catalogada. La referencia del motivo se recibe completa para recomputar el
compromiso que contiene la decision; no se confia en una huella declarada.

VEC-AD-2 conserva el orden contractual VEC-AD-1 de DecisionAutorizacion e
inserta sus cuatro campos V2 en el lugar que ocupan tras CorrelacionRef. Las
cuatro coordenadas completas del motivo se escriben despues de la decision.
Las listas deben llegar ya ordenadas por bytes UTF-8; nunca se corrigen.

```go
func SerializarMensajeAtestacionDenegacionAutorizacionV1(
	cabecera CabeceraAtestacionDenegacionAutorizacionV1,
	decision DecisionAutorizacion,
	referenciaMotivo ReferenciaEntradaCatalogo,
) ([]byte, error)
```

SerializarMensajeAtestacionDenegacionAutorizacionV1 produce VEC-AD-D-1: la
prueba binaria de una decision negativa V2 y de la referencia catalogada que
el PDP resolvio al evaluarla. Nunca acepta una concesion ni emite VEC-AD-2.
GarantiaMinima puede estar vacia en denegaciones anteriores a seleccionar
una concesion; si esta presente debe pertenecer al vocabulario gobernado.
Las listas deben llegar ya ordenadas por bytes UTF-8, igual que en VEC-AD-2.

```go
func TuplaHechoCobroValida(tipo TipoHechoCobro, estado EstadoCobro, accion AccionCobro) bool
```

TuplaHechoCobroValida es la lista positiva de semanticas publicables. Evita
que un hecho o un evento aislado combinen un nombre valido con el estado o
la accion de otra operacion.

```go
func ValidarRequisitosRelacionesDocumento(relaciones []RelacionDocumento, requisitos []RequisitoRelacionDocumento) error
```

ValidarRequisitosRelacionesDocumento comprueba las cardinalidades declaradas
sin revelar referencias concretas en los errores.

### Tipos

```go
type AccionActoFuenteAutoridad string

const (
	AccionActoPublicarFuenteAutoridad           AccionActoFuenteAutoridad = "publicar"
	AccionActoSuspenderFuenteAutoridad          AccionActoFuenteAutoridad = "suspender"
	AccionActoLevantarSuspensionFuenteAutoridad AccionActoFuenteAutoridad = "levantar_suspension"
	AccionActoDerogarFuenteAutoridad            AccionActoFuenteAutoridad = "derogar"
)
func (a AccionActoFuenteAutoridad) Valida() bool

type AccionCobro string

const (
	AccionCobroCrearOrden          AccionCobro = "cobros.orden.crear"
	AccionCobroIniciarOperacion    AccionCobro = "cobros.operacion.iniciar"
	AccionCobroProcesarResultado   AccionCobro = "cobros.resultado.procesar"
	AccionCobroSolicitarDevolucion AccionCobro = "cobros.devolucion.solicitar"
	AccionCobroProcesarDevolucion  AccionCobro = "cobros.devolucion.procesar"
	AccionCobroConciliar           AccionCobro = "cobros.conciliar"
	AccionCobroCancelar            AccionCobro = "cobros.cancelar"
	AccionCobroCaducar             AccionCobro = "cobros.caducar"
)
func (a AccionCobro) Valida() bool

type AltaOrdenCobro struct {
	ID                     string
	IndiceIdempotenciaHMAC string
	ExpedienteRef          string
	SolicitudRef           string
	LiquidacionRef         string
	Tarifa                 ReferenciaTarifaCobro
	SujetoRef              string
	RepresentacionRef      string
	Importe                DineroCobro
	Concepto               string
	Finalidad              string
	CorrelacionRef         string
	CreadaEn               time.Time
	CaducaEn               time.Time
	EvidenciaCreacionRef   string
	HuellaEvidenciaSHA256  string
	Motivo                 string
}

type AmbitoFuenteAutoridad struct {
	DimensionClave string   `json:"dimension_clave"`
	ValoresClave   []string `json:"valores_clave"`
}
```

AmbitoFuenteAutoridad usa claves gobernadas. El nucleo no interpreta valores
como colectivos, territorios o centros ni compila sus catalogos.

```go
func (a AmbitoFuenteAutoridad) Validar() error

type AmbitoPerfil struct {
	Clave   string   `json:"clave"`
	Valores []string `json:"valores"`
}
```

AmbitoPerfil restringe una dimension del recurso. Varias dimensiones se
combinan con AND; los valores de una misma dimension se combinan con OR.
No existen comodines positivos: cada valor autorizable se enumera de
forma exacta para que una ampliacion futura del catalogo no amplie esta
asignacion.

```go
func (a AmbitoPerfil) Validar() error

type AnalisisCargaDocumental struct {
	ObjetoReferencia      string              `json:"objeto_referencia"`
	ObjetoVersion         string              `json:"objeto_version"`
	HuellaObjetoSHA256    string              `json:"huella_objeto_sha256"`
	ConectorAnalizadorID  string              `json:"conector_analizador_id"`
	VersionConector       int                 `json:"version_conector"`
	Estado                EstadoAnalisisCarga `json:"estado"`
	CodigoResultado       string              `json:"codigo_resultado"`
	EvidenciaRef          string              `json:"evidencia_ref"`
	HuellaEvidenciaSHA256 string              `json:"huella_evidencia_sha256"`
	CompletadoEn          time.Time           `json:"completado_en"`
}
```

AnalisisCargaDocumental fija el resultado normalizado del motor sobre una
version exacta. La salida cruda del antivirus no entra en el dominio.

```go
func (a AnalisisCargaDocumental) Validar() error

type AplicacionPoliticaCotejo struct {
	Referencia               ReferenciaPoliticaCotejo `json:"referencia"`
	ClaseAcceso              ClaseAccesoCotejo        `json:"clase_acceso"`
	CamposPublicos           []CampoPublicoCotejo     `json:"campos_publicos,omitempty"`
	PermiteDescargaDocumento bool                     `json:"permite_descarga_documento"`
	RequiereTitularidad      bool                     `json:"requiere_titularidad"`
	RolesTitularidad         []string                 `json:"roles_titularidad,omitempty"`
	RequiereFirma            bool                     `json:"requiere_firma"`
	RequiereSelloTiempo      bool                     `json:"requiere_sello_tiempo"`
	RequiereRegistro         bool                     `json:"requiere_registro"`
	GarantiaMinima           AuthAssurance            `json:"garantia_minima"`
	DiasPlazoActivacion      int                      `json:"dias_plazo_activacion"`
	DiasDisponibilidad       int                      `json:"dias_disponibilidad"`
}

func (a AplicacionPoliticaCotejo) Validar() error

type AsignacionPerfil struct {
	AsignacionID    string                 `json:"asignacion_id"`
	Version         int                    `json:"version"`
	PerfilActivoRef string                 `json:"perfil_activo_ref"`
	PrincipalID     string                 `json:"principal_id"`
	VersionRolRef   string                 `json:"version_rol_ref"`
	Estado          EstadoAsignacionPerfil `json:"estado"`
	Ambitos         []AmbitoPerfil         `json:"ambitos"`
	VigenteDesde    time.Time              `json:"vigente_desde"`
	VigenteHasta    time.Time              `json:"vigente_hasta"`
	EmitidaPor      string                 `json:"emitida_por"`
	EmitidaEn       time.Time              `json:"emitida_en"`
	RevocadaPor     string                 `json:"revocada_por,omitempty"`
	RevocadaEn      time.Time              `json:"revocada_en,omitempty"`
	RevocacionRef   string                 `json:"revocacion_ref,omitempty"`
}
```

AsignacionPerfil enlaza un perfil activo opaco con una unica version de rol.
Seleccionar un perfil impide sumar permisos de otros perfiles del mismo
principal.

```go
func (a AsignacionPerfil) Cubre(recurso RecursoAutorizable) bool

func (a AsignacionPerfil) HuellaSHA256() (string, error)

func (a AsignacionPerfil) Referencia() string

func (a AsignacionPerfil) Validar() error

func (a AsignacionPerfil) VigenteEn(instante time.Time) bool

type AtestacionAutenticacionCobro struct {
	// Has unexported fields.
}
```

AtestacionAutenticacionCobro es opaca y no serializable. Su valor cero
deniega. Conserva el resultado de una verificacion de sesion, no datos
recibidos directamente en la operacion de pago.

```go
func NuevaAtestacionAutenticacionCobro(
	ctx context.Context,
	verificador VerificadorAutenticacionCobro,
	sesionRef, huellaSesionHMAC string,
	instante time.Time,
) (AtestacionAutenticacionCobro, error)

func (a AtestacionAutenticacionCobro) Format(estado fmt.State, _ rune)

func (a AtestacionAutenticacionCobro) GoString() string

func (AtestacionAutenticacionCobro) MarshalJSON() ([]byte, error)

func (AtestacionAutenticacionCobro) MarshalText() ([]byte, error)

func (AtestacionAutenticacionCobro) String() string

func (*AtestacionAutenticacionCobro) UnmarshalJSON([]byte) error

type AuditEntry struct {
	ID                   string            `json:"id"`
	Seq                  int64             `json:"seq"`
	ActorID              string            `json:"actor_id"`
	ActorProfile         string            `json:"actor_profile,omitempty"`
	ActorRoles           []string          `json:"actor_roles,omitempty"`
	RepresentedSubjectID string            `json:"represented_subject_id,omitempty"`
	AuthMethod           AuthMethod        `json:"auth_method,omitempty"`
	AuthAssurance        AuthAssurance     `json:"auth_assurance,omitempty"`
	AuthorizationRef     string            `json:"authorization_ref,omitempty"`
	Purpose              string            `json:"purpose,omitempty"`
	Action               string            `json:"action"`
	ModuleID             string            `json:"module_id"`
	SubjectRef           string            `json:"subject_ref"`
	ObjectVersion        int               `json:"object_version,omitempty"`
	ExpedienteRef        string            `json:"expediente_ref,omitempty"`
	DocumentRef          string            `json:"document_ref,omitempty"`
	RuleRef              string            `json:"rule_ref,omitempty"`
	Reason               string            `json:"reason,omitempty"`
	Result               string            `json:"result"`
	BeforeHash           string            `json:"before_hash,omitempty"`
	AfterHash            string            `json:"after_hash,omitempty"`
	CorrelationRef       string            `json:"correlation_ref,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	OccurredAt           time.Time         `json:"occurred_at"`
	IntegrityAlgorithm   string            `json:"integrity_algorithm,omitempty"`
	PrevSignature        string            `json:"prev_signature,omitempty"`
	Signature            string            `json:"signature"`
}

type AutenticacionRevalidadaV1 struct {
	AutenticacionRef             string                         `json:"autenticacion_ref"`
	AutenticacionHuellaSHA256    string                         `json:"autenticacion_huella_sha256"`
	AsercionRef                  string                         `json:"asercion_ref"`
	SesionRef                    string                         `json:"sesion_ref"`
	ControlSesionRef             string                         `json:"control_sesion_ref"`
	ControlSesionRevision        uint64                         `json:"control_sesion_revision"`
	ControlSesionHuellaSHA256    string                         `json:"control_sesion_huella_sha256"`
	CuentaRef                    string                         `json:"cuenta_ref"`
	CuentaOrdinariaRef           string                         `json:"cuenta_ordinaria_ref"`
	CuentaPrivilegiada           bool                           `json:"cuenta_privilegiada"`
	Superficie                   SuperficieAutenticacionActorV1 `json:"superficie"`
	MetodoObservado              AuthMethod                     `json:"metodo_observado"`
	GarantiaObservada            AuthAssurance                  `json:"garantia_observada"`
	PoliticaGarantiaRef          string                         `json:"politica_garantia_ref"`
	PoliticaGarantiaHuellaSHA256 string                         `json:"politica_garantia_huella_sha256"`
	AutenticacionVerificadaEn    time.Time                      `json:"autenticacion_verificada_en"`
	SesionEmitidaEn              time.Time                      `json:"sesion_emitida_en"`
	SesionValidaHasta            time.Time                      `json:"sesion_valida_hasta"`
	SesionRevalidadaEn           time.Time                      `json:"sesion_revalidada_en"`
}
```

AutenticacionRevalidadaV1 es el resultado tipado de la autoridad de sesion.
No se construye a partir de un DTO HTTP: el puerto de revalidacion debe
resolver estos datos desde sus registros autoritativos y devolverlos todos.
Una omision no se completa ni se interpreta como un valor predeterminado.

```go
func (a AutenticacionRevalidadaV1) Validar() error

type AuthAssurance string

const (
	AuthAssuranceLow         AuthAssurance = "bajo"
	AuthAssuranceSubstantial AuthAssurance = "sustancial"
	AuthAssuranceHigh        AuthAssurance = "alto"
)
func GarantiaAutenticacionMasAlta(primera, segunda AuthAssurance) (AuthAssurance, error)

func (a AuthAssurance) Cumple(minima AuthAssurance) bool

func (a AuthAssurance) Valida() bool

type AuthChallenge struct {
	ID        string    `json:"id"`
	Nonce     string    `json:"nonce"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AuthChallengeResponse struct {
	ChallengeID string `json:"challenge_id"`
	Certificate string `json:"certificate"`
	Signature   string `json:"signature"`
}

type AuthMethod string

const (
	AuthMethodCertificate AuthMethod = "certificado"
	AuthMethodDNIe        AuthMethod = "dnie"
	AuthMethodSSO         AuthMethod = "sso"
	AuthMethodClave       AuthMethod = "clave"
	AuthMethodKerberos    AuthMethod = "kerberos_ad"
	AuthMethodDemo        AuthMethod = "demo"
)
func (a AuthMethod) Valido() bool

type CabeceraAtestacionAutorizacionV1 struct {
	FormatoVersion uint16
	Suite          string
	ClaveID        string
	Audiencia      string
}
```

CabeceraAtestacionAutorizacionV1 contiene toda la configuracion que debe
estar seleccionada de forma exacta antes de solicitar una firma. Suite y
ClaveID son identificadores; este tipo no implementa ni aprueba algoritmos,
proveedores o material criptografico.

```go
func (c CabeceraAtestacionAutorizacionV1) Validar() error

type CabeceraAtestacionAutorizacionV2 struct {
	FormatoVersion uint16
	Suite          string
	ClaveID        string
	Audiencia      string
}
```

CabeceraAtestacionAutorizacionV2 es una cabecera nominal distinta de V1.
Toda su configuracion debe seleccionarse antes de construir el mensaje. Este
corte solo fija los bytes canonicos: no implementa firma, COSE ni runtime.

```go
func (c CabeceraAtestacionAutorizacionV2) Validar() error

type CabeceraAtestacionDenegacionAutorizacionV1 struct {
	FormatoVersion uint16
	Suite          string
	ClaveID        string
	Audiencia      string
}
```

CabeceraAtestacionDenegacionAutorizacionV1 es nominalmente distinta de las
cabeceras de concesion. Suite, clave y audiencia se fijan en composicion;
el tipo solo define los bytes canonicos y no aprueba un proveedor de firma.

```go
func (c CabeceraAtestacionDenegacionAutorizacionV1) Validar() error

type CambioEstadoFlujo struct {
	InstanciaRef      string `json:"instancia_ref"`
	RevisionAnterior  int    `json:"revision_anterior"`
	RevisionPosterior int    `json:"revision_posterior"`
	EstadoAnterior    string `json:"estado_anterior"`
	EstadoPosterior   string `json:"estado_posterior"`
	TransicionClave   string `json:"transicion_clave"`
	DecisionReglaRef  string `json:"decision_regla_ref"`
	AutorizacionRef   string `json:"autorizacion_ref"`
	HuellaAnterior    string `json:"huella_anterior"`
	HuellaPosterior   string `json:"huella_posterior"`
}

type CampoPlantillaDocumento struct {
	Clave       string `json:"clave"`
	Etiqueta    string `json:"etiqueta"`
	Obligatorio bool   `json:"obligatorio"`
	Sensible    bool   `json:"sensible,omitempty"`
}
```

CampoPlantillaDocumento declara de forma cerrada un dato de fusion.
El servicio rechaza datos adicionales para evitar filtraciones accidentales.

```go
func (c CampoPlantillaDocumento) Validar() error

type CampoPublicoCotejo string
```

CampoPublicoCotejo es una lista cerrada por privacidad, no un catalogo de
negocio. Incorporar un campo nuevo requiere revisar de forma expresa que no
permita identificar a una persona o expediente.

```go
const (
	CampoPublicoCotejoOrgano         CampoPublicoCotejo = "organo"
	CampoPublicoCotejoTipoDocumental CampoPublicoCotejo = "tipo_documental"
	CampoPublicoCotejoFechaEmision   CampoPublicoCotejo = "fecha_emision"
	CampoPublicoCotejoHuellaSHA256   CampoPublicoCotejo = "huella_sha256"
)
func (c CampoPublicoCotejo) Valido() bool

type CapacidadPerfilFormatoDocumental string
```

CapacidadPerfilFormatoDocumental es vocabulario compilado del protocolo,
no un catalogo abierto de formatos. Una capacidad semantica nueva exige
desplegar codigo que la valide y cumpla; un registro no puede inventarla.

```go
const (
	CapacidadPerfilRenderizar             CapacidadPerfilFormatoDocumental = "renderizar"
	CapacidadPerfilMetadatoInstitucional  CapacidadPerfilFormatoDocumental = "metadato_institucional"
	CapacidadPerfilFirmaElectronica       CapacidadPerfilFormatoDocumental = "firma_electronica"
	CapacidadPerfilEdicion                CapacidadPerfilFormatoDocumental = "edicion"
	CapacidadPerfilPreservacionLargoPlazo CapacidadPerfilFormatoDocumental = "preservacion_largo_plazo"
)
func (c CapacidadPerfilFormatoDocumental) Valida() bool

type CapacidadesPerfilFormatoDocumental struct {
	// Has unexported fields.
}

func NuevasCapacidadesPerfilFormatoDocumental(
	capacidades ...CapacidadPerfilFormatoDocumental,
) (CapacidadesPerfilFormatoDocumental, error)

func (c CapacidadesPerfilFormatoDocumental) Tiene(capacidad CapacidadPerfilFormatoDocumental) bool

func (c CapacidadesPerfilFormatoDocumental) Validar() error

type CargaDocumental struct {
	ID                         string                    `json:"id"`
	Version                    int                       `json:"version"`
	PrincipalID                string                    `json:"principal_id"`
	RecursoRef                 string                    `json:"recurso_ref"`
	ModuloID                   string                    `json:"modulo_id"`
	TipoRecurso                string                    `json:"tipo_recurso"`
	OperacionRef               string                    `json:"operacion_ref"`
	CorrelacionRef             string                    `json:"correlacion_ref"`
	Finalidad                  string                    `json:"finalidad"`
	Clasificacion              string                    `json:"clasificacion"`
	MIMEDeclarado              string                    `json:"mime_declarado"`
	TamanoDeclarado            int64                     `json:"tamano_declarado"`
	HuellaDeclaradaSHA256      string                    `json:"huella_declarada_sha256"`
	IndiceIdempotenciaHMAC     string                    `json:"indice_idempotencia_hmac"`
	HuellaSolicitudHMAC        string                    `json:"huella_solicitud_hmac"`
	VinculoSesionHMAC          string                    `json:"vinculo_sesion_hmac,omitempty"`
	Estado                     EstadoCargaDocumental     `json:"estado"`
	AutorizacionPreparacionRef string                    `json:"autorizacion_preparacion_ref,omitempty"`
	AutorizacionRecepcionRef   string                    `json:"autorizacion_recepcion_ref,omitempty"`
	AutorizacionAnalisisRef    string                    `json:"autorizacion_analisis_ref,omitempty"`
	AutorizacionPromocionRef   string                    `json:"autorizacion_promocion_ref,omitempty"`
	ContenidoCuarentena        *ContenidoCargaDocumental `json:"contenido_cuarentena,omitempty"`
	Analisis                   *AnalisisCargaDocumental  `json:"analisis,omitempty"`
	ContenidoAdmitido          *ContenidoCargaDocumental `json:"contenido_admitido,omitempty"`
	CreadaEn                   time.Time                 `json:"creada_en"`
	PreparadaEn                time.Time                 `json:"preparada_en,omitempty"`
	ExpiraEn                   time.Time                 `json:"expira_en"`
	ActualizadaEn              time.Time                 `json:"actualizada_en"`
}
```

CargaDocumental es el agregado tecnico que impide que una sesion temporal,
un resultado del antivirus o una promocion se apliquen a otro expediente.
VinculoSesionHMAC permite comprobar la sesion sin persistirla en claro.

```go
func NuevaCargaDocumental(
	id, principalID, recursoRef, moduloID, tipoRecurso, operacionRef, correlacionRef,
	finalidad, clasificacion, mime string,
	tamano int64,
	huellaSHA256, indiceIdempotenciaHMAC, huellaSolicitudHMAC string,
	creadaEn, expiraEn time.Time,
) (CargaDocumental, error)

func (c CargaDocumental) Admitir(
	contenido ContenidoCargaDocumental,
	autorizacionRef string,
	instante time.Time,
) (CargaDocumental, error)

func (c CargaDocumental) HuellaSHA256() (string, error)

func (c CargaDocumental) Preparar(vinculoSesionHMAC, autorizacionRef string, preparadaEn time.Time) (CargaDocumental, error)

func (c CargaDocumental) RegistrarAnalisis(
	analisis AnalisisCargaDocumental,
	autorizacionRef string,
	instante time.Time,
) (CargaDocumental, error)

func (c CargaDocumental) RegistrarCuarentena(
	contenido ContenidoCargaDocumental,
	autorizacionRef string,
	instante time.Time,
) (CargaDocumental, error)

func (c CargaDocumental) Validar() error

func (c CargaDocumental) VigenteEn(instante time.Time) bool

type CatalogoConfigurable struct {
	ID                    string                        `json:"id"`
	Version               int                           `json:"version"`
	Revision              int                           `json:"revision"`
	VersionAnteriorRef    string                        `json:"version_anterior_ref,omitempty"`
	ModuloID              string                        `json:"modulo_id"`
	Nombre                string                        `json:"nombre"`
	Descripcion           string                        `json:"descripcion,omitempty"`
	FuenteRef             string                        `json:"fuente_ref"`
	MotivoCreacion        string                        `json:"motivo_creacion"`
	Entradas              []EntradaCatalogoConfigurable `json:"entradas"`
	Estado                EstadoCatalogoConfigurable    `json:"estado"`
	CreadoPor             string                        `json:"creado_por"`
	CreadoEn              time.Time                     `json:"creado_en"`
	UltimaModificacionPor string                        `json:"ultima_modificacion_por,omitempty"`
	UltimaModificacionEn  time.Time                     `json:"ultima_modificacion_en,omitempty"`
	MotivoModificacion    string                        `json:"motivo_modificacion,omitempty"`
	PublicadoPor          string                        `json:"publicado_por,omitempty"`
	PublicadoEn           time.Time                     `json:"publicado_en,omitempty"`
	AprobacionRef         string                        `json:"aprobacion_ref,omitempty"`
	MotivoPublicacion     string                        `json:"motivo_publicacion,omitempty"`
	RetiradoPor           string                        `json:"retirado_por,omitempty"`
	RetiradoEn            time.Time                     `json:"retirado_en,omitempty"`
	RetiradaAprobacionRef string                        `json:"retirada_aprobacion_ref,omitempty"`
	MotivoRetirada        string                        `json:"motivo_retirada,omitempty"`
}
```

CatalogoConfigurable es una instantanea completa e inmutable al publicarse.
Agregar «cosa cuatro» crea una nueva version desde la aplicacion; ningun
consumidor depende implicitamente de la ultima version.

```go
func (c CatalogoConfigurable) ActualizarBorrador(
	revisionEsperada int,
	actorID, nombre, descripcion, fuenteRef, motivo string,
	entradas []EntradaCatalogoConfigurable,
	instante time.Time,
) (CatalogoConfigurable, error)

func (c CatalogoConfigurable) ClonarCanonico() (CatalogoConfigurable, error)

func (c CatalogoConfigurable) HuellaContenidoSHA256() (string, error)
```

HuellaContenidoSHA256 identifica la semantica inmutable de la version.
No cambia al publicar o retirar, a diferencia de HuellaSHA256, que evidencia
la instantanea completa de gobierno.

```go
func (c CatalogoConfigurable) HuellaSHA256() (string, error)

func (c CatalogoConfigurable) NuevaVersion(version int, creadorID, fuenteRef, motivo string, instante time.Time) (CatalogoConfigurable, error)

func (c CatalogoConfigurable) ObtenerEntradaVigente(clave string, instante time.Time) (EntradaCatalogoConfigurable, error)

func (c CatalogoConfigurable) Publicar(actorID, aprobacionRef, motivo string, instante time.Time) (CatalogoConfigurable, error)

func (c CatalogoConfigurable) Referencia() string

func (c CatalogoConfigurable) Retirar(actorID, aprobacionRef, motivo string, instante time.Time) (CatalogoConfigurable, error)

func (c CatalogoConfigurable) Validar() error

type CitaFuenteAutoridad struct {
	Fuente    ReferenciaFuenteAutoridad `json:"fuente"`
	Preceptos []string                  `json:"preceptos"`
}
```

CitaFuenteAutoridad selecciona preceptos exactos de una version. Una lista
vacia no significa "toda la fuente".

```go
func (c CitaFuenteAutoridad) ClonarCanonica() (CitaFuenteAutoridad, error)

func (c CitaFuenteAutoridad) Validar() error

type ClaseAccesoCotejo string
```

ClaseAccesoCotejo es una frontera de seguridad estable. Las politicas
versionadas deciden que documentos usan cada clase y que campos se muestran.

```go
const (
	ClaseAccesoCotejoPublico   ClaseAccesoCotejo = "publico"
	ClaseAccesoCotejoProtegido ClaseAccesoCotejo = "protegido"
	ClaseAccesoCotejoInterno   ClaseAccesoCotejo = "interno"
)
func (c ClaseAccesoCotejo) Valida() bool

type CodigoCotejo struct {
	ID                  string                   `json:"id"`
	Revision            int                      `json:"revision"`
	Documento           ReferenciaDocumento      `json:"documento"`
	ModuloID            string                   `json:"modulo_id"`
	TipoDocumental      string                   `json:"tipo_documental"`
	Clasificacion       string                   `json:"clasificacion"`
	Organo              string                   `json:"organo"`
	ExpedienteRef       string                   `json:"-"`
	IndiceCodigoHMAC    string                   `json:"-"`
	ProteccionRef       string                   `json:"-"`
	VersionGenerador    string                   `json:"version_generador"`
	EntropiaBits        int                      `json:"entropia_bits"`
	Politica            AplicacionPoliticaCotejo `json:"politica"`
	Estado              EstadoCodigoCotejo       `json:"estado"`
	ReservadoPor        string                   `json:"reservado_por"`
	ReservadoEn         time.Time                `json:"reservado_en"`
	ReservaExpiraEn     time.Time                `json:"reserva_expira_en"`
	MotivoReserva       string                   `json:"motivo_reserva"`
	CorrelacionRef      string                   `json:"correlacion_ref"`
	VersionEmitida      *VersionEmitidaCotejo    `json:"version_emitida,omitempty"`
	ActivadoPor         string                   `json:"activado_por,omitempty"`
	ActivadoEn          time.Time                `json:"activado_en,omitempty"`
	ActivacionRef       string                   `json:"activacion_ref,omitempty"`
	EvidenciaEmisionRef string                   `json:"evidencia_emision_ref,omitempty"`
	MotivoActivacion    string                   `json:"motivo_activacion,omitempty"`
	DisponibleDesde     time.Time                `json:"disponible_desde,omitempty"`
	DisponibleHasta     time.Time                `json:"disponible_hasta,omitempty"`
	RetiradoPor         string                   `json:"retirado_por,omitempty"`
	RetiradoEn          time.Time                `json:"retirado_en,omitempty"`
	RetiradaRef         string                   `json:"retirada_ref,omitempty"`
	MotivoRetirada      string                   `json:"motivo_retirada,omitempty"`
	SustituidoPorRef    string                   `json:"sustituido_por_ref,omitempty"`
}
```

CodigoCotejo nunca expone ni serializa el valor del CSV. IndiceCodigoHMAC
permite buscarlo sin conservarlo en claro; ProteccionRef apunta a su
custodia cifrada para recuperar el mismo valor en reintentos internos
autorizados.

```go
func (c CodigoCotejo) Activar(actor, activacionRef, motivo string, evidencia EvidenciaEmisionDocumento, fecha time.Time) (CodigoCotejo, error)

func (c CodigoCotejo) ClonarCanonico() (CodigoCotejo, error)

func (c CodigoCotejo) DisponibleEn(instante time.Time) bool

func (c CodigoCotejo) HuellaEstadoSHA256() (string, error)

func (c CodigoCotejo) Referencia() string

func (c CodigoCotejo) Retirar(actor, retiradaRef, motivo string, fecha time.Time) (CodigoCotejo, error)

func (c CodigoCotejo) Sustituir(actor, retiradaRef, motivo, sustituidoPorRef string, fecha time.Time) (CodigoCotejo, error)

func (c CodigoCotejo) TieneCampoPublico(campo CampoPublicoCotejo) bool

func (c CodigoCotejo) Validar() error

type CodigoMotivoFuenteAutoridad string
```

CodigoMotivoFuenteAutoridad referencia un motivo gobernado por catalogo.
El detalle humano y su documentacion justificativa viven fuera del agregado
con clasificacion y conservacion propias.

```go
func (m CodigoMotivoFuenteAutoridad) Valido() bool

type ComandoConciliacionCobro struct {
	// Has unexported fields.
}

func (c ComandoConciliacionCobro) Datos() (DatosComandoConciliacionCobro, error)

func (c ComandoConciliacionCobro) Format(estado fmt.State, _ rune)

func (c ComandoConciliacionCobro) GoString() string

func (ComandoConciliacionCobro) MarshalJSON() ([]byte, error)

func (ComandoConciliacionCobro) MarshalText() ([]byte, error)

func (ComandoConciliacionCobro) String() string

func (*ComandoConciliacionCobro) UnmarshalJSON([]byte) error

func (c ComandoConciliacionCobro) Validar() error

type ComandoDevolucionCobro struct {
	// Has unexported fields.
}

func (c ComandoDevolucionCobro) Datos() (DatosComandoDevolucionCobro, error)

func (c ComandoDevolucionCobro) Format(estado fmt.State, _ rune)

func (c ComandoDevolucionCobro) GoString() string

func (ComandoDevolucionCobro) MarshalJSON() ([]byte, error)

func (ComandoDevolucionCobro) MarshalText() ([]byte, error)

func (ComandoDevolucionCobro) String() string

func (*ComandoDevolucionCobro) UnmarshalJSON([]byte) error

func (c ComandoDevolucionCobro) Validar() error

type ComandoInicioOperacionCobro struct {
	// Has unexported fields.
}

func (c ComandoInicioOperacionCobro) Datos() (DatosComandoInicioOperacionCobro, error)

func (c ComandoInicioOperacionCobro) Format(estado fmt.State, _ rune)

func (c ComandoInicioOperacionCobro) GoString() string

func (ComandoInicioOperacionCobro) MarshalJSON() ([]byte, error)

func (ComandoInicioOperacionCobro) MarshalText() ([]byte, error)

func (ComandoInicioOperacionCobro) String() string

func (*ComandoInicioOperacionCobro) UnmarshalJSON([]byte) error

func (c ComandoInicioOperacionCobro) Validar() error

type CompromisoTransicionFuenteAutoridadV1 struct {
	Esquema                    string                      `json:"esquema"`
	SolicitudRef               string                      `json:"solicitud_ref"`
	Fuente                     ReferenciaFuenteAutoridad   `json:"fuente"`
	RevisionPrevia             uint64                      `json:"revision_previa"`
	Secuencia                  uint64                      `json:"secuencia"`
	EstadoAnterior             EstadoFuenteAutoridad       `json:"estado_anterior"`
	EstadoNuevo                EstadoFuenteAutoridad       `json:"estado_nuevo"`
	Accion                     AccionActoFuenteAutoridad   `json:"accion"`
	ActorRef                   string                      `json:"actor_ref"`
	MotivoCodigo               CodigoMotivoFuenteAutoridad `json:"motivo_codigo"`
	HuellaHistoriaPreviaSHA256 string                      `json:"huella_historia_previa_sha256"`
	PreparadaEn                time.Time                   `json:"preparada_en"`
	ExpiraEn                   time.Time                   `json:"expira_en"`
}
```

CompromisoTransicionFuenteAutoridadV1 fija todos los datos que el
comprobador debe atestar. Cambiar cualquiera de ellos invalida la evidencia.

```go
func (c CompromisoTransicionFuenteAutoridadV1) BytesCanonicos() ([]byte, error)

func (c CompromisoTransicionFuenteAutoridadV1) HuellaSHA256() (string, error)

func (c CompromisoTransicionFuenteAutoridadV1) MarshalJSON() ([]byte, error)
```

MarshalJSON fuerza a todos los conectores a usar el mismo compromiso V1 que
se firma y cuya huella se conserva. No se serializa el tipo vivo.

```go
func (c CompromisoTransicionFuenteAutoridadV1) Validar() error

type ConcesionRol struct {
	Accion           string        `json:"accion"`
	ModuloID         string        `json:"modulo_id"`
	TipoRecurso      string        `json:"tipo_recurso"`
	Finalidades      []string      `json:"finalidades"`
	GarantiaMinima   AuthAssurance `json:"garantia_minima"`
	CamposPermitidos []string      `json:"campos_permitidos,omitempty"`
	Obligaciones     []string      `json:"obligaciones,omitempty"`
}
```

ConcesionRol es la unica parte que concede acceso (RBAC). Las politicas ABAC
pueden reducir esta concesion, pero nunca crearla ni ampliarla.

```go
func (c ConcesionRol) AdmiteFinalidad(finalidad string) bool

func (c ConcesionRol) Validar() error

type ConfiguracionBorradorFlujo struct {
	Nombre                          string
	Descripcion                     string
	FuenteRef                       string
	EstadoInicial                   string
	AccionInicio                    string
	GarantiaInicio                  AuthAssurance
	PermiteFinalizacionTrasRetirada bool
	Estados                         []EstadoFlujoConfigurable
	Transiciones                    []TransicionFlujoConfigurable
}

type ContenidoCargaDocumental struct {
	ConectorID   string             `json:"conector_id"`
	Referencia   string             `json:"referencia"`
	Version      string             `json:"version"`
	Zona         ZonaContenidoCarga `json:"zona"`
	MIME         string             `json:"mime"`
	Tamano       int64              `json:"tamano"`
	HuellaSHA256 string             `json:"huella_sha256"`
	EvidenciaRef string             `json:"evidencia_ref"`
	RegistradoEn time.Time          `json:"registrado_en"`
}
```

ContenidoCargaDocumental conserva referencias opacas y evidencia tecnica;
nunca rutas, nombres originales, URL temporales ni credenciales del almacen.

```go
func (c ContenidoCargaDocumental) Validar() error

type ContenidoDocumento struct {
	Titulo   string   `json:"titulo"`
	Parrafos []string `json:"parrafos"`
}
```

ContenidoDocumento es el modelo neutral que consumen los renderizadores.

```go
type ContenidoFuenteAutoridad struct {
	MateriaClave string                    `json:"materia_clave"`
	Nombre       string                    `json:"nombre"`
	Ambitos      []AmbitoFuenteAutoridad   `json:"ambitos"`
	Documento    DocumentoFuenteAutoridad  `json:"documento"`
	Preceptos    []PreceptoFuenteAutoridad `json:"preceptos"`
	Vigencia     PeriodoFuenteAutoridad    `json:"vigencia"`
	Efectos      PeriodoFuenteAutoridad    `json:"efectos"`
	ConocidaEn   time.Time                 `json:"conocida_en"`
}
```

ContenidoFuenteAutoridad agrupa la semantica inmutable al publicarse.
No contiene texto normativo, datos personales ni parametros de reglas.

```go
func (c ContenidoFuenteAutoridad) ClonarCanonico() (ContenidoFuenteAutoridad, error)

func (c ContenidoFuenteAutoridad) Validar() error

type ContextoActor struct {
	Principal       Principal                `json:"principal"`
	PerfilActivoRef string                   `json:"perfil_activo_ref"`
	PersonaRef      string                   `json:"persona_ref"`
	Instantanea     InstantaneaContextoActor `json:"instantanea"`
	ResueltoEn      time.Time                `json:"resuelto_en"`
}
```

ContextoActor es la proyeccion consumible por los casos de uso. Principal
usa la persona canonica como sujeto; cuenta y versiones permanecen en la
instantanea para auditoria y revalidacion posteriores.

```go
func NuevoContextoActor(
	cuenta CuentaAutenticadaContextoActor,
	instantanea InstantaneaContextoActor,
	resueltoEn time.Time,
) (ContextoActor, error)

func (c ContextoActor) Clonar() (ContextoActor, error)

func (c ContextoActor) HuellaSHA256VinculadaV1() (string, error)
```

HuellaSHA256VinculadaV1 compromete la identidad canonica, cuenta, perfil,
versiones, vigencias y referencias de modulo en un documento independiente.

```go
func (c ContextoActor) Referencias(tipo TipoReferenciaContextoActor) ([]string, error)
```

Referencias devuelve copias de las referencias opacas vigentes de un tipo.
Una clase desconocida nunca se interpreta como "todas".

```go
func (c ContextoActor) Validar() error

type ContextoAutorizacionCobro struct {
	// Has unexported fields.
}
```

ContextoAutorizacionCobro es una capacidad opaca: el valor cero y los
literales externos fallan cerrados. La huella interna liga la decision
completa, pero acredita integridad dentro del proceso, no la procedencia
criptografica de esa decision.

```go
func NuevoContextoAutorizacionCobro(
	decision DecisionAutorizacion,
	atestacion AtestacionAutenticacionCobro,
	recurso RecursoAutorizable,
	evaluadaEn time.Time,
) (ContextoAutorizacionCobro, error)
```

NuevoContextoAutorizacionCobro aplica el contrato positivo y exacto.
El futuro servicio de aplicacion debe invocarlo en la misma operacion en que
obtiene la decision del Autorizador, el recurso resuelto por el servidor
y la atestacion del servicio de identidad; nunca debe aceptar ninguno de
esos valores desde HTTP, CLI o mensajeria. Mientras ese cableado no exista,
este constructor no constituye por si solo una frontera de produccion
infabricable.

```go
func (c ContextoAutorizacionCobro) CoincideExactamenteConDecision(
	decision DecisionAutorizacion,
) bool
```

CoincideExactamenteConDecision permite a un puerto de persistencia cruzar
una evidencia reforzada con la decision inmutable que origino este contexto
sin exponer esa decision interna. No compara huellas de esquemas distintos:
vuelve a calcular dentro del dominio la misma representacion de cobros usada
al crear el contexto. Cualquier diferencia, incluso en controles de rol,
catalogo, sesion o actor, falla cerrada.

```go
func (c ContextoAutorizacionCobro) Datos() (DatosContextoAutorizacionCobro, error)

func (c ContextoAutorizacionCobro) Format(estado fmt.State, _ rune)

func (c ContextoAutorizacionCobro) GoString() string

func (ContextoAutorizacionCobro) MarshalJSON() ([]byte, error)

func (ContextoAutorizacionCobro) MarshalText() ([]byte, error)

func (ContextoAutorizacionCobro) String() string

func (*ContextoAutorizacionCobro) UnmarshalJSON([]byte) error

func (c ContextoAutorizacionCobro) ValidarEn(accion AccionCobro, recurso, finalidad, correlacion string, instante time.Time) error
```

ValidarEn comprueba el alcance y la vigencia en el instante efectivo de la
operacion. No existe una variante sin tiempo: reutilizar la hora de emision
permitiria aceptar capacidades caducadas desde una cache. El instante debe
proceder del reloj confiable del servidor, nunca de la peticion.

```go
type ContextoManifiestoPreparacionCargaDirectaV1 struct {
	CargaRef                string
	SujetoSeudonimoHMAC     string
	HuellaRecursoBaseSHA256 string
	HuellaRecursoSHA256     string
	ConectorAlmacenID       string
	EsquemaContexto         string
	AccionNegocio           string
	AccionTecnica           string
	PasoRef                 string
	EfectoRef               string
	HuellaPlanEfectoSHA256  string
	EsquemaHuellaDecision   string
	DecisionRef             string
	HuellaDecisionSHA256    string
	ContextoVerificadoEn    time.Time
	DecisionValidaHasta     time.Time
}
```

ContextoManifiestoPreparacionCargaDirectaV1 contiene la proyeccion no
autoritativa del plan que existia al preparar la carga. La fabrica la
cruza con el agregado preparado y fija una huella canonica; nunca permite
reconstruir una capacidad ni sustituye la decision original.

```go
type ControlEvidenciaInicioOperacionCobro struct {
	ConectorID               string
	VersionConector          int
	OrdenRef                 string
	LiquidacionRef           string
	OperacionProveedorRef    string
	Importe                  DineroCobro
	Concepto                 string
	VerificacionRef          string
	HuellaVerificacionSHA256 string
	RecibidaEn               time.Time
}
```

ControlEvidenciaInicioOperacionCobro expone al puerto remoto solo los datos
imprescindibles para impedir mezclar evidencia y origen de dos conectores.

```go
type ControlVigenciaVersionRol struct {
	VersionRolRef  string                          `json:"version_rol_ref"`
	Revision       uint64                          `json:"revision"`
	Estado         EstadoControlVigenciaVersionRol `json:"estado"`
	ActualizadoPor string                          `json:"actualizado_por"`
	ActualizadoEn  time.Time                       `json:"actualizado_en"`
	ActoRef        string                          `json:"acto_ref,omitempty"`
	MotivoCodigo   string                          `json:"motivo_codigo,omitempty"`
}
```

ControlVigenciaVersionRol separa la instantanea inmutable del rol de su
retirada global. Publicar una v2 retirada nunca se interpreta como retirada
de v1: cada control nombra exactamente la version afectada y tiene su propia
secuencia CAS.

```go
func (c ControlVigenciaVersionRol) HuellaSHA256() (string, error)

func (c ControlVigenciaVersionRol) Validar() error

type CuentaAutenticadaContextoActor struct {
	CuentaRef string        `json:"cuenta_ref"`
	Metodo    AuthMethod    `json:"metodo"`
	Garantia  AuthAssurance `json:"garantia"`
}
```

CuentaAutenticadaContextoActor contiene exclusivamente el identificador
tecnico opaco y la garantia ya acreditada por la frontera de identidad.
No admite DNI, correo, nombre, roles, permisos ni atributos declarados.

```go
func (c CuentaAutenticadaContextoActor) Validar() error

type DatosAltaFuenteAutoridadV1 struct {
	ID                   string
	Contenido            ContenidoFuenteAutoridad
	CreadaPor            string
	CreadaEn             time.Time
	MotivoCreacionCodigo CodigoMotivoFuenteAutoridad
}

type DatosComandoConciliacionCobro struct {
	OrdenRef                string
	VersionOrden            int
	HuellaOrdenSHA256       string
	ConectorID              string
	VersionConector         int
	OperacionProveedorRef   string
	DevolucionRef           string
	Tipo                    TipoConciliacionCobro
	Importe                 DineroCobro
	ReferenciaCierre        string
	DecisionAutorizacionRef string
	CorrelacionRef          string
}

type DatosComandoDevolucionCobro struct {
	OrdenRef                string
	VersionOrden            int
	HuellaOrdenSHA256       string
	ConectorID              string
	VersionConector         int
	OperacionProveedorRef   string
	DevolucionRef           string
	IndiceIdempotenciaHMAC  string
	Importe                 DineroCobro
	Motivo                  string
	DecisionAutorizacionRef string
	CorrelacionRef          string
}

type DatosComandoInicioOperacionCobro struct {
	OrdenRef                string
	VersionOrden            int
	HuellaOrdenSHA256       string
	LiquidacionRef          string
	IndiceIdempotenciaHMAC  string
	Importe                 DineroCobro
	Concepto                string
	CaducaEn                time.Time
	RetornoUsuarioRef       string
	NotificacionServidorRef string
	DecisionAutorizacionRef string
	CorrelacionRef          string
}

type DatosContextoAutorizacionCobro struct {
	DecisionRef          string
	ActorRef             string
	PerfilActivoRef      string
	Accion               AccionCobro
	RecursoRef           string
	Finalidad            string
	CorrelacionRef       string
	Garantia             AuthAssurance
	Metodo               AuthMethod
	AutenticacionRef     string
	SesionRef            string
	HuellaSesionHMAC     string
	CamposPermitidos     []string
	HuellaDecisionSHA256 string
	VigenteDesde         time.Time
	VigenteHasta         time.Time
	EvaluadaEn           time.Time
}
```

DatosContextoAutorizacionCobro es la proyeccion de solo lectura que permite
auditar una autorizacion ya validada. No sirve para construirla.

```go
type DatosDecisionHistoricaAtestacionAutorizacionV1 struct {
	DecisionRef                           string                           `json:"-"`
	Concedida                             bool                             `json:"-"`
	Codigo                                string                           `json:"-"`
	PrincipalID                           string                           `json:"-"`
	PerfilActivoRef                       string                           `json:"-"`
	Accion                                string                           `json:"-"`
	RecursoRef                            string                           `json:"-"`
	ModuloID                              string                           `json:"-"`
	TipoRecurso                           string                           `json:"-"`
	ContextoRecursoHuellaSHA256           string                           `json:"-"`
	Finalidad                             string                           `json:"-"`
	CorrelacionRef                        string                           `json:"-"`
	VinculoAutenticacionActor             DatosVinculoAutenticacionActorV1 `json:"-"`
	AsignacionRef                         string                           `json:"-"`
	AsignacionHuellaSHA256                string                           `json:"-"`
	VersionRolRef                         string                           `json:"-"`
	VersionRolHuellaSHA256                string                           `json:"-"`
	ControlVigenciaVersionRolRef          string                           `json:"-"`
	ControlVigenciaVersionRolRevision     uint64                           `json:"-"`
	ControlVigenciaVersionRolHuellaSHA256 string                           `json:"-"`
	RevisionCatalogoPoliticas             uint64                           `json:"-"`
	CatalogoPoliticasHuellaSHA256         string                           `json:"-"`
	PoliticasEvaluadasRefs                []string                         `json:"-"`
	PoliticasEvaluadasHuellasSHA256       map[string]string                `json:"-"`
	PoliticasRefs                         []string                         `json:"-"`
	PoliticasHuellasSHA256                map[string]string                `json:"-"`
	GarantiaMinima                        AuthAssurance                    `json:"-"`
	CamposPermitidos                      []string                         `json:"-"`
	Obligaciones                          []string                         `json:"-"`
	EmitidaEn                             time.Time                        `json:"-"`
	ValidaHasta                           time.Time                        `json:"-"`
}
```

DatosDecisionHistoricaAtestacionAutorizacionV1 enumera los treinta datos de
DecisionAutorizacion distintos del vinculo y conserva el bloque de vinculo
como DatosVinculoAutenticacionActorV1. Deliberadamente no contiene ni
puede reconstruir VinculoAutenticacionActorV1, una DecisionAutorizacion o
cualquier otra capacidad opaca.

Sus campos son una proyeccion defensiva para comparaciones exactas tras
haber verificado la firma por otra capa. No son una autorizacion y no deben
usarse como entrada de un PDP, un constructor de capacidades o una mutacion
durable.

```go
func (d DatosDecisionHistoricaAtestacionAutorizacionV1) Format(estado fmt.State, _ rune)

func (d DatosDecisionHistoricaAtestacionAutorizacionV1) GoString() string

func (*DatosDecisionHistoricaAtestacionAutorizacionV1) GobDecode([]byte) error

func (DatosDecisionHistoricaAtestacionAutorizacionV1) GobEncode() ([]byte, error)

func (d DatosDecisionHistoricaAtestacionAutorizacionV1) LogValue() slog.Value

func (DatosDecisionHistoricaAtestacionAutorizacionV1) MarshalBinary() ([]byte, error)

func (DatosDecisionHistoricaAtestacionAutorizacionV1) MarshalCBOR() ([]byte, error)

func (DatosDecisionHistoricaAtestacionAutorizacionV1) MarshalJSON() ([]byte, error)

func (DatosDecisionHistoricaAtestacionAutorizacionV1) MarshalText() ([]byte, error)

func (DatosDecisionHistoricaAtestacionAutorizacionV1) MarshalXML(*xml.Encoder, xml.StartElement) error

func (DatosDecisionHistoricaAtestacionAutorizacionV1) MarshalYAML() (any, error)

func (DatosDecisionHistoricaAtestacionAutorizacionV1) String() string

func (*DatosDecisionHistoricaAtestacionAutorizacionV1) UnmarshalBinary([]byte) error

func (*DatosDecisionHistoricaAtestacionAutorizacionV1) UnmarshalJSON([]byte) error

func (*DatosDecisionHistoricaAtestacionAutorizacionV1) UnmarshalText([]byte) error

type DatosEvidenciaServidorCobro struct {
	EvidenciaRef             string
	HuellaSHA256             string
	ConectorID               string
	VersionConector          int
	OrdenRef                 string
	LiquidacionRef           string
	OperacionProveedorRef    string
	Importe                  DineroCobro
	Concepto                 string
	Codigo                   string
	MetodoAutenticacion      MetodoAutenticacionEvidenciaCobro
	Audiencia                string
	VerificacionRef          string
	HuellaVerificacionSHA256 string
	EmitidaEn                time.Time
	RecibidaEn               time.Time
	VerificadaEn             time.Time
}

type DatosManifiestoPreparacionCargaDirectaV1 struct {
	Esquema                 string    `json:"esquema"`
	CargaID                 string    `json:"carga_id"`
	VersionCarga            int       `json:"version_carga"`
	HuellaCargaSHA256       string    `json:"huella_carga_sha256"`
	IndiceIdempotenciaHMAC  string    `json:"indice_idempotencia_hmac"`
	OperacionRef            string    `json:"operacion_ref"`
	CorrelacionRef          string    `json:"correlacion_ref"`
	CargaRef                string    `json:"carga_ref"`
	Finalidad               string    `json:"finalidad"`
	Clasificacion           string    `json:"clasificacion"`
	SujetoSeudonimoHMAC     string    `json:"sujeto_seudonimo_hmac"`
	HuellaSolicitudHMAC     string    `json:"huella_solicitud_hmac"`
	RecursoRef              string    `json:"recurso_ref"`
	ModuloID                string    `json:"modulo_id"`
	TipoRecurso             string    `json:"tipo_recurso"`
	HuellaRecursoBaseSHA256 string    `json:"huella_recurso_base_sha256"`
	HuellaRecursoSHA256     string    `json:"huella_recurso_sha256"`
	MIME                    string    `json:"mime"`
	Tamano                  int64     `json:"tamano"`
	HuellaContenidoSHA256   string    `json:"huella_contenido_sha256"`
	VinculoSesionHMAC       string    `json:"vinculo_sesion_hmac"`
	ConectorAlmacenID       string    `json:"conector_almacen_id"`
	EsquemaContexto         string    `json:"esquema_contexto"`
	AccionNegocio           string    `json:"accion_negocio"`
	AccionTecnica           string    `json:"accion_tecnica"`
	PasoRef                 string    `json:"paso_ref"`
	EfectoRef               string    `json:"efecto_ref"`
	HuellaPlanEfectoSHA256  string    `json:"huella_plan_efecto_sha256"`
	EsquemaHuellaDecision   string    `json:"esquema_huella_decision"`
	DecisionRef             string    `json:"decision_ref"`
	HuellaDecisionSHA256    string    `json:"huella_decision_sha256"`
	ContextoVerificadoEn    time.Time `json:"contexto_verificado_en"`
	DecisionValidaHasta     time.Time `json:"decision_valida_hasta"`
	PreparadaEn             time.Time `json:"preparada_en"`
	ExpiraEn                time.Time `json:"expira_en"`
	HuellaManifiestoSHA256  string    `json:"huella_manifiesto_sha256"`
}
```

DatosManifiestoPreparacionCargaDirectaV1 es una copia destinada a un
adaptador durable. Todos sus campos son escalares y la huella se recalcula
al leer; una mutacion o decodificacion parcial falla cerrada.

```go
type DatosMensajeAtestacionActoFuenteAutoridadV1 struct {
	EvidenciaRef          string
	ActoRef               string
	DocumentoRef          string
	RepresentacionRef     string
	HuellaDocumentoSHA256 string
	OrganoRef             string
	FirmasRefs            []string
	ComprobadorRef        string
	ActoOcurridoEn        time.Time
	ComprobadaEn          time.Time
}
```

DatosMensajeAtestacionActoFuenteAutoridadV1 son hechos producidos por el
comprobador. No incluyen campos derivados ni el sobre que todavía debe
firmar el mensaje.

```go
type DatosPreparacionTransicionFuenteAutoridadV1 struct {
	EstadoNuevo  EstadoFuenteAutoridad
	ActorRef     string
	MotivoCodigo CodigoMotivoFuenteAutoridad
	SolicitudRef string
	PreparadaEn  time.Time
	ExpiraEn     time.Time
}

type DatosSobreAtestacionActoFuenteAutoridadV1 struct {
	AtestacionRef          string
	HuellaAtestacionSHA256 string
	FirmaAtestacionRef     string
}

type DatosSolicitudAutorizacionLigadaV2 struct {
	ContextoActor             ContextoActor
	VinculoAutenticacionActor VinculoAutenticacionActorV1
	ReferenciaMotivo          ReferenciaEntradaCatalogo
	Accion                    string
	Recurso                   RecursoAutorizable
	Finalidad                 string
	Correlacion               ReferenciaCorrelacionAutorizacionV2
	// Has unexported fields.
}
```

DatosSolicitudAutorizacionLigadaV2 declara exclusivamente la solicitud
efectiva V2. Identidad, perfil, metodo y garantia se derivan del vinculo;
no admite Principal declarado ni un campo de texto Motivo.

```go
func (b DatosSolicitudAutorizacionLigadaV2) Format(estado fmt.State, _ rune)

func (b DatosSolicitudAutorizacionLigadaV2) GoString() string

func (*DatosSolicitudAutorizacionLigadaV2) GobDecode([]byte) error

func (DatosSolicitudAutorizacionLigadaV2) GobEncode() ([]byte, error)

func (b DatosSolicitudAutorizacionLigadaV2) LogValue() slog.Value

func (DatosSolicitudAutorizacionLigadaV2) MarshalBinary() ([]byte, error)

func (DatosSolicitudAutorizacionLigadaV2) MarshalCBOR() ([]byte, error)

func (DatosSolicitudAutorizacionLigadaV2) MarshalJSON() ([]byte, error)

func (DatosSolicitudAutorizacionLigadaV2) MarshalText() ([]byte, error)

func (DatosSolicitudAutorizacionLigadaV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (DatosSolicitudAutorizacionLigadaV2) MarshalYAML() (any, error)

func (DatosSolicitudAutorizacionLigadaV2) String() string

func (*DatosSolicitudAutorizacionLigadaV2) UnmarshalBinary([]byte) error

func (*DatosSolicitudAutorizacionLigadaV2) UnmarshalCBOR([]byte) error

func (*DatosSolicitudAutorizacionLigadaV2) UnmarshalJSON([]byte) error

func (*DatosSolicitudAutorizacionLigadaV2) UnmarshalText([]byte) error

func (*DatosSolicitudAutorizacionLigadaV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (*DatosSolicitudAutorizacionLigadaV2) UnmarshalYAML(func(any) error) error

type DatosVinculoAutenticacionActorV1 struct {
	BloqueVersion                uint16                         `json:"bloque_version"`
	AutenticacionRef             string                         `json:"autenticacion_ref"`
	AutenticacionHuellaSHA256    string                         `json:"autenticacion_huella_sha256"`
	AsercionRef                  string                         `json:"asercion_ref"`
	SesionRef                    string                         `json:"sesion_ref"`
	ControlSesionRef             string                         `json:"control_sesion_ref"`
	ControlSesionRevision        uint64                         `json:"control_sesion_revision"`
	ControlSesionHuellaSHA256    string                         `json:"control_sesion_huella_sha256"`
	CuentaRef                    string                         `json:"cuenta_ref"`
	CuentaOrdinariaRef           string                         `json:"cuenta_ordinaria_ref"`
	PrincipalID                  string                         `json:"principal_id"`
	PerfilActivoRef              string                         `json:"perfil_activo_ref"`
	CuentaPrivilegiada           bool                           `json:"cuenta_privilegiada"`
	Superficie                   SuperficieAutenticacionActorV1 `json:"superficie"`
	MetodoObservado              AuthMethod                     `json:"metodo_observado"`
	GarantiaObservada            AuthAssurance                  `json:"garantia_observada"`
	PoliticaGarantiaRef          string                         `json:"politica_garantia_ref"`
	PoliticaGarantiaHuellaSHA256 string                         `json:"politica_garantia_huella_sha256"`
	AutenticacionVerificadaEn    time.Time                      `json:"autenticacion_verificada_en"`
	SesionEmitidaEn              time.Time                      `json:"sesion_emitida_en"`
	SesionValidaHasta            time.Time                      `json:"sesion_valida_hasta"`
	SesionRevalidadaEn           time.Time                      `json:"sesion_revalidada_en"`
	ContextoActorRef             string                         `json:"contexto_actor_ref"`
	ContextoActorVersion         uint64                         `json:"contexto_actor_version"`
	ContextoActorHuellaSHA256    string                         `json:"contexto_actor_huella_sha256"`
}
```

DatosVinculoAutenticacionActorV1 contiene exactamente los 25 datos que
quedan ligados a una decision. Es una proyeccion defensiva para firmar y
persistir evidencia, no una capacidad reconstruible.

```go
func (v DatosVinculoAutenticacionActorV1) Autenticacion() AutenticacionRevalidadaV1

func (v DatosVinculoAutenticacionActorV1) Validar() error

type DecisionAutorizacion struct {
	DecisionRef                           string                      `json:"decision_ref"`
	Concedida                             bool                        `json:"concedida"`
	Codigo                                string                      `json:"codigo"`
	PrincipalID                           string                      `json:"principal_id"`
	PerfilActivoRef                       string                      `json:"perfil_activo_ref"`
	Accion                                string                      `json:"accion"`
	RecursoRef                            string                      `json:"recurso_ref"`
	ModuloID                              string                      `json:"modulo_id,omitempty"`
	TipoRecurso                           string                      `json:"tipo_recurso,omitempty"`
	ContextoRecursoHuellaSHA256           string                      `json:"contexto_recurso_huella_sha256,omitempty"`
	Finalidad                             string                      `json:"finalidad"`
	CorrelacionRef                        string                      `json:"correlacion_ref"`
	EsquemaHuellaSolicitud                string                      `json:"esquema_huella_solicitud,omitempty"`
	SolicitudHuellaSHA256                 string                      `json:"solicitud_huella_sha256,omitempty"`
	EsquemaHuellaMotivo                   string                      `json:"esquema_huella_motivo,omitempty"`
	MotivoHuellaSHA256                    string                      `json:"motivo_huella_sha256,omitempty"`
	VinculoAutenticacionActor             VinculoAutenticacionActorV1 `json:"vinculo_autenticacion_actor"`
	AsignacionRef                         string                      `json:"asignacion_ref,omitempty"`
	AsignacionHuellaSHA256                string                      `json:"asignacion_huella_sha256,omitempty"`
	VersionRolRef                         string                      `json:"version_rol_ref,omitempty"`
	VersionRolHuellaSHA256                string                      `json:"version_rol_huella_sha256,omitempty"`
	ControlVigenciaVersionRolRef          string                      `json:"control_vigencia_version_rol_ref,omitempty"`
	ControlVigenciaVersionRolRevision     uint64                      `json:"control_vigencia_version_rol_revision,omitempty"`
	ControlVigenciaVersionRolHuellaSHA256 string                      `json:"control_vigencia_version_rol_huella_sha256,omitempty"`
	RevisionCatalogoPoliticas             uint64                      `json:"revision_catalogo_politicas,omitempty"`
	CatalogoPoliticasHuellaSHA256         string                      `json:"catalogo_politicas_huella_sha256,omitempty"`
	PoliticasEvaluadasRefs                []string                    `json:"politicas_evaluadas_refs,omitempty"`
	PoliticasEvaluadasHuellasSHA256       map[string]string           `json:"politicas_evaluadas_huellas_sha256,omitempty"`
	// PoliticasRefs contiene solo el subconjunto aplicable. La evidencia del
	// catalogo completo se conserva por separado en PoliticasEvaluadasRefs.
	PoliticasRefs          []string          `json:"politicas_refs,omitempty"`
	PoliticasHuellasSHA256 map[string]string `json:"politicas_huellas_sha256,omitempty"`
	GarantiaMinima         AuthAssurance     `json:"garantia_minima,omitempty"`
	CamposPermitidos       []string          `json:"campos_permitidos,omitempty"`
	Obligaciones           []string          `json:"obligaciones,omitempty"`
	EmitidaEn              time.Time         `json:"emitida_en"`
	ValidaHasta            time.Time         `json:"valida_hasta"`
}
```

DecisionAutorizacion es una evidencia breve, no un permiso permanente.
Las referencias y huellas fijan exactamente la configuracion evaluada.

```go
func (d DecisionAutorizacion) TieneSolicitudLigadaV2() bool
```

TieneSolicitudLigadaV2 informa solo de la validez estructural de los dos
compromisos. La procedencia sigue dependiendo del PDP y del registro.

```go
func (d DecisionAutorizacion) Validar() error

func (d DecisionAutorizacion) ValidarEvidenciaInstantanea() error
```

ValidarEvidenciaInstantanea exige el formato reforzado que deben registrar
los adaptadores de autorizacion. Validar conserva temporalmente la lectura
de evidencias historicas anteriores, pero el registro CAS nunca las admite.

```go
func (d DecisionAutorizacion) ValidarEvidenciaInstantaneaSolicitudLigadaV2() error
```

ValidarEvidenciaInstantaneaSolicitudLigadaV2 es el contrato para decisiones
nuevas y efectos durables. ValidarEvidenciaInstantanea conserva lectura
historica, pero nunca basta para crear una capacidad ejecutable V2.

```go
func (d DecisionAutorizacion) VigenteEn(instante time.Time) bool

func (d DecisionAutorizacion) VigenteParaEfectoEn(instante time.Time) bool
```

VigenteParaEfectoEn excluye expresamente decisiones historicas sin el
compromiso V2 de solicitud y motivo.

```go
type DecisionReglaFlujo struct {
	DecisionRef                     string    `json:"decision_ref"`
	Concedida                       bool      `json:"concedida"`
	Codigo                          string    `json:"codigo"`
	DefinicionRef                   string    `json:"definicion_ref"`
	DefinicionContenidoHuellaSHA256 string    `json:"definicion_contenido_huella_sha256"`
	InstanciaRef                    string    `json:"instancia_ref"`
	InstanciaRevision               int       `json:"instancia_revision"`
	EstadoOrigen                    string    `json:"estado_origen"`
	TransicionClave                 string    `json:"transicion_clave"`
	ReglaRef                        string    `json:"regla_ref"`
	ActorID                         string    `json:"actor_id"`
	Finalidad                       string    `json:"finalidad"`
	CorrelacionRef                  string    `json:"correlacion_ref"`
	EntradaHuellaHMAC               string    `json:"entrada_huella_hmac"`
	ResultadoHuellaSHA256           string    `json:"resultado_huella_sha256"`
	EvaluadaEn                      time.Time `json:"evaluada_en"`
	ValidaHasta                     time.Time `json:"valida_hasta"`
}

func (d DecisionReglaFlujo) HuellaSHA256() (string, error)

func (d DecisionReglaFlujo) Validar() error

func (d DecisionReglaFlujo) VigenteEn(instante time.Time) bool

type DefinicionFlujo struct {
	ID                              string                        `json:"id"`
	Version                         int                           `json:"version"`
	Revision                        int                           `json:"revision"`
	VersionAnteriorRef              string                        `json:"version_anterior_ref,omitempty"`
	ModuloID                        string                        `json:"modulo_id"`
	TipoEntidad                     string                        `json:"tipo_entidad"`
	Nombre                          string                        `json:"nombre"`
	Descripcion                     string                        `json:"descripcion,omitempty"`
	FuenteRef                       string                        `json:"fuente_ref"`
	MotivoCreacion                  string                        `json:"motivo_creacion"`
	EstadoInicial                   string                        `json:"estado_inicial,omitempty"`
	AccionInicio                    string                        `json:"accion_inicio"`
	GarantiaInicio                  AuthAssurance                 `json:"garantia_inicio"`
	PermiteFinalizacionTrasRetirada bool                          `json:"permite_finalizacion_tras_retirada"`
	Estados                         []EstadoFlujoConfigurable     `json:"estados"`
	Transiciones                    []TransicionFlujoConfigurable `json:"transiciones"`
	Estado                          EstadoDefinicionFlujo         `json:"estado"`
	CreadaPor                       string                        `json:"creada_por"`
	CreadaEn                        time.Time                     `json:"creada_en"`
	UltimaModificacionPor           string                        `json:"ultima_modificacion_por,omitempty"`
	UltimaModificacionEn            time.Time                     `json:"ultima_modificacion_en,omitempty"`
	MotivoModificacion              string                        `json:"motivo_modificacion,omitempty"`
	PublicadaPor                    string                        `json:"publicada_por,omitempty"`
	PublicadaEn                     time.Time                     `json:"publicada_en,omitempty"`
	AprobacionRef                   string                        `json:"aprobacion_ref,omitempty"`
	MotivoPublicacion               string                        `json:"motivo_publicacion,omitempty"`
	RetiradaPor                     string                        `json:"retirada_por,omitempty"`
	RetiradaEn                      time.Time                     `json:"retirada_en,omitempty"`
	RetiradaAprobacionRef           string                        `json:"retirada_aprobacion_ref,omitempty"`
	MotivoRetirada                  string                        `json:"motivo_retirada,omitempty"`
}

func (d DefinicionFlujo) ActualizarBorrador(
	revisionEsperada int,
	actorID, motivo string,
	configuracion ConfiguracionBorradorFlujo,
	instante time.Time,
) (DefinicionFlujo, error)

func (d DefinicionFlujo) ClonarCanonico() (DefinicionFlujo, error)

func (d DefinicionFlujo) HuellaContenidoSHA256() (string, error)
```

HuellaContenidoSHA256 identifica la semantica inmutable de esta version
del flujo. No cambia al publicarla o retirarla; HuellaSHA256 conserva,
en cambio, la evidencia de la instantanea completa de gobierno.

```go
func (d DefinicionFlujo) HuellaSHA256() (string, error)

func (d DefinicionFlujo) NuevaVersion(version int, creadorID, fuenteRef, motivo string, instante time.Time) (DefinicionFlujo, error)

func (d DefinicionFlujo) ObtenerTransicion(clave, estadoActual string) (TransicionFlujoConfigurable, error)

func (d DefinicionFlujo) Publicar(actorID, aprobacionRef, motivo string, instante time.Time) (DefinicionFlujo, error)

func (d DefinicionFlujo) Referencia() string

func (d DefinicionFlujo) Retirar(actorID, aprobacionRef, motivo string, instante time.Time) (DefinicionFlujo, error)

func (d DefinicionFlujo) Validar() error

type DineroCobro struct {
	UnidadesMenores int64  `json:"unidades_menores"`
	Moneda          string `json:"moneda"`
}
```

DineroCobro representa dinero exclusivamente en la unidad menor de la
moneda. No admite cero porque las exenciones y no sujeciones son decisiones
administrativas, no cobros ficticios.

```go
func (d DineroCobro) Igual(otro DineroCobro) bool

func (d DineroCobro) Validar() error

type DocumentoFuenteAutoridad struct {
	DocumentoID           string `json:"documento_id"`
	DocumentoVersion      uint64 `json:"documento_version"`
	RepresentacionRef     string `json:"representacion_ref"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
	PublicacionOficialRef string `json:"publicacion_oficial_ref"`
	ActoOrigenRef         string `json:"acto_origen_ref"`
	OrganoEmisorRef       string `json:"organo_emisor_ref"`
}
```

DocumentoFuenteAutoridad fija la representacion concreta examinada. El
contenido y las firmas siguen custodiados por las capacidades documentales;
este agregado solo conserva referencias opacas y huellas.

```go
func (d DocumentoFuenteAutoridad) Validar() error

type DocumentoGenerado struct {
	ID                  string                   `json:"id"`
	Version             int                      `json:"version"`
	PlantillaID         string                   `json:"plantilla_id"`
	PlantillaVersion    int                      `json:"plantilla_version"`
	ModuloID            string                   `json:"modulo_id"`
	TipoDocumental      string                   `json:"tipo_documental"`
	ExpedienteRef       string                   `json:"expediente_ref"`
	Formato             FormatoDocumento         `json:"formato"`
	MIME                string                   `json:"mime"`
	NombreFichero       string                   `json:"nombre_fichero"`
	Tamano              int64                    `json:"tamano"`
	HuellaSHA256        string                   `json:"huella_sha256"`
	HuellaDatosHMAC     string                   `json:"huella_datos_hmac"`
	ReferenciaContenido string                   `json:"referencia_contenido"`
	Estado              EstadoDocumento          `json:"estado"`
	EstadoAntivirus     EstadoAntivirusDocumento `json:"estado_antivirus"`
	GeneradoPor         string                   `json:"generado_por"`
	GeneradoEn          time.Time                `json:"generado_en"`
	CorrelacionRef      string                   `json:"correlacion_ref"`
	Motivo              string                   `json:"motivo"`
	ENI                 MetadatosENI             `json:"eni"`
	FirmaRefs           []string                 `json:"firma_refs,omitempty"`
	RegistroRef         string                   `json:"registro_ref,omitempty"`
	CSV                 string                   `json:"csv,omitempty"`
}
```

DocumentoGenerado es la identidad permanente del artefacto. La referencia de
contenido es opaca: una URL temporal nunca identifica un documento.

```go
func (d DocumentoGenerado) Validar() error

type DocumentoLogico struct {
	ID               string                       `json:"id"`
	Version          int                          `json:"version"`
	Revision         int                          `json:"revision"`
	VersionAnterior  *ReferenciaDocumento         `json:"version_anterior,omitempty"`
	Plantilla        ReferenciaPlantillaDocumento `json:"plantilla"`
	ModuloID         string                       `json:"modulo_id"`
	TipoDocumental   string                       `json:"tipo_documental"`
	Clasificacion    string                       `json:"clasificacion"`
	Relaciones       []RelacionDocumento          `json:"relaciones"`
	Estado           EstadoDocumentoLogico        `json:"estado"`
	HuellaDatosHMAC  string                       `json:"huella_datos_hmac"`
	HuellaFuenteHMAC string                       `json:"huella_fuente_hmac"`
	CreadoPor        string                       `json:"creado_por"`
	CreadoEn         time.Time                    `json:"creado_en"`
	CorrelacionRef   string                       `json:"correlacion_ref"`
	Motivo           string                       `json:"motivo"`
	ENI              MetadatosENI                 `json:"eni"`
}
```

DocumentoLogico agrupa todas las representaciones de una misma version de
contenido. HuellaDatosHMAC protege los valores fusionados y HuellaFuenteHMAC
identifica la fuente semantica comun a sus representaciones.

```go
func (d DocumentoLogico) ClonarCanonico() (DocumentoLogico, error)
```

ClonarCanonico devuelve una copia independiente con relaciones ordenadas.

```go
func (d DocumentoLogico) Referencia() ReferenciaDocumento

func (d DocumentoLogico) Validar() error

type EdicionBorradorFuenteAutoridad struct {
	RevisionAnterior              uint64                      `json:"revision_anterior"`
	RevisionNueva                 uint64                      `json:"revision_nueva"`
	ActorRef                      string                      `json:"actor_ref"`
	MotivoCodigo                  CodigoMotivoFuenteAutoridad `json:"motivo_codigo"`
	RegistradaEn                  time.Time                   `json:"registrada_en"`
	HuellaContenidoAnteriorSHA256 string                      `json:"huella_contenido_anterior_sha256"`
	HuellaContenidoNuevaSHA256    string                      `json:"huella_contenido_nueva_sha256"`
	HuellaHistoriaAnteriorSHA256  string                      `json:"huella_historia_anterior_sha256"`
	HuellaHistoriaNuevaSHA256     string                      `json:"huella_historia_nueva_sha256"`
}
```

EdicionBorradorFuenteAutoridad conserva todos los actores que alteraron el
borrador y encadena la huella anterior con la nueva.

```go
type EfectoPoliticaRestrictiva string

const (
	EfectoPoliticaRestringir EfectoPoliticaRestrictiva = "restringir"
	EfectoPoliticaDenegar    EfectoPoliticaRestrictiva = "denegar"
)
type EntradaCatalogoConfigurable struct {
	Clave        string            `json:"clave"`
	Etiqueta     string            `json:"etiqueta"`
	Descripcion  string            `json:"descripcion,omitempty"`
	Orden        int               `json:"orden"`
	VigenteDesde time.Time         `json:"vigente_desde"`
	VigenteHasta time.Time         `json:"vigente_hasta,omitempty"`
	Atributos    map[string]string `json:"atributos,omitempty"`
}
```

EntradaCatalogoConfigurable es un valor de negocio administrable sin
recompilar. Atributos permite ampliar metadatos sencillos; las estructuras
complejas deben apuntar a una definicion o regla versionada independiente.

```go
func (e EntradaCatalogoConfigurable) Validar() error

func (e EntradaCatalogoConfigurable) VigenteEn(instante time.Time) bool

type EstadoAnalisisCarga string

const (
	EstadoAnalisisCargaLimpio        EstadoAnalisisCarga = "limpio"
	EstadoAnalisisCargaMalicioso     EstadoAnalisisCarga = "malicioso"
	EstadoAnalisisCargaSospechoso    EstadoAnalisisCarga = "sospechoso"
	EstadoAnalisisCargaNoConcluyente EstadoAnalisisCarga = "no_concluyente"
	EstadoAnalisisCargaError         EstadoAnalisisCarga = "error"
)
func (e EstadoAnalisisCarga) Valido() bool

type EstadoAntivirusDocumento string

const (
	EstadoAntivirusPendiente EstadoAntivirusDocumento = "pendiente"
	EstadoAntivirusLimpio    EstadoAntivirusDocumento = "limpio"
	EstadoAntivirusRechazado EstadoAntivirusDocumento = "rechazado"
	EstadoAntivirusError     EstadoAntivirusDocumento = "error"
	EstadoAntivirusNoAplica  EstadoAntivirusDocumento = "no_aplica_generado"
)
func (e EstadoAntivirusDocumento) Valido() bool

type EstadoAsignacionPerfil string

const (
	EstadoAsignacionPerfilActiva   EstadoAsignacionPerfil = "activa"
	EstadoAsignacionPerfilRevocada EstadoAsignacionPerfil = "revocada"
)
type EstadoCargaDocumental string
```

EstadoCargaDocumental expresa estados de seguridad, no estados de merito o
tramitacion administrativa. Una carga admitida aun puede ser rechazada por
RRHH por no cumplir las bases de una convocatoria.

```go
const (
	EstadoCargaDocumentalReservada         EstadoCargaDocumental = "reservada"
	EstadoCargaDocumentalPreparada         EstadoCargaDocumental = "preparada"
	EstadoCargaDocumentalCuarentena        EstadoCargaDocumental = "cuarentena"
	EstadoCargaDocumentalAnalizadaLimpia   EstadoCargaDocumental = "analizada_limpia"
	EstadoCargaDocumentalRetenidaSeguridad EstadoCargaDocumental = "retenida_seguridad"
	EstadoCargaDocumentalAdmitida          EstadoCargaDocumental = "admitida"
	EstadoCargaDocumentalAbandonada        EstadoCargaDocumental = "abandonada"
	EstadoCargaDocumentalExpirada          EstadoCargaDocumental = "expirada"
)
func (e EstadoCargaDocumental) Valido() bool

type EstadoCatalogoConfigurable string

const (
	EstadoCatalogoBorrador  EstadoCatalogoConfigurable = "borrador"
	EstadoCatalogoPublicado EstadoCatalogoConfigurable = "publicado"
	EstadoCatalogoRetirado  EstadoCatalogoConfigurable = "retirado"
)
func (e EstadoCatalogoConfigurable) Valido() bool

type EstadoCobro string

const (
	EstadoCobroCreada               EstadoCobro = "creada"
	EstadoCobroEnviadaPasarela      EstadoCobro = "enviada_a_pasarela"
	EstadoCobroResultadoPendiente   EstadoCobro = "resultado_pendiente"
	EstadoCobroConfirmada           EstadoCobro = "confirmada"
	EstadoCobroConciliada           EstadoCobro = "conciliada"
	EstadoCobroRechazada            EstadoCobro = "rechazada"
	EstadoCobroCancelada            EstadoCobro = "cancelada"
	EstadoCobroCaducada             EstadoCobro = "caducada"
	EstadoCobroResultadoDesconocido EstadoCobro = "resultado_desconocido"
	EstadoCobroDevolucionSolicitada EstadoCobro = "devolucion_solicitada"
	EstadoCobroDevolucionRechazada  EstadoCobro = "devolucion_rechazada"
	EstadoCobroDevuelta             EstadoCobro = "devuelta"
	EstadoCobroDevolucionConciliada EstadoCobro = "devolucion_conciliada"
	EstadoCobroIncidenciaBloqueada  EstadoCobro = "incidencia_bloqueada"
)
func (e EstadoCobro) Valido() bool

type EstadoCodigoCotejo string

const (
	EstadoCodigoCotejoReservado  EstadoCodigoCotejo = "reservado"
	EstadoCodigoCotejoActivo     EstadoCodigoCotejo = "activo"
	EstadoCodigoCotejoRetirado   EstadoCodigoCotejo = "retirado"
	EstadoCodigoCotejoSustituido EstadoCodigoCotejo = "sustituido"
)
func (e EstadoCodigoCotejo) Valido() bool

type EstadoControlVigenciaVersionRol string

const (
	EstadoControlVigenciaVersionRolHabilitada EstadoControlVigenciaVersionRol = "habilitada"
	EstadoControlVigenciaVersionRolRetirada   EstadoControlVigenciaVersionRol = "retirada"
)
type EstadoDefinicionFlujo string

const (
	EstadoDefinicionFlujoBorrador  EstadoDefinicionFlujo = "borrador"
	EstadoDefinicionFlujoPublicada EstadoDefinicionFlujo = "publicada"
	EstadoDefinicionFlujoRetirada  EstadoDefinicionFlujo = "retirada"
)
func (e EstadoDefinicionFlujo) Valido() bool

type EstadoDocumento string

const (
	EstadoDocumentoBorrador       EstadoDocumento = "borrador"
	EstadoDocumentoGenerado       EstadoDocumento = "generado"
	EstadoDocumentoPendienteFirma EstadoDocumento = "pendiente_firma"
	EstadoDocumentoFirmado        EstadoDocumento = "firmado"
	EstadoDocumentoRegistrado     EstadoDocumento = "registrado"
	EstadoDocumentoAnulado        EstadoDocumento = "anulado"
)
func (e EstadoDocumento) Valido() bool

type EstadoDocumentoLogico string
```

EstadoDocumentoLogico expresa el avance administrativo. Nunca se deduce
del formato: DOCX y PDF pueden representar el mismo documento en el mismo
estado.

```go
const (
	EstadoDocumentoLogicoBorrador       EstadoDocumentoLogico = "borrador"
	EstadoDocumentoLogicoEnRevision     EstadoDocumentoLogico = "en_revision"
	EstadoDocumentoLogicoCerrado        EstadoDocumentoLogico = "cerrado"
	EstadoDocumentoLogicoPendienteFirma EstadoDocumentoLogico = "pendiente_firma"
	EstadoDocumentoLogicoFirmado        EstadoDocumentoLogico = "firmado"
	EstadoDocumentoLogicoRegistrado     EstadoDocumentoLogico = "registrado"
	EstadoDocumentoLogicoAnulado        EstadoDocumentoLogico = "anulado"
)
func (e EstadoDocumentoLogico) Valido() bool

type EstadoFlujoConfigurable struct {
	Clave     string                    `json:"clave"`
	Catalogo  ReferenciaEntradaCatalogo `json:"catalogo"`
	Orden     int                       `json:"orden"`
	Terminal  bool                      `json:"terminal"`
	Atributos map[string]string         `json:"atributos,omitempty"`
}

func (e EstadoFlujoConfigurable) Validar() error

type EstadoFuenteAutoridad string

const (
	EstadoFuenteAutoridadBorrador   EstadoFuenteAutoridad = "borrador"
	EstadoFuenteAutoridadPublicada  EstadoFuenteAutoridad = "publicada"
	EstadoFuenteAutoridadSuspendida EstadoFuenteAutoridad = "suspendida"
	EstadoFuenteAutoridadDerogada   EstadoFuenteAutoridad = "derogada"
)
func (e EstadoFuenteAutoridad) Valido() bool

type EstadoPerfilDocumental string
```

EstadoPerfilDocumental solo conserva compatibilidad fail-closed del primer
corte. El perfil ya no almacena estado y Estado() devuelve valor invalido.

```go
const (
	EstadoPerfilDocumentalVigente  EstadoPerfilDocumental = "vigente"
	EstadoPerfilDocumentalRetirado EstadoPerfilDocumental = "retirado"
)
type EstadoPlantillaDocumento string
```

EstadoPlantillaDocumento gobierna el ciclo de vida de una version.
Una version publicada es inmutable; cualquier cambio crea una version nueva.

```go
const (
	EstadoPlantillaBorrador  EstadoPlantillaDocumento = "borrador"
	EstadoPlantillaPublicada EstadoPlantillaDocumento = "publicada"
	EstadoPlantillaRetirada  EstadoPlantillaDocumento = "retirada"
)
func (e EstadoPlantillaDocumento) Valido() bool

type EstadoPoliticaCotejo string

const (
	EstadoPoliticaCotejoBorrador  EstadoPoliticaCotejo = "borrador"
	EstadoPoliticaCotejoPublicada EstadoPoliticaCotejo = "publicada"
	EstadoPoliticaCotejoRetirada  EstadoPoliticaCotejo = "retirada"
)
func (e EstadoPoliticaCotejo) Valido() bool

type EstadoPoliticaRestrictiva string

const (
	EstadoPoliticaRestrictivaPublicada EstadoPoliticaRestrictiva = "publicada"
	EstadoPoliticaRestrictivaRetirada  EstadoPoliticaRestrictiva = "retirada"
)
type EstadoPublicacionPerfilDocumental string

const (
	EstadoPublicacionPerfilVigente  EstadoPublicacionPerfilDocumental = "vigente"
	EstadoPublicacionPerfilRevocada EstadoPublicacionPerfilDocumental = "revocada"
	EstadoPublicacionPerfilRetirada EstadoPublicacionPerfilDocumental = "retirada"
)
func (e EstadoPublicacionPerfilDocumental) Valido() bool

type EstadoRepresentacionDocumento string
```

EstadoRepresentacionDocumento solo describe disponibilidad tecnica.

```go
const (
	EstadoRepresentacionPendiente  EstadoRepresentacionDocumento = "pendiente"
	EstadoRepresentacionDisponible EstadoRepresentacionDocumento = "disponible"
	EstadoRepresentacionCuarentena EstadoRepresentacionDocumento = "cuarentena"
	EstadoRepresentacionRechazada  EstadoRepresentacionDocumento = "rechazada"
	EstadoRepresentacionRetirada   EstadoRepresentacionDocumento = "retirada"
)
func (e EstadoRepresentacionDocumento) Valido() bool

type EstadoVersionRol string

const (
	EstadoVersionRolPublicada EstadoVersionRol = "publicada"
	EstadoVersionRolRetirada  EstadoVersionRol = "retirada"
)
type EstadoVinculoContextoActor string
```

EstadoVinculoContextoActor es una lista cerrada. Una version revocada se
conserva como evidencia historica, pero nunca puede producir un contexto.

```go
const (
	EstadoVinculoContextoActorActivo   EstadoVinculoContextoActor = "activo"
	EstadoVinculoContextoActorRevocado EstadoVinculoContextoActor = "revocado"
)
func (e EstadoVinculoContextoActor) Valido() bool

type Event struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	ModuleID   string            `json:"module_id"`
	SubjectRef string            `json:"subject_ref"`
	ActorID    string            `json:"actor_id"`
	Payload    map[string]string `json:"payload,omitempty"`
	OccurredAt time.Time         `json:"occurred_at"`
}

type EvidenciaActoFuenteAutoridad struct {
	EvidenciaRef                string                    `json:"evidencia_ref"`
	Accion                      AccionActoFuenteAutoridad `json:"accion"`
	FuenteID                    string                    `json:"fuente_id"`
	FuenteVersion               uint64                    `json:"fuente_version"`
	HuellaContenidoSHA256       string                    `json:"huella_contenido_sha256"`
	ActoRef                     string                    `json:"acto_ref"`
	DocumentoRef                string                    `json:"documento_ref"`
	RepresentacionRef           string                    `json:"representacion_ref"`
	HuellaDocumentoSHA256       string                    `json:"huella_documento_sha256"`
	OrganoRef                   string                    `json:"organo_ref"`
	FirmasRefs                  []string                  `json:"firmas_refs"`
	ComprobadorRef              string                    `json:"comprobador_ref"`
	AtestacionRef               string                    `json:"atestacion_ref"`
	HuellaAtestacionSHA256      string                    `json:"huella_atestacion_sha256"`
	FirmaAtestacionRef          string                    `json:"firma_atestacion_ref"`
	HuellaCompromisoSHA256      string                    `json:"huella_compromiso_sha256"`
	HuellaMensajeAtestadoSHA256 string                    `json:"huella_mensaje_atestado_sha256"`
	ActoOcurridoEn              time.Time                 `json:"acto_ocurrido_en"`
	ComprobadaEn                time.Time                 `json:"comprobada_en"`
}
```

EvidenciaActoFuenteAutoridad es una atestacion neutral producida por un
puerto de comprobacion. Validar comprueba coherencia estructural, no firma,
competencia ni procedencia criptografica.

```go
func (e EvidenciaActoFuenteAutoridad) ClonarCanonica() (EvidenciaActoFuenteAutoridad, error)

func (e EvidenciaActoFuenteAutoridad) Validar() error

type EvidenciaAprobacionFlujo struct {
	AprobacionRef                   string        `json:"aprobacion_ref"`
	AprobadorID                     string        `json:"aprobador_id"`
	PerfilAprobadorRef              string        `json:"perfil_aprobador_ref"`
	Garantia                        AuthAssurance `json:"garantia"`
	SolicitanteID                   string        `json:"solicitante_id"`
	DefinicionRef                   string        `json:"definicion_ref"`
	DefinicionContenidoHuellaSHA256 string        `json:"definicion_contenido_huella_sha256"`
	InstanciaRef                    string        `json:"instancia_ref"`
	InstanciaRevision               int           `json:"instancia_revision"`
	EstadoOrigen                    string        `json:"estado_origen"`
	TransicionClave                 string        `json:"transicion_clave"`
	DecisionReglaRef                string        `json:"decision_regla_ref"`
	Motivo                          string        `json:"motivo"`
	EvidenciaHuellaSHA256           string        `json:"evidencia_huella_sha256"`
	AprobadaEn                      time.Time     `json:"aprobada_en"`
	ValidaHasta                     time.Time     `json:"valida_hasta"`
}
```

EvidenciaAprobacionFlujo fija una aprobacion independiente a la revision y
estado exactos que fueron revisados. Un simple identificador recibido del
cliente nunca basta para satisfacer el doble control.

```go
func (e EvidenciaAprobacionFlujo) HuellaSHA256() (string, error)

func (e EvidenciaAprobacionFlujo) Validar() error

func (e EvidenciaAprobacionFlujo) VigenteEn(instante time.Time) bool

type EvidenciaConciliacionCobro struct {
	// Has unexported fields.
}

func NuevaEvidenciaConciliacionCobroVerificada(datos DatosEvidenciaServidorCobro, tipo TipoConciliacionCobro, conciliacionRef, devolucionRef string) (EvidenciaConciliacionCobro, error)

func (e EvidenciaConciliacionCobro) Format(s fmt.State, _ rune)

func (e EvidenciaConciliacionCobro) GoString() string

func (EvidenciaConciliacionCobro) MarshalJSON() ([]byte, error)

func (EvidenciaConciliacionCobro) MarshalText() ([]byte, error)

func (e EvidenciaConciliacionCobro) String() string

func (*EvidenciaConciliacionCobro) UnmarshalJSON([]byte) error

func (e EvidenciaConciliacionCobro) Validar() error

type EvidenciaEmisionDocumento struct {
	Documento      ReferenciaDocumento  `json:"documento"`
	VersionEmitida VersionEmitidaCotejo `json:"version_emitida"`
	Apta           bool                 `json:"apta"`
	EvidenciaRef   string               `json:"evidencia_ref"`
}

func (e EvidenciaEmisionDocumento) Validar() error

type EvidenciaInicioOperacionCobro struct {
	// Has unexported fields.
}

func NuevaEvidenciaInicioOperacionCobroVerificada(datos DatosEvidenciaServidorCobro) (EvidenciaInicioOperacionCobro, error)
```

NuevaEvidenciaInicioOperacionCobroVerificada no verifica criptografia.
Es una fabrica de frontera para implementaciones del puerto
VerificadorPasarelaCobro; los adaptadores HTTP no deben invocarla.

```go
func (e EvidenciaInicioOperacionCobro) Control() (ControlEvidenciaInicioOperacionCobro, error)

func (e EvidenciaInicioOperacionCobro) Format(s fmt.State, _ rune)

func (e EvidenciaInicioOperacionCobro) GoString() string

func (EvidenciaInicioOperacionCobro) MarshalJSON() ([]byte, error)

func (EvidenciaInicioOperacionCobro) MarshalText() ([]byte, error)

func (e EvidenciaInicioOperacionCobro) String() string

func (*EvidenciaInicioOperacionCobro) UnmarshalJSON([]byte) error

func (e EvidenciaInicioOperacionCobro) Validar() error

type EvidenciaResultadoCobro struct {
	// Has unexported fields.
}

func NuevaEvidenciaResultadoCobroVerificada(datos DatosEvidenciaServidorCobro, resultado ResultadoOperacionCobro) (EvidenciaResultadoCobro, error)
```

NuevaEvidenciaResultadoCobroVerificada no constituye por si misma una prueba
criptografica. Valida exactamente la salida de un verificador confiable,
sin corregirla, y sella todos sus campos para detectar cambios posteriores.

```go
func (e EvidenciaResultadoCobro) Format(s fmt.State, _ rune)

func (e EvidenciaResultadoCobro) GoString() string

func (EvidenciaResultadoCobro) MarshalJSON() ([]byte, error)

func (EvidenciaResultadoCobro) MarshalText() ([]byte, error)

func (e EvidenciaResultadoCobro) String() string

func (*EvidenciaResultadoCobro) UnmarshalJSON([]byte) error

func (e EvidenciaResultadoCobro) Validar() error

type EvidenciaResultadoDevolucionCobro struct {
	// Has unexported fields.
}

func NuevaEvidenciaResultadoDevolucionCobroVerificada(datos DatosEvidenciaServidorCobro, devolucionRef string, resultado ResultadoDevolucionCobro) (EvidenciaResultadoDevolucionCobro, error)

func (e EvidenciaResultadoDevolucionCobro) Format(s fmt.State, _ rune)

func (e EvidenciaResultadoDevolucionCobro) GoString() string

func (EvidenciaResultadoDevolucionCobro) MarshalJSON() ([]byte, error)

func (EvidenciaResultadoDevolucionCobro) MarshalText() ([]byte, error)

func (e EvidenciaResultadoDevolucionCobro) String() string

func (*EvidenciaResultadoDevolucionCobro) UnmarshalJSON([]byte) error

func (e EvidenciaResultadoDevolucionCobro) Validar() error

type FormatoDocumento string
```

FormatoDocumento identifica un formato de salida que debe proporcionar un
adaptador. DOCX es el formato Word editable; el binario historico DOC no se
genera por seguridad e interoperabilidad.

```go
const (
	FormatoDocumentoPDF  FormatoDocumento = "pdf"
	FormatoDocumentoDOCX FormatoDocumento = "docx"
)
func (f FormatoDocumento) Extension() string

func (f FormatoDocumento) MIME() string

func (f FormatoDocumento) Valido() bool

type FuenteAutoridadVersionada struct {
	ID                           string                           `json:"id"`
	Version                      uint64                           `json:"version"`
	Revision                     uint64                           `json:"revision"`
	VersionAnterior              ReferenciaLinajeFuenteAutoridad  `json:"version_anterior,omitempty"`
	Contenido                    ContenidoFuenteAutoridad         `json:"contenido"`
	HuellaContenidoInicialSHA256 string                           `json:"huella_contenido_inicial_sha256"`
	HuellaHistoriaInicialSHA256  string                           `json:"huella_historia_inicial_sha256"`
	Estado                       EstadoFuenteAutoridad            `json:"estado"`
	CreadaPor                    string                           `json:"creada_por"`
	CreadaEn                     time.Time                        `json:"creada_en"`
	MotivoCreacionCodigo         CodigoMotivoFuenteAutoridad      `json:"motivo_creacion_codigo"`
	EdicionesBorrador            []EdicionBorradorFuenteAutoridad `json:"ediciones_borrador,omitempty"`
	Transiciones                 []TransicionFuenteAutoridad      `json:"transiciones,omitempty"`
}
```

FuenteAutoridadVersionada registra autoridad documental, no una regla de
negocio. Las revisiones anteriores viven en el repositorio append-only.

```go
func NuevaFuenteAutoridadBorradorV1(datos DatosAltaFuenteAutoridadV1) (FuenteAutoridadVersionada, error)

func RehidratarFuenteAutoridadV1(datos []byte) (FuenteAutoridadVersionada, error)
```

RehidratarFuenteAutoridadV1 solo acepta la representacion byte a byte
canonica de V1. Rechaza extensiones, campos repetidos, espacios, ordenes de
listas no canonicos y datos que el agregado vivo no pueda validar.

```go
func (f FuenteAutoridadVersionada) ActualizarBorrador(
	revisionEsperada uint64,
	contenido ContenidoFuenteAutoridad,
	actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	instante time.Time,
) (FuenteAutoridadVersionada, error)

func (f FuenteAutoridadVersionada) AplicarTransicionV1(
	solicitud SolicitudTransicionFuenteAutoridadV1,
	evidencia EvidenciaActoFuenteAutoridad,
	registradaEn time.Time,
) (FuenteAutoridadVersionada, error)

func (f FuenteAutoridadVersionada) Citar(preceptos ...string) (CitaFuenteAutoridad, error)
```

Citar solo expone preceptos que existen en una version ya publicada.
Una suspension o derogacion no borra la cita historica; un borrador nunca es
fuente citable.

```go
func (f FuenteAutoridadVersionada) ClonarCanonica() (FuenteAutoridadVersionada, error)

func (f FuenteAutoridadVersionada) EstadoPersistibleV1() ([]byte, error)
```

EstadoPersistibleV1 devuelve el unico JSON aceptado para la version V1.
La validacion no serializa el agregado completo; este metodo lo hace una
sola vez despues de obtener una copia defensiva canonica.

```go
func (f FuenteAutoridadVersionada) HuellaContenidoSHA256() (string, error)

func (f FuenteAutoridadVersionada) HuellaEstadoSHA256() (string, error)

func (f FuenteAutoridadVersionada) MarshalJSON() ([]byte, error)
```

MarshalJSON impide persistir por accidente la estructura viva. Todo JSON del
agregado cruza el contrato congelado EstadoPersistibleV1.

```go
func (f FuenteAutoridadVersionada) NuevaVersionV1(
	contenido ContenidoFuenteAutoridad,
	actorRef string,
	motivoCodigo CodigoMotivoFuenteAutoridad,
	instante time.Time,
) (FuenteAutoridadVersionada, error)

func (f FuenteAutoridadVersionada) PrepararSolicitudTransicionV1(
	datos DatosPreparacionTransicionFuenteAutoridadV1,
) (SolicitudTransicionFuenteAutoridadV1, error)
```

PrepararSolicitudTransicionV1 devuelve una solicitud y sus bytes canónicos
firmables. El adaptador no construye JSON ni repite los parámetros al
aplicar el acto.

```go
func (f FuenteAutoridadVersionada) ReferenciaExacta() (ReferenciaFuenteAutoridad, error)

func (f FuenteAutoridadVersionada) ReferenciaLinajeExacta() (ReferenciaLinajeFuenteAutoridad, error)

func (f *FuenteAutoridadVersionada) UnmarshalJSON(datos []byte) error
```

UnmarshalJSON impide que un adaptador eluda por accidente la rehidratacion
estricta V1 mediante encoding/json. Solo se acepta el estado canonico
exacto.

```go
func (f FuenteAutoridadVersionada) Validar() error

type HechoCobro struct {
	VersionEsquemaIntegridad    int                               `json:"version_esquema_integridad"`
	Secuencia                   int64                             `json:"secuencia"`
	Tipo                        TipoHechoCobro                    `json:"tipo"`
	EstadoAnterior              EstadoCobro                       `json:"estado_anterior,omitempty"`
	EstadoPosterior             EstadoCobro                       `json:"estado_posterior"`
	EvidenciaRef                string                            `json:"evidencia_ref"`
	EvidenciaRelacionadaRef     string                            `json:"evidencia_relacionada_ref,omitempty"`
	HuellaEvidenciaSHA256       string                            `json:"huella_evidencia_sha256"`
	HuellaMensajeOriginalSHA256 string                            `json:"huella_mensaje_original_sha256,omitempty"`
	IndiceIdempotenciaHMAC      string                            `json:"-"`
	ActorRef                    string                            `json:"actor_ref"`
	PerfilActivoRef             string                            `json:"perfil_activo_ref"`
	AccionAutorizada            AccionCobro                       `json:"accion_autorizada"`
	AutorizacionRef             string                            `json:"autorizacion_ref"`
	HuellaDecisionSHA256        string                            `json:"huella_decision_sha256"`
	AutorizacionEmitidaEn       time.Time                         `json:"autorizacion_emitida_en"`
	AutorizacionValidaHasta     time.Time                         `json:"autorizacion_valida_hasta"`
	AutorizacionEvaluadaEn      time.Time                         `json:"autorizacion_evaluada_en"`
	AtestacionAutenticacionRef  string                            `json:"atestacion_autenticacion_ref"`
	AtestacionEmitidaEn         time.Time                         `json:"atestacion_emitida_en"`
	AtestacionValidaHasta       time.Time                         `json:"atestacion_valida_hasta"`
	AutenticacionVerificadaEn   time.Time                         `json:"autenticacion_verificada_en"`
	SesionRef                   string                            `json:"sesion_ref"`
	HuellaSesionHMAC            string                            `json:"-"`
	MetodoAutenticacion         AuthMethod                        `json:"metodo_autenticacion"`
	GarantiaAutenticacion       AuthAssurance                     `json:"garantia_autenticacion"`
	CorrelacionRef              string                            `json:"correlacion_ref"`
	ConectorID                  string                            `json:"conector_id,omitempty"`
	VersionConector             int                               `json:"version_conector,omitempty"`
	OperacionProveedorRef       string                            `json:"operacion_proveedor_ref,omitempty"`
	DevolucionRef               string                            `json:"devolucion_ref,omitempty"`
	ConciliacionRef             string                            `json:"conciliacion_ref,omitempty"`
	Importe                     DineroCobro                       `json:"importe"`
	CodigoResultado             string                            `json:"codigo_resultado,omitempty"`
	VerificacionEvidenciaRef    string                            `json:"verificacion_evidencia_ref,omitempty"`
	HuellaVerificacionSHA256    string                            `json:"huella_verificacion_sha256,omitempty"`
	MetodoVerificacionEvidencia MetodoAutenticacionEvidenciaCobro `json:"metodo_verificacion_evidencia,omitempty"`
	AudienciaEvidencia          string                            `json:"audiencia_evidencia,omitempty"`
	EvidenciaEmitidaEn          time.Time                         `json:"evidencia_emitida_en,omitempty"`
	EvidenciaRecibidaEn         time.Time                         `json:"evidencia_recibida_en,omitempty"`
	EvidenciaVerificadaEn       time.Time                         `json:"evidencia_verificada_en,omitempty"`
	Motivo                      string                            `json:"motivo"`
	OcurridoEn                  time.Time                         `json:"ocurrido_en"`
	HuellaInstantaneaAltaSHA256 string                            `json:"huella_instantanea_alta_sha256,omitempty"`
	HuellaEstadoAnteriorSHA256  string                            `json:"huella_estado_anterior_sha256"`
	HuellaEstadoPosteriorSHA256 string                            `json:"huella_estado_posterior_sha256"`
}
```

HechoCobro es una entrada probatoria de solo adicion. La proyeccion puede
reconstruirse comprobando la secuencia y los estados anterior/posterior.

```go
func (h HechoCobro) Format(estado fmt.State, _ rune)

func (h HechoCobro) GoString() string

func (HechoCobro) MarshalJSON() ([]byte, error)

func (HechoCobro) MarshalText() ([]byte, error)

func (HechoCobro) String() string

func (*HechoCobro) UnmarshalJSON([]byte) error

func (h HechoCobro) Validar() error

type IdentidadSintacticaDocumental struct {
	// Has unexported fields.
}
```

IdentidadSintacticaDocumental solo identifica una familia sintactica. MIME,
extension, charset, conformidad y capacidades pertenecen al perfil.

```go
func NuevaIdentidadSintacticaDocumental(identificador string) (IdentidadSintacticaDocumental, error)

func (i IdentidadSintacticaDocumental) Identificador() string

func (i IdentidadSintacticaDocumental) Validar() error

type InstanciaFlujo struct {
	ID                              string    `json:"id"`
	TipoEntidad                     string    `json:"tipo_entidad"`
	EntidadRef                      string    `json:"entidad_ref"`
	DefinicionRef                   string    `json:"definicion_ref"`
	DefinicionContenidoHuellaSHA256 string    `json:"definicion_contenido_huella_sha256"`
	EstadoActual                    string    `json:"estado_actual"`
	Revision                        int       `json:"revision"`
	CreadaPor                       string    `json:"creada_por"`
	CreadaEn                        time.Time `json:"creada_en"`
	UltimaTransicionClave           string    `json:"ultima_transicion_clave,omitempty"`
	UltimaDecisionReglaRef          string    `json:"ultima_decision_regla_ref,omitempty"`
	UltimaAutorizacionRef           string    `json:"ultima_autorizacion_ref,omitempty"`
	UltimaAprobacionRef             string    `json:"ultima_aprobacion_ref,omitempty"`
	UltimaCorrelacionRef            string    `json:"ultima_correlacion_ref,omitempty"`
	UltimoMotivo                    string    `json:"ultimo_motivo,omitempty"`
	ActualizadaPor                  string    `json:"actualizada_por,omitempty"`
	ActualizadaEn                   time.Time `json:"actualizada_en,omitempty"`
}

func IniciarInstanciaFlujo(definicion DefinicionFlujo, id, entidadRef, actorID string, instante time.Time) (InstanciaFlujo, error)

func (i InstanciaFlujo) AplicarTransicion(
	definicion DefinicionFlujo,
	transicionClave string,
	decisionRegla DecisionReglaFlujo,
	autorizacionRef, aprobacionRef, actorID, finalidad, motivo, correlacionRef string,
	instante time.Time,
) (InstanciaFlujo, CambioEstadoFlujo, error)

func (i InstanciaFlujo) HuellaSHA256() (string, error)

func (i InstanciaFlujo) Validar() error

type InstantaneaAutorizacion struct {
	AsignacionPerfil              AsignacionPerfil          `json:"asignacion_perfil"`
	VersionRol                    VersionRol                `json:"version_rol"`
	ControlVigenciaVersionRol     ControlVigenciaVersionRol `json:"control_vigencia_version_rol"`
	Politicas                     []PoliticaRestrictiva     `json:"politicas"`
	RevisionCatalogoPoliticas     uint64                    `json:"revision_catalogo_politicas"`
	CatalogoPoliticasHuellaSHA256 string                    `json:"catalogo_politicas_huella_sha256"`
}
```

InstantaneaAutorizacion agrupa en una sola lectura coherente todos los datos
mutables que intervienen en una decision. RevisionCatalogoPoliticas cambia
ante cualquier publicacion o retirada y CatalogoPoliticasHuellaSHA256 fija
el conjunto completo de versiones actuales, incluidas las retiradas y las
que no resulten aplicables a una solicitud concreta.

```go
func (i InstantaneaAutorizacion) Validar() error

type InstantaneaContextoActor struct {
	VinculoRef      string                           `json:"vinculo_ref"`
	VinculoVersion  uint64                           `json:"vinculo_version"`
	CuentaRef       string                           `json:"cuenta_ref"`
	PersonaRef      string                           `json:"persona_ref"`
	PersonaVersion  uint64                           `json:"persona_version"`
	PerfilActivoRef string                           `json:"perfil_activo_ref"`
	PerfilVersion   uint64                           `json:"perfil_version"`
	Estado          EstadoVinculoContextoActor       `json:"estado"`
	VigenteDesde    time.Time                        `json:"vigente_desde"`
	VigenteHasta    time.Time                        `json:"vigente_hasta"`
	Vinculos        []VinculoReferenciaContextoActor `json:"vinculos"`
}
```

InstantaneaContextoActor es el resultado versionado de unir cuenta, persona
y perfil en el servidor. La fuente devuelve todas las coincidencias y la
capa de aplicacion exige exactamente una; esta estructura no concede por si
sola.

```go
func (i InstantaneaContextoActor) ClonarCanonica() (InstantaneaContextoActor, error)
```

ClonarCanonica devuelve una copia defensiva y ordena los enlaces sin elegir
uno de ellos ni eliminar duplicados silenciosamente.

```go
func (i InstantaneaContextoActor) Validar() error

func (i InstantaneaContextoActor) VigenteEn(instante time.Time) bool

type ManifiestoPreparacionCargaDirectaV1 struct {
	// Has unexported fields.
}
```

ManifiestoPreparacionCargaDirectaV1 es opaco e inmutable fuera del dominio.
Solo conserva hechos historicos no autoritativos; no contiene la sesion ni
el recibo en claro y no puede emplearse para acunar otra autorizacion.

```go
func NuevoManifiestoPreparacionCargaDirectaV1(
	carga CargaDocumental,
	contexto ContextoManifiestoPreparacionCargaDirectaV1,
) (ManifiestoPreparacionCargaDirectaV1, error)

func RestaurarManifiestoPreparacionCargaDirectaV1(
	datos DatosManifiestoPreparacionCargaDirectaV1,
) (ManifiestoPreparacionCargaDirectaV1, error)
```

RestaurarManifiestoPreparacionCargaDirectaV1 rehidrata exclusivamente
los hechos exactos que un adaptador durable obtuvo antes mediante Datos.
No completa, normaliza ni deriva ningun campo ausente: una fila parcial,
alterada o perteneciente a otro esquema falla cerrada.

```go
func (m ManifiestoPreparacionCargaDirectaV1) Datos() (DatosManifiestoPreparacionCargaDirectaV1, error)

func (m ManifiestoPreparacionCargaDirectaV1) GoString() string

func (m ManifiestoPreparacionCargaDirectaV1) HuellaSHA256() (string, error)

func (ManifiestoPreparacionCargaDirectaV1) String() string

func (m ManifiestoPreparacionCargaDirectaV1) Validar() error

func (m ManifiestoPreparacionCargaDirectaV1) ValidarContraCarga(carga CargaDocumental) error

type MarcaInstitucionalDocumento struct {
	// Has unexported fields.
}

func NuevaMarcaInstitucionalDocumento(
	institucion ReferenciaInstitucionalDocumento,
	documentoUUID string,
	perfil ReferenciaPerfilDocumental,
	fecha time.Time,
	manifiestoRef, uriPublica string,
) (MarcaInstitucionalDocumento, error)

func (m MarcaInstitucionalDocumento) DocumentoUUID() string

func (m MarcaInstitucionalDocumento) Fecha() time.Time

func (m MarcaInstitucionalDocumento) HuellaSHA256() (string, error)

func (m MarcaInstitucionalDocumento) Institucion() ReferenciaInstitucionalDocumento

func (m MarcaInstitucionalDocumento) ManifiestoRef() string

func (m MarcaInstitucionalDocumento) Perfil() ReferenciaPerfilDocumental

func (m MarcaInstitucionalDocumento) URIPublica() string

func (m MarcaInstitucionalDocumento) Validar() error

type MensajeAtestacionActoFuenteAutoridadV1 struct {
	Esquema               string                                `json:"esquema"`
	Compromiso            CompromisoTransicionFuenteAutoridadV1 `json:"compromiso"`
	EvidenciaRef          string                                `json:"evidencia_ref"`
	ActoRef               string                                `json:"acto_ref"`
	DocumentoRef          string                                `json:"documento_ref"`
	RepresentacionRef     string                                `json:"representacion_ref"`
	HuellaDocumentoSHA256 string                                `json:"huella_documento_sha256"`
	OrganoRef             string                                `json:"organo_ref"`
	FirmasRefs            []string                              `json:"firmas_refs"`
	ComprobadorRef        string                                `json:"comprobador_ref"`
	ActoOcurridoEn        time.Time                             `json:"acto_ocurrido_en"`
	ComprobadaEn          time.Time                             `json:"comprobada_en"`
}
```

MensajeAtestacionActoFuenteAutoridadV1 es el mensaje completo que cubre la
atestacion externa. Excluye unicamente el sobre criptografico que lo firma
para evitar una dependencia circular.

```go
func PrepararMensajeAtestacionActoFuenteAutoridadV1(
	solicitud SolicitudTransicionFuenteAutoridadV1,
	datos DatosMensajeAtestacionActoFuenteAutoridadV1,
) (MensajeAtestacionActoFuenteAutoridadV1, error)
```

PrepararMensajeAtestacionActoFuenteAutoridadV1 construye el único mensaje
que un conector puede firmar. El adaptador no serializa el compromiso ni
repite actor, recurso, revisión o acción.

```go
func (m MensajeAtestacionActoFuenteAutoridadV1) BytesCanonicos() ([]byte, error)

func (m MensajeAtestacionActoFuenteAutoridadV1) ConstituirEvidenciaAtestadaV1(
	sobre DatosSobreAtestacionActoFuenteAutoridadV1,
) (EvidenciaActoFuenteAutoridad, error)
```

ConstituirEvidenciaAtestadaV1 incorpora el sobre criptográfico después de
firmar/verificar el mensaje y calcula todos los campos derivados.

```go
func (m MensajeAtestacionActoFuenteAutoridadV1) HuellaSHA256() (string, error)

func (m MensajeAtestacionActoFuenteAutoridadV1) MarshalJSON() ([]byte, error)
```

MarshalJSON evita que el orden recibido de las firmas u otro detalle del
tipo vivo produzca unos bytes distintos de los entregados a Portafirmas.

```go
func (m MensajeAtestacionActoFuenteAutoridadV1) Validar() error

type MenuEntry struct {
	ID                  string   `json:"id"`
	ModuleID            string   `json:"module_id"`
	LabelKey            string   `json:"label_key"`
	Path                string   `json:"path"`
	Icon                string   `json:"icon"`
	Group               string   `json:"group"`
	Order               int      `json:"order"`
	RequiredPermissions []string `json:"required_permissions,omitempty"`
}

func (m MenuEntry) Validate() error

type MetadatosENI struct {
	Identificador     string    `json:"identificador"`
	Organo            string    `json:"organo"`
	Origen            string    `json:"origen"`
	EstadoElaboracion string    `json:"estado_elaboracion"`
	TipoDocumental    string    `json:"tipo_documental"`
	FechaCaptura      time.Time `json:"fecha_captura"`
}
```

MetadatosENI conserva el minimo transversal. Los perfiles completos y las
normas tecnicas aplicables se validaran en el adaptador de expediente ENI.

```go
func (m MetadatosENI) Validar() error

type MetodoAutenticacionEvidenciaCobro string

const (
	MetodoAutenticacionCobroFirmaMensaje        MetodoAutenticacionEvidenciaCobro = "firma_mensaje"
	MetodoAutenticacionCobroTLSMutuo            MetodoAutenticacionEvidenciaCobro = "tls_mutuo"
	MetodoAutenticacionCobroFirmaYTLSMutuo      MetodoAutenticacionEvidenciaCobro = "firma_mensaje_y_tls_mutuo"
	MetodoAutenticacionCobroConsultaAutenticada MetodoAutenticacionEvidenciaCobro = "consulta_canal_autenticado"
)
func (m MetodoAutenticacionEvidenciaCobro) Valido() bool

type ModuleManifest struct {
	ID             string       `json:"id"`
	NameKey        string       `json:"name_key"`
	DescriptionKey string       `json:"description_key"`
	Version        string       `json:"version"`
	Group          string       `json:"group"`
	BasePath       string       `json:"base_path"`
	Permissions    []Permission `json:"permissions"`
	Menu           []MenuEntry  `json:"menu"`
}

func (m ModuleManifest) Validate() error

type OrdenCobro struct {
	VersionEsquemaIntegridad         int                   `json:"-"`
	ID                               string                `json:"id"`
	Version                          int                   `json:"version"`
	IndiceIdempotenciaHMAC           string                `json:"-"`
	ExpedienteRef                    string                `json:"expediente_ref"`
	SolicitudRef                     string                `json:"solicitud_ref"`
	LiquidacionRef                   string                `json:"liquidacion_ref"`
	Tarifa                           ReferenciaTarifaCobro `json:"tarifa"`
	SujetoRef                        string                `json:"sujeto_ref"`
	RepresentacionRef                string                `json:"representacion_ref,omitempty"`
	Importe                          DineroCobro           `json:"importe"`
	Concepto                         string                `json:"concepto"`
	Finalidad                        string                `json:"finalidad"`
	CorrelacionRef                   string                `json:"correlacion_ref"`
	Estado                           EstadoCobro           `json:"estado"`
	ConectorID                       string                `json:"conector_id,omitempty"`
	VersionConector                  int                   `json:"version_conector,omitempty"`
	OperacionProveedorRef            string                `json:"operacion_proveedor_ref,omitempty"`
	CreadaEn                         time.Time             `json:"creada_en"`
	CaducaEn                         time.Time             `json:"caduca_en"`
	ConfirmadaEn                     time.Time             `json:"confirmada_en,omitempty"`
	ConciliadaEn                     time.Time             `json:"conciliada_en,omitempty"`
	ConciliacionRef                  string                `json:"conciliacion_ref,omitempty"`
	DevolucionRef                    string                `json:"devolucion_ref,omitempty"`
	IndiceIdempotenciaDevolucionHMAC string                `json:"-"`
	DevolucionSolicitadaEn           time.Time             `json:"devolucion_solicitada_en,omitempty"`
	DevueltaEn                       time.Time             `json:"devuelta_en,omitempty"`
	DevolucionConciliadaEn           time.Time             `json:"devolucion_conciliada_en,omitempty"`
	DevolucionConciliacionRef        string                `json:"devolucion_conciliacion_ref,omitempty"`
	UltimaActualizacionEn            time.Time             `json:"ultima_actualizacion_en"`
	Historial                        []HechoCobro          `json:"historial"`
	HuellaInstantaneaAltaSHA256      string                `json:"-"`
	HuellaEstadoSHA256               string                `json:"-"`
}
```

OrdenCobro no contiene PAN, CVV, PIN, criptogramas ni cargas opacas del
proveedor. Solo conserva referencias y evidencias verificadas.

```go
func NuevaOrdenCobro(alta AltaOrdenCobro, autorizacion ContextoAutorizacionCobro) (OrdenCobro, error)

func (o OrdenCobro) AplicarConciliacionServidor(evidencia EvidenciaConciliacionCobro, instante time.Time, autorizacion ContextoAutorizacionCobro, motivo string) (OrdenCobro, bool, error)

func (o OrdenCobro) AplicarResultadoDevolucionServidor(evidencia EvidenciaResultadoDevolucionCobro, instante time.Time, autorizacion ContextoAutorizacionCobro, motivo string) (OrdenCobro, bool, error)

func (o OrdenCobro) AplicarResultadoServidor(evidencia EvidenciaResultadoCobro, instante time.Time, autorizacion ContextoAutorizacionCobro, motivo string) (OrdenCobro, bool, error)

func (o OrdenCobro) Caducar(evidenciaRef, huella, motivo string, instante time.Time, autorizacion ContextoAutorizacionCobro) (OrdenCobro, bool, error)

func (o OrdenCobro) Cancelar(evidenciaRef, huella, motivo string, instante time.Time, autorizacion ContextoAutorizacionCobro) (OrdenCobro, bool, error)

func (o OrdenCobro) Clonar() OrdenCobro

func (o OrdenCobro) ControlConcurrencia() (version int, huellaEstadoSHA256 string, err error)

func (o OrdenCobro) Format(estado fmt.State, _ rune)

func (o OrdenCobro) GoString() string

func (OrdenCobro) MarshalJSON() ([]byte, error)

func (OrdenCobro) MarshalText() ([]byte, error)

func (o OrdenCobro) PrepararConciliacion(tipo TipoConciliacionCobro, referenciaCierre string, instante time.Time, autorizacion ContextoAutorizacionCobro) (ComandoConciliacionCobro, error)

func (o OrdenCobro) PrepararInicioOperacion(retornoUsuarioRef, notificacionServidorRef string, instante time.Time, autorizacion ContextoAutorizacionCobro) (ComandoInicioOperacionCobro, error)

func (o OrdenCobro) RegistrarEnvio(evidencia EvidenciaInicioOperacionCobro, instante time.Time, autorizacion ContextoAutorizacionCobro, motivo string) (OrdenCobro, bool, error)

func (o OrdenCobro) SolicitarDevolucion(solicitud SolicitudDevolucionOrdenCobro, autorizacion ContextoAutorizacionCobro) (OrdenCobro, ComandoDevolucionCobro, bool, error)

func (OrdenCobro) String() string

func (*OrdenCobro) UnmarshalJSON([]byte) error

func (o OrdenCobro) Validar() error

func (o OrdenCobro) VistaTitular() (VistaTitularOrdenCobro, error)

type PerfilFormatoDocumental struct {
	// Has unexported fields.
}
```

PerfilFormatoDocumental es una especificacion inmutable. Su estado operativo
no forma parte del perfil: se consulta en PublicacionPerfilFormatoDocumental
en cada ejecucion, permitiendo revocar sin reescribir historia.

```go
func NuevoPerfilFormatoDocumental(
	ReferenciaPerfilDocumental,
	IdentidadSintacticaDocumental,
	string, string, string,
	EstadoPerfilDocumental,
	CapacidadesPerfilFormatoDocumental,
) (PerfilFormatoDocumental, error)
```

NuevoPerfilFormatoDocumental conserva solo compatibilidad de compilacion
durante la migracion. Carece de conformidad y limite, por lo que siempre
deniega; ningun perfil legacy obtiene autoridad positiva por defecto.

```go
func NuevoPerfilFormatoDocumentalConforme(
	referencia ReferenciaPerfilDocumental,
	identidad IdentidadSintacticaDocumental,
	mime, extension, charset string,
	capacidades CapacidadesPerfilFormatoDocumental,
	conformidad ReferenciaConformidadDocumental,
	maximoBytes uint64,
) (PerfilFormatoDocumental, error)

func (p PerfilFormatoDocumental) Capacidades() CapacidadesPerfilFormatoDocumental

func (p PerfilFormatoDocumental) Charset() string

func (p PerfilFormatoDocumental) Conformidad() ReferenciaConformidadDocumental

func (p PerfilFormatoDocumental) DigestSHA256() string

func (p PerfilFormatoDocumental) Estado() EstadoPerfilDocumental

func (p PerfilFormatoDocumental) Extension() string

func (p PerfilFormatoDocumental) Identidad() IdentidadSintacticaDocumental

func (p PerfilFormatoDocumental) MIME() string

func (p PerfilFormatoDocumental) MaximoBytes() uint64

func (p PerfilFormatoDocumental) Referencia() ReferenciaPerfilDocumental

func (p PerfilFormatoDocumental) Validar() error

type PeriodoFuenteAutoridad struct {
	Desde time.Time `json:"desde"`
	Hasta time.Time `json:"hasta,omitempty"`
}
```

PeriodoFuenteAutoridad representa un intervalo semiabierto [desde, hasta).
Hasta cero significa que el periodo no tiene fin conocido. Vigencia y
efectos usan instancias distintas: ninguna de las dos se deduce de la otra.

```go
func (p PeriodoFuenteAutoridad) Contiene(instante time.Time) bool

func (p PeriodoFuenteAutoridad) Validar() error

type Permission struct {
	Key         string `json:"key"`
	LabelKey    string `json:"label_key"`
	Description string `json:"description,omitempty"`
}

type PlantillaDocumento struct {
	ID                string                    `json:"id"`
	Version           int                       `json:"version"`
	ModuloID          string                    `json:"modulo_id"`
	TipoDocumental    string                    `json:"tipo_documental"`
	Nombre            string                    `json:"nombre"`
	Titulo            string                    `json:"titulo"`
	Parrafos          []string                  `json:"parrafos"`
	Campos            []CampoPlantillaDocumento `json:"campos"`
	Formatos          []FormatoDocumento        `json:"formatos"`
	PermisoGenerar    string                    `json:"permiso_generar"`
	GarantiaMinima    AuthAssurance             `json:"garantia_minima"`
	Estado            EstadoPlantillaDocumento  `json:"estado"`
	CreadaPor         string                    `json:"creada_por"`
	CreadaEn          time.Time                 `json:"creada_en"`
	PublicadaPor      string                    `json:"publicada_por,omitempty"`
	PublicadaEn       time.Time                 `json:"publicada_en,omitempty"`
	AprobacionRef     string                    `json:"aprobacion_ref,omitempty"`
	MotivoPublicacion string                    `json:"motivo_publicacion,omitempty"`
}
```

PlantillaDocumento es una version reproducible y gobernada de una plantilla.
No existe la operacion "ultima version" para generar un acto: el caso de uso
exige ID y version explicitos.

```go
func (p PlantillaDocumento) AdmiteFormato(formato FormatoDocumento) bool

func (p PlantillaDocumento) Fusionar(datos map[string]string) (ContenidoDocumento, error)
```

Fusionar aplica una sustitucion literal y cerrada. Los valores nunca se
reinterpretan como plantilla ni se incorporan a logs o trazas.

```go
func (p PlantillaDocumento) HuellaSHA256() (string, error)

func (p PlantillaDocumento) Publicar(actor, aprobacionRef, motivo string, fecha time.Time) (PlantillaDocumento, error)
```

Publicar aplica segregacion minima: quien creo el borrador no puede
publicarlo y debe existir una referencia de aprobacion del flujo gobernado.

```go
func (p PlantillaDocumento) Validar() error

type PoliticaCotejo struct {
	ID                       string               `json:"id"`
	Version                  int                  `json:"version"`
	Revision                 int                  `json:"revision"`
	VersionAnteriorRef       string               `json:"version_anterior_ref,omitempty"`
	Nombre                   string               `json:"nombre"`
	Descripcion              string               `json:"descripcion"`
	Modulos                  []string             `json:"modulos"`
	TiposDocumentales        []string             `json:"tipos_documentales"`
	Clasificaciones          []string             `json:"clasificaciones"`
	ClaseAcceso              ClaseAccesoCotejo    `json:"clase_acceso"`
	CamposPublicos           []CampoPublicoCotejo `json:"campos_publicos,omitempty"`
	PermiteDescargaDocumento bool                 `json:"permite_descarga_documento"`
	RequiereTitularidad      bool                 `json:"requiere_titularidad"`
	RolesTitularidad         []string             `json:"roles_titularidad,omitempty"`
	RequiereFirma            bool                 `json:"requiere_firma"`
	RequiereSelloTiempo      bool                 `json:"requiere_sello_tiempo"`
	RequiereRegistro         bool                 `json:"requiere_registro"`
	GarantiaMinima           AuthAssurance        `json:"garantia_minima"`
	DiasPlazoActivacion      int                  `json:"dias_plazo_activacion"`
	DiasDisponibilidad       int                  `json:"dias_disponibilidad"`
	Estado                   EstadoPoliticaCotejo `json:"estado"`
	FuenteRef                string               `json:"fuente_ref"`
	MotivoCreacion           string               `json:"motivo_creacion"`
	CreadaPor                string               `json:"creada_por"`
	CreadaEn                 time.Time            `json:"creada_en"`
	ActualizadaPor           string               `json:"actualizada_por,omitempty"`
	ActualizadaEn            time.Time            `json:"actualizada_en,omitempty"`
	MotivoActualizacion      string               `json:"motivo_actualizacion,omitempty"`
	PublicadaPor             string               `json:"publicada_por,omitempty"`
	PublicadaEn              time.Time            `json:"publicada_en,omitempty"`
	AprobacionRef            string               `json:"aprobacion_ref,omitempty"`
	MotivoPublicacion        string               `json:"motivo_publicacion,omitempty"`
	RetiradaPor              string               `json:"retirada_por,omitempty"`
	RetiradaEn               time.Time            `json:"retirada_en,omitempty"`
	RetiradaAprobacionRef    string               `json:"retirada_aprobacion_ref,omitempty"`
	MotivoRetirada           string               `json:"motivo_retirada,omitempty"`
}
```

PoliticaCotejo es una version gobernada e inmutable tras su publicacion. Las
listas son datos de configuracion: agregar un modulo, tipo o clasificacion
no obliga a recompilar el nucleo.

```go
func (p PoliticaCotejo) ActualizarBorrador(propuesta PoliticaCotejo, actor, motivo string, fecha time.Time) (PoliticaCotejo, error)
```

ActualizarBorrador copia solo configuracion editable. Identidad, version,
autoria inicial y estado se conservan para impedir que una edicion se haga
pasar por una politica distinta o ya publicada.

```go
func (p PoliticaCotejo) Admite(documento DocumentoLogico) bool

func (p PoliticaCotejo) Aplicacion() (AplicacionPoliticaCotejo, error)

func (p PoliticaCotejo) ClonarCanonica() (PoliticaCotejo, error)

func (p PoliticaCotejo) HuellaSHA256() (string, error)

func (p PoliticaCotejo) Publicar(actor, aprobacionRef, motivo string, fecha time.Time) (PoliticaCotejo, error)

func (p PoliticaCotejo) Referencia() string

func (p PoliticaCotejo) Retirar(actor, aprobacionRef, motivo string, fecha time.Time) (PoliticaCotejo, error)

func (p PoliticaCotejo) Validar() error

type PoliticaRestrictiva struct {
	PoliticaID            string                       `json:"politica_id"`
	Version               int                          `json:"version"`
	Nombre                string                       `json:"nombre"`
	Estado                EstadoPoliticaRestrictiva    `json:"estado"`
	Efecto                EfectoPoliticaRestrictiva    `json:"efecto"`
	Acciones              []string                     `json:"acciones"`
	Modulos               []string                     `json:"modulos"`
	TiposRecurso          []string                     `json:"tipos_recurso"`
	FinalidadesPermitidas []string                     `json:"finalidades_permitidas,omitempty"`
	GarantiaMinima        AuthAssurance                `json:"garantia_minima,omitempty"`
	Restricciones         []RestriccionAtributoRecurso `json:"restricciones,omitempty"`
	RestringeCampos       bool                         `json:"restringe_campos,omitempty"`
	CamposPermitidos      []string                     `json:"campos_permitidos,omitempty"`
	Obligaciones          []string                     `json:"obligaciones,omitempty"`
	VigenteDesde          time.Time                    `json:"vigente_desde"`
	VigenteHasta          time.Time                    `json:"vigente_hasta"`
	PublicadaPor          string                       `json:"publicada_por"`
	PublicadaEn           time.Time                    `json:"publicada_en"`
	RetiradaPor           string                       `json:"retirada_por,omitempty"`
	RetiradaEn            time.Time                    `json:"retirada_en,omitempty"`
}
```

PoliticaRestrictiva expresa ABAC de efecto exclusivamente restrictivo.
No existe un efecto "permitir": sin una concesion RBAC previa siempre se
deniega.

```go
func (p PoliticaRestrictiva) AplicaA(s SolicitudAutorizacion) bool

func (p PoliticaRestrictiva) Cumple(s SolicitudAutorizacion) bool

func (p PoliticaRestrictiva) HuellaSHA256() (string, error)

func (p PoliticaRestrictiva) Referencia() string

func (p PoliticaRestrictiva) Validar() error

func (p PoliticaRestrictiva) VigenteEn(instante time.Time) bool

type PreceptoFuenteAutoridad struct {
	Clave string `json:"clave"`
	Cita  string `json:"cita"`
}
```

PreceptoFuenteAutoridad identifica un articulo, apartado, anexo o seccion.
Cita es solo una etiqueta verificable por personas; no se ejecuta ni se
interpreta como expresion.

```go
func (p PreceptoFuenteAutoridad) Validar() error

type Principal struct {
	ID            string            `json:"id"`
	DisplayName   string            `json:"display_name"`
	Email         string            `json:"email,omitempty"`
	Roles         []string          `json:"roles"`
	Permissions   []string          `json:"permissions"`
	AuthMethod    AuthMethod        `json:"auth_method"`
	AuthAssurance AuthAssurance     `json:"auth_assurance"`
	Attributes    map[string]string `json:"attributes,omitempty"`
}

func (p Principal) HasAllPermissions(permissions []string) bool

func (p Principal) HasPermission(permission string) bool

func (p Principal) Validate() error

type ProyeccionAtestacionAutorizacionV2NoAutoritativa struct {
	// Has unexported fields.
}
```

ProyeccionAtestacionAutorizacionV2NoAutoritativa acredita unicamente que
un buffer tiene la forma canonica VEC-AD-2. Sus campos son privados,
no contiene DecisionAutorizacion ni VinculoAutenticacionActorV1 y no puede
serializarse. Antes de utilizar sus compromisos, otra capa debe verificar el
sobre, la procedencia, la vigencia, la revocacion y el consumo unico.

```go
func ParsearMensajeAtestacionAutorizacionV2NoAutoritativo(
	contenido []byte,
) (ProyeccionAtestacionAutorizacionV2NoAutoritativa, error)
```

ParsearMensajeAtestacionAutorizacionV2NoAutoritativo lee estrictamente
los 35 campos de decision, los 25 del vinculo y las cuatro coordenadas del
motivo. Los limites del mensaje, textos y colecciones se comprueban antes
de reservar memoria. Al final se exige una reserializacion byte a byte
identica.

```go
func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) Cabecera() (
	CabeceraAtestacionAutorizacionV2,
	error,
)
```

Cabecera devuelve solo la seleccion nominal de suite, clave y audiencia.
No afirma que esa configuracion haya sido aprobada ni que exista una firma.

```go
func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) DecisionRef() (string, error)
```

DecisionRef devuelve el identificador opaco nominal, no una capacidad.

```go
func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) Format(estado fmt.State, _ rune)

func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) GoString() string

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) GobDecode([]byte) error

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) GobEncode() ([]byte, error)

func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) LogValue() slog.Value

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) MarshalBinary() ([]byte, error)

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) MarshalCBOR() ([]byte, error)

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) MarshalJSON() ([]byte, error)

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) MarshalText() ([]byte, error)

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) MarshalXML(*xml.Encoder, xml.StartElement) error

func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) MarshalYAML() (any, error)

func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) MotivoHuellaSHA256() (string, error)
```

MotivoHuellaSHA256 devuelve solo el compromiso nominal de la referencia de
motivo. Las cuatro coordenadas completas permanecen deliberadamente ocultas.

```go
func (p ProyeccionAtestacionAutorizacionV2NoAutoritativa) SolicitudHuellaSHA256() (string, error)
```

SolicitudHuellaSHA256 devuelve el compromiso nominal de la solicitud. No
prueba por si mismo que la solicitud existiera o fuese evaluada por el PDP.

```go
func (ProyeccionAtestacionAutorizacionV2NoAutoritativa) String() string

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) UnmarshalBinary([]byte) error

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) UnmarshalCBOR([]byte) error

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) UnmarshalJSON([]byte) error

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) UnmarshalText([]byte) error

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (*ProyeccionAtestacionAutorizacionV2NoAutoritativa) UnmarshalYAML(func(any) error) error

type ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa struct {
	// Has unexported fields.
}
```

ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa solo acredita la
forma canonica de una evidencia negativa. Conserva los datos completos en
campos privados exclusivamente para validar cruces y nunca los expone.

```go
func ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo(
	contenido []byte,
) (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa, error)
```

ParsearMensajeAtestacionDenegacionAutorizacionV1NoAutoritativo exige el
dominio VEC-AD-D-1, una decision negativa V2 completa y reserializacion
exacta.

```go
func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) Cabecera() (
	CabeceraAtestacionDenegacionAutorizacionV1,
	error,
)
```

Cabecera devuelve configuracion nominal; no selecciona una clave confiable.

```go
func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) DecisionRef() (string, error)
```

DecisionRef devuelve un identificador nominal y no una denegacion firmada.

```go
func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) Format(estado fmt.State, _ rune)

func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) GoString() string

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) GobDecode([]byte) error

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) GobEncode() ([]byte, error)

func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) LogValue() slog.Value

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MarshalBinary() ([]byte, error)

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MarshalCBOR() ([]byte, error)

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MarshalJSON() ([]byte, error)

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MarshalText() ([]byte, error)

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MarshalYAML() (any, error)

func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) MotivoHuellaSHA256() (string, error)
```

MotivoHuellaSHA256 devuelve el compromiso nominal sin revelar coordenadas.

```go
func (p ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) SolicitudHuellaSHA256() (string, error)
```

SolicitudHuellaSHA256 devuelve el compromiso nominal de solicitud.

```go
func (ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) String() string

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) UnmarshalBinary([]byte) error

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) UnmarshalCBOR([]byte) error

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) UnmarshalJSON([]byte) error

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) UnmarshalText([]byte) error

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (*ProyeccionAtestacionDenegacionAutorizacionV1NoAutoritativa) UnmarshalYAML(func(any) error) error

type ProyeccionHistoricaAtestacionAutorizacionV1 struct {
	// Has unexported fields.
}
```

ProyeccionHistoricaAtestacionAutorizacionV1 es el resultado nominal y no
autoritativo del parser estricto. Los campos internos impiden fabricar un
valor valido mediante un literal desde otro paquete. Datos devuelve siempre
copias de listas y mapas.

```go
func ParsearMensajeAtestacionAutorizacionV1NoAutoritativo(
	contenido []byte,
) (ProyeccionHistoricaAtestacionAutorizacionV1, error)
```

ParsearMensajeAtestacionAutorizacionV1NoAutoritativo interpreta el formato
historico VEC-AD-1. Antes de reservar memoria limita el mensaje, cada
texto y los conteos de listas y mapas; despues exige orden UTF-8 estricto,
semantica de concesion reforzada y una reserializacion byte a byte identica.

Esta funcion NO verifica COSE, una firma, la vigencia actual, revocaciones
ni el consumo unico de una decision. Su resultado nunca debe tratarse como
autoridad por el mero hecho de haber sido parseado.

```go
func (p ProyeccionHistoricaAtestacionAutorizacionV1) Cabecera() (CabeceraAtestacionAutorizacionV1, error)
```

Cabecera devuelve la configuracion nominal incluida en VEC-AD-1. No indica
que esa suite, clave o audiencia haya sido verificada o sea de confianza.

```go
func (p ProyeccionHistoricaAtestacionAutorizacionV1) Datos() (DatosDecisionHistoricaAtestacionAutorizacionV1, error)
```

Datos devuelve una copia defensiva de todos los datos nominales de decision.
En particular devuelve DatosVinculoAutenticacionActorV1 y nunca la capacidad
opaca VinculoAutenticacionActorV1. El bloqueo de serializacion protege esta
proyeccion completa; DatosVinculoAutenticacionActorV1, una vez extraido de
ella, conserva deliberadamente la serializacion historica definida por su
propio contrato y debe tratarse como dato personal sensible.

```go
func (p ProyeccionHistoricaAtestacionAutorizacionV1) Format(estado fmt.State, _ rune)

func (p ProyeccionHistoricaAtestacionAutorizacionV1) GoString() string

func (*ProyeccionHistoricaAtestacionAutorizacionV1) GobDecode([]byte) error

func (ProyeccionHistoricaAtestacionAutorizacionV1) GobEncode() ([]byte, error)

func (p ProyeccionHistoricaAtestacionAutorizacionV1) LogValue() slog.Value

func (ProyeccionHistoricaAtestacionAutorizacionV1) MarshalBinary() ([]byte, error)

func (ProyeccionHistoricaAtestacionAutorizacionV1) MarshalCBOR() ([]byte, error)

func (ProyeccionHistoricaAtestacionAutorizacionV1) MarshalJSON() ([]byte, error)

func (ProyeccionHistoricaAtestacionAutorizacionV1) MarshalText() ([]byte, error)

func (ProyeccionHistoricaAtestacionAutorizacionV1) MarshalXML(*xml.Encoder, xml.StartElement) error

func (ProyeccionHistoricaAtestacionAutorizacionV1) MarshalYAML() (any, error)

func (ProyeccionHistoricaAtestacionAutorizacionV1) String() string

func (*ProyeccionHistoricaAtestacionAutorizacionV1) UnmarshalBinary([]byte) error

func (*ProyeccionHistoricaAtestacionAutorizacionV1) UnmarshalJSON([]byte) error

func (*ProyeccionHistoricaAtestacionAutorizacionV1) UnmarshalText([]byte) error

type PublicacionPerfilFormatoDocumental struct {
	// Has unexported fields.
}

func NuevaPublicacionPerfilFormatoDocumental(
	publicacionRef string,
	perfil PerfilFormatoDocumental,
	revision RevisionCatalogoFormatosDocumentales,
	revisionOperativa uint64,
	estado EstadoPublicacionPerfilDocumental,
) (PublicacionPerfilFormatoDocumental, error)

func (p PublicacionPerfilFormatoDocumental) AutorizaEjecucion(
	perfil PerfilFormatoDocumental,
	revision RevisionCatalogoFormatosDocumentales,
) bool
```

AutorizaEjecucion solo concede autoridad positiva a la proyeccion operativa
actual y vigente que coincide exactamente con perfil y revision de catalogo.
Una revision historica, aunque fuera vigente en su momento, no debe usarse
sin releer el registro operativo actual.

```go
func (p PublicacionPerfilFormatoDocumental) Coincide(
	perfil PerfilFormatoDocumental,
	revision RevisionCatalogoFormatosDocumentales,
) bool

func (p PublicacionPerfilFormatoDocumental) DigestPerfilSHA256() string

func (p PublicacionPerfilFormatoDocumental) EsSucesoraDe(
	anterior PublicacionPerfilFormatoDocumental,
) bool
```

EsSucesoraDe valida la cadena append-only de la situacion operativa. Revocar
o retirar es terminal; una nueva publicacion requiere otra referencia.

```go
func (p PublicacionPerfilFormatoDocumental) Estado() EstadoPublicacionPerfilDocumental

func (p PublicacionPerfilFormatoDocumental) HuellaSHA256() string

func (p PublicacionPerfilFormatoDocumental) PerfilRef() ReferenciaPerfilDocumental

func (p PublicacionPerfilFormatoDocumental) PublicacionRef() string

func (p PublicacionPerfilFormatoDocumental) RevisionCatalogo() RevisionCatalogoFormatosDocumentales

func (p PublicacionPerfilFormatoDocumental) RevisionOperativa() uint64

func (p PublicacionPerfilFormatoDocumental) Secuencia() uint64

func (p PublicacionPerfilFormatoDocumental) Validar() error

type RecursoAutorizable struct {
	Referencia string            `json:"referencia"`
	ModuloID   string            `json:"modulo_id"`
	Tipo       string            `json:"tipo"`
	Ambitos    map[string]string `json:"ambitos,omitempty"`
	Atributos  map[string]string `json:"atributos,omitempty"`
}
```

RecursoAutorizable contiene solo el contexto de recurso obtenido por el
servidor. Los adaptadores de entrada no deben copiar atributos declarados
por el cliente sin verificarlos antes.

```go
func (r RecursoAutorizable) HuellaContextoAutorizacionSHA256() (string, error)
```

HuellaContextoAutorizacionSHA256 liga la decision a los ambitos y atributos
resueltos por el servidor sin conservarlos en la evidencia. Los adaptadores
deben aportar claves de catalogo o referencias opacas; si un valor sensible
no puede evitarse, debe llegar tokenizado/HMAC antes de construir el
recurso.

```go
func (r RecursoAutorizable) Validar() error

type ReferenciaComponenteDocumental struct {
	// Has unexported fields.
}
```

ReferenciaComponenteDocumental es el valor atestado por el registro/broker
para un rol concreto. El componente ejecutable nunca se autoacredita.

```go
func NuevaReferenciaComponenteDocumental(
	rol RolComponenteDocumental,
	identificador string,
	version uint64,
	homologacionRef, huellaHomologacionSHA256, huellaArtefactoSHA256 string,
) (ReferenciaComponenteDocumental, error)

func (r ReferenciaComponenteDocumental) HomologacionRef() string

func (r ReferenciaComponenteDocumental) HuellaArtefactoSHA256() string

func (r ReferenciaComponenteDocumental) HuellaHomologacionSHA256() string

func (r ReferenciaComponenteDocumental) Identificador() string

func (r ReferenciaComponenteDocumental) Rol() RolComponenteDocumental

func (r ReferenciaComponenteDocumental) Validar() error

func (r ReferenciaComponenteDocumental) Version() uint64

type ReferenciaConectorDocumental = ReferenciaComponenteDocumental
```

ReferenciaConectorDocumental conserva compatibilidad de compilacion hasta
retirar el contrato anterior. El nuevo camino usa siempre referencia por
rol.

```go
func NuevaReferenciaConectorDocumental(
	identificador string,
	version uint64,
	homologacionRef, huellaHomologacionSHA256, huellaArtefactoSHA256 string,
) (ReferenciaConectorDocumental, error)

type ReferenciaConformidadDocumental struct {
	// Has unexported fields.
}
```

ReferenciaConformidadDocumental compromete esquema, dialecto,
canonicalizacion y reglas concretas. Son referencias declarativas; nunca
contienen codigo, comandos ni URL ejecutables.

```go
func NuevaReferenciaConformidadDocumental(
	identificador string,
	version uint64,
	esquemaRef, dialectoRef, canonicalizacionRef, reglasRef, huellaReglasSHA256 string,
	politicaRef, huellaPoliticaSHA256 string,
) (ReferenciaConformidadDocumental, error)

func (r ReferenciaConformidadDocumental) CanonicalizacionRef() string

func (r ReferenciaConformidadDocumental) DialectoRef() string

func (r ReferenciaConformidadDocumental) DigestSHA256() string

func (r ReferenciaConformidadDocumental) EsquemaRef() string

func (r ReferenciaConformidadDocumental) HuellaPoliticaSHA256() string

func (r ReferenciaConformidadDocumental) HuellaReglasSHA256() string

func (r ReferenciaConformidadDocumental) Identificador() string

func (r ReferenciaConformidadDocumental) PoliticaRef() string

func (r ReferenciaConformidadDocumental) ReglasRef() string

func (r ReferenciaConformidadDocumental) Validar() error

func (r ReferenciaConformidadDocumental) Version() uint64

type ReferenciaCorrelacionAutorizacionV2 struct {
	// Has unexported fields.
}
```

ReferenciaCorrelacionAutorizacionV2 es una capacidad nominal opaca. Su valor
cero es invalido y no existe un constructor publico que acepte texto.

```go
func GenerarReferenciaCorrelacionAutorizacionV2(
	ctx context.Context,
	generador generadorReferenciaCorrelacionAutorizacionV2,
) (ReferenciaCorrelacionAutorizacionV2, error)
```

GenerarReferenciaCorrelacionAutorizacionV2 acuna una referencia una sola vez
mediante el puerto CSPRNG confiable y solo despues encapsula su valor. El
llamador debe reutilizar la capacidad resultante durante toda la operacion.

```go
func (r ReferenciaCorrelacionAutorizacionV2) Format(estado fmt.State, _ rune)

func (r ReferenciaCorrelacionAutorizacionV2) GoString() string

func (*ReferenciaCorrelacionAutorizacionV2) GobDecode([]byte) error

func (ReferenciaCorrelacionAutorizacionV2) GobEncode() ([]byte, error)

func (r ReferenciaCorrelacionAutorizacionV2) LogValue() slog.Value

func (ReferenciaCorrelacionAutorizacionV2) MarshalBinary() ([]byte, error)

func (ReferenciaCorrelacionAutorizacionV2) MarshalCBOR() ([]byte, error)

func (ReferenciaCorrelacionAutorizacionV2) MarshalJSON() ([]byte, error)

func (ReferenciaCorrelacionAutorizacionV2) MarshalText() ([]byte, error)

func (ReferenciaCorrelacionAutorizacionV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (ReferenciaCorrelacionAutorizacionV2) MarshalYAML() (any, error)

func (ReferenciaCorrelacionAutorizacionV2) String() string

func (*ReferenciaCorrelacionAutorizacionV2) UnmarshalBinary([]byte) error

func (*ReferenciaCorrelacionAutorizacionV2) UnmarshalCBOR([]byte) error

func (*ReferenciaCorrelacionAutorizacionV2) UnmarshalJSON([]byte) error

func (*ReferenciaCorrelacionAutorizacionV2) UnmarshalText([]byte) error

func (*ReferenciaCorrelacionAutorizacionV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (*ReferenciaCorrelacionAutorizacionV2) UnmarshalYAML(func(any) error) error

func (r ReferenciaCorrelacionAutorizacionV2) Validar() error
```

Validar permite comprobar la capacidad sin revelar su valor canonico.

```go
func (r ReferenciaCorrelacionAutorizacionV2) ValorCanonico() (string, error)
```

ValorCanonico revela el identificador solo en las fronteras que deben
comprometerlo en una decision, auditarlo o persistirlo.

```go
type ReferenciaDocumento struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}
```

ReferenciaDocumento identifica una version concreta del contenido de un
documento logico. Los cambios de estado administrativo no incrementan esta
version; una modificacion del contenido si debe crear una nueva.

```go
func (r ReferenciaDocumento) Validar() error

type ReferenciaEntradaCatalogo struct {
	CatalogoID           string `json:"catalogo_id"`
	CatalogoVersion      int    `json:"catalogo_version"`
	CatalogoHuellaSHA256 string `json:"catalogo_huella_sha256"`
	EntradaClave         string `json:"entrada_clave"`
}
```

ReferenciaEntradaCatalogo fija tanto la version como la huella del catalogo.
Una definicion de flujo nunca resuelve estados contra «la ultima version».

```go
func (r ReferenciaEntradaCatalogo) Referencia() string

func (r ReferenciaEntradaCatalogo) Validar() error

type ReferenciaFuenteAutoridad struct {
	FuenteID              string `json:"fuente_id"`
	Version               uint64 `json:"version"`
	HuellaContenidoSHA256 string `json:"huella_contenido_sha256"`
}

func (r ReferenciaFuenteAutoridad) Referencia() (string, error)

func (r ReferenciaFuenteAutoridad) Validar() error

type ReferenciaInstitucionalDocumento struct {
	// Has unexported fields.
}
```

Las referencias institucionales son opacas. Su pertenencia institucional,
ausencia de PII y URI permitida no se acreditan por prefijo ni regex:
eso lo decide una politica/catalogo institucional positivo en aplicacion.

```go
func NuevaReferenciaInstitucionalDocumento(
	entidadRef, organoRef string,
) (ReferenciaInstitucionalDocumento, error)

func (r ReferenciaInstitucionalDocumento) Entidad() string

func (r ReferenciaInstitucionalDocumento) Organo() string

func (r ReferenciaInstitucionalDocumento) Validar() error

type ReferenciaLinajeFuenteAutoridad struct {
	Fuente               ReferenciaFuenteAutoridad `json:"fuente"`
	Revision             uint64                    `json:"revision"`
	Estado               EstadoFuenteAutoridad     `json:"estado"`
	HuellaHistoriaSHA256 string                    `json:"huella_historia_sha256"`
	HuellaEstadoSHA256   string                    `json:"huella_estado_sha256"`
}
```

ReferenciaLinajeFuenteAutoridad fija no solo el contenido de la predecesora,
sino también el estado e historia exactos desde los que nació una sucesora.
No se usa como cita funcional.

```go
func (r ReferenciaLinajeFuenteAutoridad) Validar() error

type ReferenciaPerfilDocumental struct {
	// Has unexported fields.
}

func NuevaReferenciaPerfilDocumental(
	identificador string,
	version uint64,
) (ReferenciaPerfilDocumental, error)

func (r ReferenciaPerfilDocumental) Identificador() string

func (r ReferenciaPerfilDocumental) Validar() error

func (r ReferenciaPerfilDocumental) Version() uint64

type ReferenciaPlantillaDocumento struct {
	ID           string `json:"id"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}
```

ReferenciaPlantillaDocumento fija la version publicada exacta y su huella.
De este modo no existe una dependencia implicita de «la ultima plantilla».

```go
func (r ReferenciaPlantillaDocumento) Validar() error

type ReferenciaPoliticaCotejo struct {
	ID           string `json:"id"`
	Version      int    `json:"version"`
	HuellaSHA256 string `json:"huella_sha256"`
}

func (r ReferenciaPoliticaCotejo) Validar() error

type ReferenciaTarifaCobro struct {
	TarifaID        string `json:"tarifa_id"`
	Version         int    `json:"version"`
	HuellaSHA256    string `json:"huella_sha256"`
	ReglaCalculoRef string `json:"regla_calculo_ref"`
}
```

ReferenciaTarifaCobro fija la version exacta de la tarifa y su contenido.
Una orden nunca se recalcula contra la version que resulte ser la ultima.

```go
func (r ReferenciaTarifaCobro) Referencia() string

func (r ReferenciaTarifaCobro) Validar() error

type RelacionDocumento struct {
	Tipo       TipoRelacionDocumento `json:"tipo"`
	Referencia string                `json:"referencia"`
	Rol        string                `json:"rol"`
}
```

RelacionDocumento vincula el documento con entidades mediante referencias
internas opacas. Referencia no debe contener nombres, DNI, correo ni ningun
otro dato descriptivo de la persona o del expediente.

```go
func CanonizarRelacionesDocumento(relaciones []RelacionDocumento) ([]RelacionDocumento, error)
```

CanonizarRelacionesDocumento valida, clona y ordena las relaciones. Dos
relaciones con el mismo tipo, rol y referencia son un error, no dos pruebas
distintas de la misma vinculacion.

```go
func (r RelacionDocumento) Validar() error

type RepresentacionDocumento struct {
	ID                    string                        `json:"id"`
	Documento             ReferenciaDocumento           `json:"documento"`
	Tipo                  TipoRepresentacionDocumento   `json:"tipo"`
	Formato               FormatoDocumento              `json:"formato"`
	MIME                  string                        `json:"mime"`
	NombreFichero         string                        `json:"nombre_fichero"`
	Tamano                int64                         `json:"tamano"`
	HuellaContenidoSHA256 string                        `json:"huella_contenido_sha256"`
	HuellaFuenteHMAC      string                        `json:"huella_fuente_hmac"`
	ReferenciaContenido   string                        `json:"referencia_contenido"`
	EstadoTecnico         EstadoRepresentacionDocumento `json:"estado_tecnico"`
	EstadoAntivirus       EstadoAntivirusDocumento      `json:"estado_antivirus"`
	GeneradaPor           string                        `json:"generada_por"`
	GeneradaEn            time.Time                     `json:"generada_en"`
	DerivadaDeRef         string                        `json:"derivada_de_ref,omitempty"`
}
```

RepresentacionDocumento identifica los bytes exactos de un DOCX, PDF u otro
adaptador futuro. Su SHA-256 es propia; HuellaFuenteHMAC debe coincidir con
la del documento logico al que pertenece.

```go
func (r RepresentacionDocumento) Validar() error

func (r RepresentacionDocumento) ValidarPertenencia(documento DocumentoLogico) error

type RequisitoRelacionDocumento struct {
	Tipo   TipoRelacionDocumento `json:"tipo"`
	Rol    string                `json:"rol,omitempty"`
	Minimo int                   `json:"minimo"`
	Maximo int                   `json:"maximo,omitempty"`
}
```

RequisitoRelacionDocumento permite que cada plantilla o flujo declare sus
cardinalidades sin fijarlas en codigo. Rol vacio significa cualquier rol.
Maximo cero significa que no existe limite superior.

```go
func (r RequisitoRelacionDocumento) Validar() error

type RestriccionAtributoRecurso struct {
	Clave             string   `json:"clave"`
	ValoresPermitidos []string `json:"valores_permitidos"`
}

func (r RestriccionAtributoRecurso) Validar() error

type ResultadoDevolucionCobro string

const (
	ResultadoDevolucionCobroPendiente   ResultadoDevolucionCobro = "pendiente"
	ResultadoDevolucionCobroConfirmada  ResultadoDevolucionCobro = "confirmada"
	ResultadoDevolucionCobroRechazada   ResultadoDevolucionCobro = "rechazada"
	ResultadoDevolucionCobroDesconocido ResultadoDevolucionCobro = "desconocido"
)
func (r ResultadoDevolucionCobro) Valido() bool

type ResultadoGeneracionDocumento struct {
	Documento        DocumentoLogico           `json:"documento"`
	Representaciones []RepresentacionDocumento `json:"representaciones"`
	Repetida         bool                      `json:"repetida"`
}
```

ResultadoGeneracionDocumento devuelve un unico documento logico y todas sus
representaciones. Repetida indica una respuesta idempotente ya confirmada.

```go
func (r ResultadoGeneracionDocumento) ClonarCanonico() (ResultadoGeneracionDocumento, error)

func (r ResultadoGeneracionDocumento) Validar() error

type ResultadoOperacionCobro string

const (
	ResultadoOperacionCobroPendiente   ResultadoOperacionCobro = "pendiente"
	ResultadoOperacionCobroConfirmado  ResultadoOperacionCobro = "confirmado"
	ResultadoOperacionCobroRechazado   ResultadoOperacionCobro = "rechazado"
	ResultadoOperacionCobroDesconocido ResultadoOperacionCobro = "desconocido"
)
func (r ResultadoOperacionCobro) Valido() bool

type ResultadoVerificacionAutenticacionCobro struct {
	PrincipalRef     string
	Metodo           AuthMethod
	Garantia         AuthAssurance
	AutenticacionRef string
	SesionRef        string
	HuellaSesionHMAC string
	EmitidaEn        time.Time
	ValidaHasta      time.Time
}
```

ResultadoVerificacionAutenticacionCobro es la proyeccion minima que entrega
una autoridad de identidad tras comprobar la sesion. No es una concesion
de pago. La implementacion de VerificadorAutenticacionCobro es un limite de
confianza y debe proceder del servicio de identidad, nunca de la peticion.

```go
func (r ResultadoVerificacionAutenticacionCobro) Format(estado fmt.State, _ rune)

func (r ResultadoVerificacionAutenticacionCobro) GoString() string

func (ResultadoVerificacionAutenticacionCobro) MarshalJSON() ([]byte, error)

func (ResultadoVerificacionAutenticacionCobro) MarshalText() ([]byte, error)

func (ResultadoVerificacionAutenticacionCobro) String() string

func (*ResultadoVerificacionAutenticacionCobro) UnmarshalJSON([]byte) error

type RevalidadorAutenticacionActorV1 interface {
	RevalidarAutenticacionActorV1(
		context.Context,
		SolicitudRevalidacionAutenticacionActorV1,
	) (AutenticacionRevalidadaV1, error)
}
```

RevalidadorAutenticacionActorV1 es una autoridad inyectada, no un DTO.
Su implementacion debe consultar la sesion y sus controles en el origen
autoritativo. El nucleo solo emitira la capacidad opaca tras cruzar el
resultado con ContextoActor.

```go
type RevisionCatalogoFormatosDocumentales struct {
	// Has unexported fields.
}

func NuevaRevisionCatalogoFormatosDocumentales(
	numero uint64,
	huellaSHA256 string,
) (RevisionCatalogoFormatosDocumentales, error)

func (r RevisionCatalogoFormatosDocumentales) HuellaSHA256() string

func (r RevisionCatalogoFormatosDocumentales) Numero() uint64

func (r RevisionCatalogoFormatosDocumentales) Validar() error

type RolComponenteDocumental string

const (
	RolComponenteRenderizador       RolComponenteDocumental = "renderizador"
	RolComponenteMarcador           RolComponenteDocumental = "marcador"
	RolComponenteExtractorMetadatos RolComponenteDocumental = "extractor_metadatos"
	RolComponenteVerificador        RolComponenteDocumental = "verificador"
	// RolComponenteValidadorEstructural explicita el unico significado del
	// valor historico "verificador". Se conserva el alias para no reinterpretar
	// silenciosamente referencias ya publicadas.
	RolComponenteValidadorEstructural = RolComponenteVerificador
	// RolComponenteVerificadorSemantico corresponde a una carga de trabajo
	// distinta, que compara el contenido neutral con el documento producido.
	RolComponenteVerificadorSemantico RolComponenteDocumental = "verificador_semantico"
)
func (r RolComponenteDocumental) Valido() bool

type SignReceipt struct {
	ReceiptRef  string    `json:"receipt_ref"`
	DocumentRef string    `json:"document_ref"`
	SignedAt    time.Time `json:"signed_at"`
}

type SignRequest struct {
	DocumentRef string `json:"document_ref"`
	Purpose     string `json:"purpose"`
}

type SignVerification struct {
	DocumentRef string    `json:"document_ref"`
	Valid       bool      `json:"valid"`
	CheckedAt   time.Time `json:"checked_at"`
}

type SituacionOperativaPerfilDocumental = PublicacionPerfilFormatoDocumental
```

SituacionOperativaPerfilDocumental nombra expresamente el registro
append-only cuya ultima secuencia forma la proyeccion operativa actual.

```go
func NuevaSituacionOperativaPerfilDocumental(
	publicacionRef string,
	perfil PerfilFormatoDocumental,
	revision RevisionCatalogoFormatosDocumentales,
	secuencia uint64,
	estado EstadoPublicacionPerfilDocumental,
) (SituacionOperativaPerfilDocumental, error)

type SolicitudAutorizacion struct {
	Principal       Principal `json:"principal"`
	PerfilActivoRef string    `json:"perfil_activo_ref"`
	// ContextoActor y VinculoAutenticacionActor son capacidades internas. No
	// deben reconstruirse desde el cuerpo de una peticion; la frontera de
	// identidad las resuelve y revalida antes de llamar al PDP.
	ContextoActor             ContextoActor               `json:"-"`
	VinculoAutenticacionActor VinculoAutenticacionActorV1 `json:"-"`
	// ReferenciaMotivo fija para V2 la entrada exacta de un catalogo publicado.
	// Es una capacidad interna resuelta por una frontera confiable, nunca un
	// campo reconstruido directamente desde el cuerpo de una peticion.
	ReferenciaMotivo ReferenciaEntradaCatalogo `json:"-"`
	Accion           string                    `json:"accion"`
	Recurso          RecursoAutorizable        `json:"recurso"`
	Finalidad        string                    `json:"finalidad"`
	CorrelacionRef   string                    `json:"correlacion_ref"`
	Motivo           string                    `json:"motivo"`
}
```

SolicitudAutorizacion selecciona exactamente un perfil activo. Roles y
permisos incluidos en Principal son informativos y nunca son autoridad para
resolver esta solicitud.

```go
func (s SolicitudAutorizacion) TieneReferenciaMotivoAutorizacionV2() bool
```

TieneReferenciaMotivoAutorizacionV2 distingue de forma exacta una solicitud
nueva de una historica. Un valor parcialmente rellenado cuenta como presente
y sera rechazado por el constructor nominal V2.

```go
func (s SolicitudAutorizacion) Validar() error

func (s SolicitudAutorizacion) ValidarVinculoAutenticacionActor() error
```

ValidarVinculoAutenticacionActor exige la variante apta para producir una
decision durable. Validar por si solo conserva la validacion sintactica de
una solicitud, pero nunca basta para conceder ni registrar una decision.

```go
type SolicitudAutorizacionLigadaV2 struct {
	// Has unexported fields.
}
```

SolicitudAutorizacionLigadaV2 es una capacidad nominal opaca. No puede
confundirse con SolicitudAutorizacion, que conserva el contrato historico
V1.

```go
func NuevaSolicitudAutorizacionLigadaV2(
	datos DatosSolicitudAutorizacionLigadaV2,
) (SolicitudAutorizacionLigadaV2, error)
```

NuevaSolicitudAutorizacionLigadaV2 es la unica entrada al contrato V2.
Toma una copia defensiva y falla cerrado antes de crear la capacidad.

```go
func (s SolicitudAutorizacionLigadaV2) Datos() (
	DatosSolicitudAutorizacionLigadaV2,
	error,
)
```

Datos entrega una copia defensiva deliberada a la capa de aplicacion. El
resultado sigue bloqueando codecs y formato para no convertirse en DTO HTTP.

```go
func (b SolicitudAutorizacionLigadaV2) Format(estado fmt.State, _ rune)

func (b SolicitudAutorizacionLigadaV2) GoString() string

func (*SolicitudAutorizacionLigadaV2) GobDecode([]byte) error

func (SolicitudAutorizacionLigadaV2) GobEncode() ([]byte, error)

func (b SolicitudAutorizacionLigadaV2) LogValue() slog.Value

func (SolicitudAutorizacionLigadaV2) MarshalBinary() ([]byte, error)

func (SolicitudAutorizacionLigadaV2) MarshalCBOR() ([]byte, error)

func (SolicitudAutorizacionLigadaV2) MarshalJSON() ([]byte, error)

func (SolicitudAutorizacionLigadaV2) MarshalText() ([]byte, error)

func (SolicitudAutorizacionLigadaV2) MarshalXML(
	*xml.Encoder,
	xml.StartElement,
) error

func (SolicitudAutorizacionLigadaV2) MarshalYAML() (any, error)

func (SolicitudAutorizacionLigadaV2) String() string

func (*SolicitudAutorizacionLigadaV2) UnmarshalBinary([]byte) error

func (*SolicitudAutorizacionLigadaV2) UnmarshalCBOR([]byte) error

func (*SolicitudAutorizacionLigadaV2) UnmarshalJSON([]byte) error

func (*SolicitudAutorizacionLigadaV2) UnmarshalText([]byte) error

func (*SolicitudAutorizacionLigadaV2) UnmarshalXML(
	*xml.Decoder,
	xml.StartElement,
) error

func (*SolicitudAutorizacionLigadaV2) UnmarshalYAML(func(any) error) error

type SolicitudContextoActor struct {
	Cuenta          CuentaAutenticadaContextoActor `json:"cuenta"`
	PerfilActivoRef string                         `json:"perfil_activo_ref"`
}
```

SolicitudContextoActor exige un perfil concreto. Un perfil vacio no
significa "el habitual" y nunca se obtiene seleccionando el primero
disponible.

```go
func (s SolicitudContextoActor) Validar() error

type SolicitudDevolucionOrdenCobro struct {
	DevolucionRef          string
	EvidenciaRef           string
	HuellaEvidenciaSHA256  string
	IndiceIdempotenciaHMAC string
	Motivo                 string
	SolicitadaEn           time.Time
}

type SolicitudRepresentacionDocumento struct {
	Tipo    TipoRepresentacionDocumento `json:"tipo"`
	Formato FormatoDocumento            `json:"formato"`
}
```

SolicitudRepresentacionDocumento permite pedir varias salidas de una unica
fuente, por ejemplo DOCX de trabajo y PDF de visualizacion.

```go
func CanonizarSolicitudesRepresentacionDocumento(solicitudes []SolicitudRepresentacionDocumento) ([]SolicitudRepresentacionDocumento, error)

func (s SolicitudRepresentacionDocumento) Validar() error

type SolicitudRevalidacionAutenticacionActorV1 struct {
	AutenticacionRef string `json:"autenticacion_ref"`
	SesionRef        string `json:"sesion_ref"`
}
```

SolicitudRevalidacionAutenticacionActorV1 solo transporta referencias
opacas. La cuenta, persona, perfil, metodo, garantia y superficie nunca se
aceptan como atributos declarados en esta solicitud.

```go
func (s SolicitudRevalidacionAutenticacionActorV1) Validar() error

type SolicitudTransicionFuenteAutoridadV1 struct {
	// Has unexported fields.
}
```

SolicitudTransicionFuenteAutoridadV1 evita que quien integra el caso de uso
repita actor, motivo, estado o instante entre la firma y la aplicacion.

```go
func RehidratarSolicitudTransicionFuenteAutoridadV1(
	datos []byte,
) (SolicitudTransicionFuenteAutoridadV1, error)
```

RehidratarSolicitudTransicionFuenteAutoridadV1 permite reanudar de forma
segura una operación de Portafirmas tras un callback o reinicio. Solo acepta
exactamente los bytes canónicos que produjo BytesCanonicos.

```go
func (s SolicitudTransicionFuenteAutoridadV1) BytesCanonicos() ([]byte, error)

func (s SolicitudTransicionFuenteAutoridadV1) Compromiso() (CompromisoTransicionFuenteAutoridadV1, error)

func (s SolicitudTransicionFuenteAutoridadV1) MarshalJSON() ([]byte, error)
```

MarshalJSON evita que la opacidad de la solicitud se convierta
accidentalmente en {}. La representación es el compromiso V1 canónico que
puede custodiarse mientras un portafirmas completa el acto.

```go
func (s SolicitudTransicionFuenteAutoridadV1) Validar() error

type SolicitudVerificacionAutenticacionCobro struct {
	SesionRef        string
	HuellaSesionHMAC string
	Instante         time.Time
}
```

SolicitudVerificacionAutenticacionCobro identifica una sesion por una
referencia opaca y una huella HMAC de dominio separado. Nunca transporta el
token, la cookie ni material de autenticacion reutilizable.

```go
func (s SolicitudVerificacionAutenticacionCobro) Format(estado fmt.State, _ rune)

func (s SolicitudVerificacionAutenticacionCobro) GoString() string

func (SolicitudVerificacionAutenticacionCobro) MarshalJSON() ([]byte, error)

func (SolicitudVerificacionAutenticacionCobro) MarshalText() ([]byte, error)

func (SolicitudVerificacionAutenticacionCobro) String() string

func (*SolicitudVerificacionAutenticacionCobro) UnmarshalJSON([]byte) error

type SuperficieAutenticacionActorV1 string
```

SuperficieAutenticacionActorV1 es deliberadamente cerrada. La superficie
publica anonima no puede producir una decision asociada a una persona.

```go
const (
	SuperficieAutenticacionExternaPersonalV1            SuperficieAutenticacionActorV1 = "externa_personal"
	SuperficieAutenticacionInternaCorporativaV1         SuperficieAutenticacionActorV1 = "interna_corporativa"
	SuperficieAutenticacionAdministracionPrivilegiadaV1 SuperficieAutenticacionActorV1 = "administracion_privilegiada"
)
func (s SuperficieAutenticacionActorV1) Valida() bool

type TipoConciliacionCobro string

const (
	TipoConciliacionCobroIngreso    TipoConciliacionCobro = "ingreso"
	TipoConciliacionCobroDevolucion TipoConciliacionCobro = "devolucion"
)
func (t TipoConciliacionCobro) Valido() bool

type TipoHechoCobro string

const (
	HechoCobroOrdenCreada                    TipoHechoCobro = "orden_creada"
	HechoCobroOperacionEnviada               TipoHechoCobro = "operacion_enviada"
	HechoCobroResultadoPendiente             TipoHechoCobro = "resultado_pendiente"
	HechoCobroResultadoDesconocido           TipoHechoCobro = "resultado_desconocido"
	HechoCobroConfirmado                     TipoHechoCobro = "cobro_confirmado"
	HechoCobroRechazado                      TipoHechoCobro = "cobro_rechazado"
	HechoCobroCancelado                      TipoHechoCobro = "orden_cancelada"
	HechoCobroCaducado                       TipoHechoCobro = "orden_caducada"
	HechoCobroConciliado                     TipoHechoCobro = "cobro_conciliado"
	HechoCobroDevolucionSolicitada           TipoHechoCobro = "devolucion_solicitada"
	HechoCobroDevolucionResultadoPendiente   TipoHechoCobro = "devolucion_resultado_pendiente"
	HechoCobroDevolucionResultadoDesconocido TipoHechoCobro = "devolucion_resultado_desconocido"
	HechoCobroDevolucionRechazada            TipoHechoCobro = "devolucion_rechazada"
	HechoCobroDevuelto                       TipoHechoCobro = "cobro_devuelto"
	HechoCobroDevolucionConciliada           TipoHechoCobro = "devolucion_conciliada"
	HechoCobroIncidenciaDetectada            TipoHechoCobro = "incidencia_detectada"
	HechoCobroEvidenciaAdicional             TipoHechoCobro = "evidencia_adicional"
)
func (t TipoHechoCobro) Valido() bool

type TipoReferenciaContextoActor string
```

TipoReferenciaContextoActor identifica referencias opacas que pertenecen a
otros modulos. El nucleo no conoce los datos de candidato ni de empleado.

```go
const (
	TipoReferenciaContextoActorCandidato TipoReferenciaContextoActor = "candidato"
	TipoReferenciaContextoActorEmpleado  TipoReferenciaContextoActor = "empleado"
)
func (t TipoReferenciaContextoActor) Valido() bool

type TipoRelacionDocumento string
```

TipoRelacionDocumento es extensible deliberadamente. Las constantes
proporcionan un vocabulario comun, pero un modulo puede registrar nuevas
claves canonicas sin obligar a modificar el nucleo documental.

```go
const (
	TipoRelacionPersona     TipoRelacionDocumento = "persona"
	TipoRelacionProceso     TipoRelacionDocumento = "proceso"
	TipoRelacionLlamamiento TipoRelacionDocumento = "llamamiento"
	TipoRelacionContrato    TipoRelacionDocumento = "contrato"
	TipoRelacionExpediente  TipoRelacionDocumento = "expediente"
)
func (t TipoRelacionDocumento) Valido() bool

type TipoRepresentacionDocumento string
```

TipoRepresentacionDocumento describe el uso del artefacto, no su validez
administrativa. Una representacion firmada o de preservacion deriva siempre
de otra representacion inmutable.

```go
const (
	TipoRepresentacionTrabajo       TipoRepresentacionDocumento = "trabajo"
	TipoRepresentacionVisualizacion TipoRepresentacionDocumento = "visualizacion"
	TipoRepresentacionFirma         TipoRepresentacionDocumento = "firma"
	TipoRepresentacionPreservacion  TipoRepresentacionDocumento = "preservacion"
)
func (t TipoRepresentacionDocumento) Valido() bool

type TransicionFlujoConfigurable struct {
	Clave          string        `json:"clave"`
	Desde          []string      `json:"desde"`
	Hacia          string        `json:"hacia"`
	Accion         string        `json:"accion"`
	ReglaRef       string        `json:"regla_ref"`
	Prioridad      int           `json:"prioridad"`
	GarantiaMinima AuthAssurance `json:"garantia_minima"`
	// RequiereMotivo permite que la interfaz destaque una justificacion
	// reforzada. El dominio exige motivo en todas las transiciones por politica
	// general de trazabilidad administrativa.
	RequiereMotivo     bool              `json:"requiere_motivo"`
	RequiereAprobacion bool              `json:"requiere_aprobacion"`
	Automatica         bool              `json:"automatica"`
	PlazoRef           string            `json:"plazo_ref,omitempty"`
	Atributos          map[string]string `json:"atributos,omitempty"`
}

func (t TransicionFlujoConfigurable) AdmiteOrigen(estado string) bool

func (t TransicionFlujoConfigurable) Validar() error

type TransicionFuenteAutoridad struct {
	Secuencia                    uint64                       `json:"secuencia"`
	EstadoAnterior               EstadoFuenteAutoridad        `json:"estado_anterior"`
	EstadoNuevo                  EstadoFuenteAutoridad        `json:"estado_nuevo"`
	ActorRef                     string                       `json:"actor_ref"`
	MotivoCodigo                 CodigoMotivoFuenteAutoridad  `json:"motivo_codigo"`
	SolicitudRef                 string                       `json:"solicitud_ref"`
	PreparadaEn                  time.Time                    `json:"preparada_en"`
	ExpiraEn                     time.Time                    `json:"expira_en"`
	RegistradaEn                 time.Time                    `json:"registrada_en"`
	Evidencia                    EvidenciaActoFuenteAutoridad `json:"evidencia"`
	HuellaHistoriaAnteriorSHA256 string                       `json:"huella_historia_anterior_sha256"`
	HuellaHistoriaNuevaSHA256    string                       `json:"huella_historia_nueva_sha256"`
}

type VerificadorAutenticacionCobro interface {
	VerificarAutenticacionCobro(
		context.Context,
		SolicitudVerificacionAutenticacionCobro,
	) (ResultadoVerificacionAutenticacionCobro, error)
}
```

VerificadorAutenticacionCobro es el puerto minimo pendiente de cablear en
el servicio de aplicacion con la identidad opaca ya validada. El dominio no
puede demostrar el origen real de una implementacion inyectada: la raiz de
composicion debe impedir verificadores de cabeceras o datos aportados por
el llamador. Esta interfaz evita aceptar un nivel de garantia suelto como si
fuera prueba suficiente, pero no finge resolver por si sola ese limite.

```go
type VersionEmitidaCotejo struct {
	RepresentacionID      string    `json:"representacion_id"`
	ReferenciaContenido   string    `json:"referencia_contenido"`
	HuellaContenidoSHA256 string    `json:"huella_contenido_sha256"`
	MIME                  string    `json:"mime"`
	Tamano                int64     `json:"tamano"`
	FirmaRefs             []string  `json:"firma_refs,omitempty"`
	SelloTiempoRefs       []string  `json:"sello_tiempo_refs,omitempty"`
	ValidacionFirmaRef    string    `json:"validacion_firma_ref"`
	RegistroRef           string    `json:"registro_ref"`
	EmitidaEn             time.Time `json:"emitida_en"`
}
```

VersionEmitidaCotejo enlaza el CSV con los bytes exactos. Nunca se rellena
con valores declarados por un cliente: procede de la fuente de evidencias de
emision y del repositorio documental del servidor.

```go
func (v VersionEmitidaCotejo) Validar() error

type VersionRol struct {
	RolID                string           `json:"rol_id"`
	Version              int              `json:"version"`
	Nombre               string           `json:"nombre"`
	Estado               EstadoVersionRol `json:"estado"`
	Concesiones          []ConcesionRol   `json:"concesiones"`
	PublicadaPor         string           `json:"publicada_por"`
	PublicadaEn          time.Time        `json:"publicada_en"`
	RetiradaPor          string           `json:"retirada_por,omitempty"`
	RetiradaEn           time.Time        `json:"retirada_en,omitempty"`
	RetiradaRef          string           `json:"retirada_ref,omitempty"`
	MotivoRetiradaCodigo string           `json:"motivo_retirada_codigo,omitempty"`
}
```

VersionRol es una instantanea inmutable. Una asignacion siempre referencia
una version concreta, nunca "la ultima".

```go
func (v VersionRol) HuellaSHA256() (string, error)

func (v VersionRol) Referencia() string

func (v VersionRol) Validar() error

type VinculoAutenticacionActorV1 struct {
	// Has unexported fields.
}
```

VinculoAutenticacionActorV1 es una capacidad opaca. El valor cero es
invalido y otro paquete no puede rellenar sus 25 datos mediante un literal.
MarshalJSON permite guardar la evidencia; UnmarshalJSON se mantiene cerrado
hasta disponer de un rehidratador que revalide sesion y actor en una misma
transaccion autoritativa.

```go
func CrearVinculoAutenticacionActorV1(
	ctx context.Context,
	revalidador RevalidadorAutenticacionActorV1,
	solicitud SolicitudRevalidacionAutenticacionActorV1,
	actor ContextoActor,
	ahora time.Time,
) (VinculoAutenticacionActorV1, error)
```

CrearVinculoAutenticacionActorV1 es la unica fabrica publica. La fuente
autoritativa se invoca dentro de la operacion; no se acepta como argumento
un resultado de autenticacion que el llamador haya podido rellenar
directamente.

```go
func (v VinculoAutenticacionActorV1) CoincideExactamenteCon(otro VinculoAutenticacionActorV1) bool
```

CoincideExactamenteCon compara dos capacidades validas por todos sus datos
ligados. No normaliza, completa ni ignora campos; un valor cero o invalido
nunca coincide, ni siquiera con otro valor cero o invalido.

```go
func (v VinculoAutenticacionActorV1) Datos() (DatosVinculoAutenticacionActorV1, error)

func (v VinculoAutenticacionActorV1) Format(estado fmt.State, _ rune)

func (v VinculoAutenticacionActorV1) GoString() string

func (v VinculoAutenticacionActorV1) LogValue() slog.Value

func (v VinculoAutenticacionActorV1) MarshalJSON() ([]byte, error)

func (VinculoAutenticacionActorV1) MarshalText() ([]byte, error)

func (VinculoAutenticacionActorV1) String() string

func (*VinculoAutenticacionActorV1) UnmarshalJSON([]byte) error

func (*VinculoAutenticacionActorV1) UnmarshalText([]byte) error

func (v VinculoAutenticacionActorV1) Validar() error

func (v VinculoAutenticacionActorV1) ValidarPara(actor ContextoActor) error
```

ValidarPara demuestra la correspondencia exacta con el documento de actor.

```go
func (v VinculoAutenticacionActorV1) VigenteEn(instante time.Time, actor ContextoActor) bool
```

VigenteEn exige simultaneamente la sesion y el documento de actor vigentes.

```go
type VinculoReferenciaContextoActor struct {
	VinculoRef   string                      `json:"vinculo_ref"`
	Version      uint64                      `json:"version"`
	Tipo         TipoReferenciaContextoActor `json:"tipo"`
	Referencia   string                      `json:"referencia"`
	Estado       EstadoVinculoContextoActor  `json:"estado"`
	VigenteDesde time.Time                   `json:"vigente_desde"`
	VigenteHasta time.Time                   `json:"vigente_hasta"`
}
```

VinculoReferenciaContextoActor enlaza la persona canonica con una referencia
de modulo. Cada enlace tiene identidad, version, estado y vigencia propios.

```go
func (v VinculoReferenciaContextoActor) Validar() error

func (v VinculoReferenciaContextoActor) VigenteEn(instante time.Time) bool

type VistaTitularOrdenCobro struct {
	OrdenRef       string      `json:"orden_ref"`
	Estado         EstadoCobro `json:"estado"`
	Importe        DineroCobro `json:"importe"`
	CreadaEn       time.Time   `json:"creada_en"`
	CaducaEn       time.Time   `json:"caduca_en"`
	ConfirmadaEn   time.Time   `json:"confirmada_en,omitempty"`
	UltimoCambioEn time.Time   `json:"ultimo_cambio_en"`
}
```

VistaTitularOrdenCobro es la proyeccion minima que un adaptador puede
convertir expresamente en DTO. El agregado completo nunca es un DTO HTTP.

```go
type ZonaContenidoCarga string

const (
	ZonaContenidoCargaCuarentena ZonaContenidoCarga = "cuarentena"
	ZonaContenidoCargaAdmitida   ZonaContenidoCarga = "admitida"
)
func (z ZonaContenidoCarga) Valida() bool
```
