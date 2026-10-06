# Comprobar la cadena de auditoría V3

`vec-auditoria-verificar` compara un rango de registros con un checkpoint
guardado aparte. Usa la cadena existente de
`vec_autorizacion_atestada_v3.auditoria_consumo_v3` y lee archivos locales;
no escribe en la base. Admite el formato v1 de consumos y el formato v2, que
incluye intentos denegados o fallidos de AD169. Detecta cambios en los campos
cubiertos por las huellas, filas repetidas, huecos y rangos que no coinciden
con el checkpoint.

## Preparar la entrada

Un operador autorizado prepara dos ficheros privados desde una misma instantánea
de la base, sin cambiar permisos ni ejecutar migraciones:

- Un documento JSON con el manifiesto y las filas en orden ascendente de
  `secuencia`.
- Un checkpoint JSON, conservado por un canal independiente del documento.

La cuenta que obtiene la instantánea necesita la autorización correspondiente.
Este comando no concede acceso ni obtiene filas: recibe una extracción ya
autorizada. La extracción y entrega deben quedar auditadas por su consumidor
transaccional; esta pieza no implementa ese consumidor.

Distinga tanto la base como el carril interno o externo al asignar el
`cadena_id` opaco y conserve ese identificador en los dos ficheros. AD3-116
añade una cadena exterior separada, con tablas y control propios. Este corte
se ensaya con la cadena interna; no mezcle sus registros ni su checkpoint con
los de la exterior. El comando no determina la tabla de origen de un JSON.
No incluya nombres,
DNI, direcciones, credenciales o detalles de conexión.

El manifiesto y el checkpoint tienen la misma estructura:

```json
{
  "cadena_id": "cadena:sintetica:principal",
  "primera_secuencia": 1,
  "ultima_secuencia": 2,
  "anterior_sha256": "0000000000000000000000000000000000000000000000000000000000000000",
  "cabeza_sha256": "7d411c4cb1765927e6b444643a984e81435fad4b0d99488dfda18a4d7a074fa2",
  "registros": 2
}
```

El documento v1 contiene `esquema`, `manifiesto` y `registros`. El esquema es
`vec.auditoria.verificacion.v1`. Cada fila contiene exclusivamente:

| Campo | Fuente |
| --- | --- |
| `auditoria_ref`, `secuencia`, `decision_ref`, `efecto_ref` | `auditoria_consumo_v3` |
| `huella_efecto_sha256`, `anterior_sha256`, `huella_sha256` | `auditoria_consumo_v3` |
| `consumo_huella_sha256` | `consumo_decision_v3`, ligado por decisión, efecto y huella del efecto |

El documento v2 usa `vec.auditoria.verificacion.v2`. Conserva el mismo
manifiesto, pero cada fila indica `tipo_registro` y lleva **un solo** objeto:

- `consumo_confirmado`: objeto `consumo` con las ocho claves v1 de la tabla.
- `intento_nominal`: objeto `intento` con las columnas AD169 de la tabla,
  `registrada_en` en UTC con seis decimales y `contexto_canonico_base64`.

La preimagen de contexto procede de la fuente histórica de ContextoActor V2
mediante una extracción autorizada. Puede incluir referencias de empleado o
candidato. Este comando no obtiene esa preimagen ni concede permiso para
extraerla. Los archivos sintéticos
[`cadena_mixta_v2.json`](../../cmd/vec-auditoria-verificar/testdata/cadena_mixta_v2.json)
y [`checkpoint_mixto_v2.json`](../../cmd/vec-auditoria-verificar/testdata/checkpoint_mixto_v2.json)
sirven para ensayar el formato sin acceder a una base.

Para un intento, el verificador comprueba la huella del contexto V2, su forma
canónica y la coincidencia de actor y perfil. Reconstruye el material con el
prefijo `vec.auditoria.intento.v1`, las huellas de contexto y vínculo, y los 16
campos de orden de AD169 en su orden exacto. Luego comprueba el eslabón con el
prefijo `vec.auditoria.eslabon.intento.v1`, secuencia, huella anterior,
referencia, huella del material y fecha UTC con microsegundos. Los consumos del
mismo rango usan el cálculo de AD3-002. No mezcle filas de otras cadenas.

Use una instantánea coherente para leer las filas, el consumo y
`control_cadena_auditoria`. Si extrae la cadena anterior a AD207, la última
secuencia y la cabeza proceden de ese control; después de AD207, de la última fila
de `eslabon_auditoria_v5`. La primera secuencia es 1 y su anterior son
64 ceros. En un rango parcial, conserve además la huella anterior al comienzo
del rango y su cabeza final por separado; no las deduzca del documento que va
a comprobar. El contador debe coincidir con el rango inclusive.

Una cadena vacía declara primera y última secuencia 0, contador 0, ambos hashes
con 64 ceros y una lista vacía de registros. No representa un tramo vacío de
una cadena que ya contiene filas.

En v2 incluya solo el contexto Actor V2 canónico necesario para cada intento,
obtenido por una extracción histórica autorizada. No añada otros contextos de
identidad, decisiones canónicas, certificados, material criptográfico ni cargas
de negocio. Guarde los dos ficheros con permisos restrictivos y según la
política de conservación vigente.

## Asientos posteriores a AD207

Desde AD207 la cadena avanza por eslabones. Los asientos anteriores al corte
(la secuencia que conserva `control_cadena_auditoria`) se exportan como hasta
ahora. Cada asiento posterior lleva `anterior_sha256` con 64 «f» y su fila de
`eslabon_auditoria_v5` en un objeto `eslabon` dentro del registro, al lado de
`tipo_registro`:

```json
"eslabon": {"posicion": 29592, "secuencia": 29592, "anterior_sha256": "…", "eslabon_sha256": "…"}
```

En estos tramos `primera_secuencia`, `ultima_secuencia` y `registros` del
manifiesto cuentan posiciones de la cadena. Hasta el corte coinciden con el
número del asiento; después pueden no coincidir, porque el número se reserva al
escribir y la posición se asigna al sellar. Ordene los registros por posición.
`anterior_sha256` del manifiesto es el eslabón (o, antes del corte, la huella)
de la posición anterior a la primera, y `cabeza_sha256` el de la última. Solo se
exportan asientos sellados. El verificador recalcula la huella de cada asiento
según su tipo, exige el marcador y un eslabón por asiento a partir del corte, y
recalcula cada eslabón con la fórmula del contrato de auditoría común. La captura
del sello periódico fija la cabeza sellada en ese momento, así que su
`previa_secuencia` queda por debajo de su propio número.

En la base, `vec_autorizacion_atestada_v3.verificar_cadena_auditoria_v5(false)`
(cadena interna; `true` para la externa) comprueba como propietario los enlaces de
toda la cadena: numeración y enlaces hasta el corte, cada eslabón recalculado
desde su asiento, y la cola. Devuelve `pendientes` (asientos aún sin eslabón) y
`sin_sellar_fuera_de_cola` (asientos sin eslabón que tampoco esperan en la cola;
debe ser 0). No recalcula la huella de cada asiento según su tipo: eso lo hace
este comando.

## Ejecutar

Desde el repositorio:

```sh
GOCACHE=$HOME/.cache/go-build go run ./cmd/vec-auditoria-verificar \
  --checkpoint /ruta/privada/checkpoint.json \
  --max-bytes 16777216 \
  --max-registros 10000 < /ruta/privada/cadena.json
```

Los límites son obligatorios. `max-bytes` limita cada fichero, con un máximo
admitido de 1 GiB; `max-registros` limita las filas. Ajuste ambos al volumen
esperado. Las secuencias no superan 9007199254740991, conforme al consumidor
AD3-002. El comando no acepta campos desconocidos, claves repetidas, `null`,
objetos adicionales ni texto después del documento.

La salida es JSON con códigos estables. Un fallo devuelve la clave y sus dos
valores de comparación, sin mostrar referencias de decisiones, efectos o
contenido de la entrada.

| Código de salida | Resultado |
| --- | --- |
| 0 | Las filas y el rango coinciden con el checkpoint suministrado. |
| 1 | Falló la comprobación de secuencia, enlace, huella, cantidad o checkpoint. |
| 2 | Argumentos, JSON, límites o fichero de checkpoint no admitidos. |
| 4 | No se pudo escribir el informe. |

## Qué demuestra el informe

`estado: verificada` y `checkpoint_cotejado: true` indican que la cadena
coincide con el checkpoint suministrado. `cobertura` delimita exactamente las
filas comprobadas.

La huella reproduce la preimagen de AD3-002: secuencia, anterior, decisión,
efecto, huella del efecto y huella del consumo. Cada valor se encuadra con su
longitud en bytes UTF-8, dos puntos, el valor y un salto de línea. El comando
también comprueba la referencia de auditoría derivada del consumo y la
unicidad de decisiones y consumos dentro del rango.

`autenticidad_checkpoint: no_comprobada` significa que el comando no autentica
la procedencia del checkpoint. Si alguien sustituye tanto las filas como el
checkpoint, esta comparación por sí sola no detecta la sustitución. Para
detectar un recorte final necesita una cabeza obtenida y conservada con
independencia antes de la comprobación. No acredita filas posteriores al
checkpoint ni operaciones que nunca se registraron.

`contenido_consumo_recalculado: false` indica que se usa la huella del consumo
persistida; no se exportan las cargas canónicas para recalcularla. En v2,
`material_intento_recalculado` y `actor_perfil_contexto_cotejados` indican qué
se comprobó para los intentos presentes. La preimagen del vínculo de
autenticación no está en este formato, así que su huella no acredita por sí
sola la sesión. `autenticidad_fuentes_historicas: no_comprobada` deja constancia
de que el archivo no prueba la procedencia de las preimágenes. El verificador
tampoco comprueba las firmas de las decisiones originales.

`campos_fuera_huella_verificados: false` se conserva para el formato v1 y para
las partes de consumo del formato v2. Los campos de intento incluidos en su
material y eslabón sí se cotejan; la marca específica anterior refleja ese
alcance. La autenticidad del checkpoint sigue sin comprobarse.

Este corte permite una comprobación reproducible. El sellado periódico, la
firma del checkpoint, la exportación judicial autorizada y su auditoría quedan
para los siguientes cortes. La TSA de desarrollo existente conserva su
procedencia no autoritativa; el comando no la presenta como sello legal.
