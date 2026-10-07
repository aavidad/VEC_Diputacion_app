# AD175: exportación de servicios propios

Esta candidata reancla el consumidor nominal de exportación al núcleo PostgreSQL 18 posterior a AD211. La definición medida tiene SHA-256 `15982f938efccedbf4b8796395b5777308ade672e923a2c5e0193ab6034a3e76`; su cuerpo, `c0013c1311e8865379797a725f3485720cd6d9ea7523b429bd58b18c0ded3c9a`. El CHECK de audiencias, obtenido con `pg_get_constraintdef(..., false)`, tiene SHA-256 `d54d76c9f34009f3313e5f236b78cfc173e6547ec4bfbebe2be3785cd13f4a17`.

AD175 comprueba las tres huellas antes de modificar el núcleo o el CHECK. Añade el perfil técnico `exportacion_servicios_propios` a la rama de LOGIN de Personal, a la exclusión general y al contrato de consumo. La acción es `personal.registro_empleado.ficha_propia.servicios.exportar`; la audiencia es `vec_personal.registro_empleado.ficha_propia.servicios.exportar.v1`. El permiso de consultar la ficha no concede exportación. La función nueva solo queda ejecutable para el propietario de Personal. La provisión del perfil funcional, la concesión central y el origen técnico siguen en sus autoridades existentes.

La preimagen se verificó estáticamente con:

```sh
python3 deploy/postgresql/autorizacion_atestada_v3/pruebas_sql/verificar_ad175_preimagen.py \
  /ruta/privada/nucleo-post-ad211.json \
  deploy/postgresql/autorizacion_atestada_v3/migraciones/000175_consumidor_exportacion_servicios_propios.up.sql
```

El resultado es `AD175-PREIMAGEN-ESTATICA-OK`. Comprueba definición, cuerpo, propietario, ACL, configuración, marcas únicas e inversión exacta del parche; también exige la huella del CHECK antes del primer DDL. La ruta privada no forma parte de Git. Falta el ensayo de AD175 y Personal32 en una copia PG18 de la principal, seguido de las revisiones independientes del contenido final. Un cambio previo del núcleo o del CHECK detiene AD175 y exige medir otra vez; no se aplica AD74 ni se modifica SQL con historia.
