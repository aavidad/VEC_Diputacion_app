package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

func TestRegistroPeticionesSesionConsumeYRevalidaAtomico(t *testing.T) {
	solicitud := solicitudPeticionSesionPrueba()
	vigencia := time.Date(2026, time.September, 20, 12, 5, 0, 0, time.UTC)
	tx := &transaccionDoble{filas: [][]any{{
		referencia("ses_", "s"), referencia("aut_", "a"), referencia("ase_", "e"),
		referencia("cta_", "c"), referencia("cse_", "x"), "1", vigencia,
	}}}
	pool := &iniciadorDoble{transacciones: []*transaccionDoble{tx}}
	registro, err := nuevoRegistroPeticionesSesionPostgreSQL(pool, &iniciadorDoble{})
	if err != nil {
		t.Fatal("crear registro")
	}
	confirmacion, err := registro.ConsumirYRevalidar(context.Background(), solicitud)
	if err != nil || !confirmacionPeticionSesionValida(confirmacion) || tx.commits != 1 {
		t.Fatalf("consumo válido rechazado: %+v %v", confirmacion, err)
	}
	if len(tx.argumentos) != 1 || len(tx.argumentos[0]) != 13 ||
		!strings.Contains(tx.consultas[0], "consumir_asercion_peticion_sesion_v1") ||
		tx.argumentos[0][5] != solicitud.NonceSHA256 || tx.argumentos[0][9] != solicitud.Destino {
		t.Fatalf("contrato SQL alterado: %#v", tx.argumentos)
	}
	comprobarSerializable(t, pool.opciones)
}

func TestRegistroPeticionesSesionFallaCerradoAntesDeSQL(t *testing.T) {
	cases := []struct {
		nombre  string
		alterar func(*httpseguridad.SolicitudConsumoPeticionSesion)
	}{
		{"nonce nulo", func(s *httpseguridad.SolicitudConsumoPeticionSesion) { s.NonceSHA256 = strings.Repeat("0", 64) }},
		{"sesion nula", func(s *httpseguridad.SolicitudConsumoPeticionSesion) { s.SesionIDHMAC = [32]byte{} }},
		{"version fuera de bigint", func(s *httpseguridad.SolicitudConsumoPeticionSesion) { s.ClaveHMACVersion = uint64(1 << 63) }},
		{"superficie publica", func(s *httpseguridad.SolicitudConsumoPeticionSesion) {
			s.Superficie = httpseguridad.SuperficiePublicaAnonima
		}},
		{"metodo alterado", func(s *httpseguridad.SolicitudConsumoPeticionSesion) { s.Metodo = "TRACE" }},
	}
	for _, tc := range cases {
		t.Run(tc.nombre, func(t *testing.T) {
			s := solicitudPeticionSesionPrueba()
			tc.alterar(&s)
			pool := &iniciadorDoble{}
			registro, _ := nuevoRegistroPeticionesSesionPostgreSQL(pool, &iniciadorDoble{})
			_, err := registro.ConsumirYRevalidar(context.Background(), s)
			if !errors.Is(err, httpseguridad.ErrRegistroPeticionesAusente) || pool.llamadas != 0 {
				t.Fatal("entrada inválida alcanzó PostgreSQL")
			}
		})
	}
}

func TestRegistroPeticionesSesionRevocaConRevisionExacta(t *testing.T) {
	tx := &transaccionDoble{filas: [][]any{{"2"}}}
	registro, _ := nuevoRegistroPeticionesSesionPostgreSQL(&iniciadorDoble{}, &iniciadorDoble{transacciones: []*transaccionDoble{tx}})
	err := registro.RevocarSesionExacta(context.Background(), httpseguridad.SolicitudRevocarSesionExacta{
		SesionRef: referencia("ses_", "s"), ControlSesionRef: referencia("cse_", "c"),
		ControlSesionRevision: 1,
	})
	if err != nil || tx.commits != 1 || len(tx.argumentos) != 1 ||
		tx.argumentos[0][2] != "1" || !strings.Contains(tx.consultas[0], "revocar_sesion_v1") {
		t.Fatalf("revocación exacta inválida: %v %#v", err, tx.argumentos)
	}
}

func solicitudPeticionSesionPrueba() httpseguridad.SolicitudConsumoPeticionSesion {
	ahora := time.Date(2026, time.September, 20, 12, 0, 0, 0, time.UTC)
	return httpseguridad.SolicitudConsumoPeticionSesion{
		EsquemaHMAC: "vec.identidad.hmac-sha256.v1", DominioHMACRef: referencia("idh_", "d"),
		ClaveHMACID: "clave-prueba", ClaveHMACVersion: 1, SesionIDHMAC: [32]byte{1},
		NonceSHA256: strings.Repeat("a", 64), CanalVinculadoRef: "canal-prueba",
		Superficie: httpseguridad.SuperficieExternaPersonal, Metodo: "POST", Destino: "/area-personal",
		CuerpoSHA256: strings.Repeat("b", 64), EmitidaEn: ahora, ExpiraEn: ahora.Add(time.Minute),
	}
}
