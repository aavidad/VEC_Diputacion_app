# Copia local de VEC para ensayar y recorrer

El guion `deploy/principal/copia_local.sh` restaura la base de referencia,
aplica las listas SQL indicadas y arranca el portal con certificados nuevos.
Tarea F0-28 del tablero. Este primer corte arranca con las funciones opcionales
apagadas: todavía no prepara las conexiones nominales ni los actos de los
módulos. Un portal que responde 200 acredita el arranque, no el recorrido de
Bolsa y Contratación. Los apartados 3 a 8 conservan esa preparación pendiente.

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
- El material privado de copia: el respaldo de la principal (volcado), las ACL de la base y de los tipos, el kit de los dos
  administradores de copia y los clientes y configuraciones privadas de cada
  módulo. Pídeselo a dirección si no lo tienes.
- Memoria: el sistema conserva siempre 32 GB. Antes de restaurar o compilar,
  `free -g` debe dar 34 GB disponibles o más (32 para el sistema y 2 para la copia); si no, no restaures ni compiles,
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

## 2. Crear y usar una copia con el guion

Requiere Docker, Python 3, OpenSSL, la imagen local `postgres:18.4`,
la herramienta Go declarada en `go.mod` y sus dependencias ya descargadas.
No descarga herramientas ni imágenes para compilar. La variable
`VEC_COPIA_ORIGEN` apunta a la carpeta privada de referencia, solo de lectura.
Debe contener el volcado `principal-hz37-20261014.dump` y sus anexos
`acl-base-20261014.sql` y `acl-tipos-20261014.sql`.

Desde la raíz de tu rama:

```sh
deploy/principal/copia_local.sh crear --equipo l --instancia recorrido \
  --puerto-pg 55481 --puerto-app 18481 \
  --lista-sql deploy/principal/candidata_hz39/lista_sql.txt \
  --lista-sql deploy/principal/candidata_hz39/lista_sql_firma_vigencias.txt
```

Elige tu letra, una instancia nueva y dos puertos libres. Se puede repetir
`--lista-sql` para formar un único orden causal; las rutas y las huellas
opcionales se validan antes de restaurar. Se conservan los bytes validados
para aplicar y registrar exactamente la misma SQL. Una SQL repetida o una huella
incorrecta rechazan la lista entera. Solo admite migraciones `.up.sql` bajo
`deploy/postgresql`. Solo admite SQL añadida en la cabeza versionada después del corte de
referencia `44218865a`. Rechaza conservadoramente cualquier SQL que ya
existía en ese corte, aunque quedara sin instalar; esos huecos requieren
un corte específico del módulo, no repetir una receta histórica. Revisa
el orden del apartado 3. La orden `crear`
rechaza un destino existente. Una restauración fallida se conserva para
examinar sus registros; se borra y se crea otra, sin reaplicar SQL.

El resultado vive en `/dev/shm/vec/pg/<equipo>-<instancia>`, con padre `0700`.
El volcado y sus anexos aportan nombres de propietarios y beneficiarios para
crear roles `NOLOGIN`. El guion no abre ni restaura el fichero de roles
globales, sus contraseñas ni sus membresías. La restauración conserva las ACL
de los objetos; después aplica las ACL de la base y los tipos. No convierte
esos roles en identidades operativas de la aplicación.

F0-31: antes de las migraciones, el guion crea también
`vec_externo_documentos_bolsa_datos` y `vec_interno_documentos_bolsa_rrhh`
como `NOLOGIN`, aunque no aparezcan en el respaldo. B119 necesita ambos
nombres. No se añaden contraseñas, membresías ni permisos; la preparación
documental posterior sigue siendo necesaria para usarlos desde el portal.
Si el respaldo ya cita uno de esos nombres, se crea una sola vez.

El generador común crea CA, TLS y certificados de RRHH e Intervención de
90 días. `VEC_DESARROLLO_VIGENCIA_DIAS` permite de 60 a 90 días al invocarlo
por separado; el guion de copia usa 90. Un material existente se verifica,
sin renovar sus certificados ni cambiar sus claves. PostgreSQL usa TLS con
esa CA nueva. La aplicación se inicia con un entorno explícito, sin heredar
conexiones ni selectores de otra copia.

Conserva los mismos argumentos de equipo e instancia para estas órdenes:

```sh
deploy/principal/copia_local.sh comprobar --equipo l --instancia recorrido
deploy/principal/copia_local.sh reiniciar --equipo l --instancia recorrido
deploy/principal/copia_local.sh parar --equipo l --instancia recorrido
deploy/principal/copia_local.sh arrancar --equipo l --instancia recorrido
deploy/principal/copia_local.sh borrar --equipo l --instancia recorrido
```

`reiniciar` detiene aplicación y PostgreSQL y los vuelve a crear con los
mismos datos. Comprueba `/livez` y `/portal-empleado/` por HTTPS con CA y
certificado de cliente verificados. `arrancar` rechaza una copia incompleta o
una revisión de código distinta; crea otra instancia cuando cambie la rama.
`borrar` comprueba el árbol de trabajo creador, la marca de propiedad y la
etiqueta del contenedor antes
de detener y eliminar únicamente su instancia. No actúa sobre puertos o
contenedores encontrados por coincidencia.

Los registros privados de restauración, SQL, compilación y aplicación y el
diario `sql_aplicado.json` quedan en la instancia. `estado.json` conserva la
revisión del código y los segundos empleados hasta obtener los dos 200.
En el primer ensayo local sin SQL adicional se midieron 72,17 segundos;
con B117 y comprobación de permisos, 64,95 segundos. En ambos se
repitieron los dos 200 después de reiniciar ambos procesos. Falta la prueba
de otro equipo y la preparación automática de los módulos para cerrar F0-28.

### Regenerar el clon de ensayo SQL

La copia antigua de septiembre carece de AD251. Para preparar su reemplazo,
crea una instancia nueva desde el volcado de octubre, con las listas ya
instaladas que deba tener tu base de ensayo. Después:

```sh
deploy/principal/copia_local.sh exportar-clon --equipo l --instancia recorrido \
  --archivo-clon "$VEC_COPIA_CLON_DESTINO"
```

`VEC_COPIA_CLON_DESTINO` es una ruta absoluta nueva, fuera de Git y de la
instancia. La exportación publica el archivo solo al terminar correctamente. Detiene
ambos procesos y empaqueta el directorio
de PostgreSQL en frío; deja la copia parada. El archivo contiene datos y
material privado: no se sube a Git. La extracción también necesita un
contenedor con 2 GiB; comprueba su resultado antes de intentar arrancarlo. Se puede usar como `VEC_CLON_ESTADO` con
el ensayo local que ya acepta esa variable. No se sustituye su clon
compartido ni se cambia el valor predeterminado del ensayo. Se restaura cada
archivo de ensayo en una carpeta propia y se aplica una sola vez la lista
de SQL nueva; para repetir se vuelve a extraer el clon frío.

## 3. Poner la copia al día con las SQL de `main`

Usa esta lista: `deploy/principal/lista_sql_main_tras_hz37.txt`. Trae una
sola vez, con su huella, cada migración añadida a `main` después de
`44218865a`, en el orden en que se fusionaron sus PR y, dentro de cada PR, en
el orden de su propia lista. Se pasa sola, con un único `--lista-sql`; no la
combines con las listas antiguas de cada PR, que repiten migraciones y harían
que el guion rechazara todo. Toda PR que añada migraciones añade sus líneas
al final de esta lista en la misma PR, con la ruta y su SHA-256.

Los dos roles sin LOGIN que necesita B119 los crea ya el guion (F0-31, apartado 2);
no hace falta crearlos a mano.

Lo que sigue explica de dónde sale la lista y cómo se hacía a mano hasta el
18/10. Sirve para comprobarla, no para aplicar otra cosa.

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
| 1 | `candidata_hz39/lista_sql.txt` | Las once de la candidata (confianza externa, cargos y plan de firma, organización de la firma, informe firmado, detalle RRHH). No trae las cinco de la firma: van en la lista 1b |
| 1b | `candidata_hz39/lista_sql_firma_vigencias.txt` | CT218, IS20, AUT97, CT219 y CT224 (firma de tres pasos, #1137). Las mismas que `lista_sql_codexh_firma_vigencias_20261014.txt`: aplica solo una de las dos, una vez |
| 2 | ninguna | AD266, secuencia global de confianza ADMIN (#1134). La necesita el gobierno de usuarios ADMIN (apartado 4). P la aplicó después de las once primeras *(por verificar si debe ir antes de CT218)* |
| 3 | `candidata_hz39/personal/lista_sql.txt` | Ocho de Personal y Bolsa nativos (alta y cese desde la bolsa). Exige antes B110 y AD261, que deberían estar en HZ37 *(por verificar)* |
| 4 | `lista_sql_codexs_varias_plazas_20261015.txt` | B117, ofertas con plazo desde la publicación (#1145) |

Corrección del 16/10 (T): la lista 1 tiene once SQL; las cinco de la firma
están solo en la lista 1b y se aplican aparte, sin duplicarlas.
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

## 9. Reinicio y recuperación

Usa `reiniciar` del apartado 2 para la copia creada por el guion. La orden
conserva PostgreSQL y su material, y repite las comprobaciones HTTPS.
Después de preparar los módulos, el ensayo del recorrido debe recuperar
sus lecturas y recibos con las mismas claves de operación: mismas fechas y
versiones, sin duplicados. Los dos 200 de arranque no sustituyen esa prueba.

Si se apagó el equipo, `/dev/shm` está vacío: crea otra instancia con el
mismo volcado y las listas necesarias.

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
| La consulta de firmas da 503 con «autenticación histórica inválida» | Falta IS20 | Aplicar la lista 1b |
| El contenedor de la app no arranca al montar `/app` en solo lectura y `/app/var` en escritura | Falta la carpeta `var/` en tu carpeta de trabajo | Crea `var/` en tu carpeta de trabajo antes de montar (lo vio T el 16/10) |
| Rechazo de una fecha que es el mismo instante | Representación distinta: AUT32 compara texto con el canon nominal; CA25 y los motivos de Organización histórica exigen seis decimales | Generar las fechas en el formato canónico de cada guarda; corregir el dato de la copia |
| Los datos de empleados firmantes piden 400 días de relación cubierta | Las asignaciones de la copia duran 90 días | Ajustar en la copia las relaciones a las vigencias existentes (P usó 89 días por SQL) |
| Firma del paso 3 rechazada por unidad | CT224 solo admite el paso 3 con unidad y paso publicados en el plan vigente | Publicar el plan con esa unidad |
| `free -g` por debajo de 34 GB | Copias o cachés acumuladas | No restaurar ni compilar; borrar lo propio y avisar |
| `git worktree move` falla hacia `/dev/shm` | Cambio de dispositivo | Copiar la carpeta y `git worktree repair` |
