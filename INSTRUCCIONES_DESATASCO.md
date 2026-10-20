# Desatascar el trabajo de VEC

Actualizado el 20 de octubre de 2026. Las instrucciones de septiembre y sus
ampliaciones están en el [archivo histórico](docs/historico/INSTRUCCIONES_DESATASCO_20260903_20260919.md).
No fijan la prioridad ni el método de trabajo actuales.

La prioridad vigente es Bolsa y Contratación temporal, con las bases que
necesitan. Se trabaja en copias locales con datos ficticios; cidonia está
congelada. El único caso de aceptación es el
[apartado 4.0 del flujo](docs/producto/FLUJO_APP.md#40-caso-de-aceptación-de-la-presentación).

Ante un atasco:

1. Compruebe la tarea en el [tablero](docs/producto/TABLERO_TAREAS.md), la
   configuración y la [receta de copia local](deploy/principal/LEEME_copia_local.md).
   Revise qué está integrado antes de programar otra solución.
2. Si faltan datos ficticios, corrija solo los de su propia copia. No cambie
   el código para fabricar datos ni quite comprobaciones de seguridad.
3. Si se repite el mismo fallo tres veces o pasan veinte minutos sin avance,
   escriba el bloqueo y el error concreto en el canal de coordinación.
   Continúe únicamente con lo que no dependa de ese bloqueo.
4. Conserve el trabajo en su rama. Haga commit y comunique el estado al menos
   cada treinta minutos. Si la cabeza está congelada esperando revisión o
   integración continua y no hay trabajo sin guardar, basta con comunicar el
   estado; dirección lo aclaró el 20/10 a las 12:22.
5. Entregue el cambio para revisión de dirección con sus comprobaciones y
   límites. Dirección integra; un ensayo parcial no cierra la fase.

Las reglas completas están en [AGENTS.md](AGENTS.md) y en el
[plan, apartado 10](docs/producto/PLAN_IMPLEMENTACION.md#10-cómo-se-trabaja).
El [estado del proyecto](ESTADO_PROYECTO.md) resume los avances y la
[guía de recorrido](GUIA_RECORRIDO_ALBERTO.md) explica cómo comprobarlos.
