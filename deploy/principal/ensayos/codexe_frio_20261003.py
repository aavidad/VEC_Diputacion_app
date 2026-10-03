#!/usr/bin/env python3
"""Ejecuta una lista causal exacta en un contenedor PostgreSQL aislado existente.
No crea bases ni reaplica migraciones. Lista JSON: id, path, sha256, installed_sql.
"""
import argparse, hashlib, json, pathlib, subprocess, sys
p=argparse.ArgumentParser()
p.add_argument('container');p.add_argument('manifest');p.add_argument('output')
p.add_argument('--rollback',action='store_true',help='Ensaya cada migración y revierte su transacción final; no instala dependencias entre piezas.')
a=p.parse_args();out=pathlib.Path(a.output);out.mkdir(parents=True,exist_ok=True)
cmd=['docker','exec','-i',a.container,'psql','-h','/tmp','-U','postgres','-d','postgres','-X','-q','-At','-v','ON_ERROR_STOP=1']
def sql(s):
 r=subprocess.run(cmd,input=s.encode(),stdout=subprocess.PIPE,stderr=subprocess.PIPE)
 if r.returncode: raise RuntimeError(r.stderr.decode())
 return r.stdout.decode().strip()
def snapshot(label):
 rows=json.loads(sql("""
 CREATE TEMP TABLE ensayo_historia (nombre text PRIMARY KEY, valor text);
 DO $snapshot$
 DECLARE r record; v text;
 BEGIN
 FOR r IN SELECT n.nspname,c.relname FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE c.relkind='r' AND n.nspname LIKE 'vec_%' ORDER BY 1,2 LOOP
 EXECUTE format($q$SELECT count(*)||'|'||encode(sha256(convert_to(coalesce(string_agg(to_jsonb(t)::text,chr(10) ORDER BY to_jsonb(t)::text),''),'UTF8')),'hex') FROM %I.%I t$q$,r.nspname,r.relname) INTO v;
 INSERT INTO ensayo_historia VALUES(r.nspname||'.'||r.relname,v);
 END LOOP;
 END $snapshot$;
 SELECT json_object_agg(nombre,valor ORDER BY nombre) FROM ensayo_historia;
 """))
 role_fingerprints=json.loads(sql("SELECT json_object_agg(rolname,encode(sha256(convert_to(to_jsonb(r)::text,'UTF8')),'hex') ORDER BY rolname) FROM pg_roles r;"))
 roles=sql("SELECT encode(sha256(convert_to(coalesce(jsonb_agg(to_jsonb(r) ORDER BY rolname)::text,'[]'),'UTF8')),'hex') FROM pg_roles r;")
 acl=sql("SELECT encode(sha256(convert_to(coalesce(jsonb_agg(jsonb_build_array(n.nspname,c.relname,c.relowner,c.relacl,c.relrowsecurity,c.relforcerowsecurity) ORDER BY n.nspname,c.relname)::text,'[]'),'UTF8')),'hex') FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%';")
 procs=sql("SELECT n.nspname||'.'||p.proname||'|'||oidvectortypes(p.proargtypes)||'|'||encode(sha256(convert_to(pg_get_functiondef(p.oid),'UTF8')),'hex')||'|'||coalesce(p.proacl::text,'NULL') FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace WHERE n.nspname LIKE 'vec_%' AND p.prokind='f' ORDER BY 1;")
 catalog=sql("""SELECT encode(sha256(convert_to(jsonb_build_object(
 'schemas',(SELECT jsonb_agg(jsonb_build_array(nspname,nspowner,nspacl) ORDER BY nspname) FROM pg_namespace WHERE nspname LIKE 'vec_%'),
 'relations',(SELECT jsonb_agg(to_jsonb(c) ORDER BY c.oid) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%'),
 'attributes',(SELECT jsonb_agg(to_jsonb(a) ORDER BY a.attrelid,a.attnum) FROM pg_attribute a JOIN pg_class c ON c.oid=a.attrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%'),
 'constraints',(SELECT jsonb_agg(to_jsonb(c) ORDER BY c.oid) FROM pg_constraint c JOIN pg_namespace n ON n.oid=c.connamespace WHERE n.nspname LIKE 'vec_%'),
 'policies',(SELECT jsonb_agg(to_jsonb(p) ORDER BY p.oid) FROM pg_policy p JOIN pg_class c ON c.oid=p.polrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%'),
 'triggers',(SELECT jsonb_agg(to_jsonb(t) ORDER BY t.oid) FROM pg_trigger t JOIN pg_class c ON c.oid=t.tgrelid JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname LIKE 'vec_%'),
 'memberships',(SELECT jsonb_agg(to_jsonb(m) ORDER BY m.roleid,m.member,m.grantor) FROM pg_auth_members m),
 'database',(SELECT jsonb_build_array(datname,datdba,datacl) FROM pg_database WHERE datname=current_database())
 )::text,'UTF8')),'hex');""")
 d={'history':rows,'role_fingerprints':role_fingerprints,'roles_sha256':roles,'tables_acl_sha256':acl,'schema_catalog_sha256':catalog,'functions':procs.splitlines()}
 (out/(label+'.json')).write_text(json.dumps(d,indent=2)+'\n');return d
manifest=json.loads(pathlib.Path(a.manifest).read_text());log=[]
before=snapshot('before')
for item in manifest:
 data=pathlib.Path(item['path']).read_bytes();digest=hashlib.sha256(data).hexdigest()
 if digest!=item['sha256']: raise RuntimeError('PARO clave=artefacto_sha256 actual='+digest+' esperado='+item['sha256'])
 installed=sql(item['installed_sql'])=='t'
 state={'id':item['id'],'sha256':digest,'installed':installed};log.append(state)
 print('INVENTARIO',item['id'],'instalada='+str(installed),flush=True)
(out/'inventory.json').write_text(json.dumps(log,indent=2)+'\n')
for item,state in zip(manifest,log):
 if state['installed']:
  print('OMITIDA_INSTALADA',item['id'],flush=True);continue
 pre=snapshot('pre-'+item['id'])
 data=pathlib.Path(item['path']).read_bytes()
 if a.rollback:
  # Sólo archivos con cierre explícito: nunca añadir ROLLBACK después de COMMIT.
  if not data.rstrip().endswith(b'COMMIT;'):
   raise RuntimeError('PARO clave=cierre_transaccion esperado=COMMIT_final')
  data=data.rstrip()[:-len(b'COMMIT;')]+b'ROLLBACK;\n'
 r=subprocess.run(cmd,input=data,stdout=subprocess.PIPE,stderr=subprocess.PIPE)
 state['exit_code']=r.returncode
 state['stderr']=r.stderr.decode();state['stdout_sha256']=hashlib.sha256(r.stdout).hexdigest()
 post=snapshot('post-'+item['id'])
 state['history_changes']=[t for t,h in pre['history'].items() if post['history'].get(t)!=h]
 state['roles_preserved']=pre['roles_sha256']==post['roles_sha256']
 state['previous_roles_preserved']=all(post['role_fingerprints'].get(k)==v for k,v in pre['role_fingerprints'].items())
 state['new_roles']=sorted(set(post['role_fingerprints'])-set(pre['role_fingerprints']))
 if a.rollback:
  state['rollback_snapshot_preserved']=pre==post
  if r.returncode==0 and not state['rollback_snapshot_preserved']:
   raise RuntimeError('PARO clave=rollback_snapshot actual=distinto esperado=identico')
 (out/'results.json').write_text(json.dumps(log,indent=2)+'\n')
 print('OK' if not r.returncode else 'FALLO',item['id'],'historia_cambios='+str(len(state['history_changes'])),'roles_previos_conservados='+str(state['previous_roles_preserved']),'roles_nuevos='+str(len(state['new_roles'])),flush=True)
 if r.returncode:
  print(r.stderr.decode(),flush=True);print('ROLLBACK_HISTORY',not state['history_changes'],flush=True)
  sys.exit(1)
print('ENSAYO-OK',flush=True)
