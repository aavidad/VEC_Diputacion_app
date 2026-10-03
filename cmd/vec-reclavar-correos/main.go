// vec-reclavar-correos es una operación offline de transición, fuera del
// servidor. Nunca envía correos ni suministra claves antiguas al externo.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
)

var errEntrada = errors.New("reclaveado_entrada_invalida")
var errCripto = errors.New("reclaveado_cripto_incompatible")
var errOperacion = errors.New("reclaveado_operacion_no_confirmada")

type instantanea struct {
	Persona   string                     `json:"persona_ref"`
	Preimagen string                     `json:"preimagen_sha256"`
	Filas     map[string]json.RawMessage `json:"filas"`
}

type informe struct {
	Persona   string          `json:"persona_ref"`
	Preimagen string          `json:"preimagen_sha256"`
	Recuentos map[string]int  `json:"recuentos"`
	Modo      string          `json:"modo"`
	Recibo    json.RawMessage `json:"recibo,omitempty"`
}

type opciones struct {
	modo                                 string
	planFD, dsnFD, anteriorFD, externaFD int
	caFD                                 int
}

func main() {
	var o opciones
	fs := flag.NewFlagSet("vec-reclavar-correos", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&o.modo, "modo", "inventario", "")
	fs.IntVar(&o.planFD, "plan-fd", -1, "")
	fs.IntVar(&o.dsnFD, "dsn-fd", -1, "")
	fs.IntVar(&o.anteriorFD, "anterior-fd", -1, "")
	fs.IntVar(&o.externaFD, "externa-fd", -1, "")
	fs.IntVar(&o.caFD, "ca-fd", -1, "")
	if fs.Parse(os.Args[1:]) != nil || fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, errEntrada)
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	err := ejecutar(ctx, o, os.Stdout)
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func ejecutar(ctx context.Context, o opciones, salida io.Writer) error {
	if o.modo != "inventario" && o.modo != "ensayo" && o.modo != "aplicar" {
		return errEntrada
	}
	fds := map[int]bool{}
	for _, fd := range []int{o.planFD, o.dsnFD, o.anteriorFD, o.externaFD} {
		if fd < 3 || fds[fd] {
			return errEntrada
		}
		fds[fd] = true
	}
	b, err := leerDescriptor(o.planFD, 2<<20)
	if err != nil {
		return errEntrada
	}
	p, err := cargarPlan(b, o.modo)
	clear(b)
	if err != nil {
		return errEntrada
	}
	b, err = leerDescriptor(o.dsnFD, 8192)
	if err != nil {
		return errEntrada
	}
	var ca []byte
	if p.Conexion.SSLMode == "verify-full" {
		if fds[o.caFD] {
			clear(b)
			return errEntrada
		}
		ca, err = leerDescriptor(o.caFD, 1<<20)
		if err != nil {
			clear(b)
			return errEntrada
		}
	}
	cfg, err := configuracionConexion(b, p, ca)
	clear(b)
	clear(ca)
	if err != nil {
		return errEntrada
	}
	cfg.ConnectTimeout = 5 * time.Second
	var anterior, externa [32]byte
	defer clear(anterior[:])
	defer clear(externa[:])
	for i, fd := range []int{o.anteriorFD, o.externaFD} {
		b, err = leerDescriptor(fd, 32)
		if err != nil || len(b) != 32 {
			clear(b)
			return errEntrada
		}
		if i == 0 {
			copy(anterior[:], b)
		} else {
			copy(externa[:], b)
		}
		clear(b)
	}
	if anterior == externa || anterior == ([32]byte{}) || externa == ([32]byte{}) {
		return errCripto
	}
	vieja, propia := fuenteDesdeSemilla(anterior, false), fuenteDesdeSemilla(externa, true)
	defer vieja.borrar()
	defer propia.borrar()
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return errOperacion
	}
	defer conn.Close(context.Background())
	var sistema string
	if err := conn.QueryRow(ctx, `SELECT system_identifier::text FROM pg_catalog.pg_control_system()`).Scan(&sistema); err != nil || sistema != p.Sistema {
		return errOperacion
	}
	for _, objetivo := range p.Personas {
		i, err := procesar(ctx, conn, o.modo, p, objetivo, vieja, propia)
		if err != nil {
			return err
		}
		if json.NewEncoder(salida).Encode(i) != nil {
			return errOperacion
		}
	}
	return nil
}
