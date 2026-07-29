# Revisión de las fachadas nominales CT-000045

Fecha: 29 de julio de 2026.

## Resultado

**GO técnico independiente** para el alcance CT-000045.

| Elemento | Valor |
| --- | --- |
| Base estable | `c3e93ba` |
| Candidato revisado | `02b02edc0054950ed79eb36f8375b7be872dbd6b` |
| PostgreSQL | 18.4 fijado por digest |
| P0 | 0 |
| P1 | 0 |
| P2 | 1 heredado, no bloqueante |

Este cierre no acredita todavía composición raíz, TLS/mTLS viva, interfaz web
conectada, E2E administrativo ni autorización de producción.

## Alcance funcional acreditado

El paquete instala exactamente dos funciones exteriores nominales:

- `consultar_cuadro_rrhh_atestado_v1`;
- `consultar_detalle_rrhh_atestado_v1`.

Ambas:

- reciben doce argumentos de entrada tipados;
- fijan acción, finalidad, audiencia, módulo y tipo de recurso como constantes;
- exigen una capacidad VEC nueva y el canon funcional exacto;
- ejecutan la consulta bajo `SERIALIZABLE READ WRITE`;
- consumen autorización, registran el acceso y leen el motor privado dentro de
  la misma transacción;
- normalizan los rechazos sin convertir ausente, ajeno, denegado o versión
  incorrecta en oráculos;
- rechazan cualquier salida obligatoria nula antes del `COMMIT`;
- devuelven únicamente la proyección minimizada contratada.

El rol nominal de ejecución solo obtiene `EXECUTE` sobre las dos fachadas. No
recibe acceso directo al motor, a sus tipos privados ni a las tablas. El
paquete avanza las barreras exactas `24/8` a `25/9`; la reversión exige
`25/9`, vuelve a `24/8`, usa `RESTRICT` y preserva la historia.

Huella semántica final:

```text
400de6a2b39a2b65dbd2d32137268e3ceaff784342cd0fb2c058eaec670f6b8a
```

## Evidencia reproducida

Comando integral:

```text
bash deploy/postgresql/contratacion_temporal/probar_o4_05_fachadas_nominales_ct45_pg18_4.sh
```

Resultado:

```text
CT-000045 superada sobre PostgreSQL 18.4 fijado por digest
```

La matriz cubre:

- instalación, retirada y reentrada sobre una base efímera;
- OID, firmas, constantes, ACL, topología, límites y huella;
- transacción serializable de lectura y escritura;
- rechazo de identidad indirecta, sesión puente y privilegios excesivos;
- enlaces escalares exactos;
- éxito atómico y rollback;
- mutación de capacidad VEC y de constantes;
- replay;
- preservación de los cuatro SQLSTATE transitorios;
- carrera causal;
- rollback ante cinco familias de salidas nulas;
- barrera de retirada segura de CT-000044;
- barrera futura y dependencias;
- deriva de ACL y configuración.

También quedaron verdes:

```text
bash -n
shellcheck
git diff --check
go test ./internal/modules/contrataciontemporal/...
go test -race ./internal/modules/contrataciontemporal/...
go vet ./internal/modules/contrataciontemporal/...
gitleaks sobre c3e93ba..02b02ed
```

Todos los ficheros permanecen por debajo del máximo de 800 líneas; el runner
mayor tiene 792.

## Revisión independiente

Un revisor distinto del productor examinó el candidato exacto `02b02ed`,
incluido el último delta de cuatro ficheros de prueba. Confirmó que:

- no cambió el DDL productivo ni su contrato;
- los auxiliares de prueba viven fuera de `public`, tienen ACL explícita y se
  eliminan;
- los motores instrumentados recuperan su nombre original;
- el trigger sintético de SQLSTATE se elimina al terminar;
- `ON_ERROR_STOP` impide falsos positivos;
- los enlaces nombrados son válidos para `psql`;
- los identificadores sintéticos concuerdan con la normalización de PostgreSQL.

El único P2 es heredado: un runner histórico transmite centinelas
exclusivamente sintéticos mediante variables de entorno de `docker exec`. No
pertenece al delta CT-000045, no contiene datos reales y no bloquea este
cierre. Se mantiene visible como deuda de endurecimiento, sin ampliar ahora el
camino crítico.

## Límites y siguiente puerta

CT-000045 publica la frontera SQL mínima; no crea una segunda identidad,
autorización, auditoría o sesión y no introduce lógica web.

La siguiente tarea es implementar únicamente el adaptador PostgreSQL de
`SesionConsultaRRHH`, reutilizando el puerto, las órdenes y los casos de uso
existentes. Ese corte no modificará el dominio, la aplicación, HTTP ni la
composición raíz.

