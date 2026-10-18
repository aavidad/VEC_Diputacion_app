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
```

`--solo-base` comprueba únicamente la restauración del esquema. El segundo
comando instala, en orden, `deploy/principal/lista_sql_main_tras_hz37.txt`.
Tras restaurar y antes de cualquier preparación o migración, ambos recorren
todas las tablas ordinarias y vistas materializadas de `public` y los esquemas
`vec_`, incluidas las hojas de particiones. Excluyen objetos propios de
extensiones, cuentan las filas reales de cada relación y fallan si alguna
contiene datos. El resultado indica cuántas tablas se comprobaron.
La opción `--preparacion` ejecuta antes de la lista únicamente
`deploy/postgresql/fase1/preparacion_fase1.sql`, comprobando que coincide con
la versión confirmada en Git. La opción no se admite con `--solo-base`.
Ambos terminan con estado distinto de cero si falla una validación, el
arranque, la restauración o una migración. No ejecutan datos de prueba ni
acreditan todavía las pruebas SQL del recorrido de la fase 1.

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
