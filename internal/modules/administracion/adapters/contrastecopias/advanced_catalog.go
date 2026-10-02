package contrastecopias

// Referencias de catálogo, no valores de aplicación. Los OID sólo localizan
// filas; las representaciones finales usan nombres, firmas y orden semántico.
// Esta consulta devuelve una columna text y requiere search_path=pg_catalog.
// Las definiciones son evidencia privada transitoria: no deben llegar a logs.
// Documentación de columnas: https://www.postgresql.org/docs/18/catalogs.html
const esquemaAvanzadoSQL = `
 WITH
 cs_type AS (
   SELECT t.oid,jsonb_build_array(n.nspname,t.typname) AS ident
   FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
 ),
 cs_proc AS (
   SELECT p.oid,jsonb_build_array(n.nspname,p.proname,pg_get_function_identity_arguments(p.oid)) AS ident
   FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 ),
 cs_relation AS (
   SELECT r.oid,jsonb_build_array(n.nspname,r.relname) AS ident
   FROM pg_class r JOIN pg_namespace n ON n.oid=r.relnamespace
 ),
 cs_collation AS (
   SELECT c.oid,jsonb_build_array(n.nspname,c.collname,c.collencoding) AS ident
   FROM pg_collation c JOIN pg_namespace n ON n.oid=c.collnamespace
 ),
 cs_operator AS (
   SELECT o.oid,jsonb_build_array(n.nspname,o.oprname,l.ident,r.ident) AS ident
   FROM pg_operator o JOIN pg_namespace n ON n.oid=o.oprnamespace
   LEFT JOIN cs_type l ON l.oid=o.oprleft LEFT JOIN cs_type r ON r.oid=o.oprright
 ),
 cs_family AS (
   SELECT f.oid,jsonb_build_array(n.nspname,f.opfname,a.amname) AS ident
   FROM pg_opfamily f JOIN pg_namespace n ON n.oid=f.opfnamespace JOIN pg_am a ON a.oid=f.opfmethod
 ),
 cs_class AS (
   SELECT c.oid,jsonb_build_array(n.nspname,c.opcname,a.amname) AS ident
   FROM pg_opclass c JOIN pg_namespace n ON n.oid=c.opcnamespace JOIN pg_am a ON a.oid=c.opcmethod
 ),
 cs_constraint AS (
   SELECT c.oid,jsonb_build_array(n.nspname,r.ident,t.ident,c.conname) AS ident
   FROM pg_constraint c JOIN pg_namespace n ON n.oid=c.connamespace
   LEFT JOIN cs_relation r ON r.oid=c.conrelid LEFT JOIN cs_type t ON t.oid=c.contypid
 ),
 cs_trigger AS (
   SELECT t.oid,CASE WHEN t.tgisinternal
     THEN jsonb_build_array('internal trigger',r.ident,p.ident,c.ident,t.tgtype)
     ELSE jsonb_build_array('trigger',r.ident,t.tgname) END AS ident
   FROM pg_trigger t JOIN cs_relation r ON r.oid=t.tgrelid
   JOIN cs_proc p ON p.oid=t.tgfoid LEFT JOIN cs_constraint c ON c.oid=t.tgconstraint
 ),
 cs_addresses AS (
   SELECT classoid AS classid,objoid AS objid,objsubid FROM pg_description
   UNION SELECT classoid,objoid,0 FROM pg_shdescription
   UNION SELECT classoid,objoid,objsubid FROM pg_seclabel
   UNION SELECT classoid,objoid,0 FROM pg_shseclabel
   UNION SELECT d.classid,d.objid,d.objsubid FROM pg_depend d
     WHERE d.refclassid='pg_extension'::regclass AND d.deptype IN ('e','x')
 ),
 cs_address AS (
   SELECT a.classid,a.objid,a.objsubid,
     CASE WHEN a.classid IN ('pg_largeobject'::regclass,'pg_largeobject_metadata'::regclass)
                    OR rn.nspname ~ '^pg_toast'
            THEN jsonb_build_array('unresolved internal metadata')
          WHEN a.classid='pg_trigger'::regclass THEN t.ident
          ELSE jsonb_build_array(i.type,i.schema,i.name,i.identity) END AS ident
   FROM cs_addresses a
   CROSS JOIN LATERAL pg_identify_object(a.classid,a.objid,a.objsubid) i
   LEFT JOIN cs_trigger t ON a.classid='pg_trigger'::regclass AND t.oid=a.objid
   LEFT JOIN pg_class rr ON a.classid='pg_class'::regclass AND rr.oid=a.objid
   LEFT JOIN pg_namespace rn ON rn.oid=rr.relnamespace
 ),
 cs_safe_deparse AS (
   SELECT NOT (` + hooksTipoNoSegurosSQL + `) AS ok
 )
 SELECT jsonb_build_array('routine',n.nspname,p.proname,pg_get_function_identity_arguments(p.oid),
   pg_get_userbyid(p.proowner),l.lanname,p.prokind,p.procost,p.prorows,p.prosecdef,p.proleakproof,
   p.proisstrict,p.proretset,p.provolatile,p.proparallel,p.pronargs,p.pronargdefaults,
   rt.ident,vt.ident,sp.ident,
   coalesce((SELECT jsonb_agg(jsonb_build_array(a.ord,t.ident) ORDER BY a.ord)
     FROM unnest(p.proargtypes::oid[]) WITH ORDINALITY a(typ,ord) JOIN cs_type t ON t.oid=a.typ),'[]'::jsonb),
   coalesce((SELECT jsonb_agg(jsonb_build_array(a.ord,t.ident) ORDER BY a.ord)
     FROM unnest(p.proallargtypes) WITH ORDINALITY a(typ,ord) JOIN cs_type t ON t.oid=a.typ),'[]'::jsonb),
   p.proargmodes,p.proargnames,
   coalesce((SELECT jsonb_agg(t.ident ORDER BY t.ident) FROM unnest(p.protrftypes) a(typ) JOIN cs_type t ON t.oid=a.typ),'[]'::jsonb),
   coalesce((SELECT jsonb_agg(v ORDER BY v) FROM unnest(p.proconfig) a(v)),'[]'::jsonb),
   CASE WHEN p.prokind<>'a' AND (SELECT ok FROM cs_safe_deparse) THEN pg_get_functiondef(p.oid) END)::text
 FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace JOIN pg_language l ON l.oid=p.prolang
 JOIN cs_type rt ON rt.oid=p.prorettype LEFT JOIN cs_type vt ON vt.oid=p.provariadic
 LEFT JOIN cs_proc sp ON sp.oid=p.prosupport WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('aggregate',pr.ident,a.aggkind,a.aggnumdirectargs,
   tf.ident,ff.ident,cf.ident,sf.ident,df.ident,mtf.ident,mif.ident,mff.ident,
   a.aggfinalextra,a.aggmfinalextra,a.aggfinalmodify,a.aggmfinalmodify,so.ident,
   tt.ident,a.aggtransspace,mtt.ident,a.aggmtransspace,a.agginitval,a.aggminitval)::text
 FROM pg_aggregate a JOIN pg_proc p ON p.oid=a.aggfnoid JOIN pg_namespace n ON n.oid=p.pronamespace
 JOIN cs_proc pr ON pr.oid=p.oid JOIN cs_proc tf ON tf.oid=a.aggtransfn
 LEFT JOIN cs_proc ff ON ff.oid=a.aggfinalfn LEFT JOIN cs_proc cf ON cf.oid=a.aggcombinefn
 LEFT JOIN cs_proc sf ON sf.oid=a.aggserialfn LEFT JOIN cs_proc df ON df.oid=a.aggdeserialfn
 LEFT JOIN cs_proc mtf ON mtf.oid=a.aggmtransfn LEFT JOIN cs_proc mif ON mif.oid=a.aggminvtransfn
 LEFT JOIN cs_proc mff ON mff.oid=a.aggmfinalfn LEFT JOIN cs_operator so ON so.oid=a.aggsortop
 JOIN cs_type tt ON tt.oid=a.aggtranstype LEFT JOIN cs_type mtt ON mtt.oid=a.aggmtranstype
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('type',n.nspname,t.typname,pg_get_userbyid(t.typowner),t.typlen,t.typbyval,
   t.typtype,t.typcategory,t.typispreferred,t.typisdefined,t.typdelim,t.typalign,t.typstorage,
   t.typnotnull,t.typtypmod,t.typndims,bt.ident,et.ident,at.ident,rr.ident,coll.ident,
   inp.ident,outp.ident,recv.ident,snd.ident,minp.ident,mout.ident,an.ident,sub.ident,
   CASE WHEN (SELECT ok FROM cs_safe_deparse) THEN pg_get_expr(t.typdefaultbin,0,false) END,t.typdefault,
   CASE WHEN t.typtype='d' AND (SELECT ok FROM cs_safe_deparse) THEN format_type(t.typbasetype,t.typtypmod) END)::text
 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace
 LEFT JOIN cs_type bt ON bt.oid=t.typbasetype LEFT JOIN cs_type et ON et.oid=t.typelem
 LEFT JOIN cs_type at ON at.oid=t.typarray LEFT JOIN cs_relation rr ON rr.oid=t.typrelid
 LEFT JOIN cs_collation coll ON coll.oid=t.typcollation
 LEFT JOIN cs_proc inp ON inp.oid=t.typinput LEFT JOIN cs_proc outp ON outp.oid=t.typoutput
 LEFT JOIN cs_proc recv ON recv.oid=t.typreceive LEFT JOIN cs_proc snd ON snd.oid=t.typsend
 LEFT JOIN cs_proc minp ON minp.oid=t.typmodin LEFT JOIN cs_proc mout ON mout.oid=t.typmodout
 LEFT JOIN cs_proc an ON an.oid=t.typanalyze LEFT JOIN cs_proc sub ON sub.oid=t.typsubscript
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('enum_value',tr.ident,row_number() OVER(PARTITION BY t.oid ORDER BY e.enumsortorder),e.enumlabel)::text
 FROM pg_enum e JOIN pg_type t ON t.oid=e.enumtypid JOIN pg_namespace n ON n.oid=t.typnamespace
 JOIN cs_type tr ON tr.oid=t.oid WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('composite_field',tr.ident,
   row_number() OVER(PARTITION BY t.oid ORDER BY a.attnum),a.attname,ft.ident,a.atttypmod,
   coll.ident,a.attnotnull,a.attidentity,a.attgenerated,a.attstorage,a.attcompression,a.attoptions)::text
 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace JOIN cs_type tr ON tr.oid=t.oid
 JOIN pg_attribute a ON a.attrelid=t.typrelid JOIN cs_type ft ON ft.oid=a.atttypid
 LEFT JOIN cs_collation coll ON coll.oid=a.attcollation
 WHERE ` + userNamespace + ` AND t.typtype='c' AND a.attnum>0 AND NOT a.attisdropped
 UNION ALL
 SELECT jsonb_build_array('range_type',tr.ident,sub.ident,multi.ident,coll.ident,opc.ident,can.ident,diff.ident)::text
 FROM pg_range r JOIN pg_type t ON t.oid=r.rngtypid JOIN pg_namespace n ON n.oid=t.typnamespace
 JOIN cs_type tr ON tr.oid=r.rngtypid JOIN cs_type sub ON sub.oid=r.rngsubtype
 JOIN cs_type multi ON multi.oid=r.rngmultitypid LEFT JOIN cs_collation coll ON coll.oid=r.rngcollation
 JOIN cs_class opc ON opc.oid=r.rngsubopc LEFT JOIN cs_proc can ON can.oid=r.rngcanonical
 LEFT JOIN cs_proc diff ON diff.oid=r.rngsubdiff WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('constraint_attributes',cr.ident,c.contype,c.conenforced,c.convalidated,
   c.condeferrable,c.condeferred,c.conislocal,c.coninhcount,c.connoinherit,c.conperiod,
   par.ident,ir.ident,fr.ident,c.confupdtype,c.confdeltype,c.confmatchtype,
   coalesce((SELECT jsonb_agg(jsonb_build_array(k.ord,a.attname) ORDER BY k.ord)
     FROM unnest(c.conkey) WITH ORDINALITY k(num,ord)
     LEFT JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.num),'[]'::jsonb),
   coalesce((SELECT jsonb_agg(jsonb_build_array(k.ord,a.attname) ORDER BY k.ord)
     FROM unnest(c.confkey) WITH ORDINALITY k(num,ord)
     LEFT JOIN pg_attribute a ON a.attrelid=c.confrelid AND a.attnum=k.num),'[]'::jsonb),
   coalesce((SELECT jsonb_agg(a.attname ORDER BY a.attname)
     FROM unnest(c.confdelsetcols) k(num) JOIN pg_attribute a ON a.attrelid=c.conrelid AND a.attnum=k.num),'[]'::jsonb),
   coalesce((SELECT jsonb_agg(o.ident ORDER BY k.ord) FROM unnest(c.conpfeqop) WITH ORDINALITY k(op,ord)
     JOIN cs_operator o ON o.oid=k.op),'[]'::jsonb),
   coalesce((SELECT jsonb_agg(o.ident ORDER BY k.ord) FROM unnest(c.conppeqop) WITH ORDINALITY k(op,ord)
     JOIN cs_operator o ON o.oid=k.op),'[]'::jsonb),
   coalesce((SELECT jsonb_agg(o.ident ORDER BY k.ord) FROM unnest(c.conffeqop) WITH ORDINALITY k(op,ord)
     JOIN cs_operator o ON o.oid=k.op),'[]'::jsonb),
   coalesce((SELECT jsonb_agg(o.ident ORDER BY k.ord) FROM unnest(c.conexclop) WITH ORDINALITY k(op,ord)
     JOIN cs_operator o ON o.oid=k.op),'[]'::jsonb),
   CASE WHEN (SELECT ok FROM cs_safe_deparse) THEN pg_get_constraintdef(c.oid,false) END)::text
 FROM pg_constraint c JOIN pg_namespace n ON n.oid=c.connamespace JOIN cs_constraint cr ON cr.oid=c.oid
 LEFT JOIN cs_constraint par ON par.oid=c.conparentid LEFT JOIN cs_relation ir ON ir.oid=c.conindid
 LEFT JOIN cs_relation fr ON fr.oid=c.confrelid WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('policy',rr.ident,p.polname,p.polpermissive,p.polcmd,
   coalesce((SELECT jsonb_agg(CASE WHEN roleid=0 THEN 'PUBLIC' ELSE pg_get_userbyid(roleid) END
     ORDER BY CASE WHEN roleid=0 THEN 'PUBLIC' ELSE pg_get_userbyid(roleid) END)
     FROM unnest(p.polroles) a(roleid)),'[]'::jsonb),
   CASE WHEN (SELECT ok FROM cs_safe_deparse) THEN pg_get_expr(p.polqual,p.polrelid,false) END,
   CASE WHEN (SELECT ok FROM cs_safe_deparse) THEN pg_get_expr(p.polwithcheck,p.polrelid,false) END)::text
 FROM pg_policy p JOIN pg_class r ON r.oid=p.polrelid JOIN pg_namespace n ON n.oid=r.relnamespace
 JOIN cs_relation rr ON rr.oid=r.oid WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('trigger_attributes',tr.ident,t.tgisinternal,t.tgenabled,t.tgtype,
   p.ident,c.ident,cr.ident,ir.ident,par.ident,t.tgdeferrable,t.tginitdeferred,t.tgnargs,
   encode(t.tgargs,'hex'),t.tgoldtable,t.tgnewtable,
   coalesce((SELECT jsonb_agg(a.attname ORDER BY a.attname) FROM unnest(t.tgattr::smallint[]) k(num)
     JOIN pg_attribute a ON a.attrelid=t.tgrelid AND a.attnum=k.num),'[]'::jsonb),
   CASE WHEN (SELECT ok FROM cs_safe_deparse) THEN
     CASE WHEN t.tgisinternal THEN
       overlay(pg_get_triggerdef(t.oid,false) PLACING quote_ident('__internal__')
         FROM length(CASE WHEN t.tgconstraint<>0 THEN 'CREATE CONSTRAINT TRIGGER ' ELSE 'CREATE TRIGGER ' END)+1
         FOR length(quote_ident(t.tgname)))
       ELSE pg_get_triggerdef(t.oid,false) END END)::text
 FROM pg_trigger t JOIN pg_class r ON r.oid=t.tgrelid JOIN pg_namespace n ON n.oid=r.relnamespace
 JOIN cs_trigger tr ON tr.oid=t.oid JOIN cs_proc p ON p.oid=t.tgfoid
 LEFT JOIN cs_constraint c ON c.oid=t.tgconstraint LEFT JOIN cs_relation cr ON cr.oid=t.tgconstrrelid
 LEFT JOIN cs_relation ir ON ir.oid=t.tgconstrindid LEFT JOIN cs_trigger par ON par.oid=t.tgparentid
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('collation',n.nspname,c.collname,c.collencoding,pg_get_userbyid(c.collowner),
   c.collprovider,c.collisdeterministic,c.collcollate,c.collctype,c.colllocale,c.collicurules,
   c.collversion,pg_collation_actual_version(c.oid))::text
 FROM pg_collation c JOIN pg_namespace n ON n.oid=c.collnamespace
 UNION ALL
 SELECT jsonb_build_array('operator',op.ident,pg_get_userbyid(o.oprowner),o.oprkind,o.oprcanmerge,o.oprcanhash,
   rt.ident,comm.ident,neg.ident,fn.ident,rest.ident,joinfn.ident)::text
 FROM pg_operator o JOIN pg_namespace n ON n.oid=o.oprnamespace JOIN cs_operator op ON op.oid=o.oid
 JOIN cs_type rt ON rt.oid=o.oprresult LEFT JOIN cs_operator comm ON comm.oid=o.oprcom
 LEFT JOIN cs_operator neg ON neg.oid=o.oprnegate LEFT JOIN cs_proc fn ON fn.oid=o.oprcode
 LEFT JOIN cs_proc rest ON rest.oid=o.oprrest LEFT JOIN cs_proc joinfn ON joinfn.oid=o.oprjoin
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('operator_family',f.ident,pg_get_userbyid(o.opfowner))::text
 FROM pg_opfamily o JOIN pg_namespace n ON n.oid=o.opfnamespace JOIN cs_family f ON f.oid=o.oid
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('operator_class',cl.ident,pg_get_userbyid(o.opcowner),f.ident,t.ident,
   o.opcdefault,k.ident)::text
 FROM pg_opclass o JOIN pg_namespace n ON n.oid=o.opcnamespace JOIN cs_class cl ON cl.oid=o.oid
 JOIN cs_family f ON f.oid=o.opcfamily JOIN cs_type t ON t.oid=o.opcintype
 LEFT JOIN cs_type k ON k.oid=o.opckeytype WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('operator_family_operator',f.ident,l.ident,r.ident,a.amopstrategy,a.amoppurpose,
   o.ident,m.amname,sf.ident)::text
 FROM pg_amop a JOIN pg_opfamily fam ON fam.oid=a.amopfamily JOIN pg_namespace n ON n.oid=fam.opfnamespace
 JOIN cs_family f ON f.oid=fam.oid JOIN cs_type l ON l.oid=a.amoplefttype
 JOIN cs_type r ON r.oid=a.amoprighttype JOIN cs_operator o ON o.oid=a.amopopr
 JOIN pg_am m ON m.oid=a.amopmethod LEFT JOIN cs_family sf ON sf.oid=a.amopsortfamily
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('operator_family_function',f.ident,l.ident,r.ident,a.amprocnum,p.ident)::text
 FROM pg_amproc a JOIN pg_opfamily fam ON fam.oid=a.amprocfamily JOIN pg_namespace n ON n.oid=fam.opfnamespace
 JOIN cs_family f ON f.oid=fam.oid JOIN cs_type l ON l.oid=a.amproclefttype
 JOIN cs_type r ON r.oid=a.amprocrighttype JOIN cs_proc p ON p.oid=a.amproc
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('extended_statistics',n.nspname,s.stxname,pg_get_userbyid(s.stxowner),rr.ident,
   s.stxstattarget,s.stxkind,
   coalesce((SELECT jsonb_agg(a.attname ORDER BY a.attname) FROM unnest(s.stxkeys::smallint[]) k(num)
     JOIN pg_attribute a ON a.attrelid=s.stxrelid AND a.attnum=k.num),'[]'::jsonb),
   CASE WHEN (SELECT ok FROM cs_safe_deparse) THEN pg_get_statisticsobjdef(s.oid) END)::text
 FROM pg_statistic_ext s JOIN pg_namespace n ON n.oid=s.stxnamespace JOIN cs_relation rr ON rr.oid=s.stxrelid
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('text_search_config',n.nspname,c.cfgname,pg_get_userbyid(c.cfgowner),
   pn.nspname,p.prsname)::text
 FROM pg_ts_config c JOIN pg_namespace n ON n.oid=c.cfgnamespace
 JOIN pg_ts_parser p ON p.oid=c.cfgparser JOIN pg_namespace pn ON pn.oid=p.prsnamespace
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('text_search_mapping',n.nspname,c.cfgname,m.maptokentype,m.mapseqno,dn.nspname,d.dictname)::text
 FROM pg_ts_config c JOIN pg_namespace n ON n.oid=c.cfgnamespace JOIN pg_ts_config_map m ON m.mapcfg=c.oid
 JOIN pg_ts_dict d ON d.oid=m.mapdict JOIN pg_namespace dn ON dn.oid=d.dictnamespace
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('text_search_dictionary',n.nspname,d.dictname,pg_get_userbyid(d.dictowner),
   tn.nspname,t.tmplname,d.dictinitoption)::text
 FROM pg_ts_dict d JOIN pg_namespace n ON n.oid=d.dictnamespace
 JOIN pg_ts_template t ON t.oid=d.dicttemplate JOIN pg_namespace tn ON tn.oid=t.tmplnamespace
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('text_search_parser',n.nspname,t.prsname,s.ident,k.ident,e.ident,h.ident,l.ident)::text
 FROM pg_ts_parser t JOIN pg_namespace n ON n.oid=t.prsnamespace JOIN cs_proc s ON s.oid=t.prsstart
 JOIN cs_proc k ON k.oid=t.prstoken JOIN cs_proc e ON e.oid=t.prsend
 LEFT JOIN cs_proc h ON h.oid=t.prsheadline JOIN cs_proc l ON l.oid=t.prslextype
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('text_search_template',n.nspname,t.tmplname,i.ident,l.ident)::text
 FROM pg_ts_template t JOIN pg_namespace n ON n.oid=t.tmplnamespace
 LEFT JOIN cs_proc i ON i.oid=t.tmplinit JOIN cs_proc l ON l.oid=t.tmpllexize
 WHERE ` + userNamespace + `
 UNION ALL
 SELECT jsonb_build_array('language',l.lanname,pg_get_userbyid(l.lanowner),l.lanispl,l.lanpltrusted,
   c.ident,i.ident,v.ident,
   coalesce((SELECT jsonb_agg(jsonb_build_array(pg_get_userbyid(a.grantor),
     CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,a.privilege_type,a.is_grantable)
     ORDER BY pg_get_userbyid(a.grantor),CASE WHEN a.grantee=0 THEN 'PUBLIC' ELSE pg_get_userbyid(a.grantee) END,
     a.privilege_type,a.is_grantable)
     FROM aclexplode(coalesce(l.lanacl,acldefault('l',l.lanowner))) a),'[]'::jsonb))::text
 FROM pg_language l LEFT JOIN cs_proc c ON c.oid=l.lanplcallfoid
 LEFT JOIN cs_proc i ON i.oid=l.laninline LEFT JOIN cs_proc v ON v.oid=l.lanvalidator
 UNION ALL
 SELECT jsonb_build_array('extension_configuration',e.extname,
   coalesce((SELECT jsonb_agg(jsonb_build_array(rr.ident,e.extcondition[a.ord]) ORDER BY rr.ident)
     FROM unnest(e.extconfig) WITH ORDINALITY a(rel,ord) JOIN cs_relation rr ON rr.oid=a.rel),'[]'::jsonb))::text
 FROM pg_extension e
 UNION ALL
 SELECT jsonb_build_array('extension_dependency',child.extname,parent.extname,d.deptype)::text
 FROM pg_depend d JOIN pg_extension child ON d.classid='pg_extension'::regclass AND child.oid=d.objid
 JOIN pg_extension parent ON d.refclassid='pg_extension'::regclass AND parent.oid=d.refobjid
 UNION ALL
 SELECT jsonb_build_array('extension_member',e.extname,a.ident)::text
 FROM pg_depend d JOIN pg_extension e ON e.oid=d.refobjid
 JOIN cs_address a ON a.classid=d.classid AND a.objid=d.objid AND a.objsubid=d.objsubid
 WHERE d.refclassid='pg_extension'::regclass AND d.deptype='e'
 UNION ALL
 SELECT jsonb_build_array('extension_object_dependency',a.ident,e.extname,d.deptype)::text
 FROM pg_depend d JOIN pg_extension e ON e.oid=d.refobjid
 JOIN cs_address a ON a.classid=d.classid AND a.objid=d.objid AND a.objsubid=d.objsubid
 WHERE d.refclassid='pg_extension'::regclass AND d.deptype='x'
 UNION ALL
 SELECT jsonb_build_array('comment',a.ident,d.description)::text
 FROM pg_description d JOIN cs_address a ON a.classid=d.classoid AND a.objid=d.objoid AND a.objsubid=d.objsubid
 UNION ALL
 SELECT jsonb_build_array('shared_comment',a.ident,d.description)::text
 FROM pg_shdescription d JOIN cs_address a ON a.classid=d.classoid AND a.objid=d.objoid AND a.objsubid=0
 UNION ALL
 SELECT jsonb_build_array('security_label',a.ident,s.provider,s.label)::text
 FROM pg_seclabel s JOIN cs_address a ON a.classid=s.classoid AND a.objid=s.objoid AND a.objsubid=s.objsubid
 UNION ALL
 SELECT jsonb_build_array('shared_security_label',a.ident,s.provider,s.label)::text
 FROM pg_shseclabel s JOIN cs_address a ON a.classid=s.classoid AND a.objid=s.objoid AND a.objsubid=0
`

// Los decompiladores pueden invocar typoutput/typmodout al representar constantes
// o modificadores. READ ONLY no impide efectos externos de un conversor C propio.
// Se comprueba también la implementación: CREATE OR REPLACE conserva el OID.
// Se admiten símbolos internos homónimos de PG18, sin biblioteca propia, y el
// alias nativo histórico txid_snapshot_out -> pg_snapshot_out sólo como salida
// del tipo PG18 txid_snapshot. La pareja consta en pg_type.dat y pg_proc.dat
// de REL_18_STABLE; no amplía la admisión a módulos ni conversores de aplicación.
// Cualquier otro alias requiere validación aislada específica.
// La autenticidad del binario/runtime la acredita la política de ejecución.
const hooksTipoNoSegurosSQL = `EXISTS (
 SELECT 1 FROM pg_type t
 JOIN pg_namespace tn ON tn.oid=t.typnamespace
 JOIN pg_proc p ON p.oid=t.typoutput OR p.oid=t.typmodout
 JOIN pg_namespace n ON n.oid=p.pronamespace
 JOIN pg_language l ON l.oid=p.prolang
 WHERE n.nspname<>'pg_catalog' OR p.oid>=16384 OR l.lanname<>'internal'
   OR p.probin IS NOT NULL OR (p.prosrc<>p.proname::text AND NOT (
     tn.nspname='pg_catalog' AND t.typname='txid_snapshot' AND t.oid=2970
     AND p.oid=t.typoutput AND p.oid=2940
     AND p.proname='txid_snapshot_out' AND p.prosrc='pg_snapshot_out'
   ))
)`

// Estos límites bloquean la validez de la copia; no se omiten componentes.
// Las estadísticas calculadas por ANALYZE son estado derivado; se compara su
// definición persistente, no pg_statistic_ext_data ni muestras de planificación.
var comprobacionesAvanzadas = []comprobacion{
	{"conversion_tipo_catalogo_no_aislada", `SELECT ` + hooksTipoNoSegurosSQL},
	{"tipo_incompleto_no_comprobable", `SELECT EXISTS(SELECT 1 FROM pg_type t JOIN pg_namespace n ON n.oid=t.typnamespace WHERE ` + userNamespace + ` AND NOT t.typisdefined)`},
	{"conversion_codificacion_no_comprobable", `SELECT EXISTS(SELECT 1 FROM pg_conversion c JOIN pg_namespace n ON n.oid=c.connamespace WHERE ` + userNamespace + ` OR c.oid>=16384)`},
	{"transformacion_tipo_no_comprobable", `SELECT EXISTS(SELECT 1 FROM pg_transform)`},
	{"metodo_acceso_propio_no_comprobable", `SELECT EXISTS(SELECT 1 FROM pg_am WHERE oid>=16384)`},
	{"catalogo_avanzado_sistema_no_comprobable", `SELECT EXISTS(
 SELECT 1 FROM (
   SELECT t.oid,t.typnamespace AS ns FROM pg_type t
   UNION ALL SELECT o.oid,o.oprnamespace FROM pg_operator o
   UNION ALL SELECT o.oid,o.opcnamespace FROM pg_opclass o
   UNION ALL SELECT o.oid,o.opfnamespace FROM pg_opfamily o
   UNION ALL SELECT s.oid,s.stxnamespace FROM pg_statistic_ext s
   UNION ALL SELECT t.oid,t.cfgnamespace FROM pg_ts_config t
   UNION ALL SELECT t.oid,t.dictnamespace FROM pg_ts_dict t
   UNION ALL SELECT t.oid,t.prsnamespace FROM pg_ts_parser t
   UNION ALL SELECT t.oid,t.tmplnamespace FROM pg_ts_template t
 ) o JOIN pg_namespace n ON n.oid=o.ns
 WHERE o.oid>=16384 AND (n.nspname ~ '^pg_' OR n.nspname='information_schema')
 )`},
	{"metadatos_objeto_grande_sin_identidad_semantica", `SELECT EXISTS(
 SELECT 1 FROM pg_description WHERE classoid IN('pg_largeobject'::regclass,'pg_largeobject_metadata'::regclass)
 ) OR EXISTS(SELECT 1 FROM pg_seclabel WHERE classoid IN('pg_largeobject'::regclass,'pg_largeobject_metadata'::regclass))`},
	{"metadatos_almacen_interno_no_comprobables", `SELECT EXISTS(
 SELECT 1 FROM pg_class r JOIN pg_namespace n ON n.oid=r.relnamespace
 WHERE n.nspname ~ '^pg_toast' AND (
   EXISTS(SELECT 1 FROM pg_description d WHERE d.classoid='pg_class'::regclass AND d.objoid=r.oid)
   OR EXISTS(SELECT 1 FROM pg_seclabel s WHERE s.classoid='pg_class'::regclass AND s.objoid=r.oid)
 ))`},
	{"objetos_extension_sin_captura_semantica", `SELECT EXISTS(
 SELECT 1 FROM pg_depend d
 WHERE d.refclassid='pg_extension'::regclass AND d.deptype IN ('e','x')
 AND d.classid NOT IN (
   'pg_namespace'::regclass,'pg_class'::regclass,'pg_proc'::regclass,'pg_type'::regclass,
   'pg_constraint'::regclass,'pg_trigger'::regclass,'pg_rewrite'::regclass,'pg_policy'::regclass,
   'pg_collation'::regclass,'pg_operator'::regclass,'pg_opclass'::regclass,'pg_opfamily'::regclass,
   'pg_amop'::regclass,'pg_amproc'::regclass,'pg_statistic_ext'::regclass,
   'pg_ts_config'::regclass,'pg_ts_dict'::regclass,'pg_ts_parser'::regclass,'pg_ts_template'::regclass,
   'pg_language'::regclass
 ))`},
}
