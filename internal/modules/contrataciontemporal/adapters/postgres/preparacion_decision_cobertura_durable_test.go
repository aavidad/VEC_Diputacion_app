package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
)

func TestPreparadorDecisionCoberturaDurableRechazaDependenciaNula(
	t *testing.T,
) {
	var pool *iniciadorPreparacionPrueba
	if _, err := nuevoPreparadorOperacionDecisionCoberturaDurablePostgreSQL(
		pool,
	); !errors.Is(
		err,
		errPersistenciaDecisionCoberturaDurableNoDisponible,
	) {
		t.Fatalf("dependencia nula aceptada: %v", err)
	}
}

func TestConsultaSQLPreparacionDecisionCoberturaDurableEsCerrada(
	t *testing.T,
) {
	tx := &transaccionPreparacionPrueba{
		fila: filaPreparacionPrueba{valores: []any{
			"requiere_validacion",
			"hmac-sha256:vec.contratacion-temporal." +
				"cobertura-decision.ambito/v1:" +
				strings.Repeat("a", 64),
			"hmac-sha256:vec.contratacion-temporal." +
				"cobertura-decision.semantica/v1:" +
				strings.Repeat("b", 64),
			"",
		}},
	}
	fila, err := consultarPreparacionDecisionCoberturaDurable(
		context.Background(),
		tx,
		[]byte(`{"esquema":"prueba"}`),
		nil,
	)
	if err != nil ||
		fila.resultado != "requiere_validacion" ||
		!strings.Contains(
			tx.consulta,
			funcionPrepararDecisionCoberturaDurable,
		) ||
		strings.Contains(tx.consulta, "INSERT ") ||
		strings.Contains(tx.consulta, "UPDATE ") {
		t.Fatalf("consulta SQL no cerrada: %#v / %v / %s", fila, err, tx.consulta)
	}
}

func TestDecodificadorCargaDecisionCoberturaDurableFallaCerrado(
	t *testing.T,
) {
	type cargaPrueba struct {
		Valor string `json:"valor"`
	}
	casos := map[string][]byte{
		"desconocido": []byte(`{"valor":"x","otro":true}`),
		"trailing":    []byte(`{"valor":"x"} {}`),
		"vacío":       nil,
		"excesivo":    make([]byte, maximoBytesCargaDecisionCoberturaDurable+1),
	}
	for nombre, contenido := range casos {
		t.Run(nombre, func(t *testing.T) {
			var carga cargaPrueba
			if err := decodificarCargaDecisionCoberturaDurable(
				contenido,
				&carga,
			); !errors.Is(
				err,
				errPersistenciaDecisionCoberturaDurableNoDisponible,
			) || errors.Is(
				err,
				cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida,
			) {
				t.Fatalf("carga hostil aceptada: %v", err)
			}
		})
	}
	var carga cargaPrueba
	if err := decodificarCargaDecisionCoberturaDurable(
		[]byte(`{"valor":"acotado"}`),
		&carga,
	); err != nil || carga.Valor != "acotado" {
		t.Fatalf("carga válida rechazada: %#v / %v", carga, err)
	}
}

func TestContenidoDurableAdulteradoNoSeClasificaComoColisionUsuario(
	t *testing.T,
) {
	err := decodificarCargaDecisionCoberturaDurable(
		[]byte(`{"campo_ajeno":"adulterado"}`),
		&cargaPropietariaDecisionCoberturaDurableV1{},
	)
	normalizado := normalizarErrorDecisionCoberturaDurable(
		context.Background(),
		err,
	)
	if !errors.Is(
		normalizado,
		errPersistenciaDecisionCoberturaDurableNoDisponible,
	) || errors.Is(
		normalizado,
		cobertura.ErrOperacionDecisionCoberturaIdempotenteInvalida,
	) || errors.Is(
		normalizado,
		cobertura.ErrClaveOperacionDecisionCoberturaUsada,
	) {
		t.Fatalf("corrupción durable mal clasificada: %v", normalizado)
	}
}

func TestPreparadorDecisionCoberturaDurableConfiguraSerializable(
	t *testing.T,
) {
	tx := &transaccionPreparacionPrueba{
		fila: filaPreparacionPrueba{err: pgx.ErrNoRows},
	}
	iniciador := &iniciadorPreparacionPrueba{tx: tx}
	preparador, err :=
		nuevoPreparadorOperacionDecisionCoberturaDurablePostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	transaccion, err := preparador.iniciar(context.Background(), pgx.ReadOnly)
	if err != nil {
		t.Fatal(err)
	}
	_ = transaccion.Rollback(context.Background())
	if iniciador.opciones.IsoLevel != pgx.Serializable ||
		iniciador.opciones.AccessMode != pgx.ReadOnly ||
		!tx.configurada {
		t.Fatalf(
			"transacción insegura: %#v configurada=%t",
			iniciador.opciones,
			tx.configurada,
		)
	}
}
