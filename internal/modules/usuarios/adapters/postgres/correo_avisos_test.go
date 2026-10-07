package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/modules/usuarios/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func v3AvisosPGPrueba(t *testing.T, audiencia string) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_avisos", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), ports.AccionV3CorreoAvisosLlamamiento, "bolsa:1", strings.Repeat("e", 64), audiencia, ahora, ahora.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	v3, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), []byte("contexto"), 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return v3
}

func registroAvisosPGPrueba(tx *txCorreoPGPrueba) *RegistroCorreoAvisosPostgreSQL {
	return &RegistroCorreoAvisosPostgreSQL{iniciar: func(context.Context) (transaccionCorreos, error) { return tx, nil }}
}

func respuestaAvisosPG(t *testing.T, v map[string]any) filaCorreoPGPrueba {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return filaCorreoPGPrueba{valor: b}
}

var materialAvisosPG = []byte(`{"esquema":"vec.usuarios.correo-avisos-llamamiento.v1"}`)

func TestLeerCorreoAvisosDevuelveSobreYConfirma(t *testing.T) {
	persona := "per_" + strings.Repeat("p", 24)
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, respuestaAvisosPG(t, map[string]any{
		"encontrado": true, "auditoria_ref": "aud_1", "persona_ref": persona, "correo_ref": refCorreoPG, "sobre": sobrePG(),
	})}}
	l, err := registroAvisosPGPrueba(tx).LeerCorreoActivoAvisos(context.Background(), materialAvisosPG, v3AvisosPGPrueba(t, ports.AudienciaCorreoAvisosLlamamientoInterna))
	if err != nil || !l.Encontrado || l.PersonaRef != persona || l.CorreoRef != refCorreoPG || l.Sobre.CorreoRef != refCorreoPG || tx.commits != 1 {
		t.Fatalf("%+v %v commits=%d", l, err, tx.commits)
	}
	ultima := tx.llamadas[len(tx.llamadas)-1]
	if !strings.Contains(ultima.sql, "correo_activo_avisos_llamamiento_v1") || ultima.args[0] != string(materialAvisosPG) {
		t.Fatalf("llamada SQL: %s", ultima.sql)
	}
}

func TestLeerCorreoAvisosSinActivo(t *testing.T) {
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, respuestaAvisosPG(t, map[string]any{"encontrado": false, "auditoria_ref": "aud_1"})}}
	l, err := registroAvisosPGPrueba(tx).LeerCorreoActivoAvisos(context.Background(), materialAvisosPG, v3AvisosPGPrueba(t, ports.AudienciaCorreoAvisosLlamamientoInterna))
	if err != nil || l.Encontrado || l.PersonaRef != "" || tx.commits != 1 {
		t.Fatalf("%+v %v", l, err)
	}
}

func TestLeerCorreoAvisosRechazaRespuestasIncoherentes(t *testing.T) {
	persona := "per_" + strings.Repeat("p", 24)
	malas := []map[string]any{
		{"encontrado": false, "auditoria_ref": "aud_1", "correo_ref": refCorreoPG},
		{"encontrado": false, "auditoria_ref": "aud_1", "persona_ref": persona},
		{"encontrado": true, "auditoria_ref": "aud_1", "persona_ref": persona, "correo_ref": refCorreoPG},
		{"encontrado": true, "auditoria_ref": "aud_1", "persona_ref": "can_" + strings.Repeat("p", 24), "correo_ref": refCorreoPG, "sobre": sobrePG()},
		{"encontrado": true, "auditoria_ref": "aud_1", "persona_ref": persona, "correo_ref": "correo:1", "sobre": sobrePG()},
		{"encontrado": true, "auditoria_ref": "", "persona_ref": persona, "correo_ref": refCorreoPG, "sobre": sobrePG()},
		{"encontrado": true, "auditoria_ref": "aud_1", "persona_ref": persona, "correo_ref": refCorreoPG, "sobre": sobrePG(), "direccion": "x@example.org"},
	}
	for i, m := range malas {
		tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, respuestaAvisosPG(t, m)}}
		if _, err := registroAvisosPGPrueba(tx).LeerCorreoActivoAvisos(context.Background(), materialAvisosPG, v3AvisosPGPrueba(t, ports.AudienciaCorreoAvisosLlamamientoInterna)); !errors.Is(err, ports.ErrCorreosNoDisponible) || tx.commits != 0 {
			t.Fatalf("caso %d: %v commits=%d", i, err, tx.commits)
		}
	}
}

func TestLeerCorreoAvisosTraduceErroresYNoConfirma(t *testing.T) {
	for codigo, esperado := range map[string]error{"42501": ports.ErrCorreosProhibido, "40001": ports.ErrCorreosNoDisponible, "22023": ports.ErrCorreosInvalidos} {
		tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: true}, {err: &pgconn.PgError{Code: codigo}}}}
		if _, err := registroAvisosPGPrueba(tx).LeerCorreoActivoAvisos(context.Background(), materialAvisosPG, v3AvisosPGPrueba(t, ports.AudienciaCorreoAvisosLlamamientoInterna)); !errors.Is(err, esperado) || tx.commits != 0 || tx.rollbacks == 0 {
			t.Fatalf("%s: %v", codigo, err)
		}
	}
	// Un LOGIN que no es el ejecutor interno exclusivo no llega a leer.
	tx := &txCorreoPGPrueba{respuestas: []filaCorreoPGPrueba{{valor: false}}}
	if _, err := registroAvisosPGPrueba(tx).LeerCorreoActivoAvisos(context.Background(), materialAvisosPG, v3AvisosPGPrueba(t, ports.AudienciaCorreoAvisosLlamamientoInterna)); !errors.Is(err, ports.ErrCorreosNoDisponible) || len(tx.llamadas) != 2 {
		t.Fatalf("login no acreditado: %v %d", err, len(tx.llamadas))
	}
	// Otra audiencia no abre transacción.
	tx = &txCorreoPGPrueba{}
	if _, err := registroAvisosPGPrueba(tx).LeerCorreoActivoAvisos(context.Background(), materialAvisosPG, v3AvisosPGPrueba(t, ports.AudienciaConsultarCorreosInterna)); !errors.Is(err, ports.ErrCorreosProhibido) || len(tx.llamadas) != 0 {
		t.Fatalf("otra audiencia: %v", err)
	}
}
