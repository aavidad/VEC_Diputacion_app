# Integración de Cronos y Dietas en el Portal del Empleado

Fecha de corte: 19 de julio de 2026.

## Resultado

El Portal del Empleado conserva el mismo armazón visual, menú lateral, tema,
cabecera, accesibilidad y vistas de Gestión de Bolsas. Cronos y Dietas se han
incorporado como módulos intercambiables mediante composición, sin copiar el
shell ni crear identidades o sesiones propias.

Rutas estables:

- `#portal`: catálogo completo del portal;
- `#bolsa/resumen` y el resto de vistas `#bolsa/*`: Gestión de Bolsas;
- `#cronos`: autoservicio de jornada, fichajes, permisos y recibos;
- `#dietas`: autoservicio de comisiones, rutas, gastos y recibos.

En el inicio se mantiene visible el catálogo completo. Dentro de un módulo, el
lateral muestra únicamente Inicio y el módulo activo; el submenú extenso de
Bolsa solo aparece en las rutas de Bolsa. Esto evita que las opciones de gestión
queden bajo el pliegue y que Bolsa parezca activa al consultar Cronos o Dietas.

## Decisiones reutilizables en producción

- `portal-modulos-coordinador.js` es la raíz de composición del frontend. El
  router y las vistas no contienen reglas de negocio de Cronos ni Dietas.
- El catálogo productivo procede de los manifiestos registrados por el núcleo
  mediante `/api/vec/modules`; la plantilla HTML no enumera módulos.
- Bolsa, Cronos y Dietas reciben por referencia el mismo `ContextoActor`
  validado e inmutable. Las capacidades se entregan separadas y se aplican con
  denegación por defecto.
- Las capacidades de la demostración son exclusivamente de autoservicio propio.
  No se conceden capacidades de jefatura, aprobación de terceros ni auditoría
  general por inferencia de rol.
- Los estilos de módulo heredan las variables del tema común. La portada añade
  color semántico estable: Bolsa azul/índigo, Cronos cian/violeta y Dietas
  verde/naranja. Los módulos inactivos permanecen sobrios.
- Al abandonar un módulo se retiran sus manejadores. Un montaje asíncrono
  obsoleto no puede sustituir la vista más reciente.
- Los recibos usan el puerto documental común. El PDF de presentación incluye
  identidad visual institucional, texto equivalente al recibo real, marca DEMO,
  referencia opaca y QR de cotejo sin datos personales.

## Límite de la demostración

Son sustituibles y deben excluirse del despliegue productivo los adaptadores o
fixtures cuyo nombre contiene `presentacion`. Mantienen datos sintéticos en
memoria volátil y no registran, firman, notifican, pagan ni persisten actos.

Las vistas, contratos, presentadores, catálogos i18n, estilos y coordinador se
conservan. Al avanzar a producción se conectarán adaptadores autenticados a los
puertos existentes; no será necesario rehacer la web.

## Evidencia de verificación

- 136 pruebas del Portal del Empleado superadas.
- Navegación real comprobada en Chromium a 1440, 1024 y 390 píxeles, sin errores
  de consola, vistas en blanco ni desbordamiento horizontal de la página.
- Comprobadas las transiciones mediante botones desde la portada hacia Bolsa,
  Cronos y Dietas.
- Descarga real de un recibo Cronos: PDF 1.4, una página A4, sin JavaScript,
  con texto institucional, referencia y bloque de comprobación QR.

Las capturas y el PDF de revisión se generan fuera del repositorio para evitar
incorporar artefactos temporales o información del entorno de desarrollo.
