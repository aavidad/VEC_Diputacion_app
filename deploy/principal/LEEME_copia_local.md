# Copia local de VEC para ensayar y recorrer

Receta única para montar en el equipo local una copia de la aplicación con
los datos ficticios de la principal (cidonia), restaurarla, ponerla al día con
`main`, arrancarla y recorrerla en Chrome. Reúne lo que cada equipo resolvió
por su cuenta el 14 y el 15/10 (canal de coordinación, relevo de la PR #1128,
`docs/producto/FLUJO_APP.md` y `docs/producto/TRD.md`). Tarea F0-25 del tablero
(`docs/producto/TABLERO_TAREAS.md`).

Reglas que no cambian:

- Todo es local. Nada de esta receta toca cidonia; solo dirección sube a la
  principal, y únicamente lo ensayado entero en una copia.
- Los datos son ficticios. En tu copia, si un dato, una huella o una fila te
  estorba más de quince minutos, bórralo o recréalo por SQL y sigue (orden de
  Alberto del 14/10). Eso vale para la copia, nunca para el código ni para el
  SQL que va a Git, que no rebajan ninguna guarda.
- Claves, certificados, conexiones, contraseñas y recibos se quedan en el
  material privado de copia (fuera de Git), con modo `0700` en carpetas y `0600`
  en ficheros. En este texto se nombran solo por su función.
- No ejecutes tal cual el lanzador, el JSON o el SQL privado de otro equipo:
  cópialo y cambia puertos, rutas, nombre de contenedor e identificadores. No
  llames a servicios ni a bases de otro equipo.

Lo marcado *(por verificar)* no está acreditado en el canal ni en los
documentos; compruébalo antes de darlo por bueno y corrige esta receta.

## 1. Qué se necesita y límites de memoria

- Un PostgreSQL 18 propio en contenedor (`postgres:18.4`, que es la imagen de
  los ensayos), un `vec-server` compilado desde tu rama y el Chrome del sistema
  manejado con Playwright.
- El material privado de copia: el respaldo de la principal (volcado y
  globales), las ACL de la base y de los tipos, el kit de los dos
  administradores de copia y los clientes y configuraciones privadas de cada
  módulo. Pídeselo a dirección si no lo tienes.
- Memoria: el sistema conserva siempre 32 GB. Antes de restaurar o compilar,
  `free -g` debe dar 40 GB disponibles o más; si no, no restaures ni compiles,
  borra copias y temporales tuyos que ya no uses y avisa en el canal. No toques
  copias ajenas.
- Todo el trabajo va en memoria, bajo `/dev/shm/vec`:
  - carpeta de trabajo: `/dev/shm/vec/wt/<rama>`;
  - datos de PostgreSQL: `/dev/shm/vec/pg/<letra>-<asunto>`;
  - `GOCACHE=/dev/shm/vec/go-build` y `TMPDIR=/dev/shm/vec/tmp/<letra>`.
  Lo que está en memoria se pierde al apagar el equipo: commit y push a menudo.
- Una sola copia viva por equipo y una sola compilación a la vez.
- El contenedor de PostgreSQL se lanza siempre con `--rm --restart=no
  --memory 2g`, los datos montados desde `/dev/shm/vec/pg/...` y escuchando
  solo en `127.0.0.1`. `--rm` borra el contenedor al pararlo, pero no los
  datos, que viven en la carpeta montada.
- Al terminar la pieza, borra tu copia, tu carpeta y tus temporales.

`git worktree move` falla al pasar de disco a `/dev/shm` (cambia de
dispositivo). Haz commit y push, copia la carpeta y ejecuta
`git worktree repair`, comprobando que la cabeza y el árbol coinciden.

## 2. Restaurar la copia de la principal

Base de partida: lo instalado en cidonia, HZ37 con la web HZ38, código
`44218865a` (`deploy/principal/candidata_hz39/LEEME.md`). El respaldo es la
copia de seguridad de cidonia del 13/10.

1. Crea la carpeta de datos en `/dev/shm/vec/pg/<letra>-<asunto>` y arranca un
   PostgreSQL 18.4 vacío con las opciones del apartado 1.
2. Restaura los globales (roles y grupos). Su lectura muestra verificadores de
   contraseñas ficticias: no los vuelques a la salida de herramientas, al canal
   ni a Git. Si ocurre, trata esa salida como sensible y avisa.
3. Restaura el volcado de la base *(por verificar el formato y la orden
   exacta: `pg_restore` o `psql`, según cómo venga el respaldo)*.
4. Aplica las ACL de la base: el volcado no las incluye. Tras restaurar,
   `PUBLIC` conserva `CONNECT` y `TEMP` y faltan los `CONNECT` nominales de los
   grupos de sesión, y la aplicación se detiene en las comprobaciones previas.
   No las reconstruyas a mano: el material privado trae el fichero exacto sacado
   de cidonia (revoca todo a `PUBLIC` y concede los 151 permisos nominales de
   conexión, creación y temporales).
5. Aplica también las ACL de los tipos de fila, que el volcado tampoco
   conserva (`deploy/principal/seleccion_bolsa_constituida/LEEME.md`).
6. Configura TLS en PostgreSQL. Las conexiones de la aplicación verifican la
   CA de PostgreSQL contra el nombre del servidor, y esa CA puede ser distinta
   de la de la web *(por verificar cómo se genera el certificado del servidor
   en una copia nueva; P reinició su copia con el TLS original)*.
7. Comprueba que el contenedor, la base, el puerto y el material son de tu
   copia y guarda una preimagen antes de seguir.

## 3. Poner la copia al día con las SQL de `main`

La copia tiene lo que tiene cidonia. Le faltan las migraciones fusionadas
después de `44218865a`. Para saber cuáles son:

```sh
git diff --name-only --diff-filter=A 44218865a origin/main -- \
  'deploy/principal/lista_sql_*' 'deploy/principal/*/lista_sql*'
git diff --name-only --diff-filter=A 44218865a origin/main -- \
  'deploy/postgresql/**/*.up.sql'
```

La primera orden da las listas nuevas; la segunda, todas las migraciones
nuevas. Coteja las dos: una migración que no aparezca en ninguna lista también
falta. A 15/10 son 26 migraciones, que se aplican en este orden:

| Orden | Lista | Qué trae |
| --- | --- | --- |
| 1 | `candidata_hz39/lista_sql.txt` | Las once de la candidata (confianza externa, cargos y plan de firma, organización de la firma, informe firmado, detalle RRHH) y al final CT218, IS20, AUT97, CT219 y CT224 |
| 2 | ninguna | AD266, secuencia global de confianza ADMIN (#1134). La necesita el gobierno de usuarios ADMIN (apartado 4). P la aplicó después de las once primeras *(por verificar si debe ir antes de CT218)* |
| 3 | `candidata_hz39/personal/lista_sql.txt` | Ocho de Personal y Bolsa nativos (alta y cese desde la bolsa). Exige antes B110 y AD261, que deberían estar en HZ37 *(por verificar)* |
| 4 | `lista_sql_codexs_varias_plazas_20261015.txt` | B117, ofertas con plazo desde la publicación (#1145) |

Las cinco últimas de la lista 1 son las mismas de
`candidata_hz39/lista_sql_firma_vigencias.txt` y de
`lista_sql_codexh_firma_vigencias_20261014.txt`. Aplícalas una sola vez.
`lista_sql_codexh_firma_tres_pasos_20261013.txt`,
`lista_sql_codexk_personal_bolsa_nativa_20261013.txt` y
`lista_sql_codexp_confianza_externa_20261014.txt` son subconjuntos de las
listas 1 y 3.

Cómo aplicarlas:

- Cada fichero, una vez, con `psql -X -v ON_ERROR_STOP=1`. Comprueba antes su
  SHA-256 cuando la lista lo trae.
- Nunca ejecutes un `DOWN` ni reapliques una migración ya instalada. Si una
  falla porque ya estaba, consérvala y sigue con la siguiente.
- Las SQL de PR sin fusionar (por ejemplo AD282 de #1151) solo se aplican
  cuando ensayas esa PR.

Datos del recorrido, después de las SQL: `lista_sql_codexk_datos_recorrido_20261014.txt`
(centro 520, puesto 9/01, plaza 2026/0033, bolsa de Administrativo, con
`-v organismo=<organismo_ref>`), luego `candidata_hz39/personal/datos_plantilla.sql`
y, para la firma, `candidata_hz39/firma/datos_empleados_firmantes.sql`. Las
condiciones de cada uno están en su `LEEME.md`.

## 4. Administradores de la copia

Los dos administradores de cidonia no salen de cidonia y sus claves no se dan
a ningún equipo. Cada copia usa dos administradores ficticios propios.

1. Copia del material privado el kit de administración de copia y el
   «fixture ADMIN» (`fixture-admin-hz37.sql.in`). El fixture hace que las
   vinculaciones de la base coincidan con ese material: desactiva los
   disparadores de inmutabilidad y borra fuentes, vínculos, outbox, arranque y
   control previo. Solo se ejecuta en tu copia (dirección lo autorizó el 14/10
   a las 22:35). Guarda antes una preimagen de Administración.
2. Sigue `candidata_hz39/admin_copia/LEEME.md`: `preparar_copia.py` y después
   `arranque_principal_h10.py` por etapas (fuentes, unidad, arranque y
   verificación), con los planes y aprobaciones de tu copia. No copies recibos,
   LOGIN, puertos ni SQL con identificadores de otro equipo.
3. `preparar_vec_admin.py` crea los LOGIN y además inserta ocho filas técnicas
   de ejecución. No te saltes esa parte.
4. Con `cmd/vec-gobierno-usuarios-admin` y el conjunto 5 (requiere AD266)
   publica la confianza de gobierno y genera la fase `version-bolsa`.
5. Arranca `vec-admin` con la configuración de usuarios
   (`VEC_ADMIN_USUARIOS_CONFIG_FILE`), que monta su frontera técnica con el
   grupo `vec_admin_frontera_tecnica_ejecutor` de AD189. No uses el compositor
   heredado de `vec-admin` (el que arranca sin esa configuración) y no crees el
   grupo `vec_admin_perfiles_auditoria_ejecutor`: llama a una función que no
   existe (consenso de seguridad del 15/10; arreglo en F0-02).

Renovación diaria. Hasta que entre F0-03 (#1151, vida configurable hasta 90
días en desarrollo), las claves y la confianza de gobierno ADMIN duran 24
horas. Cuando venzan, renovar no es alargar: genera audiencias y claves nuevas
con la vida actual por la vía normal del kit (`preparar`, aprobación,
`aplicar`, `verificar`) y haz la propuesta y el cierre con los dos
administradores dentro de esa ventana. No toques las claves históricas.
La ventana de un día del plan técnico de admisión (AUT61) y la del plan de
publicación de cargos se mantienen: son el plazo para aplicar un acto, no la
vigencia de lo publicado.

Perfiles y asignaciones de la copia:

- Las asignaciones ficticias de perfiles de Contratación temporal y sus
  ámbitos pueden ir como dato de semilla por SQL. Los cargos, los certificados
  y el plan de firma van por actos reales con los dos administradores.
- Contratación temporal fija la unidad `unidad:desarrollo:rrhh`. El plan, los
  cargos y las asignaciones deben cubrir esa unidad; una unidad nueva de la
  copia no basta.
- No insertes asignaciones con dimensiones que ningún consumidor valida (por
  ejemplo centro y categoría): quedarían sin acotar. Acota por organización y
  unidad gestora mediante el lote ordinario. Lo que de verdad exija centro y
  categoría queda pendiente de Administración (F0-06).
- Una asignación retirada no se revive: se otorga una nueva.

## 5. Identidad del aspirante externo

El respaldo HZ37 trae vacía la tabla de admisiones técnicas de identidad
externa (`vec_identidad_externa_v1.llamante_autorizacion`). Sin ellas, Mi Bolsa
devuelve «instantánea obsoleta» al revalidar la sesión.

- En tu copia, restituye las admisiones por SQL y anótalo en tu guion de
  ensayo como «dato de copia».
- Nunca como activación: ni en `deploy/principal` ni presentado como
  «aprobado por Identidad». No existe todavía el productor gobernado (una
  función para un grupo ADMIN que exija aprobación, audite en la misma
  transacción y permita retirar). Es tarea del tablero (consenso de seguridad
  del 15/10).
- Mi Bolsa exige además que el `candidato_ref` del manifiesto coincida con el
  vínculo vigente de esa persona.

## 6. Organización histórica y lectura de Personal en el servidor combinado

La selección desde la bolsa constituida (#1088) consulta el puesto en la
Organización histórica de Personal. Hasta el 15/10 ese GET solo se montaba en
`vec-interno`; el `vec-server` combinado publicaba la clave pero no servía la
consulta, y la selección fallaba.

- #1150 (pendiente de fusión a 15/10) monta esa consulta en el combinado, solo
  en desarrollo; en producción manda la cápsula de `vec-interno`. Se activa
  con `VEC_ORGANIZACION_HISTORICA_CONSULTA_ENABLED=true` y
  `VEC_ORGANIZACION_HISTORICA_MATERIAL_DIR`, además de
  `VEC_ORGANIZACION_HISTORICA_GOBIERNO_ENABLED=true` para publicar la capacidad.
  Los pasos (perfil con sus seis campos, motivos, material y cuatro conexiones)
  están en el guion `deploy/principal/organizacion_historica_combinado_20261015.md`
  de esa PR. Las fechas de los motivos llevan seis decimales.
- La lectura de Personal para Contratación (M10) sigue
  `candidata_hz39/personal/LEEME.md`. Copia del material privado el cliente
  `cliente_m10_ct.json` y cambia su origen al puerto de tu propio servicio; el
  que trae apunta al de otro equipo. `VEC_CT_BOLSA_ORGANIZACION_SOURCE_PATH`
  apunta a esa fuente.
- Unidad gestora y unidad del puesto: el puesto 9/01 de la semilla es de
  `unidad:centro:520` y la unidad gestora de Contratación es la de RRHH. Hasta
  que entre F1-21 (#1153), la selección rechaza ese puesto. No cambies el dato
  para esconderlo.
- Necesidad de la RPT: la solicitud 2026/94011 de la copia no tiene necesidad
  guardada y está en acumulación de tareas, que no admite puesto ni huella. En
  tu copia puedes usar la variante vacante del caso 4.0 con dos personas
  solicitadas, como dato de copia y sin atribuirla a la fiscalización anterior.

## 7. Arrancar `vec-server`

Base común (`docs/producto/TRD.md`, apartado 4):

- `VEC_HTTP_ADDR` en `127.0.0.1`, TLS con `VEC_TLS_CERT_FILE` y
  `VEC_TLS_KEY_FILE`, e identidad por certificado.
- Doble llave de desarrollo: `VEC_EXECUTION_PROFILE`, `VEC_AUTH_MODE` y
  `VEC_DEVELOPMENT_GUARD`. Sin ella no se componen Bolsa ni Contratación.
- Conexiones nominales de tu copia, con TLS verificado y un LOGIN por
  consumidor. Nunca la conexión de superusuario ni la de otro módulo.
- `config/selectores_dependencias_bolsa_ct_v1.json` lista lo que pide cada
  selector. Al arrancar, el registro avisa de cada dependencia que falta.

Orden de arranque:

1. Primero sin ninguna configuración opcional. El lanzador de S para HZ37
   apaga firma, Documentos, incorporación y ajustes avanzados. Deben responder
   `/livez`, el portal, la sesión y Bolsa.
2. Después enciende una configuración cada vez y reinicia vigilando el
   arranque. Con ella ausente o inválida, la aplicación arranca y solo esa
   función responde 503 o 404.

| Configuración opcional | Dónde está la receta |
| --- | --- |
| Firma de cinco pasos y verificador | `candidata_hz39/firma/LEEME.md` (`identidad/firma-vec.json`, `VEC_FIRMA_VERIFICACION_*`) |
| Personal, alta y cese desde la bolsa | `candidata_hz39/personal/LEEME.md` |
| Organización histórica en el combinado | Guion de #1150 (apartado 6) |
| Ofertas con plazo desde la publicación | `activacion_codexs_ofertas_20261015.md` (`VEC_BOLSA_REGLAS_SOURCE_PATH`) |
| Auditoría de RRHH | `habilitacion_auditoria_rrhh_20261013.md` (tarea F0-01) |

Un fichero que el proceso lee al arrancar no se cambia con la aplicación en
marcha: para, cambia y arranca. Si algo falla, vuelve al binario y la
configuración anteriores. El TRD avisa de que no todos los montajes cumplen
todavía la regla de cerrar solo su función: si un componente tumba el
arranque, anótalo como fallo del producto.

## 8. Comprobar en Chrome

- Chrome del sistema con Playwright, un certificado por perfil.
- Direcciones: `/portal-empleado/` (RRHH e Intervención),
  `/portal-empleado/peticiones-centro/` (centro y ratificador) y
  `/area-personal/` (aspirante).
- El recorrido es el caso 4.0 de `docs/producto/FLUJO_APP.md`. Para el tramo
  de Personal, el orden es 17, 18, 16a, 16b, 19, 20 y 21.
- Por cada paso: respuesta HTTP, recibo y captura a 1440 y 390 píxeles. Sin
  errores de JavaScript, sin cookies ni almacenamiento web y sin
  desbordamiento horizontal.
- Un recibo local no confirma por sí solo el efecto en otro módulo: comprueba
  también la pantalla del consumidor (Mi Bolsa, ficha de Personal).

## 9. Reiniciar y comprobar que todo sigue

1. Para `vec-server`.
2. Para el contenedor de PostgreSQL. Con `--rm` desaparece, pero los datos
   siguen en `/dev/shm/vec/pg/...`.
3. Arranca otra vez el contenedor con la misma carpeta y las mismas opciones, y
   después la aplicación.
4. Repite las lecturas y los reintentos con la misma clave de operación: mismos
   recibos y fechas, mismas versiones y ninguna fila duplicada.

Si se apagó el equipo, `/dev/shm` está vacío: vuelve al apartado 2.

## 10. Problemas conocidos

| Síntoma | Causa | Arreglo |
| --- | --- | --- |
| La aplicación se detiene en la comprobación previa de preferencias o de Contratación | Faltan las ACL de la base tras restaurar | Aplicar el fichero de ACL de base del material privado (apartado 2) |
| Errores de permisos sobre tipos de fila | Faltan las ACL de los tipos | Aplicarlas tras restaurar |
| Los administradores del kit no coinciden con ninguna vinculación de la copia | Falta el fixture ADMIN | Aplicar el fixture en tu copia (apartado 4) |
| Propuesta o cierre de gobierno rechazados por claves vencidas | Vida de 24 h hasta F0-03 | Generar claves y audiencias nuevas con el kit y aplicarlas en la ventana |
| `vec-admin` pide `vec_admin_perfiles_auditoria_ejecutor` | Se está usando el compositor heredado | Arrancar con `VEC_ADMIN_USUARIOS_CONFIG_FILE`; no crear el grupo |
| Plan técnico de admisión (AUT61) caducado | Ventana de un día para aplicar | Prepararlo de nuevo y aplicarlo el mismo día |
| Mi Bolsa responde 503 o «instantánea obsoleta» | Admisiones de identidad externa vacías en HZ37 | Restituirlas como dato de copia (apartado 5) |
| La selección desde la bolsa constituida da 404, 409 o 503 | Organización histórica sin montar en el combinado o sin material | #1150 y su material (apartado 6) |
| La selección rechaza el puesto 9/01 | Unidad gestora frente a unidad del puesto | Esperar a F1-21 (#1153); no tocar el dato |
| La selección exige una necesidad de la RPT | La solicitud de la copia no la guardó | Variante vacante del caso 4.0 como dato de copia |
| El cliente de Personal (M10) llama a un puerto ajeno | `cliente_m10_ct.json` copiado sin cambiar | Apuntarlo a tu servicio |
| La consulta de firmas da 503 con «autenticación histórica inválida» | Falta IS20 | Aplicar la lista 1 completa |
| Rechazo de una fecha que es el mismo instante | Representación distinta: AUT32 compara texto con el canon nominal; CA25 y los motivos de Organización histórica exigen seis decimales | Generar las fechas en el formato canónico de cada guarda; corregir el dato de la copia |
| Los datos de empleados firmantes piden 400 días de relación cubierta | Las asignaciones de la copia duran 90 días | Ajustar en la copia las relaciones a las vigencias existentes (P usó 89 días por SQL) |
| Firma del paso 3 rechazada por unidad | CT224 solo admite el paso 3 con unidad y paso publicados en el plan vigente | Publicar el plan con esa unidad |
| `free -g` por debajo de 40 GB | Copias o cachés acumuladas | No restaurar ni compilar; borrar lo propio y avisar |
| `git worktree move` falla hacia `/dev/shm` | Cambio de dispositivo | Copiar la carpeta y `git worktree repair` |
