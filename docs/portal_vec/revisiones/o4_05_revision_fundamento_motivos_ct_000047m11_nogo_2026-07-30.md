# Revisión del fundamento de motivos CT-000047M1.1

Fecha: 30 de julio de 2026.

## Resultado

**NO-GO técnico independiente**.

| Elemento | Valor |
| --- | --- |
| Base del candidato | `503ee22` |
| Commit candidato no integrado | `99c1e1b` |
| P0 | 0 |
| P1 | 1 |
| P2 | 0 |

El candidato permanece fuera de la rama integradora.

## Hallazgo P1 reproducido

Después de una instalación limpia en PostgreSQL 18.4, el revisor:

1. sustituyó el trigger de inmutabilidad por un homónimo `BEFORE INSERT`;
2. sustituyó la política RLS por una homónima `FOR SELECT TO PUBLIC
   USING (true)`;
3. eliminó la clave foránea hacia la entrada de motivo;
4. volvió a ejecutar `000008`.

La migración confirmó la transacción y conservó los tres objetos degradados.
La reentrada solo reconocía nombres, tabla, propietario y comentarios
parciales; no cotejaba la definición estructural exacta.

El `down` eliminaba además una restricción `UNIQUE` añadida a una tabla previa
aunque el `up` pudiera haber adoptado un objeto preexistente sin marca de
procedencia.

## Corrección exigida

- validar columnas, tipos, nulos, valores predeterminados, restricciones y
  claves foráneas exactas;
- validar evento, orientación, función y habilitación de cada trigger;
- validar comando, roles, `USING` y `WITH CHECK` exactos de cada política;
- marcar la restricción añadida con procedencia propia, rechazar homónimos sin
  marca y retirarla solo si conserva definición y marca;
- incorporar los estados envenenados a la matriz automatizada;
- repetir instalación, reentrada, reversión y PostgreSQL 18.4 real;
- obtener una nueva revisión independiente.

## Garantías que sí resultaron verdes

En una instalación no adulterada superaron 2/2 ejecuciones PostgreSQL, DML
hostil, checkpoint, RLS forzada, ACL, tipos, roles `NOLOGIN`, `search_path`,
`down` limpio y bloqueo del `down` con evidencia. Bash, ShellCheck, tamaños,
diff y Gitleaks también fueron verdes. Estas pruebas no compensan el P1.
