# O4-05: revisión final del contrato tipado RRHH

Fecha: 28 de julio de 2026

Ámbito: C2-D2-D, migración CT `000040`, tipos y cánones internos del motor
de consultas RRHH

Estado: `GO` técnico con revisión independiente; no autoriza producción

## Alcance cerrado

La migración `000040` aporta, sin ejecutar todavía lecturas de negocio:

- tipos nominales para alcance, cuadro, familia de paginación, detalle y
  evidencia de resultado;
- cánones cerrados compatibles con Go y funciones auxiliares sin privilegio
  para `PUBLIC` ni para usuarios de ejecución;
- límites de 256 KiB, profundidad 24 y 16.384 nodos para material JSON;
- límite de página entre 1 y 100 y cursor canónico;
- control de esquema `20/4`, RLS forzada y catálogo semántico;
- igualdad obligatoria y validada entre la huella de sesión registrada y la
  huella autoritativa del control de sesión;
- reversión transaccional `RESTRICT` que rechaza objetos o dependencias
  posteriores.

La huella de sesión no constituye una autoridad nueva. Queda definida como
`control_sesion_huella_sha256`, procedente del vínculo de autenticación de la
decisión VEC-AD3 y de la revalidación viva de Identidad. El motor posterior
deberá comprobar esa igualdad antes y después de la lectura.

## Evidencia reproducida

El runner
[`probar_o4_05_contrato_motor_rrhh_pg18_4.sh`](../../../deploy/postgresql/contratacion_temporal/probar_o4_05_contrato_motor_rrhh_pg18_4.sh)
usa PostgreSQL 18.4 fijado por
`sha256:1961f96e6029a02c3812d7cb329a3b03a3ac2bb067058dec17b0f5596aca9296`.

La revisión independiente ejecutó tres ciclos completos, todos con salida
cero. Dirección repitió después el runner sobre la rama estable integrada.
El estado observado en las dos últimas reproducciones independientes fue:

```text
20|4|true|true|true|true|true|badc3c6db4d91ddeb3328a58adc86832e666d24e3ce18010159b74d50f908e9e
```

La matriz acredita:

- los tres vectores de alcance Go/SQL para organización, centro y unidad;
- instalación, reentrada, `UP → DOWN → UP` y barreras futuras;
- rechazo transaccional de una historia previa con huellas de sesión y
  control distintas;
- restricción inmediata, validada y no diferible;
- detección de índices, reglas, herencia, estadísticas, publicaciones y
  etiquetas de seguridad añadidas posteriormente;
- rechazo de dependencias futuras sin `CASCADE`;
- propietarios, RLS, políticas, ACL y firmas exactas;
- ausencia de permisos para `PUBLIC` y para los roles de ejecución.

No se reprodujo la inestabilidad histórica de la prueba de bloqueo de
`000039`: las tres pasadas del revisor y la pasada de dirección fueron verdes.

## Hashes congelados

| Fichero | Líneas | SHA-256 |
|---|---:|---|
| `000040 up` | 726 | `f0195aba5ff3f84e075e893b505811d2e6b9ee4a3c4b4afaa8b06c0f6be6c722` |
| `000040 down` | 455 | `7ab40d4f31378296d04121bb5d7f4294666013702ef49b94b92fe0af5e7ecd8a` |
| prueba SQL | 496 | `373d1ad6b0380e9db432d0db7c9e6fe166757399a24874b9f6a06e235f14aca0` |
| runner | 351 | `7fe59f52a16c2e02bb7dfe1284df5017086ec4bdecf4caf4d887ebe8b2fbef05` |
| apoyo CT `000039` corregido | 315 | `05a488d30e00aee40cc663e444cebedd3874d289541b077945ee2421c5c03441` |

`bash -n`, ShellCheck, la puerta de 800 líneas, `git diff --check` y Gitleaks
del rango quedaron verdes. Los hallazgos del barrido histórico pertenecen a
la línea base de pruebas y no al cambio revisado.

## Dictamen y trabajo siguiente

CT `000040` queda cerrada. No existe todavía un motor productivo ni se
autoriza despliegue. Antes de la ejecución interna se ha separado CT `000041`
como contrato probatorio:

1. constructor de detalle minimizado con presupuesto máximo de 256 KiB;
2. seis estados operativos publicables;
3. cánones cerrados del contenido de cuadro y detalle;
4. recibo V2 ligado al resultado, consumo VEC, identidad y cadena de acceso.

Después siguen:

- CT `000042`: motor interno propietario, lectura y confirmación;
- CT `000043`: fachadas nominales y privilegio mínimo;
- adaptador PostgreSQL Go, composición raíz y E2E HTTP/web.

La métrica oficial pasa a Contratación temporal `20/46` (43 %). O4-05
permanece en `3/5`, Bolsa en `1/14` (7 %) y producción en `NO-GO`.
