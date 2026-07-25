# Revisión independiente O3-04

Fecha: 24 de julio de 2026.

## Veredicto del candidato 88d3250

`NO-GO`. O3-04 continúa abierta y no aumenta el contador de tareas
verificadas.

Tres revisores distintos del productor comprobaron el límite Go, la barrera
PostgreSQL y el recorrido completo. Las pruebas funcionales y transaccionales
eran verdes, pero la revisión estática encontró vías de entrada que la batería
todavía no atacaba.

## Bloqueantes encontrados

1. Algunas comparaciones SQL sensibles usaban lógica de tres valores. Un
   `null` controlado podía evitar que una condición `<>` fuese verdadera.
2. PostgreSQL aceptaba números o instantes semánticamente equivalentes, aunque
   su representación JSON no fuese la que puede rehidratar el dominio Go.
3. La decisión VEC V3 ligaba el análisis funcional, pero no el conjunto exacto
   de fuentes, la unidad de la política ni el motivo de rectificación.
4. El replay terminal cotejaba identificadores principales, pero no un
   compromiso íntegro de la operación recibida.
5. El candidato modificaba migraciones históricas en vez de proporcionar una
   actualización posterior compatible.
6. Faltaban listas positivas exactas para el agregado siguiente y pruebas de
   downgrade con historia O3.

## Aspectos que sí superaron la revisión

- `SECURITY DEFINER` fija `search_path` y comprueba el rol de sesión.
- La transacción exige aislamiento serializable y aplica CAS.
- Versión, actuación, consumos, auditoría, outbox, recibo y reserva se
  confirman o revierten juntos.
- El adaptador reconcilia una respuesta perdida y recupera el mismo recibo
  después de reconstruir servicio y conexión.
- RLS, ACL, revocación viva, concurrencia y fallos inyectados permanecen
  verdes.

## Corrección exigida

La tarea solo podrá cerrarse mediante un commit posterior que:

- restaure las migraciones históricas y añada una migración de actualización;
- niegue `null`, claves desconocidas y representaciones no canónicas;
- incorpore a la decisión las huellas y campos probatorios restantes;
- cierre el replay y la reversión con historia;
- pruebe cada ataque con un snapshot completo antes y después;
- supere una segunda revisión independiente.

Esta acta conserva el `NO-GO` aunque el candidato se corrija. El veredicto de
cierre se documentará en un acta nueva para que la trazabilidad no reescriba
la historia.
