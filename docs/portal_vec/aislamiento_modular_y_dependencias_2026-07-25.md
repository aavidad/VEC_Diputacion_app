# Aislamiento modular y propagación de fallos

**Fecha de decisión:** 25 de julio de 2026.

## Regla obligatoria

Un fallo solo puede propagarse a una capacidad que declare depender de la
capacidad fallida. La pertenencia al mismo portal, proceso, contenedor o
repositorio no constituye una dependencia.

Ejemplos:

- una caída de Dietas o de su cartografía no impide usar Bolsa, Contratación
  temporal o Cronos;
- una caída de Bolsa bloquea la selección de candidaturas de Contratación
  temporal, pero no el análisis presupuestario de un expediente ya iniciado;
- una caída del generador documental bloquea generar o firmar un documento,
  pero no consultar el expediente si esa lectura no necesita el documento;
- la falta de identidad, autorización o auditoría obligatoria impide la
  mutación administrativa afectada. No convierte una denegación en éxito ni
  obliga a retirar una consulta pública independiente.

## Unidad de aislamiento

La unidad preferida es la **capacidad**, no el módulo entero. Cada caso de uso
debe declarar:

1. capacidades comunes obligatorias;
2. capacidades de otros módulos obligatorias;
3. dependencias opcionales que solo enriquecen la respuesta;
4. comportamiento degradado permitido;
5. estado que se mostrará cuando una dependencia no esté disponible.

No se permiten importaciones de tablas, adaptadores o estado interno de otro
módulo. La comunicación se realiza mediante puertos, casos de uso o eventos
versionados.

## Estados operativos

| Estado | Significado | Comportamiento del portal |
| --- | --- | --- |
| `disponible` | Dependencias obligatorias sanas. | Lecturas y acciones autorizadas. |
| `degradado` | Falta una dependencia opcional. | Se conserva el recorrido y se explica la función ausente. |
| `no_disponible` | Falta una dependencia obligatoria del caso de uso. | Solo se bloquea el caso de uso dependiente. |
| `mantenimiento` | Retirada gobernada y temporal. | Sin reintentos agresivos; mensaje y referencia de soporte. |
| `denegado` | El actor carece de autorización. | Respuesta cerrada, sin revelar si existe el recurso. |

`no_disponible` y `denegado` no son equivalentes. El navegador nunca debe
deducir permisos a partir de una sonda de salud.

## Dependencias transversales

Identidad, autorización, auditoría, tiempo confiable, documentos, firma,
registro y comunicaciones continúan siendo puertos intercambiables. Que sean
transversales no autoriza a convertirlos en un proceso monolítico.

Para una operación con efecto jurídico:

```text
identidad válida
→ autorización exacta
→ caso de uso
→ persistencia y auditoría atómicas
→ recibo
```

La ausencia de cualquiera de las dependencias obligatorias produce fallo
cerrado de esa operación. Las lecturas o módulos que no dependan de ella
permanecen disponibles.

## Aplicación actual de la decisión

### Presentación Docker

- El perfil `presentacion` arranca únicamente el portal y su proxy.
- Dietas, mediador OSRM y teselas pertenecen al perfil opcional
  `presentacion-cartografia`.
- El proxy base no declara `depends_on` hacia servicios cartográficos.
- Los destinos opcionales se resuelven dinámicamente para que su ausencia no
  impida arrancar Nginx.
- `scripts/tests/test_presentacion_modular.py` impide reintroducir el
  acoplamiento.

### Composición del navegador

- El shell, el catálogo y la identidad de presentación forman la base común.
- Contratación temporal, Cronos y Dietas se importan dinámicamente por
  separado.
- Solo se ejecutan los cargadores de módulos incluidos en el ámbito autorizado
  del actor; una superficie denegada no se sondea ni se precarga.
- Cada módulo conserva su propio grupo de dependencias internas.
- `Promise.allSettled` aísla los rechazos entre módulos.
- Cada carga tiene un plazo máximo de dos segundos. Una importación pendiente
  pasa a `no_disponible` y no puede inmovilizar indefinidamente el portal.
- Un error al crear el adaptador de un módulo deja únicamente ese módulo sin
  componer.
- Antes de resolver una sesión nueva se elimina la composición anterior. Una
  identidad ausente o inválida nunca puede conservar permisos de la carga
  precedente.
- Cada intento recibe una generación monotónica. Una carga antigua que termine
  después de una sesión nueva no puede volver a publicar catálogo, contexto ni
  permisos.
- La regresión de
  `web/static/portal-empleado/portal-modulos-coordinador.test.mjs` fuerza la
  caída y el bloqueo indefinido de Dietas y demuestra que Cronos continúa
  disponible. La navegación diferencia `no_disponible` de `denegado`.

La presentación no concede un perfil predeterminado. Una URL sin el parámetro
de perfil permanece cerrada y debe volver al selector de recorridos. Nunca se
interpreta la ausencia de perfil como administración.

## Trabajo pendiente para producción

1. Ampliar el manifiesto de módulos con dependencias obligatorias y opcionales
   versionadas, después de aprobar su contrato compatible.
2. Construir la raíz interna por capacidades y no por un único arranque
   monolítico.
3. Publicar salud por capacidad sin datos personales ni información útil para
   enumerar permisos.
4. Aplicar límites de tiempo, cancelación, cortacircuitos y reintentos con
   presupuesto por conector.
5. Separar procesos o contenedores cuando cambie la frontera de confianza,
   exposición de red, clasificación del dato o escala; no por la mera
   existencia de un paquete Go.
6. Ejecutar pruebas de caos que retiren cada módulo y cada conector y acrediten
   que solo fallan sus dependientes declarados.

## Puerta de aceptación

Ninguna capacidad se considera integrada sin estas pruebas:

- caída antes de empezar;
- caída durante una lectura;
- caída antes y después del `COMMIT`;
- reinicio y recuperación;
- tiempo agotado y cancelación;
- dependencia opcional ausente;
- dependencia obligatoria ausente;
- verificación HTTP o de escritorio de que los módulos independientes siguen
  disponibles.

La matriz de dependencias y las pruebas se actualizan en el mismo commit que
incorpore una dependencia nueva.
