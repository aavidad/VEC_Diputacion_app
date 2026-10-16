# Tablero de tareas de VEC

Actualizado el 16 de octubre de 2026 sobre `origin/main@9f3b2d1e1`.

Este tablero dice qué falta, quién lo hace y en qué estado está. Sustituye
la lista de tareas que había en `PLAN_IMPLEMENTACION.md` §6; los criterios de
cierre de cada fase siguen en `PLAN_IMPLEMENTACION.md` §5. Parte del borrador
del 15/10 corregido con el consenso de dirección de ese mismo día. Todo se
trabaja en local: cidonia queda fuera hasta nueva orden, y los datos son
siempre inventados.

## 1. Cómo se usa

### Quién lo cambia

Solo dirección actualiza este tablero. Los equipos citan la tarea en la PR y
en el canal con la forma `Tarea: F1-21`.

### Identificadores

No se renumeran nunca. Una tarea que se abandona o se
funde con otra queda como «Cancelada», con la fecha y el motivo en la propia
fila. Las tareas nuevas toman el siguiente número libre de su fase.

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
| 0 Base común | 14 | 3 | 5 | 0 | 5 | 0 | 3 | 1 | 31 |
| 1 Bolsa y CT | 37 | 6 | 10 | 0 | 9 | 3 | 2 | 2 | 69 |
| 2 Personal y Organización | 10 | 3 | 1 | 0 | 0 | 0 | 0 | 1 | 15 |
| 3 Cronos | 5 | 4 | 1 | 0 | 0 | 0 | 0 | 0 | 10 |
| 4 Dietas | 2 | 4 | 1 | 0 | 0 | 0 | 0 | 0 | 7 |
| 5 Sustituir Convoca | 8 | 5 | 0 | 0 | 0 | 0 | 0 | 0 | 13 |
| 6 Resto de módulos | 0 | 0 | 0 | 0 | 0 | 0 | 13 | 0 | 13 |
| 7 Uso real | 2 | 13 | 0 | 0 | 0 | 0 | 1 | 0 | 16 |
| **Total** | **78** | **38** | **18** | **0** | **14** | **3** | **19** | **4** | **174** |

Empezables ya: 24 en las fases 0 y 1 (pendientes con todas sus
dependencias hechas o fusionadas). De ellas, 16 tienen equipo:

- T: F0-07, F1-39, F1-42, F1-44 y F1-47.
- H: F0-14, F0-15, F0-22, F0-23 y F0-24.
- O: F0-21 y F1-46.
- L: F0-29.
- R: F1-67 (por la lista del recuento; en realidad espera a AD288), F1-69
  y F1-70.

Sin equipo todavía: F0-11, F0-16, F0-17, F1-53, F1-56 (opcional), F1-60,
F1-62 y F1-63; tampoco F1-61, que espera a F1-40. F1-58 es de H y espera a
que termine F1-36; F0-31 es de L y espera a F0-28; F1-17 es de S y va tras
F1-18. F0-08 está bloqueada. Fuera de las fases 0 y 1 hay tres más sin dependencias
(F2-02, F3-02 y F7-13), pero la prioridad vigente no deja encargarlas antes de
cerrar la fase 1.

Cambios del 16/10 por la tarde: fusionadas F1-08 (#1144, #1170 y #1178),
F1-21 (#1153), F1-45 (#1183, solo documentación; la valoración espera al
baremador) y la corrección de la auditoría de la lectura de inscripción del
aspirante externo (#1188), que se anota como F1-68. F1-36 tiene fusionadas
las partes A (#1172) y B (#1173) y le falta C (#1179); F1-64 tiene fusionada
la parte 1 (#1181) y le falta la 2; F0-28 tiene fusionado su primer corte
(#1175). Pasan a en curso F1-09 (#1190), F1-12 (#1187), F1-18 (#1180 y
#1184) y F1-40 (#1177), que #1188 desbloqueó. Tareas nuevas pequeñas: F1-69
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

La puerta era F1-21 (separar la unidad gestora de CT de la unidad del
puesto), fusionada el 16/10 con #1153. F1-08 también quedó fusionada ese día;
sigue pendiente F1-22 (#1150) para el llamamiento del recorrido (F1-25).

Cadena más larga, por días laborables a partir del 16/10:

1. F1-22 (#1150) y F1-21, equipo R: hasta el 19/10.
2. F1-08, R1a (#1144): ensayo y fusión, 20/10.
3. F1-09, acto de comprobación del art. 9, apartado cuarto (L): del 21 al
   26/10. T hace la pantalla (F1-57) en paralelo.
4. F1-10, R1b en CT: 27 y 28/10.
5. F1-31, reinicio, repetición y acta: P la prepara desde el 20/10
   (F1-23 → F1-24 → F1-25 → F1-27 → F1-28 → F1-33) y la cierra cuando
   termine F1-10.

Con el reparto del apartado 4, la fase 1 se cierra hacia el 27-30/10.
Hay dos cadenas casi igual de largas: F1-07 → F1-16 → F1-19 → F1-20 → F1-31
(varias plazas hasta la propuesta) y F1-16 → F1-12 → F1-13 → F1-14 → F1-31
(modalidades en «Mi Bolsa»). Un retraso en #1147 o en #1146 las vuelve
críticas.

Trabajo que falta para cerrar la fase 1, contando sus tareas sin las hechas,
canceladas o bloqueadas por RRHH, más F0-01, F0-02, F0-03, F0-04, F0-06,
F0-21, F0-23, F0-24 y F0-25: 77 jornadas por tamaño (21 tareas S, 31 M y
5 L). Si se descuenta la mitad de las diez que ya están en curso y la tarea
opcional F1-56, quedan unas 68 jornadas. Es el cálculo del 15/10: no
descuenta F0-06, aparcada el 16/10, ni suma las tareas nuevas de ese día.

Las cargas no están igualadas: H (unas 16 jornadas) y O (unas 12) superan el
plazo del 30/10. Lo que la fase 1 necesita de ellos va primero (F0-03 y F0-02
de H; F1-07, F1-12, F1-13 y F1-14 de O); lo demás puede pasar de esa fecha
sin mover el cierre.

### Después de la fase 1: simplificar primero

La fase 2 empieza por las cuatro tareas de simplificación, F2-12 a F2-15
(permisos por datos, lecturas sin autorización de un solo uso, base SQL
consolidada y arranque que solo compone). Se hacen antes que el resto de
Personal, Cronos y Dietas.

### Orden de fusión de la cadena central de SQL

El orden es fijo; ninguna PR de esta cadena se fusiona antes que la anterior:

1. #1147 (AD277, AD281, AD284, AD287), F1-07, ya fusionada el 16/10
2. #1144 (AD276), F1-08, fusionada el 16/10
3. #1170 (CT230), fusionada el 16/10
4. #1178 (CT231, fila O6 de la ficha), fusionada el 16/10
5. #1146 (AD278), F1-16
6. #1155 (AD283), F1-04
7. AD279, F1-12
8. AD288 (equipo R, consumidor de F1-06)
9. AD273, F0-18

#1151 (AD282, F0-03) ya está fusionada y sale de la lista.

## 4. Reparto por equipo

Cada equipo trabaja su lista en este orden. Una flecha indica que la
siguiente no empieza hasta acabar la anterior.

| Equipo | Zona | Orden |
| --- | --- | --- |
| R | Criterios de Bolsa y puente con CT; dueño de las migraciones nuevas del núcleo de autorización | F1-22 → F1-21 → F1-08 → F1-09 → F1-10 → F1-11 |
| S | Expediente y oferta de CT; reglas de situación | F1-16 → F1-04 (con lo que era F1-05) → F1-06 → F1-18 → F1-17 → F1-55 |
| O | «Mi Bolsa», Documentos y preferencias | F1-07 → F1-12 → F1-13 → F1-14 → F0-21 → F1-46 → F1-29 → F1-45 |
| H | Administración, identidad y firma | F0-03 → F0-02 → F1-36 y F1-37 → F1-58 → F0-08 → F0-23 y F0-24 → F0-04 con F0-05 → F0-14, F0-15 y F0-22 → F1-64 y F1-65A |
| T | Pantallas y textos | F1-01, F1-02 y F1-03 → F1-44 → F1-42 y F1-47 → F1-39 → F1-54 → F1-57 → F0-07; parado por cuota desde el 16/10 a las 02:58, vuelve con F1-57 cuando R publique su contrato |
| P | Recorrido, desde el 20/10 | F1-23 → F1-24 a F1-30 → F1-32 y F1-33 → F1-34 → F1-31 |
| C y D | Cronos y Dietas, desde el 20/10 | Solo F0-18 con F3-01 (C) y F4-01, #1136 y #1139 (D) |
| B | Arranque y configuración | F0-27 → F0-09 → F1-40 (corte acordado el 16/10; el cierre de extremo a extremo espera a F1-61) |
| L | Limpieza y herramientas comunes, desde el 16/10 | F0-28 → F0-29 → F0-30 |
| Dirección | Documentación y coordinación | F0-25 → F0-26 → F6-13 |

F0-06 está aparcada desde el 16/10: el recorrido asigna por organización y
unidad gestora con la vía ordinaria (AUT44). Nadie asigna por SQL perfiles con
ámbitos de centro y categoría.

## 5. Fase 0. Base común

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F0-01 | Encender la auditoría de RRHH (`VEC_RRHH_AUDITORIA_ENABLED`) en la copia local, con su configuración | Configuración; `deploy/principal/auditoria_rrhh_definicion_rol.json` | - | - | S | B | #1162 | Fusionada | 16/10 | La consulta por persona devuelve sus lecturas y sus actos | PLAN §5, fase 0 |
| F0-02 | Retirar el compositor heredado de `vec-admin` o hacer que falle cerrado: exige el grupo `vec_admin_perfiles_auditoria_ejecutor` (`composicion.go:91`) y la función `registrar_denegacion_frontera_admin_v1`, que no existe. No se crea el grupo | `cmd/vec-admin/composicion.go` | - | - | M | H | #1158 | Fusionada | 16/10 | `vec-admin` arranca en una copia limpia o se niega con un motivo claro; nada llama a la función inexistente; dos revisiones sensibles | Consenso de dirección 15/10 |
| F0-03 | Vida de claves y confianza de gobierno configurable: 24 h por defecto y en producción, hasta 90 días en desarrollo; el arranque de CT adopta la vigente sin recortarla (AD282) | Administración; `internal/app/bootstrap` | - | - | M | H | #1151 | En curso | 15/10 | Arranque con y sin el valor; producción rechaza más de 24 h; desarrollo acepta hasta 90 días; el reinicio conserva la confianza | Consenso de dirección 15/10 |
| F0-04 | Pantalla de `vec-admin` para asignar y retirar un perfil desde la ficha del usuario (A8). Se hace junto con F0-05 | `cmd/vec-admin`; `web/static/administracion-perfiles` | F0-23, F0-24 | - | L | H | rama del candidato #733 | Pendiente | 15/10 | Formularios, confirmación y recibo en Chrome a 1440 y 390 px | PLAN §6 anterior al tablero |
| F0-05 | Recorrido real de asignar y retirar con dos administradores (A9). Su tamaño va dentro de F0-04 | `cmd/vec-admin` | F0-02, F0-04 | - | - | H | - | Pendiente | 15/10 | Asignar, retirar, reintento, conflicto y denegación auditados; mismo recibo tras reiniciar | PLAN §5, fase 0 |
| F0-06 | Vía gobernada para asignar los perfiles de ofertas de CT con ámbitos de centro y categoría. Aparcada el 16/10: #1146 (F1-16) pasa a organización y unidad gestora por la vía ordinaria (AUT44), así que el recorrido no necesita ámbitos de centro y categoría | `internal/modules/administracion` | F0-04 | - | M | H | - | Aparcada | 16/10 | Las tres asignaciones de #1146 se crean sin SQL a mano | Consenso de dirección 15/10 |
| F0-07 | Administración muestra nombres de persona y de unidad con lectura autorizada (hoy sale «Nombre de la unidad no consultado»; el vídeo 16 se retiró por eso) | `web/static/admin`; `internal/modules/administracion` | - | - | M | T | - | Pendiente | 15/10 | Lista y ficha con nombres; cada lectura auditada | Borrador 15/10 |
| F0-08 | Quitar del arranque las escrituras directas en tablas de autorización y la herencia de `vec_ad3_o207_gobierno` | `internal/app/bootstrap` | - | Espera un contrato de publicación nominal segregada del gobierno interno, trabajo de núcleo de R | M | H | - | Bloqueada | 16/10 | Arranque en una copia sin esas escrituras | PLAN §6 anterior al tablero |
| F0-09 | Acto de catálogo con las fuentes de política de cese, configuración, origen CT y contacto completo; versión del rol de RRHH con el especializador (#1100). Antes, comprobar si sigue haciendo falta | Catálogo de acciones; SQL de autorización | - | - | S | B | - | En curso | 16/10 | El rol publicado incluye esas acciones y el recorrido no da 403 | PLAN §6 anterior al tablero |
| F0-10 | Antivirus (ClamAV) y análisis de PDF con cuarentena antes de custodiar. Cancelada el 16/10: la cubre F1-07 (#1147), que ya acredita ClamAV real, analizador con TLS mutuo y cuarentena con admisión 201 | `cmd/vec-analizador`; `deploy/analizador` | - | - | M | O | #1147 | Cancelada | 16/10 | Ensayo con un clamd real; ningún fichero dudoso llega a custodia | PLAN §6 anterior al tablero |
| F0-11 | Documentos y S3 (D16 a D18): autorización nominal del aspirante externo y composición | Documentos; `deploy/almacen_s3_ceph` | - | - | M | Sin asignar | rama D16-D18 | Pendiente | 15/10 | El aspirante externo sube y descarga con su permiso | Borrador 15/10 |
| F0-12 | Cliente del gestor documental Mayan | Documentos | F0-10 | Sistemas: TLS y configuración | M | Sin asignar | rama Mayan | Bloqueada | 15/10 | Documento custodiado en Mayan con el TLS del destino | Borrador 15/10 |
| F0-13 | Correo de la Diputación configurable desde Administración (B5) | `internal/modules/administracion` | - | Duda 88 (Informática) | M | Sin asignar | - | Bloqueada | 15/10 | Envío de prueba auditado desde Administración | PLAN §6 anterior al tablero |
| F0-14 | Auditoría, corte A: denegaciones y errores de las lecturas de contactos de Bolsa | `internal/modules/bolsa`; auditoría común | - | - | S | H | - | Pendiente | 15/10 | Una lectura denegada deja apunte en la cadena común | Borrador 15/10 |
| F0-15 | Auditoría, corte B: la descarga de un borrador de CT registra documento, formato y fallo | `internal/modules/contrataciontemporal`; auditoría común | - | - | M | H | - | Pendiente | 15/10 | La auditoría dice qué documento se bajó y en qué formato | Borrador 15/10 |
| F0-16 | Interruptor de módulos en Administración, persistente y auditado | `internal/modules/administracion`; `web/static/admin` | - | - | L | Sin asignar | - | Pendiente | 15/10 | Apagar un módulo cierra su menú y sus rutas | PRD §8 |
| F0-17 | Selector de perfil activo en el portal interno | Identidad; `cmd/vec-interno` | - | - | M | Sin asignar | - | Pendiente | 15/10 | Una persona con dos perfiles elige uno sin cambiar de usuario | PRD §8 |
| F0-18 | Consolidada AD273 del núcleo de autorización: consume AD277 y AD279, añade el original de Cronos (D21), la autoridad documental de Dietas y la lectura propia de Personal para Cronos y Dietas (incluye lo que era F2-10, aprobado el 15/10). Va la última de la cadena SQL. Además: subir a garantía alta y método certificado las ramas de RRHH de AD277:80 y AD281:2744; no recrear los objetos de los comprobantes de Méritos que ya crean AD277, AUT95, D20 y M7; retirar del consumidor central la rama `documentos_bolsa_externa`, que no se alcanza | SQL del núcleo de autorización | F1-07, F1-08, F1-12, F1-16 | - | L | C y D | #1142 | En curso | 14/10 | Ensayo en copia y dos revisiones sensibles; Dietas lee lo suyo sin la consulta de RRHH | Consenso de dirección 15/10 |
| F0-19 | Exportación judicial de la auditoría (rol de Aplicación v7, fachada de captura, emisor) | Auditoría; `cmd/vec-auditoria-verificar` | - | - | L | Sin asignar | rama `codexl-auditoria-nominal` | Aparcada | 15/10 | Paquete firmado verificable y exportación auditada | Borrador 15/10 |
| F0-20 | Copias de seguridad y restauración desde Administración (14 a 25 jornadas) | `cmd/vec-admin-copias` y `cmd/vec-copias-*` | - | Dudas 112-115 | XL | Sin asignar | - | Aparcada | 15/10 | Restauración con doble control en una base desechable | PRD §8 |
| F0-21 | Productor gobernado de admisiones de identidad externa: función SECURITY DEFINER solo para el grupo ADMIN, que exige una aprobación existente, audita en la misma transacción y permite la retirada | Identidad; SQL de `internal/modules/aspirantes` | - | - | M | O | - | Pendiente | 15/10 | Admisión con aprobación previa da recibo; sin aprobación se niega; retirada auditada; ensayo y revisión SQL independiente. Duda del contrato: qué aprobación de ADMIN aplica (`aprobacion_alta` solo cubre cuentas) y retirada prospectiva con recibo; O propone reutilizar el acto y la pantalla de Administración. Consultar a Fable y Astra antes | Consenso de dirección 15/10 |
| F0-22 | Auditoría, corte C: la consulta de datos de contacto deja apunte | `internal/modules/bolsa`; auditoría común | - | - | S | H | - | Pendiente | 15/10 | Cada consulta de contacto registra quién, de qué persona y con qué motivo | Consenso de dirección 15/10 |
| F0-23 | Lecturas A6 de Administración, que necesita la pantalla de F0-04 | `cmd/vec-admin`; `internal/modules/administracion` | - | - | M | H | - | Pendiente | 15/10 | La ficha del usuario lee sus perfiles con lectura autorizada y auditada | Consenso de dirección 15/10 |
| F0-24 | Deuda P2 de la revisión de A7, antes de F0-04 | `cmd/vec-admin` | - | - | S | H | - | Pendiente | 15/10 | Pruebas focales de cada punto P2 | Consenso de dirección 15/10 |
| F0-25 | `deploy/principal/LEEME_copia_local.md`: receta de la copia local (material ADMIN, datos de prueba, admisiones, OH y unidad del puesto) | `deploy/principal` | - | - | S | Dirección | #1156 y #1161 | Fusionada | 16/10 | Otro equipo levanta una copia local siguiendo solo esa receta | Consenso de dirección 15/10 |
| F0-26 | Limpieza documental: manual de administración y manifiesto de Bolsa al día | `docs/`; manifiesto del módulo Bolsa | - | - | S | Dirección | #1156 | Fusionada | 16/10 | Ningún paso del manual ni del manifiesto contradice `main` | Consenso de dirección 15/10 |
| F0-27 | Arranque sin Contratación temporal configurada: la app arranca y la función falla con error catalogado | `internal/app/bootstrap` | - | - | S | B | #1160 | Fusionada | 16/10 | Arranque con y sin la configuración de CT; sin ella, el resto funciona y CT responde con el error catalogado | Consenso de dirección 16/10 |
| F0-28 | Copia local en menos de 15 minutos con un guion único, sin pasos a mano; sustituye las recetas de cada equipo | `deploy/principal`; `scripts/` | F0-25 | - | M | L | #1175 (primer corte fusionado: guion de restauración y reinicio) | En curso | 16/10 | Otro equipo levanta la copia desde cero con el guion en menos de 15 minutos | Consenso de dirección con Fable y Astra, 16/10 |
| F0-29 | CI que instala desde cero todas las migraciones del recorrido de la fase 1 y ejecuta sus pruebas SQL, incluidas denegaciones. Incluye arreglar GoSec en `internal/app/bootstrap` (R aisló la causa: cargar el directorio, no ficheros sueltos) | `.github/workflows/ci.yml`; `deploy/postgresql` | - | - | M | L | - | Pendiente | 16/10 | La CI falla si una migración del recorrido no instala o una prueba SQL, también de denegación, no pasa | Consenso de dirección con Fable y Astra, 16/10 |
| F0-30 | Retirada continua de código muerto, empezando por `accionesDeltaVersionarBolsa`; nada de SQL instalado. T paró por cuota; sigue L | Todo el repositorio | - | - | M | L | #1166 (#1166, #1167 y #1168 fusionadas) | En curso | 16/10 | Cada retirada dice qué protección daba y qué prueba la sigue cubriendo; puertas de la CI verdes | Consenso de dirección con Fable y Astra, 16/10 |
| F0-31 | El guion de la copia local crea sin LOGIN los dos roles que necesita B119 (`vec_externo_documentos_bolsa_datos` y `vec_interno_documentos_bolsa_rrhh`); hoy se crean a mano, fuera de Git | `deploy/principal`; `scripts/` | F0-28 | - | S | L | - | Pendiente | 16/10 | Una copia levantada solo con el guion instala B119 sin crear roles a mano | Revisión de #1188, 16/10 |

## 6. Fase 1. Bolsa y Contratación temporal

Criterio de cierre (PLAN §5, adaptado a «todo en local»): recorrido del caso
4.0 con reinicio y acta, selectores encendidos, ningún «no disponible» sin
motivo y lecturas por debajo de 300 ms.

### 6.1 Lo que pidió RRHH el 15/10

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F1-01 | Situaciones en singular, sacadas del catálogo, en etiquetas, filtros y desplegables | `web/static/bolsa`; `locales` | - | - | S | T | #1148 | Fusionada | 16/10 | Chrome a 1440 y 390 px en castellano e inglés; revisión de usabilidad; pruebas de textos accesibles al día | RRHH 15/10 |
| F1-02 | «Cambiar situación» completo, con el motivo dentro del panel | `web/static/bolsa` | - | - | S | T | #1149 | Fusionada | 16/10 | Chrome a 1440, 1024 y 390 px; revisión de usabilidad | RRHH 15/10 |
| F1-03 | Cuadro de Bolsas: la columna «En revisión» se ve a 1440 px | `web/static/bolsa` | - | - | S | T | #1152 | En curso | 15/10 | Chrome a 1440 y 390 px sin desbordamiento | RRHH 15/10 |
| F1-04 | Publicar reglas de situación: fachada sobre B32 con permiso propio, versión esperada, motivo, recibo y autorización en la misma transacción; el arranque solo lee. Incluye la activación explícita de la fuente normativa (antes F1-05) y, para los candidatos «En revisión», confirmar la respuesta o devolverlos al turno con motivo | `internal/modules/bolsa`; SQL de Bolsa | - | - | M | S | #1155 | En curso | 15/10 | Publicación con recibo y replay igual; sin activación se niega con motivo; el arranque no sobrescribe; un candidato «En revisión» se confirma o vuelve al turno con su motivo | RRHH 15/10; duda 157 |
| F1-05 | Activar la fuente normativa del catálogo de situaciones. Cancelada el 15/10: fundida en F1-04, misma PR de S | - | - | - | - | S | - | Cancelada | 15/10 | - | Consenso de dirección 15/10 |
| F1-06 | Pantalla de reglas de situación para RRHH: ver y editar la política de transiciones con versiones y motivo, y resolver los candidatos «En revisión» | `web/static/bolsa` | F1-04 | - | M | S | - | Pendiente | 15/10 | Carmen la abre sin 403, cambia una regla, devuelve un candidato al turno y ve el recibo; tras reiniciar sigue igual | RRHH 15/10; duda 157 |
| F1-07 | Justificante PDF de la renuncia justificada: lo sube el aspirante y RRHH lo descarga en la ficha según su rol (AD277, D22, B119, CT222). Primera de la cadena SQL | `internal/modules/bolsa`; Documentos | - | - | M | O | #1147 | Fusionada | 16/10 | PDF subido y descargado en Chrome; más de 16 MiB da 413 avisado antes; reinicio | RRHH 15/10 |
| F1-08 | R1a: regla publicada de criterios (art. 9, apartado cuarto, y 11.2), motivos de exclusión visibles con lectura propia auditada (AD276) y parada «pendiente de comprobación por RRHH» | `internal/modules/bolsa`; `internal/modules/contrataciontemporal` | F1-21, F1-22 | Dudas 163 y 165 (configurables, no bloquean) | M | R | #1144, #1170 y #1178 | Fusionada | 16/10 | La selección nominal se detiene con el motivo; tabla de motivos con nombres; una bolsa sin política sigue el orden de siempre | RRHH 15/10 |
| F1-09 | R1b, parte de servidor: acto de Bolsa que acredita los hechos del art. 9, apartado cuarto (evidencia, resultado calculado, versión esperada, recibo). La pantalla va en F1-57 | `internal/modules/bolsa`; SQL de Bolsa | F1-07, F1-08 | Dudas 163 y 165 (configurables) | L | R | #1190 | En curso | 16/10 | RRHH resuelve un caso pendiente por la API y la selección sigue; reintento con el mismo recibo | RRHH 15/10 |
| F1-10 | R1b en CT: confirmar contexto y reservar con ese acto; descontar solo el tiempo desempeñado; resolución detenida hasta el cese efectivo | `internal/modules/contrataciontemporal` | F1-09 | Duda 165 (configurable) | M | R | - | Pendiente | 15/10 | Un caso con cese futuro queda detenido con el motivo a la vista | RRHH 15/10 |
| F1-11 | Modalidad y causa gobernadas en los datos de la oferta y en su canon de publicación (pacto entre R y S) | `internal/modules/bolsa` | F1-08, F1-16 | - | M | R | - | Pendiente | 15/10 | Una oferta publicada lleva modalidad y causa con fuente y versión | RRHH 15/10 |
| F1-12 | Preferencias internas de «Mi Bolsa»: migración propia (AD279) con consumidor y control de auditoría sacados de AD273, y montaje | `internal/modules/bolsa`; `web/static/area-personal` | F1-16 | - | M | O | #1187 | En curso | 16/10 | GET y PUT con recibo; reinicio en copia | RRHH 15/10 |
| F1-13 | R1c, servidor: preferencias de modalidad con selector `?tipo=modalidades`, versión, recibo y permiso propios; solo filtran la vista y los avisos de ofertas publicadas | `internal/modules/bolsa` | F1-11, F1-12 | Duda 164 (configurable) | M | O | - | Pendiente | 15/10 | Prueba de que la preferencia no excluye ni cuenta como renuncia | RRHH 15/10 |
| F1-14 | R1c, pantalla: elegir modalidades en «Mi Bolsa», aviso si la lista queda vacía y «Ver todas» sin guardar | `web/static/area-personal` | F1-13 | - | M | O | - | Pendiente | 15/10 | Chrome a 1440 y 390 px; reinicio | RRHH 15/10 |
| F1-15 | Publicación de ofertas con plazo desde la publicación y recuperación del recibo (B117) | `internal/modules/bolsa` | - | - | - | S | #1145 | Hecha | 15/10 | Dos plazas, aceptación en «Mi Bolsa» y reinicio en la copia de S | RRHH 15/10 |
| F1-16 | Publicar la oferta de varias plazas desde la ficha: reserva en CT, vínculo, cierre de intención y descarte (AD278, B120, B121, CT220), guarda de cancelación (CT223) y recuperación sin la clave del navegador | `internal/modules/contrataciontemporal`; `internal/modules/bolsa` | F1-07, F1-21, F1-22 | - | L | S | #1146 | En curso | 15/10 | Ensayo en copia: publicar, reintento, recarga tras respuesta incierta; cancelar con oferta en preparación da 409 | RRHH 15/10 |
| F1-17 | Retirar la oferta publicada al cancelar el expediente: Bolsa consume `ct.expediente_cancelado.v1` con historia y no adjudica plazas de una oferta retirada | `internal/modules/bolsa` | F1-16, F1-18 | - | M | S | - | Pendiente | 16/10 | Cancelar retira la oferta una sola vez, también si el evento se repite | RRHH 15/10 |
| F1-18 | Aviso de la oferta por correo como intención durable por el puerto de canales; no abre el plazo | `internal/modules/bolsa` | F1-16 | Correo real: duda 88 (vale el buzón de pruebas) | M | S | #1180 y #1184 | En curso | 16/10 | El aviso queda pendiente y se envía tras reiniciar, sin duplicar | RRHH 15/10 |
| F1-19 | Propuesta de nombramiento desde una oferta adjudicada | `internal/modules/contrataciontemporal` | F1-16 | - | L | Sin asignar | - | Pendiente | 15/10 | Tras adjudicar dos plazas, propuesta con recibo para cada persona | RRHH 15/10 |
| F1-20 | Variante de dos personas del caso 4.0 recorrida hasta la propuesta en la copia local | Recorrido | F1-19, F1-25 | - | S | Sin asignar | - | Pendiente | 15/10 | Dos aceptaciones, adjudicación por posición y propuestas; anotada en FLUJO 4.0 | FLUJO 4.0 |

### 6.2 Recorrido del caso 4.0 en copia local

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F1-21 | Separar la unidad gestora de CT de la unidad orgánica del puesto en el puente de la bolsa constituida. Es la puerta del camino crítico | `internal/app/bootstrap/contratacion_temporal_llamamiento_datos_constituidos.go:255` | - | - | M | R | #1153 | Fusionada | 16/10 | La selección del puesto 9/01 (unidad del centro 520) funciona con la unidad gestora de RRHH | Canal 15/10 |
| F1-22 | Organización histórica en `vec-server` combinado, con su rol de lectura de seis campos | `internal/modules/personal`; organización | - | - | M | R | #1150 | En curso | 15/10 | GET 200 con la sesión de RRHH; sin configuración solo esa función da 503 | Canal 15/10 |
| F1-23 | Retomar el ensayo de P: copia local al día con `main` (firma, ofertas), lista SQL causal, administradores, firmantes y empleados de copia | `deploy/principal/candidata_hz39` | F0-03 (o renovar el material cada día) | Equipo P en pausa hasta el 20/10 | M | P | #1128 | Bloqueada | 15/10 | La copia arranca con todo lo fusionado y las ocho variables de la fase | Consenso de dirección 15/10 |
| F1-24 | Pasos 1 a 8 (de la petición a la fiscalización, con las dos firmas del informe en 7b) | Recorrido | F1-23 | - | S | P | - | Pendiente | 15/10 | Recibo y versión de cada paso en Chrome | FLUJO 4.0 |
| F1-25 | Pasos 9 a 11: llamamiento desde la bolsa constituida, aceptación en «Mi Bolsa» y resolución | Recorrido | F1-21, F1-22, F1-24 | - | S | P | - | Pendiente | 15/10 | Recibo del llamamiento a Antonio y de su aceptación | FLUJO 4.0 |
| F1-26 | Fiscalización con reparo y subsanación: montar la subsanación del reparo y la ruta de Plantillas, y recorrerlas | `internal/modules/contrataciontemporal`; Plantillas | F1-24 | - | M | P | - | Pendiente | 15/10 | Reparo, informe nuevo y fiscalización favorable con su recibo; la plantilla se abre desde su ruta | Consenso de dirección 15/10 |
| F1-27 | Pasos 12 a 15: propuesta, resolución, documentos y tres firmas de la resolución en el mismo expediente | Recorrido | F1-25 | - | S | P | - | Pendiente | 15/10 | PDF verificados y descargables desde la ficha | FLUJO 4.0 |
| F1-28 | Pasos 17 a 21: incorporación acreditada, alta en Personal, GINPIX manual, cese, vuelta a la bolsa y cierre | Recorrido | F1-27 | - | S | P | - | Pendiente | 15/10 | Antonio vuelve a «Disponible» o «Disponible desde» | FLUJO 4.0 |
| F1-29 | Pasos e1 a e5 del aspirante con «Mi Bolsa» interna, encendiendo la pestaña de inscripción (hoy oculta en `vistas.json`) | `web/static/area-personal/vistas.json` | F1-23 | - | M | O | - (#1140 rechazada; no hay piezas en curso) | Pendiente | 15/10 | Inscripción, méritos, admisión, incorporación y alegación con recibos | FLUJO 4.0 |
| F1-30 | Cese sin candidato vinculado sin bloquear el relevo (#774 en `main`) | Recorrido | F1-23 | - | S | P | - | Pendiente | 15/10 | Alta, cursor, vínculo posterior y estado pendiente en «Mi Bolsa» | FLUJO 4.0 |
| F1-31 | Reiniciar aplicación y PostgreSQL, repetir las consultas y escribir el acta (respuesta HTTP y captura por paso), con las variantes de dos plazas y de vacante | `docs/estudio_requisitos/` | F1-06, F1-10, F1-14, F1-20, F1-26, F1-28, F1-32, F1-33, F1-55 | - | L | P | - | Pendiente | 15/10 | Mismos recibos y fechas, ningún duplicado; acta en `docs/estudio_requisitos/` | PLAN §5, fase 1 |

### 6.3 Resto de condiciones de cierre

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F1-32 | Las ocho variables de la fase encendidas en la copia, incluida la verificación de firma; el aviso de dependencias no muestra ausencias | Configuración de la copia | F0-01, F1-23, F0-03 (o renovar el material cada día) | - | S | P | - | Pendiente | 15/10 | Registro de arranque limpio | PLAN §5, fase 1 |
| F1-33 | Ningún «no disponible» sin motivo y ninguna respuesta 404 o 503 de Bolsa o CT durante el recorrido | `internal/modules/bolsa`; `internal/modules/contrataciontemporal` | F1-28 | - | M | P | - | Pendiente | 15/10 | Registro del recorrido sin esos códigos | PLAN §5, fase 1 |
| F1-34 | Lecturas en el percentil 95 por debajo de 300 ms (cuadro de CT, lista, ficha, bolsas, candidatos y ficha del candidato) y base por debajo de 100 ms, medidas en copia local con el volumen de la principal | `internal/modules/contrataciontemporal`; `internal/modules/bolsa` | F1-23 | - | M | P | - | Pendiente | 15/10 | Medición HTTP con su acta de rendimiento | PLAN §5, fase 1 |

### 6.4 Firma, después de #1137

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F1-35 | Cinco firmas reales, PDF desde la ficha tras reiniciar, plantillas v4 y detector de datos personales que respeta las huellas | Firma | - | - | - | H | #1137 | Hecha | 15/10 | Cinco firmas en la copia de H y PDF desde la ficha tras reiniciar | Canal 15/10 |
| F1-36 | Recuperación histórica de firmas con unidad nula y AD280 (fuera del recorte de CT224). En tres partes: A recupera las firmas originales con cargo y unidad vigentes y B restringe los selectores a la unidad autorizada, las dos fusionadas; falta C, consultar las firmas anteriores desde la ficha | Firma; SQL de firma | - | - | M | H | #1172 y #1173 fusionadas; C en #1179 | En curso | 16/10 | Consulta por unidad de una firma anterior al plan | Revisión de #1137 |
| F1-37 | Menores de la revisión de #1137: CT218 por organización, límite del PDF firmado (1 y 2 MiB), `nil` en `cerrarAuditoria` | Firma | - | - | S | H | #1163 | Fusionada | 16/10 | Pruebas focales | Revisión de #1137 |

### 6.5 Resto de la fase 1

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F1-38 | Manual de la jefatura de RRHH con el apartado 7 «Hoy, Falta, Al final» | `docs/` | - | - | - | Dirección | #1143 | Hecha | 15/10 | Es un documento: basta la fusión | Canal 15/10 |
| F1-39 | Textos: cómo corregir o publicar la categoría antes de vincularla | `web/static`; `locales` | - | - | S | T | #1008 (rama conservada) | Pendiente | 15/10 | Revisión de usabilidad y fusión | PLAN §6 anterior al tablero |
| F1-40 | Inscripción: regla por convocatoria para elegir solicitud separada o conjunta | `internal/modules/bolsa` | - | - | M | B | #1177 (desbloqueada por #1188) | En curso | 16/10 | RRHH la fija antes de publicar; queda en la versión de las bases | PLAN §6 anterior al tablero |
| F1-41 | Solicitud conjunta con su recibo y reintento (B105 solo cubre la separada; no instalar B105 sola) | `internal/modules/bolsa` | F1-40 | - | M | Sin asignar | rama B105 | Pendiente | 15/10 | Las dos modalidades con recibo y reintento | PLAN §6 anterior al tablero |
| F1-42 | Documentos del expediente en un solo sitio de la ficha: comprobar el commit local `046a2f2fa` | `web/static`; `internal/modules/contrataciontemporal` | - | - | S | T | commit local de S | Pendiente | 15/10 | Todos se ven y se descargan desde la ficha | PLAN §6 anterior al tablero |
| F1-43 | Tarjetas de plazo en Inicio (CT192): comprobar la rama `codexs-ct-filtros-plazos-20261008`. Aparcada el 16/10: ampliar el tipo V1 y reconstruir 16 funciones va contra las reglas de simplificación; espera a la tarea de lecturas de la fase 2 (F2-13) | `web/static` | - | - | S | T | rama sin PR | Aparcada | 16/10 | Cada tarjeta abre su lista filtrada | Borrador 15/10 |
| F1-44 | Retirar la duda 162 citando la norma y corregir FLUJO 4.2, que aún dice puesto en texto libre (#1114 ya lo resolvió) | `dudas.md`; `docs/producto/FLUJO_APP.md` | - | - | S | T | - | Pendiente | 15/10 | `dudas.md` sin la 162 y FLUJO 4.2 al día | Consenso de dirección 15/10 |
| F1-45 | Vista de la cartera de méritos en «Mi Bolsa» con la API real, sin cifras inventadas | `web/static/area-personal` | - | - | M | O | #1183 | Fusionada | 16/10 | La aspirante ve sus méritos por categoría. Se cierra con los hechos de la cartera, sin puntos; la valoración espera al motor común de baremación (fuera de la fase 1). #1183 solo cambia documentación | PLAN §6 anterior al tablero |
| F1-46 | Cuenta externa propia del aspirante: confianza externa nominal (HZ38, opción B) y motivo documental en la autoridad externa | Identidad; `internal/modules/aspirantes` | - | - | M | O | - (#1140 rechazada) | Pendiente | 15/10 | Entrada externa a «Mi Bolsa» con 200 y reinicio. Posible cierre por #1147; dirección lo comprueba | PLAN §6 anterior al tablero |
| F1-47 | Pantalla de no incorporación: comprobar si funciona y recorrerla (el vídeo 15 se omitió porque no estaba acreditada) | `web/static`; `internal/modules/contrataciontemporal` | - | - | S | T | - | Pendiente | 15/10 | Recorrido con recibo | Borrador 15/10 |
| F1-48 | Petición del centro firmada por el técnico y por el diputado o la diputada | `internal/modules/contrataciontemporal`; Firma | - | Decisión de RRHH: dudas 166-170 (propuestas) | L | Sin asignar | - | Bloqueada | 15/10 | Petición con dos firmas verificadas | Consenso de dirección 15/10 |
| F1-49 | Orden configurable del ofrecimiento antes o después de Intervención | `internal/modules/contrataciontemporal` | - | Decisión de RRHH: dudas 166-170 (propuestas) | L | Sin asignar | - | Bloqueada | 15/10 | Cambio por catálogo sin tocar código | Consenso de dirección 15/10 |
| F1-50 | Bandeja propia de Intervención | `internal/modules/contrataciontemporal`; `web/static` | - | Decisión de RRHH: dudas 166-170 (propuestas) | M | Sin asignar | - | Bloqueada | 15/10 | RRHH decide si hace falta; si sí, Intervención ve solo lo suyo | Consenso de dirección 15/10 |
| F1-51 | Plantillas definitivas de contratos, informes y resoluciones | Plantillas de CT | - | Duda 124 (RRHH) | S | Sin asignar | - | Bloqueada | 15/10 | Documentos con las plantillas oficiales | PLAN §6 anterior al tablero |
| F1-52 | Oferta al SAE cuando la bolsa se agota | `internal/modules/contrataciontemporal` | - | Dudas 72, 73 y 126 (RRHH) | L | Sin asignar | - | Bloqueada | 15/10 | Oferta con sus documentos | Borrador 15/10 |
| F1-53 | Momento de la toma de posesión y fecha de efectos del cese como valor definitivo (hoy funciona con valor de catálogo) | Catálogo de CT; Personal | - | Duda 140 (no impide funcionar) | S | Sin asignar | - | Pendiente | 15/10 | Valor publicado por RRHH | Borrador 15/10 |
| F1-54 | Corregir el catálogo de plazos de CT citando la norma: c04 con el art. 12.4 del RD 424/2017, y c03 | `cmd/vec-preparar-catalogo-reglas-ct`; catálogo de plazos | - | - | S | T | #1157 | Fusionada | 16/10 | Cada plazo del catálogo lleva su cita; c03 y c04 coinciden con la norma | Consenso de dirección 15/10 |
| F1-55 | Ensayo de varias plazas con «sin respuesta», retirada por cancelación y recuperación tras un fallo | `internal/modules/bolsa`; `internal/modules/contrataciontemporal` | F1-16, F1-17 | - | M | S | - | Pendiente | 15/10 | Una plaza sin respuesta pasa al siguiente; cancelar retira la oferta; tras el fallo, mismo recibo y sin duplicados | Consenso de dirección 15/10 |
| F1-56 | Correspondencia verificada entre la persona y su fila de Convoca (opcional) | Carga de Convoca en Bolsa | - | - | M | Sin asignar | - | Pendiente | 15/10 | Cada persona cargada enlaza con una sola fila de Convoca, comprobada | Consenso de dirección 15/10 |
| F1-57 | R1b, pantalla: RRHH resuelve el caso «pendiente de comprobación» desde la ficha del candidato | `web/static/bolsa` | F1-09 | - | M | T | - | Pendiente | 15/10 | Chrome a 1440 y 390 px: RRHH resuelve, ve el recibo y la selección sigue | Consenso de dirección 15/10 |
| F1-58 | Firma de F1-36 por la vía de cargos de firma: AUT98 mínima, operación anexada, plan v2, sucesión de asignaciones y lector CT22x que acepte v2 | Firma; SQL de firma | F1-36 | - | M | H | - | Pendiente | 16/10 | Firma por cargo con plan v2 leída por CT tras reiniciar; ensayo SQL y dos revisiones | Consenso de dirección 16/10 |
| F1-59 | Vínculo entre la respuesta publicada y su oferta, para abrir la resolución de la respuesta publicada en F1-06 | `internal/modules/bolsa`; `internal/modules/contrataciontemporal` | F1-06 | - | S | S | - | Cancelada | 16/10 | Desde F1-06 se abre la resolución de una respuesta publicada con su oferta. Cancelada el 16/10: en las ofertas publicadas solo existe Aceptar (disposiciones) y la adjudicación va por orden; no hay respuesta en revisión (FLUJO §4.1). S comprueba que disposiciones conserva `oferta_ref` | Consenso de dirección 16/10 |
| F1-60 | La bolsa guarda su organización y su unidad gestora como dato gobernado de su constitución, para que los actos de Bolsa (B100, B17, B128) cotejen el ámbito del recurso con la asignación del actor | `internal/modules/bolsa`; SQL de `bolsa_llamamientos` | - | - | M | Sin asignar | - | Pendiente | 16/10 | Un acto sobre una bolsa de otra unidad gestora se deniega aunque el actor tenga permiso en la suya; ensayo y revisión SQL independiente | Consenso de dirección con Fable y Astra, 16/10 |
| F1-61 | Pieza real que confirma la publicación de las bases de una convocatoria (`ConfirmarPublicacion` sin adaptador productivo), para cerrar F1-40 de extremo a extremo | `internal/modules/bolsa` | F1-40 | - | M | Sin asignar | - | Pendiente | 16/10 | RRHH elige la modalidad, publica las bases y la inscripción la consume, tras reiniciar | Consenso de dirección con Fable y Astra, 16/10 |
| F1-62 | Productor administrativo gobernado para publicar catálogos configurables CC1; hoy solo se cargan en la copia con `generar_carga_cc1.py` | `internal/modules/administracion`; SQL de catálogos | - | - | M | Sin asignar | - | Pendiente | 16/10 | Un catálogo CC1 se publica con dos aprobaciones y su recibo, sin SQL a mano, y sigue tras reiniciar | Consenso de dirección 16/10 |
| F1-63 | Normalizar la organización legada de las firmas (CA45, AUT37, Personal31) en el lote ordinario de Administración | Firma; `internal/modules/administracion` | - | - | M | Sin asignar | - | Pendiente | 16/10 | Las firmas anteriores quedan con su organización por la vía ordinaria, sin asignaciones por SQL en la copia. Sube de prioridad: la candidata de desarrollo fija la organización legada en el código (`alta_desarrollo.go:25`, `peticion_centro.go:201`, `alta_autorizacion_desarrollo.go:331-354`, `plantillas_desarrollo.go:503-509` y `623-625`) y AUT53 no tiene herramienta de aprobación por dos ADMIN; sin esta tarea la firma no se activa de verdad | Consenso de dirección 16/10 |
| F1-64 | Menores de vigencias de AD282: quitar la clave y la raíz fijas de desarrollo 2026-2036 (`contratacion_temporal_postgresql_utilidades_desarrollo.go:229-233` y `ventanaAutoridadSintetica` de CT) y la vigencia fija de 24 h de `portal_externo_publicacion_v3.go:215-220`. Las vigencias pasan a ser configurables: 90 días como máximo en desarrollo y 24 h en producción. La parte de rotación de la raíz y la clave de CT 2026-2036 queda pendiente de consulta de dirección (`gobierno_desarrollo.go:50-79` exige fechas exactas; propuesta de H: material e identificadores nuevos conservando los publicados) | Firma; `internal/modules/contrataciontemporal` | F1-36 | - | S | H | #1181 (parte 1 fusionada) | En curso | 16/10 | Pruebas focales con vigencias distintas en desarrollo y en producción | Consenso de dirección 16/10 |
| F1-65A | Menores de CT226: la restricción CHECK de `catalogo_plantillas_historia_v1` solo admite la organización de desarrollo y una unicidad global; retirar la v1 de compatibilidad cuando no tenga consumidores | SQL de plantillas de CT | - | - | S | H | #1182 | Fusionada | 16/10 | El CHECK admite cualquier organización activa y la v1 ya no tiene consumidores | Consenso de dirección 16/10 |
| F1-66 | Plantillas para varias organizaciones: restricción CHECK, índice de unicidad, operador y AD99 de `catalogo_plantillas_historia_v1` | SQL de plantillas de CT | - | - | S | H | - | Aparcada | 16/10 | El CHECK, el índice, el operador y AD99 admiten cualquier organización activa | Consenso de dirección 16/10. Aparcada: sin consumidor, la instalación tiene una sola organización |
| F1-67 | Preparación de bases (AD153 a BC8) sobre el núcleo vigente, con guardar y consultar la preparación, para que RRHH elija desde la web la modalidad de F1-40 | `internal/modules/bolsa`; SQL | AD288 | - | M | R | - | Pendiente | 16/10 | RRHH elige la modalidad desde la web y la preparación se conserva tras reiniciar | Consenso de dirección 16/10 |
| F1-68 | Auditoría de la lectura de inscripción del aspirante externo: el cotejo del contexto solo miraba el almacén interno y toda lectura externa daba 503 (CA47 y AD289) | SQL de contexto del actor y de autorización | - | - | S | Dirección | #1188 | Fusionada | 16/10 | La lectura externa queda asentada; un recibo inexistente, ajeno o presente en los dos almacenes se deniega sin asiento | Ensayo de #1177, 16/10 |
| F1-69 | Menores de CT231: la base no impide una segunda selección confirmada en el mismo expediente (hoy lo frena la pantalla y la consulta termina con P1405), y la búsqueda de selecciones por campos JSON no tiene índice | SQL de `internal/modules/contrataciontemporal` | - | - | S | R | - | Pendiente | 16/10 | Una segunda selección confirmada en el mismo expediente se rechaza en la base; la búsqueda usa índice | Revisión de #1178, 16/10 |
| F1-70 | Menor de CT230: la rama nativa solo admite la autoridad `autoridad:bolsa:llamamiento:desarrollo`; producción necesitará su propia migración | SQL de `internal/modules/contrataciontemporal` | - | - | S | R | - | Pendiente | 16/10 | La rama nativa acepta la autoridad de producción con su migración; la de desarrollo sigue igual | Revisión de #1170, 16/10 |

## 7. Fase 2. Personal y Organización (por capacidad)

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F2-12 | Simplificación 1, permisos por datos: verificador estable y catálogo cerrado de acciones, sin distribuidor central ni versiones numeradas de rol. Objetivo: añadir una acción normal en una jornada sin tocar el núcleo. Incluye el rol externo del aspirante, hoy con semilla de copia para las acciones del justificante (F1-07) y los campos de modalidad (F1-40), que se retiran al integrarla | SQL del núcleo de autorización; catálogo de acciones | F1-31 | - | L | Sin asignar | - | Pendiente | 16/10 | Se añade una acción normal en una jornada sin migración del núcleo; denegaciones y recorrido iguales | Consenso de dirección con Fable y Astra, 16/10 |
| F2-13 | Simplificación 2, lecturas con sesión acreditada, permiso vigente y auditoría, sin consumir una autorización de un solo uso | Lecturas de todos los módulos | F1-31 | - | L | Sin asignar | - | Pendiente | 16/10 | Equivalencia ruta a ruta: misma respuesta, misma denegación y mismo apunte de auditoría | Consenso de dirección con Fable y Astra, 16/10 |
| F2-14 | Simplificación 3, base SQL consolidada para instalaciones nuevas y vía de actualización para las existentes, sin perder historia | `deploy/postgresql` | F1-31 | - | L | Sin asignar | - | Pendiente | 16/10 | Instalación nueva y actualizada coinciden en esquema, permisos y comportamiento | Consenso de dirección con Fable y Astra, 16/10 |
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

El equipo C vuelve el 20/10. #1142 conserva calendario, correcciones e
informes con política provisional (decisión del 15/10 a las 00:05).

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F3-01 | Cerrar #1142: montar las vistas nuevas, el saldo con permisos y C8 | `internal/modules/cronos`; `web/static` | F0-18 | - | M | C | #1142 | En curso | 14/10 | Recorrido en Chrome con reinicio | Plan del módulo |
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

El equipo D vuelve el 20/10. DIE-01 está en `main` (#838).

| ID | Tarea | Zona de código | Depende de | Bloqueo | Tam. | Equipo | Rama/PR | Estado | Desde | Cómo se comprueba | Fuente |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| F4-01 | Cerrar #1136 y #1139: activación, bandejas, tarifas publicadas y liquidación en preparación | `internal/modules/dietas`; `cmd/vec-dietas` | F0-18 | - | M | D | #1136, #1139 | En curso | 14/10 | CI, ensayo y revisiones | Plan del módulo |
| F4-02 | DIE-02: envío con justificantes por Documentos, con antivirus | `internal/modules/dietas`; Documentos | F0-10, F0-18 | Dudas 24 y 50 (configurables) | M | Sin asignar | - | Pendiente | 15/10 | Envío con recibo; un fallo del antivirus lo deja pendiente | Plan del módulo |
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
| F5-05 | Acreditación de títulos: copia escaneada pendiente y cotejo único reutilizable | `internal/modules/meritos` | F0-10 | Duda 160 | M | Sin asignar | - | Pendiente | 15/10 | Un cotejo hecho una vez sirve en dos bolsas | Plan del módulo |
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

## 12. Fase 7. Uso real (no es programación)

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
| F7-15 | Volver a instalar en cidonia lo ensayado en local | `deploy/principal` | fase 1 | - | - | Dirección | - | Aparcada | 15/10 | Subida con copia y vuelta atrás | Orden de dirección 15/10 |
| F7-16 | Que RRHH vea cada fase funcionando y la dé por buena | - | fase 1 | RRHH | - | Dirección | - | Bloqueada | 15/10 | Visto bueno de RRHH | PLAN §5 |
