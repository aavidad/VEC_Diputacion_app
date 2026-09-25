# Capa de supervisión de VEC — M0: encaje, licencias y presupuesto

Este documento corresponde a la minitarea **M0** del plan acordado entre Claude y Astra el
25/09/2026 (errores visibles para Sistemas y avisos). Contiene tres cosas: el
**presupuesto de recursos medido** de Zabbix 7.0 LTS, el **inventario de licencias** y
la **recomendación de encaje** junto a VEC y su PostgreSQL. No instala nada en cidonia.

La capa de supervisión no bloquea la aplicación. Ninguna petición de VEC espera a
Zabbix; eso se construye en M1/M2. Aquí solo se mide cuánto cuesta la plataforma y se
fijan sus topes.

## Contenido

| Ruta | Qué es |
|---|---|
| `zabbix/imagenes.lock` | Imágenes fijadas por digest (Zabbix 7.0.31, PostgreSQL 18.6, Alpine 3.24.2). |
| `zabbix/recursos.env` | Fuente única de límites (CPU, memoria, pids, E/S) y ajustes de PG, servidor y PHP. |
| `zabbix/quadlet/` | Unidades Podman rootless (Quadlet): dos redes, un volumen y cuatro contenedores. |
| `zabbix/servidor-sin-nmap/Containerfile` | Servidor oficial **sin nmap** (licencia NPSL, no libre según OSI; ver licencias). |
| `zabbix/agente/vec_supervision.conf` | Lista de claves permitidas del agente (lectura solo bajo `/host/cgroup`, sin `system.run`). |
| `zabbix/plantillas/vec_cgroup_contenedores.yaml` | Plantilla que descubre contenedores leyendo cgroup v2 (12 elementos por contenedor). |
| `zabbix/piloto/zabbix_api.py` | Cliente mínimo de la API usado por el piloto (solo biblioteca estándar). |
| `../../scripts/probar_presupuesto_supervision.sh` | Modos `recursos`, `coherencia`, `rootless` y `piloto`. |

## Arquitectura desplegada

- **PostgreSQL propio** (`vec-supervision-pg`), en la red `vec-supervision-datos`
  (`Internal=true`) y con volumen propio. No comparte instancia con VEC: tiene que poder
  registrar la caída del PostgreSQL de VEC.
- **Servidor Zabbix** en las redes de datos y frontal. Publica 10051 **solo en
  127.0.0.1** para el agente activo. Tiene `AllowSoftwareUpdateCheck=0` (ninguna llamada a
  zabbix.com) y los scripts globales desactivados.
- **Frontend** publicado **solo en 127.0.0.1:8090**. Sistemas entra por túnel SSH. El
  frontend sirve para mantenimiento y para la API que consumirá Administración (M4). El
  panel diario está en VEC.
- **Agente 2** en modo solo activo, sin puerto de escucha, con `--network=host` y
  `--pid=host`:
  - El host se lee por `/proc`: CPU, memoria, carga, procesos, red y disco (`/proc/diskstats`).
  - Los contenedores se leen en `/sys/fs/cgroup`, montado **de solo lectura** en
    `/host/cgroup` y leído con claves nativas (`vfs.dir.get`, `vfs.file.contents`,
    `vfs.file.regexp`).
  - **No se monta el socket de podman ni el de docker**, ni tampoco un script propio.
  - El espacio en disco se mide con «sondas», que son directorios vacíos de cada sistema
    de ficheros montados en modo `ro`. `statfs` devuelve tamaño y ocupación sin exponer
    contenido.
- **Topes por contenedor**: `--cpus`, `--memory` (sin swap), `--pids-limit` y
  `--cpu-shares=256`, que equivale a `cpu.weight` 35 frente a 100 de un servicio normal.
  Con contención, VEC gana. Además, `Nice=10` e `IOSchedulingClass=best-effort 7` en las
  unidades.

Tras la supervisión, los contenedores aparecen con el **nombre de unidad** en Quadlet
(`vec-*.service`) o con el **identificador** en Docker. No aparece ningún dato personal.

## Piloto ejecutado (25/09/2026, este equipo de desarrollo)

Orden ejecutada:

```
MOTOR=docker SUBRED_DATOS=10.253.10.0/24 SUBRED_FRONTAL=10.253.11.0/24 \
  REPOSO_MIN=5 CARGA_MIN=15 scripts/probar_presupuesto_supervision.sh piloto
```

- **Equipo**: 32 CPU lógicas y 123 GiB de RAM. Tenía otros trabajos en marcha (carga ≈ 3,8 y
  47 contenedores ajenos) y no se paró nada.
- **Motor**: Docker. Podman **no está instalado** en esta máquina y no puede instalarse sin
  root (tampoco hay `newuidmap`). El consumo de Zabbix no depende del motor. La parte
  rootless se comprobó aparte (ver más abajo).
- **Fases**:
  - 3 min de estabilización tras crear el esquema.
  - **Reposo** de 5 min: la pila está arrancada y el host por defecto desactivado, sin
    elementos supervisados.
  - **Carga** de 15 min. El host piloto tiene enlazadas las plantillas
    *Linux by Zabbix agent active*, *Zabbix server health* y *VEC cgroup contenedores*.
    En total son **747 elementos activos, 735 soportados y 670 con valor**:
    - **51 contenedores descubiertos** por cgroup (561 elementos con valor);
    - intervalo de 1 min;
    - cada 60 s, una consulta a la API como la que hará Administración (problemas, estado
      del host y últimos valores).
- **Muestreo**: cada 15 s se leyeron los ficheros cgroup de cada contenedor (`cpu.stat`,
  `memory.current` y `memory.stat`, `io.stat`, `pids.current`) y el tamaño de la base.

### Límites efectivos, leídos del cgroup de cada contenedor

| Componente | cpu.max | cpu.weight | memory.max | swap | pids.max | io.max |
|---|---|---|---|---|---|---|
| PostgreSQL | 100000/100000 (1 CPU) | 35 | 768 MiB | 0 | 128 | lectura 50 MB/s, escritura 20 MB/s |
| Servidor | 1 CPU | 35 | 384 MiB | 0 | 256 | — |
| Frontend | 0,5 CPU | 35 | 256 MiB | 0 | 64 | — |
| Agente | 0,25 CPU | 35 | 128 MiB | 0 | 64 | — |

### Consumo medido

CPU en % de un núcleo. «mem» es `memory.current`, que incluye la caché de páginas y se
puede recuperar. «anon» es la memoria propia de los procesos, el equivalente real a RSS.

**Reposo (5 min)**

| Componente | CPU media | CPU p95 | mem máx | anon máx | caché máx | escritura | pids |
|---|---:|---:|---:|---:|---:|---:|---:|
| PostgreSQL | 0,4 % | 0,5 % | 368 MiB | 48 MiB | 300 MiB | 157 KiB/s* | 35 |
| Servidor | 0,2 % | 0,2 % | 42 MiB | 22 MiB | 9 MiB | 0 | 48 |
| Frontend | 0,0 % | 0,1 % | 30 MiB | 20 MiB | 9 MiB | 0 | 9 |
| Agente | 0,0 % | 0,0 % | 9 MiB | 8 MiB | 0 | 0 | 8 |
| **Total** | **0,6 %** | | **449 MiB** | **98 MiB** | | | |

\* Es la cola de escrituras de arranque del esquema (checkpoint y autovacuum). En carga
baja a 54 KiB/s.

**Carga (15 min, ≈ 670 métricas/min, 9,3 valores/s)**

| Componente | CPU media | CPU p95 | CPU máx | mem máx | anon máx | caché máx | escritura | pids |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| PostgreSQL | 0,7 % | 1,3 % | 1,4 % | 376 MiB | 52 MiB | 303 MiB | 54 KiB/s | 35 |
| Servidor | 0,3 % | 0,4 % | 0,5 % | 45 MiB | 24 MiB | 10 MiB | 0 | 48 |
| Frontend | 0,2 % | 0,6 % | 0,7 % | 52 MiB | 28 MiB | 17 MiB | 0 | 10 |
| Agente | 0,2 % | 0,3 % | 0,3 % | 17 MiB | 14 MiB | 0 | 0 | 8 |
| **Total** | **1,3 %** | | | **490 MiB** | **118 MiB** | | | |

- No hubo estrangulamiento por CPU (0,00 % a 0,02 %), ni terminaciones por memoria, ni
  errores del servidor salvo los esperados. Esos errores son dos:
  - «host not found» durante el reposo, porque el host se crea al empezar la carga;
  - un contenedor ajeno que se borró durante la prueba, cuyo elemento pasó a no soportado
    y lo retira la regla de descubrimiento.
- **Consulta tipo panel** (API): 15 consultas, con mediana de **90 ms** y máximo de 109 ms.

### Disco

- La **base recién creada ocupa 71 MiB** y el volumen completo de PG con WAL, 254 MB.
- Durante la carga, **historia y tendencias crecieron 1,1 MiB en 15 min**, a unos 135 bytes
  por valor con sus índices. Extrapolado a un día salen unos **100 MiB/día** para 670
  métricas/min.
- **Estimación en régimen estable** con la retención propuesta (historia 7 días, tendencias
  90 días, que son los valores de la plantilla VEC):
  - historia: unos 0,7 GiB;
  - tendencias (una fila por hora y elemento): unos 0,15 GiB;
  - WAL: hasta 0,5 GiB (`max_wal_size`);
  - total de la base: unos **1,5 GiB**.
- Las imágenes ocupan unos 0,8 GB.
- **Presupuesto de disco: 3 GiB** en el sistema de ficheros del almacenamiento de podman.

### Rootless: ¿se aplican los límites de verdad?

Resultado de `scripts/probar_presupuesto_supervision.sh rootless` en este equipo, con
usuario sin privilegios y el mecanismo de delegación de systemd que usa podman rootless:

| Control | Delegado al usuario | Prueba | Resultado |
|---|---|---|---|
| CPU | sí | `CPUQuota=20%` sobre un bucle activo | uso medido 19 %, 59 periodos estrangulados: **se aplica** |
| Memoria | sí | `MemoryMax=64M` frente a 200 MiB | proceso terminado, `Result=oom-kill`: **se aplica** |
| pids | sí | `TasksMax=8` frente a 20 procesos | `pids.max=8` y rechazos contabilizados: **se aplica** |
| **E/S** | **no** | `IOWriteBandwidthMax` | **no existe `io.max`: NO se aplica** |

**Consecuencia.** En rootless con la delegación por defecto de systemd (`cpu memory pids`),
los topes de CPU, memoria y pids funcionan, pero **el límite de E/S del PostgreSQL de
supervisión no tiene efecto**. Hay dos opciones:

- **Aceptarlo con mitigaciones.** Ya están aplicadas: `synchronous_commit=off`, escritura
  medida de unos 54 KiB/s, `IOSchedulingClass=best-effort 7` y `Nice=10`.
- **Delegar el controlador `io`.** Es decisión de Sistemas y requiere root:
  `/etc/systemd/system/user@.service.d/delegate.conf` con
  `[Service]` + `Delegate=cpu cpuset io memory pids` y reinicio de la sesión del usuario de
  servicio. Después, activar la línea `--device-*-bps` comentada en
  `vec-supervision-pg.container`.

En modo rootful (el piloto con Docker), `io.max` sí se aplicó
(`259:1 rbps=52428800 wbps=20971520`).

**Pendiente de verificar con podman real.** En cidonia (podman rootless) hay que ejecutar
`scripts/probar_presupuesto_supervision.sh rootless`. Si detecta podman, además arranca
contenedores con `--cpus`, `--memory` y `--pids-limit` y lee sus cgroup. Con Quadlet, podman
usa `--cgroups=split`: los límites quedan en el cgroup hijo `libpod-payload-*` y el
elemento «límite de memoria» leído en el nivel `.service` marcará 0 («sin límite»). En M3
se ajustará la macro o se leerá el hijo.

### Visibilidad del agente sin socket (comprobado)

- Los montajes del agente son `/sys/fs/cgroup` (ro), su configuración (ro) y dos sondas
  vacías (ro). **No se monta ningún socket del motor.**
- Ve el host: `vm.memory.size[total]`, `proc.num`, `system.uptime`, las interfaces del host
  y la ocupación de `/home` (90 %) y `/` (99,5 %) a través de las sondas.
- Ve los contenedores: **51 descubiertos**, con 561 elementos cgroup con valor.
- Deniega lo que no debe:
  - `vfs.file.contents[/host/cgroup/../../etc/passwd]` → no admitida;
  - `vfs.file.contents[/etc/hostname]` → no admitida;
  - `system.run[id]` → no admitida.

  Las reglas `DenyKey` para `..` van antes de las `AllowKey`.

## Presupuesto propuesto

| Recurso | Uso medido (carga) | Tope fijado (`recursos.env`) | Reserva recomendada en el servidor |
|---|---|---|---|
| CPU | 1,3 % de un núcleo (máx. < 3 %) | 2,75 CPU en suma (1+1+0,5+0,25) y peso bajo | 0,25 núcleo de media; picos acotados por los topes |
| Memoria | 118 MiB anónima; 490 MiB con caché | 1.536 MiB en suma (768+384+256+128) | **1,5 GiB** libres, porque la caché de PG se recupera bajo presión |
| Disco | 71 MiB iniciales; ~100 MiB/día | — | **3 GiB** (base estable ≈ 1,5 GiB, WAL e imágenes) |
| E/S | ~54 KiB/s de escritura | 20 MB/s de escritura y 50 MB/s de lectura solo con `io` delegado | irrelevante a esta tasa |
| Red | loopback y red interna | 10051 y 8090 solo en 127.0.0.1 | — |

Los topes superan con holgura el consumo, así que no hay riesgo de terminación por
memoria. Aun así, garantizan que un fallo de Zabbix (fuga o consulta desbocada) no puede
pasar de 1,5 GiB ni de 2,75 CPU. En cuanto a CPU, con `cpu.weight` 35, VEC tiene
preferencia en cualquier contención.

## Recomendación de encaje en cidonia

**Cabe**, siempre que el servidor tenga libres al menos **1,5 GiB de RAM** y **3 GiB de
disco** en el almacenamiento de podman del usuario de servicio. El consumo de CPU es
despreciable: un 1,3 % de un núcleo con unas 700 métricas por minuto. Es muy inferior a la
referencia oficial de Zabbix («pequeña»: 2 vCPU y 8 GiB para 1.000 métricas), que es un
dimensionamiento genérico y no un mínimo.

Antes de instalar, Sistemas debe ejecutar en cidonia, como usuario de servicio, los modos
`recursos` y `rootless` del script. **No se ha tocado cidonia en M0**: sus recursos libres
reales no se han medido.

**Condiciones para que no comprometa a VEC:**

1. Hace falta margen de RAM. Si la RAM disponible tras VEC, su PostgreSQL y las instancias
   de prueba es menor de 2 GiB, hay que reducir antes de instalar:
   - `PG_MEMORIA=512m` y `shared_buffers=64MB`;
   - `SERVIDOR_MEMORIA=256m`;
   - `WEB_MEMORIA=192m`.

   El total de topes quedaría en unos 1,1 GiB, y las mediciones indican que siguen
   sobrando.
2. Si falta disco:
   - historia de 3 días y tendencias de 30;
   - intervalo del descubrimiento de contenedores en 1 h;
   - recogida de contenedores cada 5 min.

   El crecimiento baja por debajo de 25 MiB/día.
3. Si falta CPU (poco probable): el intervalo de la plantilla VEC pasa a 5 min, y se
   desactivan en el host las plantillas de proceso que no se usen.
4. **E/S**: sin delegación del controlador `io`, el PG de supervisión no tiene tope de E/S.
   A 54 KiB/s es irrelevante, pero se recomienda la delegación (una línea de root).
5. **Topes permanentes**: no se retiran nunca. Una caída de Zabbix no afecta a VEC. Una
   caída de VEC o de su PG se registra, porque la base es independiente.

## Licencias

**Criterio**: solo licencias libres reconocidas por OSI o FSF, sin «fuente disponible»
(SSPL, BSL/BUSL, Elastic, Commons Clause).

| Componente | Licencia | Estado |
|---|---|---|
| Zabbix servidor, frontend y agente 2 (7.0.31) | AGPL-3.0 (desde 7.0) | Libre. No se modifica; si se modificara, se publicaría según AGPL. |
| Frontend: `composer.json` (`zabbix/ui`) | AGPL-3.0-only | Libre |
| Frontend: symfony/yaml, onelogin/php-saml, google2fa, duo_universal, firebase/php-jwt, paragonie, robrichards/xmlseclibs | MIT / BSD-3-Clause | Libres |
| Frontend: JS incluido (jQuery, jQuery UI, D3, Leaflet, markercluster, qrcode) | MIT / BSD-3-Clause / BSD-2-Clause | Libres |
| PHP 8.5 (imagen web) | PHP-3.01 (con partes Zend-2.0, BSD, LGPL, MIT, Apache) | Libre (PHP-3.01 es OSI) |
| PostgreSQL 18.6 | PostgreSQL License | Libre (OSI) |
| Uptime Kuma 2.x (sonda **exterior**, fuera de cidonia; no instalado en M0) | MIT | Libre |
| **nmap 7.99** (dentro de la imagen oficial del **servidor**) | **NPSL** (Nmap Public Source License) | **No libre según OSI; Fedora la rechaza.** Se **elimina** con `servidor-sin-nmap/Containerfile`. Zabbix solo lo usa en el script global «Detect operating system», que está desactivado. |

**Paquetes Alpine.** Se inventariaron con `apk list -I` los paquetes de las cuatro imágenes:
82 del servidor, 89 del frontend, 40 del agente y 53 de PG.

- Todas las licencias son libres: MIT, GPL-2.0/3.0, LGPL, BSD-2/3, Apache-2.0, ISC, Zlib,
  OLDAP-2.8, Net-SNMP, curl, bzip2, Libpng, ICU, X11, FTL/GPL, PostgreSQL y Public Domain.
- **Revisadas a mano**:
  - `cyrus-sasl` (BSD-4-Clause y BSD-3-Clause-Attribution): libres según FSF, con cláusula
    de publicidad;
  - `libmd` (incluye Beerware): permisiva;
  - `sudo` («custom ISC»), en la imagen del agente: ISC con variantes, libre;
  - `.postgresql-rundeps`: metapaquete virtual sin código.
- No aparece SSPL, BSL, Elastic ni Commons Clause.

**Módulos Go del agente 2** (binario y complementos `mongodb`, `mssql`, `postgresql` y
`ember-plus`), 49 módulos extraídos con `go version -m`. Todos son libres:

- 20 MIT, 12 BSD-3-Clause, 7 Apache-2.0, 3 BSD-2-Clause;
- `go-sql-driver/mysql` bajo MPL-2.0;
- `eclipse/paho.mqtt.golang` bajo EPL-2.0 y EDL-1.0;
- `BurntSushi/locker` bajo Unlicense;
- `golang.zabbix.com/sdk` bajo MIT;
- varios con Apache-2.0, UPL-1.0 y BSD.

**Complemento MongoDB.** Usa el controlador `mongo-go-driver` (Apache-2.0), no el servidor
MongoDB (SSPL), y no se usa.

**Telegram** (M3) es un servicio externo privativo. Se mantiene por petición expresa y
recibe solo una proyección técnica sin datos personales. El respaldo es correo corporativo
o Matrix autoalojado.

## MCP para Zabbix (solo informe; nada instalado)

| Proyecto | Licencia | Estado (25/09/2026) | Notas |
|---|---|---|---|
| [initMAX/zabbix-mcp-server](https://github.com/initMAX/zabbix-mcp-server) | AGPL-3.0 | v1.36.1 (07/08/2026), 206 ★, activo | Python. Admite 7.0 LTS completo y modo `read_only` por servidor y por token. Usa token de API de Zabbix. Es de un partner de Zabbix, no de Zabbix. Tiene 237 herramientas, una superficie amplia. |
| [mpeirone/zabbix-mcp-server](https://github.com/mpeirone/zabbix-mcp-server) | GPL-3.0 | V2.0.0 (10/05/2026), 255 ★ | Python. `READ_ONLY=true`. Tiene 3 herramientas genéricas que dan acceso a toda la API. |
| [mhajder/zabbix-mcp](https://github.com/mhajder/zabbix-mcp) | MIT | v0.6.1 (02/09/2026), 10 ★, activo | Python. `READ_ONLY_MODE`. Admite de 6.0 a 7.4. |
| leroylim/zabbix-mcp-server-nodejs | MIT | sin versiones; último cambio 08/2025 | Poco mantenido. |
| kairogyn/zabbix-ai-mcp | sin licencia | último cambio 03/2025 | **No utilizable**: no tiene licencia. |
| MCP oficial de Zabbix | — | «In dev» para **Zabbix 8.0 LTS** (fecha prevista, octubre de 2026) | Todavía no existe en 7.0. |

**Conclusión.** Existen MCP libres y mantenidos. El más completo para 7.0 LTS es initMAX
(AGPL-3.0). Según lo acordado, **solo podrían usarse en el entorno de Sistemas, en modo
lectura y con un token de un usuario Zabbix sin permisos de escritura**. El MCP de VEC (M8)
sigue siendo cliente de los casos de uso de Administración y no de la API de Zabbix.

**Ciclo de vida.** Zabbix 7.0 LTS tiene soporte completo hasta el 30/06/2027 y limitado
hasta el 30/06/2029. La 8.0 LTS, con MCP oficial, está prevista para octubre de 2026 y
conviene reevaluar la migración cuando salga.

## Riesgos y límites de M0

- **Podman no se ha probado en este equipo**, que no lo tiene. El consumo se midió con
  Docker. La delegación rootless se comprobó con systemd, que es el mismo mecanismo. Hay
  que repetir los modos `rootless` y `piloto` (`MOTOR=podman`) en cidonia o en un equipo
  con podman antes de M3.
- **La E/S no está limitada en rootless** sin delegación del controlador `io`.
- **El agente con `--pid=host` y `--network=host`** ve la lista de procesos y las
  interfaces del host. Eso es lo necesario para supervisar el host. Mitigaciones: sus
  claves de ficheros se limitan a `/host/cgroup` y se niegan `..` y `system.run`. Aun
  así, el servidor Zabbix es de confianza para el agente: quien controle el servidor puede
  pedir cualquier clave permitida.
- **El servidor necesitará salida a Internet en M3** (Telegram). Por eso está en la red
  frontal. La base sigue en red interna.
- **Imagen del servidor derivada**: la imagen sin nmap se construye localmente. Al
  actualizar Zabbix hay que reconstruirla desde el nuevo digest (`coherencia` lo comprueba).
- **Extrapolación de disco**: 15 minutos de carga no muestran el régimen de tendencias ni
  la limpieza periódica. La estimación de 1,5 GiB es conservadora y se verificará en M3
  con retención real.

## Reproducir

```
scripts/probar_presupuesto_supervision.sh recursos     # recursos del equipo
scripts/probar_presupuesto_supervision.sh coherencia   # Quadlet == recursos.env == imagenes.lock
scripts/probar_presupuesto_supervision.sh rootless     # límites cgroup en usuario sin privilegios
MOTOR=podman scripts/probar_presupuesto_supervision.sh piloto  # ~25 min, limpia al terminar
```

El piloto:

- corre con `nice 19` e `ionice idle`;
- genera claves aleatorias fuera de Git y cambia la clave por defecto de `Admin`;
- limpia contenedores, redes, volúmenes y secretos al terminar;
- conserva solo los CSV y resúmenes en `~/.local/state/vec-supervision-m0-piloto/resultados/`.

Si el motor agotó sus rangos de red, hay que pasar `SUBRED_DATOS` y `SUBRED_FRONTAL`.
