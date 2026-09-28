package main

import (
 "context"
 "os"
 "testing"

 "github.com/jackc/pgx/v5"
)

func TestProvisionPreflightPG18Bundle(t *testing.T) {
 dsn:=os.Getenv("VEC_CT_PLANTILLAS_MIGRADOR_DATABASE_URL")
 if dsn=="" { t.Skip("solo runner PG18") }
 ctx:=context.Background()
 c,err:=pgx.Connect(ctx,dsn)
 if err!=nil { t.Fatal(err) }
 defer c.Close(ctx)
 tx,err:=c.BeginTx(ctx,pgx.TxOptions{IsoLevel:pgx.Serializable})
 if err!=nil { t.Fatal(err) }
 defer tx.Rollback(ctx)
 var ok bool
 if err:=tx.QueryRow(ctx,sqlPreflight).Scan(&ok);err!=nil { t.Fatal(err) }
 if !ok {
  var memberships string
  err:=tx.QueryRow(ctx,`SELECT string_agg(roleid::regrole::text||':'||inherit_option||':'||set_option,',') FROM pg_auth_members WHERE member=session_user::regrole`).Scan(&memberships)
  t.Logf("membresías directas=%s, error=%v",memberships,err)
  t.Fatal("preflight nominal de provisión rechazado")
 }
}
