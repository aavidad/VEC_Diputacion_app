# Ensayo local PostgreSQL 18.4 de composición V3 (25-09-2026)

## Reproducción

Desde la raíz del repositorio:

```bash
bash deploy/principal/composicion_interna/pruebas_canonicas/prueba_cadena_contexto_pg18.sh ad3
bash deploy/principal/composicion_interna/pruebas_canonicas/prueba_cadena_contexto_pg18.sh contexto
```

Cada ejecución crea un contenedor nuevo `vec-comp-v3-canon-pg18-<pid>`, sin red ni
puertos, con datos sintéticos, y lo elimina al salir. No reutiliza ninguna base
ni ejecuta `DOWN`. El segundo comando devuelve código 3 por la guarda canónica.

## Resultado observado

- La rama `ad3` devolvió código 0. Instaló ContextoActor 000001/000002,
  Autorización 000001/000003–000007, los roles CT y AD3, y AD3
  000001/000002/000050a/000053 desde una base vacía. Las migraciones AD3
  000001/000002 se ejecutaron como un LOGIN migrador sin superusuario.
- PostgreSQL informó `180004`. La función
  `vec_autorizacion_atestada_v3.leer_configuracion_interna_v1(jsonb)` existe;
  `preflight_interno` tiene `EXECUTE` y `consumidor` no lo tiene. Las tablas de
  gobierno muestreadas tienen `RLS` y `FORCE RLS`; el LOGIN preflight carece de
  `SELECT` directo. Como ese LOGIN, la llamada con material vacío devolvió
  `42501`. Función y ACL persistieron tras reiniciar **ese** contenedor.
- La rama `contexto` devolvió código 3: tras instalar ContextoActor
  000001/000002/000003 y el rol selector, 000004 rechazó su propia preimagen
  con `manifiesto simbolico del predecesor no acreditado`. Huella observada:
  `613d1837116a903bc49accb0f63d25a3ce2094421888daa915674a1546e4cbe3`;
  huella literal esperada por 000004:
  `bddc55742ae4d509cb884bbf464ac4f90c23c6b680d338943160ea1ee3b1742c`.
  SHA256 del archivo 000004 ejecutado:
  `f1be3123b1286e7fe8ffae99073179b90b08303c31c8ee8c3783315a6078f368`.

## Límite de la evidencia

La CLI `vec-publicar-permiso-interno` necesita el vínculo corporativo de
ContextoActor 000004, Identidad 000004/000006 y F1 real. No se ejecutó contra
tablas canónicas porque esa cadena se detiene en 000004. El fixture propio de
AD3-53 es un **esquema sintético**: su primer `INSERT` en `checkpoint_gobierno`
omite `actualizada_en`, columna obligatoria en la tabla canónica. AD3-002 aún
limita las claves a la audiencia de alta CT; la lectura positiva de AD3-53
requiere cinco audiencias y depende de las migraciones posteriores que amplían
ese catálogo. Por ello este ensayo no acredita lectura positiva, rotación,
revocación, replay ni decisión PDP de una configuración publicada.

No se creó ni tocó `vec-composicion-v3-pg18-20260925` ni servicios compartidos.
