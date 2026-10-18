# Ensayo local del esquema HZ37

Este guion crea un PostgreSQL 18 desechable, restaura el esquema sin datos de
HZ37 y aplica la lista única de migraciones posteriores. Usa el archivo
`esquema_hz37.sql.gz` y comprueba las huellas y el tamaño que constan en
`esquema_hz37.json` antes de iniciar el contenedor. La extracción de roles
sin LOGIN y la comprobación de orden y huellas de migraciones reutilizan los
guiones de `deploy/principal/`.

Desde la raíz del repositorio:

```sh
python3 deploy/postgresql/fase1/ensayar.py --solo-base
python3 deploy/postgresql/fase1/ensayar.py --preparacion
python3 deploy/postgresql/fase1/ensayar.py --pruebas
```

`--solo-base` comprueba únicamente la restauración del esquema. El segundo
comando instala, en orden, `deploy/principal/lista_sql_main_tras_hz37.txt`.
Tras restaurar y antes de cualquier preparación o migración, ambos recorren
todas las tablas ordinarias y vistas materializadas de `public` y los esquemas
`vec_`, incluidas las hojas de particiones. Excluyen objetos propios de
extensiones, cuentan las filas reales de cada relación y fallan si alguna
contiene datos. El resultado indica cuántas tablas se comprobaron.
La opción `--preparacion` ejecuta primero
`deploy/postgresql/fase1/preparacion_fase1.sql`, comprobando que coincide con
la versión confirmada en Git. Repone la fila técnica del catálogo D14 que
falta en el volcado sin datos, después de verificar las 662 tablas vacías.
Después instala D15 una sola vez desde
`deploy/postgresql/fase1/lista_sql_base_hz37.txt`, antes de la lista general.
La lista propia y D15 deben coincidir por bytes con `HEAD`, y la huella de D15
debe ser la publicada en esa lista. La opción no se admite con `--solo-base`.

Aunque D15 ya estaba en Git antes del corte HZ37, el esquema de esa copia
conserva la función de D14. Esta instalación reconcilia solo la copia nueva de
CI; no cambia la lista general de migraciones ni reaplica D15 en otras bases.
`--pruebas` incluye esa preparación y ejecuta estas
siete pruebas SQL existentes, en este orden:

1. `bolsa_llamamientos/pruebas_sql/b118_criterios_sin_salto.sql`: valida criterios
   de ofrecimiento y el permiso del validador sin crear un llamamiento.
2. `documentos/pruebas_sql/000014_referencia_expediente_tipado.sql`: comprueba
   formatos, catálogo y rechazo de una referencia de otro módulo.
3. `documentos/pruebas_sql/000015_rechazo_expediente_ct_cero.sql`: rechaza la
   referencia de expediente CT con identificador cero.
4. `autorizacion_atestada_v3/pruebas_sql/ad180_rechazo_sin_consumo.sql`: comprueba
   el rechazo temprano 42501 ante una entrada nula.
5. `contexto_actor_v1/pruebas_sql/contexto_candidato_exacto_000013.sql`: publica
   una provisión de candidato con datos ficticios transaccionales; acepta su
   combinación exacta de persona, perfil y candidato, rechaza las tres
   combinaciones cruzadas y comprueba las restricciones de ejecución.
6. `identidad_sesiones_v1/pruebas_sql/000021_seleccion_perfil_bolsa_rrhh.sql`:
   comprueba propietario y permisos; rechaza la consulta de perfiles propios
   con una sesión inexistente.
7. `contexto_actor_v1/pruebas_sql/000046_perfiles_propios_bolsa_rrhh.sql`:
   comprueba propietario y permisos; rechaza una cuenta sin acreditación.

La provisión de candidato usa la función instalada y calcula la huella de sus
datos. Su preparación es una fixture SQL aislada, no una identidad certificada
ni una decisión completa de autorización. No sustituye funciones, no desactiva
disparadores y no persiste sus datos.

El guion comprueba que los bytes de cada prueba coinciden con `HEAD` y los pasa
a `psql` con `ON_ERROR_STOP=1`. Un fallo detiene el ensayo con estado distinto
de cero. Estas pruebas usan las funciones instaladas y sus transacciones
terminan en `ROLLBACK`. CT230 queda fuera porque exige confirmaciones O6
históricas, ausentes en la copia sin datos. Tampoco se acreditan aún la
autorización nominal positiva, la concurrencia, el reinicio ni el recorrido
completo: necesitan un expediente y autoridades nominales válidos.

Se necesita Docker y la imagen fijada `postgres:18.4-bookworm` disponible
localmente con la huella indicada en `ensayar.py`. El contenedor usa
`--rm --restart=no --memory=2g`, sin red ni puerto publicado. `PGDATA` queda
en una carpeta temporal propia bajo `/dev/shm`; puede elegirse otra raíz de
memoria con `VEC_PG18_DATOS_DIR`, que debe existir bajo `/dev/shm`. Antes de
empezar se exigen 34 GiB de memoria disponibles en local. En GitHub Actions
(`GITHUB_ACTIONS=true`) se exigen 3 GiB disponibles, pues el ejecutor tiene
menos memoria. En ambos casos hacen falta 2 GiB libres en esa raíz. El guion
retira su contenedor y su carpeta al terminar. Si PostgreSQL dejó los datos
con propietario interno, usa una ejecución breve de la misma imagen, sin red
y con 64 MiB de límite, para vaciar exclusivamente su carpeta de datos.

El esquema contiene propietarios y ACL que requieren roles previos. Los
roles extraídos y los dos de B119 se crean sin LOGIN. Los demás requisitos
de origen de AD278 se preparan con la semilla pública descrita abajo.

El archivo de esquema ocupa 1,49 MB comprimido (13,45 MB de SQL). Se extrajo
con PostgreSQL 18 mediante `pg_restore --schema-only --no-comments
--no-publications --no-subscriptions`, sin volcar roles globales. Antes de
incorporarlo se pasó Gitleaks local sobre el SQL descomprimido, sin hallazgos;
no contiene contraseñas, conexiones, claves privadas, rutas privadas ni
correos. Los nombres de propietario y destinatarios de permisos son técnicos.
La restauración se verifica contando las filas de las 662 tablas, antes de
aplicar cualquier preparación.

La preparación de CI crea solo el migrador sin LOGIN que necesita CTX45,
dos LOGIN ficticios sin contraseña, sus dos orígenes técnicos para AD278
y el tipo documental de resolución firmada definido por D9.
Sus vigencias son de 90 días; cada LOGIN recibe un único grupo. No concede
perfiles ni acciones de negocio. La base conserva las comprobaciones de las
migraciones y la seguridad por filas. Para aplicarla, el archivo de
preparación debe estar confirmado y coincidir con HEAD:

```sh
python3 deploy/postgresql/fase1/ensayar.py --preparacion
```

La semilla repone también la fila de control de punteros de CTX2, con
generación cero, después de comprobar que las 662 tablas están vacías y que
la tabla pertenece a su propietario previsto. Es el contador técnico que
creó CTX2; el volcado de solo esquema no conserva esa fila. Se mantienen sus
disparadores y se comprueba la postcondición antes de continuar.
