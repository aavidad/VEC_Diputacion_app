package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type proveedorPerfilEntregaRevocadoPrueba struct {
	comprobaciones int
	permitir       bool
	auditorias     int
	auditoriaFalla bool
}

func (p *proveedorPerfilEntregaRevocadoPrueba) ActorEntregaPeticionCentro(context.Context) (string, string, error) {
	return "actor:rrhh", "perfil:revocado", nil
}
func (p *proveedorPerfilEntregaRevocadoPrueba) ComprobarPerfilEntregaPeticionCentro(context.Context) error {
	p.comprobaciones++
	if p.permitir {
		return nil
	}
	if p.auditoriaFalla {
		return ports.ErrPeticionCentroNoDisponible
	}
	return ports.ErrAutorizacionDenegada
}
func (p *proveedorPerfilEntregaRevocadoPrueba) RegistrarDenegacionEntregaPreV3(context.Context) error {
	p.auditorias++
	return nil
}
func (*proveedorPerfilEntregaRevocadoPrueba) NuevaClaveAltaDePeticion(context.Context) (string, string, error) {
	return "", "", ports.ErrAutorizacionDenegada
}

func TestAuditoriaFallidaAntesDeV3DetieneProyeccionCon503(t *testing.T) {
	proveedor := &proveedorPerfilEntregaRevocadoPrueba{auditoriaFalla: true}
	lector := &lectorAmbitosProhibidoPrueba{}
	repositorio := &RepositorioEntregasPeticionCentroPostgreSQL{
		pool: &pgxpool.Pool{}, lector: lector, proveedor: proveedor, organizacionRef: "organizacion:desarrollo:dipgra",
	}
	_, err := repositorio.material(context.Background(), "preparar",
		ports.ComandoEntregarPeticionCentro{PeticionRef: "peticion:centro:ratificada", VersionEsperada: 2})
	if !errors.Is(err, ports.ErrPeticionCentroNoDisponible) || lector.consultas != 0 {
		t.Fatalf("fallo auditor llegó a SQL o no devolvió 503: %v, consultas=%d", err, lector.consultas)
	}
}
func (*proveedorPerfilEntregaRevocadoPrueba) AutorizarEntregaPeticionCentro(context.Context, ports.MaterialEntregaPeticionCentro) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrAutorizacionDenegada
}

type lectorAmbitosProhibidoPrueba struct{ consultas int }

func (l *lectorAmbitosProhibidoPrueba) QueryRow(context.Context, string, ...any) pgx.Row {
	l.consultas++
	return nil
}

type filaAmbitosNoEncontradosPrueba struct{}

func (filaAmbitosNoEncontradosPrueba) Scan(...any) error {
	return &pgconn.PgError{Code: "P0681"}
}

type lectorAmbitosNoEncontradosPrueba struct {
	consulta   string
	argumentos []any
}

func (l *lectorAmbitosNoEncontradosPrueba) QueryRow(_ context.Context, consulta string, argumentos ...any) pgx.Row {
	l.consulta = consulta
	l.argumentos = append([]any(nil), argumentos...)
	return filaAmbitosNoEncontradosPrueba{}
}

func TestPerfilEntregaRevocadoNoProyectaExistenciaDeReferencias(t *testing.T) {
	proveedor := &proveedorPerfilEntregaRevocadoPrueba{}
	lector := &lectorAmbitosProhibidoPrueba{}
	repositorio := &RepositorioEntregasPeticionCentroPostgreSQL{
		pool: &pgxpool.Pool{}, lector: lector, proveedor: proveedor, organizacionRef: "organizacion:desarrollo:dipgra",
	}
	for _, ref := range []string{"peticion:centro:ratificada", "peticion:centro:inexistente"} {
		_, err := repositorio.material(context.Background(), "preparar",
			ports.ComandoEntregarPeticionCentro{PeticionRef: ref, VersionEsperada: 2})
		if !errors.Is(err, ports.ErrAutorizacionDenegada) {
			t.Fatalf("%s devolvió un resultado distinguible: %v", ref, err)
		}
	}
	if lector.consultas != 0 || proveedor.comprobaciones != 2 {
		t.Fatalf("la revocación alcanzó la proyección SQL: consultas=%d comprobaciones=%d",
			lector.consultas, proveedor.comprobaciones)
	}
}

func TestReferenciaNoRatificadaNoDistingueDeDenegacionPreV3(t *testing.T) {
	proveedor := &proveedorPerfilEntregaRevocadoPrueba{permitir: true}
	lector := &lectorAmbitosNoEncontradosPrueba{}
	repositorio := &RepositorioEntregasPeticionCentroPostgreSQL{
		pool: &pgxpool.Pool{}, lector: lector, proveedor: proveedor, organizacionRef: "organizacion:desarrollo:dipgra",
	}
	_, err := repositorio.material(context.Background(), "preparar",
		ports.ComandoEntregarPeticionCentro{PeticionRef: "peticion:centro:inexistente", VersionEsperada: 2})
	if !errors.Is(err, ports.ErrAutorizacionDenegada) || proveedor.auditorias != 1 {
		t.Fatalf("la proyección reveló la existencia o quedó sin auditoría: %v, registros=%d",
			err, proveedor.auditorias)
	}
	if lector.consulta != "SELECT centro_ref,categoria_ref FROM vec_contratacion_temporal.ambitos_entrega_peticion_centro_v1($1,$2)" ||
		len(lector.argumentos) != 2 || lector.argumentos[0] != "peticion:centro:inexistente" || lector.argumentos[1] != repositorio.organizacionRef {
		t.Fatalf("proyección sin ámbito confiable: consulta=%q argumentos=%d", lector.consulta, len(lector.argumentos))
	}
}

func TestRecursoEntregaPostIncluyeAmbitosRatificadosYGETNoLosHereda(t *testing.T) {
	get := ports.MaterialEntregaPeticionCentro{
		Modo: "bandeja", ActorRef: "actor:rrhh", PerfilRef: "perfil:lector",
	}
	rGET, err := RecursoEntregaPeticionCentro(get, "organizacion:desarrollo:dipgra")
	if err != nil || len(rGET.Ambitos) != 1 ||
		rGET.Ambitos["organizacion_ref"] != "organizacion:desarrollo:dipgra" {
		t.Fatalf("lector GET: recurso=%+v error=%v", rGET, err)
	}
	post := ports.MaterialEntregaPeticionCentro{
		Modo: "preparar", ActorRef: "actor:rrhh", PerfilRef: "perfil:entrega",
		PeticionRef: "peticion:centro:001", CentroRef: "centro-520",
		CategoriaRef: "categoria:desarrollo:c2", VersionEsperada: 2,
		ClaveAltaCandidata: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		AmbitoAltaHMAC:     "hmac-sha256:vec.contratacion-temporal.ambito-idempotencia/v1:" + strings.Repeat("a", 64),
	}
	rPOST, err := RecursoEntregaPeticionCentro(post, "organizacion:desarrollo:dipgra")
	if err != nil || len(rPOST.Ambitos) != 3 ||
		rPOST.Ambitos["centro_ref"] != post.CentroRef ||
		rPOST.Ambitos["categoria_ref"] != post.CategoriaRef ||
		rPOST.Ambitos["organizacion_ref"] != "organizacion:desarrollo:dipgra" {
		t.Fatalf("POST: recurso=%+v error=%v", rPOST, err)
	}
	h, err := rPOST.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	post.CategoriaRef = "categoria:otra"
	rOtro, err := RecursoEntregaPeticionCentro(post, "organizacion:desarrollo:dipgra")
	if err != nil {
		t.Fatal(err)
	}
	hOtro, err := rOtro.HuellaContextoAutorizacionSHA256()
	if err != nil || h == hOtro {
		t.Fatal("cambiar categoría no cambió la huella del contexto")
	}
	rOrganizacion, err := RecursoEntregaPeticionCentro(post, "org_"+strings.Repeat("b", 32))
	if err != nil {
		t.Fatal(err)
	}
	hOrganizacion, err := rOrganizacion.HuellaContextoAutorizacionSHA256()
	if err != nil || hOtro == hOrganizacion || rOtro.Ambitos["organizacion_ref"] == rOrganizacion.Ambitos["organizacion_ref"] ||
		rOtro.Atributos["material_sha256"] != rOrganizacion.Atributos["material_sha256"] {
		t.Fatal("cambiar organización no separó el recurso conservando el material")
	}
	if _, err := RecursoEntregaPeticionCentro(post, ""); err == nil {
		t.Fatal("organización vacía aceptada")
	}
}
