# Cartografía interna del módulo Dietas

Fecha de corte: 19 de julio de 2026.

## Resultado y alcance

El módulo Dietas utiliza rutas reales por carretera y un mapa OpenStreetMap
servido por el propio despliegue. El navegador no consulta servicios públicos,
no conoce la dirección de OSRM y no envía cookies, credenciales ni datos de
identidad a los componentes cartográficos.

La cartografía de la presentación sigue siendo no autoritativa: las comisiones,
personas e importes son datos sintéticos y no producen actos administrativos.
El grafo OSRM y las teselas sí son reales, lo que permite validar la experiencia
final sin mantener dos interfaces distintas.

## Arquitectura

```text
Navegador
   |
   | http://127.0.0.1:8081
   v
Nginx de presentación (único puerto publicado)
   |-- /portal-empleado/... ----------> Portal RRHH sintético
   |-- /tiles/osm/{z}/{x}/{y}.png ----> TileServer GL
   `-- POST /api/presentacion/cartografia/rutas
                                      -> Mediador Go
                                           |
                                           `-> OSRM interno
```

Hay dos redes cartográficas internas separadas:

- el proxy, el mediador y el renderizador de teselas comparten exclusivamente
  la red de presentación cartográfica;
- el mediador y OSRM comparten exclusivamente la red de cálculo de rutas;
- Nginx no puede alcanzar OSRM de forma directa;
- OSRM, el mediador y TileServer GL no publican puertos del anfitrión;
- únicamente Nginx publica el puerto de presentación en `127.0.0.1`.

Todos los componentes, incluida la importación del PBF y la generación de
MBTiles, se ejecutan en contenedores. No se instalan demonios, paquetes,
unidades `systemd`, proxies auxiliares ni procesos cartográficos en el equipo
anfitrión.

## Contratos intercambiables

La vista de Dietas depende de dos puertos:

- el calculador de rutas devuelve alternativas, tramos, distancia, duración,
  geometría y versión del grafo;
- el visor de mapas recibe una geometría ya autorizada y solicita teselas a una
  ruta fija del mismo origen.

En Go, el puerto neutral `CalculadorRutas` separa el dominio del adaptador OSRM.
La API productiva conserva su frontera RBAC y la presentación usa un proceso
independiente sin identidad ni persistencia. Ambas superficies reutilizan el
mismo adaptador OSRM y la misma validación HTTP; no existe una implementación
paralela de la regla cartográfica.

El adaptador web de presentación solo admite:

```text
POST /api/presentacion/cartografia/rutas
GET  /tiles/osm/{z}/{x}/{y}.png
```

No acepta una URL configurable desde el navegador y falla de forma explícita
si el mediador, la versión gobernada o la respuesta no cumplen el contrato. No
hay cambio silencioso a una distancia simulada.

## Datos y versiones gobernadas

Fuente actual:

- PBF: `deploy/osrm-granada/data/granada-buffer.osm.pbf`;
- SHA-256 del PBF:
  `53aba0ad43c45c62a44ee54fa9fb68308427957c5b5504cbd19a74fc49672724`;
- ámbito: provincia de Granada con margen de enrutado;
- versión lógica del grafo:
  `granada-buffer-osrm-v1-53aba0ad43c4`;
- versión activa de teselas:
  `20260719T115334Z-53aba0ad43c4`;
- teselas generadas: 14.489, niveles 0 a 14;
- MBTiles activo: aproximadamente 84,8 MiB.

OSRM 5.26.0 omite `data_version` —y otras construcciones pueden devolverlo como
`null`— para este grafo. El despliegue declara una versión gobernada obligatoria
derivada del conjunto de datos. El mediador rechaza una versión textual del
motor si contradice la configuración e incorpora la versión gobernada a la
respuesta. Una liquidación
productiva deberá conservar esa versión junto con la referencia opaca del
cálculo; nunca debe recalcular retroactivamente un expediente cerrado.

## Controles de seguridad

- Denegación por defecto y rutas HTTP exactas.
- Límite de cuerpo, respuesta, número de paradas, alternativas y puntos de
  geometría.
- Ámbito geográfico explícito; se rechazan coordenadas fuera de él.
- Cliente OSRM sin proxy ambiental, redirecciones ni destinos fuera de las CIDR
  enumeradas; cada resolución se valida antes de conectar.
- Cabeceras de identidad, autorización, cookies, `Forwarded` y CORS retiradas
  antes de entrar en la superficie cartográfica.
- Las coordenadas viajan en el cuerpo del POST y no forman parte de la ruta ni
  del registro de acceso estructurado de Nginx.
- Contenedores de solo lectura, capacidades Linux eliminadas, prohibición de
  nuevos privilegios y límites de CPU, memoria y procesos.
- Leaflet 1.9.4 se sirve localmente con versión, licencia, huellas e integridad
  de recursos; no se usa CDN.
- La atribución de OpenStreetMap/OpenMapTiles permanece visible en el control
  del mapa real y no se duplica en otro bloque visual.

## Operación de la presentación

La composición raíz es la única entrada soportada:

```sh
docker compose --profile presentacion up -d --build
```

URL de Dietas:

```text
http://127.0.0.1:8081/portal-empleado/?presentacion=rrhh&perfil=administrador#dietas
```

El conjunto de teselas se prepara y activa con los scripts de
`deploy/osm-tiles-granada`. Esos scripts invocan contenedores fijados; no
ejecutan importadores GIS nativos en el anfitrión. El grafo OSRM se prepara con
los perfiles de construcción documentados en `deploy/osrm-granada`.

Antes de una presentación se debe comprobar:

1. salud del portal, mediador, OSRM y renderizador de teselas;
2. respuesta real para una ruta semilla multiparada;
3. carga de teselas del mismo origen sin solicitudes externas;
4. un único mapa visible en el formulario de nueva comisión;
5. ausencia de errores de consola y desbordamientos a 1440, 1024 y 390 píxeles.

## Paso a producción

La vista, el contrato, Leaflet, el visor, el puerto Go y el adaptador OSRM son
candidatos a conservar, sujetos a la aceptación de RRHH y Sistemas. Se retiran
la composición y los datos sintéticos de presentación. La futura composición y
el caso de uso productivos deben añadir identidad corporativa, autorización
RBAC y por ámbito, persistencia, auditoría, idempotencia y recibo probatorio. El
servidor vuelve a calcular y validar kilometraje, alternativa, desvíos y ajustes
antes de admitir una liquidación.

La publicación en otra red, en nube privada o con otro motor no exige cambiar
la vista: se sustituye el conector detrás de los puertos y se actualizan las
concesiones de red y versiones gobernadas en el despliegue.
