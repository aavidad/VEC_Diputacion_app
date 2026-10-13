# Versión provisional de inscripción con identidad certificada

Esta carga prepara la versión 2 de la convocatoria abierta de Administrativo/a. Su único requisito obligatorio es identificarse con certificado digital o DNIe admitido. La titulación de ESO permanece en la versión 1 y en las solicitudes presentadas bajo ella; la versión 2 no la evalúa como requisito de acceso. Valorar méritos y acreditar requisitos son decisiones distintas.

La fuente `fuente_v2.json` y las tres preimágenes públicas se obtuvieron de la copia local autorizada. `politica_v1_canonica.json` y `convocatoria_v1_canonica.json` conservan los bytes canónicos existentes; `preimagenes_politica_v1.json` conserva los ocho controles CC1 previos. No incluyen identidades ni documentos personales. El generador comprueba esas huellas antes de emitir el SQL.

Para reproducir los artefactos desde esta carpeta:

```bash
python3 generar.py --salida preparado
```

El SQL añade una publicación CC1 versión 2 y una versión de convocatoria BC1 número 2 en una transacción serializable. Reutiliza `categorias_rpt` versión 1 y conserva las entradas anteriores de la política. Añade el ámbito de gestión v2 y cambia el alcance de asociación manual al identificador público v2. Comprueba los bytes de v1 y las preimágenes de control antes de escribir; una diferencia aborta toda la transacción. No inserta solicitudes, personas ni actas, ni actualiza o revierte versiones anteriores.

Las referencias de revisión y los metadatos identifican una preparación operativa. La carga no acredita aprobación de bases, firma legal ni valoración de méritos. Dirección ejecuta el archivo solo en la base PG18 autorizada y coteja `preparado/manifiesto.json` con las lecturas BC9/CC11 y las solicitudes v1 conservadas.
