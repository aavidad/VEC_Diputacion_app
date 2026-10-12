# Preparación de expediente, alta en Personal y cese

Ensayo local del 12/10/2026 con Chrome del sistema, PostgreSQL 18.4 y datos inventados. La base procede de la copia de la principal usada en el corte de Personal46; se quitaron únicamente las seis filas privadas del antiguo fixture RPT para comprobar la preparación desde ausencia. Ningún servicio de cidonia se consultó o modificó. No se reaplicaron migraciones ni se ejecutó DOWN.

El servidor incorpora el parche CT115 de #1069 (`e36151198`). La web y el guion pertenecen a esta entrega, sobre main `46c9a52e`. La prueba focal del adaptador conserva `.415640Z`, fechas civiles y RC físico; el ensayo real preserva el JSON y la prueba históricos de v8.

## Resultado

| Paso | Comprobación |
| --- | --- |
| `--comprobar` inicial | Sólo consultas: informó RPT y expediente ausentes, cuatro certificados, ocho catálogos, flag y fecha de cese. No creó estado ni expediente. |
| `--aplicar PREPARA-ALTA-B2` | Un expediente nuevo `2026/94015`; petición y ratificación por actores distintos, alta CT, análisis, cobertura, asignación, informe, fiscalización con Intervención, selección, comunicación, aceptación, propuesta y resolución. Los 15 actos devolvieron 200/201; consulta del plan 200 con plaza `2026/0033` de Administrativo en la unidad 520. |
| Alta desde el formulario CT | Revisión y botón de registro: confirmación 200; recibo visible y botón para abrir la ficha de Personal. Una relación y una ocupación vigentes desde 01/12/2026 hasta 31/01/2027. |
| Cese desde la ficha | El intento inicial detectó la proyección física de CT115; el parche final recuperó **la misma clave** desde Chrome con 201. Fecha de efecto 20/12/2026; relación y ocupación con revisión 2 finalizada y revisión 1 conservada. |
| Retorno a Bolsa | Un contrato de cese y un recibo confirmado; disponible desde 20/05/2027 según la política v2 existente. Se conservó el primer resultado pendiente y el relevo lo recuperó sin duplicar. |
| Reinicio | Se detuvieron y recrearon sólo aplicación y PostgreSQL propios sobre el mismo PGDATA. Seguimiento, plan y ficha: 200/200/200. Replay del cese: 201 y mismo recibo/fecha. Las consultas de conservación antes, después y tras replay coinciden. |
| Repetición del guion | `PREPARADO_HISTORICO`, con el expediente y recibo conservados; no repitió actos ni SQL de datos. |

Expediente: `expediente:ct:1c882ea3ab2fc2a8ac67eadc4990c53514dabc65c056994b58ace1056b513515`.

Recibo de alta: `recibo:incorporacion-personal-b2:d5a14c96-87ca-4bbe-93c2-f2cef5c5d216`, 12/10/2026 20:38:05.674353 UTC. Empleado `emp_e7a841696e97462d9bb23b8faf758660`.

Recibo de cese: `recibo:ct-seguimiento:28f004fdef859e6b100aec28c459466f01492b5e2b7514459605bdfa17fd8251`, 12/10/2026 20:46:33.496332 UTC.

Recibo de Bolsa: `recibo:bolsa:cese:e407a93ed3029774ae0512416fa909c04296e26565c4e25057499e2778cb6b99`.

La versión 8 conserva antes y después SHA del agregado `c92cb38dd82f14c0201f2dafde3bc5e28722231f8f48b66b4cc5d86ad2fd5009` y de su prueba `90e734349243ce9193953c63469eef14acc9bc29aa19e8f37e3073e7c22d1ee6`. La prueba canónica sigue conteniendo el SHA del agregado. No se normalizó ni reescribió esa historia en el ensayo final.

## Instalación y uso

Orden: **#1069 → esta entrega**. No hay SQL de migración nuevo. La [lista ordenada](../../../deploy/principal/lista_sql_codexk_preparar_alta_b2_20261012.txt) identifica las tres preparaciones de datos y su punto del recorrido; sólo el guion las ejecuta, con parámetros del estado privado.

Como `openclaw`, desde un checkout que contenga ambas entregas:

```sh
bash scripts/operacion/preparar_expediente_alta_b2/preparar.sh --comprobar
bash scripts/operacion/preparar_expediente_alta_b2/preparar.sh --aplicar PREPARA-ALTA-B2
```

CONFIG NUEVA: `VEC_PERSONAL_B2_GOBIERNO_ENABLED=true`, ocho catálogos de régimen/modalidad publicados y `personal_b2.cese_fecha_efecto=primer_dia_sin_relacion` para este escenario. Se mantienen incorporación acreditada y retorno a Bolsa activados con el material de HZ21. Si falta algo, la comprobación imprime el comando concreto; no edita el lanzador ni la configuración por su cuenta. La duda 140 sigue fijando la decisión de RRHH para el uso real.

El guion reutiliza los contratos centro→RRHH, análisis→informe y propuesta/firma, los cuerpos acreditados de H y los fixtures B67/B68 y RPT de K. Deja la resolución preparada; **el alta y el cese los hace RRHH desde la pantalla**. Los recibos, cuerpos y claves de reintento quedan fuera de Git. Las capturas muestran [el alta](alta-confirmada.png), [el cese](cese-confirmado.png) y [la ficha finalizada](ficha-finalizada.png).

## Alcance y límites

Las firmas de remisión y el anclaje de Bolsa son preparaciones ficticias expresamente autorizadas; no verifican firma electrónica ni publicación institucional de la RPT. El archivo seleccionado para el cese aportó su huella; no acredita custodia documental. No se sustituyó ningún catálogo publicado ni se concedieron perfiles desde el guion.

La ficha tiene 13 consultas fijas; el GET del plan confirmado tiene 33 y respondió alrededor de 250–260 ms en este ensayo. El GET previo de opciones conserva la deuda de rendimiento comunicada a dirección. A 1440 y 390 px no hubo desbordamiento horizontal global. El clon tiene 503 heredados de preferencias de Usuarios y consulta de resolución posterior, y una advertencia del servicio de documentos; no se declara cero errores globales del portal. El alta, el cese y las tres lecturas de recuperación sí responden con los estados de la tabla.

Revisiones SQL y seguridad del guion: GO sobre los bytes conservados de `95acc84` y `90cc5f33`. Revisión sensible del cese: GO exacto `e36151198`; usabilidad focal de las tres etiquetas de ocupación: GO. Semgrep local, sintaxis y pruebas focales verdes; la puerta completa se ejecuta una vez en CI por PR.
