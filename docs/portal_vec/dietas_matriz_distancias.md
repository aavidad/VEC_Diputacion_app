# Dietas: matriz provincial de distancias

## Decision

Para dietas con coche propio se debe mantener una matriz provincial de
distancias por carretera, versionada y auditable. La liquidacion no debe depender
de una consulta externa en tiempo real ni de kilometros introducidos a mano sin
control.

## Fuentes de referencia

- INE codmun: catalogo anual de municipios y codigos oficiales.
  https://www.ine.es/daco/daco42/codmun/25codmun.xlsx
- INE Nomenclator: entidades y nucleos de poblacion comunicados por los
  ayuntamientos.
  https://www.ine.es/nomen2/index.do
- CNIG/IGN NGMEP: municipios y entidades de poblacion con coordenadas,
  altitud, poblacion y codigos, en CSV/MDB.
  https://centrodedescargas.cnig.es/CentroDescargas/nomenclator-geografico-municipios-entidades-poblacion
- OSRM Table Service: calcula matriz de duraciones/distancias entre pares de
  coordenadas para rutas de vehiculo.
  https://project-osrm.org/docs/v5.24.0/api/#table-service
- openrouteservice Matrix Endpoint: alternativa on-premise/API para matrices
  de distancia y tiempo por perfil.
  https://giscience.github.io/openrouteservice/api-reference/endpoints/matrix/
- GraphHopper Matrix API: alternativa profesional para matrices completas o
  asimetricas.
  https://docs.graphhopper.com/openapi/routing

## Modelo funcional

1. Catalogo de puntos de ruta:
   - Municipios INE de Granada: 174 en la relacion a 1 de enero de 2025.
   - Entidades y nucleos NGMEP/INE: necesarios para casos como Mecina Bombarón,
     que no son municipio pero si localidad/nucleo operativo.
   - Direcciones oficiales de centros de trabajo cuando el expediente no parte
     de una localidad sino de una sede concreta.

2. Matriz origen-destino:
   - Guardar pares dirigidos, no solo simetricos. La vuelta puede diferir por
     sentidos, accesos, obras o criterio de motor.
   - Para 174 municipios: 174 * 173 = 30.102 pares dirigidos.
   - Para puntos NGMEP, el numero crece con cada nucleo incorporado.
   - Cada par debe guardar origen, destino, km, minutos, motor, fuente de grafo,
     version, fecha de calculo y estado de homologacion.

3. Calculo de dieta:
   - Un itinerario multiparada se descompone en tramos:
     Granada -> Albolote -> Mecina Bombarón -> Motril -> Granada.
   - La app suma kilometros y minutos por tramo.
   - Si falta un tramo en la matriz, el expediente queda en subsanacion o
     revision tecnica.
   - Si el empleado ajusta km manualmente, debe aportar motivo y justificante.

4. Servidor de rutas:
   - OSRM no es una simple cache: es un motor HTTP que carga un grafo viario
     preprocesado y calcula rutas, tiempos y matrices sobre ese grafo.
   - La consulta de ruta debe pedir alternativas (`alternatives=3`) cuando el
     motor lo permita. La ruta recomendada no se impone como unica ruta valida:
     el gestor puede seleccionar una alternativa por corte, obras, seguridad,
     instruccion del servicio o acceso operativo.
   - Ambito del grafo: provincia de Granada + 15 km de buffer alrededor del
     limite provincial. No hace falta cargar España entera para el uso ordinario
     de dietas.
   - El ambito no queda fijo ni se infiere en codigo: `VEC_OSRM_SCOPE_NAME`
     define la etiqueta funcional y `VEC_OSRM_SCOPE_BOUNDS` define el control
     rapido de coordenadas en formato canonico
     `minLat,minLon,maxLat,maxLon`. Ambos son obligatorios si se configura el
     conector. El recorte real del grafo se gobierna por el PBF usado por OSRM.
   - La app VEC no debe consultar servicios externos. Debe llamar a un OSRM
     interno configurado con `VEC_OSRM_BASE_URL`, por ejemplo
     `http://127.0.0.1:5000` en desarrollo o una URL interna de Diputacion. Esa
     URL aprueba exactamente esquema, host y puerto; no admite ruta, consulta,
     credenciales ni fragmento.
   - `VEC_OSRM_ALLOWED_CIDRS` debe enumerar de forma positiva y canonica las
     redes a las que puede resolver el host. No tiene valor predeterminado y no
     se admite una red universal `/0`. El cliente no usa proxies del entorno,
     comprueba la IP al conectar y no sigue redirecciones, ni siquiera hacia
     otro destino aparentemente interno.
   - Las cuatro variables (`BASE_URL`, `SCOPE_NAME`, `SCOPE_BOUNDS` y
     `ALLOWED_CIDRS`) son atomicas: ausencia parcial o formato incorrecto
     impiden construir el adaptador en lugar de ampliar el alcance.
   - El OSRM interno se actualiza con extractos OSM/IGN versionados. Objetivo:
     actualizacion diaria si hay datos nuevos; maximo semanal. Mensual solo
     seria aceptable como contingencia.
   - El despliegue debe reconstruir el grafo en paralelo, probar salud/rutas
     semilla y cambiar el servicio de forma atomica para no cortar expedientes.
   - El backend VEC rechaza coordenadas fuera del ambito Granada + 15 km; el
     recorte real debe aplicarse tambien al PBF usado para construir OSRM.

5. Cache/matriz:
   - OSRM calcula la ruta al vuelo, pero VEC puede cachear resultados y debe
     precalcular la matriz provincial homologada.
   - La cache no sustituye la auditoria: cada tramo guarda motor, version de
     grafo, fecha de calculo, km, minutos y estado de homologacion.
   - Si se selecciona una ruta alternativa, el expediente debe guardar
     alternativa elegida, motivo y justificante/instruccion cuando proceda.

6. Versionado:
   - No se sobrescriben matrices usadas en expedientes cerrados.
   - Una nueva version anual o por cambio de grafo crea nueva matriz.
   - Cada liquidacion conserva `matrix_version`, tramos, total y recibo de
     auditoria.

## Implementacion actual

El prototipo VEC ya expone:

- `province_localities`: los 174 municipios INE de Granada.
- `province_route_points`: municipios y puntos de ruta; incluye Mecina Bombarón
  como nucleo semilla pendiente de importacion NGMEP completa.
- `province_route_matrix`: metadatos de cobertura y pares esperados.
- `province_route_pairs`: pares semilla para probar UI.
- `province_itinerary_examples`: ejemplo multiparada de coche propio.

La pantalla de Dietas/Kilometraje ya permite seleccionar salida, paradas y
destino desde el catalogo de puntos de ruta. Con las coordenadas disponibles
pinta el recorrido en un visor Leaflet sobre OpenStreetMap y genera enlace de
apertura en OpenStreetMap. El frontend solicita la geometria de carretera al
backend VEC (`/api/vec/dietas/road-route`); el backend solo consulta el OSRM
interno configurado en la URL y redes exactas, no sigue redirecciones y rechaza
el OSRM publico de demo.
La linea visible sigue el grafo viario y no une coordenadas en recta cuando el
motor interno esta disponible. La geometria oficial de carretera debe importarse
desde OSRM/openrouteservice on-premise con version de grafo y matriz auditada.

Antes de liquidar importes reales, falta ejecutar el ETL completo NGMEP + motor
de rutas on-premise y marcar la matriz como homologada.
