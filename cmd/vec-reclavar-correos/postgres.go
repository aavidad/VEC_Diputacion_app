package main

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

func procesar(ctx context.Context, conn *pgx.Conn, modo string, p plan, o objetivo, vieja, propia *fuenteEfimera) (informe, error) {
	i := informe{Persona: o.Persona, Preimagen: o.Preimagen, Modo: modo, Recuentos: map[string]int{}}
	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return i, errOperacion
	}
	defer tx.Rollback(context.Background())
	var permitido bool
	if err := tx.QueryRow(ctx, `SELECT session_user=current_user AND rolsuper AND current_database()=$1
 FROM pg_catalog.pg_roles WHERE rolname=session_user`, p.Base).Scan(&permitido); err != nil || !permitido {
		return i, errOperacion
	}
	// Una ejecución parcial consulta primero el recibo de cada persona;
	// no descifra de nuevo el resultado ya convertido ni genera otro nonce.
	if modo != "inventario" {
		var recibo []byte
		if err := tx.QueryRow(ctx, `SELECT vec_usuarios_correos_reclaveado.recuperar_reclaveado_persona_v1($1,$2,$3,$4)`, o.Persona, o.Preimagen, p.Lote, p.Aprobacion).Scan(&recibo); err != nil {
			return i, errOperacion
		}
		if len(recibo) > 0 && string(recibo) != "null" {
			if !json.Valid(recibo) {
				return i, errOperacion
			}
			i.Recibo = recibo
			return i, nil
		}
	}
	var b []byte
	if err := tx.QueryRow(ctx, `SELECT vec_usuarios_correos_reclaveado.snapshot_reclaveado_persona_v1($1)`, o.Persona).Scan(&b); err != nil {
		return i, errOperacion
	}
	var s instantanea
	if json.Unmarshal(b, &s) != nil || s.Persona != o.Persona || !digestValido.MatchString(s.Preimagen) || len(s.Filas) != 7 {
		clear(b)
		return i, errOperacion
	}
	clear(b)
	i.Preimagen = s.Preimagen
	if o.Preimagen != "" && s.Preimagen != o.Preimagen {
		return i, errOperacion
	}
	for _, nombre := range []string{"correos_conjunto", "correos_direccion", "correos_desafio", "correos_intento_fallido", "correos_historia", "correos_recibo", "correos_envio"} {
		var filas []json.RawMessage
		if json.Unmarshal(s.Filas[nombre], &filas) != nil || len(filas) > 10000 {
			return i, errOperacion
		}
		i.Recuentos[nombre] = len(filas)
	}
	if i.Recuentos["correos_conjunto"] != 1 {
		return i, errOperacion
	}
	var direcciones []direccionDB
	if json.Unmarshal(s.Filas["correos_direccion"], &direcciones) != nil {
		return i, errOperacion
	}
	m, err := convertir(ctx, o.Persona, direcciones, vieja, propia)
	if err != nil {
		return i, errCripto
	}
	if modo == "inventario" {
		return i, nil
	}
	material, err := json.Marshal(m)
	if err != nil {
		return i, errOperacion
	}
	defer clear(material)
	if err := tx.QueryRow(ctx, `SELECT vec_usuarios_correos_reclaveado.aplicar_reclaveado_persona_v1($1,$2,$3::jsonb,$4,$5)`, o.Persona, o.Preimagen, string(material), p.Lote, p.Aprobacion).Scan(&b); err != nil || !json.Valid(b) {
		return i, errOperacion
	}
	i.Recibo = append([]byte(nil), b...)
	if modo == "aplicar" {
		if err := tx.Commit(ctx); err != nil {
			return i, errOperacion
		}
	}
	return i, nil
}
