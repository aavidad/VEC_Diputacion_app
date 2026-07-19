# Integración de Cronos y Dietas en el Portal del Empleado

Fecha de corte: 19 de julio de 2026.

## Resultado

El Portal del Empleado conserva el mismo armazón visual, menú lateral, tema,
cabecera, accesibilidad y vistas de Gestión de Bolsas. Cronos y Dietas se han
incorporado como módulos intercambiables mediante composición, sin copiar el
shell ni crear identidades o sesiones propias.

Rutas estables:

- `#portal`: catálogo completo del portal;
- `#bolsa/resumen` y el resto de vistas `#bolsa/*`: Gestión de Bolsas;
- `#cronos`: autoservicio de jornada, fichajes, permisos y recibos;
- `#dietas`: autoservicio de comisiones, rutas, gastos y recibos.

En el inicio se mantiene visible el catálogo completo. Dentro de un módulo, el
lateral muestra únicamente Inicio y el módulo activo; el submenú extenso de
Bolsa solo aparece en las rutas de Bolsa. Esto evita que las opciones de gestión
queden bajo el pliegue y que Bolsa parezca activa al consultar Cronos o Dietas.

## Decisiones reutilizables en producción

- `portal-modulos-coordinador.js` es la raíz de composición del frontend. El
  router y las vistas no contienen reglas de negocio de Cronos ni Dietas.
- El catálogo productivo procede de los manifiestos registrados por el núcleo
  mediante `/api/vec/modules`; la plantilla HTML no enumera módulos.
- Bolsa, Cronos y Dietas reciben por referencia el mismo `ContextoActor`
  validado e inmutable. Las capacidades se entregan separadas y se aplican con
  denegación por defecto.
- Las capacidades de la demostración son exclusivamente de autoservicio propio.
  No se conceden capacidades de jefatura, aprobación de terceros ni auditoría
  general por inferencia de rol.
- Los estilos de módulo heredan las variables del tema común. La portada añade
  color semántico estable: Bolsa azul/índigo, Cronos cian/violeta y Dietas
  verde/naranja. Los módulos inactivos permanecen sobrios.
- Al abandonar un módulo se retiran sus manejadores. Un montaje asíncrono
  obsoleto no puede sustituir la vista más reciente.
- Los recibos usan el puerto documental común. El PDF de presentación incluye
  identidad visual institucional, texto equivalente al recibo real, marca DEMO,
  referencia opaca y QR de cotejo sin datos personales.
- Dietas consume un puerto neutral de cálculo de rutas y un visor cartográfico
  independiente. La presentación conecta ese puerto a un mediador Go, OSRM y
  teselas OSM locales; la vista no conoce el motor ni su dirección.
- El mediador devuelve distancia, duración, tramos, geometría y versión
  gobernada del grafo. No existe degradación silenciosa a una ruta sintética:
  un fallo se muestra como estado no disponible.

## Cartografía interna de Dietas

La ruta y el mapa se ejecutan íntegramente en la composición Docker de
presentación:

```text
Navegador -> proxy en 127.0.0.1:8081
          |-> portal de presentación
          |-> /tiles/osm/{z}/{x}/{y}.png -> TileServer GL
          `-> POST /api/presentacion/cartografia/rutas
                                          -> mediador Go -> OSRM
```

El proxy es el único servicio que publica un puerto. Portal, mediador,
TileServer GL y OSRM se separan en redes internas; el proxy no pertenece a la
red exclusiva de OSRM. El navegador recibe la geometría y las teselas por el
mismo origen, sin cookies, identidad ni peticiones a servicios cartográficos
públicos.

El conjunto actual se obtiene de
`deploy/osrm-granada/data/granada-buffer.osm.pbf`, cuya huella SHA-256 es
`53aba0ad43c45c62a44ee54fa9fb68308427957c5b5504cbd19a74fc49672724`.
El grafo se declara como `granada-buffer-osrm-v1-53aba0ad43c4` y las teselas
activas como `20260719T115334Z-53aba0ad43c4`. La versión se transporta en la
respuesta para que una futura liquidación pueda conservar la evidencia exacta
del cálculo; no debe recalcularse retroactivamente un expediente cerrado.

El detalle de generación, activación, reversión y paso a producción se mantiene
en [Cartografía interna del módulo Dietas](cartografia_interna_dietas_2026-07-19.md).

## Límite de la demostración

Son sustituibles y deben excluirse del despliegue productivo los adaptadores o
fixtures cuyo nombre contiene `presentacion`. Mantienen datos sintéticos en
memoria volátil y no registran, firman, notifican, pagan ni persisten actos. El
mediador de presentación tampoco concede validez administrativa a una ruta;
solo permite probar con cartografía real el contrato que se conservará.

Las vistas, contratos, presentadores, catálogos i18n, estilos y coordinador son
candidatos a conservar, sujetos a la aceptación de RRHH y Sistemas. Al avanzar
a producción se conectarán adaptadores autenticados a los puertos existentes;
el diseño evita rehacer la web.

## Verificación reproducible

La evidencia anterior a la cartografía real cubría navegación de Bolsa, Cronos
y Dietas, transiciones desde la portada y el recibo PDF de Cronos. No se usa esa
línea base para dar por validado el nuevo recorrido OSM/OSRM.

La composición completa se arranca y comprueba exclusivamente en Docker:

```bash
scripts/arrancar_presentacion_rrhh.sh
```

El script espera la salud de portal, proxy, mediador, OSRM y teselas, y ejecuta
una prueba rápida desde la red Docker de borde. La revisión actual de 36 vistas
y 22 flujos se lanza mediante:

```bash
docker compose --profile presentacion --profile herramientas-presentacion run \
  --rm --no-deps revision-web-presentacion
```

Su flujo de Dietas exige una respuesta real del mediador, versión gobernada,
geometría OSRM y carga efectiva de las teselas OSM. Repite el recorrido a 1440,
1024 y 390 píxeles. La ejecución cerrada el 19 de julio de 2026 obtuvo
**174/174 escenarios correctos, 174 capturas y cero hallazgos**.

Las capturas, los informes y los PDF de revisión se generan bajo `var/`, fuera
del control de versiones. Playwright, Chromium, OSRM y TileServer GL no se
instalan ni se ejecutan directamente en el anfitrión.
