---
name: orquestar-vec
description: Coordina y especializa agentes de VEC a demanda. Usar al repartir módulos, crear subagentes o descendientes, elegir modelo/esfuerzo, evitar solapamientos y reunir entregas.
---

# Coordinar VEC a demanda

Gestionar el encargo recibido hasta su entrega. El director y cualquier subagente
pueden crear otros subagentes para tareas concretas que les ayuden. Transmitir esta
facultad a los descendientes. Adaptar la jerarquía a las dependencias reales.

## Delegar

1. Inspeccionar la pieza existente y definir el resultado funcional pendiente.
2. Separar trabajo independiente de trabajo que espera un contrato o una entrega.
3. Asignar a cada hijo objetivo, archivos exclusivos, entradas, resultado esperado
   y comprobación proporcionada. Especificar modelo y esfuerzo al crearlo.
4. Subdividir sólo el ámbito recibido. Ceder al hijo los archivos que vaya a editar
   y no editarlos simultáneamente. Dejar las interfaces compartidas con un responsable.
5. Registrar el identificador real del hijo en la conversación de coordinación.
   Esperar su entrega, revisarla y comunicar el resultado al padre.
6. Reutilizar o cerrar sesiones terminadas según las herramientas disponibles.

No imponer una cuota total de especialistas ni exigir un organigrama fijo.
La concurrencia y las herramientas disponibles siguen sujetas al runtime; comprobar
el límite efectivo y repartir por tandas si se alcanza. No confundir el número de
perfiles con agentes ejecutándose, ni prometer concurrencia infinita.

## Elegir especialidad y modelo

| Encargo | Modelo | Esfuerzo |
| --- | --- | --- |
| Código, integración, pruebas con código | gpt-5.6-terra | medium |
| Documentación y síntesis sustancial | gpt-5.6-sol | medium |
| Inventarios o ediciones mecánicas breves | gpt-5.6-luna | low |
| Arquitectura difícil, revisión sensible, bloqueo complejo | gpt-6-astra | high |

Usar xhigh sólo ante dificultad concreta o encargo de revisión intensa. Elegir
primero un perfil existente; si falta especialidad, crear un agente con instrucciones
y parámetros explícitos. Pedir al director que conserve un perfil o skill reutilizable
cuando aporte conocimiento que todavía no exista. Un revisor de sólo lectura conserva
esa restricción en todos sus hijos. No ampliar permisos para delegar una acción denegada.

## Reunir e integrar

El padre responde por sus hijos. El director principal integra en la única rama
canónica y crea commits pequeños. Conservar índice, HEAD y trabajo ajenos.
Entregar archivos, interfaces, dependencias, comprobaciones y límites pendientes.
Leer sólo las skills y referencias que necesita la tarea. Mantener una o dos
compilaciones pesadas simultáneas, incluso cuando haya más agentes activos.
