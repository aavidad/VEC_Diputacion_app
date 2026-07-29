# O4-05: revisión final de CT-000043

Fecha: 29 de julio de 2026.

## Resultado

CT-000043, prueba durable del resultado y Recibo RRHH V2, obtuvo dos
dictámenes independientes:

| Revisión | P0 | P1 | P2 | Dictamen |
| --- | ---: | ---: | ---: | --- |
| Integridad, arquitectura y retirada segura | 0 | 0 | 0 | `GO` |
| PostgreSQL, concurrencia y pruebas globales | 0 | 0 | 0 | `GO` |

El productor no revisó, confirmó ni publicó su propio trabajo. Dirección
confirmó el corte como `4d38e9c` y lo integró en la rama estable como
`a1a09b9`.

Este cierre incrementa el procedimiento de Contratación temporal de `20/46`
a `21/46`. O4-05 permanece en `3/5`: CT-000043 no sustituye al motor
CT-000044, las fachadas CT-000045, el adaptador Go, la composición raíz ni el
E2E.

## Alcance revisado

Se revisaron los once artefactos completos:

```text
deploy/postgresql/contratacion_temporal/
  migraciones/000043_prueba_resultado_recibo_rrhh.up.sql
  migraciones/000043_prueba_resultado_recibo_rrhh.down.sql
  migraciones/000043_componentes/010_tipos_cierre.sql
  migraciones/000043_componentes/020_relaciones_y_prueba.sql
  migraciones/000043_componentes/030_primitiva_cierre.sql
  migraciones/000043_componentes/085_guardia_columnas_padre.sql
  migraciones/000043_componentes/090_acl_catalogo_y_barrera.sql
  migraciones/000043_componentes/095_avance_barreras.sql
  pruebas_sql/o405_prueba_resultado_recibo_rrhh.sql
  pruebas_sql/o405_prueba_resultado_recibo_rrhh_datos_sinteticos.sql
  probar_o4_05_prueba_resultado_recibo_rrhh_pg18_4.sh
```

Todos permanecen por debajo del límite de 800 líneas; el máximo es 793.

## Evidencia PostgreSQL

La imagen fijada por resumen fue:

```text
postgres:18.4-bookworm@sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296
```

El ejecutor:

```bash
./deploy/postgresql/contratacion_temporal/\
probar_o4_05_prueba_resultado_recibo_rrhh_pg18_4.sh
```

terminó correctamente en dos ciclos consecutivos del productor y en las dos
reproducciones independientes. Verificó:

- dependencias y regresiones CT-000039 a CT-000042 y AUT-000006;
- omisión y alteración de cada componente;
- instalación, reentrada y ciclo `UP → DOWN → UP`;
- barreras exactas `22/6 → 23/7`;
- cuadro, detalle y replay;
- rollback sin consumo, acceso ni prueba residual;
- propagación de `40001`, `40P01`, `55P03` y `57014`;
- dos cierres concurrentes con un único ganador;
- ocho valores autoritativos y diez piezas VEC;
- contenido, resultado, conjunto material y Recibo V2 de 38 campos;
- claves foráneas compuestas y cruces adversariales;
- RLS forzada, ACL mínima e inmutabilidad;
- revocación viva y denegación cerrada;
- evidencia durable que bloquea el `down`.

La retirada segura detecta cambios o dependencias futuras en:

- ACL y propiedades de las columnas añadidas al registro de acceso;
- índices simples o de expresión;
- restricciones;
- estadísticas extendidas;
- publicaciones parciales o completas;
- herencia de tablas;
- comentarios, triggers, políticas, reglas y dependencias.

La huella semántica literal acreditada en PostgreSQL 18.4 es:

```text
e8a4cbadc41fb73d4381dff9b8aa20a19093ce53a97058af39312957906473a3
```

## Puertas de calidad

Quedaron verdes:

```bash
bash -n \
  deploy/postgresql/contratacion_temporal/\
probar_o4_05_prueba_resultado_recibo_rrhh_pg18_4.sh

shellcheck -x -e SC2154 \
  deploy/postgresql/contratacion_temporal/\
probar_o4_05_prueba_resultado_recibo_rrhh_pg18_4.sh

scripts/comprobar_tamano_ficheros.sh

TMPDIR="$HOME/.cache" go test ./... -count=1 -timeout 20m
TMPDIR="$HOME/.cache" go test -race ./... -count=1 -timeout 30m
TMPDIR="$HOME/.cache" go vet ./...
```

También quedaron verdes la compilación, grafos y manifiestos, TLS,
`govulncheck`, diferencias, espacios y Gitleaks focal y preparado. No se
usaron datos personales, credenciales ni rutas privadas.

La suite global con el `TMPDIR` predeterminado puede heredar el marcador
externo `/tmp/.git`; no debe borrarse. Las pruebas Go se ejecutaron con
`TMPDIR="$HOME/.cache"`. Una comprobación PDF necesitó un directorio
temporal escribible para `qpdf`; repetida en ese entorno terminó
correctamente.

## Decisiones y correcciones relevantes

- La prueba VEC-AD-3 se serializa con las revocaciones mediante el checkpoint
  de gobierno; no se acepta un falso positivo por timeout.
- Los metadatos de columnas se sellan por nombre y no por `attnum`, porque
  este cambia tras `UP → DOWN → UP`.
- Los triggers internos de claves foráneas se normalizan por restricción y
  definición, no por nombres con OID.
- La guardia del padre evita que `DROP COLUMN` retire silenciosamente
  dependencias futuras.
- El contenido se persiste tipado; las huellas autoritativas se recalculan
  dentro del cierre.
- La auditoría VEC se trata como referencia opaca y se compara con la
  revalidación viva.
- Los comentarios y nombres nuevos permanecen en castellano; sólo se
  conservan palabras reservadas y nombres técnicos normalizados.

## Riesgo residual y continuación

Producción permanece en `NO-GO`. El siguiente corte es CT-000044, motor
privado y atómico de cuadro/detalle. Después siguen CT-000045, adaptador
Go/PostgreSQL, composición raíz, identidad/PDP, TLS viva y E2E desde web y un
transporte no web.
