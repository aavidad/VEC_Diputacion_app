# Recuperación del detalle tras una carrera de auditoría

Tres lecturas simultáneas del expediente 2026/CT-000127, versión 9, terminaron
en HTTP 200 con el mismo cuerpo. PostgreSQL rechazó dos intentos con `40001`
durante la auditoría; el adaptador repitió la transacción completa y quedaron
tres consumos confirmados y tres auditorías distintas, con sus eslabones V5
sellados. Los intentos abortados no dejaron otro consumo confirmado.

Se reutiliza `ejecutarConsultaRRHHEnTransaccion`, que ya recuperaba el cuadro.
La lista cerrada incorpora el detalle ordinario y el detalle con Bolsa. Cada
intento conserva consulta, argumentos, cursor y material V3; revierte antes
del siguiente y vuelve a validar el resultado antes de confirmar. Permisos,
errores de validación, interbloqueos y confirmaciones ambiguas quedan fuera
del reintento. Las otras fachadas tampoco se amplían.

La unidad de reintento incluye la selección y validación del resultado, como
indica la [documentación de PostgreSQL 18](https://www.postgresql.org/docs/18/mvcc-serialization-failure-handling.html).
Se conserva la política y el límite existentes; no se emite otra decisión ni
se renueva la autorización después del PDP.

## Procedencia y comparación

| Dato | Referencia |
| --- | --- |
| Base observada | `e4346d1141c0717e4bd2d61854d46c5fe66996dc` |
| Primera corrección, detalle ordinario | `c1429e3d595c1d28846abfbf9c92746c332cf9af` |
| Corrección de ambas variantes | `67649997776f14d3f4f14ba5f1541cf40316169b` |
| Integración local verificada, base más corrección | `80f24ae8fc62f31660017150cb6e4899589f79dc` |
| Go | 1.26.9; CGO 1; `vcs.modified=false` |
| Configuración | 26 operaciones; misma huella, certificado y CA en las tandas |

La primera corrección recuperó las lecturas del plan. Una tanda concurrente
de detalle devolvió 200/200/503: el error restante nombró
`consultar_detalle_rrhh_con_bolsa_v1`, que usa la otra constante SQL del mismo
ejecutor. Esa observación motivó incluir la segunda variante; no se cambió su
SQL ni se instaló una migración.

La tanda final repitió tres POST a
`/api/vec/contratacion-temporal/expedientes/consultas`, con el mismo cuerpo,
identidad y configuración del expediente. Usó curl asíncrono, CA y certificado
cliente originales, y validación TLS. Los originales privados permiten repetir
las peticiones; esta entrega conserva sus huellas y los resultados, sin claves
ni credenciales. El cuerpo de respuesta coincide con la referencia anterior.

Los tiempos de curl fueron **280,604 / 437,282 / 364,517 ms**; los del servidor,
**236,8 / 321,2 / 393,6 ms**. Tres muestras no permiten calcular un p95.
La mayor consulta registrada fue de 50,324 ms. Esta prueba acredita recuperación
del fallo observado y no declara cumplido el objetivo de todas las lecturas
por debajo de 300 ms.

## Evidencia y límites

- [Resultado y procedencia del binario](resultado.json).
- [Peticiones, cuerpos y auditorías](peticiones-y-auditoria.json).
- [Causas PostgreSQL y tiempos de petición](causas-y-tiempos.json).

Los metadatos canónicos cotejan principal, perfil, acción, recurso y finalidad.
La columna de canal del registro legado está vacía; no se atribuye un valor que
no se ha extraído. Un eslabón que aún no aparecía en la primera consulta quedó
sellado después por el circuito existente. No hubo sellado manual.

PostgreSQL no registra el identificador HTTP en esos errores: se conservan
instantes, PID, funciones y correlaciones de aplicación, sin inventar una
correspondencia individual entre cada conflicto y cada petición.

La guarda de preimagen que produjo un 403 sigue intacta. Esta entrega tampoco
cierra incorporación a Personal, firma, recuperación de las claves históricas
de Bolsa ni publicación del puesto y unidad que faltan en el recorrido B2.

Las pruebas de Begin, Scan y Commit fallaron antes en las variantes excluidas
y pasan después. Los negativos conservan un solo intento ante `42501`,
`40P01`, validación fallida o confirmación ambigua. Focal y race pasan;
Semgrep no registra hallazgos y Gosec completa la carga sin errores. Sus avisos
basales no afectan al bloque modificado. La CI de la cabeza final y la revisión
independiente exacta se comprueban antes de declarar la PR lista.
