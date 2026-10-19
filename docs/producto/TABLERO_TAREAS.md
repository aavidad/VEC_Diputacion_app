# Tablero de tareas de VEC

Actualizado el 19 de octubre de 2026 sobre `origin/main@d57f97dc7`, después de
la presentación.

Este tablero dice qué falta, quién lo hace y en qué estado está. Sustituye
la lista de tareas que había en `PLAN_IMPLEMENTACION.md` §6; los criterios de
cierre de cada fase siguen en `PLAN_IMPLEMENTACION.md` §5. Parte del borrador
del 15/10 corregido con el consenso de dirección de ese mismo día. Todo se
trabaja en local y los datos son siempre inventados. El 18/10 se subió a
cidonia la versión de la presentación; desde entonces vuelve a estar
congelada y el avance se mide en la copia local (F7-15).

## 1. Cómo se usa

### Quién lo cambia

Solo dirección actualiza este tablero. Los equipos citan la tarea en la PR y
en el canal con la forma `Tarea: F1-21`.

### Identificadores

No se renumeran nunca. Una tarea que se abandona o se
funde con otra queda como «Cancelada», con la fecha y el motivo en la propia
fila. Las tareas nuevas toman el siguiente número libre de su fase. Un número
ya usado no se da a otra tarea. Si ocurre, como con F0-29 y F0-31 el 18/10,
el uso nuevo se queda el número, la tarea antigua pasa a un número libre y
las dos filas lo dicen.

### Estados

Solo hay ocho:

| Estado | Significa |
| --- | --- |
| Pendiente | Se puede hacer; nadie la ha empezado o falta una dependencia |
| Bloqueada | Espera una respuesta o una decisión de fuera; la columna «Bloqueo» dice cuál |
| En curso | Hay equipo trabajando, con rama o PR |
| En revisión | La PR está lista y espera revisión |
| Fusionada | Está en `main`, pero aún no se ha recorrido en una copia local con reinicio |
| Hecha | Fusionada y recorrida en una copia local con reinicio (un documento basta con fusionarlo) |
| Aparcada | No toca ahora por prioridad |
| Cancelada | No se hará; la fila dice desde cuándo y por qué |

### Tamaños y fechas

S, hasta medio día; M, uno o dos días; L, de tres a cinco días;
XL, más de cinco (se trocea antes de encargarla). Para estimar se cuenta S
como 0,5 jornadas, M como 1,5 y L como 4.

La columna «Desde» es la fecha en que la tarea entró en su estado actual. Las
que ya existían antes del tablero llevan 15/10 salvo que su PR sea anterior.

### Atascos

Dirección revisa cada mañana estas señales y actúa:

| Señal | Qué se hace |
| --- | --- |
| En curso más del doble de su tamaño (S más de un día, M más de cuatro, L más de diez) | Se pregunta al equipo qué la frena y se trocea o se reasigna |
| En revisión más de un día | Se busca revisor ese mismo día |
| Fusionada más de tres días sin recorrido | Se encarga el recorrido en copia local |
| Bloqueada más de cinco días | Pasa a la lista semanal para Alberto |
| Pendiente con las dependencias hechas y un equipo libre más de un día | Se encarga ya |

### Parte diario

Cada equipo deja en el canal, al cerrar el día, cinco líneas:

```text
Equipo y fecha:
Tarea: F?-?? (estado al cerrar)
Hecho hoy:
Me frena:
Mañana:
```

### Comprobación del recuento

Antes de cada cambio del tablero se ejecuta
`awk -F'|' -f scripts/recuento_tablero.awk docs/producto/TABLERO_TAREAS.md`.
Comprueba que cada estado es uno de los ocho, que ningún identificador se
repite y saca el recuento y las tareas empezables. El resumen del apartado 2
se copia de su salida.

## 2. Resumen

| Fase | Pendiente | Bloqueada | En curso | En revisión | Fusionada | Hecha | Aparcada | Cancelada | Total |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 0 Base común | 14 | 4 | 6 | 1 | 8 | 0 | 3 | 1 | 37 |
| 1 Bolsa y CT | 27 | 5 | 6 | 0 | 28 | 5 | 7 | 3 | 81 |
| 2 Personal y Organización | 10 | 3 | 1 | 0 | 0 | 0 | 0 | 1 | 15 |
| 3 Cronos | 5 | 4 | 0 | 0 | 0 | 0 | 1 | 0 | 10 |
| 4 Dietas | 2 | 4 | 0 | 0 | 0 | 0 | 1 | 0 | 7 |
| 5 Sustituir Convoca | 8 | 5 | 0 | 0 | 0 | 0 | 0 | 0 | 13 |
| 6 Resto de módulos | 0 | 0 | 0 | 0 | 0 | 0 | 13 | 0 | 13 |
| 7 Uso real y producción | 18 | 14 | 0 | 0 | 0 | 0 | 1 | 0 | 33 |
| **Total** | **84** | **39** | **13** | **1** | **36** | **5** | **26** | **5** | **209** |

F0-24A y F1-65A llevan letra y el recuento no las cuenta.

Empezables ya: 20 en las fases 0 y 1 (pendientes con todas sus
dependencias hechas o fusionadas). De ellas, 13 tienen equipo:

- H: F0-14, F0-15, F0-22, F0-23, F0-24 y F1-58.
- O: F0-21 y F1-46.
- P: F1-23, desde el 20/10.
- R: F1-67 (AD288 ya está en `main` con F1-71).
- T: F1-39 y F1-82.
- L: F0-37 (la semilla de la copia va dentro de F0-28).

Sin equipo todavía: F0-11, F0-16, F0-17, F1-53, F1-56 (opcional), F1-60 y
F1-62. Fuera de las fases 0 y 1 salen seis más (F2-01, F2-02, F3-02, F7-13,
F7-23 y F7-30), pero la prioridad vigente no deja encargarlas antes de cerrar
la fase 1.

### Cambios del 18 y el 19/10

- Fusionadas el 18/10: F1-04 (#1155), F1-12 (#1187 y #1211, que trae
  AD294), F1-16 (#1146), F1-18 (#1180 y #1184), la parte C de F1-36 (#1179),
  F1-42 (#1231), F1-55 (#1207), F1-69 (#1210), F1-71 (#1164 y #1176, con
  AD288), las nuevas F1-73 a F1-76, F1-78 y F1-79, y F0-33 y F0-34. F0-03
  (#1151) y F1-44 (#1159) estaban fusionadas desde el 15/10 sin marcar. F1-81
  (#1242) se fusionó el 19/10. F1-80, el guion de la presentación, se entregó
  fuera de Git.
- Cerradas sin más trabajo: F1-06, porque la cubre la pantalla de F1-04, y
  F1-64, que se queda con la parte 1. La rotación de la raíz y la clave de CT
  pasa a la preparación de producción (F7-21).
- Aparcadas: F1-17 (sin consumidor; la pregunta 167 a RRHH va en #1197);
  F1-63 (semilla de firmas de la copia con H como responsable, que se retira
  con F2-12 y F2-14); F1-70 (hasta preparar producción, dentro de F7-21);
  F1-72 y F0-36 (hasta después del recorrido 4.0); F1-83; y F3-01 y F4-01 con
  Cronos y Dietas.
- Cambios de alcance: F1-55 reformulada; F1-13 sin permiso propio por tipo
  de oferta y en tres PR encadenadas; F1-19 con un expediente hijo por persona;
  F0-04 empieza por un lote nuevo con las dimensiones de cada rol; F0-18 ya no
  retira la rama `documentos_bolsa_externa`, que sí se usa; F0-20 pasa a
  Bloqueada por las dudas 112 a 115.
- Reparto: F1-11 pasa de R a O y F1-19 pasa a B.
- Identificadores reutilizados: el 18/10 se abrieron PR con F0-29 (#1214,
  AD295 de R) y F0-31 (#1241, versión única de la web) cuando esos números ya
  tenían tarea. Se quedan con esos usos. La CI de migraciones que era F0-29
  pasa a F0-33, y los roles de B119 que eran F0-31 pasan a F0-34; sus PR
  conservan el número antiguo en el título. Se añade F0-24A (#1185). F1-77 no
  se llegó a usar: no lo cita ninguna PR ni ninguna nota.
- Dependencias corregidas: F0-12, F4-02 y F5-05 dependían de F0-10, que
  está cancelada, y ahora dependen de F1-07, que la cubre. F1-31 dependía de
  F1-06 y ahora depende de F1-04. F1-48, F1-49 y F1-50 citaban las dudas 166 a
  170, que tratan de otros asuntos o no existen: quedan con «duda por
  numerar». F1-23 ya no figura como bloqueada: que P empiece el 20/10 no es un
  bloqueo de fuera.
- Tareas nuevas: F7-17 a F7-33 para preparar producción, cada una con la
  decisión escrita de la que sale; F0-35 a F0-37, tres excepciones temporales
  con responsable y condición de retirada; F1-82, el manual de Intervención
  (T); y F0-32, esta actualización.

### Cambios anteriores

Cambios del 16/10 por la tarde: fusionadas F1-08 (#1144, #1170 y #1178),
F1-21 (#1153), F1-03 (#1152) y F1-22 (#1150), que estaban sin marcar, F1-45 (#1183, solo documentación; la valoración espera al
baremador) y la corrección de la auditoría de la lectura de inscripción del
aspirante externo (#1188), que se anota como F1-68. F1-36 tiene fusionadas
las partes A (#1172) y B (#1173) y le falta C (#1179); F1-64 tiene fusionada
la parte 1 (#1181) y le falta la 2; F0-28 tiene fusionado su primer corte
(#1175). Pasan a en curso F1-09 (#1190), F1-12 (#1187), F1-18 (#1180 y
#1184), F1-34 (#1189; faltan los candidatos y la ficha del candidato) y
F1-40 (#1177), que #1188 desbloqueó. La confirmación de aceptaciones
directas desde la ficha del aspirante (#1164 y #1176), que sus PR llamaban
F1-06, pasa a ser F1-71; F1-06 sigue siendo la pantalla de reglas de
situación. Tareas nuevas pequeñas: F1-69
y F1-70 para R (menores de CT231 y CT230) y F0-31 para L (roles de B119 en
el guion de la copia local).

Cambios del 16/10, mañana: fusionada #1147 (F1-07) y #1182 (F1-65A);
F1-59 cancelada (en ofertas publicadas solo existe Aceptar y la adjudicación
va por orden); F1-66 aparcada (plantillas para varias organizaciones, sin
consumidor); F1-67 nueva, para R, con la preparación de bases; F1-17 pasa a
S tras F1-18; F0-08 queda bloqueada hasta que R publique un contrato nominal
segregado; F1-63 sube de prioridad; F1-46 podría cerrarse por #1147 y
dirección lo comprueba; F1-64 deja la rotación de la raíz y la clave de CT
pendiente de consulta; F0-18, F0-21 y F0-29 amplían su alcance. #1144 tiene
NO-GO y le siguen #1170 y #1178.

Cambios del 16/10 por la madrugada: fusionadas F0-27 (#1160), F1-01 (#1148)
y F1-37 (#1163); F0-30 sigue en curso con L (T paró por cuota) y lleva
fusionadas #1166, #1167 y #1168; F1-43 aparcada, porque ampliar el tipo V1
va contra las reglas de simplificación y espera a las lecturas de la fase 2;
nuevas F1-62 a F1-65 (productor de catálogos CC1, organización legada de las
firmas y menores de vigencias y de plantillas para H); el rol externo del
aspirante entra en el alcance de F2-12. Cambios anteriores del 16/10: fusionadas #1162 (F0-01, B), #1158 (F0-02, H) y #1157
(F1-54, T); F0-25 y F0-26 fusionadas antes (#1156 y #1161); F0-27 en curso
(#1160) y F0-09 en curso con B; F0-06 aparcada; F0-10 cancelada (la cubre
F1-07); F1-40 desbloqueada con el corte acordado y su cierre pendiente de la
nueva F1-61; nueva F1-60 (ámbito gobernado de la bolsa); F0-30 en curso con T y L; nuevo equipo
L con F0-28, F0-29 y F0-30; nuevas F1-58 y F1-59, y las cuatro tareas de
simplificación que abren la fase 2 (F2-12 a F2-15).

Hecho el 15/10: #1145 (ofertas con plazo desde la publicación), #1137 (cinco
firmas reales y PDF desde la ficha tras reiniciar) y #1143 (manual de la
jefatura de RRHH). Cancelado el 15/10: F1-05, fundida en F1-04, y F2-10,
fundida en F0-18.

## 3. Camino crítico de la fase 1 y estimación

Estimación de dirección del 18/10: la fase 1 se cierra en la segunda quincena
de noviembre. Sustituye a la de finales de octubre que daba este tablero el
16/10 y al cálculo por jornadas del 15/10. Se recalculará con el ritmo real
cuando los equipos recuperen la cuota.

El recorrido principal empieza en la petición del centro (`FLUJO_APP.md`,
apartado 4.0). El tramo del aspirante (inscripción, méritos y admisión, F1-29)
es una demostración aparte: el 14/10 Alberto aclaró que el recorrido puede
empezar donde convenga.

Plan del 18/10, tras el estudio de situación de dirección:

1. El núcleo de autorización sigue la cadena del apartado siguiente hasta
   AD296; después van AD291 y F0-18. AD291 no espera a capacidades sin
   cerrar: lo que falte pasa a la semana siguiente.
2. L deja en Git, dentro de F0-28, un preparador único de la copia: semillas,
   actos de Administración y portal externo del aspirante.
3. El equipo P recorre el caso 4.0 desde el 20/10 (F1-23 → F1-24 → F1-25 →
   F1-27 → F1-28 → F1-33) y lo cierra con F1-31.
4. AD292, F1-72 y el paso de CT a organizaciones gobernadas (F0-36) esperan a
   que termine el recorrido 4.0. F2-12 también empieza después.
5. Una sola revisión, salvo en SQL, permisos, identidad y cifrado, que llevan
   dos.

Cadenas que pueden volverse críticas: F1-09 → F1-10 → F1-31 (comprobación del
art. 9, apartado cuarto); F1-16 → F1-11 → F1-13 → F1-14 → F1-31 (modalidades
en «Mi Bolsa»); y F0-18 → F1-19 → F1-20 → F1-31 (varias plazas hasta la
propuesta).

### Después de la fase 1: simplificar primero

La fase 2 empieza por las cuatro tareas de simplificación, F2-12 a F2-15
(permisos por datos, lecturas sin autorización de un solo uso, base SQL
consolidada y arranque que solo compone). Se hacen antes que el resto de
Personal, Cronos y Dietas. Cronos y Dietas siguen aparcados desde el 08/10:
F3-01 y F4-01 conservan sus PR y se retoman después de esas cuatro tareas.

### Orden de fusión de la cadena central de SQL

El orden es fijo; ninguna PR de esta cadena se fusiona antes que la anterior:

1. #1147 (AD277, AD281, AD284, AD287), F1-07, fusionada el 16/10
2. #1144 (AD276), F1-08, fusionada el 16/10
3. #1170 (CT230), fusionada el 16/10
4. #1178 (CT231, fila O6 de la ficha), fusionada el 16/10
5. #1146 (AD278), F1-16, fusionada el 18/10
6. #1155 (AD283), F1-04, fusionada el 18/10
7. #1187 (AD279), F1-12, fusionada el 18/10
8. #1164 y #1176 (AD288), F1-71, fusionadas el 18/10
9. #1211 (AD294), F1-12, fusionada el 18/10
10. F1-09: CT233, AD290, B133, B134, B137 y AD293 (#1190, #1201, #1203 y
    #1196)
11. #1214 (AD295), F0-29
12. #1220 (AD296), F1-13
13. #1224 (AD291) y AD273, F0-18

Detrás de F0-18 va AD292 (#1225), el consumidor de autorización de F1-19.
#1151 (AD282, F0-03) ya está fusionada y no entra en la lista.

Regla desde el 18/10 (#1195): toda PR con migraciones añade sus líneas a
`deploy/principal/lista_sql_main_tras_hz37.txt`, la lista única de `main` que
usa la copia local.

## 4. Reparto por equipo

Cada equipo trabaja su lista en este orden. Una flecha indica que la
siguiente no empieza hasta acabar la anterior.

| Equipo | Zona | Orden |
| --- | --- | --- |
| R | Criterios de Bolsa y puente con CT; dueño de las migraciones nuevas del núcleo de autorización | F1-09 → F0-29 (AD295) → AD296 de F1-13 → AD291 con F0-18 → F1-10 → AD292 de F1-19 → F1-67. F1-69 fusionada; F1-70 aparcada |
| S | Expediente y oferta de CT | F1-16, F1-04, F1-18, F1-71 y F1-55 fusionadas; F1-79 fusionada y F1-80 hecha. Revisa F1-73 a F1-75. Cuando se retome, el lado de CT de F1-72. F1-17 aparcada |
| O | «Mi Bolsa», Documentos y preferencias | F1-11 → F1-13 (tres PR) → F1-14 → F0-21 → F1-46 → F1-29. Cuando se retome, la fuente de F1-72. F1-73 fusionada |
| H | Administración, identidad y firma | F0-24A → F0-23 y F0-24 → F0-04 (empieza por el lote del 18/10) con F0-05 → F0-14, F0-15 y F0-22 → F1-58. Semilla de firmas de la copia (F1-63, #1198). F0-08 bloqueada |
| T | Pantallas y textos | F0-31 (#1241) → F0-07 → F1-82 → F1-39. F1-47 espera a F0-36 y F1-57 a F1-09. F1-75 y F1-42 fusionadas |
| P | Recorrido, desde el 20/10 | F1-23 → F1-24 a F1-30 → F1-32 y F1-33 → F1-34 → F1-31 |
| C y D | Cronos y Dietas | Aparcados con sus módulos. F3-01 y F4-01 conservan sus PR; AD291 y F0-18 los hace R |
| B | Arranque y configuración | F0-09 → F1-40 → F1-19. F1-74 fusionada |
| L | Limpieza y herramientas comunes | F0-28 (portal externo en un paso, preparador único y semilla de F0-37) → F0-30. F0-33 y F0-34 fusionadas |
| Dirección | Documentación y coordinación | F0-32 → F0-35 → F6-13. F1-76, F1-78 y F1-81 fusionadas |


F0-06 está aparcada desde el 16/10: el recorrido asigna por organización y
unidad gestora con la vía ordinaria (AUT44). Nadie asigna por SQL perfiles con
ámbitos de centro y categoría.

## 5. Fase 0. Base común

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F0-01 | Encender la auditoría de RRHH (`VEC_RRHH_AUDITORIA_ENABLED`) en la copia local, con su configuración | Configuración; `deploy/principal/auditoria_rrhh_definicion_rol.json` | - | - | S | B | #1162 | Fusionada | 16/10 | La consulta por persona devuelve sus lecturas y sus actos | PLAN §5, fase 0 |
| F0-02 | Retirar el compositor heredado de `vec-admin` o hacer que falle cerrado: exige el grupo `vec_admin_perfiles_auditoria_ejecutor` (`composicion.go:91`) y la función `registrar_denegacion_frontera_admin_v1`, que no existe. No se crea el grupo | `cmd/vec-admin/composicion.go` | - | - | M | H | #1158 | Fusionada | 16/10 | `vec-admin` arranca en una copia limpia o se niega con un motivo claro; nada llama a la función inexistente; dos revisiones sensibles | Consenso de dirección 15/10 |
| F0-03 | Vida de claves y confianza de gobierno configurable: 24 h por defecto y en producción, hasta 90 días en desarrollo; el arranque de CT adopta la vigente sin recortarla (AD282) | Administración; `internal/app/bootstrap` | - | - | M | H | #1151 | Fusionada | 15/10 | Arranque con y sin el valor; producción rechaza más de 24 h; desarrollo acepta hasta 90 días; el reinicio conserva la confianza | Consenso de dirección 15/10 |
| F0-04 | Pantalla de `vec-admin` para asignar y retirar un perfil desde la ficha del usuario (A8). Se hace junto con F0-05. Desde el 18/10 empieza por un lote nuevo: el lote ordinario de Administración lleva las dimensiones de cada rol (unidad y ámbito para Bolsa), con una migración de autorización propia fuera del núcleo. Ese lote va antes de F0-07 y lo necesitan el perfil de Carmen y F0-05 | `cmd/vec-admin`; `web/static/administracion-perfiles`; SQL de autorización | F0-23, F0-24, F0-24A | - | L | H | rama del candidato #733 | Pendiente | 18/10 | Formularios, confirmación y recibo en Chrome a 1440 y 390 px; el lote asigna un perfil de Bolsa con su unidad y su ámbito sin SQL a mano | PLAN §6 anterior al tablero; decisión de dirección 18/10 10:10 |
| F0-05 | Recorrido real de asignar y retirar con dos administradores (A9). Su tamaño va dentro de F0-04 | `cmd/vec-admin` | F0-02, F0-04 | - | - | H | - | Pendiente | 15/10 | Asignar, retirar, reintento, conflicto y denegación auditados; mismo recibo tras reiniciar | PLAN §5, fase 0 |
| F0-06 | Vía gobernada para asignar los perfiles de ofertas de CT con ámbitos de centro y categoría. Aparcada el 16/10: #1146 (F1-16) pasa a organización y unidad gestora por la vía ordinaria (AUT44), así que el recorrido no necesita ámbitos de centro y categoría | `internal/modules/administracion` | F0-04 | - | M | H | - | Aparcada | 16/10 | Las tres asignaciones de #1146 se crean sin SQL a mano | Consenso de dirección 15/10 |
| F0-07 | Administración muestra nombres de persona y de unidad con lectura autorizada (hoy sale «Nombre de la unidad no consultado»; el vídeo 16 se retiró por eso) | `web/static/admin`; `internal/modules/administracion` | F0-04 (su lote del 18/10) | - | M | T | - | Pendiente | 18/10 | Lista y ficha con nombres; cada lectura auditada | Borrador 15/10; reparto de dirección 18/10 |
| F0-08 | Quitar del arranque las escrituras directas en tablas de autorización y la herencia de `vec_ad3_o207_gobierno` | `internal/app/bootstrap` | - | Espera un contrato de publicación nominal segregada del gobierno interno, trabajo de núcleo de R | M | H | - | Bloqueada | 16/10 | Arranque en una copia sin esas escrituras | PLAN §6 anterior al tablero |
| F0-09 | Acto de catálogo con las fuentes de política de cese, configuración, origen CT y contacto completo; versión del rol de RRHH con el especializador (#1100). Antes, comprobar si sigue haciendo falta | Catálogo de acciones; SQL de autorización | - | - | S | B | - | En curso | 16/10 | El rol publicado incluye esas acciones y el recorrido no da 403 | PLAN §6 anterior al tablero |
| F0-10 | Antivirus (ClamAV) y análisis de PDF con cuarentena antes de custodiar. Cancelada el 16/10: la cubre F1-07 (#1147), que ya acredita ClamAV real, analizador con TLS mutuo y cuarentena con admisión 201 | `cmd/vec-analizador`; `deploy/analizador` | - | - | M | O | #1147 | Cancelada | 16/10 | Ensayo con un clamd real; ningún fichero dudoso llega a custodia | PLAN §6 anterior al tablero |
| F0-11 | Documentos y S3 (D16 a D18): autorización nominal del aspirante externo y composición | Documentos; `deploy/almacen_s3_ceph` | - | - | M | Sin asignar | rama D16-D18 | Pendiente | 15/10 | El aspirante externo sube y descarga con su permiso | Borrador 15/10 |
| F0-12 | Cliente del gestor documental Mayan | Documentos | F1-07 (antes F0-10, cancelada porque la cubre F1-07) | Sistemas: TLS y configuración | M | Sin asignar | rama Mayan | Bloqueada | 15/10 | Documento custodiado en Mayan con el TLS del destino | Borrador 15/10 |
| F0-13 | Correo de la Diputación configurable desde Administración (B5) | `internal/modules/administracion` | - | Duda 88 (Informática) | M | Sin asignar | - | Bloqueada | 15/10 | Envío de prueba auditado desde Administración | PLAN §6 anterior al tablero |
| F0-14 | Auditoría, corte A: denegaciones y errores de las lecturas de contactos de Bolsa | `internal/modules/bolsa`; auditoría común | - | - | S | H | - | Pendiente | 15/10 | Una lectura denegada deja apunte en la cadena común | Borrador 15/10 |
| F0-15 | Auditoría, corte B: la descarga de un borrador de CT registra documento, formato y fallo | `internal/modules/contrataciontemporal`; auditoría común | - | - | M | H | - | Pendiente | 15/10 | La auditoría dice qué documento se bajó y en qué formato | Borrador 15/10 |
| F0-16 | Interruptor de módulos en Administración, persistente y auditado | `internal/modules/administracion`; `web/static/admin` | - | - | L | Sin asignar | - | Pendiente | 15/10 | Apagar un módulo cierra su menú y sus rutas | PRD §8 |
| F0-17 | Selector de perfil activo en el portal interno | Identidad; `cmd/vec-interno` | - | - | M | Sin asignar | - | Pendiente | 15/10 | Una persona con dos perfiles elige uno sin cambiar de usuario | PRD §8 |
| F0-18 | Consolidación del núcleo de autorización, la última de la cadena SQL: AD291 (#1224) y AD273. Consume AD277 y AD279, añade el original de Cronos (D21), la autoridad documental de Dietas y la lectura propia de Personal para Cronos y Dietas (incluye lo que era F2-10, aprobado el 15/10) y lleva la pieza del núcleo de F1-09 (audiencia y fila de origen). Además: subir a garantía alta y método certificado las ramas de RRHH de AD277:80 y AD281:2744, y no recrear los objetos de los comprobantes de Méritos que ya crean AD277, AUT95, D20 y M7. La rama `documentos_bolsa_externa` del consumidor central se conserva: sí se alcanza (AD277 → D20, justificante externo); hasta el 18/10 este tablero pedía retirarla por error. AD291 no espera capacidades sin cerrar: lo que falte pasa a la semana siguiente | SQL del núcleo de autorización | F1-07, F1-08, F1-12, F1-16, F1-09, F0-29, F1-13 | - | L | R (antes C y D) | #1224 (AD291); #1142 | En curso | 18/10 | Ensayo en copia y dos revisiones sensibles; Dietas lee lo suyo sin la consulta de RRHH | Consenso de dirección 15/10; decisiones de dirección 18/10 11:37 y 11:50 |
| F0-19 | Exportación judicial de la auditoría (rol de Aplicación v7, fachada de captura, emisor), firmada y con sellado de tiempo de una autoridad de sellado (TSA). Se retoma al preparar producción, después de la fase 1, y tiene que estar terminada antes de tratar datos reales | Auditoría; `cmd/vec-auditoria-verificar` | - | - | L | Sin asignar | rama `codexl-auditoria-nominal` | Aparcada | 19/10 | Paquete firmado y sellado verificable y exportación auditada | Borrador 15/10; TRD §7 |
| F0-20 | Copias de seguridad y restauración desde Administración (14 a 25 jornadas) | `cmd/vec-admin-copias` y `cmd/vec-copias-*` | - | Dudas 112-115 (Sistemas, Seguridad y RRHH) | XL | Sin asignar | - | Bloqueada | 19/10 | Restauración con doble control en una base desechable | PRD §8 |
| F0-21 | Productor gobernado de admisiones de identidad externa: función SECURITY DEFINER solo para el grupo ADMIN, que exige una aprobación existente, audita en la misma transacción y permite la retirada | Identidad; SQL de `internal/modules/aspirantes` | - | - | M | O | - | Pendiente | 15/10 | Admisión con aprobación previa da recibo; sin aprobación se niega; retirada auditada; ensayo y revisión SQL independiente. Duda del contrato: qué aprobación de ADMIN aplica (`aprobacion_alta` solo cubre cuentas) y retirada prospectiva con recibo; O propone reutilizar el acto y la pantalla de Administración. Consultar a Fable y Astra antes | Consenso de dirección 15/10 |
| F0-22 | Auditoría, corte C: la consulta de datos de contacto deja apunte | `internal/modules/bolsa`; auditoría común | - | - | S | H | - | Pendiente | 15/10 | Cada consulta de contacto registra quién, de qué persona y con qué motivo | Consenso de dirección 15/10 |
| F0-23 | Lecturas A6 de Administración, que necesita la pantalla de F0-04 | `cmd/vec-admin`; `internal/modules/administracion` | - | - | M | H | - | Pendiente | 15/10 | La ficha del usuario lee sus perfiles con lectura autorizada y auditada | Consenso de dirección 15/10 |
| F0-24 | Deuda P2 de la revisión de A7, antes de F0-04 | `cmd/vec-admin` | - | - | S | H | - | Pendiente | 15/10 | Pruebas focales de cada punto P2 | Consenso de dirección 15/10 |
| F0-24A | Parte de F0-24: la preparación del lote de perfiles rechaza una unidad que no está en su fuente antes de emitir una autorización o abrir la transacción | `cmd/vec-admin`; `internal/modules/administracion` | - | - | S | H | #1185 (borrador) | En curso | 16/10 | El rechazo conserva el registro del intento y no abre transacción | Revisión de A7 |
| F0-25 | `deploy/principal/LEEME_copia_local.md`: receta de la copia local (material ADMIN, datos de prueba, admisiones, OH y unidad del puesto) | `deploy/principal` | - | - | S | Dirección | #1156 y #1161 | Fusionada | 16/10 | Otro equipo levanta una copia local siguiendo solo esa receta | Consenso de dirección 15/10 |
| F0-26 | Limpieza documental: manual de administración y manifiesto de Bolsa al día | `docs/`; manifiesto del módulo Bolsa | - | - | S | Dirección | #1156 | Fusionada | 16/10 | Ningún paso del manual ni del manifiesto contradice `main` | Consenso de dirección 15/10 |
| F0-27 | Arranque sin Contratación temporal configurada: la app arranca y la función falla con error catalogado | `internal/app/bootstrap` | - | - | S | B | #1160 | Fusionada | 16/10 | Arranque con y sin la configuración de CT; sin ella, el resto funciona y CT responde con el error catalogado | Consenso de dirección 16/10 |
| F0-28 | Copia local en menos de 15 minutos con un guion único, sin pasos a mano; sustituye las recetas de cada equipo. Lo más urgente para cerrar la fase 1: que el guion componga en un paso el portal externo del aspirante (dos administradores, provisión del candidato externo, publicación de audiencias, admisiones y rol externo de semilla); sin eso no se pueden recorrer F1-29, F1-40, F1-41 ni F1-46 | `deploy/principal`; `scripts/` | F0-25 | - | M | L | #1175, #1192, #1195, #1202, #1205, #1213, #1219, #1221 y #1227 fusionadas; #1235 y #1236 abiertas | En curso | 16/10 | Otro equipo levanta la copia desde cero con el guion en menos de 15 minutos, también el portal externo | Consenso de dirección con Fable y Astra, 16/10; nota de dirección del 16/10 por la tarde |
| F0-29 | Lectura de la cadena de gobierno de Administración con registros intercalados: AD235 (`cadena_gobierno_admin_lectura_v1`) recorta el tramo si detrás hay registros de otro tipo y la verificación de Administración falla. Migración prospectiva AD295. Este identificador se reutilizó el 18/10: la CI de migraciones que lo llevaba antes es ahora F0-33 | SQL de autorización; `cmd/vec-admin` | - | - | M | R | #1214 | En curso | 18/10 | La verificación encuentra el acuse aunque haya registros posteriores; sin referencia la consulta no cambia; ensayo y revisión SQL independiente | Decisión de dirección 18/10 10:47 |
| F0-30 | Retirada continua de código muerto, empezando por `accionesDeltaVersionarBolsa`; nada de SQL instalado. T paró por cuota; sigue L | Todo el repositorio | - | - | M | L | #1166 (#1166, #1167 y #1168 fusionadas) | En curso | 16/10 | Cada retirada dice qué protección daba y qué prueba la sigue cubriendo; puertas de la CI verdes | Consenso de dirección con Fable y Astra, 16/10 |
| F0-31 | Versión única de la web: una huella calculada al construir el servidor sustituye los `?v=` puestos a mano en `portal.js`, `index.html`, `cache-publica-v1.json` y sus pruebas, que comprueban coherencia en vez de la cadena literal. Dos commits: el manual y la sustitución mecánica con un guion versionado y reproducible. Este identificador se reutilizó el 18/10: los roles de B119 que lo llevaban antes son ahora F0-34 | `web/static`; servidor de estáticos | - | - | M | T | #1241 | En curso | 18/10 | Dos PR de web seguidas no chocan por versiones; ninguna respuesta lleva el marcador sin sustituir; Chrome en una copia nueva sin errores de consola | Nota de dirección 18/10 15:40; condiciones del 18/10 22:44 |
| F0-32 | Tablero y plan al día tras la presentación: volcar las notas de dirección pendientes, quitar contradicciones y añadir las tareas de preparación de producción | `docs/producto` | - | - | S | Dirección | `trabajo/claude-tablero-f0-32-20261019` | En revisión | 19/10 | Es un documento: basta la fusión | Revisión de dirección 19/10 |
| F0-33 | CI que instala desde cero todas las migraciones del recorrido de la fase 1 y ejecuta sus pruebas SQL, incluidas denegaciones, y arreglo de GoSec en `internal/app/bootstrap`. Hasta el 18/10 era F0-29: sus PR llevan ese identificador en el título | `.github/workflows/ci.yml`; `deploy/postgresql` | - | - | M | L | #1199, #1206, #1208, #1209, #1212, #1215, #1222, #1223, #1226, #1232 y #1237 | Fusionada | 18/10 | La CI falla si una migración del recorrido no instala o una prueba SQL, también de denegación, no pasa | Consenso de dirección con Fable y Astra, 16/10 |
| F0-34 | El guion de la copia local crea sin LOGIN los dos roles que necesita B119 (`vec_externo_documentos_bolsa_datos` y `vec_interno_documentos_bolsa_rrhh`). Hasta el 18/10 era F0-31: su PR lleva ese identificador en el título | `deploy/principal`; `scripts/` | F0-28 | - | S | L | #1193 | Fusionada | 18/10 | Una copia levantada solo con el guion instala B119 sin crear roles a mano | Revisión de #1188, 16/10 |
| F0-35 | Excepción temporal: los módulos visibles del portal se eligen con la variable `VEC_PORTAL_MODULOS_VISIBLES`. Responsable: dirección. Se retira cuando esté hecha F0-16 | Configuración; portal | F0-16 | - | S | Dirección | - | Pendiente | 19/10 | Con F0-16 hecha, la variable desaparece y el menú solo obedece al interruptor de Administración | Orden de módulos 08/10; regla de excepciones del plan de simplificación 16/10 |
| F0-36 | Excepción temporal: CT de desarrollo usa el ámbito legado `organizacion:desarrollo:dipgra`, mientras el lote de Administración exige organizaciones gobernadas (`org_…`); por eso la copia no tiene un segundo técnico de RRHH. Pasar la composición local de CT a ámbitos gobernados. Responsable: dirección hasta que se asigne. Se retoma después del recorrido 4.0 | `internal/modules/contrataciontemporal`; composición de la copia | F1-31 | - | M | Sin asignar | - | Aparcada | 18/10 | Un segundo técnico de RRHH entra en CT con su asignación por la vía ordinaria y ya no se usa el ámbito legado | Notas de dirección 18/10 13:04 y 13:59; plan de dirección 18/10 14:40 |
| F0-37 | Primera entrada de una persona en un conjunto de organización y unidad: AUT50 y AUT44 solo amplían asignaciones que ya existen. Excepción temporal en la copia: semilla gemela de la del primer técnico de RRHH (L, dentro de F0-28). En producción saldrá del directorio o de Personal; falta decidirlo por consenso | Identidad; autorización | - | - | M | L (la semilla de la copia) | - | Pendiente | 18/10 | En la copia, una persona nueva entra con la semilla del guion; para producción hay una decisión escrita | Pregunta de diseño de dirección 18/10 13:24 |

## 6. Fase 1. Bolsa y Contratación temporal

Criterio de cierre (PLAN §5, adaptado a «todo en local»): recorrido del caso
4.0 con reinicio y acta, selectores encendidos, ningún «no disponible» sin
motivo y lecturas por debajo de 300 ms.

### 6.1 Lo que pidió RRHH el 15/10

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F1-01 | Situaciones en singular, sacadas del catálogo, en etiquetas, filtros y desplegables | `web/static/bolsa`; `locales` | - | - | S | T | #1148 | Fusionada | 16/10 | Chrome a 1440 y 390 px en castellano e inglés; revisión de usabilidad; pruebas de textos accesibles al día | RRHH 15/10 |
| F1-02 | «Cambiar situación» completo, con el motivo dentro del panel | `web/static/bolsa` | - | - | S | T | #1149 | Fusionada | 16/10 | Chrome a 1440, 1024 y 390 px; revisión de usabilidad | RRHH 15/10 |
| F1-03 | Cuadro de Bolsas: la columna «En revisión» se ve a 1440 px | `web/static/bolsa` | - | - | S | T | #1152 | Fusionada | 15/10 | Chrome a 1440 y 390 px sin desbordamiento | RRHH 15/10 |
| F1-04 | Publicar reglas de situación: fachada sobre B32 con permiso propio, versión esperada, motivo, recibo y autorización en la misma transacción; el arranque solo lee. Incluye la activación explícita de la fuente normativa (antes F1-05) y, para los candidatos «En revisión», confirmar la respuesta o devolverlos al turno con motivo. Su pantalla cubre también F1-06 | `internal/modules/bolsa`; SQL de Bolsa | - | - | M | S | #1155 | Fusionada | 18/10 | Publicación con recibo y replay igual; sin activación se niega con motivo; el arranque no sobrescribe; un candidato «En revisión» se confirma o vuelve al turno con su motivo | RRHH 15/10; duda 157 |
| F1-05 | Activar la fuente normativa del catálogo de situaciones. Cancelada el 15/10: fundida en F1-04, misma PR de S | - | - | - | - | S | - | Cancelada | 15/10 | - | Consenso de dirección 15/10 |
| F1-06 | Pantalla de reglas de situación para RRHH. Cancelada el 18/10: la cubre la pantalla de F1-04 (#1155) | `web/static/bolsa` | F1-04 | - | M | S | - | Cancelada | 18/10 | - | RRHH 15/10; decisión de dirección 18/10 09:20 |
| F1-07 | Justificante PDF de la renuncia justificada: lo sube el aspirante y RRHH lo descarga en la ficha según su rol (AD277, D22, B119, CT222). Primera de la cadena SQL | `internal/modules/bolsa`; Documentos | - | - | M | O | #1147 | Fusionada | 16/10 | PDF subido y descargado en Chrome; más de 16 MiB da 413 avisado antes; reinicio | RRHH 15/10 |
| F1-08 | R1a: regla publicada de criterios (art. 9, apartado cuarto, y 11.2), motivos de exclusión visibles con lectura propia auditada (AD276) y parada «pendiente de comprobación por RRHH» | `internal/modules/bolsa`; `internal/modules/contrataciontemporal` | F1-21, F1-22 | Dudas 163 y 165 (configurables, no bloquean) | M | R | #1144, #1170 y #1178 | Fusionada | 16/10 | La selección nominal se detiene con el motivo; tabla de motivos con nombres; una bolsa sin política sigue el orden de siempre | RRHH 15/10 |
| F1-09 | R1b, parte de servidor: acto de Bolsa que acredita los hechos del art. 9, apartado cuarto (evidencia, resultado calculado, versión esperada, recibo). Migraciones CT233, AD290, B133, B134, B137 y AD293. La pantalla va en F1-57 | `internal/modules/bolsa`; SQL de Bolsa | F1-07, F1-08 | Dudas 163 y 165 (configurables) | L | R | #1190, #1196, #1201 y #1203 | En curso | 16/10 | RRHH resuelve un caso pendiente por la API y la selección sigue; reintento con el mismo recibo | RRHH 15/10 |
| F1-10 | R1b en CT: confirmar contexto y reservar con ese acto; descontar solo el tiempo desempeñado; resolución detenida hasta el cese efectivo | `internal/modules/contrataciontemporal` | F1-09 | Duda 165 (configurable) | M | R | - | Pendiente | 15/10 | Un caso con cese futuro queda detenido con el motivo a la vista | RRHH 15/10 |
| F1-11 | Modalidad y causa gobernadas en los datos de la oferta y en su canon de publicación (pacto entre R y S). Incluye la activación de F1-16. Pasa de R a O el 18/10, porque O la necesita para F1-13 y F1-14 | `internal/modules/bolsa` | F1-08, F1-16 | - | M | O | #1200 | En curso | 18/10 | Una oferta publicada lleva modalidad y causa con fuente y versión | RRHH 15/10; reparto de dirección 18/10 |
| F1-12 | Preferencias internas de «Mi Bolsa»: migración propia (AD279) con consumidor y control de auditoría sacados de AD273, y montaje | `internal/modules/bolsa`; `web/static/area-personal` | F1-16 | - | M | O | #1187 y #1211 | Fusionada | 18/10 | GET y PUT con recibo; reinicio en copia | RRHH 15/10 |
| F1-13 | R1c, servidor: preferencias de modalidad en el portal externo real, con versión y recibo. Sin permiso propio por tipo de oferta: usa las mismas acciones de preferencias propias (consultar y ordenar) y el tipo va en el contrato de datos. Tres PR encadenadas; las dos lecturas van en AD296 y la sustitución del catálogo v1 por v2 usa `vec-catalogo-acciones-admitir` (AUT58 y AUT61). Las preferencias solo filtran la vista y los avisos de ofertas publicadas | `internal/modules/bolsa` | F1-11, F1-12 | Duda 164 (configurable) | M | O | #1216 y #1220 | En curso | 18/10 | Prueba de que la preferencia no excluye ni cuenta como renuncia; las lecturas solo admiten los campos cerrados | RRHH 15/10; decisiones de dirección 18/10 08:36, 10:47, 11:50 y 11:55 |
| F1-14 | R1c, pantalla: elegir modalidades en «Mi Bolsa» del portal externo real, aviso si la lista queda vacía y «Ver todas» sin guardar | `web/static/area-personal` | F1-13 | - | M | O | - | Pendiente | 18/10 | Chrome a 1440 y 390 px; reinicio | RRHH 15/10; decisión de dirección 18/10 11:50 |
| F1-15 | Publicación de ofertas con plazo desde la publicación y recuperación del recibo (B117) | `internal/modules/bolsa` | - | - | - | S | #1145 | Hecha | 15/10 | Dos plazas, aceptación en «Mi Bolsa» y reinicio en la copia de S | RRHH 15/10 |
| F1-16 | Publicar la oferta de varias plazas desde la ficha: reserva en CT, vínculo, cierre de intención y descarte (AD278, B120, B121, CT220), guarda de cancelación (CT223) y recuperación sin la clave del navegador | `internal/modules/contrataciontemporal`; `internal/modules/bolsa` | F1-07, F1-21, F1-22 | - | L | S | #1146 | Fusionada | 18/10 | Ensayo en copia: publicar, reintento, recarga tras respuesta incierta; cancelar con oferta en preparación da 409 | RRHH 15/10 |
| F1-17 | Retirar la oferta publicada al cancelar el expediente: Bolsa consume `ct.expediente_cancelado.v1` con historia y no adjudica plazas de una oferta retirada. Aparcada el 18/10: no tiene consumidor; la pregunta 167 a RRHH (#1197) dirá si se puede anular un expediente ya fiscalizado | `internal/modules/bolsa` | F1-16, F1-18 | - | M | S | #1197 (pregunta a RRHH) | Aparcada | 18/10 | Cancelar retira la oferta una sola vez, también si el evento se repite | RRHH 15/10; decisión de dirección 18/10 |
| F1-18 | Aviso de la oferta por correo como intención durable por el puerto de canales; no abre el plazo | `internal/modules/bolsa` | F1-16 | Correo real: duda 88 (vale el buzón de pruebas) | M | S | #1180 y #1184 | Fusionada | 18/10 | El aviso queda pendiente y se envía tras reiniciar, sin duplicar | RRHH 15/10 |
| F1-19 | Propuesta de nombramiento desde una oferta adjudicada: un expediente hijo por persona, derivado del padre en v6, y un acto nuevo de aceptación de la plaza adjudicada. El consumidor de autorización (AD292) lo hace R después de F0-18. Queda una pregunta a RRHH sobre la resolución común, para una segunda versión | `internal/modules/contrataciontemporal` | F1-16, F0-18 | - | L | B | #1204 y #1225 (AD292) | En curso | 18/10 | Tras adjudicar dos plazas, propuesta con recibo para cada persona | RRHH 15/10; consenso de dirección con Fable y Astra 18/10 |
| F1-20 | Variante de dos personas del caso 4.0 recorrida hasta la propuesta en la copia local | Recorrido | F1-19, F1-25 | - | S | Sin asignar | - | Pendiente | 15/10 | Dos aceptaciones, adjudicación por posición y propuestas; anotada en FLUJO 4.0 | FLUJO 4.0 |

### 6.2 Recorrido del caso 4.0 en copia local

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F1-21 | Separar la unidad gestora de CT de la unidad orgánica del puesto en el puente de la bolsa constituida. Es la puerta del camino crítico | `internal/app/bootstrap/contratacion_temporal_llamamiento_datos_constituidos.go:255` | - | - | M | R | #1153 | Fusionada | 16/10 | La selección del puesto 9/01 (unidad del centro 520) funciona con la unidad gestora de RRHH | Canal 15/10 |
| F1-22 | Organización histórica en `vec-server` combinado, con su rol de lectura de seis campos | `internal/modules/personal`; organización | - | - | M | R | #1150 | Fusionada | 16/10 | GET 200 con la sesión de RRHH; sin configuración solo esa función da 503 | Canal 15/10 |
| F1-23 | Retomar el ensayo de P: copia local al día con `main` (firma, ofertas), lista SQL causal, administradores, firmantes y empleados de copia. El equipo P empieza el 20/10 | `deploy/principal/candidata_hz39` | F0-03 (o renovar el material cada día) | - | M | P | #1128 | Pendiente | 19/10 | La copia arranca con todo lo fusionado y las ocho variables de la fase | Consenso de dirección 15/10 |
| F1-24 | Pasos 1 a 8 (de la petición a la fiscalización, con las dos firmas del informe en 7b) | Recorrido | F1-23 | - | S | P | - | Pendiente | 15/10 | Recibo y versión de cada paso en Chrome | FLUJO 4.0 |
| F1-25 | Pasos 9 a 11: llamamiento desde la bolsa constituida, aceptación en «Mi Bolsa» y resolución | Recorrido | F1-21, F1-22, F1-24 | - | S | P | - | Pendiente | 15/10 | Recibo del llamamiento a Antonio y de su aceptación | FLUJO 4.0 |
| F1-26 | Fiscalización con reparo y subsanación: montar la subsanación del reparo y la ruta de Plantillas, y recorrerlas | `internal/modules/contrataciontemporal`; Plantillas | F1-24 | - | M | P | - | Pendiente | 15/10 | Reparo, informe nuevo y fiscalización favorable con su recibo; la plantilla se abre desde su ruta | Consenso de dirección 15/10 |
| F1-27 | Pasos 12 a 15: propuesta, resolución, documentos y tres firmas de la resolución en el mismo expediente | Recorrido | F1-25 | - | S | P | - | Pendiente | 15/10 | PDF verificados y descargables desde la ficha | FLUJO 4.0 |
| F1-28 | Pasos 17 a 21: incorporación acreditada, alta en Personal, GINPIX manual, cese, vuelta a la bolsa y cierre | Recorrido | F1-27 | - | S | P | - | Pendiente | 15/10 | Antonio vuelve a «Disponible» o «Disponible desde» | FLUJO 4.0 |
| F1-29 | Pasos e1 a e5 del aspirante con «Mi Bolsa» interna, encendiendo la pestaña de inscripción (hoy oculta en `vistas.json`) | `web/static/area-personal/vistas.json` | F1-23 | - | M | O | - (#1140 rechazada; no hay piezas en curso) | Pendiente | 15/10 | Inscripción, méritos, admisión, incorporación y alegación con recibos | FLUJO 4.0 |
| F1-30 | Cese sin candidato vinculado sin bloquear el relevo (#774 en `main`) | Recorrido | F1-23 | - | S | P | - | Pendiente | 15/10 | Alta, cursor, vínculo posterior y estado pendiente en «Mi Bolsa» | FLUJO 4.0 |
| F1-31 | Reiniciar aplicación y PostgreSQL, repetir las consultas y escribir el acta (respuesta HTTP y captura por paso), con las variantes de dos plazas y de vacante | `docs/estudio_requisitos/` | F1-04, F1-10, F1-14, F1-20, F1-26, F1-28, F1-32, F1-33, F1-55 | - | L | P | - | Pendiente | 15/10 | Mismos recibos y fechas, ningún duplicado; acta en `docs/estudio_requisitos/` | PLAN §5, fase 1 |

### 6.3 Resto de condiciones de cierre

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F1-32 | Las ocho variables de la fase encendidas en la copia, incluida la verificación de firma; el aviso de dependencias no muestra ausencias | Configuración de la copia | F0-01, F1-23, F0-03 (o renovar el material cada día) | - | S | P | - | Pendiente | 15/10 | Registro de arranque limpio | PLAN §5, fase 1 |
| F1-33 | Ningún «no disponible» sin motivo y ninguna respuesta 404 o 503 de Bolsa o CT durante el recorrido | `internal/modules/bolsa`; `internal/modules/contrataciontemporal` | F1-28 | - | M | P | - | Pendiente | 15/10 | Registro del recorrido sin esos códigos | PLAN §5, fase 1 |
| F1-34 | Lecturas en el percentil 95 por debajo de 300 ms (cuadro de CT, lista, ficha, bolsas, candidatos y ficha del candidato) y base por debajo de 100 ms, medidas en copia local con el volumen de la principal | `internal/modules/contrataciontemporal`; `internal/modules/bolsa` | F1-23 | - | M | P | #1189 (medición entregada; faltan los candidatos y la ficha del candidato) | En curso | 16/10 | Medición HTTP con su acta de rendimiento | PLAN §5, fase 1 |

### 6.4 Firma, después de #1137

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F1-35 | Cinco firmas reales, PDF desde la ficha tras reiniciar, plantillas v4 y detector de datos personales que respeta las huellas | Firma | - | - | - | H | #1137 | Hecha | 15/10 | Cinco firmas en la copia de H y PDF desde la ficha tras reiniciar | Canal 15/10 |
| F1-36 | Recuperación histórica de firmas con unidad nula y AD280 (fuera del recorte de CT224), en tres partes: A recupera las firmas originales con cargo y unidad vigentes, B restringe los selectores a la unidad autorizada y C consulta las firmas anteriores desde la ficha | Firma; SQL de firma | - | - | M | H | #1172, #1173 y #1179 | Fusionada | 18/10 | Consulta por unidad de una firma anterior al plan | Revisión de #1137 |
| F1-37 | Menores de la revisión de #1137: CT218 por organización, límite del PDF firmado (1 y 2 MiB), `nil` en `cerrarAuditoria` | Firma | - | - | S | H | #1163 | Fusionada | 16/10 | Pruebas focales | Revisión de #1137 |

### 6.5 Resto de la fase 1

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F1-38 | Manual de la jefatura de RRHH con el apartado 7 «Hoy, Falta, Al final» | `docs/` | - | - | - | Dirección | #1143 | Hecha | 15/10 | Es un documento: basta la fusión | Canal 15/10 |
| F1-39 | Textos: cómo corregir o publicar la categoría antes de vincularla | `web/static`; `locales` | - | - | S | T | #1008 (rama conservada) | Pendiente | 15/10 | Revisión de usabilidad y fusión | PLAN §6 anterior al tablero |
| F1-40 | Inscripción: regla por convocatoria para elegir solicitud separada o conjunta. Menores de BC12: una modalidad no válida da B9601 en vez de 55000 (`000012:117`) y el listado entero falla si una versión trae una modalidad no válida (`:279`) | `internal/modules/bolsa` | - | - | M | B | #1177 (desbloqueada por #1188) | En curso | 16/10 | RRHH la fija antes de publicar; queda en la versión de las bases. Falta el recorrido externo en Chrome, que espera a que F0-28 monte el portal externo | PLAN §6 anterior al tablero |
| F1-41 | Solicitud conjunta con su recibo y reintento (B105 solo cubre la separada; no instalar B105 sola) | `internal/modules/bolsa` | F1-40 | - | M | Sin asignar | rama B105 | Pendiente | 15/10 | Las dos modalidades con recibo y reintento | PLAN §6 anterior al tablero |
| F1-42 | Documentos del expediente en un solo sitio de la ficha: comprobar el commit `046a2f2fa`, que ya estaba en `main` | `web/static`; `internal/modules/contrataciontemporal` | - | - | S | T | #1231 | Fusionada | 18/10 | Todos se ven y se descargan desde la ficha | PLAN §6 anterior al tablero |
| F1-43 | Tarjetas de plazo en Inicio (CT192): comprobar la rama `codexs-ct-filtros-plazos-20261008`. Aparcada el 16/10: ampliar el tipo V1 y reconstruir 16 funciones va contra las reglas de simplificación; espera a la tarea de lecturas de la fase 2 (F2-13) | `web/static` | - | - | S | T | rama sin PR | Aparcada | 16/10 | Cada tarjeta abre su lista filtrada | Borrador 15/10 |
| F1-44 | Retirar la duda 162 citando la norma y corregir FLUJO 4.2, que aún dice puesto en texto libre (#1114 ya lo resolvió) | `dudas.md`; `docs/producto/FLUJO_APP.md` | - | - | S | T | #1159 | Fusionada | 15/10 | `dudas.md` sin la 162 y FLUJO 4.2 al día | Consenso de dirección 15/10 |
| F1-45 | Vista de la cartera de méritos en «Mi Bolsa» con la API real, sin cifras inventadas | `web/static/area-personal` | - | - | M | O | #1183 | Fusionada | 16/10 | La aspirante ve sus méritos por categoría. Se cierra con los hechos de la cartera, sin puntos; la valoración espera al motor común de baremación (fuera de la fase 1). #1183 solo cambia documentación | PLAN §6 anterior al tablero |
| F1-46 | Cuenta externa propia del aspirante: confianza externa nominal (HZ38, opción B) y motivo documental en la autoridad externa | Identidad; `internal/modules/aspirantes` | - | - | M | O | - (#1140 rechazada) | Pendiente | 15/10 | Entrada externa a «Mi Bolsa» con 200 y reinicio. Posible cierre por #1147; dirección lo comprueba | PLAN §6 anterior al tablero |
| F1-47 | Pantalla de no incorporación: comprobar si funciona y recorrerla (el vídeo 15 se omitió porque no estaba acreditada). La regla exige dos personas distintas de RRHH y la copia todavía no tiene un segundo técnico de RRHH | `web/static`; `internal/modules/contrataciontemporal` | F0-36 | - | S | T | #1194 (borrador) | Pendiente | 18/10 | Recorrido con recibo | Borrador 15/10; nota de dirección 18/10 13:59 |
| F1-48 | Petición del centro firmada por el técnico y por el diputado o la diputada | `internal/modules/contrataciontemporal`; Firma | - | Decisión de RRHH: duda por numerar (las dudas 166 a 170 que se citaban tratan de otros asuntos o no existen) | L | Sin asignar | - | Bloqueada | 15/10 | Petición con dos firmas verificadas | Consenso de dirección 15/10 |
| F1-49 | Orden configurable del ofrecimiento antes o después de Intervención | `internal/modules/contrataciontemporal` | - | Decisión de RRHH: duda por numerar (las dudas 166 a 170 que se citaban tratan de otros asuntos o no existen) | L | Sin asignar | - | Bloqueada | 15/10 | Cambio por catálogo sin tocar código | Consenso de dirección 15/10 |
| F1-50 | Bandeja propia de Intervención | `internal/modules/contrataciontemporal`; `web/static` | - | Decisión de RRHH: duda por numerar (las dudas 166 a 170 que se citaban tratan de otros asuntos o no existen) | M | Sin asignar | - | Bloqueada | 15/10 | RRHH decide si hace falta; si sí, Intervención ve solo lo suyo | Consenso de dirección 15/10 |
| F1-51 | Plantillas definitivas de contratos, informes y resoluciones | Plantillas de CT | - | Duda 124 (RRHH) | S | Sin asignar | - | Bloqueada | 15/10 | Documentos con las plantillas oficiales | PLAN §6 anterior al tablero |
| F1-52 | Oferta al SAE cuando la bolsa se agota | `internal/modules/contrataciontemporal` | - | Dudas 72, 73 y 126 (RRHH) | L | Sin asignar | - | Bloqueada | 15/10 | Oferta con sus documentos | Borrador 15/10 |
| F1-53 | Momento de la toma de posesión y fecha de efectos del cese como valor definitivo (hoy funciona con valor de catálogo) | Catálogo de CT; Personal | - | Duda 140 (no impide funcionar) | S | Sin asignar | - | Pendiente | 15/10 | Valor publicado por RRHH | Borrador 15/10 |
| F1-54 | Corregir el catálogo de plazos de CT citando la norma: c04 con el art. 12.4 del RD 424/2017, y c03 | `cmd/vec-preparar-catalogo-reglas-ct`; catálogo de plazos | - | - | S | T | #1157 | Fusionada | 16/10 | Cada plazo del catálogo lleva su cita; c03 y c04 coinciden con la norma | Consenso de dirección 15/10 |
| F1-55 | Reformulada el 18/10: oferta de dos plazas con aceptación previa, adjudicación por orden de la bolsa y recuperación tras un fallo. «Sin respuesta» solo existe en el llamamiento directo y no hay retirada por cancelación (F1-17 está aparcada) | `internal/modules/bolsa`; `internal/modules/contrataciontemporal` | F1-16 | - | M | S | #1207 | Fusionada | 18/10 | Dos personas aceptan, se adjudica por orden y cada plaza conserva su recibo tras recargar o reiniciar, sin duplicados | Consenso de dirección 15/10; decisión de dirección 18/10 09:27 |
| F1-56 | Correspondencia verificada entre la persona y su fila de Convoca (opcional) | Carga de Convoca en Bolsa | - | - | M | Sin asignar | - | Pendiente | 15/10 | Cada persona cargada enlaza con una sola fila de Convoca, comprobada | Consenso de dirección 15/10 |
| F1-57 | R1b, pantalla: RRHH resuelve el caso «pendiente de comprobación» desde la ficha del candidato | `web/static/bolsa` | F1-09 | - | M | T | - | Pendiente | 15/10 | Chrome a 1440 y 390 px: RRHH resuelve, ve el recibo y la selección sigue | Consenso de dirección 15/10 |
| F1-58 | Firma de F1-36 por la vía de cargos de firma: AUT98 mínima, operación anexada, plan v2, sucesión de asignaciones y lector CT22x que acepte v2 | Firma; SQL de firma | F1-36 | - | M | H | - | Pendiente | 16/10 | Firma por cargo con plan v2 leída por CT tras reiniciar; ensayo SQL y dos revisiones | Consenso de dirección 16/10 |
| F1-59 | Vínculo entre la respuesta publicada y su oferta, para abrir la resolución de la respuesta publicada en F1-06 | `internal/modules/bolsa`; `internal/modules/contrataciontemporal` | F1-06 | - | S | S | - | Cancelada | 16/10 | Desde F1-06 se abre la resolución de una respuesta publicada con su oferta. Cancelada el 16/10: en las ofertas publicadas solo existe Aceptar (disposiciones) y la adjudicación va por orden; no hay respuesta en revisión (FLUJO §4.1). S comprueba que disposiciones conserva `oferta_ref` | Consenso de dirección 16/10 |
| F1-60 | La bolsa guarda su organización y su unidad gestora como dato gobernado de su constitución, para que los actos de Bolsa (B100, B17, B128) cotejen el ámbito del recurso con la asignación del actor | `internal/modules/bolsa`; SQL de `bolsa_llamamientos` | - | - | M | Sin asignar | - | Pendiente | 16/10 | Un acto sobre una bolsa de otra unidad gestora se deniega aunque el actor tenga permiso en la suya; ensayo y revisión SQL independiente | Consenso de dirección con Fable y Astra, 16/10 |
| F1-61 | Pieza real que confirma la publicación de las bases de una convocatoria (`ConfirmarPublicacion` sin adaptador productivo), para cerrar F1-40 de extremo a extremo | `internal/modules/bolsa` | F1-40 | - | M | Sin asignar | - | Pendiente | 16/10 | RRHH elige la modalidad, publica las bases y la inscripción la consume, tras reiniciar | Consenso de dirección con Fable y Astra, 16/10 |
| F1-62 | Productor administrativo gobernado para publicar catálogos configurables CC1; hoy solo se cargan en la copia con `generar_carga_cc1.py` | `internal/modules/administracion`; SQL de catálogos | - | - | M | Sin asignar | - | Pendiente | 16/10 | Un catálogo CC1 se publica con dos aprobaciones y su recibo, sin SQL a mano, y sigue tras reiniciar | Consenso de dirección 16/10 |
| F1-63 | Organización legada de las firmas (CA45, AUT37, Personal31). Aparcada el 18/10 por consenso: queda como semilla declarada de la copia, con H como responsable; H la pone en el guion único y en FLUJO 4.4. Se retira con F2-12 y F2-14 | Firma; `internal/modules/administracion` | - | - | M | H | #1198 (semilla de la copia) | Aparcada | 18/10 | Las firmas de la copia salen de la semilla del guion, sin asignaciones por SQL a mano; al cerrar F2-12 y F2-14 la semilla desaparece | Consenso de dirección 16/10; decisión de dirección 18/10 |
| F1-64 | Menores de vigencias de AD282: quitar la clave y la raíz fijas de desarrollo 2026-2036 y la vigencia fija de 24 h de `portal_externo_publicacion_v3.go`; las vigencias pasan a ser configurables (90 días como máximo en desarrollo y 24 h en producción). Cerrada el 18/10 con la parte 1: la raíz y la clave de desarrollo no se rotan, y la rotación con material propio pasa a la preparación de producción (F7-21) | Firma; `internal/modules/contrataciontemporal` | F1-36 | - | S | H | #1181 | Fusionada | 18/10 | Pruebas focales con vigencias distintas en desarrollo y en producción | Consenso de dirección 16/10; decisión de dirección 18/10 07:45 |
| F1-65A | Menores de CT226: la restricción CHECK de `catalogo_plantillas_historia_v1` solo admite la organización de desarrollo y una unicidad global; retirar la v1 de compatibilidad cuando no tenga consumidores | SQL de plantillas de CT | - | - | S | H | #1182 | Fusionada | 16/10 | El CHECK admite cualquier organización activa y la v1 ya no tiene consumidores | Consenso de dirección 16/10 |
| F1-66 | Plantillas para varias organizaciones: restricción CHECK, índice de unicidad, operador y AD99 de `catalogo_plantillas_historia_v1` | SQL de plantillas de CT | - | - | S | H | - | Aparcada | 16/10 | El CHECK, el índice, el operador y AD99 admiten cualquier organización activa | Consenso de dirección 16/10. Aparcada: sin consumidor, la instalación tiene una sola organización |
| F1-67 | Preparación de bases (AD153 a BC8) sobre el núcleo vigente, con guardar y consultar la preparación, para que RRHH elija desde la web la modalidad de F1-40 | `internal/modules/bolsa`; SQL | AD288 | - | M | R | - | Pendiente | 16/10 | RRHH elige la modalidad desde la web y la preparación se conserva tras reiniciar | Consenso de dirección 16/10 |
| F1-68 | Auditoría de la lectura de inscripción del aspirante externo: el cotejo del contexto solo miraba el almacén interno y toda lectura externa daba 503 (CA47 y AD289) | SQL de contexto del actor y de autorización | - | - | S | Dirección | #1188 | Fusionada | 16/10 | La lectura externa queda asentada; un recibo inexistente, ajeno o presente en los dos almacenes se deniega sin asiento | Ensayo de #1177, 16/10 |
| F1-69 | Menores de CT231: una sola selección confirmada por expediente (llamamiento y oferta), con índice parcial | SQL de `internal/modules/contrataciontemporal` | - | - | S | R | #1210 | Fusionada | 18/10 | Una segunda selección confirmada en el mismo expediente se rechaza en la base; la búsqueda usa índice | Revisión de #1178, 16/10; decisión de dirección 18/10 09:48 |
| F1-70 | Menor de CT230: la rama nativa solo admite la autoridad `autoridad:bolsa:llamamiento:desarrollo`; producción necesitará su propia migración. Aparcada el 18/10 hasta preparar producción, porque no hay emisor productivo; se hace dentro de F7-21 | SQL de `internal/modules/contrataciontemporal` | - | - | S | R | - | Aparcada | 18/10 | La rama nativa acepta la autoridad de producción con su migración; la de desarrollo sigue igual | Revisión de #1170, 16/10; decisión de dirección 18/10 09:48 |
| F1-71 | Confirmar desde la ficha del aspirante una aceptación de llamamiento directo que espera revisión: RRHH escribe el motivo y la persona pasa a pendiente de incorporación, con respuesta, historia, recibo y evento en la misma transacción (B128 y su consumidor nominal, AD288). No cubre las ofertas publicadas (ver F1-59). Hasta el 16/10 sus PR citaban F1-06 por error | `internal/modules/bolsa`; SQL de Bolsa; `web/static/bolsa` | F1-04 | - | M | S y R | #1164 (S) y #1176 (R, consumidor) | Fusionada | 18/10 | La confirmación da recibo y la misma intención lo recupera; otro contenido o una situación cambiada da conflicto; una decisión ya usada no autoriza otra | Consenso de dirección 16/10 |
| F1-72 | Propuesta y resolución identifican a la persona nombrada con nombre, apellidos y documento enmascarado, tomados del acta de la participación aceptada en la bolsa (por la `persona_ref` que CT ya recibe de Bolsa) y congelados al guardar el original firmable. Una fila de selector más en la consulta del detalle de RRHH, sin rol ni permiso nuevos. O hace la fuente, S el lado de CT y R la fila en AD291 si hace falta. Hoy la plantilla dice «identificación autorizada pendiente». Se retoma después del recorrido 4.0 | `internal/modules/bolsa`; `internal/modules/contrataciontemporal`; plantillas de CT | F1-71 | Pregunta a RRHH y al DPD sobre el DNI completo y el tablón: duda por numerar (no frena el corte) | M | O, S y R | - | Aparcada | 18/10 | Los borradores muestran nombre y documento enmascarado; el original firmado conserva el dato y la huella del acta; una fila ambigua sigue «pendiente» con aviso en la ficha | Decisión de dirección 18/10 11:25; plan de dirección 18/10 14:40 |
| F1-73 | El alta directa de RRHH transmite el número de personas de la solicitud, con el esquema `alta_necesidad.v2` (la v1 queda tal cual) | `internal/modules/contrataciontemporal` | - | - | S | O (revisa S) | #1230 | Fusionada | 18/10 | Dos personas pedidas por el alta directa llegan a CT235 y se conservan tras reiniciar. Para seguir hasta la reserva de plazas falta el plan de firma nominal de la copia | Decisiones de dirección 18/10 12:50 y 13:04 |
| F1-74 | La cabecera del expediente decía «esperando Intervención» con la fiscalización favorable v6 (CT09) | `internal/modules/contrataciontemporal`; `web/static` | - | - | S | B (revisa S) | #1229 | Fusionada | 18/10 | Tras fiscalizar, la cabecera dice el siguiente paso real | Decisión de dirección 18/10 12:50 |
| F1-75 | La opción «Propio» del contacto tenía el valor vacío y no dejaba registrar el contacto propio | `web/static/bolsa` | - | - | S | T (revisa S) | #1234 | Fusionada | 18/10 | El contacto propio se registra con recibo y se recupera | Nota de dirección 18/10 13:59 |
| F1-76 | Arreglos visibles para la presentación, solo de interfaz: Documentos de Bolsas se quedaba en «Comprobando», Avisos mostraba «Failed to fetch» en vez del texto del catálogo y el enlace a Fiscalización abría el formulario sin precargar | `web/static` | - | - | S | Dirección | #1238 | Fusionada | 18/10 | Las tres pantallas en Chrome sin esos fallos | Nota de dirección 18/10 14:55 |
| F1-78 | La preparación de la resolución de formalización respondía 503 porque pedía a CT225 una columna que no tiene; la consulta pasa a ser propia contra CT69 | `internal/modules/contrataciontemporal/adapters/postgres` | - | - | S | Dirección | #1239 | Fusionada | 18/10 | La ficha de un expediente en nombramiento abre la preparación sin 503; una prueba coteja las columnas con CT69 | Preparación de la presentación, 18/10 |
| F1-79 | Dos rastros visibles para la presentación: la cabecera del portal al entrar en Fiscalización por un enlace y el cargo del ratificador | `web/static/portal.js` | - | - | S | S | #1240 | Fusionada | 18/10 | Chrome a 1440 y 390 px con la cabecera resuelta | Preparación de la presentación, 18/10 |
| F1-80 | Guion de la presentación del 19/10: pasos por perfil, datos y puntos de parada, recorrido en la copia de la presentación. Queda fuera de Git | Fuera del repositorio | - | - | S | S | - | Hecha | 18/10 | Guion entregado el 18/10 con sus límites: inscripción, respuesta y aceptación, y cese no acreditados | Orden de dirección 18/10 |
| F1-81 | Vídeos y capturas de la presentación en `docs/presentacion/2026-10-19/` | `docs/presentacion` | - | - | S | Dirección | #1242 | Hecha | 19/10 | Es un documento: basta la fusión | Petición de Alberto 18/10 |
| F1-82 | Manual de Intervención (`docs/manuales/intervencion.md`). Hoy dice que Intervención no tuvo trámite propio, pero Fiscalización ya se ve en la copia y en la presentación | `docs/manuales` | - | - | S | T | - | Pendiente | 19/10 | El manual explica cómo fiscalizar, poner un reparo y dar la conformidad con lo que funciona en la copia local, con su apartado «Hoy, Falta, Al final» | Orden de dirección 19/10; regla de manuales 15/10 |
| F1-83 | Unificar la sesión del antiguo «Mi Bolsa» interno de desarrollo con la del portal externo real, que es donde van F1-13 y F1-14 | `web/static/area-personal`; identidad | - | - | M | Sin asignar | - | Aparcada | 18/10 | El aspirante entra a «Mi Bolsa» por una sola vía y la interna de desarrollo se retira | Decisión de dirección 18/10 11:50 |

## 7. Fase 2. Personal y Organización (por capacidad)

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F2-12 | Simplificación 1, permisos por datos: verificador estable y catálogo cerrado de acciones, sin distribuidor central ni versiones numeradas de rol. Objetivo: añadir una acción normal en una jornada sin tocar el núcleo. Incluye el rol externo del aspirante, hoy con semilla de copia para las acciones del justificante (F1-07) y los campos de modalidad (F1-40), que se retiran al integrarla | SQL del núcleo de autorización; catálogo de acciones | F1-31 | - | L | Sin asignar | - | Pendiente | 16/10 | Se añade una acción normal en una jornada sin migración del núcleo; denegaciones y recorrido iguales | Consenso de dirección con Fable y Astra, 16/10 |
| F2-13 | Simplificación 2, lecturas con sesión acreditada, permiso vigente y auditoría, sin consumir una autorización de un solo uso | Lecturas de todos los módulos | F1-31 | - | L | Sin asignar | - | Pendiente | 16/10 | Equivalencia ruta a ruta: misma respuesta, misma denegación y mismo apunte de auditoría | Consenso de dirección con Fable y Astra, 16/10 |
| F2-14 | Simplificación 3, base SQL consolidada para instalaciones nuevas y vía de actualización para las existentes, sin perder historia. Desde el 18/10 incluye retirar la semilla de firmas de la copia (F1-63) y dejar a CT en una organización gobernada (`org_…`) o en la legada canónica | `deploy/postgresql` | F1-31 | - | L | Sin asignar | - | Pendiente | 18/10 | Instalación nueva y actualizada coinciden en esquema, permisos y comportamiento | Consenso de dirección con Fable y Astra, 16/10; decisión de dirección 18/10 sobre F1-63 |
| F2-15 | Simplificación 4, el arranque solo compone: cada módulo es dueño de lo suyo y se retiran las variantes sin consumidor | `internal/app/bootstrap`; módulos | F1-31 | - | L | Sin asignar | - | Pendiente | 16/10 | El arranque no lleva reglas de módulo; las variantes retiradas no tienen consumidor; recorrido igual | Consenso de dirección con Fable y Astra, 16/10 |
| F2-01 | Consulta histórica de organización autorizada y montada para RRHH | `internal/modules/personal`; organización | F1-22 | - | L | Sin asignar | - | Pendiente | 15/10 | Consulta histórica en menos de 300 ms | Plan del módulo |
| F2-02 | Gobierno de categorías conservado y su convergencia | RPT | - | Duda 149 (configurable) | L | Sin asignar | rama conservada | Pendiente | 15/10 | Una categoría en uso se deshabilita, no se borra | Plan del módulo |
| F2-03 | Fuente e importación de plantilla y puestos (6 a 10 días: trocear en fuente, simulación, plantilla y puestos) | RPT | F2-01 | - | XL | Sin asignar | - | Pendiente | 15/10 | Importación con vista previa, aprobación y vuelta atrás | Plan del módulo |
| F2-04 | Ocupación, reserva y las tres vistas de vacantes, con la relación que aporta Personal (5 a 8 días: trocear) | RPT; `internal/modules/personal` | F2-03 | - | XL | Sin asignar | - | Pendiente | 15/10 | Persona, plaza y puesto con fechas y acto | Plan del módulo |
| F2-05 | Tipos, jerarquías, delegaciones y suplencias | Organización | F2-01 | - | L | Sin asignar | - | Pendiente | 15/10 | Suplencia con vigencia consultable | Plan del módulo |
| F2-06 | Ficha del empleado con relaciones, periodos y procedencia | `internal/modules/personal` | - | Dudas 27-29 y 34 (RRHH) | L | Sin asignar | - | Bloqueada | 15/10 | Un alta desde CT aparece con relación, puesto y documentos | Plan del módulo |
| F2-07 | Documentos ligados a cada hecho de Personal | `internal/modules/personal`; Documentos | F2-06 | - | M | Sin asignar | - | Pendiente | 15/10 | Descarga del original desde el periodo | Plan del módulo |
| F2-08 | Situaciones por régimen y acto, con sus efectos | `internal/modules/personal` | F2-06 | Dudas 27-29 (RRHH) | L | Sin asignar | - | Bloqueada | 15/10 | El cese cierra la relación y avisa a quien corresponde | Plan del módulo |
| F2-09 | Rectificación de datos del empleado | `internal/modules/personal` | F2-06 | Duda 28 (RRHH) | L | Sin asignar | - | Bloqueada | 15/10 | Propuesta, decisión y versión nueva | Plan del módulo |
| F2-10 | Puerto de lectura del empleado para Cronos y Dietas. Cancelada el 15/10: duplicaba F0-18 y se fundió en ella | - | - | - | - | C y D | - | Cancelada | 15/10 | - | Consenso de dirección 15/10 |
| F2-11 | Puerto de servicios de oficio para certificados y antecedentes de carrera (hay una rama conservada del 05/10, sin PR) | `internal/modules/personal` | F2-06 | - | M | Sin asignar | rama conservada | En curso | 05/10 | Periodos reconocidos con su fuente | Plan del módulo |

## 8. Fase 3. Cronos (por capacidad)

Cronos está aparcado (orden de módulos del 08/10 y plan de simplificación
del 16/10). #1142 conserva calendario, correcciones e
informes con política provisional (decisión del 15/10 a las 00:05).

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F3-01 | Cerrar #1142: montar las vistas nuevas, el saldo con permisos y C8. Aparcada con Cronos: se retoma después de las cuatro tareas de simplificación (F2-12 a F2-15) | `internal/modules/cronos`; `web/static` | F0-18 | - | M | C | #1142 | Aparcada | 19/10 | Recorrido en Chrome con reinicio | Plan del módulo; orden de módulos del 08/10; plan de simplificación del 16/10 |
| F3-02 | C3p: consulta de saldo propio acotada y medida | `internal/modules/cronos` | - | - | M | Sin asignar | - | Pendiente | 15/10 | Percentil 95 y número fijo de consultas | Plan del módulo |
| F3-03 | C5a y C5b: corrección de marcaje con decisión de jefatura y validación de RRHH | `internal/modules/cronos` | F0-18 | - | M | Sin asignar | - | Pendiente | 15/10 | Marcaje original intacto y recibos tras reiniciar | Plan del módulo |
| F3-04 | C4 y C3a: calendario oficial y cuadrante publicado | `internal/modules/cronos`; `internal/modules/calendarios` | F2-01 | Duda 21 (RRHH) | L | Sin asignar | - | Bloqueada | 15/10 | Un turno que cruza la medianoche se explica con su fuente | Plan del módulo |
| F3-05 | C6 y C7: permisos con doble resolución y aviso | `internal/modules/cronos` | F0-18 | Dudas 41 y 47 (RRHH) | L | Sin asignar | - | Bloqueada | 15/10 | Solicitud, resolución y aviso con recibo | Plan del módulo |
| F3-06 | C8: justificante con original custodiado (D21) | `internal/modules/cronos`; Documentos | F0-18 | - | M | Sin asignar | - | Pendiente | 15/10 | Documento ajeno o huella distinta, rechazados | Plan del módulo |
| F3-07 | C3b: saldo con jornada, asientos y permisos | `internal/modules/cronos` | F3-04, F3-05 | Dudas 100-103 (RRHH) | L | Sin asignar | - | Bloqueada | 15/10 | Saldo reproducible por versión | Plan del módulo |
| F3-08 | C10 y C11: incidencias, presencia y equipo | `internal/modules/cronos` | F3-02 | - | M | Sin asignar | - | Pendiente | 15/10 | Jefatura ve solo lo mínimo de su equipo | Plan del módulo |
| F3-09 | C12: cierre mensual e informes con permiso de exportación propio | `internal/modules/cronos` | F3-07 | Duda 118 (provisional en catálogo) | L | Sin asignar | - | Pendiente | 15/10 | Mes cerrado reproducible tras reiniciar | Plan del módulo |
| F3-10 | Fichaje web solo para teletrabajo autorizado | `internal/modules/cronos` | F3-02 | Duda 42 (RRHH) | M | Sin asignar | - | Bloqueada | 15/10 | Fichaje con la hora del servidor | Plan del módulo |

## 9. Fase 4. Dietas (por capacidad)

Dietas está aparcado por las mismas razones que Cronos. DIE-01 está en `main`
(#838).

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F4-01 | Cerrar #1136 y #1139: activación, bandejas, tarifas publicadas y liquidación en preparación. Aparcada con Dietas: se retoma después de las cuatro tareas de simplificación (F2-12 a F2-15) | `internal/modules/dietas`; `cmd/vec-dietas` | F0-18 | - | M | D | #1136, #1139 | Aparcada | 19/10 | CI, ensayo y revisiones | Plan del módulo; orden de módulos del 08/10; plan de simplificación del 16/10 |
| F4-02 | DIE-02: envío con justificantes por Documentos, con antivirus | `internal/modules/dietas`; Documentos | F1-07 (antes F0-10, cancelada porque la cubre F1-07), F0-18 | Dudas 24 y 50 (configurables) | M | Sin asignar | - | Pendiente | 15/10 | Envío con recibo; un fallo del antivirus lo deja pendiente | Plan del módulo |
| F4-03 | DIE-03: bandeja y decisión por perfil y unidad, con devolución | `internal/modules/dietas` | F4-02 | Dudas 25, 40, 46, 51 y 105 (RRHH) | M | Sin asignar | - | Bloqueada | 15/10 | Devolución y reenvío con historia | Plan del módulo |
| F4-04 | DIE-04: liquidación durable y PDF de la instantánea | `internal/modules/dietas` | F4-03 | Duda 104 (RRHH) | L | Sin asignar | - | Bloqueada | 15/10 | Liquidación con recibo y rectificación | Plan del módulo |
| F4-05 | DIE-05: fiscalización e informes con exportación separada | `internal/modules/dietas` | F4-04 | Dudas 108 y 118 (RRHH) | M | Sin asignar | - | Bloqueada | 15/10 | Informe paginado y exportación auditada | Plan del módulo |
| F4-06 | DIE-06: salida al sistema económico con acuse, sin doble pago | `internal/modules/dietas` | F4-04 | Duda 26 (Informática) | L | Sin asignar | - | Bloqueada | 15/10 | Reintento con la misma clave sin duplicar | Plan del módulo |
| F4-07 | DIE-07: recorrido completo con reinicio y revisiones | Recorrido | F4-05 | - | M | Sin asignar | - | Pendiente | 15/10 | Acta del recorrido | Plan del módulo |

## 10. Fase 5. Sustituir Convoca (por capacidad)

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F5-01 | Baremador común: editor de puntuaciones (topes, tablas, fecha de referencia) con versión y auditoría; trocear por tipo de regla | `internal/shared/baremacion`; `cmd/vec-baremador` | fase 1 | - | XL | Sin asignar | - | Pendiente | 15/10 | RRHH cambia un tope y queda una versión nueva | Plan del módulo |
| F5-02 | Simulación y conformidad con los casos del prototipo, con datos anonimizados de verdad | `internal/shared/baremacion` | F5-01, F5-13 | - | L | Sin asignar | - | Pendiente | 15/10 | Casos de conformidad en verde | Plan del módulo |
| F5-03 | Autobaremo explicable y resultado reproducible con desglose | `internal/shared/baremacion` | F5-01 | - | L | Sin asignar | - | Pendiente | 15/10 | Mismo resultado y huella al repetir | Plan del módulo |
| F5-04 | Revisión por mérito con firmantes designados, rectificación y nueva firma de todos tras un cambio (B7, B8) | `internal/shared/baremacion`; Firma | F5-03 | - | XL | Sin asignar | rama de firmantes B7 | Pendiente | 15/10 | Estados «preparado», «enviado a firma» y «firmado» | Plan del módulo |
| F5-05 | Acreditación de títulos: copia escaneada pendiente y cotejo único reutilizable | `internal/modules/meritos` | F1-07 (antes F0-10, cancelada porque la cubre F1-07) | Duda 160 | M | Sin asignar | - | Pendiente | 15/10 | Un cotejo hecho una vez sirve en dos bolsas | Plan del módulo |
| F5-06 | Recalcular la bolsa activa tras una decisión firmada, avisar y parar los llamamientos automáticos del tramo | `internal/modules/bolsa` | F5-04 | - | L | Sin asignar | - | Pendiente | 15/10 | Aviso a RRHH con las personas y los llamamientos afectados | Plan del módulo |
| F5-07 | Convocatoria gobernada y publicación con versiones | `internal/modules/seleccion` | - | Duda 109 (RRHH) | L | Sin asignar | - | Bloqueada | 15/10 | Convocatoria publicada y retirada sin borrar historia | Plan del módulo |
| F5-08 | Admisión, subsanación y listas provisional y definitiva | `internal/modules/seleccion` | F5-07 | Dudas 110, 139 y 159 (RRHH) | L | Sin asignar | - | Bloqueada | 15/10 | Exclusión motivada y subsanación con recibo | Plan del módulo |
| F5-09 | Traspaso del resultado a una bolsa constituida | `internal/modules/seleccion`; `internal/modules/bolsa` | F5-08 | Duda 111 (RRHH) | M | Sin asignar | - | Bloqueada | 15/10 | La lista aprobada crea la bolsa con su orden | Plan del módulo |
| F5-10 | Tasas: cálculo, exención y pasarela | `internal/modules/seleccion` | F5-07 | - | L | Sin asignar | - | Pendiente | 15/10 | Pago conciliado | Plan del módulo |
| F5-11 | Resto de Procesos selectivos: tribunal, pruebas y resultados (26 a 43 días) | `internal/modules/seleccion` | F5-08 | - | XL | Sin asignar | - | Pendiente | 15/10 | Criterio de la fase 5 | Plan del módulo |
| F5-12 | Provisión: concurso y libre designación (41 a 69 días) | `internal/modules/provision` | F5-04 | Duda 137 (RRHH) | XL | Sin asignar | #881 conservada | Bloqueada | 15/10 | Adjudicación sintética sin Convoca | Plan del módulo |
| F5-13 | Datos anonimizados para las pruebas (fechas desplazadas, cifras alteradas, identificadores inventados), fuera de Git | Fuera del repositorio | - | Consulta al DPD | S | Sin asignar | - | Bloqueada | 15/10 | Visto bueno del DPD | Plan del módulo |

## 11. Fase 6. Resto de módulos (días de un equipo según su plan)

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F6-01 | Formación (7 a 12) | `internal/modules/formacion` | F2-06 | Dudas 119 y 138 | XL | Sin asignar | #882 conservada | Aparcada | 15/10 | Criterio general de PLAN §5.1 | Plan del módulo |
| F6-02 | Carrera y Registro único de méritos (17 a 29) | `internal/modules/carrera` | F2-11 | Dudas 120 y 121 | XL | Sin asignar | - | Aparcada | 15/10 | Criterio general | Plan del módulo |
| F6-03 | Certificados y analítica (13 a 20) | `internal/modules/certificados` | F2-11 | Dudas 116 y 117 | XL | Sin asignar | - | Aparcada | 15/10 | Primer certificado emitido | Plan del módulo |
| F6-04 | Portal del empleado: «Mis trámites» completo (6 a 10) | `web/static/area-personal` | F3-01, F4-01 | - | XL | Sin asignar | - | Aparcada | 15/10 | Criterio general | Plan del módulo |
| F6-05 | Acción social (11 a 18) | Módulo nuevo | F2-06 | Duda 130 | XL | Sin asignar | - | Aparcada | 15/10 | Criterio general | Plan del módulo |
| F6-06 | Nóminas, primero consulta (10 a 16) | Módulo nuevo | F2-06 | Duda 129 | XL | Sin asignar | - | Aparcada | 15/10 | Criterio general | Plan del módulo |
| F6-07 | Prevención de riesgos laborales (11 a 17) | Módulo nuevo | F2-06 | Duda 131 | XL | Sin asignar | - | Aparcada | 15/10 | Criterio general | Plan del módulo |
| F6-08 | Igualdad (10 a 16) | Módulo nuevo | F2-06 | Duda 132 | XL | Sin asignar | - | Aparcada | 15/10 | Criterio general | Plan del módulo |
| F6-09 | Incompatibilidades (9 a 14) | Módulo nuevo | F2-06 | Duda 133 | XL | Sin asignar | - | Aparcada | 15/10 | Criterio general | Plan del módulo |
| F6-10 | Régimen disciplinario (10 a 16) | Módulo nuevo | F2-06 | Duda 134 | XL | Sin asignar | - | Aparcada | 15/10 | Criterio general | Plan del módulo |
| F6-11 | Relaciones sindicales (10 a 16) | Módulo nuevo | F2-06 | Duda 135 | XL | Sin asignar | - | Aparcada | 15/10 | Criterio general | Plan del módulo |
| F6-12 | Canal interno de información (7 a 11; enlace público, 2 a 3) | `web/static/canal-interno` | - | Duda 136 | XL | Sin asignar | rama sin integrar | Aparcada | 15/10 | Criterio general | Plan del módulo |
| F6-13 | Decidir el destino de #881 y #882 (se conservan; no se fusionan ahora) | `internal/modules/provision`; `internal/modules/formacion` | - | - | S | Dirección | #881, #882 | Aparcada | 15/10 | PR fusionadas o cerradas | Consenso de dirección 15/10 |

## 12. Fase 7. Uso real y preparación de producción

F7-01 a F7-16 no son programación: son decisiones, permisos y servicios de
fuera. F7-17 a F7-33 preparan la instalación de producción; cada una cita la
decisión escrita de la que sale. La exportación judicial de la auditoría ya
tiene tarea (F0-19) y las copias de seguridad también (F0-20).

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F7-01 | Evaluación de impacto en protección de datos | - | - | Duda 85 (DPD) | - | Dirección | - | Bloqueada | 15/10 | Informe aprobado | PLAN §5, fase 7 |
| F7-02 | Registro de actividades y textos de información | - | - | Dudas 86 y 87 (DPD) | - | Dirección | - | Bloqueada | 15/10 | Actividades publicadas | PLAN §5, fase 7 |
| F7-03 | Lista pública de cada bolsa y uso del contacto de Convoca | - | - | Dudas 17, 45 y 83 (DPD) | - | Dirección | - | Bloqueada | 15/10 | Decisión del DPD | PLAN §5, fase 7 |
| F7-04 | Actuaciones de aspirantes externos y su conservación | - | - | Duda 97 (DPD) | - | Dirección | - | Bloqueada | 15/10 | Decisión del DPD | PLAN §5, fase 7 |
| F7-05 | Categoría del Esquema Nacional de Seguridad y custodia de claves | - | - | Duda 89 (Informática) | - | Dirección | - | Bloqueada | 15/10 | Categorización aprobada | PLAN §5, fase 7 |
| F7-06 | Directorio corporativo y Kerberos | Identidad | - | Duda 30 (Informática) | - | Dirección | - | Bloqueada | 15/10 | El personal entra con su usuario | PLAN §5, fase 7 |
| F7-07 | Correo de la Diputación y encargados del tratamiento | Comunicaciones | F0-13 | Duda 88 (Informática) | - | Dirección | - | Bloqueada | 15/10 | Envío real auditado | PLAN §5, fase 7 |
| F7-08 | Portafirmas Firmadoc o AutoFirma | Firma | - | Duda 74 (Informática) | - | Dirección | - | Bloqueada | 15/10 | Envío real a firma | PLAN §5, fase 7 |
| F7-09 | GINPIX automático | `internal/modules/contrataciontemporal` | - | Duda 127 (Informática) | - | Dirección | - | Bloqueada | 15/10 | Envío sin grabación manual | PLAN §5, fase 7 |
| F7-10 | Retención de crédito en SICAL | `internal/modules/contrataciontemporal` | - | Duda 125 (Informática) | - | Dirección | - | Bloqueada | 15/10 | Consulta de la retención | PLAN §5, fase 7 |
| F7-11 | SMS o mensajería para los llamamientos | `internal/modules/bolsa` | - | Duda 152 (Informática) | - | Dirección | - | Bloqueada | 15/10 | Proveedor aprobado | PLAN §5, fase 7 |
| F7-12 | Delegaciones de firma y suplencias | Firma | - | Dudas 122, 128 y 143 (RRHH) | - | Dirección | - | Bloqueada | 15/10 | Circuito comunicado por Secretaría | PLAN §5, fase 7 |
| F7-13 | Clasificación según el Reglamento europeo de IA antes de usar IA en selección | - | - | - | - | Dirección | - | Pendiente | 15/10 | Clasificación aprobada | PLAN §5, fase 7 |
| F7-14 | Auditoría de seguridad con herramientas de ataque, con la aplicación terminada | Todo | fases 1 a 6 | - | - | Dirección | - | Pendiente | 15/10 | Informe sin hallazgos abiertos | PLAN §5, fase 7 |
| F7-15 | Instalar en cidonia lo ensayado en local. El 18/10 se subió el paquete de la presentación (`main@c01ff0589`), con vuelta atrás al paquete anterior. Desde entonces cidonia vuelve a estar congelada: el avance se mide en la copia local y la siguiente instalación irá al servidor nuevo (F7-22) | `deploy/principal` | fase 1 | - | - | Dirección | - | Aparcada | 18/10 | Subida con copia y vuelta atrás | Orden de dirección 15/10; subida del 18/10 |
| F7-16 | Que RRHH vea cada fase funcionando y la dé por buena | - | fase 1 | RRHH | - | Dirección | - | Bloqueada | 15/10 | Visto bueno de RRHH | PLAN §5 |
| F7-17 | Instalación de producción desde cero, reproducible en una base vacía: esquema, roles, catálogos y provisiones, sin copiar la base de cidonia | `deploy/postgresql`; `deploy/principal` | F2-14 | - | XL | Sin asignar | - | Pendiente | 19/10 | Dos instalaciones en bases vacías dan el mismo esquema, los mismos permisos y los mismos catálogos | TRD §15; orden de Alberto 12/10 |
| F7-18 | Separar los servidores de producción: `vec-interno`, `vec-publico` y `vec-admin`, con el área del aspirante aparte del portal interno, y retirar `vec-server` | `cmd/`; `deploy` | F7-17 | Falta decidir dónde va el área del aspirante | L | Sin asignar | - | Pendiente | 19/10 | Los tres arrancan por separado, un pico de aspirantes no frena a RRHH y `vec-server` ya no se instala | TRD §15; orden de Alberto 06/10 |
| F7-19 | Retirar el montaje de desarrollo: la contraseña web de Caddy, los certificados de desarrollo por perfil y los dos administradores inventados, que pasan a ser personas reales | Identidad; configuración | F7-06, F7-17 | - | M | Dirección | - | Pendiente | 19/10 | Producción solo admite la identidad corporativa y sus administradores son personas reales | TRD §5 y §15; decisión de identidad y sesión |
| F7-20 | Validador de GrxFirma para producción, desplegado aparte; ningún documento cuenta como firmado sin verificación positiva | Firma; puerto `VerificadorFirma` | F7-08 | - | M | Sin asignar | - | Pendiente | 19/10 | Una firma válida se verifica y una no válida se rechaza, con auditoría | TRD §9 y §16; decisión de firma con GrxFirma |
| F7-21 | Material propio de producción para CT: raíz y clave nuevas (lo que era la parte 2 de F1-64) y autoridad de producción para la rama nativa del llamamiento (F1-70) | `internal/modules/contrataciontemporal`; SQL de CT | F7-17 | - | M | Sin asignar | - | Pendiente | 19/10 | La instalación nueva emite con su propio material y la de desarrollo sigue igual | Decisiones de dirección 18/10 07:45 y 09:48 |
| F7-22 | Servidor nuevo con CPU dedicada y su plataforma: máquinas virtuales con Podman Quadlet o Compose, volumen de datos, tiempo máximo de recuperación y pérdida de datos admitida, e inventario de la Diputación | Fuera del repositorio | - | Sistemas | - | Dirección | - | Bloqueada | 19/10 | Servidor entregado y validado por Sistemas | TRD §17; instrucción de dirección sobre cidonia (16/10) |
| F7-23 | Prueba de carga a cientos y miles de usuarios, en local y con volumen real, antes de cada hito | `scripts/carga` | - | - | M | Sin asignar | - | Pendiente | 19/10 | Informe con el percentil 95 de cada lectura antes de cada hito; ninguna mejora quita permisos ni auditoría | TRD §10; PRD §11 |
| F7-24 | Administración solo desde la red corporativa en producción: los rangos de red van en la configuración y se aplican en Caddy y en la aplicación | `cmd/vec-admin`; configuración | F7-17 | Los rangos los da Informática | S | Sin asignar | - | Pendiente | 19/10 | Una petición de fuera de esos rangos se rechaza en Caddy y en la aplicación | PRD §9.6 |
| F7-25 | Dos tipos de administrador: Sistemas (instala, actualiza, hace copias; no ve datos personales ni reparte perfiles) y Aplicación (perfiles, cargos, catálogos, configuración, módulos y auditoría). Nadie se asigna a sí mismo; el primero de cada tipo se crea en la instalación, los siguientes necesitan dos personas y nunca puede quedar ninguno | `internal/modules/administracion` | F0-04, F7-17 | - | L | Sin asignar | - | Pendiente | 19/10 | Un administrador no puede asignarse a sí mismo, retirar al último se niega y cada tipo ve solo lo suyo | PRD §9.6 |
| F7-26 | Perfil auditor de solo lectura y consulta de quién tiene qué perfil | `internal/modules/administracion`; `web/static/admin` | F0-23 | - | M | Sin asignar | - | Pendiente | 19/10 | El auditor consulta asignaciones con su historia y no puede cambiar nada; cada consulta queda auditada | PRD §9.6 |
| F7-27 | Retirar un perfil o bloquear a una persona con efecto inmediato | Identidad; `internal/modules/administracion` | F0-04 | - | M | Sin asignar | - | Pendiente | 19/10 | Tras el bloqueo, la siguiente petición de esa persona se deniega aunque tenga sesión abierta | PRD §9.6 |
| F7-28 | Temas instalables desde Administración: instalar, consultar, aprobar, activar y retirar con versión esperada (cortes 3 y 4 de `docs/estudio_requisitos/temas_instalables_2026-10-04.md`) | `internal/modules/administracion/domain/temas`; `web/static/admin` | F0-04 | - | L | Sin asignar | - | Pendiente | 19/10 | Un tema se instala, se activa y se retira desde Administración con su versión y su auditoría; uno en uso solo se desactiva | DISENO_UI_UX, temas instalables |
| F7-29 | Manual de sistemas al día: `docs/manual_sistemas/README.md` (corte del 28/09) y `docs/manuales/sistemas.md`, con la instalación de producción, las copias, las migraciones y la operación | `docs/manual_sistemas`; `docs/manuales` | F7-17 | - | M | Sin asignar | - | Pendiente | 19/10 | Sistemas instala una copia siguiendo solo el manual | Regla de manuales 15/10 |
| F7-30 | Expediente electrónico: código seguro de verificación (CSV), sello de tiempo, registro electrónico de entrada, notificación electrónica, índice del expediente firmado y exportación según el Esquema Nacional de Interoperabilidad (ENI) | Documentos; puertos nuevos de registro y notificación | - | Sistemas de registro y notificación de la Diputación | XL | Sin asignar | - | Pendiente | 19/10 | Un expediente sintético sale con índice firmado, CSV comprobable y exportación ENI válida | TRD §9 y §17 |
| F7-31 | Accesibilidad: auditoría según las pautas WCAG y la norma EN 301 549, y declaración de accesibilidad publicada | `web/static`; documentos generados | fases 1 a 6 | - | - | Dirección | - | Pendiente | 19/10 | Informe de auditoría sin fallos abiertos y declaración publicada | TRD §17 |
| F7-32 | Plan de respuesta a incidentes de seguridad | - | F7-05 | - | - | Dirección | - | Pendiente | 19/10 | Plan aprobado y un simulacro hecho | TRD §17 |
| F7-33 | Análisis de riesgos del Esquema Nacional de Seguridad | - | F7-05 | - | - | Dirección | - | Pendiente | 19/10 | Análisis aprobado | TRD §17 |
