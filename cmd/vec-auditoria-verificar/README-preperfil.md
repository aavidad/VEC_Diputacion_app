# Comprobar registros administrativos previos al perfil

El formato `vec.auditoria.verificacion.v3` permite comprobar una cadena con
consumos confirmados, intentos de AD169 y las familias `preperfil_autenticado`
y `bootstrap_operador` de AD171. Conserva los cálculos y formatos v1 y v2.
No obtiene registros de PostgreSQL: recibe una extracción autorizada y un
checkpoint separado, igual que el verificador anterior.

Cada registro contiene `tipo_registro` y exactamente uno de estos objetos:

| Tipo | Objeto | Contenido |
| --- | --- | --- |
| `consumo_confirmado` | `consumo` | Las ocho claves del formato v1. |
| `intento_nominal` | `intento` | Las columnas y preimagen de contexto del formato v2. |
| `preperfil_autenticado` | `preperfil` | Coordenadas AD171 y referencia opaca de actor. |
| `bootstrap_operador` | `bootstrap` | Coordenadas AD171, LOGIN técnico, huella de plan y referencia de aprobación. |

Los objetos AD171 tienen las claves comunes `auditoria_ref`, `secuencia`,
`anterior_sha256`, `huella_sha256`, `registrada_en`, `evento_ref`,
`evento_material_sha256`, `modulo_id`, `accion`, `recurso_ref`, `resultado`,
`motivo_ref`, `proceso`, `canal`, `finalidad_ref`, `correlacion_ref`,
`fuente_ref` y `fuente_sha256`. Preperfil añade exclusivamente `actor_ref`;
bootstrap añade exclusivamente `operador_login`, `plan_sha256` y `aprobacion_ref`.
Todas las claves son obligatorias. No se admiten valores nulos, campos
desconocidos ni claves repetidas.

Preperfil no admite LOGIN, plan o aprobación. Bootstrap no admite actor humano.
Ninguno admite perfil activo, contexto, sesión, decisión ni efecto V3. El módulo
es `administracion`; los resultados admitidos son `permitido`, `denegado` y
`error`. Acción, canal y finalidad deben coincidir con su familia SQL.

El verificador reconstruye los marcos UTF-8 del material AD171 con el dominio
`vec.auditoria.admin-preperfil.v1`, siguiendo el orden de campos fijado por SQL.
Comprueba después el eslabón con el dominio
`vec.auditoria.eslabon.admin-preperfil.v1`, secuencia, huella anterior,
referencia `aud_v3_p_<32 hex>`, huella del material y fecha UTC con seis
decimales. Las dos familias comparten el registro y la cabeza existentes.
El formato v2 continúa rechazando estas familias nuevas.

Desde la raíz del repositorio se puede comprobar el ejemplo sintético de seis
filas intercaladas:

```sh
GOCACHE=$HOME/.cache/go-build go run -p 8 ./cmd/vec-auditoria-verificar \
  --checkpoint cmd/vec-auditoria-verificar/testdata/preperfil_checkpoint_v3.json \
  --max-bytes 16384 --max-registros 6 \
  < cmd/vec-auditoria-verificar/testdata/preperfil_mixta_v3.json
```

Los cuatro vectores de
[`preperfil_ad171_vectores.json`](testdata/preperfil_ad171_vectores.json)
proceden del ensayo SQL de AD171. Cada uno es independiente, con huella anterior
de ceros. El ejemplo intercalado conserva la primera fila antigua de AD169 y
combina un consumo con los cuatro eventos nuevos; sus eslabones fueron
calculados por separado con Python.

`estado: verificada` acredita la coincidencia de los campos, enlaces y rango
con el checkpoint suministrado. `material_evento_recalculado: true` indica que
había eventos AD171 y se reconstruyó su material. Los indicadores anteriores
conservan su alcance para consumos e intentos.

`autenticidad_checkpoint` y `autenticidad_fuentes_historicas` permanecen en
`no_comprobada`. Los archivos no prueban la procedencia de actor, LOGIN, fuente,
plan o aprobación. Esta comprobación no acredita identidad, custodia, firma,
TSA, autorización de extracción ni instalación de AD171. El checkpoint debe
obtenerse y conservarse por un canal independiente y autorizado.

Se mantienen los códigos de salida: 0 para una cadena verificada, 1 para una
divergencia, 2 para argumentos o JSON no admitidos y 4 si no se puede escribir
el informe. Los fallos AD171 muestran claves y estados fijos; no reproducen
referencias ni el contenido rechazado.

## Consumos con sello de transacción AD193

Los esquemas mixtos que admiten consumo v3 también admiten
`consumo_confirmado_v4`, con `version_consumo: 4`, dentro del objeto `consumo`.
Conserva los campos v3 y añade `transaccion_origen` y
`transaccion_consumo_origen`, ambos obligatorios e iguales. Se reciben como
cadenas decimales positivas canónicas del rango uint64; nunca como números
JSON. Por ejemplo, `"9007199254740993"` y `"18446744073709551615"` mantienen
todos sus dígitos. Cero, signos, espacios, ceros iniciales, exponentes,
desbordamientos, valores nulos y campos omitidos se rechazan.

El eslabón conserva el orden de quince campos de AD193 y añade como campo 16
el sello de auditoría. El sello del consumo se coteja por igualdad; no añade
un campo 17. Las fechas de registro y consumo siguen siendo UTC con seis
decimales e iguales. Los formatos anteriores conservan sus huellas y no
admiten los campos nuevos. LOGIN técnico y actor humano mantienen familias
separadas.

Los [vectores AD193](testdata/consumo_transaccion_ad193.json) contienen tres
consumos históricos y dos v4 encuadrados con Python/SHA256. Su
[checkpoint](testdata/consumo_transaccion_ad193_checkpoint.json) permite
usar el comando anterior sustituyendo el archivo de entrada y la ruta de
`--checkpoint` por estos dos archivos, con `--max-registros 5`. Verificar estas cadenas
no acredita COSE, procedencia de extracción ni instalación de AD193. XID
identifica una transacción dentro de una instalación; no identifica de forma
global una transacción entre bases o después de una importación.
