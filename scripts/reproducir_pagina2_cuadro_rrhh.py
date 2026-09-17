import http.client, json, ssl, sys
M="/root/.local/state/vec-diputacion/desarrollo-fiscalizacion-20260904/mtls"
ctx=ssl.create_default_context(); ctx.check_hostname=False; ctx.verify_mode=ssl.CERT_NONE
ctx.load_cert_chain(M+"/cliente.crt", M+"/cliente.key")
H={"Content-Type":"application/json","Accept":"application/json"}
U="/api/vec/contratacion-temporal/cuadro/consultas"
def pide(c, body):
    c.request("POST", U, body=json.dumps(body), headers=H); r=c.getresponse(); d=r.read(); return r.status, (json.loads(d) if d else {})
c=http.client.HTTPSConnection("127.0.0.1", 8443, context=ctx)
s1,d1=pide(c, {"filtros":{"texto":""},"paginacion":{"limite":10}})
cur=d1.get("data",{}).get("cursor_siguiente","")
print("misma conexión: p1", s1, "hay_mas", d1.get("data",{}).get("hay_mas"))
s2,d2=pide(c, {"filtros":{"texto":""},"paginacion":{"limite":10,"cursor":cur}})
print("misma conexión: p2", s2, "expedientes", len(d2.get("data",{}).get("expedientes",[])), "hay_mas", d2.get("data",{}).get("hay_mas"), d2.get("error",{}).get("codigo",""))
if s2==200:
    r1={e["expediente_ref"] for e in d1["data"]["expedientes"]}; r2={e["expediente_ref"] for e in d2["data"]["expedientes"]}
    print("solapados:", len(r1&r2))
    cur2=d2["data"].get("cursor_siguiente","")
    if cur2:
        s3,d3=pide(c, {"filtros":{"texto":""},"paginacion":{"limite":10,"cursor":cur2}})
        print("misma conexión: p3", s3, "expedientes", len(d3.get("data",{}).get("expedientes",[])), "hay_mas", d3.get("data",{}).get("hay_mas"))
c.close()
# conexión nueva con el cursor de p1 (ya consumido) y con uno fresco
c2=http.client.HTTPSConnection("127.0.0.1", 8443, context=ctx)
s1b,d1b=pide(c2, {"filtros":{"texto":""},"paginacion":{"limite":10}}); c2.close()
c3=http.client.HTTPSConnection("127.0.0.1", 8443, context=ctx)
s2b,d2b=pide(c3, {"filtros":{"texto":""},"paginacion":{"limite":10,"cursor":d1b["data"]["cursor_siguiente"]}})
print("conexión distinta: p2 con cursor fresco de otra conexión ->", s2b, d2b.get("error",{}).get("codigo",""))
