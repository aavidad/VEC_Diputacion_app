# Segundo corte PIDEN RRHH para cidonia

Fuente fijada: `trabajo/piden-rrhh-corte2-20260928` en
`da48a409b7e75249fa1b2378d612a90998aa25bd`. Incorpora CT130, Bolsa47
y las rutas nominales de Auditoría sobre el primer corte. La puerta completa
del hash fuente y dos revisiones del hash de este paquete son requisitos antes
de exportar o anunciar la entrega. Los guiones comprueban
que el árbol de producto coincide con ese hash, salvo esta carpeta de entrega;
un cambio de código, SQL o web exige revisar y fijar otro hash. Esta carpeta **no ejecuta
un despliegue**. Dirección integra en la rama canónica y decide la puesta en
servicio. El script histórico `deploy/principal/desplegar.sh` hace `checkout
main`, `pull`, reaplica SQL D6 y cambia contenedores: **no usarlo para esta
tanda**. Tampoco repetir `00_puesta_al_dia.sh`, `02_migraciones.sh`, F4/D7 ni
ningún `DOWN` sobre la principal con historia.

La lista [`migraciones.txt`](migraciones.txt) contiene **27 `UP` candidatos**
frente a la base publicada anterior al primer corte, en orden de introducción
causal. Son el mismo conjunto SQL del primer corte: si una migración o un rol
ya tiene historia en destino, **no reaplicarlo**. Incluye
AD3-98/CT134, Bolsa54 y AD3-99 antes de CT135. Incluye también
AD3-90/Bolsa44, cuya preimagen
PostgreSQL 18 y dos revisiones deben quedar
acreditadas antes de instalarse. El inventario de cidonia puede mostrar que
alguna ya tiene historia; **detenerse y conciliar** en vez de repetirla. La
lista se invalida si se añaden, retiran o modifican migraciones del candidato.
No equivale a una aprobación de SQL ni a un inventario de la base real.
[`validar_plan.sh`](validar_plan.sh) compara los **27 UP y dos deltas DBA de
rol** con el delta Git exacto de la base publicada a la fuente fijada, y exige
el orden causal AD3, CT y Bolsa antes de ensayar o empaquetar. Incluye
AD3-96 antes de CT133, `roles_calculador_politica_up.sql` inmediatamente
antes de Bolsa51 y `roles_registrador_auditoria_up.sql` inmediatamente antes
de CT136. Los dos deltas se ensayan con `ROLLBACK` y se confirman por DBA
antes de su migración respectiva. Los LOGIN y sus membresías nominales se
aprovisionan por el procedimiento privado, nunca desde este paquete.
`probar_paquete.sh` comprueba de forma local y sin PostgreSQL real que el
validador rechaza ambas omisiones de rol y la inversión AD3-96/CT133, y que
los preflights rechazan B49, Pública3 y ACL simuladas incompatibles. No
sustituye el ensayo sobre un clon PG18 íntegro.

**NO-GO B10:** Bolsa `000049_publicacion_cese_b10` y Bolsa pública
`000003_publicacion_cese_replay` quedan fuera del plan ejecutable. El
publicador separado puede acusar el evento sin probar la publicación y la
fuente V2/documental B10 no está compuesta. No instalarlas ni en el clon de
esta tanda ni en cidonia. Su presencia en el árbol Git no acredita uso. Exigen
corrección, prueba y revisión independiente con `GO`, seguidas de un plan nuevo;
los guiones rechazan esas rutas si reaparecen en `migraciones.txt`.
Si el binario nuevo o la sonda B10 requieren cualquiera de las dos, la
activación de esta tanda es **NO-GO**; no suplirlas con una instalación parcial
ni presentar B10 como publicado.
Antes de aplicar cualquier UP, [`preflight_no_go.sh`](preflight_no_go.sh) debe
confirmar también que B49/Pública3 **no están ya instaladas** en las bases
principal y pública de destino/clon. Busca sus tablas, funciones y rol; una
huella encontrada o una conexión no inventariable detiene toda la tanda.

## Puertas antes de tocar un servicio

1. Fijar el hash exacto que Dirección ha integrado y publicado. Comprobar
   `HEAD`, `origin/main`, rama remota y revisión independiente del hash final.
   Exigir puerta de Go, web y PostgreSQL 18 que corresponda a ese mismo árbol.
2. Inventariar en la principal, en modo de solo lectura, versiones y postimagen
   instaladas de AD3, Bolsa, Bolsa pública y Contratación; roles, membresías,
   propietarios, `SECURITY DEFINER`, ACL explícitas y predeterminadas, tipos de
   fila, recibos y contadores testigo. Comparar con cada precondición de la lista.
   Conservar el informe privado fuera de Git. No asumir que el número de versión
   más alto implica que las anteriores están instaladas. Dejar B49/Pública3
   expresamente fuera de la lista de aplicación aun si figuran como archivos.
3. Confirmar que las credenciales, certificados, perfiles, políticas V3,
   catálogos y fuentes externas necesarios existen en el material privado del
   servicio, con permisos restrictivos. No copiar, imprimir ni registrar sus
   valores; un secreto o configuración ausente es **NO-GO**. El consenso de
   dudas deja reglas de ejemplo configurables y pendientes de ratificación RRHH;
   el paquete no convierte avisos, borradores ni autenticación en firma,
   notificación acreditada o decisión legal.
   Este corte requiere además los LOGIN nominales separados de cálculo de
   ofertas, fuente y motivos de Auditoría y registro de frontera; sus DSN
   privados corresponden a `VEC_BOLSA_POLITICA_OFERTAS_CALCULADOR_DATABASE_URL`,
   `VEC_RRHH_AUDITORIA_FUENTE_AUTORIZACION_DATABASE_URL`,
   `VEC_RRHH_AUDITORIA_MOTIVOS_DATABASE_URL` y
   `VEC_RRHH_AUDITORIA_FRONTERA_DATABASE_URL`. Sin material, rol o permiso
   exacto, el arranque debe fallar cerrado. No incluir valores en Git o logs.
4. Validar `web/produccion.manifest` y `web/interno.manifest`.
   `mi-bolsa-historial.js` es importado por el área personal y debe figurar en
   **producción**; el inventario interno no incluye el área personal.
   `preparar_paquete.sh` falla expresamente si falta en producción. Revisar
   que no haya más imports dinámicos ausentes. El ZIP OSM público declarado
   por ambos manifiestos se aprovisiona con
   `scripts/aprovisionar_cartografia_osm.sh`, que verifica su SHA-256 fijado;
   está ignorado por Git y no aparece en un checkout nuevo. Sin ese ZIP
   verificado, el empaquetado se detiene.

## Clon exacto y ensayo PG18

Trabajar sobre un clúster **aislado y desechable** PostgreSQL 18. El origen se
consulta sin pararlo; el plan se aplica solo al clon. Usar ficheros privados
`PGSERVICEFILE`/`PGPASSFILE` con modo `0600`, nunca una URL con contraseña en
argumentos, Git o salida. Directorio de evidencia privado `0700` fuera del
repositorio. En el servidor, con nombres de servicio privados ya definidos:

```bash
umask 077
mkdir -m 700 -p "$EVIDENCIA_PRIVADA"
PGSERVICE=piden_principal pg_dump -Fc --file "$EVIDENCIA_PRIVADA/principal.dump"
PGSERVICE=piden_principal pg_dumpall --globals-only \
  --file "$EVIDENCIA_PRIVADA/globals.sql"
PGSERVICE=piden_bolsa_publica pg_dump -Fc \
  --file "$EVIDENCIA_PRIVADA/bolsa-publica.dump"
# Si Bolsa pública está en otro clúster, conservar allí sus globales por separado.
PGSERVICE=piden_bolsa_publica pg_dumpall --globals-only \
  --file "$EVIDENCIA_PRIVADA/globals-bolsa-publica.sql"
sha256sum "$EVIDENCIA_PRIVADA/principal.dump" \
  "$EVIDENCIA_PRIVADA/globals.sql" \
  "$EVIDENCIA_PRIVADA/bolsa-publica.dump" \
  "$EVIDENCIA_PRIVADA/globals-bolsa-publica.sql" \
  >"$EVIDENCIA_PRIVADA/dumps.sha256"
```

Los ficheros `globals*.sql` pueden contener verificadores de contraseñas:
custodiarlos como
secreto. Acreditar además con consultas de catálogo y salidas privadas las ACL
explícitas y predeterminadas, roles, membresías, tipos de fila y la postimagen
de funciones: el volcado lógico por sí solo no prueba esa equivalencia. En el
clúster aislado, restaurar **primero** globales, crear la base nueva desde
`template0` con propietario y codificación cotejados, y restaurar con
`pg_restore --exit-on-error --single-transaction`. Verificar que el restore
termina con código 0, sin omitir ACL ni propietarios. Comparar inventario,
recibos y contadores testigo antes de aplicar migraciones.

```bash
# PGSERVICE=piden_clon_admin apunta a la base postgres del clúster aislado.
export PGSERVICE=piden_clon_admin
export VEC_PIDEN_CLON_DB=vec_clon_piden_20260928
psql -X -v ON_ERROR_STOP=1 -f "$EVIDENCIA_PRIVADA/globals.sql"
createdb --template=template0 --owner="$CLON_OWNER" "$VEC_PIDEN_CLON_DB"
pg_restore --exit-on-error --single-transaction \
  --dbname="$VEC_PIDEN_CLON_DB" "$EVIDENCIA_PRIVADA/principal.dump"
# Repetir en el clúster/base pública aislados. Si comparten clúster,
# no restaurar globales una segunda vez.
export PGSERVICE=piden_bolsa_publica_clon_admin
export VEC_PIDEN_BOLSA_PUBLICA_CLON_DB=bolsa_publica_clon_piden_20260928
psql -X -v ON_ERROR_STOP=1 -f "$EVIDENCIA_PRIVADA/globals-bolsa-publica.sql"
createdb --template=template0 --owner="$CLON_PUBLICA_OWNER" \
  "$VEC_PIDEN_BOLSA_PUBLICA_CLON_DB"
pg_restore --exit-on-error --single-transaction \
  --dbname="$VEC_PIDEN_BOLSA_PUBLICA_CLON_DB" \
  "$EVIDENCIA_PRIVADA/bolsa-publica.dump"
```

`CLON_OWNER`, codificación, locale y extensiones se toman del inventario real;
si el destino aislado no los reproduce, detener el ensayo. No ejecutar estos
comandos apuntando a cidonia. Los servicios de origen y clon se configuran
fuera de Git y deben tener host/puerto distintos, cotejados antes de restaurar.

Configurar los dos servicios para bases restauradas cuyos nombres terminan
literalmente en `_clon_piden_20260928`; comprobar que resuelven a los
clústeres aislados. El preflight lee ambas bases y aborta si detecta B49 o
Pública3 ya instaladas, incluso si no figuran en el plan. El ensayo se ejecuta
**una vez** sobre un clon nuevo:

```bash
export PGSERVICE=piden_clon
export VEC_PIDEN_CLON_DB=vec_clon_piden_20260928
export VEC_PIDEN_BOLSA_PUBLICA_PGSERVICE=piden_bolsa_publica_clon
export VEC_PIDEN_BOLSA_PUBLICA_CLON_DB=bolsa_publica_clon_piden_20260928
bash deploy/principal/piden_rrhh_corte2_20260928/preflight_no_go.sh --clon
bash deploy/principal/piden_rrhh_corte2_20260928/preflight_roles_calculador.sh --clon
bash deploy/principal/piden_rrhh_corte2_20260928/preflight_roles_ct136.sh --clon
bash deploy/principal/piden_rrhh_corte2_20260928/ensayar_clon.sh --aplicar-en-clon
```

El guion verifica nombre de base y PG18, y aplica los `UP` por orden con
`ON_ERROR_STOP=1`; antes ensaya ambos deltas de rol con `ROLLBACK` y
comprueba sus preimágenes y ACL. Cada fichero canónico abre y confirma su propia transacción;
por eso no se simula un `ROLLBACK` exterior que no protegería nada. Si falla,
descartar el clon, corregir la causa y restaurar otro: **nunca hacer `DOWN`**.
Comprobar postimagen estructural, ACL, permisos negativos, recibos y contadores
testigo. Reiniciar solo la aplicación **aislada** contra el clon, repetir las
consultas y comprobar mismos recibos, versiones y ausencia de duplicados.
La conexión administrativa del ensayo no acredita los permisos de los LOGIN
de aplicación; probar cada rol real en el clon.

## Artefacto local y cambio controlado por Dirección

Con el ZIP OSM verificado y el checkout limpio:

```bash
bash deploy/principal/piden_rrhh_corte2_20260928/preparar_paquete.sh
```

Imprime `PAQUETE_LOCAL=/dev/shm/vec-piden-corte2-20260928...`. Construye `vec-server`,
sincroniza por `rsync` **solo las rutas** de `web/produccion.manifest` dentro
de una carpeta temporal nueva y pasa `scripts/verificar_web_produccion.sh` al
árbol extraído. Copia también los 27 `UP` y los dos deltas DBA de rol exactos,
con sus rutas relativas, y conserva los demás manifiestos en `evidencia/`.
Escribe `SHA256SUMS` y hash del commit. Exige Go 1.26.9 linux/amd64,
igual que el Dockerfile, y usa una caché Go temporal dentro
de la carpeta del paquete, que se retira antes de calcular las huellas. Usa
`CGO_ENABLED=0 GOOS=linux GOARCH=amd64`, `-trimpath` y `-ldflags='-s -w'`
como el Dockerfile del servidor; `file` y `ldd` deben confirmar un ELF amd64
sin dependencias dinámicas.
`evidencia/fuente_commit.txt` fija el segundo corte de producto;
`evidencia/commit.txt` identifica el commit de este paquete.
No incluye configuración privada. Comprobar `sha256sum -c SHA256SUMS` y que el
binario/activos proceden del hash publicado. El paquete es local y temporal;
Dirección lo copia al directorio privado de releases del servicio. No usar
`--delete` sobre la raíz de material privado ni sobre el directorio de estado.

En la ventana de cambio: detener solo la aplicación principal autorizada,
conservar binario y `web/` actuales con huellas y permisos, tomar nueva copia
de seguridad de ambas bases y ACL. Configurar `PGSERVICE` y
`VEC_PIDEN_BOLSA_PUBLICA_PGSERVICE` para sus servicios reales, junto con
`VEC_PIDEN_DESTINO_DB` y `VEC_PIDEN_BOLSA_PUBLICA_DESTINO_DB`; ejecutar
`preflight_no_go.sh --destino` y conservar su salida privada. Cualquier huella
B49/Pública3 significa **NO-GO**; no continuar por el resto del plan. Con
preflight limpio, ejecutar `preflight_roles_calculador.sh --destino` y
`preflight_roles_ct136.sh --destino` para ensayar ambos roles y ACL con
`ROLLBACK` exacto; un rechazo es **NO-GO**. Con revisiones
cerradas, aplicar en la **principal** únicamente los `UP`
inventariados como pendientes, en el orden de `migraciones.txt`, y verificar
postimagen tras cada `COMMIT`. La aplicación de producción es una decisión
manual de Dirección después del ensayo y las revisiones; este directorio no
contiene un comando que la ejecute. Sincronizar `web/` y manifiestos del
paquete al directorio de release con `rsync -a --delete` dentro de esa carpeta
dedicada, instalar binario de forma atómica, y activar esa release conservando
la anterior. Mantener configuración privada fuera de la release y comprobar
permisos antes del arranque.

En las parejas Bolsa51 y CT136, el primer fichero es un delta de **roles DBA**;
después se instala la migración del módulo con su propietario según el propio SQL. No crear
ni asignar LOGIN nominales desde este paquete. Una preimagen de rol o ACL
distinta aborta antes de CT136; no conceder privilegios amplios para forzarla.

Tras reiniciar **solo** la aplicación, exigir contenedor en ejecución y buscar
en sus logs posteriores al inicio la línea literal `vec server listening`.
Una ausencia o un error de bootstrap es fallo aunque el contenedor aparezca
activo. Ejecutar `deploy/principal/verificar.sh` con material mTLS privado y
sondas HTTP de las rutas nuevas bajo perfiles nominales y ajenos; conservar
códigos y recibos, sin cuerpos con datos personales en Git. Repetir tras
reinicio de aplicación y PostgreSQL solo en la ventana autorizada. La
recuperación exige iguales referencias, versiones, fechas y huellas y ninguna
alta duplicada.

En Chrome, a **1440 y 390 px**, recorrer RRHH (bandeja CT, plantillas,
auditoría, reincorporación, política/plazos de Bolsa) y área personal (Mi
Bolsa/histórico propio), con identidades sintéticas nominales. Registrar
HTTP y errores JS, carga de CSS/JS, overflow, cookies y almacenamiento web.
Comprobar denegación ajena y estados pendientes; no crear actos sintéticos
en la base principal sin plan de ensayo y autorización. Una sonda de lectura
no acredita las mutaciones ni el recorrido E2E.

## Reversión

Ante fallo de binario, activos, manifiestos o configuración, desactivar la
release nueva y reactivar el binario y `web/` anteriores con sus huellas;
reiniciar y repetir la sonda literal y los recibos testigo. **Conservar los
`COMMIT` SQL y su historia; no ejecutar `DOWN` ni restaurar un volcado sobre la
principal con escrituras posteriores.** Si el esquema nuevo impide arrancar el
binario anterior, mantener la aplicación detenida, aislar tráfico y resolver
con una migración correctiva revisada. Restaurar dump solo en un entorno nuevo
aislado para investigar o tras un procedimiento de recuperación de desastre
expresamente dirigido, con ACL y roles incluidos.

Estado al redactar: preparación local; sin clon, migraciones instaladas,
servicio cambiado, Chrome ni publicación acreditados por este paquete.
