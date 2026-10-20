# F1-129: ficha de Jefatura sin cargas de tramitación

La ficha de los cargos firmantes conserva el cuadro, el resumen de Inicio,
el detalle, el circuito y el panel de firma. Al abrirla no consulta el análisis,
la cancelación, los borradores publicados, la lista común de Documentos ni
el contexto de Bolsa. RRHH conserva esas funciones.

La selección de pantalla usa el único perfil atestado de la sesión. No concede
acceso a ninguna operación: el servidor mantiene sus comprobaciones.

## Activación

Instalar la aplicación con sus recursos web de la misma revisión y reiniciarla.
Los recursos usan la marca común `VEC_WEB_VERSION`; no hay versiones manuales
ni configuración nueva para este ajuste.

Para comprobarlo en una copia propia:

1. Crear la copia con `--modulos`, el material original autorizado de RRHH y
   la lista completa de main, siguiendo `LEEME_copia_local.md`.
2. Preparar el montaje nominal de cargos con las herramientas existentes de
   `candidata_hz39/firma/` y la receta genérica cedida para esta tarea. Usar dos
   administradores y los actos, asignaciones y recibos de esa copia. Conservar
   los certificados heredados; no trasladar planes ni recibos de otra copia.
3. Componer la firma con el manifiesto privado de la instancia, según el apartado
   «Firma compuesta» del LEEME:

   ```sh
   python3 deploy/principal/copia_local.py firma --equipo b --instancia f129 \
     --manifiesto-firma "$VEC_F129_MANIFIESTO_FIRMA"
   ```

La consulta histórica de firmas es una función distinta y puede permanecer
apagada. La copia debe arrancar también antes de componer la firma; el perfil
de RRHH conserva su recorrido habitual.

## Comprobación

Abrir una ficha real con el certificado de Jefatura. La sesión debe devolver
un único rol `ct_cargo_jefatura_firma_dev`; una etiqueta visible no lo acredita.
Comprobar en Chrome, en escritorio y móvil, que cuadro, detalle y circuito
responden correctamente y que no se emiten estas cinco peticiones:

| Método | Ruta bajo `/api/vec/` |
| --- | --- |
| GET | `contratacion-temporal/configuracion-analisis` |
| POST | `contratacion-temporal/cancelacion-expediente` |
| POST | `contratacion-temporal/expedientes/borradores/disponibles` |
| POST | `documentos/expedientes/consultas` |
| GET | `bolsa/bolsas` como contexto de la ficha |

La ficha no ofrece las acciones de preparación ni cancelación. Las descargas
del panel de firma siguen por su cliente propio. La tarjeta Documentos no
queda vacía: se muestra al llegar una descarga firmada y custodiada válida.
Comprobar además el montaje
de RRHH, el resumen de Inicio, el teclado y la ausencia de desbordamiento.
Tras reiniciar aplicación y PostgreSQL, repetir las lecturas de recuperación.

Preferencias e imagen propias permanecen en el portal. Su vínculo de identidad
y sus permisos en la copia corresponden a F1-130; su respuesta no se oculta ni
se presenta como resuelta en este corte.

## Contraste del perfil técnico

En la copia de aceptación, documentar el control con esta descripción:
«control sin la configuración nominal R5; con R5 el certificado técnico
resuelve a ct_cargo_tecnico_firma_dev».

Para ese contraste, conservar una copia privada exacta de la configuración,
parar la instancia por la fábrica y apagar solo `VEC_CT_FIRMA_VEC_R5_ENABLED`.
Registrar su huella real y arrancar por el procedimiento oficial. La sesión
observada debe devolver `tecnico_rrhh`; comprobar sus consultas de tramitación.
Después parar, restaurar los bytes originales y su huella real, y volver a
arrancar. Confirmar el rol único de Jefatura y que binario, plan y recibos
siguen iguales. No crear otro certificado ni repetir actos o publicaciones.

El zoom al 200 % se comprueba con el ajuste nativo de Chrome, sin `zoom` CSS.
Registrar ancho de ventana, `innerWidth` y `devicePixelRatio`. Abrir solo el
bloque desplegable de firma para comprobar su contenido; no ejecutar firmas
ni operaciones de negocio al verificar estas cargas de lectura.
