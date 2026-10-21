# Recorrido de J — F1-27

En curso, 21 de octubre de 2026. Fuente de ejecución:
`5abbbcaae5c47c65dc6954daf92015b7aad08aeb`, main con el diagnóstico de
selección nativa de F1-27. Dirección autorizó esta copia a las 09:37,
tras integrar el registro de errores fijos.

La [receta completa](../recorrido_copia/LEEME.md) crea una copia nueva con
material e identidades propios. I se ha parado con su gestor y se conserva
sin cambiar su fuente, binario, petición ni clave. H permaneció parada hasta comprobar las dos firmas de J y se retiró después
con su gestor para liberar memoria, conforme a la orden anterior de limpieza.
Sus evidencias privadas y su hoja se conservan; el expediente 2026/95032
no se reintentó. No se añaden ramas.

| Tramo | Estado |
| --- | --- |
| Creación completa y carga nativa de Bolsa | Completada en 256,57 segundos; 146 migraciones |
| Petición, ratificación y alta (1–3) | 200, 200 y 201; expediente 2026/95034, versión 1 |
| Análisis, vía, asignación e informe (4–7) | Actuaciones 201 hasta versión 5 |
| Dos firmas del informe y fiscalización (7b–8) | Dos firmas 201 y fiscalización favorable 201; versión 6 |
| Selección, aviso, contacto y aceptación (9–11) | Selección 200; aviso, contacto, aceptación, vínculo y resolución portal 201 |
| Propuesta, resolución, documentos y tres firmas (12–15) | Paso 12 bloqueado en interfaz; no se envía la propuesta |
| Incorporación, Personal, GINPIX, cese, vuelta a Bolsa y cierre (17–21) | Pendiente |
| Reinicio y recuperación de lecturas y recibos | Pendiente |

La causa del 503 de I sigue sin identificarse. El diagnóstico integrado solo
amplía el registro con errores fijos reconocidos; conserva los desconocidos
como tipo. No modifica el error devuelto, la autorización ni las reglas.
Si J confirma la selección, el recorrido continuará; si falla, se conservarán
su petición, correlación y causa observada antes de proponer una corrección.
Los materiales privados quedan fuera de Git. Esta hoja no cierra F1-27 ni F1-28.

La primera apertura del programa de firma se detuvo en la conexión TLS, antes
de mostrar el selector y sin enviar registros de firma a VEC. El navegador
se había iniciado antes de preparar las cuatro autoridades locales de J.
Se cerraron navegador y programa de firma de esa sesión y se abrió el
navegador con esas autoridades ya preparadas. Solo se modifica el entorno
privado del navegador; se conservan las comprobaciones TLS y los permisos.

Tras reabrir el navegador, ambas firmas del informe devolvieron 201,
verificación técnica válida y `firma_eficaz=false`. Hubo un registro por
firmante. El PDF descargado desde la ficha contiene 49.314 bytes y su SHA-256,
`86b8551684eaac338c3c9a7fa5c87b8508baa029b609e73254d8a576a566af99`,
coincide con el resultado de la segunda firma. Intervención confirmó después
el resultado favorable con su recibo y la versión resultante 6. Hasta este
tramo no se ha solicitado selección ni registrado contacto en J.

## Selección, contacto y aceptación

La selección devolvió 200 en su único envío, a las 10:02:08 de Madrid, con
recibo confirmado. El aviso se anotó después con respuesta 201. El resultado
de J no permite atribuir la causa del 503 de I al código ni darla por resuelta.

«Abrir llamamiento en Bolsa» llevó al asistente de otro llamamiento; no se
confirmó nada allí. Al volver a la ficha, el bloque de contacto de CT no se
muestra para una comunicación recuperada. La vía existente que sí se usó es
el seguimiento del llamamiento en Bolsa: «Ver ficha y llamar» abre la ficha de
Yago y el registro telefónico. Se confirmó «Contactado» una sola vez, con
respuesta 201 a las 10:08:51 y la hora automática del servidor. No hubo otra
selección ni un llamamiento nuevo.

Mi Bolsa mostró ese contacto y el plazo hasta el 23 de octubre a las 00:00,
hora de Madrid. Yago aceptó con su certificado; la respuesta firme devolvió
201 a las 10:09:34. RRHH vinculó por pantalla la emisión ofrecida como propia
del expediente (201), revisó la aceptación y su plazo y la confirmó (201 a
las 10:11:55). CT permanece en versión 6. No se aportó un correo ficticio
para sustituir esa respuesta ni se modificaron permisos.

## Paso 12: propuesta sin petición

La ficha muestra la candidatura aceptada y el formulario «Guardar la
propuesta», con las cuatro publicaciones cargadas. Pulsarlo no envía una
petición ni cambia el mensaje «Documentos cargados. Revise y guarde la
propuesta». Seleccionar después la comunicación existente en su lista hace
desaparecer ese formulario. No se ha enviado ningún POST de propuesta.

El inventario de código y una comprobación independiente localizan una
contradicción: `formulario-llamamiento.js:411` muestra la propuesta de Mi Bolsa
separada cuando no hay comunicación de CT seleccionada; el envío exige esa
selección en las líneas 801–805. La guarda de consulta de respuesta de las
líneas 806–807 también depende de un antecedente de comunicación de CT,
aunque la aceptación procede de Mi Bolsa. El código de preparación ya
conserva la resolución y su recibo como antecedente nominal del portal.

Se ha comunicado a dirección para asignar la corrección mínima. No se han
fabricado solicitudes ni modificado el código que prepara el envío. Las capturas a
1440 y 390 px y el registro de peticiones quedan en el material privado;
a 390 px el contenido mide 375 px y no desborda. Los pasos 12–21 y el
reinicio final siguen pendientes.

## Actualización autorizada de la aplicación

Dirección integró la corrección de propuesta en main
`0ed5c5e5e2609fc1dff336e3c526c7b63ef5a77e` y autorizó actualizar J sobre
su misma base. El árbol avanzó limpio por fast-forward. Se compiló
`vec-server` por la receta habitual, con dos procesos y el límite de memoria
acordado; terminó en 1,35 segundos. SHA-256 del binario compilado:
`03342a2500a8d2789d3a1ad0e01190f807182169721873a81fc372af023bb973`.
Sus bytes coinciden con el anterior, pues no cambia Go. La revisión y la
huella reales se registraron conservando la preimagen para vuelta atrás.
No se aplicó SQL ni se sustituyeron certificados.

Solo se cerró la aplicación anterior, comprobando su PID y ejecutable, y se
arrancó la nueva con el gestor. PostgreSQL conserva el mismo identificador
de contenedor y el mismo instante de arranque. El proceso nuevo de la
aplicación consta activo. El gestor devolvió después «ADMIN no preparado o ya
activo» al intentar arrancar de nuevo Administración, que se había conservado
activa con su mismo PID. Dirección autorizó comprobar ambos servicios sin
apagar Administración. La comprobación terminó con código 0; el detalle
recuperó la versión 6 y las mismas seis referencias de recibo. El navegador
recibió la versión web `382b427427519658` con la corrección instalada.

## Paso 12: envío de la propuesta

El formulario envió una única petición el 21 de octubre a las 11:12:33,
hora de Madrid. El servidor devolvió 503, con código
`servicio_no_disponible` y correlación
`corr_8718f8296439ab2158c4324140754f4b`. La clave conservada es
`16b2f70c-02e7-417c-a98a-e55c8082b743`. La pantalla conserva la solicitud y
no afirma que se haya guardado. No se ha repetido el envío.

La consulta posterior de la copia, dentro de una transacción de solo
lectura, encontró cero propuestas con esa clave. El expediente continúa en
la versión 6 y su último evento sigue siendo la fiscalización. Durante la
petición constan dos consumos de autorización confirmados; todavía no
permiten atribuir el fallo a una etapa concreta. El registro recoge 39
consultas, sin errores de base registrados, y clasifica la causa como
desconocida. Se conserva la petición, respuesta y registro en el material
privado. Dirección encargó localizar la causa y corregirla o añadir un
diagnóstico limitado a errores reconocidos. Los pasos 12–21 y el reinicio
final siguen pendientes.

El diagnóstico preparado en la PR #1426, revisión `57697a62a`, distingue
las etapas con textos fijos sin cambiar resultados ni permisos. Las pruebas
focales de propuesta, antecedente portal y privacidad han pasado, y la
revisión independiente de seguridad es favorable. Semgrep no encontró
hallazgos. El adaptador oficial local de GoSec completó el paquete: ninguno
en los cuatro archivos Go cambiados; sus 79 avisos están en archivos cuyos
bytes coinciden con la base `963bab91a`. Las seis puertas de CI han pasado;
calidad completa terminó en 32 minutos y 50 segundos. Dirección integró el
diagnóstico en `bf0580bcb2e4d2911b70727bd752f96f4b0c84b5`.

La orden de las 12:02 exige esperar además la integración de la corrección
de concurrencia de autorización de la PR #1428. Después se actualizará solo
la aplicación al main que contenga ambas y se hará un único reintento desde
el formulario con la misma clave. El primer intento se ejecutó en `0ed5c5e5e`,
sin esa corrección de concurrencia; se anotará la revisión exacta del segundo.
J aún conserva el binario anterior y no se ha repetido la petición.

## Segundo intento con diagnóstico y autorización serializada

J avanzó limpia a `295a0a1461cf3720906245badb7d89b35dda8181`, con las PR
#1426 y #1428 integradas. La compilación real terminó en 24,36 segundos;
SHA-256: `8f6bbb37bea62f8d13b3c2528ea233a7dce58013bfde3c7a4587c8ab4e98d21c`.
Se sustituyó solo el binario de la aplicación y se registraron su revisión y
huella reales. El gestor volvió a terminar con código 1 por Administración
ya activa; la comprobación posterior de aplicación y Administración terminó
con código 0. PostgreSQL conserva su identificador y arranque. El detalle
respondió 200, con versión 6 y las mismas seis referencias de recibo.
No se aplicó SQL ni se cambiaron certificados o configuración.

A las 12:22:44 del 21 de octubre, hora de Madrid, se pulsó una sola vez
«Volver a intentar» en la página conservada. El cuerpo completo y la clave
coinciden con el primer envío. Respondió 503, correlación
`corr_26b96ed0e5d4084687d2588009f8726e`. El diagnóstico identifica
`actor_anclaje`, con causa fija «contratacion temporal: autorizacion denegada».
Hay dos envíos en total; no se ha hecho un tercero.

La incompatibilidad está en la sesión nominal del perfil fijo de Bolsa:
la guarda privada permite consultar el anclaje desde la propuesta, pero la
sesión comprueba después que ese perfil figure en la frontera pública de
Contratación. Esa frontera solo enumera su perfil ordinario. La acción
`bolsa.aceptacion_ct.anclaje.consultar`, el recurso `anclaje_aceptacion_ct`,
la finalidad `preparar_incorporacion_ct` y su ámbito de unidad ya están
concedidos. Se prepara una corrección de la ligadura interna, sin ampliar
esa frontera ni añadir concesiones. Los pasos 12–21 siguen pendientes.

La denegación se presenta como 503, aunque debería conservar la
clasificación 403/404 que corresponda según la política del módulo.
Dirección ha pedido anotarlo por separado y mantenerlo fuera de este corte.

## Preparación de los pasos 13 a 21

Por orden de las 12:47 se hicieron consultas en J sin cambiar su fuente ni
registrar actos. Se conservó la página de Gestión con la clave pendiente;
las lecturas se hicieron en otro perfil de Chrome, cerrado al terminar.

| Tramo | Preparación comprobada | Pendiente |
| --- | --- | --- |
| 13–14 | La técnica recupera el expediente y sus antecedentes; el informe firmado se descarga con la misma huella anterior | Registrar propuesta y resolución para alcanzar v7 y v8; generar y abrir PDF y Word de los documentos del recorrido |
| 15 | El plan publicado de J fija Jefatura → Diputación → Secretaría General. Los tres entornos de firma están preparados | Las tres firmas de la resolución, sus recibos y el PDF verificado. Los cuatro documentos restantes se mantienen sin firma, según el caso de aceptación |
| 17 | El centro ve 95034, su petición ratificada y «Aún no procede» por la fase actual | La bandeja debe habilitar la confirmación tras la formalización; usar el documento que exija su regla y una fecha no futura |
| 18 y 16a–16b | Material de Personal con 27 operaciones; incorporación acreditada y gobierno B2 activos | Alta real, ficha descargada y confirmación manual de GINPIX ligada al recibo del alta |
| 19–21 | Seguimiento y cese activos; relevo de Bolsa activo; política de fecha de efecto `primer_dia_sin_relacion` | Cese, vuelta a la misma participación de Bolsa y cierre, conservando sus recibos |

La descarga técnica del informe conserva 49.314 bytes y SHA-256
`86b8551684eaac338c3c9a7fa5c87b8508baa029b609e73254d8a576a566af99`.
La vista a 390 px tiene 375 px de contenido, sin desbordamiento horizontal.
La fecha de incorporación se validará al llegar al acto: el código exige
un día no futuro de Madrid y el periodo solicitado empieza el 2 de noviembre.
No se ha modificado ese periodo ni registrado una incorporación anticipada.
La hoja [de preparación de incorporación a cierre](../f1_28_preparacion_pasos_16_21.md)
conserva rutas, perfiles y límites. Estas consultas no acreditan esos actos.

## Paso 12 confirmado tras corregir la sesión del anclaje

Dirección integró la corrección en
`014dc27e8f0fe04c4d983f0ba053cc629001672c`, con dos revisiones favorables
y seis puertas de CI verdes. J avanzó limpia a esa revisión y se compiló
el binario real en 25,10 segundos; SHA-256
`dfd77b9e528904ec85d81d22b3e7659854d925c445e2c06a58029f9f27126aba`.
Se actualizó solo la aplicación. La comprobación de aplicación y
Administración terminó con código 0; PostgreSQL y los seis recibos previos
permanecieron iguales, con versión 6 antes del envío.

A las 13:17:12 de Madrid, el mismo cuerpo y la misma clave del primer intento
obtuvieron 201. Propuesta: `propuesta:64564e88-5fd2-43ee-9192-d9f4bed55273`;
recibo: `recibo:f6a15717-1c54-45b6-9188-6d358762a4d1`; versión resultante 7.
La consulta de solo lectura encontró una única propuesta con esa clave.
Hay tres envíos en total, con resultados 503, 503 y 201. No se han creado
otros antecedentes para repetir el trámite.

La resolución todavía no está registrada. Al preparar su revisión, el
perfil técnico recibe 403 en el panel de Documentos general. Esa lectura
requiere una autorización propia de Documentos; los borradores de
Contratación tienen otras rutas y permisos. Se ha comunicado a dirección
y se comprueba la descarga de Contratación que ofrece la ficha de Gestión,
sin añadir roles ni ampliar la lectura general.
