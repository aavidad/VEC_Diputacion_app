# Contrato de subida local compartida

Este contrato permite preparar los justificantes de Dietas y Cronos sobre la
custodia de Documentos. No declara esas dos subidas operativas. En este corte,
la autorización, la SQL y el montaje de `SubidorLocal` aceptan solo Méritos.
Una referencia escrita en un formulario o en otra tabla no acredita que el
original esté guardado, analizado y admitido.

## Lo que existe hoy

`internal/vec/documentos/ports/subida_local.go` publica
`SubidorLocal.Subir(ctx, SolicitudSubidaLocal, []byte)`, `Consultar(ctx,
SolicitudConsultaLocal)` y `AbrirOriginal(ctx, SolicitudConsultaLocal)`.
`SolicitudSubidaLocal` contiene el actor registrado V2, su vínculo de
autenticación V2 y una clave de idempotencia; recibe los bytes del PDF aparte.
`SolicitudConsultaLocal` conserva actor y vínculo y consulta por carga, por
documento o por la clave original. Identidad, persona, recurso, finalidad y
permiso no proceden del formulario.

El módulo dueño implementa
`ResolutorActoSubidaLocal.ResolverActoSubidaLocal(ctx, SolicitudSubidaLocal)`
y devuelve `ActoSubidaLocal`: recurso autorizable exacto, persona titular,
finalidad, clasificación y versión del dueño. Documentos pide al PDP la
decisión nueva del acto del módulo y otra decisión para cada operación del
almacén. El registro `RegistroSubidaLocal` tiene seis fachadas:
`Reservar`, `ConfirmarRecepcion`, `IniciarAnalisis`, `RegistrarAnalisis`,
`CerrarAnalisis` y `ConsultarPropia`. Las escrituras conservan recibo,
versión, auditoría e historia; la consulta deja su auditoría de lectura.
`RegistrarAnalisis` conserva el resultado antes de promover. Una recuperación
consulta la operación de almacén original y exige una decisión nueva. No
inventa otro resultado ni otro recibo.

La respuesta pública `ResultadoSubidaLocal` contiene `CargaID`,
`ComprobanteRef` (identificador documental cuando se admite),
`DocumentoVersion`, `ReciboRef`, estado, código de rechazo y fecha. El
`RecursoRef` de ese resultado queda interno. La instantánea interna también
guarda `PersonaRef`, `RecursoRef`, `HuellaPDFSHA256` de los bytes, tamaño,
MIME y los resultados de cuarentena, análisis y admisión. D20 persiste
`modulo_id`, `tipo_recurso`, `finalidad` y `propietario_version`. Hoy no
persiste una persona aportante distinta de `persona_ref` ni un custodio
jurídico separado del conector del almacén.

La declaración de Méritos usa además
`ConsultaComprobanteMeritoConfirmado.ConsultarVersionConfirmadaPropia(ctx,
persona, comprobanteRef)`. Su adaptador consulta la carga propia, exige el
recurso del hecho y la versión admitida, abre el original y coteja los bytes
con la huella guardada antes de pasar `ReferenciaDocumento{ID,Version}` a
Méritos. La consulta no acredita un título ni calcula puntos.

## Identificadores preparados para los módulos

Estas parejas figuran en `ParejaActoSubidaLocalValida` del puerto Go. Son
identificadores reservados para la integración; las acciones de Dietas y
Cronos todavía no tienen autoridad central, fachada D20 ni montaje.

| Dueño | Tipo | Finalidad | Subir | Descargar |
| --- | --- | --- | --- | --- |
| Méritos, único acto admitido en D20; recorrido pendiente | `hecho` | `custodiar_evidencia_meritos` | `meritos.cartera.comprobante.subir_propio` | `meritos.cartera.comprobante.descargar_propio` |
| Dietas, por implementar | `comision_borrador` | `custodiar_justificante_dietas` | `dietas.justificante.subir_propio` | `dietas.justificante.descargar_propio` |
| Cronos, por implementar | `justificacion` | `custodiar_justificacion_cronos` | `cronos.justificacion.subir_propio` | `cronos.justificacion.descargar_propio` |

Dietas ya preparó `vec_dietas.validar_propietario_subida_documental_v1(text,
bigint,text,text)` en Dietas16: comisión, última versión editable, persona y
contexto. Solo concede `EXECUTE` al propietario de Documentos. Cronos aún no
ha publicado su función homóloga. Su puerto C8
`FuenteOriginalJustificante.AcreditarOriginal` pide documento, versión,
SHA-256 de los bytes, solicitud titular, persona aportante y admisión antes
de enlazar el justificante. La acción Cronos `cronos.justificacion.anexar`
autoriza el enlace en Cronos; no sustituye la acción documental de subida.

## Interfaz que falta para Dietas y Cronos

1. Cada módulo publica su acción, finalidad y motivo propios, con perfil,
   superficie interna `interna_corporativa` y campos exactos. El actor y el
   vínculo llegan del certificado y del contexto registrado de la petición.
   Se reutiliza un LOGIN interno de Documentos con permiso nominal para el
   acto; no se crea una identidad técnica con permiso sobre todos los
   módulos. Dietas16 exige que ese LOGIN pertenezca a
   `vec_documentos_ejecutor`. El LOGIN externo actual
   `vec_externo_documentos_datos` solo puede pertenecer a
   `vec_documentos_subida_datos`, por lo que no sirve para Dietas16 ni debe
   recibir un segundo grupo. La versión y titularidad del recurso se
   consultan al dueño, no se deducen de una referencia.
2. Documentos publica
   `vec_documentos.validar_propietario_subida_local_v1(text,text,bigint,text,text,numeric)`
   como fachada de revalidación del dueño. Dietas16 exige esos seis tipos,
   pero todavía no llama a la función. Para la integración se fija el orden
   `(modulo_id, recurso_ref, version_dueno, persona_ref, contexto_ref,
   persona_version)`. El contexto y su versión proceden del actor registrado;
   los otros datos proceden del resolutor del módulo dueño. La función no
   acepta valores de identidad ni de titularidad aportados por HTTP.
   La función de Dietas sí tiene contrato firmado:
   `vec_dietas.validar_propietario_subida_documental_v1(p_comision_ref text,
   p_version bigint,p_persona_ref text,p_contexto_ref text) RETURNS boolean`.
   Documentos exigirá `true`; para Cronos solo podrá llamar a la función
   homóloga cuando C la publique. Una ausencia, error, versión distinta o
   estado no editable deniega solo la subida. El cotejo se hace en la
   transacción serializable que reserva o admite el original.
3. El puerto Go publicado se llama
   `ConsultaOriginalSubidaLocalPropio.ConsultarOriginalAdmitido(ctx,
   SolicitudOriginalSubidaLocalPropio)`. La solicitud contiene actor y
   vínculo V2 ya atestados, `DocumentoID`, `DocumentoVersion` y
   `CorrelacionRef`; no acepta persona, titularidad ni huella del cliente.
   Devuelve `OriginalSubidaLocalAcreditado` con `DocumentoID`, `DocumentoVersion`,
   `HuellaSHA256Bytes`, `ReciboRef`, `RecursoRef`, `PersonaTitularRef`,
   `PersonaAportanteRef`, `CustodioID`, `ModuloID`, `TipoRecurso`, `Finalidad`,
   `EstadoAdmision` y `RegistradoEn`. La proyección Go está implementada para Méritos; todavía no es
   una respuesta de la fachada SQL que consumirán Cronos y Dietas. En esta integración se
   admite únicamente la aportación propia: `PersonaAportanteRef` es la
   persona del actor atestado que realizó la reserva, y debe coincidir con
   la titular. `CustodioID` identifica el conector de almacén que figura en
   la evidencia de creación del objeto admitido; no acredita un custodio
   jurídico distinto. No se toman esos campos del navegador. La
   lectura reautoriza al actor actual, coteja documento y versión exactos,
   análisis limpio, recibo y SHA-256 de los bytes abiertos. Cronos adapta
   esa proyección a `FuenteOriginalJustificante`; Dietas coteja la línea
   y la comisión antes de conservar la referencia.
4. D20 necesita una evolución prospectiva de sus seis fachadas y de sus
   guardas positivas para cada acto de Dietas y Cronos. No se modifica ni
   reaplica la D20 instalada. Dirección ensaya la migración nueva junto a
   las fuentes de autorización, actos, LOGIN, configuración, arranque y
   recuperación. Descargar el original propio exige otra decisión nominal;
   RRHH necesita su propia acción de consulta si debe verlo. Ninguna de
   estas lecturas se concede por ser dueño del expediente.

La fachada de seis tipos valida titularidad del recurso para permitir la
carga. No es la consulta del original ya admitido. Sus datos no bastan para
devolver la huella, el recibo ni el custodio que piden C8 y Dietas.
