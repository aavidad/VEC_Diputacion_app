# Estado de Cronos para el tablero, 21 de octubre de 2026

Tarea: TABLERO-CRONOS-DIETAS. Base examinada: `origin/main@cda070a31`.
Comprobaciones ejecutadas el 20 de octubre para la actualización del 21.
No se cambia código, SQL ni configuración versionada.

Cronos tiene composición, rutas, pantallas y pruebas integradas para jornada,
saldo, solicitudes de permisos, resolución y comunicaciones. Pasan los 15
paquetes Go que contienen pruebas; otros dos no tienen pruebas. Esto acredita
componentes construidos. Ninguna de las doce capacidades completas queda
acreditada aquí como «Funciona» con recorrido web, instalación y reinicio.

## Criterio de clasificación

«Funciona» exige montaje en la aplicación, ruta, SQL aplicable, pantalla y
pruebas positivas; cuando el resultado depende de actuar desde la pantalla,
también se exige observarlo en Chrome. Por decisión de dirección del 20/10 a las 22:10, el tablero reserva
«Fusionada» para código en main con montaje, ruta, lista SQL, pantalla y
pruebas; añade «sin recorrer en la web» a su evidencia. «Hecha» exige además
el recorrido web. La ausencia de ese recorrido no basta para dejar Pendiente
un componente que cumpla las demás condiciones. «Construida sin montar» identifica piezas
con código y pruebas que la aplicación no ofrece. «A medias» indica que falta
parte del resultado o su comprobación. «No existe» se limita al efecto que no
se ha localizado en main. «Sin comprobar en la web» no significa «sin montar».

Las rutas y líneas siguientes pertenecen a la base indicada. Las pruebas
citadas están dentro de la ejecución Go descrita al final; sus dobles de base,
identidad o documentos no prueban una instalación PostgreSQL.

## Capacidades C1 a C12

| Capacidad | Estado y alcance construido | Evidencia en main y prueba | Qué falta |
| --- | --- | --- | --- |
| C1. Identidad común y organización | A medias; frontera montada, sin comprobar en la web | `internal/app/bootstrap/cronos_empleado.go:170` revalida identidad y vínculo; `:545` compone los resolutores. `internal/modules/cronos/adapters/composicion/composicion_test.go` pasa | Acreditar organización como ámbito, relación vigente y sesión nominal en una copia. No hay otro acceso de tarjeta y contraseña |
| C2. Fichajes con origen, historia y recibo | A medias; fichaje remoto montado | `internal/app/bootstrap/cronos_empleado.go:652` registra escritura, disponibilidad y recuperación. `web/static/portal-empleado/portal-composicion-empleado.js:117` monta el fichaje y renueva las lecturas. `application/remoto_test.go:132` comprueba autorización positiva y `adapters/postgres/transaccion_marcaje_test.go:172` conserva recibo al repetir | Recorrido web con teletrabajo autorizado, recuperación y reinicio; integración de terminales. El manejador de marcaje genérico no se publica en este mapa |
| C3. Saldo por día, semana, mes, año y periodo | A medias; consulta propia montada; cálculo con permisos construido sin montar | `internal/app/bootstrap/cronos_empleado.go:651`; `web/static/portal-empleado/portal-composicion-empleado.js:108`. `application/consulta_saldo_test.go:37` no inventa jornada; `:268` comprueba agrupación por fecha local. `application/ensayo_saldo_permisos.go:11` limita el cálculo con permisos a una instantánea de ensayo | Jornada histórica oficial y permisos aprobados con reglas vigentes; recorrido, medición HTTP y pintado. La optimización de memoria o SQL por separado no cierra F3-02 |
| C4. Calendario anual de organización y centro | A medias; calendario de movimientos montado; consulta histórica construida sin montar | `web/static/portal-empleado/portal-composicion-empleado.js:111` monta calendario usando movimientos propios, ruta en `internal/app/bootstrap/cronos_empleado.go:655`. `application/ensayo_calendario_historico.go:32` y sus pruebas pasan | Fuente oficial de Calendarios, adscripción histórica de Personal y cuadrante publicado. El mapa de rutas no tiene una consulta nominal de calendario oficial |
| C5. Olvido y corrección sin borrar el original | A medias; solicitud montada; decisión y aplicación construidas sin conexión durable | `internal/app/bootstrap/cronos_empleado.go:656` publica solo la solicitud. `application/correccion.go:94` contiene actuaciones; `application/correccion_test.go:100` conserva el original. `adapters/postgres/empleado_solicitudes.go:221` y `:225` devuelven dependencia no disponible en actuaciones y recuperación | Adaptador durable, fuente histórica y rutas para decisión, resolución, aplicación y recuperación. La raíz usa el constructor sin lector histórico (`cronos_empleado.go:593`) |
| C6. Catálogo versionado de permisos | A medias; consulta construida y montada, sin comprobar en la web | `internal/app/bootstrap/cronos_empleado.go:657`; `web/static/portal-empleado/portal-composicion-empleado.js:137` monta permisos. `domain/catalogo_permiso_test.go` y `adapters/postgres/empleado_solicitudes_test.go` pasan | Catálogo aprobado, gobierno y ámbito por régimen; verificar en la copia las cantidades provisionales. Existe la semilla `deploy/postgresql/cronos_v1/datos_sinteticos/catalogo_permisos_sintetico_duda41.sql` |
| C7. Solicitud, resolución y aviso de permiso | A medias; circuito construido y montado con selector propio | `internal/app/bootstrap/cronos_empleado.go:658` publica solicitud y `:661` añade bandeja, resolución y avisos. `application/resolucion_permiso_test.go:99` comprueba motivo y recibo; `:68` rechaza fuente incoherente. Portal en `portal-composicion-empleado.js:32` | Instalación contrastada de la cadena de resolución, competencias nominales y recorrido con reinicio. La evidencia parcial de #1142 pertenece a una rama sin fusionar |
| C8. Aportar y revisar justificantes custodiados | Construida sin montar | `application/justificacion.go:101` aporta y `:164` revisa; `adapters/documentosjustificacion/adapter.go:61` registra con Documentos. `application/justificacion_test.go:259` impide confirmar Cronos si falla Documentos y `:276` rechaza un registro distinto | Fuente y repositorio durables, permisos, subida/custodia y rutas/pantalla operativas. Ninguna ruta de justificación aparece en `cronos_empleado.go:650` |
| C9. Notificaciones a RRHH y avisos archivables | A medias; construidas y montadas con selectores | `internal/app/bootstrap/cronos_empleado.go:663` añade avisos y archivo; `:668` añade notificaciones y atención. Portal en `portal-composicion-empleado.js:38` y `:53`. `application/notificacion_test.go:112` comprueba recibo | Recorrido nominal e instalación de las dependencias. El adjunto transmite referencia y huella; no hay subida de bytes en este circuito (`modulos/cronos/INTEGRACION.md`, apartado inicial) |
| C10. Incidencias propias y agregados de RRHH | A medias; movimientos propios montados, agregados construidos sin montar | `internal/app/bootstrap/cronos_empleado.go:655`; `application/ensayo_agregados_incidencias_test.go:46` comprueba cobertura y datos mínimos; `:104` rechaza entradas inválidas. Terminal existente `cmd/vec-cronos-agregados-incidencias/main.go` | Fuente nominal del equipo, consulta autorizada de agregados y pantalla; no convertir ausencia de registro en ausencia laboral probada |
| C11. Equipo, presencia y vistas de responsable/RRHH | A medias; bandejas montadas, presencia construida sin montar | `internal/app/bootstrap/cronos_empleado.go:661` y `:670` publican bandejas; `application/ensayo_presencia_test.go:27` comprueba selección y `:84` conserva indeterminación con cobertura parcial. Terminal `cmd/vec-cronos-presencia/main.go` | Fuente del equipo vigente, presencia nominal, permisos y consumidor de la consulta. Las bandejas no acreditan el conjunto de la vista de equipo |
| C12. Exportación e impresión PDF | Construida sin montar en el portal | `adapters/informesaldo`, `adapters/informepermisos` y `adapters/informemovimientos` pasan sus pruebas. Herramientas `cmd/vec-cronos-informe-saldo/main.go`, `cmd/vec-cronos-informe-permisos/main.go` y `cmd/vec-cronos-informe-movimientos-csv/main.go` | Lectura nominal, selección autorizada de campos, permiso de exportación, auditoría y descarga desde pantalla; cierre mensual y rectificación. Generar un fichero local con datos ficticios no acredita ese recorrido |

En la tabla, las rutas abreviadas `application/`, `domain/` y `adapters/`
pertenecen a `internal/modules/cronos/`. Las referencias web abreviadas
pertenecen a `web/static/portal-empleado/`.

## Montaje, SQL y activación

La raíz prepara Cronos en
`internal/app/bootstrap/composicion_desarrollo.go:273`, lo monta en `:426` y
protege su manejador en `:625`. `cronos_empleado.go:650` registra ocho rutas
básicas, cuatro de resolución y cuatro de notificaciones. El portal importa
clientes y vistas en `web/static/portal-empleado/portal-modulos-coordinador.js:104`
y compone sus subvistas en `:1577`. Están conectadas en código, aunque el
menú y la activación dependan de la configuración.

Los tres selectores son `VEC_CRONOS_EMPLEADO_ENABLED`,
`VEC_CRONOS_RESOLUCION_ENABLED` y `VEC_CRONOS_NOTIFICACIONES_ENABLED`
(`config/cronos_empleado.go:9`, `:14`, `:63`). Resolución y notificaciones
exigen el primero; todos requieren el entorno de desarrollo admitido. La
preparación lee `identidad/cronos-empleado.json`, cuentas, conexiones nominales,
motivos y política del canal remoto (`internal/app/bootstrap/cronos_arranque.go:98`).
Un fallo propio deja Cronos en 503 y conserva la raíz
(`cronos_arranque.go:284`). El selector apagado no abre sus conexiones.

Main contiene las migraciones Cronos 000001 a 000010 y 000015. Los números
000011 a 000014 no están en ese directorio de migraciones de main. La lista
`deploy/principal/lista_sql_codexu_cronos_saldo_20261007.txt:1` incluye solo
000015, que modifica el libro de saldo sobre la base anterior. La lista
`deploy/principal/lista_sql_main_tras_hz37.txt` no contiene Cronos: enumera
incrementos posteriores al volcado de referencia, y su ausencia no demuestra
qué está instalado en ese volcado. La copia local descrita abajo no acredita
la instalación completa de la cadena Cronos.

La guía `web/static/portal-empleado/modulos/cronos/INTEGRACION.md`, apartado
«Orden de instalación», conserva la cadena histórica de resolución y
notificaciones: AD3-53, AD3-70, AD3-57, Cronos 000009, AD3-58 y Cronos 000010.
Es evidencia de dependencias, no una receta autorizada para reaplicar SQL.
El inventario de 37 ficheros SQL incluye pruebas y semillas; no son 37
migraciones. No se ha instalado ni modificado ninguna.

La opción `--modulos` de `deploy/principal/copia_local.py:609` prepara
conexiones de Bolsa y Contratación. Por sí sola no activa Cronos. Dirección
corrigió a las 22:10 la decisión anterior y pidió una copia local sin esa
opción, sin usar material privado ni programar. La comprobación siguiente
acredita el arranque y la indisponibilidad de Cronos; no su recorrido funcional.

Con el criterio de esa misma decisión, no basta con que el código esté
montado: las migraciones deben figurar en una lista de main. No se encontró
esa lista para Cronos 000001 a 000010; 000015 solo cubre la optimización.
Las dependencias de resolución no están en HZ37 según
`deploy/principal/candidata_cronos/arranque/LEEME.md`, apartado
«Comprobaciones de activación». Por ello los cortes que requieren esa cadena
permanecen Pendiente, con la falta de lista identificada; no se marca como
Fusionada una instalación deducida del volcado. Los guiones SQL con fachadas
de prueba no sustituyen la lista principal. Esta carencia se comunicó a
dirección por el canal.

## Comprobación en copia local, 20/10 a las 22:27

Tras la liberación del turno de D a las 22:18, se creó una única copia propia
con `deploy/principal/copia_local.py crear --equipo c --instancia
estudio-cronos-local-20261021`. Se usaron el volcado HZ37 y sus anexos
autorizados en solo lectura; la receta generó credenciales nuevas. No se usó
`--modulos`, ningún productor privado ni material de otros equipos. La base
de código fue `cb8aff928`, idéntica a `cda070a31` salvo documentación.

La creación terminó con salida 0 en 76,11 segundos y arrancó la aplicación
con Cronos apagado. No se aplicó lista SQL adicional: la lista general ya
había fallado en el ensayo de D con «AD278: origen CT nominal incompatible»;
no se repitió ese intento ni se atribuyen a la copia incrementos posteriores
a HZ37. Tampoco se instaló 000015 sobre una cadena sin acreditar.

El material recién generado no contiene `identidad/cronos-empleado.json`.
Se reinició solo la aplicación propia conservando su entorno y activando
`VEC_CRONOS_EMPLEADO_ENABLED`, `VEC_CRONOS_RESOLUCION_ENABLED` y
`VEC_CRONOS_NOTIFICACIONES_ENABLED` con el valor `true`. El proceso emitió:
`Cronos no disponible componente=cronos_material_privado causa=no_disponible`.
La aplicación siguió activa: `/livez`, `/portal-empleado/` y
`/api/vec/session` respondieron 200. La comprobación del gestor, lanzada
inmediatamente tras reiniciar, falló antes de estar lista; una vez disponible,
`copia_local.py comprobar` terminó con salida 0. No se presenta como Cronos
activado correctamente.

El primer requisito que falta es ese manifiesto. Su contrato exige ocho
conexiones nominales separadas (identidad, revalidación, contexto, fuente y
registro de autorización, motivos, ejecución y auditoría de Cronos), cuentas
ligadas al certificado, motivos positivos y de rechazo y política de fichaje
remoto (`internal/app/bootstrap/cronos_empleado.go:46` y
`internal/app/bootstrap/cronos_arranque.go:182`). Esos requisitos no se
comprobaron al faltar el fichero; tampoco se acreditaron cuentas, motivos ni
la cadena SQL en esta copia. No se fabricó el manifiesto ni se copiaron
credenciales históricas para superar el bloqueo.

Chrome del sistema, manejado con Playwright, abrió
`/portal-empleado/?lang=es#cronos` con el certificado propio de la copia,
a 1440 × 1000. Se observaron los mismos fallos con los selectores apagados
y solicitados:

| Vista o acción | Resultado observado |
| --- | --- |
| Mi jornada: saldo, movimientos y calendario | Pantalla visible, con avisos de indisponibilidad; saldo y movimientos responden 503 |
| Fichaje remoto | Disponibilidad responde 503; entrada, pausa y salida quedan deshabilitadas. No se registra marcaje ni recibo |
| Permisos y vacaciones | La vista abre; permisos propios responde 503 y muestra error. No se crea ni resuelve solicitud |
| Avisos de resolución y notificaciones a RRHH | Los botones no cambian la vista de Permisos en esta sesión; sus enlaces directos terminan en Inicio. No se acredita consulta ni acto |
| Corrección, justificantes e informes | No se ejecutan: no hay movimientos ni solicitud operativos; el inventario de montaje conserva sus límites |

Las peticiones observadas fueron
`/api/interna/cronos/saldos/propio?periodo=hoy`,
`/api/interna/cronos/movimientos/propio?periodo=anio`,
`/api/interna/cronos/marcajes/remoto/disponibilidad` y
`/api/interna/cronos/permisos/propio?anio=2026`, todas con 503.
La consola también registra fallos de preferencias; no es un recorrido sin
errores. La sesión 200 acredita acceso al portal, sin demostrar relación ni
competencia nominal de Cronos.

La captura `cronos-local-no-disponible.png`, las instantáneas de jornada y
permisos, las peticiones y el diagnóstico se conservan en la evidencia local
del equipo, fuera de Git. SHA-256 de la captura:
`956744c19d3c54820834ed56eb050513b02eb4fbd57c533942f71bf960f40e6e`.
La captura prueba la pantalla de indisponibilidad, sin acreditar fichaje,
saldo, solicitud, resolución, recibo o recuperación tras reinicio de la base.

Se cerró Chrome y se ejecutaron `copia_local.py parar` y `copia_local.py
borrar` sobre la instancia propia. No se modificaron código ni SQL, ni se
accedió a cidonia. Ninguna tarea pasa a «Hecha» por esta comprobación.

## Qué está fusionado y qué permanece en otras ramas

El historial de main y las PR fusionadas acreditan composición (#36),
movimientos y permisos (#40), certificado (#46), resolución y comunicaciones
(#51), consultas y conservación de formularios (#251 a #254, #266, #270,
#349, #353, #357, #386 y #387), informes locales (#311, #329, #371, #576,
#655, #659 y #664), saldo optimizado (#831 y #877), vínculo y auditoría
(#851 y #861) y aislamiento del arranque (#1141).

La PR [#1142](https://github.com/aavidad/VEC_Diputacion_app/pull/1142) sigue
abierta, cabeza `a03c7ce40`. Su permiso concedido desde Chrome y recuperado
tras reanudar la aplicación corresponde a una copia con SQL parcial; su acta
`deploy/principal/candidata_cronos/resultado_recorrido.md`, en esa rama,
declara pendiente el reinicio de PostgreSQL y el ensayo completo. Sus vistas,
SQL y configuración pendientes no se atribuyen a main. F3-01 sigue Aparcada.

El plan conserva el cómputo histórico 0/12 formal y 3/12 técnico (C6, C7 y
C9). La ficha anterior decía que C8 no tenía operación; main sí contiene
aportación y revisión, todavía sin montaje. También existen ensayos de equipo,
agregados e informes. Esas diferencias se reflejan aquí sin dar por cerrado
el alcance completo de C8, C10, C11 o C12.

## Comprobaciones y traslado al tablero

`go test -p 6 ./internal/modules/cronos/...` terminó con salida 0 sobre
`cda070a31`: 15 paquetes con pruebas positivas y dos sin pruebas; sin
resultados de caché. Incluye servicios, dominio, composición, manejadores,
adaptadores PostgreSQL con dobles, informes y enlace con Documentos. No se
repitió la batería sobre esa base. No se ejecutaron pruebas SQL ni Node adicionales; la copia local no
acredita instalación completa, sesión nominal de Cronos o persistencia tras
reiniciar la base. La puerta completa ejecutada por D terminó con salida 0
sobre el mismo código de main, incluidas 4.169 pruebas Node; no se repite.

Se consultaron el historial pedido (`git log` sobre main), las PR fusionadas
con «cronos», títulos C1/C2/C3 y las ramas remotas de Cronos. Las búsquedas por
numeración también devuelven trabajos de Contratación ajenos al módulo: no se
usan como evidencia de Cronos. El índice y `gopls` no están disponibles en
esta sesión; se leyeron los ficheros y consumidores existentes.

F3-02 a F3-10 conservan su alcance y sus dependencias. Sus notas deben
reconocer las piezas ya construidas y precisar la comprobación que falta.
Se añaden al tablero las capacidades de catálogo y comunicaciones que no
tenían fila propia; permanecen Pendiente por la cadena SQL sin lista
completa y sin recorrer en la web. Las piezas sin montaje conservan esa
clasificación. No se propone ninguna «Hecha» nueva.

Configuración nueva: ninguna. SQL a instalar: ninguno. Activación: ninguna;
solo cambia documentación. Dirección conserva la revisión, integración y
cierre de la tarea.
