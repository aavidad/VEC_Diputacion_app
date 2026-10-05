# Fin por causa según modalidad — preparación del corte

La sustitución puede terminar con la reincorporación de la persona titular y la vacante con su cobertura reglamentaria. En esos casos VEC guarda la causa y la regla publicada, sin asignar una fecha de fin. Los expedientes fechados conservan su representación anterior.

El paquete de ejemplo para desarrollo usa estos archivos, cuyos SHA-256 deben comprobarse antes de activar el corte:

| Archivo | SHA-256 |
| --- | --- |
| `data/demo/reglas/ct_reglas.ejemplo.demo.v3.json` | `31b03fca216c897ccd4256ca77c9d671354d608293f676bbc648e21032c5087a` |
| `data/demo/plantillas/ct_plantillas_documentos.ejemplo.v2.demo.json` | `854789f2d18abb4c269ab0d3ed72cf7d89e11706998f645f06b3cb415e0ce9be` |

Las migraciones de la lista combinada tienen estas huellas:

| Migración | SHA-256 |
| --- | --- |
| CT165 | `7a7ac82c0137d77339996022e234c416843a2525cf426f306430c0c66a05bf6e` |
| CT166 | `e6642ac29bf4b76f9315901f7f1c1e1bcd435d4f2e409990fcc5b039a17e94ab` |
| CT167 | `961951e7f34fe120ded4e8230efde321eb33d824d0f1af1ebf6f365cc6494790` |
| Bolsa B74 | `9a99fa259160d18d206fe9efa21d5769f1717e767f3faabff32b79d287b48b09` |

En desarrollo, `VEC_CT_REGLAS_SOURCE_PATH` debe apuntar a las reglas v3 y `VEC_CT_PLANTILLAS_SOURCE_PATH` a las plantillas v2. Su carga conserva la doble llave de desarrollo. El catálogo documental se publica mediante su perfil fijo, con comprobación de preimagen y versión esperada; la presencia del archivo no acredita por sí sola esa publicación. Si la preimagen no coincide, se detiene la publicación y se revisa el catálogo vigente antes de actuar.

La lista `lista_sql_codexr_fin_modalidad_20261002.txt` fija las migraciones nuevas de este corte. El ensayo parte de `~/.local/state/vec-clon/estado-cidonia-20260929-hito1.tgz`, añade en orden las listas de `hito3/paquete/lista_sql_h3.txt` y `hito4/paquete/lista_sql_h4.txt`, y después esta lista. Se usa PostgreSQL 18 en un clon desechable con datos en `/dev/shm`; no se ejecuta DOWN ni se reaplica una migración ya instalada. En la principal solo instala dirección, con copia de seguridad previa y tras comprobar los números existentes. La instalación y el binario deben quedar coordinados: el código nuevo exige las funciones nuevas y Bolsa necesita B74 para aceptar un horizonte abierto.

B74 modifica el guardado del **puente actual CT→Bolsa** (`guardar_integracion_desarrollo_v1`). La oferta telemática nueva usa `publicar_oferta_v4` de B71, que admite omitir `fecha_fin` pero no recibe la causa gobernada del expediente CT. Hoy no existe un enlace CT→oferta v4; esa integración corresponde al circuito de ofertas y no queda acreditada por este corte. El ensayo causal con B71→B70→B75→CT165→CT166→CT167→B74 conservó ambos caminos: 24 SQL aplicadas, las dos guardas abiertas de B74 presentes y el trigger B75 activo.

Tras el arranque, la comprobación sintética recorre petición del centro, ratificación, entrega a RRHH, análisis, cobertura, aviso, expediente y borrador. Debe observarse `causa_fin` con referencia, versión y huella de la regla, sin valor de `fin`; el coste total permanece pendiente cuando no hay horizonte. El cese efectivo y la disponibilidad proceden de Personal, no del fin previsto. Una respuesta válida se contrasta con recibo, historia y recuperación tras reiniciar la aplicación y PostgreSQL. Estas plantillas son borradores de desarrollo, sin firma ni aprobación jurídica de RRHH.
