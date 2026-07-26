# Revisión visual de la web RRHH — NO-GO

Fecha: 24 de julio de 2026.

## Candidato

- Rama: `origin/agent/ct-web-presentacion-rrhh`.
- SHA: `cdf9314df63b67b6b7b83eb6c36cc5f96be47641`.
- Productor distinto del revisor.

## Evidencia favorable

- `272/272` pruebas JavaScript.
- Siete paquetes Go del módulo verdes.
- Manifiestos, tamaños y revisión del diff verdes.
- Datos sintéticos y adaptador de presentación fuera de los manifiestos
  interno y productivo.
- Contratos, presentador, vista, i18n y tema común separados.

## Veredicto

`NO-GO` visual y de paridad con el documento de RRHH.

El productor generó un cuadro y diecisiete capturas de tareas, pero las
diecisiete pantallas numeradas del documento de RRHH no son la misma
enumeración. La referencia exige:

1. cuadro de mando;
2. nueva petición;
3. análisis RRHH;
4. gestión de bolsa;
5. bandeja de unidad;
6. informe jurídico;
7. firma y envío;
8. fiscalización;
9. subsanación;
10. llamamiento;
11. selección;
12. resultado;
13. traslado;
14. documentación para formalización;
15. datos para GINPIX;
16. resumen y envío a GINPIX;
17. generación documental para formalización.

Además, las imágenes de referencia son vistas de `1536 × 1024`. Las evidencias
del candidato son capturas de página completa de `1440` píxeles de ancho y
entre `1270` y `2317` píxeles de alto. Esa técnica introduce artefactos
visibles de elementos fijos:

- gran espacio vacío superior;
- cabecera y barra lateral desplazadas respecto del contenido;
- enlace «Saltar al contenido principal» visible en posiciones impropias;
- pantallas demasiado largas para compararlas uno a uno con la referencia.

Por tanto, ausencia de error de consola y de desbordamiento horizontal no
acredita todavía el aspecto solicitado.

## Corrección y nueva evidencia exigidas

1. matriz contractual exacta 1..17 con el orden del documento;
2. captura principal de cada pantalla a `1536 × 1024`, sin página completa,
   asentada al inicio y sin foco accidental en el enlace de salto;
3. pasadas adicionales a 1440 y 1280 para compatibilidad de escritorio;
4. comparación de jerarquía, densidad, navegación, campos, acciones, estados,
   tipografía, color y elementos fijos;
5. corrección de huecos y desplazamientos antes de una nueva revisión
   independiente.

El candidato queda publicado y reutilizable, pero no debe integrarse ni
presentarse como paridad aprobada hasta superar esta puerta.
