# Cierre O4-04E: confirmación durable de cobertura

**Fecha:** 26 de julio de 2026
**Producto:** `faa5a5f`
**Pruebas:** `5954c29`
**Resultado:** cerrado con doble `GO` independiente

## Capacidad cerrada

O4-04E materializa en PostgreSQL la decisión de vía de cobertura producida por
O4-03. Una única función exterior:

1. revalida contexto, decisión y vínculo VEC;
2. consume una sola vez el lote C1;
3. aplica la versión cercada del expediente;
4. persiste actuación, decisión, auditoría, outbox y terminal;
5. devuelve un recibo durable;
6. permite reconciliar un resultado ambiguo mediante lectura primaria fuerte.

La frontera no reintenta escrituras ante `40001` ni `40P01`. Después de una
pérdida de respuesta se consulta el primario con la identidad semántica
completa; nunca se interpreta la ausencia como éxito.

## Seguridad y trazabilidad

- Rol confirmador dedicado, sin login, administración, propiedad ni
  pertenencias transitivas.
- Credencial nominativa con una sola membresía directa y sin capacidad de
  `SET ROLE`.
- ACL y RLS de lista positiva; `PUBLIC` no recibe ejecución ni uso implícitos.
- Canon estricto de entradas, referencias, versiones, huellas e instantes.
- Vínculo criptográfico recalculado entre decisión, contexto, orden, lote,
  referencias y microsegundos de registro/revalidación.
- Auditoría y outbox encadenados; recibo, terminal e historia inmutables.
- Reversiones protegidas contra historia, capas superiores, ACL residuales,
  propiedades divergentes, cambios de rol y carreras TOCTOU.
- Pruebas de fallo después de cada escritura con rollback completo.

## Evidencia ejecutada

El runner
`deploy/postgresql/contratacion_temporal/probar_o4_04e_pg18_4.sh` terminó con
código cero sobre la imagen fijada `postgres:18.4-bookworm`:

- instalación fresca de roles, Autorización y migraciones CT `000001–000034`;
- ciclo descendente y ascendente completo;
- ACL, RLS, credencial mínima y separación de funciones;
- decisiones concedidas y denegadas, replay y retirada en ambos órdenes;
- C1 con 1 y 512 evidencias;
- inyección de fallos sobre todas las escrituras relevantes;
- carreras por reserva, CAS, evidencia compartida y escritor VEC externo;
- deadlock real `40P01` y abortos SSI `40001`;
- ocho sesiones concurrentes;
- reinicio real de PostgreSQL y replay durable;
- negativa final que impide revertir historia O4-04E.

También terminaron correctamente:

```text
go test -race -count=1 ./internal/modules/contrataciontemporal/...
go test -count=1 ./...
go vet ./...
bash -n deploy/postgresql/contratacion_temporal/*.sh
shellcheck -x deploy/postgresql/contratacion_temporal/*.sh
```

Las revisiones independientes dieron `GO` sin bloqueadores para:

- migraciones descendentes, Autorización, rol e historia;
- concurrencia, snapshots de rollback, huella VEC, deadlines y limpieza.

## Decisiones de prueba

El perdedor de una carrera C1 puede devolver `23505` por unicidad o `40001`
por SSI antes de alcanzar esa restricción. Ambos estados forman parte del
contrato concurrente. La prueba no se limita al código: exige exactamente un
`COMMIT`, ausencia primaria del perdedor, snapshot íntegro, cardinalidades
exactas y avance único de las cadenas.

Los scripts principales permanecen por debajo de 500 líneas. Tres migraciones
atómicas superan ese objetivo orientativo pero quedan por debajo del límite
duro de 800; dividirlas expondría estados intermedios instalables.

## Alcance no incluido

Este cierre no registra rutas HTTP ni convierte la pantalla en productiva.
La composición neutral para web, escritorio, CLI y MCP, el cliente del portal
y el E2E visual pertenecen a O4-05. La siguiente referencia obligatoria es
[el plan O4-05](o4_05_plan_integracion_web_2026-07-26.md).
