# Dietas 000012 — borrador de liquidación económica

Número reservado por Codex-G en `RESERVAS_MIGRACIONES.md`, fuera de Git.
Requisitos: D6 y D9 de la ficha de Dietas. No es una migración instalable.
No contiene `UP`, `DOWN`, concesiones ni funciones ejecutables.

La operación futura registrará una liquidación como acto propio. Su efecto
deberá unir en una transacción serializable:

1. La comisión y la versión enviada bloqueadas, junto con su huella.
2. La autorización nominal de liquidación y la competencia vigente de
   persona, unidad y etapa, revalidada por la fachada de Personal.
3. La versión publicada del catálogo económico, con acto y huella.
4. Las líneas originales y admitidas o rechazadas, sus motivos y los totales
   recalculados en céntimos con control de desbordamiento.
5. La instantánea económica de solo adición, recibo, auditoría y evento.
6. La transición a la fiscalización del circuito sobre esa misma versión.

La clave repetida con el mismo contenido recuperará la instantánea y el
recibo conservados, tras comprobar el acceso vigente. La misma clave con
contenido distinto producirá conflicto. Una rectificación será otro acto
enlazado; nunca actualizará la instantánea anterior. El PDF definitivo
consumirá ese contenido exacto sin consultar tarifas posteriores.

La aprobación genérica de `decidir_comision_v2` no basta para efectuar esta
liquidación. La candidata deberá impedir el avance de fase sin instantánea;
se concretará con el contrato publicado, conservando las operaciones previas.

Antes de redactar SQL ejecutable faltan la fachada de competencia de Personal,
los perfiles y contrato de AUT27 y la publicación del catálogo económico.
D debe fijar la postimagen y su posición en `~/Trabajo/hito6/ORDEN_SQL_NUCLEO.md`.
Después se prepararán ACL mínimas, guardas, negativas y ensayo sobre clon local,
con dos revisiones independientes del hash exacto. No se reaplica ninguna
migración instalada ni se ejecuta una reversión sobre historia conservada.
