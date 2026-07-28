# Coordinación CT-000041B: contrato PostgreSQL de resultados y Recibo RRHH V2

Fecha: 28 de julio de 2026

Base examinada inicialmente: `8039d8a`

Última corrección de contrato: `b116f87`

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

La amplitud nominal de `000042` exige una partición física, no otra migración.
El fichero principal es el único que coincide con el patrón numerado y
mantiene `BEGIN`, las dos barreras, el avance final y `COMMIT`. Incluye, en un
orden literal congelado, componentes bajo `000042_componentes/` mediante
`\ir`. Los componentes no contienen transacciones, barreras ni inclusiones
dinámicas. Por tanto, todos los tipos, auxiliares, cánones, ACL y controles se
instalan o revierten como una sola unidad.

El ejecutor operativo de este conector PostgreSQL es `psql` 18.4 con `-X`,
`ON_ERROR_STOP=1` y `--file`; no se presenta el fichero principal como SQL
portable para un controlador que ignore metacomandos. Un futuro migrador
embebido deberá concatenar los mismos componentes, sin reinterpretarlos, y
enviarlos dentro de la misma transacción.

Las pruebas ejecutan el principal desde un directorio de trabajo distinto,
retiran o alteran temporalmente cada componente y demuestran que cualquier
fallo conserva barreras `21/5` y no deja objetos parciales. El empaquetado y el
manual enumeran todos los componentes; omitir uno invalida el artefacto.

## Autoridad y frontera de confianza

`000042` contiene exclusivamente cálculo puro. Sus funciones son
`SECURITY INVOKER`, `IMMUTABLE`, `STRICT` y `PARALLEL SAFE`, siempre que todas
sus dependencias acrediten esos mismos atributos. Usan
`search_path = pg_catalog`, propietario exacto
`vec_contratacion_temporal_propietario` y cero permisos para `PUBLIC`,
migrador o roles de ejecución. No crean tablas, RLS ni manifiestos durables.
Los auxiliares históricos no son `PARALLEL SAFE`: se usa `pg_catalog`
directamente o auxiliares privados nuevos; no se falsea ni modifica su
contrato.

El aislamiento `SERIALIZABLE`, la escritura habilitada, la recuperación, la
sesión, la procedencia de filas y la revalidación VEC pertenecen al motor
`000044`. Incluirlos en una función canónica la haría dependiente de sesión y
contradiría su inmutabilidad. `000043`, que sí persiste prueba, tendrá RLS
forzada y primitivas internas `SECURITY DEFINER`.

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

El futuro motor `000044` deberá comprobar:

- `pg_is_in_recovery() = false`;
- transacción `SERIALIZABLE`;
- transacción de lectura y escritura;
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
    vec_contratacion_temporal.resumen_publicacion_rrhh_v1[],
    boolean,
    bytea
) RETURNS bytea

canon_contenido_detalle_rrhh_v1(
    timestamptz,
    vec_contratacion_temporal.entrada_detalle_expediente_rrhh_v1
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
permiten sobrecargas abiertas ni variantes que acepten JSON libre. El detalle
no recibe `jsonb`, un tipo de fila mutable ni valores escalares sueltos: recibe
una única entrada nominal privada.

`resumen_publicacion_rrhh_v1` es un tipo privado de quince atributos que
reproduce exactamente el resumen canónico. Evita acoplar el contrato al tipo
de fila mutable de `publicacion_version_rrhh`. Deben congelarse sus atributos,
orden, tipos, modificadores, propietario, permisos y tipo array automático.

```text
expediente_ref text
organizacion_ref text
numero_visible text
version numeric(20,0)
flujo_ref text
flujo_version numeric(20,0)
flujo_huella_sha256 text
fase_clave text
estado_clave text
centro_ref text
categoria_ref text
modalidad_clave text
unidad_ref text
creado_en timestamptz(6)
actualizado_en timestamptz(6)
```

El detalle añade los siguientes tipos privados. Sus nombres, atributos, orden,
tipos, modificadores, propietario, permisos y tipos array automáticos se
congelan igual que el resumen:

```text
solicitud_operativa_rrhh_v1
  grupo_subgrupo text
  motivo_clave text
  periodo_inicio timestamptz(6)
  periodo_fin timestamptz(6)

analisis_operativo_rrhh_v1
  modalidad_clave text
  categoria_ref text
  causa_clave text
  periodo_inicio timestamptz(6)
  periodo_fin timestamptz(6)
  porcentaje_jornada smallint
  resultado_rc text
  coste_presente boolean
  coste_centimos bigint
  coste_moneda text
  fuente_coste_ref text

comprobacion_operativa_rrhh_v1
  clave text
  resultado text

cobertura_operativa_rrhh_v1
  via_clave text
  decision_gobernada boolean
  procedimiento_ref text
  bolsa_ref text
  comprobaciones comprobacion_operativa_rrhh_v1[]

asignacion_operativa_rrhh_v1
  unidad_ref text
  asignada_en timestamptz(6)
  motivo_clave text

hito_expediente_rrhh_v1
  secuencia numeric(20,0)
  version_expediente numeric(20,0)
  accion_clave text
  realizada_en timestamptz(6)
  fase_origen text
  fase_destino text
  estado_origen text
  estado_destino text

entrada_detalle_expediente_rrhh_v1
  resumen resumen_publicacion_rrhh_v1
  solicitud solicitud_operativa_rrhh_v1
  analisis_presente boolean
  analisis analisis_operativo_rrhh_v1
  referencia_analisis numeric(20,0)
  cobertura_presente boolean
  cobertura cobertura_operativa_rrhh_v1
  referencia_cobertura numeric(20,0)
  asignacion_presente boolean
  asignacion asignacion_operativa_rrhh_v1
  referencia_asignacion numeric(20,0)
  hitos hito_expediente_rrhh_v1[]
```

La ausencia de análisis, cobertura o asignación exige indicador `false`,
compuesto SQL nulo y referencia cero. La presencia exige indicador `true`,
compuesto completo y referencia entre dos y la versión del expediente. El
coste no usa un compuesto anulable: ausente se representa con
`coste_presente = false`, cero, moneda y fuente vacías; presente exige céntimos
positivos, `EUR` y fuente válida.

Los arrays no vacíos son unidimensionales, empiezan en uno y no contienen
elementos nulos. PostgreSQL representa el array vacío canónico con
cardinalidad y número de dimensiones cero; no se le exige un límite inferior.
Los hitos siempre forman un array no vacío. No se revocan permisos ni se
eliminan directamente los tipos array automáticos: heredan el uso efectivo del
tipo elemento y desaparecen internamente con `DROP TYPE elemento RESTRICT`.

### Encuadre de contenido, resultado y recibo

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

El formateador temporal debe ser realmente inmutable. No puede delegar en
`timestamptz::text`, `to_char`, `DateStyle`, `TimeZone`, `lc_time` ni otro
auxiliar histórico declarado con garantías superiores a sus dependencias.
Convierte explícitamente a UTC, compone año, mes, día, hora, minuto, segundo y
seis microsegundos y rechaza infinitos, años fuera de `1..9999` y cualquier
valor no representable. Las pruebas cambian las GUC de fecha, zona y locale y
deben obtener exactamente los mismos bytes.

Este límite de 256 KiB no se aplica a `HuellaConjuntoSHA256`, que usa otro
encuadre y límites por pieza de hasta 1 MiB. El vector multibyte
`VECTOR-UTF8\n8:Área_Ñ\n` acredita el cómputo por octetos del encuadrador; no
es una fila RRHH válida, cuyos campos de negocio tienen gramáticas ASCII.

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

La función pura recibe `entrada_detalle_expediente_rrhh_v1`. Valida todos sus
compuestos y arrays nominales, pero no afirma que procedan de una tabla.
`000044` demostrará la procedencia, la versión exacta y que la misma colección
alimenta respuesta y canon.

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
   - si existe, céntimos positivos y moneda `EUR`;
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

La máscara solo admite `0`, `1`, `3` o `7`. Cobertura exige análisis y
asignación exige cobertura; sus secuencias crecen estrictamente. En un bloque
ausente se sellan presencia `0` y secuencia `0`. El coste ausente omite
céntimos y moneda, pero conserva la fuente vacía. Cada hito cumple
`secuencia = versión_expediente = índice + 1`; el último coincide exactamente
con fase, estado e instante actualizado del resumen.

Solicitud y análisis usan fechas civiles a las `00:00:00.000000Z`. El
porcentaje está entre uno y diez mil. El resultado RC solo admite `validada`,
`no_requerida` o `rechazada`. Las comprobaciones no gobernadas son entre una y
treinta y dos, con claves únicas; una cobertura gobernada exige referencias y
comprobaciones vacías.

El primer hito tiene fase de origen vacía y estado de origen `pendiente`. Cada
hito posterior enlaza fase, estado y orden temporal con el anterior. Las
referencias de los bloques apuntan al hito exacto; modalidad, categoría,
unidad e instante de asignación coinciden con el resumen. El último hito
coincide con fase, estado e instante actualizado del resumen.

Un detalle exitoso siempre produce `total = 1` y no tiene cursor.

### Resultado común

Cabecera:

```text
VEC-CT-RESULTADO-CONSULTA-RRHH-V1\n
```

El contenido lógico del resultado recibe:

1. `tipo_consulta`;
2. `generada_en`;
3. `total`;
4. `contenido_huella_sha256`;
5. `cursor_huella_sha256` o vacío.

La huella de resultado es SHA-256 del canon anterior.

La función histórica de `000040` no puede reutilizarse como dependencia
`IMMUTABLE` y `PARALLEL SAFE`: llama auxiliares que no acreditan esas
propiedades, incluido un formateador basado en `to_char`. `000042` crea una
versión privada nueva o integra el resultado en un auxiliar nuevo cuyo cierre
completo de dependencias sea realmente inmutable, estricto y seguro en
paralelo. No se modifica ni se falsea el contrato histórico.

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

La firma recibe las diez piezas originales y extrae de la capacidad canónica
los once textos. `emitida_en` y `expira_en` conservan literalmente
RFC3339Nano: este formato recorta ceros y es distinto del instante fijo de
seis microsegundos de contenido y recibo. Los límites son los de Go por pieza:
capacidad 32 KiB, decisión 512 KiB, motivo 64 KiB, contexto y evidencia
256 KiB, payload y COSE 1 MiB y SPKI 44 bytes. No existe límite global de
256 KiB para esta preimagen.

La capacidad mide entre 512 bytes y 32 KiB; las demás piezas son no vacías.
Las versiones de persona y perfil son enteros entre uno y `2^53-1`; se
rechazan fracciones, cero, exceso, `NaN` e infinitos antes de convertir a ocho
bytes. Los once textos cumplen las gramáticas y límites Go. Los instantes
RFC3339Nano solo admiten UTC, de cero a seis decimales sin cero final,
`expira_en > emitida_en` y una ventana máxima de cinco segundos.

La raíz pública no se valida solo por longitud: debe ser un `SubjectPublicKeyInfo`
DER canónico de Ed25519 conforme a RFC 8410, con 44 bytes y prefijo estructural
exactos. Las pruebas mutan tanto el prefijo como la clave.

### Canon del Recibo RRHH V2

Cabecera:

```text
VEC-CT-RECIBO-LECTURA-RRHH-V2\n
```

El tipo compuesto privado
`evidencia_recibo_lectura_rrhh_v2` debe contener, en este orden:

```text
esquema text
acceso_ref text
secuencia numeric(20,0)
anterior_sha256 text
huella_sha256 text
vinculo_identidad_huella_sha256 text
alcance_huella_sha256 text
registrada_en timestamptz(6)
auditoria_vec_ref text
auditoria_vec_huella_sha256 text
consumo_vec_huella_sha256 text
decision_ref text
decision_huella_sha256 text
capacidad_huella_sha256 text
material_huella_sha256 text
consulta_huella_sha256 text
correlacion_ref text
autenticacion_ref text
autenticacion_huella_sha256 text
sesion_ref text
control_sesion_ref text
control_sesion_revision numeric(20,0)
control_sesion_huella_sha256 text
actor_ref text
perfil_ref text
perfil_version numeric(20,0)
organizacion_ref text
clase_ambito text
ambito_ref text
accion text
finalidad text
expediente_ref text
version_expediente numeric(20,0)
total smallint
contenido_huella_sha256 text
resultado_huella_sha256 text
cursor_huella_sha256 text
generada_en timestamptz(6)
```

Los nombres coinciden con `000039` y quedan congelados junto con orden,
modificadores, tipo array, propietario y permisos. La secuencia, revisiones y
versiones se limitan a `2^53-1`. Génesis usa 64 ceros como huella anterior,
nunca nulo o vacío. La ausencia de alcance o cursor se adapta a texto vacío
antes del canon.

El discriminador del registrador es exactamente:

```text
vec.contratacion-temporal.recibo-acceso-rrhh.o4-05.v2
```

El sello es SHA-256 de este canon. No forma parte de su propia preimagen.

La función rechaza atributos nulos salvo las ausencias expresamente adaptadas
a texto vacío. Las huellas son hexadecimales minúsculas y no nulas; la huella
anterior solo puede ser cero en génesis. `acceso_ref` se deriva de la huella de
consumo. Secuencia, anterior, referencias, clases de ámbito, acciones y
finalidades cumplen las mismas gramáticas y relaciones que Go.

Un recibo de cuadro exige alcance, expediente y versión vacíos, total entre
cero y cien y cursor solo cuando existe resultado paginado. Un recibo de
detalle exige alcance y cursor vacíos, expediente válido, versión positiva y
total uno. En ambos casos `generada_en <= registrada_en`.

## Migración 000043: prueba durable y cierre interno

### Tabla probatoria

Crear `prueba_resultado_recibo_rrhh_v2`, append-only, con:

- acceso como clave primaria;
- tipo de consulta;
- instante generado;
- total;
- expediente y versión;
- canon y huella de contenido;
- huella de cursor nullable en persistencia, adaptada a texto vacío al
  canonizar;
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
7. verificar una huella constante e independiente de las definiciones y del
   catálogo semántico;
8. rechazar dependencias futuras;
9. usar exclusivamente `DROP ... RESTRICT`;
10. retroceder barreras solo tras retirar correctamente los objetos.

El `down` de `000043` debe fallar si existe una sola prueba o recibo. Nunca
borra trazabilidad para facilitar una reversión.

Las reentradas de `up` y `down` deben fallar sin mutar el estado.

La huella esperada del `safe-down` se congela como literal obtenido en
PostgreSQL 18.4 durante la construcción. Nunca se recalcula como «esperado» a
partir de los objetos vivos. Incluye nombres y sobrecargas, cuerpos, firma
completa, lenguaje, retorno, argumentos, modos y valores por defecto,
`SECURITY`, `leakproof`, `STRICT`, volatilidad, paralelismo, `search_path`,
propietario, ACL con otorgante y opción de concesión, tipos base y arrays,
atributos, modificadores, colaciones, comentarios, etiquetas y dependencias.

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
- coste válido `125000/EUR`;
- coste cero, negativo o moneda ajena como rechazo;
- Unicode multibyte solo en el encuadrador;
- límite exacto y exceso de 256 KiB del encuadrador;
- conjunto material VEC completo;
- Recibo V2 de génesis, secuencia uno y anterior de 64 ceros;
- Recibo V2 encadenado, secuencia dos;
- cuadro con alcance;
- detalle sin alcance.

Cada uno debe construirse independientemente en Go y SQL. La prueba de Recibo
debe mutar individualmente los 38 campos y obtener rechazo.

Antes de aprobar SQL deben quedar congelados en Go el material de 21 bloques,
su huella, dos cánones completos de Recibo V2 y sus sellos. Un esperado
calculado con el mismo auxiliar que se prueba no constituye referencia
independiente.

## Matriz adversarial mínima

La matriz abarca el recorrido `000042`–`000045`. En `000042` solo aplican
pureza, tipos, permisos de funciones, catálogo, límites y cánones. RLS,
aislamiento, recuperación, DML, consumo, cursores y procedencia se prueban en
los cortes que leen o persisten estado; no se simulan en funciones puras.

### Autoridad y privilegios

- `PUBLIC` sin uso ni ejecución;
- roles runtime sin DML ni `EXECUTE`;
- propietario y políticas exactos;
- RLS forzada en `000043` incluso con `row_security` alterada;
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

- Unicode contado como caracteres y no bytes en el encuadrador;
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
9. **Privacidad de logs.** SQL usa errores constantes, pero la ausencia de
   parámetros en logs exige perfil de Sistemas sin registro de valores,
   protocolo extendido y búsqueda de centinelas en cliente y servidor.
10. **Recursos.** PostgreSQL no ofrece una cuota fiable por función. Los
    límites estructurales se complementan con timeout de conexión y cgroup del
    runner; `work_mem` no se presenta como garantía de memoria.

## Puertas de aceptación

Antes de entregar cada migración:

- runner PostgreSQL 18.4 fijado por digest;
- instalación limpia;
- reentrada adversarial;
- `up → down → up`;
- comparación Go/PostgreSQL byte a byte;
- matriz adversarial completa;
- comprobación de catálogo, propietarios, ACL y RLS;
- GUC hostiles de fecha, zona, locale y `search_path`;
- compuestos parciales, arrays multidimensionales, límites inferiores ajenos y
  elementos nulos;
- valores `numeric` fraccionarios, `NaN`, infinitos, cero y bordes máximos;
- mutación individual de los 21 bloques materiales y los 38 campos del recibo;
- manifiesto constante del `safe-down` y ataques con homónimos, sobrecargas,
  cuerpo, propietario, ACL, configuración y dependencia futura;
- `git diff --check`;
- validador Markdown;
- Gitleaks del commit;
- conteo de líneas inferior a 800 por fichero;
- revisión independiente distinta del autor.

Superar estas puertas cierra el contrato probatorio, pero no autoriza
producción. Producción continúa en `NO-GO` hasta disponer de motor, fachadas,
adaptador PostgreSQL Go, composición raíz y pruebas E2E reales.
