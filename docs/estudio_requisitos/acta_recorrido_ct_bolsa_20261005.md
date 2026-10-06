# Acta del recorrido C1: Contratación temporal y Bolsa en un clon de la principal

Fecha: 5 de octubre de 2026, de 20:50 a 22:20 (hora de Madrid).
Código: `origin/main@dfcbab6ce`, compilado en local.
Datos: copia fría de la principal de cidonia `copia-prearranque-admin-20261005T122519Z` (SHA256 `8f860187…`), que es la principal del 5/10 a las 14:25, después de H12 y antes del arranque de administradores. Solo datos sintéticos.
Navegador: Playwright CLI con el Chrome del sistema, una sesión por certificado: centro solicitante, centro ratificador, RRHH e Intervención.

Las capturas están en [`docs/capturas/recorrido-ct-bolsa-20261005/`](../capturas/recorrido-ct-bolsa-20261005/). Las horas de las respuestas son de Madrid.

## Resumen

El recorrido llega hasta el registro de la aceptación del candidato. A partir de ahí se queda parado: la resolución de la aceptación no termina y Bolsa no la recibe. GINPIX y el cese se probaron con un expediente que la copia ya traía en «Nombramiento». El paso a Personal y la vuelta a Bolsa no funcionan con la configuración de la principal. Lo creado sobrevive al reinicio de la aplicación y de PostgreSQL.

Hay tres fallos que paran el recorrido en la principal tal como está:

1. Ningún consumo de autorización nuevo funciona sin su fila de origen (AD172). La tabla `configuracion_origen_consumos_v1` de la principal está vacía. Casi cada paso dio 403 o 503 la primera vez y hubo que añadir su fila. El kit CA26 ya lo avisaba; este recorrido da la lista completa de las que hacen falta (al final).
2. «Confirmar vía de cobertura» falla siempre por un error de SQL en CT166 (fallo SQL-2).
3. Con Bolsa B-BACK encendido, como en la principal, el llamamiento entero da 403 (fallo GO-1).

Dos fallos de pantalla tenían arreglo pequeño y van en PR aparte: #767 y #768.

## Cómo se montó el clon

- PostgreSQL 18.4 en contenedor propio (`--memory 2g`, datos en disco) con la copia fría. Certificado TLS nuevo y contraseñas nuevas para los LOGIN del clon.
- El material de identidad, claves y certificados de cliente es el de desarrollo que ya estaba en local (`traslado-20260905`). Las cuentas y perfiles de la copia lo reconocen. El material de la principal no se leyó.
- El entorno se rehízo a partir del código y de `deploy/principal/03_entorno.md`. Las DSN usan los mismos LOGIN que la principal.
- Correo de prueba Mailpit con TLS.
- Para arrancar con `main` hubo que dar los pasos que también hará falta dar en la principal al desplegar:
  - aprobar la provisión del perfil fijo RRHH «alta», de los dos perfiles del centro y de la provisión documental de Bolsa, con las huellas que da el aviso de arranque;
  - crear el archivo `auditoria-intentos.json` con su LOGIN y su fila en `configuracion_runtime_intentos`. Desde 548314741, Bolsa B-BACK no arranca sin el registrador de intentos. En la principal no existe ese LOGIN.
  - encender la organización en PostgreSQL (`VEC_PERSONAL_ORGANIZACION_POSTGRESQL=true`). Sin ella no existen las rutas de peticiones del centro (404).

Firma: el circuito de RRHH v2 exige la firma de remisión antes de fiscalizar. En el clon no hay GrxFirma ni validador, así que no se puede firmar. Desde la fiscalización se arrancó sin circuito ni registro de firma. Es el sustituto que prevé el plan mientras no esté el recorrido de dos firmas.

## Pasos

Se crearon tres expedientes. El 2026/91005 quedó bloqueado por el fallo SQL-1. El 2026/91006 quedó bloqueado en la selección del llamamiento (fallo ORQ-1). El 2026/91007 llegó hasta la aceptación. Para cada paso se indica la respuesta que dio al final y las que dio antes.

| Paso | Quién | Petición | Respuesta | Captura |
| --- | --- | --- | --- | --- |
| 1. Petición del centro | Solicitante | `POST peticiones-centro/operaciones` | 403 por terna; después 200. Recibo `recibo:peticion-centro:ba279695…`, 21:08:05 | [01](../capturas/recorrido-ct-bolsa-20261005/01_peticion_centro.webp) |
| 2. Ratificación | Ratificador | `POST peticiones-centro/operaciones` | 403 por terna; después 200, versión 2, 21:09:15 | [02](../capturas/recorrido-ct-bolsa-20261005/02_ratificacion.webp) |
| 3. Alta del expediente en RRHH | RRHH | `POST peticiones-centro/rrhh` | 403 y 503 por ternas. La reserva queda «Preparada»; «Completar registro» da 200. Expediente 2026/91005, 21:12:24 | [03](../capturas/recorrido-ct-bolsa-20261005/03_alta_rrhh_desde_peticion.webp) |
| 4. Análisis | RRHH | `POST analisis/registros` | 201 a la primera, versión 2 | [04](../capturas/recorrido-ct-bolsa-20261005/04_analisis.webp) |
| 5. Vía de cobertura (bolsa) | RRHH | `POST cobertura/decisiones` | 503 siempre (fallo SQL-2). Con un parche local en el clon, 201, versión 3 (2026/91006) | [05](../capturas/recorrido-ct-bolsa-20261005/05_via_cobertura.webp) |
| 6. Asignación a la unidad | RRHH | `POST asignaciones` | 503 por terna; después 201, versión 4. La ficha no avanza (fallo WEB-1, PR #767) | [06](../capturas/recorrido-ct-bolsa-20261005/06_asignacion.webp) |
| 7. Informe jurídico | RRHH | `POST informes-juridicos/preparaciones` | 503 por terna; después 201, versión 5 | [07](../capturas/recorrido-ct-bolsa-20261005/07_informe_juridico.webp) |
| 8. Fiscalización con reparo | Intervención | `POST fiscalizaciones/resultados` | 409 `firma_remision_pendiente` con firma; sin firma, 503 por terna y 201 al recuperar la operación. Desfavorable, versión 6 | [08](../capturas/recorrido-ct-bolsa-20261005/08_fiscalizacion_reparo.webp) |
| 9. Subsanación del reparo | RRHH | `POST subsanacion-reparos` | Solo aparece con una política privada de subsanación. 503 por terna; después 201, versión 7 | [09](../capturas/recorrido-ct-bolsa-20261005/09_subsanacion.webp) |
| 10. Informe nuevo y fiscalización favorable | RRHH e Intervención | `POST informes-juridicos/preparaciones` y `POST fiscalizaciones/resultados` | Primero 409 `informe_nuevo_pendiente`; informe 201 (v8); fiscalización favorable 201 (v9) | [10](../capturas/recorrido-ct-bolsa-20261005/10_fiscalizacion_favorable.webp) |
| 11. Inicio del llamamiento | RRHH | `POST llamamientos/seleccion` | Con B-BACK, 403 (fallo GO-1). Sin B-BACK, en 2026/91006: 409 por ternas de Bolsa y bloqueo (fallo ORQ-1). En 2026/91007: 200, 21:59:43 | [11](../capturas/recorrido-ct-bolsa-20261005/11_llamamiento_iniciado.webp) |
| 12. Comunicación del llamamiento | RRHH | `POST llamamientos/comunicaciones` | 403 por terna; después 201 con la misma clave. Queda «registrada localmente» con su intención de envío; no sale ningún correo | [12](../capturas/recorrido-ct-bolsa-20261005/12_comunicacion.webp) |
| 13. Aceptación del candidato | RRHH | `POST llamamientos/respuestas/registro` | 403 (terna oculta); 422 porque la hora de llegada estaba en el futuro y la pantalla no lo dice; después 201, 22:02:19 | [13](../capturas/recorrido-ct-bolsa-20261005/13_respuesta_aceptacion.webp) |
| 14. Resolución de la aceptación | RRHH | `POST llamamientos/resoluciones` | 403 y 403 (ternas ocultas); después 503 en todos los reintentos (fallo GO-2) | [14](../capturas/recorrido-ct-bolsa-20261005/14a_resolucion_respuesta_form.webp) |
| 15. Propuesta y resolución de nombramiento | RRHH | No se envió | No se llegó: depende del paso 14 | Sin captura |
| 16. GINPIX | RRHH | `POST confirmaciones-ginpix` | Expediente 2026/CT-000133 de la copia. Con la fecha de hoy, 409 `fecha_anterior_incorporacion`; con una fecha posterior al inicio (4/1/2027), 403 por terna y después 201, 22:08:54 | [15](../capturas/recorrido-ct-bolsa-20261005/15_ginpix.webp) |
| 17. Alta en Personal | RRHH | `GET /api/interno/…/incorporacion-personal-b2/plan/v1` | 404: Personal B2 no está compuesto (está apagado en la principal). Pantalla: «Ahora no se puede consultar la incorporación» | [17](../capturas/recorrido-ct-bolsa-20261005/17_cese.webp) |
| 18. Cese | RRHH | `POST ceses` | 403 por terna; después 201, versión 11, 22:09:53 | [17](../capturas/recorrido-ct-bolsa-20261005/17_cese.webp) |
| 19. Vuelta a Bolsa | Sistema | relevo de ceses | No llega a Bolsa (fallo BOL-1) | Sin captura |

## Reinicio

Se paró la aplicación, se reinició el contenedor de PostgreSQL y se volvió a arrancar la aplicación.

- Los recuentos de peticiones, versiones de expediente, resoluciones, selecciones, comunicaciones, cese y GINPIX son idénticos antes y después. No hay duplicados. Solo crecen los consumos de autorización, porque cada lectura deja el suyo.
- Bandeja del centro: 200, con las tres peticiones nuevas ([20](../capturas/recorrido-ct-bolsa-20261005/20_tras_reinicio_centro.webp)).
- Peticiones de los centros en RRHH: 200, con cuatro expedientes creados ([21](../capturas/recorrido-ct-bolsa-20261005/21_tras_reinicio_rrhh_peticiones.webp)).
- 2026/91007: comunicaciones 200 con la misma comunicación (recibo `recibo:b2924419…`, 22:00:26) y recibo de la respuesta 200 con el mismo recibo, fecha y justificante (`recibo:51a5bc3c…`, 22:02:19) ([22](../capturas/recorrido-ct-bolsa-20261005/22_tras_reinicio_exp3.webp)).
- 2026/CT-000133: versión 11, con los hitos de GINPIX y cese con sus horas ([24](../capturas/recorrido-ct-bolsa-20261005/24_tras_reinicio_exp133.webp)).
- Tras el reinicio, «Incorporaciones del centro» da 503 al solicitante y al ratificador. Antes daba 200 al ratificador. PostgreSQL no registra ningún error; la causa está sin determinar.

## Fallos

### Con arreglo en PR (pantalla, sin SQL)

WEB-1. Después de asignar la unidad (201), la ficha vuelve a ofrecer «Asignar expediente» y nunca enseña el informe jurídico. La ficha muestra la unidad como «Recursos Humanos» desde 32193f792, y la comprobación seguía esperando una referencia interna. PR #767.

WEB-2. Con la fiscalización favorable, al abrir el expediente no aparece «Iniciar llamamiento». La vista busca el hito `registrar_fiscalizacion` y el servidor lo guarda como `contratacion_temporal.fiscalizacion.registrar`. PR #768, que va encima de #767.

### Tocan SQL instalado (no arreglados)

SQL-1. Tras analizar una vacante sin fecha de fin, el expediente deja de abrirse (`expedientes/consultas` 404). `materializar_detalle_rrhh_v1` calcula «hay coste» con `jsonb_typeof(coste_previsto) = 'object'`, que da NULL cuando no hay coste. `canon_contenido_detalle_rrhh_v1` rechaza ese NULL. Los `WHEN OTHERS` encadenados lo convierten en «consulta RRHH rechazada». El 2026/91005 quedó sin poder abrirse. Arreglo: migración con `COALESCE(…, false)`.

SQL-2. «Confirmar vía de cobertura» falla siempre, en cualquier expediente: `column reference "x" is ambiguous` en `o404e_construir_lote_c1_v1`. Lo introdujo CT166: el alias `x` de `jsonb_array_elements` choca con la variable `x jsonb` de la función. Hace falta una migración que cambie el alias. En el clon se aplicó un parche local solo para poder seguir.

AD172. La tabla de orígenes de consumo de la principal está vacía y cada operación nueva necesita su fila. En tres sitios la falta de fila queda oculta y la API responde 403 «acceso denegado» sin pista: respuesta del candidato, consulta del justificante y resolución manual. Lo mismo pasa en la selección, que responde 409 «no reintentable». No es un fallo del código, pero sin estas filas la principal no puede tramitar nada. Lista al final.

### Código Go (no arreglados por tamaño o porque tocan seguridad)

GO-1. Con Bolsa B-BACK encendido, el autorizador de CT pasa al PDP común (`instalarDelegadoComun`, cd17d97a6). Ese catálogo no declara fronteras para el llamamiento, así que la consulta de comunicaciones, la selección, la comunicación, la respuesta y la resolución dan 403. La ficha oculta el formulario («No puede consultar estos llamamientos»). Sin B-BACK funciona. Arreglarlo exige declarar fronteras y perfiles del llamamiento en el catálogo común, o que el llamamiento no use el autorizador compartido. Es un cambio de seguridad que necesita diseño y revisión.

GO-2. La resolución de la aceptación queda «confirmado» en CT (`resolucion:5aa92953…`), pero la API sigue dando 503 en cada reintento y Bolsa no registra la aceptación. El error interno es genérico («comunicación de llamamiento no disponible») y no llega al puente con Bolsa. Causa sin determinar.

ORQ-1. Si la selección falla a medias entre CT y Bolsa, el expediente se queda bloqueado. Primer intento: falta la fila `bolsa.orden.preparar`. Segundo intento, con la clave nueva que ofrece la pantalla: Bolsa confirma la necesidad y falla en `bolsa.llamamiento.abrir`. A partir de ahí, «necesidad Bolsa ya confirmada». Reanudar con la clave del segundo intento da `seleccion_no_disponible` («terminal O6 no confirmado»). El botón «Preparar clave nueva» invita a repetir con otra clave, que es justo lo que deja el expediente sin salida.

BOL-1. La vuelta a Bolsa tras el cese no funciona. En la principal no hay LOGIN para `VEC_BOLSA_CESE_CT_DATABASE_URL`: el rol `vec_bolsa_llamamientos_relevo_cese` no tiene miembros. Además, el relevo exige B-BACK encendido, que rompe el llamamiento (GO-1). En el clon, con un LOGIN propio, el relevo de contratos entregó un contrato, pero el de ceses falla cada 30 segundos con «candidato de cese no resuelto en B13». La persona no vuelve a la bolsa.

### Configuración que falta en la principal

- Paso a Personal: sin B2 compuesto no existe. Es la configuración actual de la principal.
- Subsanación de reparos: solo existe si hay una política privada (`VEC_CT_SUBSANACION_POLITICA_FILE`). Sin ella, tras el reparo la ficha no ofrece nada y «Ir al trámite» lleva a la auditoría. Al añadirla, el perfil fijo RRHH «alta» vuelve a quedar pendiente de provisión.
- Firma: con el circuito de RRHH v2, Intervención no puede fiscalizar hasta que el informe esté firmado.

## Observaciones de uso (para C11)

- Intervención tiene que teclear la referencia interna del expediente y su versión. No tiene bandeja ni ve el informe que fiscaliza.
- El formulario de llamamiento enseña la referencia interna, la versión esperada y una «clave de operación» que hay que generar con un botón.
- La vista RRHH de peticiones de los centros enseña el motivo como «—» y la categoría, el centro y el contacto con su referencia interna. También muestra el panel «Incorporaciones del centro» con un error 403.
- «Ir al trámite de esta fase» lleva a veces a «Cancelación del expediente» o a la auditoría en lugar del formulario que toca.
- Al cambiar la fecha de fin en el análisis, el formulario se redibuja y borra la jornada, la retención de crédito y las observaciones. Los formularios de GINPIX y cese también se vacían después de cada error, incluido el justificante adjunto.
- Tras registrar el análisis, el expediente sigue en «Fase 1 de 8: Solicitud».
- El análisis no trae los datos de la petición del centro (modalidad, categoría, fechas): hay que volver a escribirlos.
- La confirmación de la vía de cobertura usa un diálogo del navegador con la referencia interna. El texto dice «avanzará a asignación de unidad» y el expediente pasa a «Gestión de bolsa».
- El aviso de GINPIX dice «La fecha de efecto es anterior al inicio de la incorporación». GINPIX solo se acepta con una fecha igual o posterior al inicio, aunque sea futura.
- La etiqueta de la causa de cese enseña la clave interna «comunicacion_reincorporacion».
- Tras registrar un cese con efecto en 2027, la ficha ya marca «Cesado».
- Al abrir una ficha, varias consultas a la vez fallan a veces con un conflicto de serialización (40001) al publicar la autorización de un perfil fijo: «No se ha podido consultar la cancelación», «CT140: consulta transitoria». Al reintentar funcionan.
- La retención de crédito no es obligatoria para presentar la petición ni para seguir (encargo C4).

## Filas de origen (AD172) que pidió el recorrido

Todas con proceso `vec-rrhh` y canal `interna_corporativa`. Las inserta el DBA como `vec_autorizacion_atestada_v3_propietario` (ver `ORIGEN_CONSUMOS_AD172.md`).

| LOGIN | Audiencia | Operación |
| --- | --- | --- |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.peticion_centro.consultar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.incorporacion.consultar_centro` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.peticion_centro.presentar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.peticion_centro.ratificar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.peticion_centro.rrhh.consultar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.peticion_centro.rrhh.entregar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.solicitud.crear` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.firmas_documento.consultar.v1` | `contratacion_temporal.documento.firmas.consultar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.unidad.asignar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.informe_juridico.generar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.fiscalizacion.registrar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.subsanacion_reparos.registrar` |
| `vec_bolsa_llamamientos_desarrollo` | `vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1` | `bolsa.orden.preparar` |
| `vec_bolsa_llamamientos_desarrollo` | `vec_bolsa_llamamientos.confirmar_integracion_desarrollo.v1` | `bolsa.llamamiento.abrir` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.llamamiento.comunicacion.registrar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.llamamiento.respuesta.registrar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.llamamiento.respuesta.consultar_justificante` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmar_alta_atestada.v1` | `contratacion_temporal.llamamiento.respuesta.validacion_manual.registrar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.confirmacion_ginpix.v1` | `contratacion_temporal.ginpix.confirmar` |
| `vec_ct_o207_runtime` | `vec_contratacion_temporal.cese.v1` | `contratacion_temporal.seguimiento.cesar` |

Faltan las de los pasos a los que no se llegó: aceptación en Bolsa, propuesta y resolución de nombramiento, incorporación.

## Límites

- No es la principal: el material y las contraseñas son del clon y el código es `main`, no el binario desplegado.
- La firma se sustituyó por su ausencia desde la fiscalización. Nada de este recorrido acredita una firma.
- El parche de CT166 solo existe en el clon. Las funciones de detalle se instrumentaron un momento para ver el error y se dejaron como estaban (se cotejó su texto).
- No se envió ningún correo real. La comunicación queda como intención de envío.
