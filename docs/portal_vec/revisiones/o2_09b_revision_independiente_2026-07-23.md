# Revisión independiente O2-09B — límites del alta web

Fecha: 23 de julio de 2026.

## Alcance y separación

- Candidato revisado: `228df6f1c79d542a2e698eb4e9c1d42d243c7ca9`.
- Base declarada y comprobada: `1323b4b382b28de21d2d8036346f13657e755242`.
- Productor: agente de corrección O2-09B.
- Revisor independiente: agente distinto del productor.
- Integración posterior: `764fd52`.

La revisión cubre exclusivamente la corrección de los dos límites que habían
motivado el NO-GO de O2-09A. No acredita todavía la composición real, el
registro de la ruta ni el recorrido E2E.

## Veredicto

**GO**, sin hallazgos bloqueantes.

## Evidencia reproducida

- Cien años civiles exactos se aceptan.
- Cien años y un día se rechazan antes de invocar el ejecutor.
- `922337203685477` céntimos se aceptan con igualdad exacta.
- El máximo más un céntimo y un valor superior a
  `Number.MAX_SAFE_INTEGER` se rechazan antes del ejecutor.
- La comparación de importes usa cadenas de dígitos antes de convertir a
  `Number`; no multiplica importes mediante coma flotante.
- Las mutaciones equivalentes del comando cerrado se rechazan también en la
  segunda barrera de `validarComandoAlta`.
- Suite focal repetida cincuenta veces: correcta.
- Portal completo: 260 de 260 pruebas correctas.
- `node --check` sobre todos los JavaScript del módulo: correcto.
- Claves nuevas de ayuda y error presentes y resueltas mediante i18n.
- Sin `fetch`, cookies, almacenamiento de navegador, credenciales,
  autoridades aportadas por cabeceras libres, URL fija ni adaptador DEMO en
  los JavaScript productivos del corte.
- `git diff --check`: correcto.
- Gitleaks: un commit analizado, sin filtraciones.
- Puerta de tamaño: correcta. El fichero mayor es la prueba focal con 788
  líneas, por debajo del límite duro de 800.

## Riesgo no bloqueante

La prueba focal está próxima al límite de tamaño. Debe dividirse si una
ampliación posterior la hace superar las 800 líneas; no se reduce cobertura
para conservar el límite.

## Condición de cierre de O2-09

Este GO permite integrar la interfaz y sus contratos de cliente. O2-09
permanece en curso hasta que O2-07 registre la composición real, O2-08 quede
alcanzable por la ruta interna y O2-10 demuestre el recorrido completo sin
adaptadores de demostración.
