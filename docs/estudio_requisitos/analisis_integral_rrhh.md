# Análisis integral del portal de Recursos Humanos

Este estudio (julio y septiembre de 2026) está absorbido en el
[PRD](../producto/PRD.md): mandato institucional, modelo de datos, módulos,
principios de las reglas configurables, intercambio entre módulos, superficies,
clasificación de la información, ficha integral del empleado y decisiones
pendientes. La exposición de cada módulo por red está en
[PRD §5, «Superficies»](../producto/PRD.md#superficies); el estado real de cada
módulo, en
[PLAN_IMPLEMENTACION §2](../producto/PLAN_IMPLEMENTACION.md#2-estado-actual) y
en [docs/plan_modulos/](../plan_modulos/). Las fichas y estudios de cada
materia siguen en esta carpeta (ver su [README](README.md)).

La versión completa anterior, con su lista de especificaciones relacionadas,
el inventario de módulos de septiembre y la secuencia de ampliación, está en la
historia de Git de este fichero. Aquí queda solo lo que no pasó a otro
documento.

## Fechas, dinero y distancias

- Los instantes van en UTC con microsegundos. Las fechas civiles se guardan
  como fecha, sin convertirlas en medianoche UTC.
- Cada contrato dice si la fecha de fin está incluida o no.
- Dinero y distancias se guardan con valores exactos y su moneda o unidad,
  nunca en coma flotante.
- Reglas, fuentes y vigencias tienen versión. Una política nueva no recalcula
  en silencio la historia antigua.

## Riesgos del código antiguo que hay que vigilar

En julio de 2026 se encontraron estos problemas en Personal, Cronos y Dietas.
Algunos siguen en el código (por ejemplo `DefaultLeavePolicies` en
`internal/modules/cronos/domain/leave.go`). Antes de ampliar esas piezas,
comprobar si el riesgo sigue:

| Pieza | Riesgo | Qué hacer |
| --- | --- | --- |
| `DefaultLeavePolicies` (Cronos) | Cupos y límites de permisos escritos en Go | Sustituir por políticas publicadas en catálogo |
| `DailyReductionForAge` (Cronos) | Edades y horas de reducción fijas | Usar una política versionada y hechos de Personal |
| `Workday` recibe la edad | Dato derivado sin fuente ni fecha | Usar una referencia de elegibilidad emitida por Personal |
| `EmploymentRegime` (Personal) | Mezcla relación, vínculo y cargo | Usar el modelo separado de [PRD §7](../producto/PRD.md#7-modelo-de-datos-que-no-se-simplifica) |
| `RPTPosition` (Personal) | Mezcla fila, dotación, puesto, categoría y estado | Modelo con historia de dos fechas |
| `CategoryRule.PointsPerMonth` y Dietas | Puntos, importes y kilómetros en `float64` | Valores exactos |
| `PayrollDraft` (Personal) | Solo suma conceptos | Es una demostración; diseñar antes el libro de nómina |
| Permisos `manage` de los manifiestos | No dicen acción, persona, unidad ni finalidad | Partir por caso de uso |

## Historia propia del expediente de Personal

Propuesta de estudio para «Mi ficha», pendiente de validar con RRHH. El
contrato preliminar está en
`deploy/postgresql/personal/borradores/000025_historia_propia_expediente.json`
y no se instala.

- La situación en una fecha y la historia son consultas distintas. Las dos
  fijan la fecha de efectos y la fecha hasta la que se conocían los hechos.
- La historia se pide por apartado y periodo, en páginas limitadas. Cada página
  vuelve a comprobar el permiso y el vínculo entre la persona y su empleado. El
  cursor queda ligado a empleado, perfil, apartado, periodo y corte, y no
  permite cambiar de ficha.
- Una rectificación conocida después de un corte aparece al consultar el corte
  posterior, sin cambiar lo que correspondía al anterior.
- Para mostrar un documento junto a un hecho, Personal guarda el vínculo entre
  el hecho y el expediente, documento y versión. Documentos conserva el
  original y su descarga, con permiso propio.
- La consulta ordinaria excluye causas clínicas, familiares reservadas y
  disciplinarias. Una fuente caída se muestra como no disponible; una parcial,
  como incompleta y con su corte.

## Procedencia

El `Peticion.pdf` y el procedimiento en Word que remitió RRHH describen Bolsa,
Contratación, documentos e incorporación. No traen especificación de Dietas ni
de Cronos: los campos propuestos para esos módulos no se atribuyen a RRHH.
