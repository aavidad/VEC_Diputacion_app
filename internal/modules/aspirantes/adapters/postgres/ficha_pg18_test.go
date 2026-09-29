package postgres

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/aspirantes/adapters/seguridad"
	"vec-diputacion-granada/internal/modules/aspirantes/application"
	"vec-diputacion-granada/internal/modules/aspirantes/canonico"
	"vec-diputacion-granada/internal/modules/aspirantes/domain"
	"vec-diputacion-granada/internal/modules/aspirantes/ports"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// Recorrido completo Go → PostgreSQL 18 con la V3 sintética de forma de
// pruebas_sql/preimagen_ad3_sintetica.sql. Lo lanza ficha_go_pg18.sh con un
// LOGIN ejecutor externo; sin DSN se omite. El DSN nunca se imprime.
func TestFichaPropiaPG18DeExtremoAExtremo(t *testing.T) {
	dsn := os.Getenv("VEC_ASPIRANTES_PG18_DSN")
	if dsn == "" {
		t.Skip("PG18 efímero no configurado")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal("DSN sintético PG18 inválido")
	}
	defer pool.Close()
	registro, err := NuevoRegistroFichasPostgreSQL(ctx, pool)
	if err != nil {
		t.Fatalf("sonda del LOGIN: %v", err)
	}
	cripto, _ := seguridad.NuevoAdaptador(fuenteClavesPG{})
	catalogo := catalogoPG{e: ports.ExigenciasContacto{CatalogoRef: "vec.aspirantes.datos_personales:1", Ejemplo: true, Campos: []domain.ExigenciaCampo{
		{Campo: domain.CampoTelefono, Obligatorio: true}, {Campo: domain.CampoMovil},
		{Campo: domain.CampoDomicilio, Condicion: "si_elige_notificacion_papel"}, {Campo: domain.CampoCodigoPostal, Condicion: "si_elige_notificacion_papel"}}}}
	servicio, err := application.NuevoServicioFichaPropia(application.Dependencias{Registro: registro, Protector: cripto, Sellador: cripto, Catalogo: catalogo, Azar: rand.Reader, AhoraUTC: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	orden := ordenPG(t, "Antonio", "Reyes", "Álvarez", "48123456G")

	v, err := servicio.Consultar(ctx, orden)
	if err != nil || v.Estado != application.EstadoSinFicha || v.Identidad.Documento != "***2345**" {
		t.Fatalf("consulta inicial %+v %v", v.Identidad, err)
	}
	r, err := servicio.Alta(ctx, orden, application.PeticionFicha{ClaveOperacion: "alta-antonio-000000001", Campos: map[string]string{"telefono": "958 12 34 56", "domicilio": "Calle Recogidas, 12, 3.º B", "codigo_postal": "18002"}})
	if err != nil || r.Version != 1 || r.Replay {
		t.Fatalf("alta %+v %v", r, err)
	}
	r2, err := servicio.Alta(ctx, orden, application.PeticionFicha{ClaveOperacion: "alta-antonio-000000001", Campos: map[string]string{"telefono": "958123456", "domicilio": "Calle Recogidas, 12, 3.º B", "codigo_postal": "18002"}})
	if err != nil || !r2.Replay || r2.ReciboRef != r.ReciboRef {
		t.Fatalf("repetición del alta %+v %v", r2, err)
	}
	if _, err := servicio.Alta(ctx, orden, application.PeticionFicha{ClaveOperacion: "alta-antonio-000000002", Campos: map[string]string{"telefono": "958123456"}}); !errors.Is(err, ports.ErrFichaExistente) {
		t.Fatalf("segunda alta: %v", err)
	}
	v, err = servicio.Consultar(ctx, orden)
	if err != nil || v.Estado != application.EstadoActiva || v.Version != 1 || v.Identidad.Nombre != "Antonio" || v.Identidad.SegundoApellido != "Álvarez" ||
		v.Contacto["telefono"] != "958123456" || v.Contacto["domicilio"] != "Calle Recogidas, 12, 3.º B" || !v.CatalogoEjemplo {
		t.Fatalf("consulta %+v %v", v, err)
	}
	r, err = servicio.Rectificar(ctx, orden, application.PeticionFicha{ClaveOperacion: "rect-antonio-000000001", VersionEsperada: 1, Motivo: "cambio_de_dato", Campos: map[string]string{"telefono": "612 345 678", "domicilio": ""}})
	if err != nil || r.Version != 2 {
		t.Fatalf("rectificar %+v %v", r, err)
	}
	if _, err := servicio.Rectificar(ctx, orden, application.PeticionFicha{ClaveOperacion: "rect-antonio-000000002", VersionEsperada: 1, Motivo: "dato_nuevo", Campos: map[string]string{"movil": "699111222"}}); !errors.Is(err, ports.ErrConflicto) {
		t.Fatalf("versión vieja: %v", err)
	}
	if _, err := servicio.Rectificar(ctx, orden, application.PeticionFicha{ClaveOperacion: "rect-antonio-000000003", VersionEsperada: 2, Motivo: "dato_nuevo", Campos: map[string]string{"telefono": "699111222"}}); !errors.Is(err, ports.ErrInvalida) {
		t.Fatalf("motivo incoherente: %v", err)
	}
	v, err = servicio.Consultar(ctx, orden)
	if err != nil || v.Version != 2 || v.Contacto["telefono"] != "612345678" || v.Contacto["domicilio"] != "" || v.Contacto["codigo_postal"] != "18002" {
		t.Fatalf("consulta final %+v %v", v.Contacto, err)
	}
	// Otra persona, con NIE, no ve la ficha de Antonio y crea la suya.
	otra := ordenPG(t, "Karim", "Benali", "", "X1234567L")
	v, err = servicio.Consultar(ctx, otra)
	if err != nil || v.Estado != application.EstadoSinFicha || v.Identidad.TipoDocumento != "nie" {
		t.Fatalf("otra persona %+v %v", v, err)
	}
}

type fuenteClavesPG struct{}

func (fuenteClavesPG) CargarClavesAspirantes(context.Context) (seguridad.ClavesAspirantes, error) {
	clave := func(ref string, b byte) seguridad.Clave {
		var m [32]byte
		for i := range m {
			m[i] = b
		}
		return seguridad.Clave{Ref: ref, Material: m}
	}
	return seguridad.ClavesAspirantes{CifradoActivo: clave("clave:prueba:cifrado:v1", 1), Indice: clave("clave:prueba:indice:v1", 2), SemanticaActiva: clave("clave:prueba:huella:v1", 3)}, nil
}

type catalogoPG struct{ e ports.ExigenciasContacto }

func (c catalogoPG) ExigenciasContactoFichaPropia(context.Context) (ports.ExigenciasContacto, error) {
	return c.e, nil
}

// proveedorPG construye la capacidad y la decisión que coteja la V3
// sintética; la huella es la del recurso canónico, como en producción.
type proveedorPG struct{}

func (proveedorPG) ProveerMaterialFicha(_ context.Context, vinculo vecdomain.VinculoAutenticacionActorV2, m ports.MaterialFicha) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	vacia := vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}
	recurso, err := canonico.Recurso(m)
	if err != nil {
		return vacia, err
	}
	huella, _ := recurso.HuellaContextoAutorizacionSHA256()
	audiencia, _ := ports.Audiencia(m.Accion)
	c, _ := json.Marshal(map[string]string{"suite": "VEC-AD-3-COSE-EDDSA-1", "audiencia_consumo": audiencia, "operacion": m.Accion,
		"efecto_ref": m.PersonaRef, "huella_efecto_sha256": huella, "relleno": strings.Repeat("x", 400)})
	var aleatorio [16]byte
	_, _ = rand.Read(aleatorio[:])
	d, _ := json.Marshal(map[string]any{"principal_id": m.PersonaRef, "perfil_activo_ref": m.PerfilRef, "decision_ref": "dec_" + hex.EncodeToString(aleatorio[:]),
		"accion": m.Accion, "modulo_id": "aspirantes", "tipo_recurso": "ficha_aspirante_propia", "finalidad": ports.FinalidadFicha,
		"recurso_ref": m.PersonaRef, "contexto_recurso_huella_sha256": huella, "vinculo_autenticacion_actor": map[string]string{"superficie": "externa_personal"},
		"concedida": true, "campos_permitidos": ports.CamposPermitidos(m.Accion), "obligaciones": []string{}})
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_pg18", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_pg18", strings.Repeat("c", 64), m.Accion, m.PersonaRef, huella, audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		return vacia, err
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	return vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(c, resumen, d, []byte("motivo"), []byte("contexto"), 1, 1,
		[]byte(hex.EncodeToString(aleatorio[:])), []byte("sobre"), []byte("evidencia"), bytes.Clone(raiz))
}
