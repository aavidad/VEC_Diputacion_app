# Clon local para comprobar recorridos

**Documento histórico.** Conserva el diseño anterior al guion H6 actual. El
comando `plan` mostrado abajo ya no acepta esa invocación sin entradas, y la
fase H1→SQL62 sí está conectada mediante `preparar-sql` y `verificar-sql`.
Para reconstruir o comprobar una copia use la [guía H6 vigente](README_CLON_H6.md).
Las secciones siguientes explican decisiones de aquel corte; sus comandos no
son instrucciones para el clon actual.

La instalación y el arranque están retenidos. `preparar`, `reiniciar` y la
publicación de `READY.json` rechazan la operación antes de restaurar H1,
acceder a Docker, ejecutar SQL o iniciar la aplicación. Falta conectar el kit
D final, aprobado y revisado, con la preimagen que le corresponde.
`LIVE_KIT=None` en `clon_sql.py` confirma esa dependencia. Cambiar una variable
o aportar un JSON privado no habilita el preparador.

La fuente H6 está fijada en
`73e56c106d12fdda0bd16d6fe573503c42c5495f`. El kit final debe conservar esa
procedencia para el binario, los activos y las SQL. Un avance de `main` no
cambia la fuente del ensayo. El paquete D recibido inicialmente está en
regeneración: sus huellas anteriores no identifican el kit final.

## Lista de instalación y procedencia

El circuito previsto aplica las 45 SQL de `lista_sql_h6.txt` del kit D final,
en su orden y con sus bytes originales. AD3-132 es la operación 46. Se ejecuta
por la CLI DBA `aplicar_000132_temp_public.py`, después de las 45 SQL y antes
de arrancar la aplicación. Su evidencia es independiente de las 45
confirmaciones del instalador.

D confirmó la cadena para la copia local: H1, las 8 SQL de H3, las 9 de H4,
las 45 de la lista funcional exacta del kit H6 y AD3-132 por CLI. Las 17 SQL
previas preparan la preimagen que la lista funcional presupone; no forman
parte de sus 45 entradas. Desde H1 son 62 SQL funcionales en total, con
AD3-132 aparte. Aprobar el plan histórico `h6_45` cambiando una bandera no
lo convierte en esa cadena. El trabajo RPT etapa 0 continúa después con sus
seis operaciones propias y su autorización; este preparador no lo ejecuta.

El manifiesto histórico `sql_main_h6_firma.txt` también tiene 45 entradas,
pero mezcla ese prefijo H3/H4 con parte de H6 y no coincide con
`lista_sql_h6.txt`. Compartir fuente y número de entradas no permite sustituir
una lista por otra. La conexión pendiente debe respetar cada lista y su
preimagen. El preparador de aquel corte no aplicaba ninguno de esos planes.
La fase H1→SQL62 añadida después se describe en la guía H6 vigente.

La copia debe conservar los objetos, ACL, identidades, claves e historia del
origen acreditado, y permitir cotejar sus diferencias autorizadas tras el kit.
La identidad externa procede del H1 acreditado; no se crean identidades ni
permisos adicionales para hacer funcionar un recorrido. La existencia de un
archivo H1 o una huella calculada localmente no demuestra por sí sola que la
copia sea idéntica a la principal. Esa afirmación requiere procedencia H1,
lista final y comprobaciones del kit.

## Leer el plan histórico

El ejemplo siguiente corresponde al parser anterior. **No se ejecuta con el
guion actual**: hoy `plan` exige entradas externas y rechaza esta invocación.

```bash
VEC_RECORRIDOS_REFERENCIA=73e56c106d12fdda0bd16d6fe573503c42c5495f \
  bash scripts/recorridos/preparar_clon.sh plan
```

Muestra las 45 entradas del manifiesto histórico de `clon_sql.py`. No muestra
la lista final de D ni aprueba instalación, AD3-132 o arranque. La diferencia
con `lista_sql_h6.txt` debe quedar resuelta y revisada antes de conectar el
circuito de instalación.

## Diario externo y recuperación

`sql-journal.json` v2 queda fuera de PostgreSQL y de cualquier repositorio,
en un directorio privado `0700`, con archivos regulares propios `0600` y sin
enlaces. No existe un esquema `vec_recorridos_clon` ni SQL auxiliar de ledger.
Las únicas SQL de instalación serán los bytes originales de la lista aprobada.

El diario liga cada confirmación a la identidad del clon y las huellas de H1,
paquete, lista y release. Antes de enviar una SQL, el instalador guarda
`pending` de forma durable. Solo lo cierra tras observar el retorno de COMMIT
y comprobar la postimagen mediante consultas de lectura del proveedor D.

Un `pending`, una caída ambigua, un diario corrupto o perdido, o un diario v1
obligan a conservar la evidencia y reconstruir una copia nueva con origen
aprobado. No se recupera desde un ledger SQL, no se convierte v1 a v2 y no se
reaplica una SQL para averiguar si había terminado. No se ejecuta `DOWN`
sobre historia conservada.

El contrato actual de `clon_sql.py` expone las APIs
`apply(..., context=contexto_h1, kit=proveedor_d)` y
`verify_live(..., context=contexto_h1, kit=proveedor_d)`. El contexto contiene
`identidad_clon`, `estado_h1_sha`, `package_sha`, `list_sha` y `release_sha`,
obtenidos de la procedencia aprobada. No se deducen del nombre del contenedor
ni se inventan para completar un registro.

Ese contrato exige `validate`, `identity`, `confirm` y `verify` al proveedor D.
Valida la fuente y la lista, coteja la identidad viva y la preimagen H1,
confirma cada postimagen y comprueba las anclas finales y la evidencia de la
operación DBA AD3-132. La comprobación final no resuelve un `pending` ni
modifica la fase del diario. Hace falta además conectar de nuevo la preparación
de material y el runtime al contrato aprobado antes de poder publicar READY.

## Consultar y retirar una copia propia

Configure el directorio privado y el nombre exactos de la copia conservada:

```bash
export VEC_RECORRIDOS_ESTADO="$HOME/.local/state/vec-recorridos"
export VEC_RECORRIDOS_CONTENEDOR=vec-recorridos-local
bash scripts/recorridos/preparar_clon.sh estado
```

`estado` lee el registro de propiedad y el diario externo. Informa un diario
pendiente, antiguo o corrupto como bloqueado, y conserva sus bytes. No consulta
Docker ni SQL, no escribe archivos y siempre devuelve `ready: false` en este
corte. Si queda un `READY.json` antiguo, señala su presencia sin aceptarlo
como comprobación de disponibilidad.

`parar` y `retirar` siguen disponibles para los recursos propios. Comprueban el
registro privado, las etiquetas del contenedor y su volumen antes de actuar;
los helpers de aplicación y correo comprueban sus propios recursos. No actúan
sobre recursos ajenos. Coordine la retirada con quienes estén usando o
revisando la copia.

```bash
bash scripts/recorridos/preparar_clon.sh parar
bash scripts/recorridos/preparar_clon.sh retirar
```

`parar` detiene la aplicación, PostgreSQL y el correo propios y conserva el
volumen. `retirar` elimina el volumen temporal propio de `/dev/shm`, las
fuentes, los binarios y los logs reconocidos del preparador; marca el registro
como `RETIRADO.json`. Conserva material, diario, recibos, capturas y archivos
no reconocidos para revisión. La copia nueva necesita otro directorio privado
y el circuito D aprobado. El bloqueo actual también impide reconstruirla.

## Comprobar el preparador sin servicios

Las pruebas focales usan fixtures desechables y dobles locales. Comprueban el
rechazo previo a servicios, el plan de lectura, diarios pendientes y v1,
READY bloqueado, permisos privados y retirada limitada a recursos propios.
No prueban una instalación ni igualdad con la principal.

Ejecute dentro del sandbox local de pruebas, con red deshabilitada, fuente y
objetos Git montados como lectura, entorno vacío, temporal privado y límites
de recursos:

```bash
bash -n scripts/recorridos/preparar_clon.sh
python3 -m unittest scripts.recorridos.test_preparar_clon
```

El corte posterior conectó las 62 SQL exactas mediante `preparar-sql` y emitió
un recibo propio en una copia desechable. La guía H6 recoge esa comprobación y
su recuperación tras reiniciar PostgreSQL. El arranque, AD132 y los recorridos
de navegador permanecen pendientes.

El kit final para arrancar la aplicación, AD132 y el recorrido navegador con
recuperación siguen pendientes. Ninguna prueba simulada acredita esos pasos.
