# Tercera revisión independiente O8-01

Fecha: 23 de julio de 2026.

## Dictamen

**NO-GO** para el candidato exacto
`3fccc15eb7f59dda4408a989d6bd4f00dd9c7918`.

El modelo corrige los bloqueos funcionales anteriores, pero todavía procesa
cardinalidades no confiables antes de aplicar límites y repite una
normalización costosa por cada actuación histórica. No se integra.

El commit posterior `06065c4` no forma parte de este dictamen. Su cambio parece
atender solo el primer hallazgo y deberá incorporarse a un nuevo candidato
completo antes de otra revisión.

## Hallazgos

### O8-01-H1 — reserva del índice antes de limitar periodos rehidratados

Severidad: alta.

`RehidratarSeguimiento` limita las actuaciones, pero reserva después un mapa
con la capacidad de `PeriodosResultantes` sin validar antes la cardinalidad de
esa colección. Un estado persistido con raíz válida y un número arbitrario de
periodos puede forzar una reserva masiva antes del rechazo semántico.

La rehidratación es una frontera de datos persistidos y debe imponer el máximo
antes de `make`, copia, indexación u otra asignación proporcional.

### O8-01-H2 — colecciones anidadas copiadas y ordenadas antes de limitarlas

Severidad: alta.

`normalizarDefinicionSeguimiento` limita las colecciones superiores, pero
clona inmediatamente las transiciones. La clonación copia motivos, documentos
y listas del calendario completas. Los máximos anidados se comprueban después
y las listas del calendario llegan a ordenarse antes del límite.

Una sola transición válida en la colección superior puede contener una lista
arbitrariamente grande, duplicar memoria y consumir CPU antes de fallar.
Publicación y restauración deben validar todas las cardinalidades anidadas
antes de copiar u ordenar.

### O8-01-M1 — rehidratación de coste multiplicativo

Severidad: media.

Cada actuación rehidratada termina consultando la vigencia. Esa consulta vuelve
a validar, publicar, restaurar, clonar, ordenar y recalcular la definición
completa. El coste es proporcional a actuaciones por definición, no lineal
razonable respecto de la entrada total como afirma la documentación.

La definición debe normalizarse y validarse una vez por reconstrucción, y las
transiciones posteriores deben usar esa representación validada e indexada sin
repetir el proceso.

## Evidencia favorable

- La reapertura elimina solo el cese vigente y permite uno nuevo sin borrar la
  historia.
- Los ciclos con documentos únicamente opcionales se consideran silenciosos.
- CAS, idempotencia, copias defensivas, canon e integridad semántica son
  coherentes en los casos probados.
- Hexagonalidad, neutralidad de canal, i18n y castellano: correctos.
- Dominio ×50, carrera, global y `go vet`: verdes sobre el SHA revisado.
- `git diff --check`: correcto.
- Gitleaks en `d2d2b75..3fccc15`: tres commits, cero fugas.
- Todos los archivos permanecen por debajo de 800 líneas.

## Corrección exigida

1. Aplicar los límites superiores y anidados antes de toda reserva, copia u
   ordenación proporcional.
2. Añadir pruebas adversariales de rehidratación, publicación y restauración
   con cardinalidades superiores al máximo.
3. Normalizar la definición una vez y reproducir actuaciones sobre la
   representación validada, con una prueba de coste o contador que impida
   revalidar por actuación.
4. Repetir puertas focales, carrera, globales, tamaños, diff y secretos.

El productor no puede autoaprobar la siguiente corrección.
