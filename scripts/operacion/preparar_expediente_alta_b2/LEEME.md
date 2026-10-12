# Preparación del expediente para Personal

**Estado: bloqueado antes del primer POST.** La selección de CT aún no consume una Bolsa constituida. El guion informa `FALTA` y no crea el expediente. Constituir la bolsa sintética que genera CT la añadiría a las listas y recuentos de RRHH; dirección ha descartado esa alternativa para el recorrido.

`preparar.sh --comprobar` consulta certificados, configuración, catálogos y RPT. No crea estado ni envía actos de negocio. `--aplicar PREPARA-ALTA-B2` también comprueba esas dependencias y se detiene mientras falte la fuente de Bolsa. No añada flags para saltar esta comprobación.

La RPT debe contener un puesto y una plaza compatibles publicados en el centro 520. El guion no amplía una versión publicada ni copia filas de otra versión. Si falta el par, informa `FALTA`.

El SQL de remisión conserva dos precondiciones ficticias, expresamente autorizadas para datos inventados, bajo el rol propietario de CT. No verifica firma electrónica. El anclaje sólo acepta una participación y situación de una Bolsa ya constituida y llama a `registrar_vinculos_candidato_v1`; no crea bolsas, instantáneas ni situaciones, ni desactiva controles.

El estado y los cuerpos de reintento se guardan fuera de Git con permisos 0600. La hora del correo se calcula una vez antes de enviarlo. El número visible procede de la política de alta y de las cuatro tablas de números y reservas. No borre el estado para reintentar.

Valores por defecto: material de cidonia y contenedor `vec-postgresql-20260906`. Puede indicar `--origen`, `--material`, `--launcher`, `--config`, `--estado`, `--motor`, `--contenedor` y `--candidato-ref`. Use el `servidor.json` del lanzador activo si conserva varios directorios Personal. La comprobación muestra comandos manuales para lo que falte; no modifica ni reinicia servicios.
