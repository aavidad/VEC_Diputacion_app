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

## Configuración

CONFIG NUEVA: `VEC_RRHH_AUDITORIA_ENABLED=true` y
`VEC_AUDITORIA_CONSULTA_CATALOGO_PATH=/vec-incorporacion/auditoria_consulta.v2.json`.

Copiar `data/catalogos/auditoria/consulta.v2.json` a esa ruta antes de arrancar.
El fichero contiene el motivo configurable, sin claves ni identidades personales.
El lector admite versiones posteriores; la referencia y la huella del motivo
deben coincidir con su publicación vigente.

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
