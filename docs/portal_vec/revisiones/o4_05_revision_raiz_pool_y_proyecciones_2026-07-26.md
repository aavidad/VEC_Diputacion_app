# O4-05: revisión de raíz, pool y proyecciones RRHH

Fecha: 26 de julio de 2026.

## Resultado de dirección

La cápsula de ciclo de vida obtuvo `GO` y se publicó en `69b6a14`. La primera
versión de la acreditación del pool PostgreSQL y la primera versión de las
proyecciones de cuadro y detalle recibieron `NO-GO` independiente. Las
proyecciones corregidas superaron la reauditoría y se publicaron en
`d6d1305`. La acreditación PostgreSQL corregida obtuvo `GO` técnico y se
publicó en `d307b42` y `d3d6a04`; continúa en `NO-GO` productivo hasta
acreditar la matriz TLS viva. Ningún corte incrementa por sí solo el porcentaje
funcional.

El contrato corregido está en
[proyecciones RRHH](../proyecciones_rrhh_cuadro_detalle_contratacion_2026-07-26.md).

## Cápsula de aplicación: GO

Quedaron acreditados:

- propietario atómico y exclusivo de servidor y recursos;
- ausencia de registro global que retenga pools o material sensible;
- apagado limitado por contexto;
- cierre inverso, idempotente y saneado;
- rechazo de copia, nulo tipado, duplicado y doble propietario;
- ausencia de cierre de recursos durante una escucha activa;
- constructor privado y producción todavía cerrada.

Evidencia:

- `go test -count=1 ./internal/app/composicion/interna`;
- `go test -race -count=1 ./internal/app/composicion/interna`;
- `go vet ./internal/app/composicion/interna`;
- cincuenta repeticiones focales con detector de carreras;
- `gofmt` y `git diff --check`.

## Pool lector O4-05: NO-GO inicial

La implementación inicial validaba la configuración estática y una conexión
física, la devolvía al pool y ejecutaba después sobre otra conexión posible.
Esto no ligaba la evidencia TLS, identidad y ACL a la operación real.

Además:

- `BeforeConnect` podía cambiar host, TLS o usuario por conexión;
- `pg_stat_ssl.ssl` confirmaba cifrado, pero no `verify-full`;
- el runner real usaba socket Unix sin TLS y no acreditaba la ruta productiva.

Condiciones de nueva revisión:

1. adquirir y acreditar la configuración efectiva y los catálogos en cada
   operación;
2. comenzar la transacción en esa misma conexión;
3. liberar exactamente una vez tras `COMMIT` o `ROLLBACK`, sin cerrar el pool;
4. rechazar callbacks capaces de alterar conexión, TLS, identidad o sesión;
5. probar el runner actual y añadir una matriz TLS PostgreSQL con CA y
   certificado de ensayo antes de autorizar producción.

## Pool lector O4-05 corregido: GO técnico

Los cortes `d307b42` y `d3d6a04` cierran los bloqueos técnicos de la lectura y
acreditan:

- un pool privado, sellado y con propietario explícito, sin registros
  globales, finalizadores ni recursos retenidos;
- rechazo de callbacks capaces de modificar destino, resolución, TLS,
  identidad, protocolo o vigilancia de contexto;
- una conexión física única desde la acreditación hasta `COMMIT` o
  `ROLLBACK`, liberada exactamente una vez;
- una única sentencia y fotografía PostgreSQL para comprobar TLS vivo,
  primario, usuarios, roles, membresías, autoridad, ACL e identidad completa
  de la función antes de invocarla;
- huellas SHA-256 del cuerpo y de la definición canónica de la función,
  recalculadas desde la migración `000035`;
- rechazo de una sustitución hostil mediante `CREATE OR REPLACE FUNCTION` que
  conserva OID y metadatos, seguido de restauración exacta;
- fallo cerrado y recuperación tras revocar y restaurar permisos y
  membresías.

La revisión independiente emitió `GO` para el corte técnico y el runner por
socket Unix. Pasaron PostgreSQL 18.4 real, pruebas del módulo, detector de
carreras con etiqueta de integración, `go vet`, formato y comprobación del
diff. No quedó ningún contenedor O4-05 residual.

Este `GO` no autoriza producción. Falta conservar evidencia viva de:

1. CA y SAN/hostname válidos con `verify-full`;
2. hostname incorrecto;
3. CA desconocida;
4. intento de degradación de TLS o cifrado;
5. rutas alternativas seguras e inseguras.

## Proyecciones RRHH: NO-GO inicial

La primera versión era hexagonal, no serializaba contexto o capacidad y
saneaba errores, pero no podía aceptarse porque:

- no validaba organización, centro o unidad en cada fila;
- no comprobaba filtros ni orden estable en el resultado;
- aceptaba `version_observada` sin cotejarla con el detalle;
- el DTO omitía datos operativos necesarios de solicitud, análisis, cobertura
  y asignación;
- serializaba referencias internas de lectura y auditoría;
- no revalidaba contexto y capacidad en el instante durable del recibo;
- el ámbito de organización no exigía su referencia exacta;
- la capacidad no tenía una duración máxima.

Condiciones de nueva revisión:

1. validar página y detalle contra la orden sellada completa;
2. ligar cada fila a organización y ámbito;
3. revalidar vigencia en el instante durable de lectura;
4. aplicar filtros y orden en la segunda barrera de aplicación;
5. usar DTO explícitos suficientes y minimizados;
6. mantener fuera actores, contactos, DNI, responsables, notificaciones,
   documentos, textos libres y recibos internos;
7. probar cruces de organización/ámbito, versión obsoleta, caducidad,
   serialización, orden y clonación defensiva.

## Proyecciones RRHH corregidas: GO técnico

El corte `d6d1305` incorpora:

- contexto, capacidad y órdenes opacas, no serializables y ligados a actor,
  sesión, perfil, organización, ámbito, acción, finalidad y vigencia;
- cuadro con filtros, orden, paginación, ámbito y recibo durable revalidados;
- detalle minimizado de solicitud, análisis, coste, cobertura, comprobaciones,
  asignación e hitos, sin actores, contactos, DNI, documentos ni textos libres;
- recibo interno excluido del JSON y no serializable;
- máscara y vínculos privados de fases;
- huella SHA-256 canónica privada de todo el contenido público del detalle,
  comparada en tiempo constante para rechazar cualquier mutación posterior.

La revisión independiente reprodujo pruebas focales, detector de carreras,
`go vet`, comprobación del diff y cien repeticiones adversarias. El `GO`
acredita el contrato en memoria de `ports/application`; no acredita todavía
PDP, identidad, PostgreSQL, HTTP ni E2E productivos.

## Composición visual gobernada: GO técnico

El corte `4714088` mantiene la interfaz existente fuera del dominio y publica
solo una composición neutral formada por claves i18n, flujos, fases, tareas,
paneles, campos, catálogos versionados y capacidades visuales informativas.
No incorpora roles, PII, etiquetas ni opciones funcionales fijas.

Tras un primer `NO-GO` se cerraron:

- límites por tarea y globales antes de reservar memoria;
- huellas canónicas de flujo y catálogos;
- huella global y recibo privado;
- atestación obligatoria por una autoridad de publicaciones independiente de
  la fuente de composición;
- orden único de paneles;
- colecciones JSON normalizadas a `[]`.

La prueba adversaria sustituye operaciones y opciones, recalcula huellas y
recibo bajo la misma versión y demuestra que la autoridad gobernada lo
rechaza. Otra entrada usa un millón de referencias y se rechaza antes de
reservar. Focales, detector de carreras, `go vet`, formato y cien repeticiones
obtuvieron `GO` independiente.

Este corte no acredita aún la independencia física del adaptador de
publicaciones ni raíz, HTTP, web conectada o E2E.

## Fundamento VEC-AD-3 para consultas: GO técnico

El corte `2a9ddb1` añade a la autoridad común un resumen tipado, minimizado y
no autoritativo de la capacidad VEC-AD-3. La implementación concreta lo deriva
del mismo documento canónico que ya valida de forma estricta; los módulos no
reinterpretan el JSON criptográfico privado.

El resumen:

- liga decisión, motivo, contexto, operación, efecto, audiencia y ventana;
- no contiene nonce, MAC, claves ni estado de gobierno;
- bloquea JSON, texto, binario, gob, CBOR, YAML, XML, formatos y registros;
- no sustituye la exportación AD-3 ni concede acceso por sí mismo;
- conserva como única autoridad la verificación y el consumo de los bytes
  originales dentro de la transacción PostgreSQL.

Pruebas focales, matriz de codecs y registros, detector de carreras,
`go vet`, formato y revisión independiente obtuvieron `GO` para este
fundamento aislado. No corrige todavía `CapacidadConsultaRRHH`, no cierra C2 y
no incrementa la métrica funcional.

## Migración de consultas RRHH a VEC-AD-3: GO técnico

La revisión independiente concedió `GO` para integrar el contrato Go de
consultas RRHH V3 y mantuvo expresamente el `NO-GO` productivo. La primera
propuesta fue rechazada porque cuadro y detalle compartían audiencia y porque
el material probatorio podía exportarse antes de ligarse a la consulta
funcional exacta.

El candidato corregido:

- separa acción, audiencia y tipo de recurso de cuadro y detalle;
- deriva el contexto únicamente de la autoridad V3 ligada al canal;
- conserva y coteja las diez piezas VEC-AD-3 sin reinterpretar su JSON;
- calcula el comienzo de vigencia como máximo y el final como mínimo de todas
  las fuentes autoritativas;
- liga capacidad y recibo a decisión, conjunto probatorio, consulta, ámbito,
  sesión, finalidad, expediente y versión;
- solo permite exportar hacia SQL desde una orden nominal que recalcula la
  huella funcional;
- bloquea serialización, formato y registro accidental de material sensible.

La matriz independiente repitió:

```text
go test -count=1 ./internal/modules/contrataciontemporal/ports \
  ./internal/modules/contrataciontemporal/application
go test -race -count=1 ./internal/modules/contrataciontemporal/ports \
  ./internal/modules/contrataciontemporal/application
go vet ./internal/modules/contrataciontemporal/ports \
  ./internal/modules/contrataciontemporal/application
```

También se probaron conjuntamente los puertos VEC y el adaptador concreto de
confianza atestada. Formato y `git diff --check` quedaron verdes. Las pruebas
hostiles rechazan cruces de audiencia, acción, recurso, referencia, huella y
caducidad, además de comprobar que material y capacidad aislados no tienen una
API pública de exportación SQL.

El riesgo residual es deliberado y visible: aún no existe el consumidor C2 de
PostgreSQL. Este deberá recibir la orden nominal, revalidar el tiempo en la
base de datos y consumir, leer y auditar en una única transacción.

## Métrica

El procedimiento permanece en `19/46` tareas verificadas, un 41 %. O4-05
permanece en tres de cinco hitos internos hasta completar raíz con dependencias,
proyecciones protegidas y E2E HTTP real.
