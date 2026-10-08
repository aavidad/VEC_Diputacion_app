# Auditoría común

Plan de continuación, 5 de octubre de 2026. Base: `main@a11790541`, con AD193
(#670) y AUT48 (#692) ya fusionadas. Recoge lo aparcado y el orden de trabajo,
para retomarlo sin repetir el análisis. No acredita instalación en la principal
ni recorridos que no estén escritos en sus PR.

## Estado

| Pieza | Situación |
| --- | --- |
| Cadena común, consumos v1 a v4, intentos AD169 y verificador | En main. AD193 está pendiente de instalar en la principal con su lista. |
| Versión vigente del rol de Aplicación (AUT48) | En main. Pendiente de instalar en la principal con su lista, después de AUT45. |
| Paquete firmado de exportación de desarrollo (#602) | En main, sin fuente de captura: no hay nadie autorizado a capturar. |
| Consulta nominal de Aplicación (WIP `f56f9f3a9`) | Rama `origin/trabajo/codexl-auditoria-nominal-20261003`. Compila y pasa sus pruebas sobre main del 05/10; le falta la misma fuente con permiso propio. |
| Lecturas fallidas CT/Bolsa (#540, #581, #582, #591) | Borradores. Esperan el ensayo nominal con la identidad RRHH CA26/IS13 y el registrador AD169. |

## Aparcado por prioridad (orden de Alberto del 05/10)

Primero van las bases de Bolsa y Contratación temporal. La exportación judicial
queda así, para retomarla en este orden:

1. **2a, AUT49: versión v7 del rol de Aplicación.** Mantenimiento v6→v7 con la
   concesión y la entrada de catálogo de `administracion.auditoria.exportar` (y
   `consultar` si se monta la consulta nominal), con ámbito de organización y una
   finalidad propia de auditoría. Sigue el patrón de AUT45: plan, preimagen, CAS,
   LOGIN exclusivo y auditoría común, más la variante 3 de `vec-mantener-admin-fijo`.
   Debe generalizar también `acreditar_perfil_aplicacion_lote_ordinario_v1`, que
   sigue fijada a v6 (ver abajo). Ensayo: lectura de usuarios en 200 con v7, que es
   además la prueba positiva de AUT48.
2. **2b, AD195: fachada de captura.** Rama nueva del núcleo para la acción de
   exportación y una fachada que, en la misma transacción, consume la decisión,
   lee un tramo acotado de la cadena común, lo proyecta al documento
   `contexto-admin-pre-v2` (el verificador ya admite todas las familias reales) con
   su cobertura, y deja el consumo v4 sellado como acuse.
3. **2c, Go.** Emisor de la decisión de exportación en vec-admin y adaptador de
   `FuenteCapturaExportacionAuditoria`. La ruta y la pantalla van después, cada una
   como minitarea, y la consulta nominal reutiliza el WIP `f56f9f3a9`.

Si la finalidad o el ámbito de la exportación necesitan criterio de RRHH o del DPD,
se pregunta en `dudas.md` antes de 2a.

### Lote ordinario fijado a v6

`acreditar_perfil_aplicacion_lote_ordinario_v1` (AUT45) exige la v6. No afecta a
la administración que va a cidonia: la principal se queda en v5 (el runbook de
arranque no aplica AUT45) y en v5 el lote ordinario no está concedido. Sólo
importa cuando se cree una versión posterior a v6, por eso entra en 2a.

## Siguiente trabajo: lecturas y descargas de Bolsa y Contratación temporal

### Corte A: lecturas de contactos de Bolsa

Las lecturas RRHH de contactos de una participación (consulta de datos de contacto, `ServicioDatosContactoParticipacion.Consultar`;
`ListarContactosParticipacion` y `ListarContactosBolsa`) ya consumen su V3 en la
transacción positiva, pero sus denegaciones y errores no dejan rastro en la
auditoría común. El corte añade el mismo decorador AD169 que #591 y publica las
referencias del acuse en la respuesta. Sólo Go, sin SQL ni permisos nuevos.

No está bloqueado para programar: AD169 y el registrador de intentos están en main.
Comparte el montaje de `bolsa_borrador_llamamiento_desarrollo.go` y la correlación
de la petición con #591, así que se apila sobre su rama. Como las otras cuatro,
el ensayo nominal por HTTP necesita la identidad RRHH CA26/IS13.

Estado: PR #719 (apilada sobre #591) cubre `ListarContactosParticipacion` y
`ListarContactosBolsa`, con revisión independiente GO. La consulta de datos de
contacto va en el corte C.

### Corte C: consulta de datos de contacto (hueco de autorización latente)

`ServicioDatosContactoParticipacion.Consultar` devolvía correo y teléfonos en
claro (`GET …/candidatos/{p}/datos-contacto?ver=completo`) sin decisión V3 propia:
la frontera común sólo exigía sesión mTLS y el perfil activo de Bolsa RRHH, y en el
catálogo esa frontera colgaba de la acción de *registrar* datos de contacto.

Estado: rama `trabajo/claude-bolsa-contacto-consulta-20261005`, apilada sobre #719
y #712. La consulta completa tiene acción, finalidad y motivo propios; AD197 (núcleo
y fachada) y B78 (lectura con consumo) la consumen en la misma transacción que la
lectura, y sus fallos van al registrador AD169. AD197 está medida sobre main + AD190
(#720), que se instala antes. Ensayo nominal en el clon H10-30 con las listas de main,
AUT51 y AD190: 403 sin concesión y sin fila de origen, 200 con asiento común,
503 con intento `error` y sin consumo, y 200 tras reiniciar. Para instalar:
`deploy/principal/lista_sql_claude_bolsa_contacto_consulta_20261005.txt`, con la fila
de origen AD172 y la provisión del rol v13-v16 que explica. La vista enmascarada
sigue sin decisión propia; si se quiere auditar también, va en otro corte. La
decisión firmada lleva la participación como recurso; la bolsa se comprueba por
pertenencia en B78, no forma parte de la firma. Revisión SQL y de seguridad
independientes: GO sobre `a0c4a4d64`.

### Corte B: descargas de borradores de Contratación temporal

Hoy la descarga PDF/DOCX de un borrador RRHH es la consulta del detalle del
expediente con otra cabecera `Accept`: la auditoría registra la consulta, no qué
documento ni en qué formato se descargó, y un fallo al generarlo después del
consumo no deja rastro. Hay dos partes:

1. Fallos (denegación, error de generación o de validación del documento) al
   registrador AD169, apilado sobre #582, que ya monta ese registrador en CT.
2. Identificar la descarga correcta (tipo de borrador, formato y huella del
   archivo) en la auditoría común. Necesita una decisión de diseño: o una acción
   propia de descarga con su concesión en el catálogo RRHH, o ligar el tipo y el
   formato al recurso de la consulta antes de consumir. Se coordina con E
   (Documentos/firma) y se escribe aquí antes de programar.

Decisión y estado (05/10, rama `trabajo/claude-ct-descargas-auditadas-20261005`,
apilada sobre #713): acción propia `contratacion_temporal.borrador_rrhh.descargar`,
con la finalidad y el motivo de la consulta del expediente. La consulta del
detalle no cambia; después de generar y validar el archivo, la descarga pide
otra decisión cuyo recurso es el expediente y cuya huella de contexto liga el
tipo de borrador, el formato, el SHA256 y el tamaño del archivo. AD199 añade el
perfil al núcleo de consultas RRHH (no al de mutaciones, así que no compite con
AD190/AD197) y CT177 la consume en la misma transacción que escribe la fila de
`descarga_borrador_rrhh_v1`. Solo entonces se entrega el archivo, con
`X-Audit-Ref` y `X-VEC-Documento-SHA256`. Las denegaciones y los errores, también
los de la consulta y los de generación cuando se pidió una descarga, van al
registrador AD169 con la acción de descarga. El perfil fijo del lector RRHH no
recibe la concesión: su descarga queda denegada hasta una provisión aparte. La
descarga de plantillas publicadas (CT133, `/expedientes/borradores`) ya tiene su
acción con tipo y formato; añadirle la huella del archivo queda pendiente.
La versión que guarda la fila es la del contenido del documento: para un
expediente en v8 o v9 el borrador sale del original de propuesta v7, y se
registra 7. La fila significa «descarga autorizada y registrada», no «bytes
recibidos por el cliente». Una consulta no observable (404), que no distingue
entre denegación e inexistencia, queda como intento «denegado».

### Provisión de identidad interna sintética — AD215

La rama `trabajo/codexv-provision-identidad-20261007` añade dos tipos a la cadena
común: provisión confirmada e intento de provisión o recuperación. El SQL AD215,
su vector y el verificador CLI aceptan ambos tipos; la operación se confirmó en
una base local sintética y devolvió el mismo recibo tras reiniciar PostgreSQL.
Este corte todavía no está instalado en la principal y no acredita empleo,
perfil, certificado ni uso en producción.

La exportación general de auditoría no incluye aún estas dos familias en
`internal/vec/auditoria/exportacion_documento.go`. El paquete de exportación
existente tampoco tiene una fuente nominal autorizada que capture el tramo.
Cuando se abra ese recorrido, crear un corte dependiente con formato versionado,
captura y consumidor autorizados para los dos tipos, y probar la cadena mixta
con asientos anteriores y nuevos. Hasta entonces, el vector SQL y el verificador
CLI prueban su formato y huellas; no equivalen a una exportación judicial completa.
