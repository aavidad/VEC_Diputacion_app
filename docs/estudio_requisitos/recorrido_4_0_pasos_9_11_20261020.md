# Recorrido del caso 4.0: pasos 9 a 11 (20/10/2026)

Tarea F1-25. Se recorrieron en Chrome los pasos 9, 10 y 11 del
[caso 4.0](../producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación)
en una copia local creada desde `main` (400f3c059) con `--modulos` y la orden
`firma`. Todos los datos son ficticios y no se tocó cidonia.

## Resumen

| Paso | Persona | Respuesta | Resultado | Estado |
| --- | --- | --- | --- | --- |
| 9 | Carmen (RRHH) | «Llamar a la siguiente persona»: `POST /llamamientos/seleccion` 409 `seleccion_no_disponible` | Nada cambia | Falla (copia) |
| 9 | Carmen | «Abrir llamamiento en Bolsa»: `POST /api/vec/bolsa/llamamientos/emisiones` 201 y `POST /expedientes/vinculos-bolsa` 201 | Llamamiento emitido y vinculado; la fase sigue en Fiscalización, v6 | Funciona, pero no sale el correo |
| 10 | Antonio | `GET /api/vec/bolsa/mi-bolsa` 403 | «No tiene acceso a esta área personal» | Falla (copia) |
| 11 | Carmen | sin respuesta que resolver | La ficha dice «Sin respuesta registrada» y no ofrece resolver | Bloqueado por el paso 10 |

Salvo las de Bolsa, las rutas cuelgan de `/api/vec/contratacion-temporal`.
A 1440 y a 390 píxeles no hubo desbordamiento horizontal. Tampoco hubo
cookies ni almacenamiento web.

## Cómo se llegó al paso 8

1. Pasos 1 a 3 en Chrome con un expediente nuevo: Lucía presentó la petición
   (200), Manuel la ratificó y Carmen creó el expediente 2026/95125 (201).
2. Paso 4 en ese expediente: «Registrar análisis» devolvió 503
   `servicio_no_disponible`. Es el fallo P1 de abajo y el camino se cortó ahí.
3. Se tomó el atajo que ya funciona: el expediente de la semilla 2026/93001
   (Administrativo, Transformación Digital, v5, informe jurídico hecho).
4. Paso 7b: el técnico y Jefatura firmaron el informe definitivo con GrxFirma
   0.0.111 (`POST /firmas-documento/registro-vec` 201 y 201).
5. Paso 8: Intervención registró la fiscalización favorable (201, v6,
   `recibo:ct-fiscalizacion:40afe193…`). La ficha pasó a «Fase 4 de 8:
   Fiscalización» y dice «Continúe en el apartado Llamamiento».

## Paso 9, con detalle

Primer intento, por la ficha:

- «Llamar a la siguiente persona» → diálogo de confirmación → aceptar.
- Respuesta 409 `seleccion_no_disponible`. La selección desde la bolsa
  constituida necesita la consulta de Organización histórica de Personal
  en el proceso combinado, con el perfil nominal del técnico de RRHH
  ([guía](../../deploy/principal/organizacion_historica_combinado_20261015.md)).
  La copia no preparó ese montaje (C1).

Segundo intento, por «Abrir llamamiento en Bolsa»:

1. «Seleccionar esta bolsa» (Administrativo, 12 personas, 12 disponibles).
2. En la lista de candidatos se marcó una persona.
3. En «Configurar llamamiento» la referencia, la categoría, el centro y la
   fecha ya venían rellenos. Se puso la modalidad «Acumulación de tareas», el
   plazo «2 días hábiles» y una descripción.
4. En «Revisar y enviar» se marcó la confirmación y se pulsó «Emitir
   llamamiento»: 201, «Llamamiento emitido. Estado: emitido, pendiente de
   respuesta».
5. En la ficha, «Llamamiento de esta petición» → «Vincular al expediente»:
   201, `recibo:ct:bolsa:4d6addb5…`. La ficha muestra «Llamamiento vinculado»,
   «Contacto registrado: Correo no enviado» y el justificante del contacto. La
   fase no cambia: Fiscalización, v6.

Se emitieron dos llamamientos. El primero fue para la persona que la lista
llama «Antonio Reyes Álvarez», con el número de orden 5
(`recibo:llamamiento:e17e8fbf…`). Después se vio que el certificado de
Antonio está vinculado a otra participación, la del orden 1, que la lista llama
«Isabel Ramírez Vargas» (C2). Se emitió un segundo llamamiento para esa
participación (`recibo:llamamiento:99be5b48…`), que es el que se vinculó al
expediente.

El buzón de pruebas no recibió ningún correo. Ninguna de las dos
participaciones tiene datos de contacto en Bolsa, y la emisión devuelve
`resultado: no_enviado` (C3).

## Pasos 10 y 11

- Paso 10: Antonio entra en `/area-personal/` con su certificado. `GET
  /api/vec/bolsa/mi-bolsa` responde 403 y la pantalla dice «No tiene acceso a
  esta área personal». En el registro del arranque aparece «Mi Bolsa sin
  asignación consumible … estado=pendiente_provision». La asignación de Mi Bolsa
  del material apunta al candidato antiguo, no al que el guion de datos vincula
  hoy a Antonio, y la aplicación no la restaura (C4).
- Paso 11: sin respuesta de Antonio no hay aceptación que resolver. La ficha
  muestra «Respuesta: Sin respuesta registrada».

## Después del reinicio

Se pararon la aplicación y PostgreSQL y se arrancaron de nuevo.

- Veinte recuentos (versión de 2026/93001, contactos, datos de contacto y
  tablas de llamamientos, vínculos, firmas y fiscalización) son idénticos antes
  y después.
- La ficha de 2026/93001 muestra lo mismo: Fiscalización, llamamiento vinculado
  a las 3:08 y «Correo no enviado».
- Mi Bolsa de Antonio sigue en 403.

## Fallos del producto

### P1. El análisis falla en todo expediente nuevo cuando la copia tiene firma (tarea nueva)

- Paso 4, «Registrar análisis» en 2026/95125: 503 `servicio_no_disponible`.
  Registro: `operacion=registro_analisis_rrhh … centinela=analisis_no_disponible`.
- La orden `firma` instala el circuito RRHH v2, y los expedientes nuevos nacen
  con circuito (`flujo:ct:rrhh:20261002` v2). Con circuito, el análisis exige
  una fuente de evidencias (`ports.FuenteEvidenciasCircuitoRRHH`) que acredite
  la petición firmada y la autorización de RRHH. Ningún adaptador la implementa
  y nadie llama a `GobernarEvidenciasCircuitoRRHH`
  (`internal/modules/contrataciontemporal/application/operacion_analisis_circuito_rrhh.go`).
- Consecuencia: con la firma compuesta, ningún expediente nuevo pasa del paso
  4. El informe de los pasos 1 a 8 no lo vio porque esa copia no tenía firma.
- Es un fallo grande y no se programó. Pide una tarea propia: componer esa
  fuente o no adjuntar el circuito mientras no exista.

### P2. La revisión del llamamiento decía «1 seleccionados» y «1 candidatos» (corregido)

Se corrigió en la PR #1278, con prueba y revisión de usabilidad (GO).

### P3. El asistente deja llamar a la persona del orden 5 sola

La pantalla dice «Los candidatos se llaman siempre por el orden de la bolsa»,
pero dejó emitir un llamamiento solo para el número 5 con los cuatro primeros
disponibles. Conviene que RRHH confirme si eso se permite. Si no se permite, es
una tarea nueva.

### P4. Al vincular, el desplegable no dice a quién se llamó

«Llamamiento de esta petición» muestra solo «2026/93001 · 20 oct 2026, 3:08».
Con dos llamamientos de la misma referencia no se sabe cuál es el de cada
persona. Falta el nombre o el orden de la persona llamada.

### Observaciones menores

- Al vincular apareció una vez «No se ha podido consultar la cancelación de
  este expediente», con botón «Reintentar». Parece la misma lectura cancelada
  al repintar la ficha que estudian F1-93 y F1-94; solo se anota.
- Al abrir el asistente, la consola muestra `GET /api/vec/bolsa/cargas-convoca`
  401 y `GET /api/vec/bolsa/llamamientos/plazo-respuesta` 503. La pantalla
  funciona igual.

## Fallos de la copia

- C1. Falta la consulta de Organización histórica: la selección de la ficha
  devuelve 409. La conexión por puerto en el mismo proceso requiere el montaje
  existente y un perfil nominal del técnico que actúa; la guía citada explica
  perfiles, motivos y publicación de capacidad. Pide tarea propia.
- C2. El nombre que la lista da a la participación vinculada al certificado de
  Antonio (orden 1) es «Isabel Ramírez Vargas», y otra participación (orden 5)
  se llama «Antonio Reyes Álvarez». El guion de datos vincula a Antonio con el
  orden 1, pero no cambia el nombre que muestra el acta. Es un ajuste de datos.
- C3. Las participaciones de los órdenes 1 y 5 no tienen datos de contacto, así
  que el aviso no sale al buzón. Es otro ajuste de datos.
- C4. La asignación de Mi Bolsa del material apunta al candidato antiguo y no
  se restaura. Antonio no puede entrar en Mi Bolsa. Hace falta reprovisionarla
  para el candidato actual, por el camino de provisión aprobada y no con filas
  escritas a mano. Encaja con F0-28.
- C5. En el material privado hubo que actualizar dos huellas que la orden
  `firma` comprueba: la revisión candidata del kit H6 y la del montador de
  `vec-admin`. Es material fuera de Git.
- C6. Con `--portal-externo`, Mi Bolsa pasa a la persona externa (Elena) y no a
  Antonio. Esa copia se descartó y se creó otra sin esa opción.

## Qué falta para cerrar los pasos 9 a 11

1. P1 para recorrer el caso entero desde la petición con firma.
2. C1 para que «Llamar a la siguiente persona» funcione desde la ficha.
3. C2 y C3 para que el aviso llegue a Antonio por correo.
4. C4 para que Antonio responda en Mi Bolsa (paso 10) y Carmen resuelva la
   aceptación (paso 11).

## Capturas

En [recorrido_4_0_pasos_9_11_20261020](recorrido_4_0_pasos_9_11_20261020/):

- `04_analisis_503.png`: el análisis rechazado en el expediente nuevo.
- `09_llamar_siguiente_409.png`: la selección desde la ficha.
- `09_asistente_candidatos.png`, `09_asistente_revision.png` y
  `09_llamamiento_emitido.png`: el asistente de Bolsa.
- `09_llamamiento_vinculado.png`: la ficha con el llamamiento vinculado.
- `09_ficha_390.png` y `09_asistente_390.png`: las mismas pantallas a 390
  píxeles.
- `10_mi_bolsa_403.png` y `10_mi_bolsa_403_390.png`: Mi Bolsa de Antonio.
- `reinicio_09_ficha.png`: la ficha tras reiniciar.
