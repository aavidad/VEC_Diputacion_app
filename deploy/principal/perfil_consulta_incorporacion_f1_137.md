# Perfil técnico de la consulta de incorporación

F1-137 corrige el montaje de incorporación de Personal. La consulta de detalle
ya usaba un perfil fijo técnico, pero incorporación seguía buscando el perfil
dinámico retirado. El montaje devolvía indisponibilidad antes de leer B2.

El comprobador común usa ahora el perfil técnico que la propia composición
ligó a ese descriptor. Mantiene su identidad de ruta y autorización, y rechaza
un fijo ajeno, el dinámico retirado y un perfil vacío. Las variantes nominales
comparten esa comprobación; no reciben una concesión nueva.

## Activación

Compilar la composición de ensayo que reúne este corte y la receta de copia
completa F1-131. No hay migraciones ni variables nuevas. La receta mantiene la
configuración propia y los actos ya confirmados. Para el ensayo autorizado en
la copia E, se conserva el origen de cada commit, el binario anterior y sus
huellas; no se cambian perfiles ni datos para eludir un rechazo.

## Comprobación

La orden `--recorrido-completo` debe terminar la preparación de Personal y
obtener 200 en `GET /api/vec/contratacion-temporal/incorporacion-personal-b2/categorias-rpt/v1`.
Comprobar también el catálogo de régimen y repetir las lecturas tras reiniciar
aplicación y PostgreSQL. Si la preparación falla, la orden Personal conserva
el material para conciliar y restaura su entorno anterior.

Las focales de código prueban el perfil ligado en asignación nominal V2, B2 y
subconsulta, y el rechazo de perfiles ajenos o retirados. El montaje B2 completo
se acredita con la receta; una focal verde no sustituye ese resultado.
