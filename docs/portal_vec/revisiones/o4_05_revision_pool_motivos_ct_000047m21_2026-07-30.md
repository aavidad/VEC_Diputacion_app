# Revisión final del pool nominal de motivos CT-000047M2.1

Fecha: 30 de julio de 2026.

## Resultado

**GO técnico independiente**.

| Elemento | Valor |
| --- | --- |
| Base estable previa | `175de4e` |
| Candidato acumulado | `57c914f` sobre `281f52b` |
| Commits integrados | `203e59d`, `e6d3979`, `b385f5c` |
| P0 | 0 |
| P1 | 0 |
| P2 | 1 no bloqueante |

El P2 conserva la petición de ampliar en otro corte la cobertura de las
fronteras privadas del creador del pool. No afecta a la denegación por
defecto observada ni habilita una composición productiva alternativa.

## Garantías verificadas

El constructor público recibe solo contexto, DSN y nombre de `LOGIN`. El
usuario de la DSN debe coincidir exactamente y el pool subyacente nunca se
expone. Pool, conexión y transacción conservan un mismo sello privado; una
copia, una configuración mutada o una transacción ajena fallan cerradas.

La acreditación inicial y la reacreditación de cada transacción comprueban:

- TLS con identidad del servidor, versión y cifrado permitidos, o socket Unix
  limitado a la prueba privada;
- servidor primario e identidad exacta
  `session_user = current_user = LOGIN`;
- atributos seguros y una sola membresía directa en el rol resolutor, con
  `ADMIN FALSE`, `INHERIT TRUE`, `SET FALSE` y otorgante bootstrap OID 10;
- ausencia de ajustes, propiedades, políticas, ACL directas o autoridad
  adicional del `LOGIN` y del grupo;
- denegación de `PUBLIC` y privilegios efectivos en todos los esquemas
  aplicativos para relaciones, columnas, secuencias, funciones y tipos;
- ACL predeterminadas, políticas RLS y dependencias compartidas, incluido el
  `dbid` de cada objeto;
- propietario, firma, configuración, comentarios, cuerpo, longitud, huella y
  OID de las dos fachadas `000010`;
- orden causal de locks M1.R → `000008` → `000009` → `000010`.

Los arrays automáticos se reconocen exclusivamente por la relación
bidireccional `typelem`/`typarray`. Su ACL implícita nula no se confunde con un
tipo aplicativo que solo declare `ELEMENT`; una ACL explícita del array vuelve
a ser inventariada.

## Hallazgos corregidos antes del GO

La primera ejecución real descubrió dos defectos:

1. el planificador podía evaluar `has_sequence_privilege` sobre una relación
   que no era secuencia;
2. una unión SQL infería `name` y truncaba las firmas de función.

Después, la revisión Sol exigió cerrar la autoridad heredada de `PUBLIC`,
añadir identidad divergente, sellar la transacción e incluir `dbid`. Una
segunda contrarrevisión detectó que `typelem <> 0` no identifica por sí solo
un array automático. Todos los candidatos anteriores quedaron descartados y
las correcciones se revisaron de nuevo sobre `57c914f`.

## Evidencia

| Fichero | SHA-256 | Líneas |
| --- | --- | ---: |
| `fabrica_pool_resolucion_motivos_rrhh.go` | `2336ae488a03bb61712b3bb040ef06b97a5b869e7ea8124beef7cdcd0c4b5e03` | 345 |
| `acreditacion_pool_resolucion_motivos_rrhh.go` | `f2a8b226bfb431c0fa08373cf2c2b2ac203d475addf00259a8ca3207e21c7095` | 642 |
| `fabrica_pool_resolucion_motivos_rrhh_test.go` | `c6ca0c5d1aba0bb381d2518f26a2688dfce60c39ccd6a53e0648c165a424de40` | 751 |

- PostgreSQL 18.4 real: estado limpio positivo antes y después de los venenos;
- rechazo real de tabla, tipo, esquema, ACL predeterminada y política RLS
  concedidos a `PUBLIC`;
- rechazo real de un tipo con `typelem` hostil no recíproco y de un array
  automático con ACL explícita;
- pruebas del paquete, carrera focal, `go vet`, formato y
  `git diff --check`: verdes;
- Gitleaks diferencial: tres commits y cero hallazgos;
- dos revisiones independientes: seguridad P0=P1=P2=0 y funcional P0=P1=0;
- ningún contenedor, DSN, contraseña ni fichero temporal quedó conservado.

## Límites

M2.1 no resuelve todavía las referencias de negocio. M2.2 implementará los dos
métodos nominales; M2.3 conservará el runner PostgreSQL completo y su matriz
adversarial; M2.4 hará la composición. No se acreditan aún identidad
corporativa, PDP, TLS/mTLS viva, web, E2E ni producción. Las métricas
funcionales no aumentan y producción permanece en `NO-GO`.
