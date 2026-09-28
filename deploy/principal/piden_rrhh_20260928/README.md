# Preparación de la tanda PIDEN RRHH para cidonia

Punto de partida: `trabajo/piden-rrhh-20260927` en
`cba5aa618372f178cd269858ca6454ffb9d86fca`. Esta carpeta **no ejecuta
un despliegue**. Dirección integra en la rama canónica y decide la puesta en
servicio. El script histórico `deploy/principal/desplegar.sh` hace `checkout
main`, `pull`, reaplica SQL D6 y cambia contenedores: **no usarlo para esta
tanda**. Tampoco repetir `00_puesta_al_dia.sh`, `02_migraciones.sh`, F4/D7 ni
ningún `DOWN` sobre la principal con historia.

La lista [`migraciones.txt`](migraciones.txt) contiene **21 `UP` candidatos**
frente a `origin/main` en este corte, en orden de introducción causal. Incluye
AD3-90/Bolsa44, cuya preimagen PostgreSQL 18 y dos revisiones deben quedar
acreditadas antes de instalarse. El inventario de cidonia puede mostrar que
alguna ya tiene historia; **detenerse y conciliar** en vez de repetirla. La
lista se invalida si se añaden, retiran o modifican migraciones del candidato.
No equivale a una aprobación de SQL ni a un inventario de la base real.

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
4. Validar las dos listas `web/produccion.manifest` y
   `web/interno.manifest`. `mi-bolsa-historial.js` es importado por el área
   personal y debe figurar en ambas. `preparar_paquete.sh` falla expresamente
   si falta. Revisar que no haya más imports dinámicos ausentes.

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
sha256sum "$EVIDENCIA_PRIVADA/principal.dump" \
  "$EVIDENCIA_PRIVADA/globals.sql" >"$EVIDENCIA_PRIVADA/dumps.sha256"
```

`globals.sql` puede contener verificadores de contraseñas: custodiarlo como
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
```

`CLON_OWNER`, codificación, locale y extensiones se toman del inventario real;
si el destino aislado no los reproduce, detener el ensayo. No ejecutar estos
comandos apuntando a cidonia. Los servicios de origen y clon se configuran
fuera de Git y deben tener host/puerto distintos, cotejados antes de restaurar.

Configurar `PGSERVICE` para la base restaurada cuyo nombre termina literalmente
en `_clon_piden_20260928`; comprobar que el servicio resuelve al clúster
aislado. El ensayo se ejecuta **una vez** sobre un clon nuevo:

```bash
export PGSERVICE=piden_clon
export VEC_PIDEN_CLON_DB=vec_clon_piden_20260928
bash deploy/principal/piden_rrhh_20260928/ensayar_clon.sh --aplicar-en-clon
```

El guion verifica nombre de base y PG18, y aplica los `UP` por orden con
`ON_ERROR_STOP=1`. Cada fichero canónico abre y confirma su propia transacción;
por eso no se simula un `ROLLBACK` exterior que no protegería nada. Si falla,
descartar el clon, corregir la causa y restaurar otro: **nunca hacer `DOWN`**.
Comprobar postimagen estructural, ACL, permisos negativos, recibos y contadores
testigo. Reiniciar solo la aplicación **aislada** contra el clon, repetir las
consultas y comprobar mismos recibos, versiones y ausencia de duplicados.
La conexión administrativa del ensayo no acredita los permisos de los LOGIN
de aplicación; probar cada rol real en el clon.

## Artefacto local y cambio controlado por Dirección

Una vez corregidos los manifiestos y con el checkout limpio:

```bash
bash deploy/principal/piden_rrhh_20260928/preparar_paquete.sh
```

Imprime `PAQUETE_LOCAL=/tmp/vec-piden-20260928...`. Construye `vec-server`,
sincroniza **todo** `web/` mediante `rsync -a --delete` dentro de una carpeta
temporal nueva, incluye manifiestos y escribe `SHA256SUMS` y hash del commit.
No incluye configuración privada. Comprobar `sha256sum -c SHA256SUMS` y que el
binario/activos proceden del hash publicado. El paquete es local y temporal;
Dirección lo copia al directorio privado de releases del servicio. No usar
`--delete` sobre la raíz de material privado ni sobre el directorio de estado.

En la ventana de cambio: detener solo la aplicación principal autorizada,
conservar binario y `web/` actuales con huellas y permisos, tomar nueva copia
de seguridad de base y ACL, aplicar en la **principal** únicamente los `UP`
inventariados como pendientes, en el orden de `migraciones.txt`, y verificar
postimagen tras cada `COMMIT`. La aplicación de producción es una decisión
manual de Dirección después del ensayo y las revisiones; este directorio no
contiene un comando que la ejecute. Sincronizar `web/` y manifiestos del
paquete al directorio de release con `rsync -a --delete` dentro de esa carpeta
dedicada, instalar binario de forma atómica, y activar esa release conservando
la anterior. Mantener configuración privada fuera de la release y comprobar
permisos antes del arranque.

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
