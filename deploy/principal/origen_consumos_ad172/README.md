# Filas de origen AD172 para vec-server

## Para qué sirve

Desde que se instaló AD172 en la principal, el núcleo de autorización solo
acepta un uso nuevo de una autorización si encuentra su fila en
`vec_autorizacion_atestada_v3.configuracion_origen_consumos_v1`. La fila es la
terna LOGIN, audiencia y operación, más el canal y el nombre del proceso. Si no
la encuentra, PostgreSQL responde `42501 origen de consumo no acreditado` y la
API lo devuelve como `403`, sin más pista.

El 6 de octubre de 2026 la tabla de la principal solo tenía las dos filas de la
administración. Por eso casi todo lo que hace RRHH en vec-server daba 403 aunque
la persona tuviera permiso. El registro de PostgreSQL de esa mañana tiene 97
rechazos con ese mensaje: «Mis preferencias», «Mi imagen», «Mis correos», el
contacto de la participación en Bolsa, las peticiones de los centros, Cronos y
la política de ofertas. El último uso correcto anterior es del 3 de octubre,
antes de instalar AD172.

Este paquete añade las filas que faltan. Las filas no conceden acciones,
perfiles ni membresías. El núcleo sigue exigiendo la decisión firmada, el
grupo ejecutor de cada perfil y que audiencia, operación y canal coincidan.
Las filas solo dicen qué proceso técnico firma el asiento de auditoría.

## La lista

`ternas.tsv` es la única fuente: la leen el guion, el inventario, el cotejo
posterior y la prueba. Tiene una fila por terna con su bloque, el LOGIN, el
grupo ejecutor, la audiencia, la operación, el canal, el proceso y el perfil
del núcleo.

| Bloque | LOGIN | Ternas | Proceso | Se instala |
| --- | --- | --- | --- | --- |
| `usuarios` | `vec_pref508a_i_ue` (interna) y `vec_pref508a_e_ue` (externa) | 21 | `vec-server` | por defecto |
| `contratacion` | `vec_ct_o207_runtime` | 50 | `vec-server` | por defecto |
| `bolsa` | `vec_bolsa_llamamientos_desarrollo` | 25 | `vec-server` | por defecto |
| `documentos` | `vec_documentos_rrhh_ejecutor_desarrollo` | 8 | `vec-server` | por defecto |
| `incorporacion` | `vec_inc_v2_registro_ct_20260910`, `vec_inc_v2_alta_personal_20260910`, `vec_inc_v2_lector_personal_20260910` | 3 | `vec-server` | por defecto |
| `incorporacionb` | seis LOGIN nominales `vec_ct_personal_b2_*` de Bolsa, CT, RPT y Personal | 21 | `vec-server` | solo si se pide |
| `cronos` | `vec_cronos_emp_ejecutor_desarrollo` | 8 | `vec-server` | solo si se pide |

`incorporacionb` pertenece al recorrido CT→Personal B2 y utiliza los seis
LOGIN que consumen el núcleo de mutación V3. Sus otros dos LOGIN leen la
fuente de autorización y los motivos; no necesitan fila AD172. La operación
`ct_detalle` pasa por el núcleo de consulta RRHH y también queda fuera. Este
bloque es distinto de las tres filas históricas de `incorporacion`.

En el GET B2 del clon HZ11 se observó `interna_corporativa` en el vínculo del
actor consumido por el LOGIN CT; las demás acciones B2 toman el contexto del
mismo montaje interno. El núcleo obtiene el canal de ese vínculo y el proceso
`vec-server` de la fila AD172, no del nombre de aplicación del pool PostgreSQL.

Los bloques anteriores a `incorporacionb` se cotejaron en la principal el 6
de octubre, solo con lecturas:

- **Perfil, audiencia y operación:** del texto vivo de
  `consumir_decision_mutacion_v3_interna`. Para cada perfil, la parte que fija
  sus audiencias y operaciones y la parte que fija el grupo ejecutor que debe
  tener el LOGIN. Los perfiles de Contratación son los que caen en su rama
  general, que exige `vec_contratacion_temporal_ejecutor`.
- **LOGIN:** el miembro de ese grupo que usa vec-server. Contratación y Bolsa
  salen de las DSN del arranque (`VEC_CT_DATABASE_URL` y
  `VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL`). Documentos y Usuarios tienen un solo
  miembro con conexión abierta desde vec-server. Incorporación sale de los
  grupos de `internal/app/composicion/interna/pools_seguimiento.go`: registro
  CT, alta en Personal y lectura de Personal. Esa composición está apagada hoy
  en la principal (no hay `VEC_CT_INCORPORACION_V2_FILE`), pero sus LOGIN
  existen y sus tres filas quedan listas para cuando se encienda.
- **Canal:** `interna_corporativa` para todo lo de RRHH, como en el historial de
  usos. Usuarios lleva además la superficie externa.

El emparejamiento de los bloques anteriores a `incorporacionb` se cotejó a
mano sobre el texto exacto del núcleo, y una segunda revisión independiente
lo repitió. Para B2, las 21 filas se comparan con el núcleo instalado
post-AD218, las operaciones del montaje CT→Personal y los seis LOGIN de los
pools B2. Requieren revisión SQL independiente antes de su aplicación. El
guion no repite ese cotejo. Sí comprueba que el núcleo es uno de los cotejados:
su huella SHA256
tiene que estar en `nucleos_cotejados.txt`. La lista incluye la definición
post-AD218, cotejada para B2. Con otra huella el guion se para; hay que volver
a cotejar las ternas y revisar la nueva huella antes de añadirla.

Unas 24 de las ternas por defecto tienen hoy audiencias sin clave de capacidad
publicada en la principal: los avisos de llamamiento de Usuarios; en
Contratación, las firmas, las plantillas, los ajustes de reglas, el circuito,
el correo del llamamiento y la reincorporación del titular; y en Bolsa, la
auditoría, la aceptación en Contratación, la política de cese, la
reincorporación del titular y la resolución de solicitudes documentales.
Mientras no haya clave, esas filas no hacen nada. Quedan listas para cuando se
publique, así que esas funciones no vuelvan a dar 403. Lo mismo pasa con las
tres de incorporación hasta que se encienda su composición. Si al encenderla
los LOGIN privados no fueran estos, las filas quedarían sin uso, sin riesgo.

Fuera de la lista, a propósito:

- **Dietas.** Sus LOGIN están en `NOLOGIN` desde la retirada P6, y F4b, que
  cambiaría esos LOGIN, no está aplicado. Una fila hoy no serviría. Cuando se
  reactive, va con sus LOGIN reales.
- **Portal del candidato, «Mi bolsa» y Aspirantes.** Son de la superficie
  externa, y sus grupos ejecutores no tienen miembros en la principal.
- **Administración.** Es otro proceso. Ya tiene sus filas o las pone su propio
  paquete.
- **Organización histórica de Personal.** Ningún LOGIN de vec-server la usa.

El cuadro, el detalle y la auditoría de Contratación no necesitan fila: pasan
por el núcleo de consulta RRHH, que no lleva AD172.

## Qué comprueba antes de escribir

- PostgreSQL 18, DBA por el socket local y base `postgres`.
- Que cada fila de la lista tiene ocho campos válidos y que no hay claves
  repetidas.
- Que AD172 está instalada: tabla con RLS forzada, su propietario, sus dos
  disparadores de inmutabilidad y ningún permiso ajeno al propietario.
- Que el núcleo vivo es el cotejado (`nucleos_cotejados.txt`), que llama al
  resolutor y que nombra cada perfil, audiencia y operación, y que cada
  audiencia está admitida para las claves de capacidad.
- Que cada LOGIN cumple lo que pide el resolutor y que su única membresía es el
  grupo que declara la lista: heredada, sin `SET` ni `ADMIN`.
- Que ninguna terna existe ya con otro proceso o canal. Si existe, se para:
  la tabla no admite cambios y eso lo decide el DBA.

Inserta como `vec_autorizacion_atestada_v3_propietario`, que es lo que exige la
política de la tabla. Las ternas que ya existen idénticas se saltan, así que
repetirlo no tiene efecto.

## Cómo se usa en la principal

Desde una copia del repositorio con esta rama, se sube el paquete a cidonia y
se ejecuta como `openclaw`. El guion solo usa los ficheros de su carpeta.

```bash
scp -r deploy/principal/origen_consumos_ad172 root@cidonia.cloud:/home/openclaw/.local/state/vec-desarrollo-20260906/
ssh root@cidonia.cloud chown -R openclaw:openclaw /home/openclaw/.local/state/vec-desarrollo-20260906/origen_consumos_ad172
```

```bash
# 1. Inventario (solo lectura)
ssh root@cidonia.cloud 'su - openclaw -c "VEC_ORIGEN_PG_CONTENEDOR=vec-postgresql-20260906 bash /home/openclaw/.local/state/vec-desarrollo-20260906/origen_consumos_ad172/ejecutar.sh --inventario"'
# 2. Ensayo (termina en ROLLBACK)
ssh root@cidonia.cloud 'su - openclaw -c "VEC_ORIGEN_PG_CONTENEDOR=vec-postgresql-20260906 bash /home/openclaw/.local/state/vec-desarrollo-20260906/origen_consumos_ad172/ejecutar.sh --ensayo"'
# 3. Aplicar (COMMIT)
ssh root@cidonia.cloud 'su - openclaw -c "VEC_ORIGEN_AD172_APLICAR=SI-REVISADO VEC_ORIGEN_PG_CONTENEDOR=vec-postgresql-20260906 bash /home/openclaw/.local/state/vec-desarrollo-20260906/origen_consumos_ad172/ejecutar.sh --aplicar"'
```

Sin `VEC_ORIGEN_BLOQUES` se instalan los cinco bloques de RRHH, 107 ternas.
Cronos se instala aparte cuando se quiera, añadiendo `VEC_ORIGEN_BLOQUES=cronos`.
Las 21 ternas de B2 se seleccionan únicamente con
`VEC_ORIGEN_BLOQUES=incorporacionb`. Antes de usar ese bloque se cotejan sus
seis LOGIN, la composición B2 y la huella viva del núcleo; la fila técnica
por sí sola no concede acciones ni acredita un alta en Personal.

- **El inventario** imprime los bloques, el número de ternas y la ruta del
  inventario.
- **El ensayo** ejecuta todo y termina en `ROLLBACK`. Debe imprimir
  `verificado: ROLLBACK`, con el inventario sin cambios. `ternas_nuevas` cuenta
  solo las filas ausentes de la base, no las 107 seleccionadas por defecto:
  en la preimagen HZ11 faltaban tres de esas 107. Para `incorporacionb` faltaban
  las 21, por lo que su ensayo allí debe mostrar `ternas_nuevas=21`.
- **Aplicar** termina en `COMMIT` y comprueba que estén todas las ternas, que
  no haya desaparecido ninguna fila previa y que no haya filas nuevas fuera de
  la lista.

Los inventarios quedan en un directorio temporal privado cuya ruta se imprime.
Hay que copiarlo a la bitácora privada, fuera de Git, porque `/tmp` puede
vaciarse.

No hace falta reiniciar la aplicación: el núcleo lee la tabla en cada uso.
Después, «Mis preferencias» con un certificado de RRHH debe dar 200, igual que
la bandeja de peticiones del centro y el contacto de la participación.

## El nombre del proceso

Todas las filas llevan `vec-server`, el proceso que tiene hoy esos LOGIN.

**Nadie envía el nombre en tiempo de ejecución.** El núcleo no lo recibe de la
aplicación. `resolver_origen_consumo_v1` lo lee de la propia fila, buscando
por el LOGIN de la sesión, la audiencia y la operación de la capacidad firmada
y el canal de la decisión firmada. Después lo escribe en el eslabón de
auditoría. Por eso no puede haber desajuste con lo que envía vec-server: el
valor es la etiqueta con la que el DBA declara qué proceso es el dueño de ese
LOGIN, y queda sellada en cada asiento.

Cotejado en la principal el 6 de octubre:

- Los LOGIN de la lista solo tienen conexiones desde el contenedor
  `vec-aplicacion-incorporacion-20260910`, que ejecuta `vec-server`.
  `application_name` es una etiqueta por pool
  (`vec-ct-desarrollo-ejecucion`, `vec-usuarios-preferencias-desarrollo`…),
  no un proceso.
- vec-server no declara hoy ningún nombre de proceso. No tiene
  `auditoria-intentos.json` en su material ni fila en
  `configuracion_runtime_intentos`. La única fila de esa tabla es la de
  vec-admin, que usa el mismo nombre en las dos tablas
  (`vec-admin-usuarios`).

**Dos procesos con el mismo LOGIN.** En producción, el portal interno lo sirve
`vec-interno` (`internal/app/composicion/interna`), con su propio material.
La clave de la tabla es LOGIN, audiencia y operación, y la fila no se puede
cambiar. Si `vec-interno` usara estos mismos LOGIN, sus consumos quedarían
sellados como `vec-server`. La opción correcta es la que fijan AD169 y AD172:
otro proceso, otro LOGIN. `vec-interno` tendrá sus propios LOGIN y su propio
paquete de filas con `vec-interno` cuando se despliegue. Estas filas son de
los LOGIN de desarrollo de vec-server y no sirven para producción.

Cuando vec-server tenga su `auditoria-intentos.json` (lo exigen «Mis correos»
desde #608 y Bolsa B-BACK), conviene usar ahí también `vec-server`. Así
intentos y consumos de un mismo proceso llevan la misma etiqueta, como en
vec-admin.

## Ensayo

`bash deploy/principal/origen_consumos_ad172/probar_pg18.sh`

Usa PostgreSQL 18.4 desechable, sin red. Los objetos de AD172 se copian tal cual
de su migración, y los roles y el texto del núcleo se generan desde la misma
`ternas.tsv`. Prueba lo siguiente:

- El ensayo con `ROLLBACK`, la aplicación con y sin confirmación y la
  repetición sin efecto.
- El resolutor con el LOGIN real de cada bloque. Acepta su terna y rechaza el
  canal cruzado, el LOGIN cruzado y otro LOGIN del mismo grupo.
- Que Cronos solo se instala si se pide.
- Los rechazos por terna en conflicto, membresía extra, grupo equivocado,
  permisos de más en la tabla, núcleo sin AD172, núcleo distinto del cotejado,
  bloque desconocido y LOGIN sin permiso de conexión.

El núcleo de la prueba es un sustituto generado desde la misma lista. Prueba el
guion, no la lista: los errores de la lista solo los detecta el cotejo con el
núcleo real descrito arriba.

En la principal se cotejó además, en una transacción de solo lectura, que las
111 ternas las reconoce el núcleo vivo y el catálogo de audiencias, que cada
LOGIN tiene la única membresía que exige su perfil y que no hay ninguna en
conflicto.
