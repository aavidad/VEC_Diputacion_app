# Preparación de los pasos 16 a 21 — F1-28-prep

Revisión de fuentes: `origin/main@92f6a7501`, 21/10/2026. Destinatario: equipo P,
para continuar el expediente 95034 de su copia J. No se ha accedido a esa copia,
creado otra ni ejecutado actos. El caso de aceptación sigue siendo
[FLUJO_APP.md §4.0](../../docs/producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación).

El orden es **17 → 18 → 16a → 16b → 19 → 20 → 21**. Las rutas de la tabla
son relativas al origen de la copia. «Listo en código» significa que existe
la conexión entre pantalla y servidor; falta comprobar cada paso en Chrome,
con sus recibos y su recuperación tras reiniciar. Ninguna fila acredita el
estado ni la configuración del proceso de J.

## Pantallas, rutas y activación

En las rutas de Contratación de esta tabla, `CT` significa
`/api/vec/contratacion-temporal`; no es una variable de configuración.

| Paso | Pantalla y módulo de interfaz | Ruta HTTP | Perfil y permiso exigido | Activación existente | Estado |
| --- | --- | --- | --- | --- | --- |
| 17. Acreditar incorporación | Centro: `/portal-empleado/peticiones-centro/`, Incorporaciones; `incorporaciones-centro.js` | `GET CT/peticiones-centro/incorporaciones`; `POST CT/peticiones-centro/incorporaciones/confirmaciones` | Certificado del centro de la petición. Consulta: `contratacion_temporal.incorporacion.consultar_centro`; confirma el rol que permita la regla c21 mediante `contratacion_temporal.incorporacion.confirmar_centro` | `VEC_CT_INCORPORACION_ACREDITADA_ENABLED=true`, seguimiento activo, regla c21 y perfiles del centro preparados por la receta | Listo en código; falta acreditar fecha y documento desde el centro de J. La pantalla debe devolver `puede_confirmar=true` antes del acto. |
| 18. Alta en Personal | RRHH: `/portal-empleado/#contratacion-temporal`, ficha, Paso a Personal; `incorporacion-personal-b2.js`, montado por `vista-expedientes-incorporacion.js` | `GET` y `POST CT/incorporacion-personal-b2/plan/v1`; `POST CT/incorporacion-personal-b2/confirmar/v1`. Vínculo previo: `GET/POST CT/incorporacion-personal-b2/vinculo-categoria-rpt/v1`; opciones: `GET CT/incorporacion-personal-b2/categorias-rpt/v1` | Certificado de RRHH original de la copia. Acciones `contratacion_temporal.incorporacion_personal.plan.consultar`, `.plan.registrar` y `.origen.confirmar`; perfiles nominales B2 por operación, además de los de Personal, Bolsa y RPT | `VEC_CT_INCORPORACION_V2_FILE` apunta al material propio generado; Personal B2, catálogos y conexiones de la receta preparados | Listo en código, incluida aceptación nativa Mi Bolsa (CT255, #1378 fusionada). Faltan el plan del expediente actual, su confirmación y la lectura de relación y ocupación en Personal. |
| 16a. Descargar ficha GINPIX | Misma ficha, bloque GINPIX tras alta confirmada; `ficha-ginpix.js` | `GET CT/incorporaciones-ejercicio/ficha-ginpix` con el selector validado por `cliente-http-ficha-ginpix.js` | Misma identidad RRHH y permiso de consulta del plan B2; el servidor coteja el origen confirmado y los hechos de Personal con sus autoridades propias | Mismo material B2 del paso 18; el montaje ofrece descarga solo con recibo confirmado y versión actual | Listo en código; falta descargar `ficha-ginpix-personal.json` desde el recibo B2 de J. El nombre histórico de la ruta no implica usar el alta antigua. |
| 16b. Confirmar grabación manual | Misma ficha, seguimiento, formulario GINPIX; `seguimiento-incorporacion-acreditada.js` | `POST CT/seguimiento-cese` para consultar; `POST CT/confirmaciones-ginpix` para registrar | Técnico RRHH: `contratacion_temporal.seguimiento.consultar` y `contratacion_temporal.ginpix.confirmar` | Seguimiento e incorporación acreditada activos; regla de GINPIX y origen de incorporación existentes | Listo en código; falta confirmar número y fecha y conservar su recibo. Es confirmación local de grabación manual, sin transmisión a GINPIX. |
| 19. Cese | Misma ficha, seguimiento, Cese; `seguimiento-cese.js` | `POST CT/seguimiento-cese`; `POST CT/ceses` | Técnico RRHH: `contratacion_temporal.seguimiento.consultar` y `contratacion_temporal.seguimiento.cesar`; el fin en Personal conserva su autorización nominal | `VEC_CT_SEGUIMIENTO_CESE_ENABLED=true`, reglas y causas publicadas; material B2 con `cese_fecha_efecto` explícito | Listo en código; faltan recibo del cese actual y consulta de fin de relación y ocupación en Personal. El justificante conserva referencia y huella, sin custodia de sus bytes en Documentos. |
| 20. Vuelta a Bolsa | Yago: `/area-personal/`, Mi Bolsa; RRHH: ficha del candidato de la bolsa constituida | Consulta propia `GET /api/vec/bolsa/mi-bolsa`. El efecto llega por el relevo de cese, sin un POST de «volver» desde la pantalla | Certificado propio del candidato para Mi Bolsa; RRHH con lectura nominal de Bolsa. El relevo usa su conexión y autoridad propias | `VEC_BOLSA_CESE_CT_ENABLED=true`, conexión nominal `VEC_BOLSA_CESE_CT_DATABASE_URL`, Bolsa y relevo de contratos activos | Falta comprobar el retorno actual de la bolsa constituida y sus fechas. S prepara F1-122 (#1414); este corte no la modifica. No crear un registro nuevo de Bolsa. |
| 21. Cierre | RRHH: ficha, seguimiento, Cierre; `seguimiento-cese.js` | `POST CT/seguimiento-cese`; `POST CT/cierres-expediente` | Técnico RRHH: consulta de seguimiento y `contratacion_temporal.expediente.cerrar` | Seguimiento activo y regla de cierre vigente; cese y GINPIX confirmados según esa regla | Listo en código; falta cierre del caso actual y recuperación de su recibo. No acredita por sí solo que Bolsa haya aplicado el cese. |

Los nombres `.plan.registrar` y `.origen.confirmar` de la fila 18 completan el
prefijo `contratacion_temporal.incorporacion_personal`. La matriz técnica está
en `internal/app/bootstrap/configuracion_incorporacion_personal_b2.go`;
`autorizacion_incorporacion_personal_b2.go` separa sus perfiles y ámbitos.
No se solicitan nuevas concesiones ni se alteran las comprobaciones.

## Qué debe estar ya preparado en J

La receta actual es [recorrido_copia/LEEME.md](recorrido_copia/LEEME.md).
Su conductor llama a `personal.preparar(c)` y después a `copia.personal(...)`:
prepara los motivos, conexiones y capacidades propias, publica el material
y comprueba con RRHH las consultas de categorías RPT y catálogos de Personal.
Su recibo de fábrica indica `configuracion_personal=true`; ese dato no prueba
los actos de incorporación ni cese.

Para cotejar la activación, P puede consultar sus materiales y registros ya
generados, sin volver a ejecutar la fábrica:

- `recorrido/personal/configuracion.json`: identidad RRHH, unidad y operaciones
  de la configuración de entrada. `recorrido_copia/personal.py` exige identidad
  original, unidad de su recibo administrativo y los 27 motivos del inventario.
- `personal/preparada.json` y `personal/material/servidor.json`: marcas del
  productor y material de ejecución; `VEC_CT_INCORPORACION_V2_FILE` debe apuntar
  al segundo, sin copiar su contenido ni sus conexiones a una entrega pública.
- `entorno_modulos.json`: la preparación existente activa
  `VEC_PERSONAL_B2_GOBIERNO_ENABLED`, `VEC_CT_INCORPORACION_ACREDITADA_ENABLED`,
  `VEC_CT_SEGUIMIENTO_CESE_ENABLED` y `VEC_BOLSA_CESE_CT_ENABLED`.
  Las dependencias completas están en
  `config/selectores_dependencias_bolsa_ct_v1.json` y
  `copia_local_personal.py:entorno_personal`.
- El intervalo `VEC_BOLSA_CONTRATOS_CT_INTERVALO` debe estar activo; en ausencia
  del ajuste se usa 30 segundos. La receta rechaza `0` y `0s`.
  La conexión de cese pertenece al rol nominal de relevo de Bolsa, con TLS.
- `recorrido_copia/datos/personal_b2.json` fija `cese_fecha_efecto` en
  `primer_dia_sin_relacion`. Se coteja con el material de J; no se deduce ni
  cambia desde el formulario. La configuración también admite
  `ultimo_dia_trabajado`; la definición funcional definitiva sigue en duda 140.

El catálogo de reglas selecciona quién confirma la incorporación y qué
documento exige. Si hay un 403, ausencia de material o una capacidad no
provisionada, detener ese paso y comunicar la ruta, código catalogado y
correlación. Cualquier necesidad de SQL o permisos nuevos se remite a
dirección antes de editar. Esta hoja no autoriza reconfigurar ni reiniciar J.

## Comprobación del caso por P

1. Tras la resolución y las firmas del paso 15, entrar con el certificado del
   centro de la petición. Comprobar que la incorporación procede y registrar
   fecha y documento exigidos. No usar el certificado de RRHH para ese acto.
2. Volver con RRHH a la misma ficha. Consultar el plan B2 y resolver desde la
   pantalla el vínculo de categoría publicado si falta. Revisar Administrativo,
   puesto 9/01 y plaza 2026/0033, las fechas y los requisitos; preparar y
   confirmar. Conservar referencias de plan, alta, relación y ocupación.
3. Descargar la ficha ligada a ese recibo y confirmar la grabación manual de
   GINPIX con número y fecha. Conservar archivo y recibo local.
4. Registrar el cese con una causa publicada, fecha de efectos y justificante.
   Consultar la misma relación y ocupación en Personal y el estado de esa
   participación en Mi Bolsa y en RRHH. La espera del relevo se resuelve con
   lecturas; no repitiendo el cese ni cambiando de candidato.
5. Registrar el cierre y comprobar expediente completado. Antes y después del
   reinicio autorizado de aplicación y base, comparar los mismos recibos,
   fechas, relación, ocupación y participación. No crear otra operación para
   recuperar una respuesta perdida. Cotejar Chrome a 1440 y 390 px.

## Faltas y límites conocidos

- Falta el recorrido conectado de estos pasos sobre el nombramiento nativo
  de J, con recuperación tras reiniciar. Las fuentes y las pruebas con dobles
  no permiten anunciarlo como completado.
- El justificante de cese es referencia y huella; su custodia en Documentos
  sigue pendiente. No presentarlo como archivo custodiado.
- F1-122 conserva el ensayo de la vuelta a la bolsa constituida; F1-135,
  aparcada, añadiría una puerta de cierre sobre recibos de Personal y Bolsa.
  La regla de cierre actual exige sus hechos de CT y GINPIX, sin esa puerta.
- [incorporacion_nativa_f1_134.md](incorporacion_nativa_f1_134.md) conserva
  referencias a integración y candidata D pendientes en su fecha de redacción.
  En la base de esta hoja #1378 ya está fusionada y CT255 está en la lista
  canónica `lista_sql_main_tras_hz37.txt`. No instalar de nuevo su lista histórica.
- La prueba de arranque con y sin configuración no se ejecuta en este corte
  documental; la preparación de Personal tiene pruebas de vuelta atrás ante
  fallo del productor y ante HTTP 503. Ninguna configuración cambia aquí.

## Pruebas focales ejecutadas

El 21/10, sobre la base indicada, **71 pruebas JavaScript y 21 Python verdes**.
No se ha modificado código ni SQL. Comandos:

```sh
node --test \
  web/static/portal-empleado/peticiones-centro/incorporaciones-centro.test.mjs \
  web/static/portal-empleado/modulos/contratacion-temporal/incorporacion-personal-b2.test.mjs \
  web/static/portal-empleado/modulos/contratacion-temporal/incorporacion-personal-b2-recuperacion-error.test.mjs \
  web/static/portal-empleado/modulos/contratacion-temporal/montaje-incorporacion-personal-b2.test.mjs \
  web/static/portal-empleado/modulos/contratacion-temporal/montaje-ficha-ginpix.test.mjs \
  web/static/portal-empleado/modulos/contratacion-temporal/seguimiento-cese.test.mjs \
  web/static/portal-empleado/modulos/contratacion-temporal/seguimiento-cese-justificante.test.mjs \
  web/static/portal-empleado/modulos/contratacion-temporal/seguimiento-incorporacion-acreditada.test.mjs
PYTHONDONTWRITEBYTECODE=1 PYTHONPATH=deploy/principal python3 -m unittest \
  deploy/principal/test_recorrido_copia_personal.py \
  deploy/principal/test_copia_local_personal.py \
  deploy/principal/test_recorrido_copia_conductor.py -v
```

Cubren contratos, transportes, errores, recuperación y composición con dobles;
no son evidencia HTTP contra J. No se ejecutan Go, ensayo SQL ni navegador por
el alcance de solo lectura y preparación, sin copia propia. La puerta completa
`scripts/verificar_calidad.sh` corresponde a la CI de la PR.

CONFIG NUEVA: ninguna. SQL a instalar: ninguno. Activación nueva: ninguna.
