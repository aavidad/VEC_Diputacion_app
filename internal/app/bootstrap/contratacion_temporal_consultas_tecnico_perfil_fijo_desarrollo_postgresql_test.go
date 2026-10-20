package bootstrap

import (
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/adapters/httpinterno"
)

// Usa la copia desechable de PostgreSQL 18 compartida por las pruebas de
// perfiles. Comprueba la historia real que una lectura no puede regenerar.
func TestConsultasTecnicoPerfilFijoNoRegeneraNiRevivePostgreSQL(t *testing.T) {
	ctx, gobierno, admin := poolesPerfilDinamicoRRHHPostgreSQLPrueba(t)
	s := soporteRRHHPostgreSQLPrueba(t, ctx, gobierno)
	if err := componerPerfilFijoConsultasTecnicoCTDesarrollo(s, time.Now().UTC().Truncate(time.Microsecond)); err != nil {
		t.Fatal(err)
	}
	p := perfilFijoConsultasTecnicoCTDesarrollo(s)
	if p == nil || s.perfilFijoParaRuta(httpinterno.RutaConsultaCuadroRRHH) != p ||
		s.perfilFijoParaRuta(httpinterno.RutaConsultaDetalleRRHH) != p {
		t.Fatal("ambas lecturas no eligieron el mismo perfil")
	}
	if err := asegurarPerfilesFijosCTDesarrollo(ctx, gobierno, s, aprobacionProvisionPerfilesRRHHDesarrollo{}, p); err != nil {
		t.Fatal(err)
	}
	inicial := historiaPerfilPostgreSQLPrueba(t, ctx, admin, p.perfilRef())
	if inicial.versiones != 1 || inicial.acto != actoAsignacionPerfilFijoCTDesarrollo {
		t.Fatalf("publicación inicial incorrecta: %+v", inicial)
	}
	for _, ruta := range []string{httpinterno.RutaConsultaCuadroRRHH, httpinterno.RutaConsultaDetalleRRHH} {
		if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, s.perfilFijoParaRuta(ruta)); !ok {
			t.Fatalf("perfil publicado no consumible en %s", ruta)
		}
	}
	if estado, err := asegurarPerfilFijoCTDesarrollo(ctx, gobierno, s, p, aprobacionProvisionPerfilesRRHHDesarrollo{},
		preimagenPropiaPerfilFijoCTDesarrollo(p, actoAsignacionPerfilFijoCTDesarrollo)); err != nil || estado != perfilFijoVigente ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, p.perfilRef()) != inicial {
		t.Fatalf("la lectura o el reinicio regeneró la asignación: %s %v", estado, err)
	}
	revocarPerfilFijoPrueba(t, ctx, gobierno, p, false)
	revocada := historiaPerfilPostgreSQLPrueba(t, ctx, admin, p.perfilRef())
	if _, ok := s.consumirPerfilFijoCTDesarrollo(ctx, p); ok {
		t.Fatal("la revocación siguió consumible")
	}
	if estado, err := asegurarPerfilFijoCTDesarrollo(ctx, gobierno, s, p, aprobacionProvisionPerfilesRRHHDesarrollo{},
		preimagenPropiaPerfilFijoCTDesarrollo(p, actoAsignacionPerfilFijoCTDesarrollo)); err != nil || estado != perfilFijoPendienteProvision ||
		historiaPerfilPostgreSQLPrueba(t, ctx, admin, p.perfilRef()) != revocada {
		t.Fatalf("el reinicio reactivó el perfil revocado: %s %v", estado, err)
	}
}
