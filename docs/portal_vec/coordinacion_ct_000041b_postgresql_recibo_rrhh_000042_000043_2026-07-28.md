# Coordinación CT-000041B: contrato PostgreSQL de resultados y Recibo RRHH V2

Fecha: 28 de julio de 2026

Base examinada: `8039d8a`

Ámbito: diseño implementable de las migraciones PostgreSQL `000042` y
`000043`, previo al motor interno `000044`

Estado: especificación de coordinación; no autoriza integración, despliegue
ni acceso productivo

## Propósito

Este corte debe cerrar la parte PostgreSQL pendiente de CT `000041`:

1. reproducir byte a byte los cánones Go de contenido de cuadro y detalle;
2. reproducir la huella del conjunto material VEC-AD-3;
3. reproducir el canon de resultado ya tipado por `000040`;
4. sellar un Recibo de lectura RRHH V2 con los 38 valores exactos;
5. conservar contenido, resultado, acceso, identidad y recibo como una única
   prueba durable;
6. dejar preparados los elementos internos que el motor `000044` ejecutará
   en una sola transacción.

No se crea aquí una fachada ni se concede `EXECUTE` al consultor RRHH. La
fachada nominal deberá quedar después del motor interno.

## Base y dependencia inmediata

La implementación debe partir de la rama estable publicada en `8039d8a`.
Antes de numerar o ejecutar estas migraciones hay que integrar y verificar la
migración `000041` de vocabulario de estados que mantiene otro agente.

La precondición esperada después de esa integración es:

- `control_migracion_cobertura_o4.version_esquema = 21`;
- `control_migracion_consultas_rrhh.version_esquema = 5`;
- restricción validada con los seis estados `pendiente`, `en_curso`,
  `espera_externa`, `completado`, `incidencia` y `cancelado`.

Este trabajo no debe modificar ni sustituir la migración de estados.

## Decisión de partición

No debe forzarse todo el contrato en una sola migración. Los cánones, el
manifiesto estructural, la tabla durable y el `safe-down` superarían el límite
de revisión razonable.

La partición obligatoria es:

- `000042`, barrera `21/5 → 22/6`: tipos privados, cánones de
  contenido, huella material y canon del Recibo V2.
- `000043`, barrera `22/6 → 23/7`: almacenamiento probatorio,
  relaciones exactas y cierre interno durable.
- `000044` futura, barrera `23/7 → 24/8`: motor propietario para
  consumo, lectura, revalidación, acceso, cursor y recibo.
- `000045` futura, barrera por definir al implementarla: fachadas
  nominales y privilegio mínimo.

Esta renumeración desplaza la previsión anterior de motor `000042` y fachadas
`000043`. Debe actualizarse el mapa general solo cuando dirección integre el
corte; este documento no modifica mapa, tablero ni relevo.

Cada fichero de migración, prueba, runner o código deberá permanecer por
debajo de 800 líneas. No se admite eludir la regla mediante líneas
artificialmente largas o SQL comprimido.

## Autoridad y frontera de confianza

Todas las funciones nuevas son internas:

- `SECURITY DEFINER`;
- `search_path = pg_catalog`;
- `row_security = on`;
- `timezone = UTC`;
- `VOLATILE` cuando lean estado, produzcan prueba o persistan;
- límites explícitos de bloqueo y sentencia;
- propietario exacto `vec_contratacion_temporal_propietario`;
- `REVOKE ALL` para `PUBLIC`, migrador y todos los roles de ejecución;
- sin `GRANT EXECUTE` para `vec_contratacion_temporal_consultor_rrhh`.

La entrada exterior nunca puede aportar como autoridad:

- la huella del material VEC;
- la huella del contenido;
- la huella del resultado;
- la huella del cursor;
- el sello del recibo;
- la identidad registrada;
- la secuencia o cabeza de la cadena de accesos.

El futuro motor `000044` recibirá la consulta tipada y las diez piezas
originales VEC-AD-3. PostgreSQL volverá a validar, consumir y derivar el resto.

Toda operación de lectura deberá comprobar:

- `pg_is_in_recovery() = false`;
- transacción `SERIALIZABLE`;
- transacción de lectura y escritura;
- zona horaria efectiva UTC;
- tamaños, tipos y escalas exactos;
- sesión y rol admitidos por el consumidor VEC existente.

La función no abre ni cierra transacciones. El adaptador Go mantendrá abierta
la transacción, reconstruirá y validará el tipo nominal de Recibo V2 con la
salida SQL y solo entonces ejecutará `COMMIT`. Un fallo Go provoca `ROLLBACK`
del consumo VEC, acceso, cursor y prueba.

## Migración 000042: contrato canónico privado

### Funciones previstas

Las firmas recomendadas son:

```sql
canon_contenido_cuadro_rrhh_v1(
    timestamptz,
    vec_contratacion_temporal.publicacion_version_rrhh[],
    boolean,
    bytea
) RETURNS bytea

canon_contenido_detalle_rrhh_v1(
    timestamptz,
    vec_contratacion_temporal.publicacion_version_rrhh,
    jsonb
) RETURNS bytea

huella_material_consumo_rrhh_v3(
    bytea, bytea, bytea, bytea, numeric, numeric,
    bytea, bytea, bytea, bytea
) RETURNS text

canon_recibo_lectura_rrhh_v2(
    vec_contratacion_temporal.evidencia_recibo_lectura_rrhh_v2
) RETURNS bytea
```

Las firmas definitivas deberán congelarse en pruebas de catálogo. No se
permiten sobrecargas abiertas ni variantes que acepten JSON libre.

### Encuadre compartido

Cada campo usa:

```text
longitud_utf8_decimal:bytes
```

seguido de LF. La longitud:

- cuenta octetos, no runas ni caracteres;
- no lleva signo ni ceros iniciales;
- se calcula antes de anexar;
- no permite superar 256 KiB;
- no deja un canon parcial si rebasa el límite.

Los instantes usan exactamente:

```text
2006-01-02T15:04:05.000000Z
```

Los booleanos son `0` o `1`. Los enteros usan decimal canónico. Los valores
ausentes se encuadran como cadena vacía cuando así lo exige Go.

### Contenido del cuadro

Cabecera:

```text
VEC-CT-CONTENIDO-CUADRO-RRHH-V1\n
```

Orden:

1. instante `generada_en`;
2. cardinalidad;
3. por cada fila, los quince campos:
   - `expediente_ref`;
   - `organizacion_ref`;
   - `numero_visible`;
   - `version`;
   - `flujo_ref`;
   - `flujo_version`;
   - `flujo_huella_sha256`;
   - `fase_clave`;
   - `estado_clave`;
   - `centro_ref`;
   - `categoria_ref`;
   - `modalidad_clave` o vacío;
   - `unidad_ref` o vacío;
   - `creado_en`;
   - `actualizado_en`;
4. `hay_mas`;
5. los 32 bytes de la huella SHA-256 del material crudo del cursor, o un
   encuadre binario vacío.

La función debe rechazar:

- más de 100 filas;
- expedientes duplicados;
- orden distinto de `actualizado_en DESC, expediente_ref COLLATE "C" DESC`;
- filas actualizadas después de `generada_en`;
- `hay_mas` sin filas;
- cursor con longitud distinta de 32;
- cursor presente cuando `hay_mas = false`;
- digest nulo;
- canon superior a 256 KiB.

El token Base64URL no entra en el canon. PostgreSQL genera 32 bytes aleatorios,
guarda solo su SHA-256 y devuelve la representación Base64URL una única vez.

### Contenido del detalle

Cabecera:

```text
VEC-CT-CONTENIDO-DETALLE-RRHH-V1\n
```

La función recibe la fila publicada exacta y el `agregado_json` de
`expediente_version_integral` para esa misma pareja expediente/versión.
Antes de extraer datos vuelve a comprobar:

- SHA-256 del JSON canónico almacenado;
- referencia y versión;
- organización y número visible;
- flujo, versión y huella de flujo;
- fase y estado;
- que la fila publicada referencia esa versión integral;
- tamaño, profundidad y nodos dentro de los límites de `000040`.

Orden canónico:

1. los quince campos del resumen;
2. solicitud:
   - grupo/subgrupo;
   - motivo;
   - inicio;
   - fin;
3. máscara decimal de bloques;
4. análisis:
   - presencia;
   - secuencia de la actuación vinculada;
   - modalidad, categoría, causa, periodo, porcentaje y resultado RC;
   - presencia de coste;
   - si existe, céntimos con signo y moneda;
   - fuente del coste;
5. cobertura:
   - presencia;
   - secuencia vinculada;
   - vía, decisión gobernada, procedimiento y bolsa;
   - cardinalidad de comprobaciones;
   - cada clave y resultado;
6. asignación:
   - presencia;
   - secuencia vinculada;
   - unidad, instante y motivo;
7. cardinalidad de actuaciones;
8. por cada actuación:
   - secuencia;
   - versión de expediente;
   - acción;
   - instante;
   - fase origen y destino;
   - estado origen y destino.

No se incluyen actores, personas, documentos, notas, observaciones ni texto
libre. Las secuencias de los bloques deben apuntar a la actuación exacta y
estar entre 2 y la versión del expediente. La cardinalidad de actuaciones
debe coincidir con la versión.

Un detalle exitoso siempre produce `total = 1` y no tiene cursor.

### Resultado común

Cabecera:

```text
VEC-CT-RESULTADO-CONSULTA-RRHH-V1\n
```

La función existente de `000040` se reutiliza sin duplicarla. Recibe:

1. `tipo_consulta`;
2. `generada_en`;
3. `total`;
4. `contenido_huella_sha256`;
5. `cursor_huella_sha256` o vacío.

La huella de resultado es SHA-256 del canon anterior.

### Huella del conjunto material VEC-AD-3

La implementación debe reproducir exactamente
`ExportacionMaterialConsumoAutorizacionAtestadaV3.HuellaConjuntoSHA256`.

Cada bloque se antepone con su longitud como entero sin signo de 64 bits en
orden de red. En PostgreSQL puede usarse `int8send(octet_length(...))`, tras
validar que el valor cabe.

Orden exacto:

1. capacidad canónica;
2. once valores textuales del resumen de capacidad:
   - `decision_ref`;
   - `decision_huella_sha256`;
   - `motivo_huella_sha256`;
   - `contexto_ref`;
   - `contexto_huella_sha256`;
   - `operacion`;
   - `efecto_ref`;
   - `efecto_huella_sha256`;
   - `audiencia_consumo`;
   - `emitida_en` en RFC3339Nano canónico;
   - `expira_en` en RFC3339Nano canónico;
3. decisión canónica;
4. motivo canónico;
5. contexto de actor canónico;
6. `persona_version` como ocho bytes big-endian;
7. `perfil_version` como ocho bytes big-endian;
8. payload VEC-AD-3;
9. sobre COSE Sign1;
10. evidencia de verificación;
11. raíz pública SPKI.

La huella final es SHA-256 del conjunto encuadrado. No debe aceptarse como
parámetro una huella calculada por Go.

### Canon del Recibo RRHH V2

Cabecera:

```text
VEC-CT-RECIBO-LECTURA-RRHH-V2\n
```

El tipo compuesto privado
`evidencia_recibo_lectura_rrhh_v2` debe contener, en este orden:

1. esquema exacto del registrador;
2. `acceso_ref`;
3. secuencia;
4. huella anterior;
5. huella de cadena del acceso;
6. huella del vínculo de identidad;
7. huella de alcance, vacía en detalle;
8. instante registrado;
9. referencia de auditoría VEC;
10. huella de auditoría VEC;
11. huella de consumo VEC;
12. referencia de decisión;
13. huella de decisión;
14. huella de capacidad;
15. huella del conjunto material;
16. huella de consulta;
17. correlación;
18. referencia de autenticación;
19. huella de autenticación;
20. sesión;
21. control de sesión;
22. revisión del control de sesión;
23. huella del control de sesión;
24. actor;
25. perfil;
26. versión de perfil;
27. organización;
28. clase de ámbito;
29. referencia de ámbito;
30. acción;
31. finalidad;
32. expediente, vacío en cuadro;
33. versión, cero en cuadro;
34. total;
35. huella del contenido;
36. huella del resultado;
37. huella del cursor;
38. instante de generación.

El discriminador del registrador es exactamente:

```text
vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2
```

El sello es SHA-256 de este canon. No forma parte de su propia preimagen.

## Migración 000043: prueba durable y cierre interno

### Tabla probatoria

Crear `prueba_resultado_recibo_rrhh_v2`, append-only, con:

- acceso como clave primaria;
- tipo de consulta;
- instante generado;
- total;
- expediente y versión;
- canon y huella de contenido;
- huella de cursor nullable;
- canon y huella de resultado;
- huella del conjunto material;
- instante de revalidación final;
- canon y sello del recibo;
- los valores probatorios necesarios para recalcular el Recibo V2.

La tabla debe tener:

- límites de 256 KiB para contenido y recibo;
- comprobaciones SHA-256 no nulas;
- `CHECK` que recalcule contenido, resultado y recibo;
- relaciones exactas con acceso, identidad y alcance;
- triggers de rechazo para `UPDATE`, `DELETE` y `TRUNCATE`;
- RLS habilitada y forzada;
- política solo para el propietario;
- cero permisos para roles de ejecución.

### Relaciones exactas

Añadir a `registro_acceso_rrhh` una clave candidata
`UNIQUE NULLS NOT DISTINCT` que incluya como mínimo:

- acceso;
- tipo;
- expediente;
- versión;
- total;
- huella de resultado;
- instante registrado.

La tabla probatoria debe referenciar esa clave. También debe referenciar:

- acceso y huella de prueba de
  `vinculo_identidad_acceso_rrhh_v2`;
- para cuadro, acceso y huella de `alcance_acceso_rrhh`;
- para detalle, un par alcance/acceso de alcance completamente nulo mediante
  `MATCH FULL`.

No basta una clave foránea únicamente a `acceso_ref`: debe ser imposible
asociar un resultado o identidad perteneciente a otra lectura.

### Primitiva de cierre

La primitiva interna de `000043`:

1. recibe material VEC original y material de contenido tipado creado por el
   futuro motor;
2. nunca recibe huellas autoritativas de contenido, resultado, material o
   recibo;
3. recalcula esas huellas;
4. exige la evidencia de un consumo nuevo obtenida en la misma ejecución del
   motor;
5. ejecuta la revalidación final VEC;
6. llama a `registrar_acceso_rrhh_interno_v2`;
7. relee acceso, vínculo de identidad y alcance ya persistidos;
8. construye los 38 valores desde esas fuentes;
9. calcula e inserta el Recibo V2;
10. devuelve una salida tipada:
    - los siete valores del registrador;
    - auditoría y consumo VEC;
    - huellas de contenido, resultado y cursor;
    - instante, expediente, versión y total;
    - sello del recibo.

Esta primitiva no se concede a la fachada. Solo el motor propietario `000044`
puede invocarla después de consumir autorización y leer el resultado.

## Flujo transaccional futuro 000044

El motor debe ejecutar, en este orden:

1. validar consulta, alcance, contexto y diez piezas VEC;
2. consumir una sola vez el wrapper nominal de cuadro o detalle;
3. exigir `consumo_nuevo = true`;
4. para una página posterior, bloquear el cursor con `FOR UPDATE` y comprobar
   familia, identidad, ámbito, filtro, vigencia, revocación y no consumo;
5. capturar `control_publicacion_rrhh.ultimo_corte`;
6. materializar una única colección de filas;
7. calcular contenido y respuesta desde esa misma colección;
8. generar cursor nuevo si procede;
9. fijar `generada_en` a microsegundo después de la lectura;
10. revalidar el consumo VEC;
11. registrar el acceso;
12. insertar familia, emisión o consumo de cursor con sus FKs diferidas;
13. insertar la prueba y sellar el Recibo V2;
14. devolver material tipado al adaptador Go;
15. permitir `COMMIT` solo si Go valida el recibo contra contexto, capacidad,
    orden y contenido.

Un fallo en cualquier punto revierte también el consumo VEC.

### Semántica de corte y página

Para el cuadro:

- elegir primero, para cada expediente, la última versión cuyo
  `corte_global <= corte_capturado`;
- aplicar alcance y filtros después de elegir esa última versión;
- ordenar con colación `C` por
  `actualizado_en DESC, expediente_ref DESC`;
- aplicar el cursor como keyset, nunca como desplazamiento;
- obtener `limite + 1`;
- publicar solo las primeras `limite`;
- usar la fila adicional exclusivamente para `hay_mas`.

No se permite consultar una vez para el DTO y otra para el canon.

### Cursores

Primera página sin continuación:

- `familia_ref` nula;
- sin token ni huella de cursor.

Primera página con continuación:

- derivar la familia de acceso, consulta y corte;
- crearla con `creada_en = registrada_en`;
- emitir cursor de página 2;
- generar 32 bytes mediante `public.gen_random_bytes(32)`;
- guardar solo SHA-256;
- devolver Base64URL sin relleno una sola vez.

Página posterior:

- consumir una sola vez el token presentado;
- `consumido_en = registrada_en`;
- si existe otra página, emitir el cursor hijo;
- ligar el hijo al consumo del padre;
- no registrar nunca el token crudo.

### Detalle

El detalle consulta exactamente la versión observada y comprueba el alcance
sobre esa versión. Un resultado válido siempre tiene:

- expediente no vacío;
- versión mayor o igual que uno;
- total uno;
- cursor vacío;
- alcance de cuadro vacío.

La ausencia y la denegación no pueden producir un falso Recibo V2 con
`total = 0`, porque Go lo rechaza. Deben usar una vía separada de auditoría de
denegación o ausencia y dar al exterior un error indistinguible.

## Safe-down

Cada reversión debe:

1. adquirir el advisory lock de migraciones RRHH;
2. comprobar las dos barreras exactas;
3. bloquear sus controles y tablas con `ACCESS EXCLUSIVE`;
4. comprobar funciones, tipos, tablas, columnas, propietarios y ACL;
5. comprobar restricciones, índices, triggers, RLS y políticas;
6. comprobar reglas, herencia, publicaciones, estadísticas extendidas y
   etiquetas de seguridad;
7. verificar la huella de las definiciones y del catálogo semántico;
8. rechazar dependencias futuras;
9. usar exclusivamente `DROP ... RESTRICT`;
10. retroceder barreras solo tras retirar correctamente los objetos.

El `down` de `000043` debe fallar si existe una sola prueba o recibo. Nunca
borra trazabilidad para facilitar una reversión.

Las reentradas de `up` y `down` deben fallar sin mutar el estado.

## Vectores Go y PostgreSQL

La prueba debe comparar bytes completos en hexadecimal y después SHA-256. No
basta comparar únicamente la huella.

Vectores ya congelados:

- resultado de cuadro vacío:
  `cb8ad45d7c31faa5100a249a840e66671b0de2319d23fc1c8878e56da7076ee0`;
- resultado de detalle mínimo:
  `0b7d78f6d34cd87f3da98fc32a0830ba953c83f36c2a7423fba9810baff78e31`;
- digest de cursor con 32 bytes `FF`:
  `af9613760f72635fbdb44a5a0a63c39f12af30f950a6ee5c971be188e89c4051`.

La imagen PostgreSQL fijada es:

```text
postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296
```

Añadir vectores para:

- cuadro de una fila sin cursor;
- cuadro de una fila con cursor;
- detalle con análisis, cobertura y asignación;
- coste negativo;
- Unicode multibyte;
- límite exacto y exceso de 256 KiB;
- conjunto material VEC completo;
- Recibo V2 de génesis, secuencia uno y anterior nulo;
- Recibo V2 encadenado, secuencia dos;
- cuadro con alcance;
- detalle sin alcance.

Cada uno debe construirse independientemente en Go y SQL. La prueba de Recibo
debe mutar individualmente los 38 campos y obtener rechazo.

## Matriz adversarial mínima

### Autoridad y privilegios

- `PUBLIC` sin uso ni ejecución;
- roles runtime sin DML ni `EXECUTE`;
- propietario y políticas exactos;
- RLS forzada incluso con `row_security` alterada;
- rechazo de inserción, actualización, borrado y truncado directos;
- envenenamiento de `search_path`;
- propietario o firma de función alterados.

### Transacción y consumo

- aislamiento inferior a `SERIALIZABLE`;
- transacción de solo lectura;
- réplica o recuperación;
- replay VEC con `consumo_nuevo = false`;
- revocación entre lectura y revalidación final;
- error después del consumo: cero consumos, accesos, cursores y pruebas tras
  `ROLLBACK`;
- timeout y orden de bloqueos;
- dos consumos concurrentes de la misma capacidad.

### Resultado y ámbito

- cruce de organización;
- cruce de centro;
- cruce de unidad;
- versión de detalle distinta;
- filtro aplicado antes de seleccionar última versión;
- fila posterior al corte;
- orden incorrecto;
- expediente duplicado;
- más de cien filas;
- contenido generado antes de la última actualización;
- consulta repetida para respuesta y canon;
- detalle ausente y no autorizado sin oráculo diferenciador.

### Cursor

- Base64URL no canónico;
- longitud distinta de 32 bytes;
- token falso;
- token consumido;
- token caducado;
- familia revocada;
- familia, filtros, identidad o ámbito distintos;
- dos consumos concurrentes;
- hijo sin consumo del padre;
- aparición del token crudo en tabla, error o log.

### Cánones y tiempos

- Unicode contado como caracteres y no bytes;
- longitud con ceros iniciales;
- entero con signo indebido;
- instante sin seis microsegundos o sin UTC;
- digest nulo;
- canon vacío;
- 256 KiB exactos y un byte de exceso;
- contenido, resultado, material o sello mutados;
- campo de Recibo omitido, reordenado o sustituido;
- esquema V1, vacío, en mayúsculas o ajeno;
- resultado anterior a la orden;
- borde exacto igual al instante de orden.

### Reversión

- barrera futura;
- dependencia futura;
- índice, regla, herencia, publicación, estadística o etiqueta añadidos;
- propietario, ACL, RLS, política, trigger o definición alterados;
- `down` con prueba durable existente;
- ciclo `up → down → up`;
- reentrada sin cambio de estado.

## Riesgos bloqueantes

1. **Huella material ausente.** PostgreSQL todavía no reproduce
   `HuellaConjuntoSHA256`, pero Recibo V2 la necesita. No puede sustituirse por
   la huella de capacidad.
2. **Detalle sin resultado.** El registro histórico admite `total = 0` en
   algunos casos, mientras Recibo V2 Go exige uno. Hace falta una vía separada
   para ausencia y denegación.
3. **Relojes Go y PostgreSQL.** Si Go adelanta su orden respecto al instante
   SQL, el recibo se rechazará. El adaptador debe validar antes del `COMMIT`.
   Sistemas debe mantener NTP; es preferible obtener el instante de orden de
   PostgreSQL al abrir la transacción. No se introduce tolerancia silenciosa.
4. **Doble lectura.** Reconsultar para construir el DTO después de sellar
   permite discrepancias. Una sola colección materializada alimenta ambos.
5. **Cierre expuesto.** Conceder la primitiva de `000043` al consultor
   permitiría saltarse el motor. Solo `000045` expondrá fachadas nominales.
6. **Cursor en claro.** Solo puede existir en memoria durante una respuesta.
   No entra en tablas, cánones, errores ni logs.
7. **Safe-down destructivo.** Una reversión que elimine recibos viola la
   trazabilidad. Con evidencia, debe fallar.
8. **Colisión de numeración.** No iniciar `000042` hasta confirmar la
   integración de `000041` y sus barreras `21/5`.

## Puertas de aceptación

Antes de entregar cada migración:

- runner PostgreSQL 18.4 fijado por digest;
- instalación limpia;
- reentrada adversarial;
- `up → down → up`;
- comparación Go/PostgreSQL byte a byte;
- matriz adversarial completa;
- comprobación de catálogo, propietarios, ACL y RLS;
- `git diff --check`;
- validador Markdown;
- Gitleaks del commit;
- conteo de líneas inferior a 800 por fichero;
- revisión independiente distinta del autor.

Superar estas puertas cierra el contrato probatorio, pero no autoriza
producción. Producción continúa en `NO-GO` hasta disponer de motor, fachadas,
adaptador PostgreSQL Go, composición raíz y pruebas E2E reales.
