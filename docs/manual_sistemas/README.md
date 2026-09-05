# Manual de Sistemas — operación diaria de VEC

Entorno de desarrollo de Contratación temporal · Corte: 5 de septiembre de 2026.

**Cierre anterior publicado:** código
`b2effbaf09fd4ad8477bf42c56e4615ff52d0c62`, con desarrollo en el equipo local.
Dirección confirmó producto local, GitHub y producto remoto en ese mismo
commit, limpios y sincronizados (`0/0`). El corrector SQL `13f7a92` y la
interfaz ya están integrados por avance directo.
Consulte primero el bloque inicial de la
[guía vigente](../../GUIA_RECORRIDO_ALBERTO.md): ambas bases y el material están
conservados aquí; las instancias remotas quedan detenidas. La instalación de
consultas está completada en ambas bases. Cada aplicación usa once conexiones
a su propia base, con identidades nominales distintas.

| Uso | Modalidad del lanzador local | HTTPS | PostgreSQL |
| --- | --- | --- | --- |
| Principal: recorrido de RRHH | `recorrido` | `8443` | `55433` |
| Secundario: consultas aisladas | `consultas` | `8444` | `55432` |

En 8443, con ese recorrido publicado, dirección confirmó bandeja de 50 expedientes,
detalle de una solicitud versión 1 y análisis real con recibo versión 2,
sin crear otra alta. **Tras reinicio, la lista conserva 50 expedientes y el
detalle la versión 2**. Una lectura independiente confirmó un único recibo,
asiento y terminal de ese análisis. Ese cierre corresponde a bandeja y análisis.

**Incluido en esta entrega; recuperación demostrada:**
dirección comprobó el registro de declaración RRHH mediante `HTTP 201`.
Corregido el defecto temporal en ambas bases y repetido el reinicio de aplicación
y PostgreSQL, el navegador recuperó selección, comunicación y respuesta con
`200/200/200`: mismos recibo, justificante y fecha, sin nuevo registro. Sin
errores JavaScript, cookies, almacenamiento web ni desbordamiento horizontal
en ese recorrido. También se confirmó un conflicto real `409` en navegador.
La entrega 3 queda cerrada funcionalmente en desarrollo.
Los identificadores observados constan en el
[manual de RRHH](../manual_rrhh/README.md).

## Alcance y referencias

Este manual sirve para operar **un entorno sintético ya preparado**, no para
instalar otro, conceder permisos, migrar bases o habilitar producción. Toma como
base la [guía del recorrido vigente](../../GUIA_RECORRIDO_ALBERTO.md) y los scripts
[arrancar VEC](../../scripts/arrancar_vec_desarrollo.sh) y
[generar o verificar credenciales](../../scripts/generar_credenciales_desarrollo.sh).

El [manual de preparación de plataforma](../estudio_requisitos/manual_sistemas_preparacion_plataforma.md)
es **planificación pendiente de aprobación, no la instalación vigente**.
No hay que desplegar su inventario para realizar estas operaciones.

Alcance funcional acreditado: **cinco pasos completos y parte del sexto**,
con selección, aviso local y declaración RRHH recuperables tras reiniciar.
La declaración no resuelve aceptación ni renuncia terminal. Lista, detalle y
análisis desde una solicitud existente están comprobados y publicados;
su persistencia tras reinicio también está contrastada.
Consulte el
[manual de RRHH](../manual_rrhh/README.md) para interpretar cada recibo.
Ninguna respuesta de salud, publicación en GitHub o credencial de desarrollo
equivale a autorización productiva, firma oficial o aceptación de RRHH.

## 1. Comprobaciones al comenzar

| Requisito | Comprobación y límite |
| --- | --- |
| Código aprobado | Hash y árbol de producto identificados por dirección, sin cambios pendientes. No arrancar el trabajo compartido como si estuviera publicado. |
| Go | Base verificada: **Go 1.26.5**, declarada también en `go.mod`. Registrar la versión efectiva; no descargar ni sustituir herramientas por este manual. |
| PostgreSQL | Base verificada: **PostgreSQL 18.4**, instancia existente e imagen fijada por digest según la guía. No usar una etiqueta flotante ni recrear contenedores. |
| Herramientas | Bash, OpenSSL, curl, Python 3 y utilidades GNU usadas por los scripts; Git para identificar la entrega. Docker para la instancia descrita en la guía. |
| Material y red | Material persistente fuera de Git, CA de PostgreSQL correcta, certificado de cliente vigente y puerto libre en bucle local. |
| Separación | Principal `8443/55433`; secundaria `8444/55432`. No ejecutar pruebas contra la base del navegador; en 55432 tampoco mientras la usa la aplicación secundaria. No redirigir conexiones para eludir fallos. |

Los valores `/RUTA/…`, `<LOGIN>` y demás marcadores deben sustituirse por
los del entorno autorizado. **No son rutas reales ni instrucciones para crear
recursos nuevos.** Use una terminal de operación sin trazado de comandos
(`set +x`), no una terminal de otro agente.

Variables de referencia, sin secretos:

```bash
set +x
vec_ops_repo='/RUTA/ABSOLUTA/AL/PRODUCTO_APROBADO'
vec_ops_material='/RUTA/ABSOLUTA/AL/MATERIAL_EXISTENTE'
vec_ops_ca_pg='/RUTA/ABSOLUTA/A/LA/CA_POSTGRESQL.crt'
vec_ops_pg_contenedor='<CONTENEDOR_EXISTENTE_DEL_NAVEGADOR>'
vec_ops_http_puerto=8443

git -C "$vec_ops_repo" rev-parse HEAD
git -C "$vec_ops_repo" status --short
go version
docker inspect "$vec_ops_pg_contenedor" \
  --format 'estado={{.State.Status}} imagen={{.Config.Image}} puertos={{json .HostConfig.PortBindings}}'
docker exec "$vec_ops_pg_contenedor" postgres --version
```

El hash debe ser el aprobado, la instancia la preservada y el puerto de
PostgreSQL estar publicado solo en `127.0.0.1`. Si algo difiere, documente el
dato y deténgase antes de arrancar. No limpie trabajo ajeno ni cambie de rama
para hacer coincidir el resultado.

## 2. Configuración local: once conexiones por instancia

Cada variable contiene una cadena de conexión PostgreSQL, también llamada
DSN. Deben llegar por el mecanismo privado aprobado para el entorno, nunca
por un archivo versionado, captura, incidencia o salida de `env`.

| Variable | Función de la identidad separada |
| --- | --- |
| `VEC_CT_DATABASE_URL` | Ejecutar operaciones de Contratación temporal. |
| `VEC_CT_GOBIERNO_DATABASE_URL` | Gobierno de desarrollo requerido por la composición; no es la identidad de ejecución. |
| `VEC_CT_CONFIRMADOR_DATABASE_URL` | Confirmar cobertura con su consumidor autorizado. |
| `VEC_CT_LECTOR_RESULTADO_DATABASE_URL` | Consultar resultados de cobertura con su lector autorizado. |
| `VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL` | Registrar decisiones de autorización. |
| `VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL` | Operar el llamamiento mediante la identidad propia de Bolsa. |
| `VEC_CT_CONSULTAS_RRHH_DATABASE_URL` | Consultar lista y detalle con registro de acceso. |
| `VEC_CT_MOTIVOS_RRHH_DATABASE_URL` | Resolver los motivos publicados de consulta; no los inventa la aplicación. |
| `VEC_CT_REGISTRO_IDENTIDAD_DATABASE_URL` | Registrar la sesión nominal breve ligada al certificado de la petición. |
| `VEC_CT_REVALIDACION_IDENTIDAD_DATABASE_URL` | Revalidar esa sesión mediante una identidad separada. |
| `VEC_CT_CONTEXTO_ACTOR_DATABASE_URL` | Resolver el contexto de actor vigente mediante el servicio existente. |

Forma orientativa, **no ejecutable ni portadora de una contraseña**:

```text
postgresql://<LOGIN_NOMINAL>@localhost:55433/<BASE_EXISTENTE>?sslmode=verify-full&sslrootcert=<RUTA_CA_POSTGRESQL>
```

El ejemplo corresponde al principal; la secundaria usa `55432`. Las once
conexiones de un proceso deben señalar su misma base, nunca mezclar ambas.
Los once usuarios deben conservar sus roles nominales distintos. No basta
variar el texto de una conexión usando la misma identidad; no se sustituyen
por un superusuario. La CA de PostgreSQL no se confunde con la CA HTTPS del
portal. Mantenga `sslmode=verify-full`; un fallo de certificado no se resuelve
desactivando la validación.

Compruebe presencia sin imprimir los valores, después de cargarlos por el
mecanismo privado:

```bash
set +x
: "${VEC_CT_DATABASE_URL:?Falta la conexión de ejecución}"
: "${VEC_CT_GOBIERNO_DATABASE_URL:?Falta la conexión de gobierno}"
: "${VEC_CT_CONFIRMADOR_DATABASE_URL:?Falta la conexión de confirmación}"
: "${VEC_CT_LECTOR_RESULTADO_DATABASE_URL:?Falta la conexión de lectura}"
: "${VEC_CT_REGISTRO_AUTORIZACION_DATABASE_URL:?Falta la conexión de autorización}"
: "${VEC_BOLSA_LLAMAMIENTOS_DATABASE_URL:?Falta la conexión de Bolsa}"
: "${VEC_CT_CONSULTAS_RRHH_DATABASE_URL:?Falta la conexión de consultas RRHH}"
: "${VEC_CT_MOTIVOS_RRHH_DATABASE_URL:?Falta la conexión de motivos RRHH}"
: "${VEC_CT_REGISTRO_IDENTIDAD_DATABASE_URL:?Falta la conexión de registro de identidad}"
: "${VEC_CT_REVALIDACION_IDENTIDAD_DATABASE_URL:?Falta la conexión de revalidación de identidad}"
: "${VEC_CT_CONTEXTO_ACTOR_DATABASE_URL:?Falta la conexión de contexto de actor}"
```

Esto comprueba presencia, **no conectividad, roles ni funcionamiento**. La
referencia operativa es el bloque local inicial de la guía, no sus comandos
remotos históricos. No reaplique migraciones ni ejecute el apéndice de
recreación para operar el entorno ya preparado.

### Dependencias ya preparadas, no instalación al arrancar

CT `000056_respuesta_recibida_rrhh` y AD3
`000014_consumidor_respuesta_recibida_rrhh` están **instaladas en ambas bases
locales**. No reaplicarlas ni recrear el entorno. El registro usa el permiso
propio `contratacion_temporal.llamamiento.respuesta.registrar`, no el de aviso
local, y mantiene las once DSN nominales por aplicación contra su única base.
No añada conexiones ni conceda permisos manuales para superar un rechazo.

Dirección aplicó en ambas bases el bloque literal `DO $fechas$` de AD3-14:
compara instantes y corrige la diferencia de ceros finales entre decisión y
capacidad, sin cambiar firmas, hashes ni permisos. Sus tres regresiones
pasaron. SHA-256 del núcleo instalado, **no de un commit de publicación**:
`42f67b75786e996c56309350389801091cf749adb85a2e7b6d40ee49c399fb62`.
La instrumentación de diagnóstico CT56 se retiró de ambas bases, que conservan
la misma función final. No ejecutar de nuevo el bloque ni reinstalar consumidores.

Dirección confirmó en 55433 las migraciones de consultas y sus dependencias
de identidad, contexto y autorización, con barreras `25/9`. El catálogo
`motivos_ct_consultas_rrhh_desarrollo`, versión `1`, se publicó con los
argumentos sintéticos originales en secuencia `8`, junto con los vínculos
de cuadro y detalle: tres resultados positivos en una transacción.
Esto completa la preparación que faltaba tras el traslado; no es una receta
para copiar filas de 55432 ni para publicar de nuevo el catálogo.

No herede conexiones de otra terminal ni habilite parcialmente las cinco
dependencias de consultas. No reaplique `000001` de contexto ni conceda roles
para salvar un rechazo. El publicador de instalación no sustituye ninguna
de las once identidades de la aplicación ni añade una conexión de ejecución.

## 3. Material y certificados: reutilizar, no regenerar

El directorio de material contiene más que HTTPS: identidad sintética,
certificados y claves, material de autorización, claves de idempotencia,
manifiesto y avisos. **Conservar solo la base de datos no permite recuperar
el mismo entorno.**

Para verificar el material existente sin solicitar la creación de otro:

```bash
(
  cd "$vec_ops_repo" &&
  test -d "$vec_ops_material" &&
  test ! -L "$vec_ops_material" &&
  test -f "$vec_ops_material/manifiesto.json" &&
  scripts/generar_credenciales_desarrollo.sh "$vec_ops_material"
)
```

El generador verifica correspondencia de claves y certificados, vigencia,
manifiesto y permisos. Si el destino no existe, su comportamiento normal es
**generar** material: por eso se comprueba primero su existencia. Si rechaza
el directorio, no lo borre, sustituya ni regenere para eludir el fallo.

Mantenga directorios `0700` y archivos privados `0600`, sin enlaces
simbólicos ni acceso de otros usuarios. No aplique cambios recursivos a rutas
sin inspeccionar. El lanzador valida el material y establece su configuración;
no hace falta ejecutar `desarrollo.env` como código.

Para el navegador use perfiles temporales separados: RRHH
(`mtls/cliente.p12`) e Intervención (`mtls/intervencion.p12`).
Importe `ca/ca.crt` y únicamente el paquete del perfil necesario mediante el
procedimiento privado de la guía. Use su archivo de contraseña en el diálogo
local, sin mostrarlo en consola. No copie `ca.key`, claves de idempotencia
o el directorio completo al navegador.

## 4. Arrancar y comprobar sin confundir salud con funcionalidad

Use preferentemente el lanzador local conservado fuera de Git, indicado en
la guía: `recorrido` para 8443 y `consultas` para 8444. El comando inferior es
su alternativa de arranque directo, no una segunda copia simultánea.
Tras las comprobaciones previas y el aviso de dirección, en la terminal que
conserva las once conexiones del entorno elegido:

```bash
(
  cd "$vec_ops_repo" &&
  test -d "$vec_ops_material" &&
  test -f "$vec_ops_material/manifiesto.json" &&
  scripts/arrancar_vec_desarrollo.sh \
    --puerto "$vec_ops_http_puerto" \
    --directorio-material "$vec_ops_material"
)
```

Déjelo en primer plano. Construye un binario temporal y escucha exclusivamente
en `127.0.0.1`, con **TLS 1.3 y certificado de cliente obligatorio**.
No abre producción ni instala un servicio permanente. Si el puerto está
ocupado, identifique su dueño: no mate procesos por nombre.

Desde otra terminal del mismo equipo local, con las variables de referencia:

```bash
curl --silent --show-error --fail \
  --cacert "$vec_ops_material/ca/ca.crt" \
  --cert "$vec_ops_material/mtls/cliente.crt" \
  --key "$vec_ops_material/mtls/cliente.key" \
  "https://localhost:$vec_ops_http_puerto/livez"
```

| Señal | Interpretación correcta |
| --- | --- |
| `/livez` devuelve `200` y `status: ok` | El proceso responde. Es lo que espera el lanzador, no una prueba de persistencia. |
| `/readyz` o `/healthz` devuelve `503` | El comprobador de disponibilidad no está configurado o no ha satisfecho su control. No se cambia a éxito artificialmente. |
| Página cargada | Los recursos web se sirven; no demuestra que sus operaciones estén conectadas. |
| Recibo contrastado con persistencia y recuperado tras reiniciar | Evidencia del recorrido concreto de la guía, no de los ocho pasos ni de una entrega de correo. |

El lanzador imprime ejemplos de consultas de cuadro y detalle, incluidas
referencias de demostración. **No son una prueba de bandeja real ni datos que
deban copiarse para operar.**

En el equipo del operador abra `https://localhost:8443/portal-empleado/`
(principal) o `https://localhost:8444/portal-empleado/` (secundaria), con su
certificado de RRHH; no necesita un túnel al remoto detenido.
Para continuar una solicitud existente, use los controles de su fila en
la bandeja y el formulario de análisis del detalle, como indica la guía.
No cree otra alta ni copie referencias de los ejemplos del lanzador.
La declaración de respuesta es la tercera operación del mismo formulario de
llamamiento, tras recuperar la comunicación confirmada `v2`; requiere respuesta
declarada, referencia opaca, recepción UTC y `.eml` de hasta 2 MiB para calcular
su huella local. No cambia la comunicación `v2`, el expediente observado `v6`
ni el estado Bolsa. No acredita envío, entrega ni aceptación terminal.
Un acceso desde otro equipo requiere coordinación de red y certificado;
mantenga siempre la escucha en bucle local.
No abra el puerto a Internet, publique un proxy o desactive TLS para facilitar
el acceso. Las operaciones funcionales se prueban con los datos y claves de
la guía, no con envíos repetidos como comprobador de salud.

## 5. Copias: base de datos, material y avisos juntos

El registro de respuesta conserva declaración, actor, referencia, SHA-256 y
recibo, **no el contenido del `.eml`**. WebCrypto lo procesa localmente sin
subirlo ni guardarlo en VEC. El original sigue en el sistema de correo: una
copia de VEC no es copia ni custodia de ese correo, ni verifica origen o firma.

Antes de una actualización o intervención, acuerde una ventana sin escrituras:
termine VEC ordenadamente y confirme que ningún otro proceso, agente o prueba
escribe en ese entorno. No detenga ni copie una instancia que pertenezca a otro
trabajo. Para una copia lógica se mantiene PostgreSQL disponible.

| Elemento que se conserva | Motivo |
| --- | --- |
| Copia nativa completa de la base | Expedientes, historia, autorizaciones, recibos, órdenes y eventos pendientes. No basta exportar una tabla. |
| Material completo | Claves e identidad ligadas a los datos y a su recuperación; preserve las generaciones anteriores. |
| `comunicaciones/` dentro del material | Ficheros de aviso local, además del registro persistente de intención. No son correo enviado ni abren plazo. |
| Configuración y contexto de instalación | Hash de código, versión e imagen de PostgreSQL, inventario de migraciones, roles y concesiones; certificados y referencias de configuración bajo custodia privada. |

Ejemplo de copia local, **solo en esa ventana autorizada**, con un directorio
padre ya existente y una cuenta de copia que ya tenga permiso. No concede
permisos ni prepara una restauración:

```bash
(
  set -euo pipefail
  set +x
  umask 077
  vec_ops_copias='/RUTA/ABSOLUTA/EXISTENTE/PARA/COPIAS'
  vec_ops_base='<BASE_EXISTENTE>'
  vec_ops_login_copia='<LOGIN_DE_COPIA_YA_AUTORIZADO>'
  test -d "$vec_ops_copias"
  test -d "$vec_ops_material/comunicaciones"
  vec_ops_respaldo=$(mktemp -d "$vec_ops_copias/vec-respaldo.XXXXXX")

  docker exec "$vec_ops_pg_contenedor" \
    pg_dump --username="$vec_ops_login_copia" \
    --dbname="$vec_ops_base" --format=custom \
    > "$vec_ops_respaldo/base.dump"
  tar -C "$vec_ops_material" -cpf "$vec_ops_respaldo/material.tar" .
  test -s "$vec_ops_respaldo/base.dump"
  test -s "$vec_ops_respaldo/material.tar"
  sha256sum "$vec_ops_respaldo/base.dump" "$vec_ops_respaldo/material.tar"
)
```

El comando de copia usa el cliente del contenedor PostgreSQL existente.
No introduzca contraseñas en sus argumentos. Si faltan acceso, permisos o
espacio, conserve el error y no declare válida la copia parcial.

`pg_dump` no incluye los roles globales ni toda la configuración del servidor:
el paquete debe completarse con su respaldo aprobado y con la configuración
privada del entorno. Los archivos anteriores contienen material sensible;
consérvelos con acceso restringido, cifrado y copia externa según la política
aprobada. No los adjunte a GitHub ni sustituya una copia anterior.

**Archivo existente y huella calculada no equivalen a restauración probada.**
Este manual no acredita un ensayo de restauración ni fija tiempos de
recuperación. Ese ensayo requiere destino aislado y autorización específica;
no se restaura sobre el entorno preservado como comprobación rutinaria.

## 6. Parada, reinicio, actualización y reversión

### Parada y reinicio ordinarios

1. Deje de iniciar operaciones y conserve las claves de las que estén en
   curso. Una respuesta perdida no demuestra que no haya efecto persistido.
2. Pulse **Ctrl-C** en la terminal del lanzador y espere a que termine.
   El script envía terminación al servidor que supervisa, espera su salida y
   retira su binario temporal. No borra PostgreSQL ni el material.
3. Mantenga base, once conexiones y material. Reinicie con el mismo comando
   del apartado 4; no regenere certificados o claves.
4. Compruebe salud y después el resultado funcional conforme a la guía.
   Para recuperar una operación, use sus mismos datos y clave, con autorización
   vigente. No pulse «Preparar clave nueva» para recuperar un recibo.

La guía acredita recuperación de selección, comunicación y declaración RRHH
conservando recibos y fechas, sin nuevos registros. No demuestra una
recuperación automática de cualquier operación,
ni que alta y análisis tengan ya una vista de recarga de recibos tras cerrar
la página.

### Actualización aprobada

1. Dirección identifica la entrega exacta y su compatibilidad con el esquema.
   Revise inventario y estado sin alterar ramas, archivos o trabajo pendiente.
2. Conserve el hash anterior y realice la copia conjunta del apartado 5.
3. Pare ordenadamente. El integrador proporciona el producto aprobado; este
   manual no hace fusiones, copias de código entre ramas ni instalaciones SQL.
4. Arranque esa entrega con la configuración y material acordados. Compruebe
   salud y un recorrido focal de la guía, conservando los mismos identificadores
   cuando se trate de una recuperación.
5. Registre el resultado antes de reabrir el uso. Una instalación SQL correcta
   no acredita por sí sola el recorrido ni su recuperación tras reinicio.

### Reversión conservadora

Si la entrega falla, detenga únicamente su proceso y conserve estado y
evidencias. Volver a una versión anterior del código solo es admisible cuando
dirección confirme que entiende el esquema y los datos actuales. Un cambio
de código no revierte migraciones ni efectos ya confirmados.

Si esa compatibilidad no está demostrada, mantenga el entorno detenido y
solicite un plan de recuperación. No ejecute migraciones de vuelta, borre
volúmenes, reinicie tablas o restaure una copia encima de escrituras nuevas.
No use operaciones Git destructivas para recuperar un arranque.

Los DOWN de CT56/AD3-14 bloquean la reversión (`55000`) si hay registros o
dependencias. Es una protección de la evidencia, no una pérdida de datos.
No ejecutarlos en estas bases con historial; recuperación mediante respaldo
o avance correctivo bajo dirección, no borrado de declaraciones.

## 7. Incidencias y registro de evidencia

| Incidencia | Respuesta operativa |
| --- | --- |
| Puerto ocupado | Identificar proceso y responsable; no terminar procesos ajenos. |
| Material ausente, incoherente o caducado | Verificar ruta y vigencia; conservarlo y coordinar recuperación o renovación. No regenerar sobre el mismo historial. |
| Error de TLS o autenticación | Comprobar CA, nombre del servidor, certificado de cliente y reloj. No usar `--insecure` ni rebajar validación de PostgreSQL. |
| Conexiones incompletas o no separadas | Revisar nombres de variables e identidades autorizadas sin mostrar sus valores; no sustituirlas por un superusuario. |
| Huella de función o esquema distinta | Detener arranque funcional y comparar versión instalada con la entrega; no reaplicar ni omitir la guarda. |
| Bandeja o servicio devuelve `503` | Registrar ruta y estado. No implica ausencia de expedientes ni autoriza nuevos registros. |
| `409`, operación ocupada o resultado indeterminado | Conservar solicitud y clave por el procedimiento seguro; comprobar estado antes de repetir. No inventar otra clave. |
| Falla la escritura del aviso local | Comprobar espacio y acceso al directorio; la base puede tener ya el registro. Recuperar solo de forma expresa con la misma clave y autorización vigente. |
| Copia incompleta | No marcarla válida ni retirar la última copia conservada. Registrar el fallo sin exponer contenido. |

Registre en el seguimiento existente: fecha y zona horaria, entorno,
hash aprobado, versiones, paso intentado, ruta sin parámetros sensibles,
estado HTTP, mensaje saneado y referencias mínimas de recibo cuando sean
necesarias. Indique si el resultado es salud, comprobación funcional o
recuperación, y si sigue pendiente.

No guarde contraseñas, cadenas de conexión, claves, paquetes de certificados,
volcados, archivos de identidad, cabeceras o cuerpos completos de peticiones.
Revise capturas y logs antes de compartirlos; no use `set -x`, `env` ni
volcados completos de configuración como evidencia. Ante un resultado incierto,
conserve el historial y declare la incertidumbre: nunca la convierta en éxito.
