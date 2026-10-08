# Bolsa de trabajo

**Estado a 8 de octubre de 2026:** Bolsa funciona en cidonia con HZ8 instalado (canales del llamamiento y seguimiento por teléfono, reglas de RRHH v5, «Mi Bolsa» del aspirante y documentos encendidos); faltan la vuelta a la bolsa tras un cese, la carga del Excel de CONVOCA desde una pantalla y el correo de la Diputación.

Base comprobada: `origin/main@033fda6ed` más las PR fusionadas el 08/10 (#895 a #904). Los criterios para dar Bolsa por cerrada están en [OBJETIVOS.md](OBJETIVOS.md). Las preguntas a RRHH se citan por su número en [`dudas.md`](../../dudas.md).

## Ceses sin candidato — revisión de #774, 8 de octubre de 2026

B81 permite continuar el relevo cuando el llamamiento del puente CT eligió
una participación que no pertenece a ninguna bolsa constituida. Conserva el
cese y su proyección con la referencia del evento y de la auditoría común CT115;
el registro local identifica al proceso de entrega. Si la bolsa ya está
constituida y falta el vínculo, el cese sigue pendiente y el relevo se detiene.
Si el vínculo aparece después, B90 mantiene al candidato pendiente desde el
instante real de recepción B13. El mismo relevo aplica B45; hasta entonces no
ocupa turno ni resulta elegible. El replay B81 conserva el registro original.

La rama se actualiza con main y el cursor se toma de la definición instalada
en postHX + HZ + B85 + B86 + CT193 + B87. CT197 añade un testigo técnico que
verifica el cese publicado con CT129 y entrega a Bolsa su `auditoria_ref` exacta.
La función no concede lectura de la tabla CT a Bolsa ni modifica CT129.
CT197, B81 y B90 están reservadas para #774; la lista las instala en ese orden.
B90 calcula el pendiente desde B13 incluso si B81 aún no registró la
proyección. Aplica el corte temporal de cada lectura: un cese futuro no cambia
el orden anterior y una resolución B45 posterior no borra el intervalo
pendiente. La misma migración actualiza el orden y el resumen que consultan
las pantallas y la selección, y ofrece una lectura por lote.
Claude revisa las SQL exactas y el ensayo final antes de integrar. Ninguna
está instalada en una base compartida. CONFIG NUEVA: ninguna.

En un clon PostgreSQL 18.4 postHX + HZ + B85 + B86 + CT193 + B87, las tres
SQL se instalaron una vez (104–106 ms cada una). Las pruebas B81/B90 pasaron
en ROLLBACK (205/206 ms): alta y replay únicos, vínculo posterior, pendiente
sin fecha inventada, restricción B45, permisos e historia. CT129 real verifica
el origen; la fixture prepara CT/B13 directamente y no acredita un cese desde
RRHH ni navegador. En 10.000 filas, el índice por participación y el cursor
tardaron 0,019 y 0,014 ms. En diez lecturas SQL con 2.390 estados variados,
el lote tuvo p95 de 94,764 ms; con 10.000 estados mezclados tardó 546,4 ms.
La base observada tiene 41 vínculos. Estas cifras no miden HTTP ni acreditan
pantalla <300 ms: la fuente RRHH aún carga todas las entradas antes de paginar.
Faltan la medición HTTP y la revisión final del corte.

## Lo que falta

| # | Qué | Quién o qué lo frena | Cómo se comprueba |
| --- | --- | --- | --- |
| 1 | **Vuelta a la bolsa tras el cese de CT.** En cidonia falta `VEC_BOLSA_CESE_CT_ENABLED` y el LOGIN del relevo. El LOGIN y el entorno están preparados en #771. | Claude, en el despliegue de cidonia. | Cese en CT y la persona vuelve a «Disponible» (o «disponible desde») en su bolsa, también tras reiniciar. |
| 2 | **Cese de una bolsa no constituida sin bloquear el relevo** (CT197, B81, B90). | Codex-T, PR #774 en borrador; CI y medición HTTP pendientes. | Revisión SQL completada, ensayo PG18 verde; falta CI de la cabeza final y medir el recorrido HTTP nominal. |
| 3 | **Carga de bolsas desde el Excel de CONVOCA en pantalla (B1).** Cadena #751 (núcleo, verde) → #752 (HTTP, en conflicto) → #759 (pantalla). Faltan rutas, concesión al rol de RRHH, plantilla publicada por Administración, LOGIN propio de constitución (B80) y AD218 en cidonia. | Claude (ramas `claude-b1-*`). Dudas 144 y 145. La rama `codexu-b1-nucleo-20261008` repite #751: no encargarla otra vez. | RRHH sube un Excel sintético, ve los errores por fila y confirma; la bolsa aparece con sus posiciones. |
| 4 | **Recorrido de la oferta con varias plazas y del llamamiento directo**, de punta a punta. El código está; falta el acta. Va dentro del recorrido completo de [OBJETIVOS.md](OBJETIVOS.md). | Claude, con el recorrido de CT. | Acta con varias aceptaciones, adjudicación por posición, «sin respuesta» sin consecuencia y llamamiento directo. |
| 5 | **Reglas que ya no encajan con el Reglamento (duda 147).** El Reglamento (BOP 16/01/2026, arts. 8 y 11) no recoge el plazo de 24 horas por plaza ni «no responder equivale a no aceptar» en la oferta. Basta con cambiar el catálogo. | Dirección decide; luego, cambio de catálogo. | Catálogo nuevo cargado y la pregunta 147 retirada de `dudas.md`. |
| 6 | **Renuncia justificada con el fichero.** Hoy «Mi Bolsa» envía la huella y la referencia del justificante, no el fichero. Con documentos ya encendidos, hay que comprobar si la custodia lo recoge. | Sin dueño. Comprobar antes de encargar. | El justificante se descarga desde la ficha de RRHH. |
| 7 | **Cifras globales de Bolsa pulsables.** Las cifras por bolsa ya llevan a su lista (#883). Los tres totales globales y «llamamientos en curso» necesitan una lectura global paginada con auditoría en la misma transacción. Hoy cuentan participaciones, no personas. | Equipo V. No hay rama. | Cada total abre su lista filtrada con el mismo número. |
| 8 | **Correo de la Diputación configurable desde Administración (B5).** No existe: el servidor de correo solo se configura con variables de entorno. Para la presentación vale el buzón de pruebas actual. | Decidir antes qué se rescata de #179, #204, #206 y #209. Datos del servidor: Informática (duda 88). | Prueba de envío desde Administración, con auditoría. |

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
