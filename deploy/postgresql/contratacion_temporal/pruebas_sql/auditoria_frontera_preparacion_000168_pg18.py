#!/usr/bin/env python3
"""Emite el ensayo CT168 para psql del clon cedido por Dirección, sin conectar."""
from pathlib import Path

raiz = Path(__file__).resolve().parents[4]
up = (raiz / "deploy/postgresql/contratacion_temporal/migraciones/000168_auditoria_frontera_preparacion.up.sql").read_text()
pruebas = (raiz / "deploy/postgresql/contratacion_temporal/pruebas_sql/auditoria_frontera_preparacion_000168.sql").read_text()
if up.count("\nBEGIN;\n") != 1 or not up.endswith("COMMIT;\n"):
    raise SystemExit("CT168: estructura transaccional del UP inesperada")
up = up.replace("\nBEGIN;\n", "\nSAVEPOINT ct168_nuevo_up;\n", 1)
up = up[:-len("COMMIT;\n")] + "RELEASE SAVEPOINT ct168_nuevo_up;\n"
# La postimagen de CT162 es la preimagen del ensayo; el UP comprueba sus SHA.
estado = """jsonb_build_object(
 'funcion',(SELECT to_jsonb(p) FROM pg_proc p WHERE p.oid='vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'::regprocedure),
 'dependencias',(SELECT jsonb_agg(to_jsonb(d) ORDER BY d.refclassid,d.refobjid,d.refobjsubid,d.deptype) FROM pg_depend d WHERE d.classid='pg_proc'::regclass AND d.objid='vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'::regprocedure),
 'dependencias_compartidas',(SELECT jsonb_agg(to_jsonb(d) ORDER BY d.refclassid,d.refobjid,d.deptype) FROM pg_shdepend d WHERE d.classid='pg_proc'::regclass AND d.objid='vec_contratacion_temporal.registrar_auditoria_frontera_ruta_exacta_v1(text,text,text,text,text)'::regprocedure AND d.dbid=(SELECT oid FROM pg_database WHERE datname=current_database())),
 'tabla',(SELECT to_jsonb(c) FROM pg_class c WHERE c.oid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass),
 'constraints',(SELECT jsonb_agg(to_jsonb(c) ORDER BY c.conname) FROM pg_constraint c WHERE c.conrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass),
 'triggers',(SELECT jsonb_agg(to_jsonb(t) ORDER BY t.tgname) FROM pg_trigger t WHERE t.tgrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass),
 'policies',(SELECT jsonb_agg(to_jsonb(p) ORDER BY p.polname) FROM pg_policy p WHERE p.polrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass),
 'indices',(SELECT jsonb_agg(to_jsonb(i) ORDER BY i.indexrelid) FROM pg_index i WHERE i.indrelid='vec_contratacion_temporal.auditoria_frontera_ruta_exacta'::regclass),
 'secuencia',(SELECT jsonb_build_object('last_value',s.last_value,'log_cnt',s.log_cnt,'is_called',s.is_called) FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta_evento_id_seq s),
 'historia',(SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(t) ORDER BY t.evento_id)::text,'[]'),'UTF8')),'hex') FROM vec_contratacion_temporal.auditoria_frontera_ruta_exacta t))"""
print("\\set ON_ERROR_STOP on")
print("SET search_path=pg_catalog,pg_temp;")
print("CREATE TEMP TABLE ct168_restauracion AS SELECT " + estado + " AS estado;")
print("BEGIN; SET LOCAL lock_timeout='5s'; SET LOCAL statement_timeout='30s';")
print(up)
print(pruebas)
print("ROLLBACK;")
print("DO $restauracion$ BEGIN IF (SELECT estado FROM ct168_restauracion) IS DISTINCT FROM " + estado + " THEN RAISE EXCEPTION 'CT168: postROLLBACK no equivale a postCT162'; END IF; END $restauracion$;")
print("SELECT 'CT168-ROLLBACK-POSTCT162-FUNCION-ACL-HISTORIA-SECUENCIA-IDENTICA' AS resultado;")
