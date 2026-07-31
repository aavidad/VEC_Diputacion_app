# Revisión final C2.2-A: organización corporativa versionada

Fecha: 31 de julio de 2026.

## Veredicto

**GO técnico independiente: P0=0, P1=0 y P2=0.**

C2.2-A supera su revisión técnica compuesta. Este resultado no autoriza datos
reales, no cierra la composición productiva y no modifica las métricas
funcionales. Producción VEP continúa en `NO-GO`.

## Corte revisado

| Fase | Commit | Resultado |
| --- | --- | --- |
| A0 | `2b59884f9c31abe60e44a561804aa235bd443b97` | Contrato cerrado tras cuatro ciclos de revisión. |
| A1 | `1d13d4bd0a7a06188c749f193ffefe3262cdcf7d` | Migración reversible y runner PostgreSQL 18.4. |
| A2 | `ae232426810674ae1fb08370ca1953cdb03059ed` | Ejecución literal del `down` mediante `pgx`. |
| A3 | `ce8959c35f7ac43e9c03c5e5852488bb5043590d` | Composición directa y documentación operativa. |

El árbol revisado tenía además `4d7e07d`, una corrección aislada del runner de
Bolsa pública. Ese commit no modifica C2.2-A y no se incluyó en su valoración
funcional.

Huellas SHA-256 congeladas:

| Artefacto | Huella |
| --- | --- |
| `000003 up` | `6f6e7682ac13a50ee3c9132ce05074f1bf197918c9523844449f27c0083901f6` |
| `000003 down` | `7383217f25c904a63213822a9fb6c66e6f225d09fb59dae24cd985e89bf19828` |
| runner focal | `64446a727c4cd836a3b86ebe67de53bb25daa2f982a19c0a5271f38e878175c2` |
| prueba `pgx` | `fa7cc4797f6a4e798a14106a3d8910fac66b43b32b28a89590ad231ac7a4fffb` |
| runner principal | `c5499435e7c73f08b53e718b36663c7421ff17eca8d589fe8b373982ee3f0ec9` |
| README del módulo | `2f96859cd6168c912a09da939e2ed526b5fabc91b8e2d9cda0c75b13e82ddbf9` |

Los cuatro artefactos de `000001` y `000002` conservan las huellas fijadas en
A0. Ningún fichero revisado supera el límite de 800 líneas.

## Garantías reproducidas

La revisión independiente acreditó conjuntamente:

- orden de barreras `A SHARED → B EXCLUSIVE` en alta y retirada;
- fotografía inmóvil y reacreditación posterior a los bloqueos;
- historia de solo adición y puntero ligado a la generación común;
- gramática ASCII exacta de `org_`, sin depender de una colación nominal;
- restricciones, índices, propietarios, TOAST y dependencias exactos;
- RLS activada y forzada, con una única política del propietario;
- cero acceso directo o efectivo para `PUBLIC`, runtime, selector o LOGIN;
- dos triggers exactos de historia y tres de puntero;
- opt-in literal, instalación vacía, inventario completo y `RESTRICT`;
- rechazo de filas, derivas, consumidores hostiles y una `000004` sintética;
- carreras de alta, retirada, DDL y DML sin interbloqueo ni retirada parcial;
- ejecución de los bytes completos mediante una conexión `pgx` dedicada;
- cancelación real, `TxStatus E → I`, `ROLLBACK`, `RESET` y destrucción de una
  conexión que no puede sanearse;
- composición directa y exactamente una vez, como última matriz focal.

## Pruebas

El productor, dos revisores y dirección ejecutaron en distintos ciclos:

```text
probar_organizacion_corporativa_v1_pg18_4.sh × múltiples ejecuciones
probar_integracion.sh completo
probar_integracion.sh con VEC_CONTEXTO_ACTOR_OMITIR_GO=1
go test del paquete ContextoActor/PostgreSQL
go test -race del paquete ContextoActor/PostgreSQL
go vet del paquete ContextoActor/PostgreSQL
bash -n
ShellCheck
gofmt
git diff --check
Gitleaks focal
scripts/verificar_calidad.sh
```

Todas terminaron en verde. La puerta global incluyó además pruebas normales y
de carrera de todo el repositorio, builds, aislamiento de superficies,
manifiestos web, carga TLS no privilegiada, análisis de vulnerabilidades y
límites de fichero.

## Frontera y continuación

C2.2-A no publica ni revoca organizaciones, no selecciona actores, no
autoriza consultas RRHH y no incorpora AD, nómina, RPT o datos reales. A5 debe
sincronizar estado y publicar el corte. Solo entonces se desbloquea C2.2-B,
historia y puntero del vínculo corporativo.
