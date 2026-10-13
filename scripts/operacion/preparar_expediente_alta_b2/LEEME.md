# Preparación del expediente para Personal

**Estado: pendiente del contrato de necesidad en la entrega ratificada.** La fuente de selección B110 ya lee una Bolsa constituida. La ruta actual de entrega de la petición centro→RRHH no transmite la necesidad versionada con el puesto RPT; sin ese dato, la selección nativa fallaría después de crear el expediente. No ejecute `--aplicar` para un expediente nuevo hasta integrar y comprobar ese contrato.

`preparar.sh --comprobar` consulta certificados, configuración M10, B110/AD261, una Bolsa Administrativo vigente única, catálogos y RPT. No crea estado ni envía actos de negocio; las consultas HTTP nominales pueden dejar su auditoría habitual. `--aplicar PREPARA-ALTA-B2` conserva claves y cuerpos antes de cada POST.

La RPT debe contener un puesto y una plaza compatibles publicados en el centro 520. El guion no amplía una versión publicada ni copia filas de otra versión. Si falta el par, informa `FALTA`.

El SQL de remisión conserva dos precondiciones ficticias, expresamente autorizadas para datos inventados, bajo el rol propietario de CT. No verifica firma electrónica. El llamamiento B7 se enlaza al expediente por CT201 con su recibo real; la respuesta firme B30 la registra el candidato desde Mi Bolsa. El guion no crea bolsas, participaciones ni anclajes B67/B68.

El estado y los cuerpos de reintento se guardan fuera de Git con permisos 0600. La hora del correo se calcula una vez antes de enviarlo. El número visible procede de la política de alta y de las cuatro tablas de números y reservas. No borre el estado para reintentar.

Valores por defecto: material de cidonia y contenedor `vec-postgresql-20260906`. Puede indicar `--origen`, `--material`, `--launcher`, `--config`, `--estado`, `--motor` y `--contenedor`. Use el `servidor.json` del lanzador activo si conserva varios directorios Personal. La comprobación muestra comandos manuales para lo que falte; no modifica ni reinicia servicios. Dirección coordina la prueba API/Chrome: este guion aún no acredita el recorrido completo.
