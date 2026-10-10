# Bolsa de trabajo

**Estado a 8 de octubre de 2026:** Bolsa funciona en cidonia con HZ8 instalado (canales del llamamiento y seguimiento por teléfono, reglas de RRHH v5, «Mi Bolsa» del aspirante y documentos encendidos); faltan la vuelta a la bolsa tras un cese, la carga del Excel de CONVOCA desde una pantalla y el correo de la Diputación.

Base comprobada: `origin/main@033fda6ed` más las PR fusionadas el 08/10 (#895 a #904). Los criterios para dar Bolsa por cerrada están en [OBJETIVOS.md](OBJETIVOS.md). Las preguntas a RRHH se citan por su número en [`dudas.md`](../../dudas.md).

## Ceses sin candidato — revisión de #774, 8 de octubre de 2026

B81 permite continuar el relevo cuando el llamamiento del puente CT eligió
una participación sin candidato vinculado. Conserva el cese, su motivo y las
referencias del evento y de la auditoría común CT115. Si la bolsa ya está
constituida, comprueba antes que esa participación figura en su acta; una
referencia ajena sigue rechazándose. Cuando aparece el vínculo, B90 mantiene
al candidato pendiente desde la recepción B13. El relevo aplica B45; hasta
entonces no ocupa turno ni aparece disponible en «Mi Bolsa». Si una aplicación
falla, el relevo intenta los demás ceses y vuelve a intentar el fallido en la
pasada siguiente. El replay B81 conserva el registro original.

La rama se actualiza con main y el cursor se toma de la definición instalada
en postHX + HZ + B85 + B86 + CT193 + B87. CT197 añade un testigo técnico que
verifica el cese publicado con CT129 y entrega a Bolsa su `auditoria_ref` exacta.
La función no concede lectura de la tabla CT a Bolsa ni modifica CT129.
CT197, B81 y B90 están reservadas para #774; la lista las instala en ese orden.
B90 calcula el pendiente desde B13 incluso si B81 aún no registró la
proyección. Aplica el corte temporal de cada lectura: un cese futuro no cambia
el orden anterior y una resolución B45 posterior no borra el intervalo
pendiente. La misma migración actualiza el orden, el resumen y «Mi Bolsa» sin
cambiar el consumo V3 existente, y ofrece una lectura por lote. B81 pagina
sus pendientes por posición y referencia para que un fallo no tape los demás.
Claude revisa las SQL exactas y el ensayo final antes de integrar. Ninguna
está instalada en una base compartida. CONFIG NUEVA: ninguna.

En un clon PostgreSQL 18.4 postHX + 14 HZ + B85 + B86 + CT193 + B87, las
tres SQL se instalaron una vez (unos 106 ms cada una). Las pruebas B81/B90
pasaron en ROLLBACK (211/309 ms): bolsa constituida sin vínculo, pertenencia
al acta, replay, cursor paginado, cese pendiente, permisos e historia. Con
20.001 entradas, el resumen global respondió y la fachada limitó la petición
individual. La página de 100 entre 10.000 marcadores tardó 0,076 ms con
índice; el lote de 2.390 estados variados tuvo p95 de 91,879 ms en el ensayo
anterior, con el mismo SQL B90. CT129 verifica el origen; la fixture prepara
CT/B13 directamente, sin acreditar una actuación RRHH ni «Mi Bolsa» con V3
nominal. La línea base HTTP de la fuente anterior dio p95 de 285,223 ms con
2.000 candidaturas y 2.009 consultas por petición, pero B90 estaba apagada:
no mide este cambio. Faltan la revisión SQL final de Claude y la CI del nuevo
hash. La mejora de consultas RRHH va aparte en #917.

## Lo que falta

| # | Qué | Quién o qué lo frena | Cómo se comprueba |
| --- | --- | --- | --- |
| 1 | **Vuelta a la bolsa tras el cese de CT.** En cidonia falta `VEC_BOLSA_CESE_CT_ENABLED` y el LOGIN del relevo. El LOGIN y el entorno están preparados en #771. | Claude, en el despliegue de cidonia. | Cese en CT y la persona vuelve a «Disponible» (o «disponible desde») en su bolsa, también tras reiniciar. |
| 2 | **Cese sin candidato vinculado sin bloquear el relevo** (CT197, B81, B90), también en bolsa constituida si la participación consta en su acta. | PR #774 integrada en `main`; falta instalar y recorrer el relevo en cidonia. | Comprobar alta, cursor, vínculo posterior, restricción B45 y estado pendiente en «Mi Bolsa». |
| 3 | **Carga de bolsas desde el Excel de CONVOCA en pantalla (B1).** #751 conserva el núcleo y CLI segregado; #752 añade vista previa paginada con consumo V3 y confirmación; #759 presenta el recorrido ES/EN. Las tres PR siguen candidatas en ese orden. | Codex-U corrige y verifica; Claude revisa SQL y fusiona. Faltan la concesión B1 gobernada y su asignación nominal; no se crean al arrancar. Duda 144 pendiente. | Verificaciones locales y Chrome con API de prueba; falta RRHH → API nominal → PostgreSQL → recibo y recuperación tras reinicio. |
| 4 | **Recorrido de la oferta con varias plazas y del llamamiento directo**, de punta a punta. El código está; falta el acta. Va dentro del recorrido completo de [OBJETIVOS.md](OBJETIVOS.md). | Claude, con el recorrido de CT. | Acta con varias aceptaciones, adjudicación por posición, «sin respuesta» sin consecuencia y llamamiento directo. |
| 5 | **Reglas que ya no encajan con el Reglamento (duda 147).** El Reglamento (BOP 16/01/2026, arts. 8 y 11) no recoge el plazo de 24 horas por plaza ni «no responder equivale a no aceptar» en la oferta. Basta con cambiar el catálogo. | Dirección decide; luego, cambio de catálogo. | Catálogo nuevo cargado y la pregunta 147 retirada de `dudas.md`. |
| 6 | **Renuncia justificada con el fichero.** Hoy «Mi Bolsa» envía la huella y la referencia del justificante, no el fichero. Con documentos ya encendidos, hay que comprobar si la custodia lo recoge. | Sin dueño. Comprobar antes de encargar. | El justificante se descarga desde la ficha de RRHH. |
| 7 | **Cifras globales de Bolsa pulsables.** Las cifras por bolsa ya llevan a su lista (#883). Los tres totales globales y «llamamientos en curso» necesitan una lectura global paginada con auditoría en la misma transacción. Hoy cuentan participaciones, no personas. | Equipo V. No hay rama. | Cada total abre su lista filtrada con el mismo número. |
| 8 | **Correo de la Diputación configurable desde Administración (B5).** No existe: el servidor de correo solo se configura con variables de entorno. Para la presentación vale el buzón de pruebas actual. | Decidir antes qué se rescata de #179, #204, #206 y #209. Datos del servidor: Informática (duda 88). | Prueba de envío desde Administración, con auditoría. |

**DEPENDENCIA para encender la carga CONVOCA (09/10).** #931 prepara la versión del rol RRHH existente y conserva sus asignaciones; #932 prepara la ratificación de siete descriptores ADMIN que faltan en la copia de ensayo. Ambas PR siguen en borrador. El ensayo ya publicó ADMIN v8/v9 y admitió el catálogo B1 con las CLI reales, pero aún no hizo la propuesta y el cierre con dos personas ADMIN distintas.

La copia conserva sus dos vínculos de certificado, pero los kits disponibles no corresponden a ellos y no hay sesiones ADMIN vigentes. Dirección debe recuperar los dos certificados y claves originales del arranque 2+1, con las fuentes privadas de identidad que exige IS16, para abrir las sesiones y emitir dos decisiones V3 nuevas. Si ese material no se recupera, hace falta un corte separado de renovación gobernada en Identidad; no se sustituyen vínculos mediante SQL manual. Hasta completar ese ensayo y las aprobaciones reales, la carga B1 no está habilitada en cidonia. El orden y las comprobaciones de encendido están en [`habilitacion_b1_convoca_20261009.md`](../../deploy/principal/habilitacion_b1_convoca_20261009.md).

## Esperan a RRHH o al DPD

No se programan hasta tener respuesta. Las propuestas con fuente pública están en el informe de dudas del 08/10 y en la PR #908.

- Lista pública de cada bolsa: qué campos y para quién (dudas 17 y 83, DPD). Propuesta: solo para integrantes identificados, nombre y cuatro cifras del documento, y nunca abierta en las bolsas del turno de discapacidad.
- Causas de baja definitiva y documentación tras aceptar (duda 18). El Reglamento fija seis causas (art. 11) y cuatro renuncias justificadas (art. 10); falta el plazo para aportar documentos.
- Uso del correo y teléfono que vienen de CONVOCA (duda 45, DPD).
- Oferta por correo frente a «dos días desde la publicación» del art. 8.1 del Reglamento: que RRHH confirme que el correo hace de publicación.
- Plazo de respuesta tras contactar por teléfono en el llamamiento directo (duda 151).

Publicación de convocatorias (B7) y baremación con firma (B8) van con Selectivos, que está aparcado. No hacen falta para empezar con las bolsas importadas.

## Ya hecho, no reencargar

- Oferta con varias plazas, adjudicación por posición y llamamiento directo (código): #125, `5427942ba`, `e9cef91f8`.
- Renuncia justificada y «en revisión» (código): #323.
- Asistente antiguo retirado del área personal: #749. La PR #909 completa lo que quedaba (menú y formularios sin servicio, textos por idioma); no repite #749.
- Canales del llamamiento y seguimiento por teléfono (B87): #898.
- Cifras por bolsa pulsables y vuelta atrás con filtro: #883 y #887.
- Ficha que solo pide las secciones disponibles: #901 y #902.
- Recuento de llamamientos sin consultas por bolsa (B85) y marca de completitud (B86): en cidonia desde HZ6.
- Lista pública por páginas: #795. Causas de baja en catálogo: `8d60d22b4`, `d8a0b05dd`.
- Selectores de Bolsa y CT con aviso al arrancar: #745.

Ficheros que conviene no engordar: `internal/app/bootstrap/bolsa_borrador_llamamiento_desarrollo.go`, `web/static/portal-empleado/portal-bolsas-api.js` y `internal/modules/bolsa/domain/llamamientos.go`. Lo nuevo va en ficheros nuevos.

## Candidata: cifras globales y sus listas (Codex-W, 08/10)

Rama `trabajo/codexw-bolsa-global-20261008`: los totales de aspirantes,
disponibles y renuncia abren su relación paginada. Las filas muestran categoría,
posición de acta y situación; «Ver bolsa» abre la relación con nombres ya existente.
Los llamamientos completos tienen su lista por bolsa o general (SQL nueva B94).
La cifra y las páginas conservan el mismo corte durante cinco minutos; al caducar,
la pantalla ofrece actualizarlo. Se cuentan participaciones, no personas distintas.

Ensayo B94: PostgreSQL 18.4 propio, postHX + 14 HZ + B85 + B86 + CT193 + B87.
Sin reconstruir SQL instalada ni cambiar permisos o configuración. La lectura usa
la vía de Bolsa de main admitida por Dirección el 08/10 a las 19:12; no cierra C4
ni acredita auditoría nominal común en cada lectura. Revisión, CI e instalación
pendientes de Dirección; no se declara terminado el punto 7 de este plan.

Medición con 2.390 participaciones: lector de cuatro consultas p95 19,96 ms
(100 muestras); manejador HTTP de resumen p95 17,31 ms y páginas p95 < 1 ms
(30 muestras por filtro). HTTP medido sobre el cargador y manejador reales contra
el clon: no incluye TLS ni la frontera nominal. B94: EXPLAIN ANALYZE 2,64 ms.
