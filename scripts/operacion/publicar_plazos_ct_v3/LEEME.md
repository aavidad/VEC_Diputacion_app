# Publicación de los plazos de contratación temporal

Tarea: F1-54. Fuente: `data/catalogos/contratacion-temporal/reglas.v3.json`.
Conserva la publicación v2. No añade migraciones, permisos, rutas ni variables de arranque.
La pantalla que lo consume es **Contratación temporal → Reglas vigentes**; los plazos se consultan también en la ficha y en la bandeja.

La fiscalización tiene diez días hábiles, cinco si es urgente: [RD 424/2017, arts. 10.2 y 10.3](https://www.boe.es/buscar/act.php?id=BOE-A-2017-5192#a1-2).
El mismo artículo prevé cinco días para fiscalización limitada previa; VEC aún no distingue ese régimen de forma independiente.
La subsanación tiene quince días hábiles: [art. 12.4](https://www.boe.es/buscar/act.php?id=BOE-A-2017-5192#a1-4) y [Ley 39/2015, art. 30.2](https://www.boe.es/buscar/act.php?id=BOE-A-2015-10565#a30).
Lo recogen también los arts. 12.2 y 14.2 del [Reglamento de Control Interno de Diputación, BOP de 31/12/2021, páginas 5 y 6](https://bop.dipgra.es/export/sites/bop/publica/boletines-anteriores/.galleries/Boletines-Anteriores/20211231-2021250.pdf).

VEC cuenta la subsanación desde que el servicio recibe el reparo. La aceptación no se registra por separado; la nota de la regla declara este límite. La publicación no modifica instantáneas de expedientes ya tramitados.

## Preparación y activación

Solo en una copia propia de la principal, con PostgreSQL 18 y CT148/190/191 ya instaladas. La conexión libpq y las credenciales se proporcionan por el canal privado habitual. El guion usa la cuenta técnica, el preparador y el publicador CT190 existentes. Su referencia de operación documenta el encargo de dirección; no acredita una aprobación ni firma de RRHH.

1. Detener la admisión de escrituras en la copia; conservarla detenida hasta verificar la activación. Guardar una copia de la base y de la configuración anterior.
2. Comprobar memoria disponible y construir el preparador:

```bash
go build -p 6 -o "$TMPDIR/vec-preparar-plazos-ct-v3" ./cmd/vec-preparar-catalogo-reglas-ct
export VEC_REGLAS_CT_PREPARADOR="$TMPDIR/vec-preparar-plazos-ct-v3"
bash scripts/operacion/publicar_plazos_ct_v3/publicar.sh --comprobar
```

3. La comprobación reutiliza la lectura de la publicación v2 y exige su cabeza exacta y **ningún ajuste registrado**. Si hay ajustes, detener la activación y comunicarlo a dirección: deben tratarse en un corte separado. No ejecutar el rebase de v1 a v2 sobre v3 ni borrar historia para activar este catálogo.
4. Con la comprobación aprobada:

```bash
bash scripts/operacion/publicar_plazos_ct_v3/publicar.sh --aplicar PUBLICA-PLAZOS-CT-V3
```

El publicador conserva la fuente y el canónico con sus huellas, comprueba la secuencia esperada y activa v3 en una transacción. El guion no cambia el catálogo de Bolsa ni el circuito de firma.

5. Con la aplicación detenida, cambiar el valor de `VEC_CT_REGLAS_SOURCE_PATH` a `data/catalogos/contratacion-temporal/reglas.v3.json` (ruta relativa al directorio de arranque, o su equivalente absoluto en el despliegue). Conservar las demás conexiones y opciones existentes, incluido el selector de ajustes si ya estaba activado. Guardar el valor anterior y arrancar vigilando la salud. No hay variables ni selectores nuevos. En Chrome, con certificado de RRHH, abrir «Reglas vigentes», comprobar versión 3, valores 10/5 y 15, unidad hábil y citas. Comprobar que esas dos reglas no ofrecen edición; un plazo organizativo como el de análisis debe conservarla. Revisar a 1440 y 390 px. Reiniciar y recuperar los mismos valores e historial. Solo después reabrir la admisión.
6. Arrancar también la configuración anterior sin esta publicación: v2 debe seguir siendo consumible. No afirmar que v3 está instalada si solo se ha validado el fichero.

Si se pierde la respuesta del publicador, conservar la admisión detenida y consultar la cabeza con las herramientas existentes. No repetir a ciegas: la precomprobación rechaza una cabeza que ya sea v3. La vuelta atrás del ensayo restaura la copia completa anterior; no modifica ni elimina publicaciones instaladas.

## Comprobaciones focales

```bash
go test -p 6 ./internal/vec/reglas -run '^TestPlazosCTPublicadosV3RespetanLaNorma$' -count=1
bash -n scripts/operacion/publicar_plazos_ct_v3/publicar.sh
```

El resultado del arranque, la publicación, el rechazo con ajustes y Chrome se registra en la PR. La prueba del cargador y de la validación no acredita el ensayo completo ni una instalación en cidonia.
