# Guía de recorrido y recibos conservados de VEC

## Recorrido vigente — 10 de septiembre de 2026

### Backend y panel integrados en código; runtime pendiente

El backend confirmado es
`76a2b7c13309bc0526e45d585c2fdf7ca77aecf6`. Sobre la base `3375d527`, este
corte conecta consulta, proyección, HTTP y panel `ORIGINAL`.

Cuando el montaje y la autorización de runtime estén disponibles, la consulta
prevista será:

```text
GET /api/vec/contratacion-temporal/incorporaciones-ejercicio/seguimiento?expediente_ref=<referencia_sintetica>
```

El campo `original_incorporacion` identifica el alta histórica que abrió el
seguimiento. No debe interpretarse como el estado actual. Use una referencia
sintética existente y un perfil autorizado; esta guía no publica credenciales
ni referencias privadas.

La evidencia actual es de código: dos revisiones independientes Astra/high
emitieron `GO`; seis pruebas focales Go están verdes en cinco paquetes; Node
está verde `7/7`, incluido el montaje principal.

El primer arnés Chromium usó CSS inventado y no acreditaba el CSS de producto.
Con el índice candidato de `1385b134`, los doce CSS reales y referencias
hexadecimales completas de 64 caracteres, la siguiente ejecución detectó
`scrollWidth=834` en el viewport de 390 px; escritorio quedó correcto. El
hunk único del snapshot CSS
`752b025e99ca05335cd1053b7c8bd2dc3780358977d3191ddfd4d5a896013583`
corrigió el ancho. `cssfix3` midió `1440/1440` y `390/390`, pero contenía
una navegación inventada superpuesta, por lo que no acreditaba ausencia de
obstrucción.

El arnés final `cssfix4` usa el wrapper mínimo real del componente CT, sin
`nav` ni `aside`, los doce CSS reales y transporte interceptado. El
resultado fue `PASS`: ancho de documento igual al viewport a 1440 y 390 px,
una consulta, referencias completas y cero errores JavaScript, consola,
almacenamiento o hijos tras destruir la raíz. En ambos anchos,
`elementFromPoint` devolvió `H3` sobre el título y `BUTTON` sobre el
botón; los dos figuran visibles y sin obstrucción. Dirección inspeccionó la
captura de 390 px: es legible, no muestra superposición y conserva el botón
completo.

Artefactos:

- `arnes-seguimiento-candidata-cssfix4.mjs`, SHA256
  `cd68fe9accce94267e2d5829546ef197a8c1d12c38ac8ba59a3861aea6338fad`.
- `resultado-candidata-cssfix4.json`, SHA256
  `149ae2ab956667d624834169b66dd987f18b2f6d60dc821dfff69dea49e937e2`.
- `seguimiento-candidata-cssfix4-390.png`, SHA256
  `38b3f50681b7f39fdb0c41c6f40207f81df7c44dfd252cb3d658903a292a2c66`.
- `seguimiento-candidata-cssfix4-1440.png`, SHA256
  `8f92badaa69f1b2c0b344a0b218c12d0a53d8f942f9225fc0576ce1a2311b91c`.

Esta evidencia acredita el componente CT con transporte sintético interceptado.
No acredita el shell completo ni un E2E de runtime. La
campaña global `go test ./...` no se ejecutó porque la revisión automática
rechazó su alcance masivo.

Auth13 `ef6704a6` no está instalado, por lo que la recuperación `GET` sigue
bloqueada en runtime. GINPIX `e2831250` está integrado sin prueba sobre un
recibo recuperado. No registre otra incorporación ni atribuya una respuesta
nueva a GINPIX. Permanecen abiertos los objetivos 11, 12, 13 y 14:
recuperación, GINPIX en runtime, anotación/cierre y recorrido conjunto. En el
objetivo 6, el inicio y la política siguen pendientes de elección y
acreditación.

El servidor privado conserva 52 expedientes sintéticos. Se pueden recorrer
cinco pasos completos y partes del sexto y séptimo; no es la aplicación
completa ni un entorno de producción. El
[manual de Sistemas](docs/manual_sistemas/README.md#entorno-privado-vigente)
es la referencia de acceso: escucha `127.0.0.1:8443`, con certificado cliente
de pruebas. Dirección comunicó `GET 200` del portal a las 10:08 UTC.
No hay URL externa; un túnel anterior en `18443` no acredita acceso actual.

Para presentar el trabajo conservado:

1. Sistemas prepara el acceso privado y el perfil de pruebas. Abra
   `/portal-empleado/` en ese acceso, no en un `localhost` sin preparar.
2. En Contratación temporal, busque el expediente indicado por el operador
   y abra su detalle. No cree otra solicitud para volver al caso.
3. El caso de resolución manual sintética está en `v8`: consulte el recibo
   ya confirmado. Su registro `201` y recuperación `200` con la misma clave
   tras reiniciar aplicación y PostgreSQL ya están acreditados, sin duplicado.
4. Los seis botones descargan los borradores de la propuesta histórica `v7`:
   informe definitivo, resolución, diligencia, toma de posesión, notificación
   y comunicación al centro. No son documentos firmados. El caso original
   `v7` permanece intacto; no hay que validarlo para enseñar el avance.
5. Si la respuesta es incierta, conserve pestaña, datos y clave originales
   y pida comprobar el resultado. No repita campañas de reinicio, PDF o SQL
   para seguir esta guía.

La canónica `c6359f98` contiene piezas de incorporación ya ensambladas;
no reconstruirlas. CT70–85, trece pools y tres capacidades con catálogos y
definición están provisionados, sin precargas de negocio. El 10 de septiembre,
a las `13:07:06.614186Z`, el formulario obtuvo `GET 200` y `POST 200` reales
con sus dos casillas confirmadas, sin errores JS ni desbordamiento a 1440 o
390 px. Registró solicitud `7`, expediente `8`, seguimiento `0→1`, periodo
`2027Q1` y recibo
`2bc3d281379b33ed532825768a29001d19cc7af96c58cead9badb4d13fa6290b`.

La escritura dejó una alta de Personal, una relación, una ocupación, una
auditoría y un outbox; en CT, una incorporación, una auditoría, un outbox, una
raíz y dos estados. `firma_oficial=false` y
`eficacia_administrativa=false`. Tras reiniciar aplicación y PostgreSQL, la
lectura devolvió `GET 503` por restauración histórica Auth12: la recuperación
no está acreditada y todavía no hay ciclo completo. Auth13 tiene dos `GO` y
regresión verde en `ef6704a6`, pendiente de instalación. No repetir ahora el
registro. El siguiente corte sigue siendo GINPIX: modelo, mapeo y codificador
existentes; descarga V2 y cierre visible pendientes.

### Cómo leer los antecedentes

Los apartados fechados siguientes conservan hechos, recibos y cifras de cada
prueba. No cambian de fecha ni se convierten en pruebas del 10 de septiembre.
Las recetas locales/remotas antiguas y el apéndice de recreación no son
instrucciones para operar o reconstruir la instancia conservada.
Para uso funcional, siga los manuales de
[usuario](docs/manual_usuario/manual_portal_bolsas.md) y
[RRHH](docs/manual_rrhh/README.md); para continuar el código, el
[manual del programador](docs/manual_programador/README.md).

<a id="acceso-vigente-en-openclaw--6-de-septiembre-de-2026"></a>

## Acceso en OpenClaw — referencia histórica del 6 de septiembre de 2026

El acceso remoto de aquel corte usaba el túnel abierto por el operador en
`localhost:18443`. Para la vista de RRHH abra
`https://localhost:18443/portal-empleado/peticiones-centro/?vista=rrhh` con el
certificado sintético de identificación/autenticación de RRHH. Ese certificado
identifica a quien opera la vista; no firma documentos ni acredita firma legal.

El arranque de la aplicación y PostgreSQL privados corresponde a Sistemas. Los
apartados históricos inferiores que describen un arranque local en el puerto
`8443` o lanzadores locales se conservan como referencia y **no deben ejecutarse
para continuar el acceso remoto vigente**. Esta guía no contiene secretos ni la
ubicación del material privado.

## Bandeja de expedientes — incremento web del 7 de septiembre

Antecedente de esa prueba: su alcance local no describe el estado del servidor
del 10 de septiembre. No se repite ni se amplía aquí su evidencia visual.

Este incremento está preparado y probado en sandbox; aún no está acreditado
en el navegador servido. No requiere publicar una URL externa. Se conserva el
acceso privado anterior bajo Sistemas; no arranque los lanzadores históricos.

En **Contratación temporal**, consulte la bandeja de expedientes (distinta de
la bandeja de peticiones del centro):

El detalle del incremento local muestra **Período previsto** como fechas legibles
(por ejemplo, «4 sept 2026 — 31 dic 2026»), sin desplazar el día por la zona
horaria del navegador. Esta presentación está comprobada localmente; no afirma
que se haya instalado aún en la aplicación servida.

1. Aplique los filtros de búsqueda, estado y fase. El número mostrado corresponde
   a esa página, de hasta 100 expedientes; no representa un total global.
   Sin coincidencias, **Mantener la fase aplicada** conserva la fase elegida.
   Seleccione **Todos** o pulse **Limpiar** para retirarla expresamente.
2. Pulse **Página siguiente** para consultar más resultados. Los filtros se
   mantienen. Abra un expediente desde la página en la que aparece.
3. Use **Reiniciar consulta** para volver a la primera página con los mismos
   filtros. Actualizar también comienza de nuevo; no hay navegación hacia atrás
   dentro de una consulta anterior.
4. Si se interrumpe la consulta, no significa que no haya más expedientes.
   **Reiniciar consulta** o **Reintentar** comienza desde la primera página;
   no reenvía la continuación interrumpida. Reaplique filtros si desea cambiarlos.
5. Si el expediente cambia de versión al actualizar, se cierra el detalle anterior
   y aparece un aviso. Vuelva a abrirlo desde la bandeja para consultar sus datos
   actualizados. Si la versión no cambia, se conserva la selección. Un aviso de
   resultado indeterminado sigue vigente: el refresco no confirma una actuación.

La consulta no crea altas ni resoluciones. El 8 de septiembre se comprobó
también la interfaz en Chromium local a 1440 y 390 píxeles, con transporte
sintético; no es el recorrido de la instancia privada. Con teclado, al paginar
o aplicar filtros el foco vuelve al formulario de filtros; Tab permite seguir
desde el buscador. Al abrir un detalle se enfoca la tarea, si existe, o su
encabezado. Tras un error de continuación permanecen visibles las filas
anteriores y el aviso; use Reiniciar consulta para obtener una lectura nueva.
El recorrido funcional del objetivo 10 sigue pendiente y ningún certificado
de autenticación equivale a firma legal.

## Resolución manual de ejercicio — formulario comprobado localmente el 8 de septiembre

Antecedente local conservado. La comprobación privada posterior del
[corte vigente](#recorrido-vigente--10-de-septiembre-de-2026) ya acreditó
registro y recuperación; el resultado de aquella prueba local no se reescribe.

Preparado en el código de trabajo, **todavía no validado en la instancia privada**.
No se presenta como firma oficial, eficacia administrativa ni incorporación en
Personal. La publicación de una URL no es requisito para esta comprobación local.

1. Abra un expediente que tenga propuesta de formalización. El formulario carga
   sus referencias y versiones desde la consulta de preparación; no las invente.
2. Revise número, fecha y motivo de la resolución y las dos confirmaciones de
   revisión y ejercicio manual. Pulse **Registrar validación manual** y confirme
   expresamente la operación.
3. Si el servidor permite corregir los datos, estos se conservan y el foco vuelve
   al formulario. Si movió el foco fuera mientras esperaba, permanece allí.
4. Ante resultado incierto, conserve la pestaña y los datos originales. El reintento
   conserva la misma operación; un conflicto consulta el recibo histórico, sin
   registrar automáticamente otra resolución. No interprete ese recibo como
   confirmación de un contenido distinto enviado por error.
5. Con recibo confirmado, la reapertura muestra la historia en solo lectura. Una
   resolución manual de ejercicio no habilita por sí sola una incorporación legal.

Comprobado en Chromium a 1440/390 con módulos reales y transporte sintético:
validación HTML, 422 corregible, recibo/reapertura, red incierta, recuperación de
409 y cambio de expediente sin mezclar respuestas. Sin errores JavaScript ni
almacenamiento del navegador. Quedan pendientes identidad nominal, PostgreSQL
y recuperación tras reinicio en el entorno privado; no ejecute los lanzadores
históricos para suplir esas comprobaciones.

## Petición del centro y ratificación

Disponible en la instancia principal preparada, con identidades ficticias
separadas para solicitante, ratificador y RRHH. Use para cada actuación la
identidad y el certificado de autenticación que correspondan; el certificado
de RRHH no sustituye a las identidades del centro.

1. Como solicitante, pulse **Presentar petición**. Complete contacto, categoría,
   grupo, motivo, detalle y fechas. El centro corresponde a su identidad.
2. Revise y confirme. Aparece el recibo y la petición pendiente de ratificación.
3. En la ventana del ratificador, recargue la bandeja y pulse **Revisar**.
   Compruebe quién solicita, cargo, necesidad, crédito y documentación.
4. Abra la ratificación, escriba un motivo y marque la confirmación. Se guarda
   otro recibo y la versión2. El solicitante no puede ratificarse a sí mismo.

Demostrado el 6 de septiembre: petición `abd01fd9…`, presentación y ratificación
`200` desde los formularios. Tras reiniciar aplicación/PostgreSQL, ambos reintentos
recuperan los recibos originales y sus fechas; bandeja ratificada `200`, una
petición, dos revisiones y dos eventos, sin alterar los 51 expedientes previos.
Si aparece **Resultado pendiente**, conserve la pestaña y use **Reintentar la
misma operación**: la clave vive en memoria, no en almacenamiento del navegador.

### Entregar la petición ratificada a RRHH

1. Con la identidad y el certificado de identificación de RRHH, abra
   `/portal-empleado/peticiones-centro/?vista=rrhh` en el acceso privado
   preparado por Sistemas para esta sesión.
2. Pulse **Revisar** en la petición ratificada y compruebe centro, responsables
   y necesidad. Si aún no consta el alta, pulse **Crear expediente en RRHH**
   o **Completar registro**, según el estado.
3. Marque la confirmación expresa, pulse **Crear expediente en RRHH** y conserve
   el recibo. Si ya consta **Expediente creado**, **Revisar** muestra el recibo
   histórico: no hay que crear ni confirmar otra alta.
4. Ante **Resultado pendiente**, conserve la pestaña y pulse **Reintentar la
   misma operación**. Si reabre la página, consulte el recibo cuando conste
   **Expediente creado**; solo si sigue **Preparada para crear expediente** use
   **Completar registro**. El POST repetido de la comprobación técnica no es
   un botón de repetición para entregas ya confirmadas.

Caso acreditado: expediente
`expediente:ct:4ff4285d7ae5d6c4fb34d199a942d7c7d66ba4274ebbaab44fc8296089e8c5cf`,
número `2026/CT-d06f98d5506ded3ee3b8a7d34d867b1e`, versión `1`; recibo
`recibo:ct-alta:9e45c28d21cadddc4f0bfd548249c555b0e0e3c55ce04b7d14052a65eec46467`,
creado `2026-09-06 17:30:26.404646Z`.

Dirección acreditó el caso de recuperación con Chrome: después del reinicio,
`GET 200` y `POST 200`, recibo entero idéntico. El primer `POST 503` había
creado el alta; el reintento la enlazó sin duplicar. Quedaron 52 expedientes,
52 versiones, 2 revisiones de petición, 1 reserva, 1 confirmación y 1 evento de
entrega, con huellas anteriores intactas. En 1440/1024/390 el ancho de scroll
coincide con el ancho visible, la tabla desplaza internamente y se observaron
0 errores JS, cookies o almacenamiento web.

El certificado anterior **identifica y autentica** a RRHH para operar; no es
una firma electrónica o documental, no firma una resolución y no acredita una
firma legal. Este puente tampoco aumenta la métrica: siguen cinco pasos
completos más partes del sexto y séptimo.

AD3-24 y CT68 ya están instaladas con historia: **no reaplique `DOWN` ni `UP`**.
La definición SQL estructural se ajustó una sola vez sin perder tablas. Para
restaurar hacen falta también las ACL de base y los 43 tipos de fila, además
del dump; el procedimiento privado está fuera de Git. No use esta guía para
reconstruirlo.

La edición del organigrama no configura por sí sola quién solicita, ratifica o
actúa como RRHH.

## Centros y organización de referencia

En la bandeja de Contratación, abra **Centros y organización de referencia**,
o visite `/portal-empleado/organizacion/` por el túnel remoto vigente con el
navegador de desarrollo ya preparado. Puede buscar por denominación o código,
filtrar por tipo y consultar la adscripción y página de la fuente.

La versión inicial recoge los 41 centros de la RPT fechada el 7 de mayo de
2026, sus 14 agrupaciones de encabezados y once puestos cotejados de
Transformación Digital. Es preparación: no contiene el organigrama funcional
completo, ocupantes ni permisos de ratificación. La conexión multicientro del
alta sigue pendiente. Este catálogo no aumenta los cinco pasos completos.

El editor persistente se demostró en la instancia principal del corte siguiente.
Seleccione **Nueva unidad** o **Editar**, indique denominación, tipo, adscripción
y motivo; pulse **Revisar cambio** y después **Confirmar cambio**. Una unidad
nueva recibe clave técnica, no un código oficial. Los cambios se marcan
**Cambio local** y no conceden permisos ni alteran expedientes.

Recorrido demostrado el 6 de septiembre: alta de un cargo sintético bajo
Transformación Digital y edición de su denominación, ambos `200` desde Firefox.
Se interrumpió de forma sintética la respuesta del segundo guardado, después
del `200` real. La pantalla mantuvo el cuerpo y bloqueó nuevas operaciones.
Tras reiniciar aplicación y PostgreSQL principal, **Reintentar el mismo cambio**
recuperó el recibo original, misma fecha y `replay_confirmado`, sin duplicar.
Si aparece ese aviso, no cierre ni recargue la pestaña: el pendiente vive en
memoria, no en almacenamiento web.

Resultado conservado: **67 unidades** —66 de referencia y un cargo sintético—,
**3 revisiones**, **2 cambios**, **2 eventos** y los **51 expedientes** intactos.
El recibo de edición es `recibo:c1087e22-e2e4-46e3-9255-78863cbeb12c`,
fecha `2026-09-06T13:18:41.385823Z`, revisión3; clave de recuperación
`fc51ace8-5e71-46be-b8c8-e4f91f29b0e1`. La revisión inicial permanece inmutable.
Una revisión antigua con otra clave devuelve `409`; Intervención `401`;
cabecera Cookie `400`. La edición no está instalada en la base secundaria.

El [manual de Sistemas](docs/manual_sistemas/README.md#organización-de-referencia-configurable)
explica la fuente única y la inicialización. En principal ya están instaladas
las migraciones de autorización22 y Contratación66: **no reaplicar ni revertir**.
Use el [acceso vigente](#recorrido-vigente--10-de-septiembre-de-2026)
descrito al principio. Los lanzadores locales descritos más abajo son
históricos y corresponden a Sistemas; no los ejecute para continuar en remoto.

## Recorridos locales del 5 y 6 de septiembre — historial conservado

No ejecutar sus arranques ni sus pruebas de escritura para operar el servidor
actual. Los recibos, cifras y huellas siguientes mantienen su fecha original.

Primer PDF publicado: `5c57b29f`. El cierre de bandeja,
detalle y análisis corresponde a
`b2effbaf09fd4ad8477bf42c56e4615ff52d0c62`, con el corrector SQL `13f7a92`
y la interfaz integrados por avance directo. El desarrollo utiliza la misma rama
`trabajo/ct-app-llamamiento-b4a-20260905`, en el worktree local
`.worktrees/ct-app-llamamiento-b4a-20260905`. Las instancias de desarrollo
remotas están detenidas y conservadas; no se programa en dos líneas paralelas.

**Corte 3 incluido en esta entrega; recuperación demostrada:** respuesta declarada
por RRHH registrada desde el navegador (`201`). Después del parche temporal y
del segundo reinicio de aplicación y PostgreSQL principal, dirección confirmó
`200/200/200` al recuperar selección, comunicación y respuesta con sus claves
originales. Se conservan el mismo justificante, recibo y fecha de registro.
El defecto de comparación de fechas está corregido en ambas bases y el
diagnóstico temporal retirado. El corte queda cerrado técnicamente.
La métrica de aquel corte era **5 de 8 pasos completos más parte del sexto**.

**Corte 4 publicado: aceptación manual sintética registrada.** Dirección comprobó
`200/200/200/201`: tres antecedentes originales recuperados y aceptación con
API, V3, CT58 y Bolsa4 reales; cero errores JS, cookies, almacenamiento web y
desbordamiento. Tras reiniciar app/PostgreSQL principal: `200/200/200/200`,
mismo recibo/fecha, sin duplicados; cierre técnico de aceptación manual sintética.
AD3-15/Bolsa4 y AD3-17/CT58 están instaladas en ambas bases y ambas apps usan
la compilación corregida. AD3-16/CT57 ya estaban en ambas.
No reaplicar migraciones. No hay política legal aprobada ni correo corporativo.

**Corte 5 incluido en esta entrega: renuncia manual sintética cerrada funcionalmente.**
Dirección comprobó navegador real `200/201/201/201` y, tras reiniciar aplicación
y PostgreSQL principal, `200/200/200/200`: mismos recibo, resolución, auditoría,
fecha e intención pendiente con su carga real, sin duplicados. Cero errores JS,
cookies, almacenamiento web y desbordamiento. La base principal conserva
**51 solicitudes**, con bandeja y detalle consultables;
AD3-18/Bolsa5/CT59 instaladas en ambas bases. No reaplicar. En aquel corte aún no
se había ejecutado el siguiente candidato. El criterio manual provisional solo sirve al ejercicio
sintético: no es aval legal ni aprobación del operador. Aquel corte mantenía **5/8 más parte del 6**.

**Objetivo 7: continuación tras renuncia `201` real desde la quinta operación.**
Nuevo llamamiento abierto y recibo CT confirmado, sin errores JS, cookies,
almacenamiento web ni desbordamiento. Tras reiniciar app/PostgreSQL principal:
**`200/200/200/200/200`, mismos 14 campos salvo `estado_local: replay_confirmado`**.
Objetivo 7 cerrado funcionalmente solo tras renuncia sintética; quinta clave abajo.
No implica aviso enviado, entregado ni aceptación del nuevo llamamiento.

**Objetivo 8 cerrado funcionalmente en desarrollo:** propuesta desde aceptación,
Chrome `201` y, tras reiniciar app/PostgreSQL principal, cuatro antecedentes `200`
y propuesta `200`, mismos identificadores/fecha/v7. Cero errores JS, cookies,
almacenamiento web y desbordamiento. Caso y cinco claves en el apartado objetivo 8.
Métrica vigente: **5/8 más partes del sexto y séptimo**. Esta revisión incorpora
el cierre funcional; el hash publicado se comprueba en Git. Objetivo 9: primer informe
borrador `200`. Tras corregir con AD3-21 el `404` de consultas por fechas equivalentes,
cinco POST de bandeja/detalle/PDF `200`, también tras reiniciar app/PostgreSQL principal,
sin `404`, PDF idéntico e historia CT/Bolsa conservada. El `502` separado de curl con límite 50
no queda corregido por esta evidencia.
Segundo PDF demostrado: resolución `200`, informe original idéntico en la misma sesión;
seis POST `200` antes y después de reiniciar aplicación/PostgreSQL principal,
ambos PDF, historial y recibos idénticos; cero errores JS, cookies y almacenamiento web.
Tercer borrador demostrado: diligencia `200`, siete POST `200`, informe y resolución
idénticos; también tras reiniciar aplicación/PostgreSQL principal: siete POST `200`,
PDF e historial conservados, cero errores JS, cookies y almacenamiento web.
Comunicación al centro: diez POST `200` antes y después del reinicio principal,
seis PDF e historial idénticos. Objetivo 9 cerrado funcionalmente en desarrollo: **6/6 borradores**.
Siguiente objetivo 10, fuente/circuito de firma admitido; sin firmas, posesión real,
nombramiento eficaz, envío, entrega, plazo legal ni orden de incorporación.
AD3-20/CT61 y AD3-21 instaladas en ambas bases; no reaplicar. La descarga no añadió SQL propio.
Navegador acreditado solo en principal.
Aviso local al sucesor CT60: `201` y recuperación `200` tras reinicio principal; CT62 instalada en ambas
bases, no reaplicar ni ejecutar DOWN. [Recuperación y evidencia](#aviso-local-al-sucesor-ct62).
Declaración del sucesor CT63: seis antecedentes `200` y registro `201`; tras reinicio principal, siete `200`,
mismos justificante/recibo/auditoría/fecha e historia, sin duplicados.
[Séptima operación, correo y recibo](#declaración-de-respuesta-del-sucesor-ct63). CT63 instalada en ambas bases;
no reaplicar ni ejecutar DOWN con historial. Esa declaración no es una resolución.
CT64 confirma separadamente la aceptación manual sintética del sucesor: recuperación con la misma clave,
ocho POST `200` antes y después del reinicio principal; mismos recibo/fecha/evaluación/auditoría e historia.
[Octava operación y recuperación del primer `503`](#resolución-manual-del-sucesor-ct64).
CT64 instalada en ambas bases, no reaplicar ni DOWN con resolución sucesora; sin SQL remoto.
CT65 incorpora la propuesta del sucesor: ocho antecedentes `200` y propuesta `201`;
tras reiniciar app/PostgreSQL principal, nueve `200`, mismos recibo/fecha/v7 e historia.
[Novena operación del panel, clave y evidencia](#propuesta-desde-la-aceptación-del-sucesor-ct65).
No es un noveno paso RRHH: siguen **5/8 más partes del sexto y séptimo**.

Las dos bases y el material de desarrollo se han trasladado sin regenerar
identidades, claves ni expedientes. Las copias físicas se verificaron antes
de arrancar. En la comprobación del traslado, después de reiniciar aplicación
y PostgreSQL locales, la base secundaria mostró los mismos 21 expedientes y
abrió el mismo detalle, versión 1. La base principal conservó sus 50 altas
confirmadas y 24 asientos de tramitación.
Son conjuntos diferentes: no mezclarlos ni usar uno para rellenar el otro.

| Modalidad del lanzador | Portal local | Base y alcance actual |
| --- | --- | --- |
| `recorrido` — principal | `https://localhost:8443/portal-empleado/` | PostgreSQL 55433: cinco pasos y partes del sexto y séptimo; propuesta de desarrollo recuperada tras reinicio. |
| `consultas` — secundaria | `https://localhost:8444/portal-empleado/` | PostgreSQL 55432: bandeja y detalle reales en el entorno aislado. No ejecutar pruebas simultáneas contra esta base mientras la usa la aplicación. |

Ambas bases tienen instalada la dependencia de consultas. Dirección confirmó
en 55433 la publicación del catálogo sintético original y sus dos vínculos:
tres resultados positivos en una transacción, secuencia de catálogo `8`.
No copiar filas entre bases ni repetir esa instalación para arrancar.
Las migraciones de registro de respuesta `000056` de Contratación temporal y
`000014` de autorización también están instaladas en **ambas bases**: no
reaplicarlas. Dirección aplicó literalmente el bloque `DO $fechas$` de la
migración `000014` en ambas bases, con sus tres comprobaciones incorporadas
correctas. El diagnóstico temporal de la función de respuesta también se retiró
de ambas bases. La huella SHA256 histórica del núcleo al cierre de AD3-18,
comprobada entonces igual en ambas bases por dirección, fue
`e6c3d28c27b7cb864916ffe967a8b2fa47611cb3528ad8148302d8bbedd11bf6`.
La anterior `02453e…` corresponde al corte 4, tras AD3-16/17.
La anterior `42f67b…` corresponde al corte histórico AD3-14, no al arranque vigente.
No repetir el bloque temporal para arrancar: también está aplicado.
Las huellas anteriores del historial no son referencias operativas actuales.
Cada modalidad requiere **once conexiones PostgreSQL separadas**, todas a su
propia base. Sus variables y funciones están en el
[manual de sistemas, apartado 2](docs/manual_sistemas/README.md#2-configuración-local-once-conexiones-por-instancia).
Dirección renovó el TLS caducado de PostgreSQL por 30 días, con respaldo y material
privados; se conserva `verify-full`. No repetir la renovación como paso de arranque.

El operador conserva fuera de Git el lanzador `arrancar-local.sh`, las bases
y los certificados. La bitácora local identifica su ruta exacta. Defina
`VEC_ESTADO_LOCAL` con ese directorio privado existente, sin generar otro:

```bash
: "${VEC_ESTADO_LOCAL:?Indique el directorio privado conservado en la bitácora local}"
test -f "$VEC_ESTADO_LOCAL/arrancar-local.sh"
bash "$VEC_ESTADO_LOCAL/arrancar-local.sh" recorrido
# Solo si se necesita la instancia secundaria, en otra terminal:
bash "$VEC_ESTADO_LOCAL/arrancar-local.sh" consultas
```

No arranque otra copia si el puerto ya está ocupado. Ctrl-C detiene esa
aplicación, no borra la base. El lanzador reutiliza y comprueba el material
existente; las instrucciones de importación del certificado se muestran al
arrancar. Es desarrollo sintético, no producción ni identidad corporativa.

### Abrir una solicitud existente y registrar su análisis

Para el acceso local de RRHH no hace falta el certificado personal del operador.
Con la aplicación principal arrancada, abra una terminal en el worktree activo:

```bash
bash scripts/abrir_vec_desarrollo.sh "$VEC_MATERIAL_RRHH"
```

`VEC_MATERIAL_RRHH` es el directorio privado existente que contiene `ca/` y
`mtls/`, no uno nuevo. El lanzador abre Firefox con un perfil exclusivo y el
certificado sintético ya conservado. Requiere Firefox, `certutil`, `pk12util`
y `rg`; comprueba primero que el portal responde. En Firefox Snap el perfil
queda en `$HOME/snap/firefox/common/navegador-vec-rrhh`; en instalaciones
nativas, junto al material. No cambia el perfil habitual ni la confianza del
sistema. No envíe certificados personales, paquetes de claves ni contraseñas.
Si la ventana ya está abierta, úsela: el lanzador impide otra copia simultánea.
Un enlace abierto en otro navegador sin preparar no equivale a este acceso.

La API conserva la autenticación por certificado también en las consultas del
catálogo. El servidor admite exclusivamente el anuncio `TE: trailers` de
HTTP/2 en Contratación; no admite trailers de petición ni credenciales en
cabeceras. No se desactiva TLS ni HTTP/2 para entrar.

Comprobación de acceso del 6 de septiembre: Firefox con ese mismo perfil,
validación TLS activa y HTTP/2 mostró los 51 expedientes; la búsqueda por el
número `2026/CT-f5a5578760afec875187195d4108606a` devolvió una fila y abrió su
detalle. Se descargó el informe borrador (29.267 bytes), y se abrió el formulario
real de nueva petición. Sin certificado inyectado por automatización, sin
adaptador de muestra y sin registrar otra alta. Esto corrige el acceso local;
no suma otro paso completo a los ocho de RRHH.

Recorrido confirmado por dirección el 5 de septiembre en **8443**, incluido
en el código publicado: bandeja de **50 expedientes** → solicitud existente versión `1` →
formulario real de análisis → respuesta `201` → recibo versión `2`.
Se conservaron las 50 altas: esta actuación no registra una solicitud nueva.

1. Abra el portal principal con el certificado de RRHH y entre en
   **Contratación temporal**. Use la bandeja, no **Nueva petición**.
2. Localice una fila en fase **Solicitud**, en curso, versión `1`, y ábrala
   con los controles de esa fila. La aplicación lleva su referencia al
   detalle: no hace falta copiar identificadores ni llamar a la API a mano.
3. Compruebe el expediente y su versión. Debe aparecer el formulario real
   **Análisis por Recursos Humanos** para esa solicitud, sin reconstruir el alta.
4. Complete sus campos con los catálogos y datos sintéticos válidos del
   formulario. Pulse **Registrar análisis** una sola vez.
5. Compruebe que el recibo conserva el expediente y devuelve versión `2`.
   En la red, lista y detalle usan `/cuadro/consultas` y
   `/expedientes/consultas`; el análisis usa `/analisis/registros`, con
   respuesta `201`. Todas pertenecen a `/api/vec/contratacion-temporal`.

En la comprobación se usó la solicitud identificada por el tramo
`b50fa719…` de su referencia; el recibo fue
`rec_ct_an_c4c3b1fc86cc0d5531e655b26bd68096`, versión `2`.
La referencia abreviada solo identifica la evidencia: no se pega en formularios.
Esa solicitud ya está analizada; no repetirla como si continuara en versión `1`.
Abrir otra solicitud para analizarla constituye otra actuación, no una consulta.

Si el formulario no aparece, hay una actualización pendiente o se muestra un
error, no fuerce la versión ni cree otra alta para eludirlo. Una pantalla
cargada no sustituye al recibo real. El análisis no añade por sí solo otro paso
completo; el aviso sigue siendo local, no correo corporativo.

**Reinicio confirmado por dirección:** la base principal sigue mostrando
50 expedientes y el detalle conserva la versión `2`. Una lectura independiente
confirmó un único recibo, asiento y terminal del análisis en esa versión.
La comprobación conserva el efecto único, sin otra alta.
La entrega de bandeja y análisis ya está integrada y publicada en `b2effbaf`;
no queda código de ese corte pendiente de integrar. Publicar el código no
reactiva las instancias remotas ni autoriza producción.

### Corte 3: registrar la declaración de una respuesta recibida por RRHH

El recorrido real en la principal devolvió `201`: RRHH declaró una
**aceptación recibida**, con referencia del correo, huella SHA256, identidad
de quien registra y justificante persistentes. La lectura de la base confirmó
**una respuesta, un asiento de historial y un evento de salida**.
Es una declaración registrada, **no una aceptación terminal**: no cambia el
estado de Bolsa, no avanza el expediente y la comunicación sigue en versión `2`.
No verifica origen, firma ni custodia del correo; tampoco acredita envío o
entrega del aviso local. La cuarta operación permite una resolución manual
sintética con autorización y comprobaciones propias; no convierte esta
declaración automáticamente en aceptación.

Para recuperar este caso use los datos originales, sin crear otra solicitud
ni cambiar claves. El acceso requiere el certificado y contexto autorizado
de RRHH; conocer las referencias no sustituye el permiso.

1. Abra el llamamiento del expediente
   `expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001`.
   Recupere la selección con versión `6` y clave
   `90d52c16-a63d-4ef1-bcf7-62c7c455f9aa`, sin preparar una selección nueva.
2. Recupere su comunicación local con la clave
   `d1b5428f-2188-4b8f-98b7-42f82ad88c2a`. El recibo aporta las referencias
   de llamamiento y comunicación al formulario de respuesta; compruebe
   versión de comunicación `2`. No invente ni cambie esas referencias.
3. En el formulario de respuesta, mantenga los valores de la tabla y cargue
   el mismo [correo sintético de ejemplo](docs/manual_rrhh/ejemplos/respuesta_sintetica.eml),
   sin editarlo ni cambiar sus saltos de línea. La huella se calcula en el
   navegador: el archivo `.eml` (máximo 2 MiB) **no se sube ni queda custodiado
   por la aplicación**. Se envían su referencia y huella declaradas, no su
   contenido ni nombre de archivo. La fecha del formulario se expresa en UTC.

| Dato de la respuesta original | Valor que debe conservarse |
| --- | --- |
| Clave de operación | `c3e0f431-b274-48fd-a2e8-4b1e6d220056` |
| Respuesta declarada | `aceptacion` |
| Referencia del correo | `correo:sintetico:respuesta-20260905-0056` |
| Fecha de recepción declarada, UTC | `2026-09-05T10:00:00Z` |
| SHA256 del archivo | `7984edfd3ba13c87b0c04160dbfa8b338b356ead70d80df04066e67e4ed419b9` |

Desde el directorio de esta guía puede comprobar los bytes del ejemplo:

```bash
sha256sum docs/manual_rrhh/ejemplos/respuesta_sintetica.eml
```

El registro inicial se realizó en
`/api/vec/contratacion-temporal/llamamientos/respuestas/registro`, con estado
`registrada_por_rrhh`. Conservó el justificante
`84727d1d-31ef-4fde-92c8-d3a8e2953931`, el recibo
`9e14599d-2edc-42aa-afde-170420c838aa` y la fecha de registro
`2026-09-05T18:09:06.065542Z`.

**Defecto corregido y recuperación confirmada:** los rechazos intermitentes
`403` procedían de comparar fechas como texto (seis decimales frente a una
representación que omite ceros finales), aunque expresasen el mismo instante;
no del reinicio. Ambas bases comparan ahora instantes mediante conversión a
`timestamptz`, sin cambiar bytes firmados ni permisos. El diagnóstico temporal
está retirado.

Después de aplicar esa corrección y del segundo reinicio de aplicación y
PostgreSQL principal, dirección confirmó desde el navegador las tres
recuperaciones `200/200/200`: selección, comunicación y respuesta. Con las
mismas claves y material, la respuesta devuelve `replay_registrada_por_rrhh`
y conserva el justificante `84727d1d-31ef-4fde-92c8-d3a8e2953931`, el recibo
`9e14599d-2edc-42aa-afde-170420c838aa` y la fecha original
`2026-09-05T18:09:06.065542Z`. También se confirmó ausencia de errores de
JavaScript, cookies, almacenamiento web y desbordamiento horizontal en móvil.
También se comprobó desde el navegador el rechazo `409` por conflicto,
sin generar un duplicado. El corte está incluido en esta entrega, cerrado
técnicamente; no equivale a cerrar todo el paso 6.
Ante un rechazo posterior, no encadene reintentos ni genere otra clave para
sortearlo: conservar la operación original sigue siendo obligatorio.

### Corte 4: aceptación manual registrada, solo ejercicio sintético

Tras recuperar la declaración `aceptacion` anterior, el mismo formulario muestra
**4. Solicitar resolución de respuesta** (`data-ct-llamamiento-form="resolucion"`).
Organización, expediente, llamamiento y comunicación proceden del recibo original;
la versión esperada es `2` y `prueba_respuesta_ref` es su `justificante_ref`, no
el recibo ni una referencia inventada. Use una clave propia de resolución,
distinta de las tres anteriores. Para recuperar el caso ya registrado, use
exactamente `018f47a6-5d2b-4c10-8a11-1234567890ef`, con los mismos antecedentes,
ambas revisiones y criterio. No se carga otro `.eml` ni se introduce identidad.

Las dos casillas empiezan desmarcadas. Solo tras comprobar el caso sintético marque:

- **He comprobado la respuesta y su justificante**.
- **Para este ejercicio sintético, he comprobado que la respuesta llegó dentro del plazo del ejercicio**.

El criterio de solo lectura es `politica:ct:revision-manual-sintetica:20260906`.
**Validación manual de desarrollo: no acredita entrega de correo ni plazo legal real**.
Con ambas casillas marcadas, pulse **Revisar y solicitar resolución**
y confirme expresamente. El servidor autoriza y conserva la declaración, actor y
política; solo devuelve éxito tras confirmar CT y Bolsa, no por marcar casillas.

Dirección observó `201` y **Aceptación registrada · ejercicio sintético**:
recibo CT `recibo:d6bdcc7b-e22e-4fe9-8aac-a1eb554a4103`, resolución
`7a3e4a2e-d142-4562-ae5c-59c95b011e0c`, fecha `2026-09-05T22:27:02.861379Z`.
Tras reiniciar app/PostgreSQL principal, dirección confirmó `200/200/200/200`,
con ese mismo recibo y fecha: una resolución CT58, una aceptación Bolsa, tres
historias y tres eventos, sin duplicados. Cierre técnico solo del ejercicio sintético;
no se atribuye este recorrido a la secundaria por tener la misma instalación.

Sin ambas revisiones la UI no envía; la petición antigua de ocho campos conserva
el `409 validacion_respuesta_pendiente`, sin efectos. Solo ante ese rechazo conocido
puede corregir las casillas conservando la clave. Ante resultado ambiguo mantenga
congelados clave y material; no hay reintentos automáticos ni claves sustitutas.
La prueba aislada anterior de Bolsa4 (`8197db3`) usó un doble privado transaccional.
El roundtrip de aceptación UP/DOWN AD3-15/Bolsa4/AD3-17/CT58 comprobó reversión exacta en
ROLLBACK, sin modificar autorización ni usar dobles; el navegador usó criptografía real.
Faltan vencimiento y correo corporativo; la continuación tras renuncia se describe abajo;
la propuesta de desarrollo del objetivo 8 se describe abajo y no hay política legal
aprobada. Véase el [plan vigente](ESTADO_PROYECTO.md).

### Corte 5: recuperar la renuncia manual sintética

Es otro expediente, creado desde la UI y fiscalizado favorablemente en `v6`:
`expediente:ct:fe4934a1c7a9f9ad91aaccc6026ff7d39a494031d14d8a98dcd0d6a140619ba7`.
No reutilice las claves ni el correo de aceptación. Con el perfil RRHH, use
el mismo formulario de llamamiento y recupere, en orden, las cuatro operaciones:

| Operación | Clave original exacta |
| --- | --- |
| Selección, expediente `v6` | `a77d3f10-a635-46fd-b9eb-a00000000001` |
| Comunicación local | `a77d3f10-a635-46fd-b9eb-a00000000002` |
| Declaración RRHH | `a77d3f10-a635-46fd-b9eb-a00000000003` |
| Resolución manual de renuncia | `a77d3f10-a635-46fd-b9eb-a00000000004` |
| Continuación tras renuncia (objetivo 7, apartado siguiente) | `a77d3f10-a635-46fd-b9eb-a00000000005` |
| Aviso local al sucesor (CT62) | `a77d3f10-a635-46fd-b9eb-a00000000006` |
| Declaración RRHH del sucesor (CT63) | `a77d3f10-a635-46fd-b9eb-a00000000007` |
| Resolución manual del sucesor (CT64) | `a77d3f10-a635-46fd-b9eb-a00000000008` |
| Propuesta desde aceptación del sucesor (CT65) | `a77d3f10-a635-46fd-b9eb-a00000000009` |

El recibo de selección identifica
`llamamiento:nccfkjnioljdeikkpipkcgcpilbogjnociankdfbapmnaekanagbiioaahphbmgj`.
Deje que los recibos rellenen los antecedentes siguientes; la comunicación
antecedente es `v2`. Para la declaración elija `renuncia`, referencia
`correo:sintetico:renuncia-20260906`, recepción UTC `2026-09-05T23:00:42.971Z`.
Cargue el `.eml` original conservado fuera de Git por el operador (ubicación en
su bitácora local), hasta
2 MiB, cuya SHA256 es `b3d14ec8b017425a1a867023f7170ab72600f6e5ddf34baed2fa8f5776c4e397`.
No use `respuesta_sintetica.eml`, que corresponde a la aceptación anterior;
si falta el original de renuncia, solicítelo, no reconstruya bytes ni edite la huella.
El contenido no se sube ni se guarda en VEC.

La declaración devuelve `justificante:19fc8396-846c-4a3b-8cab-ebd57054f298`.
La cuarta operación deriva de ese recibo `respuesta=renuncia` y el justificante;
no permite sustituirlos. Mantenga las dos revisiones expresas y el criterio fijo
del corte 4, confirme con la clave de resolución original y sin otro `.eml`.
Resultado: **Renuncia registrada · ejercicio sintético**, versión resultante `3`,
con **Siguiente candidato pendiente**, no seleccionado ni avisado.

| Evidencia conservada tras reiniciar app y PostgreSQL principal | Valor |
| --- | --- |
| Recibo CT | `recibo:408fda57-638d-4a3b-a441-4ef56396e23a` |
| Resolución | `resolucion:0c5fdea4-be11-4bdd-bd0b-5bc035dd9ae0` |
| Auditoría | `aud_v3_01af1fd2958fa327d4fc5425be4f6e98` |
| Fecha UTC | `2026-09-05T23:00:45.289468Z` |
| Intención de siguiente | `intencion:f4bd0049-8b96-410b-8144-4384bdb47ed0`, `pendiente` |

Dirección confirmó `200/200/200/200` con esos mismos datos y la carga de intención
durable en la misma fila CT. Se conservan dos resoluciones CT (aceptación y renuncia),
seis registros Bolsa (dos órdenes, dos propuestas y ambos terminales), seis historias
y seis eventos, sin duplicados. La aceptación previa `d6bdcc7b…`/`22:27:02.861379Z`
y la declaración `9e14599d…`/`18:09:06.065542Z` permanecen intactas.
AD3-18/Bolsa5/CT59 superaron UP/DOWN con ACL, funciones y comprobaciones correctas
según dirección y están instaladas en ambas bases. Eso no autoriza DOWN sobre
estos datos ni atribuye a la secundaria el recorrido realizado en la principal.
Se cierra el objetivo 5 solo para desarrollo sintético; no el siguiente candidato,
el vencimiento, el correo corporativo ni el paso 6 completo.

### Objetivo 7: recuperar la continuación tras renuncia

Recupere primero las cuatro operaciones del caso anterior con sus claves originales.
La quinta operación (`data-ct-llamamiento-form="siguiente"`)
deriva expediente, resolución e intención de esos recibos; solo admite una clave
propia. Para este caso conserve **`a77d3f10-a635-46fd-b9eb-a00000000005`**.
Revise y confirme expresamente que abrirá un único nuevo llamamiento después de
la renuncia; sin otro `.eml`, sin reintento automático ni cambio de clave ante error.
El resultado ambiguo congela clave y material; `409` no autoriza evadir la operación.

Dirección confirmó `201` real, **Siguiente llamamiento abierto · ejercicio sintético**:

| Recibo de continuación | Valor conservado |
| --- | --- |
| Recibo CT | `recibo:b5bb611f-0126-4806-90c5-85b9f9b63778` |
| Auditoría CT | `aud_v3_37035fd6958b5b07c1d4847fb1b6db6b` |
| Confirmada en UTC | `2026-09-05T23:57:11.037866Z` |
| Nuevo llamamiento, versión `1` | `llamamiento:eipmgkkfjncbalgebihinpeeajhllfmkoikjhiodlbcdlhaohmgpojefdfilcmoe` |
| Recibo Bolsa | `recibo:ipbencdeoegpleefmcjipdpanjmnbfnheoehlmelcjacjaeglgkliadcmcpehikj` |
| Confirmación Bolsa UTC | `2026-09-05T23:57:10.99675Z` |

La quinta operación informa intención `despachada`; el recibo original de renuncia
mantiene sus bytes y su `pendiente` histórico. Tras obtener el nuevo recibo se
desactiva otro envío. No sustituya el antecedente de la primera comunicación.
En el corte `9ccef45` el aviso del sucesor aún no estaba conectado;
el corte CT62 descrito debajo lo registra por separado, sin presumir envío.
Bolsa conserva siete registros (dos órdenes, tres propuestas, aceptación y renuncia),
siete historias y siete eventos. Cero errores JS, cookies, almacenamiento web y
desbordamiento. **Tras reiniciar app/PostgreSQL principal, cinco POST `200/200/200/200/200`**:
los 14 campos de la continuación se conservan salvo `estado_local: replay_confirmado`,
incluidos recibos CT/Bolsa, auditoría, fecha y referencias de ambos llamamientos.
Objetivo 7 cerrado funcionalmente solo tras renuncia sintética, sin duplicados.
Ese corte mantenía **5/8 más parte del sexto**; no acredita entrega, aceptación ni plazo legal.

### Aviso local al sucesor CT62

En el caso de renuncia `fe4934a1…`, recupere las cinco operaciones anteriores con
sus claves originales. El mismo panel ofrece **6. Registrar aviso local al sucesor**
(`data-ct-llamamiento-form="comunicacion_siguiente"`). Solo introduzca la clave original
**`a77d3f10-a635-46fd-b9eb-a00000000006`** y confirme expresamente el registro local.
Organización, expediente, llamamiento `eipmg…`, versión `1` y antecedente
`recibo:b5bb611f-0126-4806-90c5-85b9f9b63778` se derivan de la continuación, no se editan.
No sustituye la primera comunicación; habilita la declaración manual del paso 7, no una resolución.
Ante ambigüedad conserve clave/material; reintento solo explícito. `409` no permite otra clave.

Dirección confirmó cinco antecedentes `200` y aviso `201`; sin tipo o con recibo ajeno,
`403` sin efectos. **Reinicio de aplicación/PostgreSQL principal confirmado:** seis operaciones
`200/200/200/200/200/200`, misma comunicación, recibo, auditoría, fecha, versión `2` e intención local.

| Aviso local, versión resultante `2` | Valor conservado |
| --- | --- |
| Comunicación | `comunicacion:7bdf8ba7-6388-4ffe-9d10-09b7c7b66228` |
| Recibo | `recibo:21bf6275-70e6-45c4-ba7f-7732dc8799e0` |
| Auditoría | `aud_v3_67952097afa2953b25700d92c1462bc0` |
| Registrada en UTC | `2026-09-06T03:59:10.809598Z` |
| Intención local pendiente | `outbox:a07c1143-97ae-4192-bdbe-7637e7c6ba16` |

JSON del aviso: **778 bytes**, SHA256
`1c80c14d0937d135453bfe85094f58c268de5188325a8c27fb4b716fbf063277`.
Tras reinicio, mismo JSON/huella y tres comunicaciones/historias/outbox, sin duplicados;
dos comunicaciones v1 byteidénticas, dos resoluciones CT completas y siete registros Bolsa intactos.
Inspección desktop y móvil 390 px; no se repitieron los PDF de cierres anteriores;
cero errores JS, cookies, almacenamiento web y desbordamiento. Solo principal acreditada.
CT62: doble GO, UP/DOWN transaccional exacto en secundaria y UP instalada en ambas bases;
no reaplicar ni ejecutar DOWN con este historial. El aviso conserva `recibo_continuacion_ref`;
no acredita envío, entrega ni plazo. La declaración del sucesor se describe debajo;
siguen pendientes correo corporativo, plazo y circuito de firma; la resolución manual CT64 se describe debajo.
Métrica: **5/8 más partes del sexto y séptimo**.

### Declaración de respuesta del sucesor CT63

En el mismo caso `fe4934a1…`, recupere las seis operaciones anteriores con sus claves originales.
Tras validar el aviso local en versión `2`, aparece **7. Registrar respuesta recibida del sucesor**
(`data-ct-llamamiento-form="respuesta_siguiente"`). Organización, expediente, llamamiento `eipmg…`,
comunicación `7bdf8ba7…` y versión `2` se derivan de ese aviso; no se editan.
Use **`a77d3f10-a635-46fd-b9eb-a00000000007`**, respuesta **aceptación declarada**,
referencia `correo:sintetico:sucesor-20260906` y recepción UTC **`2026-09-06T04:00:00Z`**
(control de fecha: `2026-09-06T04:00`). Seleccione **`respuesta-sucesor-sintetica.eml`**,
original sintético de **386 bytes**, conservado fuera de Git por el operador; ruta en su bitácora local.
SHA256: `9f5fde0e55589d26349467081df601ede5b1b15e3da2d69bc671de13eff9fe17`.
No use los correos anteriores ni reconstruya bytes si falta el archivo: solicite el original.
El `.eml` admite hasta 2 MiB; solo su huella se calcula en RAM, sin subir ni guardar contenido.
Confirme expresamente la declaración RRHH; no acredita origen, firma, custodia, envío, entrega ni plazo.

| Declaración del sucesor, comunicación en versión `2` | Valor conservado tras reinicio |
| --- | --- |
| Justificante | `justificante:6353cd84-616d-467f-95c2-ec8a27e15c64` |
| Recibo | `recibo:f2f54bfc-d537-4d1b-a4e9-81c7a653e5d8` |
| Auditoría | `aud_v3_11dc51ba44d91c00c8ae765544a83b24` |
| Registrada en UTC | `2026-09-06T04:26:40.108845Z` |

Dirección confirmó seis antecedentes `200` y declaración `201`; dos cruces de comunicación/llamamiento
devuelven `409` sin efectos. Tres declaraciones/historias/outbox; las dos declaraciones anteriores,
tres comunicaciones, dos resoluciones CT y siete registros Bolsa conservan sus huellas.
El ajuste de segundos de `datetime-local` fue del arnés Playwright, sin cambiar producto.
**Tras reiniciar aplicación/PostgreSQL principal: siete operaciones `200`**, mismos justificante,
recibo, auditoría y fecha; solo cambia `estado` a `replay_registrada_por_rrhh`. Historia exacta,
tres respuestas/historias/outbox, sin duplicados; antecedentes anteriores intactos.
Cero errores JS, cookies, almacenamiento web y desbordamiento. Dirección inspeccionó el recibo
desktop y móvil 390 px; evidencia solo principal. Ningún PDF repetido.
Para recuperar, repita esos mismos datos, archivo y clave; nunca otra clave para eludir un `409`. La ambigüedad
congela clave/material y exige recuperación explícita, sin reintento automático.
CT63 instalada en ambas bases tras revisión y UP/DOWN transaccional en secundaria sin persistir;
no reaplicar ni ejecutar DOWN con respuestas del sucesor. Remoto apagado, sin aplicar SQL.
No cambia la resolución original ni resuelve automáticamente al sucesor: requiere la octava operación separada.
Métrica **5/8 más tramos del sexto y séptimo**; seis PDF cerrados, no repetidos.

### Resolución manual del sucesor CT64

En el mismo expediente `fe4934a1…`, recupere los siete antecedentes con las claves de la tabla.
Tras validar el justificante CT63 aparece la octava operación
(`data-ct-llamamiento-form="resolucion_siguiente"`). Deriva organización, expediente, llamamiento,
comunicación en versión `2`, respuesta y prueba del recibo, sin editar esos antecedentes.
Use la clave original **`a77d3f10-a635-46fd-b9eb-a00000000008`**, no otra para eludir un error.
Revise el ejercicio y marque expresamente ambas casillas, inicialmente vacías:

- **He comprobado la respuesta y su justificante**.
- **Para este ejercicio sintético, he comprobado que la respuesta llegó dentro del plazo del ejercicio**.

El criterio fijo es `politica:ct:revision-manual-sintetica:20260906`; no es política legal aprobada.
Revise y confirme expresamente: once campos, sin otro `.eml`. Ante ambigüedad conserve clave y material;
solo el `409 validacion_respuesta_pendiente` conocido, sin ambigüedad previa, permite corregir revisiones conservando la clave.
Sin reintentos automáticos. El recibo de éxito requiere confirmación de CT y Bolsa.

El primer intento devolvió **`503`**: CT ya había persistido a `04:52:46.758226Z`, pero Bolsa seguía
con siete operaciones. La referencia de evaluación UUID colisionaba con la heurística DNI.
El parche mínimo, con doble GO y focal PASS de 4 ms, admite formato `evaluacion:UUIDv4` únicamente
en ese campo; no cambia autoridad ni el validador general. **No se regeneró la evaluación ni la clave**.
La recuperación con la misma clave y los siete antecedentes dio **ocho POST `200`**;
no se atribuye un `201` ni una aceptación completa al intento fallido.

| Aceptación sintética del sucesor | Valor conservado |
| --- | --- |
| Resolución | `resolucion:c1d55777-ce6c-49e6-a7fc-3eaac0f728bb` |
| Recibo CT | `recibo:de377a72-ace7-4365-b865-f9384a4c3196` |
| Evaluación | `evaluacion:409ebaed-71ee-418f-be7c-4025729222c7` |
| Auditoría | `aud_v3_42ae013bb945deb79a258006391db31c` |
| Resuelta en UTC | `2026-09-06T04:52:46.758226Z` |
| Versión resultante | `3`; expediente `6` al cierre CT64, antes del avance CT65 a `7` |

SHA256 del recibo en PostgreSQL:
`f8a0b0c80db4797eeee0216ea020d6925a001422866d90d0c5bd19c4be01952d`.
Única operación nueva Bolsa:
`operacion-aceptacion-rrhh:imbiennjhbamlaibfdlclpmmdiplepafdgpmnmmdmhbmjgieejadaeceokcpomfn`.
**Tras reiniciar PostgreSQL y aplicación principal: otros ocho POST `200`**, mismos recibo, fecha,
evaluación, auditoría e historia. Tres resoluciones CT y ocho operaciones/historias/outbox Bolsa;
las siete Bolsa anteriores, tres respuestas, tres comunicaciones y dos resoluciones CT previas conservan sus huellas.
Cero errores JS, cookies, almacenamiento web y desbordamiento; dirección inspeccionó recibo desktop y móvil 390 px.
Capturas privadas; navegador acreditado solo en principal. No se repitieron los seis PDF cerrados.
CT64 instalada en ambas bases locales: no reaplicar ni ejecutar DOWN con resolución sucesora; sin SQL remoto.
Sin envío, plazo legal, firma, propuesta automática ni tercer llamamiento. La propuesta posterior
se recupera por separado como CT65. Métrica **5/8 más tramos del sexto y séptimo**, sin incremento.

### Propuesta desde la aceptación del sucesor CT65

En el caso `fe4934a1…`, recupere los ocho antecedentes con sus claves originales de la tabla.
La aceptación del sucesor es `resolucion:c1d55777-ce6c-49e6-a7fc-3eaac0f728bb`,
recibo `recibo:de377a72-ace7-4365-b865-f9384a4c3196`. Solo después aparece el formulario común
**Propuesta de nombramiento · desarrollo** (`data-ct-llamamiento-form="propuesta"`):
novena operación del panel, no otro paso RRHH ni formulario paralelo.
Use **`a77d3f10-a635-46fd-b9eb-a00000000009`** y conserve **versión esperada `6`, incluso en replay desde `7`**.
Antecedentes y cuatro publicaciones se derivan; no se editan. Espere su carga, revise el llamamiento
y la resolución mostrados y confirme expresamente, sin otro `.eml`. No se registra automáticamente.
Ante ambigüedad conserve clave/material; ningún `409` autoriza otra clave para eludirlo.

| Propuesta del sucesor | Valor conservado tras reinicio principal |
| --- | --- |
| Propuesta | `propuesta:e3e68788-3fa8-46fb-ba94-c4ded2ef4196` |
| Recibo | `recibo:6e3602be-22d9-42d9-b32a-fe80ee4103b0` |
| Confirmada en UTC | `2026-09-06T08:45:03.718917Z` |
| Auditoría persistida | `aud_v3_6b305700ef115f74637e0f81b21fcdbb` |
| Expediente | `6→7`, `nombramiento/en_curso` |

Navegador: **ocho antecedentes `200` y propuesta `201`**; cruce con llamamiento anterior,
`409 resolucion_no_aceptada`, sin efecto. Tras el único reinicio app/PostgreSQL principal:
**nueve `200`**, mismos recibo/fecha/v7 e historia exacta; los cuatro campos de referencia,
recibo, fecha y versión vistos en navegador coinciden con PostgreSQL.
SHA256 de la **fila completa de propuesta**, no del recibo:
`1eac65a224e7e36d43302565667f2e30a55c8720c1f895d957cbfd0b45c4dfe4`.
Recuentos globales: dos propuestas, 55 actuaciones y 55 outbox (+1 cada uno); tres respuestas,
tres comunicaciones, tres resoluciones y ocho Bolsa sin cambios. Propuesta original y trece
versiones previas conservan sus huellas. La primera interrupción del arnés fue al capturar un DOM
desmontado, antes del POST de propuesta: baseline intacto, corregido solo el arnés.
Cero errores JS, cookies, almacenamiento web y desbordamiento; dirección inspeccionó desktop
y recibo tras reinicio en móvil 390 px. Capturas privadas durables; E2E acreditado solo en principal.
CT65 instalada en ambas bases: no reaplicar ni ejecutar DOWN con propuesta sucesora; sin SQL remoto.
El ajuste CT60 permite desde v7 únicamente recuperar la apertura Bolsa existente con permiso nuevo,
no abrir otra. No repetir SQL65/64 ni los seis PDF cerrados. Sin firma, envío, plazo legal ni incorporación;
métrica **5/8 más partes del sexto y séptimo**, sin incremento.

### Objetivo 8: recuperar la propuesta de nombramiento

Para recuperar la propuesta del caso original de aceptación, use el expediente
`expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001`.
Recupere en el mismo panel los cuatro antecedentes descritos arriba; la operación
`data-ct-llamamiento-form="propuesta"` aparece solo tras aceptación confirmada.

| Operación | Clave original exacta |
| --- | --- |
| Selección | `90d52c16-a63d-4ef1-bcf7-62c7c455f9aa` |
| Comunicación | `d1b5428f-2188-4b8f-98b7-42f82ad88c2a` |
| Declaración RRHH | `c3e0f431-b274-48fd-a2e8-4b1e6d220056` |
| Aceptación manual | `018f47a6-5d2b-4c10-8a11-1234567890ef` |
| Propuesta | `018f47a6-5d2b-4c10-8a11-123456789008` |

Conserve **versión esperada `6`, también al recuperar cuando el agregado ya es `7`**;
no use la versión `3` de resolución. Las referencias y cuatro publicaciones de
desarrollo se derivan y no se editan; si no cargan, no se permite enviar.
Revise y confirme expresamente, sin otro `.eml`. Ante ambigüedad conserve clave
y material, reintento solo manual; un `409` no autoriza preparar otra clave.

Dirección confirmó Chrome `201`, **Propuesta registrada · ejercicio sintético**:
`propuesta:2dd1c999-44c3-4fdc-b68e-e0adde592c81`,
`recibo:3335969d-3bb5-4258-afcc-1af26b7f7207`, fecha UTC
`2026-09-06T01:28:30.697897Z`, expediente `nombramiento/en_curso/v7`.
La consulta real del detalle RRHH por API confirma v7, nombramiento y siete hitos.
Tras reiniciar app/PostgreSQL principal, cuatro antecedentes `200` y propuesta
`200`, mismos identificadores, recibo, fecha y versión `7`, sin duplicados.
Una actuación v7 y un outbox nuevos; hashes de dos resoluciones CT, siete registros
Bolsa y seis versiones anteriores idénticos antes, después del `201` y del reinicio.
Cero errores JS, cookies, almacenamiento web y desbordamiento en móvil.
AD3-20/CT61 instaladas en ambas bases, no reaplicar; no acredita otro E2E en secundaria.
En aquel corte no había descarga documental. Los seis borradores del objetivo 9
se obtienen como sigue; no hay firma, nombramiento eficaz ni correo real.

### Objetivo 9: descargar el primer informe borrador

Con el perfil RRHH en el recorrido principal ya preparado:

1. Abra **Contratación temporal → Cuadro de mando**.
2. Busque el número `2026/CT-f5a5578760afec875187195d4108606a` y aplique el filtro.
3. Pulse **Abrir expediente** en esa fila: detalle del expediente `5fe7e60e…`,
   versión `7`, `nombramiento/en_curso`, sin actualización pendiente.
4. En la cabecera pulse **Descargar informe · borrador de desarrollo**.
   Obtendrá `informe-definitivo-borrador.pdf`. No recupere ni registre la propuesta
   por POST para descargar; no se necesitan claves ni replays de llamamiento.
5. Desde la misma cabecera, **Descargar resolución · borrador de desarrollo** obtiene
   `resolucion-borrador.pdf`. Cada botón realiza solo su consulta autorizada del detalle.
6. **Descargar diligencia · borrador de desarrollo** obtiene `diligencia-borrador.pdf`:
   misma lectura v7, sin certificar hechos, comparecencia, firma ni notificación.
7. **Descargar toma de posesión · borrador de desarrollo** obtiene `toma-posesion-borrador.pdf`;
   no acredita comparecencia, posesión efectiva ni incorporación al puesto.
8. **Descargar notificación · borrador de desarrollo** obtiene `notificacion-borrador.pdf`;
   misma lectura v7, sin acreditar envío, entrega ni apertura de plazo legal.
9. **Descargar comunicación al centro · borrador de desarrollo** obtiene
   `comunicacion-centro-borrador.pdf`; no envía una comunicación ni ordena la incorporación.

Dirección confirmó Chrome `200`, PDF y pantalla inspeccionados: **29267 bytes**,
SHA256 `a3a7f6e95f00d2040a0978e122ba86499b17c924d08b065523fa24aceba37faa`.
La segunda descarga, resolución, dio Chrome `200`, **29770 bytes**,
SHA256 `e535354e1a8a7efbaf19386427dfd0f62e5832394eda808e963693e6f8e47d28`.
La misma sesión conservó la huella del informe original: seis POST `200`, cero errores
JS, cookies y almacenamiento web. Dirección inspeccionó resolución PDF y pantalla estable
de 390 px con ambos botones. **Reinicio principal de aplicación/PostgreSQL confirmado**:
otros seis POST `200`, resolución de 29770 bytes e informe de 29267 bytes, mismas
huellas anteriores; historial y recibos previos idénticos. Cero errores JS, cookies
y almacenamiento web. No se atribuye esta comprobación a secundaria.

Tercera descarga, diligencia: Chrome `200`, **28366 bytes**,
SHA256 `8fbf9ace709998781435185880b9ecbeedb5ef1e8ff46946eccc173d4c6315b4`.
Siete POST `200`, informe y resolución con las mismas huellas en esa sesión;
cero errores JS, cookies y almacenamiento web. Dirección inspeccionó PDF y pantalla
estable de 390 px con tres botones. **Reinicio principal de aplicación/PostgreSQL confirmado**:
otros siete POST `200`, misma diligencia de 28366 bytes y SHA256, informe y resolución
anteriores idénticos. Historial de dos resoluciones CT, siete registros Bolsa y versiones
1..6 exactamente igual al previo; cero errores JS, cookies y almacenamiento web.
Esta evidencia de navegador corresponde solo a principal, no a secundaria.

Cuarta descarga, toma de posesión: **29289 bytes**, SHA256
`5956f6a1095a2ba610123388b9176baeaeeca0b69b9d57b9cb2fb2155d29de72`.
Dirección confirmó ocho POST `200` antes y después de reiniciar aplicación/PostgreSQL
principal, mismo PDF y tres anteriores idénticos; historial 2 CT/7 Bolsa/versiones 1..6
conservado, cero errores JS, cookies y almacenamiento web. Inspeccionó PDF y pantalla
estable de 390 px con cuatro botones. No acredita posesión real ni E2E en secundaria.

Quinta descarga, notificación: **28583 bytes**, SHA256
`35dc1d12dff5bfa6acd1738bd1fddf1b293eb1f1fef0961389b76ef8888cf9f2`.
Dirección confirmó nueve POST `200` antes y después de reiniciar aplicación/PostgreSQL
principal: mismo PDF, cuatro anteriores idénticos e historial 2 CT/7 Bolsa/versiones 1..6
conservado. Cero errores JS, cookies, almacenamiento web y desbordamiento; PDF y pantalla
estable de 390 px con cinco botones inspeccionados. No acredita E2E en secundaria.
La primera apertura, antes de que el servidor escuchara, falló en conexión sin alcanzar
la API; se corrigió la fase de arranque, no un defecto de producto.

Sexta descarga, comunicación al centro: **29643 bytes**, SHA256
`c00c28fbc02d378122a763a088535cc7f6ed6cc1a13ac487f715e716845308b9`.
Dirección confirmó diez POST `200` antes y después de reiniciar aplicación/PostgreSQL
principal: mismo PDF y cinco anteriores idénticos; historial 2 CT/7 Bolsa/versiones 1..6
conservado. Cero errores JS, cookies, almacenamiento web y desbordamiento; PDF y pantalla
estable de 390 px con seis botones inspeccionados. No se atribuye esta evidencia a secundaria.

Antecedente de la primera descarga y su corrección de consultas:

Rectificación de evidencia: el primer fallo de navegador tras reinicio no capturó
estado HTTP. El `502 resultado_no_confiable` observado era curl con **límite 50**,
una incidencia de paginación separada y sin corrección acreditada en este corte.
Otra apertura de navegador, **límite 100**, capturó sonda `200` y vista `404`.
PostgreSQL rechazaba con `42501` fechas equivalentes (`.999340Z` y `.99934Z`)
comparadas como texto en dos funciones de lectura de AD3-3/5; AD3-14 había corregido
mutaciones, no esas lecturas. AD3-21 compara instantes sin alterar bytes firmados,
guardas ni cursor. Dirección confirmó UP/DOWN exacto de definiciones, propietario,
configuración y ACL en ROLLBACK en secundaria e instaló UP en ambas bases.
Tras AD3-21, cinco POST de bandeja/detalle/PDF `200`, PDF idéntico en tamaño y SHA256;
cero errores JS, cookies y almacenamiento web. **Reinicio principal de aplicación y
PostgreSQL confirmado**: dos cuadros iniciales con límite 100, cuadro filtrado 100,
detalle v7 y PDF, los cinco POST `200`, sin `404`; mismos 29267 bytes y SHA256,
historia CT/Bolsa conservada. Cero errores JS, cookies, almacenamiento web y desbordamiento DOM.
No se atribuye este E2E a secundaria ni se cierra la incidencia de paginación con límite 50.
Dirección inspeccionó la captura estable de **390 px**: menú oculto, sin tapar detalle
ni botón. La superposición de la captura anterior correspondía a la transición CSS
de 180 ms; no se modificó la UI. Esta observación no acredita usabilidad móvil global.
La corrección temporal queda confirmada tras reinicio. Las dos consultas iniciales
tienen disparadores distintos; no se atribuye una carrera a esa duplicación.
Ante error de descarga se conserva el detalle; no repita una actuación para sortearlo.
AD3-20/CT61 y AD3-21 instaladas en ambas bases, no reaplicar; el PDF no añade SQL propio.
No repetir UP/DOWN de AD3-21. El desarrollo remoto permanece apagado: no aplicar allí SQL ni arrancarlo.
Los seis son borradores sin firma ni eficacia administrativa, envío, entrega, plazo legal
ni orden de incorporación. Objetivo 9 cerrado funcionalmente en desarrollo: **6/6**;
siguiente 10, dependiente de fuente/circuito de firma admitido.
El aviso CT62, la declaración CT63 y la aceptación manual sintética CT64 del sucesor son recuperables;
siguen pendientes envío corporativo y plazo legal en el paso 6, sin propuesta automática desde el sucesor.
Métrica sin incremento: **5/8 más partes del sexto y séptimo**.

## Recorrido remoto del 4 de septiembre — historial conservado

Las secciones posteriores conservan los ejemplos funcionales y la evidencia
anterior. Sus comandos SSH, rutas remotas y apéndice de recreación no son
instrucciones de arranque actuales. Para operar aquí use el bloque precedente;
no recree bases ni arranque el desarrollo remoto.

Esta guía permite probar los cinco primeros pasos completos del flujo de
Recursos Humanos, incluida la Fiscalización y su devolución a la unidad, con
la aplicación real: navegador con certificado de cliente → API interna →
autorización de servidor → PostgreSQL → recibo. No usa el adaptador DEMO.

## Entorno preservado para el recorrido manual

La evidencia sintética del 4 de septiembre de 2026 sigue disponible en el
servidor. No es necesario recrearla ni borrarla para recorrer otro expediente:

- repositorio: `/srv/fabrica/proyectos/VEC_Diputacion_app`;
- producto: `.worktrees/ct-producto-ligero-20260821`;
- única línea de trabajo del paso 6:
  `.worktrees/ct-app-llamamiento-b4a-20260905`, base `83fc7631`;
- PostgreSQL exclusivo del navegador: contenedor
  `vec-ct-o2-07-browser-20260904-tls`, puerto local remoto `55433`;
- PostgreSQL reservado para pruebas Go: contenedor
  `vec-ct-o2-07-e2e-20260904-tls`, puerto local remoto `55432`;
- material HTTPS y certificados separados de RRHH e Intervención:
  `/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904`.

Regla obligatoria: el servidor usado por el navegador solo apunta a `55433`.
Las pruebas Go solo apuntan a `55432`. Nunca se usan a la vez contra la misma
instancia.

Ambas instancias ya tienen autorización `000011/000012/000013`, Bolsa `000003`
y Contratación temporal `000053/000054/000055`. No recrear, borrar ni reaplicar
migraciones o concesiones. El apéndice de recreación no se usa en este hito.

## 1. Comprobar la instancia aislada

En una terminal del servidor:

```bash
ssh root@cidonia.cloud
cd /srv/fabrica/proyectos/VEC_Diputacion_app/.worktrees/ct-app-llamamiento-b4a-20260905
git status --short --branch
docker start vec-ct-o2-07-browser-20260904-tls >/dev/null
docker inspect vec-ct-o2-07-browser-20260904-tls \
  --format 'estado={{.State.Status}} imagen={{.Config.Image}} {{range .Mounts}}volumen={{.Name}} destino={{.Destination}}{{end}} puertos={{json .HostConfig.PortBindings}}'
docker exec vec-ct-o2-07-browser-20260904-tls \
  psql -X -U postgres -d postgres -Atqc \
  "SELECT concat_ws('/',
    (SELECT count(*) FROM vec_contratacion_temporal.candidatura_alta_tecnica c
      JOIN vec_contratacion_temporal.confirmacion_agregado_alta a USING(expediente_ref)),
    (SELECT count(*) FROM vec_contratacion_temporal.expediente_alta),
    (SELECT count(*) FROM vec_contratacion_temporal.confirmacion_agregado_alta));"
docker exec vec-ct-o2-07-browser-20260904-tls \
  psql -X -U postgres -d postgres -Atqc \
  "SELECT login.rolcanlogin::text||'|'||string_agg(grupo.rolname,',' ORDER BY grupo.rolname)
     FROM pg_roles login
     JOIN pg_auth_members miembro ON miembro.member=login.oid
     JOIN pg_roles grupo ON grupo.oid=miembro.roleid
    WHERE login.rolname='vec_autorizacion_o207_registro'
    GROUP BY login.rolcanlogin;"
docker exec vec-ct-o2-07-browser-20260904-tls \
  psql -X -U postgres -d postgres -Atqc \
  "SELECT login.rolcanlogin::text||'|'||string_agg(grupo.rolname,',' ORDER BY grupo.rolname)
     FROM pg_roles login
     JOIN pg_auth_members miembro ON miembro.member=login.oid
     JOIN pg_roles grupo ON grupo.oid=miembro.roleid
    WHERE login.rolname='vec_ad3_o207_gobierno'
      AND grupo.rolname='vec_contratacion_temporal_gobernador'
    GROUP BY login.rolcanlogin;"
```

El producto debe seguir limpio; la candidata contiene el trabajo compartido
del paso 6 y no se limpia ni se sustituye. El contenedor debe estar `running`, usar la imagen
fijada por digest, montar `vec-ct-o2-07-browser-20260904-data` en
`/var/lib/postgresql` y publicar únicamente `127.0.0.1:55433`. Tras la evidencia
automatizada, los tres recuentos deben ser iguales y mayores que cero; el valor
crece con cada solicitud sintética nueva.
La última consulta debe devolver exclusivamente
`true|vec_autorizacion_registro` y la siguiente
`true|vec_contratacion_temporal_gobernador`.

Antes de arrancar VEC, compruebe que PostgreSQL conserva exactamente las
definiciones de las migraciones de Asignación, informe jurídico y
Fiscalización incluidas en el producto. La comprobación es de solo lectura y
se detiene ante cualquier diferencia:

El núcleo común incluye ahora los cambios de `000011/000012/000013`.
Dirección confirma esta huella instalada en ambas bases (`55432` y `55433`):
`5079499ad2c05ead1afc6e36b3505249b30f7ba98b913c1e5b4e1c942b8d2b57`.
Es la definición ampliada que contrasta el comando siguiente; no debe
compararse con la huella anterior a la recuperación.

```bash
set -euo pipefail

fuente_autorizacion=deploy/postgresql/autorizacion_atestada_v3/migraciones/000008_consumidor_asignacion_v3_atestada.up.sql
fuente_asignacion=deploy/postgresql/contratacion_temporal/migraciones/000050_asignacion_durable_v3_v4.up.sql
fuente_autorizacion_informe=deploy/postgresql/autorizacion_atestada_v3/migraciones/000009_consumidor_informe_juridico_v3_atestada.up.sql
fuente_informe=deploy/postgresql/contratacion_temporal/migraciones/000051_informe_juridico_durable_v4_v5.up.sql
fuente_autorizacion_fiscalizacion=deploy/postgresql/autorizacion_atestada_v3/migraciones/000010_consumidor_fiscalizacion_v3_atestada.up.sql
fuente_fiscalizacion=deploy/postgresql/contratacion_temporal/migraciones/000052_fiscalizacion_durable_v5_v6.up.sql

test "$(sha256sum "$fuente_autorizacion" | awk '{print $1}')" = \
  42f7dfef32464f1a0ce2f3bcb9af035d800b7ceb302917d0652161408488451c
test "$(sha256sum "$fuente_asignacion" | awk '{print $1}')" = \
  cba96bd9a281e583b9f14edc661d3fc3b26b2a4898515c907c4d5f56cdf89e89
test "$(sha256sum "$fuente_autorizacion_informe" | awk '{print $1}')" = \
  32542f535e58f06668987be79ff6f23c66efc04957da964a2986cd79920934b5
test "$(sha256sum "$fuente_informe" | awk '{print $1}')" = \
  4fdbc64117f0f619ce7fb3758ccc63fde08cb21fd5f6ffbfe4cd6fb138be0840
test "$(sha256sum "$fuente_autorizacion_fiscalizacion" | awk '{print $1}')" = \
  a9f79e4486cdf7cf475d66d0f20015170d0c5627c8368fd786b53e0d086845df
test "$(sha256sum "$fuente_fiscalizacion" | awk '{print $1}')" = \
  c3258be1075381d5ce1d077a3ddc25c2e38d2961b87a5c637f0906fe2746ce61

huella_funcion() {
  docker exec vec-ct-o2-07-browser-20260904-tls \
    psql -X -U postgres -d postgres -Atqc \
    "SELECT pg_catalog.encode(public.digest(pg_catalog.convert_to(pg_catalog.pg_get_functiondef('$1'::regprocedure),'UTF8'),'sha256'),'hex');"
}

test "$(huella_funcion 'vec_autorizacion_atestada_v3.consumir_decision_mutacion_v3_interna(text,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  5079499ad2c05ead1afc6e36b3505249b30f7ba98b913c1e5b4e1c942b8d2b57
test "$(huella_funcion 'vec_autorizacion_atestada_v3.registrar_y_consumir_asignacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  a384447d25ef86cab6bbdb99d32a2d478232a972a3c70e6290ea470c4917ce53
test "$(huella_funcion 'vec_contratacion_temporal.asignacion_claves_exactas_v1(jsonb,text[])')" = \
  e9dbba7cb7cb82f287a9edfb8ac43b862ada1d7ad74a196bece82735b28aa2e1
test "$(huella_funcion 'vec_contratacion_temporal.preparar_asignacion_v1(jsonb)')" = \
  977fef59d076ab551718970686ada6ee86c4bc702e5ebe679839098082badf65
test "$(huella_funcion 'vec_contratacion_temporal.consultar_asignacion_v1(jsonb)')" = \
  152a7e7ad94c6cf654aab22ffe74354b46eb18fd956a7aa5249793c54c0a84c8
test "$(huella_funcion 'vec_contratacion_temporal.confirmar_asignacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  f20c58b07740b8e2b5907d1d9e017d649b641812de0ce6ded41c64583ab02276
test "$(huella_funcion 'vec_autorizacion_atestada_v3.registrar_y_consumir_informe_juridico_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  c880864503d228663ea1058ebb4644d09aa191e2944f8ee646e2d2a606f41c8a
test "$(huella_funcion 'vec_contratacion_temporal.preparar_informe_juridico_v1(jsonb)')" = \
  a7f5d8ba0e8fa185a5ba46208e7bf608ff4f5c9d6a5bcbd05be2d4a64d4e4e44
test "$(huella_funcion 'vec_contratacion_temporal.recibo_informe_juridico_v1(text)')" = \
  1cbc9c91c9c712cb30a223e3666dbd7034d87f34c504407914644c93c1ed97ae
test "$(huella_funcion 'vec_contratacion_temporal.confirmar_informe_juridico_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  c71c053abf0f2f98fb01b7534d55f5e5c56e4ede92e81822911c36b7fe1da39e
test "$(huella_funcion 'vec_autorizacion_atestada_v3.registrar_y_consumir_fiscalizacion_v3_atestada(bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  f96438ebd4a445c53240d0741f520fc52073fb4087820afe21b5618519610147
test "$(huella_funcion 'vec_contratacion_temporal.preparar_fiscalizacion_v1(jsonb)')" = \
  34a33a97425f02e59b66c05186c496a419c545e2ab7734b958a15f7be0cfb056
test "$(huella_funcion 'vec_contratacion_temporal.recibo_fiscalizacion_v1(text)')" = \
  27fc25b3d1fb12431db2e29ccad52f25d56344ea5ec5540a63e916c918dfaa36
test "$(huella_funcion 'vec_contratacion_temporal.confirmar_fiscalizacion_v1(jsonb,bytea,bytea,bytea,bytea,numeric,numeric,bytea,bytea,bytea,bytea)')" = \
  27e369c38267d18d3ed258eca1a986c5835717d9c9630740d2291033fddeb147

printf '%s\n' 'OK: Asignación, informe y Fiscalización coinciden'
```

Si alguna comparación falla, no abra el navegador ni reaplique migraciones a
ciegas: el código y la instancia no están alineados.

## 2. Arrancar VEC con PostgreSQL real

En la misma terminal remota, desde la candidata indicada, con las seis
conexiones separadas a `55433`:

```bash
directorio_pg=$(mktemp -d /tmp/vec-o2-07-pg.XXXXXX)
docker cp \
  vec-ct-o2-07-browser-20260904-tls:/var/lib/postgresql/18/docker/o207-ca.crt \
  "$directorio_pg/ca.crt"
chmod 600 "$directorio_pg/ca.crt"

export VEC_CT_DATABASE_URL="postgresql://vec_ct_o207_runtime@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_CT_GOBIERNO_DATABASE_URL="postgresql://vec_ad3_o207_gobierno@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_CT_CONFIRMADOR_DATABASE_URL="postgresql://vec_ct_o207_confirmador@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_CT_LECTOR_RESULTADO_DATABASE_URL="postgresql://vec_ct_o207_lector@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL="postgresql://vec_autorizacion_o207_registro@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"
export VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL="postgresql://vec_bolsa_llamamientos_desarrollo@localhost:55433/postgres?sslmode=verify-full&sslrootcert=$directorio_pg/ca.crt"

material_vec=/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904
scripts/arrancar_vec_desarrollo.sh --puerto 8443 \
  --directorio-material "$material_vec"
```

El lanzador genera o valida las credenciales fuera de Git, construye un binario
temporal y escucha solo en `127.0.0.1:8443` con TLS 1.3 y certificado de cliente
obligatorio. Se deja en primer plano. No es un despliegue.

## 3. Abrir el túnel y preparar el certificado del navegador

En una segunda terminal del equipo desde el que se abrirá el navegador:

```bash
directorio_navegador=$(mktemp -d /tmp/vec-o2-07-navegador.XXXXXX)
chmod 700 "$directorio_navegador"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/ca/ca.crt \
  "$directorio_navegador/ca.crt"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls/cliente.p12 \
  "$directorio_navegador/cliente.p12"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls/cliente.p12.password \
  "$directorio_navegador/cliente.p12.password"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls/intervencion.p12 \
  "$directorio_navegador/intervencion.p12"
scp root@cidonia.cloud:/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls/intervencion.p12.password \
  "$directorio_navegador/intervencion.p12.password"
chmod 600 "$directorio_navegador"/*
ssh -N -L 18443:127.0.0.1:8443 root@cidonia.cloud
```

No muestre ni copie las contraseñas en la consola. Impórtelas directamente
desde sus ficheros cuando el navegador las pida. Use perfiles temporales
separados: `cliente.p12` representa RRHH e `intervencion.p12` representa
Intervención; ninguno sustituye al otro.

1. Antes de importar el certificado personal, abra
   `https://localhost:18443/portal-empleado/` en un perfil temporal: el acceso
   debe quedar bloqueado por falta de certificado de cliente. Eso demuestra la
   denegación predeterminada.
2. Importe `ca.crt` como autoridad de confianza para sitios web.
3. Importe `cliente.p12` como certificado personal usando su fichero de
   contraseña, cierre el perfil anterior y abra uno nuevo para RRHH.
4. Acceda a `https://localhost:18443/portal-empleado/` y elija
   **Contratación temporal** → **Nueva petición**.

Al acabar, cierre el perfil temporal. Elimine el directorio temporal mediante
el mecanismo de papelera o borrado seguro aprobado en su equipo; contiene una
credencial de desarrollo.

## 4. Registrar una solicitud, analizarla, decidir la cobertura, asignarla, informar y fiscalizar

1. Seleccione un centro, una persona de contacto referenciada, una categoría,
   un grupo o subgrupo y un motivo de los catálogos mostrados.
2. Escriba un detalle sintético inequívocamente nuevo, por ejemplo
   `Prueba manual Alberto 2026-09-04 07:15`.
3. Indique fechas válidas y marque que no existe retención de crédito para este
   dato sintético.
4. Pulse **Revisar solicitud** y luego **Confirmar y registrar** una sola vez.
5. En las herramientas de red del navegador debe aparecer un único
   `POST /api/vec/contratacion-temporal/solicitudes` con estado `201`.
6. La pantalla debe mostrar **Solicitud registrada** y un recibo real con
   referencia de expediente, número visible, versión, referencia de recibo y
   fecha de confirmación. Anote esas referencias.

Si se vuelve a enviar exactamente el mismo formulario con una clave nueva, la
aplicación lo rechaza con `409` para impedir un duplicado semántico. No repita
la solicitud: revise el expediente existente. Cambie el detalle solo cuando
se trate realmente de otra petición.

Compruebe el último recibo persistido desde una tercera terminal remota:

```bash
ssh root@cidonia.cloud \
  "docker exec vec-ct-o2-07-browser-20260904-tls psql -X -U postgres -d postgres -P pager=off -c \"SELECT numero_visible, expediente_ref, recibo_ref, version_expediente, confirmada_en FROM vec_contratacion_temporal.confirmacion_agregado_alta ORDER BY confirmada_en DESC LIMIT 1;\""
```

Los identificadores de PostgreSQL deben coincidir con los del recibo visible.

Después del recibo de Alta aparece el formulario **Análisis por Recursos
Humanos**. Para repetir el recorrido acreditado:

1. Seleccione **Sustitución**, **Categoría C2**, **Grupo C2** y **Necesidad
   temporal**.
2. Use las fechas `2027-01-01` y `2027-03-31`, jornada `10000` y la entrada
   **Retención de crédito sintética 001**.
3. Pulse **Registrar análisis** una sola vez. En la red debe aparecer un único
   `POST /api/vec/contratacion-temporal/analisis/registros` con estado `201`.
4. El segundo recibo debe mostrar la misma referencia de expediente, versión
   resultante `2`, una referencia `rec_ct_an_…` y fecha de confirmación.

Compruebe el último Análisis persistido:

```bash
ssh root@cidonia.cloud \
  "docker exec vec-ct-o2-07-browser-20260904-tls psql -X -U postgres -d postgres -P pager=off -c \"SELECT recibo_json->>'expediente_ref' AS expediente_ref, recibo_json->>'version_resultante' AS version_resultante, recibo_json->>'recibo_ref' AS recibo_ref, confirmada_en FROM vec_contratacion_temporal.confirmacion_operacion_analisis ORDER BY confirmada_en DESC LIMIT 1;\""
```

Los tres datos deben coincidir con el segundo recibo visible. Las cinco
modalidades disponibles —Sustitución, Vacante, Acumulación de tareas, Programa
y Relevo— pasan por el mismo servicio real; el recorrido acreditado usa
Sustitución.

Después del recibo de Análisis aparece **Decidir la vía de cobertura**:

1. Espere a que la propuesta muestre estado viable y vía recomendada
   **Bolsa vigente**. En la red debe aparecer un único
   `POST /api/vec/contratacion-temporal/cobertura/propuesta` con estado `200`.
2. Pulse **Confirmar vía de cobertura** una sola vez y acepte el diálogo de
   confirmación. Anote la `clave_idempotencia` del cuerpo de la petición en las
   herramientas de red; se utilizará solo para consultar el resultado.
3. Debe aparecer un único
   `POST /api/vec/contratacion-temporal/cobertura/decisiones` con estado `201`.
4. El tercer recibo debe mostrar el mismo expediente, versión resultante `3`,
   una referencia `recibo:ct:cobertura:…`, la referencia de la decisión y la
   fecha de confirmación.

Compruebe decisión, autorización y recibo persistidos sustituyendo las dos
referencias por las que muestra la pantalla:

```bash
expediente_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_EXPEDIENTE'
recibo_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_TERCER_RECIBO'
docker exec -i vec-ct-o2-07-browser-20260904-tls \
  psql -X -v ON_ERROR_STOP=1 -v expediente_ref="$expediente_ref" \
  -v recibo_ref="$recibo_ref" -U postgres -d postgres -P pager=off <<'SQL'
BEGIN TRANSACTION READ ONLY;
SELECT d.expediente_ref, d.version_expediente, d.recibo_ref,
       d.decision_ref, d.via_elegida, d.persistida_en,
       count(a.decision_ref) AS concesiones_autorizacion
  FROM vec_contratacion_temporal.decision_cobertura_gobernada_durable d
  JOIN vec_autorizacion.decision_concedida_contexto_actor_v3 a
    ON a.decision_ref=d.decision_vec_ref
 WHERE d.expediente_ref=:'expediente_ref'
   AND d.recibo_ref=:'recibo_ref'
 GROUP BY d.decision_ref;
COMMIT;
SQL
```

Debe devolver una sola fila, versión `3`, vía `bolsa_vigente`, las mismas
referencias visibles y `concesiones_autorizacion = 1`.

Después del recibo de Cobertura aparece **Asignar expediente a la unidad
responsable**:

1. Compruebe que la unidad mostrada es `unidad:desarrollo:rrhh` y la persona
   responsable seleccionada es `persona:responsable-sintetica-001`.
2. Marque **He comprobado el expediente, la unidad y la referencia
   responsable**.
3. Pulse **Confirmar asignación** una sola vez y acepte el diálogo de
   confirmación.
4. En la red debe aparecer un único
   `POST /api/vec/contratacion-temporal/asignaciones` con estado `201`. Conserve
   su cuerpo exacto de cinco campos para la comprobación posterior al reinicio.
5. El cuarto recibo debe mostrar el mismo expediente, versión resultante `4`,
   una referencia de recibo y la fecha de confirmación.

Compruebe la Asignación persistida sustituyendo la referencia por la visible:

```bash
expediente_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_EXPEDIENTE'
docker exec -i vec-ct-o2-07-browser-20260904-tls \
  psql -X -v ON_ERROR_STOP=1 -v expediente_ref="$expediente_ref" \
  -U postgres -d postgres -P pager=off <<'SQL'
BEGIN TRANSACTION READ ONLY;
SELECT r.expediente_ref,
       t.recibo_json->>'version_resultante' AS version_resultante,
       t.recibo_json->>'recibo_ref' AS recibo_ref,
       r.unidad_ref, r.responsable_ref, r.estado,
       count(*) OVER () AS terminales_del_expediente
  FROM vec_contratacion_temporal.reserva_asignacion r
  JOIN vec_contratacion_temporal.terminal_asignacion t
    USING (ambito_hmac)
 WHERE r.expediente_ref=:'expediente_ref';
COMMIT;
SQL
```

Debe devolver una sola fila: versión `4`, estado `confirmada`, la misma
referencia de recibo, la unidad y la persona responsable mostradas y
`terminales_del_expediente = 1`.

Después del recibo de Asignación aparece **Preparar informe jurídico**:

1. Compruebe que la pantalla muestra el mismo expediente y la versión asignada
   `4`.
2. Marque **He comprobado el expediente y entiendo que se generará un
   documento de desarrollo sin firma**.
3. Pulse **Confirmar y preparar informe** una sola vez y acepte el diálogo.
4. En la red debe aparecer un único
   `POST /api/vec/contratacion-temporal/informes-juridicos/preparaciones` con
   estado `201`. Conserve el cuerpo exacto de tres campos para comprobar la
   repetición después del reinicio.
5. La pantalla debe mostrar el quinto recibo, la versión resultante `5` y el
   contenido del documento encabezado por
   **DOCUMENTO DE DESARROLLO — SIN FIRMA NI VALIDEZ JURIDICA**.

Compruebe informe, documento y autorización persistidos:

```bash
expediente_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_EXPEDIENTE'
docker exec -i vec-ct-o2-07-browser-20260904-tls \
  psql -X -v ON_ERROR_STOP=1 -v expediente_ref="$expediente_ref" \
  -U postgres -d postgres -P pager=off <<'SQL'
BEGIN TRANSACTION READ ONLY;
SELECT r.expediente_ref, r.estado,
       t.documento_ref, t.decision_ref, t.confirmada_en,
       a.version AS version_resultante,
       v.agregado_json->>'fase_actual' AS fase_actual,
       count(DISTINCT d.documento_ref) AS documentos,
       count(DISTINCT c.decision_ref) AS consumos_autorizacion
  FROM vec_contratacion_temporal.reserva_informe_juridico r
  JOIN vec_contratacion_temporal.terminal_informe_juridico t
    USING (ambito_hmac)
  JOIN vec_contratacion_temporal.documento_informe_juridico_desarrollo d
    USING (ambito_hmac)
  JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 c
    ON c.decision_ref=t.decision_ref
  JOIN vec_contratacion_temporal.expediente_integral_actual a
    USING (expediente_ref)
  JOIN vec_contratacion_temporal.expediente_version_integral v
    USING (expediente_ref, version)
 WHERE r.expediente_ref=:'expediente_ref'
 GROUP BY r.expediente_ref, r.estado, t.documento_ref, t.decision_ref,
          t.confirmada_en, a.version, v.agregado_json;
COMMIT;
SQL
```

Debe devolver una fila en estado `confirmada`, versión `5`, fase
`informe_juridico`, un documento y un consumo de autorización. Las referencias
de documento y recibo de la pantalla deben corresponder al mismo resultado.

Después del informe aparece **Registrar resultado de Fiscalización**. Esta
operación pertenece a Intervención, no a RRHH. En el perfil que vaya a enviarla
debe estar instalado `intervencion.p12`; una petición hecha con `cliente.p12`
queda denegada sin escritura.

En el entorno de desarrollo, si el navegador ya ha fijado el certificado de
RRHH para ese origen, abra un perfil temporal separado que contenga solo la CA
y `intervencion.p12`. Abra el portal normalmente y entre en
**Contratación temporal**. La pantalla separada de Intervención no contiene las
funciones de alta o análisis reservadas a RRHH. Pegue la referencia del
expediente mostrada en el recibo anterior, conserve la versión remitida `5` y
pulse **Abrir fiscalización**. Aparecerá el formulario real **Registrar
resultado de Fiscalización**. No abra la consola ni llame directamente a la
API.

1. Seleccione **Favorable**, **Favorable con observaciones** o
   **Desfavorable**. Los dos últimos exigen observaciones.
2. Para comprobar la vuelta a la unidad elija **Desfavorable**, escriba una
   observación inequívocamente sintética y pulse **Registrar resultado** una
   sola vez.
3. En la red debe aparecer un único
   `POST /api/vec/contratacion-temporal/fiscalizaciones/resultados` con estado
   `201`. Conserve su cuerpo exacto de cinco campos para el reinicio.
4. El sexto recibo visible debe indicar versión `6`, fase
   `subsanacion_unidad`, estado `incidencia`, una referencia de auditoría y el
   retorno a `unidad:desarrollo:rrhh` con
   `persona:responsable-sintetica-001`.

Compruebe la historia y el retorno persistidos:

```bash
expediente_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_EXPEDIENTE'
docker exec -i vec-ct-o2-07-browser-20260904-tls \
  psql -X -v ON_ERROR_STOP=1 -v expediente_ref="$expediente_ref" \
  -U postgres -d postgres -P pager=off <<'SQL'
BEGIN TRANSACTION READ ONLY;
SELECT a.version, v.fase_clave, v.estado,
       r.resultado, r.estado AS estado_reserva,
       u.estado AS estado_retorno, u.unidad_ref, u.responsable_ref,
       t.auditoria_ref,
       count(DISTINCT c.decision_ref) AS consumos_autorizacion
  FROM vec_contratacion_temporal.expediente_integral_actual a
  JOIN vec_contratacion_temporal.expediente_version_integral v
    USING (expediente_ref, version)
  JOIN vec_contratacion_temporal.reserva_fiscalizacion r
    USING (expediente_ref)
  JOIN vec_contratacion_temporal.terminal_fiscalizacion t
    USING (ambito_hmac)
  JOIN vec_contratacion_temporal.retorno_fiscalizacion_unidad u
    USING (ambito_hmac)
  JOIN vec_autorizacion_atestada_v3.consumo_decision_v3 c
    ON c.decision_ref=t.decision_ref
 WHERE a.expediente_ref=:'expediente_ref'
 GROUP BY a.version, v.fase_clave, v.estado, r.resultado, r.estado,
          u.estado, u.unidad_ref, u.responsable_ref, t.auditoria_ref;
COMMIT;
SQL
```

Debe devolver una fila: versión `6`, fase `subsanacion_unidad`, estado
`incidencia`, reserva `confirmada`, retorno `pendiente` y un único consumo de
autorización.

## 5. Demostrar que sobrevive al reinicio

1. En la primera terminal pulse `Ctrl-C`. Solo debe terminar VEC; no detenga
   PostgreSQL.
2. Sin cambiar las seis variables exportadas ni el worktree, arranque otra
   vez:

```bash
scripts/arrancar_vec_desarrollo.sh --puerto 8443 \
  --directorio-material /root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904
```

3. Repita las seis consultas SQL del apartado anterior. Deben devolver las
   mismas referencias después del reinicio.
4. Consulte el resultado de cobertura, sin repetir la decisión, con la clave
   anotada en las herramientas de red:

```bash
expediente_ref='PEGUE_AQUI_LA_REFERENCIA_DEL_EXPEDIENTE'
clave_idempotencia='PEGUE_AQUI_LA_CLAVE_DE_LA_DECISION'
cuerpo_consulta=$(EXPEDIENTE_REF="$expediente_ref" \
  CLAVE_IDEMPOTENCIA="$clave_idempotencia" python3 -c \
  'import json, os; print(json.dumps({"expediente_ref": os.environ["EXPEDIENTE_REF"], "clave_idempotencia": os.environ["CLAVE_IDEMPOTENCIA"]}, separators=(",", ":")))')
curl --silent --show-error --fail-with-body \
  --cacert /root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/ca/ca.crt \
  --cert /root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls/cliente.crt \
  --key /root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls/cliente.key \
  -H 'Accept: application/json' \
  -H 'Content-Type: application/json; charset=utf-8' \
  --data-binary "$cuerpo_consulta" \
  https://localhost:8443/api/vec/contratacion-temporal/cobertura/resultados
```

La respuesta debe indicar `confirmado` e incluir exactamente el mismo tercer
recibo. Esta consulta no repite ni modifica la decisión.

5. En la consola del mismo navegador, repita el cuerpo exacto de cinco campos
   que conservó de Asignación, sin generar otra clave:

```javascript
const cuerpoAsignacion = PEGUE_AQUI_EL_OBJETO_JSON_DE_CINCO_CAMPOS;
await fetch("/api/vec/contratacion-temporal/asignaciones", {
  method: "POST",
  credentials: "same-origin",
  headers: {
    Accept: "application/json",
    "Content-Type": "application/json; charset=utf-8",
  },
  body: JSON.stringify(cuerpoAsignacion),
}).then(async (respuesta) => ({ estado: respuesta.status, cuerpo: await respuesta.json() }));
```

Debe responder otra vez `201`, con el mismo expediente, versión `4` y
referencia de recibo. Repita la consulta SQL de Asignación: debe seguir habiendo
un único terminal para ese expediente. El recibo interno contiene trazabilidad
adicional que no expone la API; compare semánticamente los campos públicos, no
el texto JSON literal. Esta repetición recupera el resultado y no crea una
segunda asignación.

6. En la consola del navegador repita también el cuerpo exacto de tres campos
   conservado del informe, sin generar otra clave:

```javascript
const cuerpoInforme = PEGUE_AQUI_EL_OBJETO_JSON_DE_TRES_CAMPOS;
await fetch("/api/vec/contratacion-temporal/informes-juridicos/preparaciones", {
  method: "POST",
  credentials: "same-origin",
  headers: {
    Accept: "application/json",
    "Content-Type": "application/json; charset=utf-8",
  },
  body: JSON.stringify(cuerpoInforme),
}).then(async (respuesta) => ({ estado: respuesta.status, cuerpo: await respuesta.json() }));
```

Debe responder otra vez `201` con el mismo informe, documento, recibo, huella
e instante de confirmación. La consulta SQL debe seguir mostrando un único
documento y un único consumo de autorización para ese informe.

7. En el perfil temporal de Intervención, repita el cuerpo exacto de cinco
   campos conservado de Fiscalización, sin generar otra clave:

```javascript
const cuerpoFiscalizacion = PEGUE_AQUI_EL_OBJETO_JSON_DE_CINCO_CAMPOS;
await fetch("/api/vec/contratacion-temporal/fiscalizaciones/resultados", {
  method: "POST",
  credentials: "same-origin",
  headers: {
    Accept: "application/json",
    "Content-Type": "application/json; charset=utf-8",
  },
  body: JSON.stringify(cuerpoFiscalizacion),
}).then(async (respuesta) => ({ estado: respuesta.status, cuerpo: await respuesta.json() }));
```

Debe responder otra vez `201` con exactamente el mismo recibo, auditoría,
evento, instante y retorno. La consulta SQL de Fiscalización debe seguir
mostrando una sola reserva, un solo terminal, un solo retorno y un solo consumo
de autorización.

La consulta protegida anterior recupera el resultado de cobertura. Alta y
Análisis todavía no tienen una vista que recargue sus recibos cerrados después
de desmontar la página; su persistencia se comprueba con las consultas SQL.

## 6. Llamamiento y aviso local — recorrido real recuperado tras reinicio

**Cinco pasos completos más este tramo del sexto recorrible.** Dirección
comprobó en navegador real selección `200` y comunicación local `201`.
Tras reiniciar VEC a las `00:46:33 UTC` del 5 de septiembre de 2026, abrió
un contexto nuevo de navegador y repitió las mismas claves: `200/200`,
mismo JSON de selección y mismo recibo de comunicación, ahora con estado
`replay_registrada_localmente`. Se conservaron las claves del servidor,
PostgreSQL y el directorio de material; no se depende de almacenamiento web.

Datos sintéticos ya registrados, para recuperar sin crear otro expediente:

- Expediente: `expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001`;
  versión de entrada `6`, fiscalización favorable.
- Clave de selección: `90d52c16-a63d-4ef1-bcf7-62c7c455f9aa`.
  Recibo: `recibo:pldjefhkpgifphgejphkpgcmkjlkphjgmfiedehlnpbphnoefikkcfmjancmdhce`;
  fecha conservada: `2026-09-05T00:45:25.244696Z`.
- Clave de comunicación: `d1b5428f-2188-4b8f-98b7-42f82ad88c2a`.
  Recibo: `recibo:68d32388-b423-4483-b7b1-2fcd091624a8`;
  fecha conservada: `2026-09-05T00:45:25.506363Z`.
  Intención de aviso: `outbox:2e3b64dc-ec3c-4076-bbdb-f036090aaa64`.

### Recorrerlo a mano

1. Arranque la candidata y abra el túnel con los comandos de los apartados
   2 y 3. Use el certificado de RRHH y abra
   **Contratación temporal → Nueva petición → Llamamiento y comunicación**.
   No registre otra solicitud ni repita la fiscalización.
2. Introduzca el expediente anterior, versión `6` y su clave de selección.
   Pulse **Revisar e iniciar llamamiento** y confirme. La ruta
   `POST /api/vec/contratacion-temporal/llamamientos/seleccion` devuelve
   `200` y el recibo anterior. No pulse **Preparar clave nueva**.
3. En **Registrar comunicación**, organización, llamamiento, versión `1` y
   recibo antecedente se rellenan desde la selección real. No invente ni
   cambie esas referencias. Introduzca la clave de comunicación anterior,
   pulse **Revisar y registrar comunicación** y confirme.
4. La repetición devuelve `200`, **Registro local recuperado · Sin entrega
   acreditada**, el mismo recibo, fecha e intención. La primera escritura
   devolvió `201`. La versión local resultante `2` no es la del expediente.
5. Para comprobar otro reinicio, termine solo VEC con `Ctrl-C` en su terminal,
   conserve las seis conexiones del apartado 2 y arranque desde la misma
   candidata, sin regenerar material ni detener PostgreSQL:

```bash
scripts/arrancar_vec_desarrollo.sh --puerto 8443 \
  --directorio-material /root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904
```

Abra un contexto nuevo de navegador con el certificado de RRHH y repita
los puntos 2–4 con exactamente los mismos datos. Compare los recibos y
fechas anteriores; no espere un recibo nuevo.

### Comprobar lo persistido, sin modificarlo

La revisión independiente contrastó en lectura real una orden original
(`2026-09-05T00:07:16.220132Z`), una propuesta, un llamamiento y un terminal
de selección. La recuperación conserva una historia y un evento pendiente,
con contador de exclusión `1 → 2`. Comunicación, historia y evento pendiente
también tienen una fila cada uno. El intento interrumpido se recuperó con su
misma clave, sin borrar la orden ni duplicar esos efectos.

```bash
expediente_ref='expediente:ct:5fe7e60e7632213e9f20cee64aa0e8fb913187513d728da76a4c6de54c49c001'
docker exec -i vec-ct-o2-07-browser-20260904-tls \
  psql -X -v ON_ERROR_STOP=1 -v expediente_ref="$expediente_ref" \
  -U postgres -d postgres -P pager=off <<'SQL'
BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY;
SELECT count(*) OVER () AS ejecuciones, s.situacion,
       s.recibo_json->>'recibo_ref' AS recibo_seleccion,
       s.recibo_json->>'llamamiento_ref' AS llamamiento,
       c.estado, c.recibo_json->>'ReciboRef' AS recibo_comunicacion,
       c.registrada_en, c.recibo_json->>'IntencionEnvioRef' AS intencion,
       (SELECT count(*) FROM vec_contratacion_temporal.historia_comunicacion_llamamiento_local h
         WHERE h.comunicacion_ref=c.comunicacion_ref) AS historia,
       (SELECT count(*) FROM vec_contratacion_temporal.outbox_comunicacion_llamamiento_local o
         WHERE o.comunicacion_ref=c.comunicacion_ref AND o.estado='pendiente') AS outbox_pendiente
  FROM vec_contratacion_temporal.ejecucion_seleccion_llamamiento_o6 s
  LEFT JOIN vec_contratacion_temporal.comunicacion_llamamiento_local c
    ON c.seleccion_clave=s.clave_idempotencia
 WHERE s.solicitud_json->>'expediente_ref'=:'expediente_ref';
ROLLBACK;
SQL
```

Para el expediente indicado debe aparecer una ejecución `confirmada`, los
recibos anteriores, comunicación `registrada_localmente`, una historia y un
evento pendiente de comunicación. Ese evento por sí solo no acredita el
fichero. Se verificó también el aviso: **803 bytes**, directorio `0700`,
archivo `0600`, SHA256
`d0173929c18b437d44b84fcab632ef60659b3246bb98586a09cc0fc506a48c17`.

En la terminal remota, compruebe tamaño, permisos y huella sin mostrar contenido:

```bash
material_vec=/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904
intencion_envio_ref='outbox:2e3b64dc-ec3c-4076-bbdb-f036090aaa64'
huella_intencion=$(printf '%s' "$intencion_envio_ref" | sha256sum | cut -d ' ' -f 1)
aviso_local="$material_vec/comunicaciones/aviso-$huella_intencion.json"
stat -c '%a %s %n' "$material_vec/comunicaciones" "$aviso_local"
sha256sum "$aviso_local"
```

**Es selección más aviso LOCAL persistido, no correo enviado, entrega,
aceptación ni apertura de plazo.** La fuente de Bolsa es sintética firmada;
los recibos y efectos descritos sí están guardados de verdad.

## Resultado y siguiente paso funcional

De los ocho pasos solicitados por Recursos Humanos, este corte permite recorrer
manualmente 1, **Solicitud**, 2, **Análisis**, 3, **Bolsa**, 4,
**Asignación** y 5, **Informe jurídico y Fiscalización**. El contador funcional
queda en **5 de 8**. Fiscalización acepta los tres resultados reales y el
desfavorable devuelve automáticamente el expediente a la Unidad conservando
el histórico completo.

El apartado 6 añade **selección y aviso local recorribles**, con los mismos
recibos recuperados tras reiniciar. El resultado es **5 pasos completos más
un tramo del sexto**, no el 100 % del llamamiento corporativo. Faltan envío y
entrega reales, aceptación o renuncia y gestión del plazo; no están
acreditados por un aviso local. Después quedan 7, **Nombramiento**, con sus
seis documentos, incluida la Diligencia, y 8, **Incorporación, GINPIX y
Seguimiento**.

La evidencia corresponde a la línea de trabajo indicada. Es un recorrido de
desarrollo: ni su integración ni su publicación en GitHub autorizan producción
o sustituyen la aceptación de Recursos Humanos.

## Apéndice: recreación segura de la instancia aislada

No ejecute este apéndice durante la repetición pendiente de Dirección. Los
recursos actuales no llevan etiqueta de propiedad porque preceden a esta guía;
el script se negará a borrarlos. Su retirada inicial exige inspección y
confirmación humana explícita de los nombres, digest, volumen, montaje, puerto,
ausencia de clientes y ausencia de un proceso `vec-server`.

Después de esa retirada inicial autorizada, el siguiente bloque crea recursos
propios y solo reemplaza recursos que conserven exactamente su etiqueta:

```bash
set -Eeuo pipefail

fuente=vec-ct-o2-07-e2e-20260904-tls
volumen_fuente=vec-ct-o2-07-e2e-20260904-a-data
destino=vec-ct-o2-07-browser-20260904-tls
volumen_destino=vec-ct-o2-07-browser-20260904-data
imagen='postgres@sha256:882236b897e39051d2368c5ccc6cda944904723506b2dfc97f2a8f5bc9afa382'
propietario='o2-07-browser-guide-v1'
clave_etiqueta='es.dipgra.vec.propietario'

if pgrep -af '/vec-server'; then
  printf '%s\n' 'ERROR: detenga VEC antes de clonar PostgreSQL' >&2
  exit 1
fi

test "$(docker inspect "$fuente" --format '{{.State.Status}}')" = running
test "$(docker inspect "$fuente" --format '{{.Config.Image}}')" = "$imagen"
test "$(docker inspect "$fuente" --format '{{range .Mounts}}{{if eq .Destination "/var/lib/postgresql"}}{{.Name}}{{end}}{{end}}')" = "$volumen_fuente"
test "$(docker inspect "$fuente" --format '{{(index (index .HostConfig.PortBindings "5432/tcp") 0).HostIp}}:{{(index (index .HostConfig.PortBindings "5432/tcp") 0).HostPort}}')" = '127.0.0.1:55432'
docker image inspect "$imagen" >/dev/null

estado() {
  docker exec "$1" psql -X -U postgres -d postgres -Atqc "SELECT concat_ws('/',
    (SELECT count(*) FROM vec_autorizacion_atestada_v3.clave_capacidad_version),
    (SELECT COALESCE(max(orden),0) FROM vec_autorizacion_atestada_v3.puntero_clave_emision),
    (SELECT revision FROM vec_autorizacion_atestada_v3.checkpoint_gobierno WHERE control_id),
    (SELECT secuencia FROM vec_autorizacion_atestada_v3.control_cadena_auditoria WHERE control_id));"
}

lector() {
  docker exec "$1" psql -X -U postgres -d postgres -Atqc "SELECT count(*)
    FROM pg_catalog.pg_shdepend
    WHERE refclassid='pg_catalog.pg_authid'::pg_catalog.regclass
      AND refobjid='vec_contratacion_temporal_lector_resultado_cobertura'::pg_catalog.regrole;"
}

test "$(docker exec "$fuente" psql -X -U postgres -d postgres -Atqc "SELECT count(*) FROM pg_stat_activity WHERE backend_type='client backend' AND pid <> pg_backend_pid();")" = 0
test "$(estado "$fuente")" = '23/23/23/13'
test "$(lector "$fuente")" = 3

if docker container inspect "$destino" >/dev/null 2>&1; then
  test "$(docker inspect "$destino" --format "{{index .Config.Labels \"$clave_etiqueta\"}}")" = "$propietario" || {
    printf '%s\n' 'ERROR: contenedor existente sin propiedad exacta; no se toca' >&2
    exit 1
  }
  docker stop "$destino"
  docker rm "$destino"
fi

if docker volume inspect "$volumen_destino" >/dev/null 2>&1; then
  test "$(docker volume inspect "$volumen_destino" --format "{{index .Labels \"$clave_etiqueta\"}}")" = "$propietario" || {
    printf '%s\n' 'ERROR: volumen existente sin propiedad exacta; no se toca' >&2
    exit 1
  }
  docker volume rm "$volumen_destino"
fi

docker volume create --label "$clave_etiqueta=$propietario" "$volumen_destino"
fuente_detenida=false
restaurar_fuente() {
  if test "$fuente_detenida" = true; then
    docker start "$fuente" >/dev/null
  fi
}
trap restaurar_fuente EXIT

docker stop "$fuente"
fuente_detenida=true
docker run --rm --pull=never --network none --read-only \
  --label "$clave_etiqueta=$propietario" \
  --mount "type=volume,src=$volumen_fuente,dst=/origen,readonly" \
  --mount "type=volume,src=$volumen_destino,dst=/destino" \
  --entrypoint /bin/sh "$imagen" -ceu '
    test -f /origen/PG_VERSION
    test ! -e /destino/PG_VERSION
    cp -a /origen/. /destino/
    sync
  '
docker start "$fuente" >/dev/null
fuente_detenida=false
trap - EXIT

test "$(estado "$fuente")" = '23/23/23/13'
test "$(lector "$fuente")" = 3

docker run -d --name "$destino" --pull=never --restart=no \
  --label "$clave_etiqueta=$propietario" \
  --publish 127.0.0.1:55433:5432 \
  --mount "type=volume,src=$volumen_destino,dst=/var/lib/postgresql" \
  "$imagen"

for intento in $(seq 1 100); do
  if docker exec "$destino" pg_isready -U postgres -d postgres >/dev/null 2>&1; then
    break
  fi
  test "$intento" -lt 100
  sleep 0.1
done

docker exec -i "$destino" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
DO $provision$
BEGIN
  IF NOT EXISTS (
    SELECT FROM pg_catalog.pg_roles
    WHERE rolname='vec_autorizacion_o207_registro'
  ) THEN
    CREATE ROLE vec_autorizacion_o207_registro LOGIN;
  END IF;
END
$provision$;
GRANT vec_autorizacion_registro TO vec_autorizacion_o207_registro;
GRANT vec_contratacion_temporal_gobernador TO vec_ad3_o207_gobierno;
SQL

test "$(docker exec "$destino" psql -X -U postgres -d postgres -Atqc "SELECT login.rolcanlogin::text||'|'||string_agg(grupo.rolname,',' ORDER BY grupo.rolname) FROM pg_roles login JOIN pg_auth_members miembro ON miembro.member=login.oid JOIN pg_roles grupo ON grupo.oid=miembro.roleid WHERE login.rolname='vec_autorizacion_o207_registro' GROUP BY login.rolcanlogin;")" = 'true|vec_autorizacion_registro'
test "$(docker exec "$destino" psql -X -U postgres -d postgres -Atqc "SELECT login.rolcanlogin::text||'|'||string_agg(grupo.rolname,',' ORDER BY grupo.rolname) FROM pg_roles login JOIN pg_auth_members miembro ON miembro.member=login.oid JOIN pg_roles grupo ON grupo.oid=miembro.roleid WHERE login.rolname='vec_ad3_o207_gobierno' AND grupo.rolname='vec_contratacion_temporal_gobernador' GROUP BY login.rolcanlogin;")" = 'true|vec_contratacion_temporal_gobernador'

docker exec -i "$destino" psql -X -v ON_ERROR_STOP=1 -U postgres -d postgres <<'SQL'
BEGIN;
SET LOCAL session_replication_role = replica;
TRUNCATE TABLE
  vec_autorizacion_atestada_v3.clave_capacidad_version,
  vec_autorizacion_atestada_v3.puntero_clave_emision,
  vec_autorizacion_atestada_v3.revocacion_clave_capacidad,
  vec_autorizacion_atestada_v3.configuracion_confianza_version,
  vec_autorizacion_atestada_v3.raiz_confianza_version,
  vec_autorizacion_atestada_v3.configuracion_raiz,
  vec_autorizacion_atestada_v3.puntero_configuracion_actual,
  vec_autorizacion_atestada_v3.revocacion_configuracion,
  vec_autorizacion_atestada_v3.revocacion_raiz,
  vec_autorizacion_atestada_v3.atestacion_decision_v3,
  vec_autorizacion_atestada_v3.consumo_decision_v3,
  vec_autorizacion_atestada_v3.auditoria_consumo_v3;
UPDATE vec_autorizacion_atestada_v3.checkpoint_gobierno
SET revision=0, configuracion_secuencia_minima=0, raiz_version_minima=0,
    actualizada_en=clock_timestamp()
WHERE control_id;
UPDATE vec_autorizacion_atestada_v3.control_cadena_auditoria
SET secuencia=0, cabeza_sha256=pg_catalog.repeat('0',64),
    actualizada_en=clock_timestamp()
WHERE control_id;
COMMIT;
SQL

test "$(estado "$destino")" = '0/0/0/0'
test "$(lector "$destino")" = 3
test "$(estado "$fuente")" = '23/23/23/13'
test "$(lector "$fuente")" = 3
```

El bloque no elimina datos de la fuente. Detiene la fuente solo durante la copia
en frío, la vuelve a arrancar y reinicia únicamente el gobierno de autorización
en el clon. La aplicación inicializa ese gobierno al arrancar contra `55433`.
