# Catalogos configurables y gobernados

Fecha de referencia: 14-07-2026.

## Objetivo

Las opciones de negocio no se fijan en el codigo. Estados de una bolsa, tipos
de merito, categorias, certificados, motivos, canales y futuras «cosas cuatro»
se publican como versiones de un catalogo administrable. Los consumidores fijan
la version exacta utilizada y nunca dependen implicitamente de «la ultima».

Esto no convierte en configurables las invariantes tecnicas de seguridad. Por
ejemplo, `borrador`, `publicado` y `retirado` son estados internos del gobierno
de una configuracion; no son la lista de estados de una bolsa.

## Modelo implementado

Cada version contiene:

- identificador, version, revision optimista y referencia a la version anterior;
- modulo propietario, nombre, descripcion, fuente y motivo de creacion;
- entradas con clave estable, etiqueta, descripcion, orden, vigencia desde/hasta
  y atributos extensibles;
- estado tecnico de gobierno;
- actores, instantes, aprobaciones y motivos de publicacion o retirada;
- huella SHA-256 canonica de la instantanea completa.

Las entradas se ordenan de forma canonica y las fechas se normalizan a UTC antes
de calcular la huella. Se limitan numero de entradas, atributos y tamano total
para evitar configuraciones que agoten memoria o almacenamiento.

## Gobierno

```text
crear borrador (revision 1)
          |
          +--> actualizar con revision esperada (revision N+1)
          |
          v
publicar por una persona distinta de creador/ultimo modificador
          |
          +--> consumir por catalogo + version + clave + instante
          |
          v
retirar con nueva aprobacion y persona distinta del publicador
```

Cada paso exige autenticacion de nivel alto, decision RBAC+ABAC obtenida dentro
del caso de uso, finalidad, motivo, correlacion, auditoria encadenada y evento de
outbox. Metadatos, auditoria y configuracion se confirman conjuntamente en el
adaptador de memoria.

La escritura concurrente utiliza revision optimista: de dos modificaciones
basadas en la misma revision solo una puede confirmarse. La otra debe recargar,
comparar y volver a proponerse; nunca sobrescribe silenciosamente.

## Aplicacion a Bolsa

Los primeros catalogos previstos son, como minimo:

- clases y estados públicos/internos de procesos selectivos;
- tipos de solicitud, merito, acreditacion, alegacion y llamamiento;
- causas de disponibilidad, renuncia, exclusion, sancion y reincorporacion;
- categorias profesionales, tipos de contrato/nombramiento y causas de cese;
- plantillas de comunicacion y preferencias de aviso.

La primera proyeccion conectada es el catalogo compartido de categorias
profesionales. Su procedencia, contrato publico, limites de demostracion y
criterios de aceptacion se detallan en
[`catalogo_categorias_profesionales.md`](catalogo_categorias_profesionales.md).

Las transiciones entre estados no se guardaran como atributos informales. El
siguiente corte del nucleo es una definicion de flujo y reglas versionada que
referenciara estas claves de catalogo y conservara la decision de cada cambio.

## Pendiente de produccion

- pantallas de alta, comparacion, simulacion, aprobacion y retirada;
- esquemas gobernados para validar atributos complejos;
- repositorio PostgreSQL con restricciones unicas y bloqueo optimista;
- firma/sello de configuraciones criticas y custodia WORM de su evidencia;
- dependencias entre catalogos y comprobacion de impacto antes de retirar;
- exportacion e importacion firmada entre entornos, sin datos personales.
