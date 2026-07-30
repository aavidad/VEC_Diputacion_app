# Revisión final del fundamento de motivos CT-000047M1.1

Fecha: 30 de julio de 2026.

## Resultado

**GO técnico independiente**.

| Elemento | Valor |
| --- | --- |
| Base de desarrollo | `503ee22` |
| Candidato final | `4ef2745` |
| Commit integrado final | `047e52c` |
| P0 | 0 |
| P1 | 0 |
| P2 | 0 |

La integración conserva cuatro commits pequeños: fundamento inicial,
equivalencia estructural, reentrada endurecida y retirada segura. Los
NO-GO intermedios no se ocultan; sirvieron para ampliar la matriz de regresión.

## Garantías verificadas

La migración `000008` instala el fundamento privado de las vinculaciones
nominales de motivos para cuadro y detalle. No publica ni resuelve motivos.

La instalación y la reentrada exigen:

- tablas ordinarias permanentes `heap`, sin partición, herencia ni reglas;
- columnas, tipos, nulos, valores predeterminados y colaciones exactos;
- ACL de tabla y columna sin beneficiarios distintos del propietario;
- restricciones, claves foráneas e índices semánticos exactos;
- cinco disparadores propios con definición y metadatos completos;
- doce disparadores internos de integridad referencial habilitados y
  canónicos;
- políticas RLS exactas y forzadas;
- procedencia inequívoca de la restricción añadida a la tabla anterior.

La retirada bloquea cualquier borrado si existe evidencia. Antes de su primer
`DROP`, bajo bloqueo exclusivo, comprueba por separado identidad, propietario,
tipo y marca de las dos tablas y las dos funciones, además de la estructura y
procedencia de la restricción anterior. No usa `CASCADE`.

## Matriz adversarial

PostgreSQL 18.4 acreditó fallo cerrado y conservación del estado hostil para:

- tablas `UNLOGGED`;
- disparadores homónimos, con evento alterado o `WHEN(false)`;
- ACL ajena de tabla y ACL ajena de columna en casos separados;
- regla DML no nominal;
- colación no determinista;
- índice `UNIQUE` adicional;
- disparadores internos de FK deshabilitados en ambos extremos;
- política RLS homónima degradada;
- clave foránea ausente;
- restricción anterior sin marca;
- cada tabla y cada función sin marca, en cuatro casos independientes.

La retirada normal, la reinstalación y el bloqueo por evidencia permanecen
verdes.

## Evidencia reproducida

Productor, revisor y dirección ejecutaron el arnés completo en PostgreSQL
18.4. La última corrección obtuvo 3/3 ejecuciones del productor, 2/2 del
revisor y una ejecución adicional de dirección. Bash, ShellCheck, revisión del
diff, límites de tamaño, `go vet` y Gitleaks del rango terminaron en verde.

La puerta global local conserva siete fallos anteriores en
`internal/app/bootstrap`: al ejecutarse dentro de `.worktrees`, clasifica
rutas temporales externas como parte del repositorio. Dirección reprodujo los
mismos fallos en la rama estable anterior a `000008`; la migración no modifica
código Go. El CI de la rama estable anterior estaba completamente verde.

## Límites conservados

Este cierre no implementa `000009`, `000010`, el adaptador Go M2 ni la
composición raíz. Tampoco sustituye las decisiones de Sistemas y DBA sobre
KMS/HSM, catálogo aprobado, TLS o autoridad productiva.
