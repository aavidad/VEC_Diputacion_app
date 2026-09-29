package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/domain"
)

func TestResolutorUsuariosExternoConfirmaSoloCanonSinOtrosPerfiles(t *testing.T) {
	solicitud, fila := solicitudYFilaContextoActorV2(t)
	actor, err := domain.RehidratarContextoActorVinculadoV2(fila.valores[2].([]byte))
	if err != nil {
		t.Fatal(err)
	}
	actor.Instantanea.Vinculos = nil
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	manifiesto, err := domain.RehidratarManifiestoProcedenciaContextoActorV1(fila.valores[4].([]byte))
	if err != nil {
		t.Fatal(err)
	}
	manifiesto.Vinculos = []domain.ProcedenciaVinculoReferenciaContextoActorV1{}
	procedencia, err := manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	h1, h2 := sha256.Sum256(canon), sha256.Sum256(procedencia)
	fila.valores[2], fila.valores[3] = canon, hex.EncodeToString(h1[:])
	fila.valores[4], fila.valores[5] = procedencia, hex.EncodeToString(h2[:])
	primera := &txContextoActorDoble{filas: []pgx.Row{fila}}
	pool := &poolContextoActorDoble{transacciones: []*txContextoActorDoble{primera}}
	r, _ := nuevoResolutorRegistroContextoActorExternoPostgreSQLV1(pool,
		bytes.NewReader(bytes.Repeat([]byte{0x55}, bytesAleatoriosReferenciaContextoActorV2)))
	r.usuarios = true
	confirmacion, err := r.ResolverYRegistrarContextoActorV2(context.Background(), solicitud)
	if err != nil || confirmacion.ValidarParaProductiva(solicitud) != nil {
		t.Fatalf("canon Usuarios: %v", err)
	}
	if len(primera.consultas) != 1 || !strings.Contains(primera.consultas[0], "resolver_contexto_usuarios_externo_v1") ||
		strings.Contains(primera.consultas[0], "resolver_contexto_candidato_externo_v1") || len(primera.argumentos[0]) != 5 {
		t.Fatal("Usuarios consultó otra población o cambió la petición")
	}
}

func TestUsuariosExternoDeniegaCanonCandidatoValido(t *testing.T) {
	solicitud, fila := solicitudYFilaContextoActorV2(t)
	respuesta := respuestaContextoActorPostgreSQL{operacionRef: fila.valores[0].(string), reciboRef: fila.valores[1].(string),
		representacion: fila.valores[2].([]byte), huella: fila.valores[3].(string), manifiesto: fila.valores[4].([]byte),
		huellaManifiesto: fila.valores[5].(string), autoridadEfectiva: fila.valores[6].(string), resueltoEn: fila.valores[7].(time.Time)}
	if _, err := confirmarRespuestaContextoActor(solicitud, respuesta); err != nil {
		t.Fatal(err)
	}
	if _, err := confirmarRespuestaUsuariosExterno(solicitud, respuesta); err == nil {
		t.Fatal("Usuarios aceptó vínculo candidato")
	}
}
