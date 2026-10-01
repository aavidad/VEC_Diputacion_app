# Ensayo de saldo con permisos

La CLI lee un catálogo y escenarios sintéticos y devuelve JSON. Conserva los minutos
trabajados y muestra el crédito de permiso por separado. Todavía no consulta PostgreSQL,
no registra saldo y no acredita concesiones ni aprobación de RRHH.

Desde la raíz del repositorio:

```sh
go run ./cmd/vec-cronos-ensayo \
  -catalogo data/demo/reglas/cronos-efectos-permisos.demo.json \
  -catalogo-sha256 3e24a2b1a0643dc721e4edd49178ee99e07aca9a6d3a844dc68564f588af290c \
  -escenarios data/demo/cronos/escenarios-saldo.json \
  -escenarios-sha256 b55c791f7173b59ebaaff1d0d3548b273ec47f17d8544c22873704e4ad8f0c65 \
  -idioma es
```

Los archivos son obligatorios, regulares y de hasta 1 MiB; no se admite un enlace en el fichero final. El idioma debe
existir en el catálogo. La salida incluye las dos huellas exactas, la versión de política,
la regla aplicada y las referencias de programación, solicitud y resolución. `textos`
explica las causas en el idioma elegido. Un resultado ausente se representa con `null`.
Una entrada incompatible termina con código 1 y un error JSON sin reproducir la entrada.

El primer escenario tiene una programación sintética de 420 minutos y cero trabajo:
saldo base −420, permiso computado 420, saldo ajustado 0. Los minutos vienen del JSON.
Los casos parciales, con trabajo concurrente, permisos coincidentes o fuentes incompletas
dejan el crédito y el saldo ajustado ausentes. Retirar una regla del JSON retira su efecto;
hay que actualizar ambas huellas y la huella de política en los escenarios para ensayarlo.

Las tres reglas de ejemplo usan el cómputo de servicio efectivo de
[EBEP, artículo 49.a, b y c](https://www.boe.es/buscar/act.php?id=BOE-A-2015-11719#a49),
limitado aquí a jornada completa y a las referencias exactas del catálogo y colectivo.
Su aplicación horaria queda como ensayo. No se usa el artículo 17.4 del reglamento de 2010:
Diputación publica su anulación por acuerdo de 02/09/2026 en
[Normativa de Recursos Humanos](https://www.dipgra.es/contenidos/normativa-recursos-humanos/).
El paquete de ejemplo se retira antes de producción. Las huellas verifican bytes;
la autorización, el gobierno de reglas y la lectura conjunta durable siguen pendientes.

Las reglas de ejemplo se aplican desde 01/01/2026 inclusive hasta 01/01/2027 exclusivo.
La fuente se fija a la publicación consolidada de 30/07/2025, consultada el 01/10/2026.
El intervalo es el alcance del ensayo, no una declaración de vigencia legal. `sin_trabajo`
es un hecho explícito: un futuro lector deberá derivarlo de duraciones exactas, nunca de
minutos redondeados. Si falta o hay trabajo, no se concede crédito de permiso.
