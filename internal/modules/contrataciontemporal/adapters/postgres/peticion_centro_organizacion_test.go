package postgres

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type filaPeticionAmbitoPrueba struct{}

func (filaPeticionAmbitoPrueba) Scan(destinos ...any) error {
	if len(destinos) != 1 {
		return errors.New("columnas de consulta distintas")
	}
	b, ok := destinos[0].(*[]byte)
	if !ok {
		return errors.New("tipo de respuesta distinto")
	}
	*b = []byte(`{}`)
	return nil
}

func exportacionPeticionAmbitoPrueba(t *testing.T, accion string, recursoRef, huella string) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	marco := strings.Repeat("a", 64)
	ahora := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:peticion:ambito:001", marco, marco, "contexto:peticion:ambito", marco,
		accion, recursoRef, huella, audienciaPeticionCentro, ahora, ahora.Add(5*time.Second),
	)
	if err != nil {
		t.Fatal(err)
	}
	spki, err := x509.MarshalPKIXPublicKey(ed25519.NewKeyFromSeed(make([]byte, 32)).Public())
	if err != nil {
		t.Fatal(err)
	}
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		[]byte(strings.Repeat("x", 512)), resumen, []byte("{}"), []byte("{}"), []byte("{}"),
		1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), spki,
	)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestRecursoConsultaPeticionCentroLigaOrganizacionSinCambiarMaterial(t *testing.T) {
	c := ports.ConsultaPeticionCentro{
		Modo: "bandeja",
		Actor: domain.ActorPeticionCentro{
			ActorRef: "actor:centro:001", PerfilRef: "perfil:centro:001",
			CentroRef: "centro:001", PuestoRef: "puesto:001",
		},
		Referencia: "centro:001",
	}
	primero, err := RecursoConsultaPeticionCentro(c, "organizacion:desarrollo:dipgra")
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := RecursoConsultaPeticionCentro(c, "org_"+strings.Repeat("b", 32))
	if err != nil {
		t.Fatal(err)
	}
	h1, err := primero.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	h2, err := segundo.HuellaContextoAutorizacionSHA256()
	if err != nil || h1 == h2 || primero.Ambitos["organizacion_ref"] == segundo.Ambitos["organizacion_ref"] ||
		primero.Atributos["material_sha256"] != segundo.Atributos["material_sha256"] {
		t.Fatalf("organizaciones sin separación de ámbito y huella: %v", err)
	}
	if _, err := RecursoConsultaPeticionCentro(c, ""); err == nil {
		t.Fatal("organización vacía aceptada")
	}
}

func TestConsultaPeticionCentroPasaMismoAmbitoAFuncionSQL(t *testing.T) {
	const org = "organizacion:desarrollo:dipgra"
	c := ports.ConsultaPeticionCentro{
		Modo: "bandeja", Referencia: "centro:001",
		Actor: domain.ActorPeticionCentro{ActorRef: "actor:centro:001", PerfilRef: "perfil:centro:001", CentroRef: "centro:001", PuestoRef: "puesto:001"},
	}
	recurso, err := RecursoConsultaPeticionCentro(c, org)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	a := exportacionPeticionAmbitoPrueba(t, ports.AccionConsultarPeticionCentro, recurso.Referencia, huella)
	tx := &transaccionAltaCandidataPrueba{fila: filaPeticionAmbitoPrueba{}}
	pool := &iniciadorAltaCandidataPrueba{transacciones: []pgx.Tx{tx}}
	r := &RepositorioPeticionesCentroPostgreSQL{pool: pool, organizacionRef: org}
	falloValidacion := errors.New("respuesta no validada")
	err = r.ejecutar(context.Background(), "consultar_peticiones_centro_v1", contenido, a, ports.AccionConsultarPeticionCentro, recurso,
		func([]byte) error { return falloValidacion })
	if !errors.Is(err, falloValidacion) || tx.commits != 0 || tx.rollbacks != 1 || pool.inicios != 1 ||
		len(tx.argumentos) != 12 || tx.argumentos[0] != string(contenido) || tx.argumentos[1] != org ||
		!strings.Contains(tx.consulta, "($1::text,$2::text,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)") {
		t.Fatalf("consulta no ligó organización y material antes de COMMIT: %v, argumentos=%d", err, len(tx.argumentos))
	}
	suma := sha256.Sum256(contenido)
	if recurso.Atributos["material_sha256"] != hex.EncodeToString(suma[:]) {
		t.Fatal("la organización cambió la huella del material")
	}
	rOtra := &RepositorioPeticionesCentroPostgreSQL{pool: pool, organizacionRef: "org_" + strings.Repeat("b", 32)}
	err = rOtra.ejecutar(context.Background(), "consultar_peticiones_centro_v1", contenido, a, ports.AccionConsultarPeticionCentro, recurso,
		func([]byte) error { return nil })
	if !errors.Is(err, domain.ErrRatificacionCentroDenegada) || pool.inicios != 1 {
		t.Fatalf("organización ajena alcanzó la transacción: %v, inicios=%d", err, pool.inicios)
	}
}
