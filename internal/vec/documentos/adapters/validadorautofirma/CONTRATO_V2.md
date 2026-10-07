# Cliente del dictamen v2 de GrxFirma

El cliente pide `autofirmav2.dictamen-verificacion.v2` por `POST /v2/verify`
y envía además `contrato_solicitado` con ese valor. Solo interpreta `dictamen`;
los indicadores y textos heredados de la envoltura no deciden el resultado.
Mantiene TLS 1.3, CA privada explícita, credencial Bearer o certificado del
proxy, rechazo de redirecciones y límites de tiempo, tamaño y profundidad.
El contrato v2 publicado admite PAdES; no hay retorno automático a v1.

`Cliente.VerificarFirmas` implementa el puerto neutral
`ports.VerificadorFirmasDocumento`. Devuelve todas las revisiones, incluidas
las firmas de tipo `sello_tiempo_documento`, con certificado y cambios propios.
La aplicación comprueba el cargo y la competencia de cada firmante. El tipo
de firma distingue los sellos técnicos de las firmas humanas. El puerto anterior
solo admite una firma humana; una cadena con más entradas queda indeterminada.

Antes de devolver evidencia, el cliente verifica el contrato y el esquema,
las claves requeridas y exactas, los duplicados, los catálogos y los enteros.
Contrasta el eco del original y del documento firmado con los bytes enviados.
Exige orden consecutivo y longitudes de revisión crecientes. Recalcula
`revisionHuellaSHA256` sobre el prefijo y `contenidoFirmadoHuellaSHA256` sobre
los dos segmentos de `byteRange`, sin admitir solapes ni desbordamientos.
Un vínculo acreditado exige el original custodiado como prefijo de la primera
revisión. El análisis del PDF y la verificación CMS corresponden a GrxFirma.

Un estado `valida` exige aspectos concluyentes en todas las entradas y cambios
`ninguno` o `permitidos`. Un estado negativo o indeterminado nunca se promueve
por indicadores favorables de la envoltura. La traducción de motivos nuevos
usa los motivos neutrales de integridad y conserva las categorías de cambios.
No se propagan asunto, emisor, serie ni nombres de campos del PDF.

## Artefactos fijados

Fuente: GrxFirma 0.0.107, `docs/schema/dictamen-verificacion-v2.schema.json`
y `testdata/dictamen-v2`, más el acuerdo de VEC y GrxFirma del 2 de octubre.
El esquema se conserva sin modificar y se comprueba al cargarlo:

`e57e5cc0f47ad1ffdde30ff5ae8df3e0f1f93671f4714416ddce425e6142db6e`

Se incluyen los 20 casos con original, PDF firmado y dictamen esperado.
Son 60 archivos sintéticos; no se incluyen las claves ni el directorio PKI.
`testdata/dictamen-v2/SHA256SUMS` registra cada huella. Su SHA-256 es:

`72c7467d41f3d317e08b1974851ba03aca2f456e56e94d7331b9572df67249db`

## Discrepancia de la publicación

Cinco casos contienen categorías que el esquema fijado no admite:

| Casos | Campo | Categoría ausente |
| --- | --- | --- |
| 12, 16 y 20 | Detalles de cambios | `dss_anadido` |
| 18 y 20 | Detalles entre firmas | `sello_tiempo_documento_anadido` |
| 19 | Motivo de integridad | `sello_documento_no_valido` |
| 19 | Detalles entre firmas | `sello_tiempo_documento_pendiente_confianza` |

Los casos 12, 16, 18 y 20 declaran `valida`; el 19 declara `no_valida`.
El cliente rechaza los cinco con `indeterminada/respuesta_no_interpretable`
y sin evidencia parcial. Los otros 15 conservan el estado y todas las revisiones
del dictamen. Las pruebas fijan ambos resultados. Para admitir aquellos casos
hace falta una publicación corregida del esquema con su nueva huella.

## Dependencia y comprobación

La rama incorpora el contrato neutral de puertos del commit
`113120e7e76c274365d67881b4846d81fafc6647`, entregado por otro productor,
mediante el cherry-pick autorizado por dirección. Es el único archivo ajeno
al paquete incorporado a esta candidata: `ports/firma_multiple.go`.

Se ejecutan pruebas normales de `./internal/vec/documentos/...`, y pruebas
de concurrencia y `vet` del adaptador y su servidor sintético. Gosec se limita
al paquete cambiado. Semgrep usa cuatro reglas locales, con métricas y
comprobación de versión desactivadas; no es una auditoría global. La revisión
focal de seguridad inspecciona transporte, límites, interpretación del proveedor,
huellas y minimización. Las dos revisiones independientes corresponden a
dirección y deben identificar el commit exacto de cliente y contrato.

Las pruebas usan el compilador local Go 1.26.6, `GOTOOLCHAIN=local`,
`GOPROXY=off`, `GOSUMDB=off`, `GOCACHE=/dev/shm/go-build`, `TMPDIR=/tmp`
y `-p 8`. El índice `codebase-memory-mcp` se consultó mediante su CLI; su
resultado no contenía este paquete, por lo que se leyeron las rutas concretas
del encargo. Se usó `gopls` desde terminal para consultar definiciones.

Esta entrega acredita traducción y pruebas sintéticas. No instala GrxFirma,
no cambia SQL ni composición, y no acredita un circuito RRHH, custodia durable,
competencia administrativa o validez jurídica de la firma.
