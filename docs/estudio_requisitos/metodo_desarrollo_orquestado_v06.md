# Método de desarrollo asistido con Orquesta V06

Estado: **criterio de trabajo basado en la capacidad actual observada; deberá actualizarse
cuando Orquesta evolucione**.

Fecha de revisión: 14 de julio de 2026.

Fuente revisada íntegramente:
`/home/alberto/Trabajo/orquesta-rebuild/docs/reconstruccion/runbook_agente_director_v06.md`.

## 1. Conclusión

Orquesta V06 puede ayudar a dividir, ordenar, paralelizar y conservar el resultado de
tareas de programación. No es todavía un programador autónomo que modifique, compile,
pruebe e integre el proyecto por sí solo.

Se utilizará como plano de dirección de trabajo durante el desarrollo, nunca como plano de
ejecución de la aplicación ni como sustituto de Kubernetes, integración continua, revisión
de código o pruebas.

## 2. Capacidad real actual

El agente director externo:

1. inspecciona el repositorio y los requisitos;
2. selecciona contexto mínimo y suficiente;
3. divide el objetivo en tareas con dependencias y conjuntos de escritura;
4. crea el objetivo en Orquesta;
5. vigila estados y lee artefactos;
6. revisa críticamente las propuestas;
7. solo si tiene autorización, aplica los cambios al repositorio y ejecuta las pruebas.

Orquesta V06:

- persiste objetivo y grafo de tareas;
- calcula qué tareas están preparadas;
- paraleliza tareas sin escrituras incompatibles;
- invoca trabajadores Codex;
- conserva artefactos y recibos;
- permite consultar, listar y modificar el objetivo dentro de su contrato.

Los trabajadores actuales no disponen por sí mismos de un workspace escribible del
proyecto ni aplican cambios. Su salida es texto, propuesta o parche para revisar.

## 3. Límites que no deben ocultarse

- `Goal succeeded` significa que Orquesta terminó sus tareas, no que el código esté
  aplicado, compilado, probado o integrado.
- Una dependencia ordena tareas, pero no inserta automáticamente el artefacto anterior en
  el contexto de la siguiente.
- Si una segunda fase necesita el resultado exacto de la primera, el director debe leerlo y
  crear un nuevo objetivo con ese contenido.
- La división por conjuntos de escritura reduce colisiones, pero no sustituye la revisión
  semántica.
- No se enviarán repositorios completos, cursos completos ni datos personales como contexto.
- No se iniciará un runtime de Orquesta ni se instalarán conectores sin preflight,
  autorización y entorno dedicado.

## 4. Flujo propuesto para este proyecto

```text
requisito aprobado
      |
      v
director inspecciona código/documentos exactos
      |
      v
objetivo Orquesta: análisis o propuestas paralelas
      |
      v
director lee y contrasta artefactos
      |
      v
revisión independiente cuando el riesgo lo exige
      |
      v
aplicación controlada en rama/worktree
      |
      v
formato + pruebas + seguridad + documentación
      |
      v
revisión humana y promoción por CI/GitOps
```

Tipos de tarea apropiados para V06:

- análisis de contratos y brechas por módulo;
- propuesta de puertos/adaptadores con escrituras separadas;
- casos de prueba y matrices de autorización;
- revisión de documentación y consistencia;
- preparación de parches independientes;
- comparación de opciones técnicas;
- auditoría cruzada de un artefacto antes de aplicarlo.

No se delegarán sin control:

- decisiones jurídicas o de categorización ENS;
- aceptación de riesgos;
- cambios de producción;
- secretos, certificados o accesos;
- migraciones destructivas;
- decisiones automatizadas sobre personas;
- integración de un parche que no ha sido probado.

## 5. Cómo redactar un objetivo

Cada objetivo incluirá:

- resultado concreto y verificable;
- ficheros o símbolos estrictamente necesarios;
- fuentes de verdad y decisiones aprobadas;
- lo que queda fuera;
- conjunto de escritura permitido;
- dependencias reales;
- pruebas y documentación esperadas;
- prohibiciones de seguridad y privacidad;
- formato del artefacto de salida.

Las tareas paralelas no escribirán los mismos ficheros. La tarea de integración dependerá de
las anteriores y será responsabilidad del director, no una suma ciega de parches.

## 6. Documentación continua

Cada cambio funcional deberá actualizar en la misma entrega:

- código comentado donde aporte intención y no obviedad;
- contrato de API/evento;
- diccionario o migración de datos;
- pruebas;
- manual de usuario/ayuda afectado;
- manual técnico u operativo;
- decisión arquitectónica si cambia una frontera;
- versión, compatibilidad y procedimiento de reversión.

Orquesta podrá crear tareas distintas para código, pruebas y documentación, pero la entrega
no se cerrará hasta que el director compruebe su coherencia conjunta.

## 7. Evolución esperada

El propio runbook sitúa en versiones posteriores capacidades como workspace/Git, revisión
independiente y dirección más completa. Cuando estén implantadas y verificadas se revisará
este método. No se redactarán especificaciones suponiendo que ya existen.

## 8. Requisitos verificables

- **ORQ-001:** cada objetivo tiene alcance, contexto, escrituras y aceptación explícitos.
- **ORQ-002:** tareas paralelas no tienen conjuntos de escritura solapados.
- **ORQ-003:** ningún artefacto se considera integrado antes de aplicarlo y probarlo.
- **ORQ-004:** una fase que consume un artefacto previo recibe su contenido verificado en
  un nuevo contexto.
- **ORQ-005:** no se incluyen datos personales, secretos ni repositorios completos sin
  necesidad y autorización.
- **ORQ-006:** los cambios se aplican en rama/worktree, pasan CI y requieren revisión.
- **ORQ-007:** código, pruebas, contratos y manuales evolucionan en la misma entrega.
- **ORQ-008:** la promoción a producción se hace por la cadena CI/GitOps aprobada, nunca por
  un trabajador de Orquesta.
