# Comprobar la cadena de auditoría V3

`vec-auditoria-verificar` compara un rango de registros con un checkpoint
guardado aparte. Usa la cadena existente de
`vec_autorizacion_atestada_v3.auditoria_consumo_v3`; no crea otra auditoría ni
escribe en la base. Puede detectar cambios en las coordenadas cubiertas por la
huella, filas repetidas, huecos y un rango que no coincide con el checkpoint.

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

El documento contiene `esquema`, `manifiesto` y `registros`. El esquema es
`vec.auditoria.verificacion.v1`. Cada fila contiene exclusivamente:

| Campo | Fuente |
| --- | --- |
| `auditoria_ref`, `secuencia`, `decision_ref`, `efecto_ref` | `auditoria_consumo_v3` |
| `huella_efecto_sha256`, `anterior_sha256`, `huella_sha256` | `auditoria_consumo_v3` |
| `consumo_huella_sha256` | `consumo_decision_v3`, ligado por decisión, efecto y huella del efecto |

Use una instantánea coherente para leer las filas, el consumo y
`control_cadena_auditoria`. Si extrae la cadena completa, la última secuencia y
la cabeza proceden de ese control. La primera secuencia es 1 y su anterior son
64 ceros. En un rango parcial, conserve además la huella anterior al comienzo
del rango y su cabeza final por separado; no las deduzca del documento que va
a comprobar. El contador debe coincidir con el rango inclusive.

Una cadena vacía declara primera y última secuencia 0, contador 0, ambos hashes
con 64 ceros y una lista vacía de registros. No representa un tramo vacío de
una cadena que ya contiene filas.

No extraiga decisiones canónicas, contextos de identidad, certificados,
material criptográfico ni cargas de negocio. Guarde los dos ficheros con
permisos restrictivos y según la política de conservación vigente.

## Ejecutar

Desde el repositorio:

```sh
GOCACHE=/dev/shm/go-build go run ./cmd/vec-auditoria-verificar \
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
persistida; no se exportan las cargas canónicas para recalcularla.
`campos_fuera_huella_verificados: false` advierte que esta cadena no cubre por
sí misma otros campos, como `registrada_en`. Tampoco autentica al actor ni
verifica las firmas de las decisiones originales.

Este corte permite una comprobación reproducible. El sellado periódico, la
firma del checkpoint, la exportación judicial autorizada y su auditoría quedan
para los siguientes cortes. La TSA de desarrollo existente conserva su
procedencia no autoritativa; el comando no la presenta como sello legal.
