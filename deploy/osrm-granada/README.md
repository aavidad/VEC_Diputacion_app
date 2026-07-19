# OSRM Granada

Servidor de rutas interno para Dietas/VEC. No usa el servicio publico de OSRM:
VEC debe apuntar a este contenedor con:

```bash
VEC_OSRM_BASE_URL=http://127.0.0.1:5000
VEC_OSRM_SCOPE_NAME="Granada provincia + 15 km"
VEC_OSRM_SCOPE_BOUNDS="36.45,-4.6,38.25,-2.15"
VEC_OSRM_ALLOWED_CIDRS="127.0.0.1/32"
VEC_OSRM_GRAPH_VERSION="granada-buffer-osrm-v1-53aba0ad43c4"
```

Las cinco variables forman una unica configuracion positiva. Si falta alguna,
si los limites o las redes no son canonicos, o si el host resuelve fuera de las
redes enumeradas, VEC no habilita el calculo. `VEC_OSRM_BASE_URL` aprueba
exactamente esquema, host y puerto; no admite rutas, consultas, credenciales ni
redirecciones. En otro despliegue se debe declarar la CIDR real aprobada por
Sistemas: este ejemplo de loopback no autoriza ninguna red de produccion.

La imagen predeterminada está fijada simultáneamente por versión y digest:
OSRM 5.26.0, índice OCI
`sha256:af5d4a83fb90086a43b1ae2ca22872e6768766ad5fcbb07a29ff90ec644ee409`.
No se debe volver a una etiqueta móvil como `latest`. Una sustitución mediante
`OSRM_IMAGE` exige registrar y validar previamente la nueva imagen.

## Ambito

El fichero de entrada por defecto es `data/granada-buffer.osm.pbf`: provincia
de Granada con 15 km de buffer alrededor del limite provincial.

El PBF usado en la presentación del 19 de julio de 2026 tiene SHA-256
`53aba0ad43c45c62a44ee54fa9fb68308427957c5b5504cbd19a74fc49672724`.
Su versión gobernada es `granada-buffer-osrm-v1-53aba0ad43c4`. OSRM puede
omitir `data_version` o responder con `data_version: null`; en ambos casos el
adaptador conserva la versión declarada por el despliegue. Si el motor declara
otra versión, la respuesta se rechaza por contradicción y nunca se corrige
silenciosamente.

Ese recorte se debe generar en el proceso ETL GIS con el limite oficial
provincial y el extracto OSM/IGN correspondiente. El contenedor solo prepara y
sirve el grafo resultante.

Para otro ambito no se cambia codigo: se cambia el PBF, el nombre base y las
variables de VEC:

```bash
OSRM_DATA_BASENAME=jaen-buffer docker compose --profile build run --rm osrm-extract
VEC_OSRM_SCOPE_NAME="Jaen provincia + 15 km"
VEC_OSRM_SCOPE_BOUNDS="37.25,-4.35,38.65,-2.35"
```

`OSRM_DATA_BASENAME` solo admite minusculas ASCII, numeros, guion y guion bajo.
El contenedor valida esa lista positiva antes de construir cualquier ruta y
ejecuta una operacion OSRM fija sin interpolarla en `sh -c`.

## Preparar grafo

```bash
cd deploy/osrm-granada
mkdir -p data
# copiar o generar data/granada-buffer.osm.pbf
docker compose --profile build run --rm osrm-extract
docker compose --profile build run --rm osrm-partition
docker compose --profile build run --rm osrm-customize
docker compose --profile build run --rm --no-deps osrm-normalize
```

La fase final normaliza dentro de Docker todos los productos
`<nombre>.osrm*` a modo inmutable `0444`. El servidor se ejecuta después como
UID/GID `65534`, verifica antes de arrancar que cada artefacto sea regular,
legible y no escribible, y falla cerrado si la preparación está incompleta.
No se debe corregir permisos durante el arranque del servicio.

## Levantar OSRM

```bash
docker compose up -d osrm
curl 'http://127.0.0.1:5000/route/v1/driving/-3.598600,37.177300;-3.655400,37.230600?overview=false'
```

El servidor fija el nivel de registro `WARNING`: una consulta no deja las
coordenadas del itinerario en `docker logs`. Los registros de OSRM son técnicos
y nunca deben utilizarse como auditoría de desplazamientos de una persona.

## Actualizacion

Objetivo operativo: actualizar a diario si hay datos nuevos, y como maximo cada
semana. La reconstruccion debe hacerse en paralelo sobre otro directorio de
datos, probar rutas semilla y cambiar el volumen activo de forma atomica.

VEC guarda la version de grafo usada en cada expediente; las liquidaciones ya
cerradas no se recalculan retroactivamente.

## Comprobacion local de seguridad

```bash
deploy/osrm-granada/tests/probar_inicio_seguro.sh
```

La prueba rechaza nombres hostiles, impide reintroducir `sh -c` y valida la
configuracion efectiva con Docker Compose.
