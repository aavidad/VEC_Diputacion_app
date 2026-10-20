# Referencia organizativa de Contratación temporal

## Contrato de los adaptadores

Los repositorios de peticiones del centro y de entrega a RRHH reciben la
organización de la composición al construirse. Esa referencia se conserva
en la instancia y forma parte del ámbito del recurso de autorización. Las
funciones que construyen esos recursos también la exigen como argumento.
Una referencia vacía o inválida impide construir el repositorio o el recurso.

Los repositorios de ajustes de reglas, catálogo de plantillas y plantillas
documentales admiten la referencia opaca explícita que reciben de composición.
Conservan el cotejo exacto entre esa referencia y el material de la operación.

El ámbito cambia la huella del recurso autorizado. El documento JSON y su
huella de material conservan el contrato SQL existente. Una decisión emitida
para otra organización no coincide con el recurso del repositorio y se
rechaza antes de confirmar el efecto.

## Activación de este corte

No incorpora configuración ni migraciones nuevas. Las pantallas de
«Peticiones del centro», su entrega a RRHH y las de ajustes y plantillas
conservan su montaje actual. No hay actos de Administración adicionales.
La aplicación se arranca con su configuración existente.

Este corte prepara los contratos para recibir una referencia maestra. La
composición todavía conserva su referencia anterior: no acredita la
activación de Contratación con una organización publicada ni desbloquea
la derivación de nombramientos.

## Continuación pendiente

El montaje debe leer una organización canónica publicada y una unidad de
Personal de una configuración explícita, comprobar su fuente y transmitirlas
a todos los consumidores: contexto de autorización, padre e hijo,
idempotencia y adaptadores. Las funciones SQL de petición, entrega, reglas,
plantillas y Documentos deben conservar esa misma referencia. No se amplía
la validación de organizaciones maestras ni se crean correspondencias
implícitas.

El ensayo final requiere publicar las fuentes por su vía nativa, recorrer
las pantallas con las referencias nuevas y comprobar recibos y denegaciones
después de reiniciar la aplicación y PostgreSQL.
