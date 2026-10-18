# Habilitar la consulta de auditoría de RRHH

Desde Bolsas → Auditoría y control → Auditoría y trazabilidad, RRHH puede
seleccionar un expediente, una bolsa constituida o una persona vinculada a
Personal, indicar el motivo y consultar su historia por fechas. Cada búsqueda,
consulta y lectura de opciones registra quién consulta, su perfil y su finalidad.
La lectura y su asiento se confirman en la misma transacción.

## Instalación

Aplicar una vez `lista_sql_auditoria_rrhh.txt`, en este orden:

1. AUT82: cotejo del permiso actual de consulta.
2. CT208: búsqueda e historia del expediente bajo su organización.
3. B109: búsqueda e historia de las bolsas constituidas.
4. P47: búsqueda e historia de las personas vinculadas a Personal.
5. AD251: asiento común de lectura, filtros y motivo conservados; consulta paginada.

La lista parte de la principal posterior a AD249. La preimagen del CHECK común
es `1a299eefbc2936a06b414d47defbb17b213fec7a29f2fa6eec448a008727d8ba`.
La instalación no crea LOGIN, no asigna permisos y no cambia la historia existente.
Con historia nueva no aplicar DOWN ni repetir estas SQL. La vuelta atrás del kit
de dirección repone conjuntamente la copia fría de base, configuración y binario.

F0-01 añade AD285 después de AD251, mediante
`lista_sql_codexb_f0_01_20261015.txt`. Corrige únicamente los paréntesis de las
extracciones JSON al construir la huella del filtro. Exige la función instalada
con su preimagen exacta y conserva propietario, permisos y demás metadatos.
No modifica el núcleo ni el CHECK común de auditoría. Se sirve con el código de
este corte: el preflight comprueba el aislamiento de Bolsa sin pedir acceso a su
esquema y el cargador admite el catálogo operativo.

## Configuración

CONFIG NUEVA: `VEC_RRHH_AUDITORIA_ENABLED=true` y
`VEC_AUDITORIA_CONSULTA_CATALOGO_PATH=/vec-incorporacion/auditoria_consulta.v2.json`.

Copiar `data/catalogos/auditoria/consulta.v2.json` a esa ruta antes de arrancar.
El fichero contiene el motivo configurable, sin claves ni identidades personales.
Esta instalación exige la versión 2 del catálogo; la referencia y la huella
deben coincidir con su publicación vigente.

El catálogo v2 declara su predecesor y comienza en revisión 1. Antes de activar,
comprobar la fuente de motivos de autorización: si falta la versión 1, hay que
provisionarla primero desde su fuente histórica mediante
`vec_autorizacion.publicar_motivos_autorizacion_v2`, con secuencia y referencia
de evento propias. La función instalada exige esa versión anterior; no basta
con copiar el fichero v2. Cada versión conserva su huella, sus entradas y su
evento. Una versión ya publicada con otra huella requiere una decisión de
dirección; este guion no la sobrescribe.

Conservar los LOGIN segregados ya preparados y declarar sus DSN privados:

- `VEC_RRHH_AUDITORIA_FUENTE_AUTORIZACION_DATABASE_URL`: LOGIN miembro únicamente
  de `vec_autorizacion_fuente`.
- `VEC_RRHH_AUDITORIA_MOTIVOS_DATABASE_URL`: LOGIN miembro únicamente de
  `vec_autorizacion_motivos_evaluador`.
- `VEC_RRHH_AUDITORIA_FRONTERA_DATABASE_URL`: LOGIN miembro únicamente de
  `vec_contratacion_temporal_registrador_auditoria`.

La consulta usa el LOGIN de consultas RRHH de CT ya existente,
`vec_contratacion_temporal_consultor_rrhh`. No reutilizar LOGIN de propietario,
fuente, motivos, ejecución o administración para ese pool. Los tres DSN anteriores
no comparten LOGIN entre sí ni con otros pools configurados.

`VEC_AUDITORIA_CONSULTA_EXPEDIENTE_CT` y
`VEC_AUDITORIA_CONSULTA_EXPEDIENTE_BOLSA` dejan de ser necesarios. La composición
no los lee ni publica material de consumo V3 para estas lecturas.

## Provisión por Administración

Usar las autoridades ADMIN existentes, con los dos operadores distintos y sus
huellas/CAS actuales. El arranque no publica ni sustituye asignaciones o roles.

1. Incorporar al catálogo gobernado de acciones el descriptor
   `vec.auditoria.consultar`: módulo `auditoria`, recurso `historial_auditoria`,
   finalidad `consulta_auditoria_rrhh`, garantía `alto`, obligación `auditar` y los
   18 campos exactos de `auditoria_rrhh_definicion_rol.json`.
2. El descriptor admite exactamente `organizacion_ref`, `unidad_ref` y
   `ambito_ref`, cada uno con un valor. Tomar la organización de CT/Personal y la
   unidad y el ámbito de la asignación RRHH de Bolsa ya publicada. No deducirlos
   del nombre del cargo, del expediente elegido ni de una participación.
3. Publicar el catálogo con `registrar_catalogo_acciones_admin_v1` y
   `aplicar_catalogo_acciones_admin_v1`. Con su cabeza actual, proponer el rol
   `consulta_auditoria_rrhh` mediante `/gobierno-roles/propuestas`; otro ADMIN
   confirma `/gobierno-roles/cierres`. El JSON adjunto es la definición; las
   referencias, versiones y huellas del catálogo/propuesta se obtienen de las
   lecturas nominales existentes, no de valores inventados.
4. Clasificar el rol como asignable mediante la autoridad de perfiles existente
   y asignarlo al perfil nominal de auditoría de la cuenta RRHH. Conservar la
   asignación anterior; cualquier sustitución necesita su huella y CAS actuales.
5. Reiniciar la aplicación tras la provisión para renovar el contexto esperado.
   No conceder permiso CT de escritura ni permiso de administración al auditor.

El perfil nominal conserva el identificador que ya genera
`discriminadorContextoAuditoriaConsultaDesarrollo("ct")`. Para obtenerlo sin
crear otra autoridad se reutiliza `referenciaAltaContratacionTemporalDesarrollo`:
prefijo `prf_`, material `principal_id + NUL + certificate_sha256 + NUL +
"perfil-auditoria-consulta-ct-v1"`. La función calcula SHA256 de
`"vec.ct.alta.desarrollo.v1" + NUL + material` y toma los primeros 16 bytes
hexadecimales. El certificado y la identidad son los de la cuenta RRHH existente.

Bolsa se consulta con unidad y ámbito RRHH, conforme a la decisión de dirección
del 13/10 a las 03:15. No se exige contrato ni adscripción inventada para consultar
una bolsa constituida. CT y Personal cotejan su organización en cada proyección.

## Comprobación y límites

### Activación en una copia local (F0-01)

Esta tarea reutiliza la consulta y la definición del rol existentes. Restaurar
el respaldo en un PostgreSQL 18 propio, con sus ACL, y comprobar primero la
presencia de AUT82, CT208, B109, P47 y AD251. Si ya están instaladas, no se
ejecuta otra vez `lista_sql_auditoria_rrhh.txt`.

1. Conservar el material privado del respaldo como solo lectura. Preparar otra
   carpeta para la copia, sus certificados, conexiones y resultados.
2. Comprobar el arranque con la auditoría apagada. Guardar el resultado de
   salud y una consulta habitual de RRHH antes de activar nada.
3. Preparar el catálogo v2 y los tres LOGIN separados de esta guía. El rol de
   consulta tiene exactamente los 18 campos del JSON adjunto y tres ámbitos:
   organización de CT/Personal, unidad y ámbito de la asignación de Bolsa.
   AUT82 vuelve a comprobar los tres; no se añaden centro ni categoría.
   El perfil necesita además un vínculo corporativo vigente y aceptado por
   CA40 para la misma cuenta, persona, perfil y ContextoActor, con sus versiones
   actuales. La asignación del rol por sí sola no crea ese vínculo.
   Mantener también la configuración de auditoría de intentos, incluidos sus
   archivos `.dsn`, en la misma copia y con su CA. Bolsa necesita una
   configuración SMTP completa aunque este ensayo solo haga consultas; el
   buzón local debe comprobar TLS y no reenviar fuera de la copia.
4. Activar la configuración completa y reiniciar el proceso de la copia.
   Con el certificado de RRHH, abrir Bolsas → Auditoría y control → Auditoría
   y trazabilidad. Seleccionar Persona, buscar una persona de Personal,
   indicar el motivo y un intervalo de hasta 31 días y consultar.
5. Conservar una lectura y un acto de esa persona. La consulta devuelve su
   referencia de auditoría; una segunda consulta permite comprobar también
   el registro de la primera. Una referencia ajena no devuelve datos.
6. Reiniciar la aplicación y PostgreSQL. Recuperar los mismos hechos y
   comprobar que permanecen las referencias de auditoría anteriores. Volver
   a apagar el selector y comprobar la salud y la consulta habitual de RRHH.

Los datos ficticios preparados directamente en la copia sirven para ensayar la
consulta. No acreditan la publicación del catálogo, el rol o la asignación por
Administración. Esa activación conserva el procedimiento de dos personas
administradoras descrito arriba. Los certificados, conexiones y respuestas
completas se guardan fuera de Git.

En el ensayo F0-01 del 16/10, el respaldo carecía del vínculo del perfil de
auditoría. Se preparó como dato ficticio de la copia, con 90 días de vigencia,
sin cambiar el vínculo de Bolsa. No se acreditó su publicación por el gobierno
común: esa provisión sigue siendo un requisito de cualquier activación fuera
de la copia. Chrome mostró el mismo acto y recibo tras reiniciar aplicación y
PostgreSQL y recuperó tres lecturas anteriores. Intervención recibió 403 y su
denegación quedó registrada. Con auditoría apagada, portal, sesión y `/livez`
respondieron 200; las rutas de auditoría quedaron cerradas con 404.

### Ensayo anterior de las proyecciones

La copia actual de `principal.sql` del 13/10 se restauró completa en PostgreSQL 18
propio, con 2 GB y datos en disco, en 85,7 s. Las cinco SQL se instalaron en
0,456 s; no se reaplicó SQL ni se ejecutó DOWN. El clon conserva 86 expedientes y
16 bolsas. Las búsquedas propietarias devolvieron 21 expedientes paginados, 16 bolsas y una persona. EXPLAIN ANALYZE de historia: CT 2,379 ms, Bolsa 4,737 ms y Persona 1,002 ms. Son proyecciones SQL, no tiempos de pantalla ni una lectura nominal positiva completa.

Dirección indicó a las 04:25 que hará la instalación y el recorrido Chrome en
cidonia mediante su kit con copia fría y vuelta atrás. Comprobar allí:

- Apertura del menú sin ficha previa; GET de opciones 200 con permiso propio.
- Buscar y consultar CT, Bolsa y Persona, con motivos y fechas distintos.
- Un asiento nominal por operación y ausencia de datos si falta el permiso.
- Paginación, cambio de fuente, vacío, error y recuperación desde la primera página.
- Historial anterior intacto y tiempos reales: pantalla <300 ms, base <100 ms.

Persona muestra y busca su referencia cuando falta un lector nominal de nombre.
No se descifran nombres usando un permiso de otra faceta. Los nombres cifrados
ya tienen lector e índice propios; montarlos requiere su autorización específica.

La nueva familia `lectura_auditoria_rrhh` participa en la cadena SQL V5. El
exportador/verificador offline actual todavía no representa esta familia: no
afirmar verificación offline completa de un corte que la incluya ni omitir filas.
Los asientos comunes antiguos sin vínculo demostrable de organización no se
atribuyen a un sujeto por coincidencia de referencia; su historia de negocio se
obtiene de CT, Bolsa o Personal, cada uno bajo su propietario.
