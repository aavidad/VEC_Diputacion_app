# Ensayo de adjudicación de Provisión

Esta CLI prepara una propuesta sintética desde valoraciones de Provisión ya calculadas. No guarda datos ni ejecuta una resolución. No acredita admisión, firma, publicación, auditoría ni efectos en Personal o RPT.

Desde la raíz del repositorio:

```sh
GOCACHE=/dev/shm/go-build go run ./cmd/vec-simular-adjudicacion < internal/modules/provision/adapters/simulacion/adjudicacion_ejemplo.sintetico.json
```

El ejemplo tiene dos personas y dos vacantes. Ambas prefieren A antes que B. La persona A obtiene A y la persona B obtiene B. Las valoraciones proceden de `domain.Calcular`, que usa `internal/shared/baremacion`; la adjudicación no vuelve a calcular méritos.

La petición contiene `configuracion` y `entrada`. El JSON exige claves exactas, campos obligatorios, puntos como cadenas enteras en micropuntos y un único documento de hasta 2 MiB. Rechaza claves repetidas, campos desconocidos y más de 24 niveles de profundidad. Los errores se devuelven como códigos y campos, sin repetir los datos recibidos.

La configuración identifica proceso, versión, bases y su huella, política y su versión, método, incompatibilidad y cadena de desempates. Fija también la versión del motor de valoración, la versión de reglas y su huella. El ensayo solo soporta la selección explícita `aceptacion_diferida_personas_proponen_ensayo_v1`, prioridad `total_descendente` e incompatibilidad `un_puesto_por_persona`. Otro método devuelve `no_soportado`. No hay método predeterminado ni una afirmación de que esta familia responda a unas bases legales aprobadas.

Cada elemento de `desempates` identifica una regla del desglose y el sentido `mayor` o `menor`. El orden es significativo y la cadena no puede estar vacía. Las referencias, los nombres y los UUID no son criterios de desempate. No se admiten fórmulas libres.

La entrada declara `sintetico: true`, universo y versión, y si está cerrado. Cada solicitud identifica una referencia opaca de persona y su instantánea, con versión propia y preferencias ordenadas. Hay como máximo 256 personas, 256 vacantes y 4096 preferencias. Cada vacante tiene una única plaza y cada persona recibe como máximo una. La colección de preferencias conserva su orden. El orden de transporte de personas y vacantes solo se usa para representar y sellar el resultado de forma reproducible.

Cada preferencia indica `admitida`, `excluida`, `pendiente` o `renunciada`. Estos estados son declaraciones del ejercicio sintético. Una preferencia admitida requiere una valoración completa del mismo proceso, puesto, instantánea y configuración. Se comprueba la huella del resultado recibido. Las huellas prueban coherencia reproducible; quien aporta un JSON puede volver a calcularlas. No prueban autenticidad ni constituyen una firma.

El algoritmo funciona por rondas simultáneas: cada persona sin plaza propone a su siguiente preferencia admitida. Cada vacante compara las propuestas y a su ocupante provisional mediante el total y la cadena configurada. Una persona desplazada continúa con su siguiente preferencia. Exclusiones y renuncias se omiten sin convertirlas en puntos cero.

Un universo abierto, una admisión pendiente o una valoración incompleta devuelve `pendiente` sin asignaciones. Una igualdad entre máximos después de toda la cadena también detiene el conjunto global. Esta decisión es conservadora: puede detenerse en una ronda aunque una propuesta posterior pudiera superar ese empate. No selecciona provisionalmente a una persona empatada para seguir el algoritmo.

Cuando el ensayo termina, devuelve `propuesta_simulada`, las asignaciones, personas y vacantes sin asignación, incidencias y huellas de configuración, entrada y resultado. La salida conserva referencias de solicitud, versión, preferencia y valoración. Una rectificación se ensaya con otra versión de universo y solicitud; no modifica un resultado anterior. No calcula la cadena jurídica de alegaciones o renuncias.

Las pruebas cubren desplazamientos entre tres personas y dos vacantes, preferencias, toda la cadena y su sentido, empates, fuentes no disponibles, exclusión, renuncia y rectificación sintéticas, reproducción, rechazos de valoraciones cruzadas y entrada JSON hostil. Este corte no monta una API ni un recorrido de navegador y no incluye persistencia.
